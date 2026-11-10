package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

var cidrOverlapPairAttributeTypes = map[string]attr.Type{
	"cidr_a": types.StringType,
	"cidr_b": types.StringType,
}

type cidrOverlapAnalysisFunction struct{}

func NewCIDROverlapAnalysisFunction() function.Function {
	return &cidrOverlapAnalysisFunction{}
}

func (f *cidrOverlapAnalysisFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_overlap_analysis"
}

func (f *cidrOverlapAnalysisFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports every pair of CIDR blocks in a list that overlap - e.g. conflicting route-table entries.",
		Description: "Each unordered pair is reported once. An empty result means no two entries in the list share any address.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				ElementType: types.StringType,
				Description: "CIDR blocks to check pairwise, e.g. the destination_cidr_block of every route in a route table.",
			},
		},
		Return: function.ListReturn{
			ElementType: types.ObjectType{AttrTypes: cidrOverlapPairAttributeTypes},
		},
	}
}

func (f *cidrOverlapAnalysisFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrs []string
	resp.Error = req.Arguments.Get(ctx, &cidrs)
	if resp.Error != nil {
		return
	}

	pairs, err := netutil.CIDROverlapPairs(cidrs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid cidr in list: %s", err))
		return
	}

	results := make([]attr.Value, 0, len(pairs))
	for _, pair := range pairs {
		obj, diags := types.ObjectValue(cidrOverlapPairAttributeTypes, map[string]attr.Value{
			"cidr_a": types.StringValue(cidrs[pair[0]]),
			"cidr_b": types.StringValue(cidrs[pair[1]]),
		})
		if diags.HasError() {
			resp.Error = function.FuncErrorFromDiags(ctx, diags)
			return
		}
		results = append(results, obj)
	}

	listValue, diags := types.ListValue(types.ObjectType{AttrTypes: cidrOverlapPairAttributeTypes}, results)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	resp.Error = resp.Result.Set(ctx, listValue)
}
