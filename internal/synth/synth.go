// Package synth builds synthetic game-format byte streams for tests.
//
// Golden rule 2: the test suite never reads a game install, so every fixture is
// a byte stream constructed here from the documented format contracts. This
// package exists because the same builders are now needed in four different test
// packages — and one of them, a 24-bpp Windows BMP writer, existed nowhere in the
// tree. Four hand-copied writers would be four chances to disagree about the same
// public format.
//
// It builds **inputs only**. No expected image, rectangle, ordering or count
// lives here, so a separate-context test that uses these builders still derives
// its oracle from the specification rather than from anything shared with the
// implementation.
//
// It lives under internal/ as a non-tier build/test helper: archtest does not
// police internal/, nothing outside a _test.go file imports it, and it therefore
// enters no shipped binary. It imports only the standard library — in particular
// it does not import pkg/render/menu, which is why the menu-asset builder takes
// the placement tables and the entry prefix as parameters rather than reading
// them from the package under test.
package synth

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"sort"
)

// ---------------------------------------------------------------------------
// .res archives (RES-HDR-002..RES-NODE-008; see docs/0001-res-archive/spec.md)
// ---------------------------------------------------------------------------

// File is one file to place in a synthetic archive. Path is slash-separated and
// may name directories, which are created implicitly.
type File struct {
	Path string
	Data []byte
}

const (
	resSignature  = 0x31415926
	resHeaderSize = 0x18
	resNodeSize   = 0x20
	resTypeFile   = 0
	resTypeDir    = 1
	resNameLen    = 16
)

type resNode struct {
	name     string
	dir      bool
	data     []byte
	children []*resNode
}

// Archive assembles a .res (&YA1) archive holding the given files, creating the
// directory nodes their paths imply.
//
// Node indices are assigned breadth-first, which is what makes each directory's
// children a contiguous index range — the layout the reader's [off, off+size)
// child range requires. Insertion order is preserved within each directory, so
// the archive's registry order is a caller-controlled, deterministic property a
// test can assert on.
func Archive(files []File) []byte {
	var roots []*resNode
	for _, f := range files {
		parts := splitPath(f.Path)
		if len(parts) == 0 {
			panic("synth: empty archive path")
		}
		level := &roots
		for i, name := range parts {
			if i == len(parts)-1 {
				*level = append(*level, &resNode{name: name, data: f.Data})
				break
			}
			dir := findDir(*level, name)
			if dir == nil {
				dir = &resNode{name: name, dir: true}
				*level = append(*level, dir)
			}
			level = &dir.children
		}
	}

	// Breadth-first index assignment: every node in `order` is emitted in this
	// order, and a directory's children occupy the contiguous block recorded in
	// firstChild.
	order := append([]*resNode(nil), roots...)
	firstChild := make(map[*resNode]int)
	for i := 0; i < len(order); i++ {
		n := order[i]
		if !n.dir {
			continue
		}
		firstChild[n] = len(order)
		order = append(order, n.children...)
	}

	out := make([]byte, resHeaderSize)
	offsets := make(map[*resNode][2]uint32, len(order))
	for _, n := range order {
		if n.dir {
			continue
		}
		offsets[n] = [2]uint32{uint32(len(out)), uint32(len(n.data))}
		out = append(out, n.data...)
	}
	regOffset := uint32(len(out))

	binary.LittleEndian.PutUint32(out[0x00:], resSignature)
	binary.LittleEndian.PutUint32(out[0x08:], 1)
	binary.LittleEndian.PutUint32(out[0x0c:], uint32(len(roots)))
	binary.LittleEndian.PutUint32(out[0x10:], regOffset)
	binary.LittleEndian.PutUint32(out[0x14:], uint32(len(order)))

	for _, n := range order {
		node := make([]byte, resNodeSize)
		if n.dir {
			binary.LittleEndian.PutUint32(node[0x04:], uint32(firstChild[n]))
			binary.LittleEndian.PutUint32(node[0x08:], uint32(len(n.children)))
			binary.LittleEndian.PutUint32(node[0x0c:], resTypeDir)
		} else {
			o := offsets[n]
			binary.LittleEndian.PutUint32(node[0x04:], o[0])
			binary.LittleEndian.PutUint32(node[0x08:], o[1])
			binary.LittleEndian.PutUint32(node[0x0c:], resTypeFile)
		}
		writeName(node[0x10:0x20], n.name)
		out = append(out, node...)
	}
	return out
}

// writeName lays a node name into its 16-byte field: the name, a NUL, then 0xCD
// padding — the shape the shipped archives use, so the reader's "cut at the first
// NUL and drop the padding" rule is genuinely exercised rather than trivially
// satisfied by a zero fill.
func writeName(dst []byte, name string) {
	if len(name) >= resNameLen {
		panic(fmt.Sprintf("synth: archive node name %q does not fit in %d bytes with a NUL", name, resNameLen))
	}
	n := copy(dst, name)
	dst[n] = 0x00
	for i := n + 1; i < len(dst); i++ {
		dst[i] = 0xCD
	}
}

func splitPath(p string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(p); i++ {
		if i == len(p) || p[i] == '/' {
			if i > start {
				parts = append(parts, p[start:i])
			}
			start = i + 1
		}
	}
	return parts
}

func findDir(nodes []*resNode, name string) *resNode {
	for _, n := range nodes {
		if n.dir && n.name == name {
			return n
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Windows BMP
// ---------------------------------------------------------------------------

const (
	bmpFileHeader = 14
	bmpInfoHeader = 40
	bmpPaletteLen = 256 * 4
)

// BMP24 encodes img as an uncompressed 24-bpp Windows BMP with a positive height,
// i.e. rows stored bottom-up — the form every shipped menu bitmap uses.
func BMP24(img image.Image) []byte { return bmp24(img, true) }

// BMP24TopDown encodes img with a negative height, i.e. rows stored top-down.
// The same pixels as BMP24 produce a stream whose stored row order is reversed,
// which is what lets a test show a decoder honours the sign rather than assuming
// one convention.
func BMP24TopDown(img image.Image) []byte { return bmp24(img, false) }

func bmp24(img image.Image, bottomUp bool) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stride := (w*3 + 3) &^ 3
	offBits := bmpFileHeader + bmpInfoHeader

	out := make([]byte, offBits+stride*h)
	writeBMPHeader(out, w, h, 24, offBits, bottomUp)
	for y := 0; y < h; y++ {
		row := out[offBits+storedRow(y, h, bottomUp)*stride:]
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			row[x*3+0] = byte(bl >> 8)
			row[x*3+1] = byte(g >> 8)
			row[x*3+2] = byte(r >> 8)
		}
	}
	return out
}

// BMP8 encodes img as an uncompressed 8-bpp Windows BMP with a positive height.
// The palette is written from img.Palette (padded to 256 entries) and the pixel
// bytes are img.Pix verbatim, so a caller keeps exact control of the raw index at
// every pixel — which is the whole point for a hit mask.
func BMP8(img *image.Paletted) []byte { return bmp8(img, true) }

// BMP8TopDown is BMP8 with a negative height and top-down stored rows.
func BMP8TopDown(img *image.Paletted) []byte { return bmp8(img, false) }

func bmp8(img *image.Paletted, bottomUp bool) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stride := (w + 3) &^ 3
	offBits := bmpFileHeader + bmpInfoHeader + bmpPaletteLen

	out := make([]byte, offBits+stride*h)
	writeBMPHeader(out, w, h, 8, offBits, bottomUp)

	for i := 0; i < 256; i++ {
		e := out[bmpFileHeader+bmpInfoHeader+i*4:]
		if i < len(img.Palette) {
			r, g, bl, _ := img.Palette[i].RGBA()
			e[0], e[1], e[2] = byte(bl>>8), byte(g>>8), byte(r>>8)
		}
	}
	for y := 0; y < h; y++ {
		row := out[offBits+storedRow(y, h, bottomUp)*stride:]
		copy(row[:w], img.Pix[y*img.Stride:y*img.Stride+w])
	}
	return out
}

func writeBMPHeader(out []byte, w, h, bpp, offBits int, bottomUp bool) {
	out[0x00], out[0x01] = 'B', 'M'
	binary.LittleEndian.PutUint32(out[0x02:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[0x0a:], uint32(offBits))
	binary.LittleEndian.PutUint32(out[0x0e:], bmpInfoHeader)
	binary.LittleEndian.PutUint32(out[0x12:], uint32(w))
	storedH := int32(h)
	if !bottomUp {
		storedH = -storedH
	}
	binary.LittleEndian.PutUint32(out[0x16:], uint32(storedH))
	binary.LittleEndian.PutUint16(out[0x1a:], 1)
	binary.LittleEndian.PutUint16(out[0x1c:], uint16(bpp))
}

// storedRow maps display row y to the row index it occupies in the stream.
func storedRow(y, h int, bottomUp bool) int {
	if bottomUp {
		return h - 1 - y
	}
	return y
}

// Trailing returns b with n extra zero bytes appended. Every bitmap in the
// shipped main.res carries exactly two bytes beyond the size its header implies,
// so a reader that requires the stream length to equal the computed minimum
// rejects all eighteen. Tests use this to keep that case honest.
func Trailing(b []byte, n int) []byte {
	return append(append([]byte(nil), b...), make([]byte, n)...)
}

// ---------------------------------------------------------------------------
// Menu asset sets
// ---------------------------------------------------------------------------

// MenuOptions parameterises a synthetic main-menu asset set. The placement
// tables and the entry prefix are supplied by the caller rather than read from
// the package under test, which keeps this package free of any intra-module
// import.
type MenuOptions struct {
	Prefix      string             // archive path prefix, e.g. "graphics/mainmenu/"
	Hover       [8]image.Rectangle // per-button hover placement
	Pressed     [8]image.Rectangle // per-button pressed placement
	MaskIndex   [8]byte            // per-button hot mask index
	BaseSize    image.Point        // defaults to 640x480
	MaskSize    image.Point        // defaults to 640x480
	MaskPalette color.Palette      // defaults to GrayRamp()
	OmitRegions []int              // 1-based buttons whose mask region is left empty
	Edits       map[string][]byte  // path -> replacement bytes; a nil value omits the entry
}

// MenuFiles returns the eighteen menu bitmaps of a synthetic asset set, keyed by
// archive path.
//
// The base is a horizontal gradient so a composited overlay is visibly distinct
// from it; the mask carries each button's hot index across that button's hover
// rectangle, on an identity grayscale ramp; each overlay is a flat colour unique
// to its button and state, sized exactly as its own placement row requires — so a
// composed frame identifies which bitmap was drawn where.
//
// Edits is applied last: a non-nil value replaces an entry's bytes, and a key
// present with a nil value omits the entry entirely, which is how a test builds
// the defective sets the error cases need.
func MenuFiles(o MenuOptions) map[string][]byte {
	baseSize := o.BaseSize
	if baseSize == (image.Point{}) {
		baseSize = image.Point{X: 640, Y: 480}
	}
	maskSize := o.MaskSize
	if maskSize == (image.Point{}) {
		maskSize = image.Point{X: 640, Y: 480}
	}

	files := map[string][]byte{
		o.Prefix + "menu_.bmp":    BMP24(gradient(baseSize.X, baseSize.Y)),
		o.Prefix + "menumask.bmp": BMP8(maskImage(maskSize.X, maskSize.Y, o)),
	}
	for i := 0; i < 8; i++ {
		h, p := o.Hover[i], o.Pressed[i]
		files[fmt.Sprintf("%sbutton%d.bmp", o.Prefix, i+1)] = BMP24(flat(h.Dx(), h.Dy(), uint8(0x10+i), 0x00, 0x00))
		files[fmt.Sprintf("%sbutton%dp.bmp", o.Prefix, i+1)] = BMP24(flat(p.Dx(), p.Dy(), 0x00, uint8(0x10+i), 0x00))
	}
	for path, data := range o.Edits {
		if data == nil {
			delete(files, path)
			continue
		}
		files[path] = data
	}
	return files
}

// MenuArchive packs MenuFiles into a .res archive.
func MenuArchive(o MenuOptions) []byte {
	files := MenuFiles(o)
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sortStrings(paths)

	entries := make([]File, 0, len(paths))
	for _, p := range paths {
		entries = append(entries, File{Path: p, Data: files[p]})
	}
	return Archive(entries)
}

func maskImage(w, h int, o MenuOptions) *image.Paletted {
	pal := o.MaskPalette
	if pal == nil {
		pal = GrayRamp()
	}
	img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	omitted := make(map[int]bool, len(o.OmitRegions))
	for _, b := range o.OmitRegions {
		omitted[b] = true
	}
	for i := 0; i < 8; i++ {
		if omitted[i+1] {
			continue
		}
		r := o.Hover[i].Intersect(image.Rect(0, 0, w, h))
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.Pix[y*img.Stride+x] = o.MaskIndex[i]
			}
		}
	}
	return img
}

// GrayRamp is the identity grayscale ramp palette[i] = (i,i,i) the shipped hit
// mask carries. A decoder that reads through the palette rather than the raw
// index looks correct against it, which is exactly why tests also build the
// all-black variant with BlackPalette.
func GrayRamp() color.Palette {
	p := make(color.Palette, 256)
	for i := range p {
		p[i] = color.RGBA{R: uint8(i), G: uint8(i), B: uint8(i), A: 0xff}
	}
	return p
}

// BlackPalette is 256 identical black entries. A mask encoded with it carries the
// same raw indices as one encoded with GrayRamp but no distinguishable colours,
// so a hit test that resolves indices through the palette cannot agree with one
// that reads them raw.
func BlackPalette() color.Palette {
	p := make(color.Palette, 256)
	for i := range p {
		p[i] = color.RGBA{A: 0xff}
	}
	return p
}

func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*img.Stride + x*4
			img.Pix[i+0] = byte(x)
			img.Pix[i+1] = byte(y)
			img.Pix[i+2] = 0x40
			img.Pix[i+3] = 0xff
		}
	}
	return img
}

func flat(w, h int, r, g, b uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 0xff
	}
	return img
}

func sortStrings(s []string) { sort.Strings(s) }

// ---------------------------------------------------------------------------
// .alm maps (ALM-FRAME-031 corrected framing; see docs/0003-alm-container/spec.md)
// ---------------------------------------------------------------------------

// ALMObject is one placed type-4 record. Every field is written verbatim at its
// file offset, Kind included: the low 16 bits of Kind are a placed class key and
// the high half is not decoded, so nothing here narrows or reinterprets it.
//
// Ext is the 8-byte kind==0x21 extension. The reader appends it exactly when Kind
// is 0x21, so this builder writes it on the same condition and from eight zero
// bytes when Ext is nil — a caller placing the class whose key is 0x21 does not
// have to know that its record grows. An Ext that cannot be written where the
// format puts it — a length other than 8, or any value on a record of another
// Kind — panics rather than emitting a stream no reader accepts.
type ALMObject struct {
	X, Y    uint32 // +0x00, +0x04 fixed-point (/256)
	Kind    uint32 // +0x08
	Field0C uint16 // +0x0c
	Field0E uint32 // +0x0e
	Field12 uint16 // +0x12
	Ext     []byte // +0x14, 8 bytes, iff Kind == 0x21
}

// ALMUnit is one placed type-6 record, laid into the 70-byte file record with the
// rest zero except current health. Its zero-value shape writes the format's -1
// derive-maximum sentinel, so adding a unit to an older fixture cannot silently
// turn it into an authored zero-health body.
//
// ClassID is int16 here because it is int16 in the file: the loader sign-extends
// the +0x08 word, so a negative key is a real value and not a large positive one.
// Keeping the builder's field at the file's own width means a test can write
// -32767 and get the two bytes that read back as -32767, with no widening or
// narrowing anywhere between the caller and the stream.
type ALMUnit struct {
	X, Y         uint32 // +0x00, +0x04 fixed-point (/256)
	ClassID      int16  // +0x08, sign-extended by the reader
	ClassSubID   uint16 // +0x0a
	Flags        uint32 // +0x0c
	DefID        uint32 // +0x10
	CurrentHP    int16  // +0x20 when HasCurrentHP
	HasCurrentHP bool   // false writes the -1 derive-maximum sentinel
}

// ALMOptions parameterises a synthetic map. The defaults produce a valid,
// fully-decodable map; the override fields exist so a test can construct a file
// that is well-framed but fails a later stage of the decode.
//
// Every field is optional and its zero value reproduces the payload this builder
// produced before that field existed, so a caller that does not name it is
// unaffected by its existence.
type ALMOptions struct {
	Width, Height int
	Name          string // ASCII map name at type-0 +0x30; empty is normal on campaign maps
	Description   string // ASCII/CP1251 description at type-0 +0x78

	// Type1Payload replaces the type-1 (tile grid) payload when non-nil. The
	// record still declares its own real length, so the file tiles to EOF and the
	// header walk and the type-0 decode both succeed — only the grid check fails.
	// This is the one shape that makes "the metadata read passed but the full
	// decode failed" reachable.
	Type1Payload []byte

	// Altitudes replaces the type-2 (altitude grid) payload when non-nil. The
	// default is the all-zero grid, which is a perfectly valid altitude layer —
	// but a flat one, so a displaced world built over it is numerically the same
	// size as the flat one. A test that must tell the two apart supplies its own
	// relief here. Bytes, not signed values: the decode reads the sign.
	Altitudes []uint8

	// Overlay replaces the type-3 (overlay grid) payload when non-nil. The default
	// is the all-zero grid, i.e. a map with nothing placed in that layer, which is
	// why a test that needs a placed overlay cell has to supply the codes itself.
	// The bytes are the stored codes, taken verbatim: this builder gives a cell no
	// meaning and applies no bias to it.
	Overlay []uint8

	// Objects and Units are the type-4 and type-6 placed records. Empty is the
	// default — an absent payload and a zero count — and a non-empty slice is
	// written record for record with the matching type-0 count (+0x20, +0x24)
	// following from its length, since the reader drives both walks from that
	// count. Their lengths are the only thing a caller does not control.
	Objects []ALMObject
	Units   []ALMUnit

	// Type7Payload replaces the type-7 (mission script) payload when non-nil. It
	// is the WHOLE payload — the action count word, the action records, the
	// condition count word, the condition records, the trigger count word and the
	// trigger records — because the three arrays are counted independently and a
	// builder that assembled them from typed fields would be a second encoder of
	// a grammar `pkg/formats/alm` already decodes.
	//
	// The default is a bare zero count word, which is a payload the script
	// decoder REFUSES: it declares no actions and then ends, so the condition
	// count is past the buffer. That is deliberate and is depended on — it is how
	// a test reaches "a map whose script will not decode" — so the default stays
	// what it was and only a caller naming this field gets a readable script.
	Type7Payload []byte
}

// ALM assembles a valid .alm: a 20-byte file header, then ten records each with
// a 20-byte header followed by pure payload, tiling exactly to EOF with no
// trailer. Records are emitted in the corpus physical order, which is not typeId
// order, so a reader that assumes position rather than reading each record's own
// typeId is caught.
func ALM(o ALMOptions) []byte {
	w, h := o.Width, o.Height
	cells := w * h

	meta := make([]byte, 632)
	binary.LittleEndian.PutUint32(meta[0x00:], uint32(w))
	binary.LittleEndian.PutUint32(meta[0x04:], uint32(h))
	binary.LittleEndian.PutUint32(meta[0x1c:], 1)                      // one type5 group record
	binary.LittleEndian.PutUint32(meta[0x20:], uint32(len(o.Objects))) // #type4
	binary.LittleEndian.PutUint32(meta[0x24:], uint32(len(o.Units)))   // #type6
	copy(meta[0x30:], o.Name)                                          // NUL-terminated by the zero fill
	copy(meta[0x78:], o.Description)

	tiles := o.Type1Payload
	if tiles == nil {
		tiles = make([]byte, 2*cells)
	}

	altitudes := o.Altitudes
	if altitudes == nil {
		altitudes = make([]byte, cells)
	}

	overlay := o.Overlay
	if overlay == nil {
		overlay = make([]byte, cells)
	}

	group := make([]byte, 76)
	copy(group[0x0c:], "Self")

	var payloads [10][]byte
	payloads[0] = meta
	payloads[1] = tiles
	payloads[2] = altitudes
	payloads[3] = overlay
	payloads[4] = almObjects(o.Objects)
	payloads[5] = group
	payloads[6] = almUnits(o.Units)
	payloads[7] = o.Type7Payload
	if payloads[7] == nil {
		payloads[7] = le32(0) // type7: entryCount 0
	}
	payloads[8] = nil     // type8: no records (ALM-TRIG-050; not a marker tree)
	payloads[9] = le32(0) // type9: count 0

	out := concat(le32(0x0052374D), le32(20), le32(0), le32(10), le32(990))
	for _, typeID := range []int{0, 1, 2, 3, 5, 4, 9, 8, 6, 7} {
		p := payloads[typeID]
		out = concat(out, le32(7), le32(20), le32(uint32(len(p))), le32(uint32(typeID)), le32(0), p)
	}
	return out
}

const (
	almObjectRecordSize = 20   // type-4 base record
	almObjectExtSize    = 8    // type-4 kind==0x21 extension
	almObjectExtKind    = 0x21 // type-4 Kind value that appends the extension
	almUnitRecordSize   = 70   // type-6 record
)

// almObjects lays the type-4 payload: one 20-byte base record per object, each
// followed by its 8-byte extension when Kind is 0x21. A nil result for an empty
// slice is the point — it is the absent payload the record carried before this
// field existed.
func almObjects(objects []ALMObject) []byte {
	var out []byte
	for _, obj := range objects {
		rec := make([]byte, almObjectRecordSize)
		binary.LittleEndian.PutUint32(rec[0x00:], obj.X)
		binary.LittleEndian.PutUint32(rec[0x04:], obj.Y)
		binary.LittleEndian.PutUint32(rec[0x08:], obj.Kind)
		binary.LittleEndian.PutUint16(rec[0x0c:], obj.Field0C)
		binary.LittleEndian.PutUint32(rec[0x0e:], obj.Field0E)
		binary.LittleEndian.PutUint16(rec[0x12:], obj.Field12)
		out = append(out, rec...)

		if obj.Kind != almObjectExtKind {
			if obj.Ext != nil {
				panic(fmt.Sprintf("synth: type-4 Ext set on a record whose Kind is %#x, not %#x, and the format has nowhere to write it",
					obj.Kind, almObjectExtKind))
			}
			continue
		}
		ext := obj.Ext
		if ext == nil {
			ext = make([]byte, almObjectExtSize)
		}
		if len(ext) != almObjectExtSize {
			panic(fmt.Sprintf("synth: type-4 Ext is %d bytes, want %d", len(ext), almObjectExtSize))
		}
		out = append(out, ext...)
	}
	return out
}

// almUnits lays the type-6 payload: one 70-byte record per unit, the bytes beyond
// the decoded fields written at their own offsets. ClassID and an authored
// CurrentHP go out as the two bytes of their own int16, the exact inverse of the
// sign-extending reads. An absent CurrentHP writes -1, not zero.
func almUnits(units []ALMUnit) []byte {
	var out []byte
	for _, u := range units {
		rec := make([]byte, almUnitRecordSize)
		binary.LittleEndian.PutUint32(rec[0x00:], u.X)
		binary.LittleEndian.PutUint32(rec[0x04:], u.Y)
		binary.LittleEndian.PutUint16(rec[0x08:], uint16(u.ClassID))
		binary.LittleEndian.PutUint16(rec[0x0a:], u.ClassSubID)
		binary.LittleEndian.PutUint32(rec[0x0c:], u.Flags)
		binary.LittleEndian.PutUint32(rec[0x10:], u.DefID)
		currentHP := int16(-1)
		if u.HasCurrentHP {
			currentHP = u.CurrentHP
		}
		binary.LittleEndian.PutUint16(rec[0x20:], uint16(currentHP))
		out = append(out, rec...)
	}
	return out
}

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
