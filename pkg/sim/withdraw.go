package sim

import "againrom/pkg/intmath"

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

func (w *World) withdrawalPassObserved(obs *withdrawalObs) {
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		if !activity.actorActive(&w.entities[i]) {
			continue
		}
		if !w.nativePatrol(i) {
			continue
		}
		w.withdrawalActor(i, obs)
		w.syncNativePatrol(i)
	}
}

const (
	withdrawalMelee = iota
	withdrawalRanged
	withdrawalMage
)

// withdrawalClass is the threshold class of R2-ENGINE-506. Only a player's
// Retreat setting writes it (byClassRetreatMode): a hired or placed Human
// keeps the zero thresholds it is built with (R2-ENGINE-1795).
func (w *World) withdrawalClass(i int) int {
	e := &w.entities[i]
	switch {
	case !e.Humanoid:
		return withdrawalMelee
	case e.Book.WirePresent(e.KnownSpells):
		return withdrawalMage
	case e.Domain != DomainAir && w.attackRange(i) > 1:
		return withdrawalRanged
	default:
		return withdrawalMelee
	}
}

// withdrawArm is the arm of the withdrawal test an actor meets.
type withdrawArm uint8

const (
	withdrawNone withdrawArm = iota
	withdrawFlee
	withdrawBackOff
)

// withdrawalArm is the arm actor i's withdrawal tests select now and the
// hostile square that selected it. It reads and writes no state.
func (w *World) withdrawalArm(i int) (withdrawArm, []int) {
	if _, using := w.structureUseIndex(w.entities[i].ID); using {
		return withdrawNone, nil
	}
	e := w.entities[i]
	// HP > 0 is both the living-actor test used by the picker and what
	// keeps the zero constructor thresholds inert on a no-health-system
	// entity. Every placed actor with a health system is positive here.
	if !e.Living() || e.Owner == 0 || e.OffMap || w.stoneCursed(i) {
		return withdrawNone, nil
	}
	if e.ActorState == actorStateRetreat && w.retreatProgressBusy(i) {
		return withdrawNone, nil
	}
	// ROM1 reads HP as a signed WORD here (`AI-WITHDRAW-026`); this
	// build compares canonical Entity.HP as int32. The shipped bounds and
	// the unrepresentable high-health edge are recorded by DIV-352.
	if e.HP <= e.Wimpy {
		if hostile := w.withdrawalHostiles(i, w.fleeTestRadius(i)); len(hostile) != 0 {
			return withdrawFlee, hostile
		}
	}
	if e.HP <= e.Withdraw {
		if hostile := w.withdrawalHostiles(i, 2); len(hostile) != 0 {
			return withdrawBackOff, hostile
		}
	}
	return withdrawNone, nil
}

func (w *World) withdrawalActor(i int, obs *withdrawalObs) {
	step, hostile := w.withdrawalArm(i)
	e := w.entities[i]
	switch step {
	case withdrawFlee:
		before := w.entities[i]
		if w.teleportEscape(i, hostile, int64(e.ScanRange)+1, true) {
			obs.record(before, w.entities[i])
			return
		}
		if w.withdrawFrom(i, hostile) {
			w.answerRefusedFlee(i)
		}
		obs.record(before, w.entities[i])
	case withdrawBackOff:
		before := w.entities[i]
		if w.teleportEscape(i, hostile, int64(e.ScanRange)-1, false) {
			obs.record(before, w.entities[i])
			return
		}
		if w.withdrawFromAny(i, hostile) {
			w.answerRefusedFlee(i)
		}
		obs.record(before, w.entities[i])
	}
}

// withdrawalHolds reports whether a withdrawal step would walk or cast actor i
// now: the back-off arm always does, the flee arm when a living hostile stands
// in the list the arm reads. With none, the original takes the idle planner
// (R2-ENGINE-1442), which may heal.
func (w *World) withdrawalHolds(i int) bool {
	step, hostile := w.withdrawalArm(i)
	if step == withdrawNone {
		return false
	}
	if step == withdrawFlee && w.teleportReady(i) {
		hostile = w.withdrawalHostiles(i, w.fleeRadius(i))
	}
	return step == withdrawBackOff || anyLiving(w, hostile)
}

func anyLiving(w *World, list []int) bool {
	for _, ci := range list {
		if w.entities[ci].Living() {
			return true
		}
	}
	return false
}

// teleportArm is the arm of the spell that relocates its caster.
var teleportArm = armWith(func(a spellArm) bool { return a.teleport })

// teleportReady reports a unit of a game whose flee casts Teleport that books
// the spell and has the mana for it. The unit's side does not matter: the
// flee body the explicit withdraw order reaches is the automatic flee's.
func (w *World) teleportReady(i int) bool {
	e := w.entities[i]
	if !w.game.fleeTeleports {
		return false
	}
	id := uint32(w.spellIDOfArm(teleportArm))
	rule, ok := w.bookSpell(e, id)
	return ok && rule.runsArm(teleportArm) && knowsSpell(e, id) && e.Mana >= int32(rule.ManaCost)
}

// teleportEscape is the second game's computer flee and back-off by spell: a
// unit with Teleport in its book and mana for it casts the spell at the cell
// reach cells from the hostiles' mean point, away from the unit (R2-ENGINE-1418).
// It reports true when the unit holds the cast, begun now or already winding
// up; false leaves the unit to the three-cell walk. A flee needs a living
// hostile in the list; a back-off does not. The landing cell the cast takes
// is Unknown.
func (w *World) teleportEscape(i int, hostile []int, reach int64, needLiving bool) bool {
	e := w.entities[i]
	if reach < 1 || !w.teleportReady(i) {
		return false
	}
	id := uint32(w.spellIDOfArm(teleportArm))
	if needLiving {
		// The flee lists hostiles within the block radius alone; with none
		// living the original takes the idle planner, which is not modelled:
		// the unit neither casts nor walks.
		hostile = w.withdrawalHostiles(i, w.fleeRadius(i))
		if !anyLiving(w, hostile) {
			return true
		}
	}
	for _, c := range w.bookCasts {
		if c.Caster == e.ID && uint32(c.Spell) == id {
			return true
		}
	}
	tx, ty, ok := w.fleeCellAt(i, hostile, reach)
	if !ok {
		return false
	}
	w.clearOrder(i)
	w.entities[i].clearAttackBetweenCycles()
	// A cast the cell admission refuses keeps the mana and leaves the caster
	// in place: the original issues it with no cell test (R2-ENGINE-1498).
	w.castOrder(i, CastAt(e.ID, SpellID(id), CellPoint{X: tx, Y: ty}))
	return true
}

// answerRefusedFlee ends a withdrawal move whose far search found no route,
// by the answer the profile holds.
func (w *World) answerRefusedFlee(i int) { refusedFleeAnswers[w.game.refusedFlee](w, i) }

// refusedFleeRule is the answer to a refused withdrawal move.
type refusedFleeRule uint8

const (
	refusedFleeReacquires refusedFleeRule = iota
	refusedFleeStops
)

var refusedFleeAnswers = [...]func(*World, int){
	refusedFleeReacquires: (*World).reacquireAfterRefusedFlee,
	refusedFleeStops:      (*World).stopAfterRefusedFlee,
}

// reacquireAfterRefusedFlee is the first game's answer: an unlatched Move stands
// down, any other unit reacquires within reach (AI-ROUTE-045, AI-327, AI-929).
func (w *World) reacquireAfterRefusedFlee(i int) {
	e := &w.entities[i]
	if e.ActorState == actorStatePatrol || !w.fleeRefused(i) {
		return
	}
	w.clearOrder(i)
	if e.AttackPhase != AttackReady {
		return
	}
	if e.ActorState != actorStateRetreat && w.walksAtStateOne(i) && !w.walkLatched(i) {
		w.stopWalkOrder(i, false)
		return
	}
	w.reacquireWithinReach(i)
}

// walkLatched reports the saved order's command latch (AI-658).
func (w *World) walkLatched(i int) bool {
	o := w.savedOrder(w.entities[i].ID)
	return o != nil && (o.Raw[0x50]|o.Raw[0x51]|o.Raw[0x52]|o.Raw[0x53]) != 0
}

// stopAfterRefusedFlee is the second game's answer: the unit stops at its
// cell (R2-ENGINE-1498).
func (w *World) stopAfterRefusedFlee(i int) {
	e := &w.entities[i]
	if !w.fleeRefused(i) {
		return
	}
	w.clearEscort(i)
	e.clearPatrol()
	e.Retreat = RetreatContinuation{}
	w.stopWalkOrder(i, true)
	w.syncSavedActorCommand(i)
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

// fleeRadius is the radius the flee test and the Teleport arm's list start
// from: the second game's block constant, else the search radius.
func (w *World) fleeRadius(i int) int64 {
	if w.game.fleeBlockRadius != 0 {
		return w.game.fleeBlockRadius
	}
	return w.fleeSearchRadius(i)
}

// fleeTestRadius is the ring count of the flee test, the Wimpy test and the
// explicit Retreat search alike: the radius plus the profile's extra rings (10
// in the second game, R2-ENGINE-1440).
func (w *World) fleeTestRadius(i int) int64 { return w.fleeRadius(i) + w.game.fleeSightExtra }

// fleeSearchRadius is the first game's ring count of the Wimpy and explicit
// Retreat searches: mover byte +0x08 (AI-690), held by an imported Mover or by
// the World's record, the constructor's 5 otherwise (AI-692). DIV-347.
func (w *World) fleeSearchRadius(i int) int64 {
	e := &w.entities[i]
	if r, ok := moverBodies[w.game.movers].fleeRadius(w, e); ok {
		return r
	}
	if m := w.motionFor(e.ID); m != nil {
		return int64(m.Mover[8])
	}
	if r := w.moverFor(e.ID); r != nil {
		return int64(r.Radius)
	}
	return int64(ConstructorSearchRadius)
}

// withdrawalHostiles collects the square block centred on actor i's anchor
// cell, the top-left cell of its footprint (AI-696), in entity order, and
// applies map presence, directional diplomacy and invisibility. It does not
// read sight, reach or route occupancy: the decoded collection is spatial and
// the route search belongs to the ordinary move that follows. Corpses are
// retained here; withdrawFrom owns the later positive-HP gate.
//
// In the first game the square lists the ground slot of each cell: a ground
// or ghost hostile once per footprint cell inside it, a flyer never (AI-697,
// AI-698). The second game lists each hostile once by its anchor cell.
func (w *World) withdrawalHostiles(i int, radius int64) []int {
	self := w.entities[i]
	centre := cellOf(&self)
	var out []int
	for ci := range w.entities {
		if ci == i {
			continue
		}
		candidate := w.entities[ci]
		if candidate.OffMap || !candidate.OrdinaryTargetable() || !w.hostileTo(&self, &candidate) || w.withdrawalHidden(i, ci) {
			continue
		}
		out = withdrawListBodies[w.game.withdrawList](out, ci, centre, &candidate, radius)
	}
	return out
}

// withdrawListRule selects how the withdrawal square lists a hostile.
type withdrawListRule uint8

const (
	withdrawListFootprint withdrawListRule = iota
	withdrawListAnchor
)

// withdrawListBodies holds how each withdrawListRule appends hostile ci.
var withdrawListBodies = [...]func(out []int, ci int, centre cell, c *Entity, radius int64) []int{
	withdrawListFootprint: func(out []int, ci int, centre cell, c *Entity, radius int64) []int {
		if c.Domain == DomainAir {
			return out
		}
		n := footprintSide(c.TokenSize)
		for dy := int32(0); dy < n; dy++ {
			for dx := int32(0); dx < n; dx++ {
				if centre.chebyshevTo(cell{x: c.X + dx, y: c.Y + dy}) <= radius {
					out = append(out, ci)
				}
			}
		}
		return out
	},
	withdrawListAnchor: func(out []int, ci int, centre cell, c *Entity, radius int64) []int {
		if centre.chebyshevTo(cellOf(c)) <= radius {
			out = append(out, ci)
		}
		return out
	},
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
		if w.entities[ci].Living() {
			return w.withdrawFromAny(i, hostile)
		}
	}
	w.acquireStanding(i)
	return false
}

// fleeCell is the cell the decoded arithmetic sends actor i to from the whole
// filtered list, or false for an empty list. It reads and writes no state.
// In the first game the actor is read at its raw position record, cell*256
// plus the sub-cell byte per axis, with no footprint centre: 0x80 standing on
// a cell, and the paid part of the accepted stride between cells (MOVE-174).
// The hostiles are read the same way, at their anchor points (AI-698). The
// second game keeps the centres until its own evidence says otherwise
// (DIV-348).
func (w *World) fleeCell(i int, hostile []int) (tx, ty int32, ok bool) {
	return w.fleeCellAt(i, hostile, 3)
}

// fleeCellAt is fleeCell with the displacement of cells cells in place of the
// walk's three.
func (w *World) fleeCellAt(i int, hostile []int, cells int64) (tx, ty int32, ok bool) {
	var sx, sy int64
	for _, ci := range hostile {
		fx, fy := w.fleePoint(w.entities[ci])
		sx += fx
		sy += fy
	}
	count := int64(len(hostile))
	if count == 0 {
		return 0, 0, false
	}

	mx, my := sx/count, sy/count
	px, py := w.fleePoint(w.entities[i])
	dx, dy := px-mx, py-my
	if dx == 0 {
		dx = 1
	}
	if dy == 0 {
		dy = 1
	}
	stride := cells * subCell
	if intmath.Abs(dx) >= intmath.Abs(dy) {
		px += sign64(dx) * stride
		py += stride * dy / intmath.Abs(dx)
	} else {
		py += sign64(dy) * stride
		px += stride * dx / intmath.Abs(dy)
	}
	return clampPlayable(px>>8, int64(w.bounds.Width)), clampPlayable(py>>8, int64(w.bounds.Height)), true
}

// fleePoint is where the flee pick reads e, by the point the profile holds.
func (w *World) fleePoint(e Entity) (int64, int64) { return fleePoints[w.game.fleeOrigin](w, e) }

// fleeOriginRule is the point the flee pick reads a unit at.
type fleeOriginRule uint8

const (
	fleeFromRecord fleeOriginRule = iota
	fleeFromCentre
)

// fleePoints: the first game reads the raw position record, the second the
// footprint centre.
var fleePoints = [...]func(*World, Entity) (int64, int64){
	fleeFromRecord: func(w *World, e Entity) (int64, int64) {
		x, y := w.savedProjectileTargetPoint(e)
		return int64(x), int64(y)
	},
	fleeFromCentre: (*World).moverFinePoint,
}

// moverFinePoint is e's centre on the 8.8 grid. A mover of side n stores its
// footprint's top-left cell and a sub-cell offset, and the centre the range,
// edge-gap and bearing readers take is that corner plus (n-1)*128 per axis
// (MOVE-087, MOVE-175). The corner is the retained current motion of a loaded actor, the
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
	w.endRetainedCast(i)
	return true
}

// withdrawalHidden reports a hostile the withdrawal list drops for being
// invisible: it is kept while a living member of the unit's Group detects it,
// with no terrain sight asked (AI-815, R2-ENGINE-1879).
func (w *World) withdrawalHidden(i, target int) bool {
	self := &w.entities[i]
	members := w.groupLivingMembers(self.Owner, effectiveGroup(self))
	return !w.perceives(perception{purpose: perceiveGroup, members: members, target: target, detect: detectEdge})
}
