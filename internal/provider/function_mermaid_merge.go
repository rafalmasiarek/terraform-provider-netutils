package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/netutil"
)

var mermaidEntityAttributeTypes = map[string]attr.Type{
	"id":     types.StringType,
	"label":  types.StringType,
	"kind":   types.StringType,
	"parent": types.StringType,
}

var mermaidEdgeAttributeTypes = map[string]attr.Type{
	"from":  types.StringType,
	"to":    types.StringType,
	"kind":  types.StringType,
	"label": types.StringType,
}

var mermaidFragmentAttributeTypes = map[string]attr.Type{
	"entities": types.ListType{ElemType: types.ObjectType{AttrTypes: mermaidEntityAttributeTypes}},
	"edges":    types.ListType{ElemType: types.ObjectType{AttrTypes: mermaidEdgeAttributeTypes}},
}

type mermaidEntityModel struct {
	ID     string       `tfsdk:"id"`
	Label  string       `tfsdk:"label"`
	Kind   string       `tfsdk:"kind"`
	Parent types.String `tfsdk:"parent"`
}

type mermaidEdgeModel struct {
	From  string       `tfsdk:"from"`
	To    string       `tfsdk:"to"`
	Kind  types.String `tfsdk:"kind"`
	Label types.String `tfsdk:"label"`
}

type mermaidFragmentModel struct {
	Entities []mermaidEntityModel `tfsdk:"entities"`
	Edges    []mermaidEdgeModel   `tfsdk:"edges"`
}

type mermaidMergeFunction struct{}

func NewMermaidMergeFunction() function.Function {
	return &mermaidMergeFunction{}
}

func (f *mermaidMergeFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "mermaid_merge"
}

func (f *mermaidMergeFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Merges topology_fragment values from any number of independently-authored modules into one Mermaid flowchart.",
		Description: "Each fragment contributes entities (id, label, kind, optional parent) and edges (from, to, " +
			"optional kind/label) - see docs/topology-fragment-spec.md in this provider's repository for the full " +
			"contract. An entity referenced as another entity's parent renders as a Mermaid subgraph; one never " +
			"referenced as a parent renders as a plain node - this is derived from the merged set, never declared " +
			"by a single fragment. An edge or parent may reference an id no fragment declares; it still renders, " +
			"as a plain fallback node. When two fragments declare the same entity id, the one later in the list " +
			"wins, wholesale. Deterministic: the same input in the same order always produces the same output.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "fragments",
				ElementType: types.ObjectType{AttrTypes: mermaidFragmentAttributeTypes},
				Description: "topology_fragment output values, in override precedence order (last wins on an id collision).",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *mermaidMergeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var fragments []mermaidFragmentModel
	resp.Error = req.Arguments.Get(ctx, &fragments)
	if resp.Error != nil {
		return
	}

	converted := make([]netutil.MermaidFragment, len(fragments))
	for i, fr := range fragments {
		entities := make([]netutil.MermaidEntity, len(fr.Entities))
		for j, e := range fr.Entities {
			entities[j] = netutil.MermaidEntity{
				ID:     e.ID,
				Label:  e.Label,
				Kind:   e.Kind,
				Parent: e.Parent.ValueString(),
			}
		}
		edges := make([]netutil.MermaidEdge, len(fr.Edges))
		for j, e := range fr.Edges {
			edges[j] = netutil.MermaidEdge{
				From:  e.From,
				To:    e.To,
				Kind:  e.Kind.ValueString(),
				Label: e.Label.ValueString(),
			}
		}
		converted[i] = netutil.MermaidFragment{Entities: entities, Edges: edges}
	}

	diagram, err := netutil.MermaidMerge(converted)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("merging topology fragments: %s", err))
		return
	}

	resp.Error = resp.Result.Set(ctx, types.StringValue(diagram))
}
