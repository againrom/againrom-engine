package game

import (
	"testing"

	"againrom/pkg/sim"
)

// The ground drop's own tier-above (1005 round 2, `ITEM-DROP-008`,
// `ITEM-CMD-007`'s destination code 3): enqueueDrop turns a drained
// ground-drop request into a sim.Command, equipMission's own fixture
// (equip_test.go) reused rather than a second one.

// TestEnqueueDropCarriedAppendsACommandForTheNamedElement is enqueueEquip's
// own AC-8 restated for the pack origin: the container element index a
// drained request named turns into KindDropCarried at the release cell,
// with no table consulted — dropping is never refused on what the item is.
func TestEnqueueDropCarriedAppendsACommandForTheNamedElement(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode, eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, nil) // no table: a drop consults none

	mw.enqueueDrop(false, 1, 5, 5) // inside the window, on an open cell: the release cell

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command", mw.pending)
	}
	want := sim.Command{Kind: sim.KindDropCarried, Entity: 7, X: 5, Y: 5, Spell: 1}
	if got := mw.pending[0]; got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

// TestEnqueueDropWornAppendsACommandForTheNamedSlot is the doll origin's own
// case: a zero-based worn-box index is widened by one into equip.go's own
// 1..12 numbering, enqueueUnequip's own convention (KindDropWorn's Spell
// carries the same numbering dropFromEquipment reads, drop.go).
func TestEnqueueDropWornAppendsACommandForTheNamedSlot(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 1}}) // slot 1 now holds the sword
	mw := equipMission(t, w, 7, eqHero(), nil, nil)

	mw.enqueueDrop(true, 0, 5, 5) // worn cell 0 (Slots' own zero-based index for slot 1)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command", mw.pending)
	}
	want := sim.Command{Kind: sim.KindDropWorn, Entity: 7, X: 5, Y: 5, Spell: 1}
	if got := mw.pending[0]; got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

// TestEnqueueDropDoesNothingWithNoSubject is enqueueEquip's own first guard
// (world.go's doc), restated for the drop: a mapWorld with no inventory
// subject appends nothing, on either origin.
func TestEnqueueDropDoesNothingWithNoSubject(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, nil)
	mw.invSubjectSet = false

	mw.enqueueDrop(false, 0, 5, 5)
	mw.enqueueDrop(true, 0, 5, 5)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — no subject is set", mw.pending)
	}
}

// TestEnqueueDropWornDoesNothingForAnOutOfRangeSlot is enqueueUnequip's own
// defensive bound (equip_test.go), restated for the doll origin: every idx
// TakeInventoryDrop can hand back for a worn origin already names one of the
// twelve worn cells, so this is a fence rather than a reachable case.
func TestEnqueueDropWornDoesNothingForAnOutOfRangeSlot(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, nil)

	for _, idx := range []int{-1, sim.EquipSlots} {
		mw.enqueueDrop(true, idx, 5, 5)
	}

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — both slots are out of range", mw.pending)
	}
}

// TestEnqueueDropCarriedDoesNothingForANegativeIndex is the pack origin's
// own defensive bound: enqueueDrop's own idx<0 guard, the one shape a
// negative element index could take before reaching pkg/sim's own bounds
// check (drop.go), witnessed here rather than only there since a negative
// idx would otherwise wrap through the uint16 Spell conversion.
func TestEnqueueDropCarriedDoesNothingForANegativeIndex(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, nil)

	mw.enqueueDrop(false, -1, 5, 5)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — a negative element index is refused before it can wrap through Spell", mw.pending)
	}
}

// TestEnqueueDropAimsAThrowBeyondTheWindowBesideTheThrower: a release outside
// the 5x5 window, or on the thrower, would plant the sack on the thrower's own
// cell, beneath the thrower, where it cannot be seen. The command is aimed at
// the neighbour south of the thrower instead; a release on an open cell inside
// the window is aimed where it was released.
func TestEnqueueDropAimsAThrowBeyondTheWindowBesideTheThrower(t *testing.T) {
	for _, release := range [][2]int32{{9, 9}, {3, 3}, {1 << 20, 1 << 20}} {
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
			[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
		if err != nil {
			t.Fatalf("NewStockedWorld: %v", err)
		}
		mw := equipMission(t, w, 7, eqHero(), nil, nil)

		mw.enqueueDrop(false, 0, release[0], release[1])

		want := sim.DropCarried(7, 0, sim.CellPoint{X: 3, Y: 4})
		if len(mw.pending) != 1 || mw.pending[0] != want {
			t.Errorf("release %v: pending = %+v, want %+v", release, mw.pending, want)
		}
		mw.tick()
		sacks := w.Sacks()
		if len(sacks) != 1 || sacks[0].X != 3 || sacks[0].Y != 4 {
			t.Errorf("release %v: sacks %+v, want one sack at 3,4 beside the thrower at 3,3", release, sacks)
		}
	}
}
