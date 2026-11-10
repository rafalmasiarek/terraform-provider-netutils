package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type dnsTXTLengthValidateFunction struct{}

func NewDNSTXTLengthValidateFunction() function.Function {
	return &dnsTXTLengthValidateFunction{}
}

func (f *dnsTXTLengthValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dns_txt_length_validate"
}

func (f *dnsTXTLengthValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Reports whether value fits in a single DNS TXT character-string (RFC 1035: max 255 bytes).",
		Description: "A provider that transparently chunks longer values across multiple character-strings is a looser constraint than this - false means \"would need chunking\", not necessarily \"rejected outright\" by every provider.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "value"},
		},
		Return: function.BoolReturn{},
	}
}

func (f *dnsTXTLengthValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value string
	resp.Error = req.Arguments.Get(ctx, &value)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.DNSTXTLengthValidate(value))
}
