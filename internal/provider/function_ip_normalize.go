package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ipNormalizeFunction struct{}

func NewIPNormalizeFunction() function.Function {
	return &ipNormalizeFunction{}
}

func (f *ipNormalizeFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_normalize"
}

func (f *ipNormalizeFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Normalizes an IPv4 or IPv6 address to its canonical string form.",
		Description: "IPv6 addresses are compressed and lowercased per RFC 5952 (e.g. \"2001:0DB8::0001\" -> \"2001:db8::1\"). IPv4 addresses are returned as plain dotted-decimal.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "An IPv4 or IPv6 address.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ipNormalizeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = req.Arguments.Get(ctx, &ip)
	if resp.Error != nil {
		return
	}

	normalized, err := netutil.Normalize(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid IP address %q: %s", ip, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, normalized)
}
