package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

func formulaEdit(target string, edit func(*mod.SpellRow)) mod.SpellData {
	row := mod.SpellRow{Target: target}
	edit(&row)
	return spellEdits(row)
}

func spellTable(line int, vals ...int32) mod.SpellTable {
	return mod.SpellTable{Set: true, Vals: vals, Line: line}
}

func TestFormulaTablesReachTheRulesInEitherOrderOfSetMods(t *testing.T) {
	d := formulaEdit("Fire Ball", func(r *mod.SpellRow) { r.DamageFactor = spellTable(4, 60) })
	for _, setModsFirst := range []bool{true, false} {
		f := modSpellFront()
		var err error
		if setModsFirst {
			if err = f.SetMods(sim.Rules{}, mod.Set{}, false); err == nil {
				err = f.SetModSpells(d)
			}
		} else if err = f.SetModSpells(d); err == nil {
			err = f.SetMods(sim.Rules{}, mod.Set{}, false)
		}
		if err != nil {
			t.Fatal(err)
		}
		if factor, ok := f.Table.Rules.SpellFormulaAt(0+1, 2, 0); !ok || factor != 60 {
			t.Errorf("SetMods first %v: factor %d %v", setModsFirst, factor, ok)
		}
		w, err := mapload.FromALMWith(&alm.Map{Width: 10, Height: 10}, f.Table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		if !w.Rules().HasSpellFormulas() {
			t.Errorf("SetMods first %v: the world runs without the tables", setModsFirst)
		}
	}
	plain := modSpellFront()
	if err := plain.SetModSpells(spellEdits(mod.SpellRow{Target: "Fire Ball", Mana: mod.SpellInt{Set: true, Val: 2}})); err != nil {
		t.Fatal(err)
	}
	if plain.Table.Rules.HasSpellFormulas() {
		t.Error("an edit without tables froze formulas")
	}
}

func TestFormulaTableRefusalFromSetModSpellsNamesModFileAndLine(t *testing.T) {
	d := formulaEdit("Fire Arrow", func(r *mod.SpellRow) { r.DurationFactor = spellTable(9, 1000) })
	err := modSpellFront().SetModSpells(d)
	if err == nil || err.Error() != `mod "m": data/spells.toml:9: Fire Arrow has no power-scaled duration, so duration_factor does not apply to it` {
		t.Errorf("%v", err)
	}
}

func TestThePopupStatesTheTabledDamageRangeDurationAndMagnitude(t *testing.T) {
	f := modSpellFront()
	d := formulaEdit("Fire Arrow", func(r *mod.SpellRow) {
		r.DamageFactor = spellTable(4, 60)
		r.RangeBonus = spellTable(5, 6)
		r.Power = spellTable(6, 0, 0, 40)
	})
	if err := f.SetModSpells(d); err != nil {
		t.Fatal(err)
	}
	rules := mapload.SpellRules(f.Table)
	r := mapload.TableRules(f.Table)
	mage := sim.Entity{Mind: 1, Book: sim.Spellbook{State: sim.BookPresent}}
	mage.Skill[1] = 1
	sim.LearnBookSpell(&mage, 1, rules)
	sim.RefreshBook(r, &mage, rules)
	names := map[uint16]string{1: "Fire Arrow"}
	got := spellbookOf(r, mage, rules[:1], names, testSpellPopupWords(), nil)[0].Info
	want := []string{"Fire Arrow", "Mana cost: 3", "Damage: 8-16", "Range: 13"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tabled popup %q, want %q", got, want)
	}
	got = spellbookOf(sim.Rules{}, mage, rules[:1], names, testSpellPopupWords(), nil)[0].Info
	if want := []string{"Fire Arrow", "Mana cost: 3", "Damage: 17-34", "Range: 13"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the same book without the tables %q, want %q", got, want)
	}
}

func TestCaptionsReadTheMagnitudeTableAndTheRayRule(t *testing.T) {
	words := testSpellPopupWords()
	words.Hover[182] = "Speed"
	words.Hover[183] = "Protection"
	words.Hover[184] = "Scan"
	words.Hover[185] = "To-hit"
	words.Hover[186] = "Rays"
	words.Hover[217] = "Absorption"
	e := sim.Entity{Mind: 70}
	e.Skill[1], e.Skill[2], e.Skill[3] = 100, 100, 100
	var set rules.SpellFormulaSet
	for id, mag := range map[int]int32{5: 30, 7: -9, 12: 4, 18: 8, 23: 55, 24: 6} {
		set.Spell[rules.FormulaMagnitude][id] = []int32{mag}
	}
	tabled, err := sim.Rules{}.WithSpellFormulas(set)
	if err != nil {
		t.Fatal(err)
	}
	caption := func(r sim.Rules, rule sim.SpellRule) string {
		lines := spellInfoLines(rule, sim.SpellCharacteristicsFor(r, e, rule), "x", &words)
		return lines[len(lines)-1]
	}
	school := func(id uint16, school uint8) sim.SpellRule {
		return sim.SpellRule{ID: id, School: school, SpellDuration: 1}
	}
	for _, c := range []struct {
		rule            sim.SpellRule
		original, table string
	}{
		{school(24, 1), "Speed: 10", "Speed: 6"},
		{school(7, 1), "Speed: -10", "Speed: -9"},
		{school(5, 1), "Protection: +70", "Protection: +30"},
		{school(12, 1), "Scan: -5", "Scan: -4"},
		{school(23, 3), "To-hit: +132%", "To-hit: +55%"},
		{school(18, 3), "Absorption: 17", "Absorption: 8"},
	} {
		if got := caption(sim.Rules{}, c.rule); got != c.original {
			t.Errorf("spell %d unmodded caption %q, want %q", c.rule.ID, got, c.original)
		}
		if got := caption(tabled, c.rule); got != c.table {
			t.Errorf("spell %d tabled caption %q, want %q", c.rule.ID, got, c.table)
		}
	}
	rays := sim.SpellRule{ID: 14, School: 2, TargetsUnit: true, Damaging: true, DamageMin: 1, DamageMax: 2}
	if got := caption(sim.Rules{}, rays); got != "Rays: 7" {
		t.Errorf("unmodded rays caption %q", got)
	}
	rays.Rays = 40
	if got := caption(sim.Rules{}, rays); got != "Rays: 40" {
		t.Errorf("a capped spray printed %q, want the rule's 40", got)
	}
}
