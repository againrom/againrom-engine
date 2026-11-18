package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func palettePress(t *testing.T, e *ui.MapEditor, query ui.ControlQuery) {
	t.Helper()
	r, ok := e.FindControl(query)
	trigClick(t, e, r, ok, "palette control")
}

func TestEditorStructurePaletteMatchesAddRecordBytes(t *testing.T) {
	raw, e := openRecords(t)
	_, old := openRecords(t)
	e.Layout(1920, 1080)
	if err := e.Key("structure"); err != nil {
		t.Fatal(err)
	}
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: 0})
	clickCell(e, image.Pt(18, 12))
	addFixturePlacement(t, old, "structure", map[string]string{"kind": "5", "cell": "18,12"})
	m, _ := editedMap(t, e)
	if !bytes.Equal(e.Document().SourceBytes, old.Document().SourceBytes) || len(m.Objects) != 3 || m.Objects[2].Kind != 5 {
		t.Fatalf("placed structure differs from the same answers to Add (%s)", e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatalf("undo after one structure: %v", err)
	}
}

func TestEditorSackPaletteMatchesAddRecordBytes(t *testing.T) {
	raw, e := openRecords(t)
	_, old := openRecords(t)
	e.Layout(1920, 1080)
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlBar, Key: "sacks"})
	for _, n := range []int{0, 1} {
		palettePress(t, e, ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: n})
	}
	clickCell(e, image.Pt(14, 14))
	addFixtureRecord(t, old, ui.KindSack, map[string]string{"cell": "14,14", "item": "257|514"})
	_, loot := editedMap(t, e)
	if !bytes.Equal(e.Document().SourceBytes, old.Document().SourceBytes) || len(loot) != 2 ||
		len(loot[1].Elements) != 2 || loot[1].Elements[0].ItemCode() != 257 || loot[1].Elements[1].ItemCode() != 514 {
		t.Fatalf("placed sack differs from Add with the same items (%s)", e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatalf("undo after one sack: %v", err)
	}
}

func spellPaletteCell(t *testing.T, e *ui.MapEditor, id int64) int {
	t.Helper()
	choices := e.Document().Edits.(*mapInspectionEdits).RecordChoices(ui.KindEnchantment, -1, "spell")
	n := slices.IndexFunc(choices, func(c ui.InspectionChoice) bool { return c.Value == id })
	if n < 0 {
		t.Fatalf("spell %d is absent", id)
	}
	return n
}

func TestEditorSpellPaletteStructureClickWritesSignedBinding(t *testing.T) {
	const field12 = 0x8001
	const oldLink = 0xffff8001
	_, e := openRecordsWithBinding(t, field12)
	_, old := openRecordsWithBinding(t, field12)
	e.Layout(1920, 1080)
	if err := e.Key("casts"); err != nil {
		t.Fatal(err)
	}
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlListButton, N: 0, K: 0})
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: spellPaletteCell(t, e, 13)})
	clickCell(e, image.Pt(3, 4))
	m := old.Document().Edits.(*mapInspectionEdits).model
	wantIndex, err := m.AddSpellFrom(mapedit.SpellConstructor(4), 3, 4, oldLink)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetField(mapedit.KindEnchantment, wantIndex, "spell", 13); err != nil {
		t.Fatal(err)
	}
	want, err := m.Record(mapedit.KindEnchantment, wantIndex)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Document().Edits.(*mapInspectionEdits).model.Record(mapedit.KindEnchantment, 0)
	if err != nil || !bytes.Equal(got, want) || binary.LittleEndian.Uint32(got[14:]) != oldLink {
		t.Fatalf("casting structure link and record: %x, want %x (%v)", got, want, err)
	}
}

func TestEditorSpellPaletteMatchesAddRecordBytes(t *testing.T) {
	for _, c := range []struct {
		name string
		cell image.Point
		kind uint16
		copy bool
	}{
		{name: "trap", cell: image.Pt(20, 20)},
		{name: "unit", cell: image.Pt(3, 4), kind: 8},
		{name: "unit spell copy", cell: image.Pt(3, 4), kind: 8, copy: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			var recs [][]byte
			template := mapedit.SpellConstructor(c.kind)
			if c.copy {
				recs = append(recs, cellSpellBytes(19, 3, 4, c.kind, 0, 13, 20))
				template = 0
			}
			e, command := spellMap(t, recs...), spellMap(t, recs...)
			before := bytes.Clone(e.Document().SourceBytes)
			e.Layout(1920, 1080)
			palettePress(t, e, ui.ControlQuery{Role: ui.ControlBar, Key: "cellspells"})
			if c.copy {
				palettePress(t, e, ui.ControlQuery{Role: ui.ControlListRow, N: 0})
			} else {
				if c.kind == 8 {
					palettePress(t, e, ui.ControlQuery{Role: ui.ControlListButton, N: 0, K: 1})
				}
				palettePress(t, e, ui.ControlQuery{Role: ui.ControlPlaceGridCell, N: spellPaletteCell(t, e, 13)})
			}
			e.Fit()
			clickCell(e, c.cell)
			values := map[string]string{"template": strconv.Itoa(template), "spell": "13",
				"cell": strconv.Itoa(c.cell.X) + "," + strconv.Itoa(c.cell.Y)}
			if c.kind == 8 {
				values["link"] = "0"
			}
			if c.copy {
				values["power"] = "20"
			}
			addFixtureRecord(t, command, ui.KindEnchantment, values)
			got := e.Document().SourceBytes
			if bytes.Equal(got, before) || !bytes.Equal(got, command.Document().SourceBytes) {
				t.Fatalf("palette bytes differ from add_record (%s)", e.Message())
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
				t.Fatalf("one Undo after one spell: %v", err)
			}
		})
	}
}

func TestEditorSpellPaletteCopiesSelectedRecordAndCClones(t *testing.T) {
	source := cellSpellBytes(19, 3, 4, 2, 0, 13, 20, [3]uint16{5, 6, 0xdead}, [3]uint16{3, 4, 0xbeef})
	e := spellMap(t, source)
	e.Layout(1920, 1080)
	s := e.Document().Edits.(*mapInspectionEdits)
	original := s.model.Bytes()
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlBar, Key: "cellspells"})
	list, _ := e.PopupRect()
	palettePress(t, e, ui.ControlQuery{Role: ui.ControlListRow, N: 0})
	if e.Selected() != recordIndex(t, e, "Enchantment", 0) {
		t.Fatalf("spell list did not select the template: selected %d, rows %q", e.Selected(), e.ListLines())
	}
	e.Fit()
	at := image.Pt(28, 27)
	if p := e.CellPoint(at); p.In(list) {
		t.Fatalf("target %v is covered by the palette %v", p, list)
	}
	clickCell(e, at)
	want := bytes.Clone(source)
	binary.LittleEndian.PutUint32(want[4:], uint32(at.X))
	binary.LittleEndian.PutUint32(want[8:], uint32(at.Y))
	got, err := s.model.Record(mapedit.KindEnchantment, 1)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("palette copy: %x, want %x (%v, %q)", got, want, err, e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), original) {
		t.Fatalf("undo after one spell: %v", err)
	}
	e.Select(recordIndex(t, e, "Enchantment", 0), false)
	if err := e.Key("clone"); err != nil {
		t.Fatal(err)
	}
	clickCell(e, at)
	got, err = s.model.Record(mapedit.KindEnchantment, 1)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("C copy: %x, want %x (%v, %q)", got, want, err, e.Message())
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), original) {
		t.Fatalf("undo after one clone: %v", err)
	}
}
