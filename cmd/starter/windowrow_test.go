package main

import (
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/ui/widgetkit"
)

// skipWindowSystemOffMainThread skips a test that reaches the window system on
// macOS, where the window system starts only on the main thread and a test
// runs on another; the starter itself calls it from main.
func skipWindowSystemOffMainThread(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS starts the window system only on the main thread")
	}
}

func TestStarterWindowRowHoldsItsForm(t *testing.T) {
	skipWindowSystemOffMainThread(t)
	if want := (widgetkit.WindowRow{W: 760, H: 760}); starterWindow != want {
		t.Fatalf("starterWindow = %+v, want %+v", starterWindow, want)
	}
	w, h := ebiten.WindowSize()
	resizing := ebiten.WindowResizingMode()
	t.Cleanup(func() {
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowResizingMode(resizing)
	})
	widgetkit.ApplyWindowRow(starterWindow)
	if gw, gh := ebiten.WindowSize(); gw != 760 || gh != 760 {
		t.Errorf("window %dx%d, want 760x760", gw, gh)
	}
	if got := ebiten.WindowResizingMode(); got != ebiten.WindowResizingModeDisabled {
		t.Errorf("resizing mode %v, want disabled", got)
	}
}
