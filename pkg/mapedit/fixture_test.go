package mapedit_test

// The measuring apparatus for the edit model (docs/0025-mapedit-model): a rich
// synthetic map emitted in both record orders, a frame walk written from the
// container contract, and the carried-bytes comparator. Nothing in this file
// calls pkg/mapedit — a witness that asked the model where a region is would
// agree with it by construction, which is the one failure this apparatus
// exists to avoid. Every fixture is assembled in code from the documented byte
// layout; no test reads a file from disk, and code-page text is written as
// bytes with the code points it must decode to declared beside them.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
)

// ---------------------------------------------------------------------------
// The container layout (docs/0003-alm-container): a 20-byte file header, then
// ten records, each a 20-byte header followed by its pure payload, tiling the
// file exactly.
// ---------------------------------------------------------------------------

const (
	fileMagic        = 0x0052374d // "M7R\x00" as a little-endian u32
	fileHeaderSize   = 20
	recordHeaderSize = 20
	recordCount      = 10
	formatVersion990 = 990

	fhMagic         = 0x00
	fhHdrLen        = 0x04
	fhDataSize      = 0x08
	fhRecordCount   = 0x0c
	fhFormatVersion = 0x10

	recTag         = 0x00
	recHdrLen      = 0x04
	recPayloadSize = 0x08
	recTypeID      = 0x0c
	recPerMapConst = 0x10

	recordTag    = 7
	recordHdrLen = 20
)

// type-0 payload layout, payload-relative.
const (
	metaSize     = 632
	metaW        = 0x00
	metaH        = 0x04
	metaAngle    = 0x08
	metaWord0C   = 0x0c
	metaWord10   = 0x10
	metaWord14   = 0x14
	metaBitmask  = 0x18
	metaCount5   = 0x1c
	metaCount4   = 0x20
	metaCount6   = 0x24
	metaWord28   = 0x28
	metaWord2C   = 0x2c
	metaName     = 0x30
	metaWord70   = 0x70
	metaWord74   = 0x74
	metaDesc     = 0x78
	metaSlots    = 0xb8
	metaFieldLen = 64
	metaSlotLen  = 64
	metaSlotN    = 7
)

// Content-record sizes.
const (
	objectRecordSize = 20
	groupRecordSize  = 76
	unitRecordSize   = 70
)

// Unit-record offsets this story names: the two coordinate words a move
// rewrites, and the five bytes AC-2 requires a carried record to keep.
const (
	unitX      = 0x00
	unitY      = 0x04
	unitClass  = 0x08
	unitSubID  = 0x0a
	unitFlags  = 0x0c
	unitDefID  = 0x10
	unitOwner  = 0x14
	unitMark35 = 0x35
	unitMark3B = 0x3b
	unitID40   = 0x40
	unitID42   = 0x42
)

// ---------------------------------------------------------------------------
// The rich fixture's declared values
//
// Every value below is distinct from every other, so a setter writing at the
// wrong offset cannot land on a value that happens to match, and a comparator
// cannot pass because two regions looked alike. W != H for the same reason: a
// transposed cell index has to fail. Later tests must set values outside this
// table — locatedValues is the table, read back out of a fixture.
// ---------------------------------------------------------------------------

const (
	fixW = 3
	fixH = 2

	// Non-default: the corpus convention 4*W*H+72 would put 96 here, and
	// nothing in this story recomputes it.
	fixDataSize = 0x0000beef

	// One per-map constant in all ten record headers, non-zero.
	fixPerMapConst = 0xbfc02b6d

	// A signalling NaN: exponent all ones, quiet bit clear, mantissa non-zero.
	// It must survive as bits, never as a float that a conversion could quiet.
	fixAngleBits = 0x7fa00000

	fixWord0C   = 0xc0de000c
	fixWord10   = 0xc0de0010
	fixWord14   = 0xc0de0014
	fixBitmask  = 0x0000abcd
	fixWord28   = 0xc0de0028
	fixWord2C   = 0xc0de002c
	fixWord70   = 0xc0de0070
	fixWord74   = 0xc0de0074
	fixCount5   = 2
	fixCount4   = 2
	fixUnitsMax = 3 // the multi-unit variant's record count
)

// The name field: ASCII text, its terminator, then residue bytes that any
// mutation not targeting this field must carry.
var (
	fixNameText    = "Rich"
	fixNameResidue = []byte{0xb1, 0xb2, 0xb3}

	// The description field holds Windows-1251 bytes; the code points they
	// must decode to are declared beside them rather than written as text.
	fixDescBytes   = []byte{0xca, 0xe0, 0xf0, 0xf2, 0xe0}
	fixDescRunes   = []rune{0x041a, 0x0430, 0x0440, 0x0442, 0x0430}
	fixDescResidue = []byte{0xc1, 0xc2, 0xc3}

	// The three grids, row-major from their payload's first byte: cell (x,y)
	// lives at (y*W+x)*elem. Two tiles carry the impassable bit (0x2000).
	fixTiles      = []uint16{0x2141, 0x0142, 0x0143, 0x2144, 0x0145, 0x0146}
	fixAltitudes  = []uint8{0x61, 0x62, 0x63, 0x64, 0x65, 0x66}
	fixOverlay    = []uint8{0x51, 0x52, 0x53, 0x54, 0x55, 0x56}
	fixGroupNames = []string{"P1", "P2"}
)

// fixSlotFill is the byte filling text slot i of the seven at +0xb8. Real maps
// hold trigger and quest strings there; what a witness needs is that the 448
// bytes are recognisable and not zero.
func fixSlotFill(i int) byte { return byte(0xd1 + i) }

// The unit record's declared values, per file-order index. A record's other
// bytes are a per-record ramp: the model interprets none of them and must
// carry all of them, so what matters is that no byte is zero.
func fixUnitX(i int) uint32       { return 0x00001100 + 0x100*uint32(i) }
func fixUnitY(i int) uint32       { return 0x00002100 + 0x100*uint32(i) }
func fixUnitOwner(i int) byte     { return byte(0x71 + i) }
func fixUnitMark35(i int) byte    { return byte(0x81 + i) }
func fixUnitMark3B(i int) byte    { return byte(0x91 + i) }
func fixUnitID40(i int) uint16    { return uint16(0xa100 + i) }
func fixUnitID42(i int) uint16    { return uint16(0xa200 + i) }
func fixUnitClass(i int) uint16   { return uint16(0x3000 + i) }
func fixUnitSubID(i int) uint16   { return uint16(0x3100 + i) }
func fixUnitDefID(i int) uint32   { return 0x32000000 + uint32(i) }
func fixObjectX(i int) uint32     { return 0x00003100 + uint32(i) }
func fixObjectY(i int) uint32     { return 0x00004100 + uint32(i) }
func fixGroupScalar(i int) uint32 { return uint32(5000 + i) }

// ---------------------------------------------------------------------------
// Byte helpers
// ---------------------------------------------------------------------------

func putU16(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
func putU32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }
func u16At(b []byte, off int) uint16     { return binary.LittleEndian.Uint16(b[off:]) }
func u32At(b []byte, off int) uint32     { return binary.LittleEndian.Uint32(b[off:]) }

func clone(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// ---------------------------------------------------------------------------
// The fixture builder
// ---------------------------------------------------------------------------

// recordOrder is a physical record order: which typeId sits where in the file,
// and whether type-6 precedes type-0 in it — the property the second order
// exists for, asserted rather than assumed.
type recordOrder struct {
	name       string
	ids        []uint32
	type6First bool
}

var (
	// The corpus-observed order, type-0 first.
	orderType0First = recordOrder{"type-0 first", []uint32{0, 1, 2, 3, 5, 4, 9, 8, 6, 7}, false}

	// type-6 ahead of type-0. With type-0 first, the #type6 and payloadSize
	// words a place rewrites sit at lower offsets than the bytes it inserts,
	// and an applier that reverts its parts in the wrong order passes anyway;
	// here they sit above the insertion and the ordering rule is witnessed.
	orderType6First = recordOrder{"type-6 first", []uint32{6, 1, 2, 3, 5, 4, 9, 8, 0, 7}, true}

	fixtureOrders = []recordOrder{orderType0First, orderType6First}
)

// fixMeta builds the 632-byte type-0 payload for a map with nUnits units.
func fixMeta(nUnits int) []byte {
	p := make([]byte, metaSize)
	putU32(p, metaW, fixW)
	putU32(p, metaH, fixH)
	putU32(p, metaAngle, fixAngleBits)
	putU32(p, metaWord0C, fixWord0C)
	putU32(p, metaWord10, fixWord10)
	putU32(p, metaWord14, fixWord14)
	putU32(p, metaBitmask, fixBitmask)
	putU32(p, metaCount5, fixCount5)
	putU32(p, metaCount4, fixCount4)
	putU32(p, metaCount6, uint32(nUnits))
	putU32(p, metaWord28, fixWord28)
	putU32(p, metaWord2C, fixWord2C)
	putU32(p, metaWord70, fixWord70)
	putU32(p, metaWord74, fixWord74)

	copy(p[metaName:], fixNameText)
	copy(p[metaName+len(fixNameText)+1:], fixNameResidue)
	copy(p[metaDesc:], fixDescBytes)
	copy(p[metaDesc+len(fixDescBytes)+1:], fixDescResidue)

	for i := 0; i < metaSlotN; i++ {
		slot := p[metaSlots+i*metaSlotLen : metaSlots+(i+1)*metaSlotLen]
		for j := range slot {
			slot[j] = fixSlotFill(i)
		}
	}
	return p
}

func fixTilePayload() []byte {
	p := make([]byte, 2*fixW*fixH)
	for i, c := range fixTiles {
		putU16(p, i*2, c)
	}
	return p
}

func fixBytePayload(cells []uint8) []byte { return clone(cells) }

// fixObjects builds the type-4 payload: fixCount4 base records, none of them
// the kind==0x21 extension form, so the walk consumes exactly 20 bytes each.
func fixObjects() []byte {
	p := make([]byte, 0, fixCount4*objectRecordSize)
	for i := 0; i < fixCount4; i++ {
		r := make([]byte, objectRecordSize)
		for j := range r {
			r[j] = byte(0x21 + 5*i + j)
		}
		putU32(r, 0x00, fixObjectX(i))
		putU32(r, 0x04, fixObjectY(i))
		putU32(r, 0x08, 1) // kind != 0x21: no extension
		p = append(p, r...)
	}
	return p
}

// fixGroups builds the type-5 roster: fixCount5 records of 76 bytes, each with
// a terminated ASCII name and a non-zero byte after the terminator.
func fixGroups() []byte {
	p := make([]byte, 0, fixCount5*groupRecordSize)
	for i := 0; i < fixCount5; i++ {
		r := make([]byte, groupRecordSize)
		putU32(r, 0x08, fixGroupScalar(i))
		copy(r[0x0c:], fixGroupNames[i])
		r[0x0c+len(fixGroupNames[i])+1] = byte(0xe1 + i)
		p = append(p, r...)
	}
	return p
}

// fixUnitRecord builds one 70-byte type-6 record. Every byte is non-zero: the
// named fields carry their declared values and the rest is a per-record ramp,
// because a carried byte that was zero to begin with witnesses nothing.
func fixUnitRecord(i int) []byte {
	r := make([]byte, unitRecordSize)
	for j := range r {
		r[j] = byte(0xa0 + 3*i + j)
	}
	putU32(r, unitX, fixUnitX(i))
	putU32(r, unitY, fixUnitY(i))
	putU16(r, unitClass, fixUnitClass(i))
	putU16(r, unitSubID, fixUnitSubID(i))
	putU32(r, unitFlags, 0) // bit 0 clear: not the NPC path
	putU32(r, unitDefID, fixUnitDefID(i))
	r[unitOwner] = fixUnitOwner(i)
	r[unitMark35] = fixUnitMark35(i)
	r[unitMark3B] = fixUnitMark3B(i)
	putU16(r, unitID40, fixUnitID40(i))
	putU16(r, unitID42, fixUnitID42(i))
	return r
}

func fixUnits(nUnits int) []byte {
	p := make([]byte, 0, nUnits*unitRecordSize)
	for i := 0; i < nUnits; i++ {
		p = append(p, fixUnitRecord(i)...)
	}
	return p
}

// fixCountBody builds a `[u32 count][body]` payload (type-7 and type-9).
func fixCountBody(count uint32, body []byte) []byte {
	p := make([]byte, 4+len(body))
	putU32(p, 0, count)
	copy(p[4:], body)
	return p
}

func fixRamp(n int, base byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = base + byte(i)
	}
	return b
}

// fixPayloads is the rich map's payload per typeId. A roster that omits a
// record simply never asks for its entry.
func fixPayloads(nUnits int) map[uint32][]byte {
	return map[uint32][]byte{
		0: fixMeta(nUnits),
		1: fixTilePayload(),
		2: fixBytePayload(fixAltitudes),
		3: fixBytePayload(fixOverlay),
		4: fixObjects(),
		5: fixGroups(),
		6: fixUnits(nUnits),
		7: fixCountBody(3, fixRamp(8, 0xf1)),
		8: fixRamp(6, 0x11),
		9: fixCountBody(2, fixRamp(6, 0x31)),
	}
}

// rosterFixture assembles the rich map over an explicit physical roster: ids
// gives the typeId of each record in file order, and count is what the file
// header's own record count word says about them.
//
// Two parameters that are normally redundant are separate on purpose. count is
// usually len(ids), and where it is deliberately smaller the records past it are
// physically in the file but past the loader's loop bound — bytes no walk
// reaches, which is a state alm accepts and a state a fixed-length record walk
// gets wrong. An id the payload table does not have (one at or above ten, which
// selects no section) is emitted with a distinctive ramp.
func rosterFixture(ids []uint32, nUnits, count int) []byte {
	payloads := fixPayloads(nUnits)

	out := make([]byte, fileHeaderSize)
	putU32(out, fhMagic, fileMagic)
	putU32(out, fhHdrLen, fileHeaderSize)
	putU32(out, fhDataSize, fixDataSize)
	putU32(out, fhRecordCount, uint32(count))
	putU32(out, fhFormatVersion, formatVersion990)

	for i, tid := range ids {
		p, ok := payloads[tid]
		if !ok {
			p = fixRamp(4+i, 0xc1)
		}
		hdr := make([]byte, recordHeaderSize)
		putU32(hdr, recTag, recordTag)
		putU32(hdr, recHdrLen, recordHdrLen)
		putU32(hdr, recPayloadSize, uint32(len(p)))
		putU32(hdr, recTypeID, tid)
		putU32(hdr, recPerMapConst, fixPerMapConst)
		out = append(out, hdr...)
		out = append(out, p...)
	}
	return out
}

// richFixture assembles the rich map in the given record order with nUnits
// type-6 records. nUnits == 0 is the empty-payload case: the record is still
// present, typed by its own header.
func richFixture(order recordOrder, nUnits int) []byte {
	return rosterFixture(order.ids, nUnits, len(order.ids))
}

// ---------------------------------------------------------------------------
// The frame walk — where every region is, from the contract alone
// ---------------------------------------------------------------------------

// recordFrame is one record located in an image: absolute offsets, file order.
type recordFrame struct {
	typeID      uint32
	headerOff   int
	payloadOff  int
	payloadSize int
}

type frames []recordFrame

// walkFrameTo locates an image's records from the container contract: a 20-byte
// file header, then the number of records its +0x0c word counts, each header's
// +0x08 word giving the payload length. It returns the records and the offset
// the walk ended at. It is deliberately independent of pkg/formats/alm's own
// navigation, which is what makes agreeing with it evidence rather than a
// tautology.
//
// The count is read from the header rather than fixed at ten, and that is the
// same correction the model itself owed: a file may carry more than ten records,
// and it may carry bytes past the last one the count reaches.
func walkFrameTo(t *testing.T, data []byte) (frames, int) {
	t.Helper()
	if len(data) < fileHeaderSize {
		t.Fatalf("frame walk: image is %d bytes, shorter than the file header", len(data))
	}
	n := int(u32At(data, fhRecordCount))
	fs := make(frames, 0, n)
	cursor := fileHeaderSize
	for i := 0; i < n; i++ {
		if cursor+recordHeaderSize > len(data) {
			t.Fatalf("frame walk: record %d header does not fit before EOF", i)
		}
		size := int(u32At(data, cursor+recPayloadSize))
		start := cursor + recordHeaderSize
		if start+size > len(data) {
			t.Fatalf("frame walk: record %d payload (%d bytes) overruns EOF", i, size)
		}
		fs = append(fs, recordFrame{
			typeID:      u32At(data, cursor+recTypeID),
			headerOff:   cursor,
			payloadOff:  start,
			payloadSize: size,
		})
		cursor = start + size
	}
	return fs, cursor
}

// walkFrame is walkFrameTo for an image whose counted records tile it exactly,
// which is every fixture but the one built to have bytes past the last of them.
func walkFrame(t *testing.T, data []byte) frames {
	t.Helper()
	fs, end := walkFrameTo(t, data)
	if end != len(data) {
		t.Fatalf("frame walk: records end at %d, want EOF at %d", end, len(data))
	}
	return fs
}

// find returns the frame of the record carrying typeId tid, and whether the
// image has one at all.
func (fs frames) find(tid uint32) (recordFrame, bool) {
	for _, f := range fs {
		if f.typeID == tid {
			return f, true
		}
	}
	return recordFrame{}, false
}

// byType returns the frame of the record carrying typeId tid.
func (fs frames) byType(t *testing.T, tid uint32) recordFrame {
	t.Helper()
	f, ok := fs.find(tid)
	if !ok {
		t.Fatalf("frame walk: no record with typeId %d", tid)
	}
	return f
}

// region is a half-open span of one image, [off, off+n).
type region struct {
	off int
	n   int
}

// payload returns the absolute region of n bytes at payload offset off inside
// the record carrying typeId tid.
func (fs frames) payload(t *testing.T, tid uint32, off, n int) region {
	t.Helper()
	f := fs.byType(t, tid)
	if off < 0 || n < 0 || off+n > f.payloadSize {
		t.Fatalf("frame walk: [%d,%d) is outside typeId %d's %d-byte payload", off, off+n, tid, f.payloadSize)
	}
	return region{off: f.payloadOff + off, n: n}
}

// payloadSizeWord returns the region of the record header word that declares
// the payload length of the record carrying typeId tid.
func (fs frames) payloadSizeWord(t *testing.T, tid uint32) region {
	t.Helper()
	return region{off: fs.byType(t, tid).headerOff + recPayloadSize, n: 4}
}

// cell returns the absolute region of grid cell (x,y) in the record carrying
// typeId tid, whose cells are elem bytes wide: (y*W+x)*elem from payload+0.
func (fs frames) cell(t *testing.T, tid uint32, x, y, elem int) region {
	t.Helper()
	return fs.payload(t, tid, (y*fixW+x)*elem, elem)
}

// unit returns the absolute region of the whole type-6 record at file-order
// index i.
func (fs frames) unit(t *testing.T, i int) region {
	t.Helper()
	return fs.payload(t, 6, i*unitRecordSize, unitRecordSize)
}

func (fs frames) at(t *testing.T, data []byte, r region) []byte {
	t.Helper()
	if r.off < 0 || r.off+r.n > len(data) {
		t.Fatalf("region [%d,%d) is outside the %d-byte image", r.off, r.off+r.n, len(data))
	}
	return data[r.off : r.off+r.n]
}

// remainder deletes the declared regions from img and returns what is left, in
// order. The regions are the caller's: this function computes none of them,
// and rejects a set that is out of bounds or overlapping rather than guessing
// what was meant.
func remainder(img []byte, regions []region) ([]byte, error) {
	sorted := slices.Clone(regions)
	slices.SortFunc(sorted, func(a, b region) int { return a.off - b.off })

	out := make([]byte, 0, len(img))
	cursor := 0
	for _, r := range sorted {
		if r.n < 0 || r.off < 0 || r.off+r.n > len(img) {
			return nil, fmt.Errorf("region [%d,%d) is outside the %d-byte image", r.off, r.off+r.n, len(img))
		}
		if r.off < cursor {
			return nil, fmt.Errorf("region at %d overlaps the one ending at %d", r.off, cursor)
		}
		out = append(out, img[cursor:r.off]...)
		cursor = r.off + r.n
	}
	return append(out, img[cursor:]...), nil
}

// carriedIdentical reports whether every byte outside the declared regions
// appears in after exactly as it appears in before — same content, same
// relative order. Each image declares its own regions, because a
// length-changing edit relocates every later byte and the invariant binds
// carried content, not offsets.
func carriedIdentical(before, after []byte, beforeRegions, afterRegions []region) (bool, string) {
	b, err := remainder(before, beforeRegions)
	if err != nil {
		return false, "before image: " + err.Error()
	}
	a, err := remainder(after, afterRegions)
	if err != nil {
		return false, "after image: " + err.Error()
	}
	if bytes.Equal(a, b) {
		return true, ""
	}

	i := 0
	for i < min(len(a), len(b)) && a[i] == b[i] {
		i++
	}
	window := func(s []byte) []byte {
		lo, hi := max(0, i-4), min(len(s), i+4)
		return s[lo:hi]
	}
	return false, fmt.Sprintf(
		"carried bytes differ at carried index %d: before % x, after % x (carried lengths %d and %d)",
		i, window(b), window(a), len(b), len(a))
}

// ---------------------------------------------------------------------------
// SC-2 — the apparatus is pinned before it measures anything
// ---------------------------------------------------------------------------

// fixtureVariants is every fixture this story measures with: both record
// orders, each with an empty type-6 payload and with several units.
func fixtureVariants() []struct {
	name   string
	order  recordOrder
	nUnits int
} {
	var out []struct {
		name   string
		order  recordOrder
		nUnits int
	}
	for _, order := range fixtureOrders {
		for _, n := range []int{0, fixUnitsMax} {
			out = append(out, struct {
				name   string
				order  recordOrder
				nUnits int
			}{fmt.Sprintf("%s/%d units", order.name, n), order, n})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// The thin rosters — maps alm accepts that do not carry every record
// ---------------------------------------------------------------------------

// thinRoster is one such map: the records it physically holds, how many of them
// its file header counts, and which of the two optional records this story
// addresses the model must then find missing.
//
// nUnits is the type-0 #type6 word either way, so on a roster with no type-6
// record that word names units the file does not carry — the shape a shipped
// four-record map really has, and the one that separates "the count says zero"
// from "there is no record for the count to be about".
type thinRoster struct {
	name    string
	ids     []uint32
	count   int
	overlay bool // a type-3 record is inside the counted frame
	units   bool // a type-6 record is inside the counted frame
}

func (r thinRoster) image() []byte { return rosterFixture(r.ids, fixUnitsMax, r.count) }

func thinRosters() []thinRoster {
	return []thinRoster{
		{"no type-3 record", []uint32{0, 1, 2, 5, 4, 9, 8, 6, 7}, 9, false, true},
		{"no type-6 record", []uint32{0, 1, 2, 3, 5, 4, 9, 8, 7}, 9, true, false},
		{"three records, the loader's minimum", []uint32{0, 1, 2}, 3, false, false},

		// Twelve records with two stepped-over ids among them, so type-6 sits
		// past the tenth: a walk that stopped at ten would report a map with
		// units as having none.
		{"twelve records, type-6 past the tenth", []uint32{0, 1, 2, 3, 5, 4, 9, 8, 7, 11, 12, 6}, 12, true, true},

		// The whole ten-record file with a count of three. Records 3..9 are
		// physically there and are bytes no walk may reach — acceptance permits
		// them, and a walk that ignored the count would find an overlay and a
		// unit roster that the map does not have.
		{"a count of three, the rest trailing", orderType0First.ids, 3, false, false},
	}
}

// TestThinRostersAreAcceptedAndMissWhatTheyDeclare pins the thin fixtures before
// anything measures with them: each is a map alm accepts and writes back, the
// records inside its counted frame are the ones it declares, and the shipped
// reader's own Present agrees with this file's independent walk.
func TestThinRostersAreAcceptedAndMissWhatTheyDeclare(t *testing.T) {
	for _, r := range thinRosters() {
		t.Run(r.name, func(t *testing.T) {
			data := r.image()

			m, err := alm.Open(data)
			if err != nil {
				t.Fatalf("alm.Open rejected the thin fixture: %v", err)
			}
			doc, err := alm.OpenDocument(data)
			if err != nil {
				t.Fatalf("alm.OpenDocument rejected the thin fixture: %v", err)
			}
			if got := doc.Write(); !bytes.Equal(got, data) {
				t.Errorf("the document does not write the thin fixture back (%d vs %d bytes)", len(got), len(data))
			}
			if got := doc.RecordCount(); got != r.count {
				t.Errorf("the document walked %d records, want the header's %d", got, r.count)
			}

			fs, _ := walkFrameTo(t, data)
			for _, c := range []struct {
				tid  uint32
				want bool
			}{{3, r.overlay}, {6, r.units}} {
				_, got := fs.find(c.tid)
				if got != c.want {
					t.Errorf("the walk finds a type-%d record = %v, want %v", c.tid, got, c.want)
				}
				if p := m.Present(c.tid); p != c.want {
					t.Errorf("alm says Present(%d) = %v, want %v", c.tid, p, c.want)
				}
			}

			// The count word outlives its record: without it this fixture could
			// not tell a zero count from an absent section.
			if !r.units && m.Meta.Count6 == 0 {
				t.Fatal("#type6 is zero on a roster with no type-6 record, so a zero unit count witnesses nothing")
			}
			if !r.units && len(m.Units) != 0 {
				t.Fatalf("the view has %d units on a roster with no type-6 record", len(m.Units))
			}
			if !r.overlay && len(m.Overlay) != fixW*fixH {
				t.Fatalf("the manufactured overlay is %d cells, want %d", len(m.Overlay), fixW*fixH)
			}
		})
	}
}

func TestFixtureIsAcceptedInBothOrdersAndIsByteStable(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			data := richFixture(v.order, v.nUnits)

			m, err := alm.Open(data)
			if err != nil {
				t.Fatalf("alm.Open rejected the fixture: %v", err)
			}
			if m == nil {
				t.Fatal("alm.Open returned a nil map with a nil error")
			}

			doc, err := alm.OpenDocument(data)
			if err != nil {
				t.Fatalf("alm.OpenDocument rejected the fixture: %v", err)
			}
			if got := doc.Write(); !bytes.Equal(got, data) {
				t.Errorf("the document does not write the fixture back byte for byte (%d vs %d bytes)", len(got), len(data))
			}
			if got := doc.RecordTypeIDs(); !slices.Equal(got, v.order.ids) {
				t.Errorf("record order in the file = %v, want %v", got, v.order.ids)
			}

			// The two orders exist to differ where it matters: the words a
			// place rewrites sit below the bytes it inserts in one order and
			// above them in the other. Two orders that agreed on this would
			// leave the applier's ordering rule unwitnessed.
			fs := walkFrame(t, data)
			t0, t6 := fs.byType(t, 0), fs.byType(t, 6)
			if got := t6.headerOff < t0.headerOff; got != v.order.type6First {
				t.Errorf("type-6 before type-0 = %v, want %v (type-0 at %d, type-6 at %d)",
					got, v.order.type6First, t0.headerOff, t6.headerOff)
			}
		})
	}
}

// TestFixtureDecodesToItsDeclaredValues pins the fixture against the shipped
// reader rather than against this file's own idea of the layout: if a declared
// offset or code-page byte were wrong, the decoded view would say so here.
func TestFixtureDecodesToItsDeclaredValues(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			m, err := alm.Open(richFixture(v.order, v.nUnits))
			if err != nil {
				t.Fatalf("alm.Open: %v", err)
			}

			if m.Width != fixW || m.Height != fixH {
				t.Errorf("dimensions = %dx%d, want %dx%d", m.Width, m.Height, fixW, fixH)
			}
			if m.DataSize != fixDataSize {
				t.Errorf("dataSize = %#x, want %#x", m.DataSize, fixDataSize)
			}
			if bits := m.MetaRecordWord; bits != fixPerMapConst {
				t.Errorf("per-map constant bits = %#08x, want %#08x", bits, uint32(fixPerMapConst))
			}
			if bits := math.Float32bits(m.Angle); bits != fixAngleBits {
				t.Errorf("angle bits = %#08x, want the signalling NaN %#08x", bits, uint32(fixAngleBits))
			}
			if m.Name != fixNameText {
				t.Errorf("name = %q, want %q", m.Name, fixNameText)
			}
			if want := string(fixDescRunes); m.Description != want {
				t.Errorf("description = %+q, want %+q", m.Description, want)
			}

			for _, c := range []struct {
				what string
				got  uint32
				want uint32
			}{
				{"Word0C", m.Meta.Word0C, fixWord0C},
				{"Word10", m.Meta.Word10, fixWord10},
				{"Word14", m.Meta.Word14, fixWord14},
				{"Bitmask", m.Meta.Bitmask, fixBitmask},
				{"Word28", m.Meta.Word28, fixWord28},
				{"Word2C", m.Meta.Word2C, fixWord2C},
				{"Word70", m.Meta.Word70, fixWord70},
				{"Word74", m.Meta.Word74, fixWord74},
				{"#type5", m.Meta.Count5, fixCount5},
				{"#type4", m.Meta.Count4, fixCount4},
				{"#type6", m.Meta.Count6, uint32(v.nUnits)},
			} {
				if c.got != c.want {
					t.Errorf("type-0 %s = %#x, want %#x", c.what, c.got, c.want)
				}
			}

			if !slices.Equal(m.Tiles, fixTiles) {
				t.Errorf("tiles = %v, want %v", m.Tiles, fixTiles)
			}
			if !slices.Equal(m.Altitudes, fixAltitudes) {
				t.Errorf("altitudes = %v, want %v", m.Altitudes, fixAltitudes)
			}
			if !slices.Equal(m.Overlay, fixOverlay) {
				t.Errorf("overlay = %v, want %v", m.Overlay, fixOverlay)
			}

			if len(m.Units) != v.nUnits {
				t.Fatalf("units = %d, want %d", len(m.Units), v.nUnits)
			}
			for i, u := range m.Units {
				if u.X != fixUnitX(i) || u.Y != fixUnitY(i) {
					t.Errorf("unit %d at (%#x,%#x), want (%#x,%#x)", i, u.X, u.Y, fixUnitX(i), fixUnitY(i))
				}
			}
			if len(m.Objects) != fixCount4 || len(m.Groups) != fixCount5 {
				t.Errorf("objects/groups = %d/%d, want %d/%d", len(m.Objects), len(m.Groups), fixCount4, fixCount5)
			}
		})
	}
}

// TestFrameWalkAgreesWithTheDocument is the walk's own pin: on all ten records
// of every fixture, the payload the walk locates is the payload the shipped
// document hands out, and in the same file order.
func TestFrameWalkAgreesWithTheDocument(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			data := richFixture(v.order, v.nUnits)
			doc, err := alm.OpenDocument(data)
			if err != nil {
				t.Fatalf("alm.OpenDocument: %v", err)
			}
			fs := walkFrame(t, data)
			if len(fs) != recordCount {
				t.Fatalf("frame walk found %d records, want %d", len(fs), recordCount)
			}
			ids := doc.RecordTypeIDs()
			for i, f := range fs {
				if f.typeID != ids[i] {
					t.Errorf("record %d: walk says typeId %d, the document says %d", i, f.typeID, ids[i])
				}
				want := doc.RecordPayload(i)
				got := data[f.payloadOff : f.payloadOff+f.payloadSize]
				if !bytes.Equal(got, want) {
					t.Errorf("record %d (typeId %d): walk locates %d bytes, the document has %d; contents differ",
						i, f.typeID, len(got), len(want))
				}
			}
		})
	}
}

// located is one declared value, read back out of a fixture through the frame
// walk: what the walk found, and what the fixture declared.
type located struct {
	name string
	got  uint64
	want uint64
}

// locatedValues reads every distinctive value of a fixture at the offset the
// frame walk computes for it. It is both halves of one claim: the values are
// where the contract says (got == want) and they are pairwise distinct, so no
// later witness can pass by landing on the wrong one.
func locatedValues(t *testing.T, data []byte, nUnits int) []located {
	t.Helper()
	fs := walkFrame(t, data)
	var out []located
	add := func(name string, got, want uint64) {
		out = append(out, located{name, got, want})
	}
	u32 := func(name string, r region, want uint32) {
		add(name, uint64(u32At(data, r.off)), uint64(want))
	}

	add("file dataSize", uint64(u32At(data, fhDataSize)), fixDataSize)
	u32("per-map constant", region{fs[0].headerOff + recPerMapConst, 4}, fixPerMapConst)
	u32("type-0 angle bits", fs.payload(t, 0, metaAngle, 4), fixAngleBits)
	u32("type-0 Word0C", fs.payload(t, 0, metaWord0C, 4), fixWord0C)
	u32("type-0 Word10", fs.payload(t, 0, metaWord10, 4), fixWord10)
	u32("type-0 Word14", fs.payload(t, 0, metaWord14, 4), fixWord14)
	u32("type-0 bitmask", fs.payload(t, 0, metaBitmask, 4), fixBitmask)
	u32("type-0 Word28", fs.payload(t, 0, metaWord28, 4), fixWord28)
	u32("type-0 Word2C", fs.payload(t, 0, metaWord2C, 4), fixWord2C)
	u32("type-0 Word70", fs.payload(t, 0, metaWord70, 4), fixWord70)
	u32("type-0 Word74", fs.payload(t, 0, metaWord74, 4), fixWord74)

	for i, want := range fixNameResidue {
		r := fs.payload(t, 0, metaName+len(fixNameText)+1+i, 1)
		add(fmt.Sprintf("name residue byte %d", i), uint64(data[r.off]), uint64(want))
	}
	for i, want := range fixDescResidue {
		r := fs.payload(t, 0, metaDesc+len(fixDescBytes)+1+i, 1)
		add(fmt.Sprintf("description residue byte %d", i), uint64(data[r.off]), uint64(want))
	}
	for i := 0; i < metaSlotN; i++ {
		r := fs.payload(t, 0, metaSlots+i*metaSlotLen, metaSlotLen)
		slot := fs.at(t, data, r)
		add(fmt.Sprintf("text slot %d fill", i), uint64(slot[0]), uint64(fixSlotFill(i)))
		for j, b := range slot {
			if b != fixSlotFill(i) {
				t.Errorf("text slot %d byte %d = %#02x, want the whole slot filled with %#02x", i, j, b, fixSlotFill(i))
				break
			}
		}
	}

	for y := 0; y < fixH; y++ {
		for x := 0; x < fixW; x++ {
			idx := y*fixW + x
			r := fs.cell(t, 1, x, y, 2)
			add(fmt.Sprintf("tile cell (%d,%d)", x, y), uint64(u16At(data, r.off)), uint64(fixTiles[idx]))
			r = fs.cell(t, 2, x, y, 1)
			add(fmt.Sprintf("altitude cell (%d,%d)", x, y), uint64(data[r.off]), uint64(fixAltitudes[idx]))
			r = fs.cell(t, 3, x, y, 1)
			add(fmt.Sprintf("overlay cell (%d,%d)", x, y), uint64(data[r.off]), uint64(fixOverlay[idx]))
		}
	}

	for i := 0; i < nUnits; i++ {
		u := fs.unit(t, i)
		rec := fs.at(t, data, u)
		add(fmt.Sprintf("unit %d X", i), uint64(u32At(rec, unitX)), uint64(fixUnitX(i)))
		add(fmt.Sprintf("unit %d Y", i), uint64(u32At(rec, unitY)), uint64(fixUnitY(i)))
		add(fmt.Sprintf("unit %d owner +0x14", i), uint64(rec[unitOwner]), uint64(fixUnitOwner(i)))
		add(fmt.Sprintf("unit %d mark +0x35", i), uint64(rec[unitMark35]), uint64(fixUnitMark35(i)))
		add(fmt.Sprintf("unit %d mark +0x3b", i), uint64(rec[unitMark3B]), uint64(fixUnitMark3B(i)))
		add(fmt.Sprintf("unit %d id +0x40", i), uint64(u16At(rec, unitID40)), uint64(fixUnitID40(i)))
		add(fmt.Sprintf("unit %d id +0x42", i), uint64(u16At(rec, unitID42)), uint64(fixUnitID42(i)))
	}
	return out
}

func TestFixtureDistinctiveValuesArePresentAndPairwiseDistinct(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			data := richFixture(v.order, v.nUnits)
			values := locatedValues(t, data, v.nUnits)
			if len(values) < 40 {
				t.Fatalf("the distinctive-value table has only %d entries; the fixture is not rich", len(values))
			}

			for _, l := range values {
				if l.got != l.want {
					t.Errorf("%s = %#x at the offset the walk computed, want %#x", l.name, l.got, l.want)
				}
				if l.want == 0 {
					t.Errorf("%s declares zero: a byte that was already zero witnesses neither a carry nor a write", l.name)
				}
			}

			seen := make(map[uint64]string, len(values))
			for _, l := range values {
				if prev, dup := seen[l.want]; dup {
					t.Errorf("%s and %s both declare %#x; a wrong offset could pass by coincidence", prev, l.name, l.want)
					continue
				}
				seen[l.want] = l.name
			}

			// Every record header carries the one per-map constant.
			for i, f := range walkFrame(t, data) {
				if got := u32At(data, f.headerOff+recPerMapConst); got != fixPerMapConst {
					t.Errorf("record %d per-map constant = %#08x, want %#08x", i, got, uint32(fixPerMapConst))
				}
			}
		})
	}
}

// TestComparatorOwnCases drives the comparator itself: it must accept a change
// confined to a declared region and reject a change outside one, a reordering
// of carried bytes that preserves their multiset, and a carried byte that
// changed under a length-changing edit.
func TestComparatorOwnCases(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			before := richFixture(order, fixUnitsMax)
			fs := walkFrame(t, before)
			word0C := fs.payload(t, 0, metaWord0C, 4)
			word10 := fs.payload(t, 0, metaWord10, 4)

			// Accepted: the only change is inside the declared region.
			after := clone(before)
			putU32(after, word0C.off, 0x5e5e5e5e)
			if ok, why := carriedIdentical(before, after, []region{word0C}, []region{word0C}); !ok {
				t.Errorf("in-region change rejected: %s", why)
			}

			// Rejected: a change outside every declared region.
			after = clone(before)
			putU32(after, word10.off, 0x5e5e5e5e)
			if ok, _ := carriedIdentical(before, after, []region{word0C}, []region{word0C}); ok {
				t.Error("an out-of-region change was accepted")
			}

			// Rejected: the carried bytes are all still there, in a different
			// order. The altitude payload is rotated by one cell, so the
			// multiset is untouched and only the sequence changes.
			after = clone(before)
			alt := fs.byType(t, 2)
			rotated := clone(after[alt.payloadOff : alt.payloadOff+alt.payloadSize])
			copy(after[alt.payloadOff:], append(rotated[1:], rotated[0]))
			gotSet := clone(after[alt.payloadOff : alt.payloadOff+alt.payloadSize])
			wantSet := clone(before[alt.payloadOff : alt.payloadOff+alt.payloadSize])
			slices.Sort(gotSet)
			slices.Sort(wantSet)
			if !bytes.Equal(gotSet, wantSet) {
				t.Fatalf("the reordering case changed the byte multiset, so it would witness content rather than order")
			}
			if ok, _ := carriedIdentical(before, after, []region{word0C}, []region{word0C}); ok {
				t.Error("a reordering of carried bytes was accepted")
			}

			// Accepted: a length-changing edit. Seventy bytes are appended to
			// the type-6 payload and the two count words are rewritten, each
			// image declaring its own regions — in the type-6-first order the
			// count words sit above the insertion and move with it.
			grown, growRegions, beforeRegions := growByOneUnit(t, before)
			if ok, why := carriedIdentical(before, grown, beforeRegions, growRegions); !ok {
				t.Errorf("a length-changing edit's carried bytes were rejected: %s", why)
			}
			if _, err := alm.OpenDocument(grown); err != nil {
				t.Errorf("the grown image is not an acceptable map: %v", err)
			}

			// Rejected: the same edit with one carried byte disturbed.
			spoiled := clone(grown)
			slot := walkFrame(t, spoiled).payload(t, 0, metaSlots, metaSlotLen)
			spoiled[slot.off] ^= 0xff
			if ok, _ := carriedIdentical(before, spoiled, beforeRegions, growRegions); ok {
				t.Error("a length-changing edit that disturbed a carried byte was accepted")
			}
		})
	}
}

// growByOneUnit appends one type-6 record to an image the way a place must,
// returning the grown image and the declared regions of each image: the
// inserted bytes (empty in the pre-image), the type-6 payloadSize word and the
// #type6 count word, each located by walking its own image.
func growByOneUnit(t *testing.T, before []byte) (grown []byte, afterRegions, beforeRegions []region) {
	t.Helper()
	fs := walkFrame(t, before)
	t6 := fs.byType(t, 6)
	insertAt := t6.payloadOff + t6.payloadSize
	rec := fixUnitRecord(fixUnitsMax) // an index the fixture does not use

	grown = make([]byte, 0, len(before)+len(rec))
	grown = append(grown, before[:insertAt]...)
	grown = append(grown, rec...)
	grown = append(grown, before[insertAt:]...)

	// The type-6 header precedes the insertion, so its payloadSize word is at
	// the same offset in both images; it has to be corrected before the grown
	// image can be walked at all.
	putU32(grown, fs.payloadSizeWord(t, 6).off, uint32(t6.payloadSize+unitRecordSize))
	gs := walkFrame(t, grown)
	count := gs.payload(t, 0, metaCount6, 4)
	putU32(grown, count.off, u32At(grown, count.off)+1)

	beforeRegions = []region{
		{insertAt, 0},
		fs.payloadSizeWord(t, 6),
		fs.payload(t, 0, metaCount6, 4),
	}
	afterRegions = []region{
		{insertAt, unitRecordSize},
		gs.payloadSizeWord(t, 6),
		count,
	}
	return grown, afterRegions, beforeRegions
}
