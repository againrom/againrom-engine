package terrain

import "image/color"

// shadeDivisor is the per-entry divisor: the game does (chan * (96-level)) / 32
// with a trunc-toward-zero shift (TERR-LIGHT-019).
const shadeDivisor = 32

// clampLevel holds a level in the valid attenuation range [0, LevelCount-1],
// the one clamp ShadeChannel and ShadeScale both apply so the two cannot
// disagree about an out-of-range level.
func clampLevel(level int) int {
	if level < 0 {
		return 0
	}
	if level > LevelCount-1 {
		return LevelCount - 1
	}
	return level
}

// ShadeChannel attenuates one 8-bit channel by an attenuation level, exactly as
// TERR-LIGHT-019: clamp( ((chan + tint) * (LevelCount - level)) / 32, 0, 255 ),
// with integer division truncating toward zero. The multiplier (LevelCount -
// level)/32 is 3.0 at level 0, 1.0 at level 64 (the unattenuated identity row,
// TERR-LIGHT-020) and 1/32 at level 95; higher level is darker. The level is
// clamped into [0, LevelCount-1] so the function is total.
func ShadeChannel(ch, tint uint8, level int) uint8 {
	level = clampLevel(level)
	// (ch + tint) is in [0,510] and (LevelCount - level) in [1,96], so the
	// product cannot overflow an int; the result is non-negative before the
	// upper clamp.
	v := (int(ch) + int(tint)) * (LevelCount - level) / shadeDivisor
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// ShadeScale returns the per-channel multiplier m(level) = (LevelCount -
// level)/32 that ShadeChannel applies at that level: 3.0 at level 0, 1.0 at
// level 64 (the identity row) and 1/32 at level 95. The level is clamped
// into [0, LevelCount-1] through the same clampLevel ShadeChannel uses, so
// the two functions cannot disagree about an out-of-range level. Every value
// this can return is k/32 for an integer k in [1,LevelCount], which is
// exactly representable in float32 -- 3.0, 1.5625, 1.0 and 0.03125 are
// exact, and no rounding enters here.
func ShadeScale(level int) float32 {
	return float32(LevelCount-clampLevel(level)) / 32
}

// CornerLevels returns tile (col,row)'s four corner levels out of a w*h
// per-vertex level grid (as LevelGrid produces), in TL, TR, BL, BR order --
// the order the vertex builders consume. The four vertices are (col,row),
// (col+1,row), (col,row+1) and (col+1,row+1); a w*h grid carries no vertex
// beyond the last row/column, so every index is clamped into [0,w-1] x
// [0,h-1] before it is read. That is the single far-edge clamp
// CompositeLit's lit.go and compositeProjected's project.go each wrote
// independently before this function unified them -- the same defined choice
// for a region the engine itself defines no value for, not a second
// convention.
//
// CornerLevels is total: a grid shorter than w*h, a non-positive dimension, or
// a (col,row) outside the grid each answer the zero value or a clamped read
// rather than panicking or indexing out of range.
func CornerLevels(levels []uint8, w, h, col, row int) [4]uint8 {
	if w < 1 || h < 1 || len(levels) < w*h {
		return [4]uint8{}
	}
	at := func(c, r int) uint8 {
		if c < 0 {
			c = 0
		} else if c > w-1 {
			c = w - 1
		}
		if r < 0 {
			r = 0
		} else if r > h-1 {
			r = h - 1
		}
		return levels[r*w+c]
	}
	return [4]uint8{at(col, row), at(col+1, row), at(col, row+1), at(col+1, row+1)}
}

// ShadeRGBA attenuates a palette-resolved colour per channel by a level, adding
// the sky tint per channel before the multiply (TERR-LIGHT-021). Alpha is forced
// opaque: terrain pixels are fully covering. Output is 8-bit RGBA (the pre-pack
// value; RGB565 packing is out of scope).
func ShadeRGBA(c color.RGBA, tint [3]uint8, level int) color.RGBA {
	return color.RGBA{
		R: ShadeChannel(c.R, tint[0], level),
		G: ShadeChannel(c.G, tint[1], level),
		B: ShadeChannel(c.B, tint[2], level),
		A: 0xff,
	}
}

// identityLevel is the one level whose shading multiplier is exactly 1:
// (LevelCount-level)/shadeDivisor reduces to 1 iff level ==
// LevelCount-shadeDivisor, i.e. 64. It is spelled as that subtraction,
// through the two constants the multiplier is already built from, rather
// than as a bare 64: LevelCount and shadeDivisor are what make the
// multiplier what it is, so the level that sends it to 1 is honestly a fact
// about them, not a third number beside them.
const identityLevel = LevelCount - shadeDivisor

// TintChannel adds a sky tint to one already palette-resolved 8-bit channel
// with no attenuation at all: ShadeChannel at identityLevel, which is
// exactly clamp(ch+tint, 0, 255). The viewer's ground texture cache builds a
// cell's tinted pixels through this rather than through a second clamp
// spelled out beside it, so the additive tint a cache key carries and the
// pixels it maps to are the shading transform itself, read at its own
// identity row, and cannot drift from what ShadeChannel does at every other
// level.
func TintChannel(ch, tint uint8) uint8 {
	return ShadeChannel(ch, tint, identityLevel)
}

// InterpRow returns the attenuation row for a pixel at (x, y) inside a
// cell-wide square, as the truncated bilinear interpolation of the cell's
// four corner levels (TERR-LIGHT-011). Corners are l00 = top-left (the
// cell's own vertex), l10 = top-right, l01 = bottom-left, l11 =
// bottom-right.
func InterpRow(l00, l10, l01, l11 uint8, x, y, cell int) int {
	return InterpSpan(l00, l10, l01, l11, x, y, cell, cell)
}

// InterpSpan is InterpRow with the two axes' denominators given separately: the
// fraction is x/spanX, y/spanY, and everything else — the corner order, the
// bilinear blend, the truncation — is InterpRow's arithmetic unchanged. InterpRow
// is exactly the equal-denominator call InterpSpan(.., cell, cell).
//
// The axes part company on a height-displaced cell: the level is still
// interpolated across the cell's 32 columns, but down a drawn span whose
// height is the column's own S(i) rather than the 32 rows of a square.
// Passing that span here keeps the level ramp inside the pixels actually
// drawn, and keeps the one place a level is computed a single place.
//
// A non-positive denominator on either axis has no fraction to take and returns
// the top-left corner level, as InterpRow's own degenerate case always did.
func InterpSpan(l00, l10, l01, l11 uint8, x, y, spanX, spanY int) int {
	if spanX <= 0 || spanY <= 0 {
		return int(l00)
	}
	fx := float64(x) / float64(spanX)
	fy := float64(y) / float64(spanY)
	top := float64(l00)*(1-fx) + float64(l10)*fx
	bot := float64(l01)*(1-fx) + float64(l11)*fx
	return int(top*(1-fy) + bot*fy)
}
