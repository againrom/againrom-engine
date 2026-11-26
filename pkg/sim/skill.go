package sim

import "againrom/pkg/rules"

// Rules is the immutable set of game parameters a mod may change. The zero
// Rules is the original game's.
type Rules = rules.Rules

// RulesParams are the parameters a Rules value is built from.
type RulesParams = rules.Params

// NewRules builds the Rules a RulesParams describes.
func NewRules(p RulesParams) (Rules, error) { return rules.New(p) }

// SetRules replaces the world's rules. A world is built with the original
// game's; a launch under a mod calls this once, before the first tick.
func (w *World) SetRules(r Rules) { w.rules = r }

// Rules returns the rules the world runs under.
func (w *World) Rules() Rules { return w.rules }

// SetSkillLevels replaces the six levels of a native actor and refreshes the
// range cache of its book. A source-backed actor takes its levels from its own
// source block, never from here.
func (w *World) SetSkillLevels(id EntityID, levels [skillSlots]int32) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 {
		return false
	}
	w.entities[i].Skill = levels
	RefreshBook(w.rules, &w.entities[i], w.spells)
	w.refreshSavedBookRoots(i)
	return true
}

// RepairNativeSkillLevels raises saved skills and their active combat terms.
// Every other current field retains its saved value.
func (w *World) RepairNativeSkillLevels(id EntityID, levels [skillSlots]int32) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 {
		return false
	}
	e := &w.entities[i]
	for j := 1; j < skillSlots; j++ {
		levels[j] = max(e.Skill[j], min(levels[j], w.rules.EffectiveSkillLimit()))
	}
	levels[0] = e.Skill[0]
	j := int(e.XPSlot)
	if j > 0 && j < skillSlots {
		e.ToHit += skillToHitPerLevel * (levels[j] - e.Skill[j])
		e.DamageBase += levels[j]/skillDamageDivisor - e.Skill[j]/skillDamageDivisor
	}
	return w.SetSkillLevels(id, levels)
}

// wornSkillBonus folds the class-selected skill family of the worn items.
func (w *World) wornSkillBonus(i int, slot int32) int32 {
	if i >= len(w.equipment) {
		return 0
	}
	first := uint8(26)
	if skillMage(w.entities[i]) {
		first = 32
	}
	total := int32(0)
	for _, item := range w.equipment[i] {
		for _, effect := range item.Effects {
			if effect.Kind != first+uint8(slot) || slot < 0 || slot > 5 {
				continue
			}
			if effect.Mode == 0 || effect.Mode == 8 {
				total += int32(effect.Operand)
			} else {
				total += int32(int16(effect.Operand))
			}
		}
	}
	return total
}

// trainedSkill is the level a slot has been trained to: a source-backed Human
// keeps it in the base block; a native actor's levels include the worn bonus.
func (w *World) trainedSkill(ai int, slot int32) int32 {
	a := &w.entities[ai]
	if a.ActorLoad.Source.Class == 2 {
		return int32(int16(uint16(a.ActorLoad.Source.Base[2+2*slot]) | uint16(a.ActorLoad.Source.Base[3+2*slot])<<8))
	}
	return a.TrainedSkills(w.nativeSkillBonuses(ai))[slot]
}

// skillAwardProfile is the steps of the skill award that differ by game. The
// first game's award is firstGameSkillAward.
type skillAwardProfile struct {
	// offMainDivisor, when positive, pays ceil(X / offMainDivisor) into a
	// skill that is neither the actor's main skill nor freeSkill.
	offMainDivisor int64
	freeSkill      int32
	// raise: the test that raises a skill (skillRaises).
	raise skillRaiseRule
	// standingVictim: a blow pays only for a victim whose health was above
	// zero before it.
	standingVictim bool
	// victimPay: whose victim a blow pays for (victimPayRules).
	victimPay victimPayRule
	// ownedAttacker: an attacker with no owner is paid nothing for a victim
	// whose health is not below zero.
	ownedAttacker bool
	// spellNeedsMage: spell damage pays a mage alone, and a kill credited
	// to a spell calls no award for a non-mage.
	spellNeedsMage bool
	// standingKiller: the kill route calls no award for a killer whose
	// health is below zero.
	standingKiller bool
	// victimRow: the protected-relation refusal reads the victim's owner's
	// relation toward the attacker's, not the attacker's toward the victim's.
	victimRow bool
	// victimKinds, when positive, is how many victim kinds the kill route
	// counts per attacker; an award for a counted kind pays 10X / (n + 10).
	victimKinds int
	// mind: how an amount is scaled by the actor's Mind (mindScales).
	mind mindScaleRule
}

// mindScaleRule selects a Mind scaling of mindScales.
type mindScaleRule uint8

const (
	mindInteger mindScaleRule = iota
	mindBinary64
)

// mindScales holds each Mind scaling: the integer form and the first game's
// binary64 form.
var mindScales = [...]func(amount int64, mind int32) int64{
	mindInteger:  xpGain,
	mindBinary64: binaryMindGain,
}

// binaryMindGain is the first game's Mind scaling in binary64 on the signed
// Mind word, trunc(X × (Mind/30 + 0.25)), so a product just below a whole
// number truncates to the one under it (HERO-209).
func binaryMindGain(amount int64, mind int32) int64 {
	return doubleFromInt(amount).mul(doubleQuo(int64(int16(mind)), 30).add(doubleQuo(1, 4))).trunc()
}

// firstGameSkillAward is the first game's award: the refusal reads the
// victim's owner's relation toward the attacker's.
var firstGameSkillAward = skillAwardProfile{victimRow: true, mind: mindBinary64}

// secondGameSkillAward is the second game's award (R2-ENGINE-480,
// R2-ENGINE-486, R2-ENGINE-488, R2-ENGINE-1156, R2-ENGINE-1157,
// R2-ENGINE-1158).
var secondGameSkillAward = skillAwardProfile{offMainDivisor: 8, freeSkill: 5, raise: raiseAtNextLevel,
	standingVictim: true, victimPay: victimPaysComputer, ownedAttacker: true, spellNeedsMage: true, standingKiller: true,
	victimRow: true, victimKinds: maxVictimKinds}

// victimPayRule selects which victims a kill pays for.
type victimPayRule uint8

const (
	victimPaysAny victimPayRule = iota
	victimPaysComputer
)

// victimPayRow holds the refusal of a victimPayRule.
type victimPayRow struct {
	refuses func(w *World, t Entity) bool
}

var victimPayRules = [...]victimPayRow{
	victimPaysAny:      {refuses: func(*World, Entity) bool { return false }},
	victimPaysComputer: {refuses: func(w *World, t Entity) bool { return !w.computerControlledUnit(t) }},
}

// raisesSkill reports whether experience xp raises a skill whose trained
// level is level.
func (w *World) raisesSkill(xp, level int32) bool {
	return w.game.skillAward.raises(w.rules, xp, level)
}

func (p skillAwardProfile) raises(r Rules, xp, level int32) bool {
	return skillRaises[p.raise](r, xp, level)
}

// skillRaiseRule is the experience test that raises a trained level.
type skillRaiseRule uint8

const (
	raisePastOwnLevel skillRaiseRule = iota
	raiseAtNextLevel
)

// skillRaises: the first game raises a skill when its experience passes its
// own level's threshold; the second when it reaches the next level's.
var skillRaises = [...]func(r Rules, xp, level int32) bool{
	raisePastOwnLevel: func(r Rules, xp, level int32) bool { return xp > r.SkillXP(level) },
	raiseAtNextLevel:  func(r Rules, xp, level int32) bool { return xp >= r.SkillXP(level+1) },
}

// HeldSkillLevel is the trained level experience xp holds in the game of
// dialect d: the lowest level the award's raise test does not raise, at most
// the cap. A load repairs a stored level below it.
func HeldSkillLevel(d ScriptDialect, r Rules, xp int32) int32 {
	award := newGameProfile(d).skillAward
	level := int32(0)
	for level < r.SkillCap() && award.raises(r, xp, level) {
		level++
	}
	return level
}

func (w *World) awardSkill(ai int, named int32, amount int64, srcIdx int) bool {
	a := &w.entities[ai]
	if !a.GainsXP || !InPersistBand(a.TypeID) {
		return false
	}
	award := w.game.skillAward
	if srcIdx >= 0 {
		s := &w.entities[srcIdx]
		if w.paysNoReward(*s) && a.Humanoid {
			return false
		}
		// The shared sink bypasses ownership and protected-relation checks only
		// after the source has fallen below zero. A downed source at exactly zero
		// still takes both refusals.
		if !s.Dead() {
			if a.Owner == s.Owner || award.ownedAttacker && a.Owner == 0 {
				return false
			}
			from, to := a.Owner, s.Owner
			if award.victimRow {
				from, to = to, from
			}
			if w.relations.Locked(from, to) {
				return false
			}
		}
	}
	var slot int32
	mage := skillMage(*a)
	if mage {
		if named <= 0 || named >= skillSlots {
			return false
		}
		slot = named
	} else {
		if named != 0 {
			return false
		}
		slot = int32(a.XPSlot)
		if slot == 0 {
			return false
		}
	}
	if a.ActorLoad.Source.Class == 2 && !w.sourceDeriveReady(ai) {
		return false
	}
	if w.NativeTrainingNeedsProducer(a.ID) {
		return false
	}
	// The training cap applies to the trained base, never to the effective level.
	level := w.trainedSkill(ai, slot)
	if level >= w.rules.SkillCap() {
		return false
	}

	gain := mindScales[award.mind](amount, a.Mind)
	if srcIdx >= 0 && award.victimKinds > 0 {
		if n, held := a.victimKinds.find(victimKind(w.entities[srcIdx])); held {
			gain = victimKindCut(gain, n)
		}
	}
	if d := award.offMainDivisor; d > 0 && slot != award.freeSkill && slot != a.MainSkill {
		gain = (gain + d - 1) / d
	}
	if cap := int64(w.rules.SkillXP(level+1) - w.rules.SkillXP(level)); gain > cap {
		gain = cap
	}
	if a.ActorLoad.Source.Class == 2 {
		return w.sourceSkillAward(ai, slot, int32(gain))
	}

	before := level
	a.SkillXP[slot] += int32(gain)
	if a.NativeBasis.ScalarIsKnown(ScalarU130) {
		a.NativeBasis.Scalars[ScalarU130] += uint32(gain)
	}
	if !w.raisesSkill(a.SkillXP[slot], before) {
		return false
	}
	if !a.NativeTraining.Present {
		a.NativeTraining = NativeTraining{Present: true, Levels: a.TrainedSkills(w.nativeSkillBonuses(ai))}
	}
	a.NativeTraining.Levels[slot] = before + 1
	if slot > 0 && a.NativeBasis.BasePresent {
		at, level := 2+2*slot, uint16(before+1)
		a.NativeBasis.Base[at], a.NativeBasis.Base[at+1] = byte(level), byte(level>>8)
		a.NativeBasis.BaseKnown |= uint32(3) << at
	}
	a.Skill[slot] = w.rules.EffectiveSkill(before+1, w.wornSkillBonus(ai, slot))
	RefreshBook(w.rules, a, w.spells)
	w.refreshSavedBookRoots(ai)
	return true
}

// awardSpellDamage is the damage/pseudo-damage feed shared by book and item
// spells. A caster credits the row's school; a fighter leaves the name at zero
// so awardSkill substitutes the currently equipped weapon skill. The victim
// must be an owned non-human-participant actor and must have a health maximum
// for the proportional formula.
func (w *World) awardSpellDamage(ai, ti int, rule SpellRule, amount int64) {
	if ai < 0 || ai >= len(w.entities) || ti < 0 || ti >= len(w.entities) {
		return
	}
	t := &w.entities[ti]
	if !w.computerControlledUnit(*t) || t.MaxHP <= 0 {
		return
	}
	named := int32(0)
	if skillMage(w.entities[ai]) {
		named = int32(rule.School)
	} else if w.game.skillAward.spellNeedsMage {
		return
	}
	w.awardSkill(ai, named, xpRaw(t.ExperienceValue(), amount, t.MaxHP), ti)
}
