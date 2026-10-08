package alm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"

	"golang.org/x/text/encoding/charmap"
)

// Layout constants for the ROM1 .alm map container (little-endian): a 20-byte
// file header, then recordCount records, each a 20-byte header
// [tag=7][hdrLen=20][payloadSize][typeId][f32] followed by pure payload
// (ALM-FRAME-031). See the package doc and docs/0003-alm-container/spec.md.
const (
	fileMagic      = 0x0052374D // file header[0x00] "M7R\x00" as a little-endian u32
	fileHeaderSize = 20         // magic + hdrLen + dataSize + recordCount + formatVersion
	fileHdrLen     = 20         // file header[0x04]; the header's own length, constant

	// typeIDCount is the size of the typeId INDEX SPACE, not a required record
	// count: the loader dispatches each record through a 10-entry jump table, so
	// 0..9 are the ids that select a case and an id at or above it selects none
	// (ALM-REQ-055). The shipped writer emits one record per id, which is a
	// regularity of that writer and not a rule the loader applies.
	typeIDCount = 10

	// minRecordCount is the loader's ONLY count gate (ALM-REQ-055). Below it the
	// loader refuses; at or above it the file header's recordCount is simply the
	// loop bound.
	minRecordCount = 3

	maxFormatVersion = 1001 // the loader's version gate (ALM-META-024)
	skipHdrVersion   = 1000 // formatVersion that skips record headers (unimplemented, rejected)

	recordHeaderSize = 20 // tag + hdrLen + payloadSize + typeId + perMapConst
	recordTag        = 7  // record header[+0x00]; constant
	recordHdrLen     = 20 // record header[+0x04]; == its own length

	metaSize     = 632 // type-0 payload size, constant on all 38 corpus maps
	metaNameLen  = 64  // type-0 name field (+0x30)
	metaDescLen  = 64  // type-0 description field (+0x78)
	metaBlockLen = 512
	metaSlotsLen = 448 // type-0 trailing "<None>" slots (+0xb8): 7 x 64 B

	objectRecordSize = 20   // type4 base record
	objectExtSize    = 8    // type4 kind==0x21 extension
	objectExtKind    = 0x21 // type4 kind (+0x08) value that appends an extension
	groupRecordSize  = 76   // type5 record
	groupRelationOff = 0x2c // type5 record: the sixteen u16 diplomacy words
	groupRelationLen = 16   // type5 record: how many of them there are
	unitRecordSize   = 70   // type6 record

	countWordSize = 4 // type7/type9 leading count word

	// type1 (Tiles) cell bit fields (ALM-GRID-012).
	tileIndexMask = 0x03ff // bits 0-9: tile index
	impassableBit = 0x2000 // bit 13: impassable flag
)

// File-header field offsets (absolute).
const (
	fhMagic         = 0x00
	fhHdrLen        = 0x04
	fhDataSize      = 0x08
	fhRecordCount   = 0x0c
	fhFormatVersion = 0x10
)

// Record-header field offsets (record-relative).
const (
	recTag         = 0x00
	recHdrLen      = 0x04
	recPayloadSize = 0x08
	recTypeID      = 0x0c
	recWord10      = 0x10
)

const (
	metaW       = 0x00
	metaH       = 0x04
	metaAngle   = 0x08
	metaWord0C  = 0x0c
	metaWord10  = 0x10
	metaWord14  = 0x14
	metaBitmask = 0x18
	metaCount5  = 0x1c // #type5
	metaCount4  = 0x20 // #type4
	metaCount6  = 0x24 // #type6
	metaWord28  = 0x28 // read-and-discarded by the loader (Unknown)
	metaWord2C  = 0x2c // #type8 records (not required here; type8 is preserved raw)
	metaName    = 0x30
	metaWord70  = 0x70
	metaWord74  = 0x74
	metaDesc    = 0x78
	metaSlots   = 0xb8
)

// Record is one decoded record header, in physical (file) order. tag and hdrLen
// are validated constants (7, 20) and not stored; typeId, the payload byte
// length, and opaque +0x10 word are decoded (ALM-HEADER-098).
type Record struct {
	TypeID      uint32
	PayloadSize uint32
	Word10      uint32 // record header +0x10, uninterpreted bits; may differ between records
	Tag         uint32 // record header +0x00; set by the ROM2 reader only
}

// Meta holds the located type-0 scalars whose meanings the format does not yet
// decode (R-1): the ranged u32 words and the low-bit bitmask, plus the three
// content counts (which the reader also uses to drive the content walks), and
// the 448-byte trailing "<None>" slots. All are exposed raw, no invented meaning.
type Meta struct {
	Word0C, Word10, Word14 uint32 // +0x0c..+0x14 stored scalars
	Bitmask                uint32 // +0x18 low-bit bitmask (read & discarded by the loader)
	Count5                 uint32 // +0x1c #type5 (== len(Groups))
	Count4                 uint32 // +0x20 #type4 (== len(Objects) base records)
	Count6                 uint32 // +0x24 #type6 (== len(Units))
	Word28                 uint32 // +0x28 read-and-discarded (Unknown)
	Word2C                 uint32 // +0x2c #type8 records
	Word70, Word74         uint32 // +0x70, +0x74
	Slots                  []byte // +0xb8, 448 raw bytes

	Count10 uint32
	Count11 [3]uint32
	Count12 uint32
	Extra   [2]uint32
}

// Object is one placed object/structure (type4). X/Y are u32 fixed-point (/256;
// integer tile = value>>8). Kind (+0x08) is the extension discriminator: a record
// whose Kind == 0x21 appends an 8-byte extension (ALM-OBJ-034), retained raw in
// Ext (nil when absent). The ctor fields at +0x0c/+0x0e/+0x12 are raw (their field
// ids are undecoded, R-2).
type Object struct {
	X       uint32 // +0x00 (/256)
	Y       uint32 // +0x04 (/256)
	Kind    uint32 // +0x08 (0x21 => an extension follows)
	Field0C uint16 // +0x0c raw
	Field0E uint32 // +0x0e raw
	Field12 uint16 // +0x12 raw
	Ext     []byte // +0x14 extension (8 bytes) iff Kind==0x21, else nil
}

// Group is one player/group roster entry (type5). Scalar (+0x08, {0,5000}) is raw;
// Name (+0x0c) is NUL-terminated ASCII decoded to UTF-8.
//
// Relation is the record's SIXTEEN u16 at +0x2c, one per editor player slot,
// carried at the width the file writes them and interpreted in no way. They are
// the row the map author writes for this roster entry: what it thinks of each of
// the sixteen slots the editor lets a map declare (ALM-GRP-041, and the editor's
// own cap at ALM-META-025). Sixteen words of two bytes each end the record at
// exactly 0x4c = 76, which is what makes this the last field rather than a field
// somewhere in a tail.
//
// NOTHING IS NARROWED, INDEXED OR FORCED HERE, and that is the boundary between
// this leaf and its consumer rather than an omission. What the engine does with
// the row is a MAP-LOAD store, not a fact about the file: it writes each word's
// LOW BYTE into a 1-based matrix at a stride of its own, leaves column 0 alone,
// and then overwrites the diagonal with 2 (AI-DIPLO-005). All three of those are
// decisions about a matrix this package does not have and must not invent —
// notably the forced diagonal, which is load-bearing rather than decorative: the
// shipped corpus spells its own diagonal as something other than 2 in five of
// 191 records, so a decoder that "helpfully" normalised the row would erase the
// one place the file and the engine disagree.
//
// The value space is a CORPUS FACT and is not enforced. Over the 38 shipped maps
// the 3056 words take only 0, 1 and 2, so no byte store can truncate one — but
// that is a census at Medium and the field is a u16, so a map carrying 0x1234
// decodes to 0x1234 here and it is the consumer that decides what to do with it.
type Group struct {
	Color, Participant uint32
	Scalar             uint32
	Name               string
	Relation           [groupRelationLen]uint16
}

// Unit is one placed unit (type6), decoded from the 70-byte file record. X/Y are
// u32 fixed-point (/256; integer tile = value>>8).
//
// ClassID is the primary class key: a units.reg ID. That registry's class array is
// itself keyed by ID, so a consumer subscripts it with the key directly — nothing
// is translated (REG-KEY-044). It is read sign-extended because the loader MOVSXes
// it (ALM-CLS-038); a uint16 read would turn a negative key into ~65000. ClassSubID
// is the secondary key, which is also the scenario/npc.reg subscript on the NPC path.
//
// ClassID is not unconditionally the class. The engine resolves the record through
// a different table when DefID is nonzero and != 0xcdcdcdcd (a definition id), and
// when Flags bit 0 is set (the NPC path, subscripted by ClassSubID). Both conditions
// are fields of this file record, so a consumer can tell an overridden record from
// an unresolvable one. No precedence between the two is asserted: ALM-CLS-038 names
// both paths and does not order them. This leaf resolves nothing — the registry
// lookup is the data tier's job.
//
// CurrentHP is the signed word at +0x20. A raw -1 is the editor's sentinel for
// "derive current health from the resolved maximum"; HasCurrentHP is false only
// for that sentinel. Every other signed value, including zero and -10, is an
// authored override and must reach placement unchanged (UNIT-PLACE-034).
//
// The rest of the 70-byte record is decoded (ALM-UNIT-040 publishes the whole read
// map: the index at +0x18, located scalars and the two 0xFF-sentinel runs) but
// deliberately not exposed here — none of it has a consumer (DD12).
//
// Owner is the record's OWNER SLOT at +0x14, and it left that unexposed set when
// a consumer appeared for it. It is a 1-BASED INDEX INTO THE MAP'S OWN ROSTER —
// the type-5 array — and not that roster record's own id word: the rival reading
// fails 26 times per installed root, where a map whose roster ids are 6 9 10 4
// carries parameters naming slot 1 (ALM-OWN-039). So slot 1 is the first roster
// entry and ZERO NAMES NO SLOT AT ALL.
//
// It is carried at the record's own 32-bit width though no shipped map authors a
// value above 9. Narrowing it here would invent a limit the file does not state;
// the limit that is real is the roster's, sixteen editor player slots
// (ALM-GRP-041), and it belongs to the roster rather than to this word.
//
// UnitID and GroupID are the record's two identifier words, and they are the
// two the MAP'S OWN SCRIPT names: a type-7 Target_Unit parameter resolves
// against UnitID and a Target_Group parameter against GroupID. They were
// decoded and deliberately not exposed while nothing consumed them; the type-7
// grammar in script.go is that consumer.
//
// WHICH WORD IS WHICH IS MEDIUM, and it was published the other way round
// first: the loader's own running-maximum instruction over +0x42 was read as
// "a unique id", and it is not unique — on one shipped map +0x42 takes 19
// distinct values over 39 records while +0x40 takes 39. What decides the labels
// is the script's own vocabulary resolving 169 of 169 group references against
// +0x42, not an instruction that dereferences either field, so a consumer that
// binds a script by these words is standing on corpus agreement and should say
// so. The instruction is not contradicted by the correction — an engine that
// spawns groups at runtime needs to allocate GROUP ids.
type Unit struct {
	X            uint32 // +0x00 (/256)
	Y            uint32 // +0x04 (/256)
	ClassID      int16  // +0x08 primary class key, sign-extended (units.reg ID, raw)
	ClassSubID   uint16 // +0x0a secondary class key / scenario/npc.reg subscript (raw)
	Flags        uint32 // +0x0c bit 0 selects the NPC path (raw)
	DefID        uint32 // +0x10 definition id; overrides the class key when != 0 and != 0xcdcdcdcd (raw)
	Owner        uint32 // +0x14 1-based slot in the map's type-5 roster; 0 names none (raw)
	CurrentHP    int16  // +0x20 signed current health, or -1 sentinel (raw)
	HasCurrentHP bool   // true when CurrentHP is authored rather than the -1 sentinel
	UnitID       uint16 // +0x40 the id a script's Target_Unit names (raw)
	GroupID      uint32 // +0x42 the id a script's Target_Group names (raw)
	ServerID     uint32
}

// Triggers is the type7 trigger effect/instant list. EntryCount is the leading
// count word (+0x00); Body is everything after +0x04, preserved RAW.
//
// The leaf grammar is decoded, in script.go, and it is decoded THROUGH these two
// fields rather than instead of them: Map.Script() walks Body and Body keeps the
// bytes the file carried, so a map that is read and written back is unaffected by
// the leaf decoder existing. EntryCount is the first of the payload's THREE count
// words — the action count — which is why Body begins at the first action record
// and not at a second header.
type Triggers struct {
	EntryCount uint32
	Body       []byte
}

// LootSection is the type8 record: the map's own authored loot — ground sacks
// and actor stock — not a marker tree; that reading is retracted. Body is the
// whole payload, preserved raw: there is no identity to skip, and no count
// word to skip either, because the payload carries none of its own — the
// leaf grammar (loot.go) reads the record count from the type-0 metadata
// word at +0x2c instead.
//
// The leaf grammar is decoded, in loot.go, and it is decoded THROUGH this
// field rather than instead of it: Map.Loot() walks Body and Body keeps the
// bytes the file carried, so a map that is read and written back is
// unaffected by the leaf decoder existing.
type LootSection struct {
	Body []byte
}

// TileMarkers is the type9 tile-marker list. Count is the leading count word
// (+0x00); Body is the records after +0x04, preserved raw (R-2).
type TileMarkers struct {
	Count uint32
	Body  []byte
}

// EnchantmentElement is one ordered six-byte tail entry of a type-9 record.
type EnchantmentElement struct {
	Kind, Low, High uint16
}

// Enchantment is one complete type-9 record. The raw TileMarkers body remains
// the write-back source; this value is its first-class decoded leaf.
type Enchantment struct {
	Tag, X, Y uint32
	A, B, C   uint16
	SpellRaw  uint32
	Elements  []EnchantmentElement
}

// Map is a decoded ROM1 .alm map.
//
// A section field being empty does not say the file carried an empty record:
// an absent type-4 and a present but empty one both decode to no Objects, and
// Overlay is a full W*H plane whether it was read or manufactured. Present
// reports which of the two it was, and is the only thing that may be used to
// decide whether a record can be written back.
type Map struct {
	HdrLen         uint32  // file header +0x04, == 20 (the header's own length)
	DataSize       uint32  // file header +0x08, precomputed size, read & ignored
	RecordCount    uint32  // file header +0x0c, the walked record count (>= 3)
	FormatVersion  uint32  // file header +0x10, <= 1001 (990 on every shipped map)
	MetaRecordWord uint32  // +0x10 of the last type-0 record header, uninterpreted bits
	Angle          float32 // type-0 +0x08 (raw, R-1)

	Width       int
	Height      int
	Name        string // ASCII -> UTF-8
	Description string // CP1251 -> UTF-8
	Meta        Meta

	// Records is every record header the walk read, in physical (file) order,
	// including any whose typeId is outside 0..9 and which therefore selects no
	// decoder. len(Records) == RecordCount.
	Records []Record

	Tiles     []uint16 // type1, W*H pure cells (row-major); always read
	Altitudes []uint8  // type2, W*H cells; always read
	Overlay   []uint8  // type3, W*H cells; MANUFACTURED as zeros when absent (see Present)

	Objects []Object // type4; empty when the record is absent

	// Groups is the type5 roster and is EMPTY when the record is absent, which
	// means a map with no roster decodes here to NO PLAYERS. The engine does not
	// leave it at none: with the record missing it synthesises one entry with a
	// scalar of 5000 and an empty name (ALM-REQ-056). That default is a runtime
	// decision and this leaf does not take it — a reader that manufactured a
	// roster entry would put a record in the file that was never there, and
	// Write would then emit it. A consumer that needs the engine's behaviour
	// applies the default itself, and Present(5) is how it tells the two apart:
	// false means the file had no roster, not that the roster was empty.
	Groups []Group

	Units []Unit // type6; empty when the record is absent
	// AuthoredUnits is Units as decoded. A loader that withdraws placements
	// from Units leaves it whole.
	AuthoredUnits []Unit

	Extension [3][]byte

	Triggers     Triggers    // type7; zero when the record is absent
	LootSection  LootSection // type8; empty when the record is absent
	TileMarkers  TileMarkers // type9; zero when the record is absent
	Enchantments []Enchantment

	// present[t] is true iff the file actually carried a record with typeId t.
	present [typeIDCount]bool
}

// Present reports whether the accepted stream actually carried a record with
// this typeId. It is false for every id at or above 10, which no record of this
// container can be decoded from.
//
// It exists to keep "the file said so" and "this reader supplied it" apart, and
// there is exactly one field where they differ: Overlay is a full W*H plane even
// when no type-3 record was read, because the engine manufactures that plane
// too. A caller that writes a map back MUST NOT emit a record whose Present is
// false — doing so would invent bytes the input never had.
//
// It is also the only signal for the defaults this reader deliberately does NOT
// apply. The engine synthesises a one-entry type-5 roster when that record is
// missing, so an absent type-5 decodes to no Groups here where the engine would
// have one player; supplying the default is a consumer-tier decision, and
// Present(5) is what a consumer tests to take it.
func (m *Map) Present(typeID uint32) bool {
	if typeID >= typeIDCount {
		return false
	}
	return m.present[typeID]
}

// TileIndex returns the tile index of a type1 (Tiles) cell (bits 0-9,
// ALM-GRID-012). Resolving the index to a terrain class is the mapload tier's
// job, not this leaf.
func TileIndex(cell uint16) uint16 { return cell & tileIndexMask }

// Impassable reports the type1 impassable flag of a cell (bit 13, 0x2000).
func Impassable(cell uint16) bool { return cell&impassableBit != 0 }

// Open reads a ROM1 .alm map from an in-memory byte stream. It validates the
// 20-byte file header (magic, hdrLen==20, recordCount>=3, formatVersion<=1001
// and not the 1000 header-skipping dialect), walks that many length-prefixed
// records from 0x14, then decodes the type-0 metadata, the three W x H grids and
// whichever content records the file carried. Each record is typed from its own
// 20-byte header, so an empty (payloadSize 0) record is typed directly.
//
// Acceptance is the loader's, with three deliberate exceptions named in
// docs/0003-alm-container/spec.md: type-0 is required, every size constraint
// stays a rejection, and formatVersion 1000 is refused as unimplemented. A
// record the file omits is not an error — Objects, Groups, Units and the
// trigger sections come back empty and Overlay is manufactured, exactly as the
// engine manufactures it. Present says which sections were really there.
//
// Malformed input yields a non-nil error and a nil *Map, never a panic or an
// out-of-bounds read.
func Open(data []byte) (*Map, error) { return open(data, dialectROM1) }

func open(data []byte, d dialect) (*Map, error) {
	f, err := walkRecordsIn(data, d)
	if err != nil {
		return nil, err
	}

	m := &Map{
		HdrLen:         f.fh.hdrLen,
		DataSize:       f.fh.dataSize,
		RecordCount:    f.fh.recordCount,
		FormatVersion:  f.fh.formatVersion,
		MetaRecordWord: f.headerWords[0],
		Records:        f.records,
		present:        f.present,
	}

	if err := m.decodeMeta(f.payloads[0], d); err != nil {
		return nil, err
	}
	if err := m.decodeGrids(f.payloads[1], f.payloads[2], f.payloads[3], f.present[3]); err != nil {
		return nil, err
	}
	// A count word in type-0 that names records the file does not carry is
	// INERT, not a contradiction: each of these cases is entered only when its
	// own record is present, and its count is nothing but that case's loop bound
	// (ALM-REQ-056). A shipped four-record map advertises 415 type-4 records it
	// does not contain (ALM-CORP-060), and the loader reads none of them.
	if f.present[4] {
		if err := m.decodeObjects(f.payloads[4], d); err != nil {
			return nil, err
		}
	}
	if f.present[5] {
		if err := m.decodeGroups(f.payloads[5]); err != nil {
			return nil, err
		}
	}
	if f.present[6] {
		if d == dialectROM2 {
			err = m.decodeUnitsROM2(f.payloads[6])
		} else {
			err = m.decodeUnits(f.payloads[6])
		}
		if err != nil {
			return nil, err
		}
	}
	if err := m.decodeTriggers(f.payloads[7], f.payloads[8], f.payloads[9], f.present[7], f.present[9]); err != nil {
		return nil, err
	}
	if err := m.decodeEnchantments(); err != nil {
		return nil, err
	}
	if d == dialectROM2 {
		if err := m.decodeExtension(f); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Info is a map's own type-0 metadata, decoded without decoding its grids or its
// content records.
//
// It exists because listing an install's maps needs each map's recorded name and
// nothing else, and because "the metadata could not be read" and "the metadata
// read but the map does not fully decode" are two different outcomes a caller
// may have to tell apart. Through Open they are one: a map with a valid type-0
// and an inconsistent grid fails the whole call, so it could only ever be
// dropped, never listed and then reported.
//
// Name is empty on most campaign maps. That is normal, not a failure.
type Info struct {
	FormatVersion uint32
	Width, Height int
	Name          string // ASCII -> UTF-8
	Description   string // CP1251 -> UTF-8

	ListDescription string // the 512-byte block, line feeds as '#' (TEXT-083)

	// Word70 and Word74 are the map list's last two columns.
	Word70, Word74 uint32
}

// OpenInfo decodes only a map's file header, its record headers and its type-0
// metadata payload, and stops.
//
// It shares the record walk and the type-0 decoder with Open, so the two can
// never disagree about the layout; it adds no format knowledge of its own. It is
// strictly weaker than Open: every stream Open accepts, OpenInfo accepts, and a
// well-framed stream whose later records are inconsistent decodes here and fails
// there. A stream truncated inside a record fails both, because the walk
// requires every header and payload it counts to lie in bounds.
func OpenInfo(data []byte) (*Info, error) {
	f, err := walkRecords(data)
	if err != nil {
		return nil, err
	}
	_, info, _, err := decodeMetaPayload(f.payloads[0])
	if err != nil {
		return nil, err
	}
	info.FormatVersion = f.fh.formatVersion
	return &info, nil
}

// fileHeader is the decoded 20-byte file header.
type fileHeader struct {
	hdrLen, dataSize, recordCount, formatVersion uint32
}

// recordFrame is one accepted stream's frame: the file header, every record
// header in physical order, and the payload of each typeId the file carried.
//
// payloads, headerWords and present are indexed by typeId and are the reason this is
// a struct rather than a tuple — absence is now representable, so a payload
// slice can only be read together with the flag that says the file had one.
type recordFrame struct {
	fh          fileHeader
	ext         [3][]byte
	extPresent  [3]bool
	payloads    [typeIDCount][]byte
	headerWords [typeIDCount]uint32
	present     [typeIDCount]bool
	records     []Record
}

// walkRecords validates the file header and walks the recordCount
// length-prefixed records, returning each typeId's payload and per-map
// constant.
//
// This is the part of a decode that is common to every caller: it establishes
// that the stream is a well-framed .alm and hands back the pieces, without
// interpreting any of them.
//
// What it rejects, and nothing else: a stream shorter than the file header, a
// bad magic, a file hdrLen that is not fileHdrLen, a recordCount below
// minRecordCount (3), the unimplemented header-skipping formatVersion and any
// version past maxFormatVersion, a record header that does not fit before EOF,
// a record tag or hdrLen that is not the constant, a payload that overruns EOF,
// and then the three records the open path needs — type-1 and type-2, whose
// planes the world builder dereferences per cell with no null test, and type-0,
// which is required deliberately more strictly than the engine (ALM-ORD-057).
//
// What it deliberately does NOT reject, each backed by the loader's own
// behaviour rather than by leniency:
//
//   - a repeated typeId. It overwrites, because the loader's switch case simply
//     runs again over the same pointer field (ALM-REQ-055).
//   - a typeId outside the jump table (>= typeIDCount). It selects no case and
//     is stepped over with no error; its bytes stay in the stream and a
//     Document preserves them (ALM-REQ-055).
//   - records that do not tile exactly to EOF. The loop bound is recordCount,
//     so a trailer past the last record is never reached.
//   - any typeId other than 0, 1 and 2 being absent.
func walkRecords(data []byte) (recordFrame, error) { return walkRecordsIn(data, dialectROM1) }

func walkRecordsIn(data []byte, d dialect) (recordFrame, error) {
	var f recordFrame
	fh := &f.fh
	if len(data) < fileHeaderSize {
		return f, fmt.Errorf("alm: stream too small: %d bytes", len(data))
	}
	if magic := binary.LittleEndian.Uint32(data[fhMagic : fhMagic+4]); magic != fileMagic {
		return f, fmt.Errorf("alm: bad magic %#08x", magic)
	}
	fh.hdrLen = binary.LittleEndian.Uint32(data[fhHdrLen : fhHdrLen+4])
	if fh.hdrLen != fileHdrLen {
		return f, fmt.Errorf("alm: file header length %d, want %d", fh.hdrLen, fileHdrLen)
	}
	fh.dataSize = binary.LittleEndian.Uint32(data[fhDataSize : fhDataSize+4])
	fh.recordCount = binary.LittleEndian.Uint32(data[fhRecordCount : fhRecordCount+4])
	if fh.recordCount < minRecordCount {
		return f, fmt.Errorf("alm: recordCount %d, want at least %d", fh.recordCount, minRecordCount)
	}
	fh.formatVersion = binary.LittleEndian.Uint32(data[fhFormatVersion : fhFormatVersion+4])
	if err := checkVersion(fh.formatVersion, d); err != nil {
		return f, err
	}

	total := int64(len(data))

	// f.payloads[t] is the payload slice of the record with typeId t and
	// f.headerWords[t] its opaque header word, both valid only where f.present[t]. A
	// repeated id overwrites, which is the loader's own behaviour: its switch
	// case simply runs again over the same pointer field (ALM-REQ-055).
	records := make([]Record, 0, minRecordCount)

	cursor := int64(fileHeaderSize)
	for i := uint32(0); i < fh.recordCount; i++ {
		if cursor+recordHeaderSize > total {
			return f, fmt.Errorf("alm: record %d header does not fit before EOF", i)
		}
		hdr := data[cursor : cursor+recordHeaderSize]
		tag := binary.LittleEndian.Uint32(hdr[recTag : recTag+4])
		rHdrLen := binary.LittleEndian.Uint32(hdr[recHdrLen : recHdrLen+4])
		payloadSize := binary.LittleEndian.Uint32(hdr[recPayloadSize : recPayloadSize+4])
		typeID := binary.LittleEndian.Uint32(hdr[recTypeID : recTypeID+4])
		word10 := binary.LittleEndian.Uint32(hdr[recWord10 : recWord10+4])
		if tag != recordTag && d == dialectROM1 {
			return f, fmt.Errorf("alm: record %d tag %d, want %d", i, tag, recordTag)
		}
		if rHdrLen != recordHdrLen {
			return f, fmt.Errorf("alm: record %d hdrLen %d, want %d", i, rHdrLen, recordHdrLen)
		}

		payloadStart := cursor + recordHeaderSize
		if d == dialectROM2 && typeID == 0 {
			payloadSize = rom2MetaSize
		}
		payloadEnd := payloadStart + int64(payloadSize)
		if payloadEnd > total {
			return f, fmt.Errorf("alm: record %d payload (%d bytes) overruns EOF", i, payloadSize)
		}
		// An id outside the jump table selects no case and is stepped over, no
		// error recorded (ALM-REQ-055). Its bytes stay in the stream and are
		// preserved by a Document; nothing here interprets them.
		if d == dialectROM2 && typeID >= typeIDCount && typeID < typeIDCount+3 {
			f.extPresent[typeID-typeIDCount] = true
			f.ext[typeID-typeIDCount] = data[payloadStart:payloadEnd]
		}
		if typeID < typeIDCount {
			f.present[typeID] = true
			f.payloads[typeID] = data[payloadStart:payloadEnd]
			f.headerWords[typeID] = word10
		}

		rec := Record{TypeID: typeID, PayloadSize: payloadSize, Word10: word10}
		if d == dialectROM2 {
			rec.Tag = tag
		}
		records = append(records, rec)
		cursor = payloadEnd
	}
	f.records = records

	// The loop bound is recordCount, so bytes past the last record are simply
	// never reached by the loader and are not an error here either.

	// What the open path actually requires, and it is two records: the world
	// builder dereferences the type-1 and type-2 planes per cell with no null
	// test, so their absence is the loader's own status 5 / status 6
	// (ALM-REQ-055). The messages keep the engine's names for them.
	if !f.present[1] {
		return f, errors.New("alm: no type-1 record (tiles block not found)")
	}
	if !f.present[2] {
		return f, errors.New("alm: no type-2 record (altitudes block not found)")
	}
	// STRICTER THAN THE ENGINE, deliberately. With no type-0 record the loader
	// does not default: case 1's grid length and cases 4..8's loop bounds are
	// read from stack slots only case 0 writes, so a type-0-less file carrying
	// type-1 or any of 4..9 is UNDEFINED rather than defaulted (ALM-ORD-057).
	// "Match the engine" has no meaning there, so this reader requires the
	// record instead of reproducing an undefined read. type-1 is required
	// above, so in practice type-0 is required of every accepted stream.
	if !f.present[0] {
		return f, errors.New("alm: no type-0 record, and type-1 or a content record is present")
	}
	return f, nil
}

// decodeMeta decodes the 632-byte type-0 metadata payload into the map.
func (m *Map) decodeMeta(p []byte, d dialect) error {
	decode := decodeMetaPayload
	if d == dialectROM2 {
		decode = decodeMetaROM2
	}
	angle, info, meta, err := decode(p)
	if err != nil {
		return err
	}
	m.Angle = angle
	m.Width = info.Width
	m.Height = info.Height
	m.Name = info.Name
	m.Description = info.Description
	m.Meta = meta
	return nil
}

func decodeMetaPayload(p []byte) (angle float32, info Info, meta Meta, err error) {
	if len(p) != metaSize {
		return 0, info, meta, fmt.Errorf("alm: type-0 payload is %d bytes, want %d", len(p), metaSize)
	}
	angle = math.Float32frombits(binary.LittleEndian.Uint32(p[metaAngle : metaAngle+4]))
	info.Width = int(binary.LittleEndian.Uint32(p[metaW : metaW+4]))
	info.Height = int(binary.LittleEndian.Uint32(p[metaH : metaH+4]))
	info.Name = decodeASCII(p[metaName : metaName+metaNameLen])
	info.Description = decodeCP1251(p[metaDesc : metaDesc+metaDescLen])
	info.ListDescription = decodeCP1251(bytes.ReplaceAll(p[metaDesc:metaDesc+metaBlockLen], []byte{0x0a}, []byte{'#'}))

	slots := make([]byte, metaSlotsLen)
	copy(slots, p[metaSlots:metaSlots+metaSlotsLen])
	meta = Meta{
		Word0C:  binary.LittleEndian.Uint32(p[metaWord0C : metaWord0C+4]),
		Word10:  binary.LittleEndian.Uint32(p[metaWord10 : metaWord10+4]),
		Word14:  binary.LittleEndian.Uint32(p[metaWord14 : metaWord14+4]),
		Bitmask: binary.LittleEndian.Uint32(p[metaBitmask : metaBitmask+4]),
		Count5:  binary.LittleEndian.Uint32(p[metaCount5 : metaCount5+4]),
		Count4:  binary.LittleEndian.Uint32(p[metaCount4 : metaCount4+4]),
		Count6:  binary.LittleEndian.Uint32(p[metaCount6 : metaCount6+4]),
		Word28:  binary.LittleEndian.Uint32(p[metaWord28 : metaWord28+4]),
		Word2C:  binary.LittleEndian.Uint32(p[metaWord2C : metaWord2C+4]),
		Word70:  binary.LittleEndian.Uint32(p[metaWord70 : metaWord70+4]),
		Word74:  binary.LittleEndian.Uint32(p[metaWord74 : metaWord74+4]),
		Slots:   slots,
	}
	info.Word70, info.Word74 = meta.Word70, meta.Word74
	return angle, info, meta, nil
}

// decodeGrids decodes the type1/type2/type3 grid layers, each W*H pure cells at
// payload+0 (no identity overlay under the corrected framing). cells = W*H is
// computed in uint64 from two u32 factors so it cannot wrap; the type1 size is
// checked as payloadSize/2 == cells (never as 2*cells, which could wrap for a
// hostile W*H) so every allocation is bounded by the in-bounds payload length.
// An absent type-3 record is MANUFACTURED as a W*H zero plane rather than left
// empty, because that is what the engine does — it allocates W*H and zero-fills
// (ALM-REQ-056) — and the default is semantically exact rather than a guess: the
// consumer's own per-cell test is "nonzero means an object blocks this cell", so
// an all-zero plane is a valid empty one. A short or nil Overlay would also be
// the worse failure downstream, silently disabling every consumer that sizes its
// work by len(Overlay) instead of failing where it could be seen.
func (m *Map) decodeGrids(p1, p2, p3 []byte, has3 bool) error {
	cells := uint64(uint32(m.Width)) * uint64(uint32(m.Height))

	if len(p1)%2 != 0 || uint64(len(p1))/2 != cells {
		return fmt.Errorf("alm: type1 payload is %d bytes, want an even 2*W*H (W*H = %d cells)", len(p1), cells)
	}
	if uint64(len(p2)) != cells {
		return fmt.Errorf("alm: type2 payload is %d bytes, want %d (W*H)", len(p2), cells)
	}
	if has3 && uint64(len(p3)) != cells {
		return fmt.Errorf("alm: type3 payload is %d bytes, want %d (W*H)", len(p3), cells)
	}

	n := int(cells) // bounded: cells == len(p2) <= len(data)
	tiles := make([]uint16, n)
	for i := 0; i < n; i++ {
		tiles[i] = binary.LittleEndian.Uint16(p1[i*2 : i*2+2])
	}
	m.Tiles = tiles

	m.Altitudes = make([]uint8, n)
	copy(m.Altitudes, p2)
	// make zero-fills, so the manufactured plane is the copy not happening.
	m.Overlay = make([]uint8, n)
	if has3 {
		copy(m.Overlay, p3)
	}
	return nil
}

// decodeObjects walks the type4 placed-object records: #type4 base records of 20
// bytes, each followed by an 8-byte extension iff its kind (+0x08) == 0x21
// (ALM-OBJ-034). The walk must consume the payload exactly. #type4 comes from the
// type-0 count and is only cross-checked by that exact consumption, so objects
// grows by append (never a make sized from an untrusted count) — a hostile count
// is an atomic error, not an oversized allocation.
func (m *Map) decodeObjects(p []byte, d dialect) error {
	n := int(m.Meta.Count4)
	var objects []Object
	cur := 0
	for i := 0; i < n; i++ {
		if cur+objectRecordSize > len(p) {
			return fmt.Errorf("alm: type4 record %d does not fit in payload (%d bytes)", i, len(p))
		}
		rec := p[cur : cur+objectRecordSize]
		obj := Object{
			X:       binary.LittleEndian.Uint32(rec[0x00:0x04]),
			Y:       binary.LittleEndian.Uint32(rec[0x04:0x08]),
			Kind:    binary.LittleEndian.Uint32(rec[0x08:0x0c]),
			Field0C: binary.LittleEndian.Uint16(rec[0x0c:0x0e]),
			Field0E: binary.LittleEndian.Uint32(rec[0x0e:0x12]),
			Field12: binary.LittleEndian.Uint16(rec[0x12:0x14]),
		}
		cur += objectRecordSize

		if obj.Kind == objectExtKind || (d == dialectROM2 && obj.Kind&rom2ObjectExt != 0) {
			if cur+objectExtSize > len(p) {
				return fmt.Errorf("alm: type4 record %d: kind==0x21 extension does not fit in payload", i)
			}
			ext := make([]byte, objectExtSize)
			copy(ext, p[cur:cur+objectExtSize])
			obj.Ext = ext
			cur += objectExtSize
		}
		objects = append(objects, obj)
	}
	if cur != len(p) {
		return fmt.Errorf("alm: type4 walk consumed %d of %d payload bytes", cur, len(p))
	}
	m.Objects = objects
	return nil
}

// decodeGroups decodes the type5 roster: #type5 records of 76 bytes each.
//
// The relation words are read in RECORD ORDER, k = 0 first, and land at the
// same index in the array.
func (m *Map) decodeGroups(p []byte) error {
	n := int(m.Meta.Count5)
	if uint64(len(p)) != uint64(groupRecordSize)*uint64(m.Meta.Count5) {
		return fmt.Errorf("alm: type5 payload is %d bytes, want %d (76*#type5)", len(p), int64(groupRecordSize)*int64(m.Meta.Count5))
	}
	groups := make([]Group, n)
	for i := 0; i < n; i++ {
		rec := p[i*groupRecordSize : (i+1)*groupRecordSize]
		g := Group{
			Color:       binary.LittleEndian.Uint32(rec[0:4]),
			Participant: binary.LittleEndian.Uint32(rec[4:8]),
			Scalar:      binary.LittleEndian.Uint32(rec[0x08:0x0c]),
			Name:        decodeASCII(rec[0x0c:]),
		}
		for k := 0; k < groupRelationLen; k++ {
			off := groupRelationOff + 2*k
			g.Relation[k] = binary.LittleEndian.Uint16(rec[off : off+2])
		}
		groups[i] = g
	}
	m.Groups = groups
	return nil
}

// decodeUnits decodes the type6 placed units: #type6 records of 70 bytes each.
// Each record yields X/Y, the two class keys, the two words that condition the
// engine's override paths, the OWNER SLOT and the two identifier words a type-7
// script names — all raw, nothing resolved; the rest of the published read map is
// still not exposed, on the same rule that kept the identifier words out until
// script.go gave them a consumer (DD12). The primary key is read sign-extended.
//
// This function is the whole of the owner's arrival. The map DOCUMENT keeps its
// own payload bytes and writes them back untouched, so a map opened and written
// back is byte-identical whether or not this field is read out of it — decoding
// one more word off a record changes nothing about the record.
func (m *Map) decodeUnits(p []byte) error {
	n := int(m.Meta.Count6)
	if uint64(len(p)) != uint64(unitRecordSize)*uint64(m.Meta.Count6) {
		return fmt.Errorf("alm: type6 payload is %d bytes, want %d (70*#type6)", len(p), int64(unitRecordSize)*int64(m.Meta.Count6))
	}
	units := make([]Unit, n)
	for i := 0; i < n; i++ {
		rec := p[i*unitRecordSize : (i+1)*unitRecordSize]
		currentHP := int16(binary.LittleEndian.Uint16(rec[0x20:0x22]))
		units[i] = Unit{
			X: binary.LittleEndian.Uint32(rec[0x00:0x04]),
			Y: binary.LittleEndian.Uint32(rec[0x04:0x08]),
			// int16 conversion of the u16 is the loader's MOVSX (ALM-CLS-038).
			ClassID:      int16(binary.LittleEndian.Uint16(rec[0x08:0x0a])),
			ClassSubID:   binary.LittleEndian.Uint16(rec[0x0a:0x0c]),
			Flags:        binary.LittleEndian.Uint32(rec[0x0c:0x10]),
			DefID:        binary.LittleEndian.Uint32(rec[0x10:0x14]),
			Owner:        binary.LittleEndian.Uint32(rec[0x14:0x18]),
			CurrentHP:    currentHP,
			HasCurrentHP: currentHP != -1,
			UnitID:       binary.LittleEndian.Uint16(rec[0x40:0x42]),
			GroupID:      binary.LittleEndian.Uint32(rec[0x42:0x46]),
		}
	}
	m.Units = units
	m.AuthoredUnits = slices.Clone(units)
	return nil
}

// decodeTriggers decodes the type7/type9 leading count words (at payload+0) and
// preserves the type7/type8/type9 bodies raw. A record that is PRESENT must
// satisfy its constraint — type7 and type9 large enough to hold their 4-byte
// count word — while an absent one yields a zero count and an empty body, which
// is the "skipped" arm of ALM-REQ-056 and not an error. type8 needs no flag: its
// constraint is that it has none, so an absent record and an empty one decode
// alike (Present tells them apart).
func (m *Map) decodeTriggers(p7, p8, p9 []byte, has7, has9 bool) error {
	m.Triggers = Triggers{Body: cloneBytes(nil)}
	if has7 {
		if len(p7) < countWordSize {
			return fmt.Errorf("alm: type7 payload %d bytes too small for the entryCount word", len(p7))
		}
		m.Triggers = Triggers{
			EntryCount: binary.LittleEndian.Uint32(p7[0:countWordSize]),
			Body:       cloneBytes(p7[countWordSize:]),
		}
	}

	m.LootSection = LootSection{Body: cloneBytes(p8)}

	m.TileMarkers = TileMarkers{Body: cloneBytes(nil)}
	if has9 {
		if len(p9) < countWordSize {
			return fmt.Errorf("alm: type9 payload %d bytes too small for the count word", len(p9))
		}
		m.TileMarkers = TileMarkers{
			Count: binary.LittleEndian.Uint32(p9[0:countWordSize]),
			Body:  cloneBytes(p9[countWordSize:]),
		}
	}
	return nil
}

// cloneBytes returns a fresh copy of b so a decoded Map never aliases the input
// buffer. An empty (or nil) input yields a non-nil zero-length slice.
func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// cstr returns the bytes of b up to the first NUL (or all of b if none).
func cstr(b []byte) []byte {
	if i := bytes.IndexByte(b, 0x00); i >= 0 {
		return b[:i]
	}
	return b
}

// decodeASCII decodes a NUL-terminated ASCII field to a Go string. ASCII bytes
// are valid UTF-8, so no code-page mapping is applied.
func decodeASCII(b []byte) string {
	return string(cstr(b))
}

// decodeCP1251 decodes a NUL-terminated Windows-1251 field to UTF-8. CP1251
// is an ASCII superset, so an ASCII description decodes unchanged.
func decodeCP1251(b []byte) string {
	s, err := charmap.Windows1251.NewDecoder().Bytes(cstr(b))
	if err != nil {
		return string(cstr(b))
	}
	return string(s)
}
