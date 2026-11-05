package sim

import "sort"

// Group engagement runs through the World's registry and its actor orders.

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

// noticeMargin is the 4 the guard arm adds to the group's base radius
// (AI-RADIUS-014). Guard adds jitter once when its latch flips (guardLatch).
const noticeMargin = 4

// scoreSeed is the value the selection loop starts at and the value a vetoed
// candidate scores. The two are ONE constant deliberately: the veto works by
// returning the seed under a strict comparison, so a veto that were merely a very
// large number would become takeable the moment the seed moved (AI-PREF-070).
const scoreSeed int32 = 0xffffff

// Group order bytes admitted by the registry dispatcher.
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

// aiGroup projects one candidate scan's owner and living members.
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

func (w *World) engagementPassObserved(obs *castObs) { w.engagementPassTraced(obs, nil) }

// engagementPassTraced is the engagement pass with the withdrawal observer the
// registry dispatch's withdrawal tail reports to.
func (w *World) engagementPassTraced(obs *castObs, withdrawals *withdrawalObs) {
	w.engageDrew = w.engageDrew[:0]
	if !w.game.activityMask {
		w.refreshSavedGroupActivity()
	} else {
		w.refreshROM2SavedGroupActivity(w.rom2ActivityMask())
	}
	w.savedGroupPassTraced(obs, withdrawals)
	w.joinSavedEngagedGroups()
}

// mageEngageCast is the full-book draw of the engage routine (AI-554,
// MAGIC-AI-012) for a computer mage given its victim. Above Mind 59 a 30 in
// 100 draw hands the pass back to the ordinary engage. Otherwise it draws
// uniformly from the affordable non-defensive ids 1..28 of its book; an empty
// list casts id 28 when the book holds it. The cast is aimed at the victim; out
// of reach the mage walks into reach with the cast armed, and during a loaded
// strike the armed cast waits for the strike to end. The draw reads no
// cast state of the mage, so with a cast pending it replaces the retained
// order as a creature's slot draw does (AI-376). It reports whether the cast
// took the pass.
func (w *World) mageEngageCast(i int, victim EntityID) bool {
	e := w.entities[i]
	r := castRequest{issuer: castEngage, caster: i, aim: castAim{target: victim, victim: victim}}
	if w.humanParticipantUnit(e) || !isMage(e) || w.castRecoveryRefuses(r) {
		return false
	}
	if e.Mind > 59 && w.rng.aiRange(100) < 30 {
		if !w.castOwned(r) {
			// The hand-back's engage order replaces an armed one (AI-376).
			w.dropCastApproach(i)
		}
		return false
	}
	all := w.game.bookWalkAll
	var choices []SpellRule
	for _, rule := range w.spells {
		if !bookWalkTakes(rule.ID, all) {
			continue
		}
		var resolved bool
		rule, resolved = BookRuleFor(e, rule)
		if !resolved || rule.Defensive || !bookAffords(e, rule) ||
			!knowsSpell(e, uint32(rule.ID)) || !spellApplicable(rule) {
			continue
		}
		choices = append(choices, rule)
	}
	var id uint32
	fallback, falls := bookWalkFallback(all)
	switch {
	case len(choices) > 0:
		id = uint32(choices[w.rng.aiRange(int32(len(choices)))].ID)
	case falls && knowsSpell(e, fallback):
		id = fallback
	default:
		return false
	}
	// The draw read no cast state, so a cast that owns the mage takes the
	// drawn spell into its retained order only now.
	r.spell = id
	if w.castOwned(r) {
		return false
	}
	w.dropCastApproach(i)
	// The cast is at the victim; its cell is where an area row lands.
	_, vx, vy, _, ok := w.victimAimOperands(i, victim, id)
	if !ok {
		return false
	}
	refusal := w.castRefusal(r, false)
	if refusal == "" {
		return w.startEngageCast(i, victim, victim, vx, vy, false, id)
	}
	rest := refusal
	if isRangeRefusal(refusal) {
		rest = w.castRefusal(r, true)
	}
	if w.castStrikeWaits(r, rest) {
		rest = ""
	}
	if rest == "" {
		return w.armCreatureCast(i, victim, victim, vx, vy, false, id)
	}
	if isManaRefusal(rest) {
		// Admission refuses on mana alone and the order waits (MAGIC-348);
		// only the unfiltered last spell reaches here.
		vi := indexOfEntity(w.entities, victim)
		if vi < 0 {
			return false
		}
		w.standForCreatureCast(i)
		v := w.entities[vi]
		return w.queueBookCast(bookCast{Caster: e.ID, Target: victim, Spell: uint16(id), X: v.X, Y: v.Y, Retained: true})
	}
	return false
}

// underCommand reports a held movement or structure attack order.
func underCommand(e Entity) bool {
	return (e.HasTarget && !e.HasAttackTarget) || (e.HasAttackTarget && e.AttackTargetKind == AttackTargetStructure) || (e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetStructure)
}

// holdsOrderedVictim keeps a human unit's ordered victim out of stance scoring
// until its engage state ends (AI-CMD-054, AI-RETAL-056).
func holdsOrderedVictim(e Entity) bool {
	return e.Owner == SelfSlot && e.ActorState == actorStateEngage &&
		(e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit || e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetUnit)
}

// effectiveGroup selects the temporary command group while an explicit order
// holds, otherwise the placed group. All AI membership reads use this choice.
func effectiveGroup(e *Entity) uint32 {
	if e.CommandGroup != 0 {
		return e.CommandGroup
	}
	return e.Group
}

// walkHome sends victimless members to their current posts.
func (w *World) walkHome(members []int) {
	for _, mi := range members {
		w.walkToCell(mi, w.entities[mi].PostX, w.entities[mi].PostY)
	}
}

// walkToCell sends member mi with no attack target to cell (x, y), unless it
// stands there.
func (w *World) walkToCell(mi int, x, y int32) {
	e := &w.entities[mi]
	if e.HasAttackTarget {
		return //
	}
	if e.X == x && e.Y == y {
		return //
	}
	w.clearOrder(mi)
	e.TargetX, e.TargetY, e.HasTarget = x, y, true //
}

func arrived(e Entity) bool { return !e.HasTarget && e.Transit == 0 }

// groupState reads the retained native record used by older input and witnesses.
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
		if e.Owner == owner && effectiveGroup(&e) == group && e.Alive() && !e.OffMap {
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
func (w *World) candidates(g aiGroup) []int { return w.groupCandidates(g, true) }

// groupCandidates is candidates; consume false reads the attack notices
// without ageing them, for a test that decides nothing.
func (w *World) groupCandidates(g aiGroup, consume bool) []int {
	decider := w.entities[g.members[0]]
	stamp := w.groupSight(aiSight, g.members)
	w.stampAttackNotices(stamp, g.members, consume)
	var live, dead []int
	for i := range w.entities {
		c := &w.entities[i]
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
		if !w.perceives(perception{purpose: perceiveGroup, members: g.members, target: i}) {
			continue
		}
		if !w.hostileTo(&decider, c) {
			continue
		}
		if !c.OrdinaryTargetable() {
			continue
		}
		if c.InLife(LifeDecaying) {
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
		if v := uint8(cellOf(&ents[i]).chebyshevTo(centre) + int64(ents[i].ScanRange)); v > widest {
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
// noticeBase above stores the base alone, with no margin folded in. This is
// the native Group's radius (heldCircle); a registry Group's working radius
// is the one guardLatch rolls.
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
			if out[k].owner == e.Owner && out[k].group == effectiveGroup(&e) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, groupAI{owner: e.Owner, group: effectiveGroup(&e)})
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
			if e.Owner == out[i].owner && effectiveGroup(&e) == out[i].group && e.Alive() {
				members = append(members, k)
			}
		}
		out[i].base = centroidNotice(ents, members)
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
		if cellOf(&ents[i]).chebyshevTo(centre) <= r {
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
	d := int32(uint8(cellOf(&m).chebyshevTo(cellOf(&c))))

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
func cellOf(e *Entity) cell { return cell{x: e.X, y: e.Y} }

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
	if w.castHolds(i, waitCastAction) {
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
