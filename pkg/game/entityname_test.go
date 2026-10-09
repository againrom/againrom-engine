package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The class name's second hop: onto the per-tick snapshot, from the entity's
// OWN class and never from the one the death path substitutes for drawing
// (AC-2, AC-3).

const (
	// A live class and the DIFFERENTLY NAMED class its bodies are drawn from.
	// The two names differ because that is the only configuration in which
	// reading the name off Art is distinguishable from reading it off the
	// entity's own class — over shipped data the common case is a class that
	// names itself, where the defect would be invisible.
	nameWalker = 1
	nameBones  = 2
	nameNobody = 77

	walkerName    = "Walker"
	bonesName     = "Bones"
	walkerCaption = "Localized walker"
	bonesCaption  = "Localized bones"
)

// nameBundle links the walker's bodies to the differently named corpse class,
// through the production resolution rather than by hand.
func nameBundle() *terrain.UnitSet {
	live := deathLiveArt()
	live.Name = walkerName
	corpse := deathCorpseArt()
	corpse.Name = bonesName

	classes := map[int32]*terrain.UnitClass{nameWalker: live, nameBones: corpse}
	linkCorpses(classes, map[int32]int32{nameWalker: nameBones, nameBones: nameBones})
	return &terrain.UnitSet{Classes: classes}
}

// nameDraws is one snapshot over a world holding the four cases at once.
func nameDraws(t *testing.T, ents ...sim.Entity) map[uint32]struct {
	name string
	art  *terrain.UnitClass
} {
	t.Helper()
	grid := make([]byte, deathW*deathH)
	w, err := sim.NewWorld(deathSeed, sim.Bounds{Width: deathW, Height: deathH}, sim.ModeCanonical, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("name", terrain.Grid{
		Width: deathW, Height: deathH, Tiles: make([]uint16, deathW*deathH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	words := ui.AuthoredWords()
	words.UnitNames[nameWalker], words.UnitNames[nameBones] = walkerCaption, bonesCaption
	v.SetWords(words)
	mw := newMapWorld(w, nil, nameBundle(), v)
	out := make(map[uint32]struct {
		name string
		art  *terrain.UnitClass
	})
	for _, d := range mw.entityDraws() {
		out[d.ID] = struct {
			name string
			art  *terrain.UnitClass
		}{d.Name, d.Art}
	}
	return out
}

func TestSnapshotCarriesTheEntitysOwnClassName(t *testing.T) {
	draws := nameDraws(t,
		// Alive, downed and dead, all of the walker class; and one whose id
		// names no class in the bundle.
		sim.Entity{ID: 1, X: 1, Y: 1, Class: nameWalker, TypeID: nameWalker, HP: 100, MaxHP: 100},
		sim.Entity{ID: 2, X: 2, Y: 1, Class: nameWalker, TypeID: nameWalker, HP: 0, MaxHP: 100},
		sim.Entity{ID: 3, X: 3, Y: 1, Class: nameWalker, TypeID: nameWalker, HP: -1, MaxHP: 100},
		sim.Entity{ID: 4, X: 4, Y: 1, Class: nameNobody, TypeID: nameNobody, HP: 100, MaxHP: 100},
	)

	if len(draws) != 4 {
		t.Fatalf("the snapshot carries %d entries, want 4", len(draws))
	}

	for _, tc := range []struct {
		id       uint32
		want     string
		why      string
		substnow bool
	}{
		{1, walkerCaption, "a live unit carries its own installed caption", false},
		{2, walkerCaption, "a DOWNED unit is drawn as a body and is still a walker", true},
		{3, walkerCaption, "a DEAD unit is drawn as a body and is still a walker", true},
		{4, "", "an id naming no class carries the empty name", false},
	} {
		got := draws[tc.id]
		if got.name != tc.want {
			t.Errorf("entity %d: Name = %q, want %q — %s", tc.id, got.name, tc.want, tc.why)
		}
		if !tc.substnow {
			continue
		}
		// The substitution must actually have happened, or the two rows above
		// prove nothing: a test in which Art was never replaced would pass
		// with the name read off Art.
		if got.art == nil || got.art.Name != bonesName {
			t.Errorf("entity %d: Art is not the corpse class (%v) — this case did not exercise the "+
				"substitution, so it cannot witness that the name escaped it", tc.id, got.art)
		}
	}
}
