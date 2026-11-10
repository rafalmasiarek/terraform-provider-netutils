package netutil

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestCapacity(t *testing.T) {
	cases := []struct {
		cidr        string
		total, used string
	}{
		{"10.0.0.0/24", "256", "254"},
		{"10.0.0.0/31", "2", "2"},
		{"10.0.0.0/32", "1", "1"},
		{"10.0.0.0/8", "16777216", "16777214"},
		{"2001:db8::/64", "18446744073709551616", "18446744073709551616"},
		{"::/0", "340282366920938463463374607431768211456", "340282366920938463463374607431768211456"},
	}
	for _, c := range cases {
		total, usable, err := Capacity(c.cidr)
		if err != nil {
			t.Errorf("Capacity(%q) returned error: %v", c.cidr, err)
			continue
		}
		if total != c.total || usable != c.used {
			t.Errorf("Capacity(%q) = (%s, %s), want (%s, %s)", c.cidr, total, usable, c.total, c.used)
		}
	}
}

func TestIntersection(t *testing.T) {
	cases := []struct {
		a, b   string
		want   string
		wantOK bool
	}{
		{"10.0.0.0/8", "10.1.0.0/16", "10.1.0.0/16", true},
		{"10.1.0.0/16", "10.0.0.0/8", "10.1.0.0/16", true},
		{"10.0.0.0/16", "10.0.0.0/16", "10.0.0.0/16", true},
		{"10.0.0.0/16", "10.1.0.0/16", "", false},
		{"10.0.0.0/8", "2001:db8::/32", "", false},
	}
	for _, c := range cases {
		got, ok, err := Intersection(c.a, c.b)
		if err != nil {
			t.Errorf("Intersection(%q, %q) returned error: %v", c.a, c.b, err)
			continue
		}
		if ok != c.wantOK || got != c.want {
			t.Errorf("Intersection(%q, %q) = (%q, %v), want (%q, %v)", c.a, c.b, got, ok, c.want, c.wantOK)
		}
	}
}

func TestDifference(t *testing.T) {
	cases := []struct {
		cidr, exclude string
		want          []string
	}{
		{"10.0.0.0/24", "10.1.0.0/24", []string{"10.0.0.0/24"}},                   // no overlap
		{"10.0.0.0/24", "10.0.0.0/24", []string{}},                                // exact match removed entirely
		{"10.0.0.0/24", "10.0.0.0/23", []string{}},                                // exclude is a superset
		{"10.0.0.0/24", "10.0.0.128/25", []string{"10.0.0.0/25"}},                 // remove the upper half
		{"10.0.0.0/24", "10.0.0.64/26", []string{"10.0.0.0/26", "10.0.0.128/25"}}, // remove one quarter
		{"10.0.0.0/24", "10.0.0.5/32", []string{"10.0.0.0/30", "10.0.0.4/32", "10.0.0.6/31", "10.0.0.8/29", "10.0.0.16/28", "10.0.0.32/27", "10.0.0.64/26", "10.0.0.128/25"}},
	}
	for _, c := range cases {
		got, err := Difference(c.cidr, c.exclude)
		if err != nil {
			t.Errorf("Difference(%q, %q) returned error: %v", c.cidr, c.exclude, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Difference(%q, %q) = %v, want %v", c.cidr, c.exclude, got, c.want)
		}
	}
}

func TestDifferenceCoversCapacity(t *testing.T) {
	// Whatever Difference returns, the removed capacity plus the
	// remaining capacity must equal the original - a property check
	// independent of the exact split chosen.
	got, err := Difference("10.0.0.0/16", "10.0.32.0/20")
	if err != nil {
		t.Fatalf("Difference returned error: %v", err)
	}

	var sum int64
	for _, cidr := range got {
		_, usable, err := Capacity(cidr)
		if err != nil {
			t.Fatalf("Capacity(%q) returned error: %v", cidr, err)
		}
		var n int64
		for _, ch := range usable {
			n = n*10 + int64(ch-'0')
		}
		// Capacity() reports usable (total-2) for IPv4; use total instead
		// via the same helper to avoid double-subtracting here.
		total, _, _ := Capacity(cidr)
		n = 0
		for _, ch := range total {
			n = n*10 + int64(ch-'0')
		}
		sum += n
	}

	wantTotal, _, _ := Capacity("10.0.0.0/16")
	excludeTotal, _, _ := Capacity("10.0.32.0/20")
	var want, exclude int64
	for _, ch := range wantTotal {
		want = want*10 + int64(ch-'0')
	}
	for _, ch := range excludeTotal {
		exclude = exclude*10 + int64(ch-'0')
	}

	if sum != want-exclude {
		t.Errorf("Difference capacity sum = %d, want %d (%d - %d)", sum, want-exclude, want, exclude)
	}
}

func TestAggregate(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		// two adjacent /25s merge into a /24
		{[]string{"10.0.0.0/25", "10.0.0.128/25"}, []string{"10.0.0.0/24"}},
		// a contained prefix is dropped
		{[]string{"10.0.0.0/24", "10.0.0.0/25"}, []string{"10.0.0.0/24"}},
		// non-adjacent prefixes stay separate
		{[]string{"10.0.0.0/24", "10.0.2.0/24"}, []string{"10.0.0.0/24", "10.0.2.0/24"}},
		// multi-level cascade: four /26s -> one /24
		{[]string{"10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26"}, []string{"10.0.0.0/24"}},
		// exact duplicate collapses to one
		{[]string{"10.0.0.0/24", "10.0.0.0/24"}, []string{"10.0.0.0/24"}},
		// mixed families aggregate independently, IPv4 first
		{[]string{"2001:db8::/33", "2001:db8:8000::/33", "10.0.0.0/25", "10.0.0.128/25"}, []string{"10.0.0.0/24", "2001:db8::/32"}},
	}
	for _, c := range cases {
		got, err := Aggregate(c.in)
		if err != nil {
			t.Errorf("Aggregate(%v) returned error: %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Aggregate(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAggregatePreservesCapacity(t *testing.T) {
	in := []string{"10.0.0.0/26", "10.0.0.64/27", "10.0.0.96/27", "10.0.0.128/25", "10.0.1.0/24"}
	got, err := Aggregate(in)
	if err != nil {
		t.Fatalf("Aggregate(%v) returned error: %v", in, err)
	}

	sum := func(cidrs []string) int64 {
		var total int64
		for _, c := range cidrs {
			totalStr, _, err := Capacity(c)
			if err != nil {
				t.Fatalf("Capacity(%q) returned error: %v", c, err)
			}
			var n int64
			for _, ch := range totalStr {
				n = n*10 + int64(ch-'0')
			}
			total += n
		}
		return total
	}

	if sum(got) != sum(in) {
		t.Errorf("Aggregate(%v) = %v covers %d addresses, want %d (input unchanged in total)", in, got, sum(got), sum(in))
	}

	for i := range got {
		pi, _ := netip.ParsePrefix(got[i])
		for j := range got {
			if i == j {
				continue
			}
			pj, _ := netip.ParsePrefix(got[j])
			if pi.Overlaps(pj) {
				t.Errorf("Aggregate(%v) = %v has overlapping outputs %s and %s", in, got, got[i], got[j])
			}
		}
	}
}

func TestPTRName(t *testing.T) {
	cases := []struct {
		ip   string
		want string
	}{
		{"192.0.2.5", "5.2.0.192.in-addr.arpa."},
		{"2001:db8::1", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."},
	}
	for _, c := range cases {
		got, err := PTRName(c.ip)
		if err != nil {
			t.Errorf("PTRName(%q) returned error: %v", c.ip, err)
			continue
		}
		if got != c.want {
			t.Errorf("PTRName(%q) = %q, want %q", c.ip, got, c.want)
		}
	}
}

func TestIPFromPTRRoundTrip(t *testing.T) {
	ips := []string{"192.0.2.5", "8.8.8.8", "2001:db8::1", "::1"}
	for _, ip := range ips {
		ptr, err := PTRName(ip)
		if err != nil {
			t.Fatalf("PTRName(%q) returned error: %v", ip, err)
		}
		back, err := IPFromPTR(ptr)
		if err != nil {
			t.Errorf("IPFromPTR(%q) returned error: %v", ptr, err)
			continue
		}
		want, _ := Normalize(ip)
		if back != want {
			t.Errorf("IPFromPTR(PTRName(%q)) = %q, want %q", ip, back, want)
		}
	}
}

func TestIPFromPTRInvalid(t *testing.T) {
	if _, err := IPFromPTR("not.a.ptr.name."); err == nil {
		t.Error("IPFromPTR(\"not.a.ptr.name.\") expected an error, got nil")
	}
}
