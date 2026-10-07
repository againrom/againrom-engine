package synth_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/res"
	"againrom/pkg/formats/spr256"
)

// almFraming walks the .alm container framing independently of the reader under
// exercise: a 20-byte file header whose +0x0c holds the record count, then that
// many records, each a 20-byte header carrying its own payload size at +0x08
// followed by exactly that many payload bytes, tiling to EOF with no trailer.
// It exists so "well-framed but internally inconsistent" is an asserted property
// of the fixture rather than a hope.
func almFraming(data []byte) error {
	const fileHeader, recordHeader, offRecordCount, offPayloadSize = 20, 20, 0x0c, 0x08
	if len(data) < fileHeader {
		return fmt.Errorf("stream is %d bytes, shorter than the %d-byte file header", len(data), fileHeader)
	}
	n := int(binary.LittleEndian.Uint32(data[offRecordCount:]))
	if n != 10 {
		return fmt.Errorf("recordCount %d, want 10", n)
	}
	cursor := fileHeader
	for i := 0; i < n; i++ {
		if cursor+recordHeader > len(data) {
			return fmt.Errorf("record %d header does not fit before EOF", i)
		}
		size := int(binary.LittleEndian.Uint32(data[cursor+offPayloadSize:]))
		cursor += recordHeader + size
		if cursor > len(data) {
			return fmt.Errorf("record %d payload (%d bytes) overruns EOF", i, size)
		}
	}
	if cursor != len(data) {
		return fmt.Errorf("records end at %d, want exactly %d (EOF)", cursor, len(data))
	}
	return nil
}

// menuPaths is the eighteen-entry menu asset set, written out here rather than
// asked of any package: the base, the hit mask, and a normal plus a pressed
// overlay for each of the eight buttons.
func menuPaths(prefix string) []string {
	paths := []string{prefix + "menu_.bmp", prefix + "menumask.bmp"}
	for i := 1; i <= 8; i++ {
		paths = append(paths, fmt.Sprintf("%sbutton%d.bmp", prefix, i))
	}
	for i := 1; i <= 8; i++ {
		paths = append(paths, fmt.Sprintf("%sbutton%dp.bmp", prefix, i))
	}
	return paths
}

func archivePaths(a *res.Archive) []string {
	entries := a.Entries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Path)
	}
	sort.Strings(out)
	return out
}

// menuOptions is a small but fully-specified asset set: every button gets a
// distinct hover and pressed rectangle inside a 16x12 frame, and a distinct hot
// mask index.
func menuOptions(prefix string) synth.MenuOptions {
	o := synth.MenuOptions{
		Prefix:   prefix,
		BaseSize: image.Point{X: 16, Y: 12},
		MaskSize: image.Point{X: 16, Y: 12},
	}
	for i := 0; i < 8; i++ {
		o.Hover[i] = image.Rect(i, 0, i+3, 2)
		o.Pressed[i] = image.Rect(i, 2, i+4, 5)
		o.MaskIndex[i] = byte(0x80 + 0x10*i)
	}
	return o
}

// TestBuilders — T2: the shared fixture builders produce streams the real
// readers accept, so a later acceptance test that reasons about a synthetic
// archive, map or menu asset set is reasoning about the shipped formats.
func TestBuilders(t *testing.T) {
	t.Run("archive round-trips through res", func(t *testing.T) {
		files := []synth.File{
			{Path: "readme.txt", Data: []byte("a root-level file")},
			{Path: "graphics/mainmenu/x.bmp", Data: []byte{0x01, 0x02, 0x03, 0x04}},
			{Path: "graphics/mainmenu/y.bmp", Data: bytes.Repeat([]byte{0xab}, 100)},
			{Path: "graphics/tiles/t.bmp", Data: []byte{0xff}},
			{Path: "deep/a/b/c/leaf.bin", Data: []byte("nested four levels down")},
			{Path: "empty.bin", Data: nil},
		}
		data := synth.Archive(files)

		fromBytes, err := res.OpenBytes(data)
		if err != nil {
			t.Fatalf("res.OpenBytes: %v", err)
		}

		path := filepath.Join(t.TempDir(), "synthetic.res")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write archive: %v", err)
		}
		fromFile, err := res.Open(path)
		if err != nil {
			t.Fatalf("res.Open: %v", err)
		}

		for _, a := range []struct {
			how string
			arc *res.Archive
		}{{"OpenBytes", fromBytes}, {"Open", fromFile}} {
			if got, want := len(a.arc.Entries()), len(files); got != want {
				t.Errorf("%s: %d entries, want %d (directories are structure, not entries)", a.how, got, want)
			}
			for _, f := range files {
				got, err := a.arc.ReadFile(f.Path)
				if err != nil {
					t.Errorf("%s: ReadFile(%q): %v", a.how, f.Path, err)
					continue
				}
				if !bytes.Equal(got, f.Data) {
					t.Errorf("%s: ReadFile(%q) = %d bytes %x, want %d bytes %x",
						a.how, f.Path, len(got), got, len(f.Data), f.Data)
				}
			}
			want := make([]string, 0, len(files))
			for _, f := range files {
				want = append(want, f.Path)
			}
			sort.Strings(want)
			if got := archivePaths(a.arc); !equalStrings(got, want) {
				t.Errorf("%s: entry paths = %q, want %q", a.how, got, want)
			}
			if _, err := a.arc.ReadFile("graphics/mainmenu/absent.bmp"); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s: ReadFile of a missing path gave %v, want fs.ErrNotExist", a.how, err)
			}
		}
	})

	t.Run("alm round-trips through alm.Open", func(t *testing.T) {
		const w, h = 8, 5
		data := synth.ALM(synth.ALMOptions{
			Width:       w,
			Height:      h,
			Name:        "Testmap",
			Description: "a synthetic map",
		})
		if err := almFraming(data); err != nil {
			t.Fatalf("fixture framing: %v", err)
		}
		m, err := alm.Open(data)
		if err != nil {
			t.Fatalf("alm.Open: %v", err)
		}
		if m.Width != w || m.Height != h {
			t.Errorf("map is %dx%d, want %dx%d", m.Width, m.Height, w, h)
		}
		if m.Name != "Testmap" {
			t.Errorf("map name = %q, want %q", m.Name, "Testmap")
		}
		if m.Description != "a synthetic map" {
			t.Errorf("map description = %q, want %q", m.Description, "a synthetic map")
		}
	})

	// The shape a later task needs: the file tiles exactly to EOF, so the header
	// walk succeeds, but the type-1 payload disagrees with 2*W*H, so the full
	// decode must fail.
	t.Run("alm with an inconsistent type-1 payload is well-framed and rejected", func(t *testing.T) {
		const w, h = 8, 5
		for _, size := range []int{2*w*h - 2, 2*w*h + 2} {
			data := synth.ALM(synth.ALMOptions{Width: w, Height: h, Type1Payload: make([]byte, size)})
			if err := almFraming(data); err != nil {
				t.Fatalf("type-1 payload %d bytes: fixture is not well-framed: %v", size, err)
			}
			m, err := alm.Open(data)
			if err == nil {
				t.Errorf("type-1 payload %d bytes (want %d): alm.Open accepted the map, want an error", size, 2*w*h)
			}
			if err != nil && m != nil {
				t.Errorf("type-1 payload %d bytes: alm.Open reported an error and a non-nil map", size)
			}
		}
	})

	// The shape a sweep needs: a map that actually carries placements.
	t.Run("alm carries placed records of all three kinds", func(t *testing.T) {
		const w, h = 4, 3

		overlay := make([]uint8, w*h)
		overlay[0] = 1
		overlay[5] = 42
		overlay[w*h-1] = 200

		objects := []synth.ALMObject{
			{X: 0x0200, Y: 0x0300, Kind: 7, Field0C: 0x1122, Field0E: 0x33445566, Field12: 0x7788},
			{X: 0x0400, Y: 0x0500, Kind: 0x21}, // the extension defaults to eight zero bytes
			{X: 0x0600, Y: 0x0700, Kind: 0x21, Ext: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			{X: 0x0800, Y: 0x0900, Kind: 66},
		}
		wantExt := [][]byte{nil, make([]byte, 8), {1, 2, 3, 4, 5, 6, 7, 8}, nil}

		units := []synth.ALMUnit{
			{X: 0x0100, Y: 0x0100, ClassID: 1},
			{X: 0x0200, Y: 0x0100, ClassID: 80, ClassSubID: 9, Flags: 1, HasCurrentHP: true},
			{X: 0x0300, Y: 0x0100, ClassID: 33, DefID: 0xcdcdcdcd, CurrentHP: -10, HasCurrentHP: true},
			{X: 0x0400, Y: 0x0100, ClassID: 12, Flags: 1, DefID: 0x1234, CurrentHP: 321, HasCurrentHP: true},
			{X: 0x0500, Y: 0x0100, ClassID: -32767}, // 0x8001: a key that must come back negative
		}

		data := synth.ALM(synth.ALMOptions{
			Width: w, Height: h,
			Overlay: overlay,
			Objects: objects,
			Units:   units,
		})
		if err := almFraming(data); err != nil {
			t.Fatalf("fixture framing: %v", err)
		}
		m, err := alm.Open(data)
		if err != nil {
			t.Fatalf("alm.Open: %v", err)
		}

		if !bytes.Equal(m.Overlay, overlay) {
			t.Errorf("Overlay = %v, want %v", m.Overlay, overlay)
		}
		if m.Meta.Count4 != uint32(len(objects)) || m.Meta.Count6 != uint32(len(units)) {
			t.Errorf("type-0 counts = #type4 %d, #type6 %d; want %d and %d",
				m.Meta.Count4, m.Meta.Count6, len(objects), len(units))
		}

		if len(m.Objects) != len(objects) {
			t.Fatalf("%d type-4 records decoded, want %d", len(m.Objects), len(objects))
		}
		for i, want := range objects {
			got := m.Objects[i]
			if got.X != want.X || got.Y != want.Y || got.Kind != want.Kind ||
				got.Field0C != want.Field0C || got.Field0E != want.Field0E || got.Field12 != want.Field12 {
				t.Errorf("object %d = %+v, want X=%#x Y=%#x Kind=%#x Field0C=%#x Field0E=%#x Field12=%#x",
					i, got, want.X, want.Y, want.Kind, want.Field0C, want.Field0E, want.Field12)
			}
			if !bytes.Equal(got.Ext, wantExt[i]) {
				t.Errorf("object %d Ext = %v, want %v", i, got.Ext, wantExt[i])
			}
		}

		if len(m.Units) != len(units) {
			t.Fatalf("%d type-6 records decoded, want %d", len(m.Units), len(units))
		}
		for i, want := range units {
			got := m.Units[i]
			wantHP := int16(-1)
			if want.HasCurrentHP {
				wantHP = want.CurrentHP
			}
			if got.X != want.X || got.Y != want.Y || got.ClassID != want.ClassID ||
				got.ClassSubID != want.ClassSubID || got.Flags != want.Flags || got.DefID != want.DefID ||
				got.CurrentHP != wantHP ||
				got.HasCurrentHP != want.HasCurrentHP {
				t.Errorf("unit %d = %+v, want %+v (absent CurrentHP means raw -1)", i, got, want)
			}
		}
	})

	// The other half of the same contract: with none of the placement fields set,
	// the builder still produces what it produced before they existed — the
	// all-zero overlay, no type-4 or type-6 payload, and both counts zero. Every
	// caller outside this package builds maps this way.
	t.Run("alm without placements keeps the empty payloads", func(t *testing.T) {
		const w, h = 6, 4
		m, err := alm.Open(synth.ALM(synth.ALMOptions{Width: w, Height: h}))
		if err != nil {
			t.Fatalf("alm.Open: %v", err)
		}
		if got, want := len(m.Overlay), w*h; got != want {
			t.Fatalf("Overlay is %d cells, want %d", got, want)
		}
		if !bytes.Equal(m.Overlay, make([]byte, w*h)) {
			t.Errorf("Overlay = %v, want every cell zero", m.Overlay)
		}
		if len(m.Objects) != 0 || len(m.Units) != 0 {
			t.Errorf("%d type-4 and %d type-6 records, want none of either", len(m.Objects), len(m.Units))
		}
		if m.Meta.Count4 != 0 || m.Meta.Count6 != 0 {
			t.Errorf("type-0 counts = #type4 %d, #type6 %d; want 0 and 0", m.Meta.Count4, m.Meta.Count6)
		}
	})

	t.Run("menu archive holds the eighteen entries", func(t *testing.T) {
		const prefix = "graphics/mainmenu/"
		a, err := res.OpenBytes(synth.MenuArchive(menuOptions(prefix)))
		if err != nil {
			t.Fatalf("res.OpenBytes: %v", err)
		}
		want := menuPaths(prefix)
		sort.Strings(want)
		if got := archivePaths(a); !equalStrings(got, want) {
			t.Fatalf("entry paths = %q, want %q", got, want)
		}
		if got := len(a.Entries()); got != 18 {
			t.Errorf("%d entries, want 18", got)
		}
		for _, p := range want {
			b, err := a.ReadFile(p)
			if err != nil {
				t.Errorf("ReadFile(%q): %v", p, err)
				continue
			}
			if len(b) == 0 {
				t.Errorf("ReadFile(%q) returned no bytes", p)
			}
		}
	})

	t.Run("menu archive edits replace and omit entries", func(t *testing.T) {
		const prefix = "graphics/mainmenu/"
		replaced := []byte("not a bitmap at all")
		o := menuOptions(prefix)
		o.Edits = map[string][]byte{
			prefix + "button3.bmp":  replaced,
			prefix + "button4p.bmp": nil,
		}
		a, err := res.OpenBytes(synth.MenuArchive(o))
		if err != nil {
			t.Fatalf("res.OpenBytes: %v", err)
		}

		want := make([]string, 0, 17)
		for _, p := range menuPaths(prefix) {
			if p != prefix+"button4p.bmp" {
				want = append(want, p)
			}
		}
		sort.Strings(want)
		if got := archivePaths(a); !equalStrings(got, want) {
			t.Fatalf("entry paths = %q, want %q", got, want)
		}
		if got := len(a.Entries()); got != 17 {
			t.Errorf("%d entries, want 17 (one omitted)", got)
		}
		if got, err := a.ReadFile(prefix + "button3.bmp"); err != nil {
			t.Errorf("ReadFile of the replaced entry: %v", err)
		} else if !bytes.Equal(got, replaced) {
			t.Errorf("replaced entry = %q, want %q", got, replaced)
		}
		if _, err := a.ReadFile(prefix + "button4p.bmp"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadFile of the omitted entry gave %v, want fs.ErrNotExist", err)
		}
	})
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// T6 — the .256 sheet and objects-registry fixture builders
// ---------------------------------------------------------------------------

// The .256 layout, restated from docs/0002-sprites-256/spec.md's "Format
// definition" and NOT read off pkg/formats/spr256. That distinction is the whole
// value of the byte-level assertions below: a round-trip through the decoder
// shows the builder and the decoder to be mutual inverses, which they would be
// even if both were wrong about the format. Only an expectation computed from
// the contract can tell those two apart.
//
//	[ 1024 B palette, iff the trailer's bit 31 ][ frame, ... ][ 4 B trailer ]
//	palette entry: [B, G, R, reserved]
//	frame record:  0x00 u32 width | 0x04 u32 height | 0x08 u32 dataSize | data
//	trailer:       [31-bit frameCount][bit 31 = has-palette]
//	control byte:  [2-bit class | 6-bit count] — 0x00 literal, 0x40 blank rows,
//	               0x80 transparent, 0xC0 an alias of 0x80
const (
	sprPaletteLen = 1024
	sprEntryLen   = 4
	sprFrameHdr   = 12
	sprTrailerLen = 4
	sprHasPalette = 0x80000000
	sprCountMask  = 0x7fffffff

	offFrameWidth    = 0x00
	offFrameHeight   = 0x04
	offFrameDataSize = 0x08
)

// sheetFrame is one frame record as the contract's own walk sees it.
type sheetFrame struct {
	width, height int
	data          []byte
}

// sheet256Walk tiles a .256 stream by the contract's arithmetic alone: the
// trailer first, then the palette iff bit 31 is set, then frame records from the
// palette's end to the trailer. It decodes no RLE — it only says where each
// block begins and ends — so a builder that put a field at the wrong offset, or
// that let the frame count and the record walk disagree, fails here whatever any
// decoder makes of the same bytes.
func sheet256Walk(b []byte) (palette []byte, frames []sheetFrame, count int, err error) {
	if len(b) < sprTrailerLen {
		return nil, nil, 0, fmt.Errorf("stream is %d bytes, shorter than the %d-byte trailer", len(b), sprTrailerLen)
	}
	trailer := le32(b, len(b)-sprTrailerLen)
	count = int(trailer & sprCountMask)

	cursor := 0
	if trailer&sprHasPalette != 0 {
		if len(b) < sprPaletteLen+sprTrailerLen {
			return nil, nil, 0, fmt.Errorf("trailer claims a palette but the stream is only %d bytes", len(b))
		}
		palette = b[:sprPaletteLen]
		cursor = sprPaletteLen
	}

	end := len(b) - sprTrailerLen
	for cursor < end {
		if cursor+sprFrameHdr > end {
			return nil, nil, 0, fmt.Errorf("frame %d: header does not fit before the trailer", len(frames))
		}
		w := int(le32(b, cursor+offFrameWidth))
		h := int(le32(b, cursor+offFrameHeight))
		size := int(le32(b, cursor+offFrameDataSize))
		if cursor+sprFrameHdr+size > end {
			return nil, nil, 0, fmt.Errorf("frame %d: a %d-byte block overruns the trailer", len(frames), size)
		}
		frames = append(frames, sheetFrame{width: w, height: h, data: b[cursor+sprFrameHdr : cursor+sprFrameHdr+size]})
		cursor += sprFrameHdr + size
	}
	if cursor != end {
		return nil, nil, 0, fmt.Errorf("records end at %d, want exactly %d (the trailer)", cursor, end)
	}
	return palette, frames, count, nil
}

func decodeSheet(t *testing.T, b []byte) *spr256.Sprite {
	t.Helper()
	s, err := spr256.Decode(b)
	if err != nil {
		t.Fatalf("spr256.Decode: %v", err)
	}
	return s
}

// mixedFrame is the frame whose RLE bytes are hand-written below: a 4x3 grid
// whose first row is transparent / two opaque / transparent, whose second row is
// wholly transparent, and whose third is four opaque pixels.
//
// The opaque pixel carrying INDEX 0 is deliberate. Transparency here is
// structural — the opcodes carry it — so index 0 is an ordinary colour, and a
// builder that spelled a hole as index 0 could not produce this frame at all.
func mixedFrame() synth.Frame256 {
	op := func(i uint8) synth.Pixel256 { return synth.Pixel256{Index: i, Opaque: true} }
	var hole synth.Pixel256
	return synth.Frame256{
		Width:  4,
		Height: 3,
		Pixels: []synth.Pixel256{
			hole, op(7), op(0), hole,
			hole, hole, hole, hole,
			op(1), op(1), op(1), op(1),
		},
	}
}

// wideFrame is 70 opaque pixels on one row: one pixel more than a six-bit count
// can hold twice over, so the run must be split at 63 and a builder that let the
// length overflow into the control byte's class bits writes a different opcode.
func wideFrame() synth.Frame256 {
	px := make([]synth.Pixel256, 70)
	for i := range px {
		px[i] = synth.Pixel256{Index: 9, Opaque: true}
	}
	return synth.Frame256{Width: 70, Height: 1, Pixels: px}
}

// sheetPalette is a palette whose first four entries have pairwise-distinct R, G
// and B, so the on-disk order is recoverable from the bytes and a swap of any
// two channels is visible. Entry 0 is a colour like any other: the shipped
// palettes reserve index 0 as their transparent key, but that is a fact about
// the art and not about the block.
func sheetPalette() []color.RGBA {
	return []color.RGBA{
		{R: 0x11, G: 0x22, B: 0x33},
		{R: 0xff, G: 0x00, B: 0x00},
		{R: 0x00, G: 0xff, B: 0x00},
		{R: 0x01, G: 0x02, B: 0x04},
	}
}

// TestSheet256Builder — T6: the .256 builder writes the contract's own
// bytes, and the stream it writes is one pkg/formats/spr256 reads back to
// what was given.
func TestSheet256Builder(t *testing.T) {
	t.Run("the RLE program is the grammar's own bytes", func(t *testing.T) {
		// Hand-derived from the opcode table, token by token, for mixedFrame:
		//
		//	row 0  0x81                    transparent, N=1
		//	       0x02 0x07 0x00          literal, N=2, indices 7 and 0
		//	       0x81                    transparent, N=1   (1+2+1 = width)
		//	row 1  0x41                    blank rows, N=1
		//	row 2  0x04 0x01 0x01 0x01 0x01  literal, N=4     (4 = width)
		wantMixed := []byte{0x81, 0x02, 0x07, 0x00, 0x81, 0x41, 0x04, 0x01, 0x01, 0x01, 0x01}

		// wideFrame: 70 opaque pixels are a literal of 63 then a literal of 7.
		wantWide := []byte{0x3f}
		wantWide = append(wantWide, bytes.Repeat([]byte{0x09}, 63)...)
		wantWide = append(wantWide, 0x07)
		wantWide = append(wantWide, bytes.Repeat([]byte{0x09}, 7)...)

		// A wholly transparent 3x5 grid is one blank-rows token of 5.
		wantBlank := []byte{0x45}

		given := []synth.Frame256{
			mixedFrame(),
			wideFrame(),
			{Width: 3, Height: 5},
			// An empty grid still has to close its rows: a frame of no rows
			// needs no program, one of three empty rows needs the token that
			// yields three.
			{Width: 0, Height: 0},
			{Width: 3, Height: 0},
			{Width: 0, Height: 3},
		}
		wantRLE := [][]byte{wantMixed, wantWide, wantBlank, nil, nil, {0x43}}

		sheet := synth.Sheet256(synth.Sheet256Options{Palette: sheetPalette(), Frames: given})
		_, frames, _, err := sheet256Walk(sheet)
		if err != nil {
			t.Fatalf("fixture framing: %v", err)
		}
		if len(frames) != len(given) {
			t.Fatalf("%d frame records, want %d", len(frames), len(given))
		}
		for i := range given {
			// Width BEFORE height, read at the contract's own offsets. The
			// round-trip cannot pin this: a builder and a decoder that had
			// swapped the two words together would agree with each other on
			// every frame, square or not.
			if frames[i].width != given[i].Width || frames[i].height != given[i].Height {
				t.Errorf("frame %d record header is %dx%d, want %dx%d",
					i, frames[i].width, frames[i].height, given[i].Width, given[i].Height)
			}
			if !bytes.Equal(frames[i].data, wantRLE[i]) {
				t.Errorf("frame %d RLE = % x, want % x", i, frames[i].data, wantRLE[i])
			}
		}
	})

	t.Run("the palette block is 256 entries of B, G, R, reserved", func(t *testing.T) {
		colors := sheetPalette()
		sheet := synth.Sheet256(synth.Sheet256Options{Palette: colors, Frames: []synth.Frame256{mixedFrame()}})

		palette, _, _, err := sheet256Walk(sheet)
		if err != nil {
			t.Fatalf("fixture framing: %v", err)
		}
		if len(palette) != sprPaletteLen {
			t.Fatalf("palette block is %d bytes, want %d", len(palette), sprPaletteLen)
		}
		for i, c := range colors {
			got := palette[i*sprEntryLen : (i+1)*sprEntryLen]
			want := []byte{c.B, c.G, c.R, 0x00}
			if !bytes.Equal(got, want) {
				t.Errorf("palette entry %d = % x, want % x — the on-disk order is BGR with a zero reserved byte",
					i, got, want)
			}
		}
		for i := len(colors) * sprEntryLen; i < sprPaletteLen; i++ {
			if palette[i] != 0 {
				t.Errorf("palette byte %d = %#x, want 0 — entries past the supplied colours stay zero", i, palette[i])
				break
			}
		}
	})

	t.Run("the trailer carries the record count and the palette flag", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			noPalette bool
			frames    int
			want      uint32
		}{
			{"palette-bearing, three frames", false, 3, sprHasPalette | 3},
			{"palette-less, three frames", true, 3, 3},
			{"palette-bearing, no frames", false, 0, sprHasPalette},
		} {
			frames := make([]synth.Frame256, tc.frames)
			for i := range frames {
				frames[i] = mixedFrame()
			}
			sheet := synth.Sheet256(synth.Sheet256Options{
				Palette:   sheetPalette(),
				NoPalette: tc.noPalette,
				Frames:    frames,
			})
			if got := le32(sheet, len(sheet)-sprTrailerLen); got != tc.want {
				t.Errorf("%s: trailer = %#08x, want %#08x", tc.name, got, tc.want)
			}
			palette, records, count, err := sheet256Walk(sheet)
			if err != nil {
				t.Errorf("%s: fixture framing: %v", tc.name, err)
				continue
			}
			if (palette != nil) == tc.noPalette {
				t.Errorf("%s: palette block present = %t, want %t", tc.name, palette != nil, !tc.noPalette)
			}
			if len(records) != count || count != tc.frames {
				t.Errorf("%s: %d records against a trailer count of %d, want %d of each",
					tc.name, len(records), count, tc.frames)
			}
		}
	})

	t.Run("a sheet round-trips through spr256", func(t *testing.T) {
		colors := sheetPalette()
		want := []synth.Frame256{
			mixedFrame(),
			wideFrame(),
			{Width: 3, Height: 5}, // wholly transparent
			{Width: 0, Height: 0}, // the empty grid, in all three of its spellings
			{Width: 0, Height: 3},
			{Width: 3, Height: 0},
			{Width: 2, Height: 2, Pixels: []synth.Pixel256{ // an opaque index 0 beside a hole
				{Index: 0, Opaque: true}, {},
				{}, {Index: 255, Opaque: true},
			}},
		}
		s := decodeSheet(t, synth.Sheet256(synth.Sheet256Options{Palette: colors, Frames: want}))

		if !s.HasPalette {
			t.Fatal("HasPalette is false on a sheet built with a palette")
		}
		if len(s.Palette) != sprPaletteLen/sprEntryLen {
			t.Fatalf("palette has %d entries, want %d", len(s.Palette), sprPaletteLen/sprEntryLen)
		}
		for i, c := range colors {
			if got := s.Palette[i]; got.R != c.R || got.G != c.G || got.B != c.B {
				t.Errorf("palette %d = %+v, want R=%#x G=%#x B=%#x", i, got, c.R, c.G, c.B)
			}
		}

		if len(s.Frames) != len(want) {
			t.Fatalf("%d frames decoded, want %d", len(s.Frames), len(want))
		}
		for i, w := range want {
			got := s.Frames[i]
			if got.Width != w.Width || got.Height != w.Height {
				t.Errorf("frame %d is %dx%d, want %dx%d", i, got.Width, got.Height, w.Width, w.Height)
				continue
			}
			for row := 0; row < w.Height; row++ {
				for col := 0; col < w.Width; col++ {
					var wp synth.Pixel256
					if w.Pixels != nil {
						wp = w.Pixels[row*w.Width+col]
					}
					gp := got.Pixels[row*w.Width+col]
					if gp.Opaque != wp.Opaque || (wp.Opaque && gp.Index != wp.Index) {
						t.Errorf("frame %d pixel (%d,%d) = %+v, want %+v", i, col, row, gp, wp)
					}
				}
			}
		}
	})

	t.Run("Sheet256Raw validates nothing", func(t *testing.T) {
		// The escape hatch: three streams no well-formed builder can produce,
		// each rejected by the decoder — which is what makes an undecodable
		// sheet a fixture rather than an accident.
		for _, tc := range []struct {
			name   string
			frames []synth.Sheet256RawFrame
			count  uint32
		}{
			{
				"a dataSize overrunning the trailer",
				[]synth.Sheet256RawFrame{{Width: 2, Height: 2, DataSize: 99, Data: []byte{0x82, 0x82}}},
				1,
			},
			{
				"a row whose tokens do not sum to the width",
				[]synth.Sheet256RawFrame{{Width: 4, Height: 1, DataSize: 1, Data: []byte{0x82}}},
				1,
			},
			{
				"a frame count disagreeing with the records",
				[]synth.Sheet256RawFrame{{Width: 1, Height: 1, DataSize: 2, Data: []byte{0x01, 0x05}}},
				7,
			},
		} {
			raw := synth.Sheet256Raw(synth.Palette256(sheetPalette()), tc.frames, sprHasPalette|tc.count)
			s, err := spr256.Decode(raw)
			if err == nil {
				t.Errorf("%s: Decode accepted the stream, want an error", tc.name)
			}
			if s != nil {
				t.Errorf("%s: Decode returned an error and a non-nil sprite", tc.name)
			}
		}
	})
}

func regObjInt(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func regObjStr(name, s string) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x00, Str: s}
}

// descCP866 spells a short Cyrillic word under CP866 and a different byte
// sequence's worth of nonsense under any other code page. The .reg format
// defines no character encoding and nothing on this path applies one, so what
// the fixture asserts is that these exact six bytes come back — and they are
// written as hex, never as literal non-ASCII source (golden rule 2).
var descCP866 = []byte{0x84, 0xa5, 0xe0, 0xa5, 0xa2, 0xae}

func loadObjectsReg(t *testing.T, stream []byte) *data.ObjectClasses {
	t.Helper()
	r, err := reg.Parse(stream)
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	cs, err := data.LoadObjectClasses(r)
	if err != nil {
		t.Fatalf("data.LoadObjectClasses: %v", err)
	}
	return cs
}

// TestObjectsRegBuilder — T6: the objects-registry builder writes a stream
// pkg/formats/reg parses and pkg/data loads back to the keys it was given.
func TestObjectsRegBuilder(t *testing.T) {
	t.Run("a registry round-trips into LoadObjectClasses", func(t *testing.T) {
		files := []string{`trees\oak`, `stones\rock`}
		stream := synth.ObjectsReg(files,
			[]synth.RegNode{
				regObjInt("ID", 0), regObjInt("File", 0), regObjInt("Index", 3),
				regObjInt("Width", 64), regObjInt("Height", 80),
				regObjInt("CenterX", 30), regObjInt("CenterY", 70),
				regObjStr("DescText", string(descCP866)),
			},
			[]synth.RegNode{
				regObjInt("ID", 1), regObjInt("File", 1), regObjInt("Index", 0),
				regObjInt("Width", 32), regObjInt("Height", 32),
				regObjInt("CenterX", 16), regObjInt("CenterY", 24),
			},
			// Sets File itself — objects.reg's File never inherits — and takes
			// the rest of its geometry from the class whose ID is 0.
			[]synth.RegNode{
				regObjInt("ID", 2), regObjInt("File", 0), regObjInt("Parent", 0),
				regObjInt("Index", 1),
			},
		)

		if want := append(append([]byte(nil), descCP866...), 0x00); !bytes.Contains(stream, want) {
			t.Errorf("the stream does not carry the string value % x with its trailing NUL", want)
		}

		cs := loadObjectsReg(t, stream)
		if got := len(cs.All()); got != 3 {
			t.Fatalf("All() has %d classes, want 3 — [Global] ObjectCount must follow the sections written", got)
		}

		for _, want := range []struct {
			id                                     int32
			index, width, height, centerX, centerY int32
			path                                   string
			desc                                   []byte
		}{
			{0, 3, 64, 80, 30, 70, "objects/trees/oak.256", descCP866},
			{1, 0, 32, 32, 16, 24, "objects/stones/rock.256", nil},
			{2, 1, 64, 80, 30, 70, "objects/trees/oak.256", descCP866},
		} {
			c, ok := cs.ByID(want.id)
			if !ok {
				t.Errorf("ByID(%d) missed", want.id)
				continue
			}
			if c.Index != want.index || c.Width != want.width || c.Height != want.height ||
				c.CenterX != want.centerX || c.CenterY != want.centerY {
				t.Errorf("ID %d: Index/Width/Height/CenterX/CenterY = %d/%d/%d/%d/%d, want %d/%d/%d/%d/%d",
					want.id, c.Index, c.Width, c.Height, c.CenterX, c.CenterY,
					want.index, want.width, want.height, want.centerX, want.centerY)
			}
			if got := c.SpritePath(); got != want.path {
				t.Errorf("ID %d: SpritePath() = %q, want %q", want.id, got, want.path)
			}
			if got := []byte(c.DescText); !bytes.Equal(got, want.desc) {
				t.Errorf("ID %d: DescText = % x, want % x — the registry's bytes, with no code page applied",
					want.id, got, want.desc)
			}
		}
	})

	t.Run("the root's children are written name-sorted", func(t *testing.T) {
		// Eleven classes are what makes the sort observable: Object10 sorts
		// before Object2, which is the on-disk order the shipped registries
		// carry and the order the root's own kind word (bit 4) claims.
		classes := make([][]synth.RegNode, 11)
		for i := range classes {
			classes[i] = []synth.RegNode{regObjInt("ID", int32(i)), regObjInt("File", 0)}
		}
		stream := synth.ObjectsReg([]string{"art"}, classes...)

		want := []string{"Files", "Global"}
		for i := range classes {
			want = append(want, fmt.Sprintf("Object%d", i))
		}
		sort.Strings(want)

		got := make([]string, 0, len(want))
		for i := range want {
			got = append(got, trimNUL(regName(stream, i)))
		}
		if !equalStrings(got, want) {
			t.Errorf("root children in table order = %q, want %q", got, want)
		}
		at := func(name string) int {
			for i, n := range got {
				if n == name {
					return i
				}
			}
			return -1
		}
		if a, b := at("Object10"), at("Object2"); a < 0 || b < 0 || a > b {
			t.Errorf("Object10 is child %d and Object2 is child %d; the sort is over names, "+
				"so the eleventh section stands before the third", a, b)
		}

		// The order above is a fidelity claim, and the kind words are where the
		// stream makes it: the root marks its own child list sorted (bit 4) and
		// no section does, which is what every shipped registry carries. No
		// parser may reject on either — a set flag bit is never grounds — so
		// without these two assertions the flags are unobservable and the sort
		// is decoration.
		if flags := le32(stream, offRootFlags); flags != 0x11 {
			t.Errorf("rootFlags = %#x, want %#x — a directory whose child list is name-sorted", flags, 0x11)
		}
		for i := range want {
			if kind := le32(stream, regNodeAt(i)+offKind); kind != 0x01 {
				t.Errorf("child %d (%q) kind = %#x, want %#x — a plain directory, its children in the "+
					"order the caller gave them", i, got[i], kind, 0x01)
			}
		}

		// On-disk order is not load order: All() comes back in numeric section
		// order whatever the node table holds.
		cs := loadObjectsReg(t, stream)
		all := cs.All()
		if len(all) != len(classes) {
			t.Fatalf("All() has %d classes, want %d", len(all), len(classes))
		}
		for i, c := range all {
			if c.ID != int32(i) {
				t.Errorf("All()[%d].ID = %d, want %d", i, c.ID, i)
			}
		}
	})
}

// TestStaticExclusionFixtures — T6: the two builders suffice to construct
// every case the object layer's "Class to sprite frame" rule excludes, plus
// the opaque index-0 pixel that rule's structural transparency admits.
//
// Each case is BUILT here rather than argued for. What the test asserts is only
// that the fixture has the property its name claims — it is the T7 loader that
// must then skip the first three and count them, and nothing here anticipates
// how it does so.
func TestStaticExclusionFixtures(t *testing.T) {
	t.Run("a palette-less sheet", func(t *testing.T) {
		sheet := synth.Sheet256(synth.Sheet256Options{
			NoPalette: true,
			Frames:    []synth.Frame256{mixedFrame()},
		})
		if trailer := le32(sheet, len(sheet)-sprTrailerLen); trailer&sprHasPalette != 0 {
			t.Errorf("trailer = %#08x, want bit 31 clear", trailer)
		}
		if got := int(le32(sheet, offFrameWidth)); got != mixedFrame().Width {
			t.Errorf("the first u32 of the stream is %d, want the first frame's width %d — "+
				"a palette-less sheet's frames start at offset 0", got, mixedFrame().Width)
		}
		s := decodeSheet(t, sheet)
		if s.HasPalette || s.Palette != nil {
			t.Errorf("HasPalette = %t with %d palette entries, want false and none",
				s.HasPalette, len(s.Palette))
		}
		if len(s.Frames) != 1 {
			t.Errorf("%d frames, want 1 — the frames decode, only the palette is missing", len(s.Frames))
		}
	})

	t.Run("an opaque pixel carrying index 0", func(t *testing.T) {
		sheet := synth.Sheet256(synth.Sheet256Options{
			Palette: sheetPalette(),
			Frames: []synth.Frame256{{Width: 2, Height: 1, Pixels: []synth.Pixel256{
				{Index: 0, Opaque: true}, {},
			}}},
		})
		_, frames, _, err := sheet256Walk(sheet)
		if err != nil {
			t.Fatalf("fixture framing: %v", err)
		}
		// A literal of one holding the byte 0, then a transparent skip of one:
		// the index and the hole are different tokens, not one value overloaded.
		if want := []byte{0x01, 0x00, 0x81}; !bytes.Equal(frames[0].data, want) {
			t.Fatalf("RLE = % x, want % x", frames[0].data, want)
		}
		px := decodeSheet(t, sheet).Frames[0].Pixels
		if !px[0].Opaque || px[0].Index != 0 {
			t.Errorf("pixel 0 = %+v, want an opaque index 0", px[0])
		}
		if px[1].Opaque {
			t.Errorf("pixel 1 = %+v, want a transparent one", px[1])
		}
	})

	t.Run("an undecodable sheet", func(t *testing.T) {
		sheet := synth.Sheet256Raw(
			synth.Palette256(sheetPalette()),
			[]synth.Sheet256RawFrame{{Width: 8, Height: 8, DataSize: 4096, Data: []byte{0x88}}},
			sprHasPalette|1,
		)
		if _, _, _, err := sheet256Walk(sheet); err == nil {
			t.Error("the contract's own walk accepted the stream; the fixture is not undecodable")
		}
		s, err := spr256.Decode(sheet)
		if err == nil {
			t.Error("spr256.Decode accepted the stream, want an error")
		}
		if s != nil {
			t.Error("spr256.Decode returned an error and a non-nil sprite")
		}
	})

	t.Run("a class whose Index exceeds its sheet", func(t *testing.T) {
		const path = "objects/trees/oak.256"
		sheet := synth.Sheet256(synth.Sheet256Options{
			Palette: sheetPalette(),
			Frames:  []synth.Frame256{mixedFrame(), mixedFrame()},
		})
		cs := loadObjectsReg(t, synth.ObjectsReg([]string{`trees\oak`},
			[]synth.RegNode{regObjInt("ID", 0), regObjInt("File", 0), regObjInt("Index", 5)},
		))

		c, ok := cs.ByID(0)
		if !ok {
			t.Fatal("ByID(0) missed")
		}
		if got := c.SpritePath(); got != path {
			t.Fatalf("SpritePath() = %q, want %q", got, path)
		}
		frames := len(decodeSheet(t, sheet).Frames)
		if int(c.Index) < frames {
			t.Errorf("Index %d against a sheet of %d frames; the fixture must select past the last one",
				c.Index, frames)
		}
	})

	t.Run("an absent sheet", func(t *testing.T) {
		// The fourth exclusion needs no builder at all: an archive that holds
		// the registry and not the art. Recording it here is what says the
		// omission is the fixture, rather than something a builder forgot.
		stream := synth.ObjectsReg([]string{`trees\oak`},
			[]synth.RegNode{regObjInt("ID", 0), regObjInt("File", 0), regObjInt("Index", 0)},
		)
		a, err := res.OpenBytes(synth.Archive([]synth.File{{Path: "objects/objects.reg", Data: stream}}))
		if err != nil {
			t.Fatalf("res.OpenBytes: %v", err)
		}
		c, ok := loadObjectsReg(t, stream).ByID(0)
		if !ok {
			t.Fatal("ByID(0) missed")
		}
		if c.SpritePath() == "" {
			t.Fatal("SpritePath() is empty; the class must name art for its absence to be the case under test")
		}
		if _, err := a.ReadFile(c.SpritePath()); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadFile(%q) gave %v, want fs.ErrNotExist", c.SpritePath(), err)
		}
	})
}

// Font16 emits the container framing the .16 decoder walks: records tiling from
// offset 0, each [u32 w][u32 h][u32 dataSize][block], and a trailer holding the
// record count. This asserts the FRAMING only — that the pixel programs decode
// to the ink they were built from is asserted where a decoder is in scope, in
// pkg/game's loader tests.
func TestFont16Framing(t *testing.T) {
	glyphs := []synth.Font16Glyph{
		{Width: 8, Height: 6, Advance: 0},
		{Width: 8, Height: 6, Advance: 3, Ink: func(x, y int) (uint8, bool) { return 15, true }},
		{Width: 8, Height: 6, Advance: 7, Ink: func(x, y int) (uint8, bool) { return uint8(x), x%2 == 0 }},
	}
	atlas, advances := synth.Font16(glyphs)

	if len(advances) != 4*len(glyphs) {
		t.Fatalf("sidecar is %d bytes for %d records, want %d", len(advances), len(glyphs), 4*len(glyphs))
	}
	if got := binary.LittleEndian.Uint32(atlas[len(atlas)-4:]); got != uint32(len(glyphs)) {
		t.Fatalf("trailer says %d records, want %d", got, len(glyphs))
	}

	pos := 0
	for i, g := range glyphs {
		w := binary.LittleEndian.Uint32(atlas[pos:])
		h := binary.LittleEndian.Uint32(atlas[pos+4:])
		size := binary.LittleEndian.Uint32(atlas[pos+8:])
		if int(w) != g.Width || int(h) != g.Height {
			t.Fatalf("record %d header is %dx%d, want %dx%d", i, w, h, g.Width, g.Height)
		}
		if size == 0 {
			t.Fatalf("record %d carries an empty block", i)
		}
		pos += 12 + int(size)
	}
	if pos != len(atlas)-4 {
		t.Fatalf("records end at %d, want %d (the trailer's first byte)", pos, len(atlas)-4)
	}

	// A record with no ink is one blank-rows control, as record 0 is in every
	// shipped atlas.
	if got := binary.LittleEndian.Uint32(atlas[8:]); got != 1 {
		t.Fatalf("a blank record's block is %d bytes, want 1", got)
	}

	// An empty font is a lone trailer of zero.
	empty, emptyAdv := synth.Font16(nil)
	if len(empty) != 4 || binary.LittleEndian.Uint32(empty) != 0 || len(emptyAdv) != 0 {
		t.Fatalf("Font16(nil) = %v / %v, want a zero trailer and no sidecar", empty, emptyAdv)
	}
}

// TestStructuresRegBuilder — the structure-registry builder writes a
// stream pkg/formats/reg parses and pkg/data loads back to the keys it was
// given, over the ONE class registry with no [Files] table.
func TestStructuresRegBuilder(t *testing.T) {
	stream := synth.StructuresReg(
		[]synth.RegNode{
			regObjInt("ID", 1), regObjStr("File", `houses\hut`),
			regObjInt("TileWidth", 3), regObjInt("TileHeight", 2), regObjInt("FullHeight", 5),
		},
		// Spells no TileWidth at all: the omitted-key case.
		[]synth.RegNode{regObjInt("ID", 2), regObjStr("File", `houses\barn`)},
	)
	r, err := reg.Parse(stream)
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	if _, ok := r.GetInt("Global", "FileCount"); ok {
		t.Error("the stream carries FileCount; structures.reg has no [Files] table and no such key")
	}
	if _, ok := r.GetString("Files", "File0"); ok {
		t.Error("the stream carries a [Files] table; structures.reg has none")
	}
	cs, err := data.LoadStructureClasses(r)
	if err != nil {
		t.Fatalf("data.LoadStructureClasses: %v", err)
	}
	c, ok := cs.ByID(1)
	if !ok {
		t.Fatal("ByID(1) missed")
	}
	if c.File != `houses\hut` {
		t.Errorf("File = %q, want the stored path unchanged", c.File)
	}
	if c.TileWidth != 3 || c.TileHeight != 2 || c.FullHeight != 5 {
		t.Errorf("extents = %d x %d, FullHeight %d, want 3 x 2 and 5", c.TileWidth, c.TileHeight, c.FullHeight)
	}
	if got := c.SpritePath(); got != "structures/houses/hut.256" {
		t.Errorf("SpritePath() = %q, want structures/houses/hut.256", got)
	}
	if _, ok := cs.ByID(3); ok {
		t.Error("ByID(3) resolved; the registry holds two classes")
	}
}

// TestStructureSheetBuilder — the structure-sheet builder emits a .256 of
// the stated frame count at the stated frame size, and one frame of another
// size where the fixture asks for it.
//
// The grid is the CONSUMER's arithmetic and not this builder's: a sheet is a
// frame count, and which of those frames is image cell (k, c) is the class's
// TileWidth times k plus c. So what is asserted here is the count, the sizes and
// that the stream decodes — never a block base.
func TestStructureSheetBuilder(t *testing.T) {
	const tw, fh = 3, 5
	sheet := synth.StructureSheet(synth.StructureSheetOptions{
		Frames:  tw * fh,
		Width:   32,
		Height:  32,
		Palette: sheetPalette(),
		Ink:     func(frame, x, y int) (uint8, bool) { return uint8(frame), x == y },
	})
	s := decodeSheet(t, sheet)
	if len(s.Frames) != tw*fh {
		t.Fatalf("%d frames, want %d", len(s.Frames), tw*fh)
	}
	for i, f := range s.Frames {
		if f.Width != 32 || f.Height != 32 {
			t.Fatalf("frame %d is %dx%d, want 32x32", i, f.Width, f.Height)
		}
		// The ink is a function of the frame index, so a consumer that mixed two
		// grid indices up would draw a frame whose pixels say which one it is.
		if px := f.Pixels[0]; !px.Opaque || int(px.Index) != i {
			t.Fatalf("frame %d pixel (0,0) = %+v, want an opaque index %d", i, px, i)
		}
	}

	t.Run("one frame that is not CellSize square", func(t *testing.T) {
		odd := synth.StructureSheet(synth.StructureSheetOptions{
			Frames:  4,
			Width:   32,
			Height:  32,
			Odd:     map[int][2]int{2: {32, 31}},
			Palette: sheetPalette(),
		})
		frames := decodeSheet(t, odd).Frames
		if len(frames) != 4 {
			t.Fatalf("%d frames, want 4", len(frames))
		}
		if frames[2].Width != 32 || frames[2].Height != 31 {
			t.Fatalf("frame 2 is %dx%d, want 32x31", frames[2].Width, frames[2].Height)
		}
		for i, f := range frames {
			if i != 2 && (f.Width != 32 || f.Height != 32) {
				t.Fatalf("frame %d is %dx%d, want 32x32 — only frame 2 breaks the size", i, f.Width, f.Height)
			}
		}
	})

	t.Run("a palette-less structure sheet", func(t *testing.T) {
		bare := synth.StructureSheet(synth.StructureSheetOptions{
			Frames: 2, Width: 32, Height: 32, NoPalette: true,
		})
		s := decodeSheet(t, bare)
		if s.HasPalette {
			t.Error("HasPalette is set on a NoPalette sheet")
		}
		if len(s.Frames) != 2 {
			t.Errorf("%d frames, want 2 — only the palette is missing", len(s.Frames))
		}
	})
}
