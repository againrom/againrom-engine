package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestKilledPlacedHeroKeepsPCBodyBindingThroughDecay(t *testing.T) {
	const id sim.EntityID = 55
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(appearRowSword)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: 3, TypeID: sim.HeroTypeID(false, false)}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: id, Equipped: worn}})
	if err != nil {
		t.Fatal(err)
	}
	set, bodies := appearanceUnits()
	set.Classes[1] = &terrain.UnitClass{Name: "unarmed"}
	set.Classes[3] = &terrain.UnitClass{Name: "swordsman"}
	ms := &Mission{Number: 40, Map: worldFixtureMap(), World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{id: {
			ID: "npc:25", Body: string(data.BodySwordsman), BodyDir: data.HeroDirHeroes,
			Class: 3, WeaponMaterialized: true,
		}}}}
	src := missionSource{HeroPictureAddress: appearanceBodyListBytes()}
	mw := openMission(ms, nil, set, worldFixtureViewer(t, ms.Map), src, nil, nil)
	want := bodies[data.HeroBodyKey(data.HeroDirHeroesLight, data.BodySwordsman)]
	if mw.art[id] != want {
		t.Fatal("living placed Hero does not use its PC resource")
	}
	if err := w.HeadlessDamage(id, 20); err != nil {
		t.Fatal(err)
	}
	check := func(stage string) {
		t.Helper()
		if got := missionAppearanceArt(ms, set, data.NewBodyList(data.BodyUnarmed, data.BodySwordsman), src)[id]; got != want {
			t.Fatalf("%s restored binding=%p, want original PC resource %p", stage, got, want)
		}
	}
	check("fallen")
	for i := 0; i < 40; i++ {
		mw.tick()
	}
	e, exists := w.Entity(id)
	slots, equipped := w.Equipped(id)
	if !exists || e.Decay < sim.DecayBones || !equipped || slots[0] != 0 || len(w.Sacks()) == 0 {
		t.Fatalf("looted corpse control not reached: entity=%+v slots=%v equipped=%v", e, slots, equipped)
	}
	check("looted corpse")
}
