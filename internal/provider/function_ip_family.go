package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ipFamilyFunction struct{}

func NewIPFamilyFunction() function.Function {
	return &ipFamilyFunction{}
}

func (f *ipFamilyFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_family"
}

func (f *ipFamilyFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Returns \"ipv4\" or \"ipv6\" for a valid IP address.",
		Description: "Fails (via can()) for an invalid address - use this to distinguish A-record-shaped from AAAA-record-shaped values.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "An IPv4 or IPv6 address.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ipFamilyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = req.Arguments.Get(ctx, &ip)
	if resp.Error != nil {
		return
	}

	family, err := netutil.IPFamily(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid IP address %q: %s", ip, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, family)
}
