package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestSecondDeadHireIsNotOfferedAgain: three hires stand in the second town
// and march into mission 10. Hire 1 falls below -10 health, hire 3's body
// has left the world and hire 2 survives. After the departure, after one
// further mission without them, and after a town SAVE and LOAD, the inn
// offers neither fallen hire; the survivor rests one mission and is offered
// again (R2-SESSION-195, DIV-4645).
func TestSecondDeadHireIsNotOfferedAgain(t *testing.T) {
	s, err := sim.NewROM2Script(missionChecks(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fallen := sim.Entity{ID: 41, X: 10, Y: 10, HP: -20, MaxHP: 40, Owner: sim.SelfSlot, TypeID: 1, TokenSize: 1}
	survivor := sim.Entity{ID: 42, X: 14, Y: 10, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: 1, TokenSize: 1}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), []sim.Entity{fallen, survivor}, s)
	if err != nil {
		t.Fatal(err)
	}
	var bank [1024]int32
	bank[768] = 10
	for i := 0; i < 3; i++ {
		bank[512+i], bank[532+i], bank[552+i] = 1, 1, 2
	}
	if !w.SetROM2ScenarioState(bank) {
		t.Fatal("missing bank")
	}
	party := []mapload.PartyMember{
		{ID: "hire:1", Temporary: true, CampaignHire: 1},
		{ID: "hire:2", Temporary: true, CampaignHire: 2},
		{ID: "hire:3", Temporary: true, CampaignHire: 3},
	}
	ids := []sim.EntityID{41, 42, 43}

	town, mission := partyPlace{2, 2}, partyPlace{1, 10}
	c := newSecondCampaign()
	c.current, c.available = mission, []partyPlace{town, mission}
	offers := func(step string, want []innOffer) {
		t.Helper()
		c.current = town
		if got := c.hires(); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: hire offers %v want %v (availability %v)", step, got, want, c.bank[532:535])
		}
	}
	c.complete(w, party, ids)
	offers("after the mission", []innOffer{{kind: 2, npc: 2}})

	c.current, c.available = mission, []partyPlace{town, mission}
	c.completeBank(c.bank)
	offers("after the rest", []innOffer{{kind: 1, npc: 2}})

	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = secondDollTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hero.Hero.Skill[2] = 20
	f.Carried = []mapload.PartyMember{hero}
	saved := secondShopSample()
	saved.current, saved.available, saved.room, saved.bank = town, []partyPlace{town}, secondTownShop, c.bank
	f.Town.second, f.Town.gold = saved, 500
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	c = coldSecondTown(t, raw).Town.second
	offers("after a town SAVE and LOAD", []innOffer{{kind: 1, npc: 2}})
}
