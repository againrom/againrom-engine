package ui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"againrom/pkg/words"
)

// selectionDrag drives a primary drag on the map from cell a to cell b with
// Shift held or not, at the fitted view.
func selectionDrag(e *MapEditor, a, b image.Point, shift bool) {
	e.Fit()
	p, q := e.CellPoint(a), e.CellPoint(b)
	now := time.Unix(1, 0)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true, Shift: shift}, now)
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true, Shift: shift}, now)
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y, Shift: shift}, now)
}

// selectionClick is a Shift click on cell.
func selectionClick(e *MapEditor, cell image.Point) {
	e.Fit()
	p := e.CellPoint(cell)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true, Shift: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, Shift: true}, time.Unix(1, 0))
}

// lastCall is the host's last logged call.
func lastCall(c *modesCalls) string {
	if len(c.lines) == 0 {
		return ""
	}
	return c.lines[len(c.lines)-1]
}

var (
	refUnit0 = RecordRef{"Unit", 0}
	refUnit1 = RecordRef{"Unit", 1}
)

// The modes host places Unit 0 at 10,10, Unit 1 at 20,10, a 2x2 Structure at
// 30,30 and a Sack at 10,30.
func TestMapEditorRectangleSelectsExactlyTheRecordsInside(t *testing.T) {
	cases := []struct {
		a, b image.Point
		want []RecordRef
	}{
		{image.Pt(9, 9), image.Pt(20, 10), []RecordRef{refUnit0, refUnit1}},
		{image.Pt(20, 10), image.Pt(9, 9), []RecordRef{refUnit0, refUnit1}},
		{image.Pt(9, 9), image.Pt(19, 10), []RecordRef{refUnit0}},
		{image.Pt(11, 11), image.Pt(29, 29), nil},
		{image.Pt(0, 0), image.Pt(49, 49), []RecordRef{refUnit0, refUnit1, {"Structure", 0}, {"Sack", 0}}},
	}
	for _, c := range cases {
		e, _ := newModesEditor(t, "full")
		selectionDrag(e, c.a, c.b, true)
		if !slices.Equal(e.selection.refs, c.want) {
			t.Errorf("rectangle %v-%v selected %v, want %v", c.a, c.b, e.selection.refs, c.want)
		}
	}
	// A plain drag from bare ground draws the same box; with a brush on, the
	// drag is the brush's and selects nothing.
	e, _ := newModesEditor(t, "full")
	selectionDrag(e, image.Pt(9, 9), image.Pt(20, 10), false)
	if !slices.Equal(e.selection.refs, []RecordRef{refUnit0, refUnit1}) {
		t.Errorf("a plain drag selected %v", e.selection.refs)
	}
	e, _ = newModesEditor(t, "full")
	e.brush.mode = brushTerrain
	if !e.brushActive() {
		t.Fatal("the fixture has no terrain brush")
	}
	selectionDrag(e, image.Pt(9, 9), image.Pt(20, 10), false)
	if len(e.selection.refs) != 0 {
		t.Errorf("a brush drag selected %v", e.selection.refs)
	}
}

func TestMapEditorShiftClickAddsAndRemovesARecord(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.Select(0, false)
	selectionClick(e, image.Pt(20, 10))
	if !slices.Equal(e.selection.refs, []RecordRef{refUnit0, refUnit1}) || !e.selectionShown() {
		t.Fatalf("Shift click on Unit 1 left %v", e.selection.refs)
	}
	if s := e.statusText(); !strings.Contains(s, "2 selected") {
		t.Errorf("status %q does not count the group", s)
	}
	lines := e.markerLines()
	selectionClick(e, image.Pt(20, 10))
	if !slices.Equal(e.selection.refs, []RecordRef{refUnit0}) || e.selectionShown() {
		t.Errorf("a second Shift click left %v", e.selection.refs)
	}
	// Both members carry the selected record's marks; one alone carries fewer.
	if after := e.markerLines(); len(after) >= len(lines) {
		t.Errorf("marks: group %d, one record %d", len(lines), len(after))
	}
	// A plain click elsewhere drops the group.
	selectionClick(e, image.Pt(20, 10))
	e.Fit()
	p := e.CellPoint(image.Pt(25, 25))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	if len(e.selection.refs) != 0 {
		t.Errorf("a click on an empty cell kept %v", e.selection.refs)
	}
}

func TestMapEditorGroupDragMovesTheGroupByTheRecordsOffset(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	modesAct(e, "ui:group")
	selectionDrag(e, image.Pt(10, 10), image.Pt(14, 12), false)
	if got, want := lastCall(calls), fmt.Sprintf("MoveRecords %v %v", []RecordRef{refUnit0, refUnit1}, image.Pt(4, 2)); got != want {
		t.Errorf("group drag called %q, want %q", got, want)
	}
	// Control: one selected record moves alone.
	e, calls = newModesEditor(t, "full")
	e.Select(0, false)
	selectionDrag(e, image.Pt(10, 10), image.Pt(14, 12), false)
	if got := lastCall(calls); got != "Move Unit 0 (14,12)" {
		t.Errorf("single drag called %q", got)
	}
}

func TestMapEditorGroupKeys(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	modesAct(e, "ui:group")
	if e.helpScreen() != screenSelection {
		t.Fatalf("help screen %d with a group", e.helpScreen())
	}
	_ = e.Key("copy")
	if got := lastCall(calls); got != fmt.Sprintf("CopyRecords %v", []RecordRef{refUnit0, refUnit1}) {
		t.Errorf("copy called %q", got)
	}
	e.Fit()
	p := e.CellPoint(image.Pt(25, 25))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	_ = e.Key("paste")
	modesClick(e, p)
	if got := lastCall(calls); got != fmt.Sprintf("PasteRecords %v %v", modesClip{refUnit0, refUnit1}, image.Pt(25, 25)) {
		t.Errorf("paste called %q", got)
	}
	if s := e.statusText(); !strings.Contains(s, "group edit") {
		t.Errorf("status %q lacks the paste's note", s)
	}
	// The copy outlives the map: another map takes the same paste.
	if err := e.openMap("map1"); err != nil {
		t.Fatal(err)
	}
	e.Fit()
	p = e.CellPoint(image.Pt(5, 6))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	_ = e.Key("paste")
	modesClick(e, p)
	if got := lastCall(calls); got != fmt.Sprintf("PasteRecords %v %v", modesClip{refUnit0, refUnit1}, image.Pt(5, 6)) {
		t.Errorf("paste in the second map called %q", got)
	}
	modesAct(e, "ui:group")
	_ = e.Key("delete")
	if got := lastCall(calls); got != fmt.Sprintf("DeleteRecords %v", []RecordRef{refUnit0, refUnit1}) {
		t.Errorf("delete called %q", got)
	}
	modesAct(e, "ui:group")
	_ = e.Key("escape")
	if len(e.selection.refs) != 0 || e.Selected() != 1 {
		t.Errorf("escape left the group %v, selected %d", e.selection.refs, e.Selected())
	}
	// Control: one selected record deletes alone, and a paste with nothing
	// copied calls nothing and says so.
	e, calls = newModesEditor(t, "full")
	e.Select(0, false)
	_ = e.Key("delete")
	if got := lastCall(calls); got != "Delete Unit 0" {
		t.Errorf("single delete called %q", got)
	}
	n := len(calls.lines)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	_ = e.Key("paste")
	if len(calls.lines) != n || e.selection.note != e.word("editor.selection.noclip") {
		t.Errorf("paste with nothing copied: calls %v, note %q", calls.lines[n:], e.selection.note)
	}
}

// russianWord reports whether s is Russian: it has a Cyrillic letter and
// every Latin letter belongs to a key name.
func russianWord(s string) bool {
	cyrillic := false
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && r != '+' && !unicode.IsDigit(r) }) {
		switch f {
		case "Shift", "Alt", "Ctrl+A", "Ctrl+Z", "Ctrl+C", "Ctrl+X", "Ctrl+V", "Ctrl+Left", "Ctrl+Right", "Left", "Right", "Backspace", "Ctrl+E", "Ctrl+Shift+R", "Ctrl+1", "Esc", "Del", "Enter", "Tab", "F1", "F2", "F3", "F5", "F6", "F7":
			continue
		}
		if len(f) == 1 && f[0] >= 'A' && f[0] <= 'Z' {
			continue // a letter key
		}
		for _, r := range f {
			if unicode.Is(unicode.Latin, r) {
				return false
			}
			cyrillic = cyrillic || unicode.Is(unicode.Cyrillic, r)
		}
	}
	return cyrillic
}

// brushWords is every word the brushes show: the Tools menu, each brush's
// rows and title, the passability status words, the height readout and the
// help lines of the brush screens and of the keys that switch them.
func brushWords(t *testing.T, e *MapEditor) []string {
	t.Helper()
	var out []string
	for _, it := range e.menuItems("tools") {
		out = append(out, strings.TrimPrefix(strings.TrimPrefix(it.label, "[x] "), "[ ] "))
	}
	for _, mode := range []brushMode{brushTerrain, brushHeight, brushObjects, brushPassability} {
		e.chooseBrush(mode)
		if e.brush.mode != mode {
			t.Fatalf("brush %d did not turn on", mode)
		}
		out = append(out, e.brushRows()[0], e.brushStatus())
		if mode != brushObjects {
			out = append(out, e.brushRows()...)
		}
		out = append(out, e.helpLines()...)
	}
	for arm := PassBorder; arm <= PassStructure; arm <<= 1 {
		out = append(out, e.passArmWord(arm))
	}
	out = append(out, e.word("editor.pass.open"), fmt.Sprintf(e.word("editor.pass.closed"), e.passArmWord(PassWater)), fmt.Sprintf(e.word("editor.height.cell"), 3))
	for _, l := range []helpLine{lineBrush, linePassability} {
		out = append(out, e.word(l.Word))
	}
	return out
}

// paletteHost lists the ten palette terrains by their model names.
type paletteHost struct{ brushProbe }

func (p *paletteHost) BrushTerrains() []BrushTerrain {
	var out []BrushTerrain
	for i, n := range []string{"Grass", "Land", "Sand", "Cracked", "Flowers", "Stones", "Savanna", "Mountain", "Water", "Road"} {
		out = append(out, BrushTerrain{Class: i, Name: n})
	}
	return out
}

func TestMapEditorBrushWordsAreTheInstallsLanguage(t *testing.T) {
	ru, b := brushEditor(t)
	ru.doc.Edits = &paletteHost{*b}
	ru.SetWords(words.For("russian"))
	got := brushWords(t, ru)
	for _, w := range got {
		if !russianWord(w) {
			t.Errorf("Russian install shows %q", w)
		}
	}
	en, b := brushEditor(t)
	en.doc.Edits = &paletteHost{*b}
	gotEN := brushWords(t, en)
	for _, want := range []string{"Terrain", "Brushes", "Grass", "Road", "Raise", "Lower", "Block", "Unblock", "No object", "Terrain: Grass  size 2", "T: off / Drag: paint / [ ]: size", "Passable", "Water"} {
		if !slices.Contains(gotEN, want) {
			t.Errorf("English install lacks %q", want)
		}
	}
	for _, w := range gotEN {
		if russianWord(w) {
			t.Errorf("English install shows %q", w)
		}
	}
	if len(got) != len(gotEN) || len(got) < 40 {
		t.Errorf("%d Russian words, %d English", len(got), len(gotEN))
	}
	// Control: the check refuses an English word and an English help line.
	for _, w := range []string{"Raise", "T: off / Drag: paint / [ ]: size", "Grass"} {
		if russianWord(w) {
			t.Errorf("the check takes %q as Russian", w)
		}
	}
}

func TestMapEditorF1KeysAreTheInstallsLanguage(t *testing.T) {
	for _, line := range []string{"Ctrl+A: select all", "Ctrl+Left/Right: by words", "Backspace: repeat"} {
		if russianWord(line) {
			t.Fatalf("English instruction accepted: %q", line)
		}
	}
	e, _ := newModesEditor(t, "full")
	e.SetWords(words.For("russian"))
	_ = e.Key("help")
	rows, open := e.ReportOpen()
	if !open || len(rows) < 30 {
		t.Fatalf("F1 list open %v with %d rows", open, len(rows))
	}
	for _, r := range e.report.rows[:len(e.report.rows)-1] {
		if !russianWord(r.Text) {
			t.Errorf("Russian F1 row %q", r.Text)
		}
	}
	// Each list row shows its line, not the report's rule and place.
	shown, _ := e.listRows("report")
	for i, r := range e.report.rows {
		if shown[i].Text != r.Text {
			t.Errorf("F1 row %d shows %q, not %q", i, shown[i].Text, r.Text)
		}
	}
	if e.report.note != "Клавиши редактора карт" {
		t.Errorf("F1 title %q", e.report.note)
	}
	_ = e.Key("help")
	if _, open := e.ReportOpen(); open {
		t.Error("F1 does not close its own Russian list")
	}
}

// The passability fills stay drawn at the smallest zoom, where the grid
// stops: a closed cell still reads red and an open one white.
func TestMapEditorPassabilityReadsAtTheSmallestZoom(t *testing.T) {
	e, _ := passEditor(t)
	e.view.passability = true
	v := e.doc.Viewer
	v.showGrid = true
	v.cam.SetZoom(0)
	v.syncWorld()
	if px := 32 * v.cam.Zoom; px >= GridMinCellPixels {
		t.Fatalf("smallest zoom draws a cell %v pixels wide", px)
	}
	if segs := v.gridScreenSegments(); len(segs) != 0 {
		t.Fatalf("the grid draws %d segments at the smallest zoom", len(segs))
	}
	quads := e.passabilityQuads()
	if len(quads) < e.doc.Width*e.doc.Height/2 {
		t.Fatalf("%d fills at the smallest zoom", len(quads))
	}
	canvas := image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))
	grey := color.RGBA{0x80, 0x80, 0x80, 0xff}
	for i := 0; i < len(canvas.Pix); i += 4 {
		canvas.Pix[i], canvas.Pix[i+1], canvas.Pix[i+2], canvas.Pix[i+3] = grey.R, grey.G, grey.B, grey.A
	}
	e.paintPassability(canvas)
	at := func(cell image.Point) color.RGBA {
		x, y := v.cellScreenCentre(cell)
		return canvas.RGBAAt(int(x), int(y))
	}
	if c := at(image.Pt(30, 30)); !(c.R > c.G+0x20) {
		t.Errorf("closed cell reads %v", c)
	}
	if c := at(image.Pt(25, 25)); !(c.R > grey.R && c.G > grey.G && c.B > grey.B) {
		t.Errorf("open cell reads %v", c)
	}
	// Control: with the view off nothing is filled.
	e.view.passability = false
	if q := e.passabilityQuads(); len(q) != 0 {
		t.Errorf("view off: %d fills", len(q))
	}
}

func TestPassabilityBatchesHoldEveryFill(t *testing.T) {
	quads := make([]passQuad, passBatchQuads+100)
	for i := range quads {
		quads[i].fill = passClosedFill
	}
	verts, indices := passBatches(quads)
	if len(verts) != 2 || len(verts[0]) != 4*passBatchQuads || len(verts[1]) != 400 || len(indices[1]) != 600 {
		t.Fatalf("batches %d, sizes %d %d", len(verts), len(verts[0]), len(verts[len(verts)-1]))
	}
	for b := range verts {
		for _, k := range indices[b] {
			if int(k) >= len(verts[b]) {
				t.Fatalf("batch %d index %d past %d vertices", b, k, len(verts[b]))
			}
		}
		if verts[b][0].ColorA != float32(passClosedFill.A)/255 {
			t.Errorf("batch %d colour %v", b, verts[b][0].ColorA)
		}
	}
	if v, i := passBatches(nil); len(v) != 0 || len(i) != 0 {
		t.Error("no fills drew a batch")
	}
}
