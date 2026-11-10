package netutil

import (
	"reflect"
	"testing"
)

func TestIPFamily(t *testing.T) {
	cases := []struct {
		ip   string
		want string
	}{
		{"8.8.8.8", "ipv4"},
		{"2001:db8::1", "ipv6"},
	}
	for _, c := range cases {
		got, err := IPFamily(c.ip)
		if err != nil {
			t.Errorf("IPFamily(%q) returned error: %v", c.ip, err)
			continue
		}
		if got != c.want {
			t.Errorf("IPFamily(%q) = %q, want %q", c.ip, got, c.want)
		}
	}
	if _, err := IPFamily("not-an-ip"); err == nil {
		t.Error("IPFamily(\"not-an-ip\") expected an error, got nil")
	}
}

func TestCIDRFamily(t *testing.T) {
	cases := []struct {
		cidr string
		want string
	}{
		{"10.0.0.0/16", "ipv4"},
		{"2001:db8::/32", "ipv6"},
	}
	for _, c := range cases {
		got, err := CIDRFamily(c.cidr)
		if err != nil {
			t.Errorf("CIDRFamily(%q) returned error: %v", c.cidr, err)
			continue
		}
		if got != c.want {
			t.Errorf("CIDRFamily(%q) = %q, want %q", c.cidr, got, c.want)
		}
	}
	if _, err := CIDRFamily("not-a-cidr"); err == nil {
		t.Error("CIDRFamily(\"not-a-cidr\") expected an error, got nil")
	}
	if _, err := CIDRFamily("8.8.8.8"); err == nil {
		t.Error("CIDRFamily(\"8.8.8.8\") (bare IP, not a CIDR) expected an error, got nil")
	}
}

func TestCAARecordValidate(t *testing.T) {
	cases := []struct {
		flags int64
		tag   string
		want  bool
	}{
		{0, "issue", true},
		{128, "issuewild", true},
		{0, "IODEF", true}, // case-insensitive
		{-1, "issue", false},
		{256, "issue", false},
		{0, "bogus", false},
	}
	for _, c := range cases {
		got := CAARecordValidate(c.flags, c.tag)
		if got != c.want {
			t.Errorf("CAARecordValidate(%d, %q) = %v, want %v", c.flags, c.tag, got, c.want)
		}
	}
}

func TestDNSTXTLengthValidate(t *testing.T) {
	short := "v=spf1 include:_spf.example.com ~all"
	if !DNSTXTLengthValidate(short) {
		t.Errorf("DNSTXTLengthValidate(%q) = false, want true", short)
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	if DNSTXTLengthValidate(string(long)) {
		t.Error("DNSTXTLengthValidate(256 bytes) = true, want false (over the 255-byte RFC 1035 limit)")
	}

	exactly255 := make([]byte, 255)
	for i := range exactly255 {
		exactly255[i] = 'a'
	}
	if !DNSTXTLengthValidate(string(exactly255)) {
		t.Error("DNSTXTLengthValidate(255 bytes) = false, want true (exactly at the limit)")
	}
}

func TestDNSCNAMEConflicts(t *testing.T) {
	records := []DNSRecordRef{
		{Name: "www", Type: "A"},
		{Name: "www", Type: "CNAME"}, // conflicts with the A above
		{Name: "mail", Type: "MX"},
		{Name: "mail", Type: "MX"}, // same type twice - not a conflict
		{Name: "WWW2.", Type: "CNAME"},
		{Name: "www2", Type: "TXT"},  // case/trailing-dot-insensitive match with the entry above
		{Name: "app", Type: "CNAME"}, // alone - not a conflict
	}

	got := DNSCNAMEConflicts(records)
	want := []CNAMEConflict{
		{Name: "WWW2.", Types: []string{"CNAME", "TXT"}},
		{Name: "www", Types: []string{"A", "CNAME"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DNSCNAMEConflicts(%+v) = %+v, want %+v", records, got, want)
	}
}

func TestDNSCNAMEConflictsNone(t *testing.T) {
	records := []DNSRecordRef{
		{Name: "www", Type: "A"},
		{Name: "www", Type: "A"},
		{Name: "mail", Type: "MX"},
	}
	got := DNSCNAMEConflicts(records)
	if len(got) != 0 {
		t.Errorf("DNSCNAMEConflicts(%+v) = %+v, want empty", records, got)
	}
}
