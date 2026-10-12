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

func spellDamageUnder(r Rules, rule SpellRule, power int32) (base, spread int64) {
	f, ok := r.SpellFormulaAt(rules.FormulaDamage, rule.ID, power)
	if !ok {
		if !rule.Second {
			return SpellColumnDamage(rule.DamageMin, rule.DamageMax, power)
		}
		return spellDamage(rule.DamageMin, rule.DamageMax, power)
	}
	base = int64(rule.DamageMin) * int64(f) / 30
	return base, int64(rule.DamageMax)*int64(f)/30 - base
}

// SpellDamagePair is the damage interval a cast at power rolls in: the base
// and the base plus the spread, a negative spread read as zero as the
// damage and heal resolvers read it.
func SpellDamagePair(r Rules, rule SpellRule, power int32) (minimum, maximum int64) {
	base, spread := spellDamageUnder(r, rule, power)
	return base, base + max(spread, 0)
}

func (rule SpellRule) wholeDamageColumns() bool {
	return !rule.Second && (rule.DamageMin > 255 || rule.DamageMax > 255)
}

func spellDamageExceedsPayload(r Rules, rule SpellRule, power int32) bool {
	if !rule.Damaging || !rule.wholeDamageColumns() {
		return false
	}
	base, spread := spellDamageUnder(r, rule, power)
	return base > 255 || spread > 255
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
		v := spellRangeOf(w.rules, rule, p, rangeCast)
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}
