package netutil

import "testing"

func TestDNSLabelValidate(t *testing.T) {
	cases := []struct {
		label string
		want  bool
	}{
		{"web", true},
		{"web-01", true},
		{"a", true},
		{"", false},
		{"-web", false},
		{"web-", false},
		{"web_01", false},
		{"WEB", true}, // DNS is case-insensitive; validation doesn't force lowercase
	}

	label63 := make([]byte, 63)
	for i := range label63 {
		label63[i] = 'a'
	}
	if !DNSLabelValidate(string(label63)) {
		t.Error("DNSLabelValidate should accept a 63-character label (the RFC 1123 limit)")
	}
	label64 := append(label63, 'a')
	if DNSLabelValidate(string(label64)) {
		t.Error("DNSLabelValidate should reject a 64-character label (over the RFC 1123 limit)")
	}
	for _, c := range cases {
		got := DNSLabelValidate(c.label)
		if got != c.want {
			t.Errorf("DNSLabelValidate(%q) = %v, want %v", c.label, got, c.want)
		}
	}
}

func TestHostnameValidate(t *testing.T) {
	cases := []struct {
		hostname string
		want     bool
	}{
		{"web-01", true},
		{"web-01.internal", true},
		{"web-01.internal.", true},
		{"", false},
		{"web..internal", false},
		{"-web.internal", false},
	}
	for _, c := range cases {
		got := HostnameValidate(c.hostname)
		if got != c.want {
			t.Errorf("HostnameValidate(%q) = %v, want %v", c.hostname, got, c.want)
		}
	}
}

func TestHostnameNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Web_01!!", "web-01"},
		{"API.Production.Example", "api.production.example"},
		{"--web--", "web"},
		{"web..internal", "web.internal"},
	}
	for _, c := range cases {
		got := HostnameNormalize(c.in)
		if got != c.want {
			t.Errorf("HostnameNormalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFQDNJoin(t *testing.T) {
	cases := []struct {
		hostname, domain, want string
	}{
		{"web-01", "internal.example.com", "web-01.internal.example.com."},
		{"web-01.", "internal.example.com.", "web-01.internal.example.com."},
		{"web-01", "", "web-01."},
	}
	for _, c := range cases {
		got := FQDNJoin(c.hostname, c.domain)
		if got != c.want {
			t.Errorf("FQDNJoin(%q, %q) = %q, want %q", c.hostname, c.domain, got, c.want)
		}
	}
}

func TestFQDNValidate(t *testing.T) {
	if !FQDNValidate("web-01.internal.example.com.") {
		t.Error("FQDNValidate should accept a well-formed FQDN with trailing dot")
	}
	if FQDNValidate("") {
		t.Error("FQDNValidate should reject an empty string")
	}
}
