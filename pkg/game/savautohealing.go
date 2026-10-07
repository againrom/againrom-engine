package game

import (
	"fmt"
	"slices"

	"againrom/pkg/sim"
)

// Player+58 is transported on exact, unambiguous Player bindings. Equal-slot
// aliases retain their original scalars; no native policy is inferred for them.
func importSavedAutoHealing(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.GroupBindings == nil || !state.GroupBindings.PlayersPresent {
		return nil
	}
	rows, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
	if err != nil {
		return err
	}
	for i, row := range rows {
		if row.Unavailable != "" {
			continue
		}
		object := state.GroupBindings.Players[i].ObjectIndex
		percent, err := savedStructureValue(&state.Document.Objects[object-1], "F58")
		if err != nil || !world.ImportAutoHealing(row.Slot, percent) {
			return fmt.Errorf("cannot import autohealing for Player%d", row.PlayerID)
		}
	}
	return nil
}

func savedAutoHealingWorld(state *SnapshotSAVDocument, world *sim.World, project bool) error {
	if state == nil || state.GroupBindings == nil || !state.GroupBindings.PlayersPresent {
		return nil
	}
	rows, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
	if err != nil {
		return err
	}
	for i, row := range rows {
		percent, present := world.AutoHealing(row.Slot)
		if !present {
			continue
		}
		if row.Unavailable != "" {
			return fmt.Errorf("autohealing has ambiguous Player%d ownership", row.PlayerID)
		}
		object := state.GroupBindings.Players[i].ObjectIndex
		record := &state.Document.Objects[object-1]
		if project {
			record.Values = slices.Clone(record.Values)
			if err := savedActorSetValue(record, "F58", percent); err != nil {
				return err
			}
		} else if current, err := savedStructureValue(record, "F58"); err != nil || current != percent {
			return fmt.Errorf("Player%d F58 differs from current autohealing policy", row.PlayerID)
		}
	}
	return nil
}
