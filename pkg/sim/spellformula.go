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

// SpellDamagePair is the damage interval a cast at power rolls in: the base
// and the base plus the spread, a negative spread read as zero as the
// damage and heal resolvers read it.
func SpellDamagePair(game rules.SpellGame, r Rules, rule SpellRule, power int32) (minimum, maximum int64) {
	base, spread := spellRowOf(game).damageUnder(r, rule, power)
	return base, base + max(spread, 0)
}

func spellDamageExceedsPayload(s *spellRow, r Rules, rule SpellRule, power int32) bool {
	if !rule.Damaging || !s.wholeDamageColumns(rule) {
		return false
	}
	base, spread := s.damageUnder(r, rule, power)
	return base > 255 || spread > 255
}

func magnitudeUnder(r Rules, rule SpellRule, power int32) (int32, bool) {
	return r.SpellFormulaAt(rules.FormulaMagnitude, rule.ID, power)
}

func (w *World) rangeSpan(rule SpellRule) (lo, hi int64) {
	lo, hi = 255, 0
	for p := int32(0); p <= spellPowerMax; p++ {
		v := spellRangeOf(w.rules, rule, p, rangeCast)
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}
