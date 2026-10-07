package data

import "math"

// SkillSlots is how many skill levels a hero carries: six, not five and not
// ten. Slot 0 is the one the character sheet does not show; slots 1 to 5 are the
// shared set the class renames, and the shipped column titles spell both names
// in one string — `Skill.Blade (Fire)`.
//
// Six is bounded by the field that follows the array, not by a count anyone
// wrote down: ten slots would not fit before it.
const SkillSlots = 6

// The five slots a weapon's attack type names. A melee weapon ASSIGNS its own
// kind into the wielder's active-skill field, so the type and the slot are one
// number — which is why a blade weapon is the one a blade-trained hero is
// handed.
const (
	SkillGeneral = 0
	SkillBlade   = 1
	SkillAxe     = 2
	SkillBludgen = 3
	SkillPike    = 4
	SkillShoot   = 5
)

// StatCap is what a stat is clamped to before anything reads it. The real rule
// is `min(stat, 50 + (int8)modifier)` and the modifier's only writer is the
// effect dispatch, which no hero this tree builds has met — so every hero here
// is capped at the bare 50. The ceiling in principle is 100 and is reached only
// through an effect.
const StatCap int32 = 50

// The character-generation start: four stats at 25, and exactly one skill slot
// written, at 10.
//
// The 25 is the initialiser's own immediate, four times over, beside a pool of
// 100 — and `4*T(25) + 100 = 140` is the same budget the accepting side
// re-derives, so the pool and the budget are one number seen twice. The 10 is
// the ordinary arm of the value the chargen writer picks; the other arm writes
// 20 and is taken only under a conjunction of two mode tests that no shipped
// single-player campaign is shown to satisfy.
const (
	ChargenStat  int32 = 25
	ChargenSkill int32 = 10
)

// The cadence a hero swings at with nothing in his hands. A weapon ASSIGNS over
// both, and the unequip inverse restores exactly these two — which is what
// makes them the bare pair rather than a default anyone chose.
const (
	BareChargeTime int32 = 8
	BareRelaxTime  int32 = 4
)

// The constants of the derive itself, each read out of the image's own bytes
// rather than out of a decompilation: the base of every exponential in the
// graph, the damage divisor, the to-hit divisor, the to-hit multiplier on the
// active skill and the damage divisor on it.
const (
	statBase       = 1.1
	damageDivisor  = 20
	toHitDivisor   = 5
	skillToHitMult = 3
	skillDamageDiv = 5
	defenceDivisor = 3
)

// Hero is a character's four primary statistics and his six skill levels: the
// INPUTS of the derived-stat graph, and the only state a hero of this tree has.
//
// The four are the game's own registry key names, bound to the screen's rows by
// those key literals rather than by their order — a distinction that matters
// because the storage order crosses two of them. Body drives damage and health;
// Reaction drives to-hit, defence and speed; Mind and Spirit reach neither
// number this type derives.
//
// THE ZERO VALUE IS A LEGAL HERO and derives without a special case. It is
// nobody's character — chargen floors every stat at 15 — but its arithmetic is
// the same arithmetic, which is what lets a caller holding no hero yet get an
// answer instead of a branch.
type Hero struct {
	Body, Reaction, Mind, Spirit int32

	// Skill is the six levels at their own slots. Character generation zeroes
	// 1 to 5 and writes exactly one; nothing here enforces that, because a
	// hero who has trained is an ordinary hero and this type is not a screen.
	Skill [SkillSlots]int32
}

// NewHero is a hero built from a point spread and the one slot he trained: the
// four statistics as generation left them, and that slot at 10.
//
// WHICH SLOT IS THE CALLER'S. Character generation is a screen this tree does
// not have, so nothing here can read the choice from anywhere; the caller states
// it and owns the divergence. A slot outside 1..5 trains nothing, which is the
// hero the mage arm and the untrained case both produce.
//
// IT DOES NOT CHECK THE SPREAD. A hero of any four numbers is a hero and derives
// by the same arithmetic — the zero value included — so refusing one here would
// put a screen's rule inside a value constructor and leave every test that wants
// an out-of-range stat unable to build one. Whether generation could have
// produced a spread is Spread.Legal, asked by whoever is claiming that it could.
func NewHero(s Spread, slot int32) Hero {
	h := Hero{Body: s.Body, Reaction: s.Reaction, Mind: s.Mind, Spirit: s.Spirit}
	if slot > SkillGeneral && slot < SkillSlots {
		h.Skill[slot] = ChargenSkill
	}
	return h
}

// NewCampaignHero is the character generation START, trained in one skill slot:
// four statistics at the initialiser's 25 and that slot at 10.
//
// IT IS THE OPENING STATE OF A SCREEN AND NOT A CHARACTER. Twenty-five is what
// the initialiser writes into all four rows beside a pool of 100, before the
// player has spent a point; a party carrying it is carrying a screen that was
// never used. Which spread a player leaves that screen with is the caller's, and
// this tree's own answer to that is in the front end, not here.
//
// It is DEFINED IN TERMS of NewHero rather than beside it, so there is one
// construction implementation and the start cannot drift away from the general
// case it is an instance of.
func NewCampaignHero(slot int32) Hero { return NewHero(ChargenSpread(), slot) }

// Combat is what a blow reads: the eight numbers Hero.Derive produces, plus
// Reach, and what a simulation entity carries.
//
// The eight are the OUTPUT of the graph and hold no stat, which is the point
// — a consumer that wanted to re-derive one of them from a stat would have
// to go back to the Hero, and there is exactly one routine that may. Reach
// is not a ninth term of that graph: no stat feeds it. Derive still sets it
// — directly from its own weapon argument, beside the graph rather than
// inside it — and UnitDef.Combat, which has no Derive to call, sets it
// too, from its own row. SkillSlot rides beside Reach for the same reason:
// it too is a function of the weapon alone, filled by Recompute from the
// same weapon argument, through the rule activeSkill already names for which
// slot a wielded weapon's own kind is.
type Combat struct {
	// DamageBase and DamageSpread are the roll `base + U[0, spread]`. They are
	// NOT a minimum and a maximum: the sheet's own two numbers are `base` and
	// `base + spread`, and the pair adds contributor by contributor with the
	// bases summing separately from the spreads.
	DamageBase   int32
	DamageSpread int32
	// SecondBase/SecondSpread are the independent Water-reduced physical
	// component (HERO-DMG2-029), retained from an observed Human attack block.
	SecondBase, SecondSpread uint8

	// SecondaryDamage is the third damage component: one byte-width base and
	// spread reduced through the one elemental protection Selector names. Item
	// effects and ranged weapon rows meet in this same triple before it crosses
	// into canonical simulation state; it is not a second persisted record.
	SecondaryDamage SecondaryDamage

	ToHit      int32
	Defence    int32
	Absorption int32

	// AlwaysHits is the mark that skips the hit roll. NO HERO CARRIES IT: its
	// only writer is the non-hero streamer's third damage-routing arm, so this
	// is false for everything this file produces, and it is a field rather than
	// an omission because the entity it fills has one.
	AlwaysHits bool

	AttackChargeTime int32
	AttackRelaxTime  int32

	// Reach is how far a blow this combat block describes carries, in whole
	// cells. Derive fills it from its own weapon argument — 1 when w is nil,
	// w.Range otherwise — so a generated character's and a placed person's
	// reach come out of the one assignment, and HumanDef.Combat carries the
	// answer through unchanged. UnitDef.Combat, which never calls Derive, fills
	// it separately from its own row's column.
	Reach int32

	// SkillSlot is the skill slot a blow through this combat block would credit
	// a gain to: a melee weapon's own attack type, SkillGeneral for a bare hero
	// or a non-melee weapon. It is filled the same way Reach is, from
	// Recompute's own weapon argument — so a generated character's and a
	// placed person's credited slot come out of the one assignment activeSkill
	// already makes for the wielder's active skill, and this field is not a
	// second, independently-arrived-at answer to which skill a weapon trains.
	// UnitDef.Combat, which never calls Derive, takes the same answer from
	// UnitDef.SkillSlot: definitionFor stores FoldWeapon's result there and
	// mapload carries it into a creature's canonical XPSlot (1039).
	SkillSlot int32

	// SpellName and SpellPower are 0139's own pair, carried by weapon.go's
	// Weapon under the same names and assigned onto this block by FoldWeapon
	// exactly where Reach and SkillSlot already are (plan D-2, D-3) — not a
	// new channel of their own. SpellName is the token AS WRITTEN, underscores
	// intact — turning it into a Spells row's own id is a later tier's lookup
	// (spec FR-1b), not this block's. Both are zero for a bare bearer and for a
	// weapon carrying no spell, which is indistinguishable either way, because
	// that IS what "no spell" means. UnitDef.Combat, which never calls
	// FoldWeapon, carries its OWN two fields of the same name straight across
	// instead (unitdef.go).
	SpellName  string
	SpellPower int32
}

// Derive is a thin accessor over Hero.Recompute: it builds the recompute
// with a zero Profile -- which reaches only the two pools, never Combat --
// and this weapon as the Loadout's own weapon, and returns the combat block
// alone. A nil weapon is a BARE hero, which is a state of the original and
// not a hole in this one; the zero-value Hero derives without a special
// case, the same as every other input to Recompute.
//
// It reads nothing but its two arguments: no clock, no generator, no global,
// no file. The same pair yields the same eight numbers in every process.
//
// THE GRAPH ITSELF IS Recompute's OWN DOC, in recompute.go: the order, every
// term, the equipment fold and why absorption has no source but armour.
// Nothing about it is restated here, so there is exactly one place it can
// drift from.
func (h Hero) Derive(w *Weapon) Combat {
	return h.Recompute(Profile{}, Loadout{Weapon: w}).Combat
}

// The speed law's own constants: the branch, the divisor above it and the term
// added there. Each is a named immediate in the routine that computes it.
const (
	speedBranch  int32 = 12
	speedDivisor int32 = 5
)

// Speed is a thin accessor over Hero.Recompute: it builds the recompute with
// a zero Profile and an empty Loadout -- neither reaches this field -- and
// returns Speed alone.
//
// IT IS NOT ONE OF THE EIGHT and it is deliberately not on Combat. Those are
// the numbers a BLOW reads; this is the number a STEP reads, and folding it
// in would make every consumer of a blow's numbers carry a movement term it
// has no use for.
//
// THE LAW ITSELF -- the branch, the two published terms this tree omits and
// why, and the divergence between where the publishing row says the result
// lands and where this tree's own single speed field consumes it -- is
// Recompute's own step 5 doc, in recompute.go, beside Sight's.
func (h Hero) Speed() int32 {
	return h.Recompute(Profile{}, Loadout{}).Speed
}

// The sight law's own two constants: the divisor the sum of the two statistics
// is taken over, and the term added after it. Each is a named floating-point
// immediate in the routine that computes it.
const (
	sightDivisor int32 = 25
	sightBase    int32 = 4
)

// Sight is a thin accessor over Hero.Recompute: it builds the recompute with
// a zero Profile and an empty Loadout -- neither reaches this field -- and
// returns Sight alone.
//
// IT IS NOT ONE OF THE EIGHT and is deliberately not on Combat, for Speed's
// reason exactly: those are the numbers a BLOW reads and this is a number
// the ACQUISITION reads, so folding it in would hand every consumer of a
// blow's numbers a term it has no use for.
//
// THE LAW ITSELF -- the integer-vs-floating-point argument for why this
// division gives the same answer as the original's scaled one, and the
// sub-cell remainder this tree drops -- is Recompute's own step 5 doc, in
// recompute.go, beside Speed's.
func (h Hero) Sight() int32 {
	return h.Recompute(Profile{}, Loadout{}).Sight
}

// Reward is the two numbers a party member is minted with: his capped Mind,
// which the gain scaling reads, and the six per-slot experiences his current
// levels already account for. A mint site sums the latter to seed an
// entity's SkillXP with the same total the character sheet already implies,
// so the panel's first-frame number is not a reset to zero that the first
// blow then has to climb back out of.
type Reward struct {
	Mind    int32
	SkillXP [SkillSlots]int32
}

// Reward is a thin accessor over Hero.Recompute: it builds the recompute
// with a zero Profile and an empty Loadout — neither Mind nor SkillXP
// reads either — and returns the two together. IT COMPUTES NOTHING OF ITS
// OWN: step 2 of Recompute is the one place a level becomes an experience,
// and this method exists so a mint site never carries a second copy of that
// sum that could drift from it.
func (h Hero) Reward() Reward {
	d := h.Recompute(Profile{}, Loadout{})
	return Reward{Mind: d.Mind, SkillXP: d.SkillXP}
}

// activeSkill is the supported skill slot the wielder's derive reads: a melee
// weapon's own attack type in 1..5, and none at all for a bare, ranged or
// unsupported weapon.
//
// Resolution accepts both melee and ranged rows. The explicit branch keeps the
// rule — only a supported melee kind becomes active — local to the value it
// produces instead of relying on a caller to pre-filter the weapon.
func activeSkill(w *Weapon) int32 {
	if w == nil || !w.Melee() {
		return SkillGeneral
	}
	if w.AttackType <= SkillGeneral || w.AttackType >= SkillSlots {
		return SkillGeneral
	}
	return w.AttackType
}

// capStat is step 0 of the recompute, and it is DESTRUCTIVE: the smaller of the
// stat and the cap is written back, so every term after it reads the capped
// value rather than the stat.
func capStat(v int32) int32 {
	if v > StatCap {
		return StatCap
	}
	return v
}

// pow11 is 1.1 raised to a stat, the one non-linearity in the whole chain.
//
// The image calls the C runtime's `pow` on the double nearest 1.1, and two
// implementations of `pow` may differ by an ULP — which would matter, because
// the result is truncated and the truncated value is hashed. It does not matter
// HERE, and that is asserted rather than assumed: over the whole reachable stat
// range no term of this graph stands closer than 2e-4 to an integer boundary,
// a margin of order 1e12 ULPs. The test that measures it is beside this file.
func pow11(n int32) float64 { return math.Pow(statBase, float64(n)) }

// SightSubCells is HERO-SIGHT-007's sight word, `((mind + reaction)/25 + 4)`
// in 1/256 cell, whose high byte is the whole-cell radius Derived.Sight holds.
func SightSubCells(mind, reaction int32) int32 {
	return (mind+reaction)*256/sightDivisor + sightBase*256
}
