package sim

import (
	"slices"
	"sort"
)

// Command is the only way to change a world: a KIND, the entity it names, and
// two numbers whose meaning that kind decides. It is a plain value — no pointer,
// no slice, no interface — so it is copied by assignment, serializes as its
// fields, and a frame log that holds one holds nothing that still reaches into a
// world.
//
// THE MOVE-TO IS THE ZERO VALUE, which is the whole of why the field could be
// added at all: every command literal ever written names no kind and is
// therefore still exactly the order it was, so no schedule, no queue and no
// recorded stream changes meaning under this widening.
//
// The two numbers are read per kind and there is no third field for an argument:
// a move-to reads both as the cell it walks to, a damage reads X as its amount
// and nothing reads Y. A separate amount field would sit unused in every command
// but one, and be a second place a coordinate could hide.
//
// A command naming an entity no world holds is ignored, not an error: the sender
// of an order does not know what the world still contains when it arrives. So is
// one naming a kind this build does not define — a switch with no arm for it —
// for the same reason and by the same rule.
// Group is the correlation tag of a group order and is read by that kind ALONE,
// except in KindPlayerDropGold, where the same 32 bits carry the gold amount.
// Every command of that kind carrying the same tag inside one advance is one
// order over the entities they name; two different tags are two orders, and the
// later of them wins where they overlap.
//
// IT IS NOT STATE. Nothing stores it: no world field, no byte-form field,
// nothing in the digest. It correlates commands inside the one Step that
// receives them and is gone when that Step returns, which is what keeps a group
// a thing a player forms rather than a thing a world holds.
type Command struct {
	Kind   uint8
	Entity EntityID
	// Player is the roster slot addressed by KindPlayerParameter,
	// KindPlayerDropGold or by an explicit KindGroupRetreat selection. It is not an entity ID. Parameter
	// commands do not require that player to own an actor.
	Player uint32
	X, Y   int32
	Group  uint32
	Spell  uint16
}

// The command kinds. KindMoveTo is deliberately the ZERO VALUE: a caller who
// names no kind means the order this package had before there were kinds, so a
// widening cannot silently reinterpret anything already written.
//
// Kill and damage are DEBUG TOOLS and not a combat formula. Neither rolls
// anything, neither reads a class, and nothing here decides who may issue one:
// what an attack costs, reaches or hits is a later story's whole subject.
const (
	// KindMoveTo orders the entity to walk to the cell (X, Y).
	KindMoveTo uint8 = 0
	// KindKill sets the entity's health to killHP whatever its maximum, so a
	// unit with no health system is killable like any other. X and Y are unread.
	KindKill uint8 = 1
	// KindDamage subtracts X from the entity's health. Y is unread.
	KindDamage uint8 = 2
	// KindGroupMoveTo orders every entity named by a command of this kind
	// carrying the same Group tag, inside this one advance, to walk as one
	// group toward the cell (X, Y) of the FIRST such command. Where each of
	// them ends up is group.go's, and it is not (X, Y) for all of them.
	KindGroupMoveTo uint8 = 3
	// KindAttack orders the entity to attack the entity whose id rides in X —
	// the same 32 bits under another name. Y is unread.
	//
	// UNLIKE THE TWO ABOVE IT IS NOT A BLOW. It sets a victim and nothing else,
	// and the damage is the CYCLE's, resolved once per advance in the phase
	// after the move. So a command slice applied twice sets one victim twice and
	// costs the victim nothing extra, which is the property the kill and damage
	// arms cannot have.
	KindAttack uint8 = 4
	// KindEquip moves the item at container index X into equipment slot Y —
	// the original's own 1..12 numbering. Spell optionally names a second
	// equipment slot whose complete instance moves to the container in the same
	// atomic command; zero names no second slot.
	//
	// UNLIKE EVERY KIND ABOVE IT TOUCHES NO COMBAT NUMBER AND NO HEALTH: it is
	// a container-and-equipment move, whole and total over its arguments, and
	// the recompute that turns a new loadout into a combat block is a later
	// task's own single call site, never this kind's arm.
	KindEquip uint8 = 5
	// KindCast orders the entity to cast the spell whose id rides in Y at the
	// entity whose id rides in X — the same 32 bits under another name, as
	// KindAttack's victim already is. A third argument field would sit unused
	// in every command but this one, so the two fields every kind already
	// carries hold it instead.
	//
	// IT RESOLVES WHOLLY INSIDE THIS PHASE, on the tick it arrives, and
	// LEAVES NO STATE BEHIND on either entity: nothing here is stored, so a
	// command slice carrying the same order twice costs the caster twice —
	// UNLIKE KindAttack, which sets a victim once and lets the cycle read it
	// on a later tick.
	KindCast uint8 = 6
	// KindGroupStance puts every entity named by a command of this kind
	// carrying the same Group tag, inside this one advance, under the STANDING
	// ORDER whose byte rides in X — the same 32 bits under another name
	// KindAttack's victim already is. Y is unread.
	//
	// The only two bytes it acts on are the guard and stand-ground orders. Any
	// other value reaches no arm and changes nothing, on cmdGroupOrder's own
	// terms for a sub-command this build does not write: the front-end names
	// two things it can ask for and the conversion is on the far side, so a
	// third value here is a caller error rather than a behaviour.
	KindGroupStance uint8 = 7
	// KindGroupPatrolTo sets every entity named by a command of this kind
	// carrying the same Group tag patrolling between the cell it presently
	// stands on and (X, Y) of the FIRST such command.
	//
	// IT NAMES A CELL AND NOT A PATH. The ring is two nodes, which is what the
	// script's own Patrol sub-command builds; where the near end is is not the
	// caller's to say, because it is wherever the member happens to be when the
	// advance reaches it.
	KindGroupPatrolTo uint8 = 8
	// KindGroupSwarmTo is KindGroupMoveTo with the Swarm 2 order rather than
	// the Move order: the same membership rule, the same one destination, the
	// same formation distribution, and a different byte on the group record.
	// What that byte changes is in the engagement pass, not here — a Swarm 2
	// group engages what it scores whether or not it has arrived, where Move's
	// own arm engages only a member that has.
	KindGroupSwarmTo uint8 = 9
	// KindUnequip moves the code held in equipment slot X back into the
	// container, leaving the slot empty (0151, defect 4). Y is unread.
	//
	// It is equip's own inverse and takes the same command shape: an
	// entity moving a code between its equipment array and its container
	// applies in slice order and replays identically on every peer holding
	// the world, the guarantee the command queue already gives equip and
	// now gives its inverse too.
	KindUnequip uint8 = 10
	// KindAutocast sets the entity's own AUTOCAST SPELL to the id in X, 0
	// clearing it. Y is unread.
	//
	// IT IS A COMMAND AND NOT A SETTER, and that is the whole reason the toggle
	// costs a kind at all: the autocast is canonical state, so a front end
	// reaching in to write it directly would put a world change outside the one
	// entry point Step is, and a replay of the same command stream would then
	// not reproduce it. Through the queue it applies in slice order and replays
	// identically on every peer holding the world, which is the guarantee every
	// other kind here already gives.
	//
	// THE ARM REFUSES NOTHING ABOUT THE ID. An id naming no row of this
	// world's table is a runtime answer — autoCast simply finds nothing to
	// cast (spell.go) — on WeaponSpell's own ground (world.go), and an id
	// wider than the field is narrowed to 0, which is "no autocast".
	KindAutocast uint8 = 11
	// KindCastAt orders a point-target spell at cell (X,Y). Spell carries the row id.
	KindCastAt uint8 = 12
	// KindDropCarried drops one unit of the container ELEMENT at Spell (the
	// same element indexing KindEquip's own X reads) to the ground, at the
	// cell (X, Y) when ITEM-DROP-008's own window allows it (drop.go), and at
	// the entity's own cell otherwise. It is destination code 3 of
	// ITEM-CMD-007, source code 2 (1005 round 2).
	KindDropCarried uint8 = 13
	// KindDropWorn is KindDropCarried's own source restated for equipment
	// slot Spell (1..12, KindUnequip's own numbering) instead of a container
	// element — destination code 3, source code 1 (1005 round 2).
	KindDropWorn uint8 = 14
	// KindReadBook consumes one unit of the container element at X and teaches
	// its one kind-42 spell to a living mage. Y, Group and Spell are unread.
	KindReadBook uint8 = 15
	// KindTerminalKill is the corpse-drop test/tool boundary: unlike KindKill,
	// which preserves the historic shallow -1 body, it places the target at the
	// first non-targetable health value. X and Y are unread.
	KindTerminalKill uint8 = 16
	// KindAttackStructure names a structure handle in X, never a unit handle.
	KindAttackStructure uint8 = 17
	// KindGroupDefend carries the protected actor ID in X and the selection in Group.
	KindGroupDefend uint8 = 18
	// KindGroupRetreat installs explicit Retreat on the addressed Player's
	// selection. It is unrelated to KindPlayerParameter selector 3.
	KindGroupRetreat uint8 = 19
	// KindUsePotion applies one carried potion to its owner. X is the
	// container element index; admission and consumption are atomic.
	KindUsePotion uint8 = 20
	// Scroll unit orders carry target in X and carried index in Spell; point
	// orders carry the cell in X/Y and the same carried index in Spell.
	KindUseScroll    uint8 = 21
	KindUseScrollAt  uint8 = 22
	KindUseStructure uint8 = 23
	KindPickUp       uint8 = 24
	// KindPlayerParameter is ROM1 player-command opcode 0x46. Player is the
	// addressed roster slot, X is the selector/sub-code and Y is its authored
	// value. Unlike entity commands it is applied before entity lookup: a player
	// setting does not require that player to own a unit at this instant.
	KindPlayerParameter uint8 = 0x46
)

// OrderGuard and OrderStandGround are the two values KindGroupStance's X may
// carry: the group order bytes for Guard and Stand Ground.
//
// They are exported because a caller of Step has to be able to name which of
// the two it is asking for, and X is where the arm reads it. They ALIAS the
// constants the engagement pass itself forks on rather than re-spelling the
// numbers, so a caller and the arm cannot come to disagree about a value; and
// they are the only two of the six that are exported, because they are the only
// two this command carries — everything else about the vocabulary stays behind
// its own command kind.
const (
	OrderGuard       = int32(orderGuard)
	OrderStandGround = int32(orderStandGround)
)

// killHP is the shallow dead value the historic debug kill leaves behind. It
// remains finishable/restorable; KindTerminalKill names the separate -10 tool.
const killHP = -1

// Step advances w one tick against cmds, mutating w in place. It is the only
// exported call in this package that advances a world.
//
// The four phases run in this order, and the order is the contract:
//
//  1. every command in cmds, in slice order, is applied to its entity — so a
//     later command for one entity overwrites an earlier one, and "later" means
//     later in the slice, never later in entity order;
//  2. every ALIVE entity holding a target is resolved, in ascending id, by the
//     five steps below — a downed unit and a corpse are advanced by nothing;
//  3. every ALIVE entity holding a VICTIM takes one turn, in ascending id, and
//     every one of them takes it after the whole of phase 2 — so a blow is
//     resolved against the cells this tick left units on;
//  4. the tick is incremented, exactly once.
//
// Commands apply before the move, so an order given at tick t is walked at tick
// t and not at t+1.
//
// At an entity's turn:
//
//   - an entity that OWES TRANSIT TICKS pays one and does nothing else: it has
//     already taken the cell it is crossing to, and the ticks are what that cell
//     costs at its own rate. It is asked before its order is, so a mover that
//     arrived mid-stride still finishes the crossing it began;
//   - an entity BEGINNING the tick on its target neither searches nor steps: its
//     target and its route are cleared and nothing else about it moves;
//   - its stored route is taken, and replaced by a fresh FAR search when it holds
//     none, when the route's last cell is not its current target, or when the
//     sub-goal that route yields lies outside its window. Those three tests are
//     the only path to a far search and they are asked once, here;
//   - a NEAR search runs from its cell to that sub-goal;
//   - it advances to the first cell of the near route and no further, and TAKES
//     ON THE TRANSIT that cell costs it: the rate composed from its own speed,
//     its movement domain, the two cells' terrain and whether the step was
//     diagonal, turned into a whole number of ticks. A mover whose speed is zero
//     or less has no rate and owes nothing, which is a cell a tick;
//   - and if the cell it landed on is one of the first four cells of its stored
//     route, that cell and every cell before it are dropped.
//
// The two searches read different relations and carry different bounds, and
// between them that is the whole of the arrangement. The far search reads
// TERRAIN alone and sweeps the whole map: it is the one that must find a detour
// round a lake, so it may not be bounded in space, and being unit-blind its
// answer does not depend on where in a tick it runs — which is what makes
// spreading it over ticks a scheduling change later rather than a redesign. The
// near search reads terrain AND occupancy as it stands at this entity's turn, so
// it must run here and nowhere else, and it is windowed: at most 289 cells,
// however far away the order pointed.
//
// A stored route is state, not a cache. It is carried by the byte form and it
// enters the digest, because recomputing one from its own later cells does not
// give the tail back — the wave stops in the FIRST generation that labels the
// goal, so which of several equal-cost routes it returns depends on where it
// started, and the mover's start moves along the route it is walking. That
// argument used to lean on a budget that shrank as a mover closed on its goal;
// the far budget is flat now and the property survives without it, measured over
// the same corpus in routefork_test.go.
//
// WHICH search runs is the world's own routing mode, fixed when it was built and
// carried by its bytes. It decides which route is taken and nothing else about a
// tick: the resolution order, the cost model, the neighbour set and every
// clearing rule below are the same under both, and both arms of the two-tier
// arrangement run under it.
//
// The move is one cell, and it is all-or-nothing. There is no sliding along
// whichever axis happens to be free, and an entity with no route keeps its cell
// and its target whole — an obstacle is not an arrival, so HasTarget still tells
// a caller the difference.
//
// An entity holding a target has exactly FIVE outcomes IN PHASE 2 and there is
// no sixth — phase 3 can still fell it, and a felled unit's order goes with the
// rest of its residue: it owes transit ticks, and pays one; or it advances one cell, takes on
// that cell's transit and its stall count returns to zero; or its NEAR search
// finds nothing, it keeps everything, and its stall count rises by one; or that
// rise reaches the limit, and the target is cleared with no residue inside the
// same tick that raised it; or its FAR search finds nothing and the target is
// cleared in that same tick.
//
// Which search failed is the whole of what tells the last two apart, and the
// asymmetry is the relations'. Occupancy changes every tick, so a near failure
// is worth waiting out and the count measures how long a unit has been waiting.
// Terrain does not change while a world is advanced, so a far failure is a
// verdict rather than a delay: a unit that cannot reach its order will not be
// able to reach it sixteen ticks later either, having spent a whole-map sweep on
// each of them to find out.
//
// The give-up fires where the count is raised rather than on the following tick,
// so a stored count is always below the limit and the byte form's refusal of one
// at or above it can never fire on a world this package produced.
//
// Ascending id is the whole of the priority, and resolution is incremental: an
// entity resolved at its turn searches against the cells the lower ids have
// already taken this tick and the cells the higher ids have not yet left. So two
// entities converging on one cell leave it to the lower id, and a cell a lower id
// vacates within a tick is free to a higher id behind it. Only the NEAR search
// sees any of that; the far search is blind to it by construction.
//
// A world whose entities already share a cell is advanced, not repaired. A near
// search never routes an entity onto a cell another entity holds, so no cell's
// occupant count rises above one it did not already carry, and such a pair parts
// only if movement happens to part it.
//
// WHO BLOCKS IS NOT WHO MOVES, and the two lines fall in different places. A
// unit that is not alive never moves. It still blocks while it dwells or while
// ordinary Heal can restore it at those coordinates; only a terminal body past
// its dwell occupies nothing. The advance test reads ALIVE and the occupancy
// seed reads that longer lifecycle separately.
//
// Movement is CLAMPED by the map: a target may still name any cell, but an
// entity only ever moves onto a cell that is in bounds, has its blocks-ground bit
// clear, and is held by no other entity. So an order off the map, onto water or
// into a wall is not refused — it is searched for, found unreachable, and the
// entity holds its cell. A search still STARTS from the entity's own cell
// whether or not that cell is enterable, so an entity standing off the map or on
// a blocked cell can be routed off it.
//
// All of it is integer arithmetic over the world's own fields. Nothing here
// reads a clock or touches a file, and the ONE thing that draws is the attack
// cycle, from the world's own generator, whose state the byte form carries — so
// the same prior state and the same commands yield the same next state, always.
// A BLOW IS APPLIED IN PHASE 1 LIKE ANY OTHER COMMAND, which is what makes a
// kill and a move-to reach a world by the one path. The two arms are no-ops
// rather than errors wherever they cannot do their work — an entity the world no
// longer holds, one already dead, a damage of nothing or a damage to a unit with
// no health system — because the sender of a blow knows no more about what the
// world still holds than the sender of an order does.
func Step(w *World, cmds []Command) { stepWorld(w, cmds, nil, nil, nil) }

// StepTraced is Step with the mission script observed: the same tick, advanced
// the same way, returning what the script did on it.
//
// IT IS THE SAME CODE PATH. Both entry points call stepWorld and the only
// difference is the trace it is handed; every recorder returns at once on the
// nil one, so a traced run and an untraced run of the same world are the same
// world at every tick, byte for byte. Observation is not a mode this simulation
// has — it is a value the caller keeps.
//
// The returned trace is empty whenever neither script phase runs, which is
// what Empty answers. Source clocks test pass/report at different boundaries.
func StepTraced(w *World, cmds []Command) ScriptTrace {
	var tr ScriptTrace
	stepWorld(w, cmds, &tr, nil, nil)
	return tr
}

// StepObserved is Step with the tick's APPLIED CASTS reported: the same
// tick, advanced the same way, returning who cast what at whom.
//
// IT IS THE SAME CODE PATH, on StepTraced's own terms above: both entry points
// call stepWorld and the only difference is the sink they hand it, which is nil
// for Step and records nothing on a nil receiver — so an observed run and an
// unobserved run of the same world are the same world at every tick, byte for
// byte. Observation is a value the caller keeps, never a mode this simulation
// has.
//
// The result is nil on every tick no cast landed, which is most of them.
func StepObserved(w *World, cmds []Command) []CastEvent {
	return StepReported(w, cmds).Casts
}

// Report is one tick's whole client-visible observation: the applied casts, the
// cells each staged area effect painted on it, and ordered causal damage
// messages. Passive regeneration and corpse decay emit no damage message.
//
// It is a RETURN VALUE and not a mode, StepObserved's own rule: both fields are
// built inside one advance and dropped by a caller that does not want them, and
// a world advanced by Step is the same world byte for byte.
type Report struct {
	ScriptMessages []int32
	Casts          []CastEvent
	ScriptCasts    []ScriptCastEvent
	AreaPaints     []AreaPaint
	Damages        []DamageEvent
}

// StepReported is Step with all observations reported. StepObserved is this
// with one field read, so the two cannot come to disagree about what a tick did.
func StepReported(w *World, cmds []Command) Report {
	var obs castObs
	stepWorld(w, cmds, nil, &obs, nil)
	return Report{ScriptMessages: obs.scriptMessages, Casts: obs.casts, ScriptCasts: obs.scriptCasts,
		AreaPaints: obs.paints, Damages: obs.damages}
}

// StepWithdrawalTraced is Step with the successful phase-6 withdrawal
// decisions returned at the boundary between the ordinary AI producers and
// the withdrawal tail. A non-phase-6 tick, or a failed tail, returns nil.
// Observation changes no canonical state.
func StepWithdrawalTraced(w *World, cmds []Command) []WithdrawalDecision {
	var withdrawals withdrawalObs
	stepWorld(w, cmds, nil, nil, &withdrawals)
	return withdrawals.decisions
}

func stepWorld(w *World, cmds []Command, tr *ScriptTrace, obs *castObs, withdrawals *withdrawalObs) {
	w.beginTurnSubTick()
	defer w.endTurnSubTick()
	if obs != nil {
		w.damageObservation = &damageObservation{}
		defer func() {
			obs.damages = w.damageObservation.events
			w.damageObservation = nil
		}()
	}
	// A raw native decode is readable without a loader. It is not executable
	// until its source arithmetic rule is rebound: even timer expiry may call
	// derive. Refuse before advancing time or modifying attachments.
	if w.sourceDerive == nil {
		for _, e := range w.entities {
			if e.ActorLoad.Source.Class == 2 {
				return
			}
		}
	}
	w.initializeActionClocks()
	w.takePendingOrders()
	if w.hasSessionClock {
		w.beginSessionTick(tr, obs, withdrawals)
	}
	// A world with no layer at the start of a tick has nothing to recompute for
	// one first present during it.
	if len(w.effects) == 0 && len(w.areaCosts) == 0 {
		w.areaCostLive = true
	}
	w.openCostWindow()
	// The death award consumes attribution only after the tick's spell envelopes
	// and physical awards have all run. Remember actors that have not yet crossed
	// below zero; the tail scan then catches each crossing once without adding a
	// transient queue to canonical World state.
	beforeHP := make(map[EntityID]int32)
	for i := range w.entities {
		if w.entities[i].OrdinaryTargetable() {
			beforeHP[w.entities[i].ID] = w.entities[i].HP
		}
	}
	// THE SPELL EFFECT MARKS AGE FIRST, BEFORE ANYTHING CAN SET ONE. A mark set
	// later on this tick then stands its full life, and one set on the previous
	// tick loses exactly one tick of it — so the mark's life is a count of
	// ticks and not a count of ticks-that-set-no-mark.
	w.decaySpellEffects()
	// The shared effect list precedes actor attachments (MAGIC-POISONPHASE-159).
	// Root and append order determine whether a released child runs this pass.
	w.stepSavedWorldEffects(obs)
	w.stepAttachedEffects()
	// The existing world-body walker resolves pending script casts here.
	// Legacy native scheduling produces script casts later in this body, so
	// they wait for the next call. The source wrapper ran phase6 before this
	// body and its casts may already be present. This placement is the bounded
	// wrapper policy, not recovery of the original temporary-caster population
	// or its complete first-tick chronology.
	w.stepScriptCasts(obs)
	// An already-active turn owns the actor's first advance of this tick. It is
	// consumed before book wind-up and before any producer below may begin a new
	// turn, so a turn started on this tick never loses its first interval.
	w.advanceTurns()
	// A book cast applies only when its actor's decoded wind-up reaches zero.
	// Released ids are carried only across this tick so the autocast sweep does
	// not consume recovery immediately or re-arm a zero-recovery caster.
	interrupted := w.manualActionInterruptions(cmds)
	released := w.stepBookCasts(obs, interrupted)
	released = unionOfClaims(released, w.stepScrollCasts(obs, interrupted))
	// commanded is a SECOND set and not more entries in released, because the
	// two are read for different reasons. released names the casts that ended
	// this tick: it suppresses the autocast sweep AND holds recovery still for
	// the release tick, which is what makes that last tick belong wholly to the
	// old action. commanded names actors an explicit attack or cast claimed
	// this tick: it suppresses the sweep only.
	//
	// A manual replacement may claim a recovering actor, clearing its old
	// recovery. It remains distinct from a release, which must hold recovery
	// still for the tick on which the effect actually applied.
	var commanded map[EntityID]bool
	claim := func(id EntityID) {
		if commanded == nil {
			commanded = make(map[EntityID]bool)
		}
		commanded[id] = true
	}
	// consumed marks the commands a group order has already taken, and it is
	// allocated only when one arrives — so a slice carrying no group order pays
	// nothing for the possibility, and the phase stays the single pass in slice
	// order it has always been.
	var consumed []bool
	for k, c := range cmds {
		if k < len(consumed) && consumed[k] {
			continue
		}
		// Opcode 0x46 addresses a PLAYER, not an entity. Its dispatch therefore
		// stands before both the group arm and the entity lookup, beside those
		// two alternative command scopes rather than behind either one.
		if c.Kind == KindPlayerParameter {
			w.applyPlayerParameter(c.Player, PlayerParameter(c.X), c.Y)
			continue
		}
		if c.Kind == KindPlayerDropGold {
			w.applyPlayerDropGold(c.Player, c.Group, CellPoint{X: c.X, Y: c.Y})
			continue
		}
		// The group arm stands BEFORE the entity lookup, because a group order
		// is not one entity's: its first command may name an entity the world no
		// longer holds, and the order over the rest of its members must still be
		// issued. It is applied at the position of its first member, so a later
		// order still overwrites an earlier one exactly as two single orders do.
		//
		// SINCE 0146 THERE ARE FOUR GROUP KINDS AND NOT ONE, and every one takes
		// this same arm: groupOrder resolves WHO the order reaches and WHERE it is
		// aimed identically for all four, and forks on the kind once, at the
		// write.
		if isGroupKind(c.Kind) {
			if consumed == nil {
				consumed = make([]bool, len(cmds))
			}
			w.groupOrder(cmds, k, consumed)
			continue
		}
		i := indexOfEntity(w.entities, c.Entity)
		if i < 0 {
			continue
		}
		e := &w.entities[i]
		switch c.Kind {
		case KindUseStructure:
			if w.beginStructureUse(i, StructureID(uint32(c.X))) {
				claim(e.ID)
			}
		case KindMoveTo, KindPickUp:
			if c.Kind == KindPickUp {
				found := false
				for _, sack := range w.sacks {
					found = found || sack.X == c.X && sack.Y == c.Y
				}
				if !found {
					continue
				}
			}
			// A unit that is not alive takes no order, and the order is dropped
			// HERE rather than skipped at the walk: a target set and then not
			// walked is residue on a unit that may hold none, so the world would
			// end the tick in a state its own byte form refuses. Ignored is the
			// same outcome an order for an absent entity gets, and for the same
			// reason — the sender does not know what the world still holds.
			//
			// AND IT IS NOT REFUSED FOR BUSYNESS OF ANY KIND. An approaching,
			// charging, relaxing or casting unit takes this order like any
			// other: the fight ends here and the destination is written here.
			// A unit whose book cast is winding up does not advance its body until
			// that cast releases (the mover stands down for it, below), so the
			// order is attached now and walked afterwards rather than dropped.
			if !e.Alive() || w.stoneCursed(i) {
				continue
			}
			w.cancelScroll(i)
			// A stored route's last cell must equal the entity's target, or the
			// route must be empty (decodeRoutes, binary.go). The walk below
			// rebuilds a mismatched route on the same tick for an ordinary
			// mover, but not for an actor a book cast owns through its
			// wind-up or one still paying transit ticks: both are skipped by
			// the walk, below, so a route that still ends at the OLD target
			// would stand beside the new one for the rest of the tick. That
			// is a state decodeRoutes refuses, and the game's save is that
			// byte form verbatim (R3-C1, round3-review.md). Dropping the
			// stale route here, at the write, costs one fresh search on
			// whichever tick the mover next moves and nothing else. The
			// drop is unconditional on cast or transit state, because this
			// write does not know which of those the walk below is about to
			// hit; it fires only when the order actually changes the target,
			// so a re-issued identical destination keeps its route.
			w.attachMoveOrder(i, c.X, c.Y, c.Kind == KindPickUp)
			// A MOVEMENT ORDER DOES NOT CLAIM ITS ACTOR AGAINST THE AUTOCAST
			// SWEEP. It did until this round, which made "this unit has
			// somewhere to go" mean "this unit does not cast" for the tick the
			// order arrived on — and for every tick, for a client that emits a
			// move command while the button is held.
			//
			// The guarantee that motivated the claim is kept by underCommand
			// (engage.go), which autoCastOrder already consults for the
			// unbidden, unarmed Heal alone: a unit holding a destination and no
			// victim is under command, so idle healing cannot detach the order.
			// An ARMED autocast is the player's own instruction and is meant to
			// reach a moving unit.
		case KindKill:
			// Nothing is done to a unit that is already dead — not the health,
			// which would deepen it, and not the order, which it does not hold.
			if e.Dead() {
				continue
			}
			before := *e
			e.setCurrentHealth(killHP)
			w.reportHealthLoss(before, i)
			w.clearFelled(i)
		case KindTerminalKill:
			if !e.OrdinaryTargetable() {
				continue
			}
			before := *e
			e.setCurrentHealth(decayBonesHP)
			w.reportHealthLoss(before, i)
			w.clearFelled(i)
		case KindDamage:
			// A non-positive amount is refused rather than applied, so subtracting
			// it cannot heal; a non-positive maximum is a unit with no health
			// system, which damage does not reach at all. Neither is an error: an
			// amount of zero is a resolved outcome and so is a blow on a unit
			// nothing can wound.
			if !e.OrdinaryTargetable() || c.X <= 0 || e.MaxHP <= 0 {
				continue
			}
			before := *e
			e.setCurrentHealth(e.HP - c.X)
			w.reportHealthLoss(before, i)
			w.clearFelled(i)
		case KindAttack, KindAttackStructure:
			// The same refusals a move order gets, and two more. A unit that is
			// not alive takes no order; an order for an entity the world does not
			// hold was already dropped at the lookup above; and a victim the
			// world does not hold, or one that is the attacker itself, is ignored
			// for the reason the other two are — the sender of an order does not
			// know what the world still contains, and nothing here attacks
			// itself.
			victim := EntityID(uint32(c.X))
			kind := AttackTargetUnit
			if c.Kind == KindAttackStructure {
				kind = AttackTargetStructure
			}
			if !e.Alive() || w.stoneCursed(i) || w.scrollCastPending(i) || (kind == AttackTargetUnit && (victim == c.Entity || indexOfEntity(w.entities, victim) < 0)) {
				continue
			}
			if kind == AttackTargetStructure && indexOfStructure(w.structures, StructureID(victim)) < 0 {
				continue
			}
			// A GROUP'S ATTACK ORDER DOES NOT TAKE A STAFFLESS HEALER (DIV-1736).
			// The mage has nothing to strike with, and holding a victim would end
			// the idle healing the party counts on.
			if kind == AttackTargetUnit && w.stafflessHealer(i) && attackedTogether(cmds, k) {
				continue
			}
			w.cancelScroll(i)
			// AN ORDER NAMING THE VICTIM ALREADY HELD LEAVES THE CYCLE ALONE, and it
			// ENDS WHATEVER WALK THIS UNIT WAS ON. Both rules moved into orderAttack
			// when the engagement decision became a second producer of this order: a
			// rule with two callers written out at both of them is a rule that comes
			// to differ between them, and the first of these two is load-bearing on
			// the decision's side rather than a convenience — see the note there.
			//
			// The CROSSING is not ended, by the rule that already lets an arrival
			// finish one: a body between two cells is where it is whatever it has
			// been told to do next. Nor is the group rate term, which the two move
			// arms zero because the order they carry is a movement order and the
			// term is a movement rate.
			//
			// A BOOK CAST IN FLIGHT DOES NOT REFUSE THE ORDER. The victim is
			// attached and the fight on it begins after the cast, so an armed
			// mage that is in a cast nearly every tick still takes an order.
			// The one refusal kept is the vetoed target's: its acquire in place
			// would turn a caster whose facing the cast has admitted.
			//
			// AI-CMD-054's ordinary target-cost veto faces the named actor
			// and acquires in place. It does not install a named engagement.
			if kind == AttackTargetUnit {
				ti := indexOfEntity(w.entities, victim)
				if w.targetVetoed(i, ti) {
					if w.actorCastBusy(i) {
						continue
					}
					w.cancelStructureUse(e.ID)
					// Replace the previous primary action before acquiring.
					// clearOrder preserves an already admitted crossing.
					w.clearOrder(i)
					e.clearAttack()
					w.commandGroup([]int{i}, orderNone, cell{})
					w.acquireInPlace(i)
					w.syncSavedActorCommand(i)
					w.syncSavedPost(i)
					w.turnTowardActor(i, w.entities[ti].X-e.X, w.entities[ti].Y-e.Y)
					claim(e.ID)
					continue
				}
			}
			if w.attachAttack(i, victim, kind, true) {
				w.cancelStructureUse(w.entities[i].ID)
				if escortState(e.ActorState) || e.ActorState == actorStateAcquire || e.ActorState == actorStateRetreat || e.ActorState == actorStatePickupComplete || w.savedMoveGroup(e.ID) {
					w.commandGroup([]int{i}, orderNone, cell{})
				}
				// THE ORDER CARRIES THE ENGAGE STATE (1141). `AI-CMD-054`
				// reads the player's order `0x19` writing `actor+0x50 = 3`
				// at `L00014` for every member it sends at the target.
				// It is written here for the same reason the script's own
				// sub-command 10 writes it: the release above leaves a unit
				// at guard under a cleared group order, which is the one
				// population the per-actor guard arm decides — and that arm
				// would break this order off on the next tick, five cells
				// from the unit's post.
				w.publishEngagementOrder(i)
				claim(e.ID)
			}
		case KindEquip:
			w.equip(i, int(c.X), int(c.Y), int(c.Spell))
		case KindUnequip:
			// The move itself is equip.go's own unequip, on equip's own
			// precedent (D-3): the refusal of an out-of-range or
			// already-empty slot is that function's, not a guard written
			// out again here.
			w.unequip(i, int(c.X))
		case KindAutocast:
			// The whole arm, on castSpell's own precedent below: a value
			// is stored and nothing is resolved. Setting it does NOT
			// reset the wait, so a spell put on repeat casts on the next
			// tick the caster can afford it rather than after a period
			// nobody asked for.
			if c.X <= 0 || c.X > 0xffff {
				w.entities[i].AutoSpell = 0
			} else {
				w.entities[i].AutoSpell = uint16(c.X)
			}
		case KindCast, KindCastAt:
			if w.beginManualCast(i, c) {
				claim(c.Entity)
			}
		case KindDropCarried:
			// The move itself is drop.go's own function, on KindEquip's own
			// precedent (D-3): the element bounds refusal is that
			// function's, not a guard written out again here.
			w.dropFromContainer(i, int(c.Spell), c.X, c.Y)
		case KindDropWorn:
			// drop.go's own function again, restated over an equipment
			// slot instead of a container element.
			w.dropFromEquipment(i, int(c.Spell), c.X, c.Y)
		case KindReadBook:
			w.readBook(i, int(c.X))
		case KindUsePotion:
			w.usePotion(i, int(c.X))
		case KindUseScroll, KindUseScrollAt:
			if w.beginScroll(i, int(c.Spell), EntityID(uint32(c.X)), c.X, c.Y, c.Kind == KindUseScrollAt) {
				claim(e.ID)
			}
		}
		// A kind this build does not define reaches no arm and is ignored,
		// exactly as an absent entity is. A default that errored would make the
		// stream's meaning depend on which build read it.
	}

	// Unbidden spells are admission candidates, not a pre-command phase. An
	// explicit attack/cast admitted above owns the actor and makes every auto
	// candidate lose without cost or queueing; an explicit move similarly keeps
	// idle Heal from detaching the order. Losing candidates are not retained and
	// are freshly enumerated on a later idle tick.
	w.stepAutoCasts(unionOfClaims(released, commanded))

	// LEGACY NATIVE SESSIONS run their mission script here. Source-clock sessions
	// already ran the entry-phase arm in beginSessionTick and report only after
	// the body in endSessionTick. The historical native policy below is DIV-778.
	// THE MISSION SCRIPT RUNS HERE, BEFORE ANYTHING MOVES, and on one phase of a
	// sixteen-tick cycle rather than on every tick. A tick here is the original's
	// sub-tick; its scheduler runs the whole authored script on phase 6 of the
	// sixteen and reads the outcome on phase 15, so the script is evaluated once
	// per full tick and every condition measures the world as this tick found it.
	//
	// The two phases are distinct, so at most one arm runs in any tick and
	// their order relative to each other is a fact about the cycle rather than
	// about this switch. Both stand after the commands — an order given this
	// tick is visible to nothing until it has moved something — and before
	// the walk below, which is the one placement the pass's own contract fixes.
	// AND THE ENGAGEMENT DECISION RUNS ON THE SAME PHASE, after the script
	// pass. The original places both on slot 6 of the sixteen — the script by
	// its scheduler and the AI by `server+0x04 % 16 == 6` — and nothing
	// published orders the two against each other, so the order here is ours:
	// the script is the authored surface and the decision is the reaction to
	// it, so a group's world changes before it decides in it rather than after.
	// AND THE ACTOR PASS RUNS AFTER THE GROUP LAYER ON THIS SAME PHASE (D-4,
	// 0099): the group layer's clear is what hands a member over to it, so the
	// actor a group has just released acts only after that hand-over has
	// happened, in the same tick. THE WITHDRAWAL TAIL RUNS LAST (1037): its
	// decoded dispatcher position is after every group-order arm, and placing
	// it after actorPass extends that same post-dispatch rule to this build's
	// actor-layer orders. A retreat thus replaces any attack or walk the
	// ordinary decision just produced.
	//
	// It runs whatever the mission's outcome, because the two are separate slots
	// of one tick in the original and neither gates the other.
	//
	// THE TRACE IS TAKEN HERE AND NOWHERE ELSE. Its tick is the one the phase
	// arithmetic above is done on — the pre-increment tick — and the three
	// script values are read after the phase because that is where they mean
	// "what the script just did"; nothing later in a step moves any of them.
	if tr != nil && !w.hasSessionClock {
		tr.Tick = w.tick
	}
	if !w.hasSessionClock {
		switch w.tick % scriptCycle {
		case scriptPassPhase:
			if tr != nil {
				tr.Pass = true
			}
			w.scriptPassObserved(tr, obs)
			w.engagementPassObserved(obs)
			w.actorPass()
			w.withdrawalPassObserved(withdrawals)
		case scriptReportPhase:
			if tr != nil {
				tr.Report = true
			}
			w.scriptReport()
		}
	}
	if tr != nil {
		tr.Won, tr.Lost, tr.Outcome = w.won, w.lost, w.outcome
	}
	w.stepPickupCompletions()
	retreatHolds := w.stepRetreatExecutors()
	w.rechargeStructures()
	w.pruneStructureUses()
	w.takePendingAttacks()

	// One scratch for the whole tick, holding both planes. Per-unit scratch would
	// allocate the map's size once per moving unit; planes on the world would be
	// state beside the digest that UnmarshalBinary would have to rebuild.
	scratch := tickRouteScratch(w)
	defer releaseRouteScratch(scratch)

	// The slice is kept sorted by id, so walking it in order IS walking the
	// entities in ascending id, and the occupancy plane is kept level with that
	// walk: at index i the entities before i hold their post-move cells and those
	// after it their pre-move cells, which is the incremental resolution the
	// contract asks for. The counts move where a unit moves and nowhere else, so
	// what the plane answers is what a walk of the slice would have answered at
	// the moment it was asked. Walk this loop backwards and the convoy stops
	// advancing as a body.
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		e := &w.entities[i]
		if !activity.actorActive(*e) {
			continue
		}
		heldFirstCall := false
		// ONLY AN ALIVE UNIT IS ADVANCED, and the test stands before the target is
		// read: a downed unit and a corpse keep their cells whatever their
		// neighbours do, and neither is asked for a route it could not walk.
		//
		// No command can reach this loop with a target on a unit that is not
		// alive — the arm above refuses to set one, a blow clears the order it
		// fells a unit out of, the constructor clears such a unit's order and the
		// decoder refuses a form carrying one — so this test is the SECOND line
		// and not the first. It is here because the loop is where "who advances"
		// is decided, and a rule stated only at the one entrance that exists today
		// is a rule the next entrance will not have.
		if !e.Alive() {
			continue
		}
		// AND AN OFF-MAP UNIT IS NOT ADVANCED, on the line above's own ground: it
		// holds no cell to step from, contends with nothing and is contended with
		// by nothing, so a route computed for one would be a walk across a map it
		// is not on. Its coordinates stand exactly as they were, which is what
		// instant 17 returns it to.
		if e.OffMap {
			continue
		}
		if i < len(retreatHolds) && retreatHolds[i] && e.HasAttackTarget {
			continue
		}
		// The imported progress3 arm owns its physical step independently of
		// coarse transit, body facing and replacement native orders.
		if w.motionActive(e.ID) {
			if !w.stoneCursed(i) {
				w.advanceSavedMotion(i, scratch)
			}
			continue
		}
		// Stone Curse freezes the body in the exact movement state it held:
		// transit, destination, route and attack approach all resume only after
		// the canonical attached effect expires.
		if w.stoneCursed(i) {
			continue
		}
		if e.Transit == 0 && !w.actorCastBusy(i) && e.AttackPhase == AttackReady && e.AttackCountdown == 0 && w.stepEscortOrder(scratch, i) {
			continue
		}
		// The head pass above is the turn's only advance. A remainder still held
		// here withholds every body/action producer for this actor on this tick.
		if e.Turning() {
			continue
		}
		// A BOOK CAST OWNS THE BODY THROUGH ITS WIND-UP AND NOT THROUGH ITS
		// RECOVERY. The caster stands while it channels: its move order remains
		// attached and resumes afterward, and nothing here advances the body,
		// overwrites the admitted facing or turns the cast again.
		//
		// RECOVERY GATES THE NEXT ACTION AND NOT THE WALK. spec.md's own
		// wording is that release "enters the actor's equipped recovery
		// interval" and that "a later action may begin only after simulation
		// recovery" — a statement about admission, which actorActionBusy makes,
		// and not about the mover. Skipping the walk here as well is what left
		// a commanded mage standing: stepAutoCasts runs before this loop, so an
		// armed row with a target in range re-arms on the tick recovery reaches
		// zero and the body never gets a tick. Measured over 201 ticks, a mage
		// ordered seven cells away with Fire Arrow armed and a hostile in range
		// covered none of them.
		if w.bookCastInFlight(e.ID) || w.scrollInFlight(e.ID) {
			continue
		}
		// A MOVER THAT OWES TRANSIT TICKS IS PAYING FOR THE CELL IT HAS ALREADY
		// TAKEN, and nothing else about it happens: no search, no step, no
		// route, no stall count. The count falls by one and the tick is over for
		// it.
		//
		// The test stands BEFORE the target test and not after it, and that is
		// the placement rather than an ordering: a mover that reaches its target
		// mid-stride clears its order on the tick it lands, and it must still
		// finish the crossing it began — otherwise the drawn body freezes
		// between two cells and the pair stays on the entity for good, which is
		// residue the byte form would carry forever.
		//
		// It also means a mover under a crossing takes no stall count, because a
		// stall counts consecutive ticks on which a NEAR SEARCH found nothing,
		// and a mover that made no attempt has not been held up by anything.
		if e.Transit > 0 {
			e.Transit--
			// The crossing recomputes every cell of the footprint left and of
			// the footprint entered, later than the transit start that read them.
			if e.Stride.Present {
				if from, to, crossed := strideCrossing(e.Stride, int32(e.TransitTotal)-1-int32(e.Transit)); crossed {
					w.recomputeCrossing(*e, from, to)
				}
			}
			continue
		}
		// AN ATTACKER IS AIMED BEFORE IT IS ADVANCED. An attack order carries a
		// destination as well as a victim — the pursuit that lets an order name
		// something out of reach — and this is where the two are kept agreeing.
		//
		// AFTER the crossing above, by that test's own rule: a mover that
		// arrived mid-stride finishes the crossing it began whatever it has
		// been told to do next. BEFORE the target is read, so the destination
		// the rest of this body walks toward is the one this tick's victim
		// position implies rather than the previous tick's.
		//
		// It may CLEAR the target as well as set one — an attacker in reach
		// stands still — so it stands above the test below rather than beside
		// it, and an attacker that has arrived falls through that test in the
		// same tick it arrives.
		if e.HasAttackTarget && !e.PendingOrder.RowAdmitted {
			// A DESTINATION HELD BEHIND A LOADED CYCLE IS NOT WALKED, and nor is
			// a structure use pending behind one. The cycle owns the body until
			// it returns to ready; a withdrawal move written meanwhile waits in
			// the destination fields, and neither the approach nor a step may
			// clear it, aim it at the victim or spend the cycle (advanceAttack
			// cancels a cycle for a moving actor). DIV-1509, DIV-1577.
			if (e.HasTarget || w.usingStructure(e.ID)) && e.AttackPhase != AttackReady {
				if !w.endHeldCycleForWalk(i) {
					continue
				}
				heldFirstCall = true
			}
			if e.HasAttackTarget {
				if ti := w.approach(scratch, i); ti >= 0 && !w.usingStructure(e.ID) {
					w.pursue(scratch, i, ti, heldFirstCall)
					continue
				}
			}
		}
		if w.advanceStructureUse(scratch, i) {
			continue
		}
		if !e.HasTarget {
			continue
		}
		if e.X == e.TargetX && e.Y == e.TargetY {
			// Already there: cleared without a search, by the same rule that
			// clears an arrival. A cleared order leaves no residue — target,
			// stall count and route together — so that two worlds alike in
			// everything logical are alike in every field too.
			w.restAt(scratch, i)
			continue
		}

		sub, serves := w.subGoal(scratch, i)
		if !serves {
			// The stored route does not serve, so one far search: terrain alone,
			// the whole map, the order's real destination, the budget of the
			// mover's owner, and leave to SETTLE for a cell near the goal if it
			// cannot reach it.
			//
			// The decoded override gates the flat budget on two things. One is the
			// goal's own footprint being free on the static plane, and that gate
			// is LIVE rather than discharged: the refusal before any sweep used to
			// establish it, because a target not open under the relation returned
			// without a budget ever being read, and a search that may settle has to
			// sweep exactly that target to leave the labels a substitute is chosen
			// from. So the openness reaches the budget itself, where the decoded
			// store puts it, and a settling search runs the computed form rather
			// than the thousand. That is also what keeps the sweep bounded by the
			// distance ordered.
			//
			// The other is the mover's footprint and owner: a unit larger than one
			// cell takes 5 + D whoever owns it, D being the Chebyshev distance to
			// the goal. A one-cell unit of a HUMAN PARTICIPANT takes the flat form
			// and every other owner's one-cell unit max(5, D>>2) + D, so an AI
			// creature whose goal lies behind a longer detour settles for a nearer
			// cell where a participant's one-cell unit walks the detour.
			// farBudgetFor reads it. DIV-1569, DIV-1570.
			route, ok := w.searchRoute(scratch, i, terrainRelation, noWindow, w.farBudgetFor(i), settleOrdered, e.TargetX, e.TargetY)
			if !ok || len(route) == 0 {
				// The order ends here, in the tick that found it unservable, with
				// no residue and no stall. The far relation is the bounds and the
				// grid, both immutable while a world is advanced, and a unit whose
				// far search fails does not move — so a second far search from the
				// same cell is the same search, and sixteen of them are sixteen
				// identical answers bought at a whole-map sweep each.
				//
				// An attacker's pursuit is refused by it, which is the original's
				// raised route-failure flag: the order is idled or handed to the
				// victim nearest within reach (DIV-1520).
				if w.restAt(scratch, i) {
					w.answerRefusedPursuit(i)
				}
				continue
			}
			w.routes[i] = route
			// THE ORDER TAKES THE CELL THE SEARCH SETTLED ON. For a goal the
			// search reached this is the goal and the write changes nothing; for
			// one it could not, it is the substitute, and from here the order is
			// an order to that cell like any other — the route serves, the
			// arrival clears it, and the sweep that found it runs once.
			//
			// It is a DIVERGENCE and it is here rather than in the search
			// because of what this tree does not have. The engine leaves the
			// order pointing at the cell that was asked for and re-substitutes
			// from wherever the unit then stands, which it can afford because it
			// re-runs that search only every sixteenth dynamic one; the three
			// staleness tests above have no period in them, so an order left
			// pointing at an unreachable cell would buy a whole-map sweep every
			// tick for as long as the walk lasts. The cost of taking it here is
			// that the mover settles for the FIRST substitute rather than the
			// one its later position would have chosen.
			last := route[len(route)-1]
			e.TargetX, e.TargetY = last.x, last.y
			// Every cell of a fresh route is at most its own index of steps from
			// the mover, so this sub-goal is at most four cells off and cannot be
			// outside the window that has just been tested against.
			sub = route[subGoalIndex(len(route))]
		}

		// The near search keeps the scaled rule, and keeps its relation and its
		// window with it. It is aimed at a sub-goal at most four cells off, so a
		// bound shaped like the straight line is the right shape here — and it
		// is the tighter of the two bounds close in, which is the reach the
		// budget stops the flood at before the window does.
		//
		// AND IT MAY SETTLE, which is the one thing here the far search does not
		// do differently: both searches substitute, and they differ in the ring
		// bound alone. It matters because the far search reads TERRAIN and this
		// one reads BODIES — so the sub-goal handed down is routinely a cell the
		// far search could not see was occupied, and a near search that could
		// only answer "no route" to that would count a tick, sixteen times, and
		// drop an order a step aside would have served. What it settles for is
		// the cheapest cell its own wave reached in the first ring around the
		// sub-goal holding one, and the mover walks toward that instead. The
		// substitute is NOT written back anywhere: a sub-goal is derived afresh
		// from the stored route every tick, so nothing carries it and the mover
		// makes its way back to the route by the rule that already returns any
		// off-route step to it.
		//
		// A FLYER'S NEAR SEARCH READS TERRAIN ALONE. It flies through moving and
		// resting units alike, routing only around what its own domain's terrain
		// closes to it, so its step is a straight line and never a detour. Where
		// it may come to REST is the one place occupancy reaches it, and that is
		// asked of the order and not of the step.
		near := unitRelation
		if e.Domain == DomainAir {
			near = terrainRelation
		}
		step, ok := w.searchRoute(scratch, i, near, dynamicWindow, scaledBudget, settleStep, sub.x, sub.y)
		if !ok || len(step) == 0 {
			// The count is the ONLY field written on this path. The position, the
			// three target fields and the whole stored route are not restored —
			// they are never touched, so there is no window in which a
			// half-applied move exists.
			//
			// A pursuit whose near search aims at the victim itself and finds
			// nothing is refused at once: the original's search raises its
			// route-failure flag when the search toward the final destination
			// comes back empty (DIV-1316). Aimed at a waypoint short of it, the
			// stall count runs on. A human participant's order always takes
			// the stall count and the whole-map check, so a friendly body that
			// passes for a few ticks cannot end it.
			if e.Owner != SelfSlot && w.aimsAtDestination(i, sub) && w.attackerRefusable(i) {
				if w.restAt(scratch, i) {
					w.answerRefusedPursuit(i)
				}
				continue
			}
			e.Stall++
			if e.Stall >= stallLimit {
				refused := w.pursuitRefused(scratch, i)
				if w.restAt(scratch, i) && refused {
					w.answerRefusedPursuit(i)
				}
			}
			continue
		}
		if w.advanceStep(scratch, i, step[0], heldFirstCall) != stepTaken {
			continue
		}

		if e.X == e.TargetX && e.Y == e.TargetY {
			w.restAt(scratch, i)
			continue
		}
		w.consume(i)
	}

	// PHASE 3, and the reason it is its own loop and not part of the one above:
	// the move loop resolves entities against an occupancy plane it keeps level
	// with its own walk, and a kill inside that walk would free the victim's
	// cell mid-tick and change what every later id routes through — giving the
	// five-outcome contract above a sixth cause. Here nothing routes and nothing
	// reads the plane, so the two phases share only the entity slice.
	//
	// It reads POST-MOVE positions, which is what makes a victim that stepped
	// away this tick out of reach on it. Ascending id again, and that order is
	// what decides which of two attackers on one victim gets the killing blow —
	// and so which of them draws at all on the tick it dies.
	//
	// A unit that is not alive is not advanced here either, and the test stands
	// before the order is read for the reason it does above: a downed unit and
	// a corpse strike nothing whatever they were told. An OFF-MAP attacker
	// strikes nothing either, on the same line's own ground: it is not on the
	// list this loop reproduces. It keeps its victim while it is away — the
	// removal writes one bit and no order — and resumes the fight if it is
	// returned, unless the victim has itself left the map or become invisible
	// to it, which advanceAttack answers for.
	for i := range w.entities {
		e := &w.entities[i]
		w.settleIdleOrderProgress(i)
		if !activity.actorActive(*e) {
			continue
		}
		if !e.Alive() {
			// A dying unit's order stays frozen (HERO-DYINGTICK-145); the
			// decay pass ends it with the dying window.
			if e.HasAttackTarget && !e.Dying() {
				e.clearAttack()
			}
			continue
		}
		if e.OffMap || w.stoneCursed(i) || !e.HasAttackTarget || e.PendingOrder.Kind != PendingNone && w.pendingLogicalProgress(i) == 0 || e.PendingOrder.RowAdmitted || i < len(retreatHolds) && retreatHolds[i] {
			continue
		}
		if _, casting := w.bookCastIndex(e.ID); casting || e.CastWait != 0 {
			continue
		}
		loaded := e.AttackPhase != AttackReady
		weaponRelease := e.AttackPhase == AttackCasting
		w.advanceAttack(i, obs)
		if weaponRelease && e.AttackPhase != AttackCasting && e.PendingOrder.Kind == PendingScroll {
			if w.cancelScroll(i) {
				e = &w.entities[i]
				e.PendingOrder = PendingOrder{Kind: PendingRelease}
			}
		}
		// Completion is observed separately from the next executor invocation.
		w.observeRetreatCompletion(i)
		w.observePendingOrderCompletion(i)
		// A move held behind the cycle replaces the retained attack order when
		// the cycle ends, so no new cycle loads and the walk starts on the next
		// movement pass. A structure use pending behind it does the same.
		// DIV-1509, DIV-1577.
		if loaded && e.AttackPhase == AttackReady && e.HasAttackTarget && (e.HasTarget || w.usingStructure(e.ID) || e.PendingOrder.Kind != PendingNone) &&
			(e.PendingOrder.Kind == PendingNone || w.pendingLogicalProgress(i) == 0) {
			if e.PendingOrder.Kind != PendingActorCast || !holdsOrderedVictim(*e) {
				e.clearActiveAttack()
			}
		}
		// A member that stood down behind the cycle waits at its own cell, and
		// the end of the cycle ends that wait (standDown).
		if loaded && !e.HasAttackTarget && e.HasTarget && e.X == e.TargetX && e.Y == e.TargetY {
			w.clearOrder(i)
		}
	}

	// Regeneration. It runs here, after every arm above that can change a
	// health and immediately before the decay ladder, because in what is being
	// reconstructed the two are loops of one dispatch and the live-list one —
	// this one — runs first. It is called every sub-tick, exactly as
	// decayPass is below, and phase-gates only itself: there is no dispatch
	// here for it to join.
	if !w.hasSessionClock {
		w.regenPass()
	}

	// PHASE 4: the decay ladder. It stands last because it reads what the whole
	// tick has left — a body felled by the cycle above is on the ladder before
	// this runs — and because removing an entity is the one thing here that
	// changes who the world holds, which nothing earlier in a tick may see
	// happen underneath it.
	// Consume deaths caused by commands, attacks and effects before a zero-dwell
	// flying body can be removed by the ladder. Run the same scan again after
	// the ladder for a downed actor whose decay walk itself crossed below zero;
	// processNewKillCredits deletes each consumed id from this tick's set.
	w.drainControlSpirit()
	w.processNewKillCredits(beforeHP)
	w.decayPass()
	w.processNewKillCredits(beforeHP)

	// Recovery ages after every possible action producer. A count that reaches
	// zero here admits a fresh action on the next tick, never later in this one.
	w.ageCastRecovery(released)
	w.observeAdmissionFans(obs)

	// The area-effect pass follows every actor's update.
	w.settleCostWindow()
	w.syncAreaCosts()

	if w.hasSessionClock {
		w.endSessionTick(tr)
	} else {
		w.tick++
	}
	if w.savedObjects != nil && len(w.savedObjects.BookRoots) != 0 {
		for i := range w.entities {
			w.refreshSavedBookRoots(i)
		}
	}
	w.syncCurrentActionCarriers()
}

// lookAhead is how far along a stored route the near search aims: the sub-goal
// is the route's cell at index min(lookAhead, len-1), so up to four cells along
// it, and the target itself once four or fewer remain.
//
// It is one number in one place, and the consumption rule reads the same one:
// the cells a mover may land on and drop are exactly the cells a sub-goal may be
// chosen from. Were the two allowed to differ, a mover could land on its own
// sub-goal and be asked to search from a cell to itself.
const lookAhead = 3

// subGoalIndex is min(lookAhead, n-1) for a route of n cells.
func subGoalIndex(n int) int {
	if n-1 < lookAhead {
		return n - 1
	}
	return lookAhead
}

// subGoal is the cell entity i's near search aims at, and whether the stored
// route serves at all.
//
// The ways it does not are the whole of when a far search runs: the entity holds
// no route; the route's last cell is not its current target; a flyer may no
// longer rest at its end; or the sub-goal is outside the entity's window.
//
// Nothing here counts ticks. A refresh period would be a second canonical byte
// per unit and a whole-map sweep every so often for every mover, where these
// three fire exactly when the stored route cannot be used.
func (w *World) subGoal(s *routeScratch, i int) (cell, bool) {
	route := w.routes[i]
	if len(route) == 0 {
		return cell{}, false
	}
	e := &w.entities[i]
	last := route[len(route)-1]
	if last.x != e.TargetX || last.y != e.TargetY {
		return cell{}, false
	}
	// A stored route is not retired when a cell of it closes, and the byte form
	// holds one. It is tested here only over the cells the near search may aim
	// at, the first four: a cell closed among them cannot be consumed, so the
	// mover would circle it, and the route is replaced by one far search.
	// A closed cell further along stays in the route until it comes within that
	// reach. DIV-2048.
	for k := 0; k <= subGoalIndex(len(route)); k++ {
		if !w.terrainOpenFootprint(*e, route[k].x, route[k].y) {
			return cell{}, false
		}
	}
	// A further way a stored route stops serving: its last cell is the mover's
	// target, and the mover may no longer come to rest there. restFree answers
	// true for every mover but a flyer, so this is the air domain's test and no
	// other mover's route is ever discarded by it.
	//
	// It is asked AFTER the last-cell test, so only a route still ending at the
	// target pays for it, and BEFORE the window test, so a discarded route costs
	// no window arithmetic. It reads MID-TICK state -- which entities have
	// already resolved -- where the three tests around it are pure world state.
	// The walk is in ascending id exactly as the plane's own updates are, so
	// determinism survives; what it costs is that this is no longer a function of
	// the world alone.
	if !w.restFree(s, i, last.x, last.y) {
		return cell{}, false
	}
	sub := route[subGoalIndex(len(route))]
	win := window{centre: cell{x: e.X, y: e.Y}, half: dynamicWindow}
	if !win.holds(sub.x, sub.y) {
		return cell{}, false
	}
	return sub, true
}

// consume drops the cells of entity i's stored route up to and including the one
// it has just landed on, when that cell is among the first four.
//
// A mover that stepped somewhere else keeps its WHOLE route and makes its way
// back to it — that is the case a "drop the head" rule would get wrong, and it is
// ordinary: a near search routes round a unit standing in the way, and the cell
// it takes need not be on the route at all.
func (w *World) consume(i int) {
	e := &w.entities[i]
	route := w.routes[i]
	n := len(route)
	if n > lookAhead+1 {
		n = lookAhead + 1
	}
	for k := 0; k < n; k++ {
		if route[k].x == e.X && route[k].y == e.Y {
			w.routes[i] = route[k+1:]
			return
		}
	}
}

// attachMoveOrder writes a walk to (x, y) for entity i: the destination, the
// ended fight and the fresh command group of one that a move order and a
// pickup order share.
func (w *World) attachMoveOrder(i int, x, y int32, pickup bool) {
	e := &w.entities[i]
	if !e.HasTarget || x != e.TargetX || y != e.TargetY {
		w.routes[i] = nil
	}
	w.cancelTurnForTargetChange(i, x, y)
	e.TargetX, e.TargetY, e.HasTarget = x, y, true
	// AND IT ENDS THE FIGHT THIS UNIT WAS IN BETWEEN CYCLES. Walking and
	// attacking are ONE state and not two, so an entity holds a
	// destination or a victim and never both — the state byte being
	// reconstructed carries one value for the walk and another for the
	// strike, and the strike's sub-phases run only under the second. The
	// one exception is a loaded cycle, which the move waits behind
	// (AI-CMD-033, MOVE-FORM-036, AI-RETREAT-271, AI-ORDER-039; DIV-1563).
	if !pickup || w.pendingLogicalProgress(i) == 0 && e.AttackPhase == AttackReady {
		e.clearAttackBetweenCycles()
	} else {
		e.clearPendingAttack()
	}
	if pickup {
		e.PendingOrder = PendingOrder{Kind: PendingPickup, RowAdmitted: w.pendingLogicalProgress(i) == 0, X: x, Y: y}
	}
	// A PLAIN ORDER IS A FRESH GROUP OF ONE, and a fresh group's rate
	// term starts at zero — there is no non-group player move in what is
	// being reconstructed, so this is that order's own behaviour rather
	// than a clearing rule of ours. It is a no-op on every entity that
	// has not been in a formation order, which is every entity any
	// schedule written before this story can produce.
	e.clearGroupSpeed()
	// AND NOW IT REALLY IS ONE: before this story the claim above was true
	// only of the rate term — the entity itself stayed in whatever group
	// the map placed it in, or a command group an earlier order left it in,
	// and a guard idle on arrival walked home to its post because it never
	// left the guard's own group at all. commandGroup (group.go) is what
	// closes that gap: it releases this one member from whatever group it
	// stood in and puts it, alone, into a fresh command group held at the
	// move order — which has no walk home.
	w.commandGroup([]int{i}, orderMove, cell{x: x, y: y})
	w.syncSavedDestination(i)
}

// clearOrder drops everything entity i's order consists of: the target, the
// stall count that only means something while it holds one, and the stored
// route.
//
// The three go together at every site rather than one at a time, because "an
// entity holding no target holds no route and a zero count" is then true by
// construction. It is not tidiness: both a nonzero count and a stored route on
// an entity with no target are states the byte form REFUSES, so a site that
// forgot one would build a world this package can marshal and then not read back.
func (w *World) clearOrder(i int) {
	w.noteActorMotionOrder(w.entities[i].ID)
	w.entities[i].clearTarget()
	w.routes[i] = nil
}

// restAt ends entity i's order and puts it into its layer's plane if ending the
// order is what put it there. It reports whether the order actually ended.
//
// It is clearOrder plus one consequence, and the consequence exists because a
// flyer is counted only while it holds no target: the tick that ends its order
// is the tick it becomes visible to its peers, and it must become visible BEFORE
// the next entity is resolved or a peer could come to rest on top of it.
//
// The test is "was it counted before, and is it counted now" rather than "is it
// a flyer", so a ground mover — counted throughout — is not counted twice, and
// nothing here has to know which domains the seed treats specially. That is the
// same predicate the seed and the self-presence term ask.
//
// A MOVING FLYER IS THE TRANSITION THAT NEEDS ADMISSION. It is absent from the
// plane while it holds a target and becomes counted when that target is cleared.
// The complete-footprint enterable predicate is therefore asked BEFORE either
// the order or the plane changes. Refusal leaves both movement and pursuit
// intact, so approach can keep closing instead of stranding an uncounted flyer
// where it first came into attack range.
//
// The four clearing sites inside the move loop call this; clearFelled keeps the
// plain call, because a felled unit is not counted anyway — and that is the whole
// of the reason now that it has a caller on both sides of the scratch's life,
// phase 1 running before it exists and phase 3 after it stops being read.
func (w *World) restAt(s *routeScratch, i int) bool {
	before := counted(w.entities[i])
	after := w.entities[i]
	after.clearTarget()
	if !before && counted(after) && !w.enterable(s, i, after.X, after.Y) {
		return false
	}
	w.clearOrder(i)
	if before || !counted(w.entities[i]) {
		return true
	}
	e := w.entities[i]
	s.addFootprint(w, e.Domain.layer(), e.TokenSize, e.X, e.Y, 1)
	return true
}

// clearFelledActions clears native actions without advancing a death stage.
// Group membership remains until teardown, including after a new death.
func (w *World) clearFelledActions(i int) {
	w.entities[i].Retreat = RetreatContinuation{}
	w.cancelStructureUse(w.entities[i].ID)
	w.invalidateActorMotion(w.entities[i].ID, "native death supersedes original movement")
	w.entities[i].clearTurn()
	w.clearOrder(i)
	// And the crossing with it. It is dropped HERE and at no other site,
	// because arriving does not end a crossing and neither does losing an
	// order: only leaving the living does. A pair left on a unit the loop
	// no longer advances would never fall to zero, and the byte form
	// refuses one on a unit that is not alive for exactly that reason.
	w.entities[i].clearTransit()
	// Clear the native movement rate without changing the saved Group's byte.
	w.entities[i].clearGroupSpeed()
	// And the ATTACK ORDER, because a felled unit takes no order of either
	// kind and holds neither. It goes here rather than beside the walk's own
	// clearing for the same reason the crossing does: the three are one
	// statement about leaving the living, made once.
	w.entities[i].clearAttack()
	// And the book action itself. stepBookCasts runs before commands and
	// physical attacks, so either can fell a caster after its record has
	// already advanced for this tick. Leaving the record until the next tick
	// makes the dead actor's action part of the intervening hash and save.
	w.clearActorCast(i)
	// And THE PATROL, on the group term's own ground: a felled actor holds no
	// ring and runs no arm, so a state naming one is residue exactly as a stale
	// victim is. clearPatrol is the entity's own half of leaving a patrol, so
	// the state, the ring and the leg cannot come apart into a partial clear
	// here any more than they can at the constructor.
	w.entities[i].clearPatrol()
	// AND THE ESCORT ORDER WITH IT (0166 D-13), on the same rule and in
	// the same sentence: a felled actor holds no order of either layer,
	// so a defend or a follow state on a body is residue exactly as a
	// patrol state is. patrolFault refuses both.
	w.clearEscort(i)
}

// clearFelled performs the one death transition after combat, command,
// equipment or derived-health application makes entity i not alive. Alive is
// the shared predicate: health 0 of 0 names an actor without a health system and
// is not a death, while health 0 of a positive maximum is downed and is. The
// transition clears every action representation before any save or hash can
// observe residue.
func (w *World) clearFelled(i int) {
	if !w.entities[i].Alive() {
		w.clearFelledActions(i)
		// AND THE DEATH ITSELF, which is the one thing here that is not a
		// clearing: the stage, the dwell it owes, and the halved defence.
		//
		// It is HERE rather than in the pass at the end of a tick, and the
		// placement is load-bearing. This function has callers in phase 1 as
		// well as phase 3, and the occupancy plane is seeded between the two —
		// so a unit felled by a command and given its stage only at the end of
		// the tick would spend that tick neither living nor dwelling, and the
		// seed would read it as ground nobody holds.
		//
		// The stage guard is what makes it fire ONCE. Finishing blows may reach a
		// body through health -9, and none may halve its defence twice.
		if w.entities[i].Decay == DecayNone {
			if m := w.motionFor(w.entities[i].ID); m != nil {
				cell := uint16(uint8(w.entities[i].X)) | uint16(uint8(w.entities[i].Y))<<8
				m.Position.Cell, m.Position.PackedCell = cell, cell
			}
			w.entities[i].Decay = DecayFallen
			w.entities[i].Dwell = dwellOf(w.entities[i])
			// An ARITHMETIC SHIFT and not a division, which is the operation
			// being reproduced rather than the one that reads more naturally.
			// The two agree on every non-negative defence, and a defence is
			// never negative on any path this tree builds, so what the shift
			// buys is being right at the boundary rather than being right about
			// the numbers that occur.
			w.entities[i].setCurrentDefence(w.entities[i].Defence >> 1)

		}
		// A zero-dwell body that crosses the boundary in a command or blow
		// transfers immediately, preserving resolution order when several die
		// in one tick. A body still playing its fall waits for decayPass.
		if e := &w.entities[i]; e.Decay == DecayFallen && e.Dwell == 0 && !e.OrdinaryTargetable() {
			if stage := decayStageFor(e.HP); stage > DecayFallen && stage <= decayLast {
				if w.dropTerminalLoot(i) {
					w.entities[i].Decay = stage
					w.detachSavedMember(w.entities[i].ID)
				}
			}
		}
		w.clearInvalidTargetReferences(w.entities[i].ID)
	}
}

// Terminal teardown transfers holdings once before advancing DecayFallen.
// Failure retains loot and delays stage advancement. DIV-1441.
func (w *World) dropTerminalLoot(i int) bool {
	x, y, validCell := w.terminalLootCell(i)
	if !validCell && w.hasTerminalLoot(i) {
		return false
	}
	if w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		if !n.sourceTerminalLoot(i, x, y, validCell) || !n.savedMutationValid() {
			return false
		}
		*w = n
		return true
	}
	if w.savedObjects != nil && w.actorHasSavedItems(i) {
		n := w.sourceMutationCopy(i)
		if !n.terminalSavedNative(i, x, y, validCell) || !n.savedMutationValid() {
			return false
		}
		*w = n
		return true
	}
	before := w.beginLoadMutation(i)
	if w.entities[i].SuppressCorpseLoot {
		w.carried[i] = nil
		w.equipment[i] = [EquipSlots]ItemInstance{}
		syncWeaponItem(&w.entities[i], ItemInstance{})
		if validCell {
			if gold := w.deathGold(i); gold != 0 {
				w.pourSack(x, y, gold, nil)
			}
		}
	} else if validCell {
		gold := w.deathGold(i)
		if gold != 0 || w.holdsSomething(i) {
			items := expandItems(w.carried[i])
			// Original drop order: carried items, slot 2, slot 1, then slots
			// 3..12 ascending.
			if item := w.equipment[i][slotSecond]; !item.Empty() {
				items = append(items, item.Clone())
				w.equipment[i][slotSecond] = ItemInstance{}
			}
			if item := w.equipment[i][slotWeapon]; !item.Empty() && !item.innateWeapon() {
				items = append(items, item.Clone())
				w.equipment[i][slotWeapon] = ItemInstance{}
			}
			for slot := 2; slot < EquipSlots; slot++ {
				if item := w.equipment[i][slot]; !item.Empty() {
					items = append(items, item.Clone())
					w.equipment[i][slot] = ItemInstance{}
				}
			}
			if gold != 0 || len(items) != 0 {
				w.pourSack(x, y, gold, items)
			}
			if w.equipment[i][slotWeapon].Empty() {
				syncWeaponItem(&w.entities[i], ItemInstance{})
			}
			w.carried[i] = nil
		}
	}
	// Suppressed template contents are deleted even without a sack destination.
	if len(w.carried[i]) == 0 && w.entities[i].ActorLoad.Present {
		w.entities[i].ActorLoad.ContainerPresent = true
		w.entities[i].ActorLoad.InsertIndex = 10000
	}
	return w.finishLoadMutation(i, before)
}

// PrepareAuthoredBody applies the one death-time combat adjustment to a body
// that was authored non-positive in a map. It deliberately does not run
// clearFelled: map stock and worn equipment are authored starting state, not
// possessions dropped by a death that happened during this world.
//
// The decay stage is also the once-only marker. NewWorld therefore preserves
// this presentation instead of treating the body as a newly felled actor, and
// repeated preparation cannot halve defence a second time.
func PrepareAuthoredBody(e *Entity) {
	if e == nil || e.MaxHP <= 0 || e.HP > 0 || e.Decay != DecayNone {
		return
	}
	e.Decay = DecayFallen
	e.Dwell = dwellOf(*e)
	e.setCurrentDefence(e.Defence >> 1)
}

// restoreAfterHealthGain is the dedicated fallen-to-living transition shared
// by Heal and script property writes. The positive decay-stage guard is the
// once-only marker: the first crossing clears it and restores the death-time
// arithmetic shift; a later heal of that living actor cannot double defence.
// It never calls clearFelled, so equipment retained by an authored body stays
// on that actor.
func (w *World) restoreAfterHealthGain(i int, before int32) {
	e := &w.entities[i]
	if before > 0 || e.HP <= 0 || e.Decay == DecayNone {
		return
	}
	e.clearDecay()
	e.setCurrentDefence(e.Defence << 1)
}

// deathGold performs the Units-row death roll (HERO-KILL-027). Only type ids
// above 0x40 are eligible. The chance comparison and both bounded draws are
// inclusive at the upper end because rng.uniform is U[0,n]. Conversion to
// uint32 preserves the purse and sack arithmetic's wrapping width.
func (w *World) deathGold(i int) uint32 {
	e := w.entities[i]
	if e.TypeID <= 0x40 {
		return 0
	}
	if e.GoldChance <= w.rng.uniform(100) {
		return 0
	}
	return uint32(e.TreasureMin + w.rng.uniform(e.TreasureMax))
}

// clearTarget drops e's target and the stall count that goes with it. It is the
// entity's own half of clearing an order; the route is the world's, and
// clearOrder is what does both.
func (e *Entity) clearTarget() {
	e.TargetX, e.TargetY, e.HasTarget, e.Stall = 0, 0, false, 0
	e.EscortOrder = escortOrderNone
}

// searchRoute is the route the world's own mode chooses from the entity at index
// self to (tx, ty).
//
// This is the ONE place the mode is read. It steers which search runs and
// nothing else: the resolution order, the cost model, the neighbour set, the
// clearing rules and the stall count are the same under both, so a world's mode
// changes what a route IS and never what a tick does with one. A second place
// that branched on it would be a second contract to keep in agreement with this
// one.
//
// It is read from the world, which is where the byte form and the digest carry
// it. A parameter, a build tag or a package variable would each be a
// behaviour-steering value living outside the encoding, and a replay against it
// would diverge for a reason no recorded state could explain.
//
// The RELATION, the WINDOW and the budget RULE are parameters, and together they
// are what makes a search far or near. Both searches read either relation and
// honour either window, so the mode stays orthogonal to that choice — which is
// why there are two searches here and not four.
//
// The rule reaches the wave alone. The optimised search runs its heap until
// nothing is left to pop, so it has no generation count to bound and MUST NOT
// gain one; what bounds it is the window, and with noWindow it is the whole map
// — which is the far search's contract in that mode.
func (w *World) searchRoute(s *routeScratch, self int, r relation, half int32, rule budgetRule, settle settleRule, tx, ty int32) ([]cell, bool) {
	if w.mode == ModeOptimised {
		return w.optimisedRoute(s, self, r, half, settle, tx, ty)
	}
	return w.canonicalRoute(s, self, r, half, rule, settle, tx, ty)
}

// Route is the cells entity id still intends to walk, in the order it will walk
// them, ending at the cell its order has settled on — empty for an entity this
// world does not hold, one holding no order, and one whose order has not been
// through an advance yet.
//
// IT IS A READ AND IT IS NOT A SEARCH. The route a mover walks is already state
// here: the far search stores one, the advance consumes its walked head, and the
// three staleness tests replace it when it stops serving. So the question "where
// is this unit going" is answered by reading what the tick already computed, not
// by running a second search beside it — which is why there is no cache in front
// of this call and nothing for one to hold. A caller asking every frame gets the
// same slice-worth of copying and no wave at all.
//
// IT COPIES. The stored route is canonical state that the byte form carries and
// the digest covers, so handing the slice out would give a caller a write into a
// world's hashed fields through a query. The copy is into a plain [][2]int32 for
// the same reason a route is a slice of positions: the value a caller holds
// names no type of this package and still reaches into no world.
//
// It reads the mover's own current cell nowhere and does not prepend it. The
// route is what is LEFT to walk, and where the unit standing at its head is is a
// fact the caller already holds; a query that answered both would be the one
// place the two could disagree.
func (w *World) Route(id EntityID) [][2]int32 {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return nil
	}
	route := w.routes[i]
	if len(route) == 0 {
		return nil
	}
	out := make([][2]int32, len(route))
	for k, c := range route {
		out[k] = [2]int32{c.x, c.y}
	}
	return out
}

// stepRate is THE ONE PLACE a world's own bytes become the movement law's
// arguments: the rate e moves at across the transit from `from` to `to`, and
// how many ticks that transit runs for.
//
// It exists because the number is now read by something other than the advance.
// A second composition beside this one would agree on the day it was written and
// diverge at the first clamp, substitute or bounds test that was changed in one
// and not the other — and a debug readout that disagrees with the mover it
// describes is worse than none, because it is believed over the game.
//
// THE TWO PLANES ARE READ HERE, at the cell left and the cell taken, and
// this is the only site that reads them for a rate.
//
// A transit with an endpoint OFF THE MAP composes its rate from four zeros
// instead, which is what the advance passed before either plane existed. A
// search may legitimately begin from a cell outside the bounds, so the case is
// reachable; the planes describe no such cell, and taking zero for one of them
// alone would not be neutral — a zero cost is a level and not a difference, so it
// would halve the mean and give such a transit a rate no earlier build gave it.
// With nothing to compose from, the law's own substitute is the answer.
//
// moverSpeed AND NOT e.Speed: the group term replaces the class speed at BOTH of
// the law's arms, because the two arms read the same two sources in the same
// order and differ only in what they do with the number afterwards.
//
// IT IS NOT THE WHOLE PUBLISHED LAW, and the term it does not carry is
// named here because this is the site a reader reaches when asking what the
// number is made of: the MULTIPLIER'S SOURCE. rateOf applies it at the shipped default, written
// as a constant; the original reads it per map from `data/map.reg [Path
// Finding]`, and this tree has no reader for that key. Every shipped map
// carries the default, so the arithmetic is right today and a CUSTOMISED map
// would be rated as though it had not customised it.
//
// It is not folded into the result to make it look complete.
//
// The cost accessor's write-back is carried by readCost: a ground mover's read
// of a layered cell divides the stored byte by four and keeps the quotient,
// and commit says whether this call keeps it. Only the advance commits.
func (w *World) stepRate(e Entity, from, to cell, commit bool) (rate, transit int32) {
	e = w.groupRateEntity(e)
	var costSrc, costDst, hSrc, hDst uint8
	if w.describes(from) && w.describes(to) {
		// Only the ground arm reads the cost plane, source cell first. A read
		// of a layered cell divides its stored byte, and commit keeps that.
		if e.Domain == DomainGround {
			costSrc, costDst = w.readCost(from, commit), w.readCost(to, commit)
		} else {
			costSrc, costDst = w.costAt(from), w.costAt(to)
		}
		hSrc, hDst = w.heightAt(from), w.heightAt(to)
	}
	v := rateOf(e.Domain, moverSpeed(e), costSrc, costDst, hSrc, hDst)
	return v, transitOf(v, from.x != to.x && from.y != to.y)
}

// StepRate is what one step onto (x, y) would cost the entity id stands for:
// the movement law's rate, the ticks that transit takes, and whether the two
// cells are a single step apart at all.
//
// IT IS A READ. It composes through the very function the advance rates a
// mover with, holds nothing, writes nothing, and returns plain integers
// naming no type of this package — so a caller cannot reach a world
// through it and the byte form, its version and the digest are untouched by
// its existence.
//
// FOUR INPUTS ANSWER "NO VALUE", and the middle two are the point. An id
// this world does not hold is Route's own rule. A mover that is NOT ALIVE
// and a mover of ZERO EFFECTIVE SPEED are the advance's own gates — the
// move loop skips the first and `rated` skips the second — and rateOf is
// TOTAL where the advance is not: handed a zero speed it clamps to the rate
// floor and transitOf turns that into the grid's longest transit. That pair
// of numbers is plausible, is wrong, and is reachable only by asking a
// question no tick asks. A destination EQUAL TO THE SOURCE is refused for
// its own reason: a transit from a cell to itself is not a step, and no
// advance can produce one.
//
// The refusals live here rather than in the caller so that a second caller
// inherits them instead of having to remember them.
//
// ADJACENCY IS REPORTED AND NOT LEFT TO THE CALLER, because only this side knows
// which cell the law was actually evaluated FROM. A front-end deriving it from
// its own copy of the mover's position could mark — or fail to mark — a number
// that was computed about a different pair. It is false for every pair further
// than one cell; the rate and transit are still the law's answer for that ordered
// pair, and are a single hypothetical step rather than the cost of a path.
func (w *World) StepRate(id EntityID, x, y int32) (rate, transit int32, adjacent, ok bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return 0, 0, false, false
	}
	e := w.groupRateEntity(w.entities[i])
	if !e.Alive() || !rated(e) {
		return 0, 0, false, false
	}
	from, to := cell{x: e.X, y: e.Y}, cell{x: x, y: y}
	if from == to {
		return 0, 0, false, false
	}
	r, t := w.stepRate(e, from, to, false)
	dx, dy := x-e.X, y-e.Y
	return r, t, dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1, true
}

// indexOfEntity is a binary search for id over a slice kept sorted by ascending
// id, returning -1 when the slice holds no such entity. It is a search and not a
// map lookup on purpose: Go randomises map iteration order per process, so no
// map appears on any path that touches a world.
func indexOfEntity(ents []Entity, id EntityID) int {
	i := sort.Search(len(ents), func(i int) bool { return ents[i].ID >= id })
	if i < len(ents) && ents[i].ID == id {
		return i
	}
	return -1
}

// The decay pass's period and its phase within that period.
//
// ONE FULL TICK IS scriptCycle SUB-TICKS, and the walk moves a body's health on
// every SECOND full tick — so the period here is that constant taken twice
// rather than a thirty-two written down beside it, and the two cannot come to
// disagree about how long a full tick is.
//
// The phase is the slot of the sixteen this pass is dispatched on. WHICH of the
// two full ticks in a period carries the walk is not published — the filter is a
// low bit on a counter this package does not model — so 12 rather than
// 12+scriptCycle is ours, and the choice can only move the tick index a rung is
// reached on, never the ladder itself.
const (
	decayCycle = 2 * scriptCycle
	decayPhase = 12
)

// decayPass advances every body on the ladder by one tick, and removes the ones
// that have reached the bottom of it.
//
// It runs at the END of an advance, after the cycle, so it sees post-move,
// post-blow positions and health and so a unit felled anywhere in this tick is
// already at its first stage when this runs. It is its own loop for the reason
// the cycle is: nothing in it routes and nothing reads the occupancy plane,
// which by here is no longer read at all.
//
// Per body, in ascending id:
//
//   - a DWELL still owed falls by one, and nothing else happens. The body holds
//     its cells for exactly as long as this lasts;
//   - the tick the dwell reaches zero is the TEARDOWN. A mover of a non-ground
//     domain is pinned below the ladder's last rung there, which is what makes
//     the flying classes leave no body at all: they play their fall, hold it for
//     the dwell, and are gone;
//   - from then on the WALK takes one health on one tick of the period, and the
//     STAGE is read off health on every tick — so a body overshot by its killing
//     blow lands on the rung its health names rather than climbing to it.
//
// Ascending id costs nothing here, no body's decay reading another's, and it is
// kept anyway so that the one walk order in this file is the walk order in every
// other.
func (w *World) decayPass() {
	if !w.hasSessionClock {
		w.decayHeldDead(w.tick%decayCycle == decayPhase)
	}
	var gone []EntityID
	for i := range w.entities {
		e := &w.entities[i]
		beforeHP, beforeStage := e.HP, e.Decay
		if e.Decay == DecayNone {
			continue
		}
		// Already torn-down bodies belong to the pre-body phase-12 loop in
		// source-clock sessions; their ladder is not an actor-body action.
		if w.hasSessionClock && e.Decay >= DecayBones {
			continue
		}
		wasTargetable := e.OrdinaryTargetable()
		if e.Dwell > 0 {
			e.Dwell--
			if e.Dwell > 0 {
				continue
			}
			e.clearAttack()
			// The teardown, on the tick the dwell reaches zero and not on the
			// one after it — so a dwell of one is one tick of minimum occupancy.
			// A terminal body then releases the cell; a restorative target keeps
			// it through counted until revival or the finished-body floor.
			if e.Domain != DomainGround {
				e.setCurrentHealth(noCorpseHP)
			}
		}
		// The walk. It saturates rather than wrapping: the floor is the
		// representation's and not the ladder's, so a health already at it is a
		// state this leaves alone rather than one it turns back into a living
		// unit.
		if !w.hasSessionClock && w.tick%decayCycle == decayPhase && e.HP < 0 && e.HP > minHP {
			e.setCurrentHealth(e.HP - 1)
		}
		if e.Decay == DecayFallen && e.Dwell == 0 && !e.OrdinaryTargetable() {
			if !w.dropTerminalLoot(i) {
				continue
			}
			e = &w.entities[i]
			// HERO-DEATH-026: teardown itself calls the dead ladder once,
			// independently of the ordinary phase-12 dead-list walk.
			if w.hasSessionClock && w.fullTick&1 == 0 {
				decaySessionHealth(e)
			}
		}
		if s := decayStageFor(e.HP); s > decayLast {
			// A downed body whose walk crosses below zero and leaves in the same
			// pass is never seen by the kill scan after the ladder.
			if beforeHP >= 0 {
				w.recordDiaryKill(i)
			}
			if e.Decay < DecayBones {
				w.releaseAtTeardown(i)
			}
			gone = append(gone, e.ID)
		} else if s > e.Decay {
			// The stage never falls, and this test is what says so rather than a
			// comment: health only falls and the ladder is monotone in it, so
			// this can only ever be an advance — and written this way a body
			// whose health has not reached the first threshold stays where the
			// death put it instead of being rewritten to the same value.
			teardown := e.Decay < DecayBones && s >= DecayBones
			e.Decay = s
			if teardown {
				w.releaseAtTeardown(i)
				w.detachSavedMember(e.ID)
			}
		}
		if wasTargetable && !e.OrdinaryTargetable() {
			w.clearInvalidTargetReferences(e.ID)
		}
		w.syncOriginalDeadState(*e, beforeHP, beforeStage)
	}
	if len(gone) > 0 {
		w.remove(gone)
	}
}

// remove takes the named entities out of the world: their records, their
// routes, their containers, and every attack order a survivor holds on one of
// them.
//
// It is ONE COMPACTION after the pass rather than a deletion where each body is
// found, because deleting in place invalidates the index the pass is walking —
// the kind of defect that survives every test removing one entity and fails only
// when two go at once.
//
// THE FOUR SLICES ARE REBUILT TOGETHER, in the one loop, because they are
// parallel by construction and every reader of one indexes the others with
// the same i. carried joins entities and routes here rather than being swept
// in a second pass afterwards: a separate walk could come to disagree with
// this one about which entities survived, which is exactly how the two came
// apart before this fix — carried stayed at its old length while entities
// and routes shrank, so an index past a later removal read another entity's
// container, and Stock() (carry.go), which pairs w.carried[i] with
// w.entities[i], could run past the end of the shorter slice. equipment
// joins them on the identical ground (0124 T2): it is parallel to entities
// exactly as carried is, and a walk that swept it separately could come to
// disagree with this one about which entities survived in exactly the way
// carried once did. Ascending id survives every one of the four, being a
// subsequence of an ascending one.
//
// The order sweep is the constructor's second pass done again, for the reason it
// exists there: an order naming an entity the world does not hold is a shape the
// byte form refuses, so a tick that produced one would build a world this package
// can marshal and then not read back. It runs only when something was removed.
//
// NOTHING ELSE IS SWEPT. The group word and the owner slot are membership rather
// than references — the count a script asks for is a count of the living members
// of a group whose dead are still in it — and a script's own reference resolves
// by id every time it is read, so a body that has finished decaying is measured
// by nothing, exactly as a unit the world never held is. purses is by roster
// slot and not by entity, so a removal cannot shrink it either.
func (w *World) remove(gone []EntityID) bool {
	complete := true
	if w.savedObjects != nil {
		gone = slices.Clone(gone)
		gone = slices.DeleteFunc(gone, func(id EntityID) bool {
			if w.retireRemovedActorObjects(id) {
				return false
			}
			complete = false
			return true
		})
	}
	for _, id := range gone {
		w.unlinkActorTraversal(id)
		w.detachSavedProjectileTargets(id)
		w.invalidateActorMotion(id, "native actor removal supersedes original movement")
		w.detachTerminalRegistry(id, true)
		w.detachSavedMember(id)
		if w.savedGroups != nil {
			w.savedGroups.Orders = slices.DeleteFunc(w.savedGroups.Orders, func(o SavedActorOrder) bool { return o.Entity == id })
			for i := range w.savedGroups.Orders {
				o := &w.savedGroups.Orders[i]
				if o.EscortBound && o.EscortTarget == id {
					// Retain the source key; its materialized binding expired.
					o.EscortTarget, o.EscortBound = 0, false
				}
			}
		}
	}
	keep := make([]Entity, 0, len(w.entities))
	routes := make([][]cell, 0, len(w.routes))
	carried := make([][]ItemStack, 0, len(w.carried))
	equipment := make([][EquipSlots]ItemInstance, 0, len(w.equipment))
	for i := range w.entities {
		if containsID(gone, w.entities[i].ID) {
			w.entityIDFloor = max(w.entityIDFloor, uint64(w.entities[i].ID)+1)
			w.retireOriginalDead(w.entities[i].ID)
			w.retainRemovedNativeBasis(w.entities[i])
			continue
		}
		keep = append(keep, w.entities[i])
		routes = append(routes, w.routes[i])
		carried = append(carried, w.carried[i])
		equipment = append(equipment, w.equipment[i])
	}
	w.entities, w.routes, w.carried, w.equipment = keep, routes, carried, equipment
	for i := range w.entities {
		if w.entities[i].HasPendingAttackTarget && w.entities[i].PendingAttackTargetKind == AttackTargetUnit && indexOfEntity(w.entities, w.entities[i].PendingAttackTarget) < 0 {
			w.entities[i].clearPendingAttack()
		}
		if w.entities[i].HasAttackTarget && w.entities[i].AttackTargetKind == AttackTargetUnit &&
			indexOfEntity(w.entities, w.entities[i].AttackTarget) < 0 {
			w.entities[i].clearActiveAttack()
		}
		// THE KILL CREDIT IS THE OTHER REFERENCE INTO THIS SLICE, and it was
		// missing here (owner). A credit names the entity that dealt the killing
		// damage, and that entity's own body decays away on the ordinary ladder
		// like any other: kill a unit, let its killer fall and finish decaying,
		// and a survivor holds a credit pointing at nothing.
		//
		// Nothing at run time notices — processKillCredit resolves the source by
		// id and returns when it is absent — so the world plays on and the
		// ENCODER writes the dangling pointer out. Five of the owner's seventeen
		// current-format saves were unreadable for this.
		//
		// Normalised rather than refused, which is the constructor's own
		// choice for the same shape, reached through the same clear.
		if w.entities[i].HasKillCredit &&
			indexOfEntity(w.entities, w.entities[i].KillCreditSource) < 0 {
			w.entities[i].clearKillCredit()
		}
	}
	return complete
}

// containsID is a linear search over a removal list, which is short: a tick
// removes the bodies whose walk crossed the last threshold on it, and a body
// crosses it once.
func containsID(ids []EntityID, id EntityID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// stepOutcome is what advanceStep did with a mover's next cell.
type stepOutcome uint8

const (
	stepTurned stepOutcome = iota
	stepBlocked
	stepTaken
)

// advanceStep turns entity i toward next or steps it there: the turn when its
// facing must change first, nothing when the cell is held, and otherwise the
// step with its transit, its plane record and the end of its stall.
func (w *World) advanceStep(scratch *routeScratch, i int, next cell, heldFirstCall bool) stepOutcome {
	e := &w.entities[i]
	// The move and the index's own record of it are one statement's worth of
	// work: an advance that told the plane nothing would leave the cell just
	// vacated reading as held and the cell just taken as free, and the next
	// unit resolved would route straight onto an occupied cell.
	from := cell{x: e.X, y: e.Y}
	facingBefore := e.Facing
	if w.turnToward(i, next.x-from.x, next.y-from.y) {
		return stepTurned
	}
	// The walk's first call after a held cycle steps in that call only when
	// the facing already equals the direction to the first node; otherwise
	// it turns and the step comes at the next call (MOVE-090).
	if heldFirstCall && e.Facing != facingBefore {
		return stepTurned
	}
	w.requestFootprintCasts(i, next.x, next.y)
	if e.Domain != DomainAir && !w.occupancyOpenFootprint(scratch, i, next.x, next.y) {
		return stepBlocked
	}
	w.invalidateActorMotion(e.ID, "native cell step supersedes original movement")
	e.X, e.Y = next.x, next.y
	e.replaceDrawnTurn()
	e.clearStride()
	// A MOVER FACES THE CELL IT STEPPED TO, written from the step's own delta
	// — the cell taken less the cell left — and here rather than anywhere
	// else, because this is the one site a cell changes at. So a mover that is
	// blocked, that gives up or that arrives keeps the facing its last step
	// left, by there being no assignment on any of those paths rather than by
	// a rule.
	//
	// It is written on the crossing's FIRST tick, the tick the cell is
	// committed, and the ticks that pay for the crossing leave it alone —
	// which is what keeps a rated mover pointing along the stride it is
	// drawn sliding down for the whole of it.
	//
	// The engine derives the same facing from the same NEXT CELL, and the one
	// thing it does with it that this does not is WAIT: a step there is taken
	// only once the two facing bytes agree, and an arc above one direction
	// costs ticks and destroys the route. Both are named divergences, and
	// neither changes the facing this arrives at. THE RATE IS TAKEN HERE,
	// once, from the two cells this step joins, and held until the next one
	// — so "once per cell transit" is a position in this function rather
	// than a rule someone keeps elsewhere. WHAT it is composed of is
	// stepRate's, and the composition lives there rather than here so that the
	// read which reports this number to a front-end cannot come to compose it
	// differently.
	//
	// The transit's own tick is its FIRST, so the owed count is one short of
	// the length and the cadence is exactly that length of ticks per cell.
	// Only a rated mover writes either: an unrated one owes nothing, and keeps
	// the cell-a-tick cadence every mover here had before — and that gate is
	// why stepRate's OTHER caller must refuse an unrated mover rather than let
	// the law's total arithmetic answer for one.
	if rated(w.groupRateEntity(*e)) {
		rate, t := w.stepRate(*e, from, cell{x: e.X, y: e.Y}, true)
		e.TransitTotal = uint16(t)
		e.Transit = uint16(t - 1)
		e.retainStride(from, rate)
	}
	e.startAction(w.tick, max(1, int64(e.TransitTotal)))
	// Only a mover its plane counts WHILE IT MOVES tells the plane anything.
	// A flyer under orders is counted nowhere, so it commits its position and
	// touches no count — which is what lets two of them cross one cell.
	if w.savedMotion != nil {
		scratch.occupy(w)
	} else if counted(*e) {
		scratch.moved(w, *e, from, cell{x: e.X, y: e.Y})
	}
	// Any advance ends the stall, so the count measures CONSECUTIVE ticks
	// held up and not ticks held up in total.
	e.Stall = 0
	return stepTaken
}
