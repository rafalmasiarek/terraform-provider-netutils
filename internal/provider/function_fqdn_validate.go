package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type fqdnValidateFunction struct{}

func NewFQDNValidateFunction() function.Function {
	return &fqdnValidateFunction{}
}

func (f *fqdnValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "fqdn_validate"
}

func (f *fqdnValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Validates a fully-qualified domain name.",
		Description: "Every dot-separated label must pass dns_label_validate, and the total length (excluding an optional trailing dot) must be at most 253 characters.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "fqdn",
				Description: "A fully-qualified domain name, with or without a trailing dot.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *fqdnValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var fqdn string
	resp.Error = req.Arguments.Get(ctx, &fqdn)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.FQDNValidate(fqdn))
}
