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

func (w *World) awardSkill(ai int, named int32, amount int64, srcIdx int) bool {
	a := &w.entities[ai]
	if !a.GainsXP || !InPersistBand(a.TypeID) {
		return false
	}
	if srcIdx >= 0 {
		s := &w.entities[srcIdx]
		// The shared sink bypasses ownership and protected-relation checks only
		// after the source has fallen below zero. A downed source at exactly zero
		// still takes both refusals.
		if s.HP >= 0 {
			if a.Owner == s.Owner {
				return false
			}
			if w.relations.Locked(a.Owner, s.Owner) {
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

	gain := xpGain(amount, a.Mind)
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
	if a.SkillXP[slot] <= w.rules.SkillXP(before) {
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
	if t.Owner == 0 || t.Owner == SelfSlot || t.MaxHP <= 0 {
		return
	}
	named := int32(0)
	if skillMage(w.entities[ai]) {
		named = int32(rule.School)
	}
	w.awardSkill(ai, named, xpRaw(t.ExperienceValue(), amount, t.MaxHP), ti)
}
