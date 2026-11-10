package netutil

import (
	"math/big"
	"net/netip"
)

// addrBigInt converts addr to a big.Int - uniform arithmetic for IPv4 (32
// bits) and IPv6 (128 bits) without overflowing a native uint type.
func addrBigInt(addr netip.Addr) *big.Int {
	return new(big.Int).SetBytes(addr.AsSlice())
}

// bigIntAddr converts bi back into an address of the given family. bi must
// fit in 4 bytes for IPv4 or 16 bytes for IPv6 - callers are expected to
// keep arithmetic within the address space of the family they started
// from.
func bigIntAddr(bi *big.Int, is4 bool) netip.Addr {
	size := 16
	if is4 {
		size = 4
	}
	b := bi.Bytes()
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	addr, _ := netip.AddrFromSlice(padded)
	if is4 {
		addr = addr.Unmap()
	}
	return addr
}

// blockSize returns the number of addresses covered by a prefix with the
// given bit length, within an address family of bitLen total bits (32 for
// IPv4, 128 for IPv6).
func blockSize(bitLen, bits int) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), uint(bitLen-bits))
}

func addrBitLen(addr netip.Addr) int {
	if addr.Is4() {
		return 32
	}
	return 128
}
