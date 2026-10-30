package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A placed person the client draws with the hero class takes the hero body of
// his worn equipment while his World type id stays his row's staffless mage;
// without that hero type he draws his row's class sheet (R2-ENGINE-680).
func TestAClientHeroDrawsTheHeroBodyOverHisRowsClass(t *testing.T) {
	const id sim.EntityID = 36
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: 23, TypeID: 23}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: id}})
	if err != nil {
		t.Fatal(err)
	}
	set, bodies := appearanceUnits()
	ms := &Mission{Number: 40, Map: worldFixtureMap(), World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{id: {
			ID: "Mage_Hero", Mage: true, Class: 23, WeaponMaterialized: true,
		}}}}
	src := missionSource{HeroPictureAddress: appearanceBodyListBytes()}
	list := data.NewBodyList(data.BodyUnarmed, data.BodySwordsman)
	heroes := map[sim.EntityID]int32{id: sim.HeroTypeID(true, true)}
	want := bodies[data.HeroBodyKey(data.HeroDirHeroes, data.BodyMage)]
	if got := missionAppearanceArt(ms, set, list, src, heroes)[id]; got != want {
		t.Fatalf("client hero draws %v, want the hero mage body %v", got, want)
	}
	if e, _ := w.Entity(id); e.TypeID != 23 {
		t.Fatalf("World type id %d, want the row's 23", e.TypeID)
	}
	if got := missionAppearanceArt(ms, set, list, src, nil)[id]; got != set.Classes[23] {
		var name string
		if got != nil {
			name = got.Name
		}
		t.Fatalf("without the client hero type the person draws %q, want his class sheet", name)
	}
	if got := drawnTypeID(sim.Entity{ID: 7, TypeID: 1}, heroes); got != 1 {
		t.Fatalf("another person draws type %d, want his own 1", got)
	}
}
