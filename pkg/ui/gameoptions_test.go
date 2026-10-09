package ui

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"
)

type optionWrite struct {
	option GameOption
	value  int
}

// optionsApp opens Game Options over a map with every write recorded in order.
func optionsApp(t *testing.T, values *GameOptionValues, log *[]optionWrite, fail *bool) *App {
	t.Helper()
	a := NewApp("options", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	a.SetGameOptionControls(GameOptionControls{
		Read: func(bool) GameOptionValues { return *values },
		Write: func(onMap bool, o GameOption, v int) error {
			if !onMap {
				t.Fatal("map menu lost its owning screen")
			}
			if fail != nil && *fail {
				return errors.New("read-only profile")
			}
			(*values)[o] = v
			*log = append(*log, optionWrite{o, v})
			return nil
		},
		Words: GameOptionWords{Title: "Game Options", Tips: "Tips", OK: "OK", Cancel: "Cancel", Speed: "Game Speed",
			Labels:    [11]string{"Day/Night", "Health", "Damage", "Formation", "Retreat", "Pathfinding", "Smoothing", "Shadows", "Lighting", "Animation", "AutoHealing"},
			Formation: [3]string{"Off", "Auto", "On"}, Retreat: [3]string{"Never", "Low", "Medium"}, AutoHealing: [3]string{"No", "Standard", "Often"}},
	})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	return a
}

func clickRect(t *testing.T, a *App, r image.Rectangle) {
	t.Helper()
	p := r.Min.Add(r.Size().Div(2))
	if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", p.X, p.Y); err != nil {
		t.Fatal(err)
	}
}

func TestGameOptionsKeepTheMapShortcutsImmediate(t *testing.T) {
	values := GameOptionValues{1, 1, 1, 1, 0}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	a.HeadlessKey("escape")
	a.HeadlessKey("escape")
	for _, key := range []string{"ctrl-n", "ctrl-h", "ctrl-l", "ctrl-f", "ctrl-w"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if values != (GameOptionValues{0, 0, 0, 2, 1}) || len(log) != 5 {
		t.Fatal("shortcuts missed the shared writer", values, log)
	}
}

// A click changes the dialog's own copy and nothing else (MENU-074).
func TestGameOptionsClicksWriteNothingUntilOK(t *testing.T) {
	values := GameOptionValues{1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	if got := len(a.HeadlessRows()); got != 18 || a.GameMenuPanel() == nil {
		t.Fatal("complete options panel unavailable", got)
	}
	clickRect(t, a, gameOptionChoiceRect(gameMenuFormation, 0))
	clickRect(t, a, gameOptionChoiceRect(gameMenuRetreat, 2))
	for _, action := range []gameMenuAction{gameMenuDayNight, gameMenuHealth, gameMenuDamage, gameMenuAnimation} {
		clickRect(t, a, gameOptionRect(action))
	}
	if len(log) != 0 || values != (GameOptionValues{1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1}) {
		t.Fatal("a click reached the stored options", values, log)
	}
	if got := a.flow.optionValues(); got != (GameOptionValues{0, 0, 0, 0, 2, 0, 1, 1, 1, 0, 1}) {
		t.Fatal("the dialog's copy did not take the clicks", got)
	}
	// Cancel writes nothing, and reopening shows the stored values again.
	clickRect(t, a, gameOptionRect(gameMenuOptionsCancel))
	if a.flow.menuPage != gameMenuRoot || len(log) != 0 || values[GameOptionDayNight] != 1 {
		t.Fatal("Cancel wrote or stayed", a.flow.menuPage, log)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if a.flow.optionValues() != values {
		t.Fatal("a cancelled copy came back", a.flow.optionValues())
	}
	// Escape discards the copy as Cancel does.
	clickRect(t, a, gameOptionRect(gameMenuDayNight))
	a.HeadlessKey("escape")
	if a.flow.menuPage != gameMenuRoot || len(log) != 0 {
		t.Fatal("Escape wrote the copy", log)
	}
}

// OK writes in the claim's order, forces Lighting 0 under Animation 0 and sends
// the three party commands whether or not they changed (MENU-074).
func TestGameOptionsOKWritesInOrderAndForcesLighting(t *testing.T) {
	values := GameOptionValues{1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	clickRect(t, a, gameOptionRect(gameMenuAnimation))
	clickRect(t, a, gameOptionRect(gameMenuPageReturn))
	if a.flow.menuPage != gameMenuRoot {
		t.Fatal("OK did not return to the root menu")
	}
	want := []optionWrite{{GameOptionDayNight, 1}, {GameOptionSmoothing, 1}, {GameOptionShadows, 1},
		{GameOptionLighting, 0}, {GameOptionAnimation, 0}, {GameOptionHealth, 1}, {GameOptionDamage, 1},
		{GameOptionPathfinding, 0}, {GameOptionFormation, 1}, {GameOptionRetreat, 0}, {GameOptionAutoHealing, 1}}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("OK wrote %v want %v", log, want)
	}
	// The commands go out again on an OK that changed nothing.
	log = nil
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	var commands int
	for _, w := range log {
		if w.option == GameOptionFormation || w.option == GameOptionRetreat || w.option == GameOptionAutoHealing {
			commands++
		}
	}
	if commands != 3 {
		t.Fatal("an unchanged OK skipped a party command", log)
	}
}

func TestGameOptionsFailedWriteKeepsThePageOpen(t *testing.T) {
	values := GameOptionValues{1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1}
	var log []optionWrite
	fail := true
	a := optionsApp(t, &values, &log, &fail)
	clickRect(t, a, gameOptionRect(gameMenuHealth))
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if a.flow.menuPage != gameMenuGameOptionsPage || !strings.Contains(a.HeadlessMessage(), "read-only profile") {
		t.Fatal("a failed OK left the page or hid the error", a.flow.menuPage, a.HeadlessMessage())
	}
	if a.flow.optionValues()[GameOptionHealth] != 0 {
		t.Fatal("the copy was lost with the failure")
	}
}

func TestGameOptionsSpeedSliderIsStagedAndStepsLevels(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	a.HeadlessKey("escape")
	a.flow.rung = 7
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	d := a.flow.gameOptions.draft
	if d.speed != 5 {
		t.Fatalf("slider starts at %d, want the default level 5", d.speed)
	}
	track := gameOptionRect(gameMenuSpeedDown)
	press := func(x int) {
		a.HeadlessPointer("press", x, track.Min.Y+4)
		a.HeadlessPointer("release", x, track.Min.Y+4)
	}
	// MENU-118: x maps to trunc(N*(x-L-H-2)/(W-2H-4)) on the track; the
	// endcaps step by one.
	press(track.Min.X + 26)
	if d.speed != 0 {
		t.Fatalf("track start is level %d", d.speed)
	}
	press(track.Max.X - 25)
	if d.speed != gameSpeedLevels {
		t.Fatalf("track end is level %d", d.speed)
	}
	press(track.Min.X + 2)
	if d.speed != gameSpeedLevels-1 {
		t.Fatalf("left endcap stepped to level %d", d.speed)
	}
	press(track.Max.X - 2)
	if d.speed != gameSpeedLevels {
		t.Fatalf("right endcap stepped to level %d", d.speed)
	}
	a.HeadlessKey("escape")
	if a.flow.rung != 7 {
		t.Fatal("Escape applied the slider", a.flow.rung)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	track = gameOptionRect(gameMenuSpeedDown)
	press(track.Min.X + 26)
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if a.flow.rung != 7 || !a.flow.stopped {
		t.Fatal("OK did not pause while preserving the resume rung", a.flow.rung, a.flow.stopped)
	}
}
