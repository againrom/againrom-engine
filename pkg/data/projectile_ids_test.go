package data

import (
	"slices"
	"testing"
)

func TestProjectileIDsAreSortedIndependentAddresses(t *testing.T) {
	p, err := LoadProjectiles(projReg(t, 3, projRow(1007, "far"), projRow(3, "near"), projRow(81, "middle")))
	if err != nil {
		t.Fatal(err)
	}
	want := []int32{3, 81, 1007}
	if got := p.IDs(); !slices.Equal(got, want) {
		t.Fatalf("addresses %v, want %v", got, want)
	}
	ids := p.IDs()
	ids[0] = 900
	if got := p.IDs(); !slices.Equal(got, want) {
		t.Fatalf("caller changed loaded addresses: %v", got)
	}
	if got := (*Projectiles)(nil).IDs(); got != nil {
		t.Fatalf("absent registry addresses %v", got)
	}
}
