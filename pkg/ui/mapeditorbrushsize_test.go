package ui

import (
	"image"
	"strconv"
	"strings"
	"testing"
	"time"
)

// pressBrushButton presses button k of row row of the brush list.
func pressBrushButton(t *testing.T, e *MapEditor, row, k int) {
	t.Helper()
	s, ok := e.listSpec()
	if !ok {
		t.Fatal("no brush list")
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == row && k < len(r.Buttons) {
			passClick(e, r.Buttons[k])
			return
		}
	}
	t.Fatalf("the brush list has no button %d on row %d", k, row)
}

// wheelAt turns the wheel by notches over cell, with Alt held or not.
func wheelAt(e *MapEditor, cell image.Point, notches float64, alt bool) {
	pt := e.CellPoint(cell)
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y, WheelY: notches, Alt: alt}, time.Unix(5, 0))
}

// The list's minus and plus buttons, Alt with the wheel and the bracket keys
// all move the one size a stroke carries; the size stays in 0..the host's
// maximum. Control: the wheel without Alt zooms and leaves the size.
func TestMapEditorBrushSizeControlsMoveOneSize(t *testing.T) {
	e, p := brushEditor(t)
	_ = e.Key("terrain")
	e.ShowCell(image.Pt(30, 30))
	s, _ := e.listSpec()
	if len(s.Rows) != 2 || s.Rows[0].Value != "2" || s.Rows[1].Value != "Circle" {
		t.Fatalf("brush list rows %+v", s.Rows)
	}
	steps := []struct {
		name string
		act  func()
		want int
	}{
		{"plus", func() { pressBrushButton(t, e, 0, 1) }, 3},
		{"minus", func() { pressBrushButton(t, e, 0, 0) }, 2},
		{"Alt+wheel up", func() { wheelAt(e, image.Pt(30, 30), 1, true) }, 3},
		{"Alt+wheel down twice", func() { wheelAt(e, image.Pt(30, 30), -2, true) }, 1},
		{"half notches", func() { wheelAt(e, image.Pt(30, 30), 0.5, true); wheelAt(e, image.Pt(30, 30), 0.5, true) }, 2},
		{"]", func() { _ = e.Key("radiusup") }, 3},
		{"[", func() { _ = e.Key("radiusdown") }, 2},
	}
	for _, st := range steps {
		st.act()
		if got := e.brushRadius(); got != st.want {
			t.Fatalf("%s: size %d, want %d", st.name, got, st.want)
		}
		if s, _ := e.listSpec(); s.Rows[0].Value != strconv.Itoa(st.want) {
			t.Fatalf("%s: the list shows %q", st.name, s.Rows[0].Value)
		}
	}
	zoom := e.doc.Viewer.cam.Zoom
	wheelAt(e, image.Pt(30, 30), 1, false)
	if e.brushRadius() != 2 {
		t.Fatal("the wheel without Alt changed the size")
	}
	if e.doc.Viewer.cam.Zoom == zoom {
		t.Fatal("control: the wheel without Alt did not zoom")
	}
	for range 40 {
		pressBrushButton(t, e, 0, 1)
	}
	if e.brushRadius() != 16 {
		t.Fatalf("plus stops at %d, want the host's 16", e.brushRadius())
	}
	wheelAt(e, image.Pt(30, 30), -40, true)
	if e.brushRadius() != 0 {
		t.Fatalf("Alt+wheel stops at %d, want 0", e.brushRadius())
	}
	_ = e.Key("radiusup")
	e.ShowCell(image.Pt(30, 30))
	drag(e, image.Pt(30, 30))
	if len(p.strokes) != 1 || p.strokes[0].Radius != 1 || p.strokes[0].Shape != BrushCircle {
		t.Fatalf("stroke %+v", p.strokes)
	}
}

// The shape buttons choose Circle or Square for every plane, Circle first;
// the stroke carries the shape.
func TestMapEditorBrushShapeButtonsReachTheStroke(t *testing.T) {
	for _, key := range []string{"terrain", "height", "objects", "passability"} {
		e, p := brushEditor(t)
		_ = e.Key(key)
		if e.brush.shape != BrushCircle {
			t.Fatalf("%s: the brush starts %d", key, e.brush.shape)
		}
		pressBrushButton(t, e, 1, 1)
		if s, _ := e.listSpec(); s.Rows[1].Value != "Square" {
			t.Fatalf("%s: the list shows %q", key, s.Rows[1].Value)
		}
		e.ShowCell(image.Pt(30, 30))
		drag(e, image.Pt(30, 30))
		pressBrushButton(t, e, 1, 0)
		drag(e, image.Pt(31, 30))
		if len(p.strokes) != 2 || p.strokes[0].Shape != BrushSquare || p.strokes[1].Shape != BrushCircle {
			t.Fatalf("%s: strokes %+v", key, p.strokes)
		}
	}
}

// oddCells is a host whose brush walk is one far cell, whatever the stroke.
type oddCells struct{ brushProbe }

func (o *oddCells) BrushCells(BrushStroke) []image.Point { return []image.Point{{X: 40, Y: 40}} }

// The marker outlines the cells the host's walk gives: for a square of size 2
// the 5x5 block's 20 outer edges, for a circle of size 2 the 13 cells' 20; a
// host whose walk is one cell gets that cell's four edges.
func TestMapEditorBrushMarkerOutlinesTheHostCells(t *testing.T) {
	e, p := brushEditor(t)
	_ = e.Key("terrain")
	e.ShowCell(image.Pt(30, 30))
	pt := e.CellPoint(image.Pt(30, 30))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(6, 0))
	edges := func(cells []image.Point) int {
		in := map[image.Point]bool{}
		for _, c := range cells {
			in[c] = true
		}
		n := 0
		for _, c := range cells {
			for _, d := range cellSides {
				if !in[c.Add(d)] {
					n++
				}
			}
		}
		return n
	}
	for _, shape := range []BrushShape{BrushCircle, BrushSquare} {
		e.brush.shape = shape
		want := edges(probeBrushCells(BrushStroke{Radius: 2, Shape: shape, Cells: []image.Point{{X: 30, Y: 30}}}, 50, 50))
		if got := len(e.brushMarkerLines()); got != want || want == 0 {
			t.Fatalf("shape %d: %d marker lines, want %d", shape, got, want)
		}
	}
	e.doc.Edits = &oddCells{*p}
	if got := len(e.brushMarkerLines()); got != 4 {
		t.Fatalf("a one-cell walk draws %d lines", got)
	}
}

// Every brush help screen names the size's controls, the Alt+wheel and the
// list's - and + first, then the brush's own line with the brackets. The
// shape buttons name themselves.
func TestMapEditorBrushHelpNamesTheSizeControls(t *testing.T) {
	e, _ := brushEditor(t)
	for _, key := range []string{"terrain", "height", "objects", "passability"} {
		_ = e.Key(key)
		help := strings.Join(e.helpLines(), "\n")
		if !strings.HasPrefix(help, "Alt+wheel or - +: size\n") || !strings.Contains(help, "[ ]: size") {
			t.Fatalf("%s help: %q", key, help)
		}
		_ = e.Key(key)
	}
}

// The border view is on by default: it dims exactly the cells the host's
// plane closes by the border arm and draws the edge between them and the
// playable cells. The View row hides it; the passability view still names
// the border.
func TestMapEditorBorderViewShowsTheBorderArm(t *testing.T) {
	e, _ := passEditor(t)
	e.ShowCell(image.Pt(5, 3))
	quads := e.borderQuads()
	if len(quads) == 0 {
		t.Fatal("the border view is off by default")
	}
	for _, q := range quads {
		if q.fill != borderFill {
			t.Fatalf("fill %v", q.fill)
		}
	}
	want := 0
	e.doc.Viewer.forEachDrawnTile(func(col, row int) {
		if e.inBorder(image.Pt(col, row)) {
			want++
		}
		if e.BorderCell(image.Pt(col, row)) != (row == 0 && col >= 0 && col < e.doc.Width) {
			t.Fatalf("cell (%d,%d) border %t", col, row, e.BorderCell(image.Pt(col, row)))
		}
	})
	if len(quads) != want {
		t.Fatalf("%d fills for %d drawn border cells", len(quads), want)
	}
	if lines := e.borderLines(); len(lines) == 0 {
		t.Fatal("no edge line")
	}
	row, ok := e.MenuRow("view", "border")
	if !ok {
		t.Fatal("the View menu has no border row")
	}
	passMenu(t, e, "view", row)
	if len(e.borderQuads()) != 0 || len(e.borderLines()) != 0 || e.BorderCell(image.Pt(5, 0)) {
		t.Fatal("the View row did not hide the border")
	}
	passMenu(t, e, "view", row)
	if len(e.borderQuads()) == 0 {
		t.Fatal("the View row did not show the border again")
	}
	e.view.passability = true
	if len(e.borderQuads()) != 0 {
		t.Fatal("the border dims over the passability view's fills")
	}
	pt := e.CellPoint(image.Pt(5, 0))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(7, 0))
	if s := e.statusText(); !strings.Contains(strings.ToLower(s), "border") {
		t.Fatalf("status %q", s)
	}
}
