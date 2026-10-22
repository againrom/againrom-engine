package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// projectCurrentWithdrawalThresholds writes every live actor's current
// Withdraw and Wimpy into its Order block (AI-WITHDRAW-027). The original
// restores that block raw and tests it every tick (AI-WITHDRAW-026), so a
// zero here keeps a monster from ever fleeing there. Current state wins over
// the loaded bytes: the Wimpy command rewrites both (AI-CLASS-030).
func projectCurrentWithdrawalThresholds(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World) error {
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		byID[e.ID] = e
	}
	for _, a := range state.Actors {
		if a.Retired || a.ObjectIndex == 0 || int(a.ObjectIndex) > len(doc.Objects) {
			continue
		}
		e, ok := byID[a.EntityID]
		if !ok {
			continue
		}
		order, err := savedActorRaw(&doc.Objects[a.ObjectIndex-1], "U158", 148)
		if err != nil {
			return fmt.Errorf("actor %d withdrawal thresholds: %w", e.ID, err)
		}
		sav.PutOrderWithdrawalThresholds(order, e.Withdraw, e.Wimpy)
	}
	return nil
}

// matchCurrentWithdrawalThresholds makes the file's Order bytes the LOAD
// authority for both thresholds: the engine's SAVE writes the current values
// there, and an original resave keeps or rewrites them in place.
func matchCurrentWithdrawalThresholds(ms *Mission, a *currentActionData) {
	doc := ms.savedDocument.Document
	objects := map[sim.EntityID]uint16{}
	for _, b := range ms.savedDocument.Actors {
		if !b.Retired {
			objects[b.EntityID] = b.ObjectIndex
		}
	}
	for i := range a.Actions.Actors {
		row := &a.Actions.Actors[i]
		object := objects[row.Entity]
		if object == 0 || int(object) > len(doc.Objects) {
			continue
		}
		for _, raw := range doc.Objects[object-1].Raw {
			if raw.Name != "U158" {
				continue
			}
			if withdraw, wimpy, ok := sav.OrderWithdrawalThresholds(raw.Bytes); ok && len(raw.Bytes) == 148 {
				row.Withdraw, row.Wimpy = withdraw, wimpy
			}
		}
	}
}
