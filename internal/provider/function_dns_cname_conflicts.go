package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

var dnsRecordRefAttributeTypes = map[string]attr.Type{
	"name": types.StringType,
	"type": types.StringType,
}

var dnsCNAMEConflictAttributeTypes = map[string]attr.Type{
	"name":  types.StringType,
	"types": types.ListType{ElemType: types.StringType},
}

type dnsRecordRefModel struct {
	Name string `tfsdk:"name"`
	Type string `tfsdk:"type"`
}

type dnsCNAMEConflictsFunction struct{}

func NewDNSCNAMEConflictsFunction() function.Function {
	return &dnsCNAMEConflictsFunction{}
}

func (f *dnsCNAMEConflictsFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dns_cname_conflicts"
}

func (f *dnsCNAMEConflictsFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Reports every name in a list of DNS records where a CNAME was declared alongside any other record type.",
		Description: "Per RFC 1034, a CNAME must be the only record at its name. Names are compared case-insensitively with " +
			"trailing dots ignored. Multiple records of the same non-CNAME type at one name (e.g. round-robin A records) are not a conflict.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "records",
				ElementType: types.ObjectType{AttrTypes: dnsRecordRefAttributeTypes},
				Description: "Each record's name (defaulted/relative name as it will actually be created, not an unrelated internal label) and type.",
			},
		},
		Return: function.ListReturn{
			ElementType: types.ObjectType{AttrTypes: dnsCNAMEConflictAttributeTypes},
		},
	}
}

func (f *dnsCNAMEConflictsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var records []dnsRecordRefModel
	resp.Error = req.Arguments.Get(ctx, &records)
	if resp.Error != nil {
		return
	}

	refs := make([]netutil.DNSRecordRef, len(records))
	for i, r := range records {
		refs[i] = netutil.DNSRecordRef{Name: r.Name, Type: r.Type}
	}

	conflicts := netutil.DNSCNAMEConflicts(refs)

	results := make([]attr.Value, 0, len(conflicts))
	for _, c := range conflicts {
		typeValues := make([]attr.Value, len(c.Types))
		for i, t := range c.Types {
			typeValues[i] = types.StringValue(t)
		}
		typesList, diags := types.ListValue(types.StringType, typeValues)
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		if resp.Error != nil {
			return
		}

		obj, diags := types.ObjectValue(dnsCNAMEConflictAttributeTypes, map[string]attr.Value{
			"name":  types.StringValue(c.Name),
			"types": typesList,
		})
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		if resp.Error != nil {
			return
		}
		results = append(results, obj)
	}

	listValue, diags := types.ListValue(types.ObjectType{AttrTypes: dnsCNAMEConflictAttributeTypes}, results)
	resp.Error = function.FuncErrorFromDiags(ctx, diags)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, listValue)
}
