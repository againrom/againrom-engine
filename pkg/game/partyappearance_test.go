package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// THE PARTY'S DRAWN BODY IS DERIVED AT MISSION OPEN (owner defect reports,
// two in one message: a mage with no staff equipped drawn holding one, and a
// fighter dressed completely in the shop walking onto the map bare-handed
// and unarmoured).
//
// PartyMember.Body, .BodyDir and .Class were derived once at party assembly and
// never again, so every later equipment change left them naming an old loadout;
// partyArt keys the art bundle off exactly those two fields. These tests drive
// openMission over a world whose entity wears something the party RECORD does
// not, and assert both the record and the art bundle answer for the world.
//
// The list is synthetic and small: row 1 is the empty-slot arm HeroBodyFor
// falls to, row 2 a one-handed sword body, row 9 the staff body. A code's row
// is field D, its low five bits (data.ComposeItemCode).

const (
	appearRowUnarmed = 1
	appearRowSword   = 2
	appearRowStaff   = 9
)

func appearanceCode(row int) uint16 { return uint16(data.ComposeItemCode(0, 0, 0, row)) }

// appearanceUnits pre-seeds the bundle with every body these tests can derive,
// each under data.HeroBodyKey's own composed key — the same key partyArt reads
// and LoadHeroBody writes. A seeded key makes LoadHeroBody return before it
// touches an archive, which is what lets these run install-free.
func appearanceUnits() (*terrain.UnitSet, map[string]*terrain.UnitClass) {
	byKey := map[string]*terrain.UnitClass{}
	for _, pair := range [][2]string{
		{data.HeroDirHeroes, string(data.BodyMage)},
		{data.HeroDirHeroes, string(data.BodyMageStaff)},
		{data.HeroDirHeroes, string(data.BodyUnarmed)},
		{data.HeroDirHeroes, string(data.BodySwordsman)},
		{data.HeroDirHeroesLight, string(data.BodyUnarmed)},
		{data.HeroDirHeroesLight, string(data.BodySwordsman)},
	} {
		key := data.HeroBodyKey(pair[0], data.HeroBody(pair[1]))
		byKey[key] = &terrain.UnitClass{Name: key}
	}
	set := &terrain.UnitSet{
		Bodies: map[string]*terrain.UnitClass{},
		Classes: map[int32]*terrain.UnitClass{
			23: {Name: "installed-staffless-mage"},
			24: {Name: "installed-staff-mage"},
		},
	}
	for k, v := range byKey {
		set.Bodies[k] = v
	}
	return set, byKey
}

// appearanceBodyListBytes is the shipped list's own format: one body name per
// line, the row being the line number. Rows 3 to 8 are empty, which HeroBodyFor
// refuses — no code in these tests names one.
func appearanceBodyListBytes() []byte {
	lines := make([]byte, 0, 64)
	for row := 1; row <= appearRowStaff; row++ {
		switch row {
		case appearRowUnarmed:
			lines = append(lines, data.BodyUnarmed...)
		case appearRowSword:
			lines = append(lines, data.BodySwordsman...)
		case appearRowStaff:
			lines = append(lines, data.BodyMageStaff...)
		}
		lines = append(lines, '\n')
	}
	return lines
}

// appearanceMission opens a mission whose single party member is entity 99,
// wearing worn in the world and carrying the RECORD stated by member. The
// source serves the synthetic body list at the address ReadBodyList reads.
func appearanceMission(t *testing.T, member mapload.PartyMember, worn [sim.EquipSlots]uint16) (
	*mapWorld, *Mission, map[string]*terrain.UnitClass) {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 99, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 99, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	set, byKey := appearanceUnits()
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{member},
		Start: mapload.Start{IDs: []sim.EntityID{99}}}
	src := missionSource{HeroPictureAddress: appearanceBodyListBytes()}
	mw := openMission(ms, nil, set, worldFixtureViewer(t, m), src, nil, nil)
	return mw, ms, byKey
}

// A mage whose record says he holds a staff and whose entity holds nothing is
// drawn staffless. This is the first report verbatim.
func TestAMageWithNoStaffEquippedIsDrawnWithoutOne(t *testing.T) {
	member := mapload.PartyMember{
		ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		// The record as party assembly left it, before the staff came off.
		Body: string(data.BodyMageStaff), BodyDir: data.HeroDirHeroes, Class: 24,
		// No fallback: this member's starting weapon has already materialized,
		// so nothing may widen slot 1 on his behalf.
		WeaponMaterialized: true,
	}
	mw, ms, byKey := appearanceMission(t, member, [sim.EquipSlots]uint16{})

	if got := ms.Party[0].Body; got != string(data.BodyMage) {
		t.Errorf("party record Body = %q, want %q — his entity holds nothing in slot 1",
			got, data.BodyMage)
	}
	if got := ms.Party[0].Class; got != 23 {
		t.Errorf("party record Class = %d, want 23, the staffless mage's own drawn class", got)
	}
	want := byKey[data.HeroBodyKey(data.HeroDirHeroes, data.BodyMage)]
	if got := mw.art[99]; got != want {
		t.Errorf("the drawn class at tick zero is %v, want the staffless mage body %v", got, want)
	}
}

// A temporary map-roster actor is not in Party and therefore cannot be fixed
// by canonicalizePartyAppearance. It keeps its Humans-row class whatever it
// wears (ANIM-106).
func TestMissionRosterHumanKeepsItsRowClassArtAtTickZero(t *testing.T) {
	const id sim.EntityID = 55
	for _, tc := range []struct {
		name   string
		typeID int32
		worn   uint16
	}{
		{"staff-class row, empty weapon slot", 24, 0},
		{"staff-class row, sword worn", 24, appearanceCode(appearRowSword)},
		{"unarmed row, sword worn", 1, appearanceCode(appearRowSword)},
	} {
		var worn [sim.EquipSlots]uint16
		worn[0] = tc.worn
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
			sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10, TypeID: tc.typeID}}, nil, sim.Relations{}, nil,
			[]sim.Stock{{ID: id, Equipped: worn}})
		if err != nil {
			t.Fatalf("NewStockedWorld: %v", err)
		}
		set, _ := appearanceUnits()
		set.Classes[1] = &terrain.UnitClass{Name: "installed-unarmed"}
		m := worldFixtureMap()
		member := mapload.PartyMember{ID: "npc:row", Mage: true, Class: tc.typeID, WeaponMaterialized: true}
		ms := &Mission{Number: 20, Map: m, World: w,
			Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{id: member}}}
		src := missionSource{HeroPictureAddress: appearanceBodyListBytes()}
		mw := openMission(ms, nil, set, worldFixtureViewer(t, m), src, nil, nil)
		if got, want := mw.art[id], set.Classes[tc.typeID]; got != want {
			t.Errorf("%s: tick-zero roster art = %v, want the row class %d art %v", tc.name, got, tc.typeID, want)
		}
		if got := ms.Start.Roster[id]; got.Class != tc.typeID || got.Body != "" {
			t.Errorf("%s: roster class %d body %q, want the authored class and no body", tc.name, got.Class, got.Body)
		}
	}
}

func TestKilledRosterHumanKeepsItsDrawnBodyAfterEquipmentDrops(t *testing.T) {
	const id sim.EntityID = 55
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(appearRowSword)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: 3, TypeID: 3}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: id, Equipped: worn}})
	if err != nil {
		t.Fatal(err)
	}
	set, _ := appearanceUnits()
	set.Classes[1] = &terrain.UnitClass{Name: "unarmed"}
	set.Classes[3] = &terrain.UnitClass{Name: "swordsman"}
	m := worldFixtureMap()
	ms := &Mission{Number: 20, Map: m, World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{id: {
			ID: "npc", Body: string(data.BodySwordsman), BodyDir: data.HeroDirHeroes,
			Class: 3, WeaponMaterialized: true,
		}}}}
	mw := openMission(ms, nil, set, worldFixtureViewer(t, m),
		missionSource{HeroPictureAddress: appearanceBodyListBytes()}, nil, nil)
	before := mw.art[id]
	if before != set.Classes[3] {
		t.Fatalf("opening body = %v, want swordsman", before)
	}
	if err := w.HeadlessDamage(id, 20); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if slots, ok := w.Equipped(id); ok && slots[0] == 0 {
			break
		}
		mw.tick()
	}
	slots, ok := w.Equipped(id)
	if !ok || slots[0] != 0 || len(w.Sacks()) == 0 {
		t.Fatalf("death did not drop equipment: slots=%v held=%t sacks=%v", slots, ok, w.Sacks())
	}
	if e, held := w.Entity(id); !held || e.Decay < sim.DecayBones {
		t.Fatalf("death did not enter the looted corpse stage: %+v held=%t", e, held)
	}
	if after := missionAppearanceArt(ms, set, data.BodyList{data.BodyUnarmed, data.BodySwordsman}, nil)[id]; after != before {
		t.Fatalf("restored body = %v, want pre-save corpse body %v", after, before)
	}
}

func TestTickRefreshesPartyBodyAfterWornItemDrops(t *testing.T) {
	member := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true,
		Body: string(data.BodyUnarmed), BodyDir: data.HeroDirHeroes, Class: 1,
		WeaponMaterialized: true}
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(appearRowSword)
	mw, _, byKey := appearanceMission(t, member, worn)
	const id sim.EntityID = 99
	if got := mw.art[id]; got != byKey[data.HeroBodyKey(data.HeroDirHeroesLight, data.BodySwordsman)] {
		t.Fatalf("opening body = %v, want swordsman", got)
	}
	mw.pending = append(mw.pending, sim.DropWorn(id, 1, sim.CellPoint{X: 3, Y: 3}))
	mw.tick()
	slots, ok := mw.world.Equipped(id)
	sacks := mw.world.Sacks()
	if !ok || slots[0] != 0 || len(sacks) != 1 || len(sacks[0].Items) != 1 ||
		sacks[0].Items[0] != appearanceCode(appearRowSword) {
		t.Fatalf("drop did not reach the ground: slots=%v held=%t sacks=%v", slots, ok, mw.world.Sacks())
	}
	want := byKey[data.HeroBodyKey(data.HeroDirHeroesLight, data.BodyUnarmed)]
	if got := mw.art[id]; got != want {
		t.Fatalf("body after drop = %v, want unarmed %v", got, want)
	}
}

// The mirror: a mage whose entity DOES hold a staff keeps the staff body.
// Without it the test above would pass against an implementation that never
// draws a staff.
func TestAMageHoldingAStaffKeepsTheStaffBody(t *testing.T) {
	member := mapload.PartyMember{
		ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Body: string(data.BodyMage), BodyDir: data.HeroDirHeroes, Class: 23,
		WeaponMaterialized: true,
	}
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(appearRowStaff)
	mw, ms, byKey := appearanceMission(t, member, worn)

	if got := ms.Party[0].Body; got != string(data.BodyMageStaff) {
		t.Errorf("party record Body = %q, want %q — his entity holds a staff", got, data.BodyMageStaff)
	}
	want := byKey[data.HeroBodyKey(data.HeroDirHeroes, data.BodyMageStaff)]
	if got := mw.art[99]; got != want {
		t.Errorf("the drawn class at tick zero is %v, want the staff body %v", got, want)
	}
}

// A fighter who bought and wore a weapon after his record was written walks
// onto the map wearing it. This is the second report verbatim, at the
// appearance derivation rather than at the shop compositor that also changed.
func TestAFighterDressedAfterAssemblyIsDrawnDressed(t *testing.T) {
	member := mapload.PartyMember{
		ID: "hero", PlayerCharacter: true, StartingHero: true,
		Body: string(data.BodyUnarmed), BodyDir: data.HeroDirHeroesLight, Class: 1,
		WeaponMaterialized: true,
	}
	var worn [sim.EquipSlots]uint16
	worn[0] = appearanceCode(appearRowSword)
	mw, ms, byKey := appearanceMission(t, member, worn)

	if got := ms.Party[0].Body; got != string(data.BodySwordsman) {
		t.Errorf("party record Body = %q, want %q — his entity holds a sword", got, data.BodySwordsman)
	}
	want := byKey[data.HeroBodyKey(data.HeroDirHeroesLight, data.BodySwordsman)]
	if got := mw.art[99]; got != want {
		t.Errorf("the drawn class at tick zero is %v, want the swordsman body %v", got, want)
	}
}

// DIV-070's population is unaffected: a member whose starting weapon never
// reached the equipment array is still DRAWN holding it. Without this the
// derivation would draw him bare-handed on the map while his own doll, composed
// through currentFigureEquipment, shows him armed.
func TestAnUnmaterializedStartingWeaponStillDrawsItsOwnBody(t *testing.T) {
	member := mapload.PartyMember{
		ID: "npc:22", PlayerCharacter: true,
		Body: string(data.BodyUnarmed), BodyDir: data.HeroDirHeroesLight, Class: 1,
		Weapon:             &data.Weapon{Name: "sword", Code: data.ItemCode(appearanceCode(appearRowSword))},
		WeaponMaterialized: false,
	}
	_, ms, _ := appearanceMission(t, member, [sim.EquipSlots]uint16{})

	if got := ms.Party[0].Body; got != string(data.BodySwordsman) {
		t.Errorf("party record Body = %q, want %q — his starting weapon has not materialized, "+
			"so he is drawn holding it", got, data.BodySwordsman)
	}
}

// And the latch retires it: the same member with the latch raised is drawn
// bare-handed. This is the discriminator for the test above — without it that
// test would pass against an implementation that widens slot 1 unconditionally.
func TestAMaterializedStartingWeaponNoLongerDrawsItsOwnBody(t *testing.T) {
	member := mapload.PartyMember{
		ID: "npc:22", PlayerCharacter: true,
		Body: string(data.BodySwordsman), BodyDir: data.HeroDirHeroesLight, Class: 3,
		Weapon:             &data.Weapon{Name: "sword", Code: data.ItemCode(appearanceCode(appearRowSword))},
		WeaponMaterialized: true,
	}
	_, ms, _ := appearanceMission(t, member, [sim.EquipSlots]uint16{})

	if got := ms.Party[0].Body; got != string(data.BodyUnarmed) {
		t.Errorf("party record Body = %q, want %q — the latch is raised, so nothing widens slot 1",
			got, data.BodyUnarmed)
	}
}
