package game

import (
	"againrom/pkg/sim"
	"testing"
)

func TestReleaseChickenLaunchAtEveryMissionStart(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		f := releaseFront(t)
		if f.ChickenAtMissionStart() {
			t.Fatal("launch option defaults on")
		}
		f.SetChickenAtMissionStart(enabled)
		f.SetDeterministicFrames(true)
		app := f.App("mission launch privilege")
		app.Layout(1024, 768)
		for _, mission := range []int{10, 20, 30} {
			if err := app.OpenMission(f.MissionOpener(mission)); err != nil {
				t.Fatal(err)
			}
			want := byte(0)
			if enabled {
				want = 255
			}
			if f.live.cheats.privilege[sim.SelfSlot] != want {
				t.Fatalf("launch=%t mission%d privilege%d", enabled, mission, f.live.cheats.privilege[sim.SelfSlot])
			}
			f.live.cheats.privilege[sim.SelfSlot] = 0
		}
	}
}
