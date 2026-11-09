package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// treeStrokeCells are cells the Forest scatter never plants on: one holding
// another object, a road cell, a water cell, a structure footprint cell, the
// drop cell and a unit cell.
var treeStrokeCells = []struct {
	name string
	cell image.Point
}{
	{"another object", image.Pt(30, 18)},
	{"a road", image.Pt(30, 24)},
	{"water", image.Pt(15, 15)},
	{"a structure footprint", image.Pt(20, 20)},
	{"the drop cell", image.Pt(30, 30)},
	{"a unit", image.Pt(34, 5)},
}

// treeStrokeMap is the forest fixture map with a unit at (34,5) and the drop
// cell at (30,30).
func treeStrokeMap(t *testing.T) []byte {
	t.Helper()
	tiles := make([]byte, 2*40*40)
	overlay := make([]byte, 40*40)
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			w := uint16(forestLand)
			switch {
			case x >= 14 && x <= 16 && y >= 12 && y <= 28:
				w = forestWater
			case y == 24:
				w = forestRoad
			}
			binary.LittleEndian.PutUint16(tiles[2*(y*40+x):], w)
		}
	}
	for x := 0; x < 40; x += 5 {
		overlay[18*40+x] = forestOther
	}
	raw := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type1Payload: tiles, Overlay: overlay, Type7Payload: make([]byte, 12),
		Objects: []synth.ALMObject{{X: 20*256 + 128, Y: 20*256 + 128, Kind: 5, Field12: 7}},
		Units:   []synth.ALMUnit{{X: 34*256 + 128, Y: 5*256 + 128}}})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetDropCell(30, 30); err != nil {
		t.Fatal(err)
	}
	return m.Bytes()
}

// The Trees palette's One kind stroke of a tree class writes what the object
// brush's row of that class wrote: the class on every covered cell, over
// another object, a road, water, a structure footprint, the drop cell and a
// unit, where the Forest scatter writes nothing.
func TestEditorTreesOneKindStrokeWritesTheObjectBrushBytes(t *testing.T) {
	raw := treeStrokeMap(t)
	doc, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	fenced := forestFence(doc, paletteInspector().table)
	for _, c := range treeStrokeCells {
		i := c.cell.Y*40 + c.cell.X
		open := doc.Overlay[i] == 0 && !fenced[i]
		if c.name == "a road" || c.name == "water" {
			open = false
		}
		if open {
			t.Fatalf("%s at %v is a cell the Forest scatter plants on", c.name, c.cell)
		}
	}
	for _, c := range treeStrokeCells {
		e, ck := paletteEditorOn(t, raw)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "trees"})
		ck.click(e, r, ok, "the Trees entry")
		r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlListButton, N: 2, K: 0})
		ck.click(e, r, ok, "the One kind button")
		if err := e.Key("down"); err != nil {
			t.Fatal(err)
		}
		for range 20 {
			_ = e.Key("radiusdown")
		}
		clickMapCell(ck, e, c.cell)

		o, _ := paletteEditorOn(t, raw)
		want, err := objectEdits(t, o).Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: forestOakish, Cells: []image.Point{c.cell}})
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(want.SourceBytes, raw) {
			t.Fatalf("%s: the object brush's oak row wrote nothing", c.name)
		}
		if !bytes.Equal(e.Document().SourceBytes, want.SourceBytes) {
			t.Errorf("%s at %v: the Trees one-kind stroke wrote other bytes than the object brush's oak row (%q)", c.name, c.cell, e.Message())
		}
	}
}
