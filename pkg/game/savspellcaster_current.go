package game

import (
	"fmt"

	"againrom/pkg/sim"
)

// Both endpoints are relocatable archive addresses, never original pointer
// claims. Caster zero records a caster that has already left the population.
type currentSpellCaster struct {
	Object uint16
	Caster uint16
}

func restoreCurrentSpellCasters(ms *Mission, rows []currentSpellCaster) error {
	if len(rows) == 0 {
		return nil
	}
	g := ms.World.SavedSpellGraph()
	state := ms.savedDocument
	if g == nil || state.WorldEffects == nil || len(g.Nodes) != len(state.WorldEffects.SpellNodes) {
		return fmt.Errorf("current spell caster has no graph")
	}
	actors := map[uint16]sim.EntityID{}
	for _, a := range state.Actors {
		if !a.Retired {
			actors[a.ObjectIndex] = a.EntityID
		}
	}
	for _, d := range ms.World.OriginalDeadActors() {
		actors[d.Source.ArchiveIndex] = d.ID
	}
	seen := map[uint16]bool{}
	for _, row := range rows {
		if seen[row.Object] || row.Object == 0 {
			return fmt.Errorf("repeated or null current spell node")
		}
		seen[row.Object] = true
		found := false
		for i, b := range state.WorldEffects.SpellNodes {
			if b.ObjectIndex != row.Object {
				continue
			}
			n := &g.Nodes[i]
			if n.Retired || n.Value.Class != "PointEffect" {
				return fmt.Errorf("current spell caster has an invalid node")
			}
			var ok bool
			n.Caster, ok = actors[row.Caster]
			n.HasCaster = row.Caster != 0
			if n.HasCaster && !ok {
				return fmt.Errorf("current spell caster has no actor")
			}
			found = true
		}
		if !found {
			return fmt.Errorf("current spell node is absent")
		}
	}
	return ms.World.ImportSavedSpellGraph(g)
}
