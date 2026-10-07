//go:build sessioncorpusaudit

package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Mission Snapshot.Party is the entry roster used to reconstruct entity IDs;
// Snapshot.Gold is the town purse. The running World owns current holdings.
func currentRoundTripSnapshot(f *FrontEnd, onMap bool) (Snapshot, string, error) {
	s, label, err := f.Snapshot(onMap)
	if err != nil || s.Mission == 0 {
		return s, label, err
	}
	if f.live == nil || f.live.world == nil || f.live.mission == nil || f.live.mission.state == nil {
		return Snapshot{}, "", fmt.Errorf("mission comparison requires the current World and roster bindings")
	}
	m := f.live.mission
	s.Gold = int(f.live.world.Purse(sim.SelfSlot))
	s.Party = mapload.CarryRoster(m.party, f.live.world, m.ids, m.state.Start.Roster)
	return s, label, nil
}

func TestRoundTripObservationUsesCurrentMissionHoldings(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 10, MaxHP: 10}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{9, 5, 7}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)},
		CampaignSession: CampaignSession{Town: saveTown(t), Carried: []mapload.PartyMember{{ID: "hero"}}, liveMission: 20}}
	f.live = &mapWorld{world: w, mission: &missionNotices{party: f.Carried, ids: []sim.EntityID{1}, entryCount: 1, state: &Mission{World: w}}}
	before, _, err := currentRoundTripSnapshot(f, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.TakeSack(1, 3, 3); err != nil {
		t.Fatal(err)
	}
	wantHash := w.Hash()
	after, _, err := currentRoundTripSnapshot(f, true)
	if err != nil {
		t.Fatal(err)
	}
	if before.Gold != 0 || after.Gold != 40 || reflect.DeepEqual(roundTripPack(before.Party[0]), roundTripPack(after.Party[0])) {
		t.Fatal("comparison missed the current purse or acquired items")
	}
	if len(roundTripPack(after.Party[0])) != 3 || w.Hash() != wantHash || len(mapload.MemberCarriedItems(f.Carried[0], nil)) != 0 {
		t.Fatal("observation changed the World or entry roster")
	}
	town, _, err := currentRoundTripSnapshot(f, false)
	if err != nil || town.Gold != f.Town.Gold() || !reflect.DeepEqual(town.Party, f.Carried) {
		t.Fatal("town comparison stopped using the town's state", err)
	}
}
