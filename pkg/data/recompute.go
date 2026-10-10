package data

import (
	"math"

	"againrom/pkg/rules"
)

// The recompute's own constants, named the same way hero.go's are rather
// than inlined: how many discrete units a slot's experience scales by, the
// divisor experience is normalised through before it feeds a pool's
// logarithm, the divisor of a pool's own exponential growth term, the
// protection clamp's additive floor and its ceiling, the class multiplier
// the health and mana arms invert between them, and the divisor Spirit is
// halved by for a protection's base and again for the clamp's own ceiling
// term.
const (
	skillXPScale           = 1000
	poolXPDivisor          = 5000
	poolGrowthDivisor      = 100
	protectionClampBase    = 70
	protectionClampCeiling = 100
	classMult              = 2
	spiritHalfDivisor      = 2
)

// The carry capacity's own two constants: capacity is the capped Body times
// ten, plus one. HERO-SIGHT-007 states both constants (a multiply by ten and an
// add of one at L03918 and L03919) and the field they land in,
// `actor+0x92`. They are named rather than inlined for the reason every
// other constant in this file is: the arithmetic is one line and the two
// numbers are the whole of what a reader has to check.
const (
	capacityMultiplier = 10
	capacityAddend     = 1
)

// SkillFloor and SkillCap bound a trained skill level. The original clamps a
// restored level to this pair twice inside one recompute; this tree holds the
// base to the training cap, adds the bonus and bounds the sum by the effective
// bound of pkg/rules, so a bonus may lift the level past SkillCap.
const (
	SkillFloor int32 = 0
	SkillCap   int32 = rules.ROM1SkillCap
)

// Profile is what a character's four statistics and six skill levels do not
// say about him: whether the class flag that doubles one pool's growth arm
// and halves the other's experience arm is set, and whether the health and
// mana columns a shipped row carries are nonzero. Its zero value is flag
// clear and both columns absent, which is exactly what a generated character
// is able to state about himself -- nothing.
//
// HealthColumn IS NOT THE HEALTH ARM'S GATE. What HealthColumn still says is
// narrower -- that a health maximum is trustworthy to hand out for this
// character -- and it keeps exactly one reader for that fact: the party
// mint (pkg/mapload/start.go), which trusts the graph's own derived value
// only when this bit is set, a provisional health otherwise. A shipped base
// row is one way to earn that trust; `chargenProfile`'s own "no base row
// resolves" fallback (pkg/game/hero.go) sets it too, because HERO-HP-072's
// derive needs only Body and the class bit, neither of which needs a
// shipped row. So this bit is not "came from a row" -- it is "this
// character's health inputs are ready," which a row supplies and a
// constructed fallback can supply just as well.
// ManaColumn is unchanged: the mana arm below still gates its WHOLE
// derivation on it, so a character with no mana column has no pool at all
// rather than a narrowed one.
//
// WHICH ARCHETYPE ACTUALLY SETS Fighter IS NOT DECIDED BY THIS TREE. The
// field is named for the claim wording that describes the arm taking the
// doubled health term, not for a resolved mapping from a class id to the
// flag; a later story that decodes which classes carry it wires this field,
// this one only carries it.
//
// Rider is the +10 speed arm (HERO-SPEED-008).
type Profile struct{ Fighter, HealthColumn, ManaColumn, Rider bool }

// RiderSpeedBonus is the rider speed term.
const RiderSpeedBonus int32 = 10

// RiderTypeID reports a rider type id.
func RiderTypeID(typeID int32) bool { return typeID == 0x13 || typeID == 0x15 }

// SpeedBonus is p's rider speed term.
func (p Profile) SpeedBonus() int32 {
	if p.Rider {
		return RiderSpeedBonus
	}
	return 0
}

// EquipMod is the additive contribution every equipped item OTHER than the
// weapon in hand folds into: a to-hit term, a damage base and a damage
// spread term, a defence and an absorption term, and the five elemental
// protection and five damage-kind resistance terms a character's armour and
// accessories would state. It is added, never multiplied -- there is no
// multiply anywhere on the path from an item to a damage number.
//
// Item effects may fill every remaining scalar, family and triple before the
// recompute. FoldWear still supplies only the shipped armour and shield base
// contributions it owns; ApplyItemEffects supplies the separately authored
// enchantment contribution.
//
// SecondaryDamage is one byte-width base, spread and protection selector.
// Kinds 44 through 48 replace this whole value in equipped-slot and stored-
// effect order. They are not five components that can coexist.
type SecondaryDamage struct {
	Base, Spread uint8
	Selector     uint8
}

type EquipMod struct {
	ToHit, DamageBase, DamageSpread, Defence, Absorption int32
	Protection, Resistance                               [5]int32
	Body, Reaction, Mind, Spirit                         int32
	HealthMax, ManaMax                                   int32
	HealthRegeneration, ManaRegeneration                 int32
	Speed, RotationSpeed, Sight                          int32
	SecondaryDamage                                      SecondaryDamage
	// HasSecondaryDamage distinguishes no elemental item effect from an
	// authored zero-valued effect. Weapon::Equip writes its built-in ranged
	// component before walking the item's effects; the later effect replaces
	// the whole triple even when all three bytes are zero.
	HasSecondaryDamage bool

	// SkillBonus is the per-slot bonus block the restore adds to a base
	// level (FR-6a). It is the ONE contribution here
	// that is not folded into an output number: it moves the LEVEL, and
	// the level then moves to-hit and the damage base through the
	// active-skill terms. Slot 0 is never read from it, because slot 0 is
	// in neither the restore loop nor the clamp loop.
	SkillBonus [SkillSlots]int32
}

// Loadout is what a character is wearing, as the recompute reads it: the
// weapon in his hand, and the accumulated EquipMod every other equipped
// item folds into.
//
// THE WEAPON IS KEPT SEPARATE BECAUSE IT IS NOT MERELY ADDITIVE. A resolved
// weapon ASSIGNS the wielder's cadence, per half, and his reach and
// active-skill slot; folding it into EquipMod would lose those three
// assignments and leave only its additive numbers. A nil Weapon is a BARE
// character, which is a state of the original and not a hole in this one.
type Loadout struct {
	Weapon *Weapon
	Mod    EquipMod

	// RotationSpeed is the definition-row base term. See the type doc above.
	RotationSpeed int32

	// Rules bound a restored skill level and give each level's experience. The
	// zero value is the original game's.
	Rules rules.Rules
}

// Derived is everything Recompute produces from a Hero, a Profile and a
// Loadout: the four capped statistics, the six per-slot experience costs and
// their sum, the two pool maxima, the eight combat numbers and reach, the
// five elemental protections, the five damage-kind resistances, the step
// rate and the sight radius (spec "Derived set"). It is a value of arrays
// and integers -- comparable, copied by assignment -- so a test can pin a
// whole recompute with one == and a caller cannot half-apply one.
//
// PROTECTION AND RESISTANCE ARE FIVE WIDE, MATCHING data.UnitDef, NOT SIX.
// The block each family lives in on the original carries six slots; the
// sixth of each -- ahead of the five a shipped column fills and this routine
// derives -- is filled by no column and derived by nothing, so a reader must
// not mistake the width here for a truncation.
//
// HealthMax AND ManaMax ARE WIRED, AS OF 0119-chargen, OFF A GENERATED
// CHARACTER'S BASE ROW -- and, as of 0133-person-health, off a PLACED
// PERSON'S row too (HumanDef.DerivedMaximum, humandef.go), through this same
// graph and no other. The three inputs Profile states come from the shipped
// Humans row a character's Profile method reads (HumanDef.Profile,
// chargenbase.go): ManaColumn and HealthColumn are the row's own mana and
// health maxima being nonzero, and Fighter is the row's own mana maximum
// being zero -- not a decomposition of the row's TypeID column;
// chargenbase.go's own doc carries the finding that TypeID cannot supply an
// archetype at all.
type Derived struct {
	Body, Reaction, Mind, Spirit int32

	// Skill is the six levels AS RESTORED (FR-6a): slots 1 to 5 are the
	// incoming level, held to the training cap, plus the loadout's own bonus,
	// within [SkillFloor, the effective bound] (rules.EffectiveSkill); slot 0
	// passes through untouched, because the
	// original's restore loop and its clamp loop both run 1 to 5 and slot
	// 0 is in neither. It is what the active-skill terms above read, so a
	// bonus that raises a level raises the to-hit and the damage base with
	// it.
	Skill [SkillSlots]int32

	// SkillXP is the per-slot experience -- REAL STORAGE in the original,
	// a dword per slot, not a display of the level. Experience is its sum.
	//
	// A LEVEL AND ITS EXPERIENCE ARE TWO SPELLINGS OF ONE VALUE, joined by S(n)
	// = ftol((1.1^n - 1) * 1000) and by S's inverse, and BOTH directions are
	// live in the original. This routine writes one of them: level to
	// experience. The other -- experience to level -- is the seam this tree
	// does not carry, and it is a seam rather than an absence: the loss path
	// takes a tenth off every slot's experience and then puts the resulting
	// LEVEL back into the level array through that inverse. What is unread is
	// the inverse's own arithmetic and which slot a gain is credited to, not
	// whether either exists.
	//
	// The sum is over ALL SIX slots, including slot 0, which the restore
	// and the clamp both skip. That asymmetry is the original's and is not
	// smoothed out here.
	SkillXP    [SkillSlots]int32
	Experience int32

	HealthMax, ManaMax                                  int32
	Combat                                              Combat
	Protection                                          [5]int32
	Resistance                                          [5]int32
	Speed, Sight                                        int32
	HealthRegeneration, ManaRegeneration, RotationSpeed int32
	SecondaryDamage                                     SecondaryDamage

	// SpeedModifier is the signed modifier part of Speed: the worn items'
	// speed effects. Speed less this is the base the overload penalty and
	// its floor act on before the modifier is added (rules.HumanSpeed).
	SpeedModifier int32

	// Capacity is how much this character may carry before the overload penalty
	// applies: `Body x 10 + 1` off the CAPPED Body, and never a column
	// (HERO-SIGHT-007, High; the multiply by ten and the add of one are at
	// L03918 and L03919). It is the actor field `+0x92`.
	//
	// IT IS IN THE LOAD'S OWN UNITS, which are the `Data.bin` weight column's:
	// a shipped Plate Cuirass states 100 and a Body of 41 gives a capacity of
	// 411, so a capacity is roughly four heavy pieces. The display divides by
	// ten and the comparison does not.
	//
	// SPEED ABOVE IS THE UNENCUMBERED SPEED. The penalty is not applied
	// here, because this graph is not handed a container: pkg/sim applies it
	// where the mover's rate is read, from this Capacity and the load it
	// maintains itself (moverSpeed, pkg/sim/world.go).
	//
	// A CAPACITY IS NEVER ZERO for a character this graph derives -- the
	// `+1` sees to that even at Body 0 -- which is what lets pkg/sim read a
	// zero as "no capacity stated" for a creature no recompute ran for.
	Capacity int32
}

// logBase11 is log base 1.1 of x -- the pool graph's own inverse of pow11,
// computed the ordinary way a logarithm to a non-e, non-10 base is computed
// without a dedicated libm entry point: math.Log(x) / math.Log(statBase).
//
// THE MEASUREMENT 0119-chargen OWED IS DONE, over two domains
// (poolmargin_test.go, AC-13). Over the WHOLE reachable integer experience
// range -- 0 to 6*ftol((pow11(100)-1)*skillXPScale), the six skill slots
// each maxed -- the worst nonzero distance any pool's own
// L(e)*classMultiplier stands from an integer is 1.7763568394002505e-15, at
// e=1050 with multiplier 1. That is not the generic float-noise floor: it is
// one of exactly THREE experience values in the whole range where
// e/poolXPDivisor+1 is exactly 1.1^k as a real number -- e = 500, 1050,
// 1655, for k = 1, 2, 3, because e = (1.1^k-1)*poolXPDivisor is an integer
// only for k that small, a fact of poolXPDivisor's own factorisation and not
// of this routine. At k=1 this tree's logBase11 lands on the integer
// exactly; at k=2 and k=3 it sits a few ULPs BELOW the integer the true
// logarithm hits, so ftol answers one less than the real exponent there -- 1
// where exact arithmetic says 2, 2 where it says 3 -- and a
// differently-rounded log could answer the true value instead.
//
// OVER THE RANGE A GENERATED CHARACTER CAN ACTUALLY REACH, the risk above
// does not apply. Generation trains exactly one skill slot, at level 10 or
// level 20, the other five at zero, so the only experience values a
// Recompute this tree runs ever sees are 0, 1593 and 5727 -- and the
// smallest nonzero margin among them is 0.008861348791139534, ten orders of
// magnitude clear of the whole-range worst case above. A hero this tree
// generates cannot land on, or near, one of the three risky boundaries; only
// a value nothing in this tree constructs could.
func logBase11(x float64) float64 { return math.Log(x) / math.Log(statBase) }

func SkillXPFor(level int32) int32 {
	return rules.Rules{}.SkillXPExtended(level)
}

// RepairSkillLevel is the load-time repair of one slot's stored level against
// its stored experience. A slot at level L holds experience in
// (S(L-1), S(L)]. A save may hold experience above S(L+1), which the award
// sink would then spend one level per award. Such a slot takes the level its
// experience implies, the smallest level whose threshold is not below the
// experience. A level that is zero or at the original cap, and experience not
// above the next threshold, return the level unchanged.
func RepairSkillLevel(level, xp int32) int32 {
	if level <= 0 || level >= 100 || xp <= SkillXPFor(level+1) {
		return level
	}
	repaired := level + 1
	for repaired < 100 && xp > SkillXPFor(repaired) {
		repaired++
	}
	return repaired
}

// SightWord is a Human's sight word in 1/256 cell: the fractional part
// ((mind+reaction)/25 + 4) carries, and the whole-cell byte is replaced by
// cells, the value the derive reports after equipment and effects.
func SightWord(mind, reaction int32, cells int32) uint16 {
	raw := SightSubCells(mind, reaction)
	raw += cells<<8 - raw&^0xff
	if raw <= 0 || raw > 0xffff {
		return uint16(max(0, min(cells, 255)) << 8)
	}
	return uint16(raw)
}

// Recompute is a native character's derived set: his four statistics, his
// six skill levels, his profile and what he is wearing, through DeriveHuman.
// It reads nothing but its arguments. The speed it reports is the
// unencumbered sum; pkg/sim applies the load (rules.NativeHumanSpeed).
func (h Hero) Recompute(p Profile, l Loadout) Derived {
	return h.recompute(p, l, nil)
}

// RecomputeWithSkillXP is the live-character form of Recompute. A created
// character begins with S(level) in every slot, but later awards make the six
// stored experiences independent of the six stored levels. Pool derivation
// reads their running total, so a live recompute must supply those canonical
// counters rather than reconstructing an opening value from levels.
func (h Hero) RecomputeWithSkillXP(p Profile, l Loadout, live [SkillSlots]int32) Derived {
	return h.recompute(p, l, &live)
}

func (h Hero) recompute(p Profile, l Loadout, live *[SkillSlots]int32) Derived {
	// A slot at level L accounts for S(L) experience unless the caller holds
	// the live per-slot counters; all six slots are summed, slot 0 included.
	var skillXP [SkillSlots]int32
	var experienceSum int64
	for i, level := range h.Skill {
		if live != nil {
			skillXP[i] = (*live)[i]
		} else {
			skillXP[i] = l.Rules.SkillXPExtended(level)
		}
		experienceSum += int64(skillXP[i])
	}
	experience := int32(min(experienceSum, math.MaxInt32))

	// The weapon's additive part joins the worn items' terms before the
	// clamps, as the original's modifier block holds both (HERO-MOD-016);
	// its cadence, reach and ranged component stay FoldWeapon's.
	bare := Combat{AttackChargeTime: BareChargeTime, AttackRelaxTime: BareRelaxTime, Reach: 1, SkillSlot: activeSkill(l.Weapon)}
	combat := FoldWeapon(bare, l.Weapon, h.Skill[SkillGeneral])
	mod := l.Mod
	in := HumanInput{Skill: h.Skill, Active: activeSkill(l.Weapon), Experience: experience,
		Fighter: p.Fighter, ManaPool: p.ManaColumn, Rider: p.Rider, TrainingCap: l.Rules.SkillCap(),
		Terms: HumanTerms{HealthMax: mod.HealthMax, ManaMax: mod.ManaMax, Sight: mod.Sight << 8, SkillBonus: mod.SkillBonus,
			ToHit: mod.ToHit + combat.ToHit, DamageBase: mod.DamageBase + combat.DamageBase, DamageSpread: mod.DamageSpread + combat.DamageSpread,
			Defence: mod.Defence + combat.Defence, Absorption: mod.Absorption + combat.Absorption,
			Protection: mod.Protection, Resistance: mod.Resistance}}
	// A native stat already holds its bonus, at most 100, as an effect writes
	// it; the bonus raises the cap by the same amount (HERO-CAP-015).
	for i, pair := range [4][2]int32{{h.Body, mod.Body}, {h.Reaction, mod.Reaction}, {h.Mind, mod.Mind}, {h.Spirit, mod.Spirit}} {
		in.Stat[i], in.StatCap[i] = effectStat(pair[0], pair[1]), pair[1]
	}
	// A native Human holds the unencumbered sum; pkg/sim applies the load at
	// its own speed producer (rules.NativeHumanSpeed).
	out, _ := DeriveHuman(in)
	skill := out.Skill
	// DIV-2784: the General bonus raises a native hero's live General level.
	skill[SkillGeneral] += mod.SkillBonus[SkillGeneral]

	combat.ToHit, combat.DamageBase, combat.DamageSpread = out.ToHit, int32(out.DamageBase), int32(out.DamageSpread)
	combat.Defence, combat.Absorption = out.Defence, out.Absorption
	if mod.HasSecondaryDamage {
		combat.SecondaryDamage = mod.SecondaryDamage
	}
	return Derived{
		Body: out.Stat[0], Reaction: out.Stat[1], Mind: out.Stat[2], Spirit: out.Stat[3],
		Skill:              skill,
		SkillXP:            skillXP,
		Experience:         experience,
		HealthMax:          out.HealthMax,
		ManaMax:            out.ManaMax,
		Combat:             combat,
		Protection:         out.Protection,
		Resistance:         out.Resistance,
		Speed:              out.BaseSpeed + mod.Speed,
		SpeedModifier:      mod.Speed,
		Sight:              out.Sight >> 8,
		HealthRegeneration: mod.HealthRegeneration,
		ManaRegeneration:   mod.ManaRegeneration,
		RotationSpeed:      l.RotationSpeed + mod.RotationSpeed,
		SecondaryDamage:    combat.SecondaryDamage,
		Capacity:           out.Capacity,
	}
}

func effectStat(base, bonus int32) int32 {
	v := capStat(base) + bonus
	if v > 100 {
		return 100
	}
	return v
}

// SkillLevelFor is S's inverse: the largest level n in [0, 100] whose curve
// value S(n) = ftol((1.1^n - 1) * 1000) is at most xp. It is a bounded
// downward scan, not a solve — it walks n from 100 to 0 and returns the
// first one whose S(n) does not exceed xp, computing S(n) with the same
// pow11 and the same ftol truncation step 2 above already uses for the
// forward direction, so the two cannot come to disagree about where a
// boundary falls. There is no tolerance anywhere in it and no float is ever
// compared against a float: the comparison at each step is between xp and a
// truncated int32, exactly as step 2 produces one.
//
// THIS INVERSE IS AUTHORED, NOT DECODED. The original carries its own
// experience-to-level routine — the loss path reads it, taking a tenth off a
// slot's experience and putting the resulting level back through it — and
// that routine's own arithmetic is undecoded: nobody has read it. What
// licenses writing one here anyway is what IS decoded about S, not a guess
// at the original's bytes: S is non-decreasing over [0, 100] (every step
// argument fed to pow11 only grows) and a level is clamped to that same
// range (SkillFloor, SkillCap, Vocabulary's own "Level of a slot"). Under
// those two facts "the largest n with S(n) <= xp" has exactly one possible
// value for any xp — there is no second monotone, clamped curve through the
// same 101 points that this scan could disagree with. Getting the
// ORIGINAL's bytes right would still take reading its routine; getting THIS
// DEFINITION right does not, because the definition has only one witness.
//
// S(0) = 0, so for any xp >= 0 the scan always returns from inside the loop
// — level 0 is always a candidate, and the trailing return is unreachable
// for such an xp.
func SkillLevelFor(xp int32) int32 {
	return rules.Rules{}.SkillLevelFor(xp)
}

// ClampPools is the recompute's last step: a live health and a live mana,
// clamped to this Derived's own maxima. A zero mana column therefore leaves
// a character at zero mana THROUGH THE MAXIMUM -- ManaMax comes out 0 when
// p.ManaColumn is false and nothing feeds it otherwise -- not through a
// second rule written here.
//
// It is a method on Derived rather than two fields on it, because the
// derived set is what Recompute produces and a live health or mana is state
// the caller holds; putting the clamp here keeps this step inside the one
// implementation without Derived pretending to own a pool.
func (d Derived) ClampPools(health, mana int32) (int32, int32) {
	if health > d.HealthMax {
		health = d.HealthMax
	}
	if mana > d.ManaMax {
		mana = d.ManaMax
	}
	return health, mana
}
