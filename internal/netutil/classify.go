package netutil

import "net/netip"

// Address classification labels returned by Classify.
const (
	ClassUnspecified   = "unspecified"
	ClassLoopback      = "loopback"
	ClassMulticast     = "multicast"
	ClassLinkLocal     = "link-local"
	ClassCGNAT         = "cgnat"
	ClassDocumentation = "documentation"
	ClassBenchmarking  = "benchmarking"
	ClassPrivate       = "private"
	ClassPublic        = "public"
	ClassReserved      = "reserved"
)

// cgnatRange is RFC 6598 Shared Address Space (100.64.0.0/10), used by
// carrier-grade NAT - distinct from RFC 1918 private space.
var cgnatRange = netip.MustParsePrefix("100.64.0.0/10")

// documentationRanges are addresses reserved for documentation/examples -
// RFC 5737 (IPv4 TEST-NET-1/2/3) and RFC 3849 (IPv6 documentation).
var documentationRanges = []netip.Prefix{
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:db8::/32"),
}

// benchmarkingRanges are RFC 2544 (IPv4) and RFC 5180 (IPv6) network
// benchmarking device testing ranges.
var benchmarkingRanges = []netip.Prefix{
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("2001:2::/48"),
}

// Classify returns a single classification label for ipStr, in priority
// order: unspecified, loopback, multicast (link-local or global),
// link-local unicast, cgnat, documentation, benchmarking, private, public.
// Anything that is valid but matches none of these (e.g. other IANA
// special-purpose ranges not enumerated above) is reported as "reserved"
// rather than misclassified as "public".
func Classify(ipStr string) (string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", err
	}

	switch {
	case addr.IsUnspecified():
		return ClassUnspecified, nil
	case addr.IsLoopback():
		return ClassLoopback, nil
	case addr.IsMulticast():
		return ClassMulticast, nil
	case addr.IsLinkLocalUnicast():
		return ClassLinkLocal, nil
	}

	if addr.Is4() || addr.Is4In6() {
		if cgnatRange.Contains(addr.Unmap()) {
			return ClassCGNAT, nil
		}
	}

	for _, r := range documentationRanges {
		if r.Contains(addr) {
			return ClassDocumentation, nil
		}
	}

	for _, r := range benchmarkingRanges {
		if r.Contains(addr) {
			return ClassBenchmarking, nil
		}
	}

	if addr.IsPrivate() {
		return ClassPrivate, nil
	}

	if addr.IsGlobalUnicast() {
		return ClassPublic, nil
	}

	return ClassReserved, nil
}

// Normalize returns the canonical string form of ipStr - compressed and
// lowercase for IPv6 (RFC 5952), plain dotted-decimal for IPv4.
func Normalize(ipStr string) (string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

// InCIDR reports whether ipStr falls within cidrStr.
func InCIDR(ipStr, cidrStr string) (bool, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, err
	}
	prefix, err := netip.ParsePrefix(cidrStr)
	if err != nil {
		return false, err
	}
	return prefix.Contains(addr), nil
}

// CIDRContains reports whether childStr is fully contained within
// parentStr - every address in childStr is also in parentStr. A prefix
// always contains itself.
func CIDRContains(parentStr, childStr string) (bool, error) {
	parent, err := netip.ParsePrefix(parentStr)
	if err != nil {
		return false, err
	}
	child, err := netip.ParsePrefix(childStr)
	if err != nil {
		return false, err
	}
	if parent.Addr().Is4() != child.Addr().Is4() {
		return false, nil
	}
	if parent.Bits() > child.Bits() {
		return false, nil
	}
	return parent.Contains(child.Masked().Addr()), nil
}

// CIDROverlap reports whether aStr and bStr share at least one address.
func CIDROverlap(aStr, bStr string) (bool, error) {
	a, err := netip.ParsePrefix(aStr)
	if err != nil {
		return false, err
	}
	b, err := netip.ParsePrefix(bStr)
	if err != nil {
		return false, err
	}
	return a.Overlaps(b), nil
}
