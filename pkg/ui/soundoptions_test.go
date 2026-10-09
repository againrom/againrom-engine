package ui

import (
	"errors"
	"image"
	"strings"
	"testing"

	"againrom/pkg/audio"
)

func TestSoundOptions1187PointerKeyboardPersistenceFailureAndModalExit(t *testing.T) {
	a := NewApp("sound options", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	volumes := audio.ChannelVolumes{50, 50, 50}
	writes, fail := 0, false
	enabled, master := true, 100
	a.SetGameMenuSettings(nil, nil,
		func() (bool, int, bool) { return enabled, master, true },
		func(on bool, volume int) error { enabled, master = on, volume; return nil })
	a.SetSoundOptionControls(SoundOptionControls{Read: func() audio.ChannelVolumes { return volumes },
		Write: func(channel audio.Channel, value int) error {
			if fail {
				return errors.New("read-only profile")
			}
			volumes[channel] = value
			writes++
			return nil
		}, Words: DefaultSoundOptionWords()})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	if len(a.HeadlessRows()) != 8 || a.GameMenuPanel() == nil {
		t.Fatal("channel panel unavailable")
	}
	pointer := func(kind string, p image.Point) {
		t.Helper()
		if err := a.HeadlessPointer(kind, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	for channel := audio.Channel(0); channel < audio.ChannelCount; channel++ {
		start, end := SoundSliderPoint(channel, 0), SoundSliderPoint(channel, 100)
		pointer("press", start)
		if volumes[channel] != 0 || writes != 2*int(channel)+1 {
			t.Fatal("the press did not apply the slider at the control event", volumes, writes)
		}
		pointer("move", end)
		if volumes[channel] != 100 || writes != 2*int(channel)+2 {
			t.Fatal("the drag did not apply each move", volumes, writes)
		}
		pointer("release", end)
		if volumes[channel] != 100 || writes != 2*int(channel)+2 {
			t.Fatal("release wrote again or lost full gain", volumes, writes)
		}
	}
	if err := a.HeadlessKey("left"); err != nil {
		t.Fatal(err)
	}
	stepped := soundSliderPercent(soundSliderRange - soundKeyStep)
	if volumes != (audio.ChannelVolumes{100, 100, stepped}) {
		t.Fatal("keyboard did not adjust the focused speech channel", volumes)
	}
	fail = true
	if err := a.HeadlessKey("left"); err != nil {
		t.Fatal(err)
	}
	if volumes[audio.SpeechChannel] != stepped || !strings.Contains(a.HeadlessMessage(), "read-only profile") {
		t.Fatal("failed write changed the value or hid the error")
	}
	fail = false
	// A release alone must not choose an unrelated slider or button.
	track := soundSliderRect(audio.MusicChannel)
	pointer("release", track.Min)
	if volumes[audio.MusicChannel] != 100 {
		t.Fatal("unpaired release changed a slider")
	}
	// Escape has no restore: a slider keeps the value it was moved to.
	pointer("press", SoundSliderPoint(audio.MusicChannel, 0))
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuPage != gameMenuRoot || !a.flow.viewer.menuUp || volumes[audio.MusicChannel] != 0 {
		t.Fatal("Escape lost modal ownership or undid the applied slider", volumes)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	button := soundOptionRect(gameMenuPageReturn)
	p := button.Min.Add(button.Size().Div(2))
	pointer("press", p)
	pointer("release", p)
	if a.flow.menuPage != gameMenuRoot {
		t.Fatal("OK did not return to menu root")
	}
}
