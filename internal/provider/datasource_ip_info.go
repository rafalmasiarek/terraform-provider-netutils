package provider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/intelligence/iplookup"
)

type ipInfoModel struct {
	IP             types.String `tfsdk:"ip"`
	ASN            types.Bool   `tfsdk:"asn"`
	Location       types.Bool   `tfsdk:"location"`
	Abuse          types.Bool   `tfsdk:"abuse"`
	RBL            types.Bool   `tfsdk:"rbl"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`

	ResultIP    types.String `tfsdk:"result_ip"`
	Hostname    types.String `tfsdk:"hostname"`
	Network     types.String `tfsdk:"network"`
	ASNNumber   types.String `tfsdk:"asn_number"`
	ASNName     types.String `tfsdk:"asn_name"`
	ASNType     types.String `tfsdk:"asn_type"`
	Country     types.String `tfsdk:"country"`
	CountryName types.String `tfsdk:"country_name"`
	Region      types.String `tfsdk:"region"`
	City        types.String `tfsdk:"city"`
	IsEU        types.Bool   `tfsdk:"is_eu"`
	AbuseRaw    types.String `tfsdk:"abuse_raw"`
	RBLRaw      types.String `tfsdk:"rbl_raw"`
}

type ipInfoDataSource struct {
	endpoint string
}

func NewIPInfoDataSource() datasource.DataSource {
	return &ipInfoDataSource{}
}

func (d *ipInfoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_info"
}

func (d *ipInfoDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*netutilsProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *netutilsProviderData for netutils_ip_info.")
		return
	}
	d.endpoint = data.IPLookupEndpoint
}

func (d *ipInfoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "IP enrichment lookup (ASN, geolocation) against the provider's configured ip_lookup_endpoint. " +
			"A successful response does not guarantee every optional section (asn, location, abuse, rbl) is populated.",
		Attributes: map[string]schema.Attribute{
			"ip": schema.StringAttribute{
				Optional:    true,
				Description: "IP to look up. Omit to look up the public IP observed by the API itself.",
			},
			"asn": schema.BoolAttribute{
				Optional:    true,
				Description: "Request ASN information. Defaults to true.",
			},
			"location": schema.BoolAttribute{
				Optional:    true,
				Description: "Request geolocation information. Defaults to true.",
			},
			"abuse": schema.BoolAttribute{
				Optional:    true,
				Description: "Request abuse-contact information. Defaults to false.",
			},
			"rbl": schema.BoolAttribute{
				Optional:    true,
				Description: "Request realtime-blackhole-list status. Defaults to false.",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "HTTP timeout for the lookup request, in seconds. Defaults to 10.",
			},
			"result_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The IP address the response describes.",
			},
			"hostname": schema.StringAttribute{
				Computed: true,
			},
			"network": schema.StringAttribute{
				Computed:    true,
				Description: "The CIDR block the IP's network announces, if known.",
			},
			"asn_number": schema.StringAttribute{
				Computed:    true,
				Description: "Empty if the asn section wasn't requested or wasn't available.",
			},
			"asn_name": schema.StringAttribute{
				Computed: true,
			},
			"asn_type": schema.StringAttribute{
				Computed: true,
			},
			"country": schema.StringAttribute{
				Computed: true,
			},
			"country_name": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				Computed: true,
			},
			"city": schema.StringAttribute{
				Computed: true,
			},
			"is_eu": schema.BoolAttribute{
				Computed: true,
			},
			"abuse_raw": schema.StringAttribute{
				Computed: true,
				Description: "Raw JSON of the response's abuse section, null if absent. The upstream API's exact field set for " +
					"this section isn't part of this provider's own documented contract yet - parse it yourself with jsondecode() if needed.",
			},
			"rbl_raw": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON of the response's rbl section, null if absent - same caveat as abuse_raw.",
			},
		},
	}
}

func (d *ipInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ipInfoModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout := 10 * time.Second
	if !config.TimeoutSeconds.IsNull() {
		timeout = time.Duration(config.TimeoutSeconds.ValueInt64()) * time.Second
	}

	opts := iplookup.LookupOptions{
		IP:       config.IP.ValueString(),
		ASN:      config.ASN.IsNull() || config.ASN.ValueBool(),
		Location: config.Location.IsNull() || config.Location.ValueBool(),
		Abuse:    !config.Abuse.IsNull() && config.Abuse.ValueBool(),
		RBL:      !config.RBL.IsNull() && config.RBL.ValueBool(),
	}

	client := iplookup.NewClient(d.endpoint, timeout)
	result, err := client.Lookup(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Unable to look up IP information", err.Error())
		return
	}

	config.ResultIP = types.StringValue(result.Data.IP)
	config.Hostname = types.StringValue(result.Data.Hostname)
	config.Network = types.StringValue(result.Data.Network)

	if result.Data.ASN != nil {
		config.ASNNumber = types.StringValue(result.Data.ASN.ASN)
		config.ASNName = types.StringValue(result.Data.ASN.Name)
		config.ASNType = types.StringValue(result.Data.ASN.Type)
	} else {
		config.ASNNumber = types.StringValue("")
		config.ASNName = types.StringValue("")
		config.ASNType = types.StringValue("")
	}

	if result.Data.Location != nil {
		config.Country = types.StringValue(result.Data.Location.Country)
		config.CountryName = types.StringValue(result.Data.Location.CountryName)
		config.Region = optionalString(result.Data.Location.Region)
		config.City = optionalString(result.Data.Location.City)
		config.IsEU = types.BoolValue(result.Data.Location.IsEU)
	} else {
		config.Country = types.StringValue("")
		config.CountryName = types.StringValue("")
		config.Region = types.StringNull()
		config.City = types.StringNull()
		config.IsEU = types.BoolValue(false)
	}

	config.AbuseRaw = types.StringNull()
	if result.Data.Abuse != nil {
		config.AbuseRaw = rawJSON(*result.Data.Abuse)
	}
	config.RBLRaw = types.StringNull()
	if result.Data.RBL != nil {
		config.RBLRaw = rawJSON(*result.Data.RBL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func optionalString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func rawJSON(v any) types.String {
	b, err := json.Marshal(v)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(b))
}
