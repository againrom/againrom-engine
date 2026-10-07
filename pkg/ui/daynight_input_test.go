package ui

import (
	"testing"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// dayNightApp is a front-end sitting on the map screen over one viewer with
// relief in it, so a relight has something to change.
func dayNightApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a := NewApp("t", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	a.flow.viewer = v
	a.flow.screen = ScreenMap
	return a, v
}

// TestDayNightKeysDoNothingUnpressed - AC-15: a snapshot naming neither key
// leaves the switch and the drawn sun exactly where they were, over many ticks.
func TestDayNightKeysDoNothingUnpressed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := dayNightApp(t)
	v.SetLightClock(0)
	sun := v.Sun()
	for i := 0; i < 50; i++ {
		a.step(appInput{}, now)
	}
	if !v.TimeFlow() {
		t.Error("the switch moved with no key pressed")
	}
	if v.Sun() != sun {
		t.Errorf("the sun moved with no key pressed: %+v, want %+v", v.Sun(), sun)
	}
}

func TestTimeFlowKeyFlipsOncePerPress(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := dayNightApp(t)

	a.step(appInput{TimeFlow: true}, now)
	if v.TimeFlow() {
		t.Fatal("one press left the switch on")
	}
	a.step(appInput{}, now)
	a.step(appInput{}, now)
	if v.TimeFlow() {
		t.Fatal("the switch came back on with no key pressed")
	}
	a.step(appInput{TimeFlow: true}, now)
	if !v.TimeFlow() {
		t.Fatal("a second press did not restore the switch")
	}
}

func TestTimeFlowKeyMovesTheDrawnSun(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := dayNightApp(t)
	v.SetLightClock(0) // minute 0, the day arm's start
	running := v.Sun()

	a.step(appInput{TimeFlow: true}, now)
	if v.Sun() == running {
		t.Fatalf("N did not move the sun off the cycle: still %+v", v.Sun())
	}
	if want := terrain.SunAt(0, false); v.Sun() != want {
		t.Errorf("with the cycle off the sun is %+v, want %+v", v.Sun(), want)
	}
	a.step(appInput{TimeFlow: true}, now)
	if v.Sun() != running {
		t.Errorf("N did not restore the cycle's sun: %+v, want %+v", v.Sun(), running)
	}
}

func TestLightStepKeyAdvancesAnHour(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := dayNightApp(t)
	v.SetLightClock(0)

	a.step(appInput{LightStep: true}, now)
	if want := terrain.SunAt(60, true); v.Sun() != want {
		t.Errorf("after one F3 the sun is %+v, want minute 60's %+v", v.Sun(), want)
	}
	if !v.TimeFlow() {
		t.Error("F3 moved the switch")
	}
	for i := 1; i < 24; i++ {
		a.step(appInput{LightStep: true}, now)
	}
	if want := terrain.SunAt(0, true); v.Sun() != want {
		t.Errorf("after 24 F3 presses the sun is %+v, want the opening %+v", v.Sun(), want)
	}
}

// TestDayNightKeysAreReadOnTheMapArmAlone - AC-15: on the menu and the picker
// both keys do nothing at all.
func TestDayNightKeysAreReadOnTheMapArmAlone(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, screen := range []Screen{ScreenMenu, ScreenPicker} {
		a, v := dayNightApp(t)
		a.flow.screen = screen
		v.SetLightClock(0)
		sun := v.Sun()

		a.step(appInput{TimeFlow: true}, now)
		a.step(appInput{LightStep: true}, now)

		if !v.TimeFlow() {
			t.Errorf("%v: N moved the switch off the map screen", screen)
		}
		if v.Sun() != sun {
			t.Errorf("%v: the sun moved off the map screen", screen)
		}
	}
}
