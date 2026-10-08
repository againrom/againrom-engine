package sim

import "fmt"

// RestoreCurrentPlayerIdentities renames exact IDs without changing dispatch.
func (w *World) RestoreCurrentPlayerIdentities(ids map[uint32]uint32) error {
	players, present := w.CurrentPlayers()
	if !present || len(ids) != len(players) {
		return fmt.Errorf("sim: current Player identity map is incomplete")
	}
	seen := map[uint32]bool{}
	for _, player := range players {
		id := ids[player.ID]
		if id == 0 || seen[id] {
			return fmt.Errorf("sim: current Player identity map is not one to one")
		}
		seen[id] = true
	}
	next := *w
	next.currentPlayers = cloneCurrentPlayers(w.currentPlayers)
	next.savedGroups = cloneSavedGroups(w.savedGroups)
	if next.currentPlayers != nil {
		for i := range next.currentPlayers.Players {
			p := &next.currentPlayers.Players[i]
			p.ID = ids[p.ID]
		}
		for i := range next.currentPlayers.Participants {
			p := &next.currentPlayers.Participants[i]
			p.PlayerID = ids[p.PlayerID]
		}
	}
	if next.savedGroups != nil && next.savedGroups.PlayersPresent {
		for i := range next.savedGroups.Players {
			p := &next.savedGroups.Players[i]
			p.ID = ids[p.ID]
		}
		for i := range next.savedGroups.Groups {
			g := &next.savedGroups.Groups[i]
			if g.ContainerID != 0 {
				g.ContainerID = ids[g.ContainerID]
			}
			if g.OwnerID != 0 {
				g.OwnerID = ids[g.OwnerID]
			}
		}
		for i := range next.savedGroups.Formations {
			p := &next.savedGroups.Formations[i]
			p.PlayerID = ids[p.PlayerID]
		}
	}
	if err := next.currentPlayersFault(); err != nil {
		return err
	}
	if err := savedGroupsFault(next.savedGroups, next.entities, next.originalDead); err != nil {
		return err
	}
	*w = next
	return nil
}
