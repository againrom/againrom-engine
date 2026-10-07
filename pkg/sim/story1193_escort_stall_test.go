package sim

import "testing"

// escortWallGrid returns a Bounds and grid whose only ground-open rows
// are 4 and 5, flanked by blocked ground on every other row within height.
// A 1x1 walker can stand on either open row, but a 2x2 footprint's own
// top-left anchor (route.go's own convention) admits only row 4: an anchor
// at row 5 needs row 6 open too, and row 6 is closed. That is the shape a
// two-row-wide bridge deck between railings takes for a bigger-than-one-cell
// mover — one anchor row, not two — and it is what the owner's own saves show
// for mission121's Horisontal Bridge (docs/1193/story.md).
func escortWallGrid(width, height int32) (Bounds, []byte) {
	b := Bounds{Width: width, Height: height}
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		if y == 4 || y == 5 {
			continue
		}
		for x := int32(0); x < width; x++ {
			grid[y*width+x] = blockGround
		}
	}
	return b, grid
}

// DIV-1316
func TestAnAttackerBlockedByItsTargetsEscortHoldsTheOrderAndNeverAdvances(t *testing.T) {
	t.Parallel()

	const width, height = 24, 10
	b, grid := escortWallGrid(width, height)

	troll := engFighter(1, 2, 2, 4)
	troll.TokenSize = 2
	troll.ScanRange = 100 // wide enough that sight never breaks across this fixture
	victim := engFighter(2, 3, 20, 4)

	ents := []Entity{troll, victim}
	escortCols := []int32{8, 9, 10, 11, 12} // the owner's own saves show up to five
	for i, x := range escortCols {
		ents = append(ents, engFighter(EntityID(10+i), 5, x, 4))
	}

	rel := engRel(t, [3]uint32{2, 3, 1})
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}

	engRun(w, 1)
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("the fixture's own first decision holds %v/%v, want entity 2 acquired", v, held)
	}

	const runTicks = 400
	maxStall := uint8(0)
	sawWalkCancelled := false
	farthest := troll.X
	for k := 0; k < runTicks; k++ {
		Step(w, nil)
		e := w.entities[indexOfEntity(w.entities, 1)]
		if !e.HasAttackTarget || e.AttackTarget != 2 {
			t.Fatalf("tick %d: the order was released (HasAttackTarget=%v AttackTarget=%v) — "+
				"the victim never leaves ScanRange in this fixture, so this test's own premise is wrong",
				k, e.HasAttackTarget, e.AttackTarget)
		}
		if e.Stall > maxStall {
			maxStall = e.Stall
		}
		if !e.HasTarget {
			sawWalkCancelled = true
		}
		if e.X > farthest {
			farthest = e.X
		}
	}

	// The escort's leftmost body stands at column 8, so a 2x2 footprint
	// whose anchor reaches column 7 already touches it (span 7-8); this
	// fixture's own jam is falsified if the attacker's footprint is ever
	// seen at or past that column.
	if farthest >= 7 {
		t.Errorf("the attacker's footprint reached column %d, at or past the escort wall (column 8) — "+
			"this fixture no longer jams it", farthest)
	}
	if maxStall < stallLimit-1 {
		t.Errorf("stall reached at most %d, want it to climb to one below the cap (%d) at least once — "+
			"the retry loop this test documents never actually fired", maxStall, stallLimit-1)
	}
	if !sawWalkCancelled {
		t.Error("the walk order (HasTarget) was never seen cancelled — restAt's own sixteen-tick cap never fired")
	}

	if final := w.entities[indexOfEntity(w.entities, 1)]; !final.HasAttackTarget {
		t.Error("the order ended released — this test documents it staying held for the whole run")
	}
}
