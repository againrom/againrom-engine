package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

// facingMap is the join fixture's two placed actors on a map with two drop
// cells, so the join walk's drop pick takes one draw after the placements.
func facingMap(t *testing.T) *alm.Map {
	t.Helper()
	m := startMap(t, 40, 40, mapload.Cell{X: 5, Y: 30}, mapload.Cell{X: 30, Y: 5})
	m.Units = joinMap().Units
	return m
}

func facingStart(t *testing.T, m *alm.Map, draws *sim.Draws) (*sim.World, mapload.Start) {
	t.Helper()
	w, st, err := mapload.StartCampaignMissionScriptedWith(m, joinTable(), mapload.DifficultyNormal,
		joinParty(), 0, nil, draws)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return w, st
}

// TestOriginalStreamDrawsOneFacingPerPlacedActorBeforeTheDrop pins the draw
// order on the original's stream (MOVE-119, MOVE-120, MOVE-122): each map
// actor's constructor draws 0x40 plus Range(0x80) in record order, then the
// join walk draws the drop cell. The expected values come from an independent
// MSVC sequence at the same state.
func TestOriginalStreamDrawsOneFacingPerPlacedActorBeforeTheDrop(t *testing.T) {
	const state = 0x1234567
	cells := mapload.DropCells(facingMap(t))
	for _, seed := range []uint32{state, state + 1, state + 2} {
		ref := random.MSVC{State: seed}
		want := []uint8{uint8(0x40 + ref.Range(0x80)), uint8(0x40 + ref.Range(0x80))}
		drop := cells[ref.Range(int32(len(cells)-1))]
		w, st := facingStart(t, facingMap(t), sim.NewOriginalDraws(seed))
		ents := w.Entities()
		for i, f := range want {
			if ents[i].Facing != f || ents[i].DesiredFacing != f {
				t.Errorf("state %#x: placed actor %d faces %#x wanting %#x, want %#x", seed, i,
					ents[i].Facing, ents[i].DesiredFacing, f)
			}
		}
		if st.Drop != drop {
			t.Errorf("state %#x: drop %v, want %v after the two facing draws", seed, st.Drop, drop)
		}
		if hero := ents[st.IDs[0]]; hero.Facing != 0 || hero.DesiredFacing != 0 {
			t.Errorf("state %#x: the carried hero faces %#x, want the join walk's constructor 0", seed, hero.Facing)
		}
	}
}

// TestSeededFacingsAreKeyedAndShiftNoPlacementDraw: in seeded mode the same
// start gives the same facings, every facing lies in 0x40..0xC0, and the drop
// pick and the World stream are those of the same map with no placed actor.
func TestSeededFacingsAreKeyedAndShiftNoPlacementDraw(t *testing.T) {
	a, sa := facingStart(t, facingMap(t), nil)
	b, _ := facingStart(t, facingMap(t), nil)
	for i, e := range a.Entities()[:2] {
		if e.Facing < 0x40 || e.Facing > 0xC0 || e.DesiredFacing != e.Facing {
			t.Errorf("placed actor %d faces %#x wanting %#x, want 0x40..0xC0 and equal", i, e.Facing, e.DesiredFacing)
		}
		if e.Facing != b.Entities()[i].Facing {
			t.Errorf("placed actor %d faces %#x then %#x from one seed", i, e.Facing, b.Entities()[i].Facing)
		}
	}
	if a.Hash() != b.Hash() {
		t.Error("two seeded starts of one map hash differently")
	}
	bare := facingMap(t)
	bare.Units = nil
	c, sc := facingStart(t, bare, nil)
	if sa.Drop != sc.Drop || a.RandomState() != c.RandomState() {
		t.Errorf("the placed actors moved the seeded draws: drop %v/%v, World stream %#x/%#x",
			sa.Drop, sc.Drop, a.RandomState(), c.RandomState())
	}
}

// TestOriginalStreamDrawsAFacingForAWithdrawnRingPlacement: the original
// constructs a ring placement, drawing its facing, before the cell refusal
// deletes it (MOVE-120, MOVE-141). The withdrawn record between two ordinary
// ones takes the second draw; the actors after it and the drop pick follow.
func TestOriginalStreamDrawsAFacingForAWithdrawnRingPlacement(t *testing.T) {
	const state = 0x1234567
	build := func(withdraw bool) *alm.Map {
		m := facingMap(t)
		ordinary := m.Units
		x, y := cellAt(0, 20)
		ring := alm.Unit{X: x, Y: y, UnitID: 900}
		m.AuthoredUnits = []alm.Unit{ordinary[0], ring, ordinary[1]}
		m.Units = []alm.Unit{ordinary[0], ordinary[1]}
		if !withdraw {
			m.AuthoredUnits = m.Units
		}
		return m
	}
	cells := mapload.DropCells(build(true))
	differs := 0
	for _, seed := range []uint32{state, state + 1, state + 2} {
		ref := random.MSVC{State: seed}
		first := uint8(0x40 + ref.Range(0x80))
		_ = ref.Range(0x80) // the withdrawn actor's, discarded with it
		second := uint8(0x40 + ref.Range(0x80))
		drop := cells[ref.Range(int32(len(cells)-1))]
		w, st := facingStart(t, build(true), sim.NewOriginalDraws(seed))
		ents := w.Entities()
		if ents[0].Facing != first || ents[1].Facing != second {
			t.Errorf("state %#x: facings %#x %#x, want %#x %#x", seed, ents[0].Facing, ents[1].Facing, first, second)
		}
		if st.Drop != drop {
			t.Errorf("state %#x: drop %v, want %v after three facing draws", seed, st.Drop, drop)
		}
		// Loss control: with the withdrawal unrecorded (the old behaviour) the
		// second actor and the drop come from one draw fewer.
		ow, ost := facingStart(t, build(false), sim.NewOriginalDraws(seed))
		if ow.Entities()[1].Facing != second || ost.Drop != drop {
			differs++
		}
	}
	if differs == 0 {
		t.Error("the unrecorded withdrawal drew the same facings and drop on every seed; the control has no force")
	}
}
