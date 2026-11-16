package main

import (
	"testing"

	"againrom/pkg/ini"
	"againrom/pkg/launch"
)

// The window filter is Catmull-Rom until the player steps it; the choice goes
// through the settings file and comes back, and the game is launched with the
// flag the launch table names for it.
func TestWindowFilterDefaultsToCatmullRomAndRoundTrips(t *testing.T) {
	s := defaultSettings()
	if s.WindowFilter != launch.FilterCatmullRom {
		t.Fatalf("default window filter %q", s.WindowFilter)
	}
	args, err := s.launchArgs(gameLaunch{Root: "r", ModsDir: "m"})
	if err != nil || len(args) != 2 {
		t.Fatalf("default launch %q (%v) passes more than the base", args, err)
	}

	a := rigWithLanguage(t, "english").app()
	openSettings(t, a)
	a.s.WindowFilter = launch.FilterCatmullRom
	do(t, a, action{kind: aWindowFilter})
	if a.s.WindowFilter != launch.FilterMitchell {
		t.Fatalf("one step gives %q", a.s.WindowFilter)
	}

	f, _ := ini.Parse(nil)
	a.s.store(f)
	again := loadSettings(mustParse(t, string(f.Bytes())))
	if again.WindowFilter != launch.FilterMitchell {
		t.Fatalf("stored and loaded %q", again.WindowFilter)
	}
	args, err = again.launchArgs(gameLaunch{Root: "r", ModsDir: "m"})
	if err != nil || args[len(args)-1] != "-mitchell" {
		t.Fatalf("launch %q (%v) lacks -mitchell", args, err)
	}

	do(t, a, action{kind: aWindowFilter})
	if a.s.WindowFilter != launch.FilterCatmullRom {
		t.Fatalf("second step gives %q", a.s.WindowFilter)
	}
}
