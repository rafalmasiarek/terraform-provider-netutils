package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrIntersectionFunction struct{}

func NewCIDRIntersectionFunction() function.Function {
	return &cidrIntersectionFunction{}
}

func (f *cidrIntersectionFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_intersection"
}

func (f *cidrIntersectionFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Returns the overlapping range of two CIDR blocks, or null if they don't overlap.",
		Description: "CIDR prefixes are hierarchical - whenever two overlap, one always fully contains the other, so their " +
			"intersection is exactly the more specific (larger prefix length) of the two. A family mismatch is treated as no overlap.",
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
		Return: function.StringReturn{},
	}
}

func (f *cidrIntersectionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = req.Arguments.Get(ctx, &a, &b)
	if resp.Error != nil {
		return
	}

	result, ok, err := netutil.Intersection(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("invalid cidr_a %q or cidr_b %q: %s", a, b, err))
		return
	}

	if !ok {
		resp.Error = resp.Result.Set(ctx, types.StringNull())
		return
	}

	resp.Error = resp.Result.Set(ctx, types.StringValue(result))
}
