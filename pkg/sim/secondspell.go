package sim

// secondSpellArms maps a second-game spell id to the first-game arm of the
// same spell (R2-ENGINE-019 names the rows, R2-ENGINE-024 groups them by
// arm). Ice Missile, Blizzard, Diamond Dust and Summon have no first-game
// counterpart and run no singular behaviour here (DIV-2617, DIV-2618).
var secondSpellArms = [...]uint16{
	1: 1, 2: 2, 3: 3, 4: 5, 5: ArmNone, 6: 8, 7: ArmNone, 8: 10, 9: 9, 10: 13,
	11: 14, 12: 15, 13: 16, 14: 17, 15: 12, 16: ArmNone, 17: 19, 18: 20, 19: 22,
	20: 23, 21: 24, 22: 25, 23: 26, 24: 6, 25: ArmNone, 26: 11, 27: 18, 28: 27,
	29: 28,
}

// SecondGameSpellArm is the arm a second-game row id runs; an id past the
// published table runs none.
func SecondGameSpellArm(id uint16) uint16 {
	if int(id) < len(secondSpellArms) && id != 0 {
		return secondSpellArms[id]
	}
	return ArmNone
}

// AssignSecondGameArms marks every row of a second-game table with its arm.
func AssignSecondGameArms(rules []SpellRule) {
	for i := range rules {
		rules[i].Arm, rules[i].Second = SecondGameSpellArm(rules[i].ID), true
	}
}

// secondDrawSpan is the span of the second game's 1..n draw: a spread of 0
// still draws 1 (R2-ENGINE-026).
func secondDrawSpan(spread int64) int32 {
	return int32(max(spread, 1) - 1)
}

// secondDrainAmount is the second game's Drain Life roll before the cap:
// (base + U[1, spread]) cut by the target's Astral resistance
// (R2-ENGINE-026). draw is the 0-based draw of secondDrawSpan.
func secondDrainAmount(base int64, astral, draw int32) int64 {
	p := int64(min(max(astral, 0), 100))
	return (base + 1 + int64(draw)) * (100 - p) / 100
}

// secondBlankCurse is the second game's Stone Curse whose Effects string
// parses to no kind: the cast lands and changes nothing (R2-ENGINE-025,
// DIV-2620).
func secondBlankCurse(rule SpellRule) bool {
	return rule.Second && rule.arm() == 20 && rule.EffectKind == EffectNone
}

// SpellRuleLands reports a row this build can land: an applicable row that is
// not a staged area with no stage program. The census reads it; casting
// reaches the same refusal.
func SpellRuleLands(rule SpellRule) bool {
	if !spellApplicable(rule) {
		return false
	}
	return !rule.Area || areaModeFor(rule) != areaModeRing || ringStageCount(rule.arm()) != 0
}

// armSpellID is the row id an attached effect of arm carries. A first-game
// table returns arm itself; a second-game table returns the row running arm,
// or ArmNone, which no effect carries, when no row runs it.
func (w *World) armSpellID(arm uint16) uint16 {
	if len(w.spells) == 0 || !w.spells[0].Second {
		return arm
	}
	for _, r := range w.spells {
		if r.arm() == arm {
			return r.ID
		}
	}
	return ArmNone
}

// HasEffectArm reports whether target carries the lasting effect of the row
// running arm: presentation asks by arm, so a second-game row draws as its
// own effect and never as the first-game spell of the same id.
func (w *World) HasEffectArm(target EntityID, arm uint16) bool {
	return w.hasAttachedSpell(target, w.armSpellID(arm))
}

// SpellArm is the arm the table row id runs; an id no row carries is its own
// arm.
func (w *World) SpellArm(id uint16) uint16 { return w.spellArm(id) }
