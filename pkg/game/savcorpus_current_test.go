package game

import (
	"fmt"

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
