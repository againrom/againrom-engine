package ui

// The seam's own half of the life state: what the three constants are, and that
// this package derives none of it.
//
// What crosses the seam is checked on the far side, in pkg/game, where a world
// exists to be classified. What is checked HERE is the shape of what arrives —
// because the shape is this package's contract, and the zero value in particular
// is what every MapEntity written before this story depends on.

import (
	"reflect"
	"testing"
)

func TestTheThreeLifeConstantsAreDistinctAndAliveIsTheZeroValue(t *testing.T) {
	if LifeAlive != 0 {
		t.Errorf("LifeAlive is %d; an entry that names no life must be alive", LifeAlive)
	}
	seen := map[uint8]string{}
	for _, tc := range []struct {
		name string
		v    uint8
	}{
		{"LifeAlive", LifeAlive},
		{"LifeDowned", LifeDowned},
		{"LifeDead", LifeDead},
	} {
		if other, ok := seen[tc.v]; ok {
			t.Errorf("%s and %s are both %d", other, tc.name, tc.v)
		}
		seen[tc.v] = tc.name
	}
}

func TestAnEntryArrivesCarryingWhatItWasHandedAndNothingIsRecomputed(t *testing.T) {
	given := []MapEntity{
		{ID: 1, Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: 2, Life: LifeDowned, HP: 0, MaxHP: 100},
		{ID: 3, Life: LifeDead, HP: -40, MaxHP: 100},
		// Say dead of a unit at full health, and downed of one below zero. A
		// viewer reading the pair instead of the byte disagrees with both.
		{ID: 4, Life: LifeDead, HP: 100, MaxHP: 100},
		{ID: 5, Life: LifeDowned, HP: -7, MaxHP: 100},
		// And a pair naming no health system at all, which is alive.
		{ID: 6, Life: LifeAlive, HP: 0, MaxHP: 0},
	}

	v := &Viewer{}
	v.SetEntities(given)
	if got := v.EntityMarkers(); got != len(given) {
		t.Fatalf("the viewer holds %d entries, want %d", got, len(given))
	}
	for i, want := range given {
		if got := v.entities[i]; !reflect.DeepEqual(got, want) {
			t.Errorf("entry %d is %+v, want %+v — the viewer stores what it was handed", i, got, want)
		}
	}
}
