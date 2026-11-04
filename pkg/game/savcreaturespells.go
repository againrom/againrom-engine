package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// projectCurrentCreatureSpells writes all six current slot words, including
// zeros, into the order block (SAV-1066). The engine document is the first
// game's form; a profile's form carries the slots into its own window.
func projectCurrentCreatureSpells(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World) error {
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		byID[e.ID] = e
	}
	form := sav.FirstGame
	block, width := form.CreatureSpellBlock()
	for _, a := range state.Actors {
		if a.Retired || a.ObjectIndex == 0 || int(a.ObjectIndex) > len(doc.Objects) {
			continue
		}
		e, ok := byID[a.EntityID]
		if !ok {
			continue
		}
		order, err := savedActorRaw(&doc.Objects[a.ObjectIndex-1], block, width)
		if err != nil {
			return err
		}
		var slots [sav.CreatureSpellSlots]sav.SavedCreatureSpell
		for i, s := range e.CreatureSpells {
			slots[i] = sav.SavedCreatureSpell{ID: s.ID, Threshold: s.Threshold}
		}
		if err := form.PutCreatureSpells(order, slots); err != nil {
			return err
		}
	}
	return nil
}

func creatureHoldsSlots(e sim.Entity) bool {
	for _, s := range e.CreatureSpells {
		if s.ID != 0 || s.Threshold != 0 {
			return true
		}
	}
	return false
}
