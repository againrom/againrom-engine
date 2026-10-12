package mapedit

import (
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
	return append(out, Field{Key: "additem", Label: "Add item code", Offset: len(rec), Width: 2, Min: 0, Max: math.MaxUint16,
		Check: CheckItemCode, Element: lootElementSize, Counter: lootCountWord})
}
