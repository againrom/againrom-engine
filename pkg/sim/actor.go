package sim

// THE ACTOR LAYER: what one actor does when its group is not deciding for
// it.
//
// engage.go's whole file is written as though every living, owned entity
// belongs to a group that is being decided for — guard, stand ground,
// swarm, move, swarm 2. This file is the "instead" — one pass, one
// dispatch keyed on a per-entity byte, and one arm.
//
// D-1: THE DISPATCH IS THE SEAM, and 1141 is the order of this family
// that came through it. The guard arm and the guard post — 0099's own
// D-3, the divergence that left a patroller unable to fight — added a
// case below and a file of their own (guardarm.go) and touched neither
// the pass, nor the field, nor the byte form, nor the group-order clear.
// The seam's claim stays checkable by deletion: delete any one case
// below and this file still compiles, and every entity in that state is
// left in every field, on every tick.

// actorPass runs the actor layer's whole tick: every LIVING entity, in
// ASCENDING id, dispatched on its own ActorState.
//
// The walk is w.entities IN SLICE ORDER and not a sort of its own: the
// slice is kept sorted by id and the whole step loop already relies on
// that (world.go's own doc on World.entities), so re-deriving the order
// here would be a second place for it to be right.
//
// THE SWITCH HAS NO DEFAULT, and that absence is D-1's seam. Deleting every
// case below leaves a switch with none: still compiling, still reaching no
// arm for anything, on every tick. Pickup completion is the one defined
// state still without a case; its own pass owns it (pickupcompletion.go).
//
// Patrol and both escort arms share this pass with acquire-in-place (1087).
// Group order zero delegates to these actor states rather than deciding twice.
func (w *World) actorPass() {
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		if !activity.actorActive(&w.entities[i]) {
			continue
		}
		if w.savedGroups != nil && !w.nativePatrol(i) {
			continue
		}
		if _, using := w.structureUseIndex(w.entities[i].ID); using {
			continue
		}
		if !w.entities[i].Alive() {
			continue
		}
		// Command17 retains the old actor state, but its Group now owns
		// evaluation. A retained patrol/escort must not decide a second time.
		if order, _, ok := w.groupState(w.entities[i].Owner, effectiveGroup(&w.entities[i])); ok && order == orderRoam {
			continue
		}
		switch w.entities[i].ActorState {
		case actorStateGuard:
			// THE ONE CASE WITH A GATE OF ITS OWN (1141). Guard is the constructor's
			// default for every entity, not a state a command writes, so it is the
			// one arm whose population is not already fixed by its setter.
			// actorLayerDecides is the law's own entry condition for the whole
			// per-actor machine (AI-ORDER-010, AI-POST-095): group order 0 and
			// nothing else.
			if w.actorLayerDecides(i) {
				w.armGuard(i)
			}
		case actorStateAcquire:
			w.armAcquire(i)
		case actorStatePatrol:
			w.armPatrol(i)
			w.syncNativePatrol(i)
		case actorStateDefend:
			w.armDefend(i)
		case actorStateFollow:
			w.armFollow(i)
		case actorStateRetreat:
			w.armRetreat(i)
		}
	}
}

// legX and legY are the ring cell PatrolLeg currently names: the head
// pair under patrolLegHead, the tail pair under patrolLegTail.
//
// THE LEG IS AN INDEX AND NOT A CELL (D-2; world.go's own doc on
// PatrolLeg), and this is the one place that reads it as one. Every site
// below that wants "the actor's current waypoint" calls one of these two
// rather than testing the byte itself, so the selector and the two
// coordinate pairs it chooses between can never be read out of step with
// one another.
func (e *Entity) legX() int32 {
	if e.PatrolLeg == patrolLegHead {
		return e.PatrolHeadX
	}
	return e.PatrolTailX
}

func (e *Entity) legY() int32 {
	if e.PatrolLeg == patrolLegHead {
		return e.PatrolHeadY
	}
	return e.PatrolTailY
}

// armPatrol is the patrol arm, for the one actor at index i.
//
// STEP 1 — ITS ORDER IS CLEARED FIRST: target, stall count and stored
// route, through the one call that already keeps those three from coming
// apart (clearOrder, step.go).
//
// STEP 2 — THE ARRIVAL TEST IS POSITION EQUALITY, NOT arrived(). arrived()
// (engage.go: !HasTarget && Transit == 0) is the GROUP layer's own test
// for a member that has stopped moving, and it cannot answer this
// question: step 1, on this SAME call, has just cleared HasTarget on
// every patroller this pass reaches, so arrived() would read true for
// every one of them whether or not it stands on its waypoint. The cell is
// what a patroller standing on its waypoint has in common across a
// crossing — the law's own test, and the one this reproduces.
//
// THE ADVANCE IS PatrolLeg ^= 1, NOT A SEARCH (D-2). The law stores the
// current waypoint's cell and searches the ring's two-node list for it on
// arrival, falling back to the head when the search misses. Over every
// ring THIS BUILD CAN CONSTRUCT the two answers agree exactly: the
// command that builds a ring (script.go's cmdGroupPatrol) always makes
// the tail the current leg, and the tail is always one of the two cells
// the flip toggles between — so there is no ring this package can produce
// whose leg the flip could lose track of. The one input the two readings
// would part on needs a routine no live caller of this build reaches
// (D-2's own words).
//
// STEP 3 — NO CLAMP HERE (D-5). Both ring cells were clamped once, where
// the command built them, so the destination this writes is in bounds by
// construction and needs no second clamp — the one issueGroupDestination
// performs is for a formation offset this arm does not compute.
//
// AND THE STEPS ARE NOW FOUR, NOT THREE (1141). `AI-POST-042` gives the same
// routine's other half — "patrol is guard with a moving post" — and
// `AI-PATROL-018` the latch that moves it. This arm is now those four in
// that order, and 0099's D-3 is closed.
//
// THE GATE IS "NO VICTIM AND NO DESTINATION". The law tests `ord+0x08` for 0
// or `0xb` (`L00594`…`L00595`), the idle order and the idle turn: the
// two values guard leaves behind when it neither engaged (`ord+0x08` 5,
// `AI-PURSUE-040`) nor sent the actor home (`ord+0x08` 1). This package
// spells that byte as a victim and a destination, so the two values it must
// exclude are exactly a victim and a destination — the same substitution
// acquireStanding already documents, read here instead of written.
//
// THE RE-ANCHOR IS DERIVED, NOT STORED (D-6, 1141). The law latches
// `ord+0x04` on every advance and consumes it at the next entry to move
// the post to the cell the actor stands on THEN (`AI-PATROL-018`,
// `L00423` and `L00424`…`L00425`) — a one-tick delay that is the
// whole mechanism: consumed at entry the post equals the actor's current
// cell, so a patroller reads as home, its five-cell block travels with
// it and the leash never walks it back, while the tick guard interrupts
// leaves the post standing where the patrol broke off and leashes the
// pursuit to it (`AI-BREAK-041`).
//
// What the latch records is precisely "the previous entry reached the
// tail", and the tail's own last write is a destination equal to the
// current leg. So patrolInterrupted reads that write back instead of
// storing a second copy of it, on D-2's own ground — the leg is an index
// and not a cell there for the same reason. Where the two part is named
// in DIV-985 and it is one case: a pursuit dropped between two entries.
//
// AI-PATROL-013
func (w *World) armPatrol(i int) {
	e := &w.entities[i]
	// The turn decision is TAKEN FIRST AND APPLIED LAST. It is the same
	// question this arm always asked — is the destination about to be
	// written the one already held — but clearOrder below, and guard
	// after it, both destroy the answer, so the three fields it reads are
	// captured here and consumed at the advance. Nothing else moved: the
	// cell compared against is still the leg BEFORE the flip.
	held, heldX, heldY := e.HasTarget, e.TargetX, e.TargetY
	legX, legY := e.legX(), e.legY()
	// AND THE LATCH IS READ HERE, BEFORE clearOrder, for the same reason (pass
	// 1 review, F-1). clearOrder zeroes HasTarget, so a latch read after it can
	// only ever see the victim half of the question and never the walk home —
	// which made the post re-anchor on the entry after every break-off and
	// turned `AI-BREAK-041`'s post-relative leash into an actor-relative one
	// that no pursuit ends.
	interrupted := w.patrolInterrupted(i)

	w.clearOrder(i) // step 1

	// step 2: consume the derived latch, then guard.
	if !interrupted {
		e.PostX, e.PostY = e.X, e.Y
	}
	w.armGuard(i)

	e = &w.entities[i]
	if e.HasAttackTarget || e.HasTarget {
		return // guard left an order that is neither idle nor the idle turn
	}

	if e.X == e.legX() && e.Y == e.legY() {
		e.PatrolLeg ^= 1 // step 3
	}
	if !held || heldX != legX || heldY != legY {
		w.clearTurnUnlessCasting(i)
	}
	e.TargetX, e.TargetY = e.legX(), e.legY()
	e.HasTarget = true // step 4
}

// patrolInterrupted reports whether the PREVIOUS entry of the patrol arm
// was one whose tail did not run — the tick guard engaged, or sent the
// actor home — which is the one fact the law's `ord+0x04` latch records.
//
// It is read BEFORE step 1's clearOrder, so the two fields are still the
// ones the previous tick left behind. A victim is guard's engagement
// (`ord+0x08 = 5`). A destination that is not the ring cell this actor is
// walking to is guard's walk home, whose destination is the post
// (`AI-GUARD-012`, `L00334`); a destination that IS the ring cell is
// the tail's own last write and means the tail ran.
//
// AN ARRIVED WALK HOME IS NOT A PARTING CASE even though its destination
// has been consumed: an actor that reached its post stands on it, so
// re-anchoring writes the cell already there.
func (w *World) patrolInterrupted(i int) bool {
	return w.entities[i].patrolInterrupted()
}

func (e Entity) patrolInterrupted() bool {
	if e.HasAttackTarget {
		return true
	}
	return e.HasTarget && (e.TargetX != e.legX() || e.TargetY != e.legY())
}
