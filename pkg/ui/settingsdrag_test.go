package ui

import (
	"image"
	"testing"

	"againrom/pkg/audio"
)

func TestGameOptionsSpeedDragLeavesTheFocusWhereItWas(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	row := -1
	for i, r := range a.flow.menuRows() {
		if isGameOptionCheck(r.Action) && r.Enabled {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatal("no checkbox row on Game Options")
	}
	a.flow.menuList.Select(row)
	track := gameOptionRect(gameMenuSpeedDown)
	for _, x := range []int{track.Min.X + 26, track.Max.X - 25} {
		a.HeadlessPointer("press", x, track.Min.Y+4)
		a.HeadlessPointer("move", x+3, track.Min.Y+4)
		a.HeadlessPointer("release", x+3, track.Min.Y+4)
		if got := a.flow.menuList.Selection(); got != row {
			t.Fatalf("a speed drag moved the focus from row %d to %d", row, got)
		}
	}
	action := a.flow.menuRows()[row].Action
	option := GameOption(action - gameMenuDayNight)
	before := a.flow.optionValues()[option]
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if after := a.flow.optionValues()[option]; after == before {
		t.Fatal("Enter after the drag did not toggle the focused checkbox")
	}
}

func TestSoundOptionsDragReleasedOutsideTheWindowLeavesTheFocus(t *testing.T) {
	a := NewApp("sound options", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	volumes := audio.ChannelVolumes{50, 50, 50}
	a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, func(bool, int) error { return nil })
	a.SetSoundOptionControls(SoundOptionControls{Read: func() audio.ChannelVolumes { return volumes },
		Write: func(channel audio.Channel, value int) error { volumes[channel] = value; return nil }, Words: DefaultSoundOptionWords()})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	a.flow.menuList.Select(0)
	start := SoundSliderPoint(audio.Channel(1), 0)
	a.HeadlessPointer("press", start.X, start.Y)
	out := image.Pt(-40, -40)
	a.HeadlessPointer("move", out.X, out.Y)
	a.HeadlessPointer("release", out.X, out.Y)
	if got := a.flow.menuList.Selection(); got != 0 {
		t.Fatalf("a drag released outside the window focused row %d", got)
	}
	end := SoundSliderPoint(audio.Channel(1), 100)
	a.HeadlessPointer("press", start.X, start.Y)
	a.HeadlessPointer("release", end.X, end.Y)
	if got := a.flow.menuList.Selection(); got == 0 {
		t.Fatal("a drag released inside the window no longer focuses its row")
	}
}
