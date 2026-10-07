package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Earlier World forms cannot contain the exact formation carrier.
const firstSavedFormationWorldForm = 90

func savedPlayerFormationFields(record *sav.DocumentRecordData) (int16, uint32, []byte, error) {
	command, err := savedStructureValue(record, "Slot")
	if err != nil || command > 0xffff {
		return 0, 0, nil, fmt.Errorf("saved Player lacks its command identifier")
	}
	trigger, err := savedStructureValue(record, "SlotAgain")
	if err != nil {
		return 0, 0, nil, err
	}
	var tail []byte
	for _, raw := range record.Raw {
		if raw.Name == "PRaw32" {
			if tail != nil || len(raw.Bytes) != sav.PlayerTailLen {
				return 0, 0, nil, fmt.Errorf("saved Player has an invalid formation block")
			}
			tail = raw.Bytes
		}
	}
	if tail == nil {
		return 0, 0, nil, fmt.Errorf("saved Player lacks its formation block")
	}
	return int16(uint16(command)), trigger, tail, nil
}

func savedFormationOwnerID(bindings *SnapshotSAVGroupBindings, owner SnapshotSAVGroupReferenceBinding) uint32 {
	if owner.Class == 1 {
		for _, p := range bindings.Players {
			if p.ObjectIndex == owner.ObjectIndex {
				return p.ID
			}
		}
	}
	return 0
}

func importSavedFormations(doc *sav.DocumentData, world *sim.World, bindings *SnapshotSAVGroupBindings) error {
	var players []sim.SavedPlayerFormation
	for _, p := range bindings.Players {
		command, trigger, raw, err := savedPlayerFormationFields(&doc.Objects[p.ObjectIndex-1])
		if err != nil {
			return err
		}
		players = append(players, sim.SavedPlayerFormation{PlayerID: p.ID, CommandID: command, TriggerID: trigger, Mode: raw[sav.PlayerTailFormationByte]})
	}
	var owners []sim.SavedGroupOwner
	for _, g := range bindings.Groups {
		owners = append(owners, sim.SavedGroupOwner{GroupID: g.ID, PlayerID: savedFormationOwnerID(bindings, g.Owner)})
	}
	if err := world.ImportSavedPlayerFormations(players, owners); err != nil {
		return err
	}
	bindings.FormationsPresent = true
	return nil
}

// Native state is required even if the retained Document has a full PRaw32.
// Projection writes only byte31 and preserves the other31 bytes. LOAD checks
// that pairing; it never copies the Document into an absent native carrier.
func savedFormationWorld(state *SnapshotSAVDocument, world *sim.World, project bool) error {
	players, present := world.SavedPlayerFormations()
	if state == nil || state.GroupBindings == nil {
		if present {
			return fmt.Errorf("saved Player formations lack their exact document bindings")
		}
		return nil
	}
	b := state.GroupBindings
	if present != b.FormationsPresent {
		return fmt.Errorf("saved Player formation presence differs from native state")
	}
	if !present {
		if project {
			for _, p := range b.Players {
				r := &state.Document.Objects[p.ObjectIndex-1]
				_, slot, raw, err := savedPlayerFormationFields(r)
				if err != nil {
					return err
				}
				raw[sav.PlayerTailFormationByte] = world.FormationMode(slot)
			}
		}
		return nil // Old native state keeps its existing per-slot modes.
	}
	native := nativeSavedPlayerBindings(b)
	if len(players) != len(native) {
		return fmt.Errorf("saved Player formations lack exact bindings")
	}
	for i, p := range players {
		binding := native[i]
		if state.Document == nil || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) || state.Document.Objects[binding.ObjectIndex-1].Class != "Player" {
			return fmt.Errorf("saved Player formation lacks its bound Player object")
		}
		command, trigger, raw, err := savedPlayerFormationFields(&state.Document.Objects[binding.ObjectIndex-1])
		if err != nil || p.PlayerID != binding.ID || p.CommandID != command || p.TriggerID != trigger {
			return fmt.Errorf("saved Player formation identity differs from its bound object")
		}
		if project {
			raw[sav.PlayerTailFormationByte] = p.Mode
		} else if raw[sav.PlayerTailFormationByte] != p.Mode {
			return fmt.Errorf("saved Player formation differs from current native byte")
		}
	}
	// An unrelated export coverage gap can retain an older Group document.
	// Each still-existing Group must nevertheless keep the same exact owner
	// in its native reference, Player binding and retained G44 field.
	groups, _, _ := world.SavedGroups()
	bindings := make(map[uint32]SnapshotSAVGroupBinding, len(b.Groups))
	for _, g := range b.Groups {
		bindings[g.ID] = g
	}
	for _, g := range groups {
		if binding, bound := bindings[g.ID]; bound {
			if err := validateSavedFormationOwner(state, binding, g); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSavedFormationOwner(state *SnapshotSAVDocument, binding SnapshotSAVGroupBinding, g sim.SavedGroup) error {
	if g.OwnerID != savedFormationOwnerID(state.GroupBindings, binding.Owner) {
		return fmt.Errorf("saved Group %d formation owner differs from exact binding", g.ID)
	}
	owner := g.Owner
	if g.Authored {
		// Native command Groups keep a typed owner with a zero source key.
		// The exact Player join above authorizes its separate output wire key.
		if owner.Class != 1 || owner.Key != 0 || owner.Archive != 0 {
			return fmt.Errorf("saved Group %d formation owner lacks native command authority", g.ID)
		}
		owner.Key = binding.Owner.Key
	}
	if owner.Key != binding.Owner.Key || owner.Class != binding.Owner.Class || owner.Owner != binding.Owner.Owner {
		return fmt.Errorf("saved SAV Group %d reference differs from its native registry", g.ID)
	}
	record := &state.Document.Objects[binding.PlayerObject-1].Groups[binding.InlineIndex]
	key, err := savedStructureValue(record, "G44")
	// An unresolved import can retain its source address behind a coverage gap;
	// the supported projector spells that same resolved-null owner as zero.
	if err != nil || key != binding.Owner.Key && !(binding.Owner.Class == 0 && key == 0) {
		return fmt.Errorf("saved Group %d formation owner differs from retained G44", g.ID)
	}
	return nil
}
