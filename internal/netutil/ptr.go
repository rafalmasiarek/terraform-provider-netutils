package netutil

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// PTRName returns the reverse-DNS (PTR) name for ipStr: reversed octets
// under ".in-addr.arpa." for IPv4, or reversed nibbles of the full
// expanded form under ".ip6.arpa." for IPv6. The result has a trailing
// dot (a fully-qualified name).
func PTRName(ipStr string) (string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", err
	}

	if addr.Is4() {
		octets := strings.Split(addr.String(), ".")
		reversed := make([]string, 4)
		for i, o := range octets {
			reversed[3-i] = o
		}
		return strings.Join(reversed, ".") + ".in-addr.arpa.", nil
	}

	expanded := strings.ReplaceAll(addr.StringExpanded(), ":", "")
	nibbles := make([]string, len(expanded))
	for i, c := range expanded {
		nibbles[len(expanded)-1-i] = string(c)
	}
	return strings.Join(nibbles, ".") + ".ip6.arpa.", nil
}

// IPFromPTR parses a PTR name (as produced by PTRName) back into its
// normalized IP address.
func IPFromPTR(ptr string) (string, error) {
	name := strings.TrimSuffix(ptr, ".")

	switch {
	case strings.HasSuffix(name, ".in-addr.arpa"):
		labels := strings.Split(strings.TrimSuffix(name, ".in-addr.arpa"), ".")
		if len(labels) != 4 {
			return "", fmt.Errorf("invalid in-addr.arpa PTR name: %q", ptr)
		}
		reversed := make([]string, 4)
		for i, l := range labels {
			reversed[3-i] = l
		}
		return Normalize(strings.Join(reversed, "."))

	case strings.HasSuffix(name, ".ip6.arpa"):
		labels := strings.Split(strings.TrimSuffix(name, ".ip6.arpa"), ".")
		if len(labels) != 32 {
			return "", fmt.Errorf("invalid ip6.arpa PTR name: %q", ptr)
		}
		nibbles := make([]byte, 32)
		for i, l := range labels {
			if len(l) != 1 {
				return "", fmt.Errorf("invalid ip6.arpa PTR name: %q", ptr)
			}
			nibbles[31-i] = l[0]
		}
		var sb strings.Builder
		for i, n := range nibbles {
			if i > 0 && i%4 == 0 {
				sb.WriteByte(':')
			}
			sb.WriteByte(n)
		}
		return Normalize(sb.String())

	default:
		return "", errors.New("PTR name must end in .in-addr.arpa or .ip6.arpa: " + ptr)
	}
}
