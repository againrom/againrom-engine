package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func bridgeWaterModel(t *testing.T, objects ...synth.ALMObject) *mapedit.Editor {
	t.Helper()
	tiles := make([]byte, 2*40*40)
	for i := 0; i < 40*40; i++ {
		binary.LittleEndian.PutUint16(tiles[2*i:], forestWater)
	}
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40,
		Type1Payload: tiles, Overlay: make([]byte, 40*40), Type7Payload: make([]byte, 12), Objects: objects}))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func bridgeWalkTable() *mapload.Table {
	table := authoringTable()
	buildings := make(dbCollection, 34)
	for i := 1; i < len(buildings); i++ {
		buildings[i] = dbEntry{name: "structure", params: []int32{3, 3, 0, 0, -1, -1, 0, 0}}
	}
	table.Buildings = buildings
	return table
}

func bridgeCellsIn(rects ...image.Rectangle) map[image.Point]bool {
	cells := make(map[image.Point]bool)
	for _, r := range rects {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				cells[image.Pt(x, y)] = true
			}
		}
	}
	return cells
}

func checkBridgeWalk(t *testing.T, m *mapedit.Editor, want ...image.Rectangle) {
	t.Helper()
	doc, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	table := bridgeWalkTable()
	counts := mapload.StructureCensus(doc, table)
	if counts.Refused != 0 || counts.Abandoned != 0 {
		t.Errorf("structure pass refused %d cells and abandoned %d", counts.Refused, counts.Abandoned)
	}
	plane := mapload.PassabilityWith(doc, table)
	water := mapload.Passability(doc)
	wanted := bridgeCellsIn(want...)
	stored := make(map[image.Point]bool)
	for _, r := range bridgeRects(t, m) {
		if !r.In(image.Rect(0, 0, 40, 40)) || r.Dx() > 255 || r.Dy() > 255 {
			t.Errorf("out-of-bounds stored rectangle %v", r)
		}
		for cell := range bridgeCellsIn(r) {
			if stored[cell] {
				t.Errorf("stored rectangles share cell %v", cell)
			}
			stored[cell] = true
			if plane[cell.Y*40+cell.X]&1 != 0 {
				t.Errorf("stored bridge cell %v is closed", cell)
			}
			if water[cell.Y*40+cell.X]&1 == 0 {
				t.Fatalf("synthetic water cell %v was already open", cell)
			}
		}
	}
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			cell := image.Pt(x, y)
			if stored[cell] != wanted[cell] {
				t.Errorf("cell %v stored=%v, want %v", cell, stored[cell], wanted[cell])
			}
		}
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table, structures: &terrain.StructureSet{}}
	findings, err := x.checkMapPages(m, "water.alm", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Rule == "structure-overlap" {
			t.Errorf("map report: %s", finding.Text)
		}
	}
}

func bridgeStrokeUndo(t *testing.T, m *mapedit.Editor, erase bool, points []image.Point, want ...image.Rectangle) {
	t.Helper()
	before, hadUndo := m.Bytes(), m.CanUndo()
	value := 1
	if erase {
		value = 0
	}
	changed, err := paintStroke(m, ui.BrushStroke{Plane: bridgePlane, Value: value, Cells: points}, bridgeWalkTable())
	if err != nil || !changed {
		t.Fatalf("host stroke changed=%v: %v", changed, err)
	}
	checkBridgeWalk(t, m, want...)
	after := m.Bytes()
	if !m.Undo() || !bytes.Equal(before, m.Bytes()) || m.CanUndo() != hadUndo {
		t.Fatal("one Undo did not restore the exact pre-stroke map and history")
	}
	if !m.Redo() || !bytes.Equal(after, m.Bytes()) {
		t.Fatal("Redo did not restore the exact stroke bytes")
	}
}

func TestBridgeStrokeKeepsWalkableCells(t *testing.T) {
	band := []image.Point{{10, 10}, {11, 10}, {12, 10}, {13, 10}, {14, 10}}
	leg := []image.Point{{10, 11}, {10, 12}, {10, 13}, {10, 14}}
	for _, tc := range []struct {
		name    string
		objects []synth.ALMObject
		strokes [][]image.Point
		erase   bool
		want    []image.Rectangle
	}{
		{"straight", nil, [][]image.Point{band}, false, []image.Rectangle{image.Rect(9, 10, 16, 12)}},
		{"L", nil, [][]image.Point{append(append([]image.Point{}, band...), leg...)}, false,
			[]image.Rectangle{image.Rect(9, 10, 16, 12), image.Rect(9, 12, 12, 16)}},
		{"L-two-strokes", nil, [][]image.Point{band, {{10, 12}, {10, 13}, {10, 14}}}, false,
			[]image.Rectangle{image.Rect(9, 10, 16, 12), image.Rect(9, 12, 12, 16)}},
		{"T", nil, [][]image.Point{append(append([]image.Point{}, band...), image.Pt(12, 11), image.Pt(12, 12), image.Pt(12, 13), image.Pt(12, 14))}, false,
			[]image.Rectangle{image.Rect(9, 10, 16, 12), image.Rect(11, 12, 14, 16)}},
		{"join-down", []synth.ALMObject{bridgeObject(image.Rect(5, 10, 14, 13), 7)},
			[][]image.Point{{{12, 13}, {12, 14}, {12, 15}, {12, 16}, {12, 17}}}, false,
			[]image.Rectangle{image.Rect(5, 10, 14, 13), image.Rect(11, 13, 14, 19)}},
		{"corner", []synth.ALMObject{bridgeObject(image.Rect(5, 4, 8, 10), 7)},
			[][]image.Point{{{9, 10}, {10, 10}, {11, 10}, {12, 10}, {13, 10}, {14, 10}}}, false,
			[]image.Rectangle{image.Rect(5, 4, 8, 10), image.Rect(8, 10, 16, 12)}},
		{"touching-pair", []synth.ALMObject{bridgeObject(image.Rect(5, 10, 14, 12), 7), bridgeObject(image.Rect(11, 12, 14, 17), 8)},
			[][]image.Point{{{3, 10}}}, false, []image.Rectangle{image.Rect(2, 10, 14, 12), image.Rect(11, 12, 14, 17)}},
		{"erase-middle", []synth.ALMObject{bridgeObject(image.Rect(5, 10, 14, 13), 7)},
			[][]image.Point{{{9, 10}, {9, 11}}}, true, []image.Rectangle{image.Rect(5, 10, 8, 13), image.Rect(11, 10, 14, 13)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := bridgeWaterModel(t, tc.objects...)
			for i, points := range tc.strokes {
				want := tc.want
				if i+1 < len(tc.strokes) {
					want = []image.Rectangle{image.Rect(9, 10, 16, 12)}
				}
				bridgeStrokeUndo(t, m, tc.erase, points, want...)
			}
		})
	}
}

func TestBridgeStrokeLeavesIndirectRecordBytes(t *testing.T) {
	m := bridgeWaterModel(t, bridgeObject(image.Rect(5, 10, 14, 12), 7), bridgeObject(image.Rect(11, 12, 14, 17), 8))
	before, err := m.Record(mapedit.KindStructure, 1)
	if err != nil {
		t.Fatal(err)
	}
	bridgePaint(t, m, false, image.Pt(3, 10))
	after, err := m.Record(mapedit.KindStructure, 1)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("indirect record changed: %x -> %x (%v)", before, after, err)
	}
}

func TestBridgeStrokeAvoidsOtherStructures(t *testing.T) {
	other := synth.ALMObject{X: 11*256 + 128, Y: 10*256 + 128, Kind: 1, Field12: 19}
	for _, objects := range [][]synth.ALMObject{
		{other},
		{bridgeObject(image.Rect(7, 10, 10, 12), 7), other},
		{other, bridgeObject(image.Rect(7, 10, 10, 12), 7)},
	} {
		m := bridgeWaterModel(t, objects...)
		bridgeStrokeUndo(t, m, false, []image.Point{{8, 10}, {9, 10}, {10, 10}, {11, 10}, {12, 10}, {13, 10}, {14, 10}, {15, 10}, {16, 10}},
			image.Rect(7, 10, 11, 12), image.Rect(14, 10, 18, 12))
	}
}
