package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type portRangeValidateFunction struct{}

func NewPortRangeValidateFunction() function.Function {
	return &portRangeValidateFunction{}
}

func (f *portRangeValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "port_range_validate"
}

func (f *portRangeValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Validates a port range.",
		Description: "Both from and to must be within 1-65535, and from <= to.",
		Parameters: []function.Parameter{
			function.Int64Parameter{Name: "from", Description: "Starting port."},
			function.Int64Parameter{Name: "to", Description: "Ending port."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *portRangeValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var from, to int64
	resp.Error = req.Arguments.Get(ctx, &from, &to)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.PortRangeValidate(int(from), int(to)))
}
