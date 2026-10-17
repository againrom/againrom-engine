package sim

// reacquireWithinReach is the order machine's answer to a raised route-failure
// flag for an actor that does not patrol, the flee row of refusedAnswers. The
// pick becomes the attack order, which nothing in the pursuit arms ends
// (AI-PURSUE-040, AI-BREAK-041).
func (w *World) reacquireWithinReach(i int) {
	w.answerRefusedRoute(i, refusedFlee)
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
func (w *World) suppressedAcquirer(e Entity) bool {
	return e.Owner != 0 && w.humanParticipantUnit(e) && isMage(e) && e.Reach < 2
}

// engageRefused reports whether the engage routine refuses actor i outright:
// a first-game suppressed acquirer. The second game keeps its engage; no
// second-game claim reads this test (AI-GUARD-007).
func (w *World) engageRefused(i int) bool {
	return !w.SecondGame() && w.suppressedAcquirer(w.entities[i])
}
