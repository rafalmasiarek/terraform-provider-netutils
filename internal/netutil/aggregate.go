package netutil

import (
	"math/big"
	"net/netip"
	"sort"
)

// Aggregate merges cidrStrs into the minimal set of non-overlapping CIDRs
// covering exactly the same addresses - redundant entries (duplicates, or
// one fully contained in another) are dropped, and sibling blocks that
// together form a larger aligned block are merged, repeated to a fixed
// point. IPv4 and IPv6 entries are aggregated independently; the result
// lists IPv4 first, then IPv6, each in ascending address order.
func Aggregate(cidrStrs []string) ([]string, error) {
	var v4, v6 []netip.Prefix
	for _, s := range cidrStrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, err
		}
		p = p.Masked()
		if p.Addr().Is4() {
			v4 = append(v4, p)
		} else {
			v6 = append(v6, p)
		}
	}

	merged := append(aggregateFamily(v4), aggregateFamily(v6)...)

	out := make([]string, 0, len(merged))
	for _, p := range merged {
		out = append(out, p.String())
	}
	return out, nil
}

func aggregateFamily(prefixes []netip.Prefix) []netip.Prefix {
	if len(prefixes) == 0 {
		return nil
	}

	current := dedupeContained(prefixes)

	for {
		merged, changed := mergeSiblingsPass(current)
		current = merged
		if !changed {
			break
		}
	}

	sort.Slice(current, func(i, j int) bool {
		return addrBigInt(current[i].Addr()).Cmp(addrBigInt(current[j].Addr())) < 0
	})
	return current
}

// dedupeContained drops any prefix that is fully contained within another
// (distinct) prefix in the list, including exact duplicates.
func dedupeContained(prefixes []netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	for i, p := range prefixes {
		contained := false
		for j, q := range prefixes {
			if i == j {
				continue
			}
			if q.Bits() < p.Bits() && q.Contains(p.Addr()) {
				contained = true
				break
			}
			if q.Bits() == p.Bits() && q.Addr() == p.Addr() && j < i {
				// exact duplicate - keep only the first occurrence.
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, p)
		}
	}
	return out
}

// mergeSiblingsPass performs a single pass merging any pair of same-size
// sibling prefixes into their shared parent. changed is true if at least
// one merge happened, signaling the caller to run another pass.
func mergeSiblingsPass(prefixes []netip.Prefix) ([]netip.Prefix, bool) {
	used := make([]bool, len(prefixes))
	var out []netip.Prefix
	changed := false

	for i := range prefixes {
		if used[i] {
			continue
		}
		merged := false
		for j := i + 1; j < len(prefixes); j++ {
			if used[j] {
				continue
			}
			if parent, ok := mergeSiblings(prefixes[i], prefixes[j]); ok {
				out = append(out, parent)
				used[i] = true
				used[j] = true
				changed = true
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, prefixes[i])
		}
	}

	return out, changed
}

// mergeSiblings reports whether a and b are the two halves of one aligned
// parent block one bit shorter, returning that parent if so.
func mergeSiblings(a, b netip.Prefix) (netip.Prefix, bool) {
	if a.Bits() != b.Bits() || a.Bits() == 0 {
		return netip.Prefix{}, false
	}
	if a.Addr().Is4() != b.Addr().Is4() {
		return netip.Prefix{}, false
	}

	bitLen := addrBitLen(a.Addr())
	bits := a.Bits()
	half := blockSize(bitLen, bits)
	parentBlock := new(big.Int).Lsh(half, 1)

	aInt := addrBigInt(a.Addr())
	bInt := addrBigInt(b.Addr())

	// a must be the lower half, aligned to the parent block boundary, and
	// b must start exactly one block after it.
	if new(big.Int).Mod(aInt, parentBlock).Sign() != 0 {
		return netip.Prefix{}, false
	}
	expectedB := new(big.Int).Add(aInt, half)
	if bInt.Cmp(expectedB) != 0 {
		return netip.Prefix{}, false
	}

	parentAddr := bigIntAddr(aInt, a.Addr().Is4())
	parentPrefix, _ := parentAddr.Prefix(bits - 1)
	return parentPrefix, true
}
