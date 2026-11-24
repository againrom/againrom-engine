package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// secondSpellFront is a front end over a second-game spell table: one row
// per name of that game's catalog profile, Ice Missile alone with a damage
// pair of 10 and 10.
func secondSpellFront() *FrontEnd {
	rows := dbCollection{{}}
	for _, s := range rules.SpellProfile(rules.SpellGameSecond) {
		p := make([]int32, 19)
		p[1], p[2], p[4], p[6], p[8], p[14] = 1, 2, 1, 6, 1, 2
		if s.Name == "Ice_Missile" {
			p[16], p[17] = 10, 10
		}
		rows = append(rows, dbEntry{name: strings.ReplaceAll(s.Name, "_", " "), params: p})
	}
	f := &FrontEnd{}
	f.Table = &mapload.Table{Spells: rows, SpellArms: mapload.SecondGameSpellArms}
	return f
}

func secondSpellTablesFixture(t *testing.T) mod.SpellData {
	t.Helper()
	src, err := os.ReadFile("../modrt/testdata/mods/second-spell-tables/data/spells.toml")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ParseSpells("second-spell-tables", mod.SpellsFile, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// secondCast casts spell id from a caster of power mind-30 at victim and
// returns the world on the tick the cast lands: the victim lost health or
// holds an effect.
func secondCast(t *testing.T, f *FrontEnd, mind int32, victim sim.EntityID, id uint16) *sim.World {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, DyingTime: 200, Mind: mind, MaxMana: 50, Mana: 50,
		KnownSpells: 1<<5 | 1<<12, ScanRange: 12}
	target := sim.Entity{ID: 2, X: 5, Y: 2, HP: 100, MaxHP: 100, DyingTime: 200}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, target}, nil, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	w.SetRules(mapload.TableRules(f.Table))
	cmds := []sim.Command{sim.Cast(1, victim, sim.SpellID(id))}
	for n := 0; n < 120; n++ {
		sim.StepObserved(w, cmds)
		cmds = nil
		if e, _ := w.Entity(victim); e.HP < 100 || len(w.ActiveEffects()) > 0 {
			return w
		}
	}
	t.Fatalf("spell %d at Mind %d never landed", id, mind)
	return nil
}

func TestSecondGameFixtureTablesSetTheCastNumbersAtTwoPowers(t *testing.T) {
	d := secondSpellTablesFixture(t)
	for _, c := range []struct {
		modded         bool
		mind           int32
		damage         int32
		invisibleTicks uint16
	}{
		{false, 31, 10, 16}, {false, 32, 10, 32},
		{true, 31, 15, 96}, {true, 32, 30, 240},
	} {
		f := secondSpellFront()
		if c.modded {
			if err := f.SetModSpells(d); err != nil {
				t.Fatal(err)
			}
		}
		if e, _ := secondCast(t, f, c.mind, 2, 5).Entity(2); 100-e.HP != c.damage {
			t.Errorf("mod %v power %d: Ice Missile took %d health, want %d", c.modded, c.mind-30, 100-e.HP, c.damage)
		}
		effects := secondCast(t, f, c.mind, 1, 12).ActiveEffects()
		if len(effects) != 1 || effects[0].Spell != 12 || effects[0].Remaining != c.invisibleTicks {
			t.Errorf("mod %v power %d: Invisibility attached %+v, want %d ticks", c.modded, c.mind-30, effects, c.invisibleTicks)
		}
	}
}

func TestSecondGameFormulaRefusalsNameTheRowFieldAndLimit(t *testing.T) {
	for _, c := range []struct {
		target string
		edit   func(*mod.SpellRow)
		want   string
	}{
		{"Stone_Curse", func(r *mod.SpellRow) { r.DurationFactor = spellTable(9, 1000) },
			"Stone Curse draws its duration from 0 up to a ceiling of power in this game, so duration_factor does not apply to it"},
		{"Fire_Arrow", func(r *mod.SpellRow) { r.DurationFactor = spellTable(9, 1000) },
			"Fire Arrow has no power-scaled duration, so duration_factor does not apply to it"},
		{"Invisibility", func(r *mod.SpellRow) { r.Magnitude = spellTable(9, 4) },
			"Invisibility has no power-scaled magnitude, so magnitude does not apply to it"},
		{"Light", func(r *mod.SpellRow) { r.DamageFactor = spellTable(9, 60) },
			"Light has no damage or healing pair, so damage_factor does not apply to it"},
		{"Poison_Cloud", func(r *mod.SpellRow) { r.DurationFactor = spellTable(9, 2000) },
			"Poison Cloud has no power-scaled duration, so duration_factor does not apply to it"},
		{"Shield", func(r *mod.SpellRow) { r.DurationFactor = spellTable(9, 2000) }, ""},
		{"Slow", func(r *mod.SpellRow) { r.Magnitude = spellTable(9, -4) }, ""},
	} {
		f := secondSpellFront()
		err := f.SetModSpells(formulaEdit(c.target, c.edit))
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", c.target, err)
			}
			continue
		}
		if want := `mod "m": data/spells.toml:9: ` + c.want; err == nil || err.Error() != want {
			t.Errorf("%s: %v, want %s", c.target, err, want)
		}
	}
	f := secondSpellFront()
	if err := f.SetModSpells(formulaEdit("Slow", func(r *mod.SpellRow) { r.Magnitude = spellTable(9, -4) })); err != nil {
		t.Fatal(err)
	}
	if v, ok := f.Table.Rules.SpellFormulaAt(rules.FormulaMagnitude, 29, 0); !ok || v != -4 {
		t.Errorf("the second game's spell 29 magnitude table reads %d %v", v, ok)
	}
	_, err := mod.ParseSpells("m", mod.SpellsFile, []byte("[[spell]]\ntarget = \"Ice_Missile\"\ndamage_factor = [30, 40000]\n"), nil)
	if err == nil || err.Error() != "data/spells.toml:3: damage_factor entry 1 is 40000, outside 0..30000" {
		t.Errorf("an entry past the limit: %v", err)
	}
}
