package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/ui"
)

func TestStarterWindowRowHoldsItsForm(t *testing.T) {
	if want := (ui.WindowRow{W: 760, H: 760}); starterWindow != want {
		t.Fatalf("starterWindow = %+v, want %+v", starterWindow, want)
	}
	w, h := ebiten.WindowSize()
	resizing := ebiten.WindowResizingMode()
	t.Cleanup(func() {
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowResizingMode(resizing)
	})
	ui.ApplyWindowRow(starterWindow)
	if gw, gh := ebiten.WindowSize(); gw != 760 || gh != 760 {
		t.Errorf("window %dx%d, want 760x760", gw, gh)
	}
	if got := ebiten.WindowResizingMode(); got != ebiten.WindowResizingModeDisabled {
		t.Errorf("resizing mode %v, want disabled", got)
	}
}
