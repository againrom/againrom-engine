package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

// TestReleaseOriginalStreamStartDrawsAFacingForEveryRingPlacement starts the
// three shipped missions that hold a placement on the map's border ring (80,
// 90, 100) on the original's stream. The original constructs each map record,
// drawing its facing, before the cell refusal deletes a ring record
// (MOVE-120, SESS-090, MOVE-141). An independent MSVC sequence draws once per
// authored record in record order: the kept records take their draw as their
// start facing and the withdrawn ones discard it, then the drop pick follows.
func TestReleaseOriginalStreamStartDrawsAFacingForEveryRingPlacement(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	for _, tc := range []struct{ mission, withdrawn int }{{80, 1}, {90, 1}, {100, 2}} {
		addr, ok := MissionMap(tc.mission)
		if !ok {
			t.Fatalf("mission %d has no map", tc.mission)
		}
		b, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatal(err)
		}
		m, err := tableFiles(f.Table).openMap(b)
		if err != nil {
			t.Fatal(err)
		}
		if got := mapload.WithdrawBorderPlacements(m); got != tc.withdrawn {
			t.Fatalf("mission %d: %d ring placements, want %d", tc.mission, got, tc.withdrawn)
		}
		kept := map[uint16]bool{}
		for _, u := range m.Units {
			kept[u.UnitID] = true
		}
		const state = 0x2468ace
		ref := random.MSVC{State: state}
		var want []uint8
		for _, u := range m.AuthoredUnits {
			v := uint8(0x40 + ref.Range(0x80))
			if kept[u.UnitID] {
				want = append(want, v)
			}
		}
		drops := mapload.DropCells(m)
		wantDrop := drops[ref.Range(int32(len(drops)-1))]
		draws := sim.NewOriginalDraws(state)
		ms, err := startMissionFromWith(m, addr, tc.mission, f.Table, openDifficulty, nil, draws)
		if err != nil {
			t.Fatalf("mission %d: %v", tc.mission, err)
		}
		if len(want) != len(m.Units) {
			t.Fatalf("mission %d: %d kept ids for %d kept records", tc.mission, len(want), len(m.Units))
		}
		ents := ms.World.Entities()
		for i, v := range want {
			if ents[i].Facing != v {
				t.Fatalf("mission %d: record %d faces %#x, want %#x", tc.mission, i, ents[i].Facing, v)
			}
		}
		if ms.Start.Drop != wantDrop {
			t.Errorf("mission %d: drop %v, want %v after %d facing draws", tc.mission, ms.Start.Drop, wantDrop,
				len(m.AuthoredUnits))
		}
	}
}
