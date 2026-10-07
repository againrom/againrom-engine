package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/audio"
)

type recordingMusicSource struct {
	loads     []string
	missing   map[string]bool
	malformed map[string]bool
	wrongRate map[string]bool
	oddFrame  map[string]bool
}

func (s *recordingMusicSource) Track(name string) (audio.Track, bool) {
	s.loads = append(s.loads, name)
	if s.missing[name] {
		return audio.Track{}, false
	}
	if s.malformed[name] {
		return audio.Track{Rate: audio.DeviceRate}, true
	}
	if s.wrongRate[name] {
		return audio.Track{Rate: audio.DeviceRate * 2, StereoPCM: []byte("rate")}, true
	}
	if s.oddFrame[name] {
		return audio.Track{Rate: audio.DeviceRate, StereoPCM: []byte("odd!!")}, true
	}
	pcm := make([]byte, (len(name)+3)&^3)
	copy(pcm, name)
	return audio.Track{Rate: audio.DeviceRate, StereoPCM: pcm}, true
}

type recordingMusicDevice struct {
	events   []string
	ended    bool
	settings []audio.Settings
	current  audio.Track
	maximum  int
}

func (d *recordingMusicDevice) Start(track audio.Track) {
	d.events = append(d.events, "start:"+strings.TrimRight(string(track.StereoPCM), "\x00"))
	d.current = track
	d.maximum = 1
	d.ended = false
}
func (d *recordingMusicDevice) Stop() {
	d.events = append(d.events, "stop")
	d.current = audio.Track{}
}
func (d *recordingMusicDevice) Ended() bool {
	return d.ended
}
func (d *recordingMusicDevice) SetSettings(st audio.Settings) {
	d.settings = append(d.settings, st)
}

func TestStaticMusicSceneTable(t *testing.T) {
	want := map[MusicScene][]string{
		MusicSilent:   nil,
		MusicMenu:     {"menu.wav"},
		MusicChargen:  {"chrgen.wav"},
		MusicCampaign: {"map.wav"},
		MusicMission: {
			"B00.wav", "B01.wav", "B02.wav", "B03.wav", "B04.wav", "B05.wav",
			"B06.wav", "B07.wav", "B08.wav", "B09.wav", "B10.wav", "B11.wav",
		},
		MusicTown:   {"town.wav"},
		MusicShop:   {"shop.wav"},
		MusicTavern: {"inn.wav"},
		MusicSchool: {"schoolw.wav", "schoolm.wav"},
	}
	for scene, tracks := range want {
		if got := staticMusicTracks(scene, false); !slices.Equal(got, tracks) {
			t.Errorf("scene %d = %v, want %v", scene, got, tracks)
		}
	}
	if got, want := staticMusicTracks(MusicSchool, true), []string{"schoolm.wav", "schoolw.wav"}; !slices.Equal(got, want) {
		t.Errorf("mage-first school = %v, want %v", got, want)
	}
}

func TestMissionShuffleIsSeededAndVisitsAllBeforeRepeat(t *testing.T) {
	run := func(seed int64) []string {
		source := &recordingMusicSource{}
		device := &recordingMusicDevice{}
		music := NewMusicController(source, device, seed)
		music.SetScene(MusicMission, false)
		for i := 0; i < 12; i++ {
			device.ended = true
			music.Update()
		}
		return source.loads
	}
	first, second := run(1070), run(1070)
	if !slices.Equal(first, second) {
		t.Fatalf("same seed differed: %v vs %v", first, second)
	}
	if len(first) != 13 {
		t.Fatalf("loads = %d, want 13: %v", len(first), first)
	}
	seen := make(map[string]bool)
	for _, name := range first[:12] {
		if seen[name] {
			t.Fatalf("track repeated before all 12: %v", first)
		}
		seen[name] = true
	}
	if len(seen) != 12 || first[12] != first[0] {
		t.Fatalf("cycle = %v; want 12 unique then first", first)
	}
}

func TestOneFileEOFRepeats(t *testing.T) {
	source := &recordingMusicSource{}
	device := &recordingMusicDevice{}
	music := NewMusicController(source, device, 1)
	music.SetScene(MusicMenu, false)
	device.ended = true
	music.Update()
	if want := []string{"menu.wav", "menu.wav"}; !slices.Equal(source.loads, want) {
		t.Fatalf("loads = %v, want %v", source.loads, want)
	}
}

func TestSceneReplacementStopsBeforeStartAndSameSceneIsIdempotent(t *testing.T) {
	source := &recordingMusicSource{}
	device := &recordingMusicDevice{}
	music := NewMusicController(source, device, 2)
	music.SetScene(MusicMenu, false)
	music.SetScene(MusicMenu, false)
	music.SetScene(MusicShop, false)
	want := []string{"start:menu.wav", "stop", "start:shop.wav"}
	if !slices.Equal(device.events, want) {
		t.Fatalf("events = %v, want %v", device.events, want)
	}
}

func TestMissingDeviceTrackAndMalformedTrackAreSilent(t *testing.T) {
	t.Run("nil device does not even load", func(t *testing.T) {
		source := &recordingMusicSource{}
		NewMusicController(source, nil, 1).SetScene(MusicMenu, false)
		if len(source.loads) != 0 {
			t.Fatalf("nil device loaded %v", source.loads)
		}
	})

	t.Run("missing and malformed candidates start nothing", func(t *testing.T) {
		for _, source := range []*recordingMusicSource{
			{missing: map[string]bool{"menu.wav": true}},
			{malformed: map[string]bool{"menu.wav": true}},
			{wrongRate: map[string]bool{"menu.wav": true}},
			{oddFrame: map[string]bool{"menu.wav": true}},
		} {
			device := &recordingMusicDevice{}
			music := NewMusicController(source, device, 1)
			music.SetScene(MusicMenu, false)
			music.Update()
			if len(device.events) != 0 || music.residentTracks() != 0 {
				t.Fatalf("events=%v resident=%d, want silence", device.events, music.residentTracks())
			}
		}
	})
}

func TestControllerRetainsAtMostOneCurrentTrack(t *testing.T) {
	trackType := reflect.TypeOf(audio.Track{})
	controllerType := reflect.TypeOf(MusicController{})
	for i := 0; i < controllerType.NumField(); i++ {
		field := controllerType.Field(i)
		if field.Type == trackType || field.Type == reflect.SliceOf(trackType) {
			t.Fatalf("controller field %s retains decoded track population: %v", field.Name, field.Type)
		}
	}
	source := &recordingMusicSource{}
	device := &recordingMusicDevice{}
	music := NewMusicController(source, device, 3)
	music.SetScene(MusicMission, false)
	for i := 0; i < 30; i++ {
		if got := music.residentTracks(); got > 1 {
			t.Fatalf("resident tracks = %d", got)
		}
		device.ended = true
		music.Update()
		if device.maximum > 1 {
			t.Fatalf("device residency reached %d tracks", device.maximum)
		}
	}
	music.Stop()
	if got := music.residentTracks(); got != 0 {
		t.Fatalf("after Stop resident tracks = %d", got)
	}
	if device.current.Frames() != 0 {
		t.Fatalf("device retained %d frame(s) after Stop", device.current.Frames())
	}
}

type recordingTownMusic struct {
	scene     MusicScene
	mageFirst bool
}

func (*recordingTownMusic) Header() string        { return "town" }
func (*recordingTownMusic) Rows() []TownRow       { return nil }
func (*recordingTownMusic) Footer() []string      { return nil }
func (*recordingTownMusic) Choose(int) TownAction { return TownAction{} }
func (*recordingTownMusic) Back() bool            { return false }
func (t *recordingTownMusic) TownMusic() (MusicScene, bool) {
	return t.scene, t.mageFirst
}

func TestAppRecordsTheStaticSceneLifecycleWithoutOverlayRestarts(t *testing.T) {
	source := &recordingMusicSource{}
	device := &recordingMusicDevice{}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetMusic(source, device, 1070)

	if err := a.OpenChargen(NewChargen(chargenLegalSetup()), nil); err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}

	town := &recordingTownMusic{scene: MusicCampaign}
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("town transition was refused")
	}
	a.syncMusic()
	for _, scene := range []MusicScene{MusicTown, MusicShop, MusicTavern} {
		town.scene = scene
		a.syncMusic()
	}
	town.scene, town.mageFirst = MusicSchool, true
	a.syncMusic()

	loadsBeforeOverlay, eventsBeforeOverlay := len(source.loads), len(device.events)
	a.flow.setScreen(ScreenGameMenu)
	a.syncMusic()
	a.syncMusic() // redraw/update of an overlay is an exact no-op
	if len(source.loads) != loadsBeforeOverlay || len(device.events) != eventsBeforeOverlay {
		t.Fatalf("overlay restarted music: loads %v events %v", source.loads, device.events)
	}

	if len(source.loads) != 8 {
		t.Fatalf("scene loads = %v, want eight transitions", source.loads)
	}
	wantFixed := []string{"menu.wav", "chrgen.wav"}
	if !slices.Equal(source.loads[:2], wantFixed) {
		t.Fatalf("first transitions = %v, want %v", source.loads[:2], wantFixed)
	}
	if got := source.loads[2]; len(got) != len("B00.wav") || got[0] != 'B' {
		t.Fatalf("mission transition = %q, want one B00..B11 candidate", got)
	}
	wantTail := []string{"map.wav", "town.wav", "shop.wav", "inn.wav"}
	if !slices.Equal(source.loads[3:7], wantTail) {
		t.Fatalf("town transition tail = %v, want %v", source.loads[3:7], wantTail)
	}
	if got := source.loads[7]; got != "schoolm.wav" && got != "schoolw.wav" {
		t.Fatalf("school transition = %q, want a school candidate", got)
	}

	a.StopMusic()
	if got := device.events[len(device.events)-1]; got != "stop" {
		t.Fatalf("teardown ended with %q, want stop", got)
	}
}
