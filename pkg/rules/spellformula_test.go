package rules

import (
	"strings"
	"testing"
)

func table(n int, v int32) []int32 {
	out := make([]int32, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestSpellFormulasAreAbsentByDefault(t *testing.T) {
	for _, r := range []Rules{{}, Default()} {
		if r.HasSpellFormulas() || !r.IsDefault() {
			t.Errorf("%+v holds formulas", r)
		}
		for k := FormulaPower; k <= FormulaMagnitude; k++ {
			if _, ok := r.SpellFormulaAt(k, 5, 7); ok || r.SpellFormulaTable(k, 5) != nil {
				t.Errorf("formula %s read from the default rules", k)
			}
		}
	}
	empty, err := Default().WithSpellFormulas(SpellFormulaSet{})
	if err != nil || empty.HasSpellFormulas() || !empty.Equal(Default()) {
		t.Errorf("an empty set: %v %+v", err, empty)
	}
}

func TestASpellTableWinsOverTheGlobalOneAndReadsClamped(t *testing.T) {
	var s SpellFormulaSet
	s.Global[FormulaDamage] = []int32{30, 60, 90}
	s.Spell[FormulaDamage][4] = []int32{5}
	s.Spell[FormulaMagnitude][7] = []int32{-3, -4}
	r, err := Default().WithSpellFormulas(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsDefault() || !r.HasSpellFormulas() {
		t.Error("rules with tables read as the original game's")
	}
	for _, c := range []struct {
		id    uint16
		index int32
		want  int32
		ok    bool
	}{{1, 0, 30, true}, {1, 1, 60, true}, {1, 2, 90, true}, {1, 255, 90, true}, {1, -4, 30, true},
		{4, 100, 5, true}, {28, 1, 60, true}, {0, 0, 0, false}, {29, 0, 0, false}} {
		if got, ok := r.SpellFormulaAt(FormulaDamage, c.id, c.index); got != c.want || ok != c.ok {
			t.Errorf("damage of %d at %d: %d %v, want %d %v", c.id, c.index, got, ok, c.want, c.ok)
		}
	}
	if got, ok := r.SpellFormulaAt(FormulaMagnitude, 7, 9); !ok || got != -4 {
		t.Errorf("magnitude %d %v", got, ok)
	}
	if _, ok := r.SpellFormulaAt(FormulaMagnitude, 8, 0); ok {
		t.Error("magnitude leaked to another spell")
	}
	if _, ok := r.SpellFormulaAt(FormulaRange, 1, 0); ok {
		t.Error("an unset formula read as set")
	}
}

func TestFrozenTablesDoNotFollowTheSetAndEqualComparesThem(t *testing.T) {
	var s SpellFormulaSet
	s.Spell[FormulaRange][1] = []int32{1, 2}
	r, err := Default().WithSpellFormulas(s)
	if err != nil {
		t.Fatal(err)
	}
	s.Spell[FormulaRange][1][0] = 99
	if got, _ := r.SpellFormulaAt(FormulaRange, 1, 0); got != 1 {
		t.Errorf("the frozen table follows the set: %d", got)
	}
	other, _ := Default().WithSpellFormulas(s)
	if r.Equal(other) || !r.Equal(r) || r.Equal(Default()) {
		t.Error("Equal ignores the tables")
	}
	s.Spell[FormulaRange][1][0] = 1
	same, _ := Default().WithSpellFormulas(s)
	if !r.Equal(same) {
		t.Error("equal tables compare unequal")
	}
}

func TestSpellFormulaSetRefusals(t *testing.T) {
	for name, c := range map[string]struct {
		set  func(*SpellFormulaSet)
		want string
	}{
		"global magnitude": {func(s *SpellFormulaSet) { s.Global[FormulaMagnitude] = []int32{1} }, "magnitude has no global table"},
		"entry too big":    {func(s *SpellFormulaSet) { s.Spell[FormulaPower][2] = []int32{256} }, "spell 2 power: entry 0 is 256, outside 0..255"},
		"negative range":   {func(s *SpellFormulaSet) { s.Global[FormulaRange] = []int32{0, -1} }, "range_bonus: entry 1 is -1"},
		"too long":         {func(s *SpellFormulaSet) { s.Spell[FormulaDamage][3] = table(257, 30) }, "257 entries, want 1..256"},
		"power too long":   {func(s *SpellFormulaSet) { s.Spell[FormulaPower][3] = table(512, 30) }, "512 entries, want 1..511"},
		"empty":            {func(s *SpellFormulaSet) { s.Spell[FormulaDuration][3] = []int32{} }, "0 entries, want 1..256"},
		"factor too big":   {func(s *SpellFormulaSet) { s.Spell[FormulaDamage][3] = []int32{30001} }, "outside 0..30000"},
		"magnitude low":    {func(s *SpellFormulaSet) { s.Spell[FormulaMagnitude][3] = []int32{-32769} }, "outside -32768..32767"},
	} {
		var s SpellFormulaSet
		c.set(&s)
		_, err := Default().WithSpellFormulas(s)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	var ok SpellFormulaSet
	ok.Spell[FormulaPower][1] = table(511, 255)
	ok.Spell[FormulaDamage][1] = table(256, 30000)
	ok.Spell[FormulaDuration][1] = table(256, 1000000)
	ok.Spell[FormulaMagnitude][1] = []int32{-32768, 32767}
	if _, err := Default().WithSpellFormulas(ok); err != nil {
		t.Errorf("the largest tables: %v", err)
	}
}
