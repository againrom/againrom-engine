package sim

// THE ESCORT ARMS: what a defending or a following unit does on its own tick.
//
// 0166 landed the two states and the order they carry — the escort target and
// the range — and gave neither an arm, so an escort stood still. This file is
// the two arms and the three helpers they share.
//
// The two are not one behaviour parameterised (`AI-FOLLOW-112`). Their
// out-of-range halves are the same three writes; their within-range halves
// are a scan performed on the ESCORTED unit's behalf on one side and, on the
// other, the ordinary acquisition any unordered unit runs.
//
// WHAT EACH PATH WRITES, IN THE LAW'S OWN ORDER BYTE: closing writes order 4
// (close on the actor, stop distance `ord+0x70`), engaging writes order 5 (a
// pursuit), the step-away writes order 1 (walk to a cell), and an
// acquisition that scores nothing writes order `0xb` (the idle turn). This
// package has one destination pair and one victim rather than an order byte,
// so the four are spelled as a destination, an attack order, a destination
// and a cleared order (SC-3).

// armDefend is actor state 8 (`AI-DEFEND-111`).
//
// The order of its three parts is the arm's own and not a convenience. THE RANGE
// TEST COMES FIRST and returns: a defender outside its range never engages,
// whatever stands beside the unit it protects. Within range it engages on the
// protected unit's behalf, and only then — if that engagement left it with
// nothing to fight — does it test how crowded it is.
//
// THE BUSY TEST IS READ AFTER THE ENGAGEMENT AND NOT INSIDE IT. The law
// re-reads `ord+0x08` through a five-arm table whose whole content is "a
// pursuit or a cast means this defender is busy"; what wrote that order does
// not enter the test, so a defender that was already on a pursuit its cover
// scan did not change is busy on the same terms as one the scan has just
// committed.
func (w *World) armDefend(i int) {
	ti, stop, dist, ok := w.escortSubject(i)
	if !ok {
		return //
	}
	if dist > stop {
		w.escortClose(i, ti) //
		return
	}
	w.coverEngage(i, ti) //
	if w.entities[i].HasAttackTarget {
		return //
	}
	if dist < escortCrowd {
		w.escortStepAway(i, ti, stop) //
	}
}

// armFollow is actor state 0x11 (`AI-FOLLOW-112`).
//
// Its out-of-range half is armDefend's, store for store. Its within-range half
// is three instructions in the law and reads as two here: at or past the
// crowding distance the follower runs the ordinary acquisition and nothing else,
// and below it the follower steps away. A follower never builds a cover block,
// never scans on the followed unit's behalf and never heals it: it fights only
// what it would have fought standing alone.
//
// IT HAS NO BUSY TEST because it has no inner table — `EnumRefs refto:L00323`
// finds the table's one reader inside the defend helper alone — and it needs
// none: the acquisition either engages, in which case the arm is already over,
// or it does not, in which case there is nothing to be busy with.
func (w *World) armFollow(i int) {
	ti, stop, dist, ok := w.escortSubject(i)
	if !ok {
		return //
	}
	if dist > stop {
		w.escortClose(i, ti) //
		return
	}
	if dist >= escortCrowd {
		w.acquireStanding(i) //
		return
	}
	w.escortStepAway(i, ti, stop) //
}

// escortCrowd is the minimum separation both arms enforce: 2, hard-coded in the
// law at both sites (`AI-FOLLOWGAP-114`, compared against 2 at L01085 and
// L01075). It is not the escort range, it is not derived from
// one, and it is the same number under both arms.
const escortCrowd = 2

// coverRadius is how far from the ESCORTED unit a defender's cover block reaches:
// 5, the mover constructor's `mover+0x08` (`AI-BREAK-041`, `L00587`), giving
// the 11x11 block `AI-DEFEND-111` reads. The claim grades the value Medium — a
// register-form store is invisible to the sweep behind it — and it is one
// constant here so that a corrected radius is one edit.
const coverRadius = 5

// escortSubject resolves an escort's order into the three numbers both arms
// open with: the escorted unit's index, the stop distance, and the present
// distance between the two.
//
// ok is false for an escort whose target names no entity this world holds,
// and that case is AUTHORED rather than transcribed. `AI-FOLLOWDEATH-119`
// establishes that the law holds the escorted unit as a pointer,
// dereferences it with no null test and no dead test, and clears it nowhere;
// there is no behaviour to reproduce for an id that names nothing. Both arms
// return without deciding anything, which leaves the escort in every field.
//
// A DEAD BUT PRESENT ESCORTED UNIT IS NOT THIS CASE. While a body is still in the
// slice its cell is still readable, and both arms keep running against it — the
// escort closes on the corpse and a defender covers it — which is exactly what
// `AI-FOLLOWDEATH-119` predicts of the original and what only a running original
// could refute.
//
// THE STOP DISTANCE FALLS BACK TO THE ESCORT'S OWN SCAN RANGE when the stored
// range is 0, which is the arm's own selector (the zero test at L01061, `actor+0xa5`
// read at L00269). `AI-FOLLOWRANGE-115` establishes positively that the stored
// range is never 0 at the arm on shipped data, and this build's setter coerces a
// 0 to 3 before the field is ever written, so the fallback is unreachable from
// any map. It is written because it is one of the selector's two arms.
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

// escortClose is the out-of-range half of both arms: order 4, close on the
// escorted unit with the stop distance.
//
// IT DROPS THE VICTIM BETWEEN CYCLES. Order 4 is written into the same byte a
// pursuit lives in, so a defender that has chased a hostile out past its range
// is pulled off it and turned around — which is the whole of "when a defender
// breaks off". clearAttackBetweenCycles is called directly and releaseAttack
// is not: that function stays pinned to the group decision. The write touches
// no progress, so a loaded cycle finishes first and the walk waits behind it
// (`AI-DEFEND-111`, `AI-FOLLOW-112`, `AI-ORDER-039`; DIV-1563).
//
// THE DESTINATION IS THE ESCORTED UNIT'S OWN CELL and not a cell at the stop
// distance from it. Where the walk ends is decided by the next actor pass
// finding the escort within range, not by the destination it was given.
func (w *World) escortClose(i, ti int) {
	w.cancelTurnForTargetChange(i, w.entities[ti].X, w.entities[ti].Y)
	w.clearOrder(i)
	e := &w.entities[i]
	e.clearAttackBetweenCycles()
	e.clearGroupSpeed()
	e.TargetX, e.TargetY = w.entities[ti].X, w.entities[ti].Y
	e.HasTarget = true
}

// coverEngage is the defender's cover scan, and every term in it belongs to
// the ESCORTED unit rather than to the defender: the block is centred on
// that unit's cell, and the hostility filter runs with that unit as the
// decider. A defender engages what threatens the unit it protects, which
// need not be its own enemy at all; `AI-DEFEND-111` discriminates that
// polarity by the arguments at all three sites that use them.
//
// WHAT IS THE DEFENDER'S is the distance the pick is chosen by: the candidate
// nearest SELF, not nearest the protected unit.
//
// THE AIR PREFERENCE is the law's `[vt+0x20] == 3` arm, which selects a preferred
// candidate where the block holds one and falls back to any. `AI-DEFEND-111`
// grades what that virtual returns 3 for as Unknown; `AI-ACQUIRE-002` reads the
// same virtual as the flier term, and lawDomain already maps this package's air
// domain onto 3 for the preference matrix, so that is the reading taken here.
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
		w.acquireStanding(i) //
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
	w.orderAttack(i, w.entities[at].ID) //
}

// coverPreferred is the cover scan's preference: a candidate the law's
// `[vt+0x20]` answers 3 for. See coverEngage's own note on that reading.
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
		// A lost standing pick must drop the old pursuit as well.
		w.entities[i].releaseBetweenCycles(PendingRelease)
		return
	}
	w.orderAcquire(i, w.entities[at].ID)
}

// actorCandidates is what the ONE actor at index i can see and will fight,
// in ascending entity id.
//
// It is candidates' body with a one-member list: one actor's own sight stamp, its
// own diplomacy row as the decider, the same off-map refusal and the same corpse
// parking. It is written out beside that function rather than through a synthetic
// aiGroup because an aiGroup carries an owner and a group id that an actor-layer
// decision has nothing to do with.
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

// The 8.8 grid the step-away computes on is rate.go's subCell, which is the same
// 256 `AI-FOLLOWGAP-114`'s helper packs into. There is no second constant here.

// escortStepAway is the crowding step of both arms: order 1, walk to a cell
// the stop distance away from the escorted unit, on the line from that unit
// through the escort's own position. Like escortClose it writes a pending
// order only and leaves a loaded cycle standing (`AI-FOLLOWGAP-114`; DIV-1563).
//
// THE ARITHMETIC IS THE HELPER'S, IN 8.8 AND IN INTEGERS. The two positions
// are packed onto the sub-cell grid, the deltas are taken there, the
// dominant axis is moved the stop distance in whole cells, and the minor
// axis takes the proportional part of that move — which is why the grid is
// kept rather than the whole computation being done in cells.
//
// THE SUB-CELL TERM IS ABSENT AND THE ZERO-FORCING IS THEREFORE LIVE. The law's
// helper reads the escort's own sub-cell into its half of the delta while the
// caller packs the escorted unit's cell with none; this package has no sub-cell
// field, so both halves are packed the same way and the delta is cell-granular.
// The helper's own two branches — a zero delta forced to 1, stated there as
// being what keeps the answer off the escorted unit's own cell — are then
// reachable exactly where that rationale says they are: an escort sharing an axis
// with its subject, or standing on its cell.
//
// THE CLAMP IS THE PLAYABLE RECTANGLE, `[8, dim-9]` on both axes, recomputed from
// the map's own extent exactly as the helper recomputes it rather than read from
// a stored rectangle.
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
	self.clearAttackBetweenCycles() //
	self.clearGroupSpeed()
	self.TargetX = targetX
	self.TargetY = targetY
	self.HasTarget = true
}

// clampPlayable is the helper's own clamp: a cell forced into `[8, dim-9]`.
//
// A dimension below 18 leaves the two bounds crossed, and the low bound is
// applied last there, exactly as the law's two consecutive compares leave it. No
// map this build can load is that small.
func clampPlayable(v, dim int64) int32 {
	if hi := dim - 9; v > hi {
		v = hi
	}
	if v < 8 {
		v = 8
	}
	return int32(v)
}

// roundDiv is num/den rounded half away from zero, for non-negative num and
// positive den. It stands in for the float-to-int the law's minor axis runs
// through, which `AI-FOLLOWGAP-114` grades Medium and reads as a use.
func roundDiv(num, den int64) int64 { return (2*num + den) / (2 * den) }

// sign64 is the direction term the packing needs; the magnitude term is
// group.go's abs64. It answers 1 for zero, which no caller here presents: both
// deltas are forced non-zero above.
func sign64(v int64) int64 {
	if v < 0 {
		return -1
	}
	return 1
}
