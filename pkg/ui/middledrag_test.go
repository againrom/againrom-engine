package ui

import (
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// middleDrag holds the middle button over the given frame points, one tick
// each, and returns the camera's movement.
func middleDrag(v *Viewer, points ...[2]int) (dx, dy float64) {
	x0, y0 := v.Camera().X, v.Camera().Y
	for _, p := range points {
		v.step(Input{MiddleDown: true, CursorX: p[0], CursorY: p[1]}, dragFrozen)
	}
	return v.Camera().X - x0, v.Camera().Y - y0
}

// TestTheMiddleDragKeepsThePointUnderThePointer drags the same three-tick
// gesture at two zoom levels on the mission map. The pointer travels (+20,-10)
// frame pixels; the camera moves the opposite way by that distance divided by
// the zoom, so the world point under the pointer at the press is under it at
// the end.
func TestTheMiddleDragKeepsThePointUnderThePointer(t *testing.T) {
	for _, tc := range []struct {
		zoom   float64
		dx, dy float64
	}{
		{1, -20, +10},
		{2, -10, +5},
	} {
		v := stepViewer(t)
		v.commandMode = true
		v.Camera().SetZoom(tc.zoom)
		v.Camera().Clamp()
		if got := v.Camera().Zoom; got != tc.zoom {
			t.Fatalf("fixture zoom = %v, want %v", got, tc.zoom)
		}
		wx, wy := v.Camera().ScreenToWorld(400, 300)
		dx, dy := middleDrag(v, [2]int{400, 300}, [2]int{450, 320}, [2]int{420, 290})
		if dx != tc.dx || dy != tc.dy {
			t.Errorf("zoom %v: the camera moved by (%v,%v), want (%v,%v)", tc.zoom, dx, dy, tc.dx, tc.dy)
		}
		if gx, gy := v.Camera().ScreenToWorld(420, 290); gx != wx || gy != wy {
			t.Errorf("zoom %v: world point under the pointer moved from (%v,%v) to (%v,%v)", tc.zoom, wx, wy, gx, gy)
		}
		if v.rightPanned || v.dragMoved != 0 || v.dragging || v.boxing || len(v.sel) != 0 {
			t.Errorf("zoom %v: the middle drag touched another gesture: rightPanned %t dragMoved %d dragging %t boxing %t sel %v",
				tc.zoom, v.rightPanned, v.dragMoved, v.dragging, v.boxing, v.sel)
		}
		// Releasing ends the gesture: the next press anchors afresh and its
		// own tick moves nothing.
		v.step(Input{CursorX: 420, CursorY: 290}, dragFrozen)
		if dx, dy := middleDrag(v, [2]int{100, 100}); dx != 0 || dy != 0 {
			t.Errorf("zoom %v: a fresh press resumed the old anchor and moved (%v,%v)", tc.zoom, dx, dy)
		}
	}
}

// TestTheMiddleDragIsClampedAtTheMapEdge drags toward the map's top-left
// corner from a view already there. The clamp holds both axes, as it holds
// every other scroll; the opposite drag still moves.
func TestTheMiddleDragIsClampedAtTheMapEdge(t *testing.T) {
	v := stepViewer(t)
	v.commandMode = true
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	x0, y0 := v.Camera().X, v.Camera().Y
	if dx, dy := middleDrag(v, [2]int{400, 300}, [2]int{460, 340}); dx != 0 || dy != 0 {
		t.Errorf("a drag past the edge moved the camera by (%v,%v), want (0,0)", dx, dy)
	}
	if v.Camera().X != x0 || v.Camera().Y != y0 {
		t.Fatalf("the clamp left the camera at (%v,%v), want (%v,%v)", v.Camera().X, v.Camera().Y, x0, y0)
	}
	if dx, dy := middleDrag(v, [2]int{400, 300}); dx != 60 || dy != 40 {
		t.Errorf("the drag back moved the camera by (%v,%v), want (60,40)", dx, dy)
	}
}

// TestTheMiddlePressWithoutMovementDoesNothing holds the middle button still
// for three ticks. The camera, the selection and every other gesture's state
// stay as they were.
func TestTheMiddlePressWithoutMovementDoesNothing(t *testing.T) {
	v := stepViewer(t)
	v.commandMode = true
	if dx, dy := middleDrag(v, [2]int{400, 300}, [2]int{400, 300}, [2]int{400, 300}); dx != 0 || dy != 0 {
		t.Errorf("a still middle press moved the camera by (%v,%v)", dx, dy)
	}
	if v.rightPanned || v.dragMoved != 0 || v.dragging || v.held || len(v.sel) != 0 {
		t.Error("a still middle press changed another gesture's state")
	}
}

// TestTheOtherScrollInputsAreUnchangedByTheMiddleButton runs each other
// scroll input once alone and once with the middle button held: the pan keys,
// the edge band, the right drag, the left drag of the standalone viewer and
// the wheel. Each moves the camera the same way both times, so a right drag
// with the wheel held moves the view once, not twice. The middle drag alone
// over the same travel is the loss control: it moves the view.
func TestTheOtherScrollInputsAreUnchangedByTheMiddleButton(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ticks []Input
	}{
		{"pan key", []Input{{PanLeft: true, CursorX: 400, CursorY: 300}, {PanLeft: true, CursorX: 400, CursorY: 300}}},
		{"edge band", []Input{{CursorX: 0, CursorY: 300}, {CursorX: 0, CursorY: 300}}},
		{"right drag", []Input{{SecondaryDown: true, CursorX: 400, CursorY: 300}, {SecondaryDown: true, CursorX: 450, CursorY: 330}}},
		{"standalone left drag", []Input{{PrimaryDown: true, CursorX: 400, CursorY: 300}, {PrimaryDown: true, CursorX: 450, CursorY: 330}}},
		{"wheel", []Input{{WheelY: 1, CursorX: 400, CursorY: 300}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func(middle bool) (x, y, z float64) {
				v := stepViewer(t)
				v.commandMode = tc.name != "standalone left drag"
				for _, in := range tc.ticks {
					in.MiddleDown = middle
					v.step(in, dragFrozen)
				}
				return v.Camera().X, v.Camera().Y, v.Camera().Zoom
			}
			x0, y0, z0 := run(false)
			x1, y1, z1 := run(true)
			if x0 != x1 || y0 != y1 || z0 != z1 {
				t.Fatalf("alone (%v,%v)x%v, with the middle button held (%v,%v)x%v", x0, y0, z0, x1, y1, z1)
			}
			v := stepViewer(t)
			if x0 == v.Camera().X && y0 == v.Camera().Y && z0 == v.Camera().Zoom {
				t.Fatal("the input moved nothing, so the comparison proves nothing")
			}
		})
	}
	v := stepViewer(t)
	v.commandMode = true
	if dx, dy := middleDrag(v, [2]int{400, 300}, [2]int{450, 330}); dx != -50 || dy != -30 {
		t.Fatalf("the middle drag alone moved (%v,%v), want (-50,-30)", dx, dy)
	}
}

// TestTheMiddleDragReachesTheMapThroughTheApp drives the gesture as a window
// does, through App.step on the mission map, and checks that it moves the
// camera and issues no order.
func TestTheMiddleDragReachesTheMapThroughTheApp(t *testing.T) {
	a, v, seam := affOnMap(t)
	v.Camera().X, v.Camera().Y = 200, 200
	v.Camera().Clamp()
	x0, y0 := v.Camera().X, v.Camera().Y
	for _, step := range []struct {
		action string
		x, y   int
	}{{"middle-press", 300, 200}, {"middle-move", 270, 180}, {"middle-release", 270, 180}} {
		if err := a.HeadlessPointer(step.action, step.x, step.y); err != nil {
			t.Fatal(err)
		}
	}
	if dx, dy := v.Camera().X-x0, v.Camera().Y-y0; dx != 30 || dy != 20 {
		t.Fatalf("the camera moved by (%v,%v), want (30,20)", dx, dy)
	}
	if len(seam.orders) != 0 || len(seam.attacks) != 0 || len(a.HeadlessSelection()) != 0 {
		t.Fatalf("the middle drag issued orders %v attacks %v selection %v", seam.orders, seam.attacks, a.HeadlessSelection())
	}
}

// TestTheMiddleDragPansTheMapEditor: the editor's canvas runs the same viewer
// step, so the middle drag moves its view the same way and selects nothing.
func TestTheMiddleDragPansTheMapEditor(t *testing.T) {
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "m", Label: "M"}}, func(string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", grid(200, 200), &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		return &InspectionDocument{Viewer: v, Source: "m", Width: 200, Height: 200, SourceBytes: []byte("untouched")}, nil
	})
	if err := e.Open("m"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1024, 768)
	v := e.doc.Viewer
	v.cam.SetZoom(1)
	v.cam.CenterOn(v.cam.WorldW()/2, v.cam.WorldH()/2)
	x0, y0, sel := v.cam.X, v.cam.Y, e.selected
	at := time.Unix(1, 0)
	e.Pointer(Input{MiddleDown: true, CursorX: 350, CursorY: 350}, at)
	e.Pointer(Input{MiddleDown: true, CursorX: 320, CursorY: 330}, at)
	e.Pointer(Input{CursorX: 320, CursorY: 330}, at)
	if v.cam.X <= x0 || v.cam.Y <= y0 {
		t.Fatalf("the editor view moved from (%v,%v) to (%v,%v), want right and down", x0, y0, v.cam.X, v.cam.Y)
	}
	if e.selected != sel {
		t.Fatalf("the middle drag changed the editor selection from %d to %d", sel, e.selected)
	}
}
