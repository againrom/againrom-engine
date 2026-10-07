package game

import "againrom/pkg/sim"

// Project every current deadline, including zero and wrap. ANIM-CLOCK-001,
// SAV-UNITPROG-156 and SAV-REGENORDER-531 identify actor+138 and its signed age
// against the independent session subtick; the supplement preserves Known.
func projectActionClocks(state *SnapshotSAVDocument, world *sim.World) error {
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	for _, a := range state.Actors {
		if a.Retired {
			continue
		}
		e, ok := entities[a.EntityID]
		if !ok {
			return worldSaveUnsupportedf("actor %d current action deadline is unavailable", a.EntityID)
		}
		if err := savedActorSetValue(&state.Document.Objects[a.ObjectIndex-1], "U138", e.ActionClock.End); err != nil {
			return err
		}
	}
	return nil
}
