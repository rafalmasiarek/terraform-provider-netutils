package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ipInCIDRFunction struct{}

func NewIPInCIDRFunction() function.Function {
	return &ipInCIDRFunction{}
}

func (f *ipInCIDRFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_in_cidr"
}

func (f *ipInCIDRFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports whether an IP address falls within a CIDR block.",
		Description: "ip and cidr must be the same address family (both IPv4 or both IPv6) - a family mismatch returns false rather than an error.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "An IPv4 or IPv6 address.",
			},
			function.StringParameter{
				Name:        "cidr",
				Description: "An IPv4 or IPv6 CIDR block, e.g. \"10.0.0.0/16\".",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *ipInCIDRFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip, cidr string
	resp.Error = req.Arguments.Get(ctx, &ip, &cidr)
	if resp.Error != nil {
		return
	}

	result, err := netutil.InCIDR(ip, cidr)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("invalid ip %q or cidr %q: %s", ip, cidr, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
