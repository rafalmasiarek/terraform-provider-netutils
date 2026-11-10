package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrDifferenceFunction struct{}

func NewCIDRDifferenceFunction() function.Function {
	return &cidrDifferenceFunction{}
}

func (f *cidrDifferenceFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_difference"
}

func (f *cidrDifferenceFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Returns the minimal list of CIDRs covering cidr minus exclude.",
		Description: "If the two don't overlap, the result is [cidr] unchanged. If exclude fully contains (or exactly equals) " +
			"cidr, the result is empty. cidr and exclude must be the same address family.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr",
				Description: "The base CIDR block.",
			},
			function.StringParameter{
				Name:        "exclude",
				Description: "The CIDR block to remove from cidr.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *cidrDifferenceFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr, exclude string
	resp.Error = req.Arguments.Get(ctx, &cidr, &exclude)
	if resp.Error != nil {
		return
	}

	result, err := netutil.Difference(cidr, exclude)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("invalid cidr %q or exclude %q: %s", cidr, exclude, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
