package sim

// commandDefend implements player opcode 0x1b: a fresh order-none group,
// selected subject acquiring in place, every other member defending it at
// default range 3 (AI-CMD-033/054, AI-FOLLOWSET-116, AI-FOLLOWRANGE-115).
// An absent/off-map subject is a bounded no-op, before membership is changed.
//
// The setter stores the state, the pending order and the escort fields and no
// attack progress, so a member holding a loaded attack cycle keeps its victim
// until the cycle ends and the state arm decides on the next pass
// (AI-CMD-054, AI-FOLLOWSET-116, AI-CMD-033, AI-ORDER-039; DIV-1577).
func (w *World) commandDefend(members []int, subject EntityID) {
	ti := indexOfEntity(w.entities, subject)
	if ti < 0 || w.entities[ti].OffMap {
		return
	}
	for _, mi := range members {
		w.cancelScroll(mi)
	}
	w.commandGroup(members, orderNone, cell{})
	for _, mi := range members {
		w.clearOrder(mi)
		e := &w.entities[mi]
		w.retainCycleForState(mi)
		e.clearGroupSpeed()
		if mi == ti {
			w.acquireInPlace(mi)
		} else {
			e.ActorState = actorStateDefend
			e.EscortTarget, e.HasEscortTarget = subject, true
			e.EscortRange = escortRangeDefault
		}
		if o := w.syncSavedActorCommand(mi); o != nil {
			o.Raw[8] = 0
		}
	}
}

// armAcquire uses the standing, reach-limited picker without a guard-post
// leash (AI-STATE-011, AI-ACQUIRE-002). A cast remains the actor's owner.
func (w *World) armAcquire(i int) {
	if w.entities[i].OffMap || w.actorCastBusy(i) || w.stoneCursed(i) {
		return
	}
	w.acquireStanding(i)
}
