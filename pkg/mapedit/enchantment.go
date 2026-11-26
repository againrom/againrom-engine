package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
)

const enchantmentHead = 26

// enchantmentSection is the type-9 row of a format whose records take the
// field table fields.
func enchantmentSection(fields func(*Editor, []byte) []Field) *section {
	return &section{name: "enchantment", typeID: alm.TypeTileMarkers, layout: enchantmentLayout, x: 4, y: 8, cellShift: 8,
		length: enchantmentLength, cellless: notCellSpell, checkCell: checkSpellCell, fields: fields,
		deleteParts: enchantmentDeleteParts,
	}
}

func enchantmentDeleteParts(e *Editor, recipes walk, index int) ([]part, error) {
	at, size := recipes.at(index)
	spatial := e.u32(at+4) != 0 || e.u32(at+8) != 0
	if spatial && !spellRecordKnown(e.data[at:at+size]) {
		return nil, fmt.Errorf("mapedit: unknown spell record kind is read-only")
	}
	if !e.locate().carries(alm.TypeLoot) {
		return nil, nil
	}
	loot, err := e.walkSection(KindLoot, "check item links before deleting a recipe")
	if err != nil {
		return nil, err
	}
	var out []part
	last := map[uint32]int{}
	for i := 0; i < loot.n(); i++ {
		at, size := loot.at(i)
		for p := at + e.lootHead() + 6; p < at+size; p += alm.LootElementSize {
			link := e.u32(p)
			if link == uint32(index)+1 && !spatial {
				return nil, fmt.Errorf("mapedit: recipe %d is still linked by loot record %d; clear its item recipe link first", index+1, i)
			}
			if link > uint32(index)+1 && (!spatial || link <= uint32(recipes.n())) {
				var b [4]byte
				binary.LittleEndian.PutUint32(b[:], link-1)
				change := replace(p, e.data[p:p+4], b[:])
				if previous, ok := last[link]; spatial && ok {
					out[previous] = change
				} else {
					last[link] = len(out)
					out = append(out, change)
				}
			}
		}
	}
	return out, nil
}

func enchantmentLength(_ *Editor, b []byte) int {
	if len(b) < enchantmentHead {
		return -1
	}
	n := uint64(binary.LittleEndian.Uint32(b[22:]))
	if n > uint64(len(b)-enchantmentHead)/6 {
		return -1
	}
	return enchantmentHead + int(n)*6
}

func enchantmentLayout(e *Editor, sp span) (arrayLayout, error) {
	if sp.payloadSize < 4 {
		return arrayLayout{}, fmt.Errorf("type-9 payload has no count")
	}
	n := uint64(e.u32(sp.payloadOff))
	if n > uint64(sp.payloadSize-4)/enchantmentHead {
		return arrayLayout{}, fmt.Errorf("type-9 record count exceeds its payload")
	}
	return arrayLayout{countAt: sp.payloadOff, first: 4, n: int(n)}, nil
}

func enchantmentFields(e *Editor, rec []byte) []Field {
	if binary.LittleEndian.Uint32(rec[4:]) != 0 || binary.LittleEndian.Uint32(rec[8:]) != 0 {
		if !spellRecordKnown(rec) {
			return nil
		}
		// A casting structure's power and scan range reach the game as their
		// low bytes (ALM-T9CAST-333), so both stop at 255 (DIV-4046).
		power, label := int64(100), "editor.cast.power"
		if binary.LittleEndian.Uint16(rec[12:]) == 4 {
			power, label = math.MaxUint8, "editor.cast.castpower"
		}
		out := []Field{
			{Key: "spell", Label: "editor.cast.spell", Offset: 18, Width: 2, Min: 0, Max: 28},
			{Key: "power", Label: label, Offset: 20, Width: 2, Min: 0, Max: power},
		}
		if binary.LittleEndian.Uint16(rec[12:]) == 4 {
			out = append(out, Field{Key: "range", Label: "editor.cast.range", Offset: 0, Width: 4, Min: 0, Max: math.MaxUint8})
		}
		if IsTrapKind(binary.LittleEndian.Uint16(rec[12:])) {
			cell := []Field{
				{Key: "link", Label: "editor.cast.link", Offset: 14, Width: 4, Min: 0, Max: math.MaxUint32},
				{Cell: CellTriggerX, Key: "triggerx", Label: "editor.cast.x", Offset: 4, Width: 4, Min: 0, Max: 255},
				{Cell: CellTriggerY, Key: "triggery", Label: "editor.cast.y", Offset: 8, Width: 4, Min: 0, Max: 255},
				{Key: "castkind", Label: "editor.cast.kind", Offset: 12, Width: 2, Min: 0, Max: 3},
				{Cell: CellSourceX, Key: "sourcex", Label: "editor.cast.sourcex", Offset: 26, Width: 2, Min: 0, Max: 255},
				{Cell: CellSourceY, Key: "sourcey", Label: "editor.cast.sourcey", Offset: 28, Width: 2, Min: 0, Max: 255},
				{Cell: CellTargetX, Key: "targetx", Label: "editor.cast.targetx", Offset: 32, Width: 2, Min: 0, Max: 255},
				{Cell: CellTargetY, Key: "targety", Label: "editor.cast.targety", Offset: 34, Width: 2, Min: 0, Max: 255},
			}
			if move := scriptprofile.VocabularyOf(e.Profile().Ops).CellMove; move != 0 && uint32(binary.LittleEndian.Uint16(rec[18:])) == move {
				cell = cellMoveFields(cell)
			}
			out = append(out, cell...)
		}
		return out
	}
	out := []Field{
		{Key: "head", Label: "Head A (effect = low8(A+43))", Offset: 12, Width: 2, Min: 0, Max: math.MaxUint16},
		{Key: "base", Label: "Head B (low byte = base)", Offset: 14, Width: 2, Min: 0, Max: math.MaxUint16},
		{Key: "spread", Label: "Head C (low byte = spread)", Offset: 16, Width: 2, Min: 0, Max: math.MaxUint16},
		{Key: "spell", Label: "Spell id (0 = none)", Offset: 18, Width: 2, Min: 0, Max: math.MaxInt16},
		{Key: "power", Label: "Spell power", Offset: 20, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16},
	}
	n := (len(rec) - enchantmentHead) / 6
	for i := 0; i < n; i++ {
		p := enchantmentHead + 6*i
		out = append(out,
			Field{Key: fmt.Sprintf("effect%d", i+1), Label: fmt.Sprintf("Effect %d kind (0 removes)", i+1), Offset: p, Width: 2, Min: 0, Max: 255, Element: 6, Counter: 22},
			Field{Key: fmt.Sprintf("low%d", i+1), Label: fmt.Sprintf("Effect %d low word", i+1), Offset: p + 2, Width: 2, Min: 0, Max: math.MaxUint16},
			Field{Key: fmt.Sprintf("high%d", i+1), Label: fmt.Sprintf("Effect %d high word", i+1), Offset: p + 4, Width: 2, Min: 0, Max: math.MaxUint16})
	}
	return append(out, Field{Key: "addeffect", Label: "Add effect kind (1..255)", Offset: len(rec), Width: 2, Min: 1, Max: 255, Element: 6, Counter: 22})
}

func spellRecordKnown(rec []byte) bool {
	k := binary.LittleEndian.Uint16(rec[12:])
	return k == 4 || k == 8 || IsTrapKind(k) && len(rec) >= enchantmentHead+12
}

func notCellSpell(rec []byte) bool {
	return !spellRecordKnown(rec) || binary.LittleEndian.Uint32(rec[4:]) == 0 && binary.LittleEndian.Uint32(rec[8:]) == 0
}

func checkSpellCell(x, y uint32) error {
	if x > 255 || y > 255 || x == 0 && y == 0 {
		return fmt.Errorf("mapedit: spell cell is outside the nonzero byte-coordinate map")
	}
	return nil
}

// SpellConstructor is the AddSpellFrom template that names kind's new
// record instead of a record of the map: every byte 0 but the kind, and a
// trap's two elements (ALM-EDITT9-328).
func SpellConstructor(kind uint16) int { return -int(kind) - 1 }

// spellTemplate is the record AddSpellFrom starts from and whether the
// constructor built it.
func (e *Editor) spellTemplate(template int) ([]byte, bool, error) {
	if template >= 0 {
		rec, err := e.Record(KindEnchantment, template)
		return rec, false, err
	}
	kind := -template - 1
	if kind != 4 && kind != 8 && !IsTrapKind(uint16(kind)) {
		return nil, false, fmt.Errorf("mapedit: no cell spell kind %d", kind)
	}
	rec := make([]byte, enchantmentHead)
	binary.LittleEndian.PutUint16(rec[12:], uint16(kind))
	if IsTrapKind(uint16(kind)) {
		rec = append(rec, make([]byte, 12)...)
		binary.LittleEndian.PutUint32(rec[22:], 2)
	}
	return rec, true, nil
}

// SpellTemplateFields describes the fields of a spell constructor or copy
// once the values sets names are written, as a new record's steps take them.
func (e *Editor) SpellTemplateFields(template int, sets ...FieldSet) []Field {
	rec, fresh, err := e.spellTemplate(template)
	if err != nil || !fresh && notCellSpell(rec) {
		return nil
	}
	if fresh {
		// Give the constructor a cell for the field table's spell check.
		rec[4] = 1
	}
	fields := e.rowOf(KindEnchantment).fields
	for _, set := range sets {
		if f, ok := fieldByKey(fields(e, rec), set.Key); ok && set.Value >= f.Min && set.Value <= f.Max && f.Element == 0 {
			rec = f.write(rec, set.Value)
		}
	}
	return fields(e, rec)
}

// AddSpellFrom adds a cell spell at x, y bound to link: a copy of record
// template, or a new record when template is a SpellConstructor. A new
// trap's source and target start at its cell; word 2 of each element is 0
// (DIV-4045), a copy keeps the template's.
func (e *Editor) AddSpellFrom(template int, x, y, link uint32) (int, error) {
	rec, fresh, err := e.spellTemplate(template)
	if err != nil {
		return -1, err
	}
	if !spellRecordKnown(rec) || !fresh && notCellSpell(rec) {
		return -1, fmt.Errorf("mapedit: select a known cell spell template")
	}
	view, err := e.Map()
	if err != nil {
		return -1, err
	}
	if err := checkSpellCell(x, y); err != nil {
		return -1, err
	}
	if x >= uint32(view.Width) || y >= uint32(view.Height) {
		return -1, fmt.Errorf("mapedit: spell cell is outside the nonzero byte-coordinate map")
	}
	if binary.LittleEndian.Uint16(rec[12:]) == 4 {
		for i, existing := range view.Enchantments {
			if existing.A == 4 && existing.X == x && existing.Y == y && uint32(existing.B)|uint32(existing.C)<<16 == link {
				return i, nil
			}
		}
	}
	binary.LittleEndian.PutUint32(rec[4:], x)
	binary.LittleEndian.PutUint32(rec[8:], y)
	if fresh && IsTrapKind(binary.LittleEndian.Uint16(rec[12:])) {
		for _, p := range []int{enchantmentHead, enchantmentHead + 6} {
			binary.LittleEndian.PutUint16(rec[p:], uint16(x))
			binary.LittleEndian.PutUint16(rec[p+2:], uint16(y))
		}
	}
	if IsTrapKind(binary.LittleEndian.Uint16(rec[12:])) {
		binary.LittleEndian.PutUint32(rec[14:], 0)
	} else {
		binary.LittleEndian.PutUint32(rec[14:], link)
	}
	mark := e.Mark()
	i, err := e.Place(KindEnchantment, rec)
	if err == nil && IsTrapKind(binary.LittleEndian.Uint16(rec[12:])) && link != 0 {
		err = e.SetField(KindEnchantment, i, "link", int64(link))
		e.Squash(mark)
	}
	return i, err
}

func (e *Editor) AddItemRecipe(lootIndex, element int, template ...int) (int, error) {
	if len(template) > 1 {
		return -1, fmt.Errorf("mapedit: item recipe has more than one template")
	}
	w, err := e.walkIndex(KindLoot, "enchant an item of", lootIndex)
	if err != nil {
		return -1, err
	}
	at, size := w.at(lootIndex)
	head := e.lootHead()
	if element < 1 || element > (size-head)/alm.LootElementSize {
		return -1, fmt.Errorf("mapedit: item number %d is outside 1..%d", element, (size-head)/alm.LootElementSize)
	}
	p := at + head + (element-1)*alm.LootElementSize + 6
	if old := e.u32(p); old != 0 && len(template) == 0 {
		return -1, fmt.Errorf("mapedit: item %d already links recipe %d; select Enchantment #%d and use E", element, old, old-1)
	}
	recipes, err := e.walkSection(KindEnchantment, "add an item recipe")
	if err != nil {
		return -1, err
	}
	record := make([]byte, enchantmentHead)
	if len(template) == 1 && template[0] >= 0 {
		record, err = e.Record(KindEnchantment, template[0])
		if err != nil {
			return -1, err
		}
		if binary.LittleEndian.Uint32(record[4:]) != 0 || binary.LittleEndian.Uint32(record[8:]) != 0 {
			return -1, fmt.Errorf("mapedit: item recipe template has cell coordinates")
		}
	}
	i, ed := e.appendRecordParts(recipes, record)
	var link [4]byte
	binary.LittleEndian.PutUint32(link[:], uint32(i+1))
	ed = append(ed, replace(p, e.data[p:p+4], link[:]))
	e.apply(ed)
	return i, nil
}
