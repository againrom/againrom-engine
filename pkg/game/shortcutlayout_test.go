package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestShortcutLayoutWriterAndReaderAgree(t *testing.T) {
	layout := ui.FirstShortcutLayout
	for i := 0; i < layout.Cells; i++ {
		var slots ui.ShortcutSlots
		slots[layout.CellSlot(i)].Spell = uint32(originalBookIDs[i+3])
		cells, err := quickSpellsToOriginalIndices(slots, firstCampaignRules{})
		if err != nil || cells[i] != int32(i+3) {
			t.Fatalf("cell %d: got %v, %v", i, cells, err)
		}
		back, err := quickSpellsFromOriginalIndices(cells[:], layout)
		if err != nil || back != slots {
			t.Fatalf("cell %d: read back %v, %v", i, back, err)
		}
		if got, ok := layout.SlotCell(layout.CellSlot(i)); !ok || got != i {
			t.Fatalf("slot of cell %d maps to cell %d, %v", i, got, ok)
		}
	}
	slots := spellSlots(0, uint32(originalBookIDs[0]))
	if cells, err := quickSpellsToOriginalIndices(slots, secondCampaignRules{}); err != nil || cells != (sav.CityShortcuts{-1, -1, -1, -1}) {
		t.Fatalf("second game wrote cells %v, %v", cells, err)
	}
}

// A second-game file's cells bind from the first key, as an older engine wrote them.
func TestShortcutLayoutSecondFileCellsBindFromTheFirstKey(t *testing.T) {
	b := originalContainer(sav.SecondGame.Magic, secondSavedBody(0, 0, nil), append(shortcutsStore(0, 1, -1, -1), secondSavedTail(0, [2]uint32{1, placeTown})...))
	sf, err := sav.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	stated, err := fileCampaign(sf).fileState(sf)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := currentQuickSpells(sf, stated.quickSpells, stated.session); err != nil || got != spellSlots(1, 2) {
		t.Fatalf("second file bound %v, %v", got, err)
	}
	if slots, err := quickSpellsFromOriginalIndices([]int32{5, -1, 7, -1}, ui.SecondShortcutLayout); err != nil || slots != (ui.ShortcutSlots{}) {
		t.Fatalf("second row bound %v, %v", slots, err)
	}
	if _, err := quickSpellsFromOriginalIndices([]int32{24, -1, -1, -1}, ui.SecondShortcutLayout); err == nil {
		t.Fatal("second row admitted an index outside 0..23")
	}
}
