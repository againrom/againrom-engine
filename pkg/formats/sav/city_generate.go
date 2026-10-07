package sav

import "encoding/binary"

// GeneratedCityMarker is the fixed sentinel serializeCityDocument requires
// (SAV-OBJ-014 and the enforcement at archive_document.go/city_semantic.go).
// A generator has no source byte to copy it from, so it is named here once
// rather than repeated as an unexplained literal at every call site.
const GeneratedCityMarker = uint32(0xbadface1)

// generatedCityTrailerLen is the fixed trailer-state length
// serializeCityDocument requires. Its content is not established for a
// document that never held original session bytes (DIV-878): a generator
// writes it zero-filled.
const generatedCityTrailerLen = 400

// NewCityStateData builds the complete, exact-shape city state directory (the
// small embedded "&YA1" store) for a document with no original session state
// to carry forward. parseCityState/parseStateStore require exactly the
// directories and leaves cityStateShape declares (SAV-CITYSTATE claims); this
// mirrors that shape field-for-field, the same construction
// TestCityProvenanceStructurallyRoutesIdentityKeyedRosterAndRemints already
// proves valid, generalized from a fixed test roster to any hero name.
//
// Every leaf takes its typed zero value except the hero's display name and
// the spellbook-shortcut bytes (DIV-878). Shortcuts is four little-endian
// int32 quick-spell indices (quickSpellsFromOriginalIndices,
// pkg/game/quickspells.go); -1 is its own "no spell bound" value, and 0 is a
// live index into originalBookIDs, so the four words are filled -1, not left
// at the zero the surrounding block otherwise takes. A fresh game's own state
// store carries no more than an empty bound set either.
func NewCityStateData(heroName string) CityStateData {
	out := CityStateData{RootKind: 17}
	for _, directory := range cityStateShape {
		out.DirectoryRecords = append(out.DirectoryRecords, CityStateDirectoryData{Path: "/" + directory.directory, Kind: 1})
		for _, child := range directory.children {
			path := "/" + directory.directory + "/" + child.name
			v := CityStateValueData{Kind: child.kind}
			switch path {
			case "/Character/Name":
				v.Bytes = append([]byte(heroName), 0)
			case "/SpellBook/Shortcuts":
				v.Bytes = make([]byte, 16)
				for i := 0; i < 4; i++ {
					binary.LittleEndian.PutUint32(v.Bytes[4*i:], 0xffffffff)
				}
			}
			out.ValueRecords = append(out.ValueRecords, CityStateRecordData{Path: path, Value: v})
		}
	}
	return out
}
