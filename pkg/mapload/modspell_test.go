package mapload_test

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func modSpellParams(mana, rng, radius, areaLife, dur, dmin, dmax int32) []int32 {
	p := spellRow(mana, 1, 1, rng, dmin, dmax, 0)
	p[9], p[11], p[14] = radius, areaLife, dur
	return p
}

func modSpellTable() *mapload.Table {
	rows := defCollection{{}}
	add := func(name string, params []int32, effect string) {
		e := defEntry{name: name, params: params}
		if effect != "" {
			e.strings = []string{effect}
		}
		rows = append(rows, e)
	}
	add("Fire Arrow", modSpellParams(3, 7, -1, -1, -1, 4, 8), "")
	add("Fire Ball", modSpellParams(9, 10, 3, 5, -1, 6, 12), "")
	add("Wall of Fire", modSpellParams(11, 8, -1, 20, -1, 2, 3), "")
	add("Row Four", modSpellParams(0, 0, -1, -1, -1, -1, -1), "")
	add("Row Five", modSpellParams(7, 5, -1, -1, 30, -1, -1), "absorbtion=2:duration 50")
	add("Heal", modSpellParams(10, 6, -1, -1, -1, 10, 20), "")
	add("Freezing Cloud", modSpellParams(8, 9, 2, 15, -1, -1, -1), "speed=-3:duration 40")
	add("Poison Cloud", modSpellParams(12, 9, 2, 15, -1, -1, -1), "health=-2:continuous 6")
	add("Row Nine", modSpellParams(1, 1, -1, -1, -1, -1, -1), "")
	add("Row Ten", modSpellParams(1, 1, -1, -1, -1, -1, -1), "")
	add("Drain Life", modSpellParams(6, 5, -1, -1, -1, 4, 6), "")
	return &mapload.Table{Spells: rows}
}

func withSpells(d mod.SpellData) *mapload.Table {
	t := modSpellTable()
	t.Mods.Spells = d
	return t
}

func ratio(num, den int64) mod.SpellRatio { return mod.SpellRatio{Set: true, Num: num, Den: den} }
func spellInt(v int32) mod.SpellInt       { return mod.SpellInt{Set: true, Val: v} }

func ruleByID(t *testing.T, rules []sim.SpellRule, id int) sim.SpellRule {
	t.Helper()
	if id < 1 || id > len(rules) {
		t.Fatalf("no rule %d in %d rows", id, len(rules))
	}
	return rules[id-1]
}

func TestAnUnmoddedTableIsTheInstalledTable(t *testing.T) {
	plain := mapload.SpellRules(modSpellTable())
	if len(plain) != 11 {
		t.Fatalf("%d rows", len(plain))
	}
	withEmpty := modSpellTable()
	withEmpty.Mods.Spells = mod.SpellData{}
	if got := mapload.SpellRules(withEmpty); !reflect.DeepEqual(got, plain) {
		t.Errorf("an empty edit changes the table:\n%+v\n%+v", got, plain)
	}
	if got := mapload.SpellRules(nil); got != nil {
		t.Errorf("a nil table yields %v", got)
	}
}

func TestEachRowKeyChangesItsOwnColumnOnly(t *testing.T) {
	base := mapload.SpellRules(modSpellTable())
	for name, c := range map[string]struct {
		row     mod.SpellRow
		target  int
		check   func(r sim.SpellRule) bool
		changed int
	}{
		"mana":          {mod.SpellRow{Target: "Fire_Ball", Mana: spellInt(20)}, 2, func(r sim.SpellRule) bool { return r.ManaCost == 20 }, 1},
		"range":         {mod.SpellRow{Target: "Fire Ball", Range: spellInt(14)}, 2, func(r sim.SpellRule) bool { return r.MaxRange == 14 }, 1},
		"radius":        {mod.SpellRow{Target: "Fire Ball", Radius: spellInt(5)}, 2, func(r sim.SpellRule) bool { return r.Radius == 5 }, 1},
		"duration":      {mod.SpellRow{Target: "Row Five", Duration: spellInt(9)}, 5, func(r sim.SpellRule) bool { return r.SpellDuration == 9 }, 1},
		"area_duration": {mod.SpellRow{Target: "Wall of Fire", AreaLife: spellInt(33)}, 3, func(r sim.SpellRule) bool { return r.AreaDuration == 33 }, 1},
		"damage": {mod.SpellRow{Target: "Fire Ball", DamageMin: spellInt(7), DamageMax: spellInt(13)}, 2,
			func(r sim.SpellRule) bool { return r.DamageMin == 7 && r.DamageMax == 13 && r.Damaging }, 2},
		"effect": {mod.SpellRow{Target: "Freezing Cloud", Effect: &mod.SpellEffectEdit{Kind: "health", Mode: "continuous",
			Magnitude: spellInt(-9), Duration: spellInt(77)}}, 7, func(r sim.SpellRule) bool {
			return r.EffectKind == sim.EffectHealth && r.EffectMode == sim.EffectContinuous && r.EffectMagnitude == -9 && r.EffectDuration == 77
		}, 4},
	} {
		got := mapload.SpellRules(withSpells(mod.SpellData{Rows: []mod.SpellRow{c.row}}))
		if !c.check(ruleByID(t, got, c.target)) {
			t.Errorf("%s: row %d is %+v", name, c.target, ruleByID(t, got, c.target))
		}
		for i := range base {
			if i+1 != c.target && !reflect.DeepEqual(got[i], base[i]) {
				t.Errorf("%s: row %d changed: %+v", name, i+1, got[i])
			}
		}
		a, b := ruleByID(t, got, c.target), ruleByID(t, base, c.target)
		changed := 0
		for f := 0; f < reflect.TypeOf(a).NumField(); f++ {
			if !reflect.ValueOf(a).Field(f).Equal(reflect.ValueOf(b).Field(f)) {
				changed++
			}
		}
		if changed != c.changed {
			t.Errorf("%s: %d fields differ from the installed row, want %d", name, changed, c.changed)
		}
	}
}

func TestEffectEditKeepsTheRowsOwnFieldsItDoesNotName(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Freezing Cloud", Effect: &mod.SpellEffectEdit{Magnitude: spellInt(-8)}}}}
	r := ruleByID(t, mapload.SpellRules(withSpells(d)), 7)
	if r.EffectKind != sim.EffectSpeed || r.EffectMode != sim.EffectDuration || r.EffectMagnitude != -8 || r.EffectDuration != 40 {
		t.Errorf("%+v", r)
	}
}

func TestPoisonDurationKeepsTheWordShift(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Poison Cloud", Effect: &mod.SpellEffectEdit{Duration: spellInt(4095)}}}}
	r := ruleByID(t, mapload.SpellRules(withSpells(d)), 8)
	if r.EffectDuration != 4095<<4 {
		t.Errorf("duration word %d", r.EffectDuration)
	}
}

func TestGlobalMultipliersRoundTowardZeroAndClamp(t *testing.T) {
	d := mod.SpellData{Global: mod.SpellGlobal{
		Damage: ratio(3, 2), Heal: ratio(1, 3), Mana: ratio(1, 2), Range: ratio(3, 2), Radius: ratio(3, 1), Duration: ratio(2, 1),
	}}
	got := mapload.SpellRules(withSpells(d))
	arrow := ruleByID(t, got, 1)
	if arrow.ManaCost != 1 || arrow.MaxRange != 10 || arrow.DamageMin != 6 || arrow.DamageMax != 12 {
		t.Errorf("arrow %+v", arrow)
	}
	ball := ruleByID(t, got, 2)
	if ball.Radius != 9 || ball.ManaCost != 4 || ball.AreaDuration != 10 {
		t.Errorf("ball %+v", ball)
	}
	if w := ruleByID(t, got, 3); w.AreaDuration != 40 || w.DamageMin != 3 || w.DamageMax != 4 {
		t.Errorf("wall %+v", w)
	}
	five := ruleByID(t, got, 5)
	if five.SpellDuration != 60 || five.EffectDuration != 100 {
		t.Errorf("five %+v", five)
	}
	heal := ruleByID(t, got, 6)
	if heal.DamageMin != 3 || heal.DamageMax != 6 || !heal.Restorative || heal.Damaging {
		t.Errorf("heal %+v", heal)
	}
	if drain := ruleByID(t, got, 11); drain.DamageMin != 6 || drain.DamageMax != 9 || drain.Damaging || drain.Restorative {
		t.Errorf("drain %+v", drain)
	}
	if four := ruleByID(t, got, 4); four.ManaCost != 0 || four.MaxRange != 0 || four.Damaging {
		t.Errorf("a zero stays zero: %+v", four)
	}
	if c := ruleByID(t, got, 7); c.EffectDuration != 80 || c.EffectMagnitude != -3 {
		t.Errorf("a global leaves magnitude alone: %+v", c)
	}

	big := mod.SpellData{Global: mod.SpellGlobal{Damage: ratio(1000000, 1), Mana: ratio(1000000, 1), Range: ratio(1000000, 1),
		Radius: ratio(1000000, 1), Duration: ratio(1000000, 1)}}
	got = mapload.SpellRules(withSpells(big))
	if r := ruleByID(t, got, 2); r.ManaCost != mod.SpellMaxMana || r.MaxRange != mod.SpellMaxRange || r.Radius != mod.SpellMaxRadius ||
		r.DamageMax != mod.SpellMaxDamage || r.AreaDuration != mod.SpellMaxAreaLife {
		t.Errorf("clamp %+v", r)
	}
	if r := ruleByID(t, got, 5); r.SpellDuration != mod.SpellMaxDuration || r.EffectDuration != mod.SpellMaxDuration {
		t.Errorf("duration clamp %+v", r)
	}
}

func TestAZeroMultiplierThatEmptiesTheDamagePairClearsTheFlag(t *testing.T) {
	r := ruleByID(t, mapload.SpellRules(withSpells(mod.SpellData{Global: mod.SpellGlobal{Damage: ratio(0, 1)}})), 1)
	if r.Damaging || r.DamageMax != 0 {
		t.Errorf("%+v", r)
	}
}

func TestARowKeyIsFinalAndTheGlobalScalesTheRest(t *testing.T) {
	d := mod.SpellData{
		Global: mod.SpellGlobal{Mana: ratio(2, 1), Damage: ratio(2, 1)},
		Rows:   []mod.SpellRow{{Target: "Fire Arrow", Mana: spellInt(5), DamageMin: spellInt(1), DamageMax: spellInt(2)}},
	}
	got := mapload.SpellRules(withSpells(d))
	if a := ruleByID(t, got, 1); a.ManaCost != 5 || a.DamageMin != 1 || a.DamageMax != 2 {
		t.Errorf("row keys are not final: %+v", a)
	}
	if b := ruleByID(t, got, 2); b.ManaCost != 18 || b.DamageMax != 24 {
		t.Errorf("global did not reach the other rows: %+v", b)
	}
}

func TestTwoRowsOfOneTargetBothApply(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{
		{Target: "Fire Ball", Mana: spellInt(1)},
		{Target: "Fire_Ball", Range: spellInt(2)},
	}}
	r := ruleByID(t, mapload.SpellRules(withSpells(d)), 2)
	if r.ManaCost != 1 || r.MaxRange != 2 {
		t.Errorf("%+v", r)
	}
}

func TestModSpellRefusals(t *testing.T) {
	table := modSpellTable()
	for name, c := range map[string]struct {
		row  mod.SpellRow
		line int
		want string
	}{
		"unknown":    {mod.SpellRow{Line: 4, Target: "Fire Storm", Mana: spellInt(1)}, 4, `spell "Fire Storm" is not a row`},
		"no pair":    {mod.SpellRow{Line: 4, Target: "Row Four", DamageMin: spellInt(1), DamageMax: mod.SpellInt{Set: true, Val: 2, Line: 7}}, 7, "Row Four has no damage or healing pair"},
		"kind alone": {mod.SpellRow{Line: 4, Target: "Row Four", Effect: &mod.SpellEffectEdit{Line: 9, Kind: "speed"}}, 9, "has no effect mode to keep"},
	} {
		line, msg := mapload.ModSpellRefusal(table, c.row)
		if line != c.line || !strings.Contains(msg, c.want) {
			t.Errorf("%s: line %d %q, want line %d %q", name, line, msg, c.line, c.want)
		}
	}
	good := mod.SpellRow{Target: "Row Four", Effect: &mod.SpellEffectEdit{Kind: "speed", Mode: "duration"}}
	if line, msg := mapload.ModSpellRefusal(table, good); msg != "" {
		t.Errorf("a kind with a mode is refused: %d %q", line, msg)
	}
	if _, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Target: "Row Five", Effect: &mod.SpellEffectEdit{Kind: "speed"}}); msg != "" {
		t.Errorf("a row that keeps its own mode is refused: %q", msg)
	}
	if _, msg := mapload.ModSpellRefusal(nil, good); msg == "" {
		t.Error("a nil table accepts an edit")
	}
}

func TestEveryModEffectNameResolves(t *testing.T) {
	for _, k := range mod.SpellEffectKinds() {
		d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Row Five", Effect: &mod.SpellEffectEdit{Kind: k}}}}
		r := ruleByID(t, mapload.SpellRules(withSpells(d)), 5)
		if (k == "none") != (r.EffectKind == sim.EffectNone) {
			t.Errorf("kind %s gives %v", k, r.EffectKind)
		}
	}
	for _, m := range mod.SpellEffectModes() {
		d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Row Five", Effect: &mod.SpellEffectEdit{Mode: m}}}}
		if r := ruleByID(t, mapload.SpellRules(withSpells(d)), 5); r.EffectMode == 0 {
			t.Errorf("mode %s gives none", m)
		}
	}
}

func TestAWorldAndABookTakeTheEditedRows(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Arrow", Mana: spellInt(21), Range: spellInt(33)}}}
	table := withSpells(d)
	w, err := mapload.FromALMWith(&alm.Map{Width: 10, Height: 10}, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	if r := ruleByID(t, w.Spells(), 1); r.ManaCost != 21 || r.MaxRange != 33 {
		t.Errorf("world row %+v", r)
	}
	e := sim.Entity{Book: sim.Spellbook{State: sim.BookPresent}}
	sim.LearnBookSpell(&e, 1, mapload.SpellRules(table))
	if s := e.Book.Slots[0]; s.ManaCost != 21 || s.Range != 33 {
		t.Errorf("book slot %+v", s)
	}
}

func TestTheDomainCeilingsSurviveTheBookSlot(t *testing.T) {
	d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Arrow", Mana: spellInt(mod.SpellMaxMana), Range: spellInt(mod.SpellMaxRange)}}}
	rules := mapload.SpellRules(withSpells(d))
	e := sim.Entity{Book: sim.Spellbook{State: sim.BookPresent}}
	sim.LearnBookSpell(&e, 1, rules)
	got, ok := sim.BookRuleFor(e, rules[0])
	if !ok || got.ManaCost != mod.SpellMaxMana || got.MaxRange != mod.SpellMaxRange {
		t.Errorf("book rule %+v", got)
	}
	slot := sim.BookSlotValues(sim.Rules{}, e, rules)[0]
	if slot.Range != mod.SpellMaxRange || slot.ManaCost != mod.SpellMaxMana {
		t.Errorf("saved slot %+v", slot)
	}
}

func targetFilterTable(d mod.SpellData) *mapload.Table {
	rows := defCollection{{}}
	add := func(name string, params []int32) {
		rows = append(rows, defEntry{name: name, params: params})
	}
	ball := modSpellParams(9, 10, 3, 5, -1, 6, 12)
	ball[8] = 2
	add("Fire Arrow", modSpellParams(3, 7, -1, -1, -1, 4, 8))
	add("Fire Ball", ball)
	add("Row Three", modSpellParams(1, 1, -1, -1, -1, -1, -1))
	add("Row Four", modSpellParams(1, 1, -1, -1, -1, -1, -1))
	add("Row Five", modSpellParams(1, 1, -1, -1, -1, -1, -1))
	add("Heal", modSpellParams(10, 6, -1, -1, -1, 10, 20))
	return &mapload.Table{Spells: rows, Mods: mapload.ModContext{Spells: d}}
}

func TestTargetFilterKeysReachTheirRowsOnly(t *testing.T) {
	yes := mod.SpellBool{Set: true, Val: true}
	d := mod.SpellData{Rows: []mod.SpellRow{
		{Target: "Heal", HealHostile: yes},
		{Target: "Fire_Arrow", SelfCast: yes},
		{Target: "Fire_Ball", SelfCast: yes, AreaHits: mod.SpellChoice{Set: true, Val: "hostile"}},
	}}
	plain := mapload.SpellRules(targetFilterTable(mod.SpellData{}))
	got := mapload.SpellRules(targetFilterTable(d))
	for i := range got {
		want := plain[i]
		switch got[i].ID {
		case 6:
			want.HealHostile = true
		case 1:
			want.SelfCast = true
		case 2:
			want.SelfCast, want.AreaHits = true, sim.AreaHitsHostile
		}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("row %d: %+v, want %+v", got[i].ID, got[i], want)
		}
	}
	d.Rows[2].AreaHits.Val = "not_own"
	if r := mapload.SpellRules(targetFilterTable(d))[1]; r.AreaHits != sim.AreaHitsNotOwn {
		t.Errorf("not_own gave %d", r.AreaHits)
	}
	d.Rows[2].AreaHits.Val = "all"
	if r := mapload.SpellRules(targetFilterTable(d))[1]; r.AreaHits != sim.AreaHitsAll {
		t.Errorf("all gave %d", r.AreaHits)
	}
	off := mod.SpellData{Rows: []mod.SpellRow{{Target: "Heal", HealHostile: mod.SpellBool{Set: true}}}}
	if !reflect.DeepEqual(mapload.SpellRules(targetFilterTable(off)), plain) {
		t.Error("heal_hostile = false changed the table")
	}
}

func TestTargetFilterRefusalsNameTheRowAndLine(t *testing.T) {
	table := targetFilterTable(mod.SpellData{})
	yes := mod.SpellBool{Set: true, Val: true, Line: 6}
	hits := mod.SpellChoice{Set: true, Val: "hostile", Line: 8}
	for name, c := range map[string]struct {
		row  mod.SpellRow
		line int
		want string
	}{
		"heal on a damaging row": {mod.SpellRow{Line: 4, Target: "Fire Ball", HealHostile: yes}, 6, "Fire Ball does not heal, so heal_hostile"},
		"heal on a plain row":    {mod.SpellRow{Line: 4, Target: "Row Three", HealHostile: yes}, 6, "Row Three does not heal"},
		"self on the heal":       {mod.SpellRow{Line: 4, Target: "Heal", SelfCast: yes}, 6, "Heal does not damage, so self_cast"},
		"area on a point row":    {mod.SpellRow{Line: 4, Target: "Fire Arrow", AreaHits: hits}, 8, "Fire Arrow is not an area spell, so area_hits"},
	} {
		line, msg := mapload.ModSpellRefusal(table, c.row)
		if line != c.line || !strings.Contains(msg, c.want) {
			t.Errorf("%s: line %d %q, want line %d %q", name, line, msg, c.line, c.want)
		}
	}
	for _, row := range []mod.SpellRow{
		{Target: "Heal", HealHostile: yes}, {Target: "Fire Arrow", SelfCast: yes},
		{Target: "Fire Ball", SelfCast: yes, AreaHits: hits},
	} {
		if line, msg := mapload.ModSpellRefusal(table, row); msg != "" {
			t.Errorf("%s refused: %d %q", row.Target, line, msg)
		}
	}
}
