package sav

import (
	"encoding/binary"
	"fmt"
)

// cellRecLen is the stride of the cell-record table, and sessionLen the extent
// of the session block. Both are read off the original's own serializers.
const (
	cellRecLen = 54
	sessionLen = 4374
)

// Session block offsets, from its first byte.
const (
	sessTriggerResults = 0    // 100 signed dwords
	sessTriggerLatches = 400  // 1000 fire-once bytes
	sessRawHead        = 1400 // 48 opaque bytes, session+0x08 (SAV-SESS-031)
	sessRawMid         = 1448 // 400 opaque bytes, session+0xa828 (SAV-SESS-031)
	sessDiplomacy      = 1856 // the 50x50 matrix: 1848 + 8
	sessWon            = 4362
	sessLose           = 4370

	triggerResultCount = 100
	triggerLatchCount  = 1000
	diplomacySide      = 50

	// rawHeadLen and rawMidLen are the two regions' own widths. Research has
	// promoted no meaning for either (SAV-SESS-031); this package carries
	// them byte for byte and interprets neither.
	rawHeadLen = 48
	rawMidLen  = 400
)

// BlockRecord is one cell of the block-plane DELTA. It is a delta over the map's
// own terrain and not a plane: every cell whose only block bits are the map's
// own is omitted, so applying these to a zero plane yields a map with no terrain
// blocking at all.
type BlockRecord struct {
	// Cell is the 256-stride plane index, packed (row << 8) | col.
	Cell uint16

	// Dyn is the runtime byte and Static the map-derived one. Bits 0..3 are
	// block bits, 4 and 5 runtime flags, 6 and 7 occupancy — and occupancy
	// appears in Dyn only.
	Dyn, Static uint8
}

// Row and Col unpack the record's cell.
func (r BlockRecord) Row() int { return int(r.Cell >> 8) }
func (r BlockRecord) Col() int { return int(r.Cell & 0xff) }

// WorldHalf is the part of a save a mid-mission file carries and a
// between-mission file does not.
type WorldHalf struct {
	// BuildingsOff/End delimit the counted MFC root list, including its count
	// and reference tags. These are byte locators, not decoded Building fields.
	BuildingsOff, BuildingsEnd int

	// Blocks is the block-plane delta, in the file's own order, which is
	// strictly increasing by cell.
	Blocks []BlockRecord

	// BlocksOff is the body offset of the array's count word; CellRecOff and
	// SessionOff are the same for the two structures after it.
	BlocksOff, CellRecOff, SessionOff int

	// Payload offsets follow the actual short or extended count encoding.
	BlocksDataOff, CellRecDataOff int

	// CellRecCount is how many 54-byte records the table holds. CellTriggers
	// projects six bytes; the complete 52-byte payload is carried verbatim.
	CellRecCount int
}

// worldHalf reads the terrain and session at the envelope's current cursor.
// Counts carry their actual payload starts: WriteCount may use two or six bytes.
func (w *walker) worldHalf() (*WorldHalf, error) {
	out := &WorldHalf{BlocksOff: w.p}
	n, err := w.count()
	if err != nil {
		return nil, err
	}
	out.BlocksDataOff = w.p
	if err := w.skip("terrain blocks", n*4); err != nil {
		return nil, err
	}
	out.Blocks = make([]BlockRecord, n)
	for i := range out.Blocks {
		v := u32(w.b, out.BlocksDataOff+4*i)
		out.Blocks[i] = BlockRecord{Cell: uint16(v >> 16), Dyn: uint8(v >> 8), Static: uint8(v)}
	}
	out.CellRecOff = w.p
	out.CellRecCount, err = w.count()
	if err != nil {
		return nil, err
	}
	out.CellRecDataOff = w.p
	if err := w.skip("terrain cells", out.CellRecCount*cellRecLen); err != nil {
		return nil, err
	}
	if err := w.skip("terrain identity", 4); err != nil {
		return nil, err
	}
	out.SessionOff = w.p
	if err := w.skip("session", sessionLen); err != nil {
		return nil, err
	}
	return out, nil
}

// CellRecord answers the i-th 54-byte record as a view into the body. Use
// CellTriggers for the supported six-byte projection of its 52-byte payload.
func (f *File) CellRecord(i int) ([]byte, error) {
	w := f.World
	if w == nil {
		return nil, fmt.Errorf("sav: this save has no world half")
	}
	if i < 0 || i >= w.CellRecCount {
		return nil, fmt.Errorf("sav: cell record %d of %d", i, w.CellRecCount)
	}
	off := w.CellRecDataOff + cellRecLen*i
	return f.Body[off : off+cellRecLen], nil
}

// SetBlockRecord writes the two bytes of one block-plane delta record. The cell
// key is not editable: the array is strictly increasing by construction and a
// caller that could rewrite a key could break that invariant silently.
func (f *File) SetBlockRecord(i int, dyn, static uint8) error {
	w := f.World
	if w == nil {
		return fmt.Errorf("sav: this save has no world half")
	}
	if i < 0 || i >= len(w.Blocks) {
		return fmt.Errorf("sav: block record %d of %d", i, len(w.Blocks))
	}
	off := w.BlocksDataOff + 4*i
	f.Body[off] = static
	f.Body[off+1] = dyn
	w.Blocks[i].Dyn, w.Blocks[i].Static = dyn, static
	return nil
}

// TriggerResult reads one of the hundred signed trigger result slots, and
// SetTriggerResult writes one.
func (f *File) TriggerResult(i int) (int32, error) {
	off, err := f.sessionAt(sessTriggerResults+4*i, i, triggerResultCount)
	if err != nil {
		return 0, err
	}
	return int32(u32(f.Body, off)), nil
}

func (f *File) SetTriggerResult(i int, v int32) error {
	off, err := f.sessionAt(sessTriggerResults+4*i, i, triggerResultCount)
	if err != nil {
		return err
	}
	put32(f.Body, off, uint32(v))
	return nil
}

// RawHead reads the 48-byte region at session+0x08 (block offset 1400), and
// SetRawHead writes it. SAV-SESS-031 locates and sizes this region; it has no
// promoted meaning, so it is carried opaquely — decoded and re-encoded
// byte for byte, never inspected or validated.
func (f *File) RawHead() ([rawHeadLen]byte, error) {
	var out [rawHeadLen]byte
	off, err := f.sessionRange(sessRawHead, rawHeadLen)
	if err != nil {
		return out, err
	}
	copy(out[:], f.Body[off:off+rawHeadLen])
	return out, nil
}

func (f *File) SetRawHead(v [rawHeadLen]byte) error {
	off, err := f.sessionRange(sessRawHead, rawHeadLen)
	if err != nil {
		return err
	}
	copy(f.Body[off:off+rawHeadLen], v[:])
	return nil
}

// RawMid reads the 400-byte region at session+0xa828 (block offset 1448), and
// SetRawMid writes it. Same opaque treatment as RawHead, above.
func (f *File) RawMid() ([rawMidLen]byte, error) {
	var out [rawMidLen]byte
	off, err := f.sessionRange(sessRawMid, rawMidLen)
	if err != nil {
		return out, err
	}
	copy(out[:], f.Body[off:off+rawMidLen])
	return out, nil
}

func (f *File) SetRawMid(v [rawMidLen]byte) error {
	off, err := f.sessionRange(sessRawMid, rawMidLen)
	if err != nil {
		return err
	}
	copy(f.Body[off:off+rawMidLen], v[:])
	return nil
}

// TriggerLatch reads one of the thousand fire-once latches, and SetTriggerLatch
// writes one. A latch is the record that a trigger has already fired, so
// clearing one is how a save is made to run an arm again.
func (f *File) TriggerLatch(i int) (uint8, error) {
	off, err := f.sessionAt(sessTriggerLatches+i, i, triggerLatchCount)
	if err != nil {
		return 0, err
	}
	return f.Body[off], nil
}

func (f *File) SetTriggerLatch(i int, v uint8) error {
	off, err := f.sessionAt(sessTriggerLatches+i, i, triggerLatchCount)
	if err != nil {
		return err
	}
	f.Body[off] = v
	return nil
}

// Diplomacy reads one entry of the 50x50 matrix, and SetDiplomacy writes one.
// The matrix is not symmetric in the original — a side may regard another
// differently from how it is regarded — so both indices are the caller's.
func (f *File) Diplomacy(a, b int) (uint8, error) {
	off, err := f.diplomacyAt(a, b)
	if err != nil {
		return 0, err
	}
	return f.Body[off], nil
}

func (f *File) SetDiplomacy(a, b int, v uint8) error {
	off, err := f.diplomacyAt(a, b)
	if err != nil {
		return err
	}
	f.Body[off] = v
	return nil
}

func (f *File) diplomacyAt(a, b int) (int, error) {
	if a < 0 || a >= diplomacySide || b < 0 || b >= diplomacySide {
		return 0, fmt.Errorf("sav: diplomacy (%d,%d) outside a %dx%d matrix", a, b, diplomacySide, diplomacySide)
	}
	return f.sessionAt(sessDiplomacy+a*diplomacySide+b, 0, 1)
}

// Counters reads the session's win and lose counters.
//
// NEITHER IS THE MISSION-OUTCOME FLAG. They live in the world half and are gone
// the moment a mission ends, which is exactly when a campaign needs to know it
// was won; the flag is the Player record's own latch.
func (f *File) Counters() (won, lost uint32, err error) {
	w, e := f.sessionAt(sessWon, 0, 1)
	if e != nil {
		return 0, 0, e
	}
	l, e := f.sessionAt(sessLose, 0, 1)
	if e != nil {
		return 0, 0, e
	}
	return u32(f.Body, w), u32(f.Body, l), nil
}

// SetCounters writes them.
func (f *File) SetCounters(won, lost uint32) error {
	w, err := f.sessionAt(sessWon, 0, 1)
	if err != nil {
		return err
	}
	l, err := f.sessionAt(sessLose, 0, 1)
	if err != nil {
		return err
	}
	put32(f.Body, w, won)
	put32(f.Body, l, lost)
	return nil
}

// sessionAt turns an offset inside the session block into a body offset, having
// checked both that there is a session block and that the caller's index is in
// range.
func (f *File) sessionAt(delta, i, n int) (int, error) {
	if f.World == nil {
		return 0, fmt.Errorf("sav: this save has no world half and therefore no session block")
	}
	if i < 0 || i >= n {
		return 0, fmt.Errorf("sav: index %d of %d", i, n)
	}
	if delta < 0 || delta+4 > sessionLen {
		return 0, fmt.Errorf("sav: session offset %d outside the %d-byte block", delta, sessionLen)
	}
	return f.World.SessionOff + delta, nil
}

// sessionRange is sessionAt for a fixed-width SPAN rather than one indexed
// element: RawHead and RawMid have no subscript to bound, only their own
// declared width against the block's.
func (f *File) sessionRange(delta, n int) (int, error) {
	if f.World == nil {
		return 0, fmt.Errorf("sav: this save has no world half and therefore no session block")
	}
	if delta < 0 || n < 0 || delta+n > sessionLen {
		return 0, fmt.Errorf("sav: session span [%d,%d) outside the %d-byte block", delta, delta+n, sessionLen)
	}
	return f.World.SessionOff + delta, nil
}

func put16(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
func put32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }
