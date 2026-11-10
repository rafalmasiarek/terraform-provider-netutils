package netutil

import "net/netip"

// Intersection returns the overlapping range of a and b, if any. CIDR
// prefixes are hierarchical (never partially overlapping) - whenever two
// prefixes overlap, one is always fully contained within the other, so
// their intersection is exactly the more specific (larger Bits()) of the
// two. ok is false, with an empty string result, when a and b don't
// overlap at all (including a family mismatch).
func Intersection(aStr, bStr string) (result string, ok bool, err error) {
	a, err := netip.ParsePrefix(aStr)
	if err != nil {
		return "", false, err
	}
	b, err := netip.ParsePrefix(bStr)
	if err != nil {
		return "", false, err
	}

	if !a.Overlaps(b) {
		return "", false, nil
	}
	if a.Bits() >= b.Bits() {
		return a.String(), true, nil
	}
	return b.String(), true, nil
}
