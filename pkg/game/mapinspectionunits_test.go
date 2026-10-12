package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// authoringTable holds Units rows (typeID, face) 40/2 and 41/2 and one Humans
// row with server id 1234; slots are the installed tables' column positions.
func authoringTable() *mapload.Table {
	row := func(slots map[int]int32) dbEntry {
		p := make([]int32, 0x20)
		for k, v := range slots {
			p[k] = v
		}
		return dbEntry{name: "row", params: p}
	}
	return &mapload.Table{
		Units:  dbCollection{{}, row(map[int]int32{0x1d: 40, 0x1e: 2}), row(map[int]int32{0x1d: 41, 0x1e: 2})},
		Humans: dbCollection{{}, row(map[int]int32{0x10: 5, 0x18: 1234})},
	}
}

func openAuthoring(t *testing.T) ([]byte, *ui.MapEditor) {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 3*256 + 128, Y: 4*256 + 77, ClassID: 40, ClassSubID: 2},
		{X: 7*256 + 3, Y: 8*256 + 5, ClassID: 40, ClassSubID: 2},
	}})
	x := &MapInspector{tiles: &terrain.Tileset{}, table: authoringTable()}
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

func authoredUnits(t *testing.T, e *ui.MapEditor) []alm.Unit {
	t.Helper()
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	return m.Units
}

func clickCell(e *ui.MapEditor, cell image.Point) {
	x, y := e.Document().Viewer.Camera().WorldToScreen(float64(cell.X*32+16), float64(cell.Y*32+16))
	p := image.Pt(int(x), int(y))
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// enterField opens the field prompt, tabs to the field-th row, replaces the
// prefilled value with text and presses Enter.
func enterField(e *ui.MapEditor, field int, text string) error {
	_ = e.Key("field")
	for i := 0; i < field; i++ {
		_ = e.Key("tab")
	}
	for i := 0; i < 24; i++ {
		_ = e.Key("backspace")
	}
	e.TextInput(text)
	return e.Key("enter")
}

func TestMapEditorClonesAUnitOntoAClickedCell(t *testing.T) {
	b, e := openAuthoring(t)
	e.Select(0, true)
	clickCell(e, image.Pt(9, 10))
	if len(authoredUnits(t, e)) != 2 {
		t.Fatal("a click without C cloned")
	}
	e.Select(0, true)
	_ = e.Key("clone")
	clickCell(e, image.Pt(9, 10))
	u := authoredUnits(t, e)
	if len(u) != 3 || u[2].X != 9*256+128 || u[2].Y != 10*256+77 || u[2].ClassID != 40 || u[2].UnitID != 1 || u[2].GroupID != u[0].GroupID {
		t.Fatalf("clone %+v", u)
	}
	if r := e.Document().Records[e.Selected()]; r.Kind != "Unit" || r.Index != 2 || r.Cell != image.Pt(9, 10) {
		t.Fatalf("the clone is not selected: %+v", r)
	}
	cloned := append([]byte(nil), e.Document().SourceBytes...)
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) || e.Document().Edits.CanUndo() {
		t.Fatal("one clone was not one exact undo step")
	}
	if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, cloned) {
		t.Fatal("redo did not restore the clone")
	}
	if _, err := e.Document().Edits.CloneUnit(0, image.Pt(16, 3)); err == nil || !bytes.Equal(e.Document().SourceBytes, cloned) {
		t.Fatal("a clone outside the map was accepted")
	}
}

func TestMapEditorDeletesTheSelectedUnit(t *testing.T) {
	b, e := openAuthoring(t)
	_ = e.Key("delete")
	if e.Message() == "" || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("delete without a selection changed the map")
	}
	_ = e.Key("escape")
	e.Select(0, true)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	u := authoredUnits(t, e)
	if len(u) != 1 || u[0].X != 7*256+3 || e.Selected() != -1 {
		t.Fatalf("after delete %+v", u)
	}
	deleted := append([]byte(nil), e.Document().SourceBytes...)
	if len(deleted) != len(b)-70 {
		t.Fatal("delete did not remove one 70-byte record")
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("undo did not restore the deleted unit's bytes")
	}
	_ = e.Key("redo")
	if !bytes.Equal(e.Document().SourceBytes, deleted) {
		t.Fatal("redo did not delete again")
	}
}

func TestMapEditorSetsUnitFieldsWithChecks(t *testing.T) {
	b, e := openAuthoring(t)
	e.Select(1, true)
	steps := []struct {
		field  int
		refuse string
		accept string
		check  func(alm.Unit) bool
	}{
		{0, "2", "1", func(u alm.Unit) bool { return u.Owner == 1 }},
		{1, "42", "41", func(u alm.Unit) bool { return u.ClassID == 41 }},
		{2, "999", "1234", func(u alm.Unit) bool { return u.DefID == 1234 }},
		{3, "40000", "15", func(u alm.Unit) bool { return u.HasCurrentHP && u.CurrentHP == 15 }},
	}
	history := [][]byte{b}
	for _, s := range steps {
		before := append([]byte(nil), e.Document().SourceBytes...)
		if err := enterField(e, s.field, s.refuse); err == nil || e.Message() == "" || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("field %d accepted %s", s.field, s.refuse)
		}
		_ = e.Key("escape")
		if err := enterField(e, s.field, s.accept); err != nil {
			t.Fatalf("field %d refused %s: %v", s.field, s.accept, err)
		}
		if u := authoredUnits(t, e); !s.check(u[1]) || u[0] != authoredUnitsOf(t, b)[0] {
			t.Fatalf("field %d: %+v", s.field, u)
		}
		history = append(history, append([]byte(nil), e.Document().SourceBytes...))
	}
	for i := len(history) - 2; i >= 0; i-- {
		_ = e.Key("undo")
		if !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("undo %d not byte exact", i)
		}
	}
	for i := 1; i < len(history); i++ {
		_ = e.Key("redo")
		if !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("redo %d not byte exact", i)
		}
	}
}

func authoredUnitsOf(t *testing.T, b []byte) []alm.Unit {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	return m.Units
}
