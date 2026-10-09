package sim

// retreatOrder resolves one player selection. AI-RETREAT-270 requires a
// resolvable first member; missing later members are skipped. The canonical
// model has no selectable intermediate death action (DIV-574).
func (w *World) retreatOrder(cmds []Command, first int, consumed []bool) {
	c := cmds[first]
	fi := indexOfEntity(w.entities, c.Entity)
	validFirst := fi >= 0 && c.Player != 0 && c.Player < relationSlots &&
		w.entities[fi].Owner == c.Player && w.entities[fi].Alive() && !w.entities[fi].OffMap
	var members []int
	count := 0
	for k := first; k < len(cmds); k++ {
		next := cmds[k]
		if next.Kind != c.Kind || next.Group != c.Group {
			continue
		}
		consumed[k] = true
		count++
		if !validFirst || count > 253 || next.Player != c.Player {
			continue
		}
		i := indexOfEntity(w.entities, next.Entity)
		if i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap || w.entities[i].Owner != c.Player {
			continue
		}
		if !containsIndex(members, i) {
			members = append(members, i)
		}
	}
	w.commandGroup(members, orderNone, cell{})
	for _, i := range members {
		// A reserved walk-to-cast is the superseded destination, not started
		// cast progress. Refund it through the existing consumable seam.
		// A started scroll keeps its wind-up and recovery before Retreat runs.
		if !w.scrollInFlight(w.entities[i].ID) {
			w.cancelScroll(i)
		}
		e := &w.entities[i]
		w.clearEscort(i)
		e.PendingOrder = PendingOrder{}
		e.ActorState = actorStateRetreat
		e.Retreat = RetreatContinuation{Known: true, Progress: w.retreatCurrentProgress(i)}
		e.clearGroupSpeed()
		// The already committed crossing and loaded attack/cast survive. Only
		// the old pending destination is cancelled (AI-RETREAT-271/272).
		w.clearOrder(i)
		if !w.actorCastBusy(i) {
			e.clearTurn()
		}
		if e.AttackPhase == AttackReady {
			e.clearAttack()
		}
		w.syncSavedActorCommand(i)
	}
}

func (w *World) retreatProgressBusy(i int) bool {
	e := w.entities[i]
	if e.Retreat.Known {
		return e.Retreat.Progress != 0
	}
	return e.Transit != 0 || e.AttackPhase != AttackReady || w.actorCastBusy(i) || w.stoneCursed(i)
}

// armRetreat is state 0x16's persistent full-tick decision. Arrival alone
// never clears it. Policy writes pending coordinates independently of progress.
// A flee cell no route serves takes the reacquisition of route failure
// (AI-RETREAT-273) and the state stays Retreat.
func (w *World) armRetreat(i int) {
	e := w.entities[i]
	if e.OffMap || e.Owner == 0 || e.PendingOrder.Kind != PendingNone {
		return
	}
	w.ensureRetreatContinuation(i)
	hostile := w.withdrawalHostiles(i, int64(e.ScanRange))
	alive := false
	for _, ci := range hostile {
		alive = alive || w.entities[ci].HP > 0
	}
	if !alive {
		if !w.retreatProgressBusy(i) {
			w.acquireStanding(i)
		}
		return
	}
	x, y, ok := w.fleeCell(i, hostile)
	if ok {
		self := &w.entities[i]
		self.Retreat.Pending, self.Retreat.X, self.Retreat.Y = true, x, y
		if self.Retreat.Progress == 0 && w.retreatExecutorActive(i) {
			w.dispatchRetreatPending(i)
		}
	}
}
