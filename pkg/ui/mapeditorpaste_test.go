package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// pastePreviewHost is the terrain host with a placement preview: the map
// after a paste holds one more unit, Unit 2, on the paste's cell. asked is
// every cell it previewed.
type pastePreviewHost struct {
	*terrainHost
	asked []image.Point
}

func (p *pastePreviewHost) PreviewPlacement(r InspectionNewRecord) (*InspectionDocument, error) {
	if r.Clip == nil {
		return nil, fmt.Errorf("not a paste")
	}
	p.asked = append(p.asked, r.Cell)
	d := p.doc(p.self)
	d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: 2, Section: 6, Cell: r.Cell, Spatial: true, Size: image.Pt(1, 1)})
	d.Viewer.SetEntities([]MapEntity{{ID: 2, Cell: r.Cell, Frame: sackFrame(18, 25), Art: &terrain.UnitClass{TileSize: 1}}})
	return d, nil
}

// pasteEditor is the terrain editor over the preview host with the box
// 8,8-12,11 copied: a clip of Unit 0 on a 5 x 4 terrain patch.
func pasteEditor(t *testing.T) (*MapEditor, *pastePreviewHost) {
	t.Helper()
	e, h, _ := terrainEditor(t)
	p := &pastePreviewHost{terrainHost: h}
	h.self, e.doc.Edits = p, p
	e.Fit()
	boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
	_ = e.Key("copy")
	if e.selection.clip == nil || e.selection.clip.Patch() != image.Pt(5, 4) {
		t.Fatalf("the copy left the clip %v", e.selection.clip)
	}
	h.calls.lines = nil
	return e, p
}

func pastePoint(e *MapEditor, cell image.Point) image.Point {
	p := e.CellPoint(cell)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	return p
}

// Ctrl+V with a clip arms the paste brush and edits nothing; the status line
// names it. Loss control: the key's former immediate paste calls
// PasteRecords at the key.
func TestPasteKeyArmsTheBrushWithoutAnEdit(t *testing.T) {
	e, h := pasteEditor(t)
	doc := e.doc
	pastePoint(e, image.Pt(20, 20))
	_ = e.Key("paste")
	if len(h.calls.lines) != 0 || e.doc != doc || e.Dirty() {
		t.Fatalf("Ctrl+V edited: %q", h.calls.lines)
	}
	if e.pick != pickPaste {
		t.Fatalf("Ctrl+V armed %q", e.pick)
	}
	if s := e.statusText(); !strings.Contains(s, e.word("editor.selection.armed")) {
		t.Fatalf("status %q does not name the armed paste", s)
	}
}

// While the brush is armed, the clip's records and terrain patch are drawn
// translucent with the clip's top-left cell on the cell under the pointer,
// and they move with the pointer. Loss controls: without the paste brush in
// placement() no frame stands; the ground drawn at full opacity fails the
// alpha check.
func TestPasteBrushGhostFollowsThePointer(t *testing.T) {
	e, h := pasteEditor(t)
	_ = e.Key("paste")
	var x []float64
	for _, cell := range []image.Point{image.Pt(20, 20), image.Pt(30, 25)} {
		pastePoint(e, cell)
		f, ok := e.placement()
		if !ok || f.Cell != cell || f.Refused {
			t.Fatalf("at %v: frame %+v open=%v", cell, f, ok)
		}
		if len(f.Sprites) != 1 || !f.Sprites[0].Translucent {
			t.Fatalf("at %v: sprites %+v, want the pasted unit translucent", cell, f.Sprites)
		}
		x = append(x, f.Sprites[0].X)
		if want := image.Rect(cell.X, cell.Y, cell.X+5, cell.Y+4); f.Ground != want || f.GroundView == nil {
			t.Fatalf("at %v: ground %v, want %v", cell, f.Ground, want)
		}
		rec := &recordingTarget{}
		e.drawPlacementGround(rec, f)
		if len(rec.calls) != 20 {
			t.Fatalf("at %v: the ground drew %d cells, want the patch's 20", cell, len(rec.calls))
		}
		for _, c := range rec.calls {
			for _, v := range c.verts {
				if v.ColorA != editorGhostAlpha {
					t.Fatalf("at %v: ground vertex alpha %v, want %v", cell, v.ColorA, editorGhostAlpha)
				}
			}
		}
	}
	if !slices.Equal(h.asked, []image.Point{image.Pt(20, 20), image.Pt(30, 25)}) || x[0] == x[1] {
		t.Fatalf("previews asked at %v; sprite x %v", h.asked, x)
	}
	if len(h.calls.lines) != 0 {
		t.Fatalf("the ghost edited: %q", h.calls.lines)
	}
	// Control: an unarmed pointer shows no ghost.
	_ = e.Key("escape")
	if f, ok := e.placement(); ok {
		t.Fatalf("an unarmed pointer showed %+v", f)
	}
}

// A left click pastes once with the clip's origin on the clicked cell and
// clears the brush; the clip stays, so a second Ctrl+V arms again. Loss
// control: a click that kept the brush armed pastes on the second click.
func TestPasteBrushClickPastesOnceAndArmsAgain(t *testing.T) {
	e, h := pasteEditor(t)
	_ = e.Key("paste")
	modesClick(e, pastePoint(e, image.Pt(20, 20)))
	want := fmt.Sprintf("PasteRecords %v (20,20)", e.selection.clip)
	if !slices.Equal(h.calls.lines, []string{want}) || e.pick != pickNone {
		t.Fatalf("the click called %q, pick %q", h.calls.lines, e.pick)
	}
	modesClick(e, pastePoint(e, image.Pt(30, 25)))
	if len(h.calls.lines) != 1 {
		t.Fatalf("a click after the paste pasted again: %q", h.calls.lines)
	}
	_ = e.Key("paste")
	if e.pick != pickPaste || len(h.calls.lines) != 1 {
		t.Fatalf("the second Ctrl+V armed %q and called %q", e.pick, h.calls.lines)
	}
	modesClick(e, pastePoint(e, image.Pt(30, 25)))
	if got := lastCall(h.calls); got != fmt.Sprintf("PasteRecords %v (30,25)", e.selection.clip) || len(h.calls.lines) != 2 {
		t.Fatalf("the second paste called %q", h.calls.lines)
	}
}

// Esc, a right click and another brush clear the armed paste with no edit,
// and the next click pastes nothing.
func TestPasteBrushClearsWithNoEdit(t *testing.T) {
	for _, way := range []string{"escape", "right-click", "terrain", "height"} {
		t.Run(way, func(t *testing.T) {
			e, h := pasteEditor(t)
			p := pastePoint(e, image.Pt(20, 20))
			_ = e.Key("paste")
			switch way {
			case "right-click":
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y, SecondaryDown: true}, time.Unix(1, 0))
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
			default:
				_ = e.Key(way)
			}
			if e.pick == pickPaste || len(h.calls.lines) != 0 {
				t.Fatalf("%s left the pick %q, called %q", way, e.pick, h.calls.lines)
			}
			if strings.Contains(e.statusText(), e.word("editor.selection.armed")) {
				t.Fatalf("%s left the status %q", way, e.statusText())
			}
			if way == "terrain" || way == "height" {
				_ = e.Key(way)
			}
			modesClick(e, p)
			for _, l := range h.calls.lines {
				if strings.HasPrefix(l, "PasteRecords") {
					t.Fatalf("a click after %s pasted: %q", way, h.calls.lines)
				}
			}
		})
	}
}
