package netutil

import (
	"reflect"
	"testing"
)

func TestCIDROverlapPairs(t *testing.T) {
	cidrs := []string{"10.0.0.0/16", "10.0.128.0/17", "192.168.0.0/16", "10.0.0.0/24"}
	got, err := CIDROverlapPairs(cidrs)
	if err != nil {
		t.Fatalf("CIDROverlapPairs returned error: %v", err)
	}
	want := [][2]int{{0, 1}, {0, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CIDROverlapPairs(%v) = %v, want %v", cidrs, got, want)
	}
}

func TestCIDROverlapPairsNoOverlap(t *testing.T) {
	cidrs := []string{"10.0.0.0/24", "10.0.1.0/24"}
	got, err := CIDROverlapPairs(cidrs)
	if err != nil {
		t.Fatalf("CIDROverlapPairs returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("CIDROverlapPairs(%v) = %v, want empty", cidrs, got)
	}
}

func TestPortRangeValidate(t *testing.T) {
	cases := []struct {
		from, to int
		want     bool
	}{
		{80, 80, true},
		{80, 443, true},
		{443, 80, false},
		{0, 100, false},
		{1, 65535, true},
		{1, 65536, false},
	}
	for _, c := range cases {
		got := PortRangeValidate(c.from, c.to)
		if got != c.want {
			t.Errorf("PortRangeValidate(%d, %d) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestPortRangeOverlap(t *testing.T) {
	cases := []struct {
		aFrom, aTo, bFrom, bTo int
		want                   bool
	}{
		{80, 443, 443, 8443, true},
		{80, 443, 444, 8443, false},
		{1, 65535, 8080, 8080, true},
	}
	for _, c := range cases {
		got, err := PortRangeOverlap(c.aFrom, c.aTo, c.bFrom, c.bTo)
		if err != nil {
			t.Errorf("PortRangeOverlap(%d,%d,%d,%d) returned error: %v", c.aFrom, c.aTo, c.bFrom, c.bTo, err)
			continue
		}
		if got != c.want {
			t.Errorf("PortRangeOverlap(%d,%d,%d,%d) = %v, want %v", c.aFrom, c.aTo, c.bFrom, c.bTo, got, c.want)
		}
	}

	if _, err := PortRangeOverlap(0, 100, 1, 2); err == nil {
		t.Error("PortRangeOverlap with an invalid range expected an error, got nil")
	}
}
