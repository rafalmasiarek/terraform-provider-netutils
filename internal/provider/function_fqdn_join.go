package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type fqdnJoinFunction struct{}

func NewFQDNJoinFunction() function.Function {
	return &fqdnJoinFunction{}
}

func (f *fqdnJoinFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "fqdn_join"
}

func (f *fqdnJoinFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Joins a hostname and a domain into a canonical, fully-qualified domain name.",
		Description: "The result always has a trailing dot. A trailing dot already present on either argument is stripped first, so joining is never doubled.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "hostname",
				Description: "The hostname part, with or without a trailing dot.",
			},
			function.StringParameter{
				Name:        "domain",
				Description: "The domain part, with or without leading/trailing dots.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *fqdnJoinFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hostname, domain string
	resp.Error = req.Arguments.Get(ctx, &hostname, &domain)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, netutil.FQDNJoin(hostname, domain))
}
