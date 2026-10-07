package game

import "againrom/pkg/sim"

func (mw *mapWorld) playerRetreat(entities []uint32) {
	if len(entities) == 0 {
		return
	}
	// One UI press is one group even when paused selections are disjoint.
	// Inferring this boundary from repeated IDs would let a refused earlier
	// press consume an independent later one through first-member rejection.
	mw.groupTag++
	for _, entity := range entities {
		id := sim.EntityID(entity)
		mw.cancelPickup(id)
		mw.pending = append(mw.pending, sim.GroupRetreat(id, sim.SelfSlot, mw.groupTag))
		mw.commanded[id] = true
	}
}
