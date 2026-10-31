package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestWindowRowsHoldEachProgramsForm(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  WindowRow
		want WindowRow
	}{
		{"game", GameWindow, WindowRow{Resizable: true, MonitorAware: true}},
		{"editor", EditorWindow, WindowRow{W: 1280, H: 800, MinW: 640, MinH: 480, Resizable: true}},
		{"viewer", ViewerWindow, WindowRow{W: 1024, H: 768, Resizable: true}},
	} {
		if tc.got != tc.want {
			t.Errorf("%s row = %+v, want %+v", tc.name, tc.got, tc.want)
		}
	}
}

func TestApplyWindowRowSetsSizeLimitsAndResizing(t *testing.T) {
	w, h := ebiten.WindowSize()
	minW, minH, maxW, maxH := ebiten.WindowSizeLimits()
	resizing := ebiten.WindowResizingMode()
	t.Cleanup(func() {
		ebiten.SetWindowSizeLimits(minW, minH, maxW, maxH)
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowResizingMode(resizing)
	})
	for _, tc := range []struct {
		name string
		row  WindowRow
		mode ebiten.WindowResizingModeType
	}{
		{"editor", EditorWindow, ebiten.WindowResizingModeEnabled},
		{"fixed", WindowRow{W: 700, H: 500}, ebiten.WindowResizingModeDisabled},
	} {
		ApplyWindowRow(tc.row)
		if gw, gh := ebiten.WindowSize(); gw != tc.row.W || gh != tc.row.H {
			t.Errorf("%s: window %dx%d, want %dx%d", tc.name, gw, gh, tc.row.W, tc.row.H)
		}
		if got := ebiten.WindowResizingMode(); got != tc.mode {
			t.Errorf("%s: resizing mode %v, want %v", tc.name, got, tc.mode)
		}
	}
	ApplyWindowRow(EditorWindow)
	if a, b, c, d := ebiten.WindowSizeLimits(); a != 640 || b != 480 || c != -1 || d != -1 {
		t.Errorf("editor limits = %d,%d,%d,%d, want 640,480,-1,-1", a, b, c, d)
	}
}
