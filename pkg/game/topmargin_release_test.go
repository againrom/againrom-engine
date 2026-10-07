package game

import (
	"testing"

	"againrom/pkg/mapload"
)

func TestReleaseTopRowHiddenOnlyWhenImpassableAndEmpty(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("top margin")
	a.Layout(1024, 768)
	kept, hidden, ogreSack := 0, 0, false
	for n := 1; n <= 200; n++ {
		if err := a.OpenMission(f.MissionOpener(n)); err != nil {
			continue
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		m := f.live.mission.state.Map
		plane := mapload.Passability(m)
		w, h := m.Width, m.Height
		first := -1
		for y := 0; y < h && first < 0; y++ {
			for x := 0; x < w; x++ {
				if plane[y*w+x]&2 == 0 {
					first = y
					break
				}
			}
		}
		if first < 0 {
			t.Errorf("mission %d: no playable row", n)
			continue
		}
		walk, held := 0, 0
		for x := 0; x < w; x++ {
			if c := plane[first*w+x]; c&2 == 0 && c&1 == 0 {
				walk++
			}
		}
		for _, e := range f.live.world.Entities() {
			if int(e.Y) == first {
				held++
			}
		}
		for _, o := range m.Objects {
			if int(o.Y>>8) == first {
				held++
			}
		}
		sacks := 0
		for _, s := range f.live.world.Sacks() {
			if int(s.Y) == first {
				held++
				sacks++
			}
		}
		keep := walk > 0 || held > 0
		cam := f.live.view.Camera()
		limit := func() float64 {
			cam.Y = -1e9
			cam.Clamp()
			return cam.Y
		}
		got := limit()
		f.live.view.SetTopRowShown(!keep)
		other := limit()
		f.live.view.SetTopRowShown(keep)
		if keep && !(got < other) || !keep && !(got > other) {
			t.Errorf("mission %d: row %d walkable=%d occupants=%d keep=%v but the upper limit is %.1f against %.1f with the other choice", n, first, walk, held, keep, got, other)
		}
		if keep {
			kept++
		} else {
			hidden++
		}
		if n == 81 && sacks > 0 && keep {
			ogreSack = true
		}
		t.Logf("mission %d %dx%d first playable row %d: walkable=%d occupants=%d sacks=%d keep=%v", n, w, h, first, walk, held, sacks, keep)
	}
	if kept+hidden == 0 {
		t.Fatal("no mission opened")
	}
	t.Logf("%d maps keep the row, %d hide it", kept, hidden)
	if !ogreSack {
		t.Error("mission 81 holds no sack in its first playable row, or did not keep it")
	}
}
