package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func modSpellFront() *FrontEnd {
	row := func(name string, mana, rng, dmin, dmax int32) dbEntry {
		p := make([]int32, 19)
		for i := range p {
			p[i] = -1
		}
		p[1], p[2], p[4], p[6], p[8], p[16], p[17], p[18] = mana, 1, 1, rng, 1, dmin, dmax, 0
		return dbEntry{name: name, params: p}
	}
	rows := dbCollection{{}, row("Fire Arrow", 3, 7, 4, 8), row("Fire Ball", 9, 10, 6, 12)}
	f := &FrontEnd{}
	f.Table = &mapload.Table{Spells: rows}
	return f
}

func spellEdits(rows ...mod.SpellRow) mod.SpellData {
	for i := range rows {
		rows[i].Mod, rows[i].File, rows[i].Line = "m", mod.SpellsFile, 3+i
	}
	return mod.SpellData{Rows: rows}
}

func TestSetModSpellsRefusalsNameModFileAndLine(t *testing.T) {
	set := func(v int32) mod.SpellInt { return mod.SpellInt{Set: true, Val: v, Line: 8} }
	for name, c := range map[string]struct {
		row  mod.SpellRow
		want string
	}{
		"no spell":  {mod.SpellRow{Target: "Fire Storm", Mana: set(1)}, `mod "m": data/spells.toml:3: spell "Fire Storm" is not a row of the spell table`},
		"no damage": {mod.SpellRow{Target: "Fire Ball", DamageMin: set(1), DamageMax: set(2)}, ""},
	} {
		err := modSpellFront().SetModSpells(spellEdits(c.row))
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := (&FrontEnd{}).SetModSpells(spellEdits(mod.SpellRow{Target: "a", Mana: mod.SpellInt{Set: true}})); err == nil {
		t.Error("a front end without a table accepted an edit")
	}
	if err := (&FrontEnd{}).SetModSpells(mod.SpellData{}); err != nil {
		t.Errorf("no edit is refused: %v", err)
	}
}

func TestSetModSpellsSurvivesSetModsInEitherOrder(t *testing.T) {
	f := modSpellFront()
	if err := f.SetModSpells(spellEdits(mod.SpellRow{Target: "Fire Ball", Mana: mod.SpellInt{Set: true, Val: 2}})); err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(sim.Rules{}, mod.Set{}, false); err != nil {
		t.Fatal(err)
	}
	if r := mapload.SpellRules(f.Table); r[1].ManaCost != 2 {
		t.Errorf("SetMods dropped the spell edits: %+v", r[1])
	}
}

func TestTheSpellbookAndTooltipShowTheEditedValues(t *testing.T) {
	names := map[uint16]string{1: "Fire Arrow", 2: "Fire Ball"}
	mage := func(rules []sim.SpellRule) sim.Entity {
		e := sim.Entity{Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
		sim.LearnBookSpell(&e, 1, rules)
		sim.LearnBookSpell(&e, 2, rules)
		return e
	}
	lines := func(f *FrontEnd) [][]string {
		rules := mapload.SpellRules(f.Table)
		var out [][]string
		for _, s := range spellbookOf(sim.Rules{}, mage(rules), rules, names, testSpellPopupWords(), nil) {
			out = append(out, s.Info)
		}
		return out
	}
	plain := lines(modSpellFront())
	wantPlain := [][]string{
		{"Fire Arrow", "Mana cost: 3", "Damage: 4-8", "Range: 7"},
		{"Fire Ball", "Mana cost: 9", "Damage: 6-12", "Range: 10"},
	}
	if !reflect.DeepEqual(plain, wantPlain) {
		t.Fatalf("unmodded popups %q", plain)
	}

	f := modSpellFront()
	d := spellEdits(mod.SpellRow{Target: "Fire Ball", Mana: mod.SpellInt{Set: true, Val: 20}, Range: mod.SpellInt{Set: true, Val: 14},
		DamageMin: mod.SpellInt{Set: true, Val: 7}, DamageMax: mod.SpellInt{Set: true, Val: 13}})
	d.Global.Mana = mod.SpellRatio{Set: true, Num: 1, Den: 3}
	if err := f.SetModSpells(d); err != nil {
		t.Fatal(err)
	}
	got := lines(f)
	want := [][]string{
		{"Fire Arrow", "Mana cost: 1", "Damage: 4-8", "Range: 7"},
		{"Fire Ball", "Mana cost: 20", "Damage: 7-13", "Range: 14"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("edited popups %q, want %q", got, want)
	}
}
