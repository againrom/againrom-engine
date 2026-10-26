package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The rows of the synthetic body list below: a weapon code's row is the line
// its body name stands on (HERO-APPEAR-042).
const (
	equipRowSword   = 2
	equipRowSword2H = 3
	equipRowStaff   = 9
)

func equipClassBodyList() []byte {
	names := map[int]data.HeroBody{1: data.BodyUnarmed, equipRowSword: data.BodySwordsman,
		equipRowSword2H: data.BodySwordsman2H, equipRowStaff: data.BodyMageStaff}
	var out []byte
	for row := 1; row <= equipRowStaff; row++ {
		out = append(out, names[row]...)
		out = append(out, '\n')
	}
	return out
}

// equipClassMission opens a mission whose one party member, entity 99, wears
// worn and carries carried, with record and actor both stating class.
func equipClassMission(t *testing.T, member mapload.PartyMember, class int32, worn [sim.EquipSlots]uint16, carried []uint16) (*mapWorld, *Mission) {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 99, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: class}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 99, Items: carried, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	set, _ := appearanceUnits()
	m := worldFixtureMap()
	member.Class = class
	ms := &Mission{Number: 1, Map: m, World: w, Party: []mapload.PartyMember{member},
		Start: mapload.Start{IDs: []sim.EntityID{99}}}
	src := missionSource{HeroPictureAddress: equipClassBodyList()}
	return openMission(ms, nil, set, worldFixtureViewer(t, m), src, nil, nil), ms
}

func requireHeroClass(t *testing.T, mw *mapWorld, ms *Mission, want int32, body data.HeroBody, when string) {
	t.Helper()
	e, ok := mw.world.Entity(99)
	if !ok {
		t.Fatalf("%s: the hero is gone", when)
	}
	if e.Class != want || ms.Party[0].Class != want || ms.Party[0].Body != string(body) {
		t.Errorf("%s: actor class %d, record class %d body %q; want %d and %q",
			when, e.Class, ms.Party[0].Class, ms.Party[0].Body, want, body)
	}
}

// A fighter who swaps his two-handed sword for a one-handed one with no shield
// is a Human Swordsman (3), no longer a Two-Handed Swordsman (5), in the World
// and in the record a SAVE writes.
func TestATwoHandedSwordSwappedForAOneHandedOneChangesTheClass(t *testing.T) {
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(equipRowSword2H)
	member := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true,
		Body: string(data.BodySwordsman2H), BodyDir: data.HeroDirHeroes, WeaponMaterialized: true}
	mw, ms := equipClassMission(t, member, 5, worn, []uint16{appearanceCode(equipRowSword)})
	requireHeroClass(t, mw, ms, 5, data.BodySwordsman2H, "at open")
	mw.pending = append(mw.pending, sim.Equip(99, 0, 1))
	mw.tickStep(nil, false)
	requireHeroClass(t, mw, ms, 3, data.BodySwordsman, "after the swap")
}

// A mage whose staff comes off is an Unarmed Mage (23), no longer a Human Mage
// (24).
func TestAMagesWeaponRemovedChangesTheClass(t *testing.T) {
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(equipRowStaff)
	member := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Body: string(data.BodyMageStaff), BodyDir: data.HeroDirHeroes, WeaponMaterialized: true}
	mw, ms := equipClassMission(t, member, 24, worn, nil)
	requireHeroClass(t, mw, ms, 24, data.BodyMageStaff, "at open")
	mw.pending = append(mw.pending, sim.Unequip(99, 1))
	mw.tickStep(nil, false)
	requireHeroClass(t, mw, ms, 23, data.BodyMage, "after the staff came off")
}

// A save that already holds a stale class opens with the class its equipment
// selects: the actor's and the record's 5 and 24 become 3 and 23.
func TestAStaleSavedClassOpensAsTheEquipmentSelects(t *testing.T) {
	var sword [sim.EquipSlots]uint16
	sword[0] = appearanceCode(equipRowSword)
	fighter := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true,
		Body: string(data.BodySwordsman2H), BodyDir: data.HeroDirHeroes, WeaponMaterialized: true}
	mw, ms := equipClassMission(t, fighter, 5, sword, nil)
	requireHeroClass(t, mw, ms, 3, data.BodySwordsman, "fighter at open")

	mage := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Body: string(data.BodyMageStaff), BodyDir: data.HeroDirHeroes, WeaponMaterialized: true}
	mw, ms = equipClassMission(t, mage, 24, [sim.EquipSlots]uint16{}, nil)
	requireHeroClass(t, mw, ms, 23, data.BodyMage, "mage at open")
}
