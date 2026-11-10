package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/diagnostics/tlscheck"
)

type tlsCertificateModel struct {
	Host            types.String `tfsdk:"host"`
	Port            types.Int64  `tfsdk:"port"`
	TimeoutSeconds  types.Int64  `tfsdk:"timeout_seconds"`
	Success         types.Bool   `tfsdk:"success"`
	Error           types.String `tfsdk:"error"`
	NotBefore       types.String `tfsdk:"not_before"`
	NotAfter        types.String `tfsdk:"not_after"`
	Issuer          types.String `tfsdk:"issuer"`
	Subject         types.String `tfsdk:"subject"`
	DNSNames        []string     `tfsdk:"dns_names"`
	Expired         types.Bool   `tfsdk:"expired"`
	DaysUntilExpiry types.Int64  `tfsdk:"days_until_expiry"`
}

type tlsCertificateDataSource struct{}

func NewTLSCertificateDataSource() datasource.DataSource {
	return &tlsCertificateDataSource{}
}

func (d *tlsCertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tls_certificate"
}

func (d *tlsCertificateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Inspects the leaf TLS certificate presented by host:port, whatever it is - including an expired or " +
			"otherwise untrusted one. This is a diagnostic, not a trust decision; it does not validate the certificate chain.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{Required: true},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "Defaults to 443.",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Defaults to 5.",
			},
			"success": schema.BoolAttribute{
				Computed:    true,
				Description: "False on a connection or handshake failure - see error. All other attributes are empty/zero in that case.",
			},
			"error":             schema.StringAttribute{Computed: true},
			"not_before":        schema.StringAttribute{Computed: true, Description: "RFC 3339."},
			"not_after":         schema.StringAttribute{Computed: true, Description: "RFC 3339."},
			"issuer":            schema.StringAttribute{Computed: true},
			"subject":           schema.StringAttribute{Computed: true},
			"dns_names":         schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"expired":           schema.BoolAttribute{Computed: true},
			"days_until_expiry": schema.Int64Attribute{Computed: true, Description: "Negative if already expired."},
		},
	}
}

func (d *tlsCertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config tlsCertificateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	port := 443
	if !config.Port.IsNull() {
		port = int(config.Port.ValueInt64())
	}
	timeout := 5 * time.Second
	if !config.TimeoutSeconds.IsNull() {
		timeout = time.Duration(config.TimeoutSeconds.ValueInt64()) * time.Second
	}

	result := tlscheck.Check(ctx, config.Host.ValueString(), port, timeout)

	config.Success = types.BoolValue(result.Success)
	config.Error = types.StringValue(result.Error)
	config.DNSNames = result.DNSNames
	if config.DNSNames == nil {
		config.DNSNames = []string{}
	}

	if result.Success {
		config.NotBefore = types.StringValue(result.NotBefore.Format(time.RFC3339))
		config.NotAfter = types.StringValue(result.NotAfter.Format(time.RFC3339))
		config.Issuer = types.StringValue(result.Issuer)
		config.Subject = types.StringValue(result.Subject)
		config.Expired = types.BoolValue(result.Expired)
		config.DaysUntilExpiry = types.Int64Value(result.DaysUntilExpiry)
	} else {
		config.NotBefore = types.StringValue("")
		config.NotAfter = types.StringValue("")
		config.Issuer = types.StringValue("")
		config.Subject = types.StringValue("")
		config.Expired = types.BoolValue(false)
		config.DaysUntilExpiry = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
