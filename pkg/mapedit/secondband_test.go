package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// secondPlanesMap is a 24 x 24 second-game map of the four plane records,
// every type-3 byte 7, so a clearing stroke shows.
func secondPlanesMap() []byte {
	const side = 24
	le := binary.LittleEndian
	raw := make([]byte, 20)
	copy(raw, "M7R\x00")
	le.PutUint32(raw[4:], 20)
	le.PutUint32(raw[8:], 4*side*side+72)
	le.PutUint32(raw[12:], 4)
	le.PutUint32(raw[16:], 1600)
	meta := make([]byte, 660)
	le.PutUint32(meta, side)
	le.PutUint32(meta[4:], side)
	for id, payload := range [][]byte{meta, make([]byte, 2*side*side), make([]byte, side*side), bytes.Repeat([]byte{7}, side*side)} {
		header := make([]byte, 20)
		le.PutUint32(header, 7)
		le.PutUint32(header[4:], 20)
		le.PutUint32(header[8:], uint32(len(payload)))
		le.PutUint32(header[12:], uint32(id))
		if id == 0 {
			le.PutUint32(header[8:], 644)
		}
		raw = append(append(raw, header...), payload...)
	}
	return raw
}

// inBand is whether cell i of a 24 x 24 grid lies inside the 8-cell edge
// ring the second game's map band keeps (R2-ENGINE-1668).
func inBand(i int) bool { x, y := i%24, i/24; return x >= 8 && y >= 8 && x <= 15 && y <= 15 }

// A height, impassable or object stroke on a second-game map writes only the
// cells inside the 8-cell edge ring, and a stroke with no cell inside it is
// refused and writes nothing.
func TestSecondMapPlaneBrushesKeepTheEdgeRing(t *testing.T) {
	strokes := []mapedit.Brush{
		{Plane: mapedit.PlaneHeight, Value: 20, Radius: 3, Falloff: mapedit.FalloffNone},
		{Plane: mapedit.PlaneImpassable, Value: 1, Radius: 3},
		{Plane: mapedit.PlaneObject, Value: 0, Radius: 3},
	}
	for _, b := range strokes {
		raw := secondPlanesMap()
		e, err := mapedit.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		if changed, err := e.Paint(b, []mapedit.Cell{{X: 3, Y: 3}}, testClass); err == nil || changed || e.CanUndo() || !bytes.Equal(e.Bytes(), raw) {
			t.Fatalf("plane %d: a stroke in the edge ring was not refused: %v %v", b.Plane, changed, err)
		}
		if changed, err := e.Paint(b, []mapedit.Cell{{X: 9, Y: 9}}, testClass); err != nil || !changed {
			t.Fatalf("plane %d: %v %v", b.Plane, changed, err)
		}
		was, err := alm.OpenROM2(raw)
		if err != nil {
			t.Fatal(err)
		}
		out, err := e.SaveBytes()
		if err != nil {
			t.Fatal(err)
		}
		got, err := alm.OpenROM2(out)
		if err != nil {
			t.Fatalf("plane %d: the saved map does not decode: %v", b.Plane, err)
		}
		for i := range was.Altitudes {
			moved := got.Altitudes[i] != was.Altitudes[i] || got.Tiles[i] != was.Tiles[i] || got.Overlay[i] != was.Overlay[i]
			if moved && !inBand(i) {
				t.Fatalf("plane %d: cell (%d,%d) in the edge ring changed", b.Plane, i%24, i/24)
			}
		}
		at := 9*24 + 8
		switch b.Plane {
		case mapedit.PlaneHeight:
			if got.Altitudes[at] != 20 || !bytes.Equal(got.Overlay, was.Overlay) {
				t.Fatalf("the height stroke wrote %d at (8,9) or moved another plane", got.Altitudes[at])
			}
		case mapedit.PlaneImpassable:
			if got.Tiles[at] != alm.TileImpassable || !bytes.Equal(got.Altitudes, was.Altitudes) {
				t.Fatalf("the impassable stroke wrote word %#x at (8,9) or moved another plane", got.Tiles[at])
			}
		case mapedit.PlaneObject:
			if got.Overlay[at] != 0 || !bytes.Equal(got.Altitudes, was.Altitudes) {
				t.Fatalf("the object stroke left byte %d at (8,9) or moved another plane", got.Overlay[at])
			}
		}
	}
}

// The whole-map heights of a second-game map leave the edge ring as loaded.
func TestSecondMapRandomHeightsKeepTheEdgeRing(t *testing.T) {
	raw := secondPlanesMap()
	e, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := e.RandomizeHeights(source(7)); err != nil || !changed {
		t.Fatalf("random heights: %v %v", changed, err)
	}
	got, err := alm.OpenROM2(e.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for i, h := range got.Altitudes {
		if h != 0 && !inBand(i) {
			t.Fatalf("cell (%d,%d) in the edge ring took height %d", i%24, i/24, h)
		}
	}
}

// A first-game map keeps no edge ring: a stroke at its corner writes.
func TestFirstMapPlaneBrushPaintsToTheEdge(t *testing.T) {
	e, err := mapedit.New(brushMap(0x14, 20))
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := e.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 10}, []mapedit.Cell{{X: 0, Y: 0}}, testClass); err != nil || !changed {
		t.Fatalf("first-game corner stroke: %v %v", changed, err)
	}
}
