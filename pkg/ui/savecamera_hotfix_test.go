package ui

import (
	"image"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/camera"
)

// A saved camera has to survive the opening start view. Every mission open arms
// one, including a LOAD, and the first Layout applies it, so a camera placed
// before that Layout and not disarmed is replaced by the map's own start
// position one frame later -- which is the reopening behaviour this hotfix is
// about.
func TestRestoreCameraSurvivesTheOpeningStartView(t *testing.T) {
	saved := applicationViewer(t)
	saved.Camera().X, saved.Camera().Y = 20*camera.CellSize, 24*camera.CellSize
	want := saved.SaveApplication()
	if want.Zoom <= 0 {
		t.Fatalf("fixture camera is not restorable: %+v", want)
	}

	// The start position of the same map, measured on its own.
	opened := applicationViewer(t)
	opened.SetStartView(image.Pt(60, 60))
	opened.Layout(1024, 768)
	start := opened.SaveApplication()
	if start.ViewX == 0 && start.ViewY == 0 {
		t.Fatal("the armed start view moved no camera, so this fixture cannot witness the defect")
	}
	if start.ViewX == want.ViewX && start.ViewY == want.ViewY {
		t.Fatalf("start position already equals the saved camera: %+v", start)
	}

	reopened := applicationViewer(t)
	reopened.SetStartView(image.Pt(60, 60))
	if err := reopened.RestoreCamera(want.ViewX, want.ViewY, want.Zoom); err != nil {
		t.Fatal(err)
	}
	reopened.Layout(1024, 768)
	got := reopened.SaveApplication()
	if got.ViewX != want.ViewX || got.ViewY != want.ViewY || got.Zoom != want.Zoom {
		t.Fatalf("reopened camera = (%v, %v, %v), want (%v, %v, %v)",
			got.ViewX, got.ViewY, got.Zoom, want.ViewX, want.ViewY, want.Zoom)
	}
}

func TestRestoreCameraMovesNothingElse(t *testing.T) {
	v := applicationViewer(t)
	v.selectAllOwnedUnits()
	v.toggleHudPanel(hudPanelPack)
	v.ToggleShowHealth()
	want := v.SaveApplication()
	if err := v.RestoreCamera(21, 22, 1); err != nil {
		t.Fatal(err)
	}
	want.ViewX, want.ViewY, want.Zoom = 21, 22, 1
	if got := v.SaveApplication(); !reflect.DeepEqual(got, want) {
		t.Fatalf("camera restore changed the application: %+v, want %+v", got, want)
	}
}

// A camera is where the player was looking, not game state. A value this build
// cannot restore keeps the map's own opening view; it never stops a map opening.
func TestRestoreCameraRefusesWithoutCostingTheOpeningView(t *testing.T) {
	armed := applicationViewer(t)
	armed.SetStartView(image.Pt(60, 60))
	armed.Layout(1024, 768)
	start := armed.SaveApplication()

	for _, test := range []struct {
		name    string
		x, y, z float64
	}{
		{"nonfinite origin", math.NaN(), 0, 1},
		{"infinite zoom", 0, 0, math.Inf(1)},
		{"absent camera", 0, 0, 0},
		{"zoom over the ceiling", 0, 0, camera.ZoomMax * 2},
		{"origin over the ceiling", math.MaxInt32 * 2, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			v := applicationViewer(t)
			v.SetStartView(image.Pt(60, 60))
			before := v.SaveApplication()
			if err := v.RestoreCamera(test.x, test.y, test.z); err == nil {
				t.Fatal("unrestorable camera accepted")
			}
			if got := v.SaveApplication(); !reflect.DeepEqual(got, before) {
				t.Fatalf("refused camera changed the viewer: %+v", got)
			}
			v.Layout(1024, 768)
			if got := v.SaveApplication(); got.ViewX != start.ViewX || got.ViewY != start.ViewY || got.Zoom != start.Zoom {
				t.Fatalf("refused camera cost the map its opening view: %+v, want %+v", got, start)
			}
		})
	}
	var absent *Viewer
	if err := absent.RestoreCamera(1, 1, 1); err == nil {
		t.Fatal("camera restored into no viewer")
	}
}
