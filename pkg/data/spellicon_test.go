package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// MAGIC-ICON-024: the grid, stated as the two corners it puts cells at and as
// the bounding box that has to fit the atlas it is cut from.
func TestSpellIconGridFitsItsAtlas(t *testing.T) {
	if x, y, ok := data.SpellIconCell(0); !ok || x != 6 || y != 6 {
		t.Errorf("slot 0 at (%d,%d) ok=%v, want (6,6)", x, y, ok)
	}
	// The last cell of the first row, and the first of the second: the wrap is
	// at twelve, so slot 11 is the row's end and slot 12 starts the next.
	if x, y, ok := data.SpellIconCell(11); !ok || x != 6+38*11 || y != 6 {
		t.Errorf("slot 11 at (%d,%d) ok=%v", x, y, ok)
	}
	if x, y, ok := data.SpellIconCell(12); !ok || x != 6 || y != 6+38 {
		t.Errorf("slot 12 at (%d,%d) ok=%v", x, y, ok)
	}

	for slot := 0; slot < data.SpellIconSlots; slot++ {
		x, y, ok := data.SpellIconCell(slot)
		if !ok {
			t.Fatalf("slot %d does not exist", slot)
		}
		if x+data.SpellIconSide > data.SpellIconAtlasW || y+data.SpellIconSide > data.SpellIconAtlasH {
			t.Errorf("slot %d's cell (%d,%d)+%d runs off the %dx%d atlas",
				slot, x, y, data.SpellIconSide, data.SpellIconAtlasW, data.SpellIconAtlasH)
		}
	}

	for _, slot := range []int{-1, data.SpellIconSlots, 99} {
		if _, _, ok := data.SpellIconCell(slot); ok {
			t.Errorf("slot %d exists, want it refused", slot)
		}
	}
}

// The slot table itself: twenty-four positions, each naming a distinct spell,
// and the four ids that are in no slot at all.
func TestSpellIconSlotIsTheEnginesOwnTable(t *testing.T) {
	want := []int32{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12,
		6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}

	seen := map[int32]int{}
	for slot, id := range want {
		got, ok := data.SpellIconSlot(id)
		if !ok || got != slot {
			t.Errorf("spell %d is at slot %d (ok=%v), want %d", id, got, ok, slot)
		}
		if prev, dup := seen[id]; dup {
			t.Errorf("spell %d is named by both slot %d and slot %d", id, prev, slot)
		}
		seen[id] = slot
	}

	// AC: ids 11, 17, 27 and 28 — drain_life, darkness, curse and slow — are in
	// no slot, so twenty-four cells serve twenty-eight spells and those four
	// have no icon of their own.
	for _, id := range []int32{11, 17, 27, 28} {
		if slot, ok := data.SpellIconSlot(id); ok {
			t.Errorf("spell %d resolved to slot %d, want no slot at all", id, slot)
		}
	}

	// And every id the shipped table does not carry answers the same way,
	// including ones outside the spell range entirely.
	for _, id := range []int32{0, 29, 30, -1, 1000} {
		if _, ok := data.SpellIconSlot(id); ok {
			t.Errorf("spell %d resolved to a slot", id)
		}
	}
}
