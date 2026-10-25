package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// editorGroundCheck fails unless the document's passability view closes
// exactly the cells the game's ground-block plane of the document's own bytes
// closes, under the install's table.
func editorGroundCheck(t *testing.T, x *MapInspector, d *ui.InspectionDocument, what string) *alm.Map {
	t.Helper()
	m, err := alm.Open(d.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	plane := mapload.PassabilityWith(m, x.table)
	if len(d.Passability) != len(plane) || len(plane) != d.Width*d.Height {
		t.Fatalf("%s: %d passability cells, %d plane cells, %dx%d map", what, len(d.Passability), len(plane), d.Width, d.Height)
	}
	for i := range plane {
		if (d.Passability[i] != 0) != (plane[i]&1 != 0) {
			t.Fatalf("%s: cell %d,%d shows arms %#x, the plane holds %#x", what, i%d.Width, i/d.Width, d.Passability[i], plane[i])
		}
	}
	return m
}

// worldGround is the ground-block bit of every cell of the world the game
// builds from m.
func worldGround(t *testing.T, x *MapInspector, m *alm.Map) []bool {
	t.Helper()
	w, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	planes := w.CellPlanes(sim.CellPlaneStart{})
	out := make([]bool, m.Width*m.Height)
	for i := range out {
		out[i] = planes.Static[(i/m.Width)<<8|i%m.Width]&1 != 0
	}
	return out
}

// The passability view of every shipped map closes exactly the cells the
// game's ground-block plane closes; on two maps that is checked against the
// built world itself. The control is the plane without the install's table:
// it misses the structure footprints, and on a map with structures it
// disagrees with the world, so the comparison sees the structure stage.
func TestReleaseMapEditorPassabilityIsTheWorldsGroundPlane(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	worlds := []string{"scenario/10.alm"}
	structural := 0
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m := editorGroundCheck(t, x, d, key)
		if !bytes.Equal(mapload.Passability(m), mapload.PassabilityWith(m, x.table)) {
			structural++
			if len(worlds) < 2 && key != worlds[0] {
				worlds = append(worlds, key)
			}
		}
	}
	if structural == 0 || len(worlds) < 2 {
		t.Fatalf("%d maps have a structure that moves the plane; the control needs one", structural)
	}
	for _, key := range worlds {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, _ := alm.Open(d.SourceBytes)
		ground := worldGround(t, x, m)
		for i, closed := range ground {
			if closed != (d.Passability[i] != 0) {
				t.Fatalf("%s: cell %d,%d is %v in the world, arms %#x in the view", key, i%m.Width, i/m.Width, closed, d.Passability[i])
			}
		}
		bare := mapload.Passability(m)
		differs := false
		for i, closed := range ground {
			differs = differs || closed != (bare[i]&1 != 0)
		}
		if key == worlds[1] && !differs {
			t.Fatalf("%s: the plane without structures agrees with the world", key)
		}
		t.Logf("%s: %dx%d, world and view agree on every cell", key, m.Width, m.Height)
	}
	t.Logf("%d maps checked, %d with a structure that moves the plane", len(installedMapKeys(x)), structural)
}

// openCell is an interior cell of d whose 3x3 neighbourhood is open ground
// and carries no record, searched outward from start.
func openCell(t *testing.T, d *ui.InspectionDocument, start image.Point, taken map[image.Point]bool) image.Point {
	t.Helper()
	used := map[image.Point]bool{}
	for _, r := range d.Records {
		if r.Spatial {
			used[r.Cell] = true
		}
	}
	for r := 0; r < d.Width; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				c := start.Add(image.Pt(dx, dy))
				if c.X < 10 || c.Y < 10 || c.X >= d.Width-10 || c.Y >= d.Height-10 || taken[c] {
					continue
				}
				ok := true
				for yy := -1; yy <= 1 && ok; yy++ {
					for xx := -1; xx <= 1 && ok; xx++ {
						n := c.Add(image.Pt(xx, yy))
						ok = d.Passability[n.Y*d.Width+n.X] == 0 && !used[n] && !taken[n]
					}
				}
				if ok {
					return c
				}
			}
		}
	}
	open := 0
	for _, a := range d.Passability {
		if a == 0 {
			open++
		}
	}
	t.Fatalf("the map has no open interior cell; %d of %d cells are open", open, len(d.Passability))
	return image.Point{}
}

func armAt(d *ui.InspectionDocument, c image.Point) ui.PassArm {
	return d.Passability[c.Y*d.Width+c.X]
}

// hoverStatus moves the pointer onto cell and returns the status line.
func hoverStatus(e *ui.MapEditor, c image.Point) string {
	p := e.CellPoint(c)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(5, 0))
	return e.StatusLine()
}

// The view follows every edit the window makes: a terrain stroke, undo and
// redo, an object stroke, a Block and an Unblock stroke and a structure
// move. Block and Unblock change bit 13 of the stroked cell and no other
// byte of the map; Unblock on Water leaves the cell closed, and the status
// line names Water as what keeps it.
func TestReleaseMapEditorPassabilityFollowsEveryEdit(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	book := words.For(x.archives.Base.Profile.Language)
	word := func(id string) string { return x.inspectionUTF8(book.Text(id)) }
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	d0 := e.Document()
	editorGroundCheck(t, x, d0, "opened")
	centre := image.Pt(d0.Width/2, d0.Height/2)
	taken := map[image.Point]bool{}
	pick := func() image.Point {
		c := openCell(t, e.Document(), centre, taken)
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				taken[c.Add(image.Pt(dx, dy))] = true
			}
		}
		return c
	}
	water, object, flag := pick(), pick(), pick()

	e.ShowCell(water)
	brushWindow(t, e, "terrain", paletteRow(t, mapedit.ClassWater))
	pointerStroke(e, []image.Point{water})
	d1 := e.Document()
	editorGroundCheck(t, x, d1, "terrain stroke")
	if armAt(d1, water)&ui.PassWater == 0 {
		t.Fatalf("painted water reads arms %#x", armAt(d1, water))
	}
	_ = e.Key("undo")
	if !slicesEqual(e.Document().Passability, d0.Passability) {
		t.Fatal("undo did not restore the view")
	}
	_ = e.Key("redo")
	if !slicesEqual(e.Document().Passability, d1.Passability) {
		t.Fatal("redo did not restore the painted view")
	}

	e.ShowCell(object)
	brushWindow(t, e, "objects", 1)
	pointerStroke(e, []image.Point{object})
	d2 := e.Document()
	editorGroundCheck(t, x, d2, "object stroke")
	if armAt(d2, object)&ui.PassObject == 0 {
		t.Fatalf("a placed object reads arms %#x", armAt(d2, object))
	}

	e.ShowCell(flag)
	brushWindow(t, e, "passability", 0)
	before := append([]byte(nil), e.Document().SourceBytes...)
	pointerStroke(e, []image.Point{flag})
	blocked := e.Document()
	editorGroundCheck(t, x, blocked, "Block")
	if armAt(blocked, flag) != ui.PassFlag {
		t.Fatalf("a blocked cell reads arms %#x", armAt(blocked, flag))
	}
	if len(blocked.SourceBytes) != len(before) {
		t.Fatal("Block changed the map's length")
	}
	diffs := 0
	for i := range before {
		if d := before[i] ^ blocked.SourceBytes[i]; d != 0 {
			diffs++
			if d != 0x20 {
				t.Fatalf("Block changed byte %#x by %#x", i, d)
			}
		}
	}
	if diffs != 1 {
		t.Fatalf("Block at radius 0 changed %d bytes", diffs)
	}
	if s := hoverStatus(e, flag); !strings.Contains(s, word("editor.pass.flag")) {
		t.Fatalf("status %q does not name the flag", s)
	}
	_ = e.Key("down")
	pointerStroke(e, []image.Point{flag})
	if armAt(e.Document(), flag) != 0 || !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatal("Unblock did not reopen the cell to the bytes before Block")
	}
	e.ShowCell(water)
	pointerStroke(e, []image.Point{water})
	if armAt(e.Document(), water) != ui.PassWater {
		t.Fatalf("unblocked water reads arms %#x", armAt(e.Document(), water))
	}
	if s := hoverStatus(e, water); !strings.Contains(s, word("editor.pass.water")) {
		t.Fatalf("status %q does not name Water", s)
	}
	_ = e.Key("passability") // the brush off

	moved := false
	for i, r := range e.Document().Records {
		if r.Kind != "Structure" || armAt(e.Document(), r.Cell) != ui.PassStructure || r.Cell.X < 12 || r.Cell.Y < 12 {
			continue
		}
		for _, off := range []image.Point{{0, 3}, {3, 0}, {0, -3}, {-3, 0}, {4, 4}} {
			prev := e.Document()
			e.Select(i, true)
			from, to := e.CellPoint(r.Cell), e.CellPoint(r.Cell.Add(off))
			now := time.Unix(10, 0)
			e.Pointer(ui.Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, now)
			e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y, PrimaryDown: true}, now)
			e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y}, now)
			if bytes.Equal(prev.SourceBytes, e.Document().SourceBytes) {
				continue
			}
			editorGroundCheck(t, x, e.Document(), "structure move")
			if slicesEqual(prev.Passability, e.Document().Passability) {
				t.Fatal("a structure move left the view as it was")
			}
			_ = e.Key("undo")
			if !slicesEqual(prev.Passability, e.Document().Passability) {
				t.Fatal("undoing the move did not restore the view")
			}
			moved = true
			break
		}
		if moved {
			break
		}
	}
	if !moved {
		t.Fatal("no structure of the map could be moved")
	}
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestReleaseMapEditorPassabilityFrames writes the owner's frames of the
// editor's terrain tools into AGAINROM_PASSABILITY_FRAMES: the map with the
// grid and the passability view off and on, the Brushes menu, the terrain
// list with its swatches and the passability and height brushes' status
// lines. Without that directory it drives the same steps.
func TestReleaseMapEditorPassabilityFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_PASSABILITY_FRAMES")
	now := time.Unix(1, 0)
	click := func(e *ui.MapEditor, r image.Rectangle, ok bool, what string) {
		t.Helper()
		if !ok {
			t.Fatalf("%s is not shown", what)
		}
		p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
		now = now.Add(time.Second)
	}
	menu := func(e *ui.MapEditor, bar, key string) {
		t.Helper()
		r, ok := e.BarRect(bar)
		click(e, r, ok, "the "+bar+" entry")
		n, ok := e.MenuRow(bar, key)
		if !ok {
			t.Fatalf("the %s menu has no %s row", bar, key)
		}
		r, ok = e.MenuRowRect(n)
		click(e, r, ok, "the "+key+" row")
	}
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{})
	}
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	d := e.Document()
	var water image.Point
	for i, a := range d.Passability {
		c := image.Pt(i%d.Width, i/d.Width)
		if a&ui.PassWater != 0 && a&ui.PassBorder == 0 && c.X > 20 && c.Y > 20 {
			water = c
			break
		}
	}
	if water == (image.Point{}) {
		t.Fatal("the map has no interior water")
	}
	e.ShowCell(water)
	menu(e, "view", "grid")
	shot(e, "view-grid")
	menu(e, "view", "passview")
	if s := hoverStatus(e, water); !strings.Contains(s, x.inspectionUTF8(words.For(x.archives.Base.Profile.Language).Text("editor.pass.water"))) {
		t.Fatalf("status %q does not name Water", s)
	}
	shot(e, "view-grid-passability")
	r, ok := e.BarRect("brushes")
	click(e, r, ok, "the Brushes entry")
	shot(e, "brushes-menu")
	n, _ := e.MenuRow("brushes", "brush.terrain")
	r, ok = e.MenuRowRect(n)
	click(e, r, ok, "the Terrain row")
	shot(e, "terrain-swatches")
	menu(e, "brushes", "brush.passability")
	_ = e.Key("down")
	pointerStroke(e, []image.Point{water})
	hoverStatus(e, water)
	shot(e, "passability-brush-unblock-water")
	menu(e, "brushes", "brush.height")
	hoverStatus(e, water)
	shot(e, "height-brush-status")
}

// At 1280x800 the status strip holds its whole line with the keys help and
// the pointer on a cell, and each brush's own help line whole.
// With the passability view naming every terrain arm of the hovered cell, the
// keys help gives up its leading lines and keeps the F1 line, which leads to
// the rest. Its control: that whole line, measured in the same font, is
// wider than the strip, so the check fails if the fit stops dropping lines.
func TestReleaseMapEditorStatusFitsWithPassability(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	d := e.Document()
	c := image.Pt(d.Width/2, d.Height/2)
	d.Passability[c.Y*d.Width+c.X] = ui.PassBorder | ui.PassMountain | ui.PassWater | ui.PassFlag | ui.PassObject
	e.ShowCell(c)
	tail, newMap := x.inspectionUTF8(x.word("editor.keys.help")), x.inspectionUTF8(x.word("editor.keys.new"))
	check := func(what string, whole bool) string {
		t.Helper()
		hoverStatus(e, c)
		line, width, full, limit := e.StatusFit()
		if width == 0 || width > limit {
			t.Fatalf("%s: %q is %d wide; the strip holds %d", what, line, width, limit)
		}
		if whole && line != e.StatusLine() {
			t.Fatalf("%s: the strip dropped part of %q (%d wide)", what, e.StatusLine(), full)
		}
		return line
	}
	if line := check("keys, view off", true); !strings.HasSuffix(line, tail) {
		t.Fatalf("keys line %q", line)
	}
	// A brush leads its help with the size line every brush shares; the strip
	// may drop it, and keeps the brush's own line whole.
	own := map[string]string{"terrain": "editor.keys.terrain", "height": "editor.keys.height", "objects": "editor.keys.objects", "passability": "editor.keys.passbrush"}
	brushCheck := func(key, what string) {
		t.Helper()
		if line := check(key+" brush"+what, false); !strings.HasSuffix(line, x.inspectionUTF8(x.word(own[key]))) {
			t.Fatalf("%s brush%s: %q does not end with the brush's own line", key, what, line)
		}
	}
	for _, key := range []string{"terrain", "height", "objects", "passability"} {
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
		brushCheck(key, "")
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
	}
	r, ok := e.BarRect("view")
	trigClick(t, e, r, ok, "View")
	n, _ := e.MenuRow("view", "passview")
	r, ok = e.MenuRowRect(n)
	trigClick(t, e, r, ok, "Passability")
	line := check("keys, view on", false)
	_, _, full, limit := e.StatusFit()
	if full <= limit {
		t.Fatalf("control: the whole line with every arm is %d wide within %d; the case does not test the fit", full, limit)
	}
	if !strings.HasSuffix(line, tail) || !strings.Contains(e.StatusLine(), newMap) || strings.Contains(line, newMap) {
		t.Fatalf("the fitted line %q does not keep the F1 line in place of the leading ones", line)
	}
	for _, key := range []string{"terrain", "height", "objects", "passability"} {
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
		brushCheck(key, ", view on")
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
	}
}
