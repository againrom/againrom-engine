package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	swingW, swingH = 10, 10

	// The one class's blocks, written out: a move block at 0, an idle block at
	// 8 and an ATTACK block at 24, each direction slot one frame wide so a
	// selected index names its own octant directly. The attack track is three
	// ticks long, which is the whole of the run.
	swingIdleBase, swingAttackBase = 8, 24
	swingRun                       = 3
)

func swingAnimDesc() terrain.UnitAnim {
	return terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 0, AttackBase: swingAttackBase, DyingBase: 60, TailBase: swingIdleBase,
		MoveSlot: 1, MoveWind: 0, IdleSlot: 1, AttackSlot: swingRun, Total: 64,
		MoveTrack: []int{0}, MoveOK: true,
		IdleTrack: []int{0}, IdleOK: true,
		AttackTrack: []int{0, 1, 2}, AttackOK: true}
}

// swingWorld is a driver over a hand-built world under the descriptor above.
func swingWorld(t *testing.T, anim terrain.UnitAnim, ents ...sim.Entity) *mapWorld {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
	c.Anim = anim
	c.Corpse = c
	v, err := ui.NewViewer("swing", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: c}}, v)
}

// swingUnit is a unit that can hold a fight, with a cycle long enough that the
// runs below are told apart by the clock rather than by a blow landing.
func swingUnit(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: 1, HP: 200, MaxHP: 200,
		AttackCharge: 8, AttackRelax: 8}
}

func swingDraw(t *testing.T, mw *mapWorld, id sim.EntityID) ui.MapEntity {
	t.Helper()
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(id) {
			return d
		}
	}
	t.Fatalf("the push carries no entry for entity %d", id)
	return ui.MapEntity{}
}

// TestAnAttackerDrawsItsSwingAndThenStands is AC-10 and AC-11 together: an
// attacker beside its victim draws the attack block for exactly the run's own
// length and the standing or idle drawing afterwards, and it points at what it
// is hitting.
//
// The run is asserted as a LENGTH and not as one frame: the decoded arm plays
// the art once and is then forced back to the standing state, so a build that
// looped the swing would draw an attack frame on every tick here.
func TestAnAttackerDrawsItsSwingAndThenStands(t *testing.T) {
	// East of the attacker: sheet octant 6.
	const east = 6
	mw := swingWorld(t, swingAnimDesc(), swingUnit(1, 4, 4), swingUnit(2, 5, 4))
	mw.strike(1, 2)

	art := mw.units.Classes[1]
	frameAt := func(i int) *terrain.StaticFrame { return art.Frames[i] }

	seen := 0
	for tick := 0; tick < swingRun+6; tick++ {
		mw.tick()
		d := swingDraw(t, mw, 1)
		switch {
		case tick < swingRun:
			want := frameAt(swingAttackBase + east*swingRun + tick)
			if d.Frame != want {
				t.Errorf("tick %d: frame %p, want the run's own frame %p (attack base %d, octant %d, step %d)",
					tick, d.Frame, want, swingAttackBase, east, tick)
			}
			seen++
		default:
			// The run is over, so the attack selection refuses and the live one
			// answers: the idle block at this entity's own octant.
			want := frameAt(swingIdleBase + east)
			if d.Frame != want {
				t.Errorf("tick %d: frame %p, want the idle frame %p — the run plays once and ends",
					tick, d.Frame, want)
			}
		}
	}
	if seen != swingRun {
		t.Errorf("the run drew %d frames, want %d", seen, swingRun)
	}
	// And it turned: the drawn octant is the victim's direction, which is what a
	// player watching the fight sees.
	if got := mw.world.Entities()[0].Facing; sheetOctant(got) != east {
		t.Errorf("the attacker faces sheet octant %d, want %d", sheetOctant(got), east)
	}
}

// TestAClassWithNoAttackBlockFallsThrough is AC-10's last clause: the swing
// selection refuses and the entity draws exactly what it drew before a swing
// could be drawn at all.
func TestAClassWithNoAttackBlockFallsThrough(t *testing.T) {
	const east = 6
	anim := swingAnimDesc()
	anim.AttackTrack, anim.AttackOK, anim.AttackSlot = nil, false, 0

	mw := swingWorld(t, anim, swingUnit(1, 4, 4), swingUnit(2, 5, 4))
	mw.strike(1, 2)
	for tick := 0; tick < 4; tick++ {
		mw.tick()
		if got, want := swingDraw(t, mw, 1).Frame, mw.units.Classes[1].Frames[swingIdleBase+east]; got != want {
			t.Fatalf("tick %d: frame %p, want the idle frame %p", tick, got, want)
		}
	}
}

// TestACorpseNeverSwings is AC-10's ordering clause. The death path is tried
// first and a felled unit holds no victim anyway, so this pins BOTH halves at
// once: the drawing is the corpse class's dying block and not an attack frame.
func TestACorpseNeverSwings(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingUnit(1, 4, 4), swingUnit(2, 5, 4))
	mw.strike(1, 2)
	mw.tick()
	mw.affect(1, true) // fell the attacker mid-run
	mw.tick()          // and the queue is applied by the advance, like any order

	d := swingDraw(t, mw, 1)
	if d.Life != ui.LifeDead {
		t.Fatalf("the attacker reports life %d, want dead", d.Life)
	}
	art := mw.units.Classes[1]
	for i := swingAttackBase; i < swingAttackBase+8*swingRun; i++ {
		if d.Frame == art.Frames[i] {
			t.Fatalf("a corpse drew the attack block's frame %d", i)
		}
	}
}

func TestTheSwingClockReachesNoWorld(t *testing.T) {
	drive := func(draw bool) uint64 {
		mw := swingWorld(t, swingAnimDesc(), swingUnit(1, 4, 4), swingUnit(2, 5, 4))
		mw.strike(1, 2)
		for tick := 0; tick < 24; tick++ {
			mw.tick()
			if draw {
				mw.entityDraws()
				mw.entityDraws()
			}
		}
		return mw.world.Hash()
	}
	if a, b := drive(false), drive(true); a != b {
		t.Errorf("drawing moved the digest: %#016x without, %#016x with", a, b)
	}
}

// TestASheetOctantIsTheSimulationsOwnDirection is AC-9. The translation is
// asserted over all eight deltas against this package's own independent
// transcription of the sheet ordering (facingOctant), never against the
// production table it replaced.
func TestASheetOctantIsTheSimulationsOwnDirection(t *testing.T) {
	for _, tc := range []struct{ dx, dy int32 }{
		{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	} {
		// The facing a simulation walks away with after a step of this delta,
		// built here through the world rather than through any conversion: one
		// unit, one order, one tick.
		w, err := sim.NewWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil,
			[]sim.Entity{{ID: 1, X: 4, Y: 4}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		sim.Step(w, []sim.Command{{Entity: 1, X: 4 + tc.dx, Y: 4 + tc.dy}})
		e := w.Entities()[0]
		if e.X != 4+tc.dx || e.Y != 4+tc.dy {
			t.Fatalf("(%d,%d): the unit did not take its step", tc.dx, tc.dy)
		}
		if got, want := sheetOctant(e.Facing), facingOctant(int(tc.dx), int(tc.dy)); got != want {
			t.Errorf("(%d,%d): the simulation's facing translates to sheet octant %d, and this "+
				"package's own table gives %d for the same delta", tc.dx, tc.dy, got, want)
		}
	}
}
