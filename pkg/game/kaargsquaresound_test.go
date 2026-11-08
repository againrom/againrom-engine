package game

import (
	"encoding/json"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/town"
)

func TestKaargSquareRequestsItsTimedSounds(t *testing.T) {
	h := &kaargInnHost{now: time.Unix(100, 0), art: &town.Art{}}
	s := town.NewScene(townDescriptions["rom2-town2"], "square", h, nil)
	s.SetRawDraw("persons", func() int { return 0 })
	s.SetActive(true, true)
	s.Advance()
	h.log = nil
	h.now = h.now.Add(2 * time.Second)
	s.Advance()
	if got := h.plays(); len(got) != 0 {
		t.Fatalf("sounds at the strict 2000 ms boundary: %v", got)
	}
	h.now = h.now.Add(time.Millisecond)
	s.Advance()
	got := h.plays()
	for _, key := range []string{"town_kaarg/kvox2.wav", "town_kaarg/kbird1.wav"} {
		if !slices.Contains(got, key) {
			t.Errorf("no request for %s after 2000 ms", key)
		}
	}
	h.log = nil
	h.now = time.Unix(145, 0)
	s.Advance()
	if slices.Contains(h.plays(), "town_kaarg/kman1.wav") {
		t.Fatal("Kman1 requested at the strict 45000 ms boundary")
	}
	h.now = h.now.Add(time.Millisecond)
	s.Advance()
	if !slices.Contains(h.plays(), "town_kaarg/kman1.wav") {
		t.Fatal("no Kman1 request after 45000 ms")
	}
	s.SetActive(false, true)
	h.log = nil
	h.now = h.now.Add(time.Minute)
	s.Advance()
	if got := h.plays(); len(got) != 0 {
		t.Fatalf("inactive square requested %v", got)
	}
}

func TestKaargSquareGuardRequestsOneStepPerCue(t *testing.T) {
	for _, raw := range []int{0, 32766, 32767} {
		h := &kaargInnHost{now: time.Unix(100, 0), art: &town.Art{Frames: map[string][]image.Image{"guard": familyFrames(55, 1, 0)}}}
		s := town.NewScene(townDescriptions["rom2-town2"], "square", h, nil)
		s.SetRawDraw("persons", func() int { return raw })
		s.SetActive(true, true)
		s.Advance()
		guard := s.Actor("persons").(*town.Families).Member("guard")
		guard.Clock, guard.Wait, guard.Active = h.now, time.Hour, true
		for _, cue := range []struct {
			frame int
			key   string
		}{{5, "ksteps2"}, {9, "ksteps1"}, {13, "ksteps2"}, {17, "ksteps1"}, {21, "ksteps2"}, {23, "ksteps1"}, {31, "ksteps3"}, {47, "ksteps2"}, {51, "ksteps1"}} {
			guard.Current = 0
			s.Advance()
			h.log = nil
			guard.Current = cue.frame
			s.Advance()
			key := cue.key
			if raw == 0 || raw == 32767 {
				key += "1"
			}
			want := "town_kaarg/" + key + ".wav"
			if got := h.plays(); !slices.Equal(got, []string{want}) {
				t.Errorf("raw %d frame %d: %v, want %s", raw, cue.frame, got, want)
			}
			s.Advance()
			if got := h.plays(); len(got) != 0 {
				t.Errorf("held frame %d repeated %v", cue.frame, got)
			}
		}
	}
}

func TestKaargSquareSoundDataRejectsInvalidChoices(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*town.Description)
		want   string
	}{
		{"too few slots", func(d *town.Description) { d.Actors[1].Members[2].FrameCues[0].Slots = nil }, "picks slots"},
		{"unknown slot", func(d *town.Description) { d.Actors[1].Members[2].FrameCues[0].Slots[0] = "absent" }, "slot"},
		{"unknown source", func(d *town.Description) { d.Actors[1].Members[2].FrameCues[0].Pick.Draw = "absent" }, "draw"},
		{"raw unbounded pick", func(d *town.Description) { p := d.Actors[1].Members[2].FrameCues[0].Pick; p.Form = ""; p.N = 0 }, "picks slots"},
		{"duplicate clock", func(d *town.Description) { d.Clocks[0].Name = d.Square.Name }, "clock"},
		{"state frame choice", func(d *town.Description) {
			d.Rooms[0].Scene.Actors[0].States[0].FrameSounds = []town.FrameCue{{Frame: 0, Slot: "kman3", Pick: &town.PickSpec{Draw: "tender", N: 1}, Slots: []string{"kman3"}}}
		}, "frame choices need a family member"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var d town.Description
			if err := json.Unmarshal(rom2Town2JSON, &d); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&d)
			if err := d.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want %s", err, tc.want)
			}
		})
	}
}
