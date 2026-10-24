package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// objectSpellMap is the records map with two casting records of structure 7,
// a trap, a spell entry of unit 0 and two item recipes, and two ground sacks
// whose item links name the recipes (links 5 and 6).
func objectSpellMap(t *testing.T) (*ui.MapEditor, [][]byte) {
	t.Helper()
	recs := [][]byte{
		cellSpellBytes(50, 4, 5, 4, 7, 5, 100),
		cellSpellBytes(1, 6, 6, 0, 0, 13, 20, [3]uint16{6, 6, 9078}, [3]uint16{6, 7, 9396}),
		cellSpellBytes(0, 3, 4, 8, 0, 6, 30),
		cellSpellBytes(60, 5, 5, 4, 7, 7, 40),
		cellSpellBytes(0, 0, 0, 0, 0, 9, 1),
		cellSpellBytes(0, 0, 0, 0, 0, 10, 2),
	}
	e := spellMap(t, recs...)
	s := e.Document().Edits.(*mapInspectionEdits)
	for i, link := range []uint32{5, 6} {
		loot := s.model.GroundSack(uint32(12+i)*256, 12*256, 257)
		binary.LittleEndian.PutUint32(loot[26:], link)
		if _, err := s.model.Place(mapedit.KindLoot, loot); err != nil {
			t.Fatal(err)
		}
	}
	return e, recs
}

// checkSpells compares the map's type-9 records with want and the ground
// sacks' item links with links.
func checkSpells(t *testing.T, s *mapInspectionEdits, what string, want [][]byte, links ...uint32) {
	t.Helper()
	if n := s.model.Count(mapedit.KindEnchantment); n != len(want) {
		t.Fatalf("%s: %d type-9 records, want %d", what, n, len(want))
	}
	for i, w := range want {
		if got, _ := s.model.Record(mapedit.KindEnchantment, i); !bytes.Equal(got, w) {
			t.Fatalf("%s: record %d %x want %x", what, i, got, w)
		}
	}
	n := s.model.Count(mapedit.KindLoot)
	for i, link := range links {
		rec, _ := s.model.Record(mapedit.KindLoot, n-len(links)+i)
		if got := binary.LittleEndian.Uint32(rec[26:]); got != link {
			t.Fatalf("%s: sack %d links recipe %d, want %d", what, i, got, link)
		}
	}
}

// Deleting a structure or a unit deletes the cell spells its id names, as
// the original editor's delete does (ALM-EDITDEL-299); the later item links
// renumber (ALM-EDITLINK-298) and one Undo restores every byte.
func TestEditorObjectDeleteTakesItsCellSpells(t *testing.T) {
	e, recs := objectSpellMap(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	structures, units := s.model.Count(mapedit.KindStructure), s.model.Count(mapedit.KindUnit)

	e.Select(recordIndex(t, e, "Structure", 0), false)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if s.model.Count(mapedit.KindStructure) != structures-1 {
		t.Fatal("structure not deleted")
	}
	checkSpells(t, s, "structure delete", [][]byte{recs[1], recs[2], recs[4], recs[5]}, 3, 4)
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("one Undo did not restore the structure delete", err)
	}

	e.Select(recordIndex(t, e, "Unit", 0), false)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if s.model.Count(mapedit.KindUnit) != units-1 {
		t.Fatal("unit not deleted")
	}
	checkSpells(t, s, "unit delete", [][]byte{recs[0], recs[1], recs[3], recs[4], recs[5]}, 4, 5)
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("one Undo did not restore the unit delete", err)
	}

	// The pointer route: a Shift drag over the unit and the structure on
	// (3,4), then the selection popup's Delete button.
	e.Layout(1800, 900)
	e.Document().Viewer.SetFlat(true)
	e.ShowCell(image.Pt(4, 5))
	selectArea(e, image.Pt(2, 3), image.Pt(5, 6))
	r, ok := e.SelectionButtonRect(2)
	messageClick(t, e, r, ok)
	if s.model.Count(mapedit.KindStructure) != structures-1 || s.model.Count(mapedit.KindUnit) != units-1 {
		t.Fatalf("selection delete left %d structures, %d units (%s)", s.model.Count(mapedit.KindStructure), s.model.Count(mapedit.KindUnit), e.StatusLine())
	}
	checkSpells(t, s, "selection delete", [][]byte{recs[1], recs[4], recs[5]}, 2, 3)
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("one Undo did not restore the selection delete", err)
	}
}

// A cell spell whose structure's id changed names no object: it offers no
// field and takes no edit, and Delete or spell 0 removes it.
func TestEditorUnboundCellSpellDeletes(t *testing.T) {
	e := spellMap(t, cellSpellBytes(50, 4, 5, 4, 7, 5, 100))
	s := e.Document().Edits.(*mapInspectionEdits)
	e.Select(recordIndex(t, e, "Structure", 0), false)
	recordFieldInput(t, e, "identity", 8)
	if err := spellRecordBound(s.model, 0); err == nil {
		t.Fatal("the record still binds after the identity change")
	}
	before := s.model.Bytes()
	if f := s.Fields("Enchantment", 0); len(f) != 0 {
		t.Fatalf("unbound record offers fields %+v", f)
	}
	if _, err := s.SetField("Enchantment", 0, "power", 30); err == nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("unbound record took an edit")
	}
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if s.model.Count(mapedit.KindEnchantment) != 0 {
		t.Fatal("Delete kept the unbound record")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("undo of the unbound delete", err)
	}
	if _, err := s.SetField("Enchantment", 0, "spell", 0); err != nil || s.model.Count(mapedit.KindEnchantment) != 0 {
		t.Fatal("spell 0 kept the unbound record", err)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("undo of the unbound spell 0", err)
	}
}
