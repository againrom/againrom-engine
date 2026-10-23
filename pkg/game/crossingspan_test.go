package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A Teleport moves its caster without a walk, so the span drawn for the jump
// is the one a caster that had not walked gets, whatever TransitTotal kept.
func TestCrossingSpanOfARelocatedCasterIgnoresTheLastWalk(t *testing.T) {
	mw := &mapWorld{}
	walked := sim.Entity{ID: 7, TransitTotal: 29}
	if got := mw.crossingSpan(walked); got != 29 {
		t.Fatalf("a walker's span = %d, want 29", got)
	}
	mw.noteRelocated([]sim.Relocation{{Caster: 7}})
	if got := mw.crossingSpan(walked); got != 0 {
		t.Fatalf("a relocated caster's span = %d, want 0", got)
	}
	if got := mw.crossingSpan(sim.Entity{ID: 8, TransitTotal: 29}); got != 29 {
		t.Fatalf("another entity's span = %d, want 29", got)
	}
	mw.noteRelocated(nil)
	if got := mw.crossingSpan(walked); got != 29 {
		t.Fatalf("span after the next step = %d, want 29", got)
	}
}
