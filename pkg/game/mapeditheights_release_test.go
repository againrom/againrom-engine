package game

import (
	"bytes"
	"image"
	"slices"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// The height brush's Set height row on the installed map: a press on a cell,
// a drag up and a release write one edit that sets the cell and blends the
// cells within the brush size toward it and no others; one undo restores the
// bytes. Frames of the drag and of the result for the owner.
func TestReleaseMapEditorSetHeightEdit(t *testing.T) {
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/10.alm")
	cell := image.Pt(20, 20)
	e.ShowCell(cell)
	_ = e.Key("height")
	_ = e.Key("down")
	_ = e.Key("down")
	for range 3 {
		_ = e.Key("radiusup")
	}
	const radius = 5
	before := slices.Clone(e.Document().SourceBytes)
	was := shownMap(t, e)
	w := int(was.Width)
	base := was.Altitudes[cell.Y*w+cell.X]
	at := e.CellPoint(cell)
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, now)
	up := image.Pt(at.X, at.Y-30)
	e.Pointer(ui.Input{CursorX: up.X, CursorY: up.Y, PrimaryDown: true}, now)
	terrainFrame(t, x, e, "set-height-drag")
	if bytes.Equal(before, e.Document().SourceBytes) == false {
		t.Fatal("the map changed before the release")
	}
	e.Pointer(ui.Input{CursorX: up.X, CursorY: up.Y}, now)
	got := shownMap(t, e)
	target := got.Altitudes[cell.Y*w+cell.X]
	if target <= base || target > 127 {
		t.Fatalf("the cell went from %d to %d after a drag up", base, target)
	}
	for i := range got.Altitudes {
		x, y := i%w-cell.X, i/w-cell.Y
		if x*x+y*y > radius*radius && got.Altitudes[i] != was.Altitudes[i] {
			t.Fatalf("height at (%d,%d) changed outside the brush radius", i%w, i/w)
		}
	}
	terrainFrame(t, x, e, "set-height-after")
	_ = e.Key("undo")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("one undo did not restore the map")
	}
}

// The height numbers zoomed out stand on every k-th vertex instead of
// vanishing; zoomed in they stand on every vertex. A frame of the zoomed-out
// state for the owner.
func TestReleaseMapEditorHeightNumbersZoomedOut(t *testing.T) {
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/10.alm")
	e.ShowCell(image.Pt(20, 20))
	_ = e.Key("heightlabels")
	cam := e.Document().Viewer.Camera()
	cam.SetZoom(2)
	e.ShowCell(image.Pt(20, 20))
	in, _ := e.HeightLabels()
	cam.SetZoom(0.4)
	e.ShowCell(image.Pt(20, 20))
	out, _ := e.HeightLabels()
	if len(in) == 0 || len(out) == 0 {
		t.Fatalf("%d numbers zoomed in, %d zoomed out", len(in), len(out))
	}
	terrainFrame(t, x, e, "heights-zoomed-out")
}
