package sim

import "sort"

// THE ENGAGEMENT DECISION: how a unit comes to be fighting something nobody told
// it to fight.
//
// THE GROUP IS WHAT FIGHTS, and that is the fact this file is shaped around
// rather than a way of organising it. The per-actor state machine is the layer a
// reader meets first and it governs about one creature in a hundred and fifty:
// over the shipped corpus the load walk leaves 8085 of 8094 placements under one
// group order and 9 under another, and none at all under the value that would
// hand a unit to its own state. Restricted to the creatures that can come after
// the player, the group rule takes 7293 of 7571 and the per-actor rule 52
// (AI-CENSUS-046, AI-CENSUS-047). A build that started at the per-actor machine
// would have built the layer almost nothing runs.
//
// The decision is taken once per full tick and is NOT sticky in any part: the
// candidate list is rebuilt from nothing, every member's target is rewritten from
// nothing, and no field anywhere remembers last time (AI-SCORE-069). It can
// afford that because the order it writes is executed by a faster clock — the
// walk and the blow run every tick, the decision every sixteenth (AI-CLOCK-080).
//
// THE ARM HAS THREE OUTCOMES, and picking between them on every decision is
// this file's whole job. A member that scores something is engaged, under
// either stance, exactly as before. A member that scores nothing is RELEASED
// under the guard stance: it ends the tick holding no victim, standing where
// it stopped and facing where it faced. A member that scores nothing under
// stand ground is the third outcome, and the arm leaves it exactly as it
// was.
//
// THE RELEASE NO LONGER LEAVES NOTHING IN THE ORDER'S PLACE, since 0106: a
// released member off its own post is walked home in the same tick, right
// after the release that freed it (walkHome below); one already on its post
// is left in every field, exactly as 0098 left every released member.
//
// WHAT THIS BUILD DOES NOT HAVE, since 0106, is only the SECOND half of the
// law's own fourth outcome. AI-GRPGUARD-074 read the walk home's own absence
// as forced — that nothing has ever written the post word (`ord+0x00`) for
// a group under the guard order from load, since AI-POST-042's complete
// writer set contains no group-order path — and that clause is RETRACTED
// at the current pin: AI-POST-095 finds the load-time guard setter writes
// the post itself. The walk home was absent for the reason the next
// paragraph gives, a field this tree did not yet carry, never because the
// game had nowhere to write it — and 0106 is where that field arrived.
//
// THE POST IS NOT THE MAP'S ORIGIN, and this paragraph said otherwise until
// 0095: it read an unwritten packed cell as 0 and 0 as outside the playable
// rectangle every shipped map declares, so it took a faithful walk home to
// mean ordering every idle guard toward the map's corner. That reading is
// wrong. The post is the cell the unit stood on when guard was issued, which
// for a placed creature is its spawn cell — so a faithful walk home would
// send an idle guard back to where the map put it, a real behaviour worth
// having.

// minimalGuardRange is the floor under a group's notice radius.
//
// It is a FILE-CARRIED customisation limit and the only one in this file: it is
// `[Scanning] MinimalGuardRange` in `World\Data\ai.reg`, which ships as 8 against
// a code default of 10 — so the code default is never the effective floor
// (AI-RADIUS-014). Lifting it changes a shipped file's bytes and no code.
//
// It is written here rather than read, because this tree reads no registry. The
// seam is a registry read reaching the world, and it lands on this constant.
const minimalGuardRange = 8

// noticeMargin is the 4 the guard arm adds to the group's base radius before it
// clips (AI-RADIUS-014). The law adds one of {-1, 0, +1} beside it, once, on the
// tick a has-members latch flips; the latch is group state this tree does not
// carry, so what is left is the roll's own midpoint (0086 D-4).
const noticeMargin = 4

// scoreSeed is the value the selection loop starts at and the value a vetoed
// candidate scores. The two are ONE constant deliberately: the veto works by
// returning the seed under a strict comparison, so a veto that were merely a very
// large number would become takeable the moment the seed moved (AI-PREF-070).
const scoreSeed int32 = 0xffffff

// The five group orders a decision may read, named as the law's own byte
// values (AI-GROUPCMD-020: 135 nodes over 20 campaign maps move one of
// them).
//
// A COMMAND NOW INSTALLS IT DELIBERATELY, which is why it is named here,
// orderNone, beside the five it used to stand apart from: the group
// command's Patrol sub-command clears a named group's order to exactly this
// value on its way to handing each member to the actor layer.
//
// THE ORDER IS STATE NOW, not a value derived on every decision: it is stored on
// groupAI, carried by the byte form and the digest, and the only writer beside
// construction is a script's group command — which this task does not yet build.
// So orderGuard and orderStandGround are the only two a record can hold this
// task, and decide below has no arm for the other three: a group under one is
// unreachable until the command exists, and this decision leaves it untouched
// rather than guessing at a behaviour that story owns.
//
// orderGuard clips the candidates to a circle about the group's centre, engages
// what is inside it, and releases a member that scores nothing — the arm 95.6 %
// of shipped hostile placements run. orderStandGround clips nothing, refuses
// anything the member could not already strike, and never releases. Both are
// unchanged from the two-value stance this build carried before 0096; only where
// the value comes from has moved.
const (
	// orderNone is the value a group object carries before anything installs
	// one, and — since 0099 — the value a command can install deliberately to
	// mean exactly that: this group is not being decided for. It is named
	// rather than left as a bare 0 at its one call site for the reason every
	// other named order already is — a byte a reader cannot tell from an
	// oversight is worth one line.
	orderNone        uint8 = 0
	orderGuard       uint8 = 1
	orderSwarm       uint8 = 2
	orderStandGround uint8 = 3
	orderMove        uint8 = 4
	orderSwarm2      uint8 = 5
)

// SelfSlot is the roster slot the map's type-5 slot 0 becomes, the slot whose
// groups the load walk sets to stand their ground while every other player's
// guard (AI-AUTHOR-015, AI-STAND-076).
//
// The map's slots are 0-based and an entity's is 1-based — zero names no slot —
// so the map's first roster entry is this. On all 38 maps the EN root ships, that
// entry is named `Self`.
//
// The load walk is described as single-player-only, and this tree has no other
// mode to distinguish it from.
//
// IT IS EXPORTED BECAUSE IT IS ALSO THE SLOT A START'S PARTY TAKES, and that is
// one fact rather than two. A second constant beside the placement would agree
// with this one on the day it was written and would be free to stop; the whole
// engagement layer indexes by this number, so the two coming apart would not be a
// disagreement, it would be a player nobody can see. A story that seats a client
// at some other slot turns this into a value, and it turns it into a value in ONE
// place.
//
// It is deliberately untyped: an entity's owner field is a uint32 and
// freezeGroups reaches stand-ground by comparing a group's owner against this
// constant for equality — every other owner takes guard — so an untyped
// constant serves both comparisons without a conversion at either.
const SelfSlot = 1

// aiGroup is one group as a decision sees it: the pair that identifies it and the
// indices of its living members, ascending.
//
// It is built per decision and stored nowhere — a different shape from groupAI
// below on purpose. The thing being reconstructed hangs a real object off a
// player and keeps a frozen radius, a latch and an order byte on it; the radius
// and, since 0096, the order and the commanded cell all have somewhere to be
// frozen against now — groupAI's own fields, carried on the World, in the byte
// form and in the digest. The latch is what is left with nothing to freeze
// against, because this tree still has no membership changes to freeze it from.
// So this struct stays the fresh, per-decision reconstruction its first line
// says it is, and groupAI is what carries every field that did gain a freeze.
type aiGroup struct {
	owner   uint32
	group   uint32
	members []int
}

// engagementPass is the whole decision: every group, once.
//
// It runs on the phase the mission script already runs on and AFTER it. Both are
// placed on the same slot of the sixteen by their own sources — the script by
// SESS-TICK-006, the AI by AI-TICK-008's `server+0x04 % 16 == 6` — and nothing
// read this round orders the two against each other, so the order here is OURS.
// What decided it: the script is the authored surface and this is the reaction to
// it, so a group command or a diplomacy change is visible to the same tick's
// decision rather than to the next one.
//
// It stands before the tick's occupancy scratch is built, which puts it in the
// same position a command is in — and that is what it is. The engage clears a
// member's destination and its stored route, and every other writer of those
// either runs in the command phase or, as the approach does, runs against a live
// plane it has to keep level. Here there is no plane yet.
func (w *World) engagementPass() { w.engagementPassObserved(nil) }

func (w *World) engagementPassObserved(obs *castObs) {
	w.engageDrew = w.engageDrew[:0]
	if w.savedGroups != nil {
		if w.rom2 == nil {
			w.refreshSavedGroupActivity()
		} else {
			w.refreshROM2SavedGroupActivity(w.rom2ActivityMask())
		}
		w.savedGroupPass(obs)
		return
	}
	for _, g := range w.aiGroups() {
		// THE MEMBER LIST IS NOT FILTERED BY WHO CAST. decide reads g.members
		// for the guard centroid and candidates reads it for the decider and
		// for the shared group sight stamp, so removing a caster moved the
		// group's notice circle and dropped that member's vision for the same
		// pass; a group whose every member cast skipped its decision entirely.
		// A member that began a cast is protected by orderAttack's own cast
		// guard instead, and a destination written to it stands until the cast
		// releases.
		for _, mi := range g.members {
			w.aiCast(mi, obs)
		}
		w.decide(g)
	}
	w.joinEngagedGroups()
}

// aiCast is the spell arm of one decoded AI decision. Map-owned mages filter
// their books to affordable non-defensive rows, choose uniformly and reuse the
// ordinary cast/apply path. The high-Mind 30-percent hand-back and every other
// refusal leave the unit to the ordinary group decision on this pass; so does a
// cast that begins, since the member stays in its group's member list either
// way. The reported bool is whether a cast began.
func (w *World) aiCast(i int, _ *castObs) bool {
	e := w.entities[i]
	if e.Owner == 0 || e.Owner == SelfSlot || e.CastWait != 0 || !isMage(e) {
		return false
	}
	var choices []SpellRule
	for _, rule := range w.spells {
		var resolved bool
		rule, resolved = BookRuleFor(e, rule)
		if !resolved || rule.Defensive || !bookAffords(e, rule) ||
			!knowsSpell(e, uint32(rule.ID)) || !spellApplicable(rule) {
			continue
		}
		choices = append(choices, rule)
	}
	if len(choices) == 0 || (e.Mind > 59 && w.rng.uniform(100) < 30) {
		return false
	}
	rule := choices[w.rng.uniform(int32(len(choices)-1))]
	target, ok := w.autoCastTarget(i, rule)
	return ok && w.beginBookSpell(i, target, uint32(rule.ID))
}

// underCommand reports whether e is executing a player's order rather than
// available to be decided over.
//
// HasTarget is the destination a move order writes and HasAttackTarget is the
// victim an attack order writes; e is under command exactly while it holds the
// first and not the second. Reading the pair costs no field of its own — it is
// derived from two this record already carries rather than stored beside them.
//
// IT IS EXACT TODAY because exactly two writers give an entity a destination
// while leaving it no victim — the plain move order and the group move
// order — and both already end whatever fight the entity was in before
// they write one, so a unit found holding a destination and no victim can
// only be one this build itself sent walking.
//
// A FUTURE WRITER BREAKS THAT EXACTNESS. The walk home this file's own
// header already names as absent — the guard arm's break-off that would
// send a member with no candidate back to its post — and any
// script-ordered move both hand an entity a destination without first taking
// a victim from it, and neither is one of the two writers this argument
// counts.
//
// THAT WRITER HAS NOW ARRIVED (0096: issueGroupDestination and armSwarm both
// hand an entity a destination with no victim taken from it). This predicate
// is UNCHANGED anyway — it still says exactly what it always said, "does
// this entity hold a destination and no victim" — because the wrong
// reading it warned about was never here: aiGroups is where a commanded
// unit's exclusion from a decision is decided, and that is where the
// narrowing landed.
func underCommand(e Entity) bool {
	return (e.HasTarget && !e.HasAttackTarget) || (e.HasAttackTarget && e.AttackTargetKind == AttackTargetStructure) || (e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetStructure)
}

// holdsOrderedVictim reports whether e is the human participant's unit and is
// fighting the unit a player's attack order named.
//
// A player's attack order builds a fresh group at group order 0 and gives each
// member the engage state with the named target (AI-CMD-033, AI-CMD-054), so no
// group scorer runs for it, and being struck issues no order and produces no
// target (AI-RETAL-056). This build leaves the member in the group the map
// placed it in, so the stance decision passes over the member while this holds
// and nothing else replaces the victim.
//
// The engage state is the mark. Every producer of an order publishes it with the
// victim, and stanceMembers returns a member to guard once the victim is gone,
// so a victim a decision acquires afterwards never reads as an ordered one.
func holdsOrderedVictim(e Entity) bool {
	return e.Owner == SelfSlot && e.ActorState == actorStateEngage &&
		(e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit || e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetUnit)
}

// stanceMembers is the part of a group its stance decision may score: the
// members that do not hold an ordered victim. A member still in the engage
// state with its victim gone has finished the order, and the decision takes it
// back at guard before scoring it.
func (w *World) stanceMembers(members []int) []int {
	out := make([]int, 0, len(members))
	for _, mi := range members {
		e := &w.entities[mi]
		if holdsOrderedVictim(*e) {
			continue
		}
		if e.Owner == SelfSlot && e.ActorState == actorStateEngage && !e.HasAttackTarget {
			e.ActorState = actorStateGuard
		}
		out = append(out, mi)
	}
	return out
}

// effectiveGroup selects the temporary command group while an explicit order
// holds, otherwise the placed group. All AI membership reads use this choice.
func effectiveGroup(e Entity) uint32 {
	if e.CommandGroup != 0 {
		return e.CommandGroup
	}
	return e.Group
}

// aiGroups partitions the world's living, owned entities into the groups the map
// placed them in.
//
// The pair is the partition because the thing being reconstructed hangs a group
// list off each PLAYER and forms each group by equality of the map's own group id
// (AI-GROUP-009). Two players' units carrying the same group word are two groups,
// and a build that partitioned on the word alone would give one decision to units
// on opposite sides.
//
// AN ENTITY OF SLOT 0 BELONGS TO NO GROUP and takes no decision. The rule is
// unchanged since 0086 and the population it names is not: it read "the hero, the
// party, everything this tree spawns" until 0094 gave a start's party the roster
// slot it stands on, so what is left here is a test's entity built without naming
// an owner, and a decode of one. It stays a candidate for every other group,
// which is the asymmetry the law has as well: an actor with no player has nothing
// to run its AI, and the sweep that looks for candidates does not care — though
// with slot 0 outside the matrix in BOTH directions, the hostility filter drops
// it before the sweep's own indifference ever shows.
//
// A UNIT UNDER COMMAND BELONGS TO NO GROUP EITHER, and for a different
// reason than slot 0's: it is not that it has nobody to decide for it, it is
// that for as long as the command holds it is in the group its own command
// built rather than the one the map placed it in. That group's whole order
// is walk, and re-issue the walk — no candidate list, no scorer, no engage
// — so leaving it out of the slice below is the whole of that group's
// behaviour and not a partial rendering of it. It stays a candidate for
// every other group on exactly the terms it had before, which is the same
// asymmetry slot 0's own paragraph already states.
//
// THE SKIP IS NARROWED, since 0096 (AC-20): underCommand's own doc names
// "any script-ordered move" as the writer that would break its exactness,
// and a group command is that writer. What still decides whether a commanded
// unit is skipped is not underCommand alone but ALSO its group's STORED
// order — 1 or 3 (Guard, Stand Ground) issue no destination, so a
// destination such a unit holds is a PLAYER'S and the old skip still
// applies; 2, 4, 5 or 17 (Swarm, Move, Swarm 2, Roam) issue their own, so a
// member walking under one of them walks BECAUSE of the group's own order
// and must keep being decided for, or the arm that sent it never runs again.
// A member this list leaves out is still reached by joinEngagedGroups while
// its group is in a fight (DIV-1523).
//
// It is a linear scan into a slice and not a map, for the reason containsIndex is
// one: Go randomises map iteration order per process, and this is a path that
// touches a world. The scan also buys the two orderings the decision needs
// without a second rule — groups in ascending lowest-member id, members ascending
// — so "the group's FIRST member", whose relation row governs the whole group, is
// the head of its slice.
func (w *World) aiGroups() []aiGroup {
	var out []aiGroup
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		e := w.entities[i]
		if !activity.groupActive(e.Owner, effectiveGroup(e)) {
			continue
		}
		// AN OFF-MAP UNIT IS NOT DECIDED FOR. THIS is the set a decision walks —
		// groupLivingMembers below is a second, narrower reader and carries the
		// same test, but a gate written only there would leave this loop putting a
		// removed unit into a group and handing it a victim. Neither is the
		// group's MEMBERSHIP: that is groupMembers (presence.go) and the script's
		// count check, and both read every member whatever its presence.
		if !e.Alive() || e.Owner == 0 || e.OffMap || w.stoneCursed(i) {
			continue
		}
		if _, using := w.structureUseIndex(e.ID); using {
			continue
		}
		if underCommand(e) && !w.groupOrderIssuesDestinations(e.Owner, effectiveGroup(e)) {
			continue
		}
		at := -1
		for k := range out {
			if out[k].owner == e.Owner && out[k].group == effectiveGroup(e) {
				at = k
				break
			}
		}
		if at < 0 {
			out = append(out, aiGroup{owner: e.Owner, group: effectiveGroup(e)})
			at = len(out) - 1
		}
		out[at].members = append(out[at].members, i)
	}
	return out
}

// groupOrderIssuesDestinations reports whether (owner, group)'s STORED order
// is one whose own arm can give a member a destination — Swarm, Move,
// Swarm 2 or Roam — which is aiGroups' discriminator above. A pair no
// record names answers false: such a pair takes no decision at all, so it is
// excluded on the same terms it always was.
func (w *World) groupOrderIssuesDestinations(owner, group uint32) bool {
	order, _, ok := w.groupState(owner, group)
	if !ok {
		return false
	}
	return order == orderSwarm || order == orderMove || order == orderSwarm2 || order == orderRoam
}

// walkHome is the guard stance's own tail: the walk home, run once over a
// whole group's members from each of decide's two guard-only exits below —
// the empty-candidate branch and the foot of the per-member loop — and
// from nowhere else. That is what makes SC-3's claim checkable by deletion
// (AC-12): both call sites gate this function on order == orderGuard before
// reaching it, so deleting the function and its two call sites leaves decide
// compiling and leaves every non-guard order's own behaviour, and
// stand-ground's and swarm 2's own share of these same two exits, untouched.
//
// It draws nothing from w.rng (SC-1): every branch below reads only the
// member's own held fields.
func (w *World) walkHome(members []int) {
	for _, mi := range members {
		e := &w.entities[mi]
		if e.HasAttackTarget {
			continue //
		}
		if e.X == e.PostX && e.Y == e.PostY {
			continue //
		}
		w.clearOrder(mi)
		e.TargetX, e.TargetY, e.HasTarget = e.PostX, e.PostY, true //
	}
}

// decide is one group's whole decision: build the candidates, clip them, and give
// every member the cheapest one it will take.
//
// A GROUP NO RECORD NAMES TAKES NO DECISION. groupState's third return says
// so, and this is its one caller: a pair with no record is left in every
// field rather than falling back to a derivation from its owner. The
// hand-over arms build a record for each new group (order 0), so they do not
// reach this case.
//
// A GROUP UNDER ORDER 0 TAKES NO DECISION EITHER, AND NEEDS ITS OWN CLAUSE
// NOW (AC-7). Order 0 used to be unreachable on a NAMED record —
// freezeGroups writes only guard or stand ground, and nothing before 0099
// could write anything else onto an existing one — so validGroupOrder's
// own refusal of the byte was what stopped this function short for that
// hypothetical case, on the same line that still guards every other
// undefined order.
//
// A MEMBER THAT SCORES NOTHING IS RELEASED WHEN ITS GROUP'S OWNER IS NOT THE
// LOCAL PARTICIPANT: it ends holding no victim, and — because
// releaseAttack is a no-op on a member holding no victim to begin with — a
// member that was not already fighting keeps its destination, its stall
// count and its facing exactly as it held them.
//
// THE KEY IS THE OWNER, NOT THE ORDER. Until this story every Guard group's
// owner was SelfSlot's opposite and every Stand Ground group was SelfSlot's
// own — freezeGroups' own construction rule — so "released under Guard, kept
// under Stand Ground" and "released off the owner" agreed on every world
// 0098 could build, and it read as a rule about the stance. A group command
// breaks that bijection (a script may stand an enemy group's ground, or set
// the participant's own group to guard), which is exactly what AC-21 is
// built to catch, and the rule below was already keyed on the owner alone —
// see TestAReleaseKeysOnTheOwnerNotTheOrder.
//
// A LOCAL PARTICIPANT'S MEMBER UNDER STAND GROUND THAT SCORES NOTHING IS
// STOOD DOWN INSTEAD: the order it holds is replaced by 0 (standDown), so a
// pursuit of a victim that left reach ends at this decision (AI-349, AI-353).
//
// THE THREE NEW ARMS EACH TAKE THIS RULE ON THEIR OWN TERMS. Move's —
// below, in moveArm — is the same per-member release the loop at the foot
// of this function runs, restricted to members that have arrived.
func (w *World) decide(g aiGroup) {
	order, base, ok := w.groupState(g.owner, g.group)
	if !ok || !validGroupOrder(order) || order == orderNone {
		return
	}
	commanded := order == orderSwarm2
	if order == orderRoam {
		at, ok := w.nativeRoam(g)
		if !ok {
			return
		}
		defer func() { w.groups[at].roamCounter++ }()
		order = orderSwarm2
	}
	cands := w.candidates(g)
	if order == orderGuard {
		cx, cy := groupCentroid(w.entities, g.members)
		r := int64(noticeRadius(base))
		cands = clipToNotice(cands, w.entities, cx, cy, r)
	}
	// A group under attack keeps its attackers past the circle clip (DIV-1523).
	cands = w.withEngagedFoes(g.owner, w.groupLivingMembers(g.owner, g.group), order, cands)

	// A member holding a victim a player's attack order named is not scored:
	// the group still sees through it and is centred on it, and only the
	// rewrite of its victim is withheld (AI-CMD-054, AI-RETAL-056).
	deciding := w.stanceMembers(g.members)

	moveArm := func(cands []int) {
		for _, mi := range deciding {
			if !arrived(w.entities[mi]) {
				continue
			}
			best, at := scoreSeed, -1
			for _, ci := range cands {
				if ci == mi {
					continue
				}
				if cost := w.candidateCost(mi, ci, orderStandGround); cost < best {
					best, at = cost, ci
				}
			}
			if at >= 0 {
				w.orderAcquire(mi, w.entities[at].ID)
			} else if g.owner != SelfSlot {
				w.releaseAttack(mi)
			}
		}
	}

	if order == orderSwarm2 && len(cands) == 0 {
		moveArm(cands)
		if commanded {
			w.resumeSwarm(aiGroup{owner: g.owner, group: g.group, members: deciding})
		}
		return
	}

	// SWARM AND MOVE ARE DISPATCHED HERE, BEFORE THE EMPTY-LIST BRANCH BELOW,
	// and that placement is load-bearing rather than an ordering of
	// convenience. Both arms score EVERY member's own path through an empty
	// list correctly by themselves — armSwarm walks an unengaged, non-idle
	// member whether or not any candidate exists at all, and moveArm's own
	// arrival filter must run before anything touches a still-walking member.
	// Guard, Stand Ground and Swarm 2's own non-empty branch have no such
	// filter to lose, so they alone reach the branch below.
	switch order {
	case orderSwarm:
		w.armSwarm(aiGroup{owner: g.owner, group: g.group, members: deciding}, cands)
		return
	case orderMove:
		moveArm(cands)
		return
	}

	// THE COUNT IS READ AS A BYTE, not compared against 256: the law reads it
	// at the width noticeBase and candidateCost already read their own numbers
	// at, and a modulus written out here would be a second spelling of the same
	// fact. The member-count half can never fire — aiGroups only ever yields
	// a group a living member was found for, so len(g.members) is at least 1
	// and its narrowed value is zero only at exactly 256, 512, ... members —
	// and it is tested anyway because the law tests both counts in the same
	// instruction pair, and testing one without the other would be a silent
	// claim about which of the two is reachable.
	if uint8(len(cands)) == 0 || uint8(len(g.members)) == 0 {
		if g.owner != SelfSlot {
			for _, mi := range deciding {
				w.releaseAttack(mi)
			}
		} else if order == orderStandGround {
			for _, mi := range deciding {
				w.standDown(mi)
			}
		}
		if order == orderGuard {
			w.walkHome(deciding)
		}
		return
	}

	for _, mi := range deciding {
		best, at := scoreSeed, -1
		for _, ci := range cands {
			// A MEMBER MAY NOT TAKE ITSELF, and this is the one refusal here that
			// is ours rather than the law's. The engage routine has no self test;
			// what stops it in the original is that a map forces its own diagonal
			// to 2, whose bit 0 is clear. This package cannot REPRESENT a unit
			// attacking itself — the constructor normalises such an order away and
			// the decoder refuses it — so the refusal is here, where it is visible,
			// rather than as a surprise two layers down. It is unreachable from any
			// map: a self-hostile diagonal is a relation no loader writes.
			if ci == mi {
				continue
			}
			if cost := w.candidateCost(mi, ci, order); cost < best {
				best, at = cost, ci
			}
		}
		if at >= 0 {
			w.orderAttack(mi, w.entities[at].ID)
		} else if g.owner != SelfSlot {
			w.releaseAttack(mi)
		} else if order == orderStandGround {
			w.standDown(mi)
		}
	}
	if order == orderGuard {
		w.walkHome(deciding)
	}
}

func arrived(e Entity) bool { return !e.HasTarget && e.Transit == 0 }

// resumeSwarm walks the idle members of a commanded Swarm 2 group on toward
// the cell they were commanded to, once the group sees nothing
// (AI-SWARM2GATE-107, AI-MOVE-023). A fight ends a member's walk, and a
// member has no stored destination of its own here, so the destination is
// re-derived from the group's commanded cell and re-issued through the group
// move writer, formation included. A member within formationSpread of that
// cell counts as arrived and is left alone, so a group that has reached its
// cell is not re-issued on every decision (DIV-1517).
func (w *World) resumeSwarm(g aiGroup) {
	cx, cy, ok := w.groupCommandedCell(g.owner, g.group)
	if !ok {
		return
	}
	to := cell{x: cx, y: cy}
	var idle []int
	for _, mi := range g.members {
		e := w.entities[mi]
		if arrived(e) && !e.HasAttackTarget && cellOf(e).chebyshevTo(to) > formationSpread {
			idle = append(idle, mi)
		}
	}
	if len(idle) > 0 {
		w.issueGroupDestination(idle, to)
	}
}

// armSwarm is order 2's own arm: no clip, the ordinary cost — order passed
// as the group's own orderSwarm, which candidateCost's single stance test
// never matches, so every candidate scores the ordinary way — every member
// scored. A member that scores a candidate engages it.
//
// IT IS A SEPARATE TOP-LEVEL FUNCTION, where moveArm is a closure, because it
// never reaches releaseAttack: TestOrderAttackIsTheOnlySetterAndDecidesOnlyClearerIsTheRelease
// pins that function to decide alone, and armSwarm simply never calls it.
func (w *World) armSwarm(g aiGroup, cands []int) {
	cx, cy, _ := w.groupCommandedCell(g.owner, g.group)
	for _, mi := range g.members {
		best, at := scoreSeed, -1
		for _, ci := range cands {
			if ci == mi {
				continue
			}
			if cost := w.candidateCost(mi, ci, orderSwarm); cost < best {
				best, at = cost, ci
			}
		}
		if at >= 0 {
			w.orderAttack(mi, w.entities[at].ID)
			continue
		}
		e := &w.entities[mi]
		if arrived(*e) && e.X == cx && e.Y == cy {
			// Idle on the commanded cell: left in every field.
			continue
		}
		if escortState(e.ActorState) {
			w.cancelTurnForTargetChange(mi, cx, cy)
			w.clearOrder(mi)
			w.clearEscort(mi)
		}
		e.TargetX, e.TargetY, e.HasTarget = cx, cy, true
		e.clearAttackBetweenCycles()
		e.clearGroupSpeed()
	}
}

// groupState is what a decision for (owner, group) reads off the frozen
// record: the order it holds and the base its clip uses, and whether a
// record names the pair at all. order and base are both zero then — not a
// chosen sentinel, the zeroed span a group object carries before anything
// installs one, on the law's own zeroed-allocation rule (AI-CMD-033).
//
// It is a scan over w.groups and not a map, for groupKeys' own reason: a map
// would put Go's randomised iteration order on a path a decision touches, and
// this is a small, fixed slice built once at construction.
func (w *World) groupState(owner, group uint32) (order, base uint8, ok bool) {
	for i := range w.groups {
		if w.groups[i].owner == owner && w.groups[i].group == group {
			return w.groups[i].order, w.groups[i].base, true
		}
	}
	return 0, 0, false
}

// FrozenGroupAI exports groupState for a constructor that builds saved group
// records from this world: the load walk's order (AI-AUTHOR-015) and the
// frozen notice base for (owner, group).
func (w *World) FrozenGroupAI(owner, group uint32) (order, base uint8, ok bool) {
	return w.groupState(owner, group)
}

// groupCommandedCell is the cell a group command last named for (owner,
// group) — read beside groupState rather than folded into it, so the
// three-value lookup engage_test.go already pins keeps its shape across this
// story. ok follows groupState's own rule — false for a pair no record
// names — though armSwarm's caller has already established one exists by
// the time it asks.
func (w *World) groupCommandedCell(owner, group uint32) (x, y int32, ok bool) {
	for i := range w.groups {
		if w.groups[i].owner == owner && w.groups[i].group == group {
			return w.groups[i].commandedX, w.groups[i].commandedY, true
		}
	}
	return 0, 0, false
}

// scriptGroupOwner is the owner a script's Target_Group id resolves to
// (AI-366). The original answers an id through one id-only map filled by walking
// the players head to tail and each player's groups head to tail with the last
// write winning, so an id carried by two owners answers the later owner's group
// and the earlier one is unreachable by id.
//
// The players follow the map's own roster in ascending owner order on every map
// that carries a repeated id, so the later owner is the greater owner id; the
// roster order is not otherwise modelled here. Every entity counts, living or
// not, on the same terms the membership and count reads already use.
func (w *World) scriptGroupOwner(group uint32) (owner uint32, ok bool) {
	for i := range w.entities {
		if e := &w.entities[i]; e.Group == group && (!ok || e.Owner > owner) {
			owner, ok = e.Owner, true
		}
	}
	return owner, ok
}

// groupsNamed is the indices into w.groups of the one record a script group
// parameter names: the raw group id resolved to the greatest owner among the
// records carrying it, which is scriptGroupOwner's rule over the group list. A
// group id no record carries answers an empty slice, which is what a node naming
// no group and one naming a group no entity carries both get - there is no
// record to change.
func (w *World) groupsNamed(group uint32) []int {
	var out []int
	for i := range w.groups {
		if w.groups[i].group != group {
			continue
		}
		if len(out) > 0 && w.groups[i].owner != w.groups[out[0]].owner {
			if w.groups[i].owner < w.groups[out[0]].owner {
				continue
			}
			out = out[:0]
		}
		out = append(out, i)
	}
	return out
}

// groupLivingMembers is groupKeys' and freezeGroups' own membership scan
// (0095), reused here for the script's Move and Swarm 2 setters: the
// distribution they hand to issueGroupDestination is over the group's
// CURRENT living members, not the ones its notice base was last frozen from.
// AN OFF-MAP MEMBER IS NOT HERE. This is the set a group DECIDES for and
// stamps its shared sight from, and a unit the script has taken off the map
// is neither decided for nor a place the group can see from.
//
// IT IS NOT THE GROUP'S MEMBERSHIP. That is groupMembers (presence.go) and
// the script's count check, and both read every member whatever its presence
// — a group whose members are all off the map still answers its full
// count. The two sets have always differed by the dead; they now differ by
// the removed as well.
func (w *World) groupLivingMembers(owner, group uint32) []int {
	var members []int
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner == owner && effectiveGroup(e) == group && e.Alive() && !e.OffMap {
			members = append(members, i)
		}
	}
	return members
}

// candidates is what the group can see and will fight, in ascending entity id.
//
// A GROUP SEES AS ONE ANIMAL. The law clears one shared visibility map, stamps
// every member's sight into it, and then sweeps the whole world actor list
// against it, so a candidate seen by any member is a candidate for every member
// (AI-GROUPSEE-068). Nothing here is per member except the scoring.
//
// WHAT EACH MEMBER STAMPS is sight.go's march, and 0086's D-1 — the Chebyshev
// disk that stood here while the predicate's two input grids had no traced
// writer — is discharged by 0090. It is worth saying what that disclosure got
// wrong as well as that it is gone: it said the law's region was a SUBSET of the
// disk, so this build could only ever acquire too much. The per-cell term is the
// observer's altitude minus the cell's and descending ground returns budget, so
// the region reaches far past the flat radius from high ground (AI-LOS-089). The
// subset relation holds on flat ground and nowhere else.
//
// The sweep is over the WHOLE entity slice, and that is the set rather than a
// cost being reproduced: the law's own sweep is head to tail with no neighbourhood
// and no spatial index, twice (AI-LIST-003), and any index here would have to
// agree with the full sweep on every world anyway.
//
// THE DIPLOMACY FILTER RUNS WITH THE GROUP'S FIRST MEMBER as the decider, so one
// player row governs the whole group. That is the law's own shape and not a
// simplification: every member of a group shares an owner here, so the two
// readings agree, and a story that lets a group hold mixed owners inherits the
// first-member rule already written down.
//
// FINISHABLE BODIES ARE PARKED, NOT DROPPED. Candidates from 0 through -9 move
// to a second list, and if the first ends empty that list is moved back. The
// owner-directed -10 cutoff is applied before the decoded below-1 parking split
// (`DIV-442`).
func (w *World) candidates(g aiGroup) []int {
	decider := w.entities[g.members[0]]
	stamp := w.groupSight(aiSight, g.members)
	w.stampAttackNotices(stamp, g.members, true)
	var live, dead []int
	for i := range w.entities {
		c := w.entities[i]
		// AN OFF-MAP UNIT IS NOT A CANDIDATE. This sweep is this package's
		// spelling of the law's own head-to-tail walk of the global on-map actor
		// list, and the removal arm's whole effect on acquisition is that the
		// actor is no longer in that list. The test stands FIRST, before sight: a
		// unit that is not on the map is not somewhere the stamp could show or
		// fail to show.
		if c.OffMap {
			continue
		}
		if !w.sightShows(stamp, cellOf(c)) {
			continue
		}
		if w.hasAttachedSpell(c.ID, w.armSpellID(15)) && !w.groupDetectsInvisible(g.members, cellOf(c)) {
			continue
		}
		if !w.hostileTo(decider, c) {
			continue
		}
		if !c.OrdinaryTargetable() {
			continue
		}
		if c.HP < 1 {
			dead = append(dead, i)
		} else {
			live = append(live, i)
		}
	}
	if len(live) == 0 {
		return dead
	}
	return live
}

// groupDetectsInvisible applies each member's own detector radius. Candidate
// terrain visibility is established by the shared group sight map before this
// helper runs; a different member may therefore supply ordinary sight and the
// detector that preserves the invisible candidate.
func (w *World) groupDetectsInvisible(members []int, candidate cell) bool {
	for _, i := range members {
		if i >= 0 && i < len(w.entities) &&
			cellOf(w.entities[i]).chebyshevTo(candidate) <= int64(w.entities[i].SeeInvisible) {
			return true
		}
	}
	return false
}

// noticeBase is the geometry-and-floor half of a guarding group's notice
// circle: the widest member's own reach out from the centre, floored by the
// registry's own floor. It is THE FREEZE — the constructor's freezeGroups
// is its only caller, and it runs once per group, at construction, never
// again. THIS FUNCTION WAS THE SEAM, and the comment standing here used to
// say so: before there was anywhere to freeze this answer to, the arm read
// it fresh every decision and discarded it. Now there is, and freezeGroups
// is what reads it — see noticeRadius below for the clip's own reader, the
// base's only other consumer.
//
// THE LAW'S RADIUS IS FROZEN AND SO IS THIS ONE. The geometry here is
// recomputed by the original on every guard tick — it is the arm's first
// act — and then never read again: the value the clip uses was frozen when
// guard was last issued, at map load, and a group that spreads out, loses
// members or crosses the map keeps the circle it started with
// (AI-RADFREEZE-075). This build freezes at the one moment it has a freeze
// for — construction.
//
// EACH MEMBER CONTRIBUTES ITS OWN SIGHT and not a shared one. The published
// expression is the maximum of `distance + that member's sight`, so the
// member that decides the circle is not necessarily the one standing
// furthest out nor the one seeing furthest — it is whichever member's pair
// is largest. With a single range for everybody the three questions had the
// same answer and the distinction was invisible.
//
// The arithmetic is the field's own width. Each member's distance plus its
// sight is narrowed to a BYTE before the maximum, because that is the width
// the frozen base is stored at. It is a customisation limit and not
// reachable on any map this tree can load — a map is at most 136 cells on
// a side — but it is what a wider map would meet.
func noticeBase(ents []Entity, members []int, cx, cy int32) uint8 {
	centre := cell{x: cx, y: cy}
	var widest uint8
	for _, i := range members {
		if v := uint8(cellOf(ents[i]).chebyshevTo(centre) + int64(ents[i].ScanRange)); v > widest {
			widest = v
		}
	}
	if widest < minimalGuardRange {
		widest = minimalGuardRange
	}
	return widest
}

// noticeRadius is the circle a guarding group notices things inside: the
// group's FROZEN base — noticeBase's answer at construction, read off the
// record rather than recomputed — widened by the arm's margin, as a byte.
//
// THIS IS THE ONLY PLACE THE MARGIN IS ADDED. noticeBase above computes and
// stores the base alone, with no margin folded in and none to invert back
// out, and this is the base's only reader on a decision path — decide calls
// it and nothing else does. Nothing here, and nothing upstream of here, ever
// recomputes a radius from live geometry.
func noticeRadius(base uint8) uint8 {
	return base + noticeMargin
}

// groupAI is one group's FROZEN record: the (owner, group) pair that names
// it, the notice base fixed for it at construction, and — since 0096 —
// the order it is under and the cell a command last gave it. It is a
// different shape from aiGroup above on purpose — aiGroup is built fresh
// every decision and named nowhere the byte form or the digest can reach,
// while this is built once, kept on the World, carried by the form and
// hashed with it. Folding the two into one type would make a struct whose
// fields are canonical on some calls and residue on others.
type groupAI struct {
	owner uint32
	group uint32
	// base is the frozen notice base: the group's LIVING members' widest
	// Chebyshev-distance-from-centroid-plus-sight, narrowed to a byte before
	// the maximum is taken, then raised to the guard-range floor. It is a byte
	// for ScanRange's own reason — the field this value is derived from —
	// and every value is legal: a decode carries it whole and refuses none.
	base uint8
	// order is the group's stored order: one of the five values
	// orderGuard..orderSwarm2, written by construction and, once the script
	// gains its command, by nothing else on a tick path. This task writes only
	// orderGuard and orderStandGround; the other three are unreachable until
	// the command exists and decide has no arm for them.
	order uint8
	// commandedX, commandedY are the cell a group command last named — read
	// by Swarm's release point and by Move's and Swarm 2's destination.
	// Construction leaves it at the origin, unread until a command writes one;
	// this task writes no other value into it. A pair of int32, not the law's
	// packed word: no map this tree can load is 256 cells on a side, and the
	// entity's own coordinates are int32 already.
	commandedX, commandedY int32
	// The Roam evaluation counter survives replacement orders too.
	roamCounter uint8
}

// groupKeys is the (owner, group) pairs the entity records name, ascending
// and unique: every OWNED entity contributes the pair it carries, alive or
// not, and an entity in slot 0 (Owner 0) belongs to no group and contributes
// none.
//
// It is a scan and a sort rather than a map, for aiGroups' own reason: a map
// would make the order — and so the byte form and the digest — depend on
// Go's randomised iteration rather than on the logical world alone. It is
// used both to freeze the constructor's own records and, unfilled, to check
// a decoded section's keys against the entities that name them.
func groupKeys(ents []Entity) []groupAI {
	var out []groupAI
	for i := range ents {
		e := ents[i]
		if e.Owner == 0 {
			continue
		}
		found := false
		for k := range out {
			if out[k].owner == e.Owner && out[k].group == effectiveGroup(e) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, groupAI{owner: e.Owner, group: effectiveGroup(e)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].owner < out[j].owner || (out[i].owner == out[j].owner && out[i].group < out[j].group)
	})
	return out
}

// freezeGroups is the constructor's own use of groupKeys: it fills each
// key's base from noticeBase and, since 0096, its order from the owner's
// slot — the two writes this task adds no third to. Both run once per
// group here and nowhere else on a tick path. A group with no living member
// passes noticeBase no member to widen past the floor, so its base IS the
// floor — AC-3's second edge, needing no clause of its own.
//
// noticeBase computes the base ALONE, with no margin folded in and none to
// invert back out. An earlier version of this file called the geometry loop
// noticeRadius, whose return already carried the arm's margin as a wrapped
// byte, and subtracted the margin back off here — correct by the same modular
// arithmetic noticeRadius(base) now runs the other way, but a margin stated
// in two places and inverted in a third. Splitting the geometry out under its
// own name is what removes both the second place and the inversion: see
// noticeRadius, the base's only other reader, for the one place the margin is
// added now.
func freezeGroups(ents []Entity) []groupAI {
	out := groupKeys(ents)
	for i := range out {
		var members []int
		for k := range ents {
			e := ents[k]
			if e.Owner == out[i].owner && effectiveGroup(e) == out[i].group && e.Alive() {
				members = append(members, k)
			}
		}
		var cx, cy int32
		if len(members) > 0 {
			cx, cy = groupCentroid(ents, members)
		}
		out[i].base = noticeBase(ents, members, cx, cy)
		if out[i].owner == SelfSlot {
			out[i].order = orderStandGround
		} else {
			out[i].order = orderGuard
		}
	}
	return out
}

// clipToNotice drops every candidate standing further than r from the group's
// centre, by Chebyshev distance in cells.
//
// It clips around the CENTROID and not around any member, and the guard arm's
// walk-home destination is the opposite — a member's own post. The two objects
// are easy to confuse and the original tells them apart by nothing but the base
// register it reaches them through, which is why they are named apart here.
//
// The result is a fresh slice: the caller's order is preserved and nothing is
// written back through it.
func clipToNotice(cands []int, ents []Entity, cx, cy int32, r int64) []int {
	centre := cell{x: cx, y: cy}
	out := cands[:0:0]
	for _, i := range cands {
		if cellOf(ents[i]).chebyshevTo(centre) <= r {
			out = append(out, i)
		}
	}
	return out
}

// preference is the 4x4 table the target choice runs through, indexed
// [member domain][candidate domain] over the law's own domain numbering
// (AI-PREF-070). Lower is better; a 0 is an absolute veto.
//
// All sixteen cells are written out so the veto can be checked against the
// source. Row 0 is what a member of reach above 1 reads from and column 0 is
// what a candidate of reach above 1 folds onto.
//
// THE TWO ZEROES ARE REACHABLE: a ground or ghost member never auto-selects a
// flier, whatever the distance, and fliers are placed on 30 of the 38 EN maps
// (AI-FLIER-073). Row 0 contains no zero, so it is reach and not class that
// decides whether anything will chase something in the air.
var preference = [4][4]uint8{
	{2, 1, 2, 4},
	{4, 2, 1, 0},
	{2, 1, 4, 0},
	{2, 1, 2, 4},
}

// lawDomain is this package's movement domain as the law's own byte: ground 1,
// ghost 2, flier 3.
//
// IT IS A TABLE AND NOT `d + 1`, and the difference is the point. The two
// numberings happen to be one apart today because this package numbers its three
// domains from zero and the law numbers its four from zero with a different
// meaning at the bottom: the law's 0 is the immobile column a reach above 1 folds
// onto, and this package has no domain for it. A fourth domain here, or a
// renumbering of ours, would silently reindex the preference matrix under the
// arithmetic and cannot under this.
//
// It is TOTAL and answers ground for anything it does not recognise, which is
// mapload's own domainFor rule and the law's: the selector has three tests and
// stores nothing when all three fail, leaving the constructor's default standing.
func lawDomain(d Domain) int {
	switch d {
	case DomainGhost:
		return 2
	case DomainAir:
		return 3
	}
	return 1
}

// turnCost is the second term of a candidate's cost: how far the member would
// have to turn to face it.
//
// IT IS A TERM THIS TREE DOES NOT HAVE. It is written as a function
// returning zero rather than dropped from the expression, because the
// expression is the law's and only its second term is missing; a later story
// fills this body and nothing else.
//
// WHAT ITS ABSENCE COSTS is bounded and small, which is why the story could be
// built over it at all: it occupies the low byte under a distance term shifted
// eight bits left, so it can only ever reorder candidates that are EQUALLY FAR
// AWAY. Under a zero the tie falls to the list order the strict comparison
// already gives it — ascending entity id (0086 D-2).
func turnCost(member, candidate Entity) int32 { return 0 }

// groupScorerReach is the second target scorer's own distance ceiling: 1, the
// value the package-level `reach` constant carried before 0104 split reach
// into a per-entity field and deleted it.
const groupScorerReach = 1

// candidateCost is what candidate ci is worth to member mi under order, lower
// being better, and scoreSeed for one the member will not take at all.
//
// The two scorers this task builds are ONE body and an order, not two
// functions. The published reading of the second is "the first with three
// differences" (AI-REACH-072), and two bodies would let the thirteen lines
// they share drift apart. The differences are marked below; the third of
// them — a candidate's domain forced to 0 for any reach above 1 rather
// than only for a ground one — is the column choice below. order is the
// group's own stored byte (0096); decide never calls this with a value this
// task has no arm for.
//
// The two modifiers are applied UNGATED under both orders, and under Guard
// that is a narrowing of the input domain rather than a missing arm. The law
// gates them on the candidate's token size being 1 and its Mind being at least
// 15; every entity in this package occupies exactly one cell, and the actor
// constructor defaults Mind to 20 with no `Data.bin` column to move it, so both
// terms are satisfied by construction wherever this build can ask. Neither is a
// field here, and the story that gives an entity a footprint inherits the gate.
//
// Not applied at all: the flat 127 a candidate carrying one active spell id
// costs. This tree has no spells (0086 D-6).
//
// THE NOTICE CIRCLE IS NOT TOUCHED AT ALL BY EITHER TASK — not the jitter,
// not the margin, not noticeBase's own freeze. Measured on the tenth mission
// through this tree's own loader: the group that intercepts holds one
// member, its frozen base is the floor, its working radius is twelve, and
// the target is acquired at a separation of five — a separation the
// one-cell jitter and the margin could not have admitted or refused either
// way, so the circle is not what decides it and a story that changed it
// would be changing something the measurement does not support.
func (w *World) candidateCost(mi, ci int, order uint8) int32 {
	m, c := w.entities[mi], w.entities[ci]

	// The distance is read as a BYTE, which is the width the law reads it at. No
	// map this tree can load is wide enough for the narrowing to bite.
	d := int32(uint8(cellOf(m).chebyshevTo(cellOf(c))))

	if order == orderStandGround && m.Owner == SelfSlot && (footprintSide(m.TokenSize) > 1 || footprintSide(c.TokenSize) > 1) {
		d = min(d, strikeDistance(m, c))
	}

	// THE COLUMN CHOICE, computed before the row choice below because the row
	// choice's own indexing expression is what consumes it: a candidate whose
	// reach carries past one cell folds onto the immobile column — the
	// literal 0, not a domain expression, for the same reason the row choice
	// indexes the table's base literally — in place of its own domain's. The
	// ordinary order folds only a GROUND candidate this way; standing its
	// ground folds a candidate of ANY domain, the one place the two variants
	// differ on this term, so the pair stays one expression with an || on the
	// order rather than two blocks, auditable against the published reading of
	// the stand-ground body as "the ordinary one with three differences."
	//
	// IT IS A STATEMENT OF ITS OWN, NOT FOLDED INTO THE ROW CHOICE BELOW: the
	// two are gated on DIFFERENT ENTITIES' reaches — this one on the
	// candidate's, the row's on the member's — and a single expression
	// reading both would be the one shape a later reader could transpose
	// without a test noticing, since a symmetric pair of reach-4 ground
	// entities scores identically either way.
	col := lawDomain(c.Domain)
	if c.Reach > 1 && (col == 1 || order == orderStandGround) {
		col = 0
	}

	// THE ROW CHOICE: a member whose reach carries past one cell reads row 0
	// ALONE — the literal 0, not lawDomain of anything. The law indexes the
	// table's base with no member term at all once reach permits it, and
	// writing a domain expression here would invite a later reader to make one
	// live where the law never asked for one. A member of reach 1 is unchanged:
	// it still reads the cell at its own domain, exactly as before this story.
	row := lawDomain(m.Domain)
	if m.Reach > 1 {
		row = 0
	}
	pref := preference[row][col]
	// THE VETO IS UNCHANGED, and it stays above everything this story adds: a
	// vetoed pair must not reach the rewrite below. What moves is its
	// CONSEQUENCE — row 0 holds no zero, so a member of reach above 1 has no
	// domain veto at all and may now select a flier, where a member of reach 1,
	// ground or ghost, still may not.
	if pref == 0 {
		return scoreSeed
	}

	// THE DISTANCE REWRITE, immediately after the row choice and under the same
	// guard: a candidate at or inside the member's reach scores a distance term
	// of exactly 1, and one beyond it scores its distance less one short of
	// that reach. Two arms in the law's own order, and NOT a max(1, d+1-reach):
	// the two agree for every value this build can produce, but the closed form
	// would hide which of the two arms a future divergence came from — the
	// same reasoning that keeps the distance narrowing above a conversion
	// rather than a check.
	//
	// NEITHER ARM NEEDS A CLAMP, and that is safe rather than lucky: the
	// constructor folds a reach of 0 to 1 and the decoder refuses one, so reach
	// is at least 1 for every entity any path here can present.
	if m.Reach > 1 {
		if d <= int32(m.Reach) {
			d = 1
		} else {
			d = d + 1 - int32(m.Reach)
		}
	}

	// THE FIRST DIFFERENCE, MOVED BELOW THE REWRITE ABOVE: the second scorer
	// refuses anything past reach outright, before the turn cost is folded in,
	// which is the whole reason a group standing its ground never takes a step
	// toward anything. groupScorerReach stays the literal 1 and stays unwired
	// from Entity.Reach — the previous story's decision, not reversed here
	// — and what lets a member of reach above 1 pass this refusal is the
	// rewrite above having already turned its distance term into 1, not any
	// change to the ceiling itself.
	//
	// THE ORDERING IS DERIVED RATHER THAN QUOTED, and graded MEDIUM
	// confidence: the published reading gives the refusal's own address
	// inside the stand-ground body and the rewrite's address inside the
	// ordinary body, and states the stand-ground body is the ordinary one
	// with three enumerated differences, none of which is the removal of the
	// rewrite — so the refusal's offset falls after the rewrite's. Nothing
	// this story is measured on depends on it: the guarding order, the arm
	// this story exists for, has no such refusal at all.
	if order == orderStandGround && d > groupScorerReach {
		return scoreSeed
	}

	cost := d<<8 + turnCost(m, c)
	// THE SECOND DIFFERENCE: the second scorer's modifiers are coarser — a
	// doubling and a halving where the first has a half and a quarter.
	switch {
	case pref == 1 && order == orderStandGround:
		cost <<= 1
	case pref == 1:
		cost += cost / 2
	case pref == 4 && order == orderStandGround:
		cost >>= 1
	case pref == 4:
		cost -= cost / 4
	}
	return cost
}

// cellOf is e's cell. It is one function so that "where is this unit" cannot come
// to be spelled two ways on a path where a transposed pair would read as a
// plausible distance.
func cellOf(e Entity) cell { return cell{x: e.X, y: e.Y} }

// orderAttack serves active-target writers and a ready pending transfer.
// Reissuing the active endpoint preserves its cycle (AI-REISSUE-077).
func (w *World) orderAttack(i int, victim EntityID, targetKind ...AttackTargetKind) bool {
	kind := AttackTargetUnit
	if len(targetKind) > 0 {
		kind = targetKind[0]
	}
	// A CAST OWNS THE ACTOR AND AN ATTACK CYCLE DOES NOT. actorCastBusy and not
	// actorActionBusy: a spell winding up or recovering refuses this order,
	// which is the non-overlap contract.md asks for between an attack command
	// and a cast. Only the player's own attack command is exempt (attachAttack).
	//
	// A creature's spell draw runs before that refusal: the original's
	// selector reads no cast state of its own actor, so a pending cast does
	// not stop the draw (AI-376).
	if kind == AttackTargetUnit && w.creatureEngageCast(i, victim) {
		return true
	}
	if w.actorCastBusy(i) {
		return false
	}
	return w.attachAttack(i, victim, kind, false)
}

func (w *World) orderAcquire(i int, victim EntityID) bool {
	if !w.orderAttack(i, victim) {
		return false
	}
	// A creature's drawn cast replaces the attack order and leaves no victim to
	// acquire.
	if w.entities[i].HasAttackTarget {
		w.entities[i].AcquirePursuit = true
	}
	return true
}

// Explicit setters queue a different victim behind a loaded cycle. AI writers
// retain their active-target replacement path (AI-354, AI-355).
func (w *World) attachAttack(i int, victim EntityID, kind AttackTargetKind, queued bool) bool {
	e := &w.entities[i]
	if !queued && e.PendingOrder.Kind == PendingRelease && e.HasAttackTarget && e.AttackTarget == victim && e.AttackTargetKind == kind {
		return false
	}
	if w.stoneCursed(i) {
		return false
	}
	if kind == AttackTargetStructure {
		si := indexOfStructure(w.structures, StructureID(victim))
		if si < 0 || !w.structures[si].physicalTarget() {
			return false
		}
	} else {
		vi := indexOfEntity(w.entities, victim)
		if vi < 0 || !w.entities[vi].OrdinaryTargetable() || w.invisibleToActor(i, vi) {
			return false
		}
		// Every writer reaches the victim store here, so the flier veto sits here.
		if w.targetVetoed(i, vi) {
			return false
		}
	}
	// A reissue at the victim a pursuit holds keeps its route, as the
	// original's order rewrite leaves the mover alone (AI-REISSUE-077,
	// DIV-2558).
	keep := kind == AttackTargetUnit && e.HasAttackTarget && e.AttackTargetKind == kind && e.AttackTarget == victim && e.Pursuit.Held
	route, tx, ty, has := w.routes[i], e.TargetX, e.TargetY, e.HasTarget
	w.clearOrder(i)
	if keep {
		w.routes[i], e.TargetX, e.TargetY, e.HasTarget = route, tx, ty, has
	}
	e.PendingOrder = PendingOrder{}
	e.clearPendingAttack()
	if queued && e.HasAttackTarget && e.AttackPhase != AttackReady && (e.AttackTarget != victim || e.AttackTargetKind != kind) {
		e.PendingAttackTarget, e.PendingAttackTargetKind, e.HasPendingAttackTarget = victim, kind, true
		return true
	}
	if !e.HasAttackTarget || e.AttackTarget != victim || e.AttackTargetKind != kind {
		// A replaced victim, unit or structure, ends the held pursuit search
		// (DIV-2557).
		e.clearTurn()
		e.Pursuit = PursuitSearch{}
		e.AttackTarget, e.HasAttackTarget = victim, true
		e.AttackTargetKind = kind
		e.AttackPhase, e.AttackCountdown = AttackReady, 0
	}
	e.AcquirePursuit = false
	e.PursuitIdle = false
	return true
}

// releaseAttack preserves nonzero progress and records its release separately.
// A same-victim AI rewrite cannot erase it; an admitted replacement can.
// Other pending player actions and actors without a victim are left alone.
func (w *World) releaseAttack(i int) {
	e := &w.entities[i]
	if e.PendingOrder.Kind != PendingNone && e.PendingOrder.Kind != PendingRelease {
		return
	}
	if !e.HasAttackTarget {
		return
	}
	e.releaseBetweenCycles(PendingRelease)
	w.clearOrder(i)
}
