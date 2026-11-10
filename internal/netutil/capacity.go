package netutil

import (
	"math/big"
	"net/netip"
)

// Capacity returns the total number of addresses in cidrStr and the
// number considered usable: for IPv4, total-2 (network + broadcast
// reserved), except /31 (2, point-to-point - RFC 3021) and /32 (1, a
// single host, nothing reserved). IPv6 has no broadcast address, so usable
// equals total.
//
// Both values are returned as decimal strings rather than a native
// numeric type - a /0 IPv6 total is 2^128, far beyond any integer type
// Terraform's Number (a float64-precision-by-default big.Float) could
// represent exactly.
func Capacity(cidrStr string) (total string, usable string, err error) {
	prefix, err := netip.ParsePrefix(cidrStr)
	if err != nil {
		return "", "", err
	}

	bitLen := addrBitLen(prefix.Addr())
	totalBig := blockSize(bitLen, prefix.Bits())

	usableBig := new(big.Int).Set(totalBig)
	if prefix.Addr().Is4() {
		switch {
		case prefix.Bits() == 32:
			usableBig.SetInt64(1)
		case prefix.Bits() == 31:
			usableBig.SetInt64(2)
		default:
			usableBig.Sub(totalBig, big.NewInt(2))
		}
	}

	return totalBig.String(), usableBig.String(), nil
}
