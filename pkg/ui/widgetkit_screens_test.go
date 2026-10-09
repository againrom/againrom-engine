package ui

import (
	"errors"
	"image"
	"strings"
	"testing"

	"againrom/pkg/audio"
)

// widgetKinds counts the shared builder calls of each kind.
func widgetKinds(calls []widgetCall) map[widgetKind]int {
	out := map[widgetKind]int{}
	for _, c := range calls {
		out[c.kind]++
	}
	return out
}

func wantWidgets(t *testing.T, screen string, calls []widgetCall, want map[widgetKind]int) {
	t.Helper()
	got := widgetKinds(calls)
	for kind, n := range want {
		if got[kind] < n {
			t.Fatalf("%s drew %d builder calls of kind %d, want at least %d (all %v)", screen, got[kind], kind, n, got)
		}
	}
}

// The in-game menu's items are the shared push button.
func TestGameMenuDrawsThroughTheSharedButton(t *testing.T) {
	a := NewApp("menu", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	calls := recordWidgets(t, func() { a.GameMenuPanel() })
	wantWidgets(t, "game menu", calls, map[widgetKind]int{widgetPushButton: 3})
}

// Game Options draws its speed slider, checkboxes, radio groups and push
// buttons through the shared builders.
func TestGameOptionsDrawsThroughTheSharedBuilders(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	calls := recordWidgets(t, func() { a.gameOptionsPicture() })
	wantWidgets(t, "game options", calls, map[widgetKind]int{widgetSlider: 1, widgetRadio: 3, widgetCheck: 7, widgetPushButton: 3})
}

// Sound Options draws its channel sliders, checkboxes, buttons and the track
// list through the shared builders.
func TestSoundOptionsDrawsThroughTheSharedBuilders(t *testing.T) {
	a := NewApp("sound options", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	volumes := audio.ChannelVolumes{50, 50, 50}
	a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, func(bool, int) error { return nil })
	a.SetSoundOptionControls(SoundOptionControls{Read: func() audio.ChannelVolumes { return volumes },
		Write: func(audio.Channel, int) error { return errors.New("unused") }, Words: DefaultSoundOptionWords()})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	calls := recordWidgets(t, func() { a.soundOptionsPicture() })
	wantWidgets(t, "sound options", calls, map[widgetKind]int{widgetSlider: 2, widgetPushButton: 2})
}

// An outcome notice's buttons are the shared push button.
func TestOutcomeNoticeDrawsThroughTheSharedButton(t *testing.T) {
	l := AuthoredOutcomeLayout()
	if l.Button.Empty() {
		t.Fatal("setup: the outcome layout has no button")
	}
	calls := recordWidgets(t, func() { RenderNotice(l, gameMenuTestFont(), "Mission complete", nil) })
	wantWidgets(t, "outcome notice", calls, map[widgetKind]int{widgetPushButton: 1})
}

// The tip panel's close control is the push button and its toggle the tip
// checkbox.
func TestTipPanelDrawsThroughTheSharedBuilders(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 312, 200))
	v := TipPanelView{Rect: dst.Bounds(), Text: "x", ToggleOn: true, Art: tipTestArt(), Font: shopTipTestFont()}
	calls := recordWidgets(t, func() { ComposeTipPanel(dst, v) })
	wantWidgets(t, "tip panel", calls, map[widgetKind]int{widgetPushButton: 1, widgetCheck: 1})
}

// The quest objectives page scrolls with the shared bar and closes with the
// shared button.
func TestQuestObjectivesDrawsThroughTheSharedBuilders(t *testing.T) {
	a := NewApp("quest", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.menuFont = gameMenuTestFont()
	a.flow.menuContext.Objective = strings.Repeat("Find the hidden road through the hills. ", 30)
	calls := recordWidgets(t, func() { a.questPicture() })
	wantWidgets(t, "quest objectives", calls, map[widgetKind]int{widgetVScrollBar: 1, widgetPushButton: 1})
}

// A mod screen scrolls with the shared bar and returns with the shared
// button; the main menu's mod entries are the shared button.
func TestModScreensDrawThroughTheSharedBuilders(t *testing.T) {
	screens := modTestScreens()
	for i := 0; i < 60; i++ {
		screens[0].Paragraphs = append(screens[0].Paragraphs, "A paragraph long enough to take its own line.")
	}
	a := modTestApp(t, screens)
	entries := recordWidgets(t, func() { a.drawModMenuEntries(image.NewRGBA(image.Rect(0, 0, 640, 480))) })
	wantWidgets(t, "main menu mod entries", entries, map[widgetKind]int{widgetPushButton: 2})
	a.flow.openModScreen(0, ScreenMenu, 0)
	calls := recordWidgets(t, func() {
		if _, err := a.composeModScreen(); err != nil {
			t.Fatal(err)
		}
	})
	wantWidgets(t, "mod screen", calls, map[widgetKind]int{widgetVScrollBar: 1, widgetPushButton: 1})
}

// The Drop Gold editor's amount is the shared edit field and its two
// controls the shared button.
func TestGoldEditorDrawsThroughTheSharedBuilders(t *testing.T) {
	_, v, _ := openPurseEditor(t)
	calls := recordWidgets(t, func() { v.goldModalPresent() })
	wantWidgets(t, "gold editor", calls, map[widgetKind]int{widgetEdit: 1, widgetPushButton: 2})
}

// Every hover help kind is the shared hover box.
func TestTooltipsDrawThroughTheSharedHoverBox(t *testing.T) {
	for _, kind := range []uint8{tooltipText, tooltipItem, tooltipSpell, tooltipCommand} {
		target := tooltipTarget{kind, "k", []string{"Sword"}, messageFont()}
		calls := recordWidgets(t, func() { tooltipPicture(target, image.Pt(100, 100), image.Rect(0, 0, 640, 480), nil) })
		wantWidgets(t, "tooltip", calls, map[widgetKind]int{widgetHoverBox: 1})
	}
}
