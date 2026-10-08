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
		admitted := (c.Kind == KindCast || c.Kind == KindCastAt) && (w.manualCastRefusal(c) == "" || w.manualCellApproach(c))
		if c.Kind == KindUseStructure {
			i := indexOfEntity(w.entities, c.Entity)
			admitted = w.validStructureUse(i, StructureID(uint32(c.X)))
			if admitted {
				if _, pending := w.scrollIndex(c.Entity); pending {
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
		return w.manualCellApproach(c) && w.beginCellCastApproach(i, c)
	}
	if !w.cancelScroll(i) {
		return false
	}
	if e := &w.entities[i]; w.retainsOldStrike(i) {
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
	if c.Kind == KindCastAt {
		return w.beginBookSpellAt(i, c.X, c.Y, uint32(c.Spell))
	}
	return w.beginBookSpellOnce(i, EntityID(uint32(c.X)), uint32(c.Y))
}

// manualCellApproach reports a player Teleport refused for range alone: the
// order walks toward the cell and casts once in range.
func (w *World) manualCellApproach(c Command) bool {
	i := indexOfEntity(w.entities, c.Entity)
	return c.Kind == KindCastAt && c.Spell == teleportSpellID && i >= 0 && w.entities[i].Owner == SelfSlot && w.manualCastRefusal(c) == refusalCellOutOfRange &&
		w.bookSpellCellAdmissionRange(i, c.X, c.Y, uint32(c.Spell), false, false, false, true) == ""
}

// beginCellCastApproach replaces the caster's order with a walk to the cell and
// arms the cast; stepBookCasts starts it when the cell is in range.
func (w *World) beginCellCastApproach(i int, c Command) bool {
	if !w.cancelScroll(i) || w.retainsOldStrike(i) || w.stoneCursed(i) {
		return false
	}
	w.clearActorCast(i)
	w.clearOrder(i)
	w.invalidateActorMotion(c.Entity, "manual cast approach supersedes original movement")
	e := &w.entities[i]
	e.Retreat = RetreatContinuation{}
	e.clearAttack()
	e.clearTransit()
	e.clearStride()
	e.clearTurn()
	if !w.queueBookCast(bookCast{Caster: e.ID, Spell: c.Spell, X: c.X, Y: c.Y, AtCell: true, Retained: true, Phase: bookApproach}) {
		return false
	}
	e.AdmittedBookSpell = c.Spell
	w.attachMoveOrder(i, c.X, c.Y, false)
	w.syncSavedActorCommand(i)
	w.syncSavedPost(i)
	return true
}

// approachHeld reports that the order which armed an approach still stands: a
// creature's attack on its victim, or a player's walk to the cast cell.
func (w *World) approachHeld(i int, c bookCast) bool {
	e := w.entities[i]
	if e.Owner == SelfSlot {
		return c.AtCell && e.HasTarget && e.TargetX == c.X && e.TargetY == c.Y
	}
	return e.HasAttackTarget
}

const teleportSpellID = 26
