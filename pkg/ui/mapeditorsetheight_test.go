package ui

import (
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// setHeightEditor is a brush editor on the Set height row with cell (20,20)
// at height 37, and the window point of that cell.
func setHeightEditor(t *testing.T) (*MapEditor, *brushProbe, image.Point) {
	t.Helper()
	e, p := brushEditor(t)
	alt := make([]uint8, e.doc.Width*e.doc.Height)
	alt[20*e.doc.Width+20] = 37
	e.doc.Viewer.grid.Altitudes = alt
	e.ShowCell(image.Pt(20, 20))
	_ = e.Key("height")
	_ = e.Key("down")
	_ = e.Key("down")
	if !e.settingHeight() {
		t.Fatal("the third height row is not Set height")
	}
	return e, p, e.CellPoint(image.Pt(20, 20))
}

func pointerAt(e *MapEditor, at image.Point, down bool, tick int64) {
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: down}, time.Unix(tick, 0))
}

// Press takes the cell's height as the anchor, the drag moves the target up
// and down and shows it on the status line and beside the pointer, and the
// release is one stroke that sets the anchor cell to the target.
func TestSetHeightDragMakesOneStroke(t *testing.T) {
	e, p, at := setHeightEditor(t)
	pointerAt(e, at, true, 1)
	g := e.brush.set
	if g == nil || g.base != 37 || g.anchor != image.Pt(20, 20) {
		t.Fatalf("gesture %+v", g)
	}
	up := image.Pt(at.X, at.Y-10)
	pointerAt(e, up, true, 2)
	want := 37 + g.pressY - g.pointer.Y
	if want <= 37 || g.target != want {
		t.Fatalf("target %d after a drag up, want %d", g.target, want)
	}
	if s := e.statusText(); !strings.Contains(s, "Height -27, target ") {
		t.Fatalf("status %q", s)
	}
	if l, ok := e.heightDragLabel(); !ok || l.text != strconv.Itoa(g.target-HeightLabelZero) {
		t.Fatal("no target beside the pointer")
	}
	if len(p.strokes) != 0 {
		t.Fatal("a stroke was written before the release")
	}
	pointerAt(e, up, false, 3)
	if len(p.strokes) != 1 {
		t.Fatalf("%d strokes", len(p.strokes))
	}
	s := p.strokes[0]
	if s.Plane != BrushPlaneHeight || !s.Set || s.Value != want || len(s.Cells) != 1 || s.Cells[0] != image.Pt(20, 20) || s.Radius != brushDefaultRadius {
		t.Fatalf("stroke %+v", s)
	}
	if e.brush.set != nil || e.brush.down {
		t.Fatal("the gesture outlived the release")
	}
	if _, ok := e.heightDragLabel(); ok {
		t.Fatal("target label after the release")
	}
}

// The target stays within 0..127 however far the pointer is dragged.
func TestSetHeightTargetClamps(t *testing.T) {
	e, p, at := setHeightEditor(t)
	for _, c := range []struct {
		to   image.Point
		want int
	}{{image.Pt(at.X, at.Y-5000), 127}, {image.Pt(at.X, at.Y+5000), 0}} {
		pointerAt(e, at, true, 1)
		pointerAt(e, c.to, true, 2)
		pointerAt(e, c.to, false, 3)
		if s := p.strokes[len(p.strokes)-1]; s.Value != c.want {
			t.Fatalf("drag to %v set %d, want %d", c.to, s.Value, c.want)
		}
		// The probe's document replaces the viewer; give it back the height.
		e.doc.Viewer.grid.Altitudes = make([]uint8, e.doc.Width*e.doc.Height)
		e.doc.Viewer.grid.Altitudes[20*e.doc.Width+20] = 37
		e.ShowCell(image.Pt(20, 20))
		at = e.CellPoint(image.Pt(20, 20))
	}
}

// Esc while dragging cancels with nothing written, and the held button then
// starts nothing.
func TestSetHeightEscapeWritesNothing(t *testing.T) {
	e, p, at := setHeightEditor(t)
	pointerAt(e, at, true, 1)
	pointerAt(e, image.Pt(at.X, at.Y-10), true, 2)
	_ = e.Key("escape")
	if e.brush.set != nil || e.brush.down {
		t.Fatal("Esc left the gesture")
	}
	pointerAt(e, image.Pt(at.X, at.Y-20), true, 3)
	pointerAt(e, image.Pt(at.X, at.Y-20), false, 4)
	if len(p.strokes) != 0 {
		t.Fatalf("%d strokes after Esc", len(p.strokes))
	}
	if e.brush.mode != brushHeight {
		t.Fatal("Esc left the height brush")
	}
}

// The fourth height row is Sample height: the press takes the pressed cell's
// height, the status line shows it as the target while the button is held,
// and the release is one stroke that sets every cell the brush covers along
// the path to it. A press on another cell samples that cell. Set height
// keeps its one-anchor stroke beside it.
func TestSampleHeightPaintsThePressedHeightAlongTheDrag(t *testing.T) {
	e, p, at := setHeightEditor(t)
	_ = e.Key("down")
	if !e.samplingHeight() || e.settingHeight() {
		t.Fatal("the fourth height row is not Sample height")
	}
	if rows := e.brushRows(); len(rows) != 4 || rows[3] != "Sample height" {
		t.Fatalf("height rows %q", rows)
	}
	pointerAt(e, at, true, 1)
	for x := 21; x <= 24; x++ {
		pointerAt(e, e.CellPoint(image.Pt(x, 20)), true, int64(x))
	}
	if s := e.statusText(); !strings.Contains(s, "target -27") {
		t.Fatalf("status %q does not show the sample", s)
	}
	if len(p.strokes) != 0 {
		t.Fatal("a stroke was written before the release")
	}
	pointerAt(e, e.CellPoint(image.Pt(24, 20)), false, 30)
	if len(p.strokes) != 1 {
		t.Fatalf("%d strokes", len(p.strokes))
	}
	s := p.strokes[0]
	want := []image.Point{{20, 20}, {21, 20}, {22, 20}, {23, 20}, {24, 20}}
	if s.Plane != BrushPlaneHeight || !s.Set || !s.Flat || s.Value != 37 || s.Radius != brushDefaultRadius || !slices.Equal(s.Cells, want) {
		t.Fatalf("stroke %+v", s)
	}
	if e.brush.sampled || e.brush.down {
		t.Fatal("the stroke outlived the release")
	}
	// The probe's document replaced the viewer; cell (18,22) is at 0.
	e.ShowCell(image.Pt(20, 20))
	other := e.CellPoint(image.Pt(18, 22))
	pointerAt(e, other, true, 31)
	pointerAt(e, other, false, 32)
	if s := p.strokes[len(p.strokes)-1]; len(p.strokes) != 2 || s.Value != 0 || !s.Flat || len(s.Cells) != 1 {
		t.Fatalf("strokes %+v", p.strokes)
	}
	_ = e.Key("up")
	pointerAt(e, other, true, 33)
	pointerAt(e, other, false, 34)
	if s := p.strokes[len(p.strokes)-1]; len(p.strokes) != 3 || s.Flat || !s.Set {
		t.Fatalf("Set height after Sample height wrote %+v", s)
	}
}

// Raise and lower keep their step.
func TestRaiseAndLowerAreUnchangedBesideSetHeight(t *testing.T) {
	e, p, at := setHeightEditor(t)
	_ = e.Key("up")
	_ = e.Key("up")
	pointerAt(e, at, true, 1)
	pointerAt(e, at, false, 2)
	if len(p.strokes) != 1 || p.strokes[0].Set || p.strokes[0].Value != brushHeightStep {
		t.Fatalf("strokes %+v", p.strokes)
	}
}
