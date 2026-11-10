package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

type cidrFamilyFunction struct{}

func NewCIDRFamilyFunction() function.Function {
	return &cidrFamilyFunction{}
}

func (f *cidrFamilyFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_family"
}

func (f *cidrFamilyFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Returns \"ipv4\" or \"ipv6\" for a valid CIDR block.",
		Description: "Fails (via can()) for an invalid CIDR, including a bare IP address with no prefix length.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "cidr",
				Description: "An IPv4 or IPv6 CIDR block.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *cidrFamilyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr string
	resp.Error = req.Arguments.Get(ctx, &cidr)
	if resp.Error != nil {
		return
	}

	family, err := netutil.CIDRFamily(cidr)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("invalid CIDR %q: %s", cidr, err))
		return
	}

	resp.Error = resp.Result.Set(ctx, family)
}
