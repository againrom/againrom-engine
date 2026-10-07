package sim

import "fmt"

// ImportOriginalActionClocks binds the serialized actor+138 deadlines to an
// unpublished original-load candidate. SAV-UNITPROG-156 stores the dword;
// SAV-REGENORDER-531 consumes it against the independent session subtick.
// This uses the literal deadline rather than bootstrapping a new idle age.
// Original notification/callback chronology remains a separate native policy.
func (w *World) ImportOriginalActionClocks(ends map[EntityID]uint32) error {
	if w == nil {
		return fmt.Errorf("original action clocks require a world")
	}
	for id := range ends {
		at := indexOfEntity(w.entities, id)
		if at < 0 || w.entities[at].ActionClock.Known {
			return fmt.Errorf("original action clock %d has no fresh actor binding", id)
		}
	}
	for i := range w.entities {
		if end, present := ends[w.entities[i].ID]; present {
			w.entities[i].ActionClock = ActionClock{Known: true, End: end}
		}
	}
	return nil
}
