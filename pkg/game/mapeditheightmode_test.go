package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// The window's height stroke reaches the model as the one height brush with
// its mode as data: Step is the add without fall-off once per cell, Level
// the flat set relative to the pressed cell, Smooth the cosine add. A Set or
// Sample height stroke reads no mode: with every mode it is the brush it was.
func TestHeightStrokeModesReachTheModelAsBrushData(t *testing.T) {
	ground := func() *mapedit.Editor {
		m, err := mapedit.NewMap(alm.NewMapSpec{Width: 40, Height: 40, Tile: 0x14, Altitude: 64})
		if err != nil {
			t.Fatal(err)
		}
		hill := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 20, Radius: 4, Falloff: mapedit.FalloffCosine}
		if _, err := m.Paint(hill, []mapedit.Cell{{X: 18, Y: 20}}, nil); err != nil {
			t.Fatal(err)
		}
		return m
	}
	drag := []image.Point{{X: 16, Y: 20}, {X: 17, Y: 20}, {X: 18, Y: 20}, {X: 17, Y: 20}}
	at := []mapedit.Cell{{X: 16, Y: 20}, {X: 17, Y: 20}, {X: 18, Y: 20}, {X: 17, Y: 20}}
	h := mapedit.Brush{Plane: mapedit.PlaneHeight, Radius: 2, Shape: mapedit.BrushCircle}
	with := func(f func(*mapedit.Brush)) mapedit.Brush { b := h; f(&b); return b }
	cases := []struct {
		name string
		op   ui.BrushStroke
		want mapedit.Brush
	}{
		{"step", ui.BrushStroke{Value: 5, Mode: ui.HeightModeStep}, with(func(b *mapedit.Brush) { b.Value, b.Once = 5, true })},
		{"level", ui.BrushStroke{Value: -5, Mode: ui.HeightModeLevel}, with(func(b *mapedit.Brush) { b.Value, b.Set, b.Flat, b.Relative = -5, true, true, true })},
		{"smooth", ui.BrushStroke{Value: 8, Mode: ui.HeightModeSmooth}, with(func(b *mapedit.Brush) { b.Value, b.Falloff = 8, mapedit.FalloffCosine })},
		{"set", ui.BrushStroke{Value: 90, Set: true, Mode: ui.HeightModeLevel}, with(func(b *mapedit.Brush) { b.Value, b.Set, b.Falloff = 90, true, mapedit.FalloffCosine })},
		{"sample", ui.BrushStroke{Value: 70, Set: true, Flat: true, Mode: ui.HeightModeStep}, with(func(b *mapedit.Brush) { b.Value, b.Set, b.Flat, b.Falloff = 70, true, true, mapedit.FalloffCosine })},
	}
	for _, c := range cases {
		op := c.op
		op.Plane, op.Radius, op.Shape, op.Cells = ui.BrushPlaneHeight, 2, ui.BrushCircle, drag
		got, want := ground(), ground()
		if changed, err := paintStroke(got, op, nil); err != nil || !changed {
			t.Fatalf("%s: paintStroke = %v, %v", c.name, changed, err)
		}
		if _, err := want.Paint(c.want, at, nil); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Bytes(), want.Bytes()) {
			t.Errorf("%s: the window's stroke is not the expected brush", c.name)
		}
	}
}
