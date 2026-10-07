package ui

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// loadMusicApp opens the menu over recording music seams and installs a save
// list with one row whose load seam answers with the given result.
func loadMusicApp(t *testing.T, load LoadGame) (*App, *recordingMusicSource, *recordingMusicDevice) {
	t.Helper()
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetMusic(source, device, 35)
	a.SetSaveSeams(
		func(bool) (string, error) { return "saved.sav", nil },
		func() []SaveEntry { return []SaveEntry{{Name: "game.sav", Label: "game"}} },
		load,
	)
	return a, source, device
}

func TestRequestSceneReplacesTheListForTheSameScene(t *testing.T) {
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	music := NewMusicController(source, device, 5)
	music.SetScene(MusicTown, false)
	music.RequestScene(MusicTown, false)
	music.SetScene(MusicTown, false)
	want := []string{"start:town.wav", "stop", "start:town.wav"}
	if !slices.Equal(device.events, want) {
		t.Fatalf("events = %v, want %v", device.events, want)
	}
}

func TestCompletedLoadRequestsTheMissionListAgain(t *testing.T) {
	a, source, device := loadMusicApp(t, func(string) (MapOpener, bool, error) {
		return okOpener(t), false, nil
	})
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	before, loadsBefore := len(device.events), len(source.loads)
	if last := device.events[before-1]; !strings.HasPrefix(last, "start:B") {
		t.Fatalf("mission list is not playing: %v", device.events)
	}
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMap {
		t.Fatalf("screen = %v, want ScreenMap", a.Screen())
	}
	got := device.events[before:]
	if len(got) != 2 || got[0] != "stop" || !strings.HasPrefix(got[1], "start:B") {
		t.Fatalf("events after the load = %v, want a stop then a mission start", got)
	}
	if len(source.loads) != loadsBefore+1 {
		t.Fatalf("track reads after the load = %d, want one", len(source.loads)-loadsBefore)
	}
	if want := staticMusicTracks(MusicMission, false); !slices.Equal(a.music.Candidates(), want) {
		t.Fatalf("candidates = %v, want the mission list", a.music.Candidates())
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(device.events) != before+2 {
		t.Fatalf("a later frame restarted the list: %v", device.events[before:])
	}
}

func TestCompletedTownLoadKeepsTheRunningTownList(t *testing.T) {
	a, _, device := loadMusicApp(t, func(string) (MapOpener, bool, error) { return nil, true, nil })
	a.SetTown(&recordingTownMusic{scene: MusicTown})
	if !a.flow.showTown("") {
		t.Fatal("town transition was refused")
	}
	a.syncMusic()
	before, logBefore := len(device.events), len(a.MusicRequestLog())
	if last := device.events[before-1]; last != "start:town.wav" {
		t.Fatalf("town list is not playing: %v", device.events)
	}
	a.flow.openLoad(ScreenTown)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenTown {
		t.Fatalf("screen = %v, want ScreenTown", a.Screen())
	}
	if len(device.events) != before {
		t.Fatalf("events after the load = %v, want none: the town owner skips an equal list", device.events[before:])
	}
	if got, want := a.MusicRequestLog()[logBefore:], []string{"skip:town.wav"}; !slices.Equal(got, want) {
		t.Fatalf("request log after the load = %v, want %v", got, want)
	}
}

func TestCompletedTownLoadFromAMissionRequestsTheTownList(t *testing.T) {
	a, _, device := loadMusicApp(t, func(string) (MapOpener, bool, error) { return nil, true, nil })
	a.SetTown(&recordingTownMusic{scene: MusicTown})
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	before, logBefore := len(device.events), len(a.MusicRequestLog())
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenTown {
		t.Fatalf("screen = %v, want ScreenTown", a.Screen())
	}
	if want := []string{"stop", "start:town.wav"}; !slices.Equal(device.events[before:], want) {
		t.Fatalf("events after the load = %v, want %v", device.events[before:], want)
	}
	if got, want := a.MusicRequestLog()[logBefore:], []string{"stop", "request:town.wav"}; !slices.Equal(got, want) {
		t.Fatalf("request log after the load = %v, want %v", got, want)
	}
}

func TestMissionLoadLogsAStopAndTheTwelveEntryRequest(t *testing.T) {
	a, _, _ := loadMusicApp(t, func(string) (MapOpener, bool, error) { return okOpener(t), false, nil })
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	logBefore := len(a.MusicRequestLog())
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if got, want := a.MusicRequestLog()[logBefore:], []string{"stop", "request:B00.wav..B11.wav"}; !slices.Equal(got, want) {
		t.Fatalf("request log after the load = %v, want %v", got, want)
	}
}

func TestExitFromAMissionStopsThenRequestsTheMenuList(t *testing.T) {
	a, _, device := loadMusicApp(t, func(string) (MapOpener, bool, error) { return nil, false, errNotASave })
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	before, logBefore := len(device.events), len(a.MusicRequestLog())
	a.flow.leaveMap()
	a.flow.toMenu()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if got, want := a.MusicRequestLog()[logBefore:], []string{"stop", "request:menu.wav"}; !slices.Equal(got, want) {
		t.Fatalf("request log after Exit = %v, want %v", got, want)
	}
	if got := device.events[before:]; len(got) != 2 || got[0] != "stop" || got[1] != "start:menu.wav" {
		t.Fatalf("events after Exit = %v, want a stop then the menu start", got)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(device.events) != before+2 {
		t.Fatalf("a later frame touched the music: %v", device.events[before:])
	}
}

func TestMissionLossPanelMakesNoMusicRequest(t *testing.T) {
	a, _, device := loadMusicApp(t, func(string) (MapOpener, bool, error) { return okOpener(t), false, nil })
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	before, logBefore := len(device.events), len(a.MusicRequestLog())
	a.flow.viewer.SetNotice("lost", NoticeFailure)
	for range 3 {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !a.flow.viewer.NoticeOpen() {
		t.Fatal("the failure panel is not open")
	}
	if len(device.events) != before || len(a.MusicRequestLog()) != logBefore {
		t.Fatalf("the failure panel touched the music: %v %v", device.events[before:], a.MusicRequestLog()[logBefore:])
	}
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(device.events) != before || len(a.MusicRequestLog()) != logBefore {
		t.Fatalf("the load dialog touched the music: %v", device.events[before:])
	}
}

func TestRefusedAndCancelledLoadsLeaveTheMusicAlone(t *testing.T) {
	a, source, device := loadMusicApp(t, func(string) (MapOpener, bool, error) {
		return nil, false, errNotASave
	})
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	before, loadsBefore := len(device.events), len(source.loads)
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenLoad || a.HeadlessMessage() == "" {
		t.Fatalf("refused load left screen %v message %q", a.Screen(), a.HeadlessMessage())
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMap {
		t.Fatalf("cancel left screen %v, want ScreenMap", a.Screen())
	}
	if len(device.events) != before || len(source.loads) != loadsBefore {
		t.Fatalf("a refused or cancelled load touched the music: %v", device.events[before:])
	}
}

func TestWindowCloseOverTheLoadDialogEndsWithoutAMusicRequest(t *testing.T) {
	a, _, device := loadMusicApp(t, func(string) (MapOpener, bool, error) { return okOpener(t), false, nil })
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	a.flow.openLoad(ScreenMap)
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	before, logBefore := len(device.events), len(a.MusicRequestLog())
	if !a.step(appInput{Close: true}, time.Now()) {
		t.Fatal("a window close over the load dialog did not end the run")
	}
	if len(device.events) != before || len(a.MusicRequestLog()) != logBefore {
		t.Fatalf("the window close requested music: %v %v", device.events[before:], a.MusicRequestLog()[logBefore:])
	}
}
