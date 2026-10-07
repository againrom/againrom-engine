package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Nil is the pre-owner native state. A retained document does not authorize
// reconstructing an absent current purse binding on native LOAD.
type SnapshotSAVPlayerPurses struct {
	Version uint32
	Players []SnapshotSAVPlayerPurse
}

// PlayerID names SavedGroupPlayer.ID. The existing Group Player binding owns
// the current document index, including changes made by graph reindexing.
// Slot is only the native purse lookup operand, never object identity.
type SnapshotSAVPlayerPurse struct {
	PlayerID    uint32
	Slot        uint32
	Unavailable string
}

const (
	savedPurseSlotUnavailable = "Player slot is outside the native purse domain"
	savedPurseSlotAmbiguous   = "distinct Player objects share one native purse slot"
)

func savedPlayerPurseRows(doc *sav.DocumentData, groups *SnapshotSAVGroupBindings) ([]SnapshotSAVPlayerPurse, error) {
	if doc == nil || groups == nil || !groups.PlayersPresent {
		return nil, fmt.Errorf("saved SAV Player purses lack exact Player bindings")
	}
	rows := make([]SnapshotSAVPlayerPurse, len(groups.Players))
	counts := make(map[uint32]int, len(rows))
	for i, binding := range groups.Players {
		if binding.ID == 0 || i > 0 && groups.Players[i-1].ID >= binding.ID || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(doc.Objects) {
			return nil, fmt.Errorf("saved SAV Player purse has invalid exact binding")
		}
		record := &doc.Objects[binding.ObjectIndex-1]
		if record.Class != "Player" {
			return nil, fmt.Errorf("saved SAV Player purse binds a non-Player")
		}
		slot, err := savedStructureValue(record, "Slot")
		if err != nil || slot > 0xffff {
			return nil, fmt.Errorf("saved SAV Player purse has invalid Slot")
		}
		if _, err := savedStructureValue(record, "Money"); err != nil {
			return nil, err
		}
		rows[i] = SnapshotSAVPlayerPurse{PlayerID: binding.ID, Slot: slot}
		counts[slot]++
	}
	for i := range rows {
		// SetPurse owns this implementation's domain. A zero World value has
		// its own fixed purse array; this probe changes no mission state and
		// avoids confusing the source u16 domain with native capacity.
		var probe sim.World
		if !probe.SetPurse(rows[i].Slot, 0) {
			rows[i].Unavailable = savedPurseSlotUnavailable
		} else if counts[rows[i].Slot] != 1 {
			rows[i].Unavailable = savedPurseSlotAmbiguous
		}
	}
	return rows, nil
}

func cloneSavedPlayerPurses(src *SnapshotSAVPlayerPurses, doc *sav.DocumentData, groups *SnapshotSAVGroupBindings) (*SnapshotSAVPlayerPurses, error) {
	if src == nil {
		return nil, nil
	}
	if src.Version != 1 {
		return nil, fmt.Errorf("saved SAV Player purse version is unsupported")
	}
	want, err := savedPlayerPurseRows(doc, groups)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(src.Players, want) {
		return nil, fmt.Errorf("saved SAV Player purse coverage differs from exact Player bindings")
	}
	return &SnapshotSAVPlayerPurses{Version: 1, Players: slices.Clone(src.Players)}, nil
}

func savedPlayerPurseWorld(state *SnapshotSAVDocument, world *sim.World, compareMoney bool) error {
	if state == nil || state.PlayerPurses == nil {
		return nil
	}
	if world == nil {
		return fmt.Errorf("saved SAV Player purses have no native world")
	}
	if _, err := cloneSavedPlayerPurses(state.PlayerPurses, state.Document, state.GroupBindings); err != nil {
		return err
	}
	players, present := world.SavedGroupPlayers()
	if !present {
		return nil
	}
	if len(players) != len(nativeSavedPlayerBindings(state.GroupBindings)) {
		return fmt.Errorf("saved SAV Player purses differ from native Player population")
	}
	byID := make(map[uint32]sim.SavedGroupPlayer, len(players))
	for _, player := range players {
		byID[player.ID] = player
	}
	for i, row := range state.PlayerPurses.Players {
		player, exists := byID[row.PlayerID]
		constructed := state.GroupBindings.Players[i].Constructed
		if constructed && exists || !constructed && (!exists || player.Slot != row.Slot) {
			return fmt.Errorf("saved SAV Player purse %d differs from its native Player", row.PlayerID)
		}
		if compareMoney && row.Unavailable == "" {
			object := state.GroupBindings.Players[i].ObjectIndex
			money, err := savedStructureValue(&state.Document.Objects[object-1], "Money")
			if err != nil || money != world.Purse(row.Slot) {
				return fmt.Errorf("saved SAV Player purse %d Money differs from native world", row.PlayerID)
			}
		}
	}
	return nil
}

// SAV-PLAYER-028 and SAV-OBF-029: Money is the entire decoded u32 at Player+38.
// No participant/first-root restriction or signed host-gold conversion applies.
// Distinct equal-slot Players cannot both claim one native purse; keep their
// exact original scalars explicitly uncovered instead of choosing a winner.
func importSavedPlayerPurses(ms *Mission, state *SnapshotSAVDocument) error {
	if state.GroupBindings == nil || !state.GroupBindings.PlayersPresent {
		return restoreSavedDocument(ms, state)
	}
	rows, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
	if err != nil {
		return err
	}
	nextState := *state
	nextState.PlayerPurses = &SnapshotSAVPlayerPurses{Version: 1, Players: rows}
	if err := savedPlayerPurseWorld(&nextState, ms.World, false); err != nil {
		return err
	}
	// SetPurse only writes the World's inline array. Both it and the eventual
	// complete-document validation run before either owner is published.
	nextWorld := *ms.World
	for i, row := range rows {
		if row.Unavailable != "" {
			continue
		}
		object := state.GroupBindings.Players[i].ObjectIndex
		money, err := savedStructureValue(&state.Document.Objects[object-1], "Money")
		if err != nil || !nextWorld.SetPurse(row.Slot, money) {
			return fmt.Errorf("saved SAV Player purse %d could not be imported", row.PlayerID)
		}
	}
	nextMission := *ms
	nextMission.World = &nextWorld
	if err := importSavedAutoHealing(&nextState, &nextWorld); err != nil {
		return err
	}
	if err := restoreSavedDocument(&nextMission, &nextState); err != nil {
		return err
	}
	*ms.World = nextWorld
	ms.savedDocument = nextMission.savedDocument
	return nil
}

// Actual simulation gold changes (including TRIG-MONEY-028 and ITEM-PICK-009)
// write World.Purse. Snapshot projects those current unsigned values only for
// the explicitly admitted exact Player/slot bindings.
func projectSavedPlayerPurses(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.PlayerPurses == nil {
		return nil
	}
	if err := savedPlayerPurseWorld(state, world, false); err != nil {
		return err
	}
	type update struct {
		index  uint16
		record sav.DocumentRecordData
	}
	updates := make([]update, 0, len(state.PlayerPurses.Players))
	for i, row := range state.PlayerPurses.Players {
		if row.Unavailable != "" {
			continue
		}
		object := state.GroupBindings.Players[i].ObjectIndex
		record := state.Document.Objects[object-1]
		record.Values = slices.Clone(record.Values)
		if err := savedActorSetValue(&record, "Money", world.Purse(row.Slot)); err != nil {
			return err
		}
		updates = append(updates, update{index: object, record: record})
	}
	for _, update := range updates {
		state.Document.Objects[update.index-1] = update.record
	}
	return nil
}
