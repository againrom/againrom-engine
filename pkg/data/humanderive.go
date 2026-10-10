package data

import (
	"fmt"

	"againrom/pkg/rules"
)

// HumanTerms are the additive terms of a Human's modifier block: worn items,
// the weapon's additive part and standing effects (HERO-MOD-016). Sight is in
// 1/256 cell.
type HumanTerms struct {
	Speed, Capacity, HealthMax, ManaMax, Sight           int32
	SkillBonus                                           [SkillSlots]int32
	ToHit, DamageBase, DamageSpread, Defence, Absorption int32
	Protection, Resistance                               [5]int32
}

// HumanInput is everything the Human derive reads. Stat holds Body,
// Reaction, Mind and Spirit as stored, StatCap the modifier's signed cap
// terms. Skill holds the live General level in slot 0 and the trained levels
// in slots 1 to 5. Two inputs differ by route: Rider (a stored Human's type
// word, a native Human's Profile.Rider) and the sight term's sub-cell part,
// which only a stored modifier carries.
type HumanInput struct {
	Stat, StatCap            [4]int32
	Skill                    [SkillSlots]int32
	Active                   int32
	Experience               int32
	Fighter, ManaPool, Rider bool
	Load                     int32
	TrainingCap              int32
	Terms                    HumanTerms
}

// HumanOutput is the derived block. Sight is in 1/256 cell; its high byte is
// the whole-cell radius. BaseSpeed is the unencumbered speed before the
// modifier; Speed is the derived word and SpeedModifier the modifier the
// derive keeps. ToHit is the signed word and the damage pair the two bytes
// the original stores (HERO-DAMAGE-022).
type HumanOutput struct {
	Stat                     [4]int32
	HealthMax, ManaMax       int32
	Sight, Capacity          int32
	BaseSpeed                int32
	Speed, SpeedModifier     int32
	Skill                    [SkillSlots]int32
	ToHit                    int32
	DamageBase, DamageSpread uint8
	Defence, Absorption      int32
	Protection, Resistance   [5]int32
}

// HumanBaseSpeed is the unencumbered speed: Reaction below 12, else
// Reaction/5 + 12, plus the rider term (HERO-SPEED-008).
func HumanBaseSpeed(reaction int32, rider bool) int32 {
	speed := reaction
	if reaction >= speedBranch {
		speed = reaction/speedDivisor + speedBranch
	}
	if rider {
		speed += RiderSpeedBonus
	}
	return speed
}

// DeriveHuman is the one Human derive (HERO-ORDER-014). Every Human goes
// through it: native heroes, the party, rearm and placed persons through
// Hero.Recompute, original-SAV, source-backed and town Humans through
// HumanState. Its steps: the stat caps (HERO-CAP-015), the two pools with a
// word between truncations (HERO-HP-005, HERO-MP-006), sight and capacity
// (HERO-SIGHT-007), speed (SAV-1116, rules.HumanSpeed), the damage pair and
// to-hit, the skill restore (DIV-2217), defence and protection, the fold of
// the modifier terms and the final clamps. A mod formula attaches here.
func DeriveHuman(in HumanInput) (HumanOutput, error) {
	var out HumanOutput
	for i, v := range in.Stat {
		out.Stat[i] = min(v, 50+in.StatCap[i])
	}
	body, reaction, mind, spirit := out.Stat[0], out.Stat[1], out.Stat[2], out.Stat[3]
	growth := logBase11(float64(in.Experience)/poolXPDivisor + 1)
	pool := func(base, stat int32, multiplier float64) (int32, error) {
		v, err := humanFTOL(float64(base) + growth*multiplier)
		if err != nil {
			return 0, err
		}
		return humanFTOL(float64(int16(v)) * (pow11(stat)/poolGrowthDivisor + 1))
	}
	healthMultiplier, manaMultiplier := float64(1), float64(classMult)
	if in.Fighter {
		healthMultiplier, manaMultiplier = classMult, 1
	}
	var err error
	if base := int32(int16(body * int32(healthMultiplier))); base != 0 {
		if out.HealthMax, err = pool(base, body, healthMultiplier); err != nil {
			return out, err
		}
	}
	if in.ManaPool {
		if out.ManaMax, err = pool(int32(int16(spirit*classMult)), spirit, manaMultiplier); err != nil {
			return out, err
		}
	}
	if out.Sight, err = humanFTOL((float64(mind+reaction)/float64(sightDivisor) + float64(sightBase)) * 256); err != nil {
		return out, err
	}
	out.Capacity = body*capacityMultiplier + capacityAddend
	out.BaseSpeed = HumanBaseSpeed(reaction, in.Rider)
	speed, kept, ok := rules.HumanSpeed(int16(out.BaseSpeed), int16(in.Terms.Speed), in.Load, out.Capacity)
	if !ok {
		return out, fmt.Errorf("Human derive would divide by zero capacity")
	}
	out.Speed, out.SpeedModifier = int32(speed), int32(kept)
	damage, err := humanFTOL(pow11(body) / damageDivisor)
	if err != nil {
		return out, err
	}
	toHit, err := humanFTOL((pow11(body) + pow11(reaction)) / toHitDivisor)
	if err != nil {
		return out, err
	}
	out.DamageBase, out.DamageSpread = uint8(damage), uint8(damage)
	out.Skill[SkillGeneral] = in.Skill[SkillGeneral]
	for i := SkillGeneral + 1; i < SkillSlots; i++ {
		out.Skill[i] = rules.EffectiveSkill(in.Skill[i], in.Terms.SkillBonus[i], in.TrainingCap)
	}
	if in.Active > SkillGeneral && in.Active < SkillSlots {
		level := out.Skill[in.Active]
		toHit += skillToHitMult * level
		out.DamageBase += uint8(level / skillDamageDiv)
	}
	t := in.Terms
	out.HealthMax += t.HealthMax
	out.ManaMax += t.ManaMax
	out.Sight += t.Sight
	out.Capacity += t.Capacity
	out.ToHit = int32(int16(toHit + t.ToHit))
	out.DamageBase += uint8(t.DamageBase)
	out.DamageSpread += uint8(t.DamageSpread)
	out.Defence = max(0, reaction/defenceDivisor+t.Defence)
	out.Absorption = max(0, t.Absorption)
	for i := range out.Protection {
		out.Protection[i] = max(0, min(protectionClampCeiling, spirit/spiritHalfDivisor+protectionClampBase, spirit/spiritHalfDivisor+t.Protection[i]))
		out.Resistance[i] = t.Resistance[i]
	}
	return out, nil
}
