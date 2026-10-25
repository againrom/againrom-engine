package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/textfit"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseMapEditorSelectionFrames drives the brush lists, the F1 key list
// and a group selection on an installed map, and writes their frames into
// AGAINROM_SELECTION_FRAMES when it is set: the Brushes menu, each brush's
// list, the key list, a Shift rectangle being drawn and the group it
// selects, the group after a drag, and a pasted copy with its status line.
// Each step's status words are the install language's.
func TestReleaseMapEditorSelectionFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_SELECTION_FRAMES")
	now := time.Unix(1, 0)
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{})
	}
	word := func(id string, a ...any) string {
		if len(a) == 0 {
			return x.inspectionUTF8(x.word(id))
		}
		return x.inspectionUTF8(fmt.Sprintf(x.word(id), a...))
	}
	status := func(e *ui.MapEditor, what string, want ...string) {
		t.Helper()
		s := e.StatusLine()
		for _, w := range want {
			if !strings.Contains(s, w) {
				t.Fatalf("%s: status %q lacks %q", what, s, w)
			}
		}
	}
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)

	r, ok := e.BarRect("brushes")
	if !ok {
		t.Fatal("no Brushes entry")
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
	shot(e, "brushes-menu")
	_ = e.Key("escape")
	for _, b := range []struct{ key, word string }{
		{"terrain", "editor.keys.terrain"}, {"height", "editor.keys.height"},
		{"passability", "editor.keys.passbrush"}, {"objects", "editor.keys.objects"},
	} {
		if err := e.Key(b.key); err != nil {
			t.Fatal(err)
		}
		status(e, b.key+" brush", word(b.word))
		shot(e, "brush-"+b.key)
		_ = e.Key(b.key)
	}
	if err := e.Key("help"); err != nil {
		t.Fatal(err)
	}
	shot(e, "keys")
	rows, _ := e.ReportOpen()
	for i := 0; i < len(rows); i++ {
		if _, visible := e.ListRowRect(i); visible {
			continue
		}
		_ = e.Key("pagedown")
		if _, visible := e.ListRowRect(i); !visible {
			_ = e.Key("pagedown")
		}
		if _, visible := e.ListRowRect(i); !visible {
			t.Fatalf("F1 row %d is unreachable", i)
		}
		shot(e, fmt.Sprintf("keys-%02d", i))
	}
	_ = e.Key("escape")

	// The group: the first unit with two more units, structures or sacks
	// within three cells of it.
	d := e.Document()
	selectable := func(r ui.InspectionRecord) bool {
		return r.Spatial && (r.Kind == "Unit" || r.Kind == "Structure" || r.Kind == "Sack")
	}
	var centre image.Point
	found := false
	for _, a := range d.Records {
		if a.Kind != "Unit" || !a.Spatial {
			continue
		}
		box := image.Rect(a.Cell.X-3, a.Cell.Y-3, a.Cell.X+4, a.Cell.Y+4)
		n := 0
		for _, b := range d.Records {
			if selectable(b) && b.Cell.In(box) {
				n++
			}
		}
		if n >= 3 {
			centre, found = a.Cell, true
			break
		}
	}
	if !found {
		t.Fatal("the map has no three records within three cells of a unit")
	}
	e.ShowCell(centre)
	from, to := centre.Sub(image.Pt(3, 3)), centre.Add(image.Pt(3, 3))
	n := 0
	for _, b := range d.Records {
		if selectable(b) && b.Cell.In(image.Rect(from.X, from.Y, to.X+1, to.Y+1)) {
			n++
		}
	}
	selectRect := func(name string) {
		t.Helper()
		pf, pt := e.CellPoint(from), e.CellPoint(to)
		e.Pointer(ui.Input{CursorX: pf.X, CursorY: pf.Y, PrimaryDown: true, Shift: true}, now)
		e.Pointer(ui.Input{CursorX: pt.X, CursorY: pt.Y, PrimaryDown: true, Shift: true}, now)
		if name != "" {
			shot(e, name)
		}
		e.Pointer(ui.Input{CursorX: pt.X, CursorY: pt.Y, Shift: true}, now)
		status(e, "the rectangle", word("editor.selection.count", n))
	}
	selectRect("selection-rectangle")
	shot(e, "selection-group")

	// Drag the unit at the centre: the group follows. The first offset at
	// which every record of the group finds an open cell is the frame's; an
	// offset that refuses a record is undone.
	moved := 0
	for _, off := range []image.Point{{2, 0}, {0, 2}, {-2, 0}, {0, -2}, {3, 1}, {1, 3}, {-3, -1}, {-1, -3}, {4, 0}, {0, 4}, {-4, 0}, {0, -4}} {
		before := map[ui.RecordRef]image.Point{}
		for _, b := range e.Document().Records {
			before[ui.RecordRef{Kind: b.Kind, Index: b.Index}] = b.Cell
		}
		pc, pd := e.CellPoint(centre), e.CellPoint(centre.Add(off))
		e.Pointer(ui.Input{CursorX: pc.X, CursorY: pc.Y, PrimaryDown: true}, now)
		e.Pointer(ui.Input{CursorX: pd.X, CursorY: pd.Y, PrimaryDown: true}, now)
		e.Pointer(ui.Input{CursorX: pd.X, CursorY: pd.Y}, now)
		moved = 0
		for _, b := range e.Document().Records {
			if c, ok := before[ui.RecordRef{Kind: b.Kind, Index: b.Index}]; ok && b.Cell == c.Add(off) {
				moved++
			}
		}
		if moved == n {
			centre = centre.Add(off)
			break
		}
		t.Logf("offset %v: %s", off, e.StatusLine())
		if e.Dirty() {
			_ = e.Key("undo")
		}
		selectRect("")
	}
	if moved != n {
		t.Fatalf("no offset moved the whole group of %d", n)
	}
	status(e, "the group move", word("editor.selection.moved", n, n))
	shot(e, "selection-moved")

	// Copy the group and paste it at the first cell near it where every
	// record finds room.
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	status(e, "the copy", word("editor.selection.copied", n))
	pasted := false
	for _, off := range []image.Point{{0, 6}, {6, 0}, {0, -6}, {-6, 0}, {6, 6}, {-6, -6}, {6, -6}, {-6, 6}, {0, 9}, {9, 0}} {
		was := bytes.Clone(e.Document().SourceBytes)
		pp := e.CellPoint(centre.Add(off).Sub(image.Pt(3, 3)))
		e.Pointer(ui.Input{CursorX: pp.X, CursorY: pp.Y}, now)
		if err := e.Key("paste"); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(e.StatusLine(), word("editor.selection.pasted", n, n)) {
			pasted = true
			break
		}
		t.Logf("paste at %v: %s", off, e.StatusLine())
		if !bytes.Equal(e.Document().SourceBytes, was) {
			_ = e.Key("undo")
		}
	}
	if !pasted {
		t.Fatal("no cell near the group took the whole paste")
	}
	status(e, "the paste", word("editor.selection.count", n))
	shot(e, "selection-pasted")
	t.Logf("%d records in the rectangle; paste status %q", n, e.StatusLine())
}

// TestReleaseMapEditorPasteTakesAShippedFlier copies a unit of a non-ground
// movement domain (the map's Bees) the installed map places on a Water or
// Mountain cell and pastes it on its own cell: the
// paste takes it. Its control: a ground unit of the same map pasted on that
// cell is refused, with the cell's arms named.
func TestReleaseMapEditorPasteTakesAShippedFlier(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1800, 900)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	arms := mapload.GroundArms(m, x.table)
	flier, ground := -1, -1
	var at image.Point
	for i, u := range m.Units {
		d, err := mapload.UnitDomain(u, x.table)
		c := image.Pt(int(u.X>>8), int(u.Y>>8))
		if err != nil || c.X >= int(m.Width) || c.Y >= int(m.Height) {
			continue
		}
		a := arms[c.Y*int(m.Width)+c.X]
		switch {
		case flier < 0 && d != sim.DomainGround && a&(mapload.GroundWater|mapload.GroundMountain) != 0 && a&mapload.GroundBorder == 0:
			flier, at = i, c
		case ground < 0 && d == sim.DomainGround && a == 0:
			ground = i
		}
	}
	if flier < 0 || ground < 0 {
		t.Fatalf("scenario/10.alm has no flier on Water or Mountain (%d) or no ground unit on an open cell (%d)", flier, ground)
	}
	word := func(id string, a ...any) string { return x.inspectionUTF8(fmt.Sprintf(x.word(id), a...)) }
	copyUnit := func(i int) {
		t.Helper()
		e.Select(recordIndex(t, e, "Unit", i), true)
		if err := e.Key("copy"); err != nil {
			t.Fatal(err)
		}
		if s := e.StatusLine(); !strings.Contains(s, word("editor.selection.copied", 1)) {
			t.Fatalf("unit %d: the copy left %q", i, s)
		}
	}
	copyUnit(flier)
	pasteAt(t, e, at)
	if s := e.StatusLine(); !strings.Contains(s, word("editor.selection.pasted", 1, 1)) {
		t.Fatalf("the flier, unit %d at %v, pasted on its own cell: %q", flier, at, s)
	}
	_ = e.Key("undo")
	copyUnit(ground)
	pasteAt(t, e, at)
	names := (&mapInspectionEdits{inspector: x}).armWords(arms[at.Y*int(m.Width)+at.X])
	refusal := word("editor.selection.closed", fmt.Sprintf(x.word("editor.selection.unit"), ground), at.X, at.Y, strings.Join(names, ", "))
	if s := e.StatusLine(); !strings.Contains(s, word("editor.selection.pasted", 0, 1)) || !strings.Contains(s, refusal) {
		t.Fatalf("the ground unit %d pasted on the flier's cell %v: %q lacks %q", ground, at, s, refusal)
	}
	t.Logf("flier unit %d at %v; ground unit %d refused: %q", flier, at, ground, e.StatusLine())
}

// widestRefusals is the map's own edits with a paste that adds refusals.
type widestRefusals struct {
	*mapInspectionEdits
	refusals []string
}

func (w widestRefusals) PasteRecords(c ui.InspectionClip, cell image.Point) (ui.SelectionEdit, error) {
	out, err := w.mapInspectionEdits.PasteRecords(c, cell)
	out.Refusals = append(slices.Clone(w.refusals), out.Refusals...)
	return out, err
}

// TestReleaseMapEditorGroupNoteFitsTheStatusStrip pastes at 1280x800 with
// three refusals of the widest wording the paste can produce in the install's
// body font: the strip names the first, cut to the room left, counts the
// rest and keeps the Esc line. Its control: the whole line is wider than the
// strip.
func TestReleaseMapEditorGroupNoteFitsTheStatusStrip(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	s := &mapInspectionEdits{inspector: x}
	const wide, far = 99999, 9999
	allArms := mapload.GroundBorder | mapload.GroundMountain | mapload.GroundWater | mapload.GroundFlag | mapload.GroundObject
	widest := ""
	for _, kind := range []string{"Unit", "Structure", "Sack"} {
		name := s.recordName(ui.RecordRef{Kind: kind, Index: wide})
		for _, r := range []string{
			fmt.Sprintf(x.word("editor.selection.outside"), name, far, far),
			fmt.Sprintf(x.word("editor.selection.closed"), name, far, far, strings.Join(s.armWords(allArms), ", ")),
			fmt.Sprintf(x.word("editor.selection.noplayer"), name, uint32(1<<32-1)),
		} {
			if x.body.Advance(x.inspectionUTF8(r)) > x.body.Advance(x.inspectionUTF8(widest)) {
				widest = r
			}
		}
	}
	// The group: the units of the first 7x7 cells round a unit that hold two
	// or more units and no structure or sack, pasted back on their own cells.
	var box image.Rectangle
	n := 0
	for _, a := range e.Document().Records {
		if a.Kind != "Unit" || !a.Spatial {
			continue
		}
		b := image.Rect(a.Cell.X-3, a.Cell.Y-3, a.Cell.X+4, a.Cell.Y+4)
		units, others := 0, 0
		for _, r := range e.Document().Records {
			if r.Spatial && r.Cell.In(b) {
				switch r.Kind {
				case "Unit":
					units++
				case "Structure", "Sack":
					others++
				}
			}
		}
		if units >= 2 && others == 0 {
			box, n = b, units
			break
		}
	}
	if n == 0 {
		t.Fatal("no 7x7 cells hold two units and no structure or sack")
	}
	e.ShowCell(box.Min)
	selectArea(e, box.Min, box.Max.Sub(image.Pt(1, 1)))
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	var c image.Point
	first := true
	for _, r := range e.Document().Records {
		if r.Kind == "Unit" && r.Spatial && r.Cell.In(box) {
			if first {
				c, first = r.Cell, false
			}
			c = image.Pt(min(c.X, r.Cell.X), min(c.Y, r.Cell.Y))
		}
	}
	host, ok := e.Document().Edits.(*mapInspectionEdits)
	if !ok {
		t.Fatalf("the document's edits are %T", e.Document().Edits)
	}
	e.Document().Edits = widestRefusals{host, []string{widest, widest, widest}}
	pasteAt(t, e, c)
	if st := e.StatusLine(); !strings.Contains(st, x.inspectionUTF8(fmt.Sprintf(x.word("editor.selection.pasted"), n, n))) {
		t.Fatalf("the paste of %d units on their own cells: %q", n, st)
	}
	line, width, whole, limit := e.StatusFit()
	t.Logf("widest refusal %q; fitted %q, %d of %d (whole %d)", widest, line, width, limit, whole)
	if width == 0 || width > limit {
		t.Fatalf("the status line %q is %d wide; the strip holds %d", line, width, limit)
	}
	if whole <= limit {
		t.Fatalf("control: the whole line is %d wide within %d; the case does not test the fit", whole, limit)
	}
	more := x.inspectionUTF8(fmt.Sprintf(x.word("editor.selection.more"), 2))
	head := x.inspectionUTF8(widest)
	if cut := strings.Index(line, textfit.Mark); cut >= 0 {
		head = head[:min(len(head), 8)]
	}
	esc := x.inspectionUTF8(x.word("editor.keys.selectionclear"))
	if !strings.Contains(line, head) || !strings.Contains(line, more) || !strings.HasSuffix(line, esc) {
		t.Fatalf("the fitted line %q does not name the first refusal (%q), count the rest (%q) and keep %q", line, head, more, esc)
	}
}
