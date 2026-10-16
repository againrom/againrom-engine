package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// openRecords opens a 16x16 map with a unit standing on the anchor of a
// kind-5 structure at 3,4, a kind-0x21 structure at 8,9 and one stock record
// of item 0x0101 carried by unit id 7. The installed set holds structure
// kinds 5, 6 and 0x21 (kind 5 and 6 a 3x2 footprint) and names items 0x0101
// and 0x0202.
func openRecords(t *testing.T) ([]byte, *ui.MapEditor) {
	t.Helper()
	raw := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: make([]byte, 12),
		Units: []synth.ALMUnit{{X: 3*256 + 128, Y: 4*256 + 128, ClassID: 40, ClassSubID: 2}},
		Objects: []synth.ALMObject{
			{X: 3*256 + 128, Y: 4*256 + 128, Kind: 5, Field12: 7},
			{X: 8*256 + 128, Y: 9*256 + 128, Kind: 0x21, Ext: []byte{2, 0, 0, 0, 2, 0, 0, 0}},
		}})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	stock := m.GroundSack(0, 0, 0x0101)
	binary.LittleEndian.PutUint32(stock[4:], 7)
	if _, err := m.Place(mapedit.KindLoot, stock); err != nil {
		t.Fatal(err)
	}
	b := m.Bytes()
	table := authoringTable()
	building := func(w, h int32) dbEntry { return dbEntry{name: "b", params: []int32{w, h, 0, 0, -1, -1, 0, 0}} }
	table.Buildings = dbCollection{{}, building(1, 1), building(1, 1), building(1, 1), building(1, 1), building(3, 2), building(3, 2)}
	table.Names = data.ItemNames{0x0101: "first", 0x0202: "second"}
	set := &terrain.StructureSet{}
	for _, k := range []int{5, 6, 0x21} {
		set.Classes[k] = &terrain.StructureClass{ID: int32(k), Name: "s", TileWidth: 3, TileHeight: 2, FullHeight: 2}
	}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table, structures: set}
	path := filepath.Join(t.TempDir(), "source.alm")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	return b, e
}

func recordIndex(t *testing.T, e *ui.MapEditor, kind string, index int) int {
	t.Helper()
	for i, r := range e.Document().Records {
		if r.Kind == kind && r.Index == index {
			return i
		}
	}
	t.Fatalf("no %s record %d", kind, index)
	return -1
}

func dragCell(e *ui.MapEditor, from, to image.Point) {
	p, q := e.CellPoint(from), e.CellPoint(to)
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, now)
}

func editedMap(t *testing.T, e *ui.MapEditor) (*alm.Map, []alm.LootRecord) {
	t.Helper()
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Loot()
	if err != nil {
		t.Fatal(err)
	}
	return m, l.Records
}

func TestMapEditorDragPrefersTheUnitOverTheStructureUnderIt(t *testing.T) {
	_, e := openRecords(t)
	dragCell(e, image.Pt(3, 4), image.Pt(6, 6))
	m, _ := editedMap(t, e)
	if m.Units[0].X>>8 != 6 || m.Objects[0].X>>8 != 3 {
		t.Fatalf("drag moved unit to %d and structure to %d", m.Units[0].X>>8, m.Objects[0].X>>8)
	}
}

func TestMapEditorEditsStructuresAndSacksWithChecks(t *testing.T) {
	b, e := openRecords(t)
	history := [][]byte{b}
	step := func(what string, ok func(*alm.Map, []alm.LootRecord) bool) {
		t.Helper()
		if m, l := editedMap(t, e); !ok(m, l) {
			t.Fatalf("%s: objects %+v loot %+v (%s)", what, m.Objects, l, e.Message())
		}
		history = append(history, append([]byte(nil), e.Document().SourceBytes...))
	}
	refused := func(what string, err error) {
		t.Helper()
		if err == nil || !bytes.Equal(e.Document().SourceBytes, history[len(history)-1]) {
			t.Fatalf("%s was accepted or wrote", what)
		}
		_ = e.Key("escape")
		_ = e.Key("escape")
	}

	// Move: select the structure, then drag it from its anchor.
	e.Select(recordIndex(t, e, "Structure", 0), true)
	dragCell(e, image.Pt(3, 4), image.Pt(5, 7))
	step("move", func(m *alm.Map, _ []alm.LootRecord) bool {
		return m.Objects[0].X == 5*256+128 && m.Objects[0].Y == 7*256+128 && m.Units[0].X>>8 == 3
	})
	_, err := e.Document().Edits.Move("Structure", 0, image.Pt(14, 14))
	refused("a footprint past the map edge", err)

	// Clone: C, then a click; the clone is selected.
	e.Select(recordIndex(t, e, "Structure", 0), true)
	clickCell(e, image.Pt(10, 2))
	if m, _ := editedMap(t, e); len(m.Objects) != 2 {
		t.Fatal("a click without C cloned")
	}
	e.Select(recordIndex(t, e, "Structure", 0), true)
	_ = e.Key("clone")
	clickCell(e, image.Pt(10, 2))
	step("clone", func(m *alm.Map, _ []alm.LootRecord) bool {
		return len(m.Objects) == 3 && m.Objects[2].X == 10*256+128 && m.Objects[2].Field12 == 8
	})
	if r := e.Document().Records[e.Selected()]; r.Kind != "Structure" || r.Index != 2 {
		t.Fatalf("the clone is not selected: %+v", r)
	}

	// Field: kind 7 is not installed, kind 6 is.
	refused("kind 7", enterField(e, 0, "7"))
	if err := enterField(e, 0, "6"); err != nil {
		t.Fatal(err)
	}
	step("kind", func(m *alm.Map, _ []alm.LootRecord) bool { return m.Objects[2].Kind == 6 })

	// Delete the selected clone.
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	step("delete", func(m *alm.Map, _ []alm.LootRecord) bool { return len(m.Objects) == 2 })

	// Add a sack: S, a click, then the item code.
	clickCell(e, image.Pt(6, 6))
	if _, l := editedMap(t, e); len(l) != 1 {
		t.Fatal("a click without S added a sack")
	}
	_ = e.Key("sack")
	clickCell(e, image.Pt(6, 6))
	refused("an unnamed item", enterField(e, 0, "999"))
	_ = e.Key("sack")
	clickCell(e, image.Pt(6, 6))
	e.TextInput("257")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	step("add sack", func(_ *alm.Map, l []alm.LootRecord) bool {
		return len(l) == 2 && l[1].Ground() && l[1].X == 6*256+128 && l[1].Y == 6*256+128 && len(l[1].Elements) == 1 && l[1].Elements[0].ItemCode() == 0x0101
	})
	if r := e.Document().Records[e.Selected()]; r.Kind != "Sack" || r.Index != 1 {
		t.Fatalf("the new sack is not selected: %+v", r)
	}

	// Sack fields: gold, item 1, add item.
	refused("gold -1", enterField(e, 0, "-1"))
	if err := enterField(e, 0, "40"); err != nil {
		t.Fatal(err)
	}
	step("gold", func(_ *alm.Map, l []alm.LootRecord) bool { return l[1].Gold == 40 })
	if err := enterField(e, 2, "514"); err != nil {
		t.Fatal(err)
	}
	step("add item", func(_ *alm.Map, l []alm.LootRecord) bool {
		return len(l[1].Elements) == 2 && l[1].Elements[1].ItemCode() == 0x0202
	})

	// The stock record: items only, no cell.
	e.Select(recordIndex(t, e, "Stock", 0), true)
	_ = e.Key("clone")
	clickCell(e, image.Pt(2, 2))
	if _, l := editedMap(t, e); len(l) != 2 || e.Message() == "" {
		t.Fatal("a stock clone was accepted")
	}
	_ = e.Key("escape")
	if err := enterField(e, 0, "514"); err != nil {
		t.Fatal(err)
	}
	step("stock item", func(_ *alm.Map, l []alm.LootRecord) bool {
		return l[0].Owner == 7 && l[0].Elements[0].ItemCode() == 0x0202
	})

	// Delete the new sack.
	e.Select(recordIndex(t, e, "Sack", 1), true)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	step("delete sack", func(_ *alm.Map, l []alm.LootRecord) bool { return len(l) == 1 })

	for i := len(history) - 2; i >= 0; i-- {
		_ = e.Key("undo")
		if !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("undo to step %d not byte exact", i)
		}
	}
	for i := 1; i < len(history); i++ {
		_ = e.Key("redo")
		if !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("redo to step %d not byte exact", i)
		}
	}
}
