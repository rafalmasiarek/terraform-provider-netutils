package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrContainsFunction struct{}

func NewCIDRContainsFunction() function.Function {
	return &cidrContainsFunction{}
}

func (f *cidrContainsFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_contains"
}

func (f *cidrContainsFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports whether every address in cidr is also within parent_cidr.",
		Description: "A prefix contains itself. A child with a shorter prefix (a larger range) than the parent can never be contained, and returns false. parent_cidr and cidr must be the same address family - a family mismatch returns false rather than an error.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "parent_cidr",
				Description: "The candidate containing CIDR block.",
			},
			function.StringParameter{
				Name:        "cidr",
				Description: "The candidate contained CIDR block.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *cidrContainsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var parent, child string
	resp.Error = req.Arguments.Get(ctx, &parent, &child)
	if resp.Error != nil {
		return
	}

	result, err := netutil.CIDRContains(parent, child)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("invalid parent_cidr %q or cidr %q: %s", parent, child, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
