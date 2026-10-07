package game

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// loadMusicRecorder records the order of stream stops and starts.
type loadMusicRecorder struct {
	menuSettingsPlayer
	events []string
}

func (p *loadMusicRecorder) Start(audio.Track) { p.events = append(p.events, "start") }
func (p *loadMusicRecorder) Stop()             { p.events = append(p.events, "stop") }

// loseMissionTen saves mission 10 and raises its failure panel through App.
func loseMissionTen(t *testing.T) (*FrontEnd, *ui.App, *loadMusicRecorder, *mapWorld) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	device := &loadMusicRecorder{}
	f.MusicPlayer = device
	app := f.App("load-music")
	t.Cleanup(app.StopAudio)
	app.SetCutscenes(nil)
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeSave(snap, "running")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(time.Unix(1, 0), data); err != nil {
		t.Fatal(err)
	}
	if list, playing := app.MusicPlaybackState(); len(list) != 12 || playing == "" {
		t.Fatalf("mission music = %v playing %q, want the 12-name list and one track", list, playing)
	}
	old := f.live
	f.LiveKill(uint32(old.mission.ids[0]))
	for i := 0; i < 5000; i++ {
		_, kind, up := f.LiveNotice()
		if up && kind == ui.NoticeFailure {
			break
		}
		if up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		} else if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeFailure {
		t.Fatal("the primary's death did not raise the failure panel")
	}
	return f, app, device, old
}

// The failure panel requests nothing; Exit then stops and requests the menu list.
func TestReleaseLossThenExitStopsThenRequestsTheMenuList(t *testing.T) {
	_, app, device, _ := loseMissionTen(t)
	logAtPanel, eventsAtPanel := len(app.MusicRequestLog()), len(device.events)
	for i := 0; i < 3; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if len(app.MusicRequestLog()) != logAtPanel || len(device.events) != eventsAtPanel {
		t.Fatalf("the displayed failure panel touched the music: %v", app.MusicRequestLog()[logAtPanel:])
	}
	if err := app.HeadlessActivate("exit to main menu"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("Exit left screen %v, want the menu", app.Screen())
	}
	log := app.MusicRequestLog()[logAtPanel:]
	t.Logf("request log after Exit: %v; device %v", log, device.events[eventsAtPanel:])
	if want := []string{"stop", "request:menu.wav"}; !slices.Equal(log, want) {
		t.Fatalf("request log after Exit = %v, want %v", log, want)
	}
	if want := []string{"stop", "start"}; !slices.Equal(device.events[eventsAtPanel:], want) {
		t.Fatalf("device after Exit = %v, want %v", device.events[eventsAtPanel:], want)
	}
	if list, playing := app.MusicPlaybackState(); !slices.Equal(list, []string{"menu.wav"}) || playing != "menu.wav" {
		t.Fatalf("music after Exit = %v playing %q, want the menu list", list, playing)
	}
}

// Load Game on the panel requests nothing until a mission save completes.
func TestReleaseLoadAfterLossRequestsTheMissionListAgain(t *testing.T) {
	f, app, device, old := loseMissionTen(t)
	missionList, _ := app.MusicPlaybackState()
	logAtPanel, eventsAtPanel := len(app.MusicRequestLog()), len(device.events)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if len(app.MusicRequestLog()) != logAtPanel || len(device.events) != eventsAtPanel {
		t.Fatalf("opening the load dialog touched the music: %v", app.MusicRequestLog()[logAtPanel:])
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.live == old {
		t.Fatalf("the load did not replace the lost session: screen %v", app.Screen())
	}
	log := app.MusicRequestLog()[logAtPanel:]
	t.Logf("request log after the load: %v; device %v", log, device.events[eventsAtPanel:])
	if want := []string{"stop", "request:B00.wav..B11.wav"}; !slices.Equal(log, want) {
		t.Fatalf("request log after the load = %v, want %v", log, want)
	}
	if want := []string{"stop", "start"}; !slices.Equal(device.events[eventsAtPanel:], want) {
		t.Fatalf("device after the load = %v, want %v", device.events[eventsAtPanel:], want)
	}
	list, playing := app.MusicPlaybackState()
	if !slices.Equal(list, missionList) || playing == "" {
		t.Fatalf("music after the load = %v playing %q, want the mission list and one track", list, playing)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(device.events) != eventsAtPanel+2 {
		t.Fatalf("a later frame restarted the list: %v", device.events[eventsAtPanel:])
	}
}

// A town LOAD requests the Town list unless it is already playing.
func TestReleaseTownSaveLoadRequestsTownListOnlyWhenItIsNotPlaying(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	device := &loadMusicRecorder{}
	f.MusicPlayer = device
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err = store.Write(time.Unix(100, 0), payload); err != nil {
		t.Fatal(err)
	}
	app := f.App("town-load-music")
	t.Cleanup(app.StopAudio)
	app.SetCutscenes(nil)
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	logAtMenu, eventsAtMenu := len(app.MusicRequestLog()), len(device.events)
	load := func(fromMenu bool) {
		t.Helper()
		if fromMenu {
			if err := app.HeadlessKey("load"); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("load"); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenTown {
			t.Fatalf("LOAD opened screen %s, want the town", app.Screen())
		}
	}
	load(true)
	log := app.MusicRequestLog()[logAtMenu:]
	t.Logf("request log after the first load: %v; device %v", log, device.events[eventsAtMenu:])
	if want := []string{"stop", "request:town.wav"}; !slices.Equal(log, want) {
		t.Fatalf("request log after the first load = %v, want %v", log, want)
	}
	if _, playing := app.MusicPlaybackState(); playing != "town.wav" {
		t.Fatalf("playing %q, want town.wav", playing)
	}
	logInTown, eventsInTown := len(app.MusicRequestLog()), len(device.events)
	load(false)
	log = app.MusicRequestLog()[logInTown:]
	t.Logf("request log after the second load: %v; device %v", log, device.events[eventsInTown:])
	if want := []string{"skip:town.wav"}; !slices.Equal(log, want) {
		t.Fatalf("request log after the second load = %v, want %v", log, want)
	}
	if len(device.events) != eventsInTown {
		t.Fatalf("the running Town list was touched: %v", device.events[eventsInTown:])
	}
}
