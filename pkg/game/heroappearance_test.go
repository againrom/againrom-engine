package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func equipClassList(t *testing.T) data.BodyList {
	t.Helper()
	list, ok := ReadBodyList(missionSource{HeroPictureAddress: equipClassBodyList()})
	if !ok || list.Len() == 0 {
		t.Fatalf("ReadBodyList: %d names, %v", list.Len(), ok)
	}
	return list
}

func heroAppearanceWorld(t *testing.T, ents []sim.Entity, stock []sim.Stock) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, ents, nil, sim.Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// The World's slots answer for an actor it holds; the record answers for one
// it does not.
func TestHeroAppearanceReadsTheWorldOverTheRecord(t *testing.T) {
	list := equipClassList(t)
	var staff [sim.EquipSlots]uint16
	staff[0] = appearanceCode(equipRowStaff)
	w := heroAppearanceWorld(t, []sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, []sim.Stock{{ID: 7, Equipped: staff}})
	member := mapload.PartyMember{Mage: true, WeaponMaterialized: true}
	member.Worn[0] = appearanceCode(equipRowSword)
	if look := heroAppearance(w, 7, member, list); !look.Held || look.Body != data.BodyMageStaff || look.Class != 24 {
		t.Fatalf("held actor: %+v, want the World's staff, class 24", look)
	}
	if look := heroAppearance(nil, 7, member, list); look.Held || look.Body != data.BodySwordsman || look.Class != 3 {
		t.Fatalf("record: %+v, want the record's sword, class 3", look)
	}
	if look := heroAppearance(w, 8, member, list); look.Held {
		t.Fatalf("an actor the World does not hold read its slots: %+v", look)
	}
}

// An empty slot 1 is widened by a starting weapon that has not materialized
// (DIV-111), on both inputs; a materialized one leaves the hero bare-handed.
func TestHeroAppearanceWidensAnUnmaterializedStartingWeapon(t *testing.T) {
	list := equipClassList(t)
	w := heroAppearanceWorld(t, []sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, []sim.Stock{{ID: 7}})
	sword := &data.Weapon{Code: data.ItemCode(appearanceCode(equipRowSword))}
	member := mapload.PartyMember{Weapon: sword}
	for _, world := range []*sim.World{w, nil} {
		if look := heroAppearance(world, 7, member, list); look.Body != data.BodySwordsman || look.Class != 3 {
			t.Fatalf("unmaterialized sword (World %v): %+v, want swordsman class 3", world != nil, look)
		}
	}
	member.WeaponMaterialized = true
	if look := heroAppearance(w, 7, member, list); look.Body != data.BodyUnarmed || look.Class != 1 {
		t.Fatalf("materialized and taken off: %+v, want unarmed class 1", look)
	}
}

// A living map-placed person of the Hero type is redressed at mission open:
// the World stores the class his worn staff selects (ANIM-106, ANIM-107) and
// the flight class rule reads it. A dead one and a non-hero type keep theirs.
func TestAPlacedHeroTypeHumanStoresTheClassHisEquipmentSelects(t *testing.T) {
	var staff [sim.EquipSlots]uint16
	staff[0] = appearanceCode(equipRowStaff)
	ents := []sim.Entity{
		{ID: 99, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: 1},
		{ID: 50, X: 5, Y: 3, HP: 10, MaxHP: 10, Class: 3, TypeID: 35, Humanoid: true},
		{ID: 51, X: 6, Y: 3, HP: -10, MaxHP: 10, Class: 3, TypeID: 35, Humanoid: true},
		{ID: 52, X: 7, Y: 3, HP: 10, MaxHP: 10, Class: 10, TypeID: 10, Humanoid: true},
	}
	stock := []sim.Stock{{ID: 99}, {ID: 50, Equipped: staff}, {ID: 51, Equipped: staff}, {ID: 52, Equipped: staff}}
	w := heroAppearanceWorld(t, ents, stock)
	set, _ := appearanceUnits()
	m := worldFixtureMap()
	roster := map[sim.EntityID]mapload.PartyMember{
		50: {Mage: true, Class: 3}, 51: {Mage: true, Class: 3}, 52: {Class: 10}}
	ms := &Mission{Number: 1, Map: m, World: w, Party: []mapload.PartyMember{{ID: "hero", PlayerCharacter: true}},
		Start: mapload.Start{IDs: []sim.EntityID{99}, Roster: roster}}
	openMission(ms, nil, set, worldFixtureViewer(t, m), missionSource{HeroPictureAddress: equipClassBodyList()}, nil, nil)
	for _, want := range []struct {
		id             sim.EntityID
		stored, flight int32
	}{{50, 24, 24}, {51, 3, 3}, {52, 10, 10}} {
		e, _ := w.Entity(want.id)
		if e.Class != want.stored || flightClassRule(e) != want.flight {
			t.Errorf("actor %d: stored class %d, flight class %d; want %d and %d", want.id, e.Class, flightClassRule(e), want.stored, want.flight)
		}
	}
	if got := ms.Start.Roster[50]; got.Class != 24 || got.Body != string(data.BodyMageStaff) {
		t.Errorf("placed hero record %+v, want class 24 and the staff body", got)
	}
}

// A dead party member keeps the class his actor holds at mission open, as a
// SAV restored it (DIV-4141); his record follows his empty slots.
func TestADeadPartyMemberKeepsTheClassTheWorldHolds(t *testing.T) {
	ents := []sim.Entity{{ID: 99, X: 3, Y: 3, HP: -10, MaxHP: 10, Class: 14, TypeID: 34, Humanoid: true}}
	w := heroAppearanceWorld(t, ents, []sim.Stock{{ID: 99}})
	set, _ := appearanceUnits()
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w, Party: []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, Class: 14}},
		Start: mapload.Start{IDs: []sim.EntityID{99}}}
	openMission(ms, nil, set, worldFixtureViewer(t, m), missionSource{HeroPictureAddress: equipClassBodyList()}, nil, nil)
	if e, _ := w.Entity(99); e.Class != 14 {
		t.Errorf("dead member stored class %d, want the restored 14", e.Class)
	}
	if got := ms.Party[0]; got.Class != 1 || got.Body != string(data.BodyUnarmed) {
		t.Errorf("dead member record class %d body %q, want 1 and unarmed", got.Class, got.Body)
	}
}
