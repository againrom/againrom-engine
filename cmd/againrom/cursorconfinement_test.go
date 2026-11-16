package main

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestCursorInWindowFlagReachesTheApp(t *testing.T) {
	for _, on := range []bool{false, true} {
		var args []string
		if on {
			args = []string{"-cursor-in-window"}
		}
		o, err := parse(args)
		if err != nil || o.cursorInWindow != on {
			t.Fatalf("parse(%q): %t, %v", args, o.cursorInWindow, err)
		}
		control, err := parse(nil)
		if err != nil {
			t.Fatal(err)
		}
		o.cursorInWindow = false
		if !reflect.DeepEqual(o, control) {
			t.Fatal("cursor confinement changed another launch option")
		}
		root := missionInstall(t)
		watchDesktop(t)
		calls := 0
		desktop.run = func(app *ui.App) error {
			calls++
			defer app.StopAudio()
			if app.CursorInWindow() != on {
				t.Errorf("App cursor confinement=%t, want %t", app.CursorInWindow(), on)
			}
			return nil
		}
		if code, stderr := launch(t, root, args...); code != 0 || calls != 1 {
			t.Fatalf("launch: exit=%d calls=%d: %s", code, calls, stderr)
		}
	}
}
