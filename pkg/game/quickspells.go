package game

import (
	"encoding/binary"
	"fmt"
	"strings"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
)

// AI-SPELLIDENT-286: controller cell/index -> real spell ID. Commands in
// Againrom already carry real IDs, so translation happens only at this boundary.
var originalBookIDs = [...]uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}

// Unknown book IDs use the ordinary unbound cell. Current SAV session policy
// retains those custom bindings; ordinary edits to a bound cell take priority.
func quickSpellsToOriginalIndices(slots [4]uint32) (sav.CityShortcuts, error) {
	indices := sav.CityShortcuts{-1, -1, -1, -1}
	if err := validateQuickSpells(slots); err != nil {
		return indices, originalCityUnsupportedf("%v", err)
	}
	for i, id := range slots {
		if id == 0 {
			continue
		}
		for cell, originalID := range originalBookIDs {
			if id == uint32(originalID) {
				indices[i] = int32(cell)
				break
			}
		}
	}
	return indices, nil
}

func quickSpellsFromOriginalIndices(indices []int32) ([4]uint32, error) {
	var slots [4]uint32
	if len(indices) != len(slots) {
		return slots, fmt.Errorf("original quick spells: got %d slots, want four", len(indices))
	}
	for i, index := range indices {
		if index == -1 {
			continue
		}
		if index < 0 || index >= int32(len(originalBookIDs)) {
			return [4]uint32{}, fmt.Errorf("original quick spells: slot %d index %d outside -1 or 0..23", i, index)
		}
		slots[i] = uint32(originalBookIDs[index])
	}
	if err := validateQuickSpells(slots); err != nil {
		return [4]uint32{}, fmt.Errorf("original quick spells: %w", err)
	}
	return slots, nil
}

// Missing legacy records default empty. Present but malformed records refuse
// the whole load, before either the town or mission adoption boundary. The
// bounded policy is ours; malformed original runtime behaviour is Unknown.
func originalQuickSpells(file *sav.File) ([4]uint32, error) {
	store, ok := file.StateStore()
	if !ok {
		return [4]uint32{}, nil
	}
	var section, value *reg.Node
	for _, n := range store.Root.Children {
		if strings.EqualFold(n.Name, "SpellBook") {
			if section != nil || !n.Dir {
				return [4]uint32{}, fmt.Errorf("original quick spells: ambiguous or invalid SpellBook")
			}
			section = n
		}
	}
	if section == nil {
		return [4]uint32{}, nil
	}
	for _, n := range section.Children {
		if strings.EqualFold(n.Name, "Shortcuts") {
			if value != nil || n.Dir || n.Type != reg.TypeIntArray {
				return [4]uint32{}, fmt.Errorf("original quick spells: ambiguous or invalid Shortcuts")
			}
			value = n
		}
	}
	if value == nil {
		return [4]uint32{}, nil
	}
	return quickSpellsFromOriginalIndices(value.Ints)
}

func originalCityQuickSpells(document *sav.CityProvenance) ([4]uint32, error) {
	// Data canonicalizes old map-form DTOs to ordered records after validation.
	for _, record := range document.Data().State.ValueRecords {
		if record.Path == "/SpellBook/Shortcuts" {
			if record.Value.Kind != 6 || len(record.Value.Bytes) != 16 {
				return [4]uint32{}, fmt.Errorf("original city quick spells: invalid source shape")
			}
			indices := make([]int32, 4)
			for i := range indices {
				indices[i] = int32(binary.LittleEndian.Uint32(record.Value.Bytes[4*i:]))
			}
			return quickSpellsFromOriginalIndices(indices)
		}
	}
	return [4]uint32{}, fmt.Errorf("original city quick spells: missing source record")
}

// Native snapshots carry real uint16 spell IDs, not original signed indices.
// Unknown-to-this-install IDs remain bound but unavailable, preserving custom
// catalogs. The producer moves duplicates, so malformed native duplicates fail.
func validateQuickSpells(slots [4]uint32) error {
	for i, id := range slots {
		if id > 65535 {
			return fmt.Errorf("quick spell F%d ID %d exceeds uint16", i+5, id)
		}
		if id == 0 {
			continue
		}
		for j := 0; j < i; j++ {
			if slots[j] == id {
				return fmt.Errorf("quick spell F%d duplicates F%d", i+5, j+5)
			}
		}
	}
	return nil
}
