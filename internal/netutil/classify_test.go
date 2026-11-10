package netutil

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		ip   string
		want string
	}{
		{"0.0.0.0", ClassUnspecified},
		{"::", ClassUnspecified},
		{"127.0.0.1", ClassLoopback},
		{"::1", ClassLoopback},
		{"224.0.0.1", ClassMulticast},
		{"ff02::1", ClassMulticast},
		{"169.254.1.1", ClassLinkLocal},
		{"fe80::1", ClassLinkLocal},
		{"100.64.0.1", ClassCGNAT},
		{"100.127.255.255", ClassCGNAT},
		{"100.63.255.255", ClassPublic}, // just below the CGNAT range
		{"192.0.2.1", ClassDocumentation},
		{"198.51.100.1", ClassDocumentation},
		{"203.0.113.1", ClassDocumentation},
		{"2001:db8::1", ClassDocumentation},
		{"198.18.0.1", ClassBenchmarking},
		{"2001:2::1", ClassBenchmarking},
		{"10.0.0.1", ClassPrivate},
		{"172.16.0.1", ClassPrivate},
		{"192.168.1.1", ClassPrivate},
		{"fc00::1", ClassPrivate},
		{"8.8.8.8", ClassPublic},
		{"2606:4700:4700::1111", ClassPublic},
	}

	for _, c := range cases {
		got, err := Classify(c.ip)
		if err != nil {
			t.Errorf("Classify(%q) returned error: %v", c.ip, err)
			continue
		}
		if got != c.want {
			t.Errorf("Classify(%q) = %q, want %q", c.ip, got, c.want)
		}
	}
}

func TestClassifyInvalid(t *testing.T) {
	if _, err := Classify("not-an-ip"); err == nil {
		t.Error("Classify(\"not-an-ip\") expected an error, got nil")
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"8.8.8.8", "8.8.8.8"},
		{"2001:0DB8:0000:0000:0000:0000:0000:0001", "2001:db8::1"},
		{"::FFFF:192.168.1.1", "::ffff:192.168.1.1"},
	}
	for _, c := range cases {
		got, err := Normalize(c.in)
		if err != nil {
			t.Errorf("Normalize(%q) returned error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInCIDR(t *testing.T) {
	cases := []struct {
		ip, cidr string
		want     bool
	}{
		{"10.0.1.5", "10.0.0.0/16", true},
		{"10.1.1.5", "10.0.0.0/16", false},
		{"2001:db8:1::5", "2001:db8:1::/64", true},
		{"2001:db8:2::5", "2001:db8:1::/64", false},
		{"10.0.1.5", "2001:db8::/32", false}, // mixed families - never match
	}
	for _, c := range cases {
		got, err := InCIDR(c.ip, c.cidr)
		if err != nil {
			t.Errorf("InCIDR(%q, %q) returned error: %v", c.ip, c.cidr, err)
			continue
		}
		if got != c.want {
			t.Errorf("InCIDR(%q, %q) = %v, want %v", c.ip, c.cidr, got, c.want)
		}
	}
}

func TestCIDRContains(t *testing.T) {
	cases := []struct {
		parent, child string
		want          bool
	}{
		{"10.0.0.0/8", "10.1.0.0/16", true},
		{"10.0.0.0/16", "10.0.0.0/16", true},  // a prefix contains itself
		{"10.1.0.0/16", "10.0.0.0/8", false},  // child is larger than parent
		{"10.0.0.0/16", "10.1.0.0/16", false}, // disjoint siblings
		{"2001:db8::/32", "2001:db8:1::/48", true},
		{"10.0.0.0/8", "2001:db8::/32", false}, // mixed families
	}
	for _, c := range cases {
		got, err := CIDRContains(c.parent, c.child)
		if err != nil {
			t.Errorf("CIDRContains(%q, %q) returned error: %v", c.parent, c.child, err)
			continue
		}
		if got != c.want {
			t.Errorf("CIDRContains(%q, %q) = %v, want %v", c.parent, c.child, got, c.want)
		}
	}
}

func TestCIDROverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"10.0.0.0/16", "10.0.128.0/17", true},
		{"10.0.0.0/17", "10.0.128.0/17", false},
		{"10.0.0.0/8", "10.0.0.0/8", true},
		{"2001:db8::/32", "2001:db8:1::/48", true},
		{"2001:db8::/32", "2001:db9::/32", false},
		{"10.0.0.0/8", "2001:db8::/32", false}, // mixed families
	}
	for _, c := range cases {
		got, err := CIDROverlap(c.a, c.b)
		if err != nil {
			t.Errorf("CIDROverlap(%q, %q) returned error: %v", c.a, c.b, err)
			continue
		}
		if got != c.want {
			t.Errorf("CIDROverlap(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
