package netutil

import (
	"net/netip"
	"testing"
)

func TestPlanSubnetsSpecExample(t *testing.T) {
	results, err := PlanSubnets(
		[]string{"10.100.0.0/16"},
		StrategyBestFit,
		[]string{"10.100.128.0/20"},
		map[string]AllocationSpec{
			"public":   {Count: 3, RequiredHosts: 200},
			"private":  {Count: 3, RequiredHosts: 800},
			"database": {Count: 3, RequiredHosts: 100},
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("PlanSubnets returned error: %v", err)
	}

	if len(results) != 9 {
		t.Fatalf("got %d results, want 9", len(results))
	}

	seen := map[string]netip.Prefix{}
	for key, r := range results {
		if r.Status != "allocated" {
			t.Errorf("%s: status = %q, want \"allocated\"", key, r.Status)
		}
		p, err := netip.ParsePrefix(r.CIDR)
		if err != nil {
			t.Fatalf("%s: invalid CIDR %q: %v", key, r.CIDR, err)
		}
		// Must not overlap the reserved range.
		reserved := netip.MustParsePrefix("10.100.128.0/20")
		if p.Overlaps(reserved) {
			t.Errorf("%s: allocated %q overlaps reserved 10.100.128.0/20", key, r.CIDR)
		}
		// Must not overlap any other allocation.
		for otherKey, otherP := range seen {
			if p.Overlaps(otherP) {
				t.Errorf("%s (%q) overlaps %s (%q)", key, r.CIDR, otherKey, otherP.String())
			}
		}
		seen[key] = p

		// Capacity must satisfy the request.
		var requiredHosts int64
		switch r.Name {
		case "public":
			requiredHosts = 200
		case "private":
			requiredHosts = 800
		case "database":
			requiredHosts = 100
		}
		var usableNum int64
		for _, ch := range r.UsableCapacity {
			usableNum = usableNum*10 + int64(ch-'0')
		}
		if usableNum < requiredHosts {
			t.Errorf("%s: usable capacity %s < required_hosts %d", key, r.UsableCapacity, requiredHosts)
		}
	}
}

func TestPlanSubnetsRequiredBitsSizing(t *testing.T) {
	results, err := PlanSubnets(
		[]string{"10.0.0.0/16"},
		StrategyFirstFit,
		nil,
		map[string]AllocationSpec{"web": {Count: 1, RequiredHosts: 200}},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("PlanSubnets returned error: %v", err)
	}
	r := results["web-0"]
	// 200 usable hosts needs a /24 (254 usable) - a /25 (126 usable) is too small.
	if r.PrefixLength != 24 {
		t.Errorf("PrefixLength = %d, want 24 for 200 required_hosts", r.PrefixLength)
	}
}

func TestPlanSubnetsExistingPreserved(t *testing.T) {
	results, err := PlanSubnets(
		[]string{"10.0.0.0/16"},
		StrategyFirstFit,
		nil,
		map[string]AllocationSpec{"web": {Count: 2, RequiredHosts: 50}},
		map[string]string{"web-0": "10.0.5.0/24"},
	)
	if err != nil {
		t.Fatalf("PlanSubnets returned error: %v", err)
	}

	preserved := results["web-0"]
	if preserved.Status != "preserved" || preserved.CIDR != "10.0.5.0/24" {
		t.Errorf("web-0 = %+v, want status=preserved cidr=10.0.5.0/24", preserved)
	}

	newAlloc := results["web-1"]
	if newAlloc.Status != "allocated" {
		t.Errorf("web-1 = %+v, want status=allocated", newAlloc)
	}
	newP, _ := netip.ParsePrefix(newAlloc.CIDR)
	existingP := netip.MustParsePrefix("10.0.5.0/24")
	if newP.Overlaps(existingP) {
		t.Errorf("web-1 (%q) overlaps the preserved existing allocation 10.0.5.0/24", newAlloc.CIDR)
	}
}

func TestPlanSubnetsExistingInsufficient(t *testing.T) {
	results, err := PlanSubnets(
		[]string{"10.0.0.0/16"},
		StrategyFirstFit,
		nil,
		map[string]AllocationSpec{"web": {Count: 1, RequiredHosts: 1000}},
		map[string]string{"web-0": "10.0.5.0/28"}, // only 14 usable - far short of 1000
	)
	if err != nil {
		t.Fatalf("PlanSubnets returned error: %v", err)
	}
	r := results["web-0"]
	if r.Status != "insufficient" || r.CIDR != "10.0.5.0/28" {
		t.Errorf("web-0 = %+v, want status=insufficient cidr=10.0.5.0/28 (preserved, not reallocated)", r)
	}
}

func TestPlanSubnetsFailsGracefullyWhenFull(t *testing.T) {
	results, err := PlanSubnets(
		[]string{"10.0.0.0/24"}, // only 256 addresses total
		StrategyFirstFit,
		nil,
		map[string]AllocationSpec{"web": {Count: 2, RequiredHosts: 200}}, // two /24-sized requests - only one can fit
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("PlanSubnets returned error: %v", err)
	}
	if results["web-0"].Status != "allocated" {
		t.Errorf("web-0 status = %q, want allocated", results["web-0"].Status)
	}
	if results["web-1"].Status != "failed" {
		t.Errorf("web-1 status = %q, want failed", results["web-1"].Status)
	}
}

func TestPlanSubnetsSequentialSkipsEarlierGap(t *testing.T) {
	// Reserving 10.0.0.32/27 out of 10.0.0.0/24 leaves free fragments
	// [10.0.0.0/27, 10.0.0.64/26, 10.0.0.128/25]. A first request needing
	// a /26 is too big for the /27 fragment, so both strategies place it
	// at 10.0.0.64/26 - advancing sequential's frontier to 10.0.0.128.
	// A second, small request (needs only a /28) now fits the untouched
	// 10.0.0.0/27 fragment: first_fit happily reuses it (it's still
	// free and big enough), but sequential refuses - that fragment lies
	// below the frontier - and is pushed into 10.0.0.128/25 instead.
	base := []string{"10.0.0.0/24"}
	reserved := []string{"10.0.0.32/27"}
	allocations := map[string]AllocationSpec{
		"big":   {Count: 1, RequiredHosts: 60},
		"small": {Count: 1, RequiredHosts: 10},
	}

	seqResults, err := PlanSubnets(base, StrategySequential, reserved, allocations, map[string]string{})
	if err != nil {
		t.Fatalf("PlanSubnets (sequential) returned error: %v", err)
	}
	ffResults, err := PlanSubnets(base, StrategyFirstFit, reserved, allocations, map[string]string{})
	if err != nil {
		t.Fatalf("PlanSubnets (first_fit) returned error: %v", err)
	}

	if seqResults["big-0"].CIDR != "10.0.0.64/26" || ffResults["big-0"].CIDR != "10.0.0.64/26" {
		t.Fatalf("both strategies should place big-0 at 10.0.0.64/26: seq=%q ff=%q", seqResults["big-0"].CIDR, ffResults["big-0"].CIDR)
	}

	if ffResults["small-0"].CIDR != "10.0.0.0/28" {
		t.Errorf("first_fit should reuse the untouched 10.0.0.0/27 fragment for small-0, got %q", ffResults["small-0"].CIDR)
	}
	if seqResults["small-0"].CIDR != "10.0.0.128/28" {
		t.Errorf("sequential should refuse the fragment below its frontier and place small-0 at 10.0.0.128/28, got %q", seqResults["small-0"].CIDR)
	}
}

func TestPlanSubnetsInvalidStrategy(t *testing.T) {
	_, err := PlanSubnets([]string{"10.0.0.0/16"}, "worst_strategy_ever", nil,
		map[string]AllocationSpec{"a": {Count: 1, RequiredHosts: 10}}, map[string]string{})
	if err == nil {
		t.Error("expected an error for an invalid allocation_strategy, got nil")
	}
}

func TestPlanSubnetsMixedFamilyRejected(t *testing.T) {
	_, err := PlanSubnets([]string{"10.0.0.0/16", "2001:db8::/32"}, StrategyFirstFit, nil,
		map[string]AllocationSpec{"a": {Count: 1, RequiredHosts: 10}}, map[string]string{})
	if err == nil {
		t.Error("expected an error for mixed-family base_cidrs, got nil")
	}
}
