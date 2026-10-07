package sim

// Which way a unit is pointing: the three conversions, the one write a step
// makes, and the field's passage through the byte form.
//
// The eight directions are written out HERE as compass names and cell deltas,
// independently of facing.go's own tables, so a table transposed in the
// production file is a failure and not a shared assumption. Nothing below reads
// stepOf to decide what it expects; stepOf is checked AGAINST this file's own
// list, once, and everything else is expected against compass names.

import (
	"testing"
)

// theEight is the direction table as MOVE-DIR-034 states it, transcribed a
// second time: index, compass name, and the cell delta that direction moves by
// on a lattice whose +y is south.
var theEight = []struct {
	dir    int
	name   string
	dx, dy int32
}{
	{0, "north", 0, -1},
	{1, "north-east", +1, -1},
	{2, "east", +1, 0},
	{3, "south-east", +1, +1},
	{4, "south", 0, +1},
	{5, "south-west", -1, +1},
	{6, "west", -1, 0},
	{7, "north-west", -1, -1},
}

// TestTheDirectionTableIsTheDecodedOne is AC-2's first half, and it is the check
// every other case in this file rests on: production's own delta table must be
// the one the claim states, transcribed above from the claim rather than from
// the code.
func TestTheDirectionTableIsTheDecodedOne(t *testing.T) {
	if len(stepOf) != len(theEight) {
		t.Fatalf("the table holds %d directions, want %d", len(stepOf), len(theEight))
	}
	for _, c := range theEight {
		if got := stepOf[c.dir]; got[0] != c.dx || got[1] != c.dy {
			t.Errorf("direction %d (%s) steps by (%d,%d), want (%d,%d)",
				c.dir, c.name, got[0], got[1], c.dx, c.dy)
		}
	}
}

// TestFacingDirIsTotalOverEveryByte is AC-2. The rounding answers in [0,8) for
// all 256 values — not only for the eight this build writes — because the field
// is carried whole and refused nowhere, so a decoded world may hand any byte to
// any consumer.
func TestFacingDirIsTotalOverEveryByte(t *testing.T) {
	for v := 0; v < 256; v++ {
		got := FacingDir(uint8(v))
		if got < 0 || got >= 8 {
			t.Fatalf("FacingDir(%d) = %d, outside [0,8)", v, got)
		}
		// The decoded rounding: a byte within half a direction of one names it.
		// Written out here as the arithmetic the claim gives rather than as the
		// expression facing.go uses.
		if want := ((v + 16) / 32) % 8; got != want {
			t.Errorf("FacingDir(%d) = %d, want %d", v, got, want)
		}
	}
}

// TestTheEightDirectionsRoundTripThroughTheirBytes is AC-2. Each direction's own
// byte names it back, and each byte is the multiple of 32 the store makes it.
func TestTheEightDirectionsRoundTripThroughTheirBytes(t *testing.T) {
	for _, c := range theEight {
		f := facingOfDir(c.dir)
		if want := uint8(c.dir * 32); f != want {
			t.Errorf("%s stores as %d, want %d", c.name, f, want)
		}
		if back := FacingDir(f); back != c.dir {
			t.Errorf("%s stores as %d and reads back as direction %d", c.name, f, back)
		}
	}
}

// TestADeltaNamesItsDirectionAndAZeroDeltaNamesNone is AC-2's last clause, over
// the eight one-cell deltas, over LONGER deltas on the same bearings — the
// magnitude never enters — and over the one input that has no answer.
func TestADeltaNamesItsDirectionAndAZeroDeltaNamesNone(t *testing.T) {
	for _, c := range theEight {
		for _, scale := range []int32{1, 3, 1000} {
			f, ok := facingToward(c.dx*scale, c.dy*scale)
			if !ok {
				t.Fatalf("(%d,%d) names no direction", c.dx*scale, c.dy*scale)
			}
			if got := FacingDir(f); got != c.dir {
				t.Errorf("(%d,%d) names direction %d, want %d (%s)",
					c.dx*scale, c.dy*scale, got, c.dir, c.name)
			}
		}
	}
	if f, ok := facingToward(0, 0); ok {
		t.Errorf("a zero delta named the facing %d; it must name none", f)
	}
	// And the world-level producer leaves a facing alone where the delta names
	// none, which is what movement, approach and cast admission share.
	w := &World{entities: []Entity{{Facing: 0x77}}, routes: make([][]cell, 1)}
	if w.turnToward(0, 0, 0) {
		t.Error("a zero delta began a turn")
	}
	if got := w.entities[0].Facing; got != 0x77 {
		t.Errorf("a zero delta moved the facing to %d", got)
	}
}

// TestAMoverFacesTheCellItSteppedTo is AC-3, over all eight directions: a unit
// ordered one cell away ends the tick facing that way.
//
// One cell and not a longer walk, so what is measured is the STEP's own facing
// and not the order's bearing — the two agree here and would not on a walk that
// has to detour.
func TestAMoverFacesTheCellItSteppedTo(t *testing.T) {
	b := Bounds{Width: 9, Height: 9}
	for _, c := range theEight {
		w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{{ID: 1, X: 4, Y: 4}})
		Step(w, []Command{{Entity: 1, X: 4 + c.dx, Y: 4 + c.dy}})

		e := w.Entities()[0]
		if e.X != 4+c.dx || e.Y != 4+c.dy {
			t.Fatalf("%s: the unit is at (%d,%d), want (%d,%d)", c.name, e.X, e.Y, 4+c.dx, 4+c.dy)
		}
		if got := FacingDir(e.Facing); got != c.dir {
			t.Errorf("%s: the unit faces direction %d, want %d", c.name, got, c.dir)
		}
	}
}

// TestAUnitThatTakesNoStepKeepsItsFacing is AC-3's second half, over the four
// ways a tick can leave a unit where it was. Each is a path with no assignment on
// it, so what this pins is that no site OTHER than the step ever writes.
func TestAUnitThatTakesNoStepKeepsItsFacing(t *testing.T) {
	const mark = 0x60 // south-east, and a value no case below could produce
	b := Bounds{Width: 5, Height: 5}

	t.Run("ordered nowhere", func(t *testing.T) {
		w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b),
			[]Entity{{ID: 1, X: 2, Y: 2, Facing: mark}})
		for i := 0; i < 4; i++ {
			Step(w, nil)
		}
		if got := w.Entities()[0].Facing; got != mark {
			t.Errorf("an idle unit's facing moved to %d", got)
		}
	})

	t.Run("walled in, and giving up", func(t *testing.T) {
		// One cell, walled on every side it could leave by: the near search finds
		// nothing, the count rises, and the give-up fires at the limit. The unit
		// never moves, so it never turns.
		grid := make([]byte, 9)
		for i := range grid {
			grid[i] = blockGround
		}
		grid[4] = 0
		w := mustWorldGrid(t, 1, Bounds{Width: 3, Height: 3}, ModeCanonical, grid,
			[]Entity{{ID: 1, X: 1, Y: 1, Facing: mark}})
		Step(w, []Command{{Entity: 1, X: 0, Y: 0}})
		for i := 0; i < stallLimit+2; i++ {
			Step(w, nil)
		}
		e := w.Entities()[0]
		if e.X != 1 || e.Y != 1 {
			t.Fatalf("the unit left its cell: (%d,%d)", e.X, e.Y)
		}
		if e.Facing != mark {
			t.Errorf("a unit that never stepped faces %d, want %d", e.Facing, mark)
		}
	})

	t.Run("felled", func(t *testing.T) {
		// A body faces the way it fell, and the blow clears four fields beside
		// this one — so this is the case that says the facing is not among them.
		w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b),
			[]Entity{{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10, Facing: mark}})
		Step(w, []Command{{Kind: KindKill, Entity: 1}})
		e := w.Entities()[0]
		if e.Alive() {
			t.Fatalf("the unit is still alive at %d/%d", e.HP, e.MaxHP)
		}
		if e.Facing != mark {
			t.Errorf("a felled unit faces %d, want the %d it died with", e.Facing, mark)
		}
	})

	t.Run("paying for a crossing", func(t *testing.T) {
		// A rated mover takes its cell on the crossing's first tick and pays for
		// it afterwards. The facing is written on that first tick and the paying
		// ticks leave it alone, so it points along the stride for the whole
		// crossing.
		w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b),
			[]Entity{{ID: 1, X: 1, Y: 2, Speed: 1}})
		Step(w, []Command{{Entity: 1, X: 4, Y: 2}})
		e := w.Entities()[0]
		if e.Transit == 0 {
			t.Fatalf("the mover owes no transit ticks; this case needs a rated one")
		}
		want := e.Facing
		for e.Transit > 0 {
			Step(w, nil)
			e = w.Entities()[0]
			if e.Transit > 0 && e.Facing != want {
				t.Fatalf("the facing moved to %d during a crossing it began at %d", e.Facing, want)
			}
		}
	})
}

// TestTheConstructorKeepsEveryFacingItIsHanded is AC-1. No value is refused and
// none is normalised — including on a unit that is NOT ALIVE, where the four
// fields beside it are cleared.
func TestTheConstructorKeepsEveryFacingItIsHanded(t *testing.T) {
	for v := 0; v < 256; v++ {
		w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5, Facing: uint8(v)},
			{ID: 2, X: 2, Y: 2, HP: -1, MaxHP: 5, Facing: uint8(v)},
		})
		if err != nil {
			t.Fatalf("facing %d was refused: %v", v, err)
		}
		for i, e := range w.Entities() {
			if e.Facing != uint8(v) {
				t.Errorf("entity %d: facing %d became %d", i, v, e.Facing)
			}
		}
	}
	// And an entity built naming none faces NORTH, the table's own index 0.
	var e Entity
	if e.Facing != 0 || FacingDir(e.Facing) != 0 {
		t.Errorf("an entity naming no facing faces %d, direction %d", e.Facing, FacingDir(e.Facing))
	}
}

// TestDrawnFacingAdvancesAcrossAMultiTickTurnWithoutMovingFacing is story
// 1047 round 3's P3 witness (adversarial return, section 2). Facing itself
// (the hashed field) stays at its pre-turn value for the whole interval by
// design (advanceTurns above); DrawnFacing is the presentation-only value
// that must visibly progress instead of standing still until the snap.
func TestDrawnFacingAdvancesAcrossAMultiTickTurnWithoutMovingFacing(t *testing.T) {
	// Clockwise, tie-broken: Facing 0 -> DesiredFacing 128, RotationSpeed 32,
	// arc 128, total = ceil(128/32) = 4 ticks. advanceTurns' own doc: "Turns
	// begun by later producers therefore stand for their whole first tick" —
	// so TurnRemaining==total (elapsed 0) draws unchanged.
	cw := []struct {
		remaining uint8
		want      uint8
	}{
		{4, 0},  // elapsed 0: stands at the start facing
		{3, 32}, // elapsed 1: one quarter of the arc
		{2, 64}, // elapsed 2: half
		{1, 96}, // elapsed 3: three quarters, never the full 128
	}
	for _, tc := range cw {
		e := Entity{Facing: 0, DesiredFacing: 128, TurnRemaining: tc.remaining, TurnTotal: 4, RotationSpeed: 32}
		if got := e.DrawnFacing(); got != tc.want {
			t.Errorf("clockwise remaining=%d: DrawnFacing()=%d, want %d", tc.remaining, got, tc.want)
		}
	}
	// Facing never actually moves: DrawnFacing is a read, not a write.
	e := Entity{Facing: 0, DesiredFacing: 128, TurnRemaining: 3, TurnTotal: 4, RotationSpeed: 32}
	e.DrawnFacing()
	if e.Facing != 0 || e.TurnRemaining != 3 {
		t.Fatalf("DrawnFacing wrote back Facing=%d TurnRemaining=%d, want 0/3 unchanged", e.Facing, e.TurnRemaining)
	}

	// Counterclockwise: Facing 32 -> DesiredFacing 224 is the SHORTER way
	// backward (arc 64, not the 192 the clockwise arm would cover),
	// RotationSpeed 32, total = 2 ticks.
	ccw := []struct {
		remaining uint8
		want      uint8
	}{
		{2, 32}, // elapsed 0
		{1, 0},  // elapsed 1: half of 64 subtracted from 32, wrapping past 0
	}
	for _, tc := range ccw {
		e := Entity{Facing: 32, DesiredFacing: 224, TurnRemaining: tc.remaining, TurnTotal: 2, RotationSpeed: 32}
		if got := e.DrawnFacing(); got != tc.want {
			t.Errorf("counterclockwise remaining=%d: DrawnFacing()=%d, want %d", tc.remaining, got, tc.want)
		}
	}

	// Not turning: DrawnFacing is exactly Facing, whatever DesiredFacing says.
	inactive := Entity{Facing: 96, DesiredFacing: 96, TurnRemaining: 0}
	if got := inactive.DrawnFacing(); got != 96 {
		t.Errorf("inactive turn: DrawnFacing()=%d, want 96 (Facing)", got)
	}

	// The compatibility arm (RotationSpeed<=0) never leaves a turn active in
	// practice (requestFacing snaps immediately), but DrawnFacing defends the
	// same way rather than dividing by zero.
	zero := Entity{Facing: 0, DesiredFacing: 128, TurnRemaining: 4, TurnTotal: 4, RotationSpeed: 0}
	if got := zero.DrawnFacing(); got != 0 {
		t.Errorf("RotationSpeed<=0: DrawnFacing()=%d, want 0 (Facing)", got)
	}
}
