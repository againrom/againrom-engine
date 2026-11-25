package game

import (
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A second-game map's document carries each cell's terrain-build value with
// the move kinds it closes, and its passability brush offers the two rows its
// file bytes reach; a stroke of the first over a type-3 cell leaves it closed
// to move kind 1 alone, and the hover says so.
func TestSecondMapPassabilityShowsAndSetsMoveKinds(t *testing.T) {
	raw := secondFixtureMap()
	tiles := alm.FileHeaderSize + 2*alm.RecordHeaderSize + 660
	plane3 := alm.FileHeaderSize + 4*alm.RecordHeaderSize + 660 + 3*32*32
	raw[plane3+12*32+20] = 3
	binary.LittleEndian.PutUint16(raw[tiles+2*(20*32+12):], alm.TileImpassable)
	x := secondTestInspector(t)
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	value := func(c image.Point) uint8 { return d.PassValues[c.Y*d.Width+c.X] }
	type3, flag, open, ring := image.Pt(20, 12), image.Pt(12, 20), image.Pt(14, 22), image.Pt(3, 3)
	if value(type3) != 5 || value(flag) != 1 || value(open) != 0 || value(ring) != 0x1f {
		t.Fatalf("values: type-3 %d, flag %d, open %d, ring %d", value(type3), value(flag), value(open), value(ring))
	}
	kinds := map[uint8]uint8{}
	for _, r := range d.PassValueRows {
		kinds[r.Value] = r.Kinds
	}
	if kinds[0] != 0 || kinds[1] != 1 || kinds[5] != 3 || kinds[0x1f] != 7 {
		t.Fatalf("value rows %+v", d.PassValueRows)
	}
	if want := []ui.PassBrushRow{{Value: 1, Word: "editor.brush.secondclose"}, {Value: 0, Word: "editor.brush.secondopen"}}; !slices.Equal(d.PassBrush, want) {
		t.Fatalf("brush rows %+v", d.PassBrush)
	}

	path := filepath.Join(t.TempDir(), "second.alm")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, x.Open)
	e.SetWords(words.For(""))
	e.Layout(1920, 1080)
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	_ = e.Key("passability")
	e.ShowCell(flag)
	hover := func(c image.Point) string {
		p := e.CellPoint(c)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
		return e.StatusLine()
	}
	if s := hover(type3); !strings.Contains(s, x.word("editor.pass.value.5")) || !strings.Contains(s, x.word("editor.mark.type3")) {
		t.Fatalf("type-3 hover %q", s)
	}
	if s := hover(open); !strings.Contains(s, x.word("editor.pass.value.0")) {
		t.Fatalf("open hover %q", s)
	}
	p := e.CellPoint(type3)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(2, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(3, 0))
	if s := hover(type3); !strings.Contains(s, x.word("editor.pass.value.1")) || strings.Contains(s, x.word("editor.mark.type3")) {
		t.Fatalf("hover after the close stroke %q", s)
	}
}
