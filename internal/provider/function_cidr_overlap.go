package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrOverlapFunction struct{}

func NewCIDROverlapFunction() function.Function {
	return &cidrOverlapFunction{}
}

func (f *cidrOverlapFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_overlap"
}

func (f *cidrOverlapFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports whether two CIDR blocks share at least one address.",
		Description: "Symmetric: cidr_overlap(a, b) == cidr_overlap(b, a). Identical blocks overlap. cidr_a and cidr_b must be the same address family - a family mismatch returns false rather than an error.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr_a",
				Description: "The first CIDR block.",
			},
			function.StringParameter{
				Name:        "cidr_b",
				Description: "The second CIDR block.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *cidrOverlapFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = req.Arguments.Get(ctx, &a, &b)
	if resp.Error != nil {
		return
	}

	result, err := netutil.CIDROverlap(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("invalid cidr_a %q or cidr_b %q: %s", a, b, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
