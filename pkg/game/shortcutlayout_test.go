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
		back, err := quickSpellsFromOriginalIndices(cells[:])
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
