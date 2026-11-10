package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type hostnameValidateFunction struct{}

func NewHostnameValidateFunction() function.Function {
	return &hostnameValidateFunction{}
}

func (f *hostnameValidateFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hostname_validate"
}

func (f *hostnameValidateFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Validates a (possibly multi-label) hostname.",
		Description: "Every dot-separated label must pass dns_label_validate, and the total length (excluding an optional trailing dot) must be at most 253 characters.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "hostname",
				Description: "A hostname, with or without dots.",
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *hostnameValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hostname string
	resp.Error = req.Arguments.Get(ctx, &hostname)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.HostnameValidate(hostname))
}
