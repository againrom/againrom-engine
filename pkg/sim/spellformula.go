package sim

import "againrom/pkg/rules"

type SpellFormulaSet = rules.SpellFormulaSet

const (
	FormulaPower     = rules.FormulaPower
	FormulaDamage    = rules.FormulaDamage
	FormulaRange     = rules.FormulaRange
	FormulaDuration  = rules.FormulaDuration
	FormulaMagnitude = rules.FormulaMagnitude
)

func spellPowerUnder(r Rules, rule SpellRule, level, mind int32) int32 {
	if v, ok := r.SpellFormulaAt(rules.FormulaPower, rule.ID, level+mind); ok {
		return v
	}
	return spellPower(level, mind)
}

func spellRecordPowerUnder(r Rules, rule SpellRule, level, mind int32) int32 {
	if v, ok := r.SpellFormulaAt(rules.FormulaPower, rule.ID, level+mind); ok {
		return v
	}
	return spellRecordPower(level, mind)
}

func spellDamageUnder(r Rules, rule SpellRule, power int32) (base, spread int64) {
	f, ok := r.SpellFormulaAt(rules.FormulaDamage, rule.ID, power)
	if !ok {
		return spellDamage(rule.DamageMin, rule.DamageMax, power)
	}
	base = int64(rule.DamageMin) * int64(f) / 30
	return base, int64(rule.DamageMax)*int64(f)/30 - base
}

func rangeBonusFor(r Rules, rule SpellRule, power int32) (int64, bool) {
	v, ok := r.SpellFormulaAt(rules.FormulaRange, rule.ID, power)
	return int64(v), ok
}

func spellRangeUnder(r Rules, rule SpellRule, power int32) int64 {
	if rule.bookInstance || rule.MaxRange == 0 {
		return spellRange(rule, power)
	}
	if bonus, ok := rangeBonusFor(r, rule, power); ok {
		return min(int64(rule.MaxRange)+bonus, 255)
	}
	return spellRange(rule, power)
}

func spellRecordRangeUnder(r Rules, rule SpellRule, power int32) int64 {
	if !rule.bookInstance && (rule.arm() == 26 || rule.MaxRange != 0) {
		if bonus, ok := rangeBonusFor(r, rule, power); ok {
			return min(int64(rule.MaxRange)+bonus, 255)
		}
	}
	return spellRecordRange(rule, power)
}

func spellLastingTicksUnder(r Rules, rule SpellRule, power int32) uint16 {
	f, ok := r.SpellFormulaAt(rules.FormulaDuration, rule.ID, power)
	if !ok {
		return spellLastingTicks(rule, power)
	}
	base := int64(rule.SpellDuration)
	if rule.arm() == 15 {
		base = 3
	}
	if base <= 0 {
		return 0
	}
	return uint16(min(base*16*int64(f)/1000, durationTickCeiling))
}

func spellPointDurationUnder(r Rules, rule SpellRule, power int32) uint16 {
	switch rule.arm() {
	case 15, 5, 10, 16, 18, 20, 22, 23, 24, 27, 28:
		return spellLastingTicksUnder(r, rule, power)
	case 8:
		// The second game's Poison Cloud effect lasts its Duration column on
		// the 1.025 law (R2-ENGINE-024).
		if rule.Second {
			return spellLastingTicksUnder(r, rule, power)
		}
	}
	return rule.EffectDuration
}

func spellRecordDurationUnder(r Rules, rule SpellRule, power int32) uint16 {
	if rule.SpellDuration > 0 {
		return spellLastingTicksUnder(r, rule, power)
	}
	return spellRecordDuration(rule, power)
}

func magnitudeUnder(r Rules, rule SpellRule, power int32) (int32, bool) {
	return r.SpellFormulaAt(rules.FormulaMagnitude, rule.ID, power)
}

func MagnitudeArm(id uint16) bool {
	switch id {
	case 5, 7, 8, 10, 12, 16, 17, 18, 22, 23, 24, 27, 28:
		return true
	}
	return false
}

func DurationArm(id uint16) bool {
	switch id {
	case 5, 10, 15, 16, 18, 20, 22, 23, 24, 27, 28:
		return true
	}
	return false
}

func (w *World) rangeSpan(rule SpellRule) (lo, hi int64) {
	lo, hi = 255, 0
	for p := int32(0); p <= spellPowerMax; p++ {
		v := spellRangeUnder(w.rules, rule, p)
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}
