package spr256

import (
	"image"
	"image/color"

	"againrom/pkg/formats/pal"
)

// Table is the sheet's own palette at full opacity, or false for a sheet that
// carries none. A table entry has no alpha: which pixels are holes is the
// frame's own structural answer, index 0 included.
func (s *Sprite) Table() (*[pal.EntryCount]color.RGBA, bool) {
	if s == nil || !s.HasPalette {
		return nil, false
	}
	t, ok := pal.TableOf(s.Palette)
	if !ok {
		return nil, false
	}
	out := t.Opaque()
	return &out, true
}

// Colors resolves every pixel through table: an opaque pixel takes the entry
// its index names, a transparent one stays the zero colour. The table is a
// caller option because a sheet without a palette is drawn through a shared
// table its registry row names (PAL-PROJ-011).
func (f Frame) Colors(table *[pal.EntryCount]color.RGBA) []color.RGBA {
	out := make([]color.RGBA, len(f.Pixels))
	for i, p := range f.Pixels {
		if p.Opaque {
			out[i] = table[p.Index]
		}
	}
	return out
}

// RGBA is Colors on an image of the frame's size.
func (f Frame) RGBA(table *[pal.EntryCount]color.RGBA) *image.RGBA {
	return pal.Picture(f.Width, f.Height, f.Colors(table))
}
