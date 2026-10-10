package sim

// reacquireWithinReach is the order machine's answer to a raised route-failure
// flag for an actor that does not patrol: the pending order is cleared and the
// victim is chosen from actors admitted by the running-nearest whole-cell gate,
// then diplomacy, health and circular-byte turn scoring (AI-360/361). The
// pick becomes the actor's attack order, which nothing in the pursuit arms ends
// (AI-PURSUE-040, AI-BREAK-041); the next dispatch of the actor's state arm is
// what rewrites it. With none in reach the order goes idle and no victim is
// written (AI-328).
//
// A human participant's mage of reach below two takes no pick: the clause both
// this routine and the engage routine carry writes the order idle after the
// target is chosen (AI-GUARD-007).
func (w *World) reacquireWithinReach(i int) {
	victim := w.reacquisitionVictim(i)
	e := &w.entities[i]
	if victim < 0 || suppressedAcquirer(*e) {
		w.clearOrder(i)
		e.clearAttack()
		return
	}
	w.orderAcquire(i, w.entities[victim].ID)
	if e.ActorState == actorStateRetreat && e.HasAttackTarget {
		e.Retreat.Progress, e.Retreat.Counter, e.Retreat.Complete = 1, 0, false
	}
}

// reacquisitionVictim is the index of the hostile actor the reacquisition
// names for actor i, or -1. Its candidates are the on-map actors other than i
// itself within i's reach, kept by diplomacy and visibility; the previous victim
// is not excluded. A body is a candidate only when no living actor is, the
// corpse fallback the selection shares with ordinary acquisition (AI-327,
// AI-FILTER-001).
func (w *World) reacquisitionVictim(i int) int {
	self := w.entities[i]
	reach := int64(self.Reach)
	nearest := reach
	var admitted []int
	for _, id := range w.actorTraversalIDs() {
		ci := indexOfEntity(w.entities, id)
		if ci < 0 {
			continue
		}
		c := w.entities[ci]
		if ci == i || c.OffMap {
			continue
		}
		d := cellOf(&self).chebyshevTo(cellOf(&c))
		if d > reach || d > nearest {
			continue
		}
		nearest = d
		admitted = append(admitted, ci)
	}
	var living, bodies []int
	for _, ci := range admitted {
		c := w.entities[ci]
		if !c.OrdinaryTargetable() || !w.hostileTo(&self, &c) || w.invisibleToActor(i, ci) || w.targetVetoed(i, ci) {
			continue
		}
		if c.HP > 0 {
			living = append(living, ci)
		} else {
			bodies = append(bodies, ci)
		}
	}
	if len(living) == 0 {
		living = bodies
	}
	winner, best := -1, int32(129)
	for _, ci := range living {
		candidate := w.entities[ci]
		direction := w.headingBetween(self, candidate)
		cost := facingArc(self.Facing, direction)
		if cost <= best {
			winner, best = ci, cost
		}
	}
	return winner
}

// suppressedAcquirer reports whether e is a human participant's spellcaster of
// reach below two, the unit the reacquisition and the engage routine refuse to
// send at a victim on their own (AI-GUARD-007).
func suppressedAcquirer(e Entity) bool {
	return e.Owner == SelfSlot && isMage(e) && e.Reach < 2
}
