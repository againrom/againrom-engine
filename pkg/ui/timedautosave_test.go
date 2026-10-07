package ui

import (
	"errors"
	"image"
	"testing"
)

func TestTimedAutosaveOptionsOKCancelAndFailure(t *testing.T) {
	var values GameOptionValues
	var writes []optionWrite
	a := optionsApp(t, &values, &writes, nil)
	v := TimedAutosaveSettings{Enabled: true, Minutes: 5}
	fail := false
	commits := 0
	a.SetTimedAutosaveControls(TimedAutosaveControls{
		Read: func() TimedAutosaveSettings { return v },
		Write: func(next TimedAutosaveSettings) error {
			if fail {
				return errors.New("profile unavailable")
			}
			v = next
			commits++
			return nil
		},
	})
	a.flow.rebuildGameMenu(gameMenuRoot, 0)
	action := func(name string) {
		t.Helper()
		if err := a.HeadlessGameMenuAction(name); err != nil {
			t.Fatal(err)
		}
	}
	action("game-options")
	minute := gameOptionRect(gameMenuAutosaveMinutes)
	clickRect(t, a, image.Rect(minute.Min.X+minute.Dx()/2, minute.Min.Y, minute.Max.X, minute.Max.Y))
	if a.flow.gameOptions.draft.autosave.Minutes != 6 {
		t.Fatal("pointer increase")
	}
	clickRect(t, a, image.Rect(minute.Min.X, minute.Min.Y, minute.Min.X+minute.Dx()/2, minute.Max.Y))
	if a.flow.gameOptions.draft.autosave.Minutes != 5 {
		t.Fatal("pointer decrease")
	}
	action("timed-autosave")
	action("autosave-minutes")
	if v != (TimedAutosaveSettings{true, 5}) || commits != 0 {
		t.Fatal("draft mutated preferences", v, commits)
	}
	action("options-cancel")
	if v != (TimedAutosaveSettings{true, 5}) || commits != 0 {
		t.Fatal("Cancel persisted", v, commits)
	}
	action("game-options")
	action("autosave-minutes")
	if err := a.HeadlessKey("left"); err != nil {
		t.Fatal(err)
	}
	if a.flow.gameOptions.draft.autosave.Minutes != 5 {
		t.Fatal("minute left control")
	}
	action("timed-autosave")
	action("autosave-minutes")
	fail = true
	action("page-return")
	if v != (TimedAutosaveSettings{true, 5}) || commits != 0 || a.flow.menuPage != gameMenuGameOptionsPage || a.HeadlessMessage() == "" {
		t.Fatal("failed OK committed or hid error", v, commits, a.HeadlessMessage())
	}
	fail = false
	action("page-return")
	if v != (TimedAutosaveSettings{false, 6}) || commits != 1 {
		t.Fatal("OK did not persist pair once", v, commits)
	}
	action("game-options")
	action("timed-autosave")
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if v != (TimedAutosaveSettings{false, 6}) || commits != 1 {
		t.Fatal("Escape persisted draft")
	}
}

func TestTimedAutosaveResetAndModalAdmission(t *testing.T) {
	var values GameOptionValues
	var writes []optionWrite
	a := optionsApp(t, &values, &writes, nil)
	resets, writesNow := 0, 0
	a.SetTimedAutosaveControls(TimedAutosaveControls{Reset: func() { resets++ }, Poll: func(_ *Viewer, onMap, ready bool) error {
		if ready {
			if !onMap {
				t.Fatal("wrong source")
			}
			writesNow++
		}
		return nil
	}})
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if writesNow != 0 {
		t.Fatal("options modal admitted a write")
	}
	if err := a.HeadlessGameMenuAction("options-cancel"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if writesNow != 1 {
		t.Fatal("resume not admitted", writesNow)
	}
	a.flow.viewer.PostMessage("ordinary message", MessageWhite, 0)
	before := writesNow
	a.flow.showTextNotice("modal")
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if writesNow != before {
		t.Fatal("notice modal admitted write")
	}
	a.flow.viewer.ClearNotice()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if writesNow != before+1 || resets != 1 {
		t.Fatal("notice resume or resets", writesNow, resets)
	}
}

func TestTimedAutosaveTownDialogueDefers(t *testing.T) {
	a := NewApp("town autosave", appAssets(t), appRows(1), nil)
	town := &dialogueOverMapTown{dialogue: image.NewRGBA(image.Rect(0, 0, 64, 64))}
	a.SetTown(town)
	a.flow.screen = ScreenTown
	writes := 0
	a.SetTimedAutosaveControls(TimedAutosaveControls{Poll: func(_ *Viewer, onMap, ready bool) error {
		if ready {
			if onMap {
				t.Fatal("town admitted map source")
			}
			writes++
		}
		return nil
	}})
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatal("town dialogue admitted autosave")
	}
	town.dialogue = nil
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatal("town dialogue resume", writes)
	}
}
