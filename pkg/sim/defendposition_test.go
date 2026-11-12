package sim

import "testing"

func TestSecondDefendStepsFromItsFinePointAlongTheLine(t *testing.T) {
	for _, second := range []bool{false, true} {
		for facing := uint8(0); ; facing += 32 {
			w := asWorld(t, second, laFighter(1, SelfSlot, 7, 20, 20), laFighter(2, SelfSlot, 8, 21, 20))
			w.entities[0].Facing = facing
			w.commandDefend([]int{1}, 1)
			w.armDefend(1)
			e := w.entities[1]
			x, y := int32(23), int32(20)
			if second {
				x, y = 24, 21
			}
			if !e.HasTarget || e.TargetX != x || e.TargetY != y {
				t.Fatalf("second=%v facing=%d: destination (%d,%d)/%v, want (%d,%d)", second, facing, e.TargetX, e.TargetY, e.HasTarget, x, y)
			}
			if facing == 224 {
				break
			}
		}
	}
}

func TestSecondDefendComparesStoredZeroBeforeItsScanFallback(t *testing.T) {
	w := asWorld(t, true, laFighter(1, SelfSlot, 7, 20, 20), laFighter(2, SelfSlot, 8, 21, 20))
	w.commandDefend([]int{1}, 1)
	w.entities[1].EscortRange = 0
	w.armDefend(1)
	if e := w.entities[1]; !e.HasTarget || e.TargetX != 20 || e.TargetY != 20 || e.EscortOrder != escortOrderClose {
		t.Fatalf("stored zero did not approach the subject: %+v", e)
	}
}

func TestSecondDefendApproachAndMapClamp(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		subject, defender, want CellPoint
	}{
		{"approach", CellPoint{20, 20}, CellPoint{30, 20}, CellPoint{20, 20}},
		{"low clamp", CellPoint{10, 10}, CellPoint{9, 9}, CellPoint{8, 8}},
		{"high clamp", CellPoint{53, 53}, CellPoint{54, 54}, CellPoint{55, 55}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := asWorld(t, true, laFighter(1, SelfSlot, 7, tc.subject.X, tc.subject.Y), laFighter(2, SelfSlot, 8, tc.defender.X, tc.defender.Y))
			w.commandDefend([]int{1}, 1)
			w.armDefend(1)
			e := w.entities[1]
			if !e.HasTarget || e.TargetX != tc.want.X || e.TargetY != tc.want.Y {
				t.Fatalf("destination (%d,%d)/%v, want %+v", e.TargetX, e.TargetY, e.HasTarget, tc.want)
			}
		})
	}
}

func TestSecondDefendFinishesItsStepAwayBeforeIdling(t *testing.T) {
	w := asWorld(t, true, laFighter(1, SelfSlot, 7, 20, 20), laFighter(2, SelfSlot, 8, 21, 20))
	w.commandDefend([]int{1}, 1)
	for range 240 {
		Step(w, nil)
		if e := w.entities[1]; e.X == 24 && e.Y == 21 && e.Transit == 0 {
			return
		}
	}
	e := w.entities[1]
	t.Fatalf("step-away ended at (%d,%d), target (%d,%d)/%v", e.X, e.Y, e.TargetX, e.TargetY, e.HasTarget)
}
