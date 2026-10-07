package sim

// RepairLegacyUnitCapacity fills the absent native Unit constructor capacity.
// Retained source/load state and nonzero values remain authoritative. The
// caller limits this repair to native saves without a retained SAV document.
func RepairLegacyUnitCapacity(w *World, capacity int32) int {
	if w == nil || w.savedObjects != nil || capacity <= 0 {
		return 0
	}
	repaired := 0
	for i := range w.entities {
		e := &w.entities[i]
		if e.Humanoid || e.Capacity != 0 || e.SourceBinding != (SourceBinding{}) ||
			e.ActorLoad != (ActorLoad{}) || e.HumanMovement != (HumanMovement{}) {
			continue
		}
		e.Capacity = capacity
		repaired++
	}
	return repaired
}
