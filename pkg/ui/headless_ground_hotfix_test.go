package ui

import "testing"

func TestHeadlessGroundPointDoesNotArmEdgeScrolling(t *testing.T) {
	for _, size := range [][2]int{{640, 360}, {800, 600}, {1600, 1200}, {2400, 1200}} {
		a := newTestApp(t, appRows(0), okLoader(t))
		if err := a.OpenMission(okOpener(t)); err != nil {
			t.Fatal(err)
		}
		a.Layout(size[0], size[1])
		x, y, err := a.HeadlessGroundPoint()
		if err != nil {
			t.Fatal(err)
		}
		if x < EdgeMargin || x >= size[0]-EdgeMargin || y < EdgeMargin || y >= size[1]-EdgeMargin {
			t.Fatalf("window%v ground(%d,%d) lies in an edge-scroll band", size, x, y)
		}
		v := a.flow.viewer
		fx, fy := v.windowToFrame(x, y)
		if v.groundSurfaceCaptures(fx, fy) {
			t.Fatal("ground point moved onto a HUD surface")
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		if left, right, top, bottom := v.edgeScrollBands(false); left || right || top || bottom {
			t.Fatal("ground pointer armed production edge scrolling")
		}
	}
}

func TestHeadlessIdleAndKeysKeepObservedWindowPointer(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	a.Layout(1600, 1200)
	v := a.flow.viewer
	if in := a.headlessIdleInput(); in.CursorX != -1 || in.CursorY != -1 || in.Viewer.CursorX != -1 || in.Viewer.CursorY != -1 {
		t.Fatalf("unobserved pointer became a window input: %+v", in)
	}
	// Include explicit origin and pointer leave, not only a safe middle point.
	for _, p := range [][2]int{{500, 400}, {0, 0}, {-1, -1}} {
		if err := a.HeadlessPointer("hover", p[0], p[1]); err != nil {
			t.Fatal(err)
		}
		left, right, top, bottom := v.edgeScrollBands(false)
		for i := 0; i < 8; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if v.winCursorX != p[0] || v.winCursorY != p[1] {
				t.Fatalf("idle moved pointer %v to (%d,%d)", p, v.winCursorX, v.winCursorY)
			}
		}
		if err := a.HeadlessKey("space"); err != nil {
			t.Fatal(err)
		}
		if v.winCursorX != p[0] || v.winCursorY != p[1] {
			t.Fatalf("key moved pointer %v to (%d,%d)", p, v.winCursorX, v.winCursorY)
		}
		l, r, u, d := v.edgeScrollBands(false)
		if l != left || r != right || u != top || d != bottom {
			t.Fatalf("idle/key changed edge intent for %v", p)
		}
		if p == [2]int{0, 0} && (!l || !u) {
			t.Fatal("explicit origin must still request left/up edge scrolling")
		}
	}
}
