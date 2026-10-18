package sim

// Manual book commands retain logical progress (AI-356). Other action
// replacements keep DIV-1015; ordered unit-cast resume keeps DIV-1544.
func (w *World) manualCastRefusal(c Command) string {
	i := indexOfEntity(w.entities, c.Entity)
	if i < 0 {
		return "caster absent"
	}
	if w.entities[i].OffMap {
		return "caster off map"
	}
	switch c.Kind {
	case KindCast:
		return w.bookSpellAdmission(i, EntityID(uint32(c.X)), uint32(c.Y), false, false, false)
	case KindCastAt:
		return w.bookSpellCellAdmission(i, c.X, c.Y, uint32(c.Spell), false, false, false)
	}
	return "not a manual cast"
}

// Do not release an old book or scroll on the same tick that a valid manual
// replacement arrives. Commands still execute in their ordinary slice order.
func (w *World) manualActionInterruptions(cmds []Command) map[EntityID]bool {
	var interrupted map[EntityID]bool
	for _, c := range cmds {
		admitted := (c.Kind == KindCast || c.Kind == KindCastAt) && !w.paidWindUp(c.Entity) && (w.manualCastRefusal(c) == "" && !w.manualCastContinues(c) || w.manualCastApproach(c))
		if c.Kind == KindUseStructure {
			i := indexOfEntity(w.entities, c.Entity)
			admitted = w.validStructureUse(i, StructureID(uint32(c.X))) && !w.paidWindUp(c.Entity)
			if admitted {
				if i >= 0 && w.orderState(&w.entities[i]).scrollPending() {
					// Refund admission can fail in source containers. Probe an
					// isolated candidate so a refused click keeps the old release.
					n := w.sourceMutationCopy(i)
					admitted = n.cancelScroll(i)
				}
			}
		}
		if admitted {
			if interrupted == nil {
				interrupted = make(map[EntityID]bool)
			}
			interrupted[c.Entity] = true
		}
	}
	return interrupted
}

func (w *World) beginManualCast(i int, c Command) bool {
	if w.manualCastRefusal(c) != "" {
		if !w.manualCastApproach(c) {
			return false
		}
		if w.paidWindUp(c.Entity) {
			return w.castAfterWindUp(i, c)
		}
		if w.retainsOldStrike(i) {
			return w.castAfterStrike(i, c)
		}
		return w.beginCastApproach(i, c)
	}
	if w.manualCastContinues(c) {
		if w.entities[i].PendingOrder.IsCastRow() {
			w.entities[i].PendingOrder = PendingOrder{}
		}
		return true
	}
	if w.paidWindUp(c.Entity) {
		return w.castAfterWindUp(i, c)
	}
	if !w.cancelScroll(i) {
		return false
	}
	// A step between two cells completes: the cast waits for its end like one
	// behind a strike, and the figure is never moved to the cell centre.
	if w.retainsOldStrike(i) || w.entities[i].Transit != 0 {
		return w.castAfterStrike(i, c)
	}
	// At progress zero a unit cast retains the player's separate attack endpoint
	// for resume. Cell casts and stance-selected victims keep acquire-in-place.
	resumes := c.Kind == KindCast && holdsOrderedVictim(w.entities[i])
	w.clearActorCast(i)
	w.clearOrder(i)
	w.invalidateActorMotion(c.Entity, "manual cast supersedes original movement")
	e := &w.entities[i]
	e.PendingOrder = PendingOrder{}
	e.Retreat = RetreatContinuation{}
	if resumes {
		e.AttackPhase, e.AttackCountdown = AttackReady, 0
	} else {
		e.clearAttack()
	}
	e.clearTransit()
	e.clearStride()
	e.clearTurn()
	e.clearGroupSpeed()
	if !resumes {
		w.commandGroup([]int{i}, orderNone, cell{})
		w.acquireInPlace(i)
		w.syncSavedActorCommand(i)
		w.syncSavedPost(i)
	}
	return w.castOrder(i, c)
}

// castAfterStrike keeps a manual cast as the caster's next order while a
// strike or a step between two cells ends; the pending row runs the click again
// then (takePendingOrders), in reach or as an approach.
func (w *World) castAfterStrike(i int, c Command) bool {
	e := &w.entities[i]
	resumes := c.Kind == KindCast && holdsOrderedVictim(*e)
	w.clearActorCast(i)
	w.clearOrder(i)
	e.clearPendingAttack()
	p := PendingOrder{Kind: PendingActorCast, Target: EntityID(uint32(c.X)), Spell: uint16(c.Y)}
	if c.Kind == KindCastAt {
		p = PendingOrder{Kind: PendingCellCast, Spell: c.Spell, X: c.X, Y: c.Y}
	}
	e.PendingOrder = p
	e.PendingOrder.RowAdmitted = w.pendingLogicalProgress(i) == 0
	if e.PendingOrder.RowAdmitted {
		e.AdmittedBookSpell = p.Spell
	}
	if !resumes {
		w.commandGroup([]int{i}, orderNone, cell{})
		w.acquireInPlace(i)
		w.syncSavedActorCommand(i)
		w.syncSavedPost(i)
	}
	return true
}

// castOrder starts the cast an order names once no step and no strike stands
// before it: the turn, the wind-up, the release and the recovery follow. A
// player's cast and the automatic cast both start here.
func (w *World) castOrder(i int, c Command) bool {
	if c.Kind == KindCastAt {
		return w.beginBookSpellAt(i, c.X, c.Y, uint32(c.Spell))
	}
	return w.beginBookSpellOnce(i, EntityID(uint32(c.X)), uint32(c.Y))
}

// paidWindUp reads the order state of the caster by id; an absent caster
// has no wind-up.
func (w *World) paidWindUp(caster EntityID) bool {
	i := indexOfEntity(w.entities, caster)
	return i >= 0 && w.orderStateAt(i).paidWindUp()
}

// castAfterWindUp keeps a manual cast as the caster's next order while a paid
// wind-up runs (AI-356, AI-354). The pending row runs after the release
// (takePendingOrders).
func (w *World) castAfterWindUp(i int, c Command) bool {
	p := PendingOrder{Kind: PendingActorCast, Target: EntityID(uint32(c.X)), Spell: uint16(c.Y)}
	if c.Kind == KindCastAt {
		p = PendingOrder{Kind: PendingCellCast, Spell: c.Spell, X: c.X, Y: c.Y}
	}
	w.entities[i].PendingOrder = p
	return true
}

// manualCastContinues reports a click that repeats the cast still winding up:
// the same spell at the same unit or cell. It is no new order, so the wind-up
// runs on without a restart and a cast queued behind it is dropped.
func (w *World) manualCastContinues(c Command) bool {
	i := indexOfEntity(w.entities, c.Entity)
	if i < 0 {
		return false
	}
	s := w.orderStateAt(i)
	if !s.charging() {
		return false
	}
	b := w.bookCasts[s.bookAt]
	if c.Kind == KindCastAt {
		return b.AtCell && b.Spell == c.Spell && b.X == c.X && b.Y == c.Y
	}
	return c.Kind == KindCast && !b.AtCell && b.Target == EntityID(uint32(c.X)) && uint32(b.Spell) == uint32(c.Y)
}

// manualCastApproach reports a player's book cast refused for reach alone, or
// at a unit the caster cannot see but a unit of its player can. The original's
// setter parks it, its executor walks the caster into reach and its admission
// tests no sight (MAGIC-340, MAGIC-239, MAGIC-348); here the order walks and
// casts once the caster sees the target in reach.
func (w *World) manualCastApproach(c Command) bool {
	i := indexOfEntity(w.entities, c.Entity)
	if i < 0 || !w.humanParticipantUnit(w.entities[i]) {
		return false
	}
	refusal := w.manualCastRefusal(c)
	if c.Kind == KindCastAt {
		return isRangeRefusal(refusal) &&
			w.bookSpellCellAdmissionRange(i, c.X, c.Y, uint32(c.Spell), false, false, false, true) == ""
	}
	victim := EntityID(uint32(c.X))
	if !isRangeRefusal(refusal) && (refusal != refusalTargetNotVisible || !w.ownerSees(w.entities[i].Owner, victim)) {
		return false
	}
	return w.bookSpellAdmissionRange(i, victim, uint32(c.Y), false, false, false, true) == ""
}

// ownerSees reports whether a living unit of owner sees the entity id.
func (w *World) ownerSees(owner uint32, id EntityID) bool {
	vi := indexOfEntity(w.entities, id)
	for j := range w.entities {
		if vi >= 0 && w.entities[j].Owner == owner && w.entities[j].Alive() && w.actorSeesEntity(j, vi) {
			return true
		}
	}
	return false
}

// beginCastApproach replaces the caster's order with a walk into reach and
// arms the cast; stepBookCasts starts it when the target is in reach.
func (w *World) beginCastApproach(i int, c Command) bool {
	if !w.cancelScroll(i) || w.retainsOldStrike(i) || w.stoneCursed(i) {
		return false
	}
	w.clearActorCast(i)
	w.clearOrder(i)
	w.invalidateActorMotion(c.Entity, "manual cast approach supersedes original movement")
	e := &w.entities[i]
	e.Retreat = RetreatContinuation{}
	e.clearAttack()
	e.clearTurn()
	rec := bookCast{Caster: e.ID, Spell: c.Spell, X: c.X, Y: c.Y, AtCell: true, Retained: true}
	if c.Kind == KindCast {
		t := w.entities[indexOfEntity(w.entities, EntityID(uint32(c.X)))]
		rec = bookCast{Caster: e.ID, Target: t.ID, Spell: uint16(c.Y), X: t.X, Y: t.Y, Retained: true}
	}
	if !w.armCastApproach(i, &rec, rec.Target, false) || !w.queueBookCast(rec) {
		return false
	}
	w.entities[i].AdmittedBookSpell = rec.Spell
	w.syncSavedActorCommand(i)
	w.syncSavedPost(i)
	return true
}

// approachHeld reports that the order which armed an approach still stands: a
// pursuing caster's attack on its victim, or a walking caster's walk.
func (w *World) approachHeld(i int, c bookCast) bool {
	e := w.entities[i]
	if !w.castApproachPursues(e) {
		return !e.HasAttackTarget && e.HasTarget
	}
	if !e.HasAttackTarget {
		return c.AtCell && e.HasTarget && e.TargetX == c.X && e.TargetY == c.Y
	}
	return e.AttackTargetKind == AttackTargetUnit
}

// approachWalksOn reports a refusal an armed approach walks on through: reach,
// and for a walking caster the target's sight. The original's executor gate
// tests facing and reach and its admission no sight (MAGIC-340, MAGIC-348).
func (w *World) approachWalksOn(i int, refusal string) bool {
	return isRangeRefusal(refusal) || refusal == refusalTargetNotVisible && !w.castApproachPursues(w.entities[i])
}

// standCaster ends a walking caster's walk and leaves it standing in place,
// as the cast install stands the member (MAGIC-340).
func (w *World) standCaster(i int) {
	w.clearOrder(i)
	w.commandGroup([]int{i}, orderNone, cell{})
	w.acquireInPlace(i)
	w.syncSavedActorCommand(i)
	w.syncSavedPost(i)
}

const teleportSpellID = 26
