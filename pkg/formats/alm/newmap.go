package alm

import (
	"encoding/binary"
	"fmt"
)

// Bounds of a new map's sides: the original editor's dialog offers 32 to 256
// in steps of 8 (ALM-NEWMAP-237). A side of 16 or less is border ring
// throughout (TERR-PLACE-208), so no unit could stand on it.
const (
	NewMapMinSide  = 32
	NewMapMaxSide  = 256
	NewMapSideStep = 8

	// NewMapDefaultSide is the side of the default map (ALM-NEWMAP-237).
	NewMapDefaultSide = 80

	// NewMapMaxAltitude is the top of the original new-map dialog's height
	// slider, 0..126 (TERR-246). It is the dialog's limit, not the format's:
	// MaxHeight is the range the height tool and the maps hold.
	NewMapMaxAltitude = 126
)

// NewMapSpec chooses the size and the fills of a new map. The zero value is
// not valid; DefaultNewMap returns the defaults.
type NewMapSpec struct {
	Width, Height int
	Tile          uint16 // the word every tile cell holds
	Altitude      uint8  // the byte every height cell holds
}

// DefaultNewMap is an 80 x 80 map of Grass at height 64 (ALM-NEWMAP-237,
// TERR-246).
func DefaultNewMap() NewMapSpec {
	return NewMapSpec{Width: NewMapDefaultSide, Height: NewMapDefaultSide, Tile: 0x11, Altitude: 64}
}

// newMapWord is one defaulted type-0 word.
type newMapWord struct {
	off    int
	value  uint32
	source string
}

// newMapMeta lists the type-0 words a new map carries other than the sides,
// the group mask and the zero words. +0x18 is recomputed from the tile plane
// (ALM-SAVE-238); +0x20, +0x24, +0x28 and +0x2c are 0 (ALM-NEWMAP-236); the
// bytes past the name and the 512 bytes at +0x78 are zero (DIV-3250).
var newMapMeta = []newMapWord{
	{metaAngle, 0x3f490fda, "ALM-NEWMAP-236"},
	{metaWord0C, 360, "ALM-NEWMAP-236, ALM-METADEFAULT-186"},
	{metaWord10, 21, "ALM-NEWMAP-236"},
	{metaWord14, 63, "ALM-NEWMAP-236"},
	{metaCount5, 2, "ALM-NEWMAP-236"},
	{metaWord70, 4, "ALM-NEWMAP-236"},
	{metaWord74, 1, "ALM-NEWMAP-236"},
}

// newMapName is the type-0 name of a new map (ALM-NEWMAP-236).
const newMapName = "Multiplayer Map"

// newMapPlayer is one roster record in the writer's order: the id word at
// +0x00, the side word at +0x04, the scalar at +0x08 and the name at +0x0c
// (ALM-NEWMAP-258).
type newMapPlayer struct {
	id, side, scalar uint32
	name             string
}

// newMapRoster is the two-player roster: the original editor's player
// constructor takes (1, 1) and (2, 2), the names are Self and Enemy, the side
// words 0 and 1 and the scalars 0 (ALM-NEWMAP-258).
var newMapRoster = []newMapPlayer{
	{1, 0, 0, "Self"},
	{2, 1, 0, "Enemy"},
}

// Relation words of a roster row: 2 on the record's own slot and 1 on each of
// the other fifteen (ALM-NEWMAP-258).
const (
	newMapRelationSelf  = 2
	newMapRelationOther = 1
)

// newMapRecords is the record order of a new map: the original writer's
// 0, 1, 2, 3, 5, 9, 8, 7 (ALM-NEWMAP-237) with the empty type-4 and type-6
// records the installed order places after type 5 and type 8 (DIV-3207).
var newMapRecords = [...]uint32{0, 1, 2, 3, 5, 4, 9, 8, 6, 7}

// Sizes of the fixed payloads (ALM-NEWMAP-237).
const (
	newMapType9Size = 4  // a zero count
	newMapType7Size = 12 // three zero counts
	newMapVersion   = 990
)

// Validate reports why the spec cannot build a map.
func (s NewMapSpec) Validate() error {
	for _, side := range []struct {
		name string
		v    int
	}{{"width", s.Width}, {"height", s.Height}} {
		if side.v < NewMapMinSide || side.v > NewMapMaxSide || side.v%NewMapSideStep != 0 {
			return fmt.Errorf("alm: new map %s %d is not %d to %d in steps of %d",
				side.name, side.v, NewMapMinSide, NewMapMaxSide, NewMapSideStep)
		}
	}
	if s.Tile&^tileSaveMask != 0 {
		return fmt.Errorf("alm: new map tile word %#x holds bits the save clears", s.Tile)
	}
	if s.Altitude > NewMapMaxAltitude {
		return fmt.Errorf("alm: new map altitude %d is above %d", s.Altitude, NewMapMaxAltitude)
	}
	return nil
}

// tileSaveMask is the tile word bits the original save keeps (ALM-SAVE-238).
const tileSaveMask = 0x23ff

// NewMap is the one constructor of a map from nothing. It returns a Document
// opened through the same route as any other stream: the bytes are built from
// the defaults tables above and then accepted by OpenDocument, so the model
// edits the result like an installed map and no second map model exists.
//
// Every default is a row of data naming what states it: a claim, or a
// divergence row where the claims leave the value open.
func NewMap(spec NewMapSpec) (*Document, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	cells := spec.Width * spec.Height

	meta := make([]byte, metaSize)
	le := binary.LittleEndian
	le.PutUint32(meta[metaW:], uint32(spec.Width))
	le.PutUint32(meta[metaH:], uint32(spec.Height))
	for _, w := range newMapMeta {
		le.PutUint32(meta[w.off:], w.value)
	}
	le.PutUint32(meta[metaBitmask:], TileGroupMask([]uint16{spec.Tile}))
	name, err := EncodeName(newMapName)
	if err != nil {
		return nil, err
	}
	copy(meta[metaName:], name)

	tiles := make([]byte, 2*cells)
	for i := 0; i < cells; i++ {
		le.PutUint16(tiles[2*i:], spec.Tile)
	}
	heights := make([]byte, cells)
	for i := range heights {
		heights[i] = spec.Altitude
	}

	roster := make([]byte, 0, groupRecordSize*len(newMapRoster))
	for i, p := range newMapRoster {
		rec := make([]byte, groupRecordSize)
		le.PutUint32(rec[0:], p.id)
		le.PutUint32(rec[4:], p.side)
		le.PutUint32(rec[8:], p.scalar)
		copy(rec[0x0c:0x2c], p.name)
		for k := 0; k < groupRelationLen; k++ {
			word := uint16(newMapRelationOther)
			if k == i {
				word = newMapRelationSelf
			}
			le.PutUint16(rec[groupRelationOff+2*k:], word)
		}
		roster = append(roster, rec...)
	}

	payload := map[uint32][]byte{
		0: meta, 1: tiles, 2: heights, 3: make([]byte, cells), 5: roster,
		4: nil, 6: nil, 8: nil,
		9: make([]byte, newMapType9Size), 7: make([]byte, newMapType7Size),
	}
	out := make([]byte, fileHeaderSize)
	le.PutUint32(out[fhMagic:], fileMagic)
	le.PutUint32(out[fhHdrLen:], fileHdrLen)
	le.PutUint32(out[fhDataSize:], uint32(4*cells+72))
	le.PutUint32(out[fhRecordCount:], uint32(len(newMapRecords)))
	le.PutUint32(out[fhFormatVersion:], newMapVersion)
	for _, id := range newMapRecords {
		hdr := make([]byte, recordHeaderSize)
		le.PutUint32(hdr[recTag:], recordTag)
		le.PutUint32(hdr[recHdrLen:], recordHdrLen)
		le.PutUint32(hdr[recPayloadSize:], uint32(len(payload[id])))
		le.PutUint32(hdr[recTypeID:], id)
		out = append(append(out, hdr...), payload[id]...)
	}
	return OpenDocument(out)
}
