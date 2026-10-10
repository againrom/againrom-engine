package game

import (
	"image"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// secondTestMusic is the second game's witness music device: it records
// starts, stops, pauses and resumes and plays nothing.
type secondTestMusic struct {
	events  []string
	started bool
	held    bool
	ended   bool
}

func (d *secondTestMusic) Start(audio.Track) {
	d.events = append(d.events, "start")
	d.started, d.held, d.ended = true, false, false
}

func (d *secondTestMusic) Stop() {
	d.events = append(d.events, "stop")
	d.started, d.held = false, false
}

func (d *secondTestMusic) Ended() bool { return d.started && !d.held && d.ended }

func (d *secondTestMusic) SetSettings(audio.Settings) {}

func (d *secondTestMusic) Pause() bool {
	if !d.started {
		return false
	}
	d.events = append(d.events, "pause")
	d.held = true
	return true
}

func (d *secondTestMusic) Resume() bool {
	if !d.held {
		return false
	}
	d.events = append(d.events, "resume")
	d.held = false
	return true
}

// secondMusicTail asserts the request log and the device events since the
// marks and moves the marks.
type secondMusicTail struct {
	app         *ui.App
	device      *secondTestMusic
	log, events int
}

func (s *secondMusicTail) expect(t *testing.T, what string, log, events []string) {
	t.Helper()
	gotLog, gotEvents := s.app.MusicRequestLog()[s.log:], s.device.events[s.events:]
	t.Logf("%s: log %v device %v", what, gotLog, gotEvents)
	if !slices.Equal(gotLog, log) || !slices.Equal(gotEvents, events) {
		t.Fatalf("%s: log %v device %v, want %v and %v", what, gotLog, gotEvents, log, events)
	}
	s.log, s.events = len(s.app.MusicRequestLog()), len(s.device.events)
}

func secondMusicDevice(t *testing.T, f *FrontEnd) *secondTestMusic {
	t.Helper()
	d, ok := f.MusicPlayer.(*secondTestMusic)
	if !ok {
		t.Fatal("the second-game front has no witness music device")
	}
	return d
}

// The second game's music archive holds the description's 21 keys and each
// decodes (R2-ASSET-083).
func TestReleaseSecondGameMusicPopulation(t *testing.T) {
	archives := secondGameRoot(t)
	d := GameMusic(archives.Base.Profile)
	bank := OpenMusicFor(os.Getenv("AGAINROM_ASSETS"), d)
	got := bank.Manifest()
	for i := range got {
		got[i] = strings.ToLower(got[i])
	}
	want := append([]string(nil), d.Tracks...)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("music archive manifest = %v, want %v", got, want)
	}
	for _, list := range [][]string{d.Tracks, d.Scenes["mission"].Tracks} {
		for _, name := range list {
			track, ok := bank.Track(name)
			if !ok || track.Rate != audio.DeviceRate || track.Frames() == 0 {
				t.Errorf("%s did not resolve and decode", name)
			}
		}
	}
}

// Each screen requests its key (R2-ENGINE-335): menu, credits, the
// generator, the first town, the destination map and a mission. Menu and
// generator keep a key already held; credits and the map set theirs each
// time; the tavern makes no request.
func TestReleaseSecondGameMusicScreens(t *testing.T) {
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	device := secondMusicDevice(t, f)
	app := f.App("second music")
	app.Layout(640, 480)
	tail := &secondMusicTail{app: app, device: device}
	tail.expect(t, "menu", []string{"request:menu.wav"}, []string{"start"})
	for _, overlay := range []string{"load game"} {
		if err := app.HeadlessActivate(overlay); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMenu {
			t.Fatal(overlay, err, app.Screen())
		}
		tail.expect(t, overlay+" and back", nil, nil)
	}
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen {
		t.Fatal("NEW GAME", err, app.Screen())
	}
	tail.expect(t, "generator", []string{"stop", "request:chrgen.wav"}, []string{"stop", "start"})
	l := f.generator()
	pre := chargenTraceMaskPoints(t, f, l.PreCreate.Mask.Key, image.Point{}, []byte{180})
	secondGeneratorClick(t, app, pre[180])
	back := l.Detail.Commands[2].Rect.Rectangle()
	secondGeneratorClick(t, app, back.Min.Add(back.Size().Div(2)))
	secondGeneratorClick(t, app, pre[180])
	if secondGeneratorStage(app) != ui.ChargenStageDetailed {
		t.Fatal("OK did not open the detail page")
	}
	tail.expect(t, "generator pages", nil, nil)
	accept := l.Detail.Commands[0].Rect.Rectangle()
	secondGeneratorClick(t, app, accept.Min.Add(accept.Size().Div(2)))
	if app.Screen() != ui.ScreenTown || f.TownScreen().Header() != "ROM2 campaign: town 1" {
		t.Fatal("Accept", app.Screen())
	}
	tail.expect(t, "town 1", []string{"stop", "request:b14.wav"}, []string{"stop", "start"})

	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	for pages := 0; pages < 64; pages++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	tail.expect(t, "tavern and talk", nil, nil)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	tail.expect(t, "destinations", []string{"stop", "request:map.wav"}, []string{"stop", "start"})
	for _, target := range []string{"mission 10", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatal("mission 10 did not open", app.Screen(), app.HeadlessMessage())
	}
	tail.expect(t, "mission", []string{"stop", "request:B00.wav..B16.wav"}, []string{"stop", "start"})
	list, playing := app.MusicPlaybackState()
	if pick := app.MusicAreaPick(); pick < 0 || len(list) != 17 || playing != list[pick] {
		t.Fatalf("mission entry pick %d playing %q of %v, want a theme of the area or head under the hero", pick, playing, list)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	tail.expect(t, "game menu over the mission", nil, nil)
	for _, row := range []string{"end", "exit-main"} {
		if err := app.HeadlessGameMenuAction(row); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("abort left screen", app.Screen())
	}
	tail.expect(t, "menu after the mission", []string{"stop", "request:menu.wav"}, []string{"stop", "start"})
	if err := app.HeadlessActivate("credits"); err != nil || app.Screen() != ui.ScreenCredits {
		t.Fatal("credits", err, app.Screen())
	}
	tail.expect(t, "credits", []string{"stop", "request:credit.wav"}, []string{"stop", "start"})
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMenu {
		t.Fatal("credits escape", err, app.Screen())
	}
	tail.expect(t, "menu after credits", []string{"stop", "request:menu.wav"}, []string{"stop", "start"})
}

// secondExpectedPick is the set of themes the area select can pick at
// (x, y), by the record walk of R2-ENGINE-336: the last record taken.
func secondExpectedPick(areas []ui.MusicArea, x, y int32) []int {
	best, taken := 1e20, -1
	for i, a := range areas {
		if a.X == 0 && a.Y == 0 {
			if best > 1e15 {
				best, taken = 1e10, i
			}
			continue
		}
		if a.Themes == [4]int32{-1, -1, -1, -1} {
			continue
		}
		d := math.Hypot(float64(x)-float64(a.X)*256, float64(y)-float64(a.Y)*256)
		if d < float64(a.Radius)*256 && d < best {
			best, taken = d, i
		}
	}
	if taken < 0 {
		return nil
	}
	var out []int
	for _, theme := range areas[taken].Themes {
		if theme >= 0 && !slices.Contains(out, int(theme)) {
			out = append(out, int(theme))
		}
	}
	return out
}

// secondMusicStepTo steps the mission until its tick passes the next
// multiple of 16.
func secondMusicStepTo(t *testing.T, app *ui.App, m *mapWorld) {
	t.Helper()
	target := (m.world.Tick()/16 + 1) * 16
	for i := 0; m.world.Tick() < target; i++ {
		if i > 64 {
			t.Fatal("the mission tick did not advance")
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}

// On mission 10 the hero placed inside a music area makes that area's theme
// the next track after the current one ends; outside every area the map's
// default record decides (R2-ASSET-084, R2-ENGINE-336). A LOAD of a mission
// save sets the list again.
func TestReleaseSecondGameMusicAreas(t *testing.T) {
	f := secondGameFront(t)
	device := secondMusicDevice(t, f)
	app := f.App("second music areas")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	m := f.live
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, m)
	areas := missionMusicAreas(m.mission.state.Map)
	if len(areas) < 3 || areas[0].X != 0 || areas[0].Y != 0 {
		t.Fatalf("mission 10 areas %+v, want the admitted head and its areas", areas)
	}
	t.Logf("mission 10: %d records, head themes %v", len(areas), areas[0].Themes)
	id, ok := m.primaryPartyID()
	if !ok {
		t.Fatal("no hero")
	}
	list, _ := app.MusicPlaybackState()
	placed := 0
	for i, a := range areas[1:] {
		if a.Themes == [4]int32{-1, -1, -1, -1} || placed == 2 {
			continue
		}
		if err := m.world.HeadlessPlace(id, a.X, a.Y); err != nil {
			continue
		}
		e, _ := m.world.Entity(id)
		hx, hy := e.X*256+128, e.Y*256+128
		want := secondExpectedPick(areas, hx, hy)
		if !slices.Equal(want, secondExpectedPick([]ui.MusicArea{a}, hx, hy)) {
			continue
		}
		_, before := app.MusicPlaybackState()
		secondMusicStepTo(t, app, m)
		pick := app.MusicAreaPick()
		if !slices.Contains(want, pick) {
			t.Fatalf("area %d at (%d,%d) themes %v: pick %d", i+1, a.X, a.Y, a.Themes, pick)
		}
		if _, now := app.MusicPlaybackState(); now != before {
			t.Fatalf("area %d: the pick changed the playing track %q to %q before its end", i+1, before, now)
		}
		device.ended = true
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if _, now := app.MusicPlaybackState(); now != list[pick] {
			t.Fatalf("area %d: track end opened %q, want %q", i+1, now, list[pick])
		}
		t.Logf("area %d at (%d,%d) radius %d themes %v: pick %d plays %s", i+1, a.X, a.Y, a.Radius, a.Themes, pick, list[pick])
		placed++
	}
	if placed != 2 {
		t.Fatalf("placed the hero in %d areas, want 2", placed)
	}

	// Outside every area the default record's themes decide.
	outside := false
	for y := int32(2); y < 200 && !outside; y += 7 {
		for x := int32(2); x < 200 && !outside; x += 7 {
			want := secondExpectedPick(areas, x*256+128, y*256+128)
			if !slices.Equal(want, secondExpectedPick(areas[:1], x*256+128, y*256+128)) {
				continue
			}
			if err := m.world.HeadlessPlace(id, x, y); err != nil {
				continue
			}
			e, _ := m.world.Entity(id)
			if !slices.Equal(secondExpectedPick(areas, e.X*256+128, e.Y*256+128), want) {
				continue
			}
			secondMusicStepTo(t, app, m)
			if pick := app.MusicAreaPick(); !slices.Contains(want, pick) {
				t.Fatalf("outside every area at (%d,%d): pick %d, want one of the head's %v", e.X, e.Y, pick, want)
			}
			t.Logf("outside every area at (%d,%d): pick %d of %v", e.X, e.Y, app.MusicAreaPick(), want)
			outside = true
		}
	}
	if !outside {
		t.Fatal("found no cell outside every area")
	}

	out := t.TempDir()
	secondMissionNamedSave(t, f, app, out, "music areas")
	cold, coldApp := secondMissionCold(t, out, "music areas.sav")
	log := coldApp.MusicRequestLog()
	t.Logf("cold LOAD log %v", log)
	if want := []string{"request:menu.wav", "stop", "request:B00.wav..B16.wav"}; !slices.Equal(log, want) {
		t.Fatalf("cold LOAD log %v, want %v", log, want)
	}
	e, _ := cold.live.world.Entity(cold.live.mission.ids[0])
	coldAreas := missionMusicAreas(cold.live.mission.state.Map)
	if pick := coldApp.MusicAreaPick(); !slices.Contains(secondExpectedPick(coldAreas, e.X*256+128, e.Y*256+128), pick) {
		t.Fatalf("LOAD pick %d at (%d,%d)", pick, e.X, e.Y)
	}
}

// A movie pauses the list and the next same-key screen resumes it; mission
// end pauses, and the destination map sets its own list (R2-ENGINE-338).
// Without the music archive every request is silent (R2-ENGINE-339).
func TestReleaseSecondGameMusicStops(t *testing.T) {
	f, app, m := firstSuccessApp(t)
	device := secondMusicDevice(t, f)
	tail := &secondMusicTail{app: app, device: device, log: len(app.MusicRequestLog()), events: len(device.events)}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenCutscene {
		t.Fatal("no completion movie", app.Screen())
	}
	movie := app.CutsceneName()
	// The destination's request precedes the movie, which pauses it at once.
	tail.expect(t, "completion movie", []string{"stop", "request:map.wav", "pause"}, []string{"stop", "start", "pause"})
	deadline := time.Now().Add(5 * time.Second)
	for app.CutsceneFrameNumber() == 0 && time.Now().Before(deadline) {
		if err := app.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessCutsceneStep("key"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || m.mission.open {
		t.Fatal("the movie did not end at the destinations", app.Screen())
	}
	tail.expect(t, "destinations after the mission", []string{"stop", "request:map.wav"}, []string{"stop", "start"})

	for presses := 0; app.Screen() == ui.ScreenTown && presses < 4; presses++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []string{"abort", "confirm-abort"} {
		if err := app.HeadlessGameMenuAction(row); err != nil {
			t.Fatal(err)
		}
	}
	tail.expect(t, "menu", []string{"stop", "request:menu.wav"}, []string{"stop", "start"})
	if !app.PlayCutscene(movie) || app.Screen() != ui.ScreenCutscene {
		t.Fatal("menu movie", movie)
	}
	tail.expect(t, "menu movie", []string{"pause"}, []string{"pause"})
	app.StopCutscene()
	tail.expect(t, "menu after the movie", []string{"resume:menu.wav"}, []string{"resume"})

	silent := secondGameFront(t)
	silent.MusicBank = nil
	quiet := secondMusicDevice(t, silent)
	a := silent.App("no music")
	a.Layout(640, 480)
	if err := a.HeadlessActivate("credits"); err != nil {
		t.Fatal(err)
	}
	if _, playing := a.MusicPlaybackState(); slices.Contains(quiet.events, "start") || playing != "" {
		t.Fatalf("without the archive: device %v playing %q", quiet.events, playing)
	}
	t.Logf("without the archive: log %v, device silent", a.MusicRequestLog())
}
