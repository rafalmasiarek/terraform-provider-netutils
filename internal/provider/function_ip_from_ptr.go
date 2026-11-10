package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type ipFromPTRFunction struct{}

func NewIPFromPTRFunction() function.Function {
	return &ipFromPTRFunction{}
}

func (f *ipFromPTRFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_from_ptr"
}

func (f *ipFromPTRFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Parses a PTR name (as produced by ptr_name) back into its normalized IP address.",
		Description: "Accepts names ending in \".in-addr.arpa\" (IPv4) or \".ip6.arpa\" (IPv6), with or without a trailing dot.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ptr",
				Description: "A reverse-DNS PTR name.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ipFromPTRFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ptr string
	resp.Error = req.Arguments.Get(ctx, &ptr)
	if resp.Error != nil {
		return
	}

	result, err := netutil.IPFromPTR(ptr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid PTR name %q: %s", ptr, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
