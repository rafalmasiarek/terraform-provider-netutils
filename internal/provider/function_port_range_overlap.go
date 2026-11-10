package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type portRangeOverlapFunction struct{}

func NewPortRangeOverlapFunction() function.Function {
	return &portRangeOverlapFunction{}
}

func (f *portRangeOverlapFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "port_range_overlap"
}

func (f *portRangeOverlapFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports whether two port ranges share at least one port.",
		Description: "Each range must independently satisfy port_range_validate.",
		Parameters: []function.Parameter{
			function.Int64Parameter{Name: "a_from", Description: "First range: starting port."},
			function.Int64Parameter{Name: "a_to", Description: "First range: ending port."},
			function.Int64Parameter{Name: "b_from", Description: "Second range: starting port."},
			function.Int64Parameter{Name: "b_to", Description: "Second range: ending port."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *portRangeOverlapFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var aFrom, aTo, bFrom, bTo int64
	resp.Error = req.Arguments.Get(ctx, &aFrom, &aTo, &bFrom, &bTo)
	if resp.Error != nil {
		return
	}

	result, err := netutil.PortRangeOverlap(int(aFrom), int(aTo), int(bFrom), int(bTo))
	if err != nil {
		resp.Error = function.NewFuncError(fmt.Sprintf("%s", err))
		return
	}

	resp.Error = resp.Result.Set(ctx, result)
}
