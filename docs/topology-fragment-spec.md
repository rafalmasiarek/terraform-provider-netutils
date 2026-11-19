# Topology fragment specification

Status: `netutils_mermaid_merge` (the Go function this spec was written
for) is implemented - see `internal/netutil/mermaid.go` and
`internal/provider/function_mermaid_merge.go` in this repository, and the
"mermaid_merge" section of the main README. No module in the
`terraform-aws-*` family emits `topology_fragment` yet; the rollout plan at
the bottom of this document still applies.

## Motivation

Several modules in the `terraform-aws-*` / `terraform-provider-netutils`
family (`terraform-aws-vpc`, `terraform-aws-subnets`, and future modules such
as a NAT gateway/instance module or `terraform-aws-ecs`) each manage one
"layer" of the same infrastructure. `terraform-aws-subnets` already renders
a standalone Mermaid diagram of its own layer (`output "topology_mermaid"`),
but that output is a fully-rendered, self-contained diagram text — it cannot
be combined with a diagram rendered the same way by a sibling module without
colliding headers, duplicated subgraphs, and un-namespaced node ids.

This spec defines a structured, composable contract — `topology_fragment` —
that any module can emit to describe its own slice of the topology, plus a
merge function (`netutils_mermaid_merge`, to be added to
`terraform-provider-netutils`) that combines any number of fragments from any
number of unrelated modules into one Mermaid diagram.

A module adopting this contract does not need to know anything about any
other module in the ecosystem. The only coordination mechanism is: reference
another entity by an id it may or may not have declared itself.

## Prior art

This shape is not invented from scratch — it mirrors patterns already proven
for "independently-authored fragments stitched into one graph without a
central registry":

- **Terraform itself** resolves a single resource dependency graph from
  declarations spread across modules, joined by resource address.
- **Kubernetes `ownerReferences`** — an object declares `{uid, kind, name}`
  pointing at another, independently-created object; a controller lazily
  builds the ownership graph from `uid`, which is globally unique by
  construction.
- **OpenTelemetry spans / AWS X-Ray** — each service independently emits
  spans/segments carrying `trace_id` + `span_id` + `parent_span_id`; a
  collector stitches one trace/service-map graph, tolerating a child that
  arrives before its parent.
- **Backstage Software Catalog** `spec.dependsOn` — entities defined in
  files scattered across many repositories reference each other by a typed
  `kind:name` string; a processor resolves the graph and computes the
  inverse relation.

Common pattern: a stable, globally unique id as the join key; a one-directional
declaration of a relationship (no two-sided coordination at author time); and
a separate stitching step that tolerates forward/missing references. This
spec follows the same shape.

## Schema

Every module that wants to participate emits one output:

```hcl
output "topology_fragment" {
  value = {
    entities = [
      {
        id     = string            # see "Identifiers" below
        label  = string            # human-readable display text
        kind   = string            # free-form classification/style hint - never used for joining
        parent = optional(string)  # id of the containing entity; omitted = top-level
      },
      # ...
    ]
    edges = [
      {
        from  = string
        to    = string
        kind  = optional(string)  # relation type hint for styling, e.g. "routes-to", "depends-on", "peers-with" - independent of label
        label = optional(string)  # free display text
      },
      # ...
    ]
  }
}
```

`optional()` here is the native Terraform object-type optional attribute
(statically declared), not an ad-hoc map with varying keys — it does not run
into the "inconsistent conditional result types" bug class this ecosystem
has hit before with `taggable`-style maps assembled from mismatched ternary
branches.

### Identifiers

- If an entity corresponds to a real AWS resource, `id` **is** that
  resource's own id or ARN, exactly as AWS assigned it (e.g. `aws_subnet.this[k].id`,
  `aws_nat_gateway.this.id`). Never invent a prefix or alias for it — the
  real id is already globally unique, which is what makes cross-module
  joining work without any naming convention to agree on.
- An ARN is a fine choice of `id` - prefer it over a bare name whenever a
  fragment might one day be merged alongside another account's or
  region's fragments, since an ARN (unlike a short name such as an Auto
  Scaling Group's) is globally unique across accounts and regions, not
  just within one. An ARN embeds the AWS account id
  (e.g. `arn:aws:autoscaling:eu-central-1:123456789012:...`), but that
  never reaches the rendered diagram: `mermaid_merge` never writes an
  `id` into its output in any form (see "Node tokens are opaque hashes,
  never the id itself" below) - only the caller-supplied `label` is ever
  shown, and the caller controls that independently.
- If an entity is a synthetic concept with no AWS-issued id (a VPC-level
  grouping is fine to key on the VPC's own id, but something like an
  availability-zone grouping or a non-resource concept is not), use a
  manually namespaced id of the form `"<kind>:<value>"`, e.g. `"az:eu-central-1a"`.
- `"internet"` is a **reserved, well-known synthetic id** for the public
  internet as a concept. Any module that creates an internet-facing gateway
  (IGW, EIGW, ...) may emit it. Multiple fragments emitting the identical
  `"internet"` entity is expected and harmless (see merge semantics).

### Node tokens are opaque hashes, never the id itself

`mermaid_merge` never writes an `id` into the rendered diagram text in any
form - not literally, not character-substituted. Every Mermaid node needs
a syntax-safe token (no hyphens, dots, or colons), and the naive way to
get one - replacing unsafe characters with `_` - would still print every
character of the original id, account id included if it came from an ARN.
Instead, each node's token is a one-way hash of its id
(`fmt.Sprintf("n%x", fnv64a(id))`): deterministic, so the same id always
gets the same token, but nothing about the id's content survives into the
token. The only human-readable text in the output is whatever the caller
put in `label` - a field it fully controls, independent of `id`.

This is why the previous section can recommend an ARN without reservation:
there is no longer a tradeoff between join-key strength and what ends up
visible in a committed README.

### Entities: leaf vs. container is a derived property, not an authored choice

A module never decides "I am a group" or "I am a node." Whether an entity
renders as a Mermaid `subgraph` or a plain node depends only on whether *any*
fragment (from any module) declares another entity with `parent` pointing at
it. An entity with at least one child renders as a subgraph; with none, it
renders as a plain node. This is what allows the tree to grow arbitrarily
deep as new module types are added (VPC > AZ > subnet > NAT, or VPC > AZ >
subnet > ECS service > ECS task) without any schema change and without any
module needing to know in advance how deeply it will be nested.

### Ownership rule

A module declares entities/edges only for resources it itself creates. A
`parent` (or an edge `from`/`to`) may reference an id owned by a different
fragment — or by no fragment at all (see below) — without declaring that id
itself.

### Dangling references

An edge or a `parent` may reference an `id` that no fragment ever declares as
an entity. This is valid. Mermaid renders an undeclared referenced id as a
plain default node. Do not treat this as an error in the merge function.

### Not a DAG

`parent` forms a tree, not a general graph — a Mermaid subgraph cannot be
nested inside two different containers at once. A resource genuinely shared
across two containers attaches to their lowest common ancestor and expresses
the sharing via `edges`, never via a second `parent`.

### Singular vs. plural attachment

A recurring shape: a resource that *can* attach to either one container or
several, depending on how the caller configured it (an Auto Scaling Group
spanning one subnet or many, an ALB/NLB spanning multiple AZs, an RDS
multi-AZ subnet group). The rule is not "always nest" or "always go to a
common ancestor" — it is conditional on the actual count at apply time:

- Exactly one container → nest directly (`parent` = that container's id,
  no edges needed).
- More than one → `parent` = the lowest common ancestor the module actually
  knows (frequently the VPC), and emit an edge to each container instead.
- If the module does not know a common ancestor id at all (see the ASG
  example below, which has no `vpc_id` input by design), leave `parent`
  unset entirely rather than inventing one. The entity floats at the
  top level of the merged graph; it is still correctly connected via its
  edges, just not visually nested.

This is the same rule applied once to subnets/NAT/EC2 (always singular) and
once to anything that can legitimately be plural — it does not need a
different mechanism per resource type.

### What NOT to model as an entity

Only resources that are either a node on a network/traffic path, or a
genuine logical container for other entities, belong in `topology_fragment`.
Config/compliance-only resources a module happens to create — IAM roles,
CloudWatch log groups for flow logs, DHCP option sets, default security
group lockdown — are deliberately excluded. Including them would litter the
diagram with isolated boxes with no edges, defeating the purpose.

The same rule excludes an ECS cluster. It has no ENI, sits on no network
path, and is never any other entity's `parent` — it is a pure control-plane
construct, same category as the IAM-role/flow-log exclusions above, even
though it is easy to reach for it as "the thing the ASG belongs to." The
ASG itself stays: it has real ENIs in real subnets, which is exactly what
this diagram exists to show.

### Edge direction

`edges` encode this diagram's one purpose - the egress/routing topology,
not an inbound request path - so direction follows actual egress traffic:
**from the resource/subnet, toward its gateway, toward the internet.**
`"internet"` is the sink every such chain points at, never the source.

- Public subnet → its IGW directly: `{ from = subnet_id, to = igw_id }`.
- Private subnet → its NAT, never directly to an IGW: `{ from = subnet_id, to = nat_id }`.
  A private subnet's route edge landing on an IGW id is invalid - it
  describes a network path AWS does not allow.
- A gateway → the next hop toward the internet: `{ from = igw_id, to = "internet" }`,
  `{ from = nat_id, to = igw_id }`.

This is the opposite direction from a request-path diagram (where an
external client is the source: `internet --> igw --> alb`) - that is a
different diagram answering a different question, and is out of scope
here. Mixing the two conventions in one diagram is what produced a visibly
wrong first draft of this spec's own worked examples (gateways pointing
down into subnets, `"internet"` as a source instead of a sink) - corrected
below. Source: the egress-path and validation conventions in
[latixengine.com's Mermaid/AWS series](https://latixengine.com/blog/vpc-architecture-diagram-mermaid/).

This rule governs egress/routing edges specifically - anything whose `kind`
describes a routing relationship. A non-routing relationship (e.g. which
subnets an Auto Scaling Group spans) is not egress traffic and is not
bound by it; pick whatever direction reads naturally for that relationship
and say so via `kind`, e.g. `{ kind = "deployed-in" }` pointing from the
spanning resource to each subnet it occupies. See the
`terraform-aws-ec2-autoscale-group` and `terraform-aws-ecs` examples below.

## Merge semantics (`netutils_mermaid_merge`)

To be implemented as a new function in `terraform-provider-netutils`,
following the existing pattern of that provider's functions: deterministic,
local, side-effect free, no external dependency added for this (hand-rolled
string building on the Go standard library — see rationale below).

```hcl
netutils_mermaid_merge(fragments: list(object({ entities = ..., edges = ... })))
  -> string   # a complete Mermaid diagram, ready to paste into a fenced ```mermaid block
```

Rules:

1. **Collision policy**: if two fragments declare an entity with the same
   `id`, the entity from the fragment **later in the input list wins,
   wholesale** (the entire entity record is replaced, not field-merged).
   This mirrors the existing `tagger_additional_services` override
   convention in this ecosystem — the caller controls precedence by the
   order it assembles the fragment list. The `"internet"` case is harmless
   under this rule since colliding declarations are identical in practice.
2. **Tree construction**: after dedup, build a parent→children map from
   every entity's `parent` field across the merged set.
3. **Render rule**: an entity with ≥1 child renders as
   `subgraph <id>[<label>] ... end`; an entity with 0 children renders as a
   plain node `<id>[<label>]`. This is evaluated after all fragments are
   merged, never per-fragment.
4. **Determinism**: iterate entities/edges in a stable sort order (e.g. by
   `id`) so the rendered text does not vary run-to-run for the same input -
   consistent with every other function in this provider, and important
   here specifically because this output lands in Terraform state/plan
   diffs.
5. **Styling hook**: `kind` (on entities) and `kind` (on edges, independent
   meaning — relation type, not entity type) are available to the renderer
   for `classDef`/`class` assignment or arrow styling. Not used for
   joining or tree construction.

### Render target (default vs. future modes)

Default output is Mermaid `graph TD` with nested `subgraph ... end` blocks
and `classDef`/`class` styling by `kind` — no icon packs, no diagram type
beyond what GitHub's bundled Mermaid renders natively in a README preview.
This was confirmed deliberately: GitHub does not render `architecture-beta`
reliably, and AWS-branded icons (via `icon: "aws:..."` or an iconify pack
like `logos:aws-lambda`) require the *host* renderer to have that icon pack
registered, which GitHub's README preview does not do. Mermaid Chart's own
icon registry (`mermaid.ai`/`icon: "aws:..."` syntax) is specific to that
hosted product and is not part of open-source mermaid.js.

An `architecture-beta`-based render mode (which maps the same
`entities`/`edges` model onto `group`/`service`/`in`/edge syntax almost 1:1)
may be added later as an explicit opt-in, for contexts where the consuming
renderer is known to support it and has an icon pack registered. It is
**not** the default and is out of scope for the first implementation.

### Why hand-rolled, not a third-party Go library

`terraform-provider-netutils`'s only direct dependency today is
`terraform-plugin-framework`; everything else in `go.mod` is transitive.
Existing Go Mermaid libraries (e.g. `go-mermaid`, `mermaidgen`) only help
with the last step — emitting syntactically valid Mermaid text — which the
Go standard library (`strings.Builder`/`fmt.Sprintf`) handles trivially for
the narrow, self-controlled subset of syntax this spec needs. None of them
solve the actual hard part (dedup, tree construction from `parent` refs),
which has to be written against this exact schema regardless. Adding a
dependency for the easy 10% was judged not worth it.

## Worked examples

### `terraform-aws-vpc`

```hcl
locals {
  topology_fragment = {
    entities = concat(
      [{ id = "internet", label = "Internet", kind = "internet", parent = null }],
      [{ id = local.vpc_id, label = var.name, kind = "vpc", parent = null }],
      !var.ipv6_only ? [{
        id     = aws_internet_gateway.this[0].id
        label  = "${var.name}-igw"
        kind   = "internet-gateway"
        parent = local.vpc_id
      }] : [],
      (var.enable_ipv6 || var.ipv6_only) ? [{
        id     = aws_egress_only_internet_gateway.this[0].id
        label  = "${var.name}-eigw"
        kind   = "egress-only-internet-gateway"
        parent = local.vpc_id
      }] : []
    )
    # Edges point from each gateway toward the internet, per the
    # "Edge direction" rule above - "internet" is a sink, never a source.
    edges = concat(
      !var.ipv6_only ? [{ from = aws_internet_gateway.this[0].id, to = "internet" }] : [],
      (var.enable_ipv6 || var.ipv6_only) ? [{ from = aws_egress_only_internet_gateway.this[0].id, to = "internet" }] : []
    )
  }
}
```

### `terraform-aws-subnets`

```hcl
locals {
  topology_fragment = {
    entities = concat(
      [for az in local._azs : { id = "az:${az}", label = az, kind = "az", parent = var.vpc_id }],
      [for k, s in aws_subnet.this : { id = s.id, label = k, kind = "subnet", parent = "az:${s.availability_zone}" }]
    )
    # Edges point from each subnet toward its own egress target - the
    # subnet is the traffic source, per the "Edge direction" rule above.
    edges = [
      for k, t in local.egress_target_by_subnet : { from = aws_subnet.this[k].id, to = t }
      if t != null
    ]
  }
}
```

Note the ownership split with `terraform-aws-vpc`: `vpc` draws
`gateway --> internet` edges (it creates the gateways); `subnets` draws only
`subnet --> gateway` edges (it creates the routing from the subnet). Neither
module needs to know the other exists.

### Future `terraform-aws-nat-gateway` (or `-nat-instance`)

A NAT gateway's own egress to the internet is automatic once an IGW exists
in the VPC - AWS routes it without any NAT-owned route table entry, so the
module needs no `igw_id` input to create the resource. Completing the
diagram's `nat --> igw` hop is purely a topology concern, so - exactly like
`topology_vpc_id` on the ASG example below - it is an explicit, *optional*,
topology-only input, never a required one:

```hcl
variable "topology_igw_id" {
  type        = string
  default     = null
  description = "Internet Gateway id, for the topology_fragment diagram only - omit to leave the nat -> igw edge undrawn."
}

locals {
  topology_fragment = {
    entities = [{
      id     = aws_nat_gateway.this.id   # or aws_instance.this.id for a NAT instance
      label  = var.name
      kind   = "nat-gateway"              # or "nat-instance"
      parent = var.subnet_id              # the real subnet id - already an entity declared by terraform-aws-subnets
    }]
    # "that private subnet -> this NAT" is drawn by terraform-aws-subnets,
    # not here - this module owns only its own onward hop toward the IGW.
    edges = var.topology_igw_id != null ? [{ from = aws_nat_gateway.this.id, to = var.topology_igw_id }] : []
  }
}
```

This module needs zero awareness of `terraform-aws-subnets`'s internals,
yet nests correctly under the right subnet purely because `var.subnet_id` is
the same real AWS id `terraform-aws-subnets` already declared as an entity.

### Future plain EC2 instance module

A module that creates a bare `aws_instance` (not through an Auto Scaling
Group) always attaches to exactly one subnet, so this is the simple,
singular case — identical shape to the NAT example above:

```hcl
locals {
  topology_fragment = {
    entities = [{
      id     = aws_instance.this.id
      label  = var.name
      kind   = "ec2-instance"
      parent = var.subnet_id
    }]
    edges = []
  }
}
```

### `terraform-aws-ec2-autoscale-group` (singular vs. plural attachment)

Checked the real module: it takes `subnet_ids` (a list — an ASG can span
one subnet or many) and deliberately has **no `vpc_id` input**, consistent
with its "zero AWS-service-specific knowledge" scope — AWS infers the VPC
from the subnets, so the module never needed one for resource creation.

Adding `topology_fragment` to it means applying the singular-vs-plural rule
above, and — since the module has no common-ancestor id to fall back on —
accepting that the "plural" branch leaves `parent` unset rather than
inventing a `vpc_id` input the module has no other reason to need:

```hcl
locals {
  topology_fragment = {
    entities = [{
      # .arn, not .name: globally unique across accounts/regions, not
      # just within one - safe to use because node tokens are hashed
      # (see "Node tokens are opaque hashes" above), so the account id
      # it carries never reaches the rendered diagram.
      id     = aws_autoscaling_group.this.arn
      label  = var.name
      kind   = "autoscaling-group"
      parent = length(var.subnet_ids) == 1 ? var.subnet_ids[0] : null
    }]
    edges = length(var.subnet_ids) > 1 ? [
      for subnet_id in var.subnet_ids :
      { from = aws_autoscaling_group.this.arn, to = subnet_id, kind = "deployed-in" }
    ] : []
  }
}
```

If a future caller wants the multi-subnet ASG visually nested inside its
VPC instead of floating at the top level, that requires an explicit,
*optional*, topology-only input (e.g. `var.topology_vpc_id`, default
`null`) — never a required one, and never derived implicitly inside the
module. This is the same "config comes from the caller" taste already
applied to every other variable in this module family.

Individual instances inside the ASG are not represented at all: ASG-managed
instances are not Terraform-tracked resources with a stable id across
scaling events, so there is no reliable per-instance node to draw. The
granularity for an ASG-backed fleet stops at the ASG itself.

### `terraform-aws-ecs` (node-group ASG only - the cluster itself is excluded)

Checked the real module: unlike the standalone ASG module, `terraform-aws-ecs`
already takes `var.vpc_id` as a required input (its security group needs
it), and each entry in `var.node_groups` already carries its own
`subnet_ids`. The node-group submodule already outputs
`autoscaling_group_arn`, which this fragment uses directly as `id` - per
the "Identifiers" rule above, an ARN is a fine choice; node tokens are
hashed, so its embedded account id never reaches the rendered diagram.
No new, topology-only variable is needed here at all - every id this
fragment needs already exists for other reasons.

The ECS cluster itself is **not** part of this fragment, per the "What NOT
to model as an entity" rule above - no ENI, no network path, nothing
nested under it. Only the node-group ASG is represented, following the
same singular-vs-plural rule as the standalone ASG example: spanning 3
public subnets here, `parent` is the VPC, with a `deployed-in` edge to
each subnet it spans.

```hcl
locals {
  node_group_entities = [
    for k, v in var.node_groups : {
      id     = module.node_group[k].autoscaling_group_arn
      label  = "${var.name}-${k}"
      kind   = "autoscaling-group"
      parent = length(v.subnet_ids) == 1 ? v.subnet_ids[0] : var.vpc_id
    }
  ]
  node_group_deployed_in_edges = flatten([
    for k, v in var.node_groups : length(v.subnet_ids) > 1 ? [
      for subnet_id in v.subnet_ids :
      { from = module.node_group[k].autoscaling_group_arn, to = subnet_id, kind = "deployed-in" }
    ] : []
  ])

  topology_fragment = {
    entities = local.node_group_entities
    edges    = local.node_group_deployed_in_edges
  }
}
```

Verified end-to-end against the real `netutils_mermaid_merge`: the ASG
floats at VPC level (a sibling of the AZ groups, not nested in any one of
them) and fans out three `deployed-in` edges to
`public-a`/`public-b`/`public-c` - while the egress topology from the
other fragments renders exactly as before, untouched.

## Open items / rollout plan

1. ~~Implement `netutils_mermaid_merge` in `terraform-provider-netutils`
   following this spec.~~ Done.
2. Refactor `terraform-aws-subnets`'s existing ad-hoc Mermaid locals
   (`_m`, `_egress_labels`, `topology_mermaid`) into `topology_fragment`
   per the example above. The current `output "topology_mermaid"` can stay
   for standalone use, or be dropped once composition is proven - decide at
   implementation time.
3. Add `topology_fragment` to `terraform-aws-vpc`.
4. Add `topology_fragment` to `terraform-aws-ec2-autoscale-group` and to
   `terraform-aws-ecs` (node-group ASG only, per the examples above -
   none of the four modules above have been touched yet; the examples
   are specced, not implemented).
5. Future modules not yet created (`terraform-aws-nat-gateway`/`-nat-instance`,
   a plain EC2 instance module, ...) adopt the contract from day one once
   they exist.
6. `architecture-beta` / AWS icon render mode: deferred, not scheduled.
