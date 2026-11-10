package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type hostnameNormalizeFunction struct{}

func NewHostnameNormalizeFunction() function.Function {
	return &hostnameNormalizeFunction{}
}

func (f *hostnameNormalizeFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hostname_normalize"
}

func (f *hostnameNormalizeFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Sanitizes a string into a valid hostname.",
		Description: "Lowercases and sanitizes each dot-separated label independently: invalid characters become hyphens, " +
			"runs of hyphens collapse to one, leading/trailing hyphens are stripped, and empty labels are dropped.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "hostname",
				Description: "A candidate hostname, with or without dots.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *hostnameNormalizeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hostname string
	resp.Error = req.Arguments.Get(ctx, &hostname)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.HostnameNormalize(hostname))
}
