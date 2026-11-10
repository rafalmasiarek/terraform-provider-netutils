package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultIPLookupEndpoint = "https://masiarek.pl/api/v2/ip/lookup"

type netutilsProvider struct {
	version string
}

type netutilsProviderModel struct {
	IPLookupEndpoint types.String `tfsdk:"ip_lookup_endpoint"`
}

// netutilsProviderData is what Configure() hands to every data source that
// needs the resolved IP lookup endpoint via req.ProviderData.
type netutilsProviderData struct {
	IPLookupEndpoint string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &netutilsProvider{version: version}
	}
}

func (p *netutilsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netutils"
	resp.Version = p.version
}

func (p *netutilsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Platform-agnostic networking utilities for Terraform: IP/CIDR intelligence, subnet planning, DNS/hostname tooling, and network diagnostics - usable standalone, with no dependency on any other provider or module.",
		Attributes: map[string]schema.Attribute{
			"ip_lookup_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Backend for netutils_ip_info enrichment lookups. Defaults to the official endpoint (" + defaultIPLookupEndpoint + ") - override only to point at a self-hosted or mirrored instance of the same API.",
			},
		},
	}
}

func (p *netutilsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg netutilsProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := cfg.IPLookupEndpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("NETUTILS_IP_LOOKUP_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = defaultIPLookupEndpoint
	}

	data := &netutilsProviderData{IPLookupEndpoint: endpoint}
	resp.DataSourceData = data
}

func (p *netutilsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSubnetPlanDataSource,
		NewIPInfoDataSource,
		NewTCPCheckDataSource,
		NewHTTPCheckDataSource,
		NewTLSCertificateDataSource,
	}
}

func (p *netutilsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *netutilsProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		// IP & CIDR core
		NewIPNormalizeFunction,
		NewIPClassifyFunction,
		NewIPInCIDRFunction,
		NewCIDRContainsFunction,
		NewCIDROverlapFunction,
		NewCIDRCapacityFunction,
		NewCIDRAggregateFunction,
		NewCIDRIntersectionFunction,
		NewCIDRDifferenceFunction,
		NewPTRNameFunction,
		NewIPFromPTRFunction,

		// DNS & hostname
		NewDNSLabelValidateFunction,
		NewHostnameValidateFunction,
		NewHostnameNormalizeFunction,
		NewFQDNJoinFunction,
		NewFQDNValidateFunction,

		// Network configuration analysis
		NewCIDROverlapAnalysisFunction,
		NewPortRangeValidateFunction,
		NewPortRangeOverlapFunction,

		// DNS record validation
		NewIPFamilyFunction,
		NewCIDRFamilyFunction,
		NewCAARecordValidateFunction,
		NewDNSTXTLengthValidateFunction,
		NewDNSCNAMEConflictsFunction,
	}
}
