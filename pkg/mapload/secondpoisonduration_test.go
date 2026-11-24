package mapload_test

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// poisonCloudTable is a table whose row 6 (the second game's Poison Cloud)
// and row 8 (the first game's) carry the shipped Effects string
// `health=-2:continuous 8` (R2-ASSET-178).
func poisonCloudTable(second bool) *mapload.Table {
	rows := defCollection{{}}
	for id := 1; id <= 8; id++ {
		e := defEntry{name: "Row", params: modSpellParams(1, 6, 2, 15, 5, -1, -1)}
		if id == 6 || id == 8 {
			e.strings = []string{"health=-2:continuous 8"}
		}
		rows = append(rows, e)
	}
	t := &mapload.Table{Spells: rows}
	if second {
		t.SpellArms = mapload.SecondGameSpellArms
	}
	return t
}

// The second game's Poison Cloud Effect lasts 8 << 4 = 128 whatever the
// caster's power, the parser's store of the Effects count (R2-ENGINE-1575,
// R2-ENGINE-697); a duration table does not reach it. The first game's Poison
// Cloud keeps its 128 counter.
func TestPoisonCloudEffectLastsItsEffectsCount(t *testing.T) {
	for _, tc := range []struct {
		second bool
		id     int
	}{{true, 6}, {false, 8}} {
		rules := mapload.SpellRules(poisonCloudTable(tc.second))
		rule := rules[tc.id-1]
		if rule.EffectDuration != 128 || rule.EffectMode != sim.EffectContinuous || sim.DurationTableRule(rule) {
			t.Fatalf("second=%v: counter %d mode %d table %v, want 128, continuous, no table", tc.second, rule.EffectDuration, rule.EffectMode, sim.DurationTableRule(rule))
		}
		for _, mind := range []int32{10, 90} {
			if got := poisonCloudLanding(t, rule, mind); got != 128 {
				t.Errorf("second=%v mind %d: the landed Effect lasts %d, want 128", tc.second, mind, got)
			}
		}
	}
}

// poisonCloudLanding casts rule from a caster of the given Mind at a unit
// and returns the remaining duration of the first Effect it lands there.
func poisonCloudLanding(t *testing.T, rule sim.SpellRule, mind int32) uint16 {
	t.Helper()
	id := rule.ID
	caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, Mind: mind, KnownSpells: 1 << id, ScanRange: 10, TokenSize: 1}
	target := caster
	target.ID, target.X, target.HP, target.MaxHP = 2, 4, 10000, 10000
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster, target}, nil, sim.Relations{}, nil, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(id))}
	for tick := 0; tick < 64; tick++ {
		sim.Step(w, commands)
		commands = nil
		for _, e := range w.ActiveEffects() {
			if e.Target == 2 && e.Spell == id {
				return e.Remaining
			}
		}
	}
	t.Fatalf("spell %d landed no Effect on the unit", id)
	return 0
}
