package sim

import "fmt"

type GroupContinuation struct {
	Group, ID uint32
	Authored  bool
	RootOnly  bool
}

func (w *World) GroupHighWater() uint32 {
	if w.savedGroups == nil {
		return 0
	}
	return w.savedGroups.HighWater
}

// The archive owns membership and AI. These local identities preserve the
// current dispatch order, command lifetime and next insertion cursor.
func (w *World) RestoreGroupContinuations(rows []GroupContinuation, highWater uint32, absentPlayers ...uint32) error {
	if w.savedGroups == nil || len(rows) != len(w.savedGroups.Groups) {
		return fmt.Errorf("sim: incomplete current Group continuation")
	}
	n := cloneSavedGroups(w.savedGroups)
	current := cloneCurrentPlayers(w.currentPlayers)
	byID := map[uint32]SavedGroup{}
	for _, g := range n.Groups {
		byID[g.ID] = g
	}
	n.Groups = nil
	n.HighWater = highWater
	for _, row := range rows {
		g, ok := byID[row.Group]
		if !ok {
			return fmt.Errorf("sim: absent or repeated current Group")
		}
		delete(byID, row.Group)
		if row.RootOnly {
			if row.Authored {
				return fmt.Errorf("sim: actor root cannot restore a native authored Group")
			}
			continue
		}
		g.ID, g.Authored = row.ID, row.Authored
		if g.Authored {
			if g.Reference != (SavedGroupReference{}) || g.Owner.Class != 1 {
				return fmt.Errorf("sim: authored Group lacks its current owner")
			}
			g.Owner.Key, g.Owner.Archive = 0, 0
		}
		n.Groups = append(n.Groups, g)
	}
	if len(absentPlayers) != 0 {
		if !n.PlayersPresent || len(absentPlayers) > len(n.Players) {
			return fmt.Errorf("sim: absent current Players lack their exact carrier")
		}
		known, absent := map[uint32]bool{}, map[uint32]bool{}
		for _, p := range n.Players {
			known[p.ID] = true
		}
		for _, id := range absentPlayers {
			if id == 0 || !known[id] || absent[id] {
				return fmt.Errorf("sim: absent current Player identity is unknown or repeated")
			}
			absent[id] = true
		}
		for _, g := range n.Groups {
			if absent[g.ContainerID] || absent[g.OwnerID] {
				return fmt.Errorf("sim: absent current Player still owns a native Group")
			}
		}
		players := n.Players[:0]
		for _, p := range n.Players {
			if !absent[p.ID] {
				players = append(players, p)
			}
		}
		n.Players = players
		formations := n.Formations[:0]
		for _, p := range n.Formations {
			if !absent[p.PlayerID] {
				formations = append(formations, p)
			}
		}
		n.Formations = formations
		if current != nil {
			kept := current.Players[:0]
			for _, p := range current.Players {
				if !absent[p.ID] {
					kept = append(kept, p)
				}
			}
			current.Players = kept
			participants := current.Participants[:0]
			for _, p := range current.Participants {
				if !absent[p.PlayerID] {
					participants = append(participants, p)
				}
			}
			current.Participants = participants
		}
	}
	if err := savedGroupsFault(n, w.entities, w.originalDead); err != nil {
		return err
	}
	if err := currentPlayersFault(current); err != nil {
		return err
	}
	w.savedGroups = n
	w.currentPlayers = current
	return nil
}

// Ordinary Players own the transported actor graph even when the native
// registry has never acquired exact container or formation identities.
func (w *World) RestoreGroupCarrierPresence(players, formations bool) error {
	if w.savedGroups == nil || formations && !players {
		return fmt.Errorf("sim: invalid current Group carrier presence")
	}
	n := cloneSavedGroups(w.savedGroups)
	if !players {
		n.Players, n.PlayersPresent = nil, false
		for i := range n.Groups {
			n.Groups[i].ContainerID = 0
		}
	}
	if !formations {
		n.Formations, n.FormationsPresent = nil, false
		for i := range n.Groups {
			n.Groups[i].OwnerID = 0
		}
	}
	if err := savedGroupsFault(n, w.entities, w.originalDead); err != nil {
		return err
	}
	w.savedGroups = n
	return nil
}
