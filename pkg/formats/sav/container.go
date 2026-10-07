package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/reg"
)

// Magic is the four bytes at offset 0 of every save the original writes.
const Magic = "Asg&"

// MinVersion is the version dword the original's reader requires AT LEAST.
// A lower value makes it report "Outdated save file.", so this package refuses
// the same files it does.
const MinVersion = 0x0BAD0002

// headerLen is the container header; labelLen is the fixed buffer the save's
// human-readable slot text lives in, immediately after the blob.
const (
	headerLen = 0x10
	labelLen  = 0x100
)

// File is one save: the container's own fields, the decoded stream, the label
// region and the tail, plus the index Open built over the stream.
//
// Body IS THE AUTHORITY. Every decoded value below carries the body offset it
// was read at, and every edit writes back through that offset — so a byte this
// package does not understand is carried across a read/write cycle because it
// was never moved, not because something remembered it.
type File struct {
	// Version is the container's version dword, written back as it was read.
	Version uint32

	// Body is the whole decoded stream.
	Body []byte

	// Label is the save's slot text, read to the first NUL of the label
	// region. It is 8-BIT TEXT AND NOT ASCII — one owner-typed label in the
	// shipped corpus opens with a Cyrillic byte — so it is bytes.
	Label []byte

	// LabelRegion is the whole 0x100-byte buffer, carried verbatim. The
	// original overwrites it IN PLACE AND NEVER CLEARS IT, so a short label
	// is followed by the tail of whatever was longer before it; keeping the
	// region is what lets a rewrite leave that debris exactly as it was.
	LabelRegion []byte

	// Store is the embedded state store's own bytes, from the label region's
	// end to THE EXTENT THE STORE'S OWN FRAMING DECLARES, and TailRest is
	// everything after it. It is nil where the tail does not frame as a
	// store, and TailRest is then the whole tail.
	//
	// THE TAIL IS THREE REGIONS AND NOT TWO (SAV-TAILEXT-062): the label
	// region, the store, and a further 268 bytes on a mid-mission save or 310
	// on a between-mission one. This field was one slice named Tail until
	// 0150, documented as carried verbatim and never opened, and a reader
	// that parsed it to EOF would take that further region as the store's
	// heap. Store+TailRest is byte-for-byte what Tail held, which is what
	// keeps Marshal's round trip exact.
	//
	// TailRest is the campaign record that follows the state store. It remains
	// byte-preserved for exact round trips, while Campaign exposes a detached
	// typed projection of the decoded fields (SAV-CAMPAIGN-076).
	Store, TailRest []byte

	// store is Store parsed, or nil. Open parses once; every accessor reads
	// this. A file whose tail does not frame as a store has none, and that is
	// not an error — see Open.
	store *reg.Reg

	// campaign is parsed from TailRest when the state store frames correctly.
	// The bytes remain authoritative; callers receive a deep copy through
	// Campaign so they cannot mutate the File through the projection.
	campaign    *CampaignProjection
	campaignErr error

	// Head is the campaign half's fixed head.
	Head Head

	// Players is the distinct non-null roster, in first-reference order.
	Players []Player

	// World is the world half, or nil where the save has none. A save taken
	// between missions has none; see Head.Mission.
	World *WorldHalf

	// Actors contains Unit-derived records in the Player owner graph.
	Actors []Actor

	// Trailer is world+0x118's 400-byte block, present at the end of every
	// document body regardless of World (SAV-790).
	Trailer TrailerBody

	// TrailerOff locates the first of the 100 trailer dwords in Body. The
	// archive walk accounts for the optional marker global and the final
	// alignment byte; len(Body)-400 is not a valid locator (SAV-790,
	// SAV-DECPAD-238). Re-indexed by Open and structural setters.
	TrailerOff int
}

// Open decodes a whole save file.
//
// It refuses a file whose magic, version or blob extents are wrong, naming which
// — the four checks are separate because a caller handed the wrong file, an
// older file and a truncated file has three different problems.
func Open(b []byte) (*File, error) {
	if len(b) < headerLen {
		return nil, fmt.Errorf("sav: %d bytes, shorter than the %d-byte header", len(b), headerLen)
	}
	if string(b[:4]) != Magic {
		return nil, fmt.Errorf("sav: magic is % x, want %q", b[:4], Magic)
	}
	blobEnd := binary.LittleEndian.Uint32(b[4:])
	version := binary.LittleEndian.Uint32(b[8:])
	blobBytes := binary.LittleEndian.Uint32(b[12:])
	if version < MinVersion {
		return nil, fmt.Errorf("sav: version %#08x is older than %#08x", version, MinVersion)
	}
	if blobEnd < headerLen+4 || int64(blobEnd) > int64(len(b)) {
		return nil, fmt.Errorf("sav: blob ends at %d, outside a %d-byte file", blobEnd, len(b))
	}
	if blobBytes != blobEnd-headerLen {
		return nil, fmt.Errorf("sav: blob length %d disagrees with its end %d", blobBytes, blobEnd)
	}
	if int64(blobEnd)+labelLen > int64(len(b)) {
		return nil, fmt.Errorf("sav: %d bytes leaves no room for the %d-byte label region after %d",
			len(b), labelLen, blobEnd)
	}
	body, err := Decompress(b[headerLen:blobEnd])
	if err != nil {
		return nil, err
	}
	f := &File{
		Version:     version,
		Body:        body,
		LabelRegion: append([]byte(nil), b[blobEnd:blobEnd+labelLen]...),
	}
	f.splitTail(b[blobEnd+labelLen:])
	f.Label = labelOf(f.LabelRegion)
	if err := f.index(); err != nil {
		return nil, err
	}
	return f, nil
}

// splitTail divides the uncompressed tail into the state store and whatever
// follows it, and parses the store (SAV-TAILEXT-062).
//
// THE SPLIT IS AT THE STORE'S OWN DECLARED EXTENT, which reg.Size reads out of
// the store's header, and NOT at the end of the tail. The difference is 268
// bytes on a mid-mission save and 310 on a between-mission one, and a reader
// that took the whole tail would read those bytes as the store's heap.
//
// A TAIL THAT DOES NOT FRAME AS A STORE IS CARRIED WHOLE AND IS NOT AN ERROR.
// A save is evidence, and a file this package opened before the tail was split
// must still open after it: refusing one here would trade a section that file
// does not have for the whole of the file that it does.
func (f *File) splitTail(tail []byte) {
	n, err := reg.Size(tail)
	if err != nil {
		f.Store, f.TailRest = nil, append([]byte(nil), tail...)
		return
	}
	parsed, err := reg.Parse(tail[:n])
	if err != nil {
		f.Store, f.TailRest = nil, append([]byte(nil), tail...)
		return
	}
	f.Store = append([]byte(nil), tail[:n]...)
	f.TailRest = append([]byte(nil), tail[n:]...)
	f.store = parsed
	if len(f.TailRest) != 0 {
		campaign, err := parseCampaignProjection(f.TailRest)
		if err != nil {
			f.campaignErr = err
		} else {
			f.campaign = &campaign
		}
	}
}

// Store reports the parsed state store, and false where the tail did not frame
// as one. The tree is the parser's own and a caller must not mutate it.
func (f *File) StateStore() (*reg.Reg, bool) { return f.store, f.store != nil }

// labelOf reads the region to its first NUL. A region with no NUL at all is the
// whole region, which is what the original's own strlen would do to it.
func labelOf(region []byte) []byte {
	if i := bytes.IndexByte(region, 0); i >= 0 {
		return append([]byte(nil), region[:i]...)
	}
	return append([]byte(nil), region...)
}

// index builds every offset the rest of the package edits through.
func (f *File) index() error {
	if err := f.readHead(); err != nil {
		return err
	}
	doc, _, err := f.exactDocument()
	if err != nil {
		return err
	}
	f.Players = nil
	for _, record := range doc.players {
		player, ok := f.playerAt(record.Off)
		if !ok {
			return fmt.Errorf("sav: invalid Player at %d", record.Off)
		}
		f.Players = append(f.Players, player)
	}
	f.World = doc.world
	f.Trailer = doc.trailer
	f.TrailerOff = doc.trailerOff
	f.Actors, err = ownerActors(doc.players, doc.dead)
	if err != nil {
		return err
	}
	return nil
}

// Marshal answers the whole file: the header, the body RE-COMPRESSED, the label
// region and the tail.
//
// It re-compresses rather than carrying the blob it read, and that is the point:
// carrying it would make a round trip prove nothing about the encoder, and the
// encoder is the half of this package that cannot be checked any other way.
//
// blobEnd and blobBytes are RECOMPUTED. The original patches them in after the
// write, so a writer that carried the values it read would be right only for as
// long as nothing changed length.
func (f *File) Marshal() []byte {
	blob := Compress(f.Body)
	region := f.LabelRegion
	if len(region) != labelLen {
		grown := make([]byte, labelLen)
		copy(grown, region)
		region = grown
	}
	out := make([]byte, headerLen, headerLen+len(blob)+labelLen+len(f.Store)+len(f.TailRest))
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], f.Version)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	out = append(out, region...)
	// THE TWO TAIL REGIONS GO BACK IN ORDER, and their concatenation is the
	// one slice this file carried before 0150 split it: the split moved no
	// byte, so a re-emit is byte-identical for the same reason it was.
	out = append(out, f.Store...)
	out = append(out, f.TailRest...)
	return out
}

// SetLabel replaces the slot text, writing the new bytes and their NUL and
// LEAVING THE REST OF THE REGION EXACTLY AS IT WAS — which is what the original
// does, and is why a corpus label reads "1" followed by the tail of the default
// text it was typed over. Clearing the remainder would produce a file the game
// reads identically and that differs from every one it writes.
func (f *File) SetLabel(text []byte) error {
	if len(text)+1 > labelLen {
		return fmt.Errorf("sav: label of %d bytes does not fit the %d-byte region", len(text), labelLen)
	}
	if len(f.LabelRegion) != labelLen {
		grown := make([]byte, labelLen)
		copy(grown, f.LabelRegion)
		f.LabelRegion = grown
	}
	copy(f.LabelRegion, text)
	f.LabelRegion[len(text)] = 0
	f.Label = append([]byte(nil), text...)
	return nil
}

// u16 and u32 read the body. Every caller has already bounds-checked its span;
// they are here so the little-endian choice is written once.
func u16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func u32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
