package game

import (
	"bytes"
	"fmt"
	"image"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// modWindowRecord is the cell of the structure each mission's map holds.
var modWindowRecord = image.Pt(20, 20)

// russianModWindow is the Russian editor with a mod of three missions that
// each replace a campaign mission, its mod window open on the missions page.
func russianModWindow(t *testing.T) (*testPlayFixture, *ui.MapEditor) {
	t.Helper()
	f := newTestPlayFixture(t)
	f.x.tiles = &terrain.Tileset{}
	f.x.structures = &terrain.StructureSet{}
	f.x.structures.Classes[5] = &terrain.StructureClass{}
	f.x.font = &text.Font{Selector: text.SelectorConverting}
	f.x.archives.Base.Profile.Language = "russian"
	e := f.x.Editor()
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput("40x56")
	for range 2 {
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if e.Document() == nil {
		t.Fatal("no map")
	}
	host := e.Document().Edits.(*mapInspectionEdits)
	if _, err := host.AddStructure(modWindowRecord, 5); err != nil {
		t.Fatal(err)
	}
	source := bytes.Clone(host.model.Bytes())
	if err := e.Key("undo"); err != nil || e.Dirty() {
		t.Fatalf("the map stays changed: %v", err)
	}
	missions := ""
	for _, n := range []int{10, 20, 30} {
		missions += fmt.Sprintf("[[mission]]\nkind = 'replace'\nkey = 'm%d'\nnumber = %d\nmap = '%d.alm'\n", n, n, n)
	}
	dir := writeNamedEditorMod(t, t.TempDir(), "unsealing__mystery", missions)
	for _, n := range []int{10, 20, 30} {
		writeTestPlayInput(t, filepath.Join(dir, fmt.Sprintf("%d.alm", n)), source)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if !e.ModWindowOpen() {
		t.Fatal("the mod window is not open")
	}
	e.Layout(1024, 768)
	return f, e
}

// The mod window's mission rows read in Russian as its Mission drop-down
// does: the choice's text, which is already in the font's bytes, reaches the
// row unchanged. No page of the window shows a run of question marks.
func TestRussianModWindowMissionRowsKeepTheirWords(t *testing.T) {
	f, e := russianModWindow(t)
	replace := f.x.word("editor.mod.replace")
	if replace != "Замена миссии кампании" {
		t.Fatalf("the Russian word book reads %q", replace)
	}
	lines := e.PopupLines()
	for _, n := range []int{10, 20, 30} {
		want := f.x.inspectionUTF8(fmt.Sprintf("m%d / %s: %d", n, replace, n))
		if !slices.Contains(lines, want) {
			t.Errorf("no row reads %q:\n%q", f.x.messageUTF8([]byte(want)), lines)
		}
	}
	for page := 0; page < 3; page++ {
		for _, line := range e.PopupLines() {
			if strings.Contains(line, "??") {
				t.Errorf("page %d: a line lost its letters: %q", page, line)
			}
		}
		_ = e.Key("package")
	}
}

// mapHearing is which input the map answered.
type mapHearing struct{ Wheel, Pan, Drag, Select bool }

// mapProbe sends input at the record of modWindowRecord, which it first
// brings to a point of the map that no window covers.
type mapProbe struct {
	t   *testing.T
	e   *ui.MapEditor
	now time.Time
}

// aim scrolls the map so the record stands beside the open window, and
// answers the record and its point.
func (m *mapProbe) aim() (int, image.Point) {
	m.t.Helper()
	e := m.e
	want := -1
	for i, r := range e.Document().Records {
		if r.Spatial && r.Cell == modWindowRecord {
			want = i
		}
	}
	if want < 0 {
		m.t.Fatalf("no record at %v", modWindowRecord)
	}
	e.ShowCell(modWindowRecord)
	area := e.MapRect()
	pop, shown := e.PopupRect()
	if shown {
		to := (area.Min.X + pop.Min.X) / 2
		if area.Max.X-pop.Max.X > pop.Min.X-area.Min.X {
			to = (pop.Max.X + area.Max.X) / 2
		}
		cam := e.Document().Viewer.Camera()
		cam.X += float64(e.CellPoint(modWindowRecord).X - to)
	}
	p := e.CellPoint(modWindowRecord)
	if shown && p.In(pop) || !p.In(area) {
		m.t.Fatalf("the record is at %v: under the window %v or off the map %v", p, pop, area)
	}
	return want, p
}

func (m *mapProbe) tick(in ui.Input, at image.Point) {
	in.CursorX, in.CursorY = in.CursorX+at.X, in.CursorY+at.Y
	m.now = m.now.Add(50 * time.Millisecond)
	m.e.Pointer(in, m.now)
}

func (m *mapProbe) click(at image.Point) {
	m.tick(ui.Input{PrimaryDown: true}, at)
	m.tick(ui.Input{}, at)
}

// hears sends the map a wheel notch, a held arrow, a middle-button drag and
// a click on the record, and answers which of them the map took.
func (m *mapProbe) hears() mapHearing {
	m.t.Helper()
	var h mapHearing
	_, p := m.aim()
	cam := m.e.Document().Viewer.Camera()
	m.tick(ui.Input{}, p)
	zoom := cam.Zoom
	m.tick(ui.Input{WheelY: -1}, p)
	h.Wheel = cam.Zoom != zoom
	m.tick(ui.Input{WheelY: 1}, p)
	h.Wheel = h.Wheel || cam.Zoom != zoom

	_, p = m.aim()
	x, y := cam.X, cam.Y
	for range 5 {
		m.tick(ui.Input{PanRight: true, PanDown: true}, p)
	}
	m.tick(ui.Input{}, p)
	h.Pan = cam.X != x || cam.Y != y

	_, p = m.aim()
	x, y = cam.X, cam.Y
	m.tick(ui.Input{MiddleDown: true}, p)
	m.tick(ui.Input{MiddleDown: true, CursorX: -20, CursorY: -20}, p)
	m.tick(ui.Input{CursorX: -20, CursorY: -20}, p)
	h.Drag = cam.X != x || cam.Y != y

	want, p := m.aim()
	if m.e.Selected() == want {
		m.t.Fatal("the record is selected before the click")
	}
	m.click(p)
	h.Select = m.e.Selected() == want
	return h
}

// The mod window floats over the map: outside its rectangle the wheel, the
// arrows, a drag and a click reach the map; on it they stay with the window.
// A selection made on the map leaves the window as it was, and a value typed
// into a field and not yet entered keeps the keyboard. The window closes by
// its cross and by Esc, and a mission row's Open loads that mission.
func TestRussianModWindowLeavesTheMapItsInput(t *testing.T) {
	all := mapHearing{true, true, true, true}
	t.Run("open", func(t *testing.T) {
		_, e := russianModWindow(t)
		if field, _, _ := e.FieldFocus(); field != -1 {
			t.Fatalf("the opening window focused field %d", field)
		}
		m := &mapProbe{t: t, e: e, now: time.Unix(100, 0)}
		before := e.PopupLines()
		if got := m.hears(); got != all {
			t.Fatalf("with the mod window open the map hears %+v", got)
		}
		if !e.ModWindowOpen() || !slices.Equal(before, e.PopupLines()) {
			t.Fatalf("the map's input changed the window:\n%q\n%q", before, e.PopupLines())
		}
	})
	t.Run("on the window", func(t *testing.T) {
		_, e := russianModWindow(t)
		m := &mapProbe{t: t, e: e, now: time.Unix(100, 0)}
		want, _ := m.aim()
		pop, _ := e.PopupRect()
		cam := e.Document().Viewer.Camera()
		cam.X += float64(e.CellPoint(modWindowRecord).X - (pop.Min.X + pop.Dx()/2))
		cam.Y += float64(e.CellPoint(modWindowRecord).Y - (pop.Max.Y - 6))
		p := e.CellPoint(modWindowRecord)
		if !p.In(pop) {
			t.Fatalf("the record at %v is not under the window %v", p, pop)
		}
		zoom, x, y := cam.Zoom, cam.X, cam.Y
		m.tick(ui.Input{WheelY: -1}, p)
		m.tick(ui.Input{WheelY: 1}, p)
		m.click(p)
		if cam.Zoom != zoom || cam.X != x || cam.Y != y || e.Selected() == want || !e.ModWindowOpen() {
			t.Fatalf("input on the window reached the map: zoom %v, selected %d", cam.Zoom, e.Selected())
		}
	})
	t.Run("typed value", func(t *testing.T) {
		_, e := russianModWindow(t)
		m := &mapProbe{t: t, e: e, now: time.Unix(100, 0)}
		if !e.FocusModField("title") {
			t.Fatal("no title field")
		}
		want, p := m.aim()
		m.click(p)
		if field, _, _ := e.FieldFocus(); field >= 0 || e.Selected() != want {
			t.Fatalf("a press on the map left the keyboard on field %d, selected %d", field, e.Selected())
		}
		if !e.FocusModField("title") {
			t.Fatal("no title field")
		}
		e.TextInput("Z")
		m.click(image.Pt(p.X, p.Y+60))
		field, _, _ := e.FieldFocus()
		if field < 0 || e.Selected() == want {
			t.Fatalf("a press on the map took the keyboard from a typed value: field %d, selected %d", field, e.Selected())
		}
		if !slices.ContainsFunc(e.PopupLines(), func(line string) bool { return strings.HasSuffix(line, "Z") }) {
			t.Fatalf("the typed value is gone:\n%q", e.PopupLines())
		}
	})
	t.Run("escape", func(t *testing.T) {
		_, e := russianModWindow(t)
		if err := e.Key("escape"); err != nil || e.ModWindowOpen() {
			t.Fatalf("Esc left the window open: %v", err)
		}
		if got := (&mapProbe{t: t, e: e, now: time.Unix(100, 0)}).hears(); got != all {
			t.Fatalf("after Esc the map hears %+v", got)
		}
	})
	t.Run("cross", func(t *testing.T) {
		_, e := russianModWindow(t)
		m := &mapProbe{t: t, e: e, now: time.Unix(100, 0)}
		r, ok := e.PopupRect()
		if !ok {
			t.Fatal("no window")
		}
		m.click(image.Pt(r.Max.X-8, r.Min.Y+8))
		if e.ModWindowOpen() {
			t.Fatal("the cross left the window open")
		}
		if got := m.hears(); got != all {
			t.Fatalf("after the cross the map hears %+v", got)
		}
	})
	t.Run("open row", func(t *testing.T) {
		f, e := russianModWindow(t)
		m := &mapProbe{t: t, e: e, now: time.Unix(100, 0)}
		selected := func() int { return e.Document().Edits.(*mapInspectionEdits).workspace.Project.Selected }
		if selected() != 0 {
			t.Fatalf("mission %d is open", selected())
		}
		row := -1
		for n := range e.PopupLines() {
			if b := e.PopupRowButtons(n); len(b) == 1 && b[0] == f.x.inspectionUTF8(f.x.word("editor.mod.open")) {
				row = n + 1 // the second of the three rows: mission 20
				break
			}
		}
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: row})
		if !ok {
			t.Fatal("no mission row with an Open button")
		}
		for x := r.Max.X - 3; x > r.Max.X-60 && selected() == 0; x -= 4 {
			m.click(image.Pt(x, r.Min.Y+r.Dy()/2))
		}
		if selected() != 1 || !e.ModWindowOpen() {
			t.Fatalf("Open left mission %d open, window %v", selected(), e.ModWindowOpen())
		}
		e.Layout(1024, 768)
		if got := m.hears(); got != all {
			t.Fatalf("after Open the map hears %+v", got)
		}
	})
}
