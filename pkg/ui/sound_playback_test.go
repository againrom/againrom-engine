package ui

import (
	"errors"
	"image"
	"testing"

	"againrom/pkg/audio"
)

func TestSound1189SelectionFocusPlayStopAndFailedWrite(t *testing.T) {
	a := NewApp("sound playback", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen, a.flow.menuFont = fiViewer(t), ScreenMap, gameMenuTestFont()
	a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, func(bool, int) error { return nil })
	p, fail := MusicPreferences{Enabled: true}, false
	a.SetSoundOptionControls(SoundOptionControls{Words: DefaultSoundOptionWords(),
		Read: func() audio.ChannelVolumes { return audio.FullChannelVolumes() }, Write: func(audio.Channel, int) error { return nil },
		ReadPlayback: func() MusicPreferences { return p }, WritePlayback: func(next MusicPreferences) error {
			if fail {
				return errors.New("profile refused")
			}
			p = next
			return nil
		},
		TrackTitle: func(name string) string { return "Title " + name }})
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	a.SetMusic(source, device, 1189)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	click := func(p image.Point) {
		t.Helper()
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	button := func(action gameMenuAction) {
		t.Helper()
		r := soundOptionRect(action)
		click(r.Min.Add(r.Size().Div(2)))
	}
	list := a.flow.soundOptions.list
	if list.Len() != 12 || list.Rows()[3].Text != "Title B03.wav" {
		t.Fatal("candidate titles", list.Rows())
	}
	click(image.Pt(130, 217))
	if list.Selection() != 3 || a.music.Playing() != "B00.wav" {
		t.Fatal("selection changed playback", list.Selection(), a.music.Playing())
	}
	button(gameMenuMusicPlay)
	if a.music.Playing() != "B03.wav" {
		t.Fatal("Play did not use selection")
	}
	if err := a.HeadlessGameMenuAction("music-tracks"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"end", "up", "up"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if list.Selection() != 9 {
		t.Fatal("list keyboard", list.Selection())
	}
	if err := a.HeadlessKey("tab"); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuRows()[a.flow.menuList.Selection()].Action != gameMenuMusicRandom {
		t.Fatal("Tab trapped in track list")
	}
	if err := a.HeadlessGameMenuAction("music-play"); err != nil {
		t.Fatal(err)
	}
	if a.music.Playing() != "B09.wav" {
		t.Fatal("keyboard Play lost chosen index")
	}
	before := len(source.loads)
	fail = true
	button(gameMenuMusicRandom)
	if p.RandomOrder || len(source.loads) != before || a.flow.msg == "" {
		t.Fatal("failed write altered playback")
	}
	fail = false
	button(gameMenuMusicRandom)
	if !p.RandomOrder || len(source.loads) != before || a.music.Playing() != "B09.wav" {
		t.Fatal("Random Order restarted track")
	}
	button(gameMenuMusicStop)
	if p.Enabled || a.music.Playing() != "" {
		t.Fatal("Stop failed")
	}
	for _, action := range []string{"page-return", "return"} {
		if err := a.HeadlessGameMenuAction(action); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if a.music.Playing() != "" || len(source.loads) != before {
		t.Fatal("menu exit resumed stopped music")
	}
}
