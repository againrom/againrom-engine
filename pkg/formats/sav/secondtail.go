package sav

import (
	"encoding/binary"
	"fmt"
)

// SecondTail is the second game's campaign tail after the state store: the
// pair list, the Scenario record and nine client buffers, tiled to the end
// of the file (R2-SESSION-021, R2-SESSION-238).
type SecondTail struct {
	// Pairs are the pair list's elements; PairTail the two DWORDs after them.
	Pairs    [][2]uint32
	PairTail [2]uint32

	// Bank is the Scenario record's 4,096-byte bank as its 1,024 DWORD slots
	// (R2-SESSION-023); Shops the 320 bytes after it: the assortment records
	// of locations 0..3, four category cells each, row-major with a 0x50-byte
	// row and a 0x14-byte cell (R2-SESSION-156, R2-ENGINE-263).
	Bank  [1024]int32
	Shops [4][4]SecondShopCell

	// Locations are the availability records in file order and Current the
	// list ordinal of the current one (R2-SESSION-155).
	Locations []SecondLocation
	Current   uint32

	// Buffers are the nine client buffers, carried as read.
	Buffers [9]SecondBuffer
}

// SecondLocation is one availability record: ID, kind and 16 raw bytes.
type SecondLocation struct {
	ID, Kind uint32
	Raw      [16]byte
}

// SecondShopCell is one category cell of an assortment record: minimum
// price, maximum price, draws, quantity bound and mask (R2-ENGINE-263).
type SecondShopCell struct {
	Min, Max, Draws, Bound int32
	Mask                   uint32
}

// SecondBuffer is one client buffer: two words and its bytes.
type SecondBuffer struct {
	A, B uint16
	Data []byte
}

const (
	secondBankLen   = 4096
	secondGridLen   = 320
	secondCellBytes = 0x14
	secondNodeLen   = 24
	maxSecondTailN  = 1 << 16
	secondBufferHdr = 8
)

// SecondTail answers a second-game file's campaign tail.
func (f *File) SecondTail() (SecondTail, error) {
	if !f.SecondGame() {
		return SecondTail{}, fmt.Errorf("sav: a first-game save has no second-game tail")
	}
	if f.secondTailErr != nil {
		return SecondTail{}, f.secondTailErr
	}
	if f.secondTail == nil {
		return SecondTail{}, fmt.Errorf("sav: the tail does not frame as a state store")
	}
	out := *f.secondTail
	out.Pairs = append([][2]uint32(nil), out.Pairs...)
	out.Locations = append([]SecondLocation(nil), out.Locations...)
	for i := range out.Buffers {
		out.Buffers[i].Data = append([]byte(nil), out.Buffers[i].Data...)
	}
	return out, nil
}

// CurrentLocation is the record Current names, and false when it names none.
func (t SecondTail) CurrentLocation() (SecondLocation, bool) {
	if uint64(t.Current) >= uint64(len(t.Locations)) {
		return SecondLocation{}, false
	}
	return t.Locations[t.Current], true
}

// parseSecondTail reads b to its end and refuses a tail that does not tile.
func parseSecondTail(b []byte) (*SecondTail, error) {
	r := tailReader{b: b}
	out := &SecondTail{}
	n := r.u32()
	if r.err == nil && (n > maxSecondTailN || uint64(n)*8 > uint64(len(b))) {
		return nil, fmt.Errorf("sav: second-game pair count %d exceeds the tail", n)
	}
	for i := uint32(0); i < n && r.err == nil; i++ {
		out.Pairs = append(out.Pairs, [2]uint32{r.u32(), r.u32()})
	}
	out.PairTail = [2]uint32{r.u32(), r.u32()}
	if bank := r.take(secondBankLen); bank != nil {
		for i := range out.Bank {
			out.Bank[i] = int32(binary.LittleEndian.Uint32(bank[4*i:]))
		}
	}
	if grid := r.take(secondGridLen); grid != nil {
		for i := range 16 {
			c := grid[i*secondCellBytes:]
			out.Shops[i/4][i%4] = SecondShopCell{Min: int32(binary.LittleEndian.Uint32(c)), Max: int32(binary.LittleEndian.Uint32(c[4:])),
				Draws: int32(binary.LittleEndian.Uint32(c[8:])), Bound: int32(binary.LittleEndian.Uint32(c[12:])), Mask: binary.LittleEndian.Uint32(c[16:])}
		}
	}
	n = r.u32()
	if r.err == nil && (n > maxSecondTailN || uint64(n)*secondNodeLen > uint64(len(b))) {
		return nil, fmt.Errorf("sav: second-game location count %d exceeds the tail", n)
	}
	for i := uint32(0); i < n && r.err == nil; i++ {
		l := SecondLocation{ID: r.u32(), Kind: r.u32()}
		copy(l.Raw[:], r.take(16))
		out.Locations = append(out.Locations, l)
	}
	out.Current = r.u32()
	for i := range out.Buffers {
		buf := SecondBuffer{A: r.u16(), B: r.u16()}
		size := r.u32()
		if r.err == nil && uint64(size) > uint64(len(b)) {
			return nil, fmt.Errorf("sav: second-game buffer %d declares %d bytes", i, size)
		}
		buf.Data = append([]byte(nil), r.take(int(size))...)
		out.Buffers[i] = buf
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.p != len(b) {
		return nil, fmt.Errorf("sav: second-game tail ends at %d of %d bytes", r.p, len(b))
	}
	return out, nil
}

// tailReader reads little-endian fields and keeps the first overrun.
type tailReader struct {
	b   []byte
	p   int
	err error
}

func (r *tailReader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.p+n > len(r.b) {
		r.err = fmt.Errorf("sav: second-game tail overruns at %d", r.p)
		return nil
	}
	out := r.b[r.p : r.p+n]
	r.p += n
	return out
}

func (r *tailReader) u16() uint16 {
	if b := r.take(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}

func (r *tailReader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}
