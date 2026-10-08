package sim

import (
	"fmt"
	"slices"
)

// CurrentPlayerSlot binds one ordinary Player's temporary slot to an absent
// native slot. Keys identify references; Player identifies exact containment.
type CurrentPlayerSlot struct {
	Player, Key, Wire, Native uint32
	Trigger                   *uint32
	Shared                    bool
}

func (w *World) RestoreCurrentPlayerSlots(rows []CurrentPlayerSlot, money map[uint32]uint32) error {
	if w.savedGroups == nil || !w.savedGroups.PlayersPresent {
		return fmt.Errorf("sim: current Player slots lack imported containers")
	}
	next := *w
	next.entities = slices.Clone(w.entities)
	next.savedGroups = cloneSavedGroups(w.savedGroups)
	next.currentPlayers = cloneCurrentPlayers(w.currentPlayers)
	byPlayer, byKey := map[uint32]CurrentPlayerSlot{}, map[uint32]CurrentPlayerSlot{}
	for _, r := range rows {
		if r.Player == 0 || r.Key == 0 || r.Wire == 0 || r.Wire > 16 || r.Wire == r.Native && r.Trigger == nil && !r.Shared || r.Native >= relationSlots ||
			r.Trigger != nil && *r.Trigger == r.Native || byPlayer[r.Player].Player != 0 || byKey[r.Key].Player != 0 {
			return fmt.Errorf("sim: invalid current Player slot binding")
		}
		found := false
		for _, p := range next.savedGroups.Players {
			if p.ID == r.Player {
				found = p.Slot == r.Wire
			}
		}
		if !found {
			return fmt.Errorf("sim: current Player slot differs from ordinary container")
		}
		byPlayer[r.Player], byKey[r.Key] = r, r
	}
	for i := range next.savedGroups.Players {
		p := &next.savedGroups.Players[i]
		if r, ok := byPlayer[p.ID]; ok {
			p.Slot = r.Native
		}
	}
	if next.currentPlayers != nil {
		for i := range next.currentPlayers.Players {
			p := &next.currentPlayers.Players[i]
			if r, ok := byPlayer[p.ID]; ok {
				if p.Slot != r.Wire {
					return fmt.Errorf("sim: current Player slot differs from its exact registry")
				}
				p.Slot = r.Native
			}
		}
	}
	actors := map[EntityID]CurrentPlayerSlot{}
	for i := range next.savedGroups.Groups {
		g := &next.savedGroups.Groups[i]
		if r, ok := byPlayer[g.ContainerID]; ok {
			for _, member := range g.Members {
				if member.Bound {
					actors[member.Entity] = r
				}
			}
		}
		for _, ref := range []*SavedGroupReference{&g.Reference, &g.Owner} {
			if r, ok := byKey[ref.Key]; ok && ref.Class == 1 {
				ref.Owner = r.Native
			}
		}
	}
	for i := range next.entities {
		e := &next.entities[i]
		if r, ok := actors[e.ID]; ok {
			if e.Owner != r.Wire {
				return fmt.Errorf("sim: current actor owner differs from exact Player")
			}
			e.Owner = r.Native
		}
		if r, ok := byKey[e.SourceBinding.GroupOwnerKey]; ok && e.SourceBinding.GroupOwnerResolved {
			e.SourceBinding.GroupOwnerSlot = uint16(r.Native)
		}
	}
	for i := range next.savedGroups.Formations {
		f := &next.savedGroups.Formations[i]
		if r, ok := byPlayer[f.PlayerID]; ok {
			f.CommandID = int16(r.Native)
			if r.Trigger != nil {
				f.TriggerID = *r.Trigger
			} else if f.TriggerID == r.Wire {
				f.TriggerID = r.Native
			}
		}
	}
	for _, r := range rows {
		occupied := false
		for _, p := range next.savedGroups.Players {
			occupied = occupied || p.Slot == r.Wire
		}
		if !occupied {
			next.purses[r.Wire] = 0
		}
	}
	for slot, value := range money {
		if slot >= relationSlots {
			return fmt.Errorf("sim: current Player money exceeds native slots")
		}
		next.purses[slot] = value
	}
	if err := savedGroupsFault(next.savedGroups, next.entities, next.originalDead); err != nil {
		return err
	}
	*w = next
	return nil
}
