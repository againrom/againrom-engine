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

// Recompute is the one implementation of the derived-stat graph in this
// tree: a character's four statistics, his six skill levels, the profile he
// carries and what he is wearing, to the whole Derived set. In the original
// it is a single virtual method; Hero.Derive, Hero.Speed and Hero.Sight are
// now one-line accessors over this and compute nothing of their own.
//
// It reads nothing but its three arguments: no clock, no generator, no
// global, no file. The same three arguments yield the same Derived value in
// every process.
//
// THE ORDER IS NORMATIVE, and four points in it are load-bearing:
//
//  1. the statistic CAPS run first, so every later term in this function
//     reads the capped value rather than the raw one;
//  2. the two POOLS are computed before the SKILLS ARE RESTORED, so a
//     bonus that raises a level reaches a pool only through the next
//     recompute -- the restore, and the damage/to-hit terms that read it,
//     both run afterwards (FR-6a);
//  3. the protection/resistance/defence/absorption BLOCK IS CLEARED after
//     defence's own input (Reaction) is read and before defence is
//     written, so nothing here accumulates across two recomputes over the
//     same Hero (AC-3);
//  4. the EQUIPMENT FOLD runs between the bare values and the final
//     protection clamp, so the clamp bounds the equipped character and not
//     the bare statistics.
//
// ABSORPTION HAS NO SOURCE BUT ARMOUR. No weapon term writes it -- see the
// equipment fold below -- and the block clears it to zero, so a character
// carrying only a weapon absorbs nothing. That is the value this routine
// produces for him, not a gap in it.
//
// HealthMax AND ManaMax ARE WIRED, AS OF 0119-chargen, OFF A GENERATED
// CHARACTER'S BASE ROW, and as of 0133-person-health off a PLACED PERSON'S
// row too: see Derived's own doc for what supplies the three inputs Profile
// states, and for the health arm's own gate below, which reads no column at
// either end.
//
// EACH OF Protection AND Resistance IS FIVE WIDE, matching data.UnitDef,
// because the block's own sixth slot is filled by no column and derived by
// nothing: see Derived's own doc.
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
	// Step 1: every later term reads the CAPPED statistic, so capping runs
	// first and destructively -- capStat is the same routine Hero.Derive,
	// Hero.Speed and Hero.Sight used to call separately, each carrying its own
	// copy of this step; here there is exactly one.
	body := effectStat(h.Body, l.Mod.Body)
	reaction := effectStat(h.Reaction, l.Mod.Reaction)
	mind := effectStat(h.Mind, l.Mod.Mind)
	spirit := effectStat(h.Spirit, l.Mod.Spirit)

	// Step 2: the level-to-experience direction. Slot n at level L accounts for
	// ftol((1.1^L - 1) * 1000); a level of 0 needs no special case, because
	// pow11(0) is 1 and (1 - 1) * 1000 truncates to 0 on its own. All six slots
	// are summed, slot 0 included -- see Derived.SkillXP for why that differs
	// from the restore's own 1..5, and for the inverse direction this tree does
	// not carry.
	//
	// IT READS THE INCOMING LEVELS, NOT THE RESTORED ONES, and that is the
	// order rather than an oversight: the restore is step 6 below, and the
	// pools that consume this sum are steps 3 and 4, so a bonus that raises a
	// level reaches the pools only through the NEXT recompute. That is the
	// second load-bearing ordering point.
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

	healthClassMult := float64(1)
	if p.Fighter {
		healthClassMult = classMult
	}
	healthMax := int32(0)
	if gate := float64(body) * healthClassMult; gate != 0 {
		health := float64(ftol(gate + logBase11(float64(experience)/poolXPDivisor+1)*healthClassMult))
		healthMax = ftol(health * (pow11(body)/poolGrowthDivisor + 1))
	}

	// Step 4: the mana maximum, the same three steps on Spirit -- with three
	// asymmetries against health, each the original's rather than a convenience
	// of ours.
	manaMax := int32(0)
	if p.ManaColumn {
		manaXPMult := float64(classMult)
		if p.Fighter {
			manaXPMult = 1
		}
		mana := float64(spirit) * classMult
		mana = float64(ftol(mana + logBase11(float64(experience)/poolXPDivisor+1)*manaXPMult))
		manaMax = ftol(mana * (pow11(spirit)/poolGrowthDivisor + 1))
	}
	healthMax += l.Mod.HealthMax
	manaMax += l.Mod.ManaMax

	// Step 5: speed and sight, from the already-capped statistics. No ordering
	// point past step 1 constrains them; they are placed where the original
	// reads their inputs.
	//
	// SPEED is `Reaction` below 12 and `Reaction/5 + 12` at 12 and above, plus
	// RiderSpeedBonus for a rider class (Profile.Rider). The overload penalty
	// is not here: this graph is handed no container, so the unencumbered
	// speed is produced and pkg/sim applies the penalty where the mover's rate
	// is read (moverSpeed, pkg/sim/world.go).
	speed := reaction
	if reaction >= speedBranch {
		speed = reaction/speedDivisor + speedBranch
	}
	speed += p.SpeedBonus()

	// SIGHT is `(Mind + Reaction)/25 + 4`. IT IS AN INTEGER DIVISION HERE AND A
	// FLOATING-POINT ONE THERE, and the two give the SAME answer rather than
	// nearly the same. The published expression is `ftol(((mind + reaction)/25
	// + 4) * 256)` stored as a sixteen-bit value in SUB-CELL units, of which
	// the whole-cell radius is the HIGH BYTE -- and `floor(trunc(256x)/256) ==
	// floor(x)` for a non-negative x, so taking the high byte of the scaled
	// truncation is taking the floor of the unscaled expression; the addend is
	// an integer, so the floor distributes over it. That is why this package's
	// stay outside the determinism wall costs nothing here: there is no
	// rounding case where the FPU sequence and this integer one part company.
	//
	// THE SUB-CELL REMAINDER IS DROPPED, which is the divergence rather
	// than a simplification: the low byte of that word is a real value
	// with no decoded consumer; a story that finds one carries it, and
	// until then a field for it would be state nothing can exercise.
	sight := (mind+reaction)/sightDivisor + sightBase
	speed += l.Mod.Speed
	sight += l.Mod.Sight

	// Step 6: the damage pair, to-hit and the active-skill terms, unchanged in
	// value from what Hero.Derive already produced. The spread is ftol(1.1^Body
	// / 20) and the base is a COPY of it, so the bare pair is equal and the
	// roll is [d, 2d]; to-hit is built from Body and Reaction together. The
	// active skill, when there is one, adds three times its level to to-hit and
	// a FIFTH of its level to the base ALONE -- the asymmetry is the reason the
	// two ends of the pair do not carry the same terms.
	spread := ftol(pow11(body) / damageDivisor)
	base := spread
	toHit := ftol((pow11(body) + pow11(reaction)) / toHitDivisor)

	// Step 6a (FR-6a): the SKILL RESTORE, and it lands here -- after the
	// bare to-hit and before the active-skill terms that read it -- because
	// that is where the original puts it. Slots 1 to 5 are the incoming
	// level plus the loadout's bonus, clamped to [SkillFloor, SkillCap];
	// SLOT 0 IS IN NEITHER LOOP and passes through whatever it arrived as,
	// which is the original's own asymmetry and not a shortcut (see
	// SkillFloor's doc for the second clamp this tree folds into this one).
	//
	// A LEVEL IS AN INTEGER. There is nothing fractional to carry, so the
	// arithmetic is integer end to end and no truncation question arises
	// here the way it does everywhere the exponential is involved.
	skill := h.Skill
	skill[SkillGeneral] += l.Mod.SkillBonus[SkillGeneral]
	for i := SkillGeneral + 1; i < SkillSlots; i++ {
		skill[i] = l.Rules.EffectiveSkill(h.Skill[i], l.Mod.SkillBonus[i])
	}

	if slot := activeSkill(l.Weapon); slot > SkillGeneral && slot < SkillSlots {
		toHit += skillToHitMult * skill[slot]
		base += skill[slot] / skillDamageDiv
	}

	// Step 7 (AC-3): the block clears here. In the original this is a literal
	// memset that runs between defence's own input (Reaction, already captured
	// above as `reaction`) being read and defence being written next, in step 8
	// -- so on a SECOND recompute over the same Hero, nothing from the first
	// pass survives into this one. Declaring fresh zero-valued locals
	// reproduces exactly that: there is nothing here to accumulate onto.
	defence := int32(0)
	absorption := int32(0)
	var protection, resistance [5]int32

	// Step 8: defence and the five protections are written; absorption and the
	// five resistances are left at the block's cleared zero -- absorption
	// because no weapon term writes it (see step 9), the resistances because
	// they are NEVER RE-DERIVED for a character at all; the block's clear above
	// is their only writer.
	defence = reaction / defenceDivisor
	for i := range protection {
		// Go's integer division already truncates toward zero for a
		// non-negative operand, which spirit always is once capped --
		// the same rounding control ftol enforces explicitly everywhere
		// else in this graph.
		protection[i] = spirit / spiritHalfDivisor
	}

	// Step 9: the equipment fold. l.Mod's additive fields fold in first,
	// including the two arrays. Its replacement-only SecondaryDamage waits
	// until after the weapon. The weapon folds through FoldWeapon, whose own
	// doc carries which of its two arms adds what: melee joins the physical
	// fields, while ranged uses General and may build the third component. A
	// later elemental item effect then replaces that complete triple, following
	// Weapon::Equip's base-before-effect order. A nil weapon leaves the bare
	// pair (BareChargeTime, BareRelaxTime) standing and Reach at its own floor
	// of 1, which is FoldWeapon's own no-op.
	toHit += l.Mod.ToHit
	base += l.Mod.DamageBase
	spread += l.Mod.DamageSpread
	defence += l.Mod.Defence
	absorption += l.Mod.Absorption
	if defence < 0 {
		defence = 0
	}
	if absorption < 0 {
		absorption = 0
	}
	for i := range protection {
		protection[i] += l.Mod.Protection[i]
		resistance[i] += l.Mod.Resistance[i]
	}

	combat := Combat{
		DamageBase:       base,
		DamageSpread:     spread,
		ToHit:            toHit,
		Defence:          defence,
		Absorption:       absorption,
		AlwaysHits:       false,
		AttackChargeTime: BareChargeTime,
		AttackRelaxTime:  BareRelaxTime,
		Reach:            1,
		// A nil weapon needs this seed because FoldWeapon's nil arm is a
		// complete no-op. A resolved weapon assigns SkillSlot again inside
		// FoldWeapon from the same activeSkill helper; that shared assignment
		// is also what the UnitDef creature path uses.
		SkillSlot: activeSkill(l.Weapon),
	}
	// HERO-GENERAL-092: the General bonus block is not restored onto live slot
	// 0. Pass the Hero's live General level, not l.Mod.SkillBonus[0].
	combat = FoldWeapon(combat, l.Weapon, h.Skill[SkillGeneral])
	if l.Mod.HasSecondaryDamage {
		combat.SecondaryDamage = l.Mod.SecondaryDamage
	}

	// Step 10: each of the five protections clamps to the smaller of Spirit/2 +
	// 70 and 100, and then floors at 0. Two guards, because the first alone
	// does nothing for a protection an EquipMod pushed negative -- it is
	// already below the ceiling -- and the second alone would not bound a
	// character whose Spirit is high enough to raise the ceiling past 100 in
	// principle (StatCap's own cap tops out at 50 ordinarily, 100 through an
	// effect no hero this tree builds has met).
	for i := range protection {
		if ceiling := spirit/spiritHalfDivisor + protectionClampBase; protection[i] > ceiling {
			protection[i] = ceiling
		}
		if protection[i] < 0 {
			protection[i] = 0
		} else if protection[i] > protectionClampCeiling {
			protection[i] = protectionClampCeiling
		}
	}

	return Derived{
		Body: body, Reaction: reaction, Mind: mind, Spirit: spirit,
		Skill:              skill,
		SkillXP:            skillXP,
		Experience:         experience,
		HealthMax:          healthMax,
		ManaMax:            manaMax,
		Combat:             combat,
		Protection:         protection,
		Resistance:         resistance,
		Speed:              speed,
		SpeedModifier:      l.Mod.Speed,
		Sight:              sight,
		HealthRegeneration: l.Mod.HealthRegeneration,
		ManaRegeneration:   l.Mod.ManaRegeneration,
		RotationSpeed:      l.RotationSpeed + l.Mod.RotationSpeed,
		SecondaryDamage:    combat.SecondaryDamage,
		Capacity:           body*capacityMultiplier + capacityAddend,
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
