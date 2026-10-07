package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The hand-built world's extent, and the row the corridor fixture leaves open.
const (
	facingW, facingH = 8, 8
	facingRow        = 2
)

// facingSeed is the seed every world here is built on. Nothing in this file
// reads the RNG — no route is contended for in a corridor and no give-up is
// driven — so the value only has to be fixed.
const facingSeed = 1

// facingAnim is an OCTANT-LEGIBLE descriptor, every field a literal: at D 8 no
// selection mirrors and no slot folds, and both tracks have period 1, so the
// scene clock cancels out and the frame a push carries names the facing and the
// walk/idle answer together —
//
//	moving: MoveBase 0 + oct*1 + 0 + 0   = frames 0..7
//	idle:   TailBase 8 + oct*1 + 0       = frames 8..15
//
// which is what lets every assertion below read WHICH direction was drawn out
// of one frame pointer instead of out of a second derivation.
func facingAnim() terrain.UnitAnim {
	return terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 0, AttackBase: 16, DyingBase: 16, TailBase: 8,
		MoveSlot: 1, MoveWind: 0, IdleSlot: 1, Total: 16,
		MoveTrack: []int{0}, MoveOK: true,
		IdleTrack: []int{0}, IdleOK: true}
}

// facingClass is the one class every entity here resolves against: a sixteen
// frame sheet under that descriptor, each frame its OWN pointer, so a test can
// say which frame was selected without re-deriving an index.
func facingClass() *terrain.UnitClass {
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, 16)
	c.Anim = facingAnim()
	return c
}

// facingOctant is the spec's own eight movement facings — S 0, SW 1, W 2, NW 3,
// N 4, NE 5, E 6, SE 7, screen +y south — written out here as a switch over the
// signs of a delta. It is this file's independent transcription and is never the
// driver's table.
func facingOctant(dx, dy int) int {
	switch {
	case dy > 0 && dx == 0:
		return 0
	case dy > 0 && dx < 0:
		return 1
	case dy == 0 && dx < 0:
		return 2
	case dy < 0 && dx < 0:
		return 3
	case dy < 0 && dx == 0:
		return 4
	case dy < 0 && dx > 0:
		return 5
	case dy == 0 && dx > 0:
		return 6
	default:
		return 7
	}
}

// facingWorld is a driver over a hand-built world: the given entities on an
// 8x8 grid with the given passability, the one class above behind all of them,
// and a viewer of the same extent so the constructor's tick-0 push has
// somewhere to go. No schedule — every order in this file is issued through the
// front-end's own seam.
func facingWorld(t *testing.T, grid []byte, ents ...sim.Entity) *mapWorld {
	t.Helper()
	w, err := sim.NewWorld(facingSeed, sim.Bounds{Width: facingW, Height: facingH}, sim.ModeCanonical, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	v, err := ui.NewViewer("facing", terrain.Grid{
		Width: facingW, Height: facingH, Tiles: make([]uint16, facingW*facingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: facingClass()}}
	return newMapWorld(w, nil, set, v)
}

// facingUnit is one entity of that world: a whole cell, class 1, at full health.
func facingUnit(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: 1, HP: 100, MaxHP: 100}
}

// facingCell is where the world says an entity stands, read through the world
// rather than through the push, so a step below is a difference between two
// facts about the world and not between two answers of the thing under test.
func facingCell(t *testing.T, mw *mapWorld, id sim.EntityID) image.Point {
	t.Helper()
	e, ok := mw.entity(id)
	if !ok {
		t.Fatalf("the world no longer holds entity %d", id)
	}
	return image.Point{X: int(e.X), Y: int(e.Y)}
}

// facingDraw is what the seam carries for one entity in the push the driver has
// most recently made.
func facingDraw(t *testing.T, mw *mapWorld, id sim.EntityID) ui.MapEntity {
	t.Helper()
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(id) {
			return d
		}
	}
	t.Fatalf("the push carries no entry for entity %d", id)
	return ui.MapEntity{}
}

// facingTick advances the driver exactly one tick and hands back the cell each
// named entity stood on BEFORE it — the reading every assertion below is made
// against, taken from the world and never from the memory under test.
func facingTick(t *testing.T, mw *mapWorld, ids ...sim.EntityID) map[sim.EntityID]image.Point {
	t.Helper()
	was := make(map[sim.EntityID]image.Point, len(ids))
	for _, id := range ids {
		was[id] = facingCell(t, mw, id)
	}
	mw.tick()
	return was
}

// facingRead judges one entity against the tick just run and reports whether it
// moved: the cell change the world performed, the step the push carries for it,
// and the frame that selected.
//
// want is the octant the caller expects the entity DRAWN facing — the step's own
// where it moved, the one it kept where it did not — so a wrong direction and a
// wrong walk/idle answer are both one frame pointer away.
func facingRead(t *testing.T, mw *mapWorld, id sim.EntityID, was image.Point, label string, want int) bool {
	t.Helper()

	delta := facingCell(t, mw, id).Sub(was)
	moved := delta != image.Point{}
	if moved {
		if got := facingOctant(delta.X, delta.Y); got != want {
			t.Fatalf("%s: entity %d stepped %v, which is octant %d and not the %d this line expects",
				label, id, delta, got, want)
		}
	}

	d := facingDraw(t, mw, id)
	if d.Step != delta {
		t.Errorf("%s: entity %d is pushed with step %v, want the step the world took, %v (FR-1)",
			label, id, d.Step, delta)
	}
	if d.Art == nil || len(d.Art.Frames) != 16 {
		t.Fatalf("%s: entity %d resolves to %v; every frame index here is stated over the 16-frame fixture sheet",
			label, id, d.Art)
	}
	frame, kind := 8+want, "idle"
	if moved {
		frame, kind = want, "walking"
	}
	if d.Frame != d.Art.Frames[frame] {
		t.Errorf("%s: entity %d draws frame %p, want its %s frame for octant %d, Frames[%d] %p (FR-1, FR-2)",
			label, id, d.Frame, kind, want, frame, d.Art.Frames[frame])
	}
	if d.Mirror {
		t.Errorf("%s: entity %d crossed mirrored; a D-8 sheet never mirrors", label, id)
	}
	return moved
}

// facingStep is the common case: one tick, one entity, and an expectation about
// whether it moved at all.
func facingStep(t *testing.T, mw *mapWorld, id sim.EntityID, label string, moved bool, want int) {
	t.Helper()
	was := facingTick(t, mw, id)
	if got := facingRead(t, mw, id, was[id], label, want); got != moved {
		t.Fatalf("%s: entity %d moved=%v over that tick; this line is written over moved=%v",
			label, id, got, moved)
	}
}

// TestAnAdjacentOrderTurnsTheUnitAndPlaysItsWalk — 0047 SC-1's AC-1 half:
// an order to an ADJACENT cell turns the unit and plays its walking frame,
// which is the owner's own report and the case the order-held classification
// could never get right.
//
// The world's own answer at that boundary is asserted first and is the whole
// reason the criterion exists: the target is already GONE by the time the push
// runs, because pkg/sim sets it and clears it on arrival inside one advance. A
// classification reading the order sees no target, draws the idle frame and
// leaves the unit facing wherever it faced before.
func TestAnAdjacentOrderTurnsTheUnitAndPlaysItsWalk(t *testing.T) {
	for _, tc := range []struct {
		name   string
		to     image.Point
		octant int
	}{
		{"east", image.Pt(3, 2), 6},
		{"north", image.Pt(2, 1), 4},
		{"south west", image.Pt(1, 3), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := facingWorld(t, nil, facingUnit(0, 2, 2))

			// The tick-0 push, before anything: no step, so the never-moved facing
			// and the idle frame.
			if got := facingDraw(t, mw, 0).Step; got != (image.Point{}) {
				t.Fatalf("the tick-0 push carries step %v for an entity that has never moved", got)
			}

			mw.enqueue(0, tc.to.X, tc.to.Y)
			facingStep(t, mw, 0, "the one-cell order", true, tc.octant)

			// The order is in force at NO tick boundary, which is what makes
			// the frame above unreachable from it.
			e, _ := mw.entity(0)
			if e.HasTarget {
				t.Fatalf("the entity still holds a target after arriving; this criterion is stated over the "+
					"boundary at which it does not, and the world put it on %v", facingCell(t, mw, 0))
			}
		})
	}
}

// TestEveryStepOfAWalkIsDrawnWalkingAndTheArrivalKeepsItsDirection — 0047
// SC-1's AC-2 half: over an order four cells away, every step INCLUDING the
// arrival step reads as a move in its own direction, and the advance after
// arrival reads idle and keeps the arrival direction.
//
// The arrival step is the second half of the defect the owner reported: it is
// the tick at which a longer walk's target is cleared, so under the order-held
// classification the last cell of every walk was drawn idle. The run turns a
// corner — four cells east, then four south — so "its own direction" is a
// direction that actually changes, and the idle frame after each leg is the leg's
// own and not a constant.
func TestEveryStepOfAWalkIsDrawnWalkingAndTheArrivalKeepsItsDirection(t *testing.T) {
	mw := facingWorld(t, nil, facingUnit(0, 2, 2))

	for _, leg := range []struct {
		name   string
		to     image.Point
		octant int
	}{
		{"east", image.Pt(6, 2), 6},
		{"south", image.Pt(6, 6), 0},
	} {
		mw.enqueue(0, leg.to.X, leg.to.Y)
		for i := 0; i < 4; i++ {
			label := leg.name + " step"
			if i == 3 {
				label = leg.name + " ARRIVAL step"
			}
			facingStep(t, mw, 0, label, true, leg.octant)
		}
		if got, want := facingCell(t, mw, 0), leg.to; got != want {
			t.Fatalf("after four %s steps the unit stands on %v, want %v", leg.name, got, want)
		}
		// The advance after arrival: no step, idle, still facing the way the
		// leg went.
		facingStep(t, mw, 0, leg.name+" the advance after arrival", false, leg.octant)
	}
}

// TestAnEntityThatTookNoStepIsIdleInTheDirectionItLastWent — 0047 SC-2
// (AC-3): the two ways an entity can take no step. One that has NEVER moved
// is idle in the never-moved direction throughout; one HELD UP for a tick by
// a neighbour standing in its way is idle on that tick and keeps the
// direction it was already going.
//
// The hold-up is made structural rather than hoped for: every cell outside one
// row is blocked, so the only route east runs through the cell the neighbour is
// standing on and there is nothing for a search to go around by.
func TestAnEntityThatTookNoStepIsIdleInTheDirectionItLastWent(t *testing.T) {
	t.Run("never moved", func(t *testing.T) {
		mw := facingWorld(t, nil, facingUnit(0, 2, 2), facingUnit(1, 5, 5))

		// A UNIT THAT HAS NEVER TURNED FACES NORTH, sheet octant 4, and this
		// assertion read 0 — south — until 0081. Neither is decoded, and the
		// change is the one visible consequence of the direction moving out of
		// this tier: the old default was the sign-octant table's centre cell,
		// reached because no step had ever been remembered here; the new one is
		// the simulation facing byte's zero, which is the movement delta table's
		// own index 0. Rewritten rather than deleted, because it is the shipped
		// statement of the behaviour that changed.
		//
		// The mover beside it walks the whole time, so "idle facing north" is
		// not what a driver that never selected anything would also say.
		mw.enqueue(1, 5, 2)
		for i := 0; i < 3; i++ {
			was := facingTick(t, mw, 0, 1)
			if facingRead(t, mw, 0, was[0], "the entity that never moved", 4) {
				t.Fatalf("the entity nothing ordered moved on tick %d", i+1)
			}
			if !facingRead(t, mw, 1, was[1], "its walking neighbour", 4) {
				t.Fatalf("the walking neighbour stood still on tick %d, so this run compares two idle "+
					"entities", i+1)
			}
		}
	})

	t.Run("held up for a tick", func(t *testing.T) {
		// Every cell blocked but one row, so the corridor east runs through the
		// cell the neighbour stands on and a search has nothing to go round by.
		grid := make([]byte, facingW*facingH)
		for i := range grid {
			grid[i] = 1 // blockGround: bit 0, this file's own spelling
		}
		for col := 0; col < facingW; col++ {
			grid[facingRow*facingW+col] = 0
		}

		mw := facingWorld(t, grid, facingUnit(0, 0, facingRow), facingUnit(1, 5, facingRow))
		mw.enqueue(0, 7, facingRow)

		steps, held := 0, false
		for i := 0; i < 8 && !held; i++ {
			was := facingTick(t, mw, 0)
			label := "walking east"
			if steps > 0 {
				label = "held up behind the neighbour"
			}
			if facingRead(t, mw, 0, was[0], label, 6) {
				steps++
				continue
			}
			// The first tick that moves it nowhere: idle, still facing east —
			// which facingRead has just judged against octant 6.
			held = steps > 0
			// And it is a HOLD-UP and not a walk that ended: the unit still
			// holds the target it could not advance toward, which is the very
			// state the order-held classification drew WALKING while the unit
			// stood still.
			if e, _ := mw.entity(0); held && !e.HasTarget {
				t.Fatalf("the walker stopped without a target, so it gave its order up rather than being "+
					"held up; it stands on %v", facingCell(t, mw, 0))
			}
		}
		if steps == 0 || !held {
			t.Fatalf("the walker took %d step(s) and was held up=%v; this criterion needs a unit that walked "+
				"and was THEN stopped, and it stands on %v", steps, held, facingCell(t, mw, 0))
		}

		// The neighbour never moved at all, so what held the walker up was a
		// unit standing still and not one that walked away.
		if got, want := facingCell(t, mw, 1), image.Pt(5, facingRow); got != want {
			t.Fatalf("the neighbour stands on %v, want %v", got, want)
		}
	})
}
