package ui

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
)

type fadingMusicDevice struct {
	recordingMusicDevice
	volume              int
	volumes             []int
	remaining, buffered time.Duration
}

func fadeFixture(t *testing.T) (*MusicController, *fadingMusicDevice, *recordingMusicSource, *time.Time) {
	t.Helper()
	d := &fadingMusicDevice{remaining: 10 * time.Second, buffered: 10 * time.Second}
	s := &recordingMusicSource{}
	m := NewMusicController(secondMusic, s, d, nil)
	now := time.Unix(0, 0)
	m.now = func() time.Time { return now }
	m.SetScene(MusicMission, false)
	return m, d, s, &now
}

func finishFade(t *testing.T, m *MusicController, now *time.Time) {
	t.Helper()
	for i := 0; i < 200 && m.fade != nil && m.active; i++ {
		*now = now.Add(100 * time.Millisecond)
		m.Update()
	}
	if m.fade != nil && m.active {
		t.Fatal("fade did not finish")
	}
}

func TestMusicFadeUsesClaimedStepsAndTimeCaps(t *testing.T) {
	for _, tc := range []struct {
		name                string
		initial             int
		remaining, buffered time.Duration
		want                int
	}{
		{"full", 0, 10 * time.Second, 10 * time.Second, -367},
		{"configured", -700, 10 * time.Second, 10 * time.Second, -1029},
		{"quarter", -2000, 10 * time.Second, 10 * time.Second, -2270},
		{"negative-base", -6000, 10 * time.Second, 10 * time.Second, -6120},
		{"source-cap", 0, 500 * time.Millisecond, 10 * time.Second, -1470},
		{"buffer-cap", 0, 10 * time.Second, 250 * time.Millisecond, -2940},
		{"unknown-times", 0, -1, -1, -367},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, d, _, now := fadeFixture(t)
			d.volume, d.remaining, d.buffered = tc.initial, tc.remaining, tc.buffered
			m.SetPreferences(MusicPreferences{RandomOrder: true})
			*now = now.Add(99 * time.Millisecond)
			m.Update()
			if len(d.volumes) != 0 {
				t.Fatal("step before due time")
			}
			*now = now.Add(time.Millisecond)
			m.Update()
			if d.volume != tc.want {
				t.Fatalf("first step = %d, want %d", d.volume, tc.want)
			}
			m.Update()
			if len(d.volumes) != 1 {
				t.Fatal("more than one step before next due time")
			}
			finishFade(t, m, now)
			if m.active || !slices.Contains(d.events, "stop") {
				t.Fatal("disabled music did not stop after fade")
			}
		})
	}
}

func TestMusicFadeStopsAfterCrossingAndBoundsOpenEdges(t *testing.T) {
	m, d, _, now := fadeFixture(t)
	m.fade = &musicFade{volume: -8000, threshold: 8000, scale: 1, due: *now, interval: 100 * time.Millisecond}
	m.Update()
	if d.volume != -8002 || !m.active {
		t.Fatal("threshold equality did not step before stopping", d.volume)
	}
	*now = now.Add(100 * time.Millisecond)
	m.Update()
	if m.active {
		t.Fatal("crossing did not stop on the next due update")
	}
	for _, tc := range []struct {
		name      string
		volume    int
		remaining time.Duration
	}{
		{"zero-time", 0, 0}, {"threshold-start", -8000, time.Second}, {"below-threshold", -9000, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, d, _, now := fadeFixture(t)
			d.volume, d.remaining = tc.volume, tc.remaining
			m.SetPreferences(MusicPreferences{RandomOrder: true})
			*now = now.Add(100 * time.Millisecond)
			m.Update()
			if m.active || len(d.volumes) != 0 {
				t.Fatal("edge did not complete without a step")
			}
		})
	}
}

func TestMusicTuneHoldsThenAreaPickWinsAndRandomClearsHold(t *testing.T) {
	m, d, s, now := fadeFixture(t)
	if !m.Tune(3) || !m.hold || m.Playing() != "B00.wav" {
		t.Fatal("tune did not arm a held cross-fade")
	}
	finishFade(t, m, now)
	if m.Playing() != "B03.wav" || !m.hold || d.volume != 0 {
		t.Fatal("fade did not reselect held entry and restore volume")
	}
	d.ended = true
	m.Update()
	if m.Playing() != "B03.wav" {
		t.Fatal("held entry did not repeat")
	}
	m.pick, d.ended = 6, true
	m.Update()
	if m.Playing() != "B06.wav" || !m.hold {
		t.Fatal("area pick lost to hold")
	}
	m.SetScene(MusicSilent, false)
	if m.Playing() != "B03.wav" || s.loads[len(s.loads)-1] != "B03.wav" {
		t.Fatal("held stop did not reselect held index")
	}
	m.SetPreferences(MusicPreferences{Enabled: true, RandomOrder: false})
	if m.hold {
		t.Fatal("set random did not clear hold")
	}
	m.pick, d.ended = -1, true
	m.Update()
	if m.Playing() != "B04.wav" {
		t.Fatal("ordinary progression did not return")
	}
}

func TestMusicHeldListReplacementKeepsHoldAndNewSelection(t *testing.T) {
	m, d, s, now := fadeFixture(t)
	m.Tune(3)
	finishFade(t, m, now)
	before := len(s.loads)
	m.RequestScene(MusicMission, false)
	if !m.hold || m.heldIndex != 3 || m.pick != -1 || m.Playing() != "B00.wav" {
		t.Fatal("list replacement lost hold or failed to select the new list")
	}
	if !slices.Equal(s.loads[before:], []string{"B03.wav", "B00.wav"}) {
		t.Fatal("missing interim held selection", s.loads[before:])
	}
	d.ended = true
	m.Update()
	if m.Playing() != "B00.wav" {
		t.Fatal("EOF used the old held index instead of current new-list index")
	}
	if m.Tune(-1) || m.Tune(100) {
		t.Fatal("invalid tune admitted")
	}
}

func TestMusicManualSelectionUsesHeldStopBeforeTheNewIndex(t *testing.T) {
	m, _, source, now := fadeFixture(t)
	m.Tune(3)
	finishFade(t, m, now)
	before := len(source.loads)
	if !m.Play(5) || !m.hold || m.Playing() != "B05.wav" {
		t.Fatal("manual selection lost hold or selected another entry")
	}
	if !slices.Equal(source.loads[before:], []string{"B03.wav", "B05.wav"}) {
		t.Fatal("manual selection bypassed the held stop", source.loads[before:])
	}
}

func TestMusicOffDuringFadeStopsImmediatelyAndRetainsFade(t *testing.T) {
	m, d, _, now := fadeFixture(t)
	m.SetPreferences(MusicPreferences{RandomOrder: true})
	m.SetPreferences(MusicPreferences{RandomOrder: true})
	if m.active || m.fade == nil || !slices.Contains(d.events, "stop") {
		t.Fatal("state-2 off did not stop and retain fade")
	}
	m.SetPreferences(MusicPreferences{Enabled: true, RandomOrder: true})
	if !m.Play(0) || m.fade == nil {
		t.Fatal("start cleared the retained fade")
	}
	*now = now.Add(100 * time.Millisecond)
	m.Update()
	if d.volume != -367 {
		t.Fatal("retained fade did not resume", d.volume)
	}
}

func TestMusicDeviceFadeUsesTheSharedDecibelGainAndRetainsSettings(t *testing.T) {
	d := &musicDevice{ctx: newReadingDevice(audio.DeviceRate), st: audio.Settings{Master: audio.MasterUnit / 2}}
	t.Cleanup(d.Stop)
	d.SelectMusic(audio.Track{Rate: audio.DeviceRate, StereoPCM: make([]byte, 4*audio.DeviceRate*10)}, false)
	if d.MusicVolume() != -602 {
		t.Fatal("configured gain did not map to decibels", d.MusicVolume())
	}
	r, q := d.MusicTimes()
	if r != 10*time.Second || q != -1 {
		t.Fatal("source time or unknown device buffer", r, q)
	}
	d.SetMusicVolume(-2000)
	if d.pl.Volume() != 0.1 {
		t.Fatal("fade gain did not use shared conversion", d.pl.Volume())
	}
	d.SetSettings(audio.Settings{Master: audio.MasterUnit / 2, Muted: true})
	if d.pl.Volume() != 0 {
		t.Fatal("fade bypassed mute")
	}
	d.SetSettings(audio.Settings{Master: audio.MasterUnit / 2})
	if d.pl.Volume() != 0.1 {
		t.Fatal("settings discarded fade gain")
	}
	d.ResetMusicVolume()
	if d.pl.Volume() != 0.5 {
		t.Fatal("configured gain was not restored")
	}
}

type cueTownMusic struct {
	recordingTownMusic
	MusicCue
}

func TestAppConsumesDialogueMusicCuesThroughTheSamePlayer(t *testing.T) {
	for _, desc := range []*MusicDescription{firstMusic, secondMusic} {
		source, d := &recordingMusicSource{}, &fadingMusicDevice{remaining: 10 * time.Second, buffered: 10 * time.Second}
		a := newTestApp(t, appRows(1), okLoader(t))
		town := &cueTownMusic{recordingTownMusic: recordingTownMusic{scene: MusicMission}}
		a.SetTown(town)
		if !a.flow.showTown("") {
			t.Fatal("town did not open")
		}
		a.SetMusic(desc, source, d, nil)
		now := time.Unix(0, 0)
		a.music.now = func() time.Time { return now }
		town.QueueMusicTune(3)
		a.updateMusic()
		if desc == firstMusic {
			if a.music.hold || a.music.fade != nil || a.music.Playing() != "B00.wav" {
				t.Fatal("first game's tune behaviour changed")
			}
			continue
		}
		finishFade(t, a.music, &now)
		if a.music.Playing() != "B03.wav" || !a.music.hold {
			t.Fatal("town cue failed to drive player")
		}
		a.updateMusic()
		if a.music.fade != nil {
			t.Fatal("consumed cue armed again")
		}
	}
}

func TestAppConsumesMissionMusicCueThroughTheSamePlayer(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	d := &fadingMusicDevice{remaining: 10 * time.Second, buffered: 10 * time.Second}
	a.SetMusic(secondMusic, &recordingMusicSource{}, d, nil)
	now := time.Unix(0, 0)
	a.music.now = func() time.Time { return now }
	a.flow.viewer.QueueMusicTune(2)
	a.updateMusic()
	finishFade(t, a.music, &now)
	if a.music.Playing() != "B02.wav" || !a.music.hold {
		t.Fatal("mission cue failed to drive player")
	}
}

func (d *fadingMusicDevice) MusicVolume() int  { return d.volume }
func (d *fadingMusicDevice) ResetMusicVolume() { d.volume = 0 }
func (d *fadingMusicDevice) SetMusicVolume(v int) {
	d.volume = v
	d.volumes = append(d.volumes, v)
}
func (d *fadingMusicDevice) MusicTimes() (time.Duration, time.Duration) {
	return d.remaining, d.buffered
}

func TestMusicOffDefersStopOnlyForTheFadeProfile(t *testing.T) {
	for _, desc := range []*MusicDescription{firstMusic, secondMusic} {
		t.Run(desc.Music, func(t *testing.T) {
			d := &fadingMusicDevice{remaining: 10 * time.Second, buffered: 10 * time.Second}
			m := NewMusicController(desc, &recordingMusicSource{}, d, nil)
			m.SetScene(MusicMenu, false)
			m.SetPreferences(MusicPreferences{})
			stopped := slices.Contains(d.events, "stop")
			if desc == secondMusic && (stopped || m.Playing() == "") {
				t.Fatalf("music off stopped immediately instead of fading: %v", d.events)
			}
			if desc == firstMusic && (!stopped || m.Playing() != "") {
				t.Fatalf("first game's immediate stop changed: %v", d.events)
			}
		})
	}
}
