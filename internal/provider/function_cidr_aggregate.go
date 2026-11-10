package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrAggregateFunction struct{}

func NewCIDRAggregateFunction() function.Function {
	return &cidrAggregateFunction{}
}

func (f *cidrAggregateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_aggregate"
}

func (f *cidrAggregateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Merges a list of CIDR blocks into the minimal non-overlapping set covering the same addresses.",
		Description: "Redundant entries (duplicates, or one fully contained in another) are dropped; sibling blocks that " +
			"together form a larger aligned block are merged, repeated to a fixed point. IPv4 and IPv6 entries aggregate " +
			"independently - the result lists IPv4 first, then IPv6, each in ascending address order.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				ElementType: types.StringType,
				Description: "CIDR blocks to merge.",
			},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *cidrAggregateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrs []string
	resp.Error = req.Arguments.Get(ctx, &cidrs)
	if resp.Error != nil {
		return
	}

	result, err := netutil.Aggregate(cidrs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid cidr in list: %s", err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
