# terraform-provider-netutils

Platform-agnostic networking utilities for Terraform: IP/CIDR operations,
subnet planning, DNS/hostname tooling, network diagnostics. No dependency
on any other Terraform provider.

## Provider configuration

```hcl
provider "netutils" {}
```

| Name | Type | Default | Description |
|---|---|---|---|
| `ip_lookup_endpoint` | `string` | `https://masiarek.pl/api/v2/ip/lookup` | Backend for `netutils_ip_info`. Falls back to the `NETUTILS_IP_LOOKUP_ENDPOINT` env var, then this default. |

## Functions

All functions are deterministic, local, and side-effect free - no network
calls, no state. Mixed address families (an IPv4 argument against an IPv6
one) return `false`/`null` rather than an error, for every function
taking two network-shaped arguments.

| Function | Parameters | Returns | Description |
|---|---|---|---|
| `ip_normalize` | `ip` string | string | Canonical form - RFC 5952 compressed/lowercase for IPv6. |
| `ip_classify` | `ip` string | string | One of: `unspecified`, `loopback`, `multicast`, `link-local`, `cgnat`, `documentation`, `benchmarking`, `private`, `public`, `reserved`. See precedence below. |
| `ip_family` | `ip` string | string | `"ipv4"` or `"ipv6"`. Fails (via `can()`) for an invalid address. |
| `ip_in_cidr` | `ip` string, `cidr` string | bool | Whether `ip` falls within `cidr`. |
| `cidr_family` | `cidr` string | string | `"ipv4"` or `"ipv6"`. Fails for an invalid CIDR, including a bare IP with no prefix length. |
| `cidr_contains` | `parent_cidr` string, `cidr` string | bool | Whether every address in `cidr` is also within `parent_cidr`. A prefix contains itself. |
| `cidr_overlap` | `cidr_a` string, `cidr_b` string | bool | Whether two CIDR blocks share at least one address. Symmetric. |
| `cidr_intersection` | `cidr_a` string, `cidr_b` string | string (nullable) | The overlapping range (always the more specific of the two), or `null` if they don't overlap. |
| `cidr_difference` | `cidr` string, `exclude` string | list(string) | Minimal CIDR list covering `cidr` minus `exclude`. |
| `cidr_aggregate` | `cidrs` list(string) | list(string) | Merges a list into the minimal non-overlapping set - drops redundant entries, merges aligned siblings to a fixed point. IPv4 first, then IPv6. |
| `cidr_capacity` | `cidr` string | object: `total` string, `usable` string | Address counts as decimal strings (not numbers - a `/0` IPv6 total is 2^128). `usable` = `total-2` for IPv4 (except `/31`=2, `/32`=1); IPv6 has no broadcast address, so `usable` = `total`. |
| `ptr_name` | `ip` string | string | Reverse-DNS name, e.g. `5.2.0.192.in-addr.arpa.`. |
| `ip_from_ptr` | `ptr` string | string | Inverse of `ptr_name`. |
| `dns_label_validate` | `label` string | bool | RFC 1123 single label: 1-63 chars, alphanumeric/hyphen, must start/end alphanumeric. |
| `hostname_validate` | `hostname` string | bool | Every dot-separated label passes `dns_label_validate`; total length (excluding an optional trailing dot) ≤ 253. |
| `hostname_normalize` | `hostname` string | string | Lowercases; sanitizes each label independently (invalid chars → hyphen, collapsed, trimmed); drops empty labels. |
| `fqdn_join` | `hostname` string, `domain` string | string | Joins into a canonical FQDN with exactly one trailing dot. |
| `fqdn_validate` | `fqdn` string | bool | Same rule as `hostname_validate`. |
| `caa_record_validate` | `flags` number, `tag` string | bool | RFC 6844: `flags` 0-255, `tag` one of `issue`/`issuewild`/`iodef` (case-insensitive). |
| `dns_txt_length_validate` | `value` string | bool | Whether `value` fits a single DNS TXT character-string (RFC 1035: max 255 bytes). A provider that transparently chunks longer values is a looser constraint than this. |
| `dns_cname_conflicts` | `records` list(object: `name` string, `type` string) | list(object: `name` string, `types` list(string)) | Every name where a `CNAME` was declared alongside any other record type (RFC 1034) - names compared case-insensitively, trailing dot ignored. |
| `cidr_overlap_analysis` | `cidrs` list(string) | list(object: `cidr_a` string, `cidr_b` string) | Every pair in the list that overlaps - e.g. conflicting route-table entries. |
| `port_range_validate` | `from` number, `to` number | bool | Both within 1-65535, `from <= to`. |
| `port_range_overlap` | `a_from` number, `a_to` number, `b_from` number, `b_to` number | bool | Whether two port ranges share a port. Each range must independently pass `port_range_validate`. |

### ip_classify precedence

Checked in this order; the first match wins:

1. `unspecified` - `0.0.0.0`, `::`
2. `loopback`
3. `multicast` - includes link-local and interface-local multicast
4. `link-local` - unicast only
5. `cgnat` - RFC 6598 `100.64.0.0/10`, IPv4 only
6. `documentation` - RFC 5737 (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`), RFC 3849 (`2001:db8::/32`)
7. `benchmarking` - RFC 2544 (`198.18.0.0/15`), RFC 5180 (`2001:2::/48`)
8. `private` - RFC 1918, RFC 4193 (ULA)
9. `public` - global unicast, none of the above
10. `reserved` - valid address, none of the above

## Data sources

### netutils_subnet_plan

Deterministic, stateless CIDR allocation planner.

```hcl
data "netutils_subnet_plan" "main" {
  base_cidrs          = ["10.100.0.0/16"]
  allocation_strategy = "best_fit" # first_fit (default) | best_fit | worst_fit | sequential
  reserved_cidrs      = ["10.100.128.0/20"]

  allocations = {
    public  = { count = 3, required_hosts = 200 }
    private = { count = 3, required_hosts = 800 }
  }

  existing_allocations = {
    # "public-0" = "10.100.1.0/24"
  }
}

# data.netutils_subnet_plan.main.results is a map keyed "<group>-<index>":
#   { name, cidr, prefix_length, address_family, total_capacity,
#     usable_capacity, index, base_cidr, status }
# status: "allocated" | "preserved" | "insufficient" | "failed"
```

Deviates from the project's own illustrative spec in one way: the
*result* is exposed as `results`, not reusing the input attribute name
`allocations` - a Terraform data source attribute can't be both a
Required input type and a different Computed output type under the same
name.

`base_cidrs` must all be the same address family - call this twice (once
per family) for dual-stack planning.

Allocation strategies (applied only to requests without an
`existing_allocations` entry):

- **first_fit** - the first free block big enough, in stable order
  (`base_cidrs` in the given order, ascending address within each). May
  skip smaller free fragments earlier in that order.
- **best_fit** - the smallest free block that is still big enough
  (minimizes leftover fragmentation for this allocation).
- **worst_fit** - the largest available free block (maximizes leftover
  for future requests).
- **sequential** - like first_fit, but never reuses space below a
  monotonically advancing frontier. Once an address has been passed, an
  earlier gap freed by fragmentation is never revisited even if it would
  fit - a request that can only be satisfied by such a gap fails under
  this strategy specifically, where first_fit would have succeeded.

A request that can't be satisfied gets `status = "failed"` (`cidr = ""`)
rather than aborting the whole plan.

`existing_allocations` is keyed the same way as `results`:
`"<group>-<index>"`. A key present there is never reallocated - its
capacity is only checked against `required_hosts`, reported as
`"preserved"` (still enough room) or `"insufficient"` (kept anyway, not
silently replaced).

No AWS/Azure/GCP-specific reserved addresses are built in - express any
such reservation explicitly via `reserved_cidrs`.

#### Feeding results into terraform-aws-vpc + terraform-aws-subnets

Neither [`terraform-aws-vpc`](https://github.com/rafalmasiarek/terraform-aws-vpc)
nor [`terraform-aws-subnets`](https://github.com/rafalmasiarek/terraform-aws-subnets)
requires this provider - `cidr_block` and `subnet_groups[*].cidr_block`
already accept a plain string from any source. A full chain, including
[`terraform-null-tagger`](https://github.com/rafalmasiarek/terraform-null-tagger)
for tagging both:

```hcl
module "tagger" {
  source  = "rafalmasiarek/tagger/null"
  version = ">= 0.1.0"

  namespace   = "acme"
  environment = "prod"
  owner       = "sre-team"
}

module "vpc" {
  source = "git::https://github.com/rafalmasiarek/terraform-aws-vpc.git?ref=v0.7.1"

  name       = "vpc"
  cidr_block = "10.100.0.0/16"

  tagger_context = module.tagger.context
}

data "netutils_subnet_plan" "main" {
  base_cidrs = ["10.100.0.0/16"]

  allocations = {
    public  = { count = 2, required_hosts = 200 }
    private = { count = 2, required_hosts = 800 }
  }
}

module "subnets" {
  source = "git::https://github.com/rafalmasiarek/terraform-aws-subnets.git?ref=v0.4.0"

  vpc_id = module.vpc.vpc_id

  route_tables = {
    public  = { routes = [{ cidr_block = "0.0.0.0/0", gateway_id = module.vpc.internet_gateway_id }] }
    private = { routes = [] }
  }

  subnet_groups = {
    for key, allocation in data.netutils_subnet_plan.main.results :
    key => {
      cidr_block              = allocation.cidr
      availability_zone       = "eu-central-1a" # distribute across AZs yourself, e.g. keyed off allocation.index
      route_table_key         = startswith(key, "public-") ? "public" : "private"
      role                    = startswith(key, "public-") ? "public" : "private"
      map_public_ip_on_launch = startswith(key, "public-")
    }
  }

  tagger_context = module.tagger.context
}
```

### netutils_ip_info

```hcl
data "netutils_ip_info" "self" {
  ip       = "8.8.8.8" # omit for the public IP the API itself observes
  asn      = true       # default
  location = true        # default
  abuse    = false       # default
  rbl      = false       # default
}
```

Outputs: `result_ip`, `hostname`, `network`, `asn_number`, `asn_name`,
`asn_type`, `country`, `country_name`, `region`, `city`, `is_eu`,
`abuse_raw`, `rbl_raw` (the last two: raw JSON strings, `null` if the
section wasn't returned - the upstream API's exact field set for those
two sections isn't part of this provider's documented contract yet).
Retries once per 5xx/network failure (configurable `timeout_seconds`,
default 10); a 4xx is never retried.

### netutils_tcp_check / netutils_http_check / netutils_tls_certificate

```hcl
data "netutils_tcp_check" "db" {
  host = "db.internal"
  port = 5432
}

data "netutils_http_check" "api" {
  url             = "https://api.internal/healthz"
  expected_status = 200 # omit to accept any 2xx/3xx
}

data "netutils_tls_certificate" "api" {
  host = "api.internal"
  # port defaults to 443
}
```

Connection/handshake failure is reported via `success = false` and
`error`, never as a plan-time error - an unreachable endpoint is a
meaningful diagnostic result. `netutils_tls_certificate` inspects
whatever certificate is actually presented (including an expired one);
it does not validate the certificate chain.

## Development

```
go build -o bin/terraform-provider-netutils .
go test ./...
```

Local testing without a published release: point Terraform at the built
binary via `dev_overrides` in a CLI config file (not committed to this
repo):

```hcl
provider_installation {
  dev_overrides {
    "rafalmasiarek/netutils" = "/absolute/path/to/bin"
  }
  direct {}
}
```

Set `TF_CLI_CONFIG_FILE` to that file's path, then run `terraform plan`
directly - `terraform init` is unnecessary (and will fail) with
`dev_overrides` active for a provider.

## Author

[Rafał Masiarek](https://github.com/rafalmasiarek)
