package sim

// THE PER-ACTOR GUARD ARM: what one actor decides on its own tick when
// nothing above it is deciding for it, and what a patroller runs before it
// takes another step.
//
// THE SIGHT TEST BETWEEN THE ANCHOR AND THE BLOCK IS NOT WRITTEN HERE.
// `AI-BREAK-041` reads `L00267`…`L00331` — `Chebyshev(me, post) >=
// actor+0xa5` writes `ord+0x08 = 1`, `ord+0x0a = post` — and grades it
// "behaviourally dead" in the law itself: every branch after it overwrites
// both fields it writes, and its one surviving corner is an engage that
// refuses while its acquire fallback writes nothing. That corner does not
// exist below. orderAttack refusing one candidate leaves the scan running,
// and a scan that engages nothing falls into the walk-home and the standing
// acquisition exactly as an empty block does, so every path through this arm
// writes the actor's order from scratch. The test is omitted rather than
// transcribed dead, which is the shape savedGuard (savedgroupsai.go) already
// carries for the same routine on the original-runtime side.
//
// WHAT THE ARM DOES NOT DECIDE is who reaches it. Guard is this build's
// constructor default for EVERY entity, so the dispatch's own case carries
// that gate; see actorLayerDecides.

// postRadius is the occupancy block's Chebyshev radius: `mover+0x08`, written
// once in the image by the mover constructor and read at every block site
// (`AI-BREAK-041`, `L00587`). It is the same number escort.go's coverRadius
// names for the defender's cover block, and it is spelled as that constant so
// a corrected `mover+0x08` remains one edit rather than two that must agree.
const postRadius = coverRadius

// postEngage is the block scan itself, shared by this arm and by the
// original-runtime savedGuard: everything within postRadius of post that the
// actor at i is hostile to, walked in ascending entity id, engaged at the
// first candidate whose order is not refused. It reports whether the actor
// comes out of it holding a victim.
//
// THE PICK IS THE FIRST SURVIVOR, NOT THE CHEAPEST, and that is what makes
// this a different picker from acquireStanding rather than a second spelling
// of it. acquireStanding scores every candidate with candidateCost and takes
// the minimum, over the actor's SIGHT rather than over a block, and caps the
// pick at reach. The two populations and the two orderings are both
// different, so both are written.
//
// ASCENDING ENTITY ID IS OURS. Ascending id is the order every other
// candidate walk in this package uses, and it is the order savedGuard has
// already been taking.
//
// THE FILTER IS DIPLOMACY AND PRESENCE, WITH NO CORPSE PREFERENCE.
// actorCandidates' live-first parking is the ACQUISITION's rule and is not
// carried here.
//
// AI-FILTER-001
func (w *World) postEngage(i int, post cell) bool {
	for ci, candidate := range w.entities {
		if ci == i || candidate.OffMap || !candidate.OrdinaryTargetable() ||
			!w.hostileTo(&w.entities[i], &candidate) || post.chebyshevTo(cellOf(&candidate)) > postRadius {
			continue
		}
		w.orderAttack(i, candidate.ID)
		if w.entities[i].HasAttackTarget {
			return true
		}
	}
	return false
}

func (w *World) armGuard(i int) {
	if w.entities[i].OffMap || w.actorCastBusy(i) || w.stoneCursed(i) {
		return
	}
	e := &w.entities[i]
	if e.PostX == 0 && e.PostY == 0 {
		e.PostX, e.PostY = e.X, e.Y
	}
	if w.postEngage(i, cell{x: e.PostX, y: e.PostY}) {
		return
	}
	if e.X != e.PostX || e.Y != e.PostY {
		w.guardWalkHome(i)
		return
	}
	w.acquireStanding(i)
}

// guardWalkHome is the arm's empty-block tail (`AI-GUARD-012`, `L00334`;
// `AI-BREAK-041`): `ord+0x08 = 1`, `ord+0x0a = ord+0x00` — a destination,
// which is the post.
//
// IT DROPS THE VICTIM BETWEEN CYCLES, and that is the whole of "a pursuit can
// be broken off". `AI-BREAK-041` states the mechanism exactly: nothing inside
// the pursuit arms ends one, what ends one is that this arm "rewrites
// `ord+0x08` from scratch on its own tick" — and `ord+0x08 = 1` is written
// over the `ord+0x08 = 5` a pursuit lives in (`AI-PURSUE-040`). This package
// spells an order byte as a destination and a victim, so the byte's
// replacement is two clears here, on escortClose's own ground. The write
// touches no progress, so a loaded cycle finishes first and the walk waits
// behind it (`AI-GUARD-012`, `AI-ORDER-039`; DIV-1563).
//
// THE CONSEQUENCE A CONSUMER MUST NOT INVERT is the claim's: how far the
// actor has chased is not measured. The distance tested is post-to-target,
// so it re-engages at any pursuit length while the target is inside the
// block and breaks off at any pursuit length once the target leaves it.
func (w *World) guardWalkHome(i int) {
	x, y := w.entities[i].PostX, w.entities[i].PostY
	w.cancelTurnForTargetChange(i, x, y)
	w.clearOrder(i)
	e := &w.entities[i]
	e.clearAttackBetweenCycles()
	e.TargetX, e.TargetY, e.HasTarget = x, y, true
}

// actorLayerDecides reports whether the per-actor machine is what decides
// entity i at all: its group's stored order is 0.
//
// `AI-ORDER-010` gives the group order byte's six-arm table and reads arm 0
// as the one that "hands control to the members' own states"; `AI-POST-095`
// measures the entry (`callto:R0008` = 2 / 2 / 0) and states the negative
// this gate exists for — a group under order 1 never evaluates `actor+0x50`.
//
// THE GATE IS HERE BECAUSE GUARD IS NOT WRITTEN BY A COMMAND AT ALL — it
// is the constructor's default for every entity in the world. The other five
// actor states are each written beside a group order cleared to 0 (group.go,
// script.go, playerdefend.go, playerretreat.go), so their arms are entered
// on that order in the ordinary case; without this gate the guard arm would
// instead run for every guarding unit on the map, including the whole
// population the group layer's own guard and stand-ground arms already
// decide, and decide it twice.
//
// THAT IS THE ORDINARY CASE AND NOT AN INVARIANT (pass 1 review, F-6). A
// setter writes the order and the state together, but a LATER group command
// can raise the order without rewriting either: a script `Group Command:
// Guard` against a group an earlier node put on patrol sets `g.order = 1`
// (cmdGroupGuard) and leaves its members in patrol state, where armPatrol —
// and through it this arm — still runs, against `AI-ORDER-010`'s "a group
// under order 1 never evaluates `actor+0x50`". The ungated armPatrol predates
// 1141; what 1141 adds is that those members also scan and engage. Not
// reached on shipped data: over the eight maps that carry a patrol node, at
// 1500 ticks each, all fourteen entities that ran the patrol arm ran it under
// group order 0 (measured in this story's adversarial pass). Recorded as
// DIV-990.
//
// A PAIR NO RECORD NAMES ANSWERS FALSE, on decide's own rule: such a pair
// takes no group decision, and giving it an actor decision instead would be
// the same back-door derivation that rule removes.
func (w *World) actorLayerDecides(i int) bool {
	e := w.entities[i]
	order, _, ok := w.groupState(e.Owner, effectiveGroup(&e))
	return ok && order == orderNone
}
