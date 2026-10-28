package ui

import (
	"image"
	"testing"
)

// fillProbe is a brush host that offers 7 look-alike tiles for class 2 and
// one for any other class.
type fillProbe struct{ *brushProbe }

func (fillProbe) BrushVariants(class int) int {
	if class == 2 {
		return 7
	}
	return 1
}

func fillEditor(t *testing.T) (*MapEditor, *brushProbe) {
	t.Helper()
	e, p := brushEditor(t)
	e.doc.Edits = fillProbe{p}
	_ = e.Key("terrain")
	e.ShowCell(image.Pt(30, 30))
	return e, p
}

// The brush window shows the centre fill row, its buttons choose pattern,
// random or a variant, the choice goes with the stroke, and it is kept while
// the terrain changes (a variant held inside the terrain's own).
func TestBrushWindowFillRowChoosesTheCentreFill(t *testing.T) {
	e, p := fillEditor(t)
	s, _ := e.listSpec()
	if len(s.Rows) != 3 || s.Rows[2].Label != "Centre" || s.Rows[2].Value != "Random" {
		t.Fatalf("brush list rows %+v", s.Rows)
	}
	stroke := func() BrushStroke {
		drag(e, image.Pt(30, 30))
		return p.strokes[len(p.strokes)-1]
	}
	if f := stroke().Fill; f.Mode != BrushFillRandom {
		t.Fatalf("a fresh brush strokes with %+v, not random", f)
	}
	pressBrushButton(t, e, 2, 0)
	if s, _ = e.listSpec(); s.Rows[2].Value != "Pattern" {
		t.Fatalf("fill row reads %q after Pattern", s.Rows[2].Value)
	}
	if f := stroke().Fill; f != (BrushFill{}) {
		t.Fatalf("Pattern strokes with %+v", f)
	}
	pressBrushButton(t, e, 2, 1)
	if s, _ = e.listSpec(); s.Rows[2].Value != "Random" {
		t.Fatalf("fill row reads %q after Random", s.Rows[2].Value)
	}
	if f := stroke().Fill; f.Mode != BrushFillRandom {
		t.Fatalf("stroke fill %+v, want random", f)
	}
	pressBrushButton(t, e, 2, 3)
	pressBrushButton(t, e, 2, 3)
	if s, _ = e.listSpec(); s.Rows[2].Value != "Variant 2 of 7" {
		t.Fatalf("fill row reads %q", s.Rows[2].Value)
	}
	if f := stroke().Fill; f != (BrushFill{Mode: BrushFillVariant, Variant: 1}) {
		t.Fatalf("stroke fill %+v, want variant 2", f)
	}
	pressBrushButton(t, e, 2, 2)
	pressBrushButton(t, e, 2, 2)
	if s, _ = e.listSpec(); s.Rows[2].Value != "Variant 7 of 7" {
		t.Fatalf("minus from the first variant reads %q", s.Rows[2].Value)
	}
	_ = e.Key("down") // the next terrain has one tile: nothing to choose, no fill row
	if s, _ = e.listSpec(); len(s.Rows) != 2 {
		t.Fatalf("a one-tile terrain shows rows %+v", s.Rows)
	}
	if f := stroke().Fill; f != (BrushFill{}) {
		t.Fatalf("one-tile terrain strokes with %+v", f)
	}
	_ = e.Key("up")
	if s, _ = e.listSpec(); len(s.Rows) != 3 || s.Rows[2].Value != "Variant 7 of 7" {
		t.Fatalf("back on the seven-tile terrain rows %+v", s.Rows)
	}
	pressBrushButton(t, e, 2, 0)
	if f := stroke().Fill; f != (BrushFill{}) {
		t.Fatalf("Pattern strokes with %+v", f)
	}
}

// A host without tile variants shows no fill row, and the height brush shows
// none either: its third row is the strength.
func TestBrushWindowHasNoFillRowWithoutVariants(t *testing.T) {
	e, _ := brushEditor(t)
	_ = e.Key("terrain")
	if s, _ := e.listSpec(); len(s.Rows) != 2 {
		t.Fatalf("rows %+v", s.Rows)
	}
	e2, _ := fillEditor(t)
	_ = e2.Key("height")
	if s, _ := e2.listSpec(); len(s.Rows) != 3 || s.Rows[2].Label != "Strength" {
		t.Fatalf("height brush rows %+v", s.Rows)
	}
}
