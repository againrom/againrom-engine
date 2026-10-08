package sim

import "fmt"

const (
	PendingNone uint8 = iota
	PendingRelease
	PendingPickup
	PendingPickupComplete
	PendingActorCast
	PendingCellCast
	PendingScroll
)

type PendingOrder struct {
	Kind        uint8
	RowAdmitted bool
	Target      EntityID
	Spell       uint16
	X, Y        int32
}

func (e *Entity) strikeAhead() bool {
	return e.HasAttackTarget && (e.AttackPhase == AttackCharging || e.AttackPhase == AttackCasting)
}

func (w *World) retainsOldStrike(i int) bool {
	e := &w.entities[i]
	if !e.HasAttackTarget {
		return false
	}
	known := e.Retreat.Known || w.savedOrder(e.ID) != nil
	// A release marker or pickup completion held beside a recovering cycle does
	// not retain it for a cast or scroll: those replace the cycle and the marker.
	held := e.PendingOrder.Kind == PendingRelease || e.PendingOrder.Kind == PendingPickupComplete
	return e.strikeAhead() || e.PendingOrder.Kind != PendingNone && !held || known && w.pendingLogicalProgress(i) != 0
}

// cycleLoaded reports a victim with a cycle in any phase but ready: the charge
// or release still ahead, the recovery, or the two boundary turns after it.
func (e *Entity) cycleLoaded() bool {
	return e.HasAttackTarget && e.AttackPhase != AttackReady
}

// holdsCycleFor reports whether a request of kind keeps the loaded cycle beside
// its pending record. The release marker and the pickup completion wait through
// recovery and both boundary turns, since the order machine consumes nonzero
// progress before it reads a pending order. A cast or scroll waits only while
// the application is ahead; in recovery it replaces the cycle (DIV-1015).
func (e *Entity) holdsCycleFor(kind uint8) bool {
	if kind == PendingRelease || kind == PendingPickupComplete {
		return e.cycleLoaded()
	}
	return e.strikeAhead()
}

func (e *Entity) releaseBetweenCycles(kind uint8) {
	e.clearPendingAttack()
	if e.holdsCycleFor(kind) || e.Retreat.Known {
		e.PendingOrder = PendingOrder{Kind: kind}
		return
	}
	e.clearAttack()
}

func (w *World) pendingLogicalProgress(i int) uint8 {
	e := w.entities[i]
	if e.Retreat.Known {
		return e.Retreat.Progress
	}
	if o := w.savedOrder(e.ID); o != nil {
		if o.Raw[9] == 1 && e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
			return 0
		}
		return o.Raw[9]
	}
	if e.Transit != 0 || w.motionActive(e.ID) {
		return 3
	}
	if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		return 1
	}
	return 0
}

func (w *World) ActorOrderProgress(id EntityID) uint8 {
	if i := indexOfEntity(w.entities, id); i >= 0 {
		p := w.entities[i].PendingOrder
		if p.RowAdmitted && (p.Kind == PendingActorCast || p.Kind == PendingCellCast || p.Kind == PendingScroll) {
			return 2
		}
		if p.RowAdmitted && p.Kind == PendingPickup && (w.entities[i].Transit != 0 || w.motionActive(id)) {
			return 3
		}
		return w.pendingLogicalProgress(i)
	}
	return 0
}

func (w *World) pendingOrderReady(i int) bool {
	return w.pendingLogicalProgress(i) == 0
}

func (w *World) takePendingOrders() {
	for i := range w.entities {
		e := &w.entities[i]
		p := e.PendingOrder
		if p.Kind == PendingPickup {
			found := false
			for _, sack := range w.sacks {
				found = found || sack.X == p.X && sack.Y == p.Y
			}
			if !found {
				e.PendingOrder = PendingOrder{}
			} else if !p.RowAdmitted && w.pendingOrderReady(i) {
				e.PendingOrder.RowAdmitted = true
			}
		}
		if p.Kind == PendingNone || p.Kind == PendingPickup || p.Kind == PendingScroll || !e.Alive() || e.OffMap || !w.pendingOrderReady(i) {
			continue
		}
		if (p.Kind == PendingActorCast || p.Kind == PendingCellCast) && !p.RowAdmitted {
			e.PendingOrder.RowAdmitted = true
			e.AdmittedBookSpell = p.Spell
			continue
		}
		if p.RowAdmitted && (e.Transit != 0 || w.motionActive(e.ID) || w.stoneCursed(i)) {
			continue
		}
		if p.Kind != PendingActorCast || !holdsOrderedVictim(*e) {
			e.clearActiveAttack()
		} else {
			e.AttackPhase, e.AttackCountdown = AttackReady, 0
		}
		e.PendingOrder = PendingOrder{}
		e.Retreat = RetreatContinuation{}
		switch p.Kind {
		case PendingActorCast:
			w.beginManualCast(i, Cast(e.ID, p.Target, SpellID(p.Spell)))
		case PendingCellCast:
			w.beginManualCast(i, CastAt(e.ID, SpellID(p.Spell), CellPoint{X: p.X, Y: p.Y}))
		}
	}
}

func (w *World) settleIdleOrderProgress(i int) {
	e := w.entities[i]
	if e.Retreat.Known || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		return
	}
	if o := w.savedOrder(e.ID); o != nil && o.Raw[9] == 1 {
		o.Raw[9] = 0
	}
}

func (w *World) observePendingOrderCompletion(i int) {
	e := w.entities[i]
	if e.PendingOrder.Kind == PendingNone || e.Retreat.Known || e.AttackPhase != AttackBoundaryOne {
		return
	}
	if o := w.savedOrder(e.ID); o != nil && o.Raw[9] == 1 {
		o.Raw[9] = 0
	}
}

func (w *World) pendingOrderFault() error {
	for _, e := range w.entities {
		p := e.PendingOrder
		if p.Kind == PendingNone {
			if p != (PendingOrder{}) {
				return fmt.Errorf("sim: absent pending order carries operands")
			}
			continue
		}
		if p.Kind > PendingScroll || !e.Alive() && !e.Dying() || e.HasPendingAttackTarget {
			return fmt.Errorf("sim: pending order conflicts with actor state")
		}
		progress := w.pendingLogicalProgress(indexOfEntity(w.entities, e.ID))
		pickupMoving := p.Kind == PendingPickup && progress == 3 && (e.Transit != 0 || w.motionActive(e.ID))
		if p.RowAdmitted && (p.Kind == PendingRelease || progress != 0 && !pickupMoving) {
			return fmt.Errorf("sim: admitted pending row conflicts with retained progress")
		}
		if p.Kind != PendingScroll && (e.CastWait != 0 || w.scrollCastPending(indexOfEntity(w.entities, e.ID))) {
			return fmt.Errorf("sim: pending order conflicts with admitted cast")
		}
		if _, ok := w.bookCastIndex(e.ID); ok {
			return fmt.Errorf("sim: pending order conflicts with book cast")
		}
		switch p.Kind {
		case PendingRelease, PendingPickupComplete, PendingScroll:
			if p.Target != 0 || p.Spell != 0 || p.X != 0 || p.Y != 0 {
				return fmt.Errorf("sim: pending order carries unused operands")
			}
		case PendingPickup:
			if p.Target != 0 || p.Spell != 0 {
				return fmt.Errorf("sim: pickup order carries cast operands")
			}
			fallthrough
		case PendingCellCast:
			if p.Target != 0 || p.X < 0 || p.Y < 0 || p.X >= w.bounds.Width || p.Y >= w.bounds.Height {
				return fmt.Errorf("sim: pending order cell is invalid")
			}
		case PendingActorCast:
			if p.X != 0 || p.Y != 0 {
				return fmt.Errorf("sim: actor cast carries cell operands")
			}
		}
		if (p.Kind == PendingActorCast || p.Kind == PendingCellCast) && p.Spell == 0 {
			return fmt.Errorf("sim: pending cast has no spell")
		}
		if p.Kind == PendingPickupComplete && e.ActorState != actorStatePickupComplete {
			return fmt.Errorf("sim: pickup completion has no completion state")
		}
		if p.Kind == PendingPickupComplete {
			if err := pickupCompletionGroupsFault([]Entity{e}, w.groups, w.savedGroups); err != nil {
				return err
			}
		}
		if p.Kind == PendingScroll {
			at, ok := w.scrollIndex(e.ID)
			if !ok || w.scrollCasts[at].Started {
				return fmt.Errorf("sim: pending scroll has no waiting pointer")
			}
		}
	}
	return nil
}
