package sim

// reacquireWithinReach is the order machine's answer to a raised route-failure
// flag for an actor that does not patrol, the flee row of refusedAnswers. The
// pick becomes the attack order, which nothing in the pursuit arms ends
// (AI-PURSUE-040, AI-BREAK-041).
func (w *World) reacquireWithinReach(i int) {
	w.answerRefusedRoute(i, refusedFlee)
}

// reacquisitionVictim filters by range and running distance before relation,
// visibility, veto and living/body fallback. The shared chooser ranks by turn.
// AI-327, AI-FILTER-001, R2-ENGINE-1019.
func (w *World) reacquisitionVictim(i int) int {
	self := w.entities[i]
	reach := int64(self.Reach)
	if w.game.orders.repickEmptyRoute {
		reach = w.attackRange(i)
	}
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
		if c.Living() {
			living = append(living, ci)
		} else {
			bodies = append(bodies, ci)
		}
	}
	if len(living) == 0 {
		living = bodies
	}
	return w.chooseTarget(i, living, orderStandGround, targetTurn)
}

// suppressedAcquirer reports whether e is a human participant's spellcaster of
// reach below two, the unit the reacquisition and the engage routine refuse to
// send at a victim on their own (AI-GUARD-007).
func (w *World) suppressedAcquirer(e Entity) bool {
	return e.Owner != 0 && w.humanParticipantUnit(e) && isMage(e) && e.Reach < 2
}

// engageRefused applies the profile's engage or acquisition exception.
// AI-GUARD-007, R2-ENGINE-1106.
func (w *World) engageRefused(i int, acquisition bool) bool {
	if acquisition && w.game.orders.mageAcquisitionException {
		e := w.entities[i]
		return w.humanParticipantUnit(e) && isMage(e) && w.attackRange(i) < 2
	}
	return !w.game.keepsSuppressedAcquirer && w.suppressedAcquirer(w.entities[i])
}

// AI-673: row 6 rechecks suppression before a progress-zero gate or turn.
func (w *World) suppressAcquisition(i int) bool {
	e := &w.entities[i]
	if e.AttackPhase != AttackReady || !e.AcquirePursuit || !w.engageRefused(i, true) {
		return false
	}
	e.AcquirePursuit, e.PursuitIdle = false, true
	if o := w.savedOrder(e.ID); o != nil {
		o.Raw[8] = 0
	}
	return true
}
