package sim

import (
	"reflect"
	"testing"
)

// areaHeadingWorld is one mage at (20, 20) on a 40 by 40 map that knows Acid
// Stream, in the first or the second game.
func areaHeadingWorld(t *testing.T, second bool) *World {
	t.Helper()
	w, err := NewSpelledWorld(0x1001, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{effectMage(1, 20, 20, 1<<9)}, nil, []SpellRule{acidStreamRule()})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// R2-ENGINE-792 and R2-ENGINE-1545: every second-game area build stores the
// octant from the caster's cell to the aimed cell, so an Acid Stream aimed at
// a unit points at the unit's cell even when the caster faces elsewhere. The
// first game keeps the facing a unit-aimed cast supplies. A queued delivery's
// saved heading follows the same rule.
func TestSecondAreaHeadingIsTheCellOctantNotTheFacing(t *testing.T) {
	north := facingOfDir(0)
	for _, tc := range []struct {
		second bool
		want   uint8
	}{{false, 0}, {true, 3}} {
		w := areaHeadingWorld(t, tc.second)
		if !w.landAreaAimed(areaAim{Target: 2, Has: true}, acidStreamRule(), 10, 1, true, 20, 20, 22, 21, north, true, nil) {
			t.Fatalf("second=%v: Acid Stream did not land", tc.second)
		}
		if got := w.effects[len(w.effects)-1].Direction; got != tc.want {
			t.Errorf("second=%v: heading %d, want %d", tc.second, got, tc.want)
		}
		// A cast at a cell supplies no facing: both games take the octant.
		if !w.landAreaAimed(areaAim{}, acidStreamRule(), 10, 1, true, 20, 20, 20, 24, 0, false, nil) {
			t.Fatalf("second=%v: cell cast did not land", tc.second)
		}
		if got := w.effects[len(w.effects)-1].Direction; got != 4 {
			t.Errorf("second=%v: cell cast heading %d, want 4", tc.second, got)
		}
		w.deliveries = append(w.deliveries, spellDelivery{Caster: 1, HasCaster: true, Target: 2, FromX: 20, FromY: 20, X: 22, Y: 21,
			Rule: acidStreamRule(), Power: 10, Facing: north, FixedFacing: true})
		if got := w.NativeSpellDeliverySaveStates()[0].Area.Direction; got != tc.want {
			t.Errorf("second=%v: saved delivery heading %d, want %d", tc.second, got, tc.want)
		}
	}
}

// R2-ENGINE-1545: the heading is the octant of the cell offset with slope
// thresholds 1/2 and 2, a tie going to the diagonal and a zero offset to 7.
// headingOf already draws it; this pins the sixteen sectors.
func TestSecondAreaHeadingSectorsFollowTheSlopeThresholds(t *testing.T) {
	for _, tc := range []struct {
		dx, dy int32
		want   uint8
	}{
		{0, 0, 7}, {0, -3, 0}, {1, -3, 0}, {1, -2, 1}, {1, -1, 1}, {2, -1, 1}, {3, -1, 2}, {3, 0, 2},
		{3, 1, 2}, {2, 1, 3}, {1, 1, 3}, {1, 2, 3}, {1, 3, 4}, {0, 3, 4}, {-1, 3, 4}, {-1, 2, 5},
		{-1, 1, 5}, {-2, 1, 5}, {-3, 1, 6}, {-3, 0, 6}, {-3, -1, 6}, {-2, -1, 7}, {-1, -1, 7}, {-1, -2, 7}, {-1, -3, 0},
	} {
		if got := areaDirection(20, 20, 20+tc.dx, 20+tc.dy); got != tc.want {
			t.Errorf("offset (%d,%d): heading %d, want %d", tc.dx, tc.dy, got, tc.want)
		}
	}
}

// R2-ENGINE-1546: per heading, pulse p hits the builder table's cells around
// the origin, and a cell whose x or y lies below 8 or above the map size
// minus 9 is skipped whole; no cell is clamped or shifted. The cells are
// built here from the claim's table, not from the engine's stage.
func TestSecondAcidStreamSkipsCellsPastTheInsetWhole(t *testing.T) {
	cone := func(d uint8, p int32) [][2]int32 {
		var out [][2]int32
		if d%2 == 0 {
			if p > 4 {
				return nil
			}
			for i := -p; i <= p; i++ {
				out = append(out, [...][2]int32{{i, -p}, {}, {p, i}, {}, {-i, p}, {}, {-p, i}}[d])
			}
			return out
		}
		for i := int32(0); i <= p; i++ {
			out = append(out, [...][2]int32{{}, {p - i, -i}, {}, {p - i, i}, {}, {-(p - i), i}, {}, {-(p - i), -i}}[d])
		}
		return out
	}
	for _, origin := range [][2]int32{{10, 10}, {29, 29}, {10, 29}, {29, 10}} {
		for d := uint8(0); d < 8; d++ {
			for p := int32(0); p < 6; p++ {
				var want []uint16
				for _, c := range cone(d, p) {
					x, y := origin[0]+c[0], origin[1]+c[1]
					if x >= 8 && y >= 8 && x <= 31 && y <= 31 {
						want = append(want, cellKey(x, y))
					}
				}
				for _, second := range []bool{false, true} {
					w := areaHeadingWorld(t, second)
					got := w.ringStageCells(cellEffect{Key: cellKey(origin[0], origin[1]), Spell: 9, Direction: d}, int(p))
					if !reflect.DeepEqual(canonicalCells(got), canonicalCells(want)) {
						t.Errorf("second=%v origin %v heading %d pulse %d: cells %x, want %x", second, origin, d, p, canonicalCells(got), canonicalCells(want))
					}
				}
			}
		}
	}
}
