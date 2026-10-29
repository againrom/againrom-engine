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
	if w.groupDialectNative() || !w.savedGroups.PlayersPresent {
		return nil, false
	}
	return slices.Clone(w.savedGroups.Players), true
}

// ImportSavedGroupPlayers installs complete initial provenance once, after the
// source Groups have been imported. Subsequent Group updates retain it.
func (w *World) ImportSavedGroupPlayers(players []SavedGroupPlayer, containers []SavedGroupContainer) error {
	if w.groupDialectNative() || w.savedGroups.PlayersPresent {
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
	if w.currentPlayers != nil && !slices.Equal(w.currentPlayers.Players, s.Players) {
		w.currentPlayers = &currentPlayerState{Players: slices.Clone(s.Players)}
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
	if !w.groupDialectNative() && w.savedGroups.PlayersPresent {
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
	if !w.groupDialectNative() && w.savedGroups.PlayersPresent {
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
	if len(members) == 0 || w.groupDialectNative() || !w.savedGroups.PlayersPresent {
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

// SaveKeepsGroup is the SAVE writer's Group rule: a Player whose participant
// word +28 is zero loses its empty Groups before the file is built (SAV-1251);
// every other Group is written, an empty one empty (SAV-1234).
func SaveKeepsGroup(participant uint32, members int) bool {
	return members != 0 || participant != 0
}

// KeepGroupsForSave is the one applier of SaveKeepsGroup: the World's prune,
// the town's prune and the town writer decide through it. at answers Group
// i's Player participant word, its member count and whether a Player holds
// it; a Group no Player holds is kept. It returns the kept Groups in order and
// whether SAVE drops each Group; with no drop it returns groups and nil.
func KeepGroupsForSave[G any](groups []G, at func(i int) (participant uint32, members int, held bool)) ([]G, []bool) {
	var drops []bool
	for i := range groups {
		participant, members, held := at(i)
		if held && !SaveKeepsGroup(participant, members) {
			if drops == nil {
				drops = make([]bool, len(groups))
			}
			drops[i] = true
		}
	}
	if drops == nil {
		return groups, nil
	}
	kept := groups[:0:0]
	for i, g := range groups {
		if !drops[i] {
			kept = append(kept, g)
		}
	}
	return kept, drops
}

// PruneGroupsForSave applies SaveKeepsGroup to the live lists, as the
// original's SAVE removes the Groups from its live Player lists before it
// opens the file (SAV-1251). The World half and the document a SAVE writes
// then hold the lists play continues with, so a LOAD resumes the same World.
// The registry a map start built is pruned as a LOADed one is. It reports
// whether a Group was removed.
func (w *World) PruneGroupsForSave() bool {
	pruned, _ := w.PruneGroupsForSaveUndo()
	return pruned
}

// PruneGroupsForSaveUndo is PruneGroupsForSave and also returns the function
// that puts the World back as it was before the prune. A save dialog builds
// the file from the pruned lists but writes nothing until the player has
// confirmed an overwrite, so it undoes the prune after the build and prunes
// again when the file is written. The undo is never nil.
func (w *World) PruneGroupsForSaveUndo() (bool, func()) {
	if w.groupDialectNative() {
		return false, func() {}
	}
	s := cloneSavedGroups(w.savedGroups)
	kept, drops := KeepGroupsForSave(s.Groups, func(i int) (uint32, int, bool) {
		participant, held := w.savedGroupParticipant(&s.Groups[i])
		return participant, len(s.Groups[i].Members), held
	})
	if drops == nil {
		return false, func() {}
	}
	before := w.savedGroups
	bindings := make([]SourceBinding, len(w.entities))
	for i := range w.entities {
		bindings[i] = w.entities[i].SourceBinding
	}
	dropped := map[uint32]bool{}
	for i, g := range s.Groups {
		if drops[i] {
			dropped[g.ID] = true
		}
	}
	s.Groups = kept
	w.savedGroups = s
	w.rebindDroppedSourceGroups(dropped)
	w.syncSourceGroupSelectors()
	return true, func() {
		w.savedGroups = before
		for i := range bindings {
			if i < len(w.entities) {
				w.entities[i].SourceBinding = bindings[i]
			}
		}
	}
}

// rebindDroppedSourceGroups moves an actor's source Group from a dropped
// Group to the Group that holds it, or to none, and copies that Group's
// selector and owner as a LOAD reads them, so the World a SAVE writes and the
// World its LOAD builds name the same source Group.
func (w *World) rebindDroppedSourceGroups(dropped map[uint32]bool) {
	for i := range w.entities {
		e := &w.entities[i]
		if !dropped[e.SourceBinding.GroupIndex] {
			continue
		}
		b := &e.SourceBinding
		b.GroupIndex, b.GroupSelector, b.GroupOwnerKey, b.GroupOwnerSlot, b.GroupOwnerResolved = 0, 0, 0, 0, false
		if g := w.savedGroupFor(e.ID); g != nil {
			b.GroupIndex, b.GroupSelector, b.GroupOwnerKey = g.ID, g.Selector, g.Owner.Key
			b.GroupOwnerSlot, b.GroupOwnerResolved = uint16(g.Owner.Owner), g.Owner.Class != 0
		}
	}
}

// savedGroupParticipant answers the participant word of the Player whose list
// holds g: the exact container's current word when the World holds one, else
// zero for the slot the human-participant rule selects and one for any other.
// A Group no Player holds answers false.
func (w *World) savedGroupParticipant(g *SavedGroup) (uint32, bool) {
	if s := w.currentPlayers; s != nil && s.ParticipantsPresent && g.ContainerID != 0 {
		for _, row := range s.Participants {
			if row.PlayerID == g.ContainerID {
				return row.Value, true
			}
		}
	}
	slot := uint32(0)
	if p, found := w.savedPlayerByID(g.ContainerID); found && g.ContainerID != 0 {
		slot = p.Slot
	} else if g.Owner.Class == 1 {
		slot = g.Owner.Owner
	}
	if slot == 0 {
		return 0, false
	}
	if w.humanParticipantSlot(slot) {
		return 0, true
	}
	return 1, true
}
