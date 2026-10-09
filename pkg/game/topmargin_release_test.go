package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"testing"
)

func TestReleaseTopRowIgnoresTerrainAndOccupants(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("top margin")
	a.Layout(1024, 768)
	kept, blocked := 0, 0
	for n := 1; n <= 200; n++ {
		if err := a.OpenMission(f.MissionOpener(n)); err != nil {
			continue
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		m := f.live.mission.state.Map
		plane := mapload.Passability(m)
		walk := 0
		for x := 8; x < m.Width-8; x++ {
			if plane[8*m.Width+x]&1 == 0 {
				walk++
			}
		}
		cam := f.live.view.Camera()
		cam.Y = -1e9
		cam.Clamp()
		origin := terrain.Project(m.Altitudes, m.Width, m.Height).MinV
		if cam.Y+float64(origin) != 256 {
			t.Errorf("mission %d native upper origin %v, want 256", n, cam.Y+float64(origin))
		}
		if walk == 0 {
			blocked++
		} else {
			kept++
		}
	}
	if kept == 0 || blocked == 0 {
		t.Fatalf("open/blocked-first-row maps %d/%d", kept, blocked)
	}
	t.Logf("%d opened maps reach row8: mapload.Passability counts %d partly ground-open and %d wholly ground-blocked rows", kept+blocked, kept, blocked)
}
