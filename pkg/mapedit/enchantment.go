package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
)

const enchantmentHead = 26

func enchantmentDeleteParts(e *Editor, recipes walk, index int) ([]part, error) {
	at, size := recipes.at(index)
	spatial := e.u32(at+4) != 0 || e.u32(at+8) != 0
	if spatial && !spellRecordKnown(e.data[at:at+size]) {
		return nil, fmt.Errorf("mapedit: unknown spell record kind is read-only")
	}
	if !e.locate().carries(lootTypeID) {
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
		for p := at + e.lootHead() + 6; p < at+size; p += lootElementSize {
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

func enchantmentFields(_ *Editor, rec []byte) []Field {
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
		if binary.LittleEndian.Uint16(rec[12:]) < 4 {
			out = append(out,
				Field{Key: "link", Label: "editor.cast.link", Offset: 14, Width: 4, Min: 0, Max: math.MaxUint32},
				Field{Key: "triggerx", Label: "editor.cast.x", Offset: 4, Width: 4, Min: 0, Max: 255},
				Field{Key: "triggery", Label: "editor.cast.y", Offset: 8, Width: 4, Min: 0, Max: 255},
				Field{Key: "castkind", Label: "editor.cast.kind", Offset: 12, Width: 2, Min: 0, Max: 3},
				Field{Key: "sourcex", Label: "editor.cast.sourcex", Offset: 26, Width: 2, Min: 0, Max: 255},
				Field{Key: "sourcey", Label: "editor.cast.sourcey", Offset: 28, Width: 2, Min: 0, Max: 255},
				Field{Key: "targetx", Label: "editor.cast.targetx", Offset: 32, Width: 2, Min: 0, Max: 255},
				Field{Key: "targety", Label: "editor.cast.targety", Offset: 34, Width: 2, Min: 0, Max: 255})
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
	return k == 4 || k == 8 || k < 4 && len(rec) >= enchantmentHead+12
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
	if kind != 4 && kind != 8 && kind > 3 {
		return nil, false, fmt.Errorf("mapedit: no cell spell kind %d", kind)
	}
	rec := make([]byte, enchantmentHead)
	binary.LittleEndian.PutUint16(rec[12:], uint16(kind))
	if kind < 4 {
		rec = append(rec, make([]byte, 12)...)
		binary.LittleEndian.PutUint32(rec[22:], 2)
	}
	return rec, true, nil
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
	if !spellRecordKnown(rec) || !fresh && binary.LittleEndian.Uint32(rec[4:]) == 0 && binary.LittleEndian.Uint32(rec[8:]) == 0 {
		return -1, fmt.Errorf("mapedit: select a known cell spell template")
	}
	view, err := e.Map()
	if err != nil {
		return -1, err
	}
	if x > 255 || y > 255 || x >= uint32(view.Width) || y >= uint32(view.Height) || x == 0 && y == 0 {
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
	if fresh && binary.LittleEndian.Uint16(rec[12:]) < 4 {
		for _, p := range []int{enchantmentHead, enchantmentHead + 6} {
			binary.LittleEndian.PutUint16(rec[p:], uint16(x))
			binary.LittleEndian.PutUint16(rec[p+2:], uint16(y))
		}
	}
	if binary.LittleEndian.Uint16(rec[12:]) < 4 {
		binary.LittleEndian.PutUint32(rec[14:], 0)
	} else {
		binary.LittleEndian.PutUint32(rec[14:], link)
	}
	mark := e.Mark()
	i, err := e.Place(KindEnchantment, rec)
	if err == nil && binary.LittleEndian.Uint16(rec[12:]) < 4 && link != 0 {
		err = e.SetField(KindEnchantment, i, "link", int64(link))
		e.Squash(mark)
	}
	return i, err
}

func (e *Editor) AddItemRecipe(lootIndex, element int) (int, error) {
	w, err := e.walkIndex(KindLoot, "enchant an item of", lootIndex)
	if err != nil {
		return -1, err
	}
	at, size := w.at(lootIndex)
	head := e.lootHead()
	if element < 1 || element > (size-head)/lootElementSize {
		return -1, fmt.Errorf("mapedit: item number %d is outside 1..%d", element, (size-head)/lootElementSize)
	}
	p := at + head + (element-1)*lootElementSize + 6
	if old := e.u32(p); old != 0 {
		return -1, fmt.Errorf("mapedit: item %d already links recipe %d; select Enchantment #%d and use E", element, old, old-1)
	}
	recipes, err := e.walkSection(KindEnchantment, "add an item recipe")
	if err != nil {
		return -1, err
	}
	i, ed := e.appendRecordParts(recipes, make([]byte, enchantmentHead))
	var link [4]byte
	binary.LittleEndian.PutUint32(link[:], uint32(i+1))
	ed = append(ed, replace(p, e.data[p:p+4], link[:]))
	e.apply(ed)
	return i, nil
}
