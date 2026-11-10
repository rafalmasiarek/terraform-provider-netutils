package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

var subnetPlanResultAttributeTypes = map[string]attr.Type{
	"name":            types.StringType,
	"cidr":            types.StringType,
	"prefix_length":   types.Int64Type,
	"address_family":  types.StringType,
	"total_capacity":  types.StringType,
	"usable_capacity": types.StringType,
	"index":           types.Int64Type,
	"base_cidr":       types.StringType,
	"status":          types.StringType,
}

type subnetPlanAllocationSpecModel struct {
	Count         int64 `tfsdk:"count"`
	RequiredHosts int64 `tfsdk:"required_hosts"`
}

type subnetPlanConfigModel struct {
	BaseCIDRs           []string                                 `tfsdk:"base_cidrs"`
	AllocationStrategy  types.String                             `tfsdk:"allocation_strategy"`
	ReservedCIDRs       []string                                 `tfsdk:"reserved_cidrs"`
	Allocations         map[string]subnetPlanAllocationSpecModel `tfsdk:"allocations"`
	ExistingAllocations map[string]string                        `tfsdk:"existing_allocations"`
	Results             types.Map                                `tfsdk:"results"`
}

type subnetPlanDataSource struct{}

func NewSubnetPlanDataSource() datasource.DataSource {
	return &subnetPlanDataSource{}
}

func (d *subnetPlanDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet_plan"
}

func (d *subnetPlanDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deterministic, stateless subnet allocation planner: carves CIDR blocks sized from required_hosts out of base_cidrs, " +
			"honoring reserved_cidrs and existing_allocations. Platform-independent - carries no AWS/Azure/GCP-specific reserved addresses; " +
			"express any such reservation explicitly via reserved_cidrs.",
		Attributes: map[string]schema.Attribute{
			"base_cidrs": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Base CIDR blocks to allocate from, all the same address family. Multiple entries are tried in the given order.",
			},
			"allocation_strategy": schema.StringAttribute{
				Optional:    true,
				Description: "\"first_fit\" (default), \"best_fit\", \"worst_fit\", or \"sequential\". See the provider README for the exact behavior of each.",
			},
			"reserved_cidrs": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "CIDR blocks within base_cidrs that must never be allocated.",
			},
			"allocations": schema.MapAttribute{
				Required: true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"count":          types.Int64Type,
					"required_hosts": types.Int64Type,
				}},
				Description: "One entry per named allocation group. count instances of that group are each allocated a block sized for required_hosts usable addresses.",
			},
			"existing_allocations": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Previously allocated CIDRs to preserve, keyed \"<group>-<index>\" (zero-based) - the same keys produced in results. " +
					"A key present here is never reallocated; its capacity is only checked against required_hosts and reported via results[key].status.",
			},
			"results": schema.MapAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: subnetPlanResultAttributeTypes,
				},
				Description: "Effective outcome per expanded allocation instance, keyed \"<group>-<index>\". status is one of " +
					"\"allocated\" (newly carved), \"preserved\" (kept from existing_allocations), \"insufficient\" (kept from " +
					"existing_allocations despite being too small for required_hosts), or \"failed\" (no free space big enough - cidr is empty).",
			},
		},
	}
}

func (d *subnetPlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config subnetPlanConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	strategy := config.AllocationStrategy.ValueString()

	specs := make(map[string]netutil.AllocationSpec, len(config.Allocations))
	for name, a := range config.Allocations {
		specs[name] = netutil.AllocationSpec{Count: a.Count, RequiredHosts: a.RequiredHosts}
	}

	results, err := netutil.PlanSubnets(config.BaseCIDRs, strategy, config.ReservedCIDRs, specs, config.ExistingAllocations)
	if err != nil {
		resp.Diagnostics.AddError("Unable to plan subnet allocation", err.Error())
		return
	}

	resultValues := make(map[string]attr.Value, len(results))
	for key, r := range results {
		obj, diags := types.ObjectValue(subnetPlanResultAttributeTypes, map[string]attr.Value{
			"name":            types.StringValue(r.Name),
			"cidr":            types.StringValue(r.CIDR),
			"prefix_length":   int64Or(r.Status != "failed", int64(r.PrefixLength)),
			"address_family":  stringOr(r.Status != "failed", r.AddressFamily),
			"total_capacity":  stringOr(r.Status != "failed", r.TotalCapacity),
			"usable_capacity": stringOr(r.Status != "failed", r.UsableCapacity),
			"index":           types.Int64Value(r.Index),
			"base_cidr":       types.StringValue(r.BaseCIDR),
			"status":          types.StringValue(r.Status),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resultValues[key] = obj
	}

	resultsMap, diags := types.MapValue(types.ObjectType{AttrTypes: subnetPlanResultAttributeTypes}, resultValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Results = resultsMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func int64Or(ok bool, v int64) types.Int64 {
	if !ok {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}

func stringOr(ok bool, v string) types.String {
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(v)
}
