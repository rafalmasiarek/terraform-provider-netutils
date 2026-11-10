package netutil

import (
	"net/netip"
	"sort"
	"strings"
)

// IPFamily returns "ipv4" or "ipv6" for a valid address.
func IPFamily(ipStr string) (string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", err
	}
	if addr.Is4() {
		return "ipv4", nil
	}
	return "ipv6", nil
}

// CIDRFamily returns "ipv4" or "ipv6" for a valid CIDR block.
func CIDRFamily(cidrStr string) (string, error) {
	prefix, err := netip.ParsePrefix(cidrStr)
	if err != nil {
		return "", err
	}
	if prefix.Addr().Is4() {
		return "ipv4", nil
	}
	return "ipv6", nil
}

var validCAATags = map[string]bool{
	"issue":     true,
	"issuewild": true,
	"iodef":     true,
}

// CAARecordValidate reports whether flags and tag are valid for a CAA
// record per RFC 6844: flags 0-255, tag one of "issue", "issuewild",
// "iodef" (case-insensitive).
func CAARecordValidate(flags int64, tag string) bool {
	if flags < 0 || flags > 255 {
		return false
	}
	return validCAATags[strings.ToLower(tag)]
}

// DNSTXTLengthValidate reports whether value fits in a single DNS TXT
// character-string per RFC 1035 (max 255 bytes). Providers that
// transparently chunk longer values across multiple character-strings
// (as some DNS APIs do) are a looser constraint than this - treat a
// false result here as "would need chunking", not necessarily "rejected
// outright".
func DNSTXTLengthValidate(value string) bool {
	return len(value) <= 255
}

// CNAMEConflict is one name where a CNAME record was declared alongside
// at least one other record type - invalid per RFC 1034: a CNAME must be
// the only record at its name.
type CNAMEConflict struct {
	Name  string
	Types []string
}

// DNSRecordRef is the minimal shape DNSCNAMEConflicts needs from a DNS
// record: its (defaulted) name and type.
type DNSRecordRef struct {
	Name string
	Type string
}

// DNSCNAMEConflicts groups records by name (case-insensitive, trailing
// dot ignored) and reports every name where "CNAME" appears alongside
// any other type. Multiple records of the same non-CNAME type at one
// name (e.g. round-robin A records) are not a conflict. Results are
// sorted by name for deterministic output.
func DNSCNAMEConflicts(records []DNSRecordRef) []CNAMEConflict {
	byName := map[string]map[string]bool{}
	order := map[string]string{} // normalized name -> first-seen original name, for output

	for _, r := range records {
		key := strings.ToLower(strings.TrimSuffix(r.Name, "."))
		if _, ok := byName[key]; !ok {
			byName[key] = map[string]bool{}
			order[key] = r.Name
		}
		byName[key][strings.ToUpper(r.Type)] = true
	}

	var conflicts []CNAMEConflict
	for key, types := range byName {
		if !types["CNAME"] || len(types) < 2 {
			continue
		}
		typeList := make([]string, 0, len(types))
		for t := range types {
			typeList = append(typeList, t)
		}
		sort.Strings(typeList)
		conflicts = append(conflicts, CNAMEConflict{Name: order[key], Types: typeList})
	}

	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Name < conflicts[j].Name })
	return conflicts
}
