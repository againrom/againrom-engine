package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
)

// The definition entry's four parameter positions, and how wide a row has to be
// before all four can be read (spec, "The footprint contract").
//
// They are POSITIONS in the entry's own parameter array and not slot names of
// ours: the collection ships its own column titles for them, and the two the
// pass reads are titled the inverse of what they do — see blockingSlot.
const (
	// sizeXSlot and sizeYSlot are the footprint's width and height in cells.
	sizeXSlot = 0
	sizeYSlot = 1

	// healthMaxSlot is the position `ALM-CLS-053` names as the one the type-4
	// spawn writes into `obj+0x42`/`+0x44`. It is read by the Field42 seed
	// (fromalm.go, Structures) and by nothing in the footprint pass.
	healthMaxSlot = 3

	// blockingSlot is the BLOCKING set. The column's shipped title says
	// passability, and a set bit makes its cell impassable, so the title is the
	// inverse of the content. It is named for the content here and nowhere for
	// the title, which is the whole defence: a builder cannot reach for the
	// wrong sense of a word that is not written down.
	blockingSlot = 4

	// attachSlot is the ATTACH set: which cells of the rectangle the structure
	// occupies at all.
	attachSlot = 5

	// entryParams is how many parameters a row must carry before the four
	// positions above can all be read. A shorter row is a skip, never a panic
	// and never a failure.
	entryParams = attachSlot + 1
)

// extensionKind is the class key whose record carries an eight-byte extension.
//
// It is compared against the WHOLE key and never its low byte, which is the
// opposite of how the same field selects a definition entry: two different
// questions asked of one field, and a key of 0x121 answers them differently —
// it resolves the entry 0x21 names and carries no extension at all.
const extensionKind = 0x21

// The two bytes of an extension that decide anything: the low byte of its first
// four is a width and the low byte of its second four a height. The other six
// bytes reach nothing.
const (
	extWidthByte  = 0
	extHeightByte = 4
	extSize       = 8
)

// footprintBits is how many cells one set can name. A rectangle longer than this
// ALIASES — cell 32 shares cell 0's bit, in the attach set and the blocking set
// alike — and that is the rule rather than a defect: the original masks its
// shift count to five bits and the shipped table contains a rectangle that
// exercises it.
const footprintBits = 32

// Footprint is one placement resolved against a definition table: where its
// rectangle starts, how big it is, and the two 32-bit sets that say which of its
// cells it occupies and which of those it closes.
//
// Col/Row is the ANCHOR CELL and the rectangle's top-left, so the walk runs
// right and down from it. Width and Height are each at least 1 and at most 255,
// because each is the low byte of its parameter and a zero on either axis yields
// no footprint at all.
//
// Blocking is named for what it DOES and never for the column title it comes
// from, which says the reverse; see blockingSlot.
type Footprint struct {
	Col, Row      int
	Width, Height int

	// Blocking closes a cell to a ground mover where its bit is SET and opens
	// one where the bit is CLEAR.
	Blocking uint32

	// Attach names the cells of the rectangle the structure occupies. A cell
	// this does not name is not touched, not counted and not read.
	Attach uint32
}

// bit reports whether the set names the cell at offset (dx, dy) of a footprint
// this wide.
//
// The index runs continuously across the rectangle, row by row, and is taken
// MODULO 32 — the aliasing footprintBits describes. The modulus is applied to
// the index rather than the shift count, so nothing here depends on how a shift
// wider than the operand behaves; Go's shift is not the original's masked one,
// and a build that leaned on that would be reading a different rule that happens
// to agree.
func bit(set uint32, dx, dy, width int) bool {
	return set>>((dy*width+dx)%footprintBits)&1 == 1
}

type StructureCounts struct {
	// Per placement. Unresolved is a key naming no entry, Short an entry
	// carrying fewer parameters than the contract reads, ZeroExtent a width or
	// height of zero from either source.
	Resolved, Unresolved, Short, ZeroExtent int

	// Per cell. Dropped is outside the map extent, Refused is a cell an earlier
	// placement already took, and Abandoned is attach-named but never walked —
	// a refusal having ended that footprint.
	Attached, Closed, Opened, Dropped, Refused, Abandoned int
}

// buildings is the collection a placement resolves against, or nil for a table
// that carries none. A nil Table and a nil field are both "no table", exactly as
// they are for the two collections a unit placement searches.
func (t *Table) buildings() data.Collection {
	if t == nil {
		return nil
	}
	return t.Buildings
}

// Footprints resolves a map's placements against a definition table: one
// footprint per placement that resolves an entry, in the order the map records
// them, and the placement-level counters filled.
//
// It is PURE AND TOTAL. Neither the map nor the collection is written through —
// the parameter array is read and copied out of, never retained — and no shape
// of either yields an error: a key naming no entry, an entry too short to read
// and an extent of zero are each a skip counted under its own name. So any map
// and any table yield a list, and the same pair always yields the same list.
//
// The cell-level counters come back zero. Whether a footprint's cells attach is
// a property of the whole pass — of every earlier placement and of the map's
// extent — and not of one resolution, so this function has nothing to say about
// them and says nothing rather than a partial number.
func Footprints(m *alm.Map, t *Table) ([]Footprint, StructureCounts) {
	var c StructureCounts
	if m == nil {
		return nil, c
	}
	c1 := t.buildings()
	out := make([]Footprint, 0, len(m.Objects))
	for _, o := range m.Objects {
		f, ok := resolve(o, c1, &c)
		if !ok {
			continue
		}
		c.Resolved++
		out = append(out, f)
	}
	return out, c
}

// buildingParams selects the definition entry a placement's class key names and
// returns that entry's parameter array.
//
// It is the ONE reading of the selection rule in this package, shared by the
// footprint pass and by the Field42 seed (fromalm.go, Structures) rather
// than written twice in terms that agree today. The rule is `ALM-CLS-053` and
// `DAT-BLD-005`: the entry index IS the class key, one-based because the
// collection skips entry 0, bounded above by the collection's own count.
//
// The key's LOW BYTE selects, so a key above 255 resolves the entry that byte
// names. The guard covers a nil interface, which is what an absent Buildings
// field is, and the collection's own length. It does NOT cover a TYPED nil — a
// (*databin.Collection)(nil) in that field is a non-nil interface and would
// panic inside Len() — which is why no caller in this tree ever builds one:
// each assigns a collection an archive yielded.
//
// The width of the returned array is NOT tested here. What a row must carry is a
// property of what the caller reads, and the two callers read different
// positions, so each applies its own test and reports its own miss.
func buildingParams(o alm.Object, c data.Collection) ([]int32, bool) {
	k := int(uint8(o.Kind))
	if c == nil || k < 1 || k >= c.Len() {
		return nil, false
	}
	return c.EntryParams(k), true
}

// resolve is one placement against one collection, and the ORDER of its five
// steps is contract rather than detail.
//
// The entry is selected by the key's LOW BYTE, so a key above 255 resolves the
// entry that byte names; the extension is selected by the WHOLE key, so a key
// whose low byte is the extension kind but whose value is not carries none. The
// zero-extent test runs LAST because a zero can arrive from the entry or from
// the extension, and one test after the override covers both sources.
//
// The extension arm is gated on the SUM of the two extent bytes and not on the
// kind. A record of the extension kind carrying two zero bytes therefore falls
// through to its entry's own rectangle, sets and all — a case no shipped record
// exercises, which is why it can only come from reading the branch rather than
// from measuring the corpus.
func resolve(o alm.Object, c data.Collection, counts *StructureCounts) (Footprint, bool) {
	p, ok := buildingParams(o, c)
	if !ok {
		counts.Unresolved++
		return Footprint{}, false
	}
	if len(p) < entryParams {
		counts.Short++
		return Footprint{}, false
	}

	// Each extent is the LOW BYTE of its parameter, so neither can exceed 255;
	// each set is the parameter's own 32 bits reinterpreted, so a row's empty
	// cell — stored as -1 — reads as an all-ones set rather than as an error.
	f := Footprint{
		Col:      int(o.X >> 8),
		Row:      int(o.Y >> 8),
		Width:    int(uint8(p[sizeXSlot])),
		Height:   int(uint8(p[sizeYSlot])),
		Blocking: uint32(p[blockingSlot]),
		Attach:   uint32(p[attachSlot]),
	}

	if o.Kind == extensionKind && len(o.Ext) >= extSize {
		w, h := int(o.Ext[extWidthByte]), int(o.Ext[extHeightByte])
		if w+h > 0 {
			// The whole rectangle occupies and none of it closes: an
			// author-sized hole in the plane, whatever the entry says.
			f.Width, f.Height = w, h
			f.Attach, f.Blocking = ^uint32(0), 0
		}
	}

	if f.Width == 0 || f.Height == 0 {
		counts.ZeroExtent++
		return Footprint{}, false
	}
	return f, true
}

// PassabilityWith is the block plane a map describes when its placed structures
// are applied to it: the five terrain arms unioned, and then this pass on top.
//
// The two stages run in that order and are NEVER interleaved, which is what
// makes a structure able to subtract: an arm that ran afterwards would close a
// deck cell the structure had just opened. It is one byte per in-bounds cell,
// row-major, and a function of the map and the table alone.
//
// PassabilityWith(m, nil) is Passability(m), byte for byte and by construction
// rather than by agreement: the arms stage IS a call to Passability, and with no
// table the second stage resolves nothing and writes nowhere. So the pre-story
// plane is not reproduced here, it is reused, and no digest pinned against a
// world built the single-argument way can move.
func PassabilityWith(m *alm.Map, t *Table) []byte {
	plane, _ := derive(m, t)
	return plane
}

func StructureCensus(m *alm.Map, t *Table) StructureCounts {
	_, c := derive(m, t)
	return c
}

// derive is the whole of the second stage: the arms' plane, then every
// placement's footprint walked over it in the order the map records them.
//
// The OCCUPANCY is one bool per in-bounds cell, shared across every placement,
// and it is what makes "one structure per cell" a property of the walk rather
// than a search: a cell an earlier footprint took is refused, and a refusal ends
// the rest of THAT footprint while leaving the cells it already attached
// attached.
//
// It returns the plane and the census together because they come out of one
// walk. Neither exported entry point above hands both to a caller — a world's
// signature is not widened for an instrument — but they cannot be produced apart.
func derive(m *alm.Map, t *Table) ([]byte, StructureCounts) {
	plane := Passability(m)
	w, h, ok := extent(m)
	if !ok {
		// No cells: there is nothing for a footprint to attach to, so the
		// placements are not resolved either and every counter stays zero.
		return plane, StructureCounts{}
	}

	prints, c := Footprints(m, t)
	if len(prints) == 0 {
		// Nothing to walk, so no occupancy is allocated: the no-table path —
		// which is every world the game front-end builds — costs exactly what
		// it cost before this story.
		return plane, c
	}
	taken := make([]bool, w*h)
	for _, f := range prints {
		walk(f, plane, taken, w, h, &c)
	}
	return plane, c
}

// walk applies one footprint: every cell its ATTACH set names, row by row from
// the anchor, running right and down.
//
// A named cell ends exactly one of four ways, tested in this order — abandoned,
// dropped, refused, attached — and a cell the attach set does not name is
// counted nowhere and read nowhere. That is what makes the four a partition of
// the named cells rather than four overlapping tallies.
//
// The abandon flag is tested FIRST so that the remainder of a refused footprint
// is counted without being walked. Dropped does NOT set it: an overhanging row
// loses its overhang and the walk carries on into the next row, because a cell
// off the plane is a cell the original writes into unused columns and cannot
// observe, while a cell already occupied is a decision it makes.
//
// The bounds test comes BEFORE the index is formed, so no byte outside the
// extent is addressed at any anchor, extent or table.
func walk(f Footprint, plane []byte, taken []bool, w, h int, c *StructureCounts) {
	abandoned := false
	for dy := 0; dy < f.Height; dy++ {
		for dx := 0; dx < f.Width; dx++ {
			if !bit(f.Attach, dx, dy, f.Width) {
				continue
			}
			if abandoned {
				c.Abandoned++
				continue
			}
			x, y := f.Col+dx, f.Row+dy
			if x < 0 || y < 0 || x >= w || y >= h {
				c.Dropped++
				continue
			}
			i := y*w + x
			if taken[i] {
				c.Refused++
				abandoned = true
				continue
			}
			taken[i] = true
			c.Attached++

			// The pass itself, and the ONE line the polarity lives on: a SET
			// bit closes the cell to a ground mover, a CLEAR bit opens it —
			// overriding whatever the arms said. The air bit is not an operand
			// of either operator, so it comes out exactly as the arms left it,
			// and neither operator can reach a bit above bit 1.
			if bit(f.Blocking, dx, dy, f.Width) {
				plane[i] |= blockGround
				c.Closed++
			} else {
				plane[i] &^= blockGround
				c.Opened++
			}
		}
	}
}
