package game

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/town"
)

var yardKeys = []string{"town/school/command1.wav", "town/school/command2.wav", "town/school/fight1.wav", "town/school/fight2.wav", "town/school/command3.wav"}

func yardFixture(t *testing.T, high bool, mage bool) (*townScreen, *time.Time, *roomSoundRecorder) {
	t.Helper()
	f, s, now, _ := schoolTrainingFixture(t)
	rec := addRoomSounds(f)
	for _, key := range yardKeys {
		f.SoundBank.named[key] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1}}, ok: true}
	}
	f.TownAmbientRandom = func(n int) int {
		if high {
			return n - 1
		}
		return 0
	}
	if mage {
		s.shopMember = 1
	}
	s.enterSchoolPage()
	rec.requests, rec.voices = nil, nil
	return s, now, rec
}

func yardSelectors(rec *roomSoundRecorder) []string {
	out := []string{}
	for _, r := range rec.requests {
		for _, key := range yardKeys {
			if r.Selector == key {
				out = append(out, key)
			}
		}
	}
	return out
}

func TestSchoolYardSoundIntervalsOrderAndStop(t *testing.T) {
	for _, high := range []bool{false, true} {
		for _, mage := range []bool{false, true} {
			t.Run(map[bool]string{false: "lower", true: "upper"}[high]+map[bool]string{false: "fighter", true: "mage"}[mage], func(t *testing.T) {
				s, now, rec := yardFixture(t, high, mage)
				start := *now
				paint := func(ms int) { *now = start.Add(time.Duration(ms) * time.Millisecond); s.AdvanceTownSurfaceAnimation() }
				want := []string{}
				check := func() {
					t.Helper()
					if got := yardSelectors(rec); !reflect.DeepEqual(got, want) {
						t.Fatalf("at %v: got %v want %v", now.Sub(start), got, want)
					}
				}
				paint(1500)
				check()
				paint(1501)
				want = append(want, yardKeys[0])
				check()
				delay, duration := 3500, 40000
				if high {
					delay, duration = 4499, 79999
				}
				paint(delay)
				check()
				paint(delay + 1)
				fight := yardKeys[2]
				if mage {
					fight = yardKeys[3]
				}
				want = append(want, yardKeys[1], fight)
				check()
				var voice *exteriorVoice
				for i, r := range rec.requests {
					if r.Selector == fight {
						if !r.Repeat || r.Priority != 128 || r.Group != audio.EffectsChannel {
							t.Fatalf("fight request %+v", r)
						}
						voice = rec.voices[i]
					}
				}
				if voice == nil || !voice.Playing() {
					t.Fatal("no playing fight")
				}
				paint(delay + 1 + duration)
				check()
				if !voice.Playing() {
					t.Fatal("fight stopped at equality")
				}
				paint(delay + 2 + duration)
				want = append(want, yardKeys[4])
				check()
				for i, r := range rec.requests {
					if r.Selector == yardKeys[1] && !rec.voices[i].Playing() {
						t.Fatal("fight end stopped Command2")
					}
				}
				if voice.Playing() || voice.stops != 1 {
					t.Fatalf("fight stop/reset: %+v", voice)
				}
				rec.endAll()
				paint(delay + 2 + duration + 1500)
				check()
				paint(delay + 2 + duration + 1501)
				check()
				closed := delay + 2 + duration
				paint(closed + delay)
				check()
				paint(closed + delay + 1)
				want = append(want, yardKeys[1], fight)
				check()
				voice = rec.voices[len(rec.voices)-1]
				paint(closed + delay + 1 + duration)
				check()
				if !voice.Playing() {
					t.Fatal("second fight stopped at equality")
				}
				paint(closed + delay + 2 + duration)
				want = append(want, yardKeys[4])
				check()
				if voice.Playing() || voice.stops != 1 {
					t.Fatalf("second fight stop/reset: %+v", voice)
				}
				rec.endAll()
				paint(closed + delay + 2 + duration + 1501)
				check()
				s.enterSchoolPage()
				rec.requests, rec.voices = nil, nil
				start, want = *now, []string{}
				paint(1500)
				check()
				paint(1501)
				want = append(want, yardKeys[0])
				check()
			})
		}
	}
}

func TestSchoolYardSoundPaintColumnAndLifecycle(t *testing.T) {
	s, now, rec := yardFixture(t, false, false)
	*now = now.Add(2 * time.Second)
	s.TownSurface()
	if len(yardSelectors(rec)) != 0 {
		t.Fatal("a snapshot advanced the yard")
	}
	s.schoolColumn().Frame = 7
	s.AdvanceTownSurfaceAnimation()
	if len(yardSelectors(rec)) != 0 {
		t.Fatal("the yard ran between column endpoints")
	}
	s.schoolColumn().Rest(0)
	s.AdvanceTownSurfaceAnimation()
	if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0]}) {
		t.Fatal(got)
	}
	*now = now.Add(2 * time.Second)
	s.AdvanceTownSurfaceAnimation()
	fight := rec.voices[len(rec.voices)-1]
	if !fight.Playing() {
		t.Fatal("fight did not start")
	}
	s.SchoolTrainingActive(false)
	if fight.Playing() {
		t.Fatal("pause kept a fight playing")
	}
	before := len(rec.requests)
	*now = now.Add(time.Hour)
	s.AdvanceTownSurfaceAnimation()
	if len(rec.requests) != before {
		t.Fatal("paused paint requested sound")
	}
	s.SchoolTrainingActive(true)
	s.AdvanceTownSurfaceAnimation()
	if len(rec.requests) != before {
		t.Fatal("resume caught up hidden time")
	}
	*now = now.Add(1501 * time.Millisecond)
	s.AdvanceTownSurfaceAnimation()
	if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0], yardKeys[1], yardKeys[2]}) {
		t.Fatal(got)
	}
	rec.endAll()
	*now = now.Add(2 * time.Second)
	s.AdvanceTownSurfaceAnimation()
	if got := yardSelectors(rec); !reflect.DeepEqual(got, []string{yardKeys[0], yardKeys[1], yardKeys[2], yardKeys[1], yardKeys[2]}) {
		t.Fatal(got)
	}
	s.leaveSchoolTraining()
	for _, v := range rec.voices {
		if v.Playing() {
			t.Fatal("leave retained a voice")
		}
	}
	s.enterSchoolPage()
	base := len(rec.requests)
	*now = now.Add(1500 * time.Millisecond)
	s.AdvanceTownSurfaceAnimation()
	if len(rec.requests) != base {
		t.Fatal("re-entry retained old timers")
	}
}

func TestSchoolYardSoundLossControls(t *testing.T) {
	cases := []struct {
		name       string
		high, mage bool
		mutate     func(*town.SceneSpec)
	}{
		{name: "no-paint-step", mutate: func(p *town.SceneSpec) { p.Steps[0].Run = p.Steps[0].Run[1:] }},
		{name: "opening-bound", mutate: func(p *town.SceneSpec) { p.Actors[0].DelayMS = 1501 }},
		{name: "start-lower", mutate: func(p *town.SceneSpec) { p.Actors[0].Delay.Base = 3499 }},
		{name: "start-upper", high: true, mutate: func(p *town.SceneSpec) { p.Actors[0].Delay.N = 999 }},
		{name: "stop-lower", mutate: func(p *town.SceneSpec) { p.Actors[0].HoldPick.Base = 39999 }},
		{name: "stop-upper", high: true, mutate: func(p *town.SceneSpec) { p.Actors[0].HoldPick.N = 39999 }},
		{name: "request-order", mutate: func(p *town.SceneSpec) { p.Actors[0].States[0].StartSounds = []string{"fight1", "command2"} }},
		{name: "fight1-loop", mutate: func(p *town.SceneSpec) {
			for i := range p.Sounds.Slots {
				if p.Sounds.Slots[i].Name == "fight1" {
					p.Sounds.Slots[i].Loop = false
				}
			}
		}},
		{name: "fight2-loop", mage: true, mutate: func(p *town.SceneSpec) {
			for i := range p.Sounds.Slots {
				if p.Sounds.Slots[i].Name == "fight2" {
					p.Sounds.Slots[i].Loop = false
				}
			}
		}},
		{name: "fight-stop", mutate: func(p *town.SceneSpec) {
			for i := range p.Actors[0].States {
				p.Actors[0].States[i].EndSounds = nil
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, now, rec := yardFixture(t, tc.high, tc.mage)
			spec := s.townDescription().Scene("school")
			backup, err := json.Marshal(spec)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := json.Unmarshal(backup, spec); err != nil {
					t.Fatal(err)
				}
			}()
			tc.mutate(spec)
			s.enterSchoolPage()
			rec.requests, rec.voices = nil, nil
			start := *now
			paint := func(ms int) { *now = start.Add(time.Duration(ms) * time.Millisecond); s.AdvanceTownSurfaceAnimation() }
			accepted := true
			check := func(want ...string) {
				accepted = accepted && reflect.DeepEqual(yardSelectors(rec), append([]string{}, want...))
			}
			paint(1500)
			check()
			paint(1501)
			check(yardKeys[0])
			delay, hold := 3500, 40000
			if tc.high {
				delay, hold = 4499, 79999
			}
			paint(delay)
			check(yardKeys[0])
			paint(delay + 1)
			fight := yardKeys[2]
			if tc.mage {
				fight = yardKeys[3]
			}
			check(yardKeys[0], yardKeys[1], fight)
			var voice *exteriorVoice
			for i, r := range rec.requests {
				if r.Selector == fight {
					accepted = accepted && r.Repeat
					voice = rec.voices[i]
				}
			}
			accepted = accepted && voice != nil
			paint(delay + 1 + hold)
			check(yardKeys[0], yardKeys[1], fight)
			accepted = accepted && voice != nil && voice.Playing()
			paint(delay + 2 + hold)
			check(yardKeys[0], yardKeys[1], fight, yardKeys[4])
			accepted = accepted && voice != nil && !voice.Playing()
			if accepted {
				t.Fatal("the interval, order and loop oracle accepted the lost behaviour")
			}
		})
	}
}
