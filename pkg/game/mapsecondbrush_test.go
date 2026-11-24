package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// secondBrushHost is the second-game fixture map open in its edit host, every
// type-3 byte 7, and the map as loaded.
func secondBrushHost(t *testing.T) (*MapInspector, *mapInspectionEdits, *alm.Map) {
	t.Helper()
	raw := secondFixtureMap()
	plane3 := alm.FileHeaderSize + 4*alm.RecordHeaderSize + 660 + 3*32*32
	copy(raw[plane3:], bytes.Repeat([]byte{7}, 32*32))
	x := secondTestInspector(t)
	x.objectRows = []ui.BrushObject{{Code: 0}, {Code: 3}}
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	was, err := alm.OpenROM2(raw)
	if err != nil {
		t.Fatal(err)
	}
	return x, d.Edits.(*mapInspectionEdits), was
}

// paintedSecond is the host's map through the editor's save, decoded back.
func paintedSecond(t *testing.T, host *mapInspectionEdits) *alm.Map {
	t.Helper()
	target, err := host.SaveAs(filepath.Join(t.TempDir(), "painted.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.OpenROM2(saved)
	if err != nil {
		t.Fatalf("the painted map does not decode: %v", err)
	}
	return m
}

// On a second-game map the height brush writes the type-2 plane, the
// passability brush the type-1 word's bit 0x2000 and the object brush clears
// type-3 bytes; each decodes back from the saved map.
func TestSecondMapBrushesPaintHeightsPassabilityAndClearObjects(t *testing.T) {
	_, host, was := secondBrushHost(t)
	strokes := []ui.BrushStroke{
		{Plane: ui.BrushPlaneHeight, Value: 12, Mode: ui.HeightModeStep, Cells: []image.Point{{X: 16, Y: 16}}},
		{Plane: ui.BrushPlanePassability, Value: 1, Cells: []image.Point{{X: 12, Y: 20}}},
		{Plane: ui.BrushPlaneObjects, Value: 0, Cells: []image.Point{{X: 20, Y: 12}}},
	}
	for _, op := range strokes {
		if d, err := host.Paint(op); err != nil || d == nil {
			t.Fatalf("plane %d: stroke refused or wrote nothing: %v", op.Plane, err)
		}
	}
	got := paintedSecond(t, host)
	height, block, object := 16*32+16, 20*32+12, 12*32+20
	if got.Altitudes[height] != 12 || got.Tiles[block] != was.Tiles[block]|alm.TileImpassable || got.Overlay[object] != 0 {
		t.Fatalf("saved planes: height %d, word %#x, object %d", got.Altitudes[height], got.Tiles[block], got.Overlay[object])
	}
	for i := range was.Altitudes {
		if i != height && got.Altitudes[i] != was.Altitudes[i] || i != block && got.Tiles[i] != was.Tiles[i] || i != object && got.Overlay[i] != was.Overlay[i] {
			t.Fatalf("cell (%d,%d) changed outside the strokes", i%32, i/32)
		}
	}
	if len(got.Units) != len(was.Units) || len(got.Objects) != len(was.Objects) {
		t.Fatal("a stroke moved a record")
	}
}

// A second-game stroke with no cell inside the 8-cell edge ring is refused
// in the editor's words and writes nothing; the marker shows the cells the
// ring leaves.
func TestSecondMapBrushRefusesTheEdgeRing(t *testing.T) {
	x, host, _ := secondBrushHost(t)
	before := host.model.Bytes()
	line := fmt.Sprintf(x.word("editor.err.brushband"), 8)
	for _, plane := range []ui.BrushPlane{ui.BrushPlaneHeight, ui.BrushPlanePassability, ui.BrushPlaneObjects} {
		op := ui.BrushStroke{Plane: plane, Value: 1, Radius: 2, Cells: []image.Point{{X: 3, Y: 28}}}
		if plane == ui.BrushPlaneObjects {
			op.Value = 0
		}
		if _, err := host.Paint(op); err == nil || err.Error() != line {
			t.Fatalf("plane %d: an edge-ring stroke was not refused with its line: %v", plane, err)
		}
		op.Cells = []image.Point{{X: 9, Y: 9}}
		for _, c := range host.BrushCells(op) {
			if c.X < 8 || c.Y < 8 {
				t.Fatalf("plane %d: the marker covers ring cell %v", plane, c)
			}
		}
	}
	if !bytes.Equal(host.model.Bytes(), before) || host.CanUndo() {
		t.Fatal("a refused stroke wrote the map")
	}
}

// The parts no claim reads stay refused in the second game's words: an object
// class, a forest, a bridge and random terrain. Random heights write the
// height plane inside the edge ring alone.
func TestSecondMapBrushRefusalsAndRandomHeights(t *testing.T) {
	x, host, was := secondBrushHost(t)
	refusals := []struct {
		op   ui.BrushStroke
		word string
	}{
		{ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: 3, Cells: []image.Point{{X: 12, Y: 12}}}, "editor.map.secondobjects"},
		{ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: 0, Forest: ui.BrushForest{Kinds: []int{3}, Share: 50}, Cells: []image.Point{{X: 12, Y: 12}}}, "editor.map.secondobjects"},
		{ui.BrushStroke{Plane: ui.BrushPlaneBridge, Value: 1, Cells: []image.Point{{X: 12, Y: 12}}}, "editor.map.secondbridge"},
	}
	for _, r := range refusals {
		if _, err := host.Paint(r.op); err == nil || err.Error() != x.word(r.word) {
			t.Fatalf("plane %d: not refused with %s: %v", r.op.Plane, r.word, err)
		}
	}
	if _, err := host.RandomTerrain(3); err == nil || err.Error() != x.word("editor.map.secondrandomterrain") {
		t.Fatalf("random terrain not refused with its line: %v", err)
	}
	if _, ok := ui.Seam[ui.InspectionWholeMap](host); !ok {
		t.Fatal("the whole-map seam is not offered")
	}
	if d, err := host.RandomHeights(3); err != nil || d == nil {
		t.Fatalf("random heights: %v", err)
	}
	got := paintedSecond(t, host)
	moved := 0
	for i := range was.Altitudes {
		x, y := i%32, i/32
		if got.Altitudes[i] != was.Altitudes[i] {
			moved++
			if x < 8 || y < 8 || x > 23 || y > 23 {
				t.Fatalf("random heights changed ring cell (%d,%d)", x, y)
			}
		}
		if got.Tiles[i] != was.Tiles[i] || got.Overlay[i] != was.Overlay[i] {
			t.Fatalf("random heights changed another plane at (%d,%d)", x, y)
		}
	}
	if moved == 0 {
		t.Fatal("random heights changed no cell")
	}
}
