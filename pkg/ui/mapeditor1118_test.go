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

type editorMoveProbe struct {
	noUnitAuthoring
	doc               *InspectionDocument
	index, undo, redo int
	dirty             bool
}

func (p *editorMoveProbe) Dirty() bool   { return p.dirty }
func (p *editorMoveProbe) CanUndo() bool { return p.dirty }
func (p *editorMoveProbe) CanRedo() bool { return !p.dirty }
func (p *editorMoveProbe) Undo() (*InspectionDocument, error) {
	p.undo++
	return nil, nil
}
func (p *editorMoveProbe) Redo() (*InspectionDocument, error) {
	p.redo++
	return nil, nil
}
func (p *editorMoveProbe) SaveAs(string) (string, error) {
	return "", fmt.Errorf("test writer refused")
}
func (p *editorMoveProbe) Move(_ string, index int, cell image.Point) (*InspectionDocument, error) {
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
	p := &editorMoveProbe{doc: e.doc, index: -1}
	e.doc.Edits = p
	e.Select(20, true)
	e.filter, e.objectTop = 2, 1
	e.objectHidden = true
	cam := *e.doc.Viewer.cam
	v := e.doc.Viewer
	from := e.doc.Records[20].Cell
	to := from.Add(image.Pt(3, 2))
	x, y := e.cellCenter(from)
	sx, sy := v.cam.WorldToScreen(x, y)
	tx, ty := e.cellCenter(to)
	tx, ty = v.cam.WorldToScreen(tx, ty)
	now := time.Unix(1, 0)
	press, drop := mapPoint(e, image.Pt(int(sx), int(sy))), mapPoint(e, image.Pt(int(tx), int(ty)))
	e.Pointer(Input{CursorX: press.X, CursorY: press.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: drop.X, CursorY: drop.Y, PrimaryDown: true, WheelY: 1, PanRight: true}, now)
	if p.index != -1 || e.Dirty() || e.doc.Records[20].Cell != from || !reflect.DeepEqual(cam, *v.cam) {
		t.Fatal("preview committed an edit or pan/zoom escaped capture")
	}
	lines := e.markerLines()
	if len(lines) != 6 || lines[0].x0 != float32(tx-5) || lines[0].y0 != float32(ty) {
		t.Fatalf("drag preview not at candidate cell: %+v", lines)
	}
	e.Pointer(Input{CursorX: drop.X, CursorY: drop.Y}, now)
	if p.index != 63 || e.Selected() != 20 || e.doc.Records[20].Cell != to || e.doc.Viewer != v || v.entities[0].Cell != to {
		t.Fatal("committed wrong unit identity or left entity presentation stale")
	}
	if e.filter != 2 || !reflect.DeepEqual(cam, *v.cam) {
		t.Fatalf("edit lost inspection context: filter=%d", e.filter)
	}
	if e.markerLines()[0].x0 != float32(tx-5) {
		t.Fatal("committed marker remained at old cell")
	}
	for _, b := range e.barEntries() {
		if b.key == "undo" && b.disabled || b.key == "redo" && !b.disabled {
			t.Fatalf("bar entry %s disabled=%v", b.key, b.disabled)
		}
	}
	chromeClick(e, barPoint(t, e, "undo"))
	chromeClick(e, barPoint(t, e, "redo"))
	if p.redo != 0 {
		t.Fatal("disabled Redo dispatched")
	}
	p.dirty = false // next probe state enables Redo
	chromeClick(e, barPoint(t, e, "redo"))
	if p.undo != 1 || p.redo != 1 {
		t.Fatal("visible Undo/Redo entries do not dispatch")
	}
	p.dirty = true
	chromeClick(e, barPoint(t, e, "save"))
	if e.prompt.kind != promptSave || e.prompt.path != "" {
		t.Fatal("Save did not open an explicit empty path field")
	}
	e.TextInput("new.alm")
	chromeClick(e, barPoint(t, e, "map")) // the prompt owns the window
	if !e.prompt.open() || e.catalog || e.menu != "" {
		t.Fatal("a bar click escaped the path prompt")
	}
	if err := e.Key("enter"); err == nil || !e.prompt.open() || !e.Dirty() {
		t.Fatal("failed Save As cleared prompt or dirty state")
	}
}

func TestMapEditor1118GroundDragStillPans(t *testing.T) {
	e := editor1092(t)
	e.Layout(1024, 768)
	e.doc.Edits = &editorMoveProbe{doc: e.doc}
	e.Select(20, true)
	x, y := e.cellCenter(image.Pt(24, 22)) // no record under this cell
	sx, sy := e.doc.Viewer.cam.WorldToScreen(x, y)
	before := e.doc.Viewer.cam.X
	now := time.Unix(1, 0)
	at := mapPoint(e, image.Pt(int(sx), int(sy)))
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: at.X - 32, CursorY: at.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: at.X - 32, CursorY: at.Y}, now)
	if e.doc.Viewer.cam.X <= before || e.Dirty() {
		t.Fatal("ground drag no longer pans or edited a unit")
	}
}

func TestMapEditor1118LongPathAndRefusalNeverOverlap(t *testing.T) {
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(800, 600), image.Pt(1280, 800)} {
		e := editor1092(t)
		e.Layout(size.X, size.Y)
		e.prompt.kind = promptSave
		e.prompt.path = strings.Repeat("long-directory/", 60) + "chosen-map.alm"
		e.message = "Save As refused: the explicit output directory does not exist."
		spec, l := promptLayout(t, e)
		if !l.Rect.In(image.Rect(0, 0, size.X, size.Y)) {
			t.Fatalf("prompt %v leaves the window %v", l.Rect, size)
		}
		last := spec.Rows[len(spec.Rows)-1]
		refusal := false
		for _, n := range spec.Notes {
			refusal = refusal || strings.Contains(n, "does not exist")
		}
		for _, rr := range l.Rows {
			if e.font.Advance(spec.Rows[rr.Index].Value) > rr.Control.Dx() {
				t.Fatalf("clipped path row at %v: %q", size, spec.Rows[rr.Index].Value)
			}
		}
		if !strings.HasSuffix(last.Value, "chosen-map.alm") || !last.Focus || !refusal {
			t.Fatalf("path caret or error disappeared at %v: %+v", size, spec)
		}
	}
}
