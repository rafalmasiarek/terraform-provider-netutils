package netutil

import (
	"math/big"
	"net/netip"
)

// splitPrefix returns the two half-size children of p (each p.Bits()+1),
// in address order. p.Bits() must be less than the address family's bit
// length - the caller is expected to check this.
func splitPrefix(p netip.Prefix) (lo, hi netip.Prefix) {
	bitLen := addrBitLen(p.Addr())
	childBits := p.Bits() + 1

	base := addrBigInt(p.Masked().Addr())
	half := blockSize(bitLen, childBits)

	loAddr := p.Masked().Addr()
	loPrefix, _ := loAddr.Prefix(childBits)

	hiBase := new(big.Int).Add(base, half)
	hiAddr := bigIntAddr(hiBase, p.Addr().Is4())
	hiPrefix, _ := hiAddr.Prefix(childBits)

	return loPrefix, hiPrefix
}

// Difference returns the minimal list of CIDRs covering every address in
// cidrStr except those also in excludeStr, as canonical CIDR strings in
// ascending address order. If the two don't overlap at all, the result is
// [cidrStr] unchanged. If excludeStr fully contains (or exactly equals)
// cidrStr, the result is empty.
func Difference(cidrStr, excludeStr string) ([]string, error) {
	cidr, err := netip.ParsePrefix(cidrStr)
	if err != nil {
		return nil, err
	}
	exclude, err := netip.ParsePrefix(excludeStr)
	if err != nil {
		return nil, err
	}

	if cidr.Addr().Is4() != exclude.Addr().Is4() || !cidr.Overlaps(exclude) {
		return []string{cidr.Masked().String()}, nil
	}

	if exclude.Bits() <= cidr.Bits() {
		// exclude is the same size or a superset of cidr - cidr is fully removed.
		return []string{}, nil
	}

	var out []netip.Prefix
	differenceRecursive(cidr.Masked(), exclude.Masked(), &out)

	result := make([]string, 0, len(out))
	for _, p := range out {
		result = append(result, p.String())
	}
	return result, nil
}

func differenceRecursive(current, exclude netip.Prefix, out *[]netip.Prefix) {
	if current.Bits() >= exclude.Bits() {
		// current is fully covered by exclude - nothing of it survives.
		return
	}

	lo, hi := splitPrefix(current)
	if lo.Contains(exclude.Addr()) {
		differenceRecursive(lo, exclude, out)
		*out = append(*out, hi)
	} else {
		*out = append(*out, lo)
		differenceRecursive(hi, exclude, out)
	}
}
