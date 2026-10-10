package sim

func (w *World) clearEscort(i int) {
	e := &w.entities[i]
	wasEscort := escortState(e.ActorState) || e.HasEscortTarget || e.EscortOrder != escortOrderNone || e.EscortTurnPending
	w.entities[i].clearEscort()
	if !wasEscort {
		return
	}
	if o := w.savedOrder(w.entities[i].ID); o != nil {
		o.State = uint32(w.entities[i].ActorState)
		o.authorNative()
		o.Raw[0x54], o.Raw[0x70] = 0, 0
	}
}

func (w *World) armDefend(i int) {
	if w.entities[i].OffMap || w.actorCastBusy(i) || w.stoneCursed(i) {
		return
	}
	ti, stop, dist, ok := w.escortSubject(i)
	if !ok {
		return
	}
	if dist > stop {
		w.escortClose(i, ti)
		return
	}
	if w.escortHeal(i, ti) {
		return
	}
	w.coverEngage(i, ti)
	if w.entities[i].HasAttackTarget {
		return
	}
	if dist < escortCrowd {
		w.escortStepAway(i, ti, stop)
	}
}

func (w *World) armFollow(i int) {
	if w.entities[i].OffMap || w.actorCastBusy(i) || w.stoneCursed(i) {
		return
	}
	ti, stop, dist, ok := w.escortSubject(i)
	if !ok {
		return
	}
	if dist > stop {
		w.escortClose(i, ti)
		return
	}
	if dist >= escortCrowd {
		w.acquireStanding(i)
		return
	}
	w.escortStepAway(i, ti, stop)
}

const escortCrowd = 2

const (
	escortOrderNone uint8 = iota
	escortOrderClose
	escortOrderIdle
)

// AI-FOLLOWHEAL-118, HERO-MP-006
func (w *World) escortHeal(i, ti int) bool {
	e, target := w.entities[i], w.entities[ti]
	if !e.Book.WirePresent(e.KnownSpells) {
		return false
	}
	floor := e.MaxMana / 4
	if percent, present := w.AutoHealing(e.Owner); present {
		floor = int32(int16(uint16(int64(e.MaxMana) * int64(percent) / 100)))
	}
	if e.ActorLoad.Source.Class != 0 {
		floor = int32(int16(e.ActorLoad.Source.ManaFloor))
	}
	if !(floor == 0 && target.HP < target.MaxHP || int64(floor) < int64(e.MaxMana)+3 && target.HP < target.MaxHP>>1) {
		return false
	}
	rule, ok := w.bookSpell(e, 6)
	mana := e.Mana
	if rule.bookInstance {
		mana = int32(int16(mana))
	}
	if !ok || !rule.Restorative || rule.ManaCost > mana || w.bookSpellRefusal(i, target.ID, 6, false, false) != "" {
		return false
	}
	w.clearOrder(i)
	w.entities[i].clearAttackBetweenCycles()
	return w.beginBookSpellOnce(i, target.ID, 6)
}

// AI-ORDER-039, AI-TURN-104, AI-RETAL-056, AI-395, AI-417
func (w *World) stepEscortOrder(s *routeScratch, i int) bool {
	e := &w.entities[i]
	if !escortState(e.ActorState) || !e.HasEscortTarget {
		return false
	}
	switch e.EscortOrder {
	case escortOrderClose:
		ti, stop, _, ok := w.escortSubject(i)
		if !ok {
			return false
		}
		target := w.entities[ti]
		if int64(strikeDistance(*e, target)) <= stop {
			if !w.restAt(s, i) {
				return true
			}
			e.EscortOrder = escortOrderClose
			w.turnTowardActor(i, target.X-e.X, target.Y-e.Y)
			return true
		}
		if !e.HasTarget || e.TargetX != target.X || e.TargetY != target.Y {
			w.routes[i] = nil
			w.walkTo(s, i, target.X, target.Y)
		}
	case escortOrderIdle:
		pending := e.EscortTurnPending
		if o := w.savedOrder(e.ID); o != nil {
			pending = pending || o.Raw[0x54] != 0
		}
		if !pending && w.rng.raw() >= 0xcd {
			return true
		}
		e.EscortTurnPending = false
		if o := w.savedOrder(e.ID); o != nil {
			o.Raw[0x54] = 0
		}
		desired := e.Facing + 0x21 + uint8(w.rng.aiRange(190))
		oldRemaining, oldDesired := e.TurnRemaining, e.DesiredFacing
		already := w.turnAlreadyStepped(e.ID)
		if e.requestFacing(desired, already) {
			if !already {
				w.markTurnStepped(e.ID)
			}
			if oldRemaining == 0 || oldDesired != desired {
				e.startAction(w.tick, int64(e.TurnRemaining))
			}
		}
		return true
	}
	return false
}

const coverRadius = 5

func (w *World) escortSubject(i int) (ti int, stop, dist int64, ok bool) {
	e := w.entities[i]
	ti = indexOfEntity(w.entities, e.EscortTarget)
	if !e.HasEscortTarget || ti < 0 || ti == i {
		return -1, 0, 0, false
	}
	stop = int64(e.EscortRange)
	if stop == 0 {
		stop = int64(e.ScanRange)
	}
	return ti, stop, cellOf(e).chebyshevTo(cellOf(w.entities[ti])), true
}

func (w *World) escortClose(i, ti int) {
	w.cancelTurnForTargetChange(i, w.entities[ti].X, w.entities[ti].Y)
	w.clearOrder(i)
	e := &w.entities[i]
	e.clearAttackBetweenCycles()
	e.clearGroupSpeed()
	e.TargetX, e.TargetY = w.entities[ti].X, w.entities[ti].Y
	e.HasTarget = true
	e.EscortOrder = escortOrderClose
}

func (w *World) coverEngage(i, ti int) {
	subject := w.entities[ti]
	var live, dead []int
	for ci := range w.entities {
		c := w.entities[ci]
		if ci == i || ci == ti || c.OffMap {
			continue
		}
		if cellOf(c).chebyshevTo(cellOf(subject)) > coverRadius {
			continue
		}
		if !w.hostileTo(subject, c) {
			continue
		}
		if !c.OrdinaryTargetable() {
			continue
		}
		if c.HP < 1 {
			dead = append(dead, ci)
			continue
		}
		live = append(live, ci)
	}
	if len(live) == 0 {
		live = dead
	}
	if len(live) == 0 {
		w.acquireStanding(i)
		return
	}
	at := -1
	best := int64(-1)
	for _, ci := range live {
		if w.targetVetoed(i, ci) {
			continue
		}
		d := cellOf(w.entities[i]).chebyshevTo(cellOf(w.entities[ci]))
		switch {
		case at < 0:
		case coverPreferred(w.entities[at]) && !coverPreferred(w.entities[ci]):
			continue
		case coverPreferred(w.entities[ci]) && !coverPreferred(w.entities[at]):
		case d >= best:
			continue
		}
		at, best = ci, d
	}
	if at < 0 {
		w.acquireStanding(i)
		return
	}
	w.orderAttack(i, w.entities[at].ID)
}

func coverPreferred(c Entity) bool { return lawDomain(c.Domain) == 3 }

func (w *World) acquireStanding(i int) {
	if p := w.entities[i].PendingOrder.Kind; p != PendingNone && p != PendingRelease {
		return
	}
	best, at := scoreSeed, -1
	for _, ci := range w.actorCandidates(i) {
		if ci == i {
			continue
		}
		if cost := w.candidateCost(i, ci, orderStandGround); cost < best {
			best, at = cost, ci
		}
	}
	if at < 0 {
		w.clearOrder(i)
		w.entities[i].releaseBetweenCycles(PendingRelease)
		if escortState(w.entities[i].ActorState) && !humanParticipantUnit(w.entities[i]) {
			w.entities[i].EscortOrder = escortOrderIdle
		}
		return
	}
	w.orderAcquire(i, w.entities[at].ID)
}

func (w *World) actorCandidates(i int) []int {
	decider := w.entities[i]
	stamp := w.groupSight(aiSight, []int{i})
	var live, dead []int
	for ci := range w.entities {
		c := w.entities[ci]
		if c.OffMap || !w.sightShows(stamp, cellOf(c)) || !w.hostileTo(decider, c) {
			continue
		}
		if !c.OrdinaryTargetable() {
			continue
		}
		if c.HP < 1 {
			dead = append(dead, ci)
			continue
		}
		live = append(live, ci)
	}
	if len(live) == 0 {
		return dead
	}
	return live
}

func (w *World) escortStepAway(i, ti int, stop int64) {
	e, t := w.entities[i], w.entities[ti]
	dx := (int64(e.X) - int64(t.X)) * subCell
	dy := (int64(e.Y) - int64(t.Y)) * subCell
	if dx == 0 {
		dx = 1
	}
	if dy == 0 {
		dy = 1
	}
	adx, ady := abs64(dx), abs64(dy)
	major := stop * subCell
	var px, py int64
	if adx >= ady {
		px = int64(t.X)*subCell + sign64(dx)*major
		py = int64(t.Y)*subCell + sign64(dy)*roundDiv(major*ady, adx)
	} else {
		py = int64(t.Y)*subCell + sign64(dy)*major
		px = int64(t.X)*subCell + sign64(dx)*roundDiv(major*adx, ady)
	}
	targetX := clampPlayable(px>>8, int64(w.bounds.Width))
	targetY := clampPlayable(py>>8, int64(w.bounds.Height))
	w.cancelTurnForTargetChange(i, targetX, targetY)
	w.clearOrder(i)
	self := &w.entities[i]
	self.clearAttackBetweenCycles()
	self.clearGroupSpeed()
	self.TargetX = targetX
	self.TargetY = targetY
	self.HasTarget = true
}

func clampPlayable(v, dim int64) int32 {
	if hi := dim - 9; v > hi {
		v = hi
	}
	if v < 8 {
		v = 8
	}
	return int32(v)
}

func roundDiv(num, den int64) int64 { return (2*num + den) / (2 * den) }

func sign64(v int64) int64 {
	if v < 0 {
		return -1
	}
	return 1
}
