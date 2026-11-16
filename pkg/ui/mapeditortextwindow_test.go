package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/catmullrom"
)

// At a device scale other than 1 the editor's smoothed text is handed to the
// overlay at device resolution after the scaler: the points frame the scaler
// resamples carries none of it. At scale 1, or while a frame read is pending,
// the overlay still draws onto the frame itself.
func TestEditorTextReachesTheOverlayNotTheScaledFrame(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.Layout(1280, 720)
	e.textSmoothingEnabled = true
	frame := ebiten.NewImage(1280, 720)

	e.deviceScale = 2
	e.drawPoints(frame)
	if len(e.windowText) == 0 || e.textKept == 0 {
		t.Fatalf("at scale 2 the window text holds %d calls of %d kept", len(e.windowText), e.textKept)
	}
	if !e.textAtWindow() {
		t.Fatal("scale 2 draws the text on the points frame")
	}

	e.deviceScale = 1
	e.drawPoints(frame)
	if len(e.windowText) != 0 || e.textAtWindow() {
		t.Fatalf("scale 1 hands %d calls to the window overlay", len(e.windowText))
	}

	e.deviceScale = 0
	e.drawPoints(frame)
	if len(e.windowText) != 0 {
		t.Fatalf("no device scale hands %d calls to the window overlay", len(e.windowText))
	}
}

func TestWindowKernelDefaultsToCatmullRom(t *testing.T) {
	t.Cleanup(func() { SetWindowKernel(false) })
	if windowKernel != catmullrom.CatmullRom {
		t.Fatalf("default window kernel %v", windowKernel)
	}
	SetWindowKernel(true)
	if windowKernel != catmullrom.Mitchell {
		t.Fatalf("soft window kernel %v", windowKernel)
	}
	SetWindowKernel(false)
	if windowKernel != catmullrom.CatmullRom {
		t.Fatalf("window kernel after reset %v", windowKernel)
	}
}
