package sim

// A CAST REQUEST AND THE BUSY RULE IT MEETS. Every cast issuer builds a
// castRequest and asks this file what the caster's current action does to it:
// the answer is the issuer's row in castBusyRows, not a branch in the issuer.
// What a refused, queued or held request does next is the issuer's own order
// (a pending row, an armed approach, a pending scroll). Every action that is
// not a cast and waits for one asks castHolds. No row differs between the two
// games; the games' differences around a cast stay gameProfile fields
// (releaseRevalidates, creatureAim).

// castIssuer names who asks for a cast.
type castIssuer uint8

const (
	// castManual is the player's book command, its pending row and the
	// read-only admission explanation the front end asks.
	castManual castIssuer = iota
	castScroll
	// castScript is the mission script's temporary caster: no actor casts.
	castScript
	// castCreature is a creature's class-slot draw in the engage pass.
	castCreature
	// castEngage is a computer mage's full-book draw in the engage pass.
	castEngage
	// castAuto is the escort heal's verdict. The automatic sweep and the
	// support cast ask castBeginRefusal, the same fresh guard, as they begin.
	castAuto
	// castRetained is a retained book order asked again at its approach or
	// its release.
	castRetained
	castIssuerCount
)

// castGate is the action guard a request's admission asks.
type castGate uint8

const (
	// gateNone asks none: the issuer ends the caster's action itself first.
	gateNone castGate = iota
	// gateFresh is actorActionBusy: a new spell may not begin.
	gateFresh
	// gateRetained is retainedCastBusy: the caster's own retained record is
	// not counted.
	gateRetained
)

// castOwned is what a cast that already owns the caster does to a request.
type castOwned uint8

const (
	// ownedPasses asks nothing: the admission's guard decides, or no actor
	// casts.
	ownedPasses castOwned = iota
	// ownedRefuses refuses the request.
	ownedRefuses
	// ownedReplaces rewrites the retained order the cast in flight carries
	// with the request's spell; a request of no spell ends it (AI-376).
	ownedReplaces
	// ownedQueuesBehindPaid keeps a paid wind-up and lets the request wait as
	// the caster's next order; any other cast the issuer replaces itself
	// (AI-356, AI-354).
	ownedQueuesBehindPaid
)

// castStrike is what a loaded strike does to a request.
type castStrike uint8

const (
	// strikePasses asks nothing: the admission's guard refuses it.
	strikePasses castStrike = iota
	// strikeQueues lets the request wait as the next order while the old
	// strike is retained (retainsOldStrike).
	strikeQueues
	// strikeHolds keeps a cast that admission refused for the strike alone
	// armed until the strike ends (strikeHoldsCast).
	strikeHolds
)

// castReaim is how a replacing request aims the retained order.
type castReaim uint8

const (
	reaimNone castReaim = iota
	// reaimSlot aims by the drawn spell's arm (creatureAimOperands).
	reaimSlot
	// reaimVictim aims at the victim itself (victimAimOperands).
	reaimVictim
)

// castBusyRow is one issuer's busy policy.
type castBusyRow struct {
	gate castGate
	// manaWaits lets admission accept a cast the caster cannot yet pay for:
	// the order waits for the mana.
	manaWaits bool
	owned     castOwned
	// repeatRunsOn makes a request repeating the wind-up in progress no new
	// order: the wind-up runs on without a restart.
	repeatRunsOn bool
	strike       castStrike
	// recoveryRefuses refuses the request during a one-shot recovery, before
	// anything is drawn.
	recoveryRefuses bool
	reaim           castReaim
}

// castBusyRows is every issuer's busy policy. The scroll and the script use
// no book admission, so their gate is none; a script cast has no caster and
// meets no busy rule.
var castBusyRows = [castIssuerCount]castBusyRow{
	castManual:   {gate: gateNone, owned: ownedQueuesBehindPaid, repeatRunsOn: true, strike: strikeQueues},
	castScroll:   {gate: gateNone, owned: ownedRefuses, strike: strikeQueues},
	castScript:   {},
	castCreature: {gate: gateFresh, manaWaits: true, owned: ownedReplaces, strike: strikeHolds, reaim: reaimSlot},
	castEngage: {gate: gateFresh, manaWaits: true, owned: ownedReplaces, strike: strikeHolds,
		recoveryRefuses: true, reaim: reaimVictim},
	castAuto:     {gate: gateFresh},
	castRetained: {gate: gateRetained, strike: strikeHolds},
}

// castAim is where a request casts: the target actor, or the cell of a cast
// at a cell. victim is the unit an engage draw is made against, from which a
// replacing draw aims the retained order.
type castAim struct {
	target EntityID
	x, y   int32
	atCell bool
	victim EntityID
}

// castRequest is one issue of a cast. caster is the caster's entity index;
// paid marks a retained record whose cost was spent, which admission lets
// wait for no mana.
type castRequest struct {
	issuer castIssuer
	caster int
	spell  uint32
	aim    castAim
	paid   bool
}

func (r castRequest) row() castBusyRow { return castBusyRows[r.issuer] }

// retainedRequest is a retained book record asked again.
func retainedRequest(ci int, c bookCast) castRequest {
	return castRequest{issuer: castRetained, caster: ci, spell: uint32(c.Spell),
		aim: castAim{target: c.Target, x: c.X, y: c.Y, atCell: c.AtCell}, paid: c.Paid}
}

// castOwned reports that a cast already owning the caster keeps it against
// r, under r's row. A replacing row rewrites the retained order's operands
// first. A refused or queued request goes on as its issuer orders.
func (w *World) castOwned(r castRequest) bool {
	switch r.row().owned {
	case ownedRefuses:
		return w.actorCastBusy(r.caster)
	case ownedReplaces:
		if !w.actorCastBusy(r.caster) {
			return false
		}
		w.replaceRetainedCast(r)
		return true
	case ownedQueuesBehindPaid:
		return w.orderStateAt(r.caster).paidWindUp()
	}
	return false
}

// castRepeats reports a request that repeats the cast still winding up: the
// same spell at the same unit or cell. It is no new order, so the wind-up
// runs on without a restart and a cast queued behind it is dropped.
func (w *World) castRepeats(r castRequest) bool {
	if !r.row().repeatRunsOn {
		return false
	}
	s := w.orderStateAt(r.caster)
	if !s.charging() {
		return false
	}
	b := w.bookCasts[s.bookAt]
	if r.aim.atCell {
		return b.AtCell && uint32(b.Spell) == r.spell && b.X == r.aim.x && b.Y == r.aim.y
	}
	return !b.AtCell && b.Target == r.aim.target && uint32(b.Spell) == r.spell
}

// castStrikeWaits reports that a loaded strike holds r under its row: a
// queuing row waits behind a retained old strike, a holding row keeps a cast
// that admission refused with refusal for the strike alone.
func (w *World) castStrikeWaits(r castRequest, refusal string) bool {
	switch r.row().strike {
	case strikeQueues:
		return w.retainsOldStrike(r.caster)
	case strikeHolds:
		return w.strikeHoldsCast(r.caster, refusal)
	}
	return false
}

// castRecoveryRefuses reports a request its row refuses during a one-shot
// recovery.
func (w *World) castRecoveryRefuses(r castRequest) bool {
	return r.row().recoveryRefuses && w.orderStateAt(r.caster).recoveryWaiting()
}

// castRefusal is the admission verdict for r under its row's guard and mana
// rule, with or without the range and sight tests.
func (w *World) castRefusal(r castRequest, skipRange bool) string {
	row := r.row()
	return w.castAdmission(r.caster, r.spell, r.aim, row.manaWaits || r.paid, row.gate, skipRange)
}

// castBeginRefusal is what a wind-up's start asks of every issuer: the fresh
// guard, whatever the request asked, because by then the issuer has ended
// what its request replaces. A retained order's cost may wait.
func (w *World) castBeginRefusal(ci int, spell uint32, aim castAim, retained bool) string {
	return w.castAdmission(ci, spell, aim, retained, gateFresh, false)
}

func (w *World) castAdmission(ci int, spell uint32, aim castAim, allowInsufficient bool, gate castGate, skipRange bool) string {
	if aim.atCell {
		return w.bookSpellCellAdmissionRange(ci, aim.x, aim.y, spell, allowInsufficient, gate, skipRange)
	}
	return w.bookSpellAdmissionRange(ci, aim.target, spell, allowInsufficient, gate, skipRange)
}

// castActionBusy is the action guard gate names, asked by admission.
func (w *World) castActionBusy(ci int, gate castGate) bool {
	switch gate {
	case gateFresh:
		return w.actorActionBusy(ci)
	case gateRetained:
		return w.retainedCastBusy(ci)
	}
	return false
}

// castWait names what an action that is not a cast waits for.
type castWait uint8

const (
	// waitCastAction is actorCastBusy: escort, guard arm, player defend,
	// retreat and its executor progress, pickup completion, saved groups,
	// the script's attack, the step's attack and escort gates, target orders
	// and the AI's attack order wait while a cast owns the actor.
	waitCastAction castWait = iota
	// waitRetreatProgress is the retreat continuation's cast progress: with
	// a pending row a recovery or a started scroll, otherwise actorCastBusy,
	// and any book record.
	waitRetreatProgress
	// waitStrikeAdvance stands the attack advance down: a book record other
	// than an approach beside a loaded strike (AI-356), or a recovery.
	waitStrikeAdvance
	// waitIdleHeal keeps the idle heal away: a recovery or any book record.
	waitIdleHeal
	// waitStructureUse keeps a paid wind-up through a structure use.
	waitStructureUse
)

// castHolds is the one busy query an action that is not a cast asks: whether
// actor i's cast holds the action why names. Each action keeps its own
// surrounding condition (OffMap, stoneCursed, Transit, the attack phase).
func (w *World) castHolds(i int, why castWait) bool {
	switch why {
	case waitRetreatProgress:
		s := w.orderStateAt(i)
		busy := s.wait || s.scroll == scrollStarted
		if w.entities[i].PendingOrder.Kind == PendingNone {
			busy = w.actorCastBusy(i)
		}
		return busy || s.booked()
	case waitStrikeAdvance:
		s := w.orderStateAt(i)
		return s.booked() && !(s.approaching() && s.strike) || s.wait
	case waitIdleHeal:
		s := w.orderStateAt(i)
		return s.wait || s.booked()
	case waitStructureUse:
		return w.orderStateAt(i).paidWindUp()
	}
	return w.actorCastBusy(i)
}

// actorCastBusy reports whether a book cast owns this actor right now: its
// wind-up record still stands, or its decoded recovery interval has not run
// out. It is the narrow half of the story's action guard.
//
// TWO PREDICATES AND NOT ONE. actorActionBusy (spell.go) asks "may a spell
// begin", and answers no for a loaded attack cycle as well; this one asks "does
// a spell already own the actor", and is what the AI's own retarget consults.
// The player's attack order does not (attachAttack). Collapsing the two is what
// made a fighting unit refuse a retarget click. A loaded attack blocks ordinary
// spell admission; a valid manual book command stores pending operands until
// physical progress ends (AI-356).
//
// Movement consults neither. A movement order from the player, from a script or
// from the AI is always taken; the mover pauses for the wind-up and the
// recovery (orderState.ownsBody) and may walk through a retained retry. A
// destination attached meanwhile advances after the recovery ends (DIV-028).
func (w *World) actorCastBusy(i int) bool { return w.orderStateAt(i).castBusy() }

// actorActionBusy is the CAST admission guard: it answers "may a new spell
// begin on this actor now", and nothing else. A spell may not begin while a
// physical or weapon attack cycle is loaded, and may not begin while a book
// cast owns the actor. That is the non-overlap the contract asks for between a
// armed autocast, idle Heal, AI cast, weapon-spell attack and physical attack.
// Manual book commands validate their target, then cancel the old action
// before reaching this ordinary admission path (DIV-1015).
//
// IT IS NOT A MOVEMENT GUARD AND IT IS NOT AN ATTACK-ORDER GUARD. A movement
// order ends whatever fight the unit was in and is never refused; an attack
// order competes with another attack order under orderAttack's own rule,
// which resets the cycle on a different victim. Applying this predicate to
// either of those made a fighting or approaching unit ignore a right-click,
// which neither contract.md nor spec.md sanctions.
//
// The approach half is deliberately absent too. HasAttackTarget with a
// destination is a walk, not an interval: non-overlap between the attack cycle
// and a cast is already total, because the attack advance and the mover both
// stand down for an actor that carries a cast (step.go), and this predicate
// refuses a cast for an actor whose cycle is loaded.
func (w *World) actorActionBusy(i int) bool {
	if w.motionActive(w.entities[i].ID) {
		return true
	}
	if w.stoneCursed(i) {
		return true
	}
	e := w.entities[i]
	// HasAttackTarget is the durable instruction, not by itself an active
	// interval. At Ready/0 the preceding strike has recovered and another
	// attack or spell may win fresh admission before the next wind-up loads.
	if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		return true
	}
	return w.actorCastBusy(i)
}

// retainedCastBusy is actorActionBusy with the caller's own retained book
// record excluded. It keeps physical/weapon actions and an older direct-cast
// recovery mutually exclusive with retry admission.
func (w *World) retainedCastBusy(i int) bool {
	if w.motionActive(w.entities[i].ID) {
		return true
	}
	if w.stoneCursed(i) {
		return true
	}
	e := w.entities[i]
	return e.AttackPhase != AttackReady || e.AttackCountdown != 0 || w.orderState(&e).recoveryWaiting()
}

// strikeHoldsCast reports a computer caster busy only with its loaded strike:
// its drawn cast waits for the strike's end (AI-554, MAGIC-239, AI-356).
func (w *World) strikeHoldsCast(i int, refusal string) bool {
	e := w.entities[i]
	return refusal == refusalCasterBusy && w.computerControlledUnit(e) &&
		!w.motionActive(e.ID) && !w.stoneCursed(i) && !w.actorCastBusy(i) &&
		(e.AttackPhase != AttackReady || e.AttackCountdown != 0)
}

// replaceRetainedCast is what a draw does to the retained cast order of a
// caster whose cast is pending. The cast in flight is the actor's own and
// runs on; a draw that selects no spell replaces the retained order with the
// ordinary engage, so no further cast follows, and a draw that selects a spell
// replaces the retained operands that the next cast takes, aimed by the
// request's row. A draw that lands during the wind-up leaves the retained
// operands as they are.
func (w *World) replaceRetainedCast(r castRequest) {
	i, id := r.caster, r.spell
	k, ok := w.bookCastIndex(w.entities[i].ID)
	if !ok || !w.bookCasts[k].Retained {
		return
	}
	c := &w.bookCasts[k]
	if id == 0 {
		switch c.Phase {
		case bookCharging:
			c.Retained = false
		case bookRelaxing:
			w.entities[i].CastWait = c.Remaining
			w.bookCasts = append(w.bookCasts[:k], w.bookCasts[k+1:]...)
		default:
			w.bookCasts = append(w.bookCasts[:k], w.bookCasts[k+1:]...)
		}
		return
	}
	if c.Phase == bookCharging || !knowsSpell(w.entities[i], id) {
		return
	}
	aim := w.victimAimOperands
	if r.row().reaim == reaimSlot {
		aim = w.creatureAimOperands
	}
	target, x, y, atCell, ok := aim(i, r.aim.victim, id)
	if !ok {
		return
	}
	c.Spell, c.Target, c.X, c.Y, c.AtCell = uint16(id), target, x, y, atCell
	// The actor's selected spell follows the order, as LOAD restores it.
	w.entities[i].AdmittedBookSpell = uint16(id)
}

// endRetainedCast is an order written over actor i's retained cast order, as
// the withdrawal's move is (AI-763): the order ends as a draw of no spell
// ends it.
func (w *World) endRetainedCast(i int) { w.replaceRetainedCast(castRequest{caster: i}) }

// The refusals a caller turns into a walk, a wait or a hold: the two of
// range, the sight refusal a walking caster walks on through, the busy
// refusal a strike may hold and the mana refusal an order may wait on. Only
// this file compares them, through the predicates below.
const (
	refusalTargetOutOfRange = "target out of range"
	refusalCellOutOfRange   = "cell out of range"
	refusalTargetNotVisible = "target not currently visible"
	refusalCasterBusy       = "caster action busy"
	refusalNoMana           = "insufficient mana"
)

func isRangeRefusal(refusal string) bool {
	return refusal == refusalTargetOutOfRange || refusal == refusalCellOutOfRange
}

func isSightRefusal(refusal string) bool { return refusal == refusalTargetNotVisible }

func isManaRefusal(refusal string) bool { return refusal == refusalNoMana }
