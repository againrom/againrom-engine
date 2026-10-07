package ui

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

type editProbe1118 struct {
	doc               *InspectionDocument
	index, undo, redo int
	dirty             bool
}

func (p *editProbe1118) Dirty() bool   { return p.dirty }
func (p *editProbe1118) CanUndo() bool { return p.dirty }
func (p *editProbe1118) CanRedo() bool { return !p.dirty }
func (p *editProbe1118) Undo() (*InspectionDocument, error) {
	p.undo++
	return nil, nil
}
func (p *editProbe1118) Redo() (*InspectionDocument, error) {
	p.redo++
	return nil, nil
}
func (p *editProbe1118) SaveAs(string) (string, error) { return "", fmt.Errorf("test writer refused") }
func (p *editProbe1118) MoveUnit(index int, cell image.Point) (*InspectionDocument, error) {
	p.index, p.dirty = index, true
	next := *p.doc
	next.Records = append([]InspectionRecord(nil), next.Records...)
	for i, r := range next.Records {
		if r.Kind == "Unit" && r.Index == index {
			next.Records[i].Cell = cell
		}
	}
	v, err := NewViewer("edited", p.doc.Viewer.grid, &terrain.Tileset{})
	if err != nil {
		return nil, err
	}
	v.SetEntities([]MapEntity{{ID: uint32(index), Cell: cell}})
	next.Viewer = v
	p.doc = &next
	return &next, nil
}

func TestMapEditor1118CaptureOriginalIdentityPreviewAndRetainView(t *testing.T) {
	e := editor1092(t)
	e.Layout(1024, 768)
	e.doc.Records[20].Index = 63 // file index is neither row nor filtered ordinal
	p := &editProbe1118{doc: e.doc, index: -1}
	e.doc.Edits = p
	e.Select(20, true)
	e.filter, e.offset, e.detailOffset = 2, 3, 5
	cam := *e.doc.Viewer.cam
	v := e.doc.Viewer
	from := e.doc.Records[20].Cell
	to := from.Add(image.Pt(3, 2))
	x, y := e.cellCenter(from)
	sx, sy := v.cam.WorldToScreen(x, y)
	tx, ty := e.cellCenter(to)
	tx, ty = v.cam.WorldToScreen(tx, ty)
	now := time.Unix(1, 0)
	e.Pointer(Input{CursorX: int(sx), CursorY: int(sy), PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: int(tx), CursorY: int(ty), PrimaryDown: true, WheelY: 1, PanRight: true}, now)
	if p.index != -1 || e.Dirty() || e.doc.Records[20].Cell != from || !reflect.DeepEqual(cam, *v.cam) {
		t.Fatal("preview committed an edit or pan/zoom escaped capture")
	}
	lines := e.markerLines()
	if len(lines) != 6 || lines[0].x0 != float32(tx-5) || lines[0].y0 != float32(ty) {
		t.Fatalf("drag preview not at candidate cell: %+v", lines)
	}
	e.Pointer(Input{CursorX: int(tx), CursorY: int(ty)}, now)
	if p.index != 63 || e.Selected() != 20 || e.doc.Records[20].Cell != to || e.doc.Viewer != v || v.entities[0].Cell != to {
		t.Fatal("committed wrong unit identity or left entity presentation stale")
	}
	if e.filter != 2 || e.offset != 3 || e.detailOffset != 5 || !reflect.DeepEqual(cam, *v.cam) {
		t.Fatalf("edit lost inspection context: filter=%d offset=%d detail=%d", e.filter, e.offset, e.detailOffset)
	}
	if e.markerLines()[0].x0 != float32(tx-5) {
		t.Fatal("committed marker remained at old cell")
	}
	if got := e.PanelImage(768).RGBAAt(2, 43); got != editorGold {
		t.Fatalf("enabled Undo not visible: %v", got)
	}
	e.railClick(10, 45)
	e.railClick(115, 45)
	if p.redo != 0 {
		t.Fatal("disabled Redo dispatched")
	}
	p.dirty = false // next probe state enables Redo
	e.railClick(115, 45)
	if p.undo != 1 || p.redo != 1 {
		t.Fatal("visible Undo/Redo buttons do not dispatch")
	}
	p.dirty = true
	e.railClick(230, 45)
	if !e.pathMode || !e.pathSave || e.path != "" {
		t.Fatal("Save As button did not open an explicit empty path field")
	}
	e.TextInput("new.alm")
	e.railClick(10, 12) // modal path input owns the rail
	if !e.pathMode || e.catalog {
		t.Fatal("rail click escaped path prompt")
	}
	if err := e.Key("enter"); err == nil || !e.pathMode || !e.Dirty() {
		t.Fatal("failed Save As cleared prompt or dirty state")
	}
}

func TestMapEditor1118GroundDragStillPans(t *testing.T) {
	e := editor1092(t)
	e.Layout(1024, 768)
	e.doc.Edits = &editProbe1118{doc: e.doc}
	e.Select(20, true)
	x, y := e.cellCenter(image.Pt(24, 22)) // no record under this cell
	sx, sy := e.doc.Viewer.cam.WorldToScreen(x, y)
	before := e.doc.Viewer.cam.X
	now := time.Unix(1, 0)
	e.Pointer(Input{CursorX: int(sx), CursorY: int(sy), PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: int(sx) - 32, CursorY: int(sy), PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: int(sx) - 32, CursorY: int(sy)}, now)
	if e.doc.Viewer.cam.X <= before || e.Dirty() {
		t.Fatal("ground drag no longer pans or edited a unit")
	}
}

func TestMapEditor1118LongPathAndRefusalNeverOverlap(t *testing.T) {
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(800, 600), image.Pt(1280, 800)} {
		e := editor1092(t)
		e.Layout(size.X, size.Y)
		e.pathMode, e.pathSave = true, true
		e.path = strings.Repeat("long-directory/", 60) + "chosen-map.alm"
		e.message = "Save As refused: the explicit output directory does not exist."
		rows := e.modalLayout(size.Y)
		path, refusal := false, false
		last := 0
		for _, row := range rows {
			if row.y < last || row.y+e.font.Height() > size.Y-88 || e.font.Advance(row.text) > 300 {
				t.Fatalf("overlapping/clipped modal at %v: %+v", size, rows)
			}
			last = row.y + max(16, e.font.Height()+2)
			path = path || strings.HasSuffix(row.text, "_")
			refusal = refusal || row.color == editorWarning
		}
		if !path || !refusal {
			t.Fatalf("path caret or error disappeared at %v: %+v", size, rows)
		}
	}
}
