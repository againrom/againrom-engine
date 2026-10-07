package sim

import (
	"fmt"
	"slices"
)

// SavedGroupPlayer identifies an exact Player container independently of its
// semantic slot. IDs are opaque native identities, not slots or source keys.
type SavedGroupPlayer struct {
	ID, Slot uint32
}

type SavedGroupContainer struct {
	GroupID, PlayerID uint32
}

const savedGroupPlayerLimit = 65535

// SavedGroupPlayers distinguishes old native saves from a present empty
// registry. It never reconstructs containment from Group.Owner or actors.
func (w *World) SavedGroupPlayers() ([]SavedGroupPlayer, bool) {
	if w.savedGroups == nil || !w.savedGroups.PlayersPresent {
		return nil, false
	}
	return slices.Clone(w.savedGroups.Players), true
}

// ImportSavedGroupPlayers installs complete initial provenance once, after the
// source Groups have been imported. Subsequent Group updates retain it.
func (w *World) ImportSavedGroupPlayers(players []SavedGroupPlayer, containers []SavedGroupContainer) error {
	if w.savedGroups == nil || w.savedGroups.PlayersPresent {
		return fmt.Errorf("sim: initial Player containers require an unbound saved Group registry")
	}
	if len(players) > savedGroupPlayerLimit || len(containers) > savedGroupPlayerLimit || len(containers) != len(w.savedGroups.Groups) {
		return fmt.Errorf("sim: Player container counts are invalid")
	}
	s := cloneSavedGroups(w.savedGroups)
	s.Players, s.PlayersPresent = slices.Clone(players), true
	byID := make(map[uint32]uint32, len(containers))
	for i, c := range containers {
		if c.GroupID == 0 || c.PlayerID == 0 || i > 0 && containers[i-1].GroupID >= c.GroupID {
			return fmt.Errorf("sim: Player container mappings are not exact ordered identities")
		}
		byID[c.GroupID] = c.PlayerID
	}
	for i := range s.Groups {
		s.Groups[i].ContainerID = byID[s.Groups[i].ID]
		if s.Groups[i].ContainerID == 0 {
			return fmt.Errorf("sim: Player containers do not cover every saved Group")
		}
	}
	if err := savedGroupPlayersFault(s); err != nil {
		return err
	}
	w.savedGroups = s
	return nil
}

func maxSavedGroupID(groups []SavedGroup) uint32 {
	var id uint32
	for _, g := range groups {
		id = max(id, g.ID)
	}
	return id
}

func savedGroupPlayersFault(s *savedGroupState) error {
	if len(s.Players) > savedGroupPlayerLimit || len(s.Groups) > savedGroupPlayerLimit || s.HighWater < maxSavedGroupID(s.Groups) || !s.PlayersPresent && len(s.Players) != 0 {
		return fmt.Errorf("sim: invalid saved Player registry bounds or Group highwater")
	}
	players := make(map[uint32]bool, len(s.Players))
	for i, p := range s.Players {
		if p.ID == 0 || i > 0 && s.Players[i-1].ID >= p.ID {
			return fmt.Errorf("sim: saved Player identities are not strictly ordered")
		}
		players[p.ID] = true
	}
	var priorContainer uint32
	unknown := false
	for _, g := range s.Groups {
		if g.ContainerID != 0 && (!s.PlayersPresent || !players[g.ContainerID]) {
			return fmt.Errorf("sim: saved Group %d has an unknown Player container", g.ID)
		}
		if s.PlayersPresent {
			if g.ContainerID == 0 {
				unknown = true
			} else {
				if unknown || g.ContainerID < priorContainer {
					return fmt.Errorf("sim: saved Group Player container traversal is not ordered")
				}
				priorContainer = g.ContainerID
			}
		}
	}
	return nil
}

func (w *World) savedPlayerByID(id uint32) (SavedGroupPlayer, bool) {
	if w.savedGroups != nil && w.savedGroups.PlayersPresent {
		for _, p := range w.savedGroups.Players {
			if p.ID == id {
				return p, true
			}
		}
	}
	return SavedGroupPlayer{}, false
}

func (w *World) uniqueSavedPlayerSlot(slot uint32) uint32 {
	var id uint32
	if w.savedGroups != nil && w.savedGroups.PlayersPresent {
		for _, p := range w.savedGroups.Players {
			if p.Slot == slot {
				if id != 0 {
					return 0
				}
				id = p.ID
			}
		}
	}
	return id
}

func (w *World) savedCommandContainer(members []int) uint32 {
	if len(members) == 0 || w.savedGroups == nil || !w.savedGroups.PlayersPresent {
		return 0
	}
	var container uint32
	allKnown := true
	owner := w.entities[members[0]].Owner
	for _, i := range members {
		if w.entities[i].Owner != owner {
			return 0
		}
		g := w.savedGroupFor(w.entities[i].ID)
		if g == nil || g.ContainerID == 0 {
			allKnown = false
			continue
		}
		p, found := w.savedPlayerByID(g.ContainerID)
		if !found || container != 0 && container != p.ID {
			return 0
		}
		if p.Slot != owner {
			allKnown = false
		}
		container = p.ID
	}
	if allKnown {
		return container
	}
	return w.uniqueSavedPlayerSlot(owner)
}

// AI-CMD-033/SAV-GRPCMD-578: one preexisting rejected (empty in this bounded
// native model) Group, in this exact Player list, before member detachment.
func (w *World) removeFirstEmptySavedGroup(container uint32) {
	if container == 0 {
		return
	}
	for i, g := range w.savedGroups.Groups {
		if g.ContainerID == container && len(g.Members) == 0 {
			w.savedGroups.Groups = slices.Delete(w.savedGroups.Groups, i, i+1)
			return
		}
	}
}

// Known containers traverse their native Player blocks. Inserting at that
// block's tail preserves old relative order and the explicit per-Player list.
// Unknown and absent-container Groups retain the older global append policy.
func (w *World) appendSavedCommandGroup(g SavedGroup) {
	if w.savedGroups.PlayersPresent && g.ContainerID != 0 {
		at := len(w.savedGroups.Groups)
		for i, old := range w.savedGroups.Groups {
			if old.ContainerID == 0 || old.ContainerID > g.ContainerID {
				at = i
				break
			}
		}
		w.savedGroups.Groups = slices.Insert(w.savedGroups.Groups, at, g)
		return
	}
	w.savedGroups.Groups = append(w.savedGroups.Groups, g)
}
