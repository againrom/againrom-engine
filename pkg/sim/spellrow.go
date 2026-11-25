package sim

import "againrom/pkg/rules"

// spellRow is how a game's spell rows run: the one place the two games'
// amount rolls, damage columns and arm rows differ. spellRows is indexed by
// the World's spell game; no code asks a spell row which game it is from.
type spellRow struct {
	// drain rolls Drain Life's amount and credits its kill.
	drain drainBody
	// damage rolls a direct-damage cast's amount.
	damage func(w *World, ci, vi int, base, spread int64, school uint8) int64
	// heal rolls a healing cast's amount.
	heal func(w *World, base, spread int64) int64
	// columns turns a row's damage columns into the cast's base and spread.
	columns func(dmin, dmax, power int32) (base, spread int64)
	// wholeColumns: columns above 255 stay whole, so a payload byte cannot
	// hold the pair.
	wholeColumns bool
	// poison scales Poison Cloud's point magnitude.
	poison func(mag, power int32) int32
	// armRows: the arm table's second-game rows apply.
	armRows bool
}

// drainBody holds how a game rolls Drain Life's amount and credits the kill.
type drainBody struct {
	amount func(w *World, ti int, base, spread int64) int64
	credit func(w *World, ci, ti int, rule SpellRule)
}

// spellRows holds each game's spell row, filled in init because its bodies
// reach the World's own spell rules. The second game's drain stores the
// caster and the spell as the victim's killer on every hit that takes health
// (R2-ENGINE-1767); the kill route applies its own class, health and owner
// test.
var spellRows [rules.SpellGameSecond + 1]spellRow

func init() {
	spellRows = [rules.SpellGameSecond + 1]spellRow{
		rules.SpellGameFirst: {
			drain: drainBody{
				amount: func(w *World, _ int, base, spread int64) int64 {
					return int64(int16(base + w.firstFromOne(spread)))
				},
				credit: func(*World, int, int, SpellRule) {},
			},
			damage: func(w *World, ci, vi int, base, spread int64, school uint8) int64 {
				return w.directDamage(ci, vi, base, spread, school)
			},
			heal: func(w *World, base, spread int64) int64 {
				return base + w.firstFromOne(spread)
			},
			columns:      SpellColumnDamage,
			wholeColumns: true,
			poison:       PoisonMagnitude,
		},
		rules.SpellGameSecond: {
			drain: drainBody{
				amount: func(w *World, ti int, base, spread int64) int64 {
					return secondDrainAmount(base, w.entities[ti].Protection[4], w.rng.uniform(secondDrawSpan(spread)))
				},
				credit: func(w *World, ci, ti int, rule SpellRule) {
					w.entities[ti].KillCreditSource, w.entities[ti].HasKillCredit = w.entities[ci].ID, true
					w.entities[ti].KillCreditSpell = int8(rule.ID)
				},
			},
			damage: func(w *World, _, vi int, base, spread int64, school uint8) int64 {
				return w.secondSpellDamage(vi, base, spread, school)
			},
			heal: func(w *World, base, spread int64) int64 {
				return base + 1 + int64(w.rng.uniform(secondDrawSpan(spread)))
			},
			columns: spellDamage,
			poison:  func(mag, power int32) int32 { return mag * (power + 45) / 45 },
			armRows: true,
		},
	}
}

// spellRowOf is the row of the game g; a game with no row runs the first.
func spellRowOf(g rules.SpellGame) *spellRow {
	if int(g) < len(spellRows) {
		return &spellRows[g]
	}
	return &spellRows[rules.SpellGameFirst]
}

// spellRow is the row of the game this World runs.
func (w *World) spellRow() *spellRow { return spellRowOf(w.game.spellGame) }

// formSpellRow is the row of the game a decoded form names: the wrapper's
// profile, else the base form's first game.
func (w *World) formSpellRow() *spellRow {
	if w.formGame != nil {
		return spellRowOf(w.formGame.spellGame)
	}
	return spellRowOf(newGameProfile(ScriptROM1).spellGame)
}

// damageUnder is the damage pair a row rolls at power under the rule set r.
func (s *spellRow) damageUnder(r Rules, rule SpellRule, power int32) (base, spread int64) {
	f, ok := r.SpellFormulaAt(rules.FormulaDamage, rule.ID, power)
	if !ok {
		return s.columns(rule.DamageMin, rule.DamageMax, power)
	}
	base = int64(rule.DamageMin) * int64(f) / 30
	return base, int64(rule.DamageMax)*int64(f)/30 - base
}

// wholeDamageColumns reports a row whose damage columns exceed a byte and
// stay whole.
func (s *spellRow) wholeDamageColumns(rule SpellRule) bool {
	return s.wholeColumns && (rule.DamageMin > 255 || rule.DamageMax > 255)
}

// second is the row's second-game arm difference, zero where the game runs
// no second-game arm rows.
func (s *spellRow) second(rule SpellRule) secondArmRow {
	if !s.armRows {
		return secondArmRow{}
	}
	return spellArmOf(rule.arm()).second
}
