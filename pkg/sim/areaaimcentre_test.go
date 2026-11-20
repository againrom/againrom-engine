package sim

import "testing"

// areaAimWorld holds a mage of the player's side at (10,10) and a hostile unit
// whose body is side cells wide, anchored at (20,20), over one area row. The
// second game runs it under its script, so its profile is the one asked.
func areaAimWorld(t *testing.T, second bool, side uint8) (*World, SpellRule) {
	t.Helper()
	rule := areaCloud()
	rule.Radius = 0
	caster := effectMage(1, 10, 10, 1<<7)
	target := engFighter(2, 3, 20, 20)
	target.TokenSize = side
	var script *Script
	if second {
		script = secondGameScript(t, nil, nil, nil)
	}
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 1})
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{},
		[]Entity{caster, target}, script, rel, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	if playsSecondGame(w) != second {
		t.Fatalf("fixture: second game %v, want %v", playsSecondGame(w), second)
	}
	return w, rule
}

// landedAt is the cell the one cloud of spell id landed on.
func landedAt(t *testing.T, w *World, id uint16) (int32, int32) {
	t.Helper()
	var got []CellEffect
	for _, c := range w.CellEffects() {
		if c.Spell == id {
			got = append(got, c)
		}
	}
	if len(got) != 1 {
		t.Fatalf("the world holds %d clouds of spell %d, want one", len(got), id)
	}
	return got[0].X, got[0].Y
}

// A book cast of an area row at a unit aims at the unit's position, and for a
// body wider than one cell at its centre less one sub-unit: the cell the
// anchor and half the side less one lie on (R2-ENGINE-1138). The first game
// lands on the anchor.
func TestAnAreaCastAtAWideUnitLandsOnTheBodyCentreInTheSecondGame(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		side         uint8
		first, other int32
	}{{1, 20, 20}, {2, 20, 20}, {3, 20, 21}, {4, 20, 21}, {5, 20, 22}} {
		for _, second := range []bool{false, true} {
			w, rule := areaAimWorld(t, second, c.side)
			if !w.castSpell(0, 2, uint32(rule.ID), nil) {
				t.Fatalf("side %d second %v: the cast was refused", c.side, second)
			}
			want := c.first
			if second {
				want = c.other
			}
			if x, y := landedAt(t, w, rule.ID); x != want || y != want {
				t.Errorf("side %d second %v: landed on (%d,%d), want (%d,%d)", c.side, second, x, y, want, want)
			}
		}
	}
}
