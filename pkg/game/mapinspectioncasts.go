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
}

func (x *MapInspector) inspectionEnchantment(i int, e alm.Enchantment) ui.InspectionRecord {
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
	return r
}

func spellCellAllowed(m *mapedit.Editor, index int) error {
	rec, err := m.Record(mapedit.KindEnchantment, index)
	if err != nil {
		return err
	}
	x, y := binary.LittleEndian.Uint32(rec[4:]), binary.LittleEndian.Uint32(rec[8:])
	if x == 0 && y == 0 {
		return fmt.Errorf("cell spells cannot use the item-recipe cell 0,0")
	}
	return insideMap(m, image.Pt(int(x), int(y)), "spell cell")
}

func spellObjectAllowed(m *mapedit.Editor, template int, cell image.Point, link uint32) error {
	view, err := m.Map()
	if err != nil {
		return err
	}
	if template < 0 || template >= len(view.Enchantments) {
		return fmt.Errorf("no cell spell template %d", template)
	}
	kind := view.Enchantments[template].A
	if kind != 4 && kind != 8 {
		return nil
	}
	check := mapedit.CheckScriptStructure
	if kind == 8 {
		check = mapedit.CheckScriptUnit
	}
	if err := checkScriptReference(check, view, int64(link), cell); err != nil {
		return fmt.Errorf("spell cell %d,%d: %w", cell.X, cell.Y, err)
	}
	return nil
}
