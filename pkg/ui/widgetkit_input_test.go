package ui

import (
	"testing"

	"againrom/pkg/audio"
)

// A thumb gesture between two Load row clicks breaks the double click.
func TestLoadThumbTapBreaksDoubleClick(t *testing.T) {
	a, calls := loadScrollApp(t, 27)
	for _, ev := range []struct {
		edge string
		x, y int
	}{
		{"press", 140, 158}, {"release", 140, 158},
		{"press", 516, 188}, {"release", 516, 188},
		{"press", 140, 158}, {"release", 140, 158},
	} {
		if err := a.HeadlessPointer(ev.edge, ev.x, ev.y); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls.loaded) != 0 {
		t.Fatalf("save loaded after intervening thumb tap: %v", calls.loaded)
	}
}

func gameOptionsMenuRowPoint(t *testing.T, a *App) (int, int) {
	t.Helper()
	i := -1
	for k, row := range a.flow.menuRows() {
		if row.Action == gameMenuGameOptions {
			i = k
			break
		}
	}
	if i < 0 {
		t.Fatal("game options row absent")
	}
	a.flow.menuList.Select(i)
	top, _ := a.flow.menuList.Visible()
	r := gameMenuRowRect(a.flow.menuPanelSurface(), i-top)
	p := r.Min.Add(r.Size().Div(2))
	return p.X, p.Y
}

func TestMenuLatchCancelsOnFocusLoss(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	x, y := gameOptionsMenuRowPoint(t, a)
	if err := a.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuPage != gameMenuRoot {
		t.Fatalf("release after focus loss activated another page: got %v, want root %v", a.flow.menuPage, gameMenuRoot)
	}
	// A fresh press and release still opens the page.
	if err := a.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuPage == gameMenuRoot {
		t.Fatal("a fresh click no longer opens Game Options")
	}
}

func TestNoticeLatchCancelsOnFocusLoss(t *testing.T) {
	a, seam := noticeApp(t)
	seam.v.SetNotice("outcome", NoticeSuccess)
	if err := a.HeadlessPointer("press", 320, 296); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", 320, 296); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 {
		t.Fatalf("release after focus loss activated outcome: %v", seam.actions)
	}
	if err := a.HeadlessPointer("press", 320, 296); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", 320, 296); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 1 {
		t.Fatalf("a fresh click no longer advances the outcome: %v", seam.actions)
	}
}

func TestNoticeLatchCancelsOnReplacement(t *testing.T) {
	a, seam := noticeApp(t)
	seam.v.SetNotice("first outcome", NoticeSuccess)
	if err := a.HeadlessPointer("press", 320, 296); err != nil {
		t.Fatal(err)
	}
	seam.v.SetNotice("replacement outcome", NoticeSuccess)
	if err := a.HeadlessPointer("release", 320, 296); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 {
		t.Fatalf("release from old notice activated replacement: %v", seam.actions)
	}
}

func TestSoundChecksToggleOnPress(t *testing.T) {
	for _, control := range []string{"random", "acknowledgments"} {
		t.Run(control, func(t *testing.T) {
			a := NewApp("sound", appAssets(t), appRows(1), nil)
			a.Layout(640, 480)
			a.flow.viewer, a.flow.screen, a.flow.menuFont = fiViewer(t), ScreenMap, gameMenuTestFont()
			a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, func(bool, int) error { return nil })
			playback := MusicPreferences{Enabled: true}
			ack := false
			a.SetSoundOptionControls(SoundOptionControls{
				Words:                DefaultSoundOptionWords(),
				Read:                 func() audio.ChannelVolumes { return audio.FullChannelVolumes() },
				Write:                func(audio.Channel, int) error { return nil },
				ReadPlayback:         func() MusicPreferences { return playback },
				WritePlayback:        func(v MusicPreferences) error { playback = v; return nil },
				ReadAcknowledgments:  func() bool { return ack },
				WriteAcknowledgments: func(v bool) error { ack = v; return nil },
			})
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
				t.Fatal(err)
			}
			action := gameMenuMusicRandom
			if control == "acknowledgments" {
				action = gameMenuAcknowledgments
			}
			r := soundOptionRect(action)
			p := r.Min.Add(r.Size().Div(2))
			value := func() bool {
				if control == "acknowledgments" {
					// The page holds its own copy until OK writes it.
					return a.flow.soundOptions.acknowledgments
				}
				return playback.RandomOrder
			}
			if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
			if !value() {
				t.Fatalf("%s stayed off on press", control)
			}
			// MENU-124: the release, inside or outside, is a no-op.
			if err := a.HeadlessPointer("release", 5, 5); err != nil {
				t.Fatal(err)
			}
			if !value() {
				t.Fatalf("%s changed again on release", control)
			}
		})
	}
}

func TestSaveCaretDrawnAfterLabel(t *testing.T) {
	a, _, _ := saveChooserStyleApp(t, 1)
	if err := a.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	if !a.blink.on() {
		t.Fatal("setup: blink is off")
	}
	if a.flow.saveDialog.caret != 0 {
		t.Fatal("setup: caret not at Home")
	}
	pix, err := a.composeSaveDialogScreen()
	if err != nil {
		t.Fatal(err)
	}
	r := saveControlRect(saveNameControl)
	origin := (editField{Rect: r, TextH: 15}).TextOrigin()
	x, y := origin.X, origin.Y+1
	if pix.RGBAAt(x, y) != editCaret || pix.RGBAAt(x+1, y) != editCaret {
		t.Fatalf("Home caret overwritten by label at (%d,%d): %v / %v, want %v", x, y, pix.RGBAAt(x, y), pix.RGBAAt(x+1, y), editCaret)
	}
}

func TestFocusedCheckboxSpaceToggles(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	clickRect(t, a, gameOptionRect(gameMenuDayNight))
	if a.flow.optionValues()[GameOptionDayNight] != 1 {
		t.Fatal("setup: checkbox not selected")
	}
	if err := a.HeadlessKey("space"); err != nil {
		t.Fatal(err)
	}
	if a.flow.optionValues()[GameOptionDayNight] != 0 {
		t.Fatalf("focused checkbox ignored Space: value %d", a.flow.optionValues()[GameOptionDayNight])
	}
}

func TestFocusedRadioDownSelectsNextRow(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	clickRect(t, a, gameOptionChoiceRect(gameMenuFormation, 0))
	if err := a.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if a.flow.optionValues()[GameOptionFormation] != 1 {
		t.Fatalf("focused radio ignored Down: selection %d, focused action %v", a.flow.optionValues()[GameOptionFormation], a.flow.menuRows()[a.flow.menuList.Selection()].Action)
	}
	if err := a.HeadlessKey("up"); err != nil {
		t.Fatal(err)
	}
	if a.flow.optionValues()[GameOptionFormation] != 0 {
		t.Fatalf("focused radio ignored Up: selection %d", a.flow.optionValues()[GameOptionFormation])
	}
	focused := func() gameMenuAction { return a.flow.menuRows()[a.flow.menuList.Selection()].Action }
	// Tab leaves the group without changing it.
	if err := a.HeadlessKey("tab"); err != nil {
		t.Fatal(err)
	}
	if focused() == gameMenuFormation || a.flow.optionValues()[GameOptionFormation] != 0 {
		t.Fatalf("Tab: focus %v, selection %d", focused(), a.flow.optionValues()[GameOptionFormation])
	}
	// Past the last row, Down moves to the next control (DIV-2592).
	clickRect(t, a, gameOptionChoiceRect(gameMenuFormation, 2))
	if err := a.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if focused() == gameMenuFormation || a.flow.optionValues()[GameOptionFormation] != 2 {
		t.Fatalf("Down past the group: focus %v, selection %d", focused(), a.flow.optionValues()[GameOptionFormation])
	}
}
