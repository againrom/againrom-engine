package sim

// WithdrawalDecision is one successful withdrawal arm observed at its exact
// phase-6 boundary. Before is the ordinary group, actor or player decision the
// tail received; After is the replacement the tail handed to movement. The
// observation is returned to its caller and never enters canonical world state.
type WithdrawalDecision struct {
	Before Entity
	After  Entity
}

type withdrawalObs struct {
	decisions []WithdrawalDecision
}

func (o *withdrawalObs) record(before, after Entity) {
	if o == nil {
		return
	}
	o.decisions = append(o.decisions, WithdrawalDecision{Before: before, After: after})
}

// withdrawalPass is the post-dispatch tail of the full AI tick. Every living,
// owned, on-map actor reaches the two gates after its group and actor decisions
// have run. Stone Curse keeps its existing promise to freeze the exact movement
// state, so a cursed actor is not re-ordered while the effect holds.
//
// The gates are absolute signed-health comparisons, Wimpy first and Withdraw
// second. A non-empty Wimpy collection consumes the tail even when every entry
// is a corpse: withdrawFrom then runs the ordinary acquisition fallback and the
// fixed-radius Withdraw arm is not tried.
func (w *World) withdrawalPass() {
	w.withdrawalPassObserved(nil)
}

func (w *World) withdrawalPassObserved(obs *withdrawalObs) {
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		if !activity.actorActive(w.entities[i]) {
			continue
		}
		if w.savedGroups != nil && !w.nativePatrol(i) {
			continue
		}
		w.withdrawalActor(i, obs)
		w.syncNativePatrol(i)
	}
}

func (w *World) withdrawalActor(i int, obs *withdrawalObs) {
	if _, using := w.structureUseIndex(w.entities[i].ID); using {
		return
	}
	e := w.entities[i]
	// HP > 0 is both the living-actor test used by the picker and what
	// keeps the zero constructor thresholds inert on a no-health-system
	// entity. Every placed actor with a health system is positive here.
	if e.HP <= 0 || e.Owner == 0 || e.OffMap || w.stoneCursed(i) {
		return
	}
	if e.ActorState == actorStateRetreat && w.retreatProgressBusy(i) {
		return
	}
	// ROM1 reads HP as a signed WORD here (`AI-WITHDRAW-026`); this
	// build compares canonical Entity.HP as int32. The shipped bounds and
	// the unrepresentable high-health edge are recorded by DIV-352.
	if e.HP <= e.Wimpy {
		if hostile := w.withdrawalHostiles(i, int64(e.ScanRange)); len(hostile) != 0 {
			before := w.entities[i]
			if w.withdrawFrom(i, hostile) {
				w.answerRefusedFlee(i)
			}
			obs.record(before, w.entities[i])
			return
		}
	}
	if e.HP <= e.Withdraw {
		if hostile := w.withdrawalHostiles(i, 2); len(hostile) != 0 {
			before := w.entities[i]
			if w.withdrawFromAny(i, hostile) {
				w.answerRefusedFlee(i)
			}
			obs.record(before, w.entities[i])
		}
	}
}

// answerRefusedFlee ends a withdrawal's move when no route serves its flee
// cell: the far search comes back with nothing, or settles on the mover's own
// cell, or the cell is the own cell of a mover between cells. The original raises the mover's
// route-failure flag there (MOVE-072, AI-335), and the order machine clears the
// pending order and reacquires a victim within reach unless the actor patrols
// (AI-ROUTE-045, AI-327). An actor holding a loaded cycle keeps it and its
// victim, since the pending move only executes once the cycle ends; one between
// cycles takes the reacquisition, whose pursuit order persists across cycles
// (AI-PURSUE-040). Both automatic arms and explicit Retreat (AI-RETREAT-273)
// call it; Retreat stays the actor's state. DIV-1555, DIV-1565.
func (w *World) answerRefusedFlee(i int) {
	e := &w.entities[i]
	if e.ActorState == actorStatePatrol || !w.fleeRefused(i) {
		return
	}
	w.clearOrder(i)
	if e.AttackPhase == AttackReady {
		w.reacquireWithinReach(i)
	}
}

// fleeRefused reports whether the far search that serves actor i's destination
// comes back with no route to walk, by the same search the movement pass runs.
func (w *World) fleeRefused(i int) bool {
	e := w.entities[i]
	return w.fleeRefusedAt(i, e.TargetX, e.TargetY)
}

// fleeRefusedAt is fleeRefused for a destination the actor has not been given
// yet. A search that settles on the mover's own cell, or whose goal is the
// cell of a mover between cells, comes back empty and is refused; a request
// for the cell of a centred mover is not refused.
func (w *World) fleeRefusedAt(i int, tx, ty int32) bool {
	e := w.entities[i]
	if e.X == tx && e.Y == ty {
		// A request equal to the cell of a centred mover returns before any
		// search and writes no failure flag (MOVE-081, MOVE-083); the move then
		// ends by the arrival rule. A mover between cells has no centred
		// shortcut: its search is seeded on the requested cell and ends with
		// no node (MOVE-080, MOVE-082), which is a refusal. DIV-1639.
		return e.Transit > 0
	}
	s := tickRouteScratch(w)
	defer releaseRouteScratch(s)
	route, ok := w.searchRoute(s, i, terrainRelation, noWindow, w.farBudgetFor(i), settleOrdered, tx, ty)
	return !ok || len(route) == 0
}

// withdrawalHostiles collects the square block centred on actor i, in entity
// order, and applies map presence, directional diplomacy and invisibility. It
// does not read sight, reach or route occupancy: the decoded collection is
// spatial and the route search belongs to the ordinary move that follows.
// Corpses are retained here; withdrawFrom owns the later positive-HP gate.
func (w *World) withdrawalHostiles(i int, radius int64) []int {
	self := w.entities[i]
	var out []int
	for ci := range w.entities {
		if ci == i {
			continue
		}
		candidate := w.entities[ci]
		if candidate.OffMap || !candidate.OrdinaryTargetable() || !w.hostileTo(self, candidate) || w.invisibleToActor(i, ci) {
			continue
		}
		if cellOf(self).chebyshevTo(cellOf(candidate)) <= radius {
			out = append(out, ci)
		}
	}
	return out
}

// withdrawFrom replaces actor i's prior decision with an ordinary move three
// cells away from the mean position of the entire filtered list, provided at
// least one entry has positive HP (AI-RETREAT-273/274, amended AI-WITHDRAW-028).
// A corpse-only list uses ordinary acquisition instead. The
// arithmetic is signed 8.8 integer arithmetic: zero axes become +1, the larger
// axis receives the full three-cell displacement, the minor axis is truncated
// toward zero, and the resulting cell is clamped to the playable rectangle.
//
// Positions are the movers' fine positions (fleeCell).
//
// It reports whether it wrote the flee move; acquisition writes none.
func (w *World) withdrawFrom(i int, hostile []int) bool {
	for _, ci := range hostile {
		if w.entities[ci].HP > 0 {
			return w.withdrawFromAny(i, hostile)
		}
	}
	w.acquireStanding(i)
	return false
}

// fleeCell is the cell the decoded arithmetic sends actor i to from the whole
// filtered list, or false for an empty list. It reads and writes no state.
// Every position is the mover's fine position on the 8.8 grid: the cell centre
// for a mover standing on a cell, and the paid part of the accepted stride for
// one between cells (AI-WITHDRAW-028, AI-RETREAT-273).
func (w *World) fleeCell(i int, hostile []int) (tx, ty int32, ok bool) {
	var sx, sy int64
	for _, ci := range hostile {
		fx, fy := w.moverFinePoint(w.entities[ci])
		sx += fx
		sy += fy
	}
	count := int64(len(hostile))
	if count == 0 {
		return 0, 0, false
	}

	mx, my := sx/count, sy/count
	px, py := w.moverFinePoint(w.entities[i])
	dx, dy := px-mx, py-my
	if dx == 0 {
		dx = 1
	}
	if dy == 0 {
		dy = 1
	}
	const stride = int64(3 * subCell)
	if abs64(dx) >= abs64(dy) {
		px += sign64(dx) * stride
		py += stride * dy / abs64(dx)
	} else {
		py += sign64(dy) * stride
		px += stride * dx / abs64(dy)
	}
	return clampPlayable(px>>8, int64(w.bounds.Width)), clampPlayable(py>>8, int64(w.bounds.Height)), true
}

// moverFinePoint is e's centre on the 8.8 grid. A mover of side n stores its
// footprint's top-left cell and a sub-cell offset, and the centre the range,
// edge-gap and bearing readers take is that corner plus (n-1)*128 per axis
// (MOVE-087). The corner is the retained current motion of a loaded actor, the
// accepted stride's paid steps for an actor between cells, and the cell centre
// (0x80 on both axes) for one standing on a cell.
func (w *World) moverFinePoint(e Entity) (int64, int64) {
	x, y := w.savedProjectileTargetPoint(e)
	half := (int64(footprintSide(e.TokenSize)) - 1) * 128
	return int64(x) + half, int64(y) + half
}

// The fixed-radius automatic arm has no positive-HP gate. Both helpers sum
// the whole list. DIV-575 bounds the custom dense-list divisor and cell model.
// It reports whether it wrote the flee move.
func (w *World) withdrawFromAny(i int, hostile []int) bool {
	tx, ty, ok := w.fleeCell(i, hostile)
	if !ok {
		w.acquireStanding(i)
		return false
	}
	w.cancelTurnForTargetChange(i, tx, ty)
	w.clearOrder(i)
	e := &w.entities[i]
	// The helpers write only an ordinary pending move; they touch neither the
	// attack target nor the cycle's progress (AI-WITHDRAW-028, AI-RETREAT-274).
	e.clearAttackBetweenCycles()
	e.clearGroupSpeed()
	e.TargetX = tx
	e.TargetY = ty
	e.HasTarget = true
	return true
}
