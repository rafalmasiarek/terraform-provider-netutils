package netutil

import (
	"fmt"
	"math/big"
	"net/netip"
	"sort"
)

// AllocationSpec is one named group's request: count separate instances,
// each needing enough room for required_hosts usable addresses.
type AllocationSpec struct {
	Count         int64
	RequiredHosts int64
}

// AllocationResult is the effective outcome for one expanded allocation
// instance ("<name>-<index>"). CIDR and BaseCIDR are empty when Status is
// "failed".
type AllocationResult struct {
	Name           string
	CIDR           string
	PrefixLength   int
	AddressFamily  string
	TotalCapacity  string
	UsableCapacity string
	Index          int64
	BaseCIDR       string
	Status         string // "allocated" | "preserved" | "insufficient" | "failed"
}

// Allocation strategies. See PlanSubnets for the exact behavior of each.
const (
	StrategyFirstFit   = "first_fit"
	StrategyBestFit    = "best_fit"
	StrategyWorstFit   = "worst_fit"
	StrategySequential = "sequential"
)

var validStrategies = map[string]bool{
	StrategyFirstFit:   true,
	StrategyBestFit:    true,
	StrategyWorstFit:   true,
	StrategySequential: true,
}

type freeBlock struct {
	prefix   netip.Prefix
	baseCIDR string
}

// PlanSubnets allocates one CIDR per expanded allocation instance from
// baseCIDRs, honoring reservedCIDRs and existing (both removed from free
// space before any new allocation is attempted).
//
// All of baseCIDRs must be the same address family - mixed IPv4/IPv6 in
// one call is rejected; call this twice (once per family) for dual-stack
// planning.
//
// existing is keyed the same way as the result: "<name>-<index>". A
// request whose key appears in existing keeps that CIDR fixed (never
// reallocated) with status "preserved" if it still has enough usable
// capacity for required_hosts, or "insufficient" if it no longer does -
// reported, never silently replaced.
//
// Strategies, applied only to requests without an existing entry:
//
//   - first_fit: the first free block big enough, in stable order
//     (base_cidrs in the given order, ascending address within each).
//     May skip smaller free fragments earlier in that order.
//   - best_fit: the smallest free block that is still big enough
//     (minimizes leftover fragmentation for this allocation).
//   - worst_fit: the largest available free block (maximizes leftover
//     for future, potentially larger, requests).
//   - sequential: like first_fit, but never reuses space below a
//     monotonically advancing frontier - once an address has been
//     passed, an earlier gap freed by fragmentation is never revisited,
//     even if it would fit. Requests that can only be satisfied by such
//     a gap fail under this strategy specifically.
//
// A request that cannot be satisfied gets Status "failed" (empty CIDR)
// rather than aborting the whole plan - callers can inspect status per
// allocation.
func PlanSubnets(baseCIDRs []string, strategy string, reservedCIDRs []string, allocations map[string]AllocationSpec, existing map[string]string) (map[string]AllocationResult, error) {
	if strategy == "" {
		strategy = StrategyFirstFit
	}
	if !validStrategies[strategy] {
		return nil, fmt.Errorf("invalid allocation_strategy %q - must be one of first_fit, best_fit, worst_fit, sequential", strategy)
	}
	if len(baseCIDRs) == 0 {
		return nil, fmt.Errorf("base_cidrs must not be empty")
	}

	basePrefixes := make([]netip.Prefix, len(baseCIDRs))
	var isIPv4 bool
	for i, s := range baseCIDRs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("invalid base_cidrs[%d] %q: %w", i, s, err)
		}
		basePrefixes[i] = p.Masked()
		if i == 0 {
			isIPv4 = p.Addr().Is4()
		} else if p.Addr().Is4() != isIPv4 {
			return nil, fmt.Errorf("base_cidrs must all be the same address family - base_cidrs[%d] %q does not match base_cidrs[0]", i, s)
		}
	}
	bitLen := 128
	if isIPv4 {
		bitLen = 32
	}

	free := make([]freeBlock, len(basePrefixes))
	for i, p := range basePrefixes {
		free[i] = freeBlock{prefix: p, baseCIDR: baseCIDRs[i]}
	}

	for _, excludeStr := range reservedCIDRs {
		var err error
		free, err = subtractFromFree(free, excludeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid reserved_cidrs entry %q: %w", excludeStr, err)
		}
	}
	for key, cidr := range existing {
		var err error
		free, err = subtractFromFree(free, cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid existing_allocations[%q] %q: %w", key, cidr, err)
		}
	}

	names := make([]string, 0, len(allocations))
	for name := range allocations {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make(map[string]AllocationResult, len(allocations))
	frontier := new(big.Int) // only consulted/advanced for StrategySequential

	for _, name := range names {
		spec := allocations[name]
		if spec.RequiredHosts <= 0 {
			return nil, fmt.Errorf("allocations[%q].required_hosts must be positive", name)
		}
		for idx := int64(0); idx < spec.Count; idx++ {
			key := fmt.Sprintf("%s-%d", name, idx)

			if cidr, ok := existing[key]; ok {
				result, err := existingResult(name, idx, cidr, spec.RequiredHosts, baseCIDRs)
				if err != nil {
					return nil, fmt.Errorf("existing_allocations[%q]: %w", key, err)
				}
				results[key] = result
				continue
			}

			requiredBits := requiredBitsFor(isIPv4, bitLen, spec.RequiredHosts)

			selected, selectedIdx := selectFreeBlock(free, requiredBits, strategy, frontier)
			if selectedIdx < 0 {
				results[key] = AllocationResult{Name: name, Index: idx, Status: "failed"}
				continue
			}

			carvedAddr := selected.prefix.Addr()
			carved, err := carvedAddr.Prefix(requiredBits)
			if err != nil {
				return nil, fmt.Errorf("internal error carving %q to /%d: %w", selected.prefix.String(), requiredBits, err)
			}

			leftover, err := Difference(selected.prefix.String(), carved.String())
			if err != nil {
				return nil, fmt.Errorf("internal error computing leftover for %q minus %q: %w", selected.prefix.String(), carved.String(), err)
			}

			free = spliceFree(free, selectedIdx, leftover, selected.baseCIDR)

			if strategy == StrategySequential {
				end := new(big.Int).Add(addrBigInt(carved.Addr()), blockSize(bitLen, carved.Bits()))
				if end.Cmp(frontier) > 0 {
					frontier = end
				}
			}

			total, usable, _ := Capacity(carved.String())
			family := "ipv6"
			if isIPv4 {
				family = "ipv4"
			}
			results[key] = AllocationResult{
				Name:           name,
				CIDR:           carved.String(),
				PrefixLength:   carved.Bits(),
				AddressFamily:  family,
				TotalCapacity:  total,
				UsableCapacity: usable,
				Index:          idx,
				BaseCIDR:       selected.baseCIDR,
				Status:         "allocated",
			}
		}
	}

	return results, nil
}

func subtractFromFree(free []freeBlock, excludeStr string) ([]freeBlock, error) {
	var out []freeBlock
	for _, b := range free {
		parts, err := Difference(b.prefix.String(), excludeStr)
		if err != nil {
			return nil, err
		}
		for _, part := range parts {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, err
			}
			out = append(out, freeBlock{prefix: p, baseCIDR: b.baseCIDR})
		}
	}
	return out, nil
}

func spliceFree(free []freeBlock, at int, leftoverCIDRs []string, baseCIDR string) []freeBlock {
	leftover := make([]freeBlock, len(leftoverCIDRs))
	for i, s := range leftoverCIDRs {
		p, _ := netip.ParsePrefix(s)
		leftover[i] = freeBlock{prefix: p, baseCIDR: baseCIDR}
	}
	out := make([]freeBlock, 0, len(free)-1+len(leftover))
	out = append(out, free[:at]...)
	out = append(out, leftover...)
	out = append(out, free[at+1:]...)
	return out
}

func selectFreeBlock(free []freeBlock, requiredBits int, strategy string, frontier *big.Int) (freeBlock, int) {
	bestIdx := -1
	var best freeBlock

	for i, b := range free {
		if b.prefix.Bits() > requiredBits {
			continue // too small
		}
		if strategy == StrategySequential && addrBigInt(b.prefix.Addr()).Cmp(frontier) < 0 {
			continue // below the frontier - permanently skipped under this strategy
		}

		switch {
		case bestIdx < 0:
			bestIdx, best = i, b
		case strategy == StrategyBestFit && b.prefix.Bits() > best.prefix.Bits():
			bestIdx, best = i, b
		case strategy == StrategyWorstFit && b.prefix.Bits() < best.prefix.Bits():
			bestIdx, best = i, b
		case (strategy == StrategyFirstFit || strategy == StrategySequential):
			// free is kept in stable order - the first match found (lowest
			// i) already is the answer, nothing to update.
		}
	}

	return best, bestIdx
}

func requiredBitsFor(isIPv4 bool, bitLen int, requiredHosts int64) int {
	for bits := bitLen; bits >= 0; bits-- {
		freeBits := bitLen - bits
		if freeBits >= 63 {
			return bits // usable is far beyond any realistic requiredHosts
		}
		total := int64(1) << freeBits
		usable := total
		if isIPv4 {
			switch {
			case bits == bitLen:
				usable = 1
			case bits == bitLen-1:
				usable = 2
			default:
				usable = total - 2
			}
		}
		if usable >= requiredHosts {
			return bits
		}
	}
	return 0
}

func existingResult(name string, idx int64, cidr string, requiredHosts int64, baseCIDRs []string) (AllocationResult, error) {
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		return AllocationResult{}, err
	}

	total, usable, err := Capacity(cidr)
	if err != nil {
		return AllocationResult{}, err
	}

	var usableNum int64
	for _, ch := range usable {
		usableNum = usableNum*10 + int64(ch-'0')
	}

	status := "preserved"
	if usableNum < requiredHosts {
		status = "insufficient"
	}

	baseCIDR := ""
	for _, b := range baseCIDRs {
		bp, err := netip.ParsePrefix(b)
		if err == nil && bp.Overlaps(p) {
			baseCIDR = b
			break
		}
	}

	family := "ipv6"
	if p.Addr().Is4() {
		family = "ipv4"
	}

	return AllocationResult{
		Name:           name,
		CIDR:           p.String(),
		PrefixLength:   p.Bits(),
		AddressFamily:  family,
		TotalCapacity:  total,
		UsableCapacity: usable,
		Index:          idx,
		BaseCIDR:       baseCIDR,
		Status:         status,
	}, nil
}
