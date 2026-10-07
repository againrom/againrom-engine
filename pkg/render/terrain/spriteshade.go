package terrain

import "image/color"

// SpriteRowCount is the number of rows in the sprite ramp: 16, row 0 the
// brightest at gain 2.0 and row 15 the darkest at 0.125 (TERR-LIGHT-060).
const SpriteRowCount = 16

// spriteRowDivisor is the ramp's divisor. The gain at row L is
// 2*(SpriteRowCount-L)/spriteRowDivisor, so row 8 is exactly 1.0 — the row that
// reproduces the raw palette wherever the tint is zero (TERR-LIGHT-062).
const spriteRowDivisor = 16

// spriteAmbientShift is how far the sun's ambient byte is shifted down to reach
// a ramp row: ambient>>2 (TERR-LIGHT-061).
const spriteAmbientShift = 2

// clampSpriteRow holds a row in [0, SpriteRowCount-1], the one clamp
// SpriteChannel and SpriteRow both apply so the two cannot disagree about an
// out-of-range row. It is what makes every entry point here total: a row is an
// ordinary int at the call sites, and ambient>>2 exceeds 15 from ambient 64 up.
func clampSpriteRow(row int) int {
	if row < 0 {
		return 0
	}
	if row > SpriteRowCount-1 {
		return SpriteRowCount - 1
	}
	return row
}

// SpriteChannel shades one 8-bit palette channel at a ramp row, exactly as
// TERR-LIGHT-060:
//
//	clamp( ((ch + tint) * (SpriteRowCount - row) * 2) / spriteRowDivisor, 0, 255 )
//
// with the division truncating toward zero. The row is clamped into
// [0, SpriteRowCount-1] first.
//
// THE SUM IS NOT CLAMPED BEFORE THE MULTIPLY. ch+tint reaches 510 and is carried
// into the product whole; the single clamp is the one written, at the end. A
// pre-multiply clamp is a different function wherever the gain is below 1 — at
// ch=200, tint=100, row=10 it answers 191 where this answers 225 — so the
// ordering is transcribed rather than tidied.
//
// The gain above 1.0 (every row below 8) SATURATES per channel at 255 rather
// than wrapping or rescaling, so bright palette entries flatten toward white at
// the low rows. That is the terrain transform's own behaviour (TERR-LIGHT-063,
// which measures the mapping reaching white), on art where it is more visible.
func SpriteChannel(ch, tint uint8, row int) uint8 {
	row = clampSpriteRow(row)
	// (ch + tint) is in [0,510] and (SpriteRowCount - row)*2 in [2,32], so the
	// product cannot overflow an int and is non-negative before the upper clamp.
	v := (int(ch) + int(tint)) * (SpriteRowCount - row) * 2 / spriteRowDivisor
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// SpriteRGBA shades one palette entry at a ramp row, per channel, adding the
// sky tint per channel before the multiply — the same tint the terrain transform
// adds (TERR-LIGHT-060/021).
//
// Alpha is forced opaque and the entry's OWN alpha is never read: a palette
// entry is a colour at full opacity and nothing may read its alpha as a
// transparency channel, so an entry a loader left at zero alpha shades to its
// colour rather than to an invisible pixel. Which pixels are holes is
// StaticPixel.Opaque's answer alone.
//
// The result depends on this entry, the tint and the row and on nothing else —
// never on another sprite's palette and never on the terrain's, which no sprite
// sheet carries in any case (TERR-LIGHT-064).
func SpriteRGBA(c color.RGBA, tint [3]uint8, row int) color.RGBA {
	return color.RGBA{
		R: SpriteChannel(c.R, tint[0], row),
		G: SpriteChannel(c.G, tint[1], row),
		B: SpriteChannel(c.B, tint[2], row),
		A: 0xff,
	}
}

// SpriteRow is the ramp row EVERY sprite of one rendered frame is drawn at:
// clamp(ambient>>2, 0, SpriteRowCount-1), from the ambient byte of the same sun
// the terrain level grid is built from (TERR-LIGHT-061).
//
// It takes a whole Light and there is no second entry point that lets a caller
// name a row without one — which is what keeps the sun's parameters reaching
// sprites through no switch of their own, and keeps one physical light from
// meaning one thing to terrain and another to sprites.
//
// NOTHING ELSE OF THE LIGHT REACHES A SPRITE. Theta, Range and the level grid do
// not enter: the decoded row carries no relief term, so altitude, the terrain's
// per-vertex level and which cell a sprite stands on are all absent by
// construction rather than by discipline. The clamp is ours — the grid holds a
// byte and the table has 16 rows, and no published path bounds one to the other.
func SpriteRow(lt Light) int {
	return clampSpriteRow(int(lt.Ambient) >> spriteAmbientShift)
}
