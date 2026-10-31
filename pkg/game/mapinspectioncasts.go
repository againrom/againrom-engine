package game

import (
	"encoding/binary"
	"fmt"
	"image"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

var castWords = map[string]struct{ Word string }{
	"spell": {Word: "editor.cast.spell"}, "power": {Word: "editor.cast.power"}, "link": {Word: "editor.cast.link"},
	"triggerx": {Word: "editor.cast.x"}, "triggery": {Word: "editor.cast.y"}, "castkind": {Word: "editor.cast.kind"},
	"sourcex": {Word: "editor.cast.sourcex"}, "sourcey": {Word: "editor.cast.sourcey"},
	"targetx": {Word: "editor.cast.targetx"}, "targety": {Word: "editor.cast.targety"},
	"tile": {Word: "editor.cast.tile"}, "unit": {Word: "editor.cast.unit"}, "structure": {Word: "editor.cast.structure"},
	"trap": {Word: "editor.cast.trap"}, "book": {Word: "editor.cast.book"}, "caster": {Word: "editor.cast.caster"}, "unknown": {Word: "editor.cast.unknown"},
	"mode0": {Word: "editor.cast.mode0"}, "mode1": {Word: "editor.cast.mode1"}, "mode2": {Word: "editor.cast.mode2"}, "mode3": {Word: "editor.cast.mode3"},
	"newtrap": {Word: "editor.cast.newtrap"}, "newbook": {Word: "editor.cast.newbook"}, "newcaster": {Word: "editor.cast.newcaster"},
	"unbound": {Word: "editor.cast.unbound"}, "castpower": {Word: "editor.cast.castpower"},
}

// spellNewWord names the Add window's choice of each kind's new record.
var spellNewWord = map[uint16]string{0: "newtrap", 8: "newbook", 4: "newcaster"}

func (x *MapInspector) inspectionEnchantment(i int, e alm.Enchantment, m *alm.Map) ui.InspectionRecord {
	r := ui.InspectionRecord{Kind: "Enchantment", Section: 9, Index: i, Label: fmt.Sprintf("item recipe %d / tag %d", i+1, e.Tag),
		Details: []string{fmt.Sprintf("X/Y raw: %d / %d", e.X, e.Y), fmt.Sprintf("A/B/C: %d / %d / %d", e.A, e.B, e.C), fmt.Sprintf("Spell raw: %d", e.SpellRaw), fmt.Sprintf("Elements: %v", e.Elements)}}
	if e.X == 0 && e.Y == 0 {
		return r
	}
	word := func(key string) string { return x.inspectionUTF8(x.word(castWords[key].Word)) }
	spell := fmt.Sprintf("%d", uint16(e.SpellRaw))
	for _, c := range x.recordCatalog("spell") {
		if c.Value == int64(uint16(e.SpellRaw)) {
			spell = c.Label
			break
		}
	}
	r.Label = fmt.Sprintf(word("unknown"), e.A)
	switch {
	case e.A < 4 && len(e.Elements) >= 2:
		from, to := "tile", "tile"
		if e.A&1 != 0 {
			from = "structure"
		}
		if e.A&2 != 0 {
			to = "unit"
		}
		r.Label = fmt.Sprintf(word("trap"), word(from), word(to), spell, uint16(e.SpellRaw>>16))
	case e.A == 8:
		r.Label = fmt.Sprintf(word("book"), uint32(e.B)|uint32(e.C)<<16, spell, uint16(e.SpellRaw>>16))
	case e.A == 4:
		r.Label = fmt.Sprintf(word("caster"), uint32(e.B)|uint32(e.C)<<16, spell, uint16(e.SpellRaw>>16), e.Tag)
	}
	if !spellBound(m, e) {
		r.Label = fmt.Sprintf(word("unbound"), r.Label)
	}
	return r
}

func (s *mapInspectionEdits) spellCellAllowed(m *mapedit.Editor, index int) error {
	rec, err := m.Record(mapedit.KindEnchantment, index)
	if err != nil {
		return err
	}
	x, y := binary.LittleEndian.Uint32(rec[4:]), binary.LittleEndian.Uint32(rec[8:])
	if x == 0 && y == 0 {
		return fmt.Errorf("cell spells cannot use the item-recipe cell 0,0")
	}
	return s.admitCell(m, image.Pt(int(x), int(y)), "spell cell", "Enchantment", admitField)
}

// spellTemplateKind is the kind of cell spell the Add window's template
// value v starts: a SpellConstructor's kind or the kind of record v.
func spellTemplateKind(m *mapedit.Editor, v int64) (uint16, bool) {
	for _, kind := range []uint16{0, 4, 8} {
		if v == int64(mapedit.SpellConstructor(kind)) {
			return kind, true
		}
	}
	view, err := m.Map()
	if err != nil || v < 0 || v >= int64(len(view.Enchantments)) {
		return 0, false
	}
	return view.Enchantments[v].A, true
}

// spellCheck is the object reference check of a unit spell or casting
// structure record, and false for a kind that names no object.
func spellCheck(kind uint16) (mapedit.FieldCheck, bool) {
	switch kind {
	case 8:
		return mapedit.CheckScriptUnit, true
	case 4:
		return mapedit.CheckScriptStructure, true
	}
	return 0, false
}

// spellBound reports that a unit spell or casting structure record's id
// names an object, as the game looks it up (ALM-T9BIND-334); a record of
// another kind is bound.
func spellBound(m *alm.Map, e alm.Enchantment) bool {
	check, ok := spellCheck(e.A)
	if !ok || e.X == 0 && e.Y == 0 || m == nil {
		return true
	}
	return scriptReferenceCount(check, m, int64(uint32(e.B)|uint32(e.C)<<16), true) > 0
}

// spellRecordBound is spellBound of record index of m.
func spellRecordBound(m *mapedit.Editor, index int) error {
	view, err := m.Map()
	if err != nil || index < 0 || index >= len(view.Enchantments) {
		return err
	}
	if !spellBound(view, view.Enchantments[index]) {
		return fmt.Errorf("cell spell %d names no object; the game skips it, and only Delete changes it", index)
	}
	return nil
}

// firstSpellObject is the id of the first object a new record of kind can
// name, 0 for none.
func firstSpellObject(m *alm.Map, kind uint16) int64 {
	switch kind {
	case 8:
		if len(m.Units) > 0 {
			return int64(m.Units[0].UnitID)
		}
	case 4:
		if len(m.Objects) > 0 {
			return int64(structureBindingID(m.Objects[0]))
		}
	}
	return 0
}

// structureBindingID is the id a casting structure record names its
// structure by: the structure's id word sign-extended (ALM-T9BIND-334).
func structureBindingID(o alm.Object) uint32 {
	return uint32(int32(int16(o.Field12)))
}

// objectSpells is the index of every cell spell that record index of kind k
// carries with it when it is deleted, in file order: a structure's casting
// records (kind 4) and a unit's spell entries (kind 8) whose id is the
// object's, as the original editor's delete selects them (ALM-EDITDEL-299).
// Item recipes (cell 0,0) and other kinds are not among them.
func objectSpells(m *mapedit.Editor, k mapedit.Kind, index int) ([]int, error) {
	if k != mapedit.KindStructure && k != mapedit.KindUnit {
		return nil, nil
	}
	view, err := m.Map()
	if err != nil {
		return nil, err
	}
	kind, id := uint16(4), uint32(0)
	switch {
	case k == mapedit.KindStructure && index >= 0 && index < len(view.Objects):
		id = structureBindingID(view.Objects[index])
	case k == mapedit.KindUnit && index >= 0 && index < len(view.Units):
		kind, id = 8, uint32(view.Units[index].UnitID)
	default:
		return nil, nil
	}
	var out []int
	for j, e := range view.Enchantments {
		if e.A == kind && (e.X != 0 || e.Y != 0) && uint32(e.B)|uint32(e.C)<<16 == id {
			out = append(out, j)
		}
	}
	return out, nil
}

// deleteObject removes record index of kind k as one history entry. A
// structure or unit takes its cell spells (objectSpells) with it, each
// through the record codec's Delete, which renumbers the later item links
// (ALM-EDITLINK-298), so one Undo restores every byte. A refused part
// leaves m as it was.
func deleteObject(m *mapedit.Editor, k mapedit.Kind, index int) error {
	spells, err := objectSpells(m, k, index)
	if err != nil {
		return err
	}
	mark := m.Mark()
	if err := m.Delete(k, index); err != nil {
		return err
	}
	for j := len(spells) - 1; j >= 0; j-- {
		if err := m.Delete(mapedit.KindEnchantment, spells[j]); err != nil {
			for m.Mark() > mark {
				m.Undo()
			}
			return err
		}
	}
	m.Squash(mark)
	return nil
}

// spellObjectAllowed refuses a unit spell or casting structure record whose
// id names no single object. The game binds the record by its id alone, so
// the cell is not compared (ALM-T9BIND-334).
func spellObjectAllowed(m *mapedit.Editor, kind uint16, link uint32) error {
	check, ok := spellCheck(kind)
	if !ok {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	return checkScriptReference(check, view, int64(link), true)
}
