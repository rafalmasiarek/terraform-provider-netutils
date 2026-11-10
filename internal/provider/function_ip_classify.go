package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ipClassifyFunction struct{}

func NewIPClassifyFunction() function.Function {
	return &ipClassifyFunction{}
}

func (f *ipClassifyFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_classify"
}

func (f *ipClassifyFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Classifies an IPv4 or IPv6 address into a single category.",
		Description: "Returns one of: \"unspecified\", \"loopback\", \"multicast\", \"link-local\", \"cgnat\" (RFC 6598, IPv4 only), " +
			"\"documentation\" (RFC 5737/3849 example ranges), \"benchmarking\" (RFC 2544/5180), \"private\" (RFC 1918 or RFC 4193 ULA), " +
			"\"public\", or \"reserved\" (valid but matches no other category above).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "An IPv4 or IPv6 address.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ipClassifyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = req.Arguments.Get(ctx, &ip)
	if resp.Error != nil {
		return
	}

	class, err := netutil.Classify(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid IP address %q: %s", ip, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, class)
}
