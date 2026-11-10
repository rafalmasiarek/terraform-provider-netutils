package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ptrNameFunction struct{}

func NewPTRNameFunction() function.Function {
	return &ptrNameFunction{}
}

func (f *ptrNameFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ptr_name"
}

func (f *ptrNameFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Returns the reverse-DNS (PTR) name for an IP address.",
		Description: "Reversed octets under \".in-addr.arpa.\" for IPv4, or reversed nibbles of the full expanded form " +
			"under \".ip6.arpa.\" for IPv6. The result has a trailing dot.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "An IPv4 or IPv6 address.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ptrNameFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = req.Arguments.Get(ctx, &ip)
	if resp.Error != nil {
		return
	}

	result, err := netutil.PTRName(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid IP address %q: %s", ip, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
