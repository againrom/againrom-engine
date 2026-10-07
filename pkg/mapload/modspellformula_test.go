package mapload_test

import (
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

func formulaTable(vals ...int32) mod.SpellTable {
	return mod.SpellTable{Set: true, Vals: vals, Line: 5}
}

func TestFormulaTableRefusalsNameTheRowAndLine(t *testing.T) {
	table := modSpellTable()
	at := func(t mod.SpellTable, line int) mod.SpellTable { t.Line = line; return t }
	one := formulaTable(1)
	for name, c := range map[string]struct {
		row  mod.SpellRow
		line int
		want string
	}{
		"damage on a row without a pair":   {mod.SpellRow{Target: "Row Four", DamageFactor: at(one, 6)}, 6, "Row Four has no damage or healing pair, so damage_factor"},
		"range on a row with range 0":      {mod.SpellRow{Target: "Row Four", RangeBonus: at(one, 7)}, 7, "Row Four has range 0, so range_bonus"},
		"duration off the duration arm":    {mod.SpellRow{Target: "Fire Arrow", DurationFactor: at(one, 8)}, 8, "Fire Arrow has no power-scaled duration"},
		"magnitude off the magnitude arms": {mod.SpellRow{Target: "Fire Ball", Magnitude: at(one, 9)}, 9, "Fire Ball has no power-scaled magnitude"},
		"unknown target":                   {mod.SpellRow{Target: "Fire Storm", Power: at(one, 3), Line: 2}, 2, `spell "Fire Storm" is not a row`},
	} {
		line, msg := mapload.ModSpellRefusal(table, c.row)
		if line != c.line || !strings.Contains(msg, c.want) {
			t.Errorf("%s: line %d %q, want line %d %q", name, line, msg, c.line, c.want)
		}
	}
	for name, row := range map[string]mod.SpellRow{
		"power on any row":          {Target: "Row Four", Power: one},
		"damage on a pair row":      {Target: "Fire Arrow", DamageFactor: one},
		"damage on the drain row":   {Target: "Drain Life", DamageFactor: one},
		"range on a ranged row":     {Target: "Fire Arrow", RangeBonus: one},
		"range on a row given one":  {Target: "Row Four", RangeBonus: one, Range: spellInt(3)},
		"duration on Protection":    {Target: "Row Five", DurationFactor: one},
		"magnitude on Protection":   {Target: "Row Five", Magnitude: one},
		"magnitude on Poison Cloud": {Target: "Poison Cloud", Magnitude: one},
	} {
		if line, msg := mapload.ModSpellRefusal(table, row); msg != "" {
			t.Errorf("%s refused: %d %q", name, line, msg)
		}
	}
}

func TestSpellFormulasResolveRowsToIdsAndKeepGlobals(t *testing.T) {
	d := mod.SpellData{
		Global: mod.SpellGlobal{Power: formulaTable(0, 9), RangeBonus: formulaTable(3)},
		Rows: []mod.SpellRow{
			{Target: "Fire Ball", DamageFactor: formulaTable(60), Magnitude: formulaTable(2, 3)},
			{Target: "Row_Five", DurationFactor: formulaTable(2000), Power: formulaTable(7)},
			{Target: "No Such Spell", Power: formulaTable(1)},
		},
	}
	set, err := mapload.SpellFormulas(modSpellTable(), d)
	if err != nil {
		t.Fatal(err)
	}
	if got := set.Global[rules.FormulaPower]; len(got) != 2 || got[1] != 9 || set.Global[rules.FormulaRange][0] != 3 {
		t.Errorf("globals %v", set.Global)
	}
	if set.Spell[rules.FormulaDamage][2][0] != 60 || set.Spell[rules.FormulaMagnitude][2][1] != 3 ||
		set.Spell[rules.FormulaDuration][5][0] != 2000 || set.Spell[rules.FormulaPower][5][0] != 7 {
		t.Errorf("row tables %v", set.Spell)
	}
	if set.Spell[rules.FormulaDamage][1] != nil || set.Spell[rules.FormulaPower][2] != nil {
		t.Error("a table reached a row that did not name it")
	}
	empty, err := mapload.SpellFormulas(modSpellTable(), mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Arrow", Mana: spellInt(2)}}})
	if err != nil || !empty.Empty() {
		t.Errorf("a row without tables: %v %v", err, empty.Empty())
	}
	if _, err := mapload.SpellFormulas(&mapload.Table{}, d); err == nil {
		t.Error("a table with no spell collection resolved rows")
	}
}

func TestTheFrozenRulesDriveTheBookAndTheTooltip(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{
		{Target: "Fire Arrow", RangeBonus: formulaTable(0, 5), DamageFactor: formulaTable(30, 90), Power: formulaTable(0, 0, 1)},
		{Target: "Row Five", DurationFactor: formulaTable(1000, 3000), Magnitude: formulaTable(5, 15)},
	}}
	table := withSpells(d)
	set, err := mapload.SpellFormulas(table, d)
	if err != nil {
		t.Fatal(err)
	}
	r, err := sim.Rules{}.WithSpellFormulas(set)
	if err != nil {
		t.Fatal(err)
	}
	spells := mapload.SpellRules(table)
	mage := sim.Entity{MaxMana: 50, Mind: 1, Book: sim.Spellbook{State: sim.BookPresent}, KnownSpells: 1<<1 | 1<<5}
	mage.Skill[1] = 1
	sim.LearnBookSpell(&mage, 1, spells)
	sim.RefreshBook(r, &mage, spells)
	if got := mage.Book.Slots[0].Range; got != 7+5 {
		t.Errorf("book range %d, want the row's 7 plus the table's 5 at power 1", got)
	}
	arrow := ruleByID(t, spells, 1)
	c := sim.SpellCharacteristicsFor(r, mage, arrow)
	if c.Power != 1 || !c.HasDamageFactor || c.DamageFactor != 90 || c.Range != 12 {
		t.Errorf("fire arrow %+v", c)
	}
	legacy := mage
	legacy.Book = sim.Spellbook{}
	if plain := sim.SpellCharacteristicsFor(sim.Rules{}, legacy, arrow); plain.HasDamageFactor || plain.Range != 7+100/30 {
		t.Errorf("without the tables %+v", plain)
	}
	if tabled := sim.SpellCharacteristicsFor(r, legacy, arrow); tabled.Range != 12 {
		t.Errorf("tabled range without a book cache %+v", tabled)
	}
	protection := sim.SpellCharacteristicsFor(r, mage, ruleByID(t, spells, 5))
	if !protection.HasMagnitude || protection.Magnitude != 5 || protection.Duration != 30*16*3000/1000 {
		t.Errorf("protection %+v", protection)
	}
	weapon := sim.Entity{MaxMana: 1, WeaponSpell: 1, WeaponSpellLevel: 1}
	got, ok := sim.WeaponSpellCharacteristicsFor(r, weapon, spells)
	if !ok || got.DamageMin != 4*90/30 || got.DamageMax != 8*90/30 {
		t.Errorf("tooltip %+v %v", got, ok)
	}
	if base, spread, ok := sim.WeaponSpellDamageFor(sim.Rules{}, weapon, spells); !ok || base != 4*31/30 || base+spread != 8*31/30 {
		t.Errorf("unmodded tooltip %d %d %v", base, spread, ok)
	}
}
