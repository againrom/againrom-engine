package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func TestEditorTrapMapSelectionMovementAndFields(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	i := recordIndex(t, e, "Enchantment", 0)
	r := e.Document().Records[i]
	if !r.Spatial || r.Cell != image.Pt(3, 4) || r.Size != image.Pt(1, 1) {
		t.Fatalf("trap has no map mark: %+v", r)
	}
	e.Select(-1, false)
	for range 6 {
		clickCell(e, image.Pt(3, 4))
		if e.Selected() == i {
			break
		}
	}
	if e.Selected() != i {
		t.Fatal("cell click never selected trap")
	}
	if _, ok := e.FieldRect(0); !ok {
		t.Fatal("cell click did not open spell popup")
	}
	e.Select(i, true)
	dragCell(e, image.Pt(3, 4), image.Pt(12, 13))
	got, _ := s.model.Record(mapedit.KindEnchantment, 0)
	seed := bytes.Clone(got)
	binary.LittleEndian.PutUint32(seed[4:], 3)
	binary.LittleEndian.PutUint32(seed[8:], 4)
	want := bytes.Clone(seed)
	binary.LittleEndian.PutUint32(want[4:], 12)
	binary.LittleEndian.PutUint32(want[8:], 13)
	if !bytes.Equal(got, want) {
		t.Fatalf("drag did not move only trigger: %x; %s", got, e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("drag undo", err)
	}
	for _, tc := range []struct {
		key   string
		off   int
		value uint16
	}{
		{"spell", 18, 7}, {"power", 20, 99}, {"sourcex", 26, 11}, {"sourcey", 28, 12}, {"targetx", 32, 13}, {"targety", 34, 14},
	} {
		next, err := s.SetField(ui.KindEnchantment, 0, tc.key, int64(tc.value))
		if err != nil || next == nil {
			t.Fatal(tc.key, err)
		}
		want := bytes.Clone(before)
		at := bytes.Index(want, seed)
		if at < 0 {
			t.Fatal("fixture record missing")
		}
		binary.LittleEndian.PutUint16(want[at+tc.off:], tc.value)
		if !bytes.Equal(want, s.model.Bytes()) {
			t.Fatal(tc.key, "changed unrelated bytes")
		}
		saved, err := s.model.SaveBytes()
		if err != nil || !bytes.Equal(want, saved) {
			t.Fatal(tc.key, "save changed unrelated bytes", err)
		}
		if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal(tc.key, "undo", err)
		}
	}
}

func TestEditorObjectSpellsFollowOwnerWithoutWriting(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	for _, tc := range []struct {
		spell int
		kind  ui.RecordKind
	}{{1, ui.KindUnit}, {2, ui.KindStructure}} {
		before := s.model.Bytes()
		if _, err := s.Move(ui.KindEnchantment, tc.spell, image.Pt(15, 15)); err == nil || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal("owner spell moved independently")
		}
		spell, _ := s.model.Record(mapedit.KindEnchantment, tc.spell)
		next, err := s.Move(tc.kind, 0, image.Pt(14, 14))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range next.Records {
			if r.Kind == ui.KindEnchantment && r.Index == tc.spell {
				found = r.Spatial && r.Cell == image.Pt(14, 14)
			}
		}
		got, _ := s.model.Record(mapedit.KindEnchantment, tc.spell)
		if !found || !bytes.Equal(spell, got) {
			t.Fatal("owner move lost mark or rewrote spell")
		}
		if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal("owner undo", err)
		}
	}
}

func TestEditorCellSpellPaletteOffersAllKinds(t *testing.T) {
	e := cellSpellFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	choices := s.inspector.cellSpellPalette()
	if len(choices) != 3 {
		t.Fatal(choices)
	}
	for i, kind := range []uint16{0, 8, 4} {
		if choices[i].Value != int64(mapedit.SpellConstructor(kind)) || choices[i].Label == "" {
			t.Fatal("palette kind", kind, choices)
		}
	}
	if err := e.Key("cellspells"); err != nil || e.ListRowCount() != 3 {
		t.Fatal("spatial list", err, e.ListRowCount())
	}
}
