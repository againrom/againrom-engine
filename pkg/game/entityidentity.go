package game

import "againrom/pkg/sim"

func ReserveResidueEntityIDs(w *sim.World, residue SnapshotResidue) {
	var ids []sim.EntityID
	for _, id := range residue.DepartedCharacters {
		ids = append(ids, sim.EntityID(id))
	}
	for _, id := range residue.Commanded {
		ids = append(ids, sim.EntityID(id))
	}
	// Only the maximum matters; map enumeration cannot affect the result.
	for id := range residue.Swing {
		ids = append(ids, sim.EntityID(id))
	}
	for id := range residue.Phase {
		ids = append(ids, sim.EntityID(id))
	}
	w.ReserveEntityIDs(ids)
}
