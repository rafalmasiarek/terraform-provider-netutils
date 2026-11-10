package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type caaRecordValidateFunction struct{}

func NewCAARecordValidateFunction() function.Function {
	return &caaRecordValidateFunction{}
}

func (f *caaRecordValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "caa_record_validate"
}

func (f *caaRecordValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Validates a CAA record's flags and tag per RFC 6844.",
		Description: "flags must be 0-255; tag must be \"issue\", \"issuewild\", or \"iodef\" (case-insensitive).",
		Parameters: []function.Parameter{
			function.Int64Parameter{Name: "flags"},
			function.StringParameter{Name: "tag"},
		},
		Return: function.BoolReturn{},
	}
}

func (f *caaRecordValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var flags int64
	var tag string
	resp.Error = req.Arguments.Get(ctx, &flags, &tag)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.CAARecordValidate(flags, tag))
}
