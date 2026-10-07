package sim

import (
	"bytes"
	"reflect"
	"testing"
)

// MAGIC-093 requires six offsets [-2,3]. uniform's argument is an inclusive
// maximum, so uniform(5)-2 already implements that domain. These literals were
// calculated independently using unsigned 64-bit SplitMix arithmetic and the
// high half of draw*6; they are native RNG controls, not ROM1 random samples.
var meteorRangeOffsets = [32][2]int32{
	{2, 3}, {-2, 0}, {0, 1}, {-1, 3}, {-1, 1}, {0, -1}, {3, 3}, {0, 1},
	{-1, 2}, {2, -1}, {3, 3}, {2, 2}, {2, -2}, {1, 0}, {3, 2}, {2, 3},
	{3, -1}, {-2, 2}, {-1, 1}, {2, -1}, {3, -1}, {2, -1}, {1, 2}, {0, 1},
	{-2, 1}, {-2, 3}, {-2, 3}, {-2, 3}, {1, -2}, {0, 1}, {1, 3}, {2, 3},
}

func TestMeteorRangeSixValuesLiteralOrderAndDrawCount(t *testing.T) {
	for _, center := range [][2]int32{{20, 20}, {8, 8}, {31, 31}, {254, 254}} {
		w := mustWorld(t, 0x175, Bounds{Width: 40, Height: 40}, nil)
		effect := cellEffect{Key: cellKey(center[0], center[1]), Spell: 21}
		for stage, offset := range meteorRangeOffsets {
			x, y := int32(uint8(center[0]+offset[0])), int32(uint8(center[1]+offset[1]))
			got := w.ringStageCells(effect, stage)
			if x < 8 || x > 31 || y < 8 || y > 31 {
				if len(got) != 0 {
					t.Fatalf("center%v stage%d emitted rejected cell %v", center, stage, got)
				}
			} else if len(got) != 1 || got[0] != uint16(x+y*256) {
				t.Fatalf("center%v stage%d = %v; want (%d,%d)", center, stage, got, x, y)
			}
		}
		if w.rng.state != 0x8dde6e5fd29f06b5 {
			t.Fatalf("center%v changed the two-draw-per-stage rule: %016x", center, w.rng.state)
		}
		before := w.rng.state
		for _, stage := range []int{-1, 32, 33} {
			if got := w.ringStageCells(effect, stage); len(got) != 0 || w.rng.state != before {
				t.Fatalf("inactive stage%d emitted a cell or consumed RNG", stage)
			}
		}
	}
}

func TestMeteorRangeRealCastNativeContinuation(t *testing.T) {
	w := paintWorld(t, apRule(21, 8))
	StepReported(w, []Command{{Kind: KindCastAt, Entity: 1, X: 44, Y: 44, Spell: 21}})
	var restored *World
	var ticks []uint64
	seenPositiveThree := false
	for range 220 {
		report := StepReported(w, nil)
		if restored != nil {
			other := StepReported(restored, nil)
			if !reflect.DeepEqual(report, other) || w.Hash() != restored.Hash() {
				t.Fatal("mid-Meteor LOAD changed next report or World hash", w.Tick())
			}
		}
		for _, paint := range report.AreaPaints {
			if paint.Spell != 21 || len(paint.Cells) != 1 {
				t.Fatal("Meteor did not emit exactly one cell", paint)
			}
			cell := paint.Cells[0]
			dx, dy := cell.X-44, cell.Y-44
			if dx < -2 || dx > 3 || dy < -2 || dy > 3 {
				t.Fatal("Meteor escaped [-2,3]", cell)
			}
			seenPositiveThree = seenPositiveThree || dx == 3 || dy == 3
			ticks = append(ticks, uint64(w.Tick()))
		}
		if len(ticks) == 13 && restored == nil {
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			restored = new(World)
			if err := restored.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			again, err := restored.MarshalBinary()
			if err != nil || !bytes.Equal(raw, again) || w.Hash() != restored.Hash() {
				t.Fatal("mid-Meteor native state changed at LOAD", err)
			}
		}
	}
	if len(ticks) != 32 || !seenPositiveThree || restored == nil {
		t.Fatal("incomplete Meteor witness", len(ticks), seenPositiveThree, restored != nil)
	}
	for i := 1; i < len(ticks); i++ {
		if ticks[i]-ticks[i-1] != 3 {
			t.Fatal("Meteor stage cadence is not three ticks", ticks)
		}
	}
}
