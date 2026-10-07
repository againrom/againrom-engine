package game

import (
	"fmt"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Authored mission scripts must extend their temporary barriers before the
// normal spell lifetime ends, then let an ordinary lever use remove its wall.
func TestReleaseScriptClockKeepsWallsUntilLever(t *testing.T) {
	for _, tc := range []struct{ mission, walls int }{{91, 2}, {101, 8}} {
		t.Run(fmt.Sprintf("mission%d", tc.mission), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.NextParty()
			prepareAcceptedCampaignMission(t, f, tc.mission)
			f.Carried = mapload.CloneParty(party)
			a := f.App("script clock walls")
			a.Layout(1024, 768)
			if err := a.OpenMission(f.MissionOpener(tc.mission)); err != nil {
				t.Fatal(err)
			}
			live := f.live
			for i := 0; i < 40; i++ {
				live.tick()
			}
			// Cut before either mission's extension condition becomes true.
			dir := t.TempDir()
			save, _, _ := nativeContinuationSeams1170(t, f, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(650, 0) })
			name, err := save(true)
			if err != nil {
				t.Fatal(err)
			}
			cold := releaseFront(t)
			cold.SetDeterministicFrames(true)
			_, _, load := agsSaveSeams(cold, SaveStore{Dir: dir}, OriginalStore{}, nil)
			open, town, err := load(name)
			if err != nil || town {
				t.Fatalf("fresh LOAD: town=%v err=%v", town, err)
			}
			ca := cold.App("loaded script clock walls")
			ca.Layout(1024, 768)
			if err := ca.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			advance := func() {
				live.tick()
				cold.live.tick()
				if live.world.Hash() != cold.live.world.Hash() {
					t.Fatalf("SAVE/LOAD diverged at tick %d", live.world.Tick())
				}
			}
			for live.world.Tick() < 1200 {
				advance()
			}
			walls := func() []sim.CellEffect {
				var found []sim.CellEffect
				for _, effect := range live.world.CellEffects() {
					if effect.Spell == 19 {
						found = append(found, effect)
					}
				}
				return found
			}
			retained := walls()
			if len(retained) != tc.walls {
				t.Fatalf("at tick1200, %d authored stone walls remain; want%d", len(retained), tc.walls)
			}
			for _, wall := range retained {
				if wall.Remaining < 28000 {
					t.Fatalf("wall(%d,%d) was not extended to30000: remaining=%d", wall.X, wall.Y, wall.Remaining)
				}
			}
			t.Logf("mission%d: %d extended walls remain at tick%d, counter93=%d, fresh LOAD hashes match",
				tc.mission, len(retained), live.world.Tick(), live.world.ScriptRegister(93))
			if tc.mission != 101 {
				return
			}
			// Authored lever71 (native10) controls only spell19 at(60,59).
			lever := live.world.Structures()[10]
			if lever.Kind != 29 || lever.Col != 56 || lever.Row != 66 || lever.Field42 != 1 {
				t.Fatalf("unexpected installed lever: %+v", lever)
			}
			cmd := sim.Command{Kind: sim.KindUseStructure, Entity: live.mission.ids[0], X: int32(lever.ID)}
			live.pending = append(live.pending, cmd)
			cold.live.pending = append(cold.live.pending, cmd)
			for i := 0; i < 3000 && len(walls()) == 8; i++ {
				advance()
			}
			if live.world.Structures()[10].Field42 != 0 || len(walls()) != 7 {
				t.Fatalf("lever use did not release one wall: state=%d walls=%d", live.world.Structures()[10].Field42, len(walls()))
			}
			for _, wall := range walls() {
				if wall.X == 60 && wall.Y == 59 {
					t.Fatal("the lever's selected wall survived")
				}
			}
			t.Logf("lever71 used at tick%d: selected wall(60,59) gone, seven others remain; native continuation agrees", live.world.Tick())
		})
	}
}
