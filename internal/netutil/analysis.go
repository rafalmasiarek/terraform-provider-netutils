package netutil

import (
	"fmt"
	"net/netip"
)

// CIDROverlapPairs reports every pair of distinct entries in cidrs that
// overlap, by index into the input slice (i < j for every pair, each
// unordered pair reported once). Useful for route-table conflict
// detection: a non-empty result means at least two declared routes claim
// the same address space.
func CIDROverlapPairs(cidrs []string) ([][2]int, error) {
	prefixes := make([]netip.Prefix, len(cidrs))
	for i, s := range cidrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, err
		}
		prefixes[i] = p
	}

	var pairs [][2]int
	for i := 0; i < len(prefixes); i++ {
		for j := i + 1; j < len(prefixes); j++ {
			if prefixes[i].Overlaps(prefixes[j]) {
				pairs = append(pairs, [2]int{i, j})
			}
		}
	}
	return pairs, nil
}

// PortRangeValidate reports whether from/to form a valid port range:
// both within 1-65535, and from <= to.
func PortRangeValidate(from, to int) bool {
	return from >= 1 && from <= 65535 && to >= 1 && to <= 65535 && from <= to
}

// PortRangeOverlap reports whether two port ranges share at least one
// port. Returns an error if either range fails PortRangeValidate.
func PortRangeOverlap(aFrom, aTo, bFrom, bTo int) (bool, error) {
	if !PortRangeValidate(aFrom, aTo) {
		return false, fmt.Errorf("invalid port range %d-%d", aFrom, aTo)
	}
	if !PortRangeValidate(bFrom, bTo) {
		return false, fmt.Errorf("invalid port range %d-%d", bFrom, bTo)
	}
	return aFrom <= bTo && bFrom <= aTo, nil
}
