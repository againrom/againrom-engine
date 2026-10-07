package ui

// The readout's key (AC-1, AC-9, success criterion SC-6).
//
// It is driven through App.step — the production dispatch, map arm and all —
// rather than by calling the toggle, because what is being asserted is WHERE the
// read stands: "on every other screen it does nothing" is a property of the map
// arm being the only arm with a statement to reach a viewer through, and a test
// that called the method directly could not see that at all.

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// readoutKeyViewer parks the front-end on the map screen over a bare loader and
// hands back the app and the viewer under it.
func readoutKeyViewer(t *testing.T) (*App, *Viewer, int) {
	t.Helper()
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("f1", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.SetFont(panelFont())
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	a := newTestApp(t, appRows(3), load)
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	if a.Screen() != ScreenMap || a.flow.viewer == nil {
		t.Fatalf("setup: screen = %v, viewer = %v", a.Screen(), a.flow.viewer)
	}
	return a, a.flow.viewer, 1
}

// readoutKeyFrame is a neutral map-screen frame carrying only this key.
func readoutKeyFrame(pressed bool) appInput {
	in := cadenceInput(false, false, false)
	in.Readout = pressed
	return in
}

// TestTheReadoutKeyTogglesOncePerPress — 0060 SC-6 (AC-1, AC-9).
func TestTheReadoutKeyTogglesOncePerPress(t *testing.T) {
	a, v, at := readoutKeyViewer(t)

	// An opened map is already showing it. That is the requirement, not a
	// convenience: a box that had to be switched on could not report the key
	// that switched it on.
	if !v.ReadoutShown() {
		t.Fatal("an opened map hides the readout")
	}

	press := func(pressed bool) {
		at++
		a.step(readoutKeyFrame(pressed), cadenceAt(at))
	}

	press(true)
	if v.ReadoutShown() {
		t.Error("the key did not hide the readout")
	}
	press(true)
	if !v.ReadoutShown() {
		t.Error("the key did not show it again")
	}

	// A HELD key delivers one true and then false, so three frames of holding
	// it act once. A level would flicker the box at the frame rate.
	press(true)
	press(false)
	press(false)
	if v.ReadoutShown() {
		t.Error("a held key over three frames toggled more than once")
	}
}

// TestTheReadoutKeyIsReadOnTheMapArmAlone — 0060 SC-6: every other
// screen's arm has no statement to reach a viewer through.
func TestTheReadoutKeyIsReadOnTheMapArmAlone(t *testing.T) {
	a, v, at := readoutKeyViewer(t)

	// Leave the map. The flow drops its pointer; this test still holds the
	// viewer, which is what makes "nothing reached it" observable.
	at++
	a.step(appInput{Escape: true}, cadenceAt(at))
	leaveViaMenu(a.flow) //
	if a.Screen() != ScreenPicker {
		t.Fatalf("Esc left the screen at %v, want the picker", a.Screen())
	}

	for _, screen := range []Screen{ScreenPicker, ScreenMenu} {
		a.flow.screen = screen
		before := v.ReadoutShown()
		for i := 0; i < 3; i++ {
			at++
			a.step(readoutKeyFrame(true), cadenceAt(at))
		}
		if v.ReadoutShown() != before {
			t.Errorf("the key reached the viewer from the %v screen", screen)
		}
	}
}

func TestTheReadoutKeyMovesNothingElse(t *testing.T) {
	a, v, at := readoutKeyViewer(t)
	v.SetGrid(true)
	cam := v.Camera()
	before := [3]float64{cam.X, cam.Y, cam.Zoom}
	counter := v.AnimationCounter()
	period := v.anim.Period()

	at++
	a.step(readoutKeyFrame(true), cadenceAt(at))

	if got := [3]float64{cam.X, cam.Y, cam.Zoom}; got != before {
		t.Errorf("the key moved the camera from %v to %v", before, got)
	}
	if len(v.sel) != 0 {
		t.Error("the key changed the selection")
	}
	if !v.GridOverlay() {
		t.Error("the readout's key moved the lattice; the two registers are separate keys")
	}
	if v.anim.Period() != period {
		t.Errorf("the key re-rated a clock: %d us became %d us", period, v.anim.Period())
	}
	// The water counter advances from the frame's own elapsed time, which is
	// this key's business not at all — what matters is that it was not reset.
	if v.AnimationCounter() < counter {
		t.Error("the key rewound the water counter")
	}
}
