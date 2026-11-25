package game

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// On a second-game map the passability brush's stroke shows in the fill at
// once, the hover names the mark and its value, and a type-3 cell shows the
// value 5; the view's word says the effect on movement is not established.
func TestSecondMapPassabilityBrushShowsTheMarkedCells(t *testing.T) {
	raw := secondFixtureMap()
	plane3 := alm.FileHeaderSize + 4*alm.RecordHeaderSize + 660 + 3*32*32
	raw[plane3+12*32+20] = 3
	path := filepath.Join(t.TempDir(), "second.alm")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	x := secondTestInspector(t)
	e := ui.NewMapEditor(nil, nil, x.Open)
	e.SetWords(words.For(""))
	e.Layout(1920, 1080)
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	_ = e.Key("passability")
	stroke, type3 := image.Pt(12, 20), image.Pt(20, 12)
	e.ShowCell(stroke)
	hover := func(c image.Point) string {
		p := e.CellPoint(c)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
		return e.StatusLine()
	}
	if e.FilledCell(stroke) || !e.FilledCell(type3) {
		t.Fatalf("before the stroke: stroke cell filled %v, type-3 cell filled %v", e.FilledCell(stroke), e.FilledCell(type3))
	}
	if s := hover(type3); !strings.Contains(s, x.word("editor.mark.type3")) || !strings.Contains(s, "5") {
		t.Fatalf("type-3 hover %q", s)
	}
	p := e.CellPoint(stroke)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(2, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(3, 0))
	if !e.FilledCell(stroke) {
		t.Fatal("the stroke's cell shows no fill")
	}
	if s := hover(stroke); !strings.Contains(s, x.word("editor.mark.flag")) {
		t.Fatalf("stroke hover %q", s)
	}
	_ = e.Key("passview")
	if s := e.StatusLine(); !strings.Contains(s, x.word("editor.view.secondpass")) {
		t.Fatalf("view word %q", s)
	}
}
