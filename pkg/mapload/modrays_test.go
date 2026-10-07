package mapload_test

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func raysTable(d mod.SpellData) *mapload.Table {
	rows := defCollection{{}}
	for i := 1; i <= 13; i++ {
		rows = append(rows, defEntry{name: "Row", params: modSpellParams(1, 1, -1, -1, -1, 1, 2)})
	}
	rows = append(rows, defEntry{name: "Prismatic Spray", params: modSpellParams(5, 9, 4, -1, -1, 3, 6)})
	return &mapload.Table{Spells: rows, Mods: mapload.ModContext{Spells: d}}
}

func TestRaysKeyCapsPrismaticSprayOnly(t *testing.T) {
	plain := mapload.SpellRules(raysTable(mod.SpellData{}))
	row := mod.SpellRow{Target: "Prismatic_Spray", Rays: spellInt(100)}
	got := mapload.SpellRules(raysTable(mod.SpellData{Rows: []mod.SpellRow{row}}))
	for i := range got {
		want := plain[i]
		if got[i].ID == sim.PrismaticSpellID {
			want.Rays, want.Radius = 100, 0
		}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("row %d: %+v, want %+v", got[i].ID, got[i], want)
		}
	}
	if plain[13].Rays != 0 || plain[13].Radius != 4 {
		t.Errorf("unmodded row %+v", plain[13])
	}
}

func TestRaysRefusalNamesTheRowAndLine(t *testing.T) {
	table := raysTable(mod.SpellData{})
	rays := mod.SpellInt{Set: true, Val: 5, Line: 7}
	line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Line: 4, Target: "Row", Rays: rays})
	if line != 7 || !strings.Contains(msg, "Row does not fire rays, so rays does not apply") {
		t.Errorf("line %d %q", line, msg)
	}
	if line, msg := mapload.ModSpellRefusal(table, mod.SpellRow{Line: 4, Target: "Prismatic_Spray", Rays: rays}); msg != "" {
		t.Errorf("Prismatic Spray refused: %d %q", line, msg)
	}
}
