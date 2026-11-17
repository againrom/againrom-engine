package mapload_test

import (
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func secondArmsTable(d mod.SpellData) *mapload.Table {
	rows := defCollection{{}}
	names := map[int]string{11: "Prismatic Spray", 14: "Darkness", 24: "Heal"}
	for i := 1; i <= 26; i++ {
		name := names[i]
		if name == "" {
			name = "Row"
		}
		rows = append(rows, defEntry{name: name, params: modSpellParams(5, 9, 4, -1, -1, 3, 6)})
	}
	return &mapload.Table{Spells: rows, SpellArms: mapload.SecondGameSpellArms, Mods: mapload.ModContext{Spells: d}}
}

func TestSecondGameRaysFollowTheSpellsOwnArm(t *testing.T) {
	rays := mod.SpellInt{Set: true, Val: 5, Line: 7}
	table := secondArmsTable(mod.SpellData{})
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Line: 4, Target: "Prismatic_Spray", Rays: rays}); msg != "" {
		t.Errorf("Prismatic Spray refused: %d %q", line, msg)
	}
	line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Line: 4, Target: "Darkness", Rays: rays})
	if line != 7 || !strings.Contains(msg, "Darkness does not fire rays, so rays does not apply") {
		t.Errorf("Darkness: line %d %q", line, msg)
	}
	row := mod.SpellRow{Target: "Prismatic_Spray", Rays: spellInt(100)}
	got := mapload.SpellRules(secondArmsTable(mod.SpellData{Rows: []mod.SpellRow{row}}))
	if r := got[10]; r.Rays != 100 || r.Radius != 0 {
		t.Errorf("Prismatic Spray rule %+v, want rays 100 radius 0", r)
	}
	if r := got[13]; r.Rays != 0 {
		t.Errorf("Darkness rule %+v kept rays", r)
	}
}

func TestSecondGameHealAndSelfCastFollowTheSpellsOwnArm(t *testing.T) {
	yes := mod.SpellBool{Set: true, Val: true, Line: 6}
	table := secondArmsTable(mod.SpellData{})
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Target: "Heal", HealHostile: yes}); msg != "" {
		t.Errorf("Heal heal_hostile refused: %d %q", line, msg)
	}
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Target: "Heal", SelfCast: yes}); line != 6 || !strings.Contains(msg, "Heal does not damage, so self_cast") {
		t.Errorf("Heal self_cast: line %d %q", line, msg)
	}
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Target: "Prismatic Spray", SelfCast: yes}); msg != "" {
		t.Errorf("Prismatic Spray self_cast refused: %d %q", line, msg)
	}
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Target: "Darkness", HealHostile: yes}); line != 6 || !strings.Contains(msg, "Darkness does not heal, so heal_hostile") {
		t.Errorf("Darkness heal_hostile: line %d %q", line, msg)
	}
	d := mod.SpellData{Rows: []mod.SpellRow{{Target: "Heal", HealHostile: yes}, {Target: "Prismatic Spray", SelfCast: yes}}}
	got := mapload.SpellRules(secondArmsTable(d))
	if !got[23].HealHostile || !got[10].SelfCast {
		t.Errorf("applied heal_hostile %v self_cast %v, want both", got[23].HealHostile, got[10].SelfCast)
	}
}

func TestSecondGameGlobalHealRatioScalesTheSpellsOwnArm(t *testing.T) {
	d := mod.SpellData{Global: mod.SpellGlobal{Heal: mod.SpellRatio{Set: true, Num: 2, Den: 1}}}
	got := mapload.SpellRules(secondArmsTable(d))
	if r := got[23]; r.DamageMin != 6 || r.DamageMax != 12 {
		t.Errorf("Heal pair %d..%d, want 6..12", r.DamageMin, r.DamageMax)
	}
	if r := got[5]; r.DamageMin != 3 || r.DamageMax != 6 {
		t.Errorf("row 6 pair %d..%d, want 3..6", r.DamageMin, r.DamageMax)
	}
}
