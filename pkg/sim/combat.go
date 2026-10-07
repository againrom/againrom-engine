package sim

import "fmt"

// AttackPhase is where an attacker stands in its attack cycle: charging
// toward a blow, casting toward a release, relaxing after either, or between
// all three and ready to begin.
//
// AttackReady is the ZERO VALUE deliberately, so an entity built without naming
// a phase is an entity that has not begun, and every world assembled before this
// field existed keeps the state it had.
//
// The six values are OURS. The thing being reconstructed numbers its own
// sub-phases 0, 5 and 7, and copying those numbers here would suggest a
// correspondence the rest of this record does not have — nothing else in an
// Entity is at the offset or the width its counterpart holds. What is
// reproduced is the SHAPE: a countdown, and a blow or a release reachable only
// from the charging or the casting one, at a count of exactly zero.
//
// AttackCasting IS NOT A FIFTH SUB-PHASE OF ANYTHING RECONSTRUCTED — it is
// this package's own second charging arm, for a wind-up that ends in a
// release rather than a blow.
//
// A byte outside the six is refused by the constructor and by the decoder
// alike, never folded onto one of them, which is the trade the routing mode and
// the movement domain already make: two values that behaved alike would map two
// byte forms onto one world.
type AttackPhase uint8

const (
	// AttackReady is an attacker that owes nothing and will load its charge at
	// its next turn. A fresh ready order loads immediately; the second boundary
	// reaches ready and returns, preserving the decoded two-turn seam.
	AttackReady AttackPhase = 0
	// AttackCharging is an attacker counting down to a blow. The blow is
	// resolved on the turn the count reaches zero and on no other.
	AttackCharging AttackPhase = 1
	// AttackRelaxing is an attacker counting down out of a blow or a release
	// it has already struck. Nothing is resolved when this count reaches zero:
	// the attacker enters the first scheduler boundary.
	AttackRelaxing AttackPhase = 2
	// AttackCasting is an attacker counting down to a RELEASE rather than a
	// blow: loaded exactly where AttackCharging is loaded, on the same charge
	// (chargeTicks) and bound by the same cycle rule (attackFault) — the only
	// difference is what advanceAttack does at a count of zero, which is
	// releaseWeaponSpell (spell.go) rather than resolveBlow.
	AttackCasting AttackPhase = 3
	// AttackBoundaryOne and AttackBoundaryTwo are the two actor turns between
	// completed recovery and the next retained-order wind-up.
	AttackBoundaryOne AttackPhase = 4
	AttackBoundaryTwo AttackPhase = 5
)

// defined reports whether p is one of the six phases this build knows. It is
// the single place that says which bytes are phases, so a construction and a
// decode cannot come to differ on it.
func (p AttackPhase) defined() bool {
	return p >= AttackReady && p <= AttackBoundaryTwo
}

// relaxJitter is the inclusive top of the draw added to recovery. The complete
// physical interval also carries ranged flight, the Humanoid penalty and two
// scheduler turns; weapon-diverted magic omits only ranged flight.
const relaxJitter = 3

// strikeDistance is the distance a blow is gated on: 1 for two bodies that touch
// and one more for each cell of gap between them (HERO-REACH-025). Each body is
// measured from its centre, the cell anchor plus half its side, so a body wider
// than one cell reaches the same distance on all four of its sides. DIV-1735
// records that the claims do not state which coordinate the original measures.
func strikeDistance(a, t Entity) int32 {
	as, ts := footprintSide(a.TokenSize), footprintSide(t.TokenSize)
	axis := func(ap, tp int32) int64 {
		d := int64(2*ap+as) - int64(2*tp+ts)
		if d < 0 {
			d = -d
		}
		return d << 7
	}
	return strikeDistanceUnits(max(axis(a.X, t.X), axis(a.Y, t.Y)), as, ts)
}

// strikeDistanceForSizes measures between two anchor cells, which is what a
// structure's position is.
func strikeDistanceForSizes(a, t Entity, as, ts int32) int32 {
	dx, dy := int64(a.X)-int64(t.X), int64(a.Y)-int64(t.Y)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return strikeDistanceUnits(max(dx, dy)<<8, as, ts)
}

// strikeDistanceUnits turns a centre distance in 1/256 cell units into the
// strike distance: the footprint term is 0 at one cell each.
func strikeDistanceUnits(units int64, as, ts int32) int32 {
	d := units - (((int64(as) + int64(ts)) << 7) - 0x100)
	if d <= 0x180 {
		return 1
	}
	d = (d + 0x40) >> 8
	if d > 2147483647 {
		return 2147483647
	}
	return int32(d)
}

// hitRollSpan and hitRollFloor are the to-hit roll's inclusive range, [-100,
// 100], as a span the draw covers and the value it is offset by. The two are
// written separately because the draw is over a count of outcomes and the offset
// is over their values, and folding them into one number would hide that the
// span is 201 wide and not 200.
const (
	hitRollSpan  = 200
	hitRollFloor = -100
)

// autoHitRoll is the roll at or above which a blow lands whatever the defence
// is. Its counterpart at the bottom of the range does NOT exist: the arm that
// would be an automatic miss assigns zero to a value already zero and changes
// nothing, so there is one band here and not two.
const autoHitRoll = 90

// clearAttack drops e's attack order and the cycle that only means something
// while it holds one.
//
// The three go together at every site rather than one at a time, because "an
// entity holding no victim holds a ready phase and a zero count" is then true by
// construction. It is not tidiness: both a phase and a count on an entity with
// no victim are states the byte form REFUSES, so a site that forgot one would
// build a world this package can marshal and then not read back.
func (e *Entity) clearAttack() {
	e.PendingOrder = PendingOrder{}
	e.clearPendingAttack()
	e.clearActiveAttack()
}

func (e *Entity) clearActiveAttack() {
	if e.HasAttackTarget {
		e.clearTurn()
	}
	e.AttackTarget, e.HasAttackTarget = 0, false
	e.AttackTargetKind = AttackTargetUnit
	e.AcquirePursuit = false
	e.PursuitIdle = false
	e.AttackPhase, e.AttackCountdown = AttackReady, 0
}

// clearAttackBetweenCycles is what a writer of a pending order does to e's
// attack order: it replaces the order at once between cycles and leaves a
// loaded cycle with its victim. The original's setters and state arms write
// pending-order fields and no progress, and its order machine consumes nonzero
// attack progress before any pending order, so the new order runs once the
// cycle has ended (AI-ORDER-039, AI-RETREAT-272, HERO-CADENCE-112). A writer
// that stores a destination or a structure use beside the cycle is held by the
// movement pass and dropped by the attack pass when the cycle returns to ready
// (step.go, DIV-1509, DIV-1577). A writer of a state alone leaves the retained
// order for the arm of that state to decide on its next pass (DIV-1577).
func (e *Entity) clearAttackBetweenCycles() {
	e.PendingOrder = PendingOrder{}
	e.clearPendingAttack()
	if e.AttackPhase == AttackReady {
		e.clearAttack()
	}
}

// retainCycleForState is clearAttackBetweenCycles for a writer that stores a
// state and no destination: a member holding a loaded cycle keeps its victim,
// and its own cell waits beside the cycle as the order that follows it. The
// original's setter stores pending order 0 and no progress, so once the cycle
// ends nothing loads another until the state's arm decides on a later pass
// (AI-CMD-054, AI-PATROL-018, AI-ORDER-039, HERO-CADENCE-112). Here the attack
// pass drops the retained victim when the cycle returns to ready, so no second
// cycle loads in the ticks before that pass (DIV-1577).
func (w *World) retainCycleForState(i int) {
	e := &w.entities[i]
	e.clearAttackBetweenCycles()
	if e.HasAttackTarget && e.AttackPhase != AttackReady && !e.HasTarget && e.Transit == 0 && !w.usingStructure(e.ID) {
		e.TargetX, e.TargetY, e.HasTarget = e.X, e.Y, true
	}
}

// clearKillCredit is the null pointer shape a credit naming nothing takes, and
// it is clearAttack's counterpart: the two fields that make a reference, and
// only those.
//
// THE SPELL BYTE IS NOT TOUCHED, because the original's own clear arms erase
// the source pointer alone (the constructor's normalisation, world.go, states
// the same thing where it was the only caller).
//
// It exists so that the three places holding this meaning — the
// constructor, remove, and the decoder's refusal — cannot drift. They
// already had: remove carried the attack half of the pair and not this one,
// and a save taken after a credited killer's body decayed away was refused
// on load (owner).
func (e *Entity) clearKillCredit() {
	e.KillCreditSource, e.HasKillCredit = 0, false
}

// chargeTicks is how long e's charge lasts: its own number, and ONE where that
// number is below one.
//
// The floor is ours, and it buys REPRESENTABILITY rather than behaviour. A count
// this returns is written straight into the countdown, and a negative one there
// is a state the byte form refuses; the floor is what makes it impossible for any
// site to store one, a later site that returns between the write and the blow
// included. What it does NOT buy is a different cycle: the decrement below is
// guarded on owing something, so a charge of nought and a charge of one load
// counts that fire on the same advance and no tick can tell the two apart. That
// is worth writing down because a floor usually IS behaviour, and this one is
// not — the test that would witness it does not exist and cannot.
func chargeTicks(e Entity) int64 {
	if e.AttackCharge < 1 {
		return 1
	}
	return int64(e.AttackCharge)
}

// relaxTicks is how long e's relax lasts before the jitter is added: its own
// number, and zero where that number is negative. A relax of zero is an ordinary
// value — the blow's cost is then the charge and the jitter alone — so this floor
// is at zero where the charge's is at one.
func relaxTicks(e Entity) int64 {
	if e.AttackRelax < 0 {
		return 0
	}
	return int64(e.AttackRelax)
}

// rangedExtra is the weapon-flight term. It is part of a physical wind-up
// whenever the current strike distance exceeds one cell.
func rangedExtra(a, t Entity) int64 {
	return rangedExtraDistance(strikeDistance(a, t))
}

func rangedExtraDistance(distance int32) int64 {
	d := int64(distance)
	if d <= 1 {
		return 0
	}
	return (d*256 + 128) / 200
}

func maxRangedExtra(e Entity) int64 {
	d := int64(e.Reach)
	if d <= 1 {
		return 0
	}
	return (d*256 + 128) / 200
}

// humanoidPenalty is the equipped-Humanoid recovery addend. Go integer
// division, like the original's signed IDIV, truncates toward zero.
func (w *World) humanoidPenalty(i int) int64 {
	e := w.entities[i]
	if !e.Humanoid || w.equipment[i][slotWeapon].Empty() {
		return 0
	}
	v := (int64(w.itemWeightOf(w.equipment[i][slotWeapon].Code)) + 5*(30-int64(e.Reaction))) / 12
	if v < 0 {
		return 0
	}
	if v > 12 {
		return 12
	}
	return v
}

// attackFault names what is wrong with e's attack cycle, or nil when the cycle
// is one a tick can produce.
//
// The refused shapes are states this package cannot write: a phase
// this build does not define; a negative count owed, since the count is only ever
// loaded from a non-negative number and only ever falls to zero; and a count
// above what its own phase could have loaded, which is the argument transitFault
// already makes about a crossing — a charging OR a casting count is written
// strictly below the charge (0139 D-4: the two are loaded from the same number
// and bounded alike, so one arm answers for both), and a relaxing one at most
// the relax plus the jitter's own top.
//
// The bounds are taken in int64 because the relax and the jitter are added
// together, and the seven numbers are carried whole: an int32 sum would wrap at
// the top of the range and admit exactly the counts this refuses.
//
// The other shapes the contract names are NOT here, and the division is
// transitFault's. A phase, a count or a victim id on an entity holding no order,
// an order on a unit that is not alive, an order on itself and an order naming a
// unit the world does not hold are all the constructor's to NORMALISE and the
// decoder's to REFUSE — so the constructor cannot produce what the decoder will
// not read back. What is left here is what both sides refuse alike, and it is
// one function so the two cannot come to differ on it.
func attackFault(e Entity) error {
	switch {
	case e.AcquirePursuit && (!e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit):
		return fmt.Errorf("acquisition pursuit requires a unit victim")
	case e.PursuitIdle && (!e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit || e.AcquirePursuit):
		return fmt.Errorf("idle pursuit requires a unit victim and no acquisition")
	case e.AttackTargetKind > AttackTargetStructure:
		return fmt.Errorf("attack target kind %d is not defined", e.AttackTargetKind)
	case e.AttackTargetKind == AttackTargetStructure && e.AttackPhase == AttackCasting:
		return fmt.Errorf("structure attack cannot hold a weapon cast")
	case !e.AttackPhase.defined():
		return fmt.Errorf("attack phase is %d, which is not defined", uint8(e.AttackPhase))
	case e.AttackCountdown < 0:
		return fmt.Errorf("attack countdown is %d, which no tick can leave", e.AttackCountdown)
	case (e.AttackPhase == AttackCharging || e.AttackPhase == AttackCasting) &&
		int64(e.AttackCountdown) >= chargeTicks(e)+maxRangedExtra(e):
		return fmt.Errorf("%d tick(s) owed on a charge of %d plus its ranged bound, which no tick can leave",
			e.AttackCountdown, chargeTicks(e))
	case e.AttackPhase == AttackRelaxing && int64(e.AttackCountdown) > relaxTicks(e)+relaxJitter+12:
		return fmt.Errorf("%d tick(s) owed on a relax of %d, whose longest is %d",
			e.AttackCountdown, relaxTicks(e), relaxTicks(e)+relaxJitter+12)
	case (e.AttackPhase == AttackBoundaryOne || e.AttackPhase == AttackBoundaryTwo) && e.AttackCountdown != 0:
		return fmt.Errorf("attack boundary phase %d owes %d tick(s), want zero", e.AttackPhase, e.AttackCountdown)
	}
	return nil
}

// approach is the PURSUIT half of an attack order: one attacker's
// destination for this tick, decided from where its victim stands right now.
//
// It is what lets an order name a victim its attacker cannot yet reach. Without
// it an attack order at a distance is a cycle that charges and relaxes forever
// against a blow the reach test refuses, and the only reachable order is one the
// player has already walked his unit next to. What is being reconstructed puts a
// victim and a stop distance in ONE order block, so an engage survives being
// farther away than reach; the two fields here are that one block, and this is
// the routine that keeps them agreeing.
//
// The three outcomes, and the order is the contract:
//
//   - a victim the world no longer holds or this actor cannot currently see
//     ends the walk;
//   - a victim CLOSED ON ends the walk too, and that is the stop distance. It
//     is closedOn and not a second comparison, so "walked close enough" and
//     "close enough to act" are one predicate and cannot come apart — the
//     cast's own admission distance while weaponSpell says this attacker is
//     even now eligible, inReach otherwise. THE ATTACKER
//     TURNS TO FACE IT HERE, on this arm and nowhere else;
//   - acquisition turns in place; engagement paths to the victim's current
//     cell (AI-PURSUE-040).
//
// Health is not a pursuit refusal. A linked, reachable target remains the
// retained target until the group/order teardown removes or replaces it, and
// advanceAttack therefore sees the same health-independent application path
// whether the target reached non-positive health this tick or an earlier one.
//
// THE STAND-AND-FACE IS THE STOP ARM'S, and that placement is decoded rather
// than convenient (AI-FACE-066, AI-FACE-067). Facing is a PRECONDITION of an
// attack in the original — the act-state is entered only while the direction
// from attacker to victim equals the attacker's current facing, and at
// progress 0 the test is re-run every tick, so a victim that circles its
// attacker drops it back to the arm that turns to face and stands. The strike
// itself neither tests a facing nor writes one. So the turn goes on the arm
// that stops the walk, beside the stop distance, and advanceAttack withholds
// the charge while that turn is still owed.
//
// THE CYCLE LATCHES (AI-ORDER-039, AI-RETREAT-272, HERO-CADENCE-112). The
// order switch holding the pursuit arms runs only at progress 0; a passing
// facing test latches progress 1, and progress arm 1 keeps the act-state
// until the cycle completes. So from the charge load to the next AttackReady
// this neither turns nor walks: a victim that moves mid-cycle is re-faced or
// chased only once the cycle ends, and a blow whose victim left reach misses.
//
// A victim on the attacker's OWN cell is a zero delta, which names no direction,
// and face leaves the facing as it found it.
func (w *World) approach(s *routeScratch, i int) {
	if w.entities[i].AttackTargetKind == AttackTargetStructure {
		w.approachStructure(s, i)
		return
	}
	ti := indexOfEntity(w.entities, w.entities[i].AttackTarget)
	if ti < 0 || !w.entities[ti].OrdinaryTargetable() || w.invisibleToActor(i, ti) {
		w.entities[i].clearActiveAttack()
		w.restAt(s, i)
		return
	}
	// INVISIBILITY ENDS AT THE APPROACH, NOT AT A LANDED BLOW. This function is
	// that approach: it runs every tick for an actor holding an attack order on
	// a victim the world still has. Removing it at the blow instead let an
	// invisible attacker that missed its to-hit roll stay invisible, and let
	// one whose victim stood out of reach stay invisible indefinitely.
	w.removeAttachedSpell(w.entities[i].ID, 15)
	if w.entities[i].PursuitIdle {
		return
	}
	latched := w.entities[i].AttackPhase != AttackReady
	if t := w.entities[ti]; w.closedOn(i, ti) {
		if !latched {
			w.turnTowardActor(i, t.X-w.entities[i].X, t.Y-w.entities[i].Y)
		}
		w.restAt(s, i)
		return
	}
	if latched {
		return
	}
	if w.entities[i].AcquirePursuit {
		w.turnTowardActor(i, w.entities[ti].X-w.entities[i].X, w.entities[ti].Y-w.entities[i].Y)
		w.restAt(s, i)
		return
	}
	w.walkTo(s, i, w.entities[ti].X, w.entities[ti].Y)
}

// pursuitRefused reports whether a route from attacker i to its unit victim is
// refused under the bodies standing now: one search over the whole map that
// may settle for the cell nearest the victim, refused when it finds nothing or
// when the cell it settles on is still out of the attacker's reach. A jam that
// a route round it outlasts, or that has cleared, is not a refusal. Only an
// attacker that could idle is searched.
func (w *World) pursuitRefused(s *routeScratch, i int) bool {
	e := w.entities[i]
	if !e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit || e.AcquirePursuit || e.AttackPhase != AttackReady {
		return false
	}
	ti := indexOfEntity(w.entities, e.AttackTarget)
	if ti < 0 {
		return false
	}
	t := w.entities[ti]
	route, ok := w.searchRoute(s, i, unitRelation, noWindow, w.farBudgetFor(i), settleOrdered, t.X, t.Y)
	if !ok {
		return true
	}
	if len(route) > 0 {
		end := route[len(route)-1]
		e.X, e.Y = end.x, end.y
	}
	return !inReach(e, t)
}

// attackerRefusable reports whether actor i holds the pursuit a refused route
// can end: an order on a unit victim between attack cycles that is not itself
// an acquisition turn.
func (w *World) attackerRefusable(i int) bool {
	e := w.entities[i]
	return e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit && !e.AcquirePursuit && e.AttackPhase == AttackReady
}

// aimsAtDestination reports whether the near search of actor i toward sub is
// aimed at the final cell of its stored route, the branch of the original's
// dynamic search that raises the route-failure flag when it comes back empty.
func (w *World) aimsAtDestination(i int, sub cell) bool {
	route := w.routes[i]
	if len(route) == 0 {
		return false
	}
	last := route[len(route)-1]
	return last.x == sub.x && last.y == sub.y
}

// answerRefusedPursuit is the order machine's answer to a refused route for a
// unit attacker. The walk is already ended. The pick is the hostile nearest
// within the attacker's reach, the previous victim not excluded, which becomes
// its acquisition order (AI-ROUTE-045, AI-327). With none in reach, or for a
// human participant's short-reach mage (AI-GUARD-007), the order goes idle and
// the victim field is left as it was, since the failure exit never writes it
// (AI-328): nothing walks, turns or strikes until another order is written,
// including a group's reissue at the same victim.
// Acquisition turns, structure victims and a cycle in progress are not a
// refused pursuit and keep their own handling.
func (w *World) answerRefusedPursuit(i int) {
	if !w.attackerRefusable(i) {
		w.answerRefusedPickupWalk(i)
		return
	}
	e := &w.entities[i]
	if victim := w.reacquisitionVictim(i); victim >= 0 && !suppressedAcquirer(*e) {
		if w.orderAcquire(i, w.entities[victim].ID) {
			return
		}
	}
	e.PursuitIdle = true
}

// answerRefusedPickupWalk is answerRefusedPursuit for a walk to a sack: the
// pick-up state is not the move's own, so the nearest hostile within reach
// replaces the request. With none, or for a short-reach human mage, the request
// stays (AI-375, AI-350).
func (w *World) answerRefusedPickupWalk(i int) {
	e := &w.entities[i]
	if e.PendingOrder.Kind != PendingPickup || e.HasAttackTarget || e.OffMap {
		return
	}
	if victim := w.reacquisitionVictim(i); victim >= 0 && !suppressedAcquirer(*e) {
		w.orderAcquire(i, w.entities[victim].ID)
	}
}

// closedOn reports whether attacker i has walked close enough to victim ti
// to stop and act: approach's own stop test and no other caller's, so
// "walked close enough" and "close enough to act" cannot come apart for a
// caster any more than inReach's own doc already keeps them from coming
// apart for a striker.
//
// IT IS THE CAST'S ADMISSION DISTANCE WHILE weaponSpell (spell.go) SAYS i IS
// EVEN NOW ELIGIBLE (FR-2b) — measured the same way castSpell's own range
// test is, by chebyshevTo in whole cells — AND inReach OTHERWISE.
//
// A CASTER IS CLOSED ON ONLY WHILE IT SEES THE VICTIM, OR STANDS NEXT TO IT.
// The release refuses a target the caster's own sight does not reach
// (DIV-1311), so stopping at the spell's range with the victim still unseen
// left a caster standing, never casting, while the rest of its group fought
// (DIV-1524). Unseen, it keeps walking as a plain fighter does until it is
// beside the victim, where it cannot come closer and waits for sight.
func (w *World) closedOn(i, ti int) bool {
	a, t := w.entities[i], w.entities[ti]
	if rule, ok := w.weaponSpell(a); ok {
		d := (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: t.X, y: t.Y})
		return d <= int64(rule.MaxRange) && (d <= 1 || w.actorSeesEntity(i, ti))
	}
	return inReach(a, t)
}

// walkTo gives entity i a destination and takes it out of its layer's plane if
// gaining one is what took it out.
//
// IT IS restAt'S EXACT INVERSE and exists for the same reason: a flyer is
// counted only while it holds no target, so the tick it gains one is the tick it
// stops being an obstacle to its peers, and it must stop before the next entity
// is resolved or a peer would refuse a cell nothing is standing on.
//
// It has one caller. Every other destination in this package is written in the
// command phase, BEFORE the scratch is built, so the seed already accounts for
// it; the approach is the one writer that runs with a live plane, which is the
// whole of why this function exists and why it did not before.
func (w *World) walkTo(s *routeScratch, i int, x, y int32) {
	before := counted(w.entities[i])
	w.cancelTurnForTargetChange(i, x, y)
	e := &w.entities[i]
	e.TargetX, e.TargetY, e.HasTarget = x, y, true
	if !before || counted(w.entities[i]) {
		return
	}
	s.addFootprint(w, e.Domain.layer(), e.TokenSize, e.X, e.Y, -1)
}

// minHP is the least health an entity can carry, and the floor a blow's
// subtraction saturates at.
//
// Health is otherwise NEVER clamped — how far below zero a unit has gone is a
// fact that survives — and this is not a clamp on the life state but on the
// representation: the seven numbers are carried whole, so a damage base near the
// top of the range on a unit near the bottom of it is a subtraction int32 cannot
// hold, and wrapping it would turn a killing blow into a resurrection.
const minHP int32 = -2147483648

// advanceAttack is one attacking entity's turn: at most one phase event, and so
// at most one blow or one release, whatever the world holds.
//
// It is written as a straight line and not as a loop over free transitions. Only
// ONE transition here is free — a ready attacker loads its charge within the
// same turn — and chaining them would be the way to make a zero cadence spin
// forever inside a single tick.
//
// The order is the contract:
//
//   - an order whose victim the world no longer holds, has reached -10 health,
//     has left the map or is invisible to this actor ENDS with no residue and
//     with nothing drawn;
//   - a READY attacker loads its charge, free, into AttackCasting when
//     weaponSpell says it is eligible RIGHT NOW and into
//     AttackCharging otherwise;
//   - it pays one of what it owes, and if it still owes something its turn ends;
//   - owing nothing, THE TRIGGER IS ASKED AGAIN (FR-3a's own live re-ask, at
//     the same site it was asked when this wind-up was loaded): an attacker
//     loaded toward a blow that has since become eligible does not land it, and
//     one loaded toward a cast that has since stopped being eligible does not
//     release it — both DIVERT (FR-3b, D-6) rather than fire. Otherwise a
//     CHARGING attacker resolves its blow and a CASTING one releases its
//     spell; either way it becomes relaxing and loads its relax, a fresh jitter
//     and the Humanoid penalty. Recovery then crosses both scheduler boundaries.
//
// So consecutive physical blows are charge + rangedExtra + relax + [0,3] +
// humanoidPenalty + 2 advances apart. A weapon-diverted release omits
// rangedExtra and otherwise uses the same machine.
func (w *World) advanceAttack(i int, obs *castObs) {
	e := &w.entities[i]

	// A victim absent from the entity list, past the -10 combat boundary, taken
	// off the map or invisible to this actor drops the order.
	structure := e.AttackTargetKind == AttackTargetStructure
	ti := -1
	var position Entity
	var closed bool
	var flight int64
	valid := false
	if structure {
		if s, ok := w.attackStructure(*e); ok {
			position, closed, valid = structurePosition(s), InStructureReach(*e, s), true
			flight = rangedExtraDistance(structureStrikeDistance(*e, s))
		}
	} else {
		ti = indexOfEntity(w.entities, e.AttackTarget)
		// A loaded order on a vetoed victim ends here.
		if ti >= 0 && w.entities[ti].OrdinaryTargetable() && !w.entities[ti].OffMap && !w.invisibleToActor(i, ti) && !w.targetVetoed(i, ti) {
			position, closed, valid = w.entities[ti], w.closedOn(i, ti), true
			flight = rangedExtra(*e, position)
		}
	}
	if !valid {
		e.clearActiveAttack()
		if structure {
			// Transit skips approach, so that pass cannot clear its destination.
			// Drop this attack's pursuit too; clearOrder leaves the crossing's
			// already-committed position and remaining transit payment intact.
			w.clearOrder(i)
		}
		return
	}

	// HERO-CROSSHOLD-146: action state 1 is "moving", and a tick spent there
	// cancels the attack cycle's own phase back to AttackReady rather than
	// freezing whatever phase was loaded — the shared state-1 dispatch
	// unconditionally stores the sub-phase to its ready value on the way in,
	// before ever reaching a blow, a release or a boundary turn. A crossing
	// tick is that state (e.Transit != 0 live, w.motionActive for one resumed
	// from a SAV), and so is a SAV's centred idle turn not yet executed
	// (savedTurnQueued, folded into w.motionActive). The countdown is left
	// exactly as it stood: the next charge overwrites it before anything
	// reads it back.
	//
	// A live turn is NOT on this list. approach turns an attacker only at
	// AttackReady, where the facing test is the act-state's own entry
	// (AI-FACE-066), so a turn withholds the charge load below and never
	// cancels a cycle already latched (AI-ORDER-039, HERO-CADENCE-112).
	if e.Transit != 0 || w.motionActive(e.ID) {
		e.AttackPhase = AttackReady
		return
	}

	// NOTHING HERE TURNS THE ATTACKER, and the absence is the decoded shape
	// rather than an omission (AI-FACE-066, AI-FACE-067). The swing start and
	// the strike were both read end to end and neither contains a facing test, a
	// facing write or a call to any turn routine; the complete turn-to-face
	// producer set is six routines and not one of them is on this path. A
	// consumer that turned an attacker as part of its swing would have invented a
	// coupling the original does not have.
	//
	// The turn belongs to the APPROACH, and approach is where this package makes
	// it: the stop-distance arm stands and faces. See the note there.
	if e.AttackPhase == AttackBoundaryOne {
		e.AttackPhase = AttackBoundaryTwo
		return
	}
	if e.AttackPhase == AttackBoundaryTwo {
		e.AttackPhase = AttackReady
		return
	}
	if e.AttackPhase == AttackReady {
		if e.HasTarget || !closed || e.Turning() || e.PursuitIdle {
			return
		}
		_, weaponSpellEligible := w.weaponSpell(*e)
		weaponSpellEligible = weaponSpellEligible && !structure
		// A WEAPON-BORNE RELEASE MUST NOT LOAD TOWARD A TARGET THIS CASTER CANNOT
		// CURRENTLY SEE (owner report, hotfix DIV-1311): the release itself
		// already refuses on the same live perception predicate a book cast uses
		// (releaseWeaponSpell, spell.go), so a wind-up loaded here regardless is a
		// full charge-and-relax cycle spent on an application already excluded.
		if weaponSpellEligible && !w.actorSeesEntity(i, ti) {
			return
		}
		e.startAction(w.tick, chargeTicks(*e)+relaxTicks(*e))
		if weaponSpellEligible {
			e.AttackPhase = AttackCasting
			e.AttackCountdown = int32(chargeTicks(*e))
		} else {
			e.AttackPhase = AttackCharging
			e.AttackCountdown = int32(chargeTicks(*e) + flight)
		}
	}
	if e.AttackCountdown > 0 {
		e.AttackCountdown--
	}
	if e.AttackCountdown > 0 {
		return
	}

	if e.AttackPhase != AttackCharging && e.AttackPhase != AttackCasting {
		if e.AttackPhase == AttackRelaxing {
			e.AttackPhase = AttackBoundaryOne
		} else {
			e.AttackPhase = AttackReady
		}
		return
	}

	// wasCharging is the kind of wind-up this advance LOADED; casting is
	// whether weaponSpell says the actor is eligible RIGHT NOW, asked again
	// rather than trusted to have held for the whole wind-up (FR-3a). The two
	// disagreeing — loaded toward a blow but now eligible, or loaded toward a
	// cast but no longer eligible — is exactly the DIVERT case (FR-3b, D-6):
	// wasCharging == casting is that disagreement written as one comparison,
	// true when the loaded kind is the wrong one for what the actor is now.
	rule, casting := w.weaponSpell(*e)
	if structure {
		casting = false // direct physical orders only; no structure spell diversion/rider
	} else if casting && !w.actorSeesEntity(i, ti) {
		// AUTHORED, not decoded (owner report, hotfix DIV-1311): a caster
		// loaded toward AttackCasting with sight, that then loses it before
		// the count reaches zero, is folded into the SAME divert FR-3a
		// already gives a weapon that stopped carrying a spell mid-charge —
		// casting is asked again here for exactly that reason, and this is a
		// second "no longer eligible" it did not check. Without it the
		// wind-up still completes and releaseWeaponSpell's own release-time
		// perception check silently refuses, spending the full recovery on a
		// release already excluded instead of returning to READY at once.
		casting = false
	}
	wasCharging := e.AttackPhase == AttackCharging
	if wasCharging == casting {
		// The divert draws exactly what a straight advance always draws at
		// this site — the relax jitter, unconditionally, on THE JITTER IS
		// DRAWN ON EVERY STRIKE TURN's own rule above — and DISCARDS it
		// rather than skip the draw: skipping would make the generator's
		// stream depend on whether the actor is a caster, which is exactly
		// the value a draw must never be skipped on (resolveBlow's own "both
		// draws are taken or neither is", one level down). The discard is
		// written as a discard and not hidden behind a variable nothing
		// reads.
		//
		// It does NOT arm the casting or the charging cycle either: doing so
		// here would store a count equal to its own maximum, a state
		// attackFault refuses, since every other phase stores strictly below
		// what it loaded. The arm happens on the NEXT advance, from
		// AttackReady, at the top of this function — with nothing owed,
		// which is what "diverts to READY" (D-6) means.
		w.rng.uniform(relaxJitter)
		e.AttackPhase, e.AttackCountdown = AttackReady, 0
		return
	}
	if e.AttackPhase == AttackCharging {
		if structure {
			w.resolveStructureBlow(i)
		} else {
			w.resolveBlow(i, ti, obs)
		}
	} else {
		w.releaseWeaponSpell(i, ti, rule, obs)
	}
	e = &w.entities[i]
	// Crossing the target to -10 clears this attack at the damage site. Do not
	// consume recovery jitter or restore a phase after that canonical clear.
	if !e.HasAttackTarget {
		return
	}
	// The relax and the jitter are added in int64 and narrowed once, because the
	// relax is carried whole: an int32 sum would wrap at the top of the range
	// into a negative count the byte form refuses.
	recovery := int32(relaxTicks(*e) + int64(w.rng.uniform(relaxJitter)) + w.humanoidPenalty(i))
	// A weapon-area application may include its source. clearFelled has already
	// removed the action at the damage site; the unconditional recovery draw is
	// still consumed, but no phase or countdown is restored on a dead actor.
	if !e.Alive() {
		w.clearFelled(i)
		return
	}
	e.AttackPhase = AttackRelaxing
	e.AttackCountdown = recovery
	if e.AttackCountdown == 0 {
		e.AttackPhase = AttackBoundaryOne
	}
}

// resolveBlow is the ONE routine that resolves a hit, and it has one call site.
//
// In order, and the order is the contract:
//
//  1. the blow is REFUSED BEFORE ANYTHING IS DRAWN when the victim is out of
//     reach or has no health system. Both are tests on state the caller already
//     holds, so putting them here costs nothing and keeps the draw count a
//     function of the two units' cells and health;
//  2. the damage is base + U[0, spread];
//  3. the roll is U[-100, 100], and the physical pair LANDS when the attacker
//     always hits, when to-hit plus the roll beats the victim's defence, or when
//     the roll alone is at or above the auto-hit band. There is no band at the
//     other end: the arm that would be one assigns zero to a value already zero;
//  4. a miss contributes no primary damage. The second physical component
//     proceeds independently; the third retains its hit-or-empty-primary gate;
//  5. an ordinary landed blow is reduced by the victim's ABSORPTION, flat;
//     an always-hit blow skips that reduction, and either arm floors the
//     physical component at zero;
//  6. XPSlot 1..5 selects the victim's Blade..Shooting resistance byte. The
//     component is multiplied by (100-resistance)/100 with the published
//     +0.75 truncation; slot 0 bypasses the family. A negative result remains
//     in the total (HERO-CLAMP-030);
//  7. a present second physical pair draws and is reduced by signed Water
//     protection, then clamped at zero. One secondary triple admitted by step 4 draws its
//     spread, selects one Fire-through-Astral protection and adds the reduced
//     component;
//  8. and what is left, if it is positive, comes off the victim's health,
//     pays the attacker experience — payExperience's own
//     five refusals, not this routine's, gate whether that payment is
//     anything but a no-op — and fires the fighter's own weapon-spell rider
//     (R3-B3, MAGIC-ITEM-007).
//
// THE FIRST TWO DRAWS ARE TAKEN OR NEITHER IS. Step 2 draws at a physical
// spread of zero and step 3 draws for an attacker that cannot miss, because a
// draw skipped on either value makes the stream's position depend on that
// value. Step 7 draws once for a present second pair, even at zero spread,
// independently of hit, absorption and weapon resistance. It draws again when
// the secondary triple is nonzero AND the physical pair hit or was empty,
// including when the secondary spread itself is zero. A physical miss with a
// nonempty pair skips the third draw, never a present second pair's draw.
//
// EVERY SUM AND COMPARISON IS IN int64. The seven numbers are carried whole, so
// to-hit plus a roll, base plus a spread and damage less absorption are each a
// sum int32 cannot hold at the ends of the range — and a to-hit near the top
// that wrapped negative would turn a certain hit into a certain miss.
//
// THE ITEM-EFFECT SECONDARY COMPONENT USES THE ORIGINAL THIRD-COMPONENT GATE
// (HERO-DMG2-029): a successful physical hit admits it, and an EMPTY physical
// base/spread pair admits it even when that pair's roll missed. A nonempty pair
// that misses suppresses the main and third components, not the second. The
// third uses its base/spread pair and one Fire-through-Astral selector; a later kind 44..48
// effect has already replaced the whole triple during recompute. Ranged weapon
// types 11/12 now enter that same canonical triple and their General accuracy
// is folded before the entity reaches this resolver. What remains absent is
// the flight time a blow at a distance takes before it lands.
//
// obs IS THREADED THROUGH FOR THE RIDER ALONE (R3-B3): resolveBlow itself
// records nothing, on ScriptTrace's own "observation only" rule
// (castevent.go) — it hands obs to weaponRiderApply below unchanged, exactly
// as advanceAttack already hands it to releaseWeaponSpell.
func (w *World) resolveBlow(ai, ti int, obs *castObs) {
	a, t := &w.entities[ai], &w.entities[ti]
	if !inReach(*a, *t) || t.MaxHP <= 0 {
		return
	}

	damageDraw := w.rng.uniform(a.DamageSpread)
	dmg := int64(a.DamageBase) + int64(damageDraw)
	if chance, ok := w.attachedMagnitude(a.ID, 23); ok && w.rng.uniform(100) < chance {
		dmg = int64(a.DamageBase) + int64(a.DamageSpread)
	} else if chance, ok := w.attachedMagnitude(a.ID, 27); ok && w.rng.uniform(100) < chance {
		dmg = int64(a.DamageBase)
	}
	roll := int64(w.rng.uniform(hitRollSpan)) + hitRollFloor
	physicalHit := a.AlwaysHits || int64(a.ToHit)+roll > int64(t.Defence) || roll >= autoHitRoll
	physicalEmpty := a.DamageBase == 0 && a.DamageSpread == 0
	secondary := a.SecondaryDamage
	secondaryPresent := secondary.Base != 0 || secondary.Spread != 0
	secondPresent := a.SecondBase != 0 || a.SecondSpread != 0
	primaryAdmitted := physicalHit || physicalEmpty && secondaryPresent
	if !primaryAdmitted && !secondPresent {
		w.reportPhysicalBlow(*t, ti)
		w.weaponRiderApply(ai, ti, false, obs)
		return
	}
	w.flipOnBlow(ai, ti)

	if !primaryAdmitted {
		dmg = 0
	} else {
		if !a.AlwaysHits {
			dmg -= int64(t.Absorption)
		}
		if dmg < 0 {
			dmg = 0
		}
		if dmg > 0 {
			if slot := int(a.XPSlot) - 1; slot >= 0 && slot < len(t.Resistance) {
				dmg = resistPhysicalDamage(dmg, t.Resistance[slot])
			}
		}
	}
	if secondPresent {
		component := int64(a.SecondBase) + int64(w.rng.uniform(int32(a.SecondSpread)))
		// target+0xc6 is signed Water protection, not the third selector.
		// Do not pre-clamp it: a negative word amplifies this component.
		component = (component*(100-int64(t.Protection[1])) + 75) / 100
		if component > 0 {
			dmg += component
		}
	}
	if secondaryPresent && primaryAdmitted {
		component := int64(secondary.Base) + int64(w.rng.uniform(int32(secondary.Spread)))
		p := int64(t.Protection[secondary.Selector])
		// HERO-CLAMP-030 reads source-current protection signed and clamps
		// only the resulting component. Negative protection amplifies it.
		// Authority belongs to the target's sheet, not the attacker's. Keep
		// the existing 0..100 consumer policy for native/native-retired sheets.
		if t.CurrentProfileBasis != ProfileOriginalCurrent {
			if p < 0 {
				p = 0
			} else if p > 100 {
				p = 100
			}
		}
		component = (component*(100-p) + 75) / 100
		if component > 0 {
			dmg += component
		}
	}
	if dmg <= 0 {
		w.reportPhysicalBlow(*t, ti)
		w.weaponRiderApply(ai, ti, false, obs)
		return
	}
	aliveBeforeBlow := !t.Dead()
	before := *t
	if hp := int64(t.HP) - dmg; hp < int64(minHP) {
		t.setCurrentHealth(minHP)
	} else {
		t.setCurrentHealth(int32(hp))
	}
	w.reportPhysicalBlow(before, ti)
	w.clearFelled(ti)
	// The physical resolver writes attribution before an item rider can replace
	// it. Fighter sources store zero; the point/area spell envelope then stores
	// its actual id when that rider applies.
	w.resolveDamageAttribution(ai, ti, int8(a.XPSlot))
	if !w.entities[ti].OrdinaryTargetable() {
		return
	}

	// The item spell runs before physical experience. Ordinary ids require a
	// positive physical result and a still-positive victim; Fire Ball alone is
	// the fallback when either half fails, including a miss or absorbed blow.
	w.weaponRiderApply(ai, ti, t.HP > 0, obs)
	// The physical award is reconsidered after the rider. A spell that drives
	// the victim to -10 or below suppresses it even though the physical result
	// itself was positive.
	if w.entities[ti].HP > -10 {
		w.payExperience(ai, ti, dmg, aliveBeforeBlow)
	}
}

// resistPhysicalDamage is HERO-DAMAGE-022's
//
//	ftol(damage * (100-resistance) / 100 + 0.75)
//
// after the pre-resistance zero clamp. Keeping the expression in signed
// integer arithmetic makes every byte, including values above 100, exact:
// Go's signed division truncates toward zero as ftol does, and int64 covers
// every reachable signed-int32-field combination times the complete uint8
// range.
func resistPhysicalDamage(damage int64, resistance uint8) int64 {
	return (damage*(100-int64(resistance)) + 75) / 100
}

const (
	xpHalving   = 2
	xpBlowBonus = 1

	// HERO-XP-010
	xpMindSlope   = 4
	xpMindOffset  = 30
	xpMindDivisor = 120
)

func xpRaw(xpValue int32, removed int64, maxHP int32) int64 {
	return int64(xpValue)*removed/(xpHalving*int64(maxHP)) + xpBlowBonus
}

func xpGain(raw int64, mind int32) int64 {
	return raw * (xpMindSlope*int64(mind) + xpMindOffset) / xpMindDivisor
}

func (w *World) payExperience(ai, ti int, removed int64, aliveBefore bool) {
	switch {
	case removed <= 0:
		return
	case !aliveBefore:
		return
	}
	t := &w.entities[ti]
	w.awardSkill(ai, 0, xpRaw(t.ExperienceValue(), removed, t.MaxHP), ti)
}

func inReach(a, t Entity) bool {
	return strikeDistance(a, t) <= int32(a.Reach)
}

// endHeldCycleForWalk ends a loaded cycle on its last boundary tick for an
// actor whose held destination is the only pending order, so the walk's first
// call falls on that tick, two ticks after recovery reaches zero (AI-375). The
// attack pass would end the same cycle later in the tick and drop the same
// record. It reports whether the cycle ended.
func (w *World) endHeldCycleForWalk(i int) bool {
	e := &w.entities[i]
	if e.AttackPhase != AttackBoundaryTwo || !e.HasAttackTarget || !e.HasTarget ||
		e.PendingOrder.Kind != PendingNone || w.usingStructure(e.ID) || e.CastWait != 0 {
		return false
	}
	e.clearActiveAttack()
	w.observeRetreatCompletion(i)
	return true
}
