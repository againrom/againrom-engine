package ui

import (
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
)

func TestFormationBindingIsCtrlFOnAPressEdge(t *testing.T) {
	expr, ok := bindingSource(t)["Formation"]
	if !ok {
		t.Fatal("readAppInput has no Formation binding")
	}
	if !strings.Contains(expr, "inpututil.IsKeyJustPressed(ebiten.KeyF)") {
		t.Errorf("Formation binding = %q, want an F press edge", expr)
	}
	if !strings.Contains(expr, "ctrlHeld()") || strings.Contains(expr, "!ctrlHeld()") {
		t.Errorf("Formation binding = %q, want Ctrl held", expr)
	}
}

func TestFormationPressLeavesTheMapOnceAndAHeldFrameDoesNotRepeatIt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := fiViewer(t)
	calls := 0
	v.SetFormationSink(func() { calls++ })
	a := NewApp("formation", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	a.flow.viewer, a.flow.screen = v, ScreenMap

	a.step(appInput{Formation: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("first Ctrl+F edge issued %d commands, want 1", calls)
	}
	// Ctrl and W remain physically held, but the snapshot carries no second
	// W edge. The command must not run at the render-frame rate.
	a.step(appInput{Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("held frame issued %d commands total, want 1", calls)
	}
	a.step(appInput{Formation: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 2 {
		t.Fatalf("second press edge issued %d commands total, want 2", calls)
	}
}

func TestFormationPressObeysPopupTownAndStoppedMapBoundaries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	v := fiViewer(t)
	calls := 0
	v.SetFormationSink(func() { calls++ })
	a := NewApp("formation-boundaries", appAssets(t), appRows(1), nil)
	a.Layout(frame.W, frame.H)
	a.flow.viewer, a.flow.screen = v, ScreenMap

	// A player-stopped map still accepts ordinary orders/settings; its far side
	// queues them until a resumed simulation tick (pkg/game's paired witness).
	a.step(appInput{Pause: true, Formation: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("stopped map issued %d formation commands, want 1", calls)
	}

	v.SetNotice("modal text", NoticeDialogue)
	a.step(appInput{Formation: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("modal map issued %d formation commands total, want 1", calls)
	}
	v.ClearNotice()

	// Town dispatch has no formation arm even if a stale viewer still exists;
	// the key belongs to the live mission map only.
	a.flow.screen = ScreenTown
	a.step(appInput{Formation: true, Viewer: Input{Ctrl: true}}, now)
	if calls != 1 {
		t.Fatalf("town issued %d formation commands total, want 1", calls)
	}
}
