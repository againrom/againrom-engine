package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// groupMap is a 40x40 map, clear of the border ring between cells 8 and 31,
// with a roster of players: unit 0 at 12,12 owned by player 1, unit 1 at
// 14,12 owned by the last player, a kind-5 structure (3x2) at 20,20 owned by
// player 1 and a ground sack of item 0x0101 at 13,15. overlay names cells
// whose scenery closes them.
func groupMap(t *testing.T, players int, overlay ...image.Point) []byte {
	t.Helper()
	cells := make([]byte, 40*40)
	for _, c := range overlay {
		cells[c.Y*40+c.X] = 1
	}
	raw := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type7Payload: make([]byte, 12), Overlay: cells,
		Units: []synth.ALMUnit{
			{X: 12*256 + 128, Y: 12*256 + 128, ClassID: 40, ClassSubID: 2},
			{X: 14*256 + 64, Y: 12*256 + 200, ClassID: 40, ClassSubID: 2},
		},
		Objects: []synth.ALMObject{{X: 20*256 + 128, Y: 20*256 + 128, Kind: 5, Field12: 7}},
	})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	for m.Count(mapedit.KindPlayer) < players {
		rec, err := m.NewPlayerRecord()
		if err == nil {
			_, err = m.Place(mapedit.KindPlayer, rec)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []struct {
		k     mapedit.Kind
		i     int
		owner int64
	}{{mapedit.KindUnit, 0, 1}, {mapedit.KindUnit, 1, int64(players)}, {mapedit.KindStructure, 0, 1}} {
		if err := m.SetField(f.k, f.i, "owner", f.owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.Place(mapedit.KindLoot, m.GroundSack(13*256+128, 15*256+128, 0x0101)); err != nil {
		t.Fatal(err)
	}
	return m.Bytes()
}

// groupEditor opens raw in an editor over the installed set openRecords
// uses.
func groupEditor(t *testing.T, raw []byte) *ui.MapEditor {
	t.Helper()
	return groupEditorWith(t, raw, authoringTable())
}

// groupEditorWith is groupEditor over table.
func groupEditorWith(t *testing.T, raw []byte, table *mapload.Table) *ui.MapEditor {
	t.Helper()
	building := func(w, h int32) dbEntry { return dbEntry{name: "b", params: []int32{w, h, 0, 0, -1, -1, 0, 0}} }
	table.Buildings = dbCollection{{}, building(1, 1), building(1, 1), building(1, 1), building(1, 1), building(3, 2)}
	table.Names = data.ItemNames{0x0101: "first"}
	set := &terrain.StructureSet{}
	set.Classes[5] = &terrain.StructureClass{ID: 5, Name: "s", TileWidth: 3, TileHeight: 2, FullHeight: 2}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table, structures: set}
	e := x.Editor()
	openSelectionMap(t, e, "first.alm", raw)
	return e
}

// openSelectionMap opens raw in e as the file name.
func openSelectionMap(t *testing.T, e *ui.MapEditor, name string, raw []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatalf("%s did not open: the map before it has unsaved edits", name)
	}
	e.Layout(1800, 900)
	e.Document().Viewer.SetFlat(true)
	closeMapPanel(t, e)
}

// selectArea is a Shift drag from cell a to cell b.
func selectArea(e *ui.MapEditor, a, b image.Point) {
	p, q := e.CellPoint(a), e.CellPoint(b)
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true, Shift: true}, now)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true, Shift: true}, now)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, Shift: true}, now)
}

// pasteAt scrolls cell into view, points at it, presses Ctrl+V and clicks
// there.
func pasteAt(t *testing.T, e *ui.MapEditor, cell image.Point) {
	t.Helper()
	p := pointAt(e, cell)
	if err := e.Key("paste"); err != nil {
		t.Fatal(err)
	}
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

// pointAt scrolls cell into view and points at it. The first pointer event
// finishes the scroll, so the point is taken again after it.
func pointAt(e *ui.MapEditor, cell image.Point) image.Point {
	e.ShowCell(cell)
	p := e.CellPoint(cell)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	p = e.CellPoint(cell)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	return p
}

// groupRecords is every unit, structure and sack of the document, in its
// order, with its cell.
func groupRecords(e *ui.MapEditor) map[ui.RecordRef]image.Point {
	out := map[ui.RecordRef]image.Point{}
	for _, r := range e.Document().Records {
		if (r.Kind == "Unit" || r.Kind == "Structure" || r.Kind == "Sack") && r.Spatial {
			out[ui.RecordRef{Kind: r.Kind, Index: r.Index}] = r.Cell
		}
	}
	return out
}

// clonedAt is the map raw with each record of src cloned, in order, so that
// the top-left of their cells lands on cell: what one Clone per record writes.
func clonedAt(t *testing.T, raw, src []byte, refs []ui.RecordRef, cell image.Point) []byte {
	t.Helper()
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	s, err := mapedit.New(src)
	if err != nil {
		t.Fatal(err)
	}
	origin := image.Pt(1<<20, 1<<20)
	for _, r := range refs {
		x, y, _ := s.Position(editKinds[r.Kind], r.Index)
		origin = image.Pt(min(origin.X, int(x>>8)), min(origin.Y, int(y>>8)))
	}
	for _, r := range refs {
		k := editKinds[r.Kind]
		x, y, _ := s.Position(k, r.Index)
		nx, ny := cellWords(image.Pt(int(x>>8), int(y>>8)).Add(cell.Sub(origin)), x, y)
		if bytes.Equal(raw, src) {
			_, err = m.CloneRecord(k, r.Index, nx, ny)
		} else {
			_, err = m.CloneFrom(s, k, r.Index, nx, ny)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return m.Bytes()
}

var groupRefs = []ui.RecordRef{{Kind: "Unit", Index: 0}, {Kind: "Unit", Index: 1}, {Kind: "Structure", Index: 0}, {Kind: "Sack", Index: 0}}

func TestMapEditorRectangleSelectsUnitsStructuresAndSacks(t *testing.T) {
	e := groupEditor(t, groupMap(t, 2))
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	if !strings.Contains(e.StatusLine(), "4 selected") {
		t.Fatalf("status %q after a rectangle over four records", e.StatusLine())
	}
	// Control: a rectangle that leaves out the structure holds three.
	selectArea(e, image.Pt(11, 11), image.Pt(19, 19))
	if !strings.Contains(e.StatusLine(), "3 selected") {
		t.Fatalf("status %q after a rectangle over three records", e.StatusLine())
	}
}

func TestMapEditorGroupMoveKeepsTheOffsetsAndUndoesInOneStep(t *testing.T) {
	raw := groupMap(t, 2)
	e := groupEditor(t, raw)
	before := groupRecords(e)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	dragCell(e, image.Pt(12, 12), image.Pt(15, 14))
	after := groupRecords(e)
	for ref, cell := range before {
		if got := after[ref]; got != cell.Add(image.Pt(3, 2)) {
			t.Errorf("%v moved from %v to %v, want %v (%s)", ref, cell, got, cell.Add(image.Pt(3, 2)), e.StatusLine())
		}
	}
	m, _ := editedMap(t, e)
	if m.Units[1].X&255 != 64 || m.Units[1].Y&255 != 200 {
		t.Errorf("unit 1 lost its fraction: %d,%d", m.Units[1].X&255, m.Units[1].Y&255)
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("one undo did not restore the map before the group move")
	}
	// Control: without the group the same drag moves unit 0 alone.
	e.Select(recordIndex(t, e, "Unit", 0), true)
	dragCell(e, image.Pt(12, 12), image.Pt(15, 14))
	if got := groupRecords(e); got[groupRefs[0]] != image.Pt(15, 14) || got[groupRefs[1]] != before[groupRefs[1]] {
		t.Errorf("a single drag left %v", got)
	}
}

func TestMapEditorGroupDeleteUndoesToTheSameBytes(t *testing.T) {
	raw := groupMap(t, 2)
	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if got := groupRecords(e); len(got) != 0 {
		t.Fatalf("delete left %v (%s)", got, e.StatusLine())
	}
	if !strings.Contains(e.StatusLine(), "Deleted 4 of 4") {
		t.Errorf("status %q", e.StatusLine())
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("one undo did not give the map's bytes back")
	}
}

func TestMapEditorPasteEqualsOneClonePerRecord(t *testing.T) {
	raw := groupMap(t, 2)
	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.StatusLine(), "Copied 4") {
		t.Errorf("status %q after the copy", e.StatusLine())
	}
	// The same map: the clip's top-left cell, 12,12, lands on 9,22.
	pasteAt(t, e, image.Pt(9, 22))
	want := clonedAt(t, raw, raw, groupRefs, image.Pt(9, 22))
	if !bytes.Equal(e.Document().SourceBytes, want) {
		t.Fatalf("the paste differs from one clone per record (%s)", e.StatusLine())
	}
	if !strings.Contains(e.StatusLine(), "Pasted 4 of 4") || !strings.Contains(e.StatusLine(), "4 selected") {
		t.Errorf("status %q after the paste", e.StatusLine())
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("one undo did not revert the whole paste")
	}
	if bytes.Equal(want, raw) {
		t.Fatal("control: the clones wrote nothing")
	}

	// Another map, opened in the same window, takes the same clip.
	second := groupMap(t, 2)
	m, _ := mapedit.New(second)
	for m.Count(mapedit.KindUnit) > 0 {
		_ = m.Delete(mapedit.KindUnit, 0)
	}
	second = m.Bytes()
	openSelectionMap(t, e, "second.alm", second)
	pasteAt(t, e, image.Pt(10, 10))
	want = clonedAt(t, second, raw, groupRefs, image.Pt(10, 10))
	if !bytes.Equal(e.Document().SourceBytes, want) {
		t.Fatalf("the paste into another map differs from one clone per record (%s)", e.StatusLine())
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, second) {
		t.Fatal("one undo did not revert the paste into another map")
	}
}

func TestMapEditorPasteRefusesARecordByName(t *testing.T) {
	raw := groupMap(t, 2)
	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	// Pasted at 30,30 unit 0 stands on an open cell, unit 1 at 32,30 and the
	// sack at 31,33 in the border ring, and the structure at 38,38 with a 3x2
	// footprint that leaves the map.
	pasteAt(t, e, image.Pt(30, 30))
	s := e.StatusLine()
	for _, want := range []string{"Pasted 1 of 4", "unit 1: cell 32,30 is closed (border)", "structure 0: structure footprint 38,38 3x2 leaves the map", "1 more"} {
		if !strings.Contains(s, want) {
			t.Errorf("status %q lacks %q", s, want)
		}
	}
	_ = e.Key("undo")

	// A closed cell: scenery on the cell unit 1 would land on.
	closed := groupMap(t, 2, image.Pt(16, 10))
	e = groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	openSelectionMap(t, e, "closed.alm", closed)
	pasteAt(t, e, image.Pt(14, 10))
	if s := e.StatusLine(); !strings.Contains(s, "unit 1: cell 16,10 is closed") || !strings.Contains(s, "Pasted 3 of 4") {
		t.Fatalf("status %q does not name the unit on the closed cell", s)
	}
	_ = e.Key("undo")

	// A roster without player 2 refuses unit 1, whose owner it is.
	single := groupMap(t, 1)
	openSelectionMap(t, e, "single.alm", single)
	pasteAt(t, e, image.Pt(10, 10))
	if s := e.StatusLine(); !strings.Contains(s, "unit 1: this map has no player 2") {
		t.Fatalf("status %q does not name the unit whose player is missing", s)
	}
	// Control: a paste with room for every record refuses none.
	_ = e.Key("undo")
	openSelectionMap(t, e, "room.alm", groupMap(t, 2))
	pasteAt(t, e, image.Pt(10, 10))
	if s := e.StatusLine(); !strings.Contains(s, "Pasted 4 of 4") || strings.Contains(s, "closed") || strings.Contains(s, "no player") {
		t.Fatalf("status %q after a paste with room", s)
	}
}

// flierTable is authoringTable with class 41 a full units row whose movement
// column names the air domain.
func flierTable() *mapload.Table {
	table := authoringTable()
	p := make([]int32, 41)
	p[0x1d], p[0x1e], p[32] = 41, 2, 3
	units := slices.Clone(table.Units.(dbCollection))
	units[2] = dbEntry{name: "flier", params: p}
	table.Units = units
	return table
}

func TestMapEditorPasteTakesAFlierOnACellClosedToGround(t *testing.T) {
	raw := groupMap(t, 2)
	m, _ := mapedit.New(raw)
	if err := m.SetField(mapedit.KindUnit, 1, "class", 41); err != nil {
		t.Fatal(err)
	}
	raw = m.Bytes()
	if d, err := mapload.UnitDomain(mustMap(t, raw).Units[1], flierTable()); err != nil || d != sim.DomainAir {
		t.Fatalf("the fixture's unit 1 is domain %d (%v), not a flier", d, err)
	}
	// Scenery closes 16,10, where unit 1 lands when the clip's top-left
	// lands on 14,10: a flier stands there.
	closed := groupMap(t, 2, image.Pt(16, 10))
	e := groupEditorWith(t, raw, flierTable())
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	openSelectionMap(t, e, "scenery.alm", closed)
	pasteAt(t, e, image.Pt(14, 10))
	if s := e.StatusLine(); !strings.Contains(s, "Pasted 4 of 4") || strings.Contains(s, "closed") {
		t.Fatalf("status %q after a flier's paste onto a scenery cell", s)
	}
	if got, _ := editedMap(t, e); len(got.Units) != 4 || got.Units[3].X>>8 != 16 || got.Units[3].Y>>8 != 10 || got.Units[3].ClassID != 41 {
		t.Fatalf("units %+v", got.Units)
	}
	_ = e.Key("undo")
	// The border closes a cell to a flier too.
	pasteAt(t, e, image.Pt(30, 30))
	if s := e.StatusLine(); !strings.Contains(s, "unit 1: cell 32,30 is closed (border)") {
		t.Fatalf("status %q: the flier on a border cell was not refused", s)
	}
	_ = e.Key("undo")
	// Control: under the table where class 41 is a ground unit the same
	// paste refuses it on the scenery cell.
	e = groupEditorWith(t, raw, authoringTable())
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	openSelectionMap(t, e, "scenery.alm", closed)
	pasteAt(t, e, image.Pt(14, 10))
	if s := e.StatusLine(); !strings.Contains(s, "unit 1: cell 16,10 is closed (object)") || !strings.Contains(s, "Pasted 3 of 4") {
		t.Fatalf("status %q: the ground unit on the scenery cell was not refused", s)
	}
}

func mustMap(t *testing.T, raw []byte) *alm.Map {
	t.Helper()
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// recipeMap is groupMap with an enchanted item recipe on its ground sack.
func recipeMap(t *testing.T) []byte {
	t.Helper()
	m, _ := mapedit.New(groupMap(t, 2))
	if _, err := m.AddItemRecipe(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetField(mapedit.KindEnchantment, 0, "addeffect", 12); err != nil {
		t.Fatal(err)
	}
	return m.Bytes()
}

func TestMapEditorPasteInTheSameMapSharesASackRecipe(t *testing.T) {
	raw := recipeMap(t)
	path := filepath.Join(t.TempDir(), "first.alm")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	e := groupEditor(t, groupMap(t, 2))
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1800, 900)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	// Another map opened and the first opened again: the paste is still into
	// the map the clip came from.
	openSelectionMap(t, e, "other.alm", groupMap(t, 2))
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1800, 900)
	pasteAt(t, e, image.Pt(9, 22))
	want := clonedAt(t, raw, raw, groupRefs, image.Pt(9, 22))
	if got := e.Document().SourceBytes; !bytes.Equal(got, want) {
		t.Fatalf("the paste differs from one clone per record: %d recipes, want %d (%s)",
			len(mustMap(t, got).Enchantments), len(mustMap(t, want).Enchantments), e.StatusLine())
	}
	if s := e.StatusLine(); strings.Contains(s, "recipes copied") || !strings.Contains(s, "Pasted 4 of 4") {
		t.Fatalf("status %q", s)
	}
	_ = e.Key("undo")
	// Control: a paste of the same clip into another map copies the recipe.
	openSelectionMap(t, e, "room.alm", groupMap(t, 2))
	pasteAt(t, e, image.Pt(9, 22))
	if s := e.StatusLine(); !strings.Contains(s, "recipes copied: 1") {
		t.Fatalf("status %q: a paste into another map did not copy the recipe", s)
	}
}

func TestMapEditorPasteIntoAnotherMapTakesFreshGroups(t *testing.T) {
	const scripted = 5
	src, _ := mapedit.New(groupMap(t, 2))
	if _, err := src.CloneRecord(mapedit.KindUnit, 0, 13*256+128, 13*256+128); err != nil {
		t.Fatal(err)
	}
	for _, f := range []struct{ i, g int64 }{{0, scripted}, {1, 0}, {2, scripted}} {
		if err := src.SetField(mapedit.KindUnit, int(f.i), "group", f.g); err != nil {
			t.Fatal(err)
		}
	}
	raw := src.Bytes()
	target, _ := mapedit.New(groupMap(t, 2))
	for target.Count(mapedit.KindUnit) > 0 {
		_ = target.Delete(mapedit.KindUnit, 0)
	}
	if err := target.AddArm(mapedit.ArmWinGroup, scripted); err != nil {
		t.Fatal(err)
	}
	second := target.Bytes()

	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	openSelectionMap(t, e, "second.alm", second)
	pasteAt(t, e, image.Pt(10, 10))
	got, _ := editedMap(t, e)
	sc, err := got.Script()
	if err != nil {
		t.Fatal(err)
	}
	named := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for k, ty := range n.Type {
				if ty == 2 {
					named[n.Value[k]] = true
				}
			}
		}
	}
	if !named[scripted] || len(got.Units) != 3 {
		t.Fatalf("the target's script names %v; %d units", named, len(got.Units))
	}
	groups := []uint32{got.Units[0].GroupID, got.Units[1].GroupID, got.Units[2].GroupID}
	if groups[0] == scripted || named[groups[0]] || groups[0] == 0 || groups[2] != groups[0] || groups[1] != 0 {
		t.Fatalf("pasted groups %v; the target's script names %v", groups, named)
	}
	if s := e.StatusLine(); !strings.Contains(s, "groups renumbered: 1") {
		t.Fatalf("status %q", s)
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, second) {
		t.Fatal("one undo did not revert the paste")
	}
	// Control: a paste into the map the clip came from keeps the groups, as
	// Clone keeps them.
	e = groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	_ = e.Key("copy")
	pasteAt(t, e, image.Pt(9, 22))
	same, _ := editedMap(t, e)
	if n := len(same.Units); n != 6 || same.Units[3].GroupID != scripted || same.Units[5].GroupID != scripted || strings.Contains(e.StatusLine(), "renumbered") {
		t.Fatalf("a paste in the same map: %d units, status %q", n, e.StatusLine())
	}
}

// A selection box drawn while a list tab is open selects that tab's records
// only, but Ctrl+C takes every unit, structure and sack in the box, and the
// paste into another map places them all. Loss control: the copy of the
// box's selected records alone pastes the units without the structure and
// the sack.
func TestMapEditorBoxCopyOnAListTabTakesEveryKind(t *testing.T) {
	raw := groupMap(t, 2)
	m, _ := mapedit.New(raw)
	for m.Count(mapedit.KindUnit) > 0 {
		_ = m.Delete(mapedit.KindUnit, 0)
	}
	second := m.Bytes()
	for _, tab := range []string{"units", "groups", "structures", "sacks"} {
		t.Run(tab, func(t *testing.T) {
			e := groupEditor(t, raw)
			r, ok := e.BarRect(tab)
			if !ok {
				t.Fatalf("no %s tab", tab)
			}
			c := r.Min.Add(r.Size().Div(2))
			e.Pointer(ui.Input{CursorX: c.X, CursorY: c.Y, PrimaryDown: true}, time.Unix(1, 0))
			e.Pointer(ui.Input{CursorX: c.X, CursorY: c.Y}, time.Unix(1, 0))
			dragCell(e, image.Pt(11, 11), image.Pt(23, 23))
			if n := len(e.SelectedRecords()); n == 0 || n == len(groupRefs) {
				t.Fatalf("the box on the %s tab selected %v", tab, e.SelectedRecords())
			}
			if err := e.Key("copy"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(e.StatusLine(), "4 records") {
				t.Errorf("status %q after the copy", e.StatusLine())
			}
			openSelectionMap(t, e, "second.alm", second)
			pasteAt(t, e, image.Pt(18, 18))
			got, sacks := editedMap(t, e)
			if len(got.Units) != 2 || len(got.Objects) != 2 || len(sacks) != 2 {
				t.Fatalf("the paste left %d units, %d structures, %d sacks (%s)", len(got.Units), len(got.Objects), len(sacks), e.StatusLine())
			}
			_ = e.Key("undo")
			if !bytes.Equal(e.Document().SourceBytes, second) {
				t.Fatal("one undo did not revert the paste")
			}
		})
	}
}

// Ctrl+V arms the paste brush and changes no byte; the click pastes as one
// undo step and disarms; a second Ctrl+V arms the same clip again. The
// brush's preview is PasteRecords on a copy of the model: it leaves the map
// as it was, so the click's paste matches one clone per record. Loss
// controls: a preview that pastes into the model itself leaves the map
// edited before the click; a Ctrl+V that pastes at once edits at the key.
func TestMapEditorPasteBrushPastesOnClickAsOneUndoStep(t *testing.T) {
	raw := groupMap(t, 2)
	e := groupEditor(t, raw)
	selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	host := e.Document().Edits
	clip, _, err := host.(ui.InspectionSelection).CopyRecords(groupRefs)
	if err != nil {
		t.Fatal(err)
	}
	cell := image.Pt(9, 22)
	want := clonedAt(t, raw, raw, groupRefs, cell)
	preview, err := host.(ui.InspectionPlacementPreview).PreviewPlacement(ui.InspectionNewRecord{Cell: cell, HasCell: true, Clip: clip})
	if err != nil || !bytes.Equal(preview.SourceBytes, want) {
		t.Fatalf("the preview is not the paste: %v", err)
	}
	p := pointAt(e, cell)
	if err := e.Key("paste"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, raw) || host.CanUndo() || !strings.Contains(e.StatusLine(), "Paste: click") {
		t.Fatalf("Ctrl+V edited or did not arm: status %q", e.StatusLine())
	}
	click := func() {
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	}
	click()
	if !bytes.Equal(e.Document().SourceBytes, want) || !strings.Contains(e.StatusLine(), "Pasted 4 of 4") {
		t.Fatalf("the click's paste differs from one clone per record (%s)", e.StatusLine())
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("one undo did not revert the paste")
	}
	click()
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("a click after the paste pasted again")
	}
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	if err := e.Key("paste"); err != nil {
		t.Fatal(err)
	}
	click()
	if !bytes.Equal(e.Document().SourceBytes, want) {
		t.Fatalf("the second Ctrl+V did not paste the same clip (%s)", e.StatusLine())
	}
}
