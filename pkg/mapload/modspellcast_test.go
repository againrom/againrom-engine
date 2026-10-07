package mapload_test

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func castTable() *mapload.Table {
	arrow := modSpellParams(3, 7, -1, -1, -1, 4, 8)
	ball := modSpellParams(5, 12, 3, 4, -1, 6, 12)
	ball[8], ball[4] = 2, 0
	return &mapload.Table{Spells: defCollection{{}, {name: "Fire Arrow", params: arrow}, {name: "Fire Ball", params: ball}}}
}

func castWorld(t *testing.T, d mod.SpellData, ents ...sim.Entity) *sim.World {
	t.Helper()
	table := castTable()
	table.Mods.Spells = d
	w, err := sim.NewSpelledWorld(7, sim.Bounds{Width: 24, Height: 24}, sim.ModeCanonical, nil, ents, nil, mapload.SpellRules(table))
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func castEnt(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200, ScanRange: 6}
}

func castMage(id sim.EntityID, x, y int32) sim.Entity {
	e := castEnt(id, x, y)
	e.Mind, e.MaxMana, e.Mana, e.KnownSpells = 60, 80, 40, 1<<1|1<<2
	return e
}

func castRun(w *sim.World, cmd sim.Command) {
	sim.Step(w, []sim.Command{cmd})
	for i := 0; i < 400; i++ {
		sim.Step(w, nil)
	}
}

func TestAUnitCastFollowsTheEditedManaRangeAndDamage(t *testing.T) {
	cast := func(d mod.SpellData, victimX int32) (mana, lost int32) {
		w := castWorld(t, d, castMage(1, 0, 0), castEnt(2, victimX, 0))
		castRun(w, sim.Cast(1, 2, 1))
		return 40 - entityOf(t, w, 1).Mana, 100 - entityOf(t, w, 2).HP
	}
	if mana, lost := cast(mod.SpellData{}, 5); mana != 3 || lost < 8 || lost > 16 {
		t.Fatalf("installed row: mana %d, lost %d", mana, lost)
	}
	edit := mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Arrow", Mana: spellInt(9), Range: spellInt(2),
		DamageMin: spellInt(30), DamageMax: spellInt(30)}}}
	if mana, lost := cast(edit, 2); mana != 9 || lost != 60 {
		t.Errorf("edited row at range 2: mana %d, lost %d, want 9 and 60", mana, lost)
	}
	if mana, lost := cast(edit, 5); mana != 0 || lost != 0 {
		t.Errorf("edited range 2 still reaches 5: mana %d, lost %d", mana, lost)
	}
	global := mod.SpellData{Global: mod.SpellGlobal{Mana: ratio(2, 1), Damage: ratio(3, 1)}}
	if mana, lost := cast(global, 5); mana != 6 || lost < 24 || lost > 48 {
		t.Errorf("global multipliers: mana %d, lost %d, want 6 and 24..48", mana, lost)
	}
}

func TestAnAreaCastCoversTheEditedRadius(t *testing.T) {
	hit := func(d mod.SpellData, dx int32) bool {
		w := castWorld(t, d, castMage(1, 0, 0), castEnt(2, 10+dx, 10))
		castRun(w, sim.CastAt(1, 2, sim.CellPoint{X: 10, Y: 10}))
		return entityOf(t, w, 2).HP < 100
	}
	if !hit(mod.SpellData{}, 3) || hit(mod.SpellData{}, 5) {
		t.Fatal("the installed radius 3 does not hit at 3 and miss at 5")
	}
	small := mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Ball", Radius: spellInt(1)}}}
	if hit(small, 3) {
		t.Error("radius 1 still hits at 3")
	}
	big := mod.SpellData{Global: mod.SpellGlobal{Radius: ratio(2, 1)}}
	if !hit(big, 5) || hit(big, 7) {
		t.Error("a doubled radius does not hit at 5 and miss at 7")
	}
}

func TestAQueuedPointCastSaturatesAtThePayloadByte(t *testing.T) {
	table := castTable()
	table.Spells.(defCollection)[1].params[5], table.Spells.(defCollection)[1].params[7] = 2, 200
	table.Mods.Spells = mod.SpellData{Rows: []mod.SpellRow{{Target: "Fire Arrow", DamageMin: spellInt(mod.SpellMaxDamage), DamageMax: spellInt(mod.SpellMaxDamage)}}}
	w, err := sim.NewSpelledWorld(7, sim.Bounds{Width: 24, Height: 24}, sim.ModeCanonical, nil,
		[]sim.Entity{castMage(1, 0, 0), castEnt(2, 3, 0)}, nil, mapload.SpellRules(table))
	if err != nil {
		t.Fatal(err)
	}
	castRun(w, sim.Cast(1, 2, 1))
	lost := 100 - entityOf(t, w, 2).HP
	if lost < 255 || lost > 300 {
		t.Errorf("victim lost %d, want the saturated 255 to 300", lost)
	}
}
