package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// projectCurrentCreatureSpells writes a creature's class spell slots into its
// order block (SAV-1066): the three ids at +0x78 and the three scaled
// probabilities at +0x84. An actor holding no slot keeps its record's bytes.
func projectCurrentCreatureSpells(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World) error {
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		byID[e.ID] = e
	}
	for _, a := range state.Actors {
		if a.Retired || a.ObjectIndex == 0 || int(a.ObjectIndex) > len(doc.Objects) {
			continue
		}
		e, ok := byID[a.EntityID]
		if !ok || !creatureHoldsSlots(e) {
			continue
		}
		order, err := savedActorRaw(&doc.Objects[a.ObjectIndex-1], "U158", 148)
		if err != nil {
			return err
		}
		for i, s := range e.CreatureSpells {
			binary.LittleEndian.PutUint32(order[sav.CreatureSpellIDOffset+4*i:], s.ID)
			binary.LittleEndian.PutUint32(order[sav.CreatureSpellThresholdOffset+4*i:], s.Threshold)
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
