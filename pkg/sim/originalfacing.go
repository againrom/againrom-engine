package sim

import "fmt"

// OriginalActorFacing names the saved mover's current direction. Incoming
// desired direction, route and turn progress remain outside this import.
type OriginalActorFacing struct {
	ID     EntityID
	Facing uint8
}

// ImportOriginalActorFacings runs before the newly built mission is published.
// The idle native desired direction follows current facing so a later new
// order starts its turn from the restored direction, without resuming an
// invented incoming turn. Both bytes already belong to the native hash/save.
func (w *World) ImportOriginalActorFacings(batch []OriginalActorFacing) error {
	if w == nil {
		return fmt.Errorf("original facing: no world")
	}
	indices := make([]int, len(batch))
	seen := make(map[EntityID]bool, len(batch))
	for i, saved := range batch {
		if seen[saved.ID] {
			return fmt.Errorf("original facing: repeated entity %d", saved.ID)
		}
		seen[saved.ID] = true
		index := indexOfEntity(w.entities, saved.ID)
		if index < 0 {
			return fmt.Errorf("original facing: missing entity %d", saved.ID)
		}
		e := w.entities[index]
		if (!e.Alive() && !originalDyingEntity(e)) || e.TurnRemaining != 0 {
			return fmt.Errorf("original facing: entity %d is not an on-map living non-turning actor", saved.ID)
		}
		indices[i] = index
	}
	for i, saved := range batch {
		e := &w.entities[indices[i]]
		e.Facing, e.DesiredFacing = saved.Facing, saved.Facing
	}
	return nil
}
