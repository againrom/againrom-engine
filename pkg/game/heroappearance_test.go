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
	if look := heroAppearance(w, 7, member, list, heroInPlay); !look.Held || look.Body != data.BodyMageStaff || look.Class != 24 {
		t.Fatalf("held actor: %+v, want the World's staff, class 24", look)
	}
	if look := heroAppearance(nil, 7, member, list, heroInPlay); look.Held || look.Body != data.BodySwordsman || look.Class != 3 {
		t.Fatalf("record: %+v, want the record's sword, class 3", look)
	}
	if look := heroAppearance(w, 8, member, list, heroInPlay); look.Held {
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
		if look := heroAppearance(world, 7, member, list, heroInPlay); look.Body != data.BodySwordsman || look.Class != 3 {
			t.Fatalf("unmaterialized sword (World %v): %+v, want swordsman class 3", world != nil, look)
		}
	}
	member.WeaponMaterialized = true
	if look := heroAppearance(w, 7, member, list, heroInPlay); look.Body != data.BodyUnarmed || look.Class != 1 {
		t.Fatalf("materialized and taken off: %+v, want unarmed class 1", look)
	}
}

// A living map-placed person of the Hero type is redressed at mission open:
// the World stores the class his worn staff selects (ANIM-106, ANIM-107) and
// the flight class rule reads it. A dead mage's staff gives 24 too (HERO-191);
// a non-hero type keeps his.
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
	}{{50, 24, 24}, {51, 24, 24}, {52, 10, 10}} {
		e, _ := w.Entity(want.id)
		if e.Class != want.stored || flightClassRule(e) != want.flight {
			t.Errorf("actor %d: stored class %d, flight class %d; want %d and %d", want.id, e.Class, flightClassRule(e), want.stored, want.flight)
		}
	}
	if got := ms.Start.Roster[50]; got.Class != 24 || got.Body != string(data.BodyMageStaff) {
		t.Errorf("placed hero record %+v, want class 24 and the staff body", got)
	}
}

// A name no arm matches (a staff with a shield) keeps the class held before:
// the stored one in play, 1 or 23 at creation and for a record. A mage with an
// empty weapon slot and a shield is 2 (HERO-190).
func TestAnUnmatchedNameKeepsTheClassHeldBefore(t *testing.T) {
	list := equipClassList(t)
	var worn [sim.EquipSlots]uint16
	worn[0], worn[1] = appearanceCode(equipRowStaff), appearanceCode(1)
	w := heroAppearanceWorld(t, []sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: 14}}, []sim.Stock{{ID: 7, Equipped: worn}})
	fighter, mage := mapload.PartyMember{WeaponMaterialized: true}, mapload.PartyMember{Mage: true, WeaponMaterialized: true}
	for _, tc := range []struct {
		name   string
		world  *sim.World
		member mapload.PartyMember
		at     heroMoment
		want   int32
	}{
		{"in play", w, fighter, heroInPlay, 14},
		{"fighter at creation", w, fighter, heroCreated, 1},
		{"mage at creation", w, mage, heroCreated, 23},
	} {
		if look := heroAppearance(tc.world, 7, tc.member, list, tc.at); look.Class != tc.want {
			t.Errorf("%s: class %d, want %d", tc.name, look.Class, tc.want)
		}
	}
	mage.Worn[0], mage.Worn[1] = worn[0], worn[1]
	if look := heroAppearance(nil, 7, mage, list, heroInPlay); look.Class != 23 {
		t.Errorf("record: class %d, want 23", look.Class)
	}
	mage.Worn[0] = 0
	if look := heroAppearance(nil, 7, mage, list, heroInPlay); look.Class != 2 {
		t.Errorf("mage with a shield and no weapon: class %d, want 2", look.Class)
	}
}

// deadHeroMission opens a mission whose one party member, entity 99, is e
// wearing worn.
func deadHeroMission(t *testing.T, e sim.Entity, member mapload.PartyMember, worn [sim.EquipSlots]uint16) (*mapWorld, *Mission) {
	t.Helper()
	e.ID, e.X, e.Y, e.MaxHP, e.TypeID, e.Humanoid = 99, 3, 3, 10, 33, true
	w := heroAppearanceWorld(t, []sim.Entity{e}, []sim.Stock{{ID: 99, Equipped: worn}})
	set, _ := appearanceUnits()
	m := worldFixtureMap()
	member.ID, member.PlayerCharacter, member.WeaponMaterialized, member.Class = "hero", true, true, e.Class
	ms := &Mission{Number: 1, Map: m, World: w, Party: []mapload.PartyMember{member},
		Start: mapload.Start{IDs: []sim.EntityID{99}}}
	return openMission(ms, nil, set, worldFixtureViewer(t, m), missionSource{HeroPictureAddress: equipClassBodyList()}, nil, nil), ms
}

// At mission open, the open after a LOAD, a hero dead at stage 1 holds 1, or
// 24 for a mage, whatever he wears; one at stage 2 to 4 takes the class his
// slots derive (HERO-191). The record's body follows the slots.
func TestADeadHeroClassAtMissionOpenFollowsHisStage(t *testing.T) {
	var sword [sim.EquipSlots]uint16
	sword[0] = appearanceCode(equipRowSword)
	fallen := sim.Entity{HP: -5, Decay: sim.DecayFallen}
	bones := sim.Entity{HP: -15, Decay: sim.DecayBones}
	for _, tc := range []struct {
		name  string
		body  sim.Entity
		class int32
		mage  bool
		worn  [sim.EquipSlots]uint16
		want  int32
	}{
		{"fighter at stage 1", fallen, 3, false, sword, 1},
		{"fighter at stage 2", bones, 1, false, sword, 3},
		{"mage at stage 1", fallen, 23, true, [sim.EquipSlots]uint16{}, 24},
		{"mage at stage 2", bones, 24, true, [sim.EquipSlots]uint16{}, 23},
	} {
		tc.body.Class = tc.class
		mw, ms := deadHeroMission(t, tc.body, mapload.PartyMember{Mage: tc.mage}, tc.worn)
		e, _ := mw.world.Entity(99)
		if e.Class != tc.want || ms.Party[0].Class != tc.want || flightClassRule(e) != tc.want {
			t.Errorf("%s: stored %d, record %d, flight %d; want %d", tc.name, e.Class, ms.Party[0].Class, flightClassRule(e), tc.want)
		}
	}
	_, ms := deadHeroMission(t, sim.Entity{HP: -5, Decay: sim.DecayFallen, Class: 3}, mapload.PartyMember{}, sword)
	if got := ms.Party[0].Body; got != string(data.BodySwordsman) {
		t.Errorf("dead swordsman's record body %q, want swordsman", got)
	}
}

// A hero who falls in play holds 1, or 24 for a mage, whatever he wears, and
// keeps it as his body decays; one who rises takes his slots' class again
// (HERO-191).
func TestADeadHeroHoldsTheDyingClassInPlay(t *testing.T) {
	var sword, staff [sim.EquipSlots]uint16
	sword[0], staff[0] = appearanceCode(equipRowSword), appearanceCode(equipRowStaff)
	for _, tc := range []struct {
		name       string
		mage       bool
		worn       [sim.EquipSlots]uint16
		alive, low int32
	}{
		{"swordsman", false, sword, 3, 1},
		{"mage with a staff", true, staff, 24, 24},
		{"bare mage", true, [sim.EquipSlots]uint16{}, 23, 24},
	} {
		mw, ms := deadHeroMission(t, sim.Entity{HP: 10, Class: tc.alive}, mapload.PartyMember{Mage: tc.mage}, tc.worn)
		mw.pending = append(mw.pending, sim.Damage(99, 15))
		mw.tickStep(nil, false)
		e, _ := mw.world.Entity(99)
		if e.Decay != sim.DecayFallen || e.Class != tc.low || ms.Party[0].Class != tc.low {
			t.Errorf("%s fell: decay %d, stored %d, record %d; want stage 1 and %d", tc.name, e.Decay, e.Class, ms.Party[0].Class, tc.low)
		}
		mw.pending = append(mw.pending, sim.Damage(99, 10))
		mw.tickStep(nil, false)
		if e, _ = mw.world.Entity(99); e.Decay < sim.DecayBones || e.Class != tc.low {
			t.Errorf("%s decayed: decay %d, stored %d; want stage 2 or deeper and %d", tc.name, e.Decay, e.Class, tc.low)
		}
	}
	mw, ms := deadHeroMission(t, sim.Entity{HP: 10, Class: 3}, mapload.PartyMember{}, sword)
	mw.world.SetClientClass(99, 1)
	mw.redressLifeChanges(map[sim.EntityID]bool{99: false})
	if e, _ := mw.world.Entity(99); e.Class != 3 || ms.Party[0].Class != 3 {
		t.Errorf("risen swordsman: stored %d, record %d; want 3", e.Class, ms.Party[0].Class)
	}
}
