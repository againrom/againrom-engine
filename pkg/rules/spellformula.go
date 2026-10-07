package rules

import (
	"errors"
	"fmt"
)

type SpellFormula uint8

const (
	FormulaPower SpellFormula = iota
	FormulaDamage
	FormulaRange
	FormulaDuration
	FormulaMagnitude
	spellFormulas
)

func (f SpellFormula) String() string {
	return [...]string{"power", "damage_factor", "range_bonus", "duration_factor", "magnitude"}[f]
}

const (
	SpellIDs                = 28
	MaxSpellPower     int32 = 255
	MaxPowerSum       int32 = 2 * EffectiveSkillBound
	MaxFormulaEntries       = int(MaxSpellPower) + 1
	MaxDamageFactor   int32 = 30000
	MaxDurationFactor int32 = 1000000
)

func FormulaBounds(f SpellFormula) (lo, hi int32, entries int) {
	switch f {
	case FormulaPower:
		return 0, MaxSpellPower, int(MaxPowerSum) + 1
	case FormulaDamage:
		return 0, MaxDamageFactor, MaxFormulaEntries
	case FormulaRange:
		return 0, 255, MaxFormulaEntries
	case FormulaDuration:
		return 0, MaxDurationFactor, MaxFormulaEntries
	}
	return -32768, 32767, MaxFormulaEntries
}

type SpellFormulaSet struct {
	Global [spellFormulas][]int32
	Spell  [spellFormulas][SpellIDs + 1][]int32
}

func (s SpellFormulaSet) Empty() bool {
	for k := range s.Global {
		if s.Global[k] != nil {
			return false
		}
		for _, t := range s.Spell[k] {
			if t != nil {
				return false
			}
		}
	}
	return true
}

type spellTables [spellFormulas][SpellIDs + 1][]int32

func (r Rules) WithSpellFormulas(s SpellFormulaSet) (Rules, error) {
	if s.Empty() {
		r.spell = nil
		return r, nil
	}
	if s.Global[FormulaMagnitude] != nil {
		return Rules{}, errors.New("magnitude has no global table")
	}
	var out spellTables
	for k := SpellFormula(0); k < spellFormulas; k++ {
		for id := 1; id <= SpellIDs; id++ {
			t := s.Spell[k][id]
			if t == nil {
				t = s.Global[k]
			}
			if t == nil {
				continue
			}
			if err := checkFormula(k, t); err != nil {
				return Rules{}, fmt.Errorf("spell %d %s: %w", id, k, err)
			}
			out[k][id] = append([]int32(nil), t...)
		}
	}
	r.spell = &out
	return r, nil
}

func checkFormula(k SpellFormula, t []int32) error {
	lo, hi, entries := FormulaBounds(k)
	if len(t) == 0 || len(t) > entries {
		return fmt.Errorf("table has %d entries, want 1..%d", len(t), entries)
	}
	for i, v := range t {
		if v < lo || v > hi {
			return fmt.Errorf("entry %d is %d, outside %d..%d", i, v, lo, hi)
		}
	}
	return nil
}

func (r Rules) HasSpellFormulas() bool { return r.spell != nil }

func (r Rules) SpellFormulaTable(k SpellFormula, id uint16) []int32 {
	if r.spell == nil || id < 1 || id > SpellIDs || k >= spellFormulas {
		return nil
	}
	return r.spell[k][id]
}

func (r Rules) SpellFormulaAt(k SpellFormula, id uint16, index int32) (int32, bool) {
	t := r.SpellFormulaTable(k, id)
	if t == nil {
		return 0, false
	}
	return t[max(0, min(index, int32(len(t)-1)))], true
}

func spellTablesEqual(a, b *spellTables) bool {
	if a == nil || b == nil {
		return a == b
	}
	for k := range a {
		for id := range a[k] {
			x, y := a[k][id], b[k][id]
			if len(x) != len(y) {
				return false
			}
			for i := range x {
				if x[i] != y[i] {
					return false
				}
			}
		}
	}
	return true
}
