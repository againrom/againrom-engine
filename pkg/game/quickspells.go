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
func quickSpellsToOriginalIndices(slots ui.ShortcutSlots, profiles ...campaignService) (sav.CityShortcuts, error) {
	indices := sav.CityShortcuts{-1, -1, -1, -1}
	if err := validateQuickSpells(slots); err != nil {
		return indices, originalCityUnsupportedf("%v", err)
	}
	start := 1
	if len(profiles) > 0 {
		start = profiles[0].shortcutStart()
	}
	if start < 0 {
		return indices, nil
	}
	for i, id := range slots[start : start+len(indices)] {
		if id.Spell == 0 {
			continue
		}
		for cell, originalID := range originalBookIDs {
			if id.Spell == uint32(originalID) {
				indices[i] = int32(cell)
				break
			}
		}
	}
	return indices, nil
}

func quickSpellsFromOriginalIndices(indices []int32) (ui.ShortcutSlots, error) {
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
		slots[i+1].Spell = uint32(originalBookIDs[index])
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
	return fileQuickSpells(file, false)
}

// fileQuickSpells reads the file's shortcut cells. emptyUnbound admits an
// empty array as no binding: the second game's files hold one, and no claim
// reads its index domain (DIV-4154).
func fileQuickSpells(file *sav.File, emptyUnbound bool) (ui.ShortcutSlots, error) {
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
	if len(value.Ints) == 0 && emptyUnbound {
		return ui.ShortcutSlots{}, nil
	}
	return quickSpellsFromOriginalIndices(value.Ints)
}

func quickSpellsFromState(state sav.DocumentStateData) (ui.ShortcutSlots, error) {
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
	return quickSpellsFromOriginalIndices(indices)
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

func originalCityQuickSpells(document *sav.CityProvenance) (ui.ShortcutSlots, error) {
	data := document.Data().State
	return quickSpellsFromState(sav.DocumentStateData{RootKind: data.RootKind, DirectoryRecords: data.DirectoryRecords, ValueRecords: data.ValueRecords})
}

// Native snapshots carry real uint16 spell IDs, not original signed indices.
// Unknown-to-this-install IDs remain bound but unavailable, preserving custom
// catalogs. The producer moves duplicates, so malformed native duplicates fail.
func validateQuickSpells(slots ui.ShortcutSlots) error {
	for i, id := range slots {
		if id.Spell > 65535 || len(id.Item) > 65536 || id.Spell != 0 && id.Item != "" {
			return fmt.Errorf("shortcut F%d has invalid content %v", i+4, id)
		}
		if id == (ui.Shortcut{}) {
			continue
		}
		for j := 0; j < i; j++ {
			if slots[j] == id {
				return fmt.Errorf("quick spell F%d duplicates F%d", i+4, j+4)
			}
		}
	}
	return nil
}
