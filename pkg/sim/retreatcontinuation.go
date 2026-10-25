package sim

// RetreatContinuation separates the pending policy order from current progress.
type RetreatContinuation struct {
	Known, Pending, Failure, Complete bool
	X, Y                              int32
	Progress, Counter                 uint8
}

func (w *World) executorProgress(i int) uint8 {
	e := w.entities[i]
	switch {
	case w.stoneCursed(i):
		return 4
	case e.Transit != 0 || w.motionActive(e.ID):
		return 3
	case w.actorCastBusy(i):
		return 2
	case e.AttackPhase != AttackReady:
		return 1
	}
	return 0
}

func (w *World) ensureRetreatContinuation(i int) {
	e := &w.entities[i]
	if !e.Retreat.Known {
		e.Retreat = RetreatContinuation{Known: true, Progress: w.executorProgress(i)}
	}
}

func (w *World) retreatExecutorActive(i int) bool {
	return w.executorActive(i, w.entities[i].Owner != 0)
}

// executorActive is the executor's entry gate for an actor at zero progress
// (AI-350, AI-364): the local player's actor runs; an actor of a registry
// Group runs while that Group's rebuilt activity count is nonzero or the
// override is set, as the Group pass is admitted; any other actor runs as
// ungrouped says.
func (w *World) executorActive(i int, ungrouped bool) bool {
	e := w.entities[i]
	if e.Owner == SelfSlot {
		return true
	}
	if g := w.savedGroupFor(e.ID); g != nil {
		return g.AI[0x45] != 0 || w.safeMode
	}
	return ungrouped
}

func (w *World) dispatchRetreatPending(i int) {
	e := &w.entities[i]
	if !e.Retreat.Pending {
		return
	}
	x, y := e.Retreat.X, e.Retreat.Y
	e.Retreat.Pending = false
	e.clearAttack()
	w.clearOrder(i)
	e.TargetX, e.TargetY, e.HasTarget = x, y, true
	w.answerRefusedFlee(i)
	if e.HasAttackTarget {
		e.Retreat.restartStrike()
	}
}

// AI-365. A supplied active failure is consumed independently of pending dispatch.
func (w *World) retreatFailureTail(i int) {
	e := &w.entities[i]
	if !e.Retreat.Failure {
		return
	}
	e.Retreat.Failure = false
	e.Retreat.Pending = false
	w.answerRefusedRoute(i, refusedRetreatFailure)
}

func (w *World) stepRetreatExecutor(i int) bool {
	e := &w.entities[i]
	if e.PendingOrder.Kind != PendingNone && e.Retreat.Known {
		if !e.Alive() || e.OffMap {
			return false
		}
		cleared := w.consumeRetreatProgress(i)
		return cleared || e.Retreat.Progress != 0 && e.AttackPhase == AttackReady
	}
	if e.ActorState != actorStateRetreat {
		e.Retreat = RetreatContinuation{}
		return false
	}
	if !e.Alive() || e.OffMap {
		return false
	}
	w.ensureRetreatContinuation(i)
	entered := e.Retreat.Progress
	if entered == 0 && !w.retreatExecutorActive(i) {
		return false
	}
	if m := w.motionFor(e.ID); m != nil && m.Current && m.Mover[0x98] != 0 {
		e.Retreat.Failure = true
		m.Mover[0x98] = 0
	}
	cleared := false
	if entered != 0 {
		if w.consumeRetreatProgress(i) {
			cleared = true
			if (e.Retreat.Pending || !e.AcquirePursuit) && e.AttackPhase == AttackReady {
				e.clearActiveAttack()
			}
		}
	} else {
		w.dispatchRetreatPending(i)
		if !e.Retreat.Pending && !e.AcquirePursuit {
			e.clearActiveAttack()
		}
		if !e.Retreat.Pending && e.HasAttackTarget && e.AcquirePursuit {
			e.Retreat.restartStrike()
		}
	}
	w.retreatFailureTail(i)
	return cleared && e.Retreat.Progress == 0
}

func (w *World) consumeRetreatProgress(i int) bool {
	e := &w.entities[i]
	if e.Retreat.Progress == 0 {
		return false
	}
	e.Retreat.Counter++
	complete := false
	switch e.Retreat.Progress {
	case 1:
		complete = e.Retreat.Complete && e.Retreat.Counter > 2
	case 2:
		s := w.orderState(e)
		busy := s.wait || s.scroll == scrollStarted
		book := s.booked()
		if e.PendingOrder.Kind == PendingNone {
			busy = w.actorCastBusy(i)
		}
		complete = !busy && !book && e.Retreat.Counter > 2
	case 3:
		complete = e.Transit == 0 && !w.motionActive(e.ID)
	case 4:
		complete = !w.stoneCursed(i)
	}
	if complete {
		e.Retreat.Progress, e.Retreat.Complete = 0, false
	}
	return complete
}

func (w *World) stepRetreatExecutors() []bool {
	holds := make([]bool, len(w.entities))
	for i := range w.entities {
		holds[i] = w.stepRetreatExecutor(i)
		e := &w.entities[i]
		if e.PendingOrder.Kind == PendingNone && e.ActorState == actorStateRetreat && e.Retreat.Known && e.Retreat.Progress == 0 && e.HasAttackTarget && (e.Retreat.Pending || !e.AcquirePursuit) {
			holds[i] = true
		}
	}
	return holds
}

func (w *World) observeRetreatCompletion(i int) {
	e := &w.entities[i]
	if (e.ActorState == actorStateRetreat || e.PendingOrder.Kind != PendingNone) && e.Retreat.Known && e.Retreat.Progress == 1 &&
		(e.AttackPhase == AttackBoundaryOne || e.AttackPhase == AttackReady && e.PendingOrder.Kind == PendingNone) {
		e.Retreat.Complete = true
	}
}
