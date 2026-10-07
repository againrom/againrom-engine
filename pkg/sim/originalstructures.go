package sim

import "fmt"

// OriginalStructureHealth updates only the two existing persisted words. The
// importer must resolve the authored map identity and validate retained shape.
type OriginalStructureHealth struct {
	ID                StructureID
	Health, MaxHealth uint16
}

// ImportOriginalStructureHealth commits a completely validated batch before a
// newly loaded mission is published. Every word value is retained, including
// zero and signed-negative current HP; importing is not a damage/death action.
// Occupancy, structure identity, footprint and ordering remain untouched.
func (w *World) ImportOriginalStructureHealth(health []OriginalStructureHealth) error {
	if w == nil {
		return fmt.Errorf("original structure health: no world")
	}
	indices := make([]int, len(health))
	seen := make(map[StructureID]bool, len(health))
	for i, saved := range health {
		if seen[saved.ID] {
			return fmt.Errorf("original structure health: repeated structure %d", saved.ID)
		}
		seen[saved.ID] = true
		indices[i] = indexOfStructure(w.structures, saved.ID)
		if indices[i] < 0 {
			return fmt.Errorf("original structure health: missing structure %d", saved.ID)
		}
	}
	for i, saved := range health {
		s := &w.structures[indices[i]]
		s.Field42, s.MaxHealth = saved.Health, saved.MaxHealth
	}
	return nil
}
