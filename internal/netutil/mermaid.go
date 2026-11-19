package netutil

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// MermaidEntity is one node or container in a topology fragment, as
// described by docs/topology-fragment-spec.md. Whether it renders as a
// Mermaid subgraph or a plain node is derived from the merged set (does
// anything reference it as a Parent), never declared up front.
type MermaidEntity struct {
	ID     string
	Label  string
	Kind   string // free-form classification/style hint - never used for joining
	Parent string // "" = top-level (no parent)
}

// MermaidEdge is a directed connection between two entity ids. From/To may
// reference an id no fragment declares as an entity - it still renders, as
// a plain fallback node.
type MermaidEdge struct {
	From  string
	To    string
	Kind  string // relation type hint, independent of Label
	Label string
}

// MermaidFragment is one module's slice of a larger topology.
type MermaidFragment struct {
	Entities []MermaidEntity
	Edges    []MermaidEdge
}

// MermaidMerge combines fragments from any number of independently-authored
// modules into one complete Mermaid flowchart. Fragments later in the list
// win, wholesale, over an earlier one declaring the same entity id.
// Deterministic: the same input in the same order always produces
// byte-identical output, regardless of Go map iteration order internally.
func MermaidMerge(fragments []MermaidFragment) (string, error) {
	entities := map[string]MermaidEntity{}
	for _, f := range fragments {
		for _, e := range f.Entities {
			entities[e.ID] = e
		}
	}

	var edges []MermaidEdge
	for _, f := range fragments {
		edges = append(edges, f.Edges...)
	}

	ensure := func(id string) {
		if id == "" {
			return
		}
		if _, ok := entities[id]; !ok {
			entities[id] = MermaidEntity{ID: id, Label: id}
		}
	}

	// Snapshot parents before mutating the map so newly-synthesized
	// fallback entities (which never have a Parent) are never themselves
	// walked for this pass.
	pendingParents := make([]string, 0, len(entities))
	for _, e := range entities {
		pendingParents = append(pendingParents, e.Parent)
	}
	for _, p := range pendingParents {
		ensure(p)
	}
	for _, edge := range edges {
		ensure(edge.From)
		ensure(edge.To)
	}

	if err := mermaidDetectCycle(entities); err != nil {
		return "", err
	}

	children := map[string][]string{}
	var roots []string
	for id, e := range entities {
		if e.Parent == "" {
			roots = append(roots, id)
			continue
		}
		children[e.Parent] = append(children[e.Parent], id)
	}
	sort.Strings(roots)
	for parent := range children {
		sort.Strings(children[parent])
	}

	tokens := mermaidAssignTokens(entities)

	var b strings.Builder
	b.WriteString("graph TD\n")
	for _, id := range roots {
		mermaidRenderEntity(&b, id, entities, children, tokens, 1)
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Label < edges[j].Label
	})
	for _, edge := range edges {
		mermaidRenderEdge(&b, edge, tokens)
	}

	mermaidRenderClasses(&b, entities, tokens)

	return strings.TrimRight(b.String(), "\n"), nil
}

// mermaidDetectCycle walks every entity's Parent chain and fails on the
// first cycle found, instead of letting the renderer recurse forever.
func mermaidDetectCycle(entities map[string]MermaidEntity) error {
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[string]int, len(entities))

	var walk func(id string, path []string) error
	walk = func(id string, path []string) error {
		switch state[id] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("parent cycle detected: %s -> %s", strings.Join(path, " -> "), id)
		}
		state[id] = visiting
		if p := entities[id].Parent; p != "" {
			if err := walk(p, append(path, id)); err != nil {
				return err
			}
		}
		state[id] = done
		return nil
	}

	ids := make([]string, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := walk(id, nil); err != nil {
			return err
		}
	}
	return nil
}

// mermaidAssignTokens maps each entity id to a Mermaid-safe node token.
// Tokens are opaque hashes of the id, never a character-substituted
// version of it: an id is routinely a full ARN (which embeds the AWS
// account id), and mermaid_merge's output is meant to be pasted into a
// README and committed - the id must never appear, in any form, anywhere
// in the rendered text. The human-readable name belongs in Label, a
// separate field the caller controls explicitly.
func mermaidAssignTokens(entities map[string]MermaidEntity) map[string]string {
	ids := make([]string, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	tokens := make(map[string]string, len(ids))
	seen := map[string]int{}
	for _, id := range ids {
		base := mermaidNodeToken(id)
		token := base
		// Defensive fallback only: a 64-bit hash collision between two
		// distinct real ids is not realistically constructible, so this
		// branch is not exercised by this package's tests.
		if n, ok := seen[base]; ok {
			n++
			seen[base] = n
			token = fmt.Sprintf("%s_%d", base, n)
		} else {
			seen[base] = 1
		}
		tokens[id] = token
	}
	return tokens
}

// mermaidNodeToken opaquely derives a Mermaid-safe node id from an entity
// id via a non-cryptographic hash - deterministic (same id always maps to
// the same token) but one-way, so none of the original id's characters,
// including an embedded AWS account id, appear in the output.
func mermaidNodeToken(id string) string {
	h := fnv.New64a()
	h.Write([]byte(id))
	return fmt.Sprintf("n%x", h.Sum64())
}

// mermaidSanitizeToken makes a caller-chosen, non-sensitive string (a
// `kind`, used as a Mermaid class name) safe as an identifier. Unlike
// mermaidNodeToken, this one preserves readability on purpose - a
// consuming doc has to type a matching `classDef <name>` by hand, which
// only works if the name is predictable from the `kind` string it came
// from.
func mermaidSanitizeToken(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	token := b.String()
	if token == "" {
		token = "n"
	}
	if token[0] >= '0' && token[0] <= '9' {
		token = "n_" + token
	}
	return token
}

func mermaidSanitizeLabel(label string) string {
	label = strings.ReplaceAll(label, "\"", "#quot;")
	label = strings.ReplaceAll(label, "\n", " ")
	return label
}

func mermaidRenderEntity(b *strings.Builder, id string, entities map[string]MermaidEntity, children map[string][]string, tokens map[string]string, depth int) {
	indent := strings.Repeat("  ", depth)
	e := entities[id]
	label := mermaidSanitizeLabel(e.Label)
	token := tokens[id]

	if kids := children[id]; len(kids) > 0 {
		fmt.Fprintf(b, "%ssubgraph %s[\"%s\"]\n", indent, token, label)
		for _, kid := range kids {
			mermaidRenderEntity(b, kid, entities, children, tokens, depth+1)
		}
		fmt.Fprintf(b, "%send\n", indent)
		return
	}

	fmt.Fprintf(b, "%s%s[\"%s\"]\n", indent, token, label)
}

func mermaidRenderEdge(b *strings.Builder, edge MermaidEdge, tokens map[string]string) {
	from := tokens[edge.From]
	to := tokens[edge.To]

	label := edge.Label
	if label == "" {
		label = edge.Kind
	}
	if label == "" {
		fmt.Fprintf(b, "%s --> %s\n", from, to)
		return
	}
	fmt.Fprintf(b, "%s -->|%s| %s\n", from, mermaidSanitizeLabel(label), to)
}

// mermaidRenderClasses groups entities by Kind into Mermaid `class`
// statements, so a consumer can style them by defining matching `classDef`
// rules separately. No colors/styles are invented here.
func mermaidRenderClasses(b *strings.Builder, entities map[string]MermaidEntity, tokens map[string]string) {
	// Iterate ids in sorted order and preserve that order per kind -
	// tokens are opaque hashes now, so sorting by token would scramble
	// the grouping into a meaningless order.
	ids := make([]string, 0, len(entities))
	for id := range entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	byKind := map[string][]string{}
	var kinds []string
	seenKind := map[string]bool{}
	for _, id := range ids {
		kind := entities[id].Kind
		if kind == "" {
			continue
		}
		byKind[kind] = append(byKind[kind], tokens[id])
		if !seenKind[kind] {
			seenKind[kind] = true
			kinds = append(kinds, kind)
		}
	}
	sort.Strings(kinds)

	for _, kind := range kinds {
		fmt.Fprintf(b, "class %s %s\n", strings.Join(byKind[kind], ","), mermaidSanitizeToken(kind))
	}
}
