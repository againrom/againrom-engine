package main

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestWindowFlagDefaultsOffAndAcceptsBooleanForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"default", nil, false},
		{"on", []string{"-window"}, true},
		{"explicit on", []string{"-window=true"}, true},
		{"explicit off", []string{"-window=false"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parse(tc.args)
			if err != nil || got.window != tc.want {
				t.Fatalf("parse(%q): window=%t err=%v, want %t", tc.args, got.window, err, tc.want)
			}
			control, err := parse(nil)
			if err != nil {
				t.Fatal(err)
			}
			got.window = false
			if !reflect.DeepEqual(got, control) {
				t.Fatal("-window changed another launch option")
			}
		})
	}
}

// TestWindowFlagReachesTheWindow launches a normal start-up with and without
// -window and reads the form the application hands the window system.
func TestWindowFlagReachesTheWindow(t *testing.T) {
	for _, windowed := range []bool{false, true} {
		install := missionInstall(t)
		watchDesktop(t)
		var got []bool
		desktop.run = func(app *ui.App) error {
			got = append(got, app.Windowed())
			app.StopAudio()
			return nil
		}
		var extra []string
		if windowed {
			extra = append(extra, "-window")
		}
		if code, stderr := launch(t, install, extra...); code != 0 {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		if len(got) != 1 || got[0] != windowed {
			t.Fatalf("-window=%t: the window opened with windowed=%v", windowed, got)
		}
	}
}
