package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentPlayerRoots(state *SnapshotSAVDocument) []SnapshotSAVGroupPlayerBinding {
	if state.PlayerRoots != nil {
		return *state.PlayerRoots
	}
	if state.GroupBindings != nil {
		return nativeSavedPlayerBindings(state.GroupBindings)
	}
	return nil
}

func cloneCurrentPlayerRoots(src *[]SnapshotSAVGroupPlayerBinding, doc *sav.DocumentData) (*[]SnapshotSAVGroupPlayerBinding, error) {
	if src == nil {
		return nil, nil
	}
	if len(*src) > 65535 || doc == nil {
		return nil, fmt.Errorf("current Player roots lack bounded document")
	}
	roots, seen := map[uint16]bool{}, map[uint16]bool{}
	for _, object := range doc.Players {
		roots[object] = object != 0
	}
	for i, row := range *src {
		if row.ID == 0 || row.ID > 65535 || i > 0 && (*src)[i-1].ID >= row.ID || row.Constructed || !roots[row.ObjectIndex] || int(row.ObjectIndex) > len(doc.Objects) || doc.Objects[row.ObjectIndex-1].Class != "Player" || seen[row.ObjectIndex] {
			return nil, fmt.Errorf("current Player roots have ambiguous exact identity")
		}
		seen[row.ObjectIndex] = true
	}
	rows := slices.Clone(*src)
	return &rows, nil
}

func importCurrentPlayerRoots(state *SnapshotSAVDocument) error {
	rows := []SnapshotSAVGroupPlayerBinding{}
	seen := map[uint16]bool{}
	for _, object := range state.Document.Players {
		if object == 0 || seen[object] {
			continue
		}
		seen[object] = true
		rows = append(rows, SnapshotSAVGroupPlayerBinding{ID: uint32(len(rows) + 1), ObjectIndex: object})
	}
	checked, err := cloneCurrentPlayerRoots(&rows, state.Document)
	if err != nil {
		return err
	}
	state.PlayerRoots = checked
	return nil
}

func bindCurrentPlayerRoots(state *SnapshotSAVDocument, w *sim.World) error {
	players, present := w.CurrentPlayers()
	if !present {
		state.PlayerRoots = nil
		return nil
	}
	byID := map[uint32]SnapshotSAVGroupPlayerBinding{}
	for _, row := range currentPlayerRoots(state) {
		byID[row.ID] = row
	}
	if state.GroupBindings != nil {
		for _, row := range state.GroupBindings.Players {
			if !row.Constructed {
				if old, exists := byID[row.ID]; exists && old.ObjectIndex != row.ObjectIndex {
					return fmt.Errorf("current Player root conflicts with command container")
				}
				byID[row.ID] = row
			}
		}
	}
	rows := make([]SnapshotSAVGroupPlayerBinding, len(players))
	for i, player := range players {
		row, exists := byID[player.ID]
		if !exists {
			return fmt.Errorf("current Player has no exact source or constructed root")
		}
		rows[i] = row
	}
	checked, err := cloneCurrentPlayerRoots(&rows, state.Document)
	if err != nil {
		return err
	}
	state.PlayerRoots = checked
	return nil
}

func validateCurrentPlayerRoots(state *SnapshotSAVDocument, w *sim.World) error {
	if state.PlayerRoots == nil {
		return nil
	}
	players, present := w.CurrentPlayers()
	if !present || len(players) != len(*state.PlayerRoots) {
		return fmt.Errorf("current Player roots do not cover current registry")
	}
	for i, player := range players {
		row := (*state.PlayerRoots)[i]
		if row.ID != player.ID {
			return fmt.Errorf("current Player root differs from current identity")
		}
		if state.GroupBindings != nil {
			for _, bound := range state.GroupBindings.Players {
				if bound.ObjectIndex == row.ObjectIndex && bound.ID != row.ID {
					return fmt.Errorf("current Player root differs from command container identity")
				}
			}
		}
	}
	return nil
}
