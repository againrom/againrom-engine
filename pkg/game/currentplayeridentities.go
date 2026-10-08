package game

import (
	"fmt"

	"againrom/pkg/sim"
)

type currentPlayerIdentity struct {
	ID     uint32
	Object uint16
}

func captureCurrentPlayerIdentities(state *SnapshotSAVDocument, w *sim.World, a *currentActionData) error {
	players, present := w.CurrentPlayers()
	if !present {
		return nil
	}
	if state.PlayerRoots == nil && state.GroupBindings == nil {
		return fmt.Errorf("current Player identity has no exact ordinary roots")
	}
	bindings := currentPlayerRoots(state)
	if len(players) != len(bindings) {
		return fmt.Errorf("current Player identity does not cover ordinary roots")
	}
	rows := make([]currentPlayerIdentity, len(players))
	for i, player := range players {
		b := bindings[i]
		if b.ID != player.ID {
			return fmt.Errorf("current Player identity differs from exact ordinary root")
		}
		rows[i] = currentPlayerIdentity{ID: player.ID, Object: b.ObjectIndex}
	}
	a.PlayerIdentities = &rows
	_, participants := w.PlayerParticipants()
	a.GroupParticipants = &participants
	return nil
}

func restoreCurrentPlayerIdentities(ms *Mission, a *currentActionData) error {
	if a.PlayerIdentities == nil {
		return nil
	}
	state := ms.savedDocument
	if state == nil || state.PlayerRoots == nil && state.GroupBindings == nil || len(*a.PlayerIdentities) > 65535 {
		return fmt.Errorf("current Player identity has no exact imported roots")
	}
	bindings := currentPlayerRoots(state)
	if len(*a.PlayerIdentities)+len(a.AbsentPlayers) != len(bindings) {
		return fmt.Errorf("current Player identity does not cover exact imported roots")
	}
	ids := make(map[uint32]uint32, len(bindings))
	byObject, unmodeled := map[uint16]uint32{}, map[uint16]bool{}
	var highWater uint32
	for i, row := range *a.PlayerIdentities {
		if row.ID == 0 || i > 0 && (*a.PlayerIdentities)[i-1].ID >= row.ID || byObject[row.Object] != 0 {
			return fmt.Errorf("current Player identities are unordered or have a different ordinary root")
		}
		byObject[row.Object] = row.ID
		highWater = max(highWater, row.ID)
	}
	for _, row := range a.AbsentPlayers {
		if byObject[row.Object] != 0 || unmodeled[row.Object] {
			return fmt.Errorf("current Player identity conflicts with a transport-only root")
		}
		unmodeled[row.Object] = true
	}
	for _, binding := range bindings {
		highWater = max(highWater, binding.ID)
	}
	var previous uint32
	for _, binding := range bindings {
		id := byObject[binding.ObjectIndex]
		if id == 0 {
			if !unmodeled[binding.ObjectIndex] {
				return fmt.Errorf("current Player identity has an unlisted ordinary root")
			}
			id = binding.ID
			if id <= previous {
				if highWater == ^uint32(0) {
					return fmt.Errorf("current Player promotion exhausts identity space")
				}
				highWater++
				id = highWater
			}
		}
		if id <= previous {
			return fmt.Errorf("current Player root order differs from exact identities")
		}
		previous, ids[binding.ID] = id, id
	}
	if err := ms.World.RestoreCurrentPlayerIdentities(ids); err != nil {
		return err
	}
	if state.PlayerRoots != nil {
		for i := range *state.PlayerRoots {
			p := &(*state.PlayerRoots)[i]
			p.ID = ids[p.ID]
		}
	}
	if state.GroupBindings != nil {
		for i := range state.GroupBindings.Players {
			p := &state.GroupBindings.Players[i]
			if id, ok := ids[p.ID]; ok {
				p.ID = id
			}
		}
		for i := range state.GroupBindings.Groups {
			g := &state.GroupBindings.Groups[i]
			if g.ContainerID != 0 {
				g.ContainerID = ids[g.ContainerID]
			}
		}
	}
	if state.PlayerPurses != nil {
		for i := range state.PlayerPurses.Players {
			p := &state.PlayerPurses.Players[i]
			if id, ok := ids[p.PlayerID]; ok {
				p.PlayerID = id
			}
		}
	}
	return nil
}
