package pal

import (
	"fmt"
	"image"
	"image/color"
)

// TableOffset is where the colour table begins: the engine's own seek, 0x36
// bytes into the file. It is a CONSTANT and not a field read out of the stream —
// the engine seeks to this immediate and consults no header, so a file whose
// declared pixel offset says otherwise is still read here.
const TableOffset = 0x36

// TableSize is how many bytes the table occupies: the engine's own read length,
// 0x400 — EntryCount entries of EntrySize bytes.
const TableSize = EntryCount * EntrySize

// EntryCount is the number of colours a table holds, and EntrySize the bytes
// each occupies. An entry is [B, G, R, reserved]: the fourth byte is read by
// nothing, here or in the engine.
const (
	EntryCount = 256
	EntrySize  = 4
)

// MinSize is the shortest stream a table can be read out of — the offset plus
// the table. Every shipped per-class palette is far longer, the remainder being
// the BMP's pixel data, which is never opened.
const MinSize = TableOffset + TableSize

// magic is the two bytes a stream of this shape begins with. See doc.go for why
// this check is ours and what it excludes.
var magic = [2]byte{'B', 'M'}

// Color is one table entry as an RGB triple. The on-disk order is BGR with a
// reserved fourth byte; Decode reorders it to RGB and drops the reserved byte,
// which is spr256.Color's own convention for the same 1024 bytes.
//
// It is a plain three-byte value with no alpha. A table entry is a colour at
// full opacity: which pixels of a sheet are holes is the sheet's own per-pixel
// answer and nothing in a colour table says it, index 0 included.
type Color struct {
	R, G, B uint8
}

// Table is a decoded colour table: exactly EntryCount colours, indexed by the
// byte a sheet's pixel carries.
//
// It is a fixed-size ARRAY and not a slice, so every index a uint8 can hold
// indexes it and a consumer needs neither a bounds test nor a fallback colour.
// Being an array it is also comparable, which is what lets a consumer ask
// whether two tables are the same colours without walking them.
type Table [EntryCount]Color

// Decode reads the colour table out of one palette file's bytes.
//
// It reads TableSize bytes at TableOffset and NOTHING ELSE: no header field
// decides where the table is or how long it is, and no byte before or after the
// window is consulted. Entry i is data[TableOffset+4i .. +2] read as B, G, R,
// with the fourth byte dropped.
//
// NO ENTRY IS INTERPRETED. Index 0 is decoded like every other and is given no
// meaning here — it is the transparent key by the sheet's convention, not by
// anything a table says — and no entry is clamped, reordered, de-duplicated or
// replaced.
//
// Two refusals, each naming its reason:
//
//   - a stream shorter than MinSize, which cannot hold the window at all;
//   - a stream whose first two bytes are not BM, which is the other palette
//     shape (see doc.go) or not a palette at all.
//
// It is total on every other input: nothing here indexes past a checked bound,
// so no stream can panic it, and a stream longer than MinSize is read exactly
// as one of MinSize is.
func Decode(data []byte) (Table, error) {
	if len(data) < MinSize {
		return Table{}, fmt.Errorf("pal: %d byte(s), want at least %d for a %d-byte table at 0x%x",
			len(data), MinSize, TableSize, TableOffset)
	}
	// Checked AFTER the length, so a stream too short to hold two bytes reports
	// the length it has rather than indexing for a magic that cannot be there.
	if data[0] != magic[0] || data[1] != magic[1] {
		return Table{}, fmt.Errorf("pal: stream begins %#02x %#02x, want %#02x %#02x",
			data[0], data[1], magic[0], magic[1])
	}

	var t Table
	copy(t[:], Entries(data[TableOffset:MinSize]))
	return t, nil
}

// Entries reads len(raw)/EntrySize consecutive [B, G, R, reserved] entries as
// RGB colours, dropping the reserved byte; a partial tail entry is not read.
// It is the one reader of this layout: a .pal table, the human-owner tables,
// a .256 or .16a sheet's leading palette and an 8-bit BMP's colour table.
func Entries(raw []byte) []Color { return entries(raw, EntrySize) }

// Pixels reads len(raw)/3 consecutive [B, G, R] triples, a 24-bit bitmap row's
// layout: the same order with no reserved byte.
func Pixels(raw []byte) []Color { return entries(raw, 3) }

func entries(raw []byte, size int) []Color {
	out := make([]Color, len(raw)/size)
	for i := range out {
		e := raw[i*size:]
		out[i] = Color{R: e[2], G: e[1], B: e[0]}
	}
	return out
}

// Opaque is the colour at full opacity.
func (c Color) Opaque() color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff} }

// Opaque is the table at full opacity, indexed as the table is.
func (t Table) Opaque() [EntryCount]color.RGBA {
	var out [EntryCount]color.RGBA
	for i, c := range t {
		out[i] = c.Opaque()
	}
	return out
}

// TableOf is cols as a Table, or false when cols does not hold exactly
// EntryCount colours.
func TableOf(cols []Color) (Table, bool) {
	var t Table
	if len(cols) != EntryCount {
		return t, false
	}
	copy(t[:], cols)
	return t, true
}

// Picture lays px, one colour per pixel in row-major order, onto a new
// width x height image. Pixels past the image are dropped; missing pixels stay
// transparent.
func Picture(width, height int, px []color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, c := range px {
		o := i * 4
		if o+3 >= len(pic.Pix) {
			break
		}
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, c.A
	}
	return pic
}
