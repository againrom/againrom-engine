package game

import (
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	swingDagger = 0x0102
	swingShield = 0x0102
)

// swingSoundTable gives classes 3 and 4 distinct slots so a test can tell them apart.
func swingSoundTable() map[int32]UnitSound {
	return map[int32]UnitSound{
		1:  {Slots: []int32{0, 0, 0}, AttackDelay: 5},
		2:  {Slots: []int32{0, 0, 0}, AttackDelay: 5},
		3:  {Slots: []int32{100, 0, 0}, AttackDelay: 5},
		4:  {Slots: []int32{101, 0, 0}, AttackDelay: 5},
		10: {Slots: []int32{120, 0, 0}, AttackDelay: 5},
		23: {Slots: []int32{0, 0, 0}, AttackDelay: 5},
	}
}

// swingCase is one attacker: type id, worn slots, party member or roster human.
type swingCase struct {
	typeID  int32
	worn    [sim.EquipSlots]uint16
	party   bool
	mage    bool
	rowType int32
}

func swingSlotsOf(t *testing.T, c swingCase) []int {
	t.Helper()
	ents := []sim.Entity{swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4)}
	ents[0].TypeID = c.typeID
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, sim.Terrain{},
		ents, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, Equipped: c.worn}})
	if err != nil {
		t.Fatal(err)
	}
	art := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
	art.Anim, art.Corpse = swingAnimDesc(), art
	v, err := ui.NewViewer("swing", terrain.Grid{Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: art}}, v)
	member := mapload.PartyMember{ID: "hero", Class: 1, Mage: c.mage}
	mw.mission = &missionNotices{list: data.NewBodyList("unarmed", "swordsman")}
	if c.party {
		mw.mission.ids = []sim.EntityID{1}
		mw.mission.party = []mapload.PartyMember{member}
	} else {
		member.Class = c.rowType
		mw.mission.state = &Mission{Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{1: member}}}
	}
	var plays []swingSoundPlay
	mw.setSwingSound(swingSoundTable(), func(slot int, owner uint32, cell image.Point) {
		plays = append(plays, swingSoundPlay{slot, cell})
	})
	mw.strike(1, 2)
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}
	var slots []int
	for _, p := range plays {
		slots = append(slots, p.slot)
	}
	return slots
}

func TestSwingSoundFollowsTheDerivedHeroClass(t *testing.T) {
	var dagger, shielded, bare [sim.EquipSlots]uint16
	dagger[0] = swingDagger
	shielded[0], shielded[1] = swingDagger, swingShield
	for _, tc := range []struct {
		name string
		c    swingCase
		want []int
	}{
		{"dagger hero is class 3", swingCase{party: true, worn: dagger}, []int{100}},
		{"dagger and shield hero is class 4", swingCase{party: true, worn: shielded}, []int{101}},
		{"bare-handed hero is silent", swingCase{party: true, worn: bare}, nil},
		{"bare-handed mage is silent", swingCase{party: true, mage: true, worn: bare}, nil},
		{"Hero-flag placement re-derives from its dagger", swingCase{typeID: sim.HeroTypeID(false, false), rowType: 1, worn: dagger}, []int{100}},
	} {
		if got := swingSlotsOf(t, tc.c); len(got) != len(tc.want) || (len(got) == 1 && got[0] != tc.want[0]) {
			t.Errorf("%s: swing slots %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A map-authored human keeps its Humans-row class (ANIM-106).
func TestSwingSoundOfAMapAuthoredHumanKeepsItsRowClass(t *testing.T) {
	var dagger, bare [sim.EquipSlots]uint16
	dagger[0] = swingDagger
	for _, tc := range []struct {
		name string
		c    swingCase
		want []int
	}{
		{"club row with a dagger", swingCase{typeID: 10, rowType: 10, worn: dagger}, []int{120}},
		{"club row bare-handed", swingCase{typeID: 10, rowType: 10, worn: bare}, []int{120}},
		{"unarmed mage row with a dagger stays silent", swingCase{typeID: 23, rowType: 23, mage: true, worn: dagger}, nil},
	} {
		if got := swingSlotsOf(t, tc.c); len(got) != len(tc.want) || (len(got) == 1 && got[0] != tc.want[0]) {
			t.Errorf("%s: swing slots %v, want %v", tc.name, got, tc.want)
		}
	}
}
