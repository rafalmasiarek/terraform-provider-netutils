package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

var cidrCapacityAttributeTypes = map[string]attr.Type{
	"total":  types.StringType,
	"usable": types.StringType,
}

type cidrCapacityFunction struct{}

func NewCIDRCapacityFunction() function.Function {
	return &cidrCapacityFunction{}
}

func (f *cidrCapacityFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_capacity"
}

func (f *cidrCapacityFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Returns the total and usable address counts of a CIDR block.",
		Description: "IPv4 usable excludes the network and broadcast address (total-2), except /31 (2, point-to-point) and /32 (1). " +
			"IPv6 has no broadcast address, so usable equals total. Both values are decimal strings, not numbers - a /0 IPv6 total is 2^128, " +
			"far beyond what Terraform's Number type can represent exactly.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr",
				Description: "An IPv4 or IPv6 CIDR block.",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: cidrCapacityAttributeTypes,
		},
	}
}

func (f *cidrCapacityFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = req.Arguments.Get(ctx, &cidr)
	if resp.Error != nil {
		return
	}

	total, usable, err := netutil.Capacity(cidr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid cidr %q: %s", cidr, err))
		return
	}

	result, diags := types.ObjectValue(cidrCapacityAttributeTypes, map[string]attr.Value{
		"total":  types.StringValue(total),
		"usable": types.StringValue(usable),
	})
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
