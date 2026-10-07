package sim

import (
	"fmt"
	"slices"
)

// SavedPlayerFormation belongs to one opaque native Player identity. CommandID
// is signed Player+04; TriggerID is Player+08. Neither is a container ID, source
// key, archive position or actor owner slot (AI-FORMCMD-315/AI-FORMTRIGGER-316).
type SavedPlayerFormation struct {
	PlayerID  uint32
	CommandID int16
	TriggerID uint32
	Mode      uint8
}

type SavedGroupOwner struct {
	GroupID, PlayerID uint32
}

func (w *World) SavedPlayerFormations() ([]SavedPlayerFormation, bool) {
	if w.savedGroups == nil || !w.savedGroups.FormationsPresent {
		return nil, false
	}
	return slices.Clone(w.savedGroups.Formations), true
}

// ImportSavedPlayerFormations installs a complete exact carrier once. Original
// import supplies serializer receivers and resolved owners; the native decoder
// supplies only its persisted footer, never reconstructed Document values.
func (w *World) ImportSavedPlayerFormations(players []SavedPlayerFormation, owners []SavedGroupOwner) error {
	if w.savedGroups == nil || !w.savedGroups.PlayersPresent || w.savedGroups.FormationsPresent {
		return fmt.Errorf("sim: formation import requires unbound exact Players")
	}
	s := cloneSavedGroups(w.savedGroups)
	s.Formations, s.FormationsPresent = slices.Clone(players), true
	if len(owners) != len(s.Groups) {
		return fmt.Errorf("sim: formation owners do not cover every Group")
	}
	byID := make(map[uint32]uint32, len(owners))
	for i, owner := range owners {
		if owner.GroupID == 0 || i > 0 && owners[i-1].GroupID >= owner.GroupID {
			return fmt.Errorf("sim: formation Group owners are not strictly ordered")
		}
		byID[owner.GroupID] = owner.PlayerID
	}
	for i := range s.Groups {
		id, found := byID[s.Groups[i].ID]
		if !found {
			return fmt.Errorf("sim: formation owner names another Group")
		}
		s.Groups[i].OwnerID = id
	}
	if err := savedFormationsFault(s); err != nil {
		return err
	}
	w.savedGroups = s
	return nil
}

func savedFormationsFault(s *savedGroupState) error {
	if !s.FormationsPresent {
		if len(s.Formations) != 0 {
			return fmt.Errorf("sim: absent Player formations carry records")
		}
	} else if !s.PlayersPresent || len(s.Formations) != len(s.Players) {
		return fmt.Errorf("sim: formations do not cover exact Players")
	}
	ids := make(map[uint32]bool, len(s.Formations))
	for i, p := range s.Formations {
		if p.PlayerID != s.Players[i].ID || uint32(uint16(p.CommandID)) != s.Players[i].Slot {
			return fmt.Errorf("sim: formation Player identity or command identifier mismatch")
		}
		ids[p.PlayerID] = true
	}
	for _, g := range s.Groups {
		if g.OwnerID != 0 && (!s.FormationsPresent || !ids[g.OwnerID] || g.Owner.Class != 1) {
			return fmt.Errorf("sim: Group %d has invalid exact formation owner", g.ID)
		}
		// Generated Group ownership is an existing native construction policy.
		// Imported owners may disagree with containment and actor ownership.
		if s.FormationsPresent && g.Authored && g.OwnerID != g.ContainerID {
			return fmt.Errorf("sim: authored Group %d formation owner differs from construction", g.ID)
		}
	}
	return nil
}

func (w *World) commandFormation(player uint32) *SavedPlayerFormation {
	for i := range w.savedGroups.Formations {
		p := &w.savedGroups.Formations[i]
		if p.CommandID == int16(uint16(player)) {
			return p // AI-FORMCMD-315: first signed-word match in list order.
		}
	}
	return nil
}

func (w *World) triggerFormation(player uint32) *SavedPlayerFormation {
	var found *SavedPlayerFormation
	for i := range w.savedGroups.Formations {
		p := &w.savedGroups.Formations[i]
		if p.TriggerID == player {
			if found != nil {
				return nil // Temporary-map collisions remain Unknown; do not guess.
			}
			found = p
		}
	}
	return found
}

func (w *World) hasSavedFormations() bool {
	return w.savedGroups != nil && w.savedGroups.FormationsPresent
}

// CommandFormationMode reads the same target the existing client command will
// update. A missing target is explicit; SelfSlot/active-client policy is unchanged.
func (w *World) CommandFormationMode(player uint32) (uint8, bool) {
	if w.hasSavedFormations() {
		if p := w.commandFormation(player); p != nil {
			return p.Mode, true
		}
		return 0, false
	}
	return w.FormationMode(player), player < relationSlots
}

func (w *World) setCommandFormation(player uint32, mode uint8) {
	if w.hasSavedFormations() {
		if p := w.commandFormation(player); p != nil {
			p.Mode = mode
		}
		return
	}
	w.setFormationMode(player, int32(mode))
}

func (w *World) savedGroupFormation(g *SavedGroup) (uint8, bool) {
	if g == nil || g.OwnerID == 0 {
		return 0, false
	}
	for _, p := range w.savedGroups.Formations {
		if p.PlayerID == g.OwnerID {
			return p.Mode, true
		}
	}
	return 0, false
}

func (w *World) savedFormationGroup(members []int) *SavedGroup {
	if len(members) == 0 {
		return nil
	}
	return w.savedGroupFor(w.entities[members[0]].ID)
}
