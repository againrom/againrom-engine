package sim

import "testing"

// A unit walking across a cell move record of event 23 is passed to the move
// once its step ends centred on the record's cell, not on the tick its stride
// crosses into the cell, and arrives at the record's target (R2-ENGINE-1778).
func TestSecondGameCellMoveWaitsForTheStepsCentre(t *testing.T) {
	hero := engFighterOne(1, SelfSlot, 8, 10)
	hero.Speed = 16
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero}, secondGameScript(t, nil, nil, nil), engRel(t))
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	if err := w.DeclareCellTails([]CellTail{{X: 10, Y: 10, Bytes: [cellTailLen]byte{cellEventRelocate, 0, 10, 10, 30, 40}}}); err != nil {
		t.Fatal(err)
	}
	cmds := []Command{MoveTo(1, CellPoint{X: 12, Y: 10})}
	entered, centred := 0, 0
	for tick := 1; tick <= 200; tick++ {
		Step(w, cmds)
		cmds = nil
		e, _ := w.Entity(1)
		if e.X == 30 && e.Y == 40 {
			if entered == 0 || centred == 0 || centred < entered {
				t.Fatalf("moved on tick %d: entered the cell on tick %d, centred on it on tick %d", tick, entered, centred)
			}
			return
		}
		if e.X == 10 && e.Y == 10 {
			if entered == 0 {
				entered = tick
				if e.Transit == 0 {
					t.Fatalf("the step into the cell took no transit (total %d)", e.TransitTotal)
				}
			}
			if centred == 0 && e.Transit == 0 {
				centred = tick
			}
		}
		if e.X > 10 {
			t.Fatalf("the unit walked on to (%d,%d) on tick %d", e.X, e.Y, tick)
		}
	}
	t.Fatalf("the unit never reached the target; entered %d, centred %d", entered, centred)
}

// The move reads the record of the unit's own cell: a two-cell unit walking
// with its footprint over the record's cell and its own cell beside it is
// not moved (R2-ENGINE-1778).
func TestSecondGameCellMoveReadsTheUnitsOwnCell(t *testing.T) {
	hero := engFighterOne(1, SelfSlot, 7, 9)
	hero.Speed, hero.TokenSize = 16, 2
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero}, secondGameScript(t, nil, nil, nil), engRel(t))
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	if err := w.DeclareCellTails([]CellTail{{X: 10, Y: 10, Bytes: [cellTailLen]byte{cellEventRelocate, 0, 10, 10, 30, 40}}}); err != nil {
		t.Fatal(err)
	}
	cmds := []Command{MoveTo(1, CellPoint{X: 13, Y: 9})}
	covered := false
	for tick := 1; tick <= 200; tick++ {
		Step(w, cmds)
		cmds = nil
		e, _ := w.Entity(1)
		if e.X == 30 && e.Y == 40 {
			t.Fatalf("a unit beside the cell moved on tick %d", tick)
		}
		covered = covered || e.Y == 9 && (e.X == 9 || e.X == 10)
		if e.X == 13 && e.Transit == 0 {
			if !covered {
				t.Fatal("the footprint never covered the record's cell")
			}
			return
		}
	}
	t.Fatal("the unit never reached (13,9)")
}
