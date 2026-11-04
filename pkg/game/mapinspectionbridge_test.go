package game

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const bridgePlane = ui.BrushPlane(4)

func bridgeObject(r image.Rectangle, id uint16) synth.ALMObject {
	return synth.ALMObject{X: uint32(r.Min.X)*256 + 128, Y: uint32(r.Min.Y)*256 + 128,
		Kind: 33, Field12: id, Ext: []byte{byte(r.Dx()), 91, 92, 93, byte(r.Dy()), 94, 95, 96}}
}

func bridgeModel(t *testing.T, objects ...synth.ALMObject) *mapedit.Editor {
	t.Helper()
	m, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Objects: objects, Type7Payload: make([]byte, 12)}))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func bridgePaint(t *testing.T, m *mapedit.Editor, erase bool, points ...image.Point) {
	t.Helper()
	value := 1
	if erase {
		value = 0
	}
	changed, err := paintStroke(m, ui.BrushStroke{Plane: bridgePlane, Value: value, Cells: points}, nil)
	if err != nil || !changed {
		t.Fatalf("bridge stroke changed=%v: %v", changed, err)
	}
}

func bridgeRects(t *testing.T, m *mapedit.Editor) []image.Rectangle {
	t.Helper()
	doc, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	var out []image.Rectangle
	for _, o := range doc.Objects {
		if o.Kind != 33 {
			continue
		}
		x, y := int(o.X>>8), int(o.Y>>8)
		w, h := int(o.Ext[0]), int(o.Ext[4])
		if w < 3 || h < 2 {
			t.Fatalf("stored undersized bridge %dx%d", w, h)
		}
		out = append(out, image.Rect(x, y, x+w, y+h))
	}
	return out
}

func wantBridgeRects(t *testing.T, m *mapedit.Editor, want ...image.Rectangle) {
	t.Helper()
	if got := bridgeRects(t, m); !reflect.DeepEqual(got, want) {
		t.Fatalf("rectangles %v, want %v", got, want)
	}
}

func TestBridgeStrokeAcrossOpenGround(t *testing.T) {
	m := bridgeModel(t)
	bridgePaint(t, m, false, image.Pt(10, 10), image.Pt(11, 10), image.Pt(12, 10), image.Pt(13, 10), image.Pt(14, 10))
	wantBridgeRects(t, m, image.Rect(9, 10, 16, 12))
}

func TestBridgeEraseSplitsAndDropsSmallPieces(t *testing.T) {
	for _, width := range []int{9, 8} {
		t.Run(string(rune('0'+width)), func(t *testing.T) {
			m := bridgeModel(t, bridgeObject(image.Rect(5, 10, 5+width, 13), 7))
			bridgePaint(t, m, true, image.Pt(9, 10), image.Pt(9, 11))
			want := []image.Rectangle{image.Rect(5, 10, 8, 13)}
			if width == 9 {
				want = append(want, image.Rect(11, 10, 14, 13))
			}
			wantBridgeRects(t, m, want...)
		})
	}
}

func TestBridgeStrokeMergesTouchingRecords(t *testing.T) {
	m := bridgeModel(t, bridgeObject(image.Rect(5, 10, 8, 12), 7), bridgeObject(image.Rect(11, 10, 14, 12), 8))
	bridgePaint(t, m, false, image.Pt(9, 10))
	wantBridgeRects(t, m, image.Rect(5, 10, 14, 12))
}

func TestBridgeStrokeStoresAnLAsTwoRectangles(t *testing.T) {
	m := bridgeModel(t)
	bridgePaint(t, m, false, image.Pt(10, 10), image.Pt(11, 10), image.Pt(12, 10), image.Pt(13, 10), image.Pt(14, 10), image.Pt(10, 11), image.Pt(10, 12), image.Pt(10, 13), image.Pt(10, 14))
	wantBridgeRects(t, m, image.Rect(9, 10, 16, 12), image.Rect(9, 12, 12, 16))
}

func TestBridgeStrokePreservesRecordBytesAndUntouchedSave(t *testing.T) {
	m := bridgeModel(t, bridgeObject(image.Rect(25, 25, 28, 28), 19), bridgeObject(image.Rect(5, 10, 8, 12), 7))
	before := m.Bytes()
	saved, err := m.SaveBytes()
	if err != nil || !bytes.Equal(before, saved) {
		t.Fatal("untouched save changed bytes", err)
	}
	if _, err := m.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 1}, []mapedit.Cell{{X: 30, Y: 30}}, nil); err != nil {
		t.Fatal(err)
	}
	saved, err = m.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := mapedit.New(saved)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		a, _ := m.Record(mapedit.KindStructure, i)
		b, _ := loaded.Record(mapedit.KindStructure, i)
		if !bytes.Equal(a, b) {
			t.Fatal("saving another brush changed bridge bytes", i)
		}
	}
	untouched, _ := m.Record(mapedit.KindStructure, 0)
	grown, _ := m.Record(mapedit.KindStructure, 1)
	bridgePaint(t, m, false, image.Pt(9, 10))
	after, _ := m.Record(mapedit.KindStructure, 0)
	if !bytes.Equal(untouched, after) {
		t.Fatal("unmet record changed")
	}
	after, _ = m.Record(mapedit.KindStructure, 1)
	grown[alm.ObjectWidth] = 6
	if !bytes.Equal(grown, after) {
		t.Fatalf("growth changed more than width: %x / %x", grown, after)
	}
}

func TestBridgeStrokeIsOneUndoStep(t *testing.T) {
	m := bridgeModel(t, bridgeObject(image.Rect(5, 10, 14, 13), 7))
	before := m.Bytes()
	bridgePaint(t, m, true, image.Pt(9, 10), image.Pt(9, 11))
	after := m.Bytes()
	if !m.Undo() || !bytes.Equal(before, m.Bytes()) || m.CanUndo() {
		t.Fatal("one undo did not restore exact map bytes")
	}
	if !m.Redo() || !bytes.Equal(after, m.Bytes()) {
		t.Fatal("redo changed stroke bytes")
	}
}

func TestBridgeBrushCellsMatchMinimumAtMapEdge(t *testing.T) {
	m := bridgeModel(t)
	s := &mapInspectionEdits{model: m}
	points := s.BrushCells(ui.BrushStroke{Plane: bridgePlane, Cells: []image.Point{{0, 0}}})
	if len(points) != 6 {
		t.Fatalf("edge footprint %v, want 3x2", points)
	}
	bridgePaint(t, m, false, image.Pt(0, 0))
	wantBridgeRects(t, m, image.Rect(0, 0, 3, 2))
}

func TestBridgeToolPointerPaintsErasesAndRefreshesView(t *testing.T) {
	e, x := forestEditor(t, forestMap(t))
	c := &terrain.StructureClass{ID: 33, VariableSize: true, VariableLayout: terrain.VariableStructureVerticalNinePatch}
	for i := 0; i < 9; i++ {
		c.Frames = append(c.Frames, structurePreviewFrame(color.RGBA{R: uint8(20*i + 40), G: 80, A: 255}))
	}
	x.structures.Classes[33] = c
	button, ok := e.ToolbarRect("brush.bridge")
	if !ok {
		t.Fatal("Bridge tool is missing from the brush toolbar")
	}
	d := &forestDriver{t: t, e: e, c: &editorClicker{t: t}}
	d.c.click(e, button, ok, "Bridge")
	if on, _, _ := e.ToolbarState("brush.bridge"); !on {
		t.Fatal("Bridge did not activate", e.Message())
	}
	for i := 0; i < 20; i++ {
		d.key("radiusdown")
	}
	d.stroke(false, image.Pt(20, 26), image.Pt(26, 26))
	if e.Message() != "" {
		t.Log("paint message:", e.Message())
	}
	m, err := mapedit.New(d.bytes())
	if err != nil {
		t.Fatal(err)
	}
	wantBridgeRects(t, m, image.Rect(19, 26, 28, 28))
	if _, counts := e.Document().Viewer.Structures(); counts.Placements.Drawn != 1 {
		t.Fatalf("editor bridge drawing: %+v", counts)
	}
	play, err := LoadMapViewer(x.tiles, m.Bytes(), "bridge", Markers{}, StaticLayer{}, StructureLayer{Set: x.structures, Art: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, counts := play.Viewer.Structures(); counts.Placements.Drawn != 1 {
		t.Fatalf("game bridge drawing: %+v", counts)
	}
	painted := e.Document().Viewer
	d.stroke(true, image.Pt(23, 26))
	m, err = mapedit.New(d.bytes())
	if err != nil {
		t.Fatal(err)
	}
	wantBridgeRects(t, m, image.Rect(19, 26, 22, 28), image.Rect(25, 26, 28, 28))
	if e.Document().Viewer == painted {
		t.Fatal("bridge erase did not refresh the view")
	}
}
