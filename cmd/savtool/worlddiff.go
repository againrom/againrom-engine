package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
)

// worldPart is a byte group of a second-game terrain entry or session
// block, named by the part R2-SESSION-311 and R2-SESSION-019 give it.
type worldPart struct {
	name   string
	lo, hi int
}

// secondEntryParts are a terrain entry's byte groups, offsets from the
// entry's first byte (its u16 key, then the 60 raw bytes).
var secondEntryParts = []worldPart{
	{"+0x00 planes", 2, 4}, {"+0x02 layers", 4, 5}, {"+0x03", 5, 6},
	{"+0x04 occupant", 6, 10}, {"+0x08 occupant3", 10, 14}, {"+0x0c building", 14, 18},
	{"+0x10 object", 18, 22}, {"+0x14 effects", 22, 46}, {"+0x2c footprint", 46, 50},
	{"+0x30 footprint3", 50, 54}, {"+0x34 event", 54, 60}, {"+0x3a key", 60, 62},
}

// secondSessionParts are the session block's parts in their transfer order
// (R2-SESSION-019): offsets from the block's first byte.
var secondSessionParts = []worldPart{
	{"latches", 4000, 5000}, {"timer", 5000, 5048}, {"+0xa728", 5048, 5448},
	{"diplomacy head", 5448, 5456}, {"+0xa48", 10356, 10357}, {"+0xa49", 10357, 10358},
	{"+0xa4c", 10358, 10362}, {"+0xbe0c", 10362, 10366}, {"+0xbe10", 10366, 10370}, {"+0xbe14", 10370, 10374},
}

// worldDiffs names each part of two second-game world halves that differs:
// the key list, the terrain entries by key and byte group, their order, the
// terrain key and the session block by part. A first-game pair answers one
// line per structure.
func worldDiffs(fx, fy *sav.File) []string {
	wx, wy := fx.World, fy.World
	if wx == nil || wy == nil {
		if (wx == nil) != (wy == nil) {
			return []string{"world half: present in one file only"}
		}
		return nil
	}
	var out []string
	ex, ey := worldEntries(fx), worldEntries(fy)
	kx, ky := entryKeys(ex), entryKeys(ey)
	if !slices.Equal(wx.Blocks, wy.Blocks) {
		out = append(out, fmt.Sprintf("world key list: %d > %d keys, %d listed differently, %d at an entry cell",
			len(wx.Blocks), len(wy.Blocks), keyListDiffers(wx.Blocks, wy.Blocks), keyListAtEntries(wx.Blocks, wy.Blocks, kx)))
	}
	if len(ex) != len(ey) {
		out = append(out, fmt.Sprintf("world entries: %d > %d", len(ex), len(ey)))
	}
	if !slices.Equal(kx, ky) {
		out = append(out, fmt.Sprintf("world entry order: differs (%d keys in one file only)", keysInOne(kx, ky)))
	}
	if !secondLayout(fx) || !secondLayout(fy) {
		if !slices.EqualFunc(ex, ey, bytes.Equal) {
			out = append(out, "world entries: bytes differ")
		}
	} else {
		byKey := map[uint16][]byte{}
		for _, e := range ey {
			byKey[binary.LittleEndian.Uint16(e)] = e
		}
		for _, part := range secondEntryParts {
			n := 0
			for _, e := range ex {
				if f, ok := byKey[binary.LittleEndian.Uint16(e)]; ok && !bytes.Equal(e[part.lo:part.hi], f[part.lo:part.hi]) {
					n++
				}
			}
			if n != 0 {
				out = append(out, fmt.Sprintf("world entry %s: %d of %d entries differ", part.name, n, len(ex)))
			}
		}
	}
	tx, sx := worldTail(fx)
	ty, sy := worldTail(fy)
	if tx != ty {
		out = append(out, fmt.Sprintf("world terrain key: %#x > %#x", tx, ty))
	}
	if !secondLayout(fx) || !secondLayout(fy) || len(sx) != 10374 || len(sy) != 10374 {
		if !bytes.Equal(sx, sy) {
			out = append(out, "world session: differs")
		}
		return out
	}
	var regs []int
	for i := range 1000 {
		if !bytes.Equal(sx[4*i:4*i+4], sy[4*i:4*i+4]) {
			regs = append(regs, i)
		}
	}
	if len(regs) != 0 {
		out = append(out, fmt.Sprintf("world session registers: %d differ %v", len(regs), regs[:min(len(regs), 8)]))
	}
	for _, part := range secondSessionParts {
		if !bytes.Equal(sx[part.lo:part.hi], sy[part.lo:part.hi]) {
			out = append(out, fmt.Sprintf("world session %s: %s", part.name, equalText(sx[part.lo:part.hi], sy[part.lo:part.hi])))
		}
	}
	matrix := 0
	for i := 5456; i < 10356; i++ {
		if sx[i] != sy[i] {
			matrix++
		}
	}
	if matrix != 0 {
		out = append(out, fmt.Sprintf("world session matrix: %d of 4900 cells differ", matrix))
	}
	return out
}

func keyListDiffers(x, y []sav.BlockRecord) int {
	in := map[sav.BlockRecord]bool{}
	for _, r := range x {
		in[r] = true
	}
	n := 0
	for _, r := range y {
		if !in[r] {
			n++
		} else {
			delete(in, r)
		}
	}
	return n + len(in)
}

// keyListAtEntries is how many keys of x that y lists differently lie at
// one of x's entry cells.
func keyListAtEntries(x, y []sav.BlockRecord, entries []uint16) int {
	in := map[sav.BlockRecord]bool{}
	for _, r := range y {
		in[r] = true
	}
	n := 0
	for _, r := range x {
		if !in[r] && slices.Contains(entries, r.Cell) {
			n++
		}
	}
	return n
}

func worldEntries(f *sav.File) [][]byte {
	var out [][]byte
	for i := range f.World.CellRecCount {
		e, err := f.CellRecord(i)
		if err != nil {
			return out
		}
		out = append(out, e)
	}
	return out
}

func entryKeys(entries [][]byte) []uint16 {
	out := make([]uint16, len(entries))
	for i, e := range entries {
		out[i] = binary.LittleEndian.Uint16(e)
	}
	return out
}

func keysInOne(x, y []uint16) int {
	n := 0
	for _, k := range x {
		if !slices.Contains(y, k) {
			n++
		}
	}
	for _, k := range y {
		if !slices.Contains(x, k) {
			n++
		}
	}
	return n
}

// worldTail is the terrain key after the entries and the session block.
func worldTail(f *sav.File) (uint32, []byte) {
	w := f.World
	if w.SessionOff < 4 || w.SessionOff > len(f.Body) {
		return 0, nil
	}
	key := binary.LittleEndian.Uint32(f.Body[w.SessionOff-4:])
	return key, f.Body[w.SessionOff:min(len(f.Body), w.SessionOff+sessionExtent(f))]
}

func sessionExtent(f *sav.File) int {
	if secondLayout(f) {
		return 10374
	}
	return 4374
}

// secondLayout is that the file's terrain entries are 62 bytes, the second
// game's u16 key and 60 raw bytes; the parts above are that layout's.
func secondLayout(f *sav.File) bool {
	w := f.World
	if w == nil || w.CellRecCount == 0 {
		return false
	}
	return (w.SessionOff-4-w.CellRecDataOff)/w.CellRecCount == 62
}
