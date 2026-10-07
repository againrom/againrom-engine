package ui

// Characterization pin for Viewer.Update, written BEFORE the method is
// restructured and modified by nothing afterwards.
//
// The standalone viewer's observable behaviour is contractually frozen while a
// second entry point is built on the same camera, and nothing in this repo
// pinned Update until now. It was never pinned because it looked unreachable
// without a window -- but it is not: every Ebitengine global Update reads
// (inpututil.IsKeyJustPressed, ebiten.IsKeyPressed, ebiten.CursorPosition,
// ebiten.Wheel) returns a deterministic zero value with no graphics context.
//
// That zero state is itself meaningful rather than inert. The cursor reads
// (0,0), which is inside the view and within the edge margin on both axes, so a
// headless Update edge-scrolls up and left by exactly PanSpeed. That is the
// shipped behaviour this file freezes: not "Update does nothing", but "Update
// applies the edge-scroll the input state implies, and nothing else".

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// pinViewer builds a viewer over a world far larger than the view, so the camera
// can move freely and a clamp cannot mask a missing pan.
func pinViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("pin", grid(400, 400), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 800, 600)
	return v
}

// TestUpdatePinnedBehaviour freezes what Update does today, so the restructuring
// that puts one camera step behind both entry points cannot change it unnoticed.
func TestUpdatePinnedBehaviour(t *testing.T) {
	t.Run("each call edge-scrolls by exactly PanSpeed on both axes", func(t *testing.T) {
		v := pinViewer(t)

		// Away from the clamp bounds in every direction, so movement is bounded
		// by nothing but the pan itself.
		v.Camera().X, v.Camera().Y = 500, 500
		v.Camera().Clamp()
		startX, startY := v.Camera().X, v.Camera().Y

		for i := 1; i <= 3; i++ {
			if err := v.Update(); err != nil {
				t.Fatalf("Update #%d = %v, want nil", i, err)
			}
			wantX := startX - float64(i)*PanSpeed
			wantY := startY - float64(i)*PanSpeed
			if v.Camera().X != wantX || v.Camera().Y != wantY {
				t.Fatalf("after %d Update calls camera = (%v,%v), want (%v,%v)",
					i, v.Camera().X, v.Camera().Y, wantX, wantY)
			}
		}
	})

	t.Run("Update reports no error and does not terminate", func(t *testing.T) {
		v := pinViewer(t)
		for i := 0; i < 5; i++ {
			if err := v.Update(); err != nil {
				if err == ebiten.Termination {
					t.Fatalf("Update returned Termination with no Esc pressed")
				}
				t.Fatalf("Update = %v, want nil", err)
			}
		}
	})

	t.Run("the zoom is untouched with no wheel input", func(t *testing.T) {
		v := pinViewer(t)
		before := v.Camera().Zoom
		for i := 0; i < 3; i++ {
			if err := v.Update(); err != nil {
				t.Fatalf("Update: %v", err)
			}
		}
		if v.Camera().Zoom != before {
			t.Errorf("Zoom = %v after three Updates, want it unchanged at %v", v.Camera().Zoom, before)
		}
	})

	t.Run("the first call only establishes the animation baseline", func(t *testing.T) {
		v := pinViewer(t)
		if got := v.AnimationCounter(); got != 0 {
			t.Fatalf("a fresh viewer's animation counter = %d, want 0", got)
		}
		// A slow startup must not fire a burst of ticks: the first Update takes
		// the baseline timestamp and advances nothing, and the calls that follow
		// it in a test measure microseconds, far below one tick.
		for i := 0; i < 3; i++ {
			if err := v.Update(); err != nil {
				t.Fatalf("Update: %v", err)
			}
		}
		if got := v.AnimationCounter(); got != 0 {
			t.Errorf("animation counter = %d after three immediate Updates, want 0", got)
		}
	})

	t.Run("the camera stays clamped inside the world", func(t *testing.T) {
		v := pinViewer(t)
		// Start at the top-left corner: the edge-scroll pushes further up and
		// left, and the clamp must hold the view inside the world rather than let
		// it walk off.
		v.Camera().X, v.Camera().Y = 0, 0
		for i := 0; i < 5; i++ {
			if err := v.Update(); err != nil {
				t.Fatalf("Update: %v", err)
			}
			if v.Camera().X < 0 || v.Camera().Y < 0 {
				t.Fatalf("camera left the world at (%v,%v)", v.Camera().X, v.Camera().Y)
			}
		}
		if v.Camera().X != 0 || v.Camera().Y != 0 {
			t.Errorf("camera = (%v,%v) at the clamp bound, want (0,0)", v.Camera().X, v.Camera().Y)
		}
	})
}
