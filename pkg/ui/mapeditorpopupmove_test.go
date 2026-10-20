package ui

import (
	"image"
	"testing"
	"time"
)

// dragTitle drags the front popup's title bar by d.
func dragTitle(t *testing.T, e *MapEditor, d image.Point) (image.Rectangle, image.Rectangle) {
	t.Helper()
	before, ok := e.TitleRect()
	if !ok {
		t.Fatal("no popup")
	}
	p := before.Min.Add(image.Pt(10, 4))
	q := p.Add(d)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	after, _ := e.TitleRect()
	return before, after
}

// The map panel, the notice and the group popup move by their title bars
// through the one drag, and the window's edges stop them.
func TestEditorPanelNoticeAndGroupMoveByTitle(t *testing.T) {
	for _, c := range []struct {
		name string
		open func(t *testing.T) *MapEditor
	}{
		{"map panel", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			if e.listKind() != "panel" {
				t.Fatalf("panel not open: %q", e.listKind())
			}
			return e
		}},
		{"notice", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			_ = e.Key("testplay")
			if e.prompt.kind != promptNotice {
				t.Fatal("no notice")
			}
			return e
		}},
		{"group popup", func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			openGroup(t, e)
			if _, _, ok := e.groupSpec(); !ok {
				t.Fatal("no group popup")
			}
			return e
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := c.open(t)
			d := image.Pt(-40, 30)
			if r, _ := e.TitleRect(); r.Min.X < 100 {
				d.X = 40
			}
			before, after := dragTitle(t, e, d)
			if after.Min != before.Min.Add(d) {
				t.Fatalf("title %v moved to %v, not by %v", before, after, d)
			}
			_, far := dragTitle(t, e, image.Pt(-5000, -5000))
			if bounds := e.popupBounds(); far.Min != bounds.Min {
				t.Fatalf("a drag past the corner left the title at %v, not %v", far, bounds.Min)
			}
		})
	}
}
