package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
)

const enchantmentHead = 26

func enchantmentDeleteParts(e *Editor, recipes walk, index int) ([]part, error) {
	at, _ := recipes.at(index)
	if e.u32(at+4) != 0 || e.u32(at+8) != 0 {
		return nil, fmt.Errorf("mapedit: coordinate-bearing spell entries cannot be deleted as item recipes")
	}
	if !e.locate().carries(lootTypeID) {
		return nil, nil
	}
	loot, err := e.walkSection(KindLoot, "check item links before deleting a recipe")
	if err != nil {
		return nil, err
	}
	var out []part
	for i := 0; i < loot.n(); i++ {
		at, size := loot.at(i)
		for p := at + e.lootHead() + 6; p < at+size; p += lootElementSize {
			link := e.u32(p)
			if link == uint32(index)+1 {
				return nil, fmt.Errorf("mapedit: recipe %d is still linked by loot record %d; clear its item recipe link first", index+1, i)
			}
			if link > uint32(index)+1 {
				var b [4]byte
				binary.LittleEndian.PutUint32(b[:], link-1)
				out = append(out, replace(p, e.data[p:p+4], b[:]))
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
		return nil
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
