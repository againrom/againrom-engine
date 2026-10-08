package game

import (
	"fmt"

	"againrom/pkg/sim"
)

func importSavedPlayerParticipants(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil {
		return nil
	}
	if err := importCurrentPlayerRoots(state); err != nil {
		return err
	}
	bindings := currentPlayerRoots(state)
	players := make([]sim.SavedGroupPlayer, len(bindings))
	rows := make([]sim.PlayerParticipant, len(bindings))
	for i, b := range bindings {
		if b.ID == 0 || b.ObjectIndex == 0 || int(b.ObjectIndex) > len(state.Document.Objects) {
			return fmt.Errorf("Participant lacks exact ordinary Player")
		}
		slot, err := savedStructureValue(&state.Document.Objects[b.ObjectIndex-1], "Slot")
		if err != nil {
			return err
		}
		value, err := savedStructureValue(&state.Document.Objects[b.ObjectIndex-1], "Participant")
		if err != nil {
			return err
		}
		players[i] = sim.SavedGroupPlayer{ID: b.ID, Slot: uint32(uint16(slot))}
		rows[i] = sim.PlayerParticipant{PlayerID: b.ID, Value: value}
	}
	return world.RestoreCurrentPlayers(players, rows, true)
}

func projectSavedPlayerParticipants(state *SnapshotSAVDocument, world *sim.World) error {
	rows, present := world.PlayerParticipants()
	if !present {
		return nil
	}
	if state == nil || state.Document == nil {
		return fmt.Errorf("current Participant lacks exact document binding")
	}
	bindings := currentPlayerRoots(state)
	if len(rows) != len(bindings) {
		return fmt.Errorf("current Participant lacks complete Player bindings")
	}
	for i, row := range rows {
		b := bindings[i]
		if b.ID != row.PlayerID || b.ObjectIndex == 0 || int(b.ObjectIndex) > len(state.Document.Objects) || state.Document.Objects[b.ObjectIndex-1].Class != "Player" {
			return fmt.Errorf("current Participant lacks exact ordinary Player")
		}
		if _, err := savedStructureValue(&state.Document.Objects[b.ObjectIndex-1], "Participant"); err != nil {
			return err
		}
	}
	for i, row := range rows {
		mustSetValue(&state.Document.Objects[bindings[i].ObjectIndex-1], "Participant", row.Value)
	}
	return nil
}
