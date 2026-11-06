package game

import (
	"testing"

	"againrom/pkg/sim"
)

// The campaign bank's clearing points (R2-ENGINE-045, R2-ENGINE-046): a new
// campaign starts from NewGame's bank, and a town entry from the world map
// clears slots 752..767 and keeps every other slot. The mission entry's
// clearing and LOAD's keeping are the SAV test's.
func TestSecondCampaignBankClearingPoints(t *testing.T) {
	if c := newSecondCampaign(); c.bank != sim.NewGameScenarioBank() {
		t.Fatal("a new campaign's bank is not NewGame's")
	}
	f, _, screen := secondCampaignFixture(t, true)
	c := newSecondCampaign()
	c.current, c.available = partyPlace{2, 3}, []partyPlace{{2, 3}, {2, 2}, {1, 30}}
	f.Town.second = c
	for i := sim.ObjectiveSlotBase; i < sim.ObjectiveSlotBase+sim.ObjectiveSlots; i++ {
		c.bank[i] = int32(i - sim.ObjectiveSlotBase + 1)
	}
	c.bank[900] = 9
	want := c.bank
	sim.EnterLocation(&want)
	if !c.leaveTown() || !screen.AtWorldMap() {
		t.Fatal("leaving town 3 did not open the map")
	}
	s := screen.worldMapNow()
	s.missions[2].Anchor = s.missions[1].Anchor
	s.startTravel(2)
	s.skipTravel()
	if act := screen.WorldMapTick(); act.Open != nil || c.current != (partyPlace{2, 2}) {
		t.Fatalf("arrival entered %v, want town 2", c.current)
	}
	if c.bank != want {
		for i := range c.bank {
			if c.bank[i] != want[i] {
				t.Fatalf("after the town entry slot %d = %d, want %d", i, c.bank[i], want[i])
			}
		}
	}
	if want[753] != 0 || want[900] != 9 || want[768] != 10 {
		t.Fatal("the reference clearing is wrong")
	}
}
