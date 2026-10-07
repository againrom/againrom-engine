package ui

import (
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
)

func TestRetreatBindingIsCtrlWAndPlainFStaysGrab(t *testing.T) {
	bindings := bindingSource(t)
	retreat, ok := bindings["Retreat"]
	if !ok {
		t.Fatal("readAppInput has no Retreat binding")
	}
	if !strings.Contains(retreat, "inpututil.IsKeyJustPressed(ebiten.KeyW)") ||
		!strings.Contains(retreat, "ctrlHeld()") || strings.Contains(retreat, "!ctrlHeld()") {
		t.Errorf("Retreat binding = %q, want Ctrl+W press edge", retreat)
	}
	grab, ok := bindings["Grab"]
	if !ok {
		t.Fatal("readAppInput has no Grab binding")
	}
	if !strings.Contains(grab, "inpututil.IsKeyJustPressed(ebiten.KeyF)") ||
		!strings.Contains(grab, "!ctrlHeld()") {
		t.Errorf("Grab binding = %q, want plain F with Ctrl excluded", grab)
	}
}

func TestRetreatPressLeavesTheMapOnceAndAHeldFrameDoesNotRepeatIt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := fiViewer(t)
	calls := 0
	v.SetRetreatSink(func() { calls++ })
	a := NewApp("retreat", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	a.flow.viewer, a.flow.screen = v, ScreenMap

	a.step(appInput{Retreat: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("first Ctrl+W edge issued %d commands, want 1", calls)
	}
	a.step(appInput{Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("held frame issued %d commands total, want 1", calls)
	}
	a.step(appInput{Retreat: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 2 {
		t.Fatalf("second press edge issued %d commands total, want 2", calls)
	}
}

func TestRetreatPressObeysPopupTownAndStoppedMapBoundaries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := fiViewer(t)
	calls := 0
	v.SetRetreatSink(func() { calls++ })
	a := NewApp("retreat-boundaries", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	a.flow.viewer, a.flow.screen = v, ScreenMap

	// A stopped map accepts the ordinary setting command into the far-side
	// queue; simulation applies it after resume.
	a.step(appInput{Pause: true, Retreat: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("stopped map issued %d retreat commands, want 1", calls)
	}

	v.SetNotice("modal text", NoticeDialogue)
	a.step(appInput{Retreat: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("modal map issued %d retreat commands total, want 1", calls)
	}
	v.ClearNotice()

	a.flow.screen = ScreenTown
	a.step(appInput{Retreat: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("town issued %d retreat commands total, want 1", calls)
	}
}
