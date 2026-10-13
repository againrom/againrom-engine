package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// brushProbe is a host that paints: a stroke returns a document whose tiles
// carry the stroke's value on its first cell.
type brushProbe struct {
	noUnitAuthoring
	doc     *InspectionDocument
	strokes []BrushStroke
	dirty   bool
}

func (p *brushProbe) Dirty() bool                        { return p.dirty }
func (p *brushProbe) CanUndo() bool                      { return p.dirty }
func (p *brushProbe) CanRedo() bool                      { return false }
func (p *brushProbe) Undo() (*InspectionDocument, error) { return nil, nil }
func (p *brushProbe) Redo() (*InspectionDocument, error) { return nil, nil }
func (p *brushProbe) SaveAs(string) (string, error)      { return "", nil }
func (p *brushProbe) BrushTerrains() []BrushTerrain {
	return []BrushTerrain{{Class: 2, Name: "Grass"}, {Class: 8, Name: "Mountain"}}
}
func (p *brushProbe) Move(string, int, image.Point) (*InspectionDocument, error) { return nil, nil }

func (p *brushProbe) Paint(s BrushStroke) (*InspectionDocument, error) {
	p.strokes = append(p.strokes, s)
	p.dirty = true
	g := p.doc.Viewer.grid
	g.Tiles = append([]uint16(nil), g.Tiles...)
	g.Tiles[s.Cells[0].Y*g.Width+s.Cells[0].X] = uint16(s.Value)
	v, err := NewViewer("painted", g, &terrain.Tileset{})
	if err != nil {
		return nil, err
	}
	next := *p.doc
	next.Viewer = v
	p.doc = &next
	return &next, nil
}

func brushEditor(t *testing.T) (*MapEditor, *brushProbe) {
	t.Helper()
	open := func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("brush", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		return &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("untouched")}, nil
	}
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "first", Label: "First"}}, open)
	if err := e.Open("first"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1024, 768)
	p := &brushProbe{doc: e.doc}
	e.doc.Edits = p
	return e, p
}

func drag(e *MapEditor, cells ...image.Point) {
	now := time.Unix(1, 0)
	for _, c := range cells {
		pt := e.CellPoint(c)
		e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y, PrimaryDown: true}, now)
	}
	pt := e.CellPoint(cells[len(cells)-1])
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, now)
}

func TestMapEditorBrushStrokeIsOneEditAndKeepsTheView(t *testing.T) {
	e, p := brushEditor(t)
	if err := e.Key("terrain"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("down"); err != nil {
		t.Fatal(err)
	}
	e.ShowCell(image.Pt(20, 20))
	cam := *e.doc.Viewer.cam
	old := e.doc.Viewer
	drag(e, image.Pt(20, 20), image.Pt(23, 20))
	if len(p.strokes) != 1 {
		t.Fatalf("a drag made %d strokes, want 1", len(p.strokes))
	}
	s := p.strokes[0]
	if s.Height || s.Value != 8 || s.Radius != brushDefaultRadius || len(s.Cells) != 4 || s.Cells[0] != image.Pt(20, 20) || s.Cells[3] != image.Pt(23, 20) {
		t.Fatalf("stroke %+v", s)
	}
	if e.doc.Viewer == old || e.doc.Viewer.cam.Zoom != cam.Zoom || e.doc.Viewer.cam.X != cam.X || e.doc.Viewer.cam.Y != cam.Y {
		t.Fatal("the painted document did not replace the viewer with the camera carried over")
	}
	if e.doc.Viewer.grid.Tiles[20*50+20] != 8 {
		t.Fatal("the window shows the old tiles")
	}
}

func TestMapEditorBrushHeightRowsAndModeSwitch(t *testing.T) {
	e, p := brushEditor(t)
	e.Key("height")
	e.Key("down")
	e.Key("radiusdown")
	e.Key("radiusdown")
	e.Key("radiusdown")
	drag(e, image.Pt(10, 10))
	if len(p.strokes) != 1 || !p.strokes[0].Height || p.strokes[0].Value != -brushHeightStep || p.strokes[0].Radius != 0 {
		t.Fatalf("height stroke %+v", p.strokes)
	}
	e.Key("terrain")
	if e.brush.mode != brushTerrain {
		t.Fatal("T did not switch to terrain mode")
	}
	e.Key("escape")
	drag(e, image.Pt(11, 10))
	if len(p.strokes) != 1 {
		t.Fatal("a stroke was made outside a brush mode")
	}
}

func TestMapEditorBrushHelpLinesFitTheRail(t *testing.T) {
	e, _ := brushEditor(t)
	for _, mode := range []brushMode{brushTerrain, brushHeight} {
		e.brush.mode = mode
		for _, line := range e.helpLines() {
			if w := e.font.Advance(line); w > 300 {
				t.Errorf("%q is %d wide; the rail holds 300", line, w)
			}
		}
	}
}
