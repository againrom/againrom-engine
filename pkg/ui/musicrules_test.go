package ui

import (
	"os"
	"slices"
	"testing"

	"againrom/pkg/random"
)

// secondMusic is the second game's music description, read from the file
// the game package embeds.
var secondMusic = func() *MusicDescription {
	data, err := os.ReadFile("../game/musics/rom2.json")
	if err != nil {
		panic(err)
	}
	d, err := DecodeMusic(data)
	if err != nil {
		panic(err)
	}
	return d
}()

// pausingMusicDevice holds a paused stream the way the production device does.
type pausingMusicDevice struct {
	recordingMusicDevice
	held bool
}

func (d *pausingMusicDevice) Pause() bool {
	if d.current.Frames() == 0 {
		return false
	}
	d.events = append(d.events, "pause")
	d.held = true
	return true
}

func (d *pausingMusicDevice) Resume() bool {
	if !d.held {
		return false
	}
	d.events = append(d.events, "resume")
	d.held = false
	return true
}

func (d *pausingMusicDevice) Stop() {
	d.held = false
	d.recordingMusicDevice.Stop()
}

func TestSecondMusicSceneTable(t *testing.T) {
	want := map[MusicScene][]string{
		MusicSilent:   nil,
		MusicMenu:     {"menu.wav"},
		MusicCredits:  {"credit.wav"},
		MusicChargen:  {"chrgen.wav"},
		MusicCampaign: {"map.wav"},
		MusicTown:     nil,
		MusicShop:     nil,
		MusicTavern:   nil,
		MusicSchool:   nil,
	}
	for scene, tracks := range want {
		if got := secondMusic.sceneTracks(scene, false); !slices.Equal(got, tracks) {
			t.Errorf("scene %d = %v, want %v", scene, got, tracks)
		}
	}
	mission := secondMusic.sceneTracks(MusicMission, false)
	if len(mission) != 17 || mission[0] != "B00.wav" || mission[16] != "B16.wav" {
		t.Errorf("mission list = %v, want B00..B16", mission)
	}
	for id, track := range map[int]string{1: "b14.wav", 2: "b16.wav", 3: "b15.wav"} {
		if got, ok := secondMusic.TownTrack(id); !ok || got != track {
			t.Errorf("town %d = %q %v, want %s", id, got, ok, track)
		}
	}
	if _, ok := secondMusic.TownTrack(4); ok {
		t.Error("town 4 names a track")
	}
	for scene, keep := range map[MusicScene]bool{MusicMenu: true, MusicChargen: true, MusicTown: true, MusicCampaign: false, MusicCredits: false, MusicMission: false} {
		if got := secondMusic.sceneSpec(scene).Keep; got != keep {
			t.Errorf("scene %d keep = %v, want %v", scene, got, keep)
		}
	}
}

// An ordered list keeps its order and starts at one draw modulo its length;
// Random Order changes nothing.
func TestOrderedListStartsAtOneDrawAndIgnoresRandomOrder(t *testing.T) {
	for _, randomOrder := range []bool{false, true} {
		source, device := &recordingMusicSource{}, &recordingMusicDevice{}
		m := NewMusicController(secondMusic, source, device, random.NewStream(77))
		m.preferences.RandomOrder = randomOrder
		m.SetScene(MusicMission, false)
		probe := random.NewStream(77)
		start := probe.Raw() % 17
		tracks := secondMusic.sceneTracks(MusicMission, false)
		if !slices.Equal(m.order, tracks) || m.position != start || m.Playing() != tracks[start] {
			t.Fatalf("random order %v: order %v at %d playing %q, want the list at %d", randomOrder, m.order, m.position, m.Playing(), start)
		}
		device.ended = true
		m.Update()
		if m.Playing() != tracks[(start+1)%17] {
			t.Fatalf("random order %v: track end opened %q, want the next entry", randomOrder, m.Playing())
		}
		m.SetPreferences(MusicPreferences{Enabled: true, RandomOrder: !randomOrder})
		if !slices.Equal(m.order, tracks) {
			t.Fatalf("a Random Order change reordered the list: %v", m.order)
		}
	}
}

// A silent request pauses; the same key resumes the held entry; a key whose
// scene does not keep sets its list again.
func TestPauseResumesKeptKeysAndRestartsTheOthers(t *testing.T) {
	source, device := &recordingMusicSource{}, &pausingMusicDevice{}
	m := NewMusicController(secondMusic, source, device, random.NewStream(3))
	m.SetScene(MusicMenu, false)
	m.SetScene(MusicSilent, false)
	m.SetScene(MusicMenu, false)
	m.SetScene(MusicChargen, false)
	m.SetScene(MusicChargen, false)
	m.SetScene(MusicCampaign, false)
	m.SetScene(MusicSilent, false)
	m.SetScene(MusicCampaign, false)
	m.SetScene(MusicSilent, false)
	m.SetScene(MusicCredits, false)
	want := []string{
		"start:menu.wav", "pause", "resume",
		"stop", "start:chrgen.wav",
		"stop", "start:map.wav", "pause", "stop", "start:map.wav",
		"pause", "stop", "start:credit.wav",
	}
	if !slices.Equal(device.events, want) {
		t.Fatalf("device = %v, want %v", device.events, want)
	}
	log := m.RequestLog()
	wantLog := []string{
		"request:menu.wav", "pause", "resume:menu.wav",
		"stop", "request:chrgen.wav",
		"stop", "request:map.wav", "pause", "stop", "request:map.wav",
		"pause", "stop", "request:credit.wav",
	}
	if !slices.Equal(log, wantLog) {
		t.Fatalf("log = %v, want %v", log, wantLog)
	}
}

// A device that holds no paused stream is stopped, and a kept key starts its
// entry again.
func TestPauseWithoutAHoldingDeviceStartsTheEntryAgain(t *testing.T) {
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	m := NewMusicController(secondMusic, source, device, random.NewStream(3))
	m.SetScene(MusicMenu, false)
	m.SetScene(MusicSilent, false)
	m.SetScene(MusicMenu, false)
	if want := []string{"start:menu.wav", "stop", "start:menu.wav"}; !slices.Equal(device.events, want) {
		t.Fatalf("device = %v, want %v", device.events, want)
	}
}

// A town track request keeps the town key across a repeated request and the
// tavern, which makes none.
func TestTownTrackRequestKeepsItsKey(t *testing.T) {
	source, device := &recordingMusicSource{}, &pausingMusicDevice{}
	m := NewMusicController(secondMusic, source, device, random.NewStream(3))
	m.requestFrom(musicRequest{scene: MusicTown, track: "b14.wav"}, false)
	m.requestFrom(musicRequest{scene: MusicTown, track: "b14.wav"}, true)
	m.requestFrom(musicRequest{scene: MusicTown, track: "b16.wav"}, false)
	if want := []string{"start:b14.wav", "stop", "start:b16.wav"}; !slices.Equal(device.events, want) {
		t.Fatalf("device = %v, want %v", device.events, want)
	}
	if want := []string{"request:b14.wav", "skip:b14.wav", "stop", "request:b16.wav"}; !slices.Equal(m.RequestLog(), want) {
		t.Fatalf("log = %v, want %v", m.RequestLog(), want)
	}
}

// heroAt is a hero standing at tile (x, y) at the given tick.
func heroAt(x, y *int32, tick *uint64) MusicHero {
	return func() (int32, int32, uint64, bool) { return *x*256 + 128, *y*256 + 128, *tick, true }
}

// The mission list: with a hero the area select replaces the drawn start;
// every period ticks the area holding the hero draws its pick; a track end
// opens the pick, else the next entry.
func TestMissionAreasPickTheThemeTheNextTrackEndOpens(t *testing.T) {
	areas := []MusicArea{
		{X: 0, Y: 0, Radius: 0, Themes: [4]int32{2, -1, -1, -1}},
		{X: 40, Y: 40, Radius: 5, Themes: [4]int32{9, 9, 9, 9}},
		{X: 80, Y: 80, Radius: 5, Themes: [4]int32{-1, -1, -1, -1}},
	}
	x, y, tick := int32(10), int32(10), uint64(5)
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	m := NewMusicController(secondMusic, source, device, random.NewStream(9))
	m.areas = musicAreaSource{areas: areas, hero: heroAt(&x, &y, &tick)}
	m.SetScene(MusicMission, false)
	if m.Playing() != "B02.wav" || m.AreaPick() != 2 {
		t.Fatalf("entry played %q pick %d, want the head's theme B02", m.Playing(), m.AreaPick())
	}
	x, y = 41, 39
	tick = 15
	m.observeAreas()
	if m.AreaPick() != 2 {
		t.Fatalf("a select ran before the tick reached the period: pick %d", m.AreaPick())
	}
	tick = 16
	m.observeAreas()
	if m.AreaPick() != 9 {
		t.Fatalf("pick %d inside the area, want 9", m.AreaPick())
	}
	if m.Playing() != "B02.wav" {
		t.Fatalf("the pick changed the playing track to %q before its end", m.Playing())
	}
	device.ended = true
	m.Update()
	if m.Playing() != "B09.wav" {
		t.Fatalf("track end opened %q, want B09", m.Playing())
	}
	x, y, tick = 80, 80, 32
	m.observeAreas()
	if m.AreaPick() != 2 {
		t.Fatalf("an area with no theme took the hero: pick %d, want the head's 2", m.AreaPick())
	}
}

// Without an admitted head the list plays in order from its drawn start until
// the hero first enters an area.
func TestMissionWithoutAHeadPlaysTheListInOrder(t *testing.T) {
	areas := []MusicArea{{X: 40, Y: 40, Radius: 5, Themes: [4]int32{-1, 7, -1, -1}}}
	x, y, tick := int32(10), int32(10), uint64(0)
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	m := NewMusicController(secondMusic, source, device, random.NewStream(4))
	m.areas = musicAreaSource{areas: areas, hero: heroAt(&x, &y, &tick)}
	m.SetScene(MusicMission, false)
	start := random.NewStream(4).Raw() % 17
	if m.position != start || m.AreaPick() != -1 {
		t.Fatalf("start %d pick %d, want the drawn %d and no pick", m.position, m.AreaPick(), start)
	}
	tick = 16
	m.observeAreas()
	device.ended = true
	m.Update()
	if m.position != (start+1)%17 {
		t.Fatalf("track end opened %d, want the next entry %d", m.position, (start+1)%17)
	}
	x, y, tick = 40, 41, 32
	m.observeAreas()
	device.ended = true
	m.Update()
	if m.Playing() != "B07.wav" {
		t.Fatalf("track end opened %q, want the area's only theme B07", m.Playing())
	}
	x, y, tick = 10, 10, 48
	m.observeAreas()
	device.ended = true
	m.Update()
	if m.Playing() != "B07.wav" {
		t.Fatalf("leaving the area dropped the last pick: %q", m.Playing())
	}
}

// The first game's description reads no area.
func TestFirstGameMissionReadsNoArea(t *testing.T) {
	areas := []MusicArea{{X: 0, Y: 0, Themes: [4]int32{3, 3, 3, 3}}}
	x, y, tick := int32(1), int32(1), uint64(0)
	m := NewMusicController(firstMusic, &recordingMusicSource{}, &recordingMusicDevice{}, random.NewStream(4))
	m.areas = musicAreaSource{areas: areas, hero: heroAt(&x, &y, &tick)}
	m.SetScene(MusicMission, false)
	tick = 32
	m.observeAreas()
	if m.AreaPick() != -1 {
		t.Fatalf("first-game pick %d", m.AreaPick())
	}
}
