package game

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The unit window's spell book: the field that holds it and the prefix of
// the key that turns one spell on (1) or off (0), "book.<spell number>".
const (
	unitBookField  = "spellbook"
	unitBookPrefix = "book."
)

// unitBookPower is the power word of a new unit spell record. Every
// installed record holds 100 (20 of 20 in the English install), and the
// actor arm reads no power (ALM-T9ACTOR-176).
const unitBookPower = 100

// unitBookFields are the unit window's spell book field of placement index.
// Where the format's row offers it, the field holds the book of the map's
// spells in the game's book layout: a cell is on where the unit's kind or
// a record of the map gives the unit the spell. Elsewhere it states the
// row's word.
func (s *mapInspectionEdits) unitBookFields(index int) []ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: unitBookField, Label: x.word("editor.unit.book"), ReadOnly: true}
	if w := s.row().unitBookWord; w != "" {
		f.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(x.word(w))}}
		return []ui.InspectionField{f}
	}
	view, err := s.model.Map()
	sheets := s.unitViewSheets()
	if err != nil || index < 0 || index >= len(view.Units) || index >= len(sheets) {
		return nil
	}
	t := x.table
	ground := spellBookRows(t)
	rows := spellBookOrder(t, ground)
	cells, tips := x.bookCells(mapload.SpellGameOf(t), rows, ground, modSpellIcons(t))
	given := view.UnitSpellRecords(view.Units[index].UnitID)
	for i, id := range rows {
		if id == 0 {
			continue
		}
		word := "editor.unit.book.off"
		switch {
		case slices.Contains(sheets[index].Values.Spells, int32(id)):
			word = "editor.unit.book.kind"
		case len(given[id]) > 0:
			word = "editor.unit.book.on"
		default:
			cells[i].Dim = true
		}
		tips[i] = fmt.Sprintf(x.inspectionUTF8(x.word(word)), tips[i])
	}
	f.Section, f.Meaning = "identity", x.word("editor.unit.book.note")
	f.Book = &ui.InspectionBook{Art: x.spellsLook().art, Cells: cells, Tips: tips, Toggle: unitBookPrefix}
	return []ui.InspectionField{f}
}

// unitBookSpell is the spell number key turns on or off in the unit book;
// false for another key.
func unitBookSpell(key string) (uint16, bool) {
	n, ok := strings.CutPrefix(key, unitBookPrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(n, 10, 16)
	return uint16(id), err == nil
}

// setUnitBook turns spell id of placement index's book on (value nonzero)
// or off on m as one edit. On adds a unit spell record at the unit's cell
// as the original editor's Unit Spellbook Add writes one, through the add
// of a new type-9 record (ALM-EDITT9-328); off deletes each record that
// gives the unit the spell. A spell the unit's kind knows stays on.
func (s *mapInspectionEdits) setUnitBook(m *mapedit.Editor, index int, id uint16, value int64) (bool, error) {
	if w := s.row().unitBookWord; w != "" {
		return false, wordErr(w)
	}
	if _, err := m.UnitRecord(index); err != nil {
		return false, err
	}
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	t := s.inspector.table
	if id == 0 || !slices.Contains(spellBookOrder(t, spellBookRows(t)), id) {
		return false, wordErr("editor.err.bookspell", id)
	}
	u := view.Units[index]
	given := view.UnitSpellRecords(u.UnitID)[id]
	sheets := s.unitViewSheets()
	kind := index < len(sheets) && slices.Contains(sheets[index].Values.Spells, int32(id))
	switch {
	case value != 0 && (kind || len(given) > 0):
		return false, nil
	case value != 0:
		batch, mark := s.batch, m.Mark()
		s.batch = m
		defer func() { s.batch = batch }()
		i, err := s.addCellSpell(ui.InspectionNewRecord{Kind: ui.KindEnchantment, Cell: image.Pt(int(u.X>>8), int(u.Y>>8)), HasCell: true,
			Values: []ui.InspectionFieldValue{{Key: "template", Value: int64(mapedit.SpellConstructor(8))}, {Key: "link", Value: int64(u.UnitID)}}})
		if err == nil {
			err = m.SetField(mapedit.KindEnchantment, i, "spell", int64(id))
		}
		if err == nil {
			err = m.SetField(mapedit.KindEnchantment, i, "power", unitBookPower)
		}
		m.Squash(mark)
		return err == nil, err
	case kind:
		return false, wordErr("editor.err.bookkind", id)
	}
	mark := m.Mark()
	for k := len(given) - 1; k >= 0; k-- {
		if err := m.Delete(mapedit.KindEnchantment, given[k]); err != nil {
			return false, err
		}
	}
	m.Squash(mark)
	return len(given) > 0, nil
}
