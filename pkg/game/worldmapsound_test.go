package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/town"
)

type routeVoice struct{ playing bool }

func (v *routeVoice) Playing() bool { return v.playing }
func (v *routeVoice) Stop()         { v.playing = false }

// secondTravel is a second-game map with a travel running and every sound
// request recorded with its voice.
func secondTravel(t *testing.T) (*worldMapState, *[]string, *[]*routeVoice) {
	t.Helper()
	a, b := image.Pt(20, 50), image.Pt(620, 50)
	s := secondMapState(t, secondMapSelect, image.Pt(320, 300), secondRecord(1, 21, a), secondRecord(2, 2, b))
	s.aim(image.Pt(619, 50))
	s.startTravel(s.hovered)
	s.route = make([]image.Point, 10*worldMapRouteReveal)
	var keys []string
	var voices []*routeVoice
	s.play = func(key string) town.Voice {
		v := &routeVoice{playing: true}
		keys, voices = append(keys, key), append(voices, v)
		return v
	}
	return s, &keys, &voices
}

// While the route grows, Point2 starts first and each later start is the
// other sound once the current one has no channel; none starts while the
// current one plays or once the reveal has reached the route's end
// (R2-ENGINE-463).
func TestSecondMapRouteSoundsAlternateFromPointTwo(t *testing.T) {
	s, keys, voices := secondTravel(t)
	if len(s.route) < 4*worldMapRouteReveal {
		t.Fatalf("the route has %d points, too short to alternate", len(s.route))
	}
	s.tick()
	if !slices.Equal(*keys, []string{"point2.wav"}) {
		t.Fatalf("first tick requests %v, want point2.wav", *keys)
	}
	s.tick()
	s.tick()
	if len(*keys) != 1 {
		t.Fatalf("a playing sound was followed by %v", *keys)
	}
	(*voices)[0].playing = false
	s.tick()
	(*voices)[1].playing = false
	s.tick()
	(*voices)[2].playing = false
	s.tick()
	want := []string{"point2.wav", "point1.wav", "point2.wav", "point1.wav"}
	if !slices.Equal(*keys, want) {
		t.Fatalf("requests %v, want %v", *keys, want)
	}
	for i := 0; i < len(s.route); i++ {
		for _, v := range *voices {
			v.playing = false
		}
		if s.tick() {
			break
		}
	}
	n := len(*keys)
	for _, v := range *voices {
		v.playing = false
	}
	s.tick()
	if len(*keys) != n {
		t.Fatalf("a sound started with the reveal past the route's end")
	}
}

// Leaving the map stops both sounds and forgets them.
func TestSecondMapLeaveStopsRouteSounds(t *testing.T) {
	s, _, voices := secondTravel(t)
	s.tick()
	(*voices)[0].playing = true
	s.stopSounds(func(v town.Voice) { v.Stop() })
	if (*voices)[0].playing || s.voices[0] != nil || s.voices[1] != nil {
		t.Fatal("a route sound outlived the map")
	}
}

// The second game's description names the two route sounds and the sheets
// whose frame counts wrap the counters; the first game's names none.
func TestMapDescriptionsNameSoundsAndCycles(t *testing.T) {
	d := worldMapDescriptions["rom2"]
	if d.Sounds == nil || !slices.Equal(d.Sounds.Route, []string{"point1.wav", "point2.wav"}) {
		t.Fatalf("second game sounds = %+v", d.Sounds)
	}
	if d.Cycle != (WorldMapCycle{Markers: "flag1", Party: "flag"}) {
		t.Fatalf("second game cycle = %+v", d.Cycle)
	}
	if r := worldMapDescriptions["rom1"]; r.Sounds != nil || r.Cycle != (WorldMapCycle{}) {
		t.Fatalf("first game names sounds or cycles: %+v %+v", r.Sounds, r.Cycle)
	}
}

// Marker frames follow Flag1's frame count and the party's frames Flag's,
// whatever the drawn sheet's own length (R2-ENGINE-430).
func TestSecondMapFramesWrapByTheNamedSheets(t *testing.T) {
	s := secondMapState(t, secondMapSelect, image.Pt(200, 300), secondRecord(1, 21, image.Pt(100, 100)))
	s.assets.markCycle, s.assets.partyCycle = 3, 2
	s.aim(image.Pt(100, 100))
	for tick := 0; tick < 12; tick++ {
		wantMark := s.assets.rules[0][tick%3%len(s.assets.rules[0])]
		if got, _ := s.marker(0); got != image.Image(wantMark) {
			t.Fatalf("tick %d: marker is not frame %d of Flag1's cycle of 3", tick, tick%3)
		}
		party := s.assets.mode(s.mode).party
		if got := s.assets.flagFrame(s.mode, s.frame); got != party[tick%2] {
			t.Fatalf("tick %d: party is not frame %d of Flag's cycle of 2", tick, tick%2)
		}
		s.tick()
	}
}

// The first game's counters wrap by the drawn sheet's own frame count.
func TestFirstMapFramesWrapByTheirOwnSheet(t *testing.T) {
	s := secondMapState(t, secondMapSelect, image.Pt(200, 300))
	s.assets.desc = worldMapDescriptions["rom1"]
	party := s.assets.mode(s.mode).party
	for n := 0; n < 10; n++ {
		if s.assets.flagFrame(s.mode, n) != party[n%len(party)] {
			t.Fatalf("frame %d does not wrap by the sheet's %d frames", n, len(party))
		}
	}
}
