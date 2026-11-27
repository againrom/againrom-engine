package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// AI-SPELLIDENT-286: controller cell/index -> real spell ID. Commands in
// Againrom already carry real IDs, so translation happens only at this boundary.
var originalBookIDs = data.FirstSpellPresentation().BookRows

// Unknown book IDs use the ordinary unbound cell. Current SAV session policy
// retains those custom bindings; ordinary edits to a bound cell take priority.
func quickSpellsToOriginalIndices(slots ui.ShortcutSlots, campaign campaignService) (sav.CityShortcuts, error) {
	indices := sav.CityShortcuts{-1, -1, -1, -1}
	if err := validateQuickSpells(slots); err != nil {
		return indices, originalCityUnsupportedf("%v", err)
	}
	layout := campaign.shortcutLayout()
	if !layout.HasCells() {
		return indices, nil
	}
	for i := range indices {
		indices[i] = originalCell(slots[layout.CellSlot(i)].Spell)
	}
	return indices, nil
}

func originalCell(spell uint32) int32 {
	if spell != 0 {
		for cell, originalID := range originalBookIDs {
			if spell == uint32(originalID) {
				return int32(cell)
			}
		}
	}
	return -1
}

func shortcutOriginalCell(bound ui.Shortcut) (int32, error) {
	if err := validateShortcut(0, bound); err != nil {
		return -1, err
	}
	return originalCell(bound.Spell), nil
}

func quickSpellsFromOriginalIndices(indices []int32, layout ui.ShortcutLayout) (ui.ShortcutSlots, error) {
	var slots ui.ShortcutSlots
	if len(indices) != len(sav.CityShortcuts{}) {
		return slots, fmt.Errorf("original quick spells: got %d slots, want four", len(indices))
	}
	for i, index := range indices {
		if index == -1 {
			continue
		}
		if index < 0 || index >= int32(len(originalBookIDs)) {
			return ui.ShortcutSlots{}, fmt.Errorf("original quick spells: slot %d index %d outside -1 or 0..23", i, index)
		}
		if layout.HasCells() {
			slots[layout.CellSlot(i)].Spell = uint32(originalBookIDs[index])
		}
	}
	if err := validateQuickSpells(slots); err != nil {
		return ui.ShortcutSlots{}, fmt.Errorf("original quick spells: %w", err)
	}
	return slots, nil
}

// Missing legacy records default empty. Present but malformed records refuse
// the whole load, before either the town or mission adoption boundary. The
// bounded policy is ours; malformed original runtime behaviour is Unknown.
func originalQuickSpells(file *sav.File) (ui.ShortcutSlots, error) {
	return fileQuickSpells(file, firstCampaignRules{}, 0)
}

// loadedShortcutLayout is the row a file's cells load into in campaign's play;
// a row without cells under the older slot numbering (0) loads the older row.
func loadedShortcutLayout(campaign campaignService, numbering uint8) ui.ShortcutLayout {
	layout := campaign.shortcutLayout()
	if !layout.HasCells() && numbering == 0 {
		return ui.OlderSecondShortcutLayout
	}
	return layout
}

// fileQuickSpells reads the file's shortcut cells into the loaded row. A row
// without cells admits an empty array as no binding (DIV-4154).
func fileQuickSpells(file *sav.File, campaign campaignService, numbering uint8) (ui.ShortcutSlots, error) {
	value, present, err := file.StateLeaves().ReadLeaf("/SpellBook/Shortcuts")
	if err != nil {
		return ui.ShortcutSlots{}, err
	}
	if !present {
		return ui.ShortcutSlots{}, nil
	}
	if value.Dir || value.Type != reg.TypeIntArray {
		return ui.ShortcutSlots{}, fmt.Errorf("original quick spells: invalid source shape")
	}
	if len(value.Ints) == 0 && !campaign.shortcutLayout().HasCells() {
		return ui.ShortcutSlots{}, nil
	}
	return quickSpellsFromOriginalIndices(value.Ints, loadedShortcutLayout(campaign, numbering))
}

func quickSpellsFromState(state sav.DocumentStateData, layout ui.ShortcutLayout) (ui.ShortcutSlots, error) {
	value, present, err := state.ReadLeaf("/SpellBook/Shortcuts")
	if err != nil {
		return ui.ShortcutSlots{}, err
	}
	if !present {
		return ui.ShortcutSlots{}, fmt.Errorf("original city quick spells: missing source record")
	}
	if value.Kind != 6 || len(value.Bytes) != 16 {
		return ui.ShortcutSlots{}, fmt.Errorf("original quick spells: invalid source shape")
	}
	indices := make([]int32, 4)
	for i := range indices {
		indices[i] = int32(binary.LittleEndian.Uint32(value.Bytes[4*i:]))
	}
	return quickSpellsFromOriginalIndices(indices, layout)
}

func writeQuickSpellLeaf(doc *sav.DocumentData, shortcuts sav.CityShortcuts) {
	_, _ = patchStateLeaf(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/SpellBook/Shortcuts", func(v *sav.CityStateValueData) error {
		cells := shortcuts[:]
		v.Bytes = make([]byte, 4*len(cells))
		for j, index := range cells {
			binary.LittleEndian.PutUint32(v.Bytes[j*4:], uint32(index))
		}
		return nil
	})
}

func originalCityQuickSpells(document *sav.CityProvenance, layout ui.ShortcutLayout) (ui.ShortcutSlots, error) {
	data := document.Data().State
	return quickSpellsFromState(sav.DocumentStateData{RootKind: data.RootKind, DirectoryRecords: data.DirectoryRecords, ValueRecords: data.ValueRecords}, layout)
}

// Native snapshots carry real uint16 spell IDs, not original signed indices.
// Unknown-to-this-install IDs remain bound but unavailable, preserving custom
// catalogs. The producer moves duplicates, so malformed native duplicates fail.
func validateQuickSpells(slots ui.ShortcutSlots) error {
	for i, id := range slots {
		if err := validateShortcut(i, id); err != nil {
			return err
		}
		if id == (ui.Shortcut{}) {
			continue
		}
		for j := 0; j < i; j++ {
			if slots[j] == id {
				return fmt.Errorf("quick spell %s duplicates %s", ui.ShortcutKeyName(i), ui.ShortcutKeyName(j))
			}
		}
	}
	return nil
}

func validateShortcut(slot int, id ui.Shortcut) error {
	if id.Spell > 65535 || len(id.Item) > 65536 || id.Spell != 0 && id.Item != "" {
		return fmt.Errorf("shortcut %s has invalid content %v", ui.ShortcutKeyName(slot), id)
	}
	return nil
}
