package mapedit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// The type-8 record (ALM-SACK-065): a head of the element count, the owner
// (0 for a ground sack, else the unit id whose stock it is), X, Y and, from
// format version 989, gold; then that many 10-byte elements, each an item
// code word, a u16 and a 1-based type-9 index.
const (
	lootTypeID      = 8
	lootCountWord   = 0x00
	lootOwner       = 0x04
	lootX           = 0x08
	lootY           = 0x0c
	lootGold        = 0x10
	lootHeadShort   = 16
	lootHeadWide    = 20
	lootElementSize = 10
	lootWideVersion = 989

	fhFormatVersion = 0x10
	metaCount8      = 0x2c
)

func (e *Editor) lootHead() int {
	if e.u32(fhFormatVersion) >= lootWideVersion {
		return lootHeadWide
	}
	return lootHeadShort
}

func lootLength(e *Editor, b []byte) int {
	head := e.lootHead()
	if len(b) < head {
		return -1
	}
	n := uint64(binary.LittleEndian.Uint32(b[lootCountWord:]))
	if n > uint64(len(b)-head)/lootElementSize {
		return -1
	}
	return head + int(n)*lootElementSize
}

// GroundSack builds one type-8 ground record in this map's head width: owner
// 0, the cell words x and y, no gold, and one element holding item with a
// zero second word and no type-9 link.
func (e *Editor) GroundSack(x, y uint32, item uint16) []byte {
	head := e.lootHead()
	rec := make([]byte, head+lootElementSize)
	binary.LittleEndian.PutUint32(rec[lootCountWord:], 1)
	binary.LittleEndian.PutUint32(rec[lootX:], x)
	binary.LittleEndian.PutUint32(rec[lootY:], y)
	binary.LittleEndian.PutUint16(rec[head:], item)
	return rec
}

// lootFields are gold, on a ground record with the wide head, then one item
// row per element and the row that appends an element. An item row set to 0
// removes its element.
func lootFields(e *Editor, rec []byte) []Field {
	var out []Field
	head := e.lootHead()
	if head == lootHeadWide && binary.LittleEndian.Uint32(rec[lootOwner:]) == 0 {
		out = append(out, Field{Key: "gold", Label: "Gold", Offset: lootGold, Width: 4, Min: 0, Max: math.MaxInt32})
	}
	n := (len(rec) - head) / lootElementSize
	for k := 0; k < n; k++ {
		out = append(out, Field{Key: fmt.Sprintf("item%d", k+1), Label: fmt.Sprintf("Item %d code (0 removes)", k+1),
			Offset: head + k*lootElementSize, Width: 2, Min: 0, Max: math.MaxUint16, Check: CheckItemCode,
			Element: lootElementSize, Counter: lootCountWord})
	}
	out = append(out, Field{Key: "additem", Label: "Add item code", Offset: len(rec), Width: 2, Min: 0, Max: math.MaxUint16,
		Check: CheckItemCode, Element: lootElementSize, Counter: lootCountWord})
	out = append(out, Field{Key: "prependitem", Label: "Insert first item code", Offset: head, Width: 2, Min: 1, Max: math.MaxUint16,
		Check: CheckItemCode, Element: lootElementSize, Counter: lootCountWord, Insert: true})
	for k := 0; k < n; k++ {
		out = append(out, Field{Key: fmt.Sprintf("duplicate%d", k+1), Label: fmt.Sprintf("Duplicate item %d", k+1),
			Offset: len(rec), Width: 2, Min: 0, Max: 1, Element: lootElementSize, Counter: lootCountWord, CopyOffset: head + k*lootElementSize})
		out = append(out, Field{Key: fmt.Sprintf("use%d", k+1), Label: fmt.Sprintf("Item %d native dispatch", k+1),
			Offset: head + k*lootElementSize + 4, Width: 2, Min: 0, Max: math.MaxUint16})
		out = append(out, Field{Key: fmt.Sprintf("recipe%d", k+1), Label: fmt.Sprintf("Item %d recipe (0 = plain)", k+1),
			Offset: head + k*lootElementSize + 6, Width: 4, Min: 0, Max: math.MaxUint32, Check: CheckItemRecipe})
	}
	return out
}

func (e *Editor) AddUnitStock(index int) (int, error) {
	stock, err := e.UnitStock(index)
	if err != nil {
		return -1, err
	}
	if stock >= 0 {
		return -1, fmt.Errorf("mapedit: unit %d already has stock %d", index, stock)
	}
	u, err := e.walkIndex(KindUnit, "add stock to", index)
	if err != nil {
		return -1, err
	}
	at, _ := u.at(index)
	id := uint32(binary.LittleEndian.Uint16(e.data[at+unitID:]))
	if e.u32(at+unitLootLink) != 0 {
		return -1, fmt.Errorf("mapedit: unit %d already links a loot record", index)
	}
	w, err := e.placementSection(KindLoot, "add stock")
	if err != nil {
		return -1, err
	}
	rec := make([]byte, e.lootHead())
	binary.LittleEndian.PutUint32(rec[lootOwner:], id)
	i, ed := e.appendRecordParts(w, rec)
	var link [4]byte
	binary.LittleEndian.PutUint32(link[:], uint32(i+1))
	ed = append(ed, replace(at+unitLootLink, e.data[at+unitLootLink:at+unitLootLink+4], link[:]))
	e.apply(ed)
	return i, nil
}

func (e *Editor) UnitStock(index int, allowUnowned ...bool) (int, error) {
	view, err := e.Map()
	if err != nil || index < 0 || index >= len(view.Units) {
		return -1, fmt.Errorf("mapedit: unit %d is unavailable", index)
	}
	id := uint32(view.Units[index].UnitID)
	if id == 0 {
		return -1, fmt.Errorf("mapedit: unit %d has no stock identity", index)
	}
	ambiguous := false
	for i, u := range view.Units {
		if i != index && uint32(u.UnitID) == id {
			if len(allowUnowned) == 0 || !allowUnowned[0] {
				return -1, fmt.Errorf("mapedit: unit identity %d is ambiguous", id)
			}
			ambiguous = true
		}
	}
	loot, err := view.Loot()
	if err != nil {
		return -1, err
	}
	stock := -1
	for i, r := range loot.Records {
		if r.Owner == id {
			if stock >= 0 {
				return -1, fmt.Errorf("mapedit: unit %d owns several stocks", index)
			}
			stock = i
		}
	}
	if ambiguous && stock >= 0 {
		return -1, fmt.Errorf("mapedit: stocked unit identity %d is ambiguous", id)
	}
	rec, err := e.UnitRecord(index)
	if err != nil {
		return -1, err
	}
	if link := binary.LittleEndian.Uint32(rec[unitLootLink:]); stock >= 0 && link != 0 && link != uint32(stock+1) {
		return -1, fmt.Errorf("mapedit: unit %d stock link disagrees with its owned stock", index)
	}
	return stock, nil
}

func refuseOwnedStockDeletion(e *Editor, units walk, index int) ([]part, error) {
	if !e.locate().carries(lootTypeID) {
		return nil, nil
	}
	at, _ := units.at(index)
	id := uint32(binary.LittleEndian.Uint16(e.data[at+unitID:]))
	if id == 0 {
		return nil, nil
	}
	loot, err := e.walkSection(KindLoot, "check stock before deleting a unit")
	if err != nil {
		return nil, nil
	}
	for i := 0; i < loot.n(); i++ {
		p, _ := loot.at(i)
		if e.u32(p+lootOwner) == id {
			return nil, fmt.Errorf("mapedit: unit %d owns Stock #%d; delete that stock first", index, i)
		}
	}
	return nil, nil
}

func cloneUnitStockParts(e *Editor, units walk, src *Editor, sunits walk, source int, rec []byte, _ bool) ([]part, error) {
	binary.LittleEndian.PutUint32(rec[unitLootLink:], 0)
	if !src.locate().carries(lootTypeID) {
		return nil, nil
	}
	loot, err := src.walkSection(KindLoot, "clone unit stock")
	if err != nil {
		return nil, nil
	}
	at, _ := sunits.at(source)
	id := uint32(binary.LittleEndian.Uint16(src.data[at+unitID:]))
	if id == 0 {
		return nil, nil
	}
	stock, err := src.UnitStock(source, true)
	if err != nil {
		return nil, err
	}
	if stock < 0 {
		return nil, nil
	}
	p, size := loot.at(stock)
	copy := cloneBytes(src.data[p : p+size])
	binary.LittleEndian.PutUint32(copy[lootOwner:], uint32(binary.LittleEndian.Uint16(rec[unitID:])))
	if src != e {
		if loot, err = e.placementSection(KindLoot, "clone unit stock"); err != nil {
			return nil, err
		}
		if lootLength(e, copy) != len(copy) {
			return nil, fmt.Errorf("mapedit: cannot clone unit %d: this map's sack records have another layout", source)
		}
	}
	extra, err := cloneStockRecipeParts(e, src, copy, false)
	if err != nil {
		return nil, err
	}
	index, ed := e.appendRecordParts(loot, copy)
	binary.LittleEndian.PutUint32(rec[unitLootLink:], uint32(index)+1)
	return append(ed, extra...), nil
}

// cloneSackRecipeParts copies the recipes a ground sack's items link when the
// sack comes from another map; a clone within one map shares them, and so
// does a clone from a copy of this map where the recipe still stands.
func cloneSackRecipeParts(e *Editor, _ walk, src *Editor, _ walk, _ int, rec []byte, copyOf bool) ([]part, error) {
	if src == e {
		return nil, nil
	}
	return cloneStockRecipeParts(e, src, rec, copyOf)
}

// cloneStockRecipeParts appends to e a copy of each recipe of src that an item
// of rec links, and points the item at its copy. With share, a link whose
// recipe e holds byte for byte at the same index keeps that link.
func cloneStockRecipeParts(e, src *Editor, rec []byte, share bool) ([]part, error) {
	var recipes, from walk
	var data []byte
	count, walked := 0, false
	for p := e.lootHead() + 6; p < len(rec); p += lootElementSize {
		link := binary.LittleEndian.Uint32(rec[p:])
		if link == 0 {
			continue
		}
		if !walked {
			walked = true
			var err error
			recipes, err = e.walkSection(KindEnchantment, "clone stock item recipes")
			if err != nil {
				return nil, err
			}
			from = recipes
			if src != e {
				if from, err = src.walkSection(KindEnchantment, "clone stock item recipes"); err != nil {
					return nil, err
				}
			}
		}
		if uint64(link) > uint64(from.n()) {
			return nil, fmt.Errorf("mapedit: stock item links missing recipe %d", link)
		}
		at, size := from.at(int(link - 1))
		if src.u32(at+4) != 0 || src.u32(at+8) != 0 {
			return nil, fmt.Errorf("mapedit: stock item links coordinate-bearing recipe %d", link)
		}
		if share && src != e && uint64(link) <= uint64(recipes.n()) {
			if here, n := recipes.at(int(link - 1)); n == size && bytes.Equal(e.data[here:here+n], src.data[at:at+size]) {
				continue
			}
		}
		data = append(data, src.data[at:at+size]...)
		count++
		binary.LittleEndian.PutUint32(rec[p:], uint32(recipes.n()+count))
	}
	if count == 0 {
		return nil, nil
	}
	_, ed := e.appendRecordsParts(recipes, data, count)
	return ed, nil
}
