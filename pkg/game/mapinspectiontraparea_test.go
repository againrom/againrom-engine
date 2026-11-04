package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func trapAreaRequest(box image.Rectangle) ui.InspectionNewRecord {
	return ui.InspectionNewRecord{Kind: ui.KindEnchantment, HasCell: true, Cell: box.Min, Area: box, Values: []ui.InspectionFieldValue{
		{Key: "template", Value: int64(mapedit.SpellConstructor(0))}, {Key: "spell", Value: 13}, {Key: "power", Value: 40},
	}}
}

func TestEditorTrapAreaMatchesSingleAddsAndDeletesOneStroke(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	box := image.Rect(10, 10, 13, 12)
	want := s.model.Clone()
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			i, err := want.AddSpellFrom(mapedit.SpellConstructor(0), uint32(x), uint32(y), 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := want.SetFields(mapedit.KindEnchantment, i, mapedit.FieldSet{Key: "spell", Value: 13}, mapedit.FieldSet{Key: "power", Value: 40}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := e.Key("traparea"); err != nil {
		t.Fatal(err)
	}
	dragCell(e, image.Pt(10, 10), image.Pt(12, 11))
	if step, _, ok := e.AddStep(); !ok || step != "spell" {
		t.Fatal("rectangle did not open shared Add", step)
	}
	addSteps(t, e, map[string]string{"spell": "13", "power": "40"})
	if !bytes.Equal(want.Bytes(), s.model.Bytes()) {
		t.Fatal("area differs from six single Adds")
	}
	saved, err := s.model.SaveBytes()
	if err != nil || !bytes.Equal(saved, want.Bytes()) {
		t.Fatal("area save normalized unrelated bytes", err)
	}
	added := s.model.Bytes()
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("one stroke undo", err)
	}
	if err := e.Key("redo"); err != nil || !bytes.Equal(added, s.model.Bytes()) {
		t.Fatal("one stroke redo", err)
	}
	e.Select(recordIndex(t, e, "Enchantment", 5), false)
	if e.Document().Records[e.Selected()].SetSize != 6 {
		t.Fatal("stroke membership missing")
	}
	if err := e.Key("delete"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("Delete did not remove exactly this stroke", err)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(added, s.model.Bytes()) || len(s.model.SpellSet(5)) != 6 {
		t.Fatal("remove undo", err)
	}
	if err := e.Key("redo"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("remove redo", err)
	}
}

func TestEditorTrapAreaKeepsIdenticalOutsiderAndEditedMember(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	r := trapAreaRequest(image.Rect(10, 10, 12, 12))
	single := r
	single.Area = image.Rectangle{}
	if _, err := s.AddRecord(single); err != nil {
		t.Fatal(err)
	}
	before := s.model.Bytes()
	if _, err := s.AddRecord(r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField(ui.KindEnchantment, 5, "power", 99); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Delete(ui.KindEnchantment, 5); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("set removal confused byte-identical outsider", err)
	}
	if _, err := s.Undo(); err != nil || len(s.model.SpellSet(6)) != 4 {
		t.Fatal("set membership did not follow undo", err)
	}
}

func TestEditorTrapAreaRefusalAndUntouchedSaveBytes(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	for _, box := range []image.Rectangle{image.Rect(0, 0, 2, 2), image.Rect(255, 10, 258, 12), image.Rect(-1, 10, 2, 12)} {
		if _, err := s.AddRecord(trapAreaRequest(box)); err == nil || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal("invalid area wrote bytes", box, err)
		}
	}
	r := trapAreaRequest(image.Rect(10, 10, 12, 12))
	r.Values[2].Value = 101
	if _, err := s.AddRecord(r); err == nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("invalid power wrote a partial area", err)
	}
	data, err := s.model.SaveBytes()
	if err != nil || !bytes.Equal(before, data) {
		t.Fatal("untouched map writeback differs", err)
	}
	file := filepath.Join(t.TempDir(), "untouched.alm")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(before, got) {
		t.Fatal("untouched file differs", err)
	}
}

func TestEditorTrapAreaCancelledDragAndDialogWriteNothing(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	if err := e.Key("traparea"); err != nil {
		t.Fatal(err)
	}
	p, q := e.CellPoint(image.Pt(10, 10)), e.CellPoint(image.Pt(12, 12))
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, now)
	if !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("drag preview changed bytes")
	}
	if err := e.Key("escape"); err != nil {
		t.Fatal(err)
	}
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, now)
	if !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("cancelled drag changed bytes")
	}
	if err := e.Key("traparea"); err != nil {
		t.Fatal(err)
	}
	dragCell(e, image.Pt(10, 10), image.Pt(12, 12))
	if err := e.Key("escape"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("cancelled Add changed bytes", err)
	}
}
