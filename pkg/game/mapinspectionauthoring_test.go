package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func TestMapEditorRecordAuthoringMissingRoutes(t *testing.T) {
	t.Run("structure identity", func(t *testing.T) {
		_, e := openRecords(t)
		e.Select(recordIndex(t, e, "Structure", 0), true)
		_ = e.Key("clone")
		clickCell(e, image.Pt(18, 10))
		m, _ := editedMap(t, e)
		if len(m.Objects) != 3 || m.Objects[2].Field12 == m.Objects[0].Field12 {
			t.Fatal("the structure clone reuses the source script identity")
		}
	})
	t.Run("first structure", func(t *testing.T) {
		_, e := openRecords(t)
		n := len(e.Document().Records)
		recordAddInput(t, e, "structure", image.Pt(18, 10), 5)
		if len(e.Document().Records) != n+1 {
			t.Fatal("J/kind/owner/cell adds no structure")
		}
	})
	t.Run("unit group", func(t *testing.T) {
		_, e := openRecords(t)
		if _, err := e.Document().Edits.SetField("Unit", 0, "group", 77); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("initial stock", func(t *testing.T) {
		_, e := openRecords(t)
		recordAddInput(t, e, "unit", image.Pt(11, 11), 40)
		b := bytes.Clone(e.Document().SourceBytes)
		if err := e.Key("stock"); err != nil || bytes.Equal(b, e.Document().SourceBytes) {
			t.Fatalf("K adds no selected-unit stock: %v", err)
		}
	})
	t.Run("item recipe", func(t *testing.T) {
		_, e := openRecords(t)
		e.Select(recordIndex(t, e, "Stock", 0), true)
		_ = e.Key("recipe")
		e.TextInput("1")
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		for _, r := range e.Document().Records {
			if r.Kind == "Enchantment" {
				return
			}
		}
		t.Fatal("I/item number/Enter adds no recipe")
	})
}

func TestMapEditorRecordAuthoringWindowHistoryAndRefusal(t *testing.T) {
	_, old := openRecords(t)
	x := old.Document().Edits.(*mapInspectionEdits).inspector
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	history := [][]byte{bytes.Clone(e.Document().SourceBytes)}
	step := func() { history = append(history, bytes.Clone(e.Document().SourceBytes)) }
	recordAddInput(t, e, "structure", image.Pt(12, 12), 5)
	step()
	recordFieldInput(t, e, "owner", 2)
	step()
	recordAddInput(t, e, "unit", image.Pt(10, 10), 40)
	step()
	recordFieldInput(t, e, "group", 77)
	step()
	if err := e.Key("stock"); err != nil {
		t.Fatal(err)
	}
	step()
	recordFieldInput(t, e, "additem", 0x0101)
	step()
	_ = e.Key("recipe")
	e.TextInput("1")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	step()
	recordFieldInput(t, e, "addeffect", 12)
	step()
	recordFieldInput(t, e, "low1", 9)
	step()
	before := bytes.Clone(e.Document().SourceBytes)
	host := e.Document().Edits.(*mapInspectionEdits)
	for _, call := range []func() error{
		func() error { _, err := host.AddStructure(image.Pt(31, 31), 5); return err },
		func() error { _, err := host.SetField("Structure", 0, "owner", 3); return err },
		func() error { _, err := host.AddStock(0); return err },
		func() error { _, err := host.Delete("Unit", 0); return err },
		func() error { _, err := host.AddItemRecipe("Stock", 0, 2); return err },
		func() error { _, err := host.SetField("Stock", 0, "recipe1", 2); return err },
	} {
		if err := call(); err == nil || !bytes.Equal(e.Document().SourceBytes, before) || !bytes.Equal(host.model.Bytes(), before) {
			t.Fatal("refused host edit changed the document or model", err)
		}
	}
	for i := len(history) - 2; i >= 0; i-- {
		if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("undo %d did not restore exact bytes: %v", i, err)
		}
	}
	for i := 1; i < len(history); i++ {
		if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, history[i]) {
			t.Fatalf("redo %d did not restore exact bytes: %v", i, err)
		}
	}
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil || len(m.Objects) != 1 || m.Objects[0].Field0E != 2 || m.Units[0].GroupID != 77 || len(m.Enchantments) != 1 || m.Enchantments[0].Elements[0].Low != 9 {
		t.Fatal("window-authored state differs", m, err)
	}
	cleanupStart := bytes.Clone(e.Document().SourceBytes)
	e.Select(recordIndex(t, e, "Stock", 0), false)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	unit, err := host.model.UnitRecord(0)
	if err != nil || binary.LittleEndian.Uint32(unit[0x18:]) != 0 || host.model.Count(mapedit.KindLoot) != 0 {
		t.Fatal("stock deletion left a unit link or stock", err)
	}
	stockRemoved := bytes.Clone(e.Document().SourceBytes)
	e.Select(recordIndex(t, e, "Unit", 0), false)
	if err := e.Key("delete"); err != nil || host.model.UnitCount() != 0 {
		t.Fatal("stock cleanup did not permit unit deletion", err)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(stockRemoved, e.Document().SourceBytes) {
		t.Fatal("unit cleanup Undo differs", err)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(cleanupStart, e.Document().SourceBytes) {
		t.Fatal("stock cleanup Undo did not restore its link atomically", err)
	}
}

func TestMapEditorReportsImportedOrphanStock(t *testing.T) {
	_, e := openRecords(t)
	host := e.Document().Edits.(*mapInspectionEdits)
	_, err := host.change(func(m *mapedit.Editor) (bool, error) {
		rec := m.GroundSack(0, 0, 0x0101)
		binary.LittleEndian.PutUint32(rec[4:], 999)
		_, err := m.Place(mapedit.KindLoot, rec)
		return err == nil, err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range host.Report() {
		if f.Rule == "stock-link" && f.Kind == "Stock" && f.Index == 1 && strings.Contains(f.Text, "resolves to 0 placed units") {
			return
		}
	}
	t.Fatal("the report did not name an imported orphan stock")
}
