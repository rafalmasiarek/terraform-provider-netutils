package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type dnsLabelValidateFunction struct{}

func NewDNSLabelValidateFunction() function.Function {
	return &dnsLabelValidateFunction{}
}

func (f *dnsLabelValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dns_label_validate"
}

func (f *dnsLabelValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Validates a single DNS label per RFC 1123.",
		Description: "1-63 characters, alphanumeric and hyphen only, must start and end with an alphanumeric character.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "label",
				Description: "A single DNS label (no dots).",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *dnsLabelValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var label string
	resp.Error = req.Arguments.Get(ctx, &label)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.DNSLabelValidate(label))
}
