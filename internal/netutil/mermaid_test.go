package netutil

import (
	"fmt"
	"strings"
	"testing"
)

func tok(id string) string {
	return mermaidNodeToken(id)
}

func TestMermaidMergeNestingAcrossFragments(t *testing.T) {
	// Mirrors the vpc + subnets + nat-gateway worked example from
	// docs/topology-fragment-spec.md: three independently-authored
	// fragments, joined purely by real AWS ids. Edge direction follows
	// that spec's "Edge direction" rule - egress flows from the
	// subnet/resource toward the internet, never the reverse.
	vpc := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "internet", Label: "Internet", Kind: "internet"},
			{ID: "vpc-123", Label: "main", Kind: "vpc"},
			{ID: "igw-abc", Label: "main-igw", Kind: "internet-gateway", Parent: "vpc-123"},
		},
		Edges: []MermaidEdge{
			{From: "igw-abc", To: "internet"},
		},
	}
	subnets := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "az:eu-central-1a", Label: "eu-central-1a", Kind: "az", Parent: "vpc-123"},
			{ID: "subnet-pub", Label: "public-a", Kind: "subnet", Parent: "az:eu-central-1a"},
			{ID: "subnet-priv", Label: "private-a", Kind: "subnet", Parent: "az:eu-central-1a"},
		},
		Edges: []MermaidEdge{
			{From: "subnet-pub", To: "igw-abc"},
			{From: "subnet-priv", To: "nat-xyz"},
		},
	}
	nat := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "nat-xyz", Label: "nat-a", Kind: "nat-gateway", Parent: "subnet-pub"},
		},
		Edges: []MermaidEdge{
			{From: "nat-xyz", To: "igw-abc"},
		},
	}

	got, err := MermaidMerge([]MermaidFragment{vpc, subnets, nat})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}

	// subnet-pub must render as a subgraph (it gained a child, nat-xyz,
	// only once the third fragment was merged in) even though the
	// subnets fragment alone declared it as a leaf.
	if !strings.Contains(got, fmt.Sprintf("subgraph %s[\"public-a\"]", tok("subnet-pub"))) {
		t.Errorf("expected subnet-pub to render as a subgraph once nat-xyz nests inside it, got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s[\"nat-a\"]", tok("nat-xyz"))) {
		t.Errorf("expected nat-xyz as a leaf node, got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s --> %s", tok("igw-abc"), tok("internet"))) {
		t.Errorf("expected igw -> internet edge (gateway is the egress source), got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s --> %s", tok("subnet-priv"), tok("nat-xyz"))) {
		t.Errorf("expected private subnet -> nat edge (subnet is the egress source), got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s --> %s", tok("nat-xyz"), tok("igw-abc"))) {
		t.Errorf("expected nat -> igw edge completing the 3-hop chain, got:\n%s", got)
	}

	// The whole point of hashing: none of the raw ids (which may be ARNs
	// embedding an AWS account id) may appear verbatim anywhere.
	// "internet" is excluded here on purpose - it is also this
	// fixture's `Kind` value, and a Kind is deliberately kept readable
	// in the rendered `class` line; that is not a leak, just a
	// coincidence of this test's fixture reusing the same word for both.
	for _, raw := range []string{"vpc-123", "igw-abc", "subnet-pub", "subnet-priv", "nat-xyz"} {
		if strings.Contains(got, raw) {
			t.Errorf("raw id %q leaked into rendered output verbatim:\n%s", raw, got)
		}
	}
}

func TestMermaidMergeCollisionLastWins(t *testing.T) {
	first := MermaidFragment{Entities: []MermaidEntity{{ID: "vpc-1", Label: "first-label", Kind: "vpc"}}}
	second := MermaidFragment{Entities: []MermaidEntity{{ID: "vpc-1", Label: "second-label", Kind: "vpc"}}}

	got, err := MermaidMerge([]MermaidFragment{first, second})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if strings.Contains(got, "first-label") {
		t.Errorf("expected the later fragment's label to win, got:\n%s", got)
	}
	if !strings.Contains(got, "second-label") {
		t.Errorf("expected the later fragment's label to win, got:\n%s", got)
	}
}

func TestMermaidMergeDanglingEdgeEndpoint(t *testing.T) {
	// An edge may reference an id no fragment ever declares as an
	// entity - it must still render as a fallback node, not error out.
	frag := MermaidFragment{
		Entities: []MermaidEntity{{ID: "subnet-a", Label: "a"}},
		Edges:    []MermaidEdge{{From: "nat-undeclared", To: "subnet-a"}},
	}

	got, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if !strings.Contains(got, fmt.Sprintf("%s[\"nat-undeclared\"]", tok("nat-undeclared"))) {
		t.Errorf("expected a fallback node for the undeclared edge endpoint, got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s --> %s", tok("nat-undeclared"), tok("subnet-a"))) {
		t.Errorf("expected the edge to still render, got:\n%s", got)
	}
}

func TestMermaidMergeDanglingParent(t *testing.T) {
	// parent may reference an id no fragment declares - it must render
	// as an (unlabeled-by-anyone-else) subgraph, not crash.
	frag := MermaidFragment{
		Entities: []MermaidEntity{{ID: "nat-a", Label: "nat-a", Parent: "subnet-undeclared"}},
	}

	got, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if !strings.Contains(got, fmt.Sprintf("subgraph %s[\"subnet-undeclared\"]", tok("subnet-undeclared"))) {
		t.Errorf("expected a fallback subgraph for the undeclared parent, got:\n%s", got)
	}
}

func TestMermaidMergeCycleIsAnError(t *testing.T) {
	frag := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "a", Label: "a", Parent: "b"},
			{ID: "b", Label: "b", Parent: "a"},
		},
	}

	if _, err := MermaidMerge([]MermaidFragment{frag}); err == nil {
		t.Error("expected a parent cycle to be reported as an error")
	}
}

func TestMermaidMergeDistinctIdsGetDistinctTokens(t *testing.T) {
	// Two different ids must never collapse onto the same node - this
	// used to be a real risk under character-substitution (".", "-" both
	// became "_"), which is exactly why node tokens are hashes now.
	frag := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "a.b", Label: "dotted"},
			{ID: "a-b", Label: "hyphenated"},
		},
	}

	got, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if tok("a.b") == tok("a-b") {
		t.Fatalf("test invariant broken: a.b and a-b hashed to the same token")
	}
	if !strings.Contains(got, fmt.Sprintf("%s[\"dotted\"]", tok("a.b"))) {
		t.Errorf("expected a.b to render under its own token, got:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("%s[\"hyphenated\"]", tok("a-b"))) {
		t.Errorf("expected a-b to render under its own token, got:\n%s", got)
	}
}

func TestMermaidMergeEdgeLabelFallsBackToKind(t *testing.T) {
	frag := MermaidFragment{
		Entities: []MermaidEntity{{ID: "asg-1", Label: "asg"}, {ID: "subnet-a", Label: "a"}},
		Edges:    []MermaidEdge{{From: "asg-1", To: "subnet-a", Kind: "deployed-in"}},
	}

	got, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if !strings.Contains(got, fmt.Sprintf("%s -->|deployed-in| %s", tok("asg-1"), tok("subnet-a"))) {
		t.Errorf("expected Kind to be used as the display label when Label is unset, got:\n%s", got)
	}
}

func TestMermaidMergeClassesByKind(t *testing.T) {
	frag := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "subnet-a", Label: "a", Kind: "subnet"},
			{ID: "subnet-b", Label: "b", Kind: "subnet"},
		},
	}

	got, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	// "subnet-a" < "subnet-b" by original id, which must be the order
	// the class statement lists them in - token values themselves are
	// opaque hashes with no meaningful order of their own.
	want := fmt.Sprintf("class %s,%s subnet", tok("subnet-a"), tok("subnet-b"))
	if !strings.Contains(got, want) {
		t.Errorf("expected %q, got:\n%s", want, got)
	}
}

func TestMermaidMergeDeterministic(t *testing.T) {
	frag := MermaidFragment{
		Entities: []MermaidEntity{
			{ID: "vpc-1", Label: "vpc"},
			{ID: "subnet-b", Label: "b", Parent: "vpc-1"},
			{ID: "subnet-a", Label: "a", Parent: "vpc-1"},
		},
	}

	first, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	second, err := MermaidMerge([]MermaidFragment{frag})
	if err != nil {
		t.Fatalf("MermaidMerge returned error: %v", err)
	}
	if first != second {
		t.Errorf("expected identical output across repeated calls, got:\n%s\n---\n%s", first, second)
	}
}

func TestMermaidNodeTokenNeverContainsAccountID(t *testing.T) {
	arn := "arn:aws:autoscaling:eu-central-1:123456789012:autoScalingGroup:a1b2c3d4-5678-90ab-cdef-example11111:autoScalingGroupName/app-cluster-default"
	token := mermaidNodeToken(arn)
	if strings.Contains(token, "123456789012") {
		t.Errorf("node token leaked the account id: %s", token)
	}
	if strings.Contains(token, "eu-central-1") || strings.Contains(token, "app-cluster-default") {
		t.Errorf("node token leaked part of the source ARN: %s", token)
	}
}
