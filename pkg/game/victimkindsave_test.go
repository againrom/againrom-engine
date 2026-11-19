package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// victimKindMission is mission 10 with three creatures of Player 2 placed by
// the server ids 9201, 9202 and 9201, each of 1 hit point and value 100.
func victimKindMission() secondMissionContent {
	row := func(server int32) []int32 {
		p := make([]int32, 62)
		for i := range p {
			p[i] = -1
		}
		p[4], p[29], p[30], p[37], p[55] = 1, 71, 1, 100, server
		return p
	}
	c := secondMissionContent{side: 128, players: 2, creatures: map[int]dbEntry{}}
	for i, server := range []int32{9201, 9202, 9201} {
		c.creatures[2+i] = dbEntry{name: "Kind" + string(rune('A'+i)), params: row(server)}
		c.units = append(c.units, alm.Unit{X: uint32(70+4*i)<<8 | 0x80, Y: 70<<8 | 0x80, ClassID: 9, ClassSubID: 1,
			ServerID: uint32(server), Owner: 2, UnitID: uint16(11 + i), GroupID: 6})
	}
	return c
}

// victimKindKill places the creature of map unit id beside the hero and
// has the hero strike it until it falls.
func victimKindKill(t *testing.T, f *FrontEnd, hero sim.EntityID, id uint16) {
	t.Helper()
	w := f.live.world
	var victim sim.Entity
	found := false
	for _, e := range w.Entities() {
		if e.MapUnitID == id {
			victim, found = e, true
		}
	}
	if !found {
		t.Fatalf("no map unit %d", id)
	}
	victimKindKillActor(t, f, hero, victim.ID)
}

// victimKindKillActor places the victim beside the hero and has the hero
// strike it until it falls.
func victimKindKillActor(t *testing.T, f *FrontEnd, hero, victim sim.EntityID) {
	t.Helper()
	w := f.live.world
	h, _ := w.Entity(hero)
	if err := w.HeadlessPlace(victim, h.X+1, h.Y); err != nil {
		t.Fatalf("actor %d cannot stand beside the hero: %v", victim, err)
	}
	f.live.strike(uint32(hero), uint32(victim))
	for range 3000 {
		f.live.tick()
		if v, _ := w.Entity(victim); !v.Alive() {
			return
		}
	}
	t.Fatalf("the hero did not kill actor %d", victim)
}

// A new game's mission counts its placed creatures under their kind words,
// the low word of each Units row's server id (R2-ENGINE-1157,
// R2-ENGINE-525); the hero's list and every
// actor's kind word cross a mission SAVE and a cold LOAD with the complete
// World.
func TestSecondGameVictimListCrossesSaveAndColdLoad(t *testing.T) {
	f, app, screen, dir := secondBagFrontOf(t, nil, victimKindMission())
	secondBagEnter(t, f, app, screen)
	hero := f.live.mission.ids[0]
	victimKindKill(t, f, hero, 11)
	victimKindKill(t, f, hero, 12)
	victimKindKill(t, f, hero, 13)
	for range 60 {
		f.live.tick()
	}
	h, _ := f.live.world.Entity(hero)
	list := h.Values().VictimKinds
	if want := []sim.VictimKindCount{{Kind: 9201, Count: 1}, {Kind: 9202}}; !slices.Equal(list, want) {
		t.Fatalf("the hero's list is %+v, want %+v", list, want)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := secondSaveFiles(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("mission SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	cold, _ := secondMissionLoadOf(t, victimKindMission(), dir, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	loaded, _ := cold.live.world.Entity(hero)
	if got := loaded.Values().VictimKinds; !slices.Equal(got, list) {
		t.Fatalf("the loaded hero's list is %+v, want %+v", got, list)
	}
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
}
