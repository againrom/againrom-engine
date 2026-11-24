package game

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// bookIDsOf are the spell numbers of a book field's cells, 0 for an empty
// cell.
func bookIDsOf(t *testing.T, f ui.InspectionField) []uint32 {
	t.Helper()
	if f.Book == nil {
		t.Fatalf("field %s holds no book", f.Key)
	}
	return bookIDs(f.Book.Cells)
}

// unitBookRecords are the records of the map image raw that give unit 0 of
// the map spell id.
func unitBookRecords(t *testing.T, raw []byte, id uint16) (*alm.Map, []int) {
	t.Helper()
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m, m.UnitSpellRecords(m.Units[0].UnitID)[id]
}

// mageViewsMap is the views map whose unit is a mage: every Humans row knows
// spell known, and the unit holds a maximum mana of 300.
func mageViewsMap(t *testing.T, known uint16) (*mapInspectionEdits, []byte) {
	t.Helper()
	s, _ := viewsMap(t)
	humans := s.inspector.table.Humans.(dbCollection)
	for row := range humans {
		p := append(slices.Clone(humans[row].params), make([]int32, max(0, 26-len(humans[row].params)))...)
		p[25] = 1 << known
		humans[row].params = p
	}
	if err := s.model.SetField(mapedit.KindUnit, 0, "manamax", 300); err != nil {
		t.Fatal(err)
	}
	s.unitViews, s.inspector.recordOptions = nil, nil
	if spells := s.unitViewSheets()[0].Values.Spells; !slices.Contains(spells, int32(known)) {
		t.Fatalf("the mage knows %v, want spell %d", spells, known)
	}
	return s, s.model.Bytes()
}

// TestEditorUnitBookTogglesTheUnitSpellRecord: in English and Russian, at
// 1920x1080, the unit window's SPELLCASTER caption opens the spell book; a
// press on an off spell writes the unit spell record the original editor
// writes (A&8, the unit's id, the unit's cell, power 100), the saved map
// holds it and a reopened map shows the spell on; a second press takes it
// away, a spell the kind knows stays on, and Undo restores the map.
func TestEditorUnitBookTogglesTheUnitSpellRecord(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(lang, func(t *testing.T) {
			known := spellBookRows(nil)[1]
			s, raw := mageViewsMap(t, known)
			x := s.inspector
			x.archives.Base.Profile.Language = lang
			e, c := motionWindow(t, s, lang, false, false)
			fields := e.ObjectFields()
			at := fieldIndex(t, fields, unitBookField)
			book := fields[at]
			if book.Label != x.word("editor.unit.book") || book.Book == nil {
				t.Fatalf("book field %q holds no book", book.Label)
			}
			caption := slices.Index(e.PopupLines(), "SPELLCASTER")
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: caption})
			c.click(e, r, ok && caption >= 0, "the SPELLCASTER caption")
			_, boxes := e.PopupCanvasBoxes()
			cell := slices.IndexFunc(book.Book.Cells, func(c ui.SpellEntry) bool { return c.ID != 0 && c.Unavailable })
			if len(boxes) < len(book.Book.Cells) || cell < 0 {
				t.Fatalf("the book window shows %d boxes for %d cells, first off cell %d:\n%s", len(boxes), len(book.Book.Cells), cell, strings.Join(e.PopupLines(), "\n"))
			}
			if lines, note := strings.Join(e.PopupLines(), " "), x.inspectionUTF8(book.Meaning); !strings.Contains(lines, note) {
				t.Fatalf("the book window states no note %q:\n%s", note, lines)
			}
			id := uint16(book.Book.Cells[cell].ID)
			c.click(e, boxes[cell], true, "an off spell")
			saved, err := s.model.SaveBytes()
			if err != nil {
				t.Fatal(err)
			}
			m, given := unitBookRecords(t, saved, id)
			if len(given) != 1 {
				t.Fatalf("the saved map gives spell %d by %d records, want 1", id, len(given))
			}
			u, rec := m.Units[0], m.Enchantments[given[0]]
			if rec.A&8 == 0 || rec.SpellRaw != uint32(id)|unitBookPower<<16 || rec.X != uint32(u.X>>8) || rec.Y != uint32(u.Y>>8) {
				t.Fatalf("unit spell record %+v, want spell %d power %d at the unit's cell %d,%d", rec, id, unitBookPower, u.X>>8, u.Y>>8)
			}
			if _, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal); err != nil {
				t.Fatalf("the game refuses the map with the unit spell record: %v", err)
			}
			reopened, err := mapedit.New(saved)
			if err != nil {
				t.Fatal(err)
			}
			kept := s.model
			s.model, s.unitViews = reopened, nil
			again := s.Fields(ui.KindUnit, 0)
			if f := again[fieldIndex(t, again, unitBookField)]; f.Book.Cells[cell].Unavailable {
				t.Fatalf("the reopened map shows spell %d off", id)
			}
			s.model, s.unitViews = kept, nil

			_, boxes = e.PopupCanvasBoxes()
			c.click(e, boxes[cell], true, "the spell turned on")
			if _, given := unitBookRecords(t, s.model.Bytes(), id); len(given) != 0 {
				t.Fatalf("a second press left %d records of spell %d", len(given), id)
			}
			for range 2 {
				if _, err := s.Undo(); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(s.model.Bytes(), raw) {
				t.Fatal("Undo did not restore the map")
			}

			// A spell the unit's kind knows stays on, in the editor's words.
			kind := slices.IndexFunc(book.Book.Cells, func(c ui.SpellEntry) bool { return c.ID == uint32(known) })
			if kind < 0 || book.Book.Cells[kind].Unavailable {
				t.Fatalf("the kind's spell %d is not on in the book", known)
			}
			_, boxes = e.PopupCanvasBoxes()
			c.click(e, boxes[kind], true, "the kind's spell")
			want := x.inspectionUTF8(strings.Replace(x.word("editor.err.bookkind"), "%d", strconv.Itoa(int(known)), 1))
			if lines := strings.Join(e.PopupLines(), " "); !strings.Contains(lines, want) {
				t.Fatalf("taking a kind's spell away states no %q:\n%s", want, lines)
			}
			if !bytes.Equal(s.model.Bytes(), raw) {
				t.Fatal("a refused press changed the map")
			}
		})
	}
}

// TestEditorUnitBookFollowsTheModSpellTable: the book's toggles are the
// spells the map's mod states, in its order; a spell the mod turns off has
// no toggle and its key is refused, and a press writes the spell the mod
// moved into the cell.
func TestEditorUnitBookFollowsTheModSpellTable(t *testing.T) {
	s, raw := viewsMap(t)
	x := s.inspector
	ground := spellBookRows(x.table)
	plain := s.Fields(ui.KindUnit, 0)
	if got := bookIDsOf(t, plain[fieldIndex(t, plain, unitBookField)]); !slices.Equal(got, bookIDs(cellsOf(ground))) {
		t.Fatalf("without a mod the book holds %v, want the game's rows %v", got, ground)
	}
	x.table.Mods.Spells = spellSwitchData(t)
	s.unitViews = nil
	modded := s.Fields(ui.KindUnit, 0)
	f := modded[fieldIndex(t, modded, unitBookField)]
	want := []uint32{3, 0, 1}
	for _, id := range ground[3:23] {
		want = append(want, uint32(id))
	}
	if got := bookIDsOf(t, f); !slices.Equal(got, want) {
		t.Fatalf("under the mod the book holds %v, want %v", got, want)
	}
	k := slices.IndexFunc(ground, func(id uint16) bool { return !slices.Contains(want, uint32(id)) })
	if k < 0 {
		t.Fatalf("the mod turns no spell of %v off", ground)
	}
	off := ground[k]
	if _, err := s.SetField(ui.KindUnit, 0, unitBookPrefix+strconv.Itoa(int(off)), 1); err == nil || !bytes.Equal(s.model.Bytes(), raw) {
		t.Fatalf("a spell the mod turns off took a toggle: %v", err)
	}
	if !f.Book.Cells[0].Unavailable {
		t.Fatal("the unit knows the mod's first spell before the press")
	}
	if _, err := s.SetField(ui.KindUnit, 0, unitBookPrefix+strconv.Itoa(int(f.Book.Cells[0].ID)), 1); err != nil {
		t.Fatal(err)
	}
	if _, given := unitBookRecords(t, s.model.Bytes(), uint16(want[0])); len(given) != 1 {
		t.Fatalf("the press on the mod's first cell gave spell %d by %d records", want[0], len(given))
	}
}

// cellsOf are book cells of spell numbers ids.
func cellsOf(ids []uint16) []ui.SpellEntry {
	out := make([]ui.SpellEntry, len(ids))
	for i, id := range ids {
		out[i].ID = uint32(id)
	}
	return out
}

// TestSecondEditorRefusesTheUnitBookAndManaMax: a second-game unit offers no
// maximum mana, and its spell book states the refusal in the second game's
// words; a book press is refused and the map stays as it was.
func TestSecondEditorRefusesTheUnitBookAndManaMax(t *testing.T) {
	raw := secondFixtureMap()
	x := secondTestInspector(t)
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	before := host.model.Bytes()
	fields := host.Fields(ui.KindUnit, 0)
	if _, ok := fieldOfKey(fields, "manamax"); ok {
		t.Fatal("a second-game unit offers maximum mana")
	}
	word := x.word("editor.map.secondbook")
	f, ok := fieldOfKey(fields, unitBookField)
	if !ok || f.Book != nil || len(f.Lines) != 1 || f.Lines[0].Text != x.inspectionUTF8(word) {
		t.Fatalf("second-game book field %+v, want the refusal %q", f, word)
	}
	if _, err := host.SetField(ui.KindUnit, 0, unitBookPrefix+"1", 1); err == nil || err.Error() != word {
		t.Fatalf("a second-game book press: %v, want %q", err, word)
	}
	if _, err := host.SetField(ui.KindUnit, 0, "manamax", 300); err == nil {
		t.Fatal("a second-game unit took maximum mana")
	}
	if !bytes.Equal(host.model.Bytes(), before) {
		t.Fatal("a refused edit changed the second-game map")
	}
}

// TestEditorDeletingAUnitTakesItsSpellRecords: a unit's spell records go with
// the unit in one edit, so the next unit that takes its id does not inherit
// them, and one Undo brings unit and records back.
func TestEditorDeletingAUnitTakesItsSpellRecords(t *testing.T) {
	// A record of the unit arm that holds no cell: the game binds it by the
	// id alone, so it goes with the unit all the same.
	const spell = 6
	e := spellMap(t, cellSpellBytes(0, 0, 0, 8, 0, spell, 30))
	s := e.Document().Edits.(*mapInspectionEdits)
	withBook := bytes.Clone(s.model.Bytes())
	if _, given := unitBookRecords(t, withBook, spell); len(given) != 1 {
		t.Fatalf("%d records give the unit spell %d, want 1", len(given), spell)
	}
	if err := deleteObject(s.model, mapedit.KindUnit, 0); err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(s.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range m.Enchantments {
		if e.A&8 != 0 && uint16(e.SpellRaw&0xff) == spell {
			t.Fatalf("record %d still gives spell %d after the unit was deleted", i, spell)
		}
	}
	if !s.model.Undo() {
		t.Fatal("nothing to undo")
	}
	if !bytes.Equal(s.model.Bytes(), withBook) {
		t.Fatal("one Undo did not restore the unit and its record")
	}
}
