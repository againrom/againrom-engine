package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

// pressHeightMode presses mode button n of the Height tool's options.
func pressHeightMode(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	s, ok := e.listSpec()
	if !ok || s.Grid == nil {
		t.Fatal("the Height tool shows no mode row")
	}
	cells := layoutPopup(s, e.font, e.bodyFont()).GridCells
	if n >= len(cells) {
		t.Fatalf("the mode row lays out %d buttons", len(cells))
	}
	passClick(e, cells[n])
}

// heightStroke presses, drags and releases along cells and answers the one
// stroke the host took.
func heightStroke(t *testing.T, e *MapEditor, p *brushProbe, cells ...image.Point) BrushStroke {
	t.Helper()
	n := len(p.strokes)
	e.ShowCell(cells[0])
	for i, c := range cells {
		pointerAt(e, e.CellPoint(c), true, int64(10+i))
	}
	pointerAt(e, e.CellPoint(cells[len(cells)-1]), false, 40)
	if len(p.strokes) != n+1 {
		t.Fatalf("%d strokes for one press, drag and release", len(p.strokes)-n)
	}
	return p.strokes[n]
}

// The Height tool's options carry the Strength row (its value with minus and
// plus) and the mode row (a picture button per mode: Step, Level, Smooth).
// Step and strength 8 are the defaults; the strength stays in 1..31 and,
// with the mode, holds while another tool is used; the list title names the
// mode beside the row and the strength beside the size.
func TestHeightToolModeAndStrengthRows(t *testing.T) {
	e, _ := brushEditor(t)
	_ = e.Key("height")
	strength := func() string {
		t.Helper()
		s, _ := e.listSpec()
		if len(s.Rows) < 3 || s.Rows[2].Label != "Strength" || len(s.Rows[2].Buttons) != 2 {
			t.Fatalf("the Height tool's rows %+v", s.Rows)
		}
		return s.Rows[2].Value
	}
	if v := strength(); v != "8" {
		t.Fatalf("strength %s, want 8", v)
	}
	g, ok := e.heightModeGridSpec()
	if !ok || len(g.Cells) != 3 || len(g.RowHeads) != 1 || g.RowHeads[0] != "Mode" || !g.Cells[0].Pressed || g.Cells[1].Pressed || g.Cells[2].Pressed {
		t.Fatalf("mode row %+v", g)
	}
	for i, c := range g.Cells {
		if c.Icon == nil || c.Icon.Bounds().Empty() {
			t.Fatalf("mode %d has no picture", i)
		}
		for j := range i {
			if reflect.DeepEqual(c.Icon.Pix, g.Cells[j].Icon.Pix) {
				t.Fatalf("modes %d and %d share a picture", j, i)
			}
		}
	}
	if title := e.brushStatus(); title != "Height: Raise  Step  size 2  strength 8" {
		t.Fatalf("title %q", title)
	}
	for range 10 {
		pressBrushButton(t, e, 2, 0)
	}
	if v := strength(); v != "1" {
		t.Fatalf("strength %s after ten minus presses, want the bound 1", v)
	}
	for range 40 {
		pressBrushButton(t, e, 2, 1)
	}
	if v := strength(); v != "31" {
		t.Fatalf("strength %s after forty plus presses, want the bound 31", v)
	}
	for n, word := range []string{"Level", "Smooth", "Step"} {
		pressHeightMode(t, e, (n+1)%3)
		if title := e.brushStatus(); !strings.Contains(title, "  "+word+"  ") {
			t.Fatalf("title %q after mode %d", title, (n+1)%3)
		}
	}
	pressHeightMode(t, e, 1)
	_ = e.Key("terrain")
	if s, _ := e.listSpec(); len(s.Rows) > 2 && s.Rows[2].Label == "Strength" || s.Grid != nil {
		t.Fatal("the Terrain tool shows the Height tool's rows")
	}
	_ = e.Key("height")
	if v := strength(); v != "31" || e.brush.heightMode != HeightModeLevel {
		t.Fatalf("after another tool: strength %s, mode %d", v, e.brush.heightMode)
	}
	_ = e.Key("down")
	_ = e.Key("down")
	if title := e.brushStatus(); strings.Contains(title, "strength") || strings.Contains(title, "Level") {
		t.Fatalf("the Set height title %q names what the row does not read", title)
	}
}

// Each mode sends its stroke: Raise the strength, Lower its negative, the
// chosen mode, every visited cell; one press and release is one stroke.
func TestHeightModesSendTheirStroke(t *testing.T) {
	e, p := brushEditor(t)
	_ = e.Key("height")
	for range 3 {
		pressBrushButton(t, e, 2, 0)
	}
	path := []image.Point{{X: 20, Y: 20}, {X: 21, Y: 20}, {X: 22, Y: 20}, {X: 21, Y: 20}}
	for _, m := range []BrushHeightMode{HeightModeStep, HeightModeLevel, HeightModeSmooth} {
		pressHeightMode(t, e, int(m))
		for row, value := range []int{5, -5} {
			e.brush.sel[brushHeight] = row
			s := heightStroke(t, e, p, path...)
			if s.Plane != BrushPlaneHeight || s.Value != value || s.Mode != m || s.Set || s.Flat || len(s.Cells) != len(path) {
				t.Fatalf("mode %d row %d: stroke %+v", m, row, s)
			}
		}
	}
}

// While a Level stroke is held the status line shows the pressed cell's
// height plus the strength (minus it for Lower) as Set height shows its
// target.
func TestLevelStrokeShowsItsTarget(t *testing.T) {
	held := func(mode BrushHeightMode, row int) string {
		e, _ := brushEditor(t)
		alt := make([]uint8, e.doc.Width*e.doc.Height)
		alt[20*e.doc.Width+20] = 37
		e.doc.Viewer.grid.Altitudes = alt
		e.ShowCell(image.Pt(20, 20))
		_ = e.Key("height")
		pressHeightMode(t, e, int(mode))
		e.brush.sel[brushHeight] = row
		pointerAt(e, e.CellPoint(image.Pt(20, 20)), true, 1)
		pointerAt(e, e.CellPoint(image.Pt(21, 20)), true, 2)
		return e.statusText()
	}
	for row, want := range []string{"target -19", "target -35"} {
		if s := held(HeightModeLevel, row); !strings.Contains(s, want) {
			t.Fatalf("row %d: status %q, want %q", row, s, want)
		}
	}
	if s := held(HeightModeStep, 0); strings.Contains(s, "target") {
		t.Fatalf("a Step stroke shows a target: %q", s)
	}
}

// Control: Set height and Sample height read neither row; with another mode
// and strength they send the strokes they send with the defaults.
func TestSetAndSampleHeightReadNeitherRow(t *testing.T) {
	strokes := func(level bool) []BrushStroke {
		e, p, at := setHeightEditor(t)
		if level {
			pressHeightMode(t, e, int(HeightModeLevel))
			for range 4 {
				pressBrushButton(t, e, 2, 1)
			}
		}
		pointerAt(e, at, true, 1)
		pointerAt(e, image.Pt(at.X, at.Y-10), true, 2)
		pointerAt(e, image.Pt(at.X, at.Y-10), false, 3)
		_ = e.Key("down")
		e.ShowCell(image.Pt(20, 20))
		heightStroke(t, e, p, image.Pt(20, 20), image.Pt(21, 20))
		return p.strokes
	}
	plain, moved := strokes(false), strokes(true)
	if len(plain) != 2 || !plain[0].Set || plain[0].Flat || !plain[1].Flat || !reflect.DeepEqual(plain, moved) {
		t.Fatalf("defaults %+v, Level and strength 12 %+v", plain, moved)
	}
}
