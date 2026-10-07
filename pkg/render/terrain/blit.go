package terrain

import (
	"image"
	"image/color"
)

// blitWalkable holds the four refusals, in one place for both entry points:
// a nil destination, a nil frame, a non-positive dimension, and a Pixels
// slice shorter than the Width*Height its header claims. They are about the
// FRAME being walkable and never about where it lands or how bright it is,
// so lighting adds no fifth — an out-of-range row is held into range by
// the ramp rather than refused, and no caller gains an error path.
func blitWalkable(dst *image.RGBA, f *StaticFrame) bool {
	return dst != nil && f != nil && f.Width > 0 && f.Height > 0 && len(f.Pixels) >= f.Width*f.Height
}

// blitPalette is THE pixel loop of this layer, and the only one: it walks the
// frame's clipped rectangle and writes, for each opaque source pixel, the colour
// pal selects at that pixel's index.
//
// pal is the frame's own palette for an unshaded blit and the ramp row built
// from it for a lit one, which is the whole of the difference between them. Both
// callers have already run blitWalkable, so nothing is re-tested here.
//
// THE CLIP IS RECT ARITHMETIC, DONE ONCE. The frame's translated rectangle
// is intersected with dst.Bounds() and only the surviving rows and columns
// are walked, so no bounds test survives in the inner loop: the store is a
// direct four-byte write into dst.Pix rather than SetRGBA, whose per-pixel
// guard would be exactly the test the intersection just removed, and whose
// bounds check covers the destination only and never the source side. A
// position leaving the frame WHOLLY off dst needs no case of its own and
// gets none — the intersection is empty, both loops run zero times, and
// the call falls out as a no-op from that arithmetic instead of from a guard
// bolted in front of it.
//
// The intersection is against dst.Bounds() and the store addresses dst through
// PixOffset, so a destination that does not begin at the origin — a sub-image —
// is clipped to ITS OWN rectangle and nothing outside it is read or written.
func blitPalette(dst *image.RGBA, f *StaticFrame, destX, destY int, pal *[256]color.RGBA) {
	// The one clip. Intersect normalises every empty overlap to the zero
	// rectangle, so an off-image position arrives here as Min == Max and the
	// loops below simply do not run.
	clip := image.Rect(destX, destY, destX+f.Width, destY+f.Height).Intersect(dst.Bounds())

	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		// Both indices are inside their slices by the clip's own construction:
		// clip is a sub-rectangle of the frame's translated rectangle, so
		// y-destY is a row of the frame and x-destX a column of it.
		row := f.Pixels[(y-destY)*f.Width:]
		o := dst.PixOffset(clip.Min.X, y)
		for x := clip.Min.X; x < clip.Max.X; x++ {
			if px := row[x-destX]; px.Opaque {
				c := pal[px.Index]
				dst.Pix[o+0] = c.R
				dst.Pix[o+1] = c.G
				dst.Pix[o+2] = c.B
				dst.Pix[o+3] = 0xff
			}
			o += 4
		}
	}
}

// spriteRampRow resolves a whole 256-entry palette through the sprite ramp
// at one row and one sky tint.
//
// It is built PER BLIT and returned by value: nothing here is cached and no row
// outlives the call that built it. A materialised [16][256] table per sheet was
// rejected — 16 KB per frame held for the session, to answer one row per render,
// and it would be the first materialised shading table in the tree against the
// standing decision to apply the transform arithmetically.
func spriteRampRow(pal *[256]color.RGBA, tint [3]uint8, row int) [256]color.RGBA {
	var out [256]color.RGBA
	for i, c := range pal {
		out[i] = SpriteRGBA(c, tint, row)
	}
	return out
}

func BlitStatic(dst *image.RGBA, f *StaticFrame, destX, destY int) {
	if !blitWalkable(dst, f) {
		return
	}
	blitPalette(dst, f, destX, destY, &f.Palette)
}

// BlitStaticLit is BlitStatic through the sprite shading ramp: the same
// walk, the same clip, the same refusals and the same coverage, over the
// frame's palette resolved at one sky tint and one ramp row.
//
// SHADING CHANGES COLOURS AND NEVER COVERAGE. The set of destination pixels this
// writes is exactly the set BlitStatic writes for the same frame and position, at
// every row: a transparent frame pixel stays a no-op leaving the bytes beneath it
// as they were, every painted pixel is written fully opaque, and the palette
// entry's own alpha is no more read here than it is there.
//
// The row is the render's own, from SpriteRow, and is held into
// [0, SpriteRowCount-1] by the ramp — so this is total on an out-of-range row
// rather than refusing one. The tint is the sun's sky tint, added per channel
// before the multiply; our own sun's is (0,0,0) everywhere, so the parameter is
// carried for the transform's shape rather than for a value anything produces.
//
// It is a SEPARATE ENTRY POINT rather than a widened BlitStatic. The
// unshaded diagnostic keeps its own name and its own unchanged call, which
// is what makes "byte for byte as they are drawn today" a property of an
// untouched code path instead of of a re-derived argument. A boolean beside
// the row was rejected: row 8 at zero tint is already the raw palette
// exactly, so the two spellings could only ever disagree by being wrong.
func BlitStaticLit(dst *image.RGBA, f *StaticFrame, destX, destY int, tint [3]uint8, row int) {
	if !blitWalkable(dst, f) {
		return
	}
	lit := spriteRampRow(&f.Palette, tint, row)
	blitPalette(dst, f, destX, destY, &lit)
}

// RGBA returns the frame as a standalone image: a fresh, wholly transparent
// canvas the size of the frame with THIS PACKAGE'S OWN BLIT run onto it at
// (0, 0).
//
// It is the blit written over itself and not a second walk of the palette,
// for the reason this file opens with: the raster's pixels and the window's
// texture are one rule applied at two places, a wrong palette index is
// invisible to anyone looking at either, and so two implementations of it
// could diverge per pixel with both looking plausible and nothing anywhere
// able to tell. The window builds its GPU texture from this, which is what
// keeps the one walk one.
//
// Transparent frame pixels are left at the canvas's own zero — alpha 0, and RGB
// 0 beneath it — so a see-through hole reaches a texture as transparency rather
// than as black at full alpha.
//
// Every call allocates its own image and nothing is cached here: the window
// caches by frame identity instead, and a shared canvas would let one
// caller's mutation reach another's texture.
//
// A nil receiver or a frame with no area yields an empty image rather than nil
// or a panic — it is the canvas this blit would have written into, and it is
// exactly as empty. A frame whose Pixels the blit will not walk still yields an
// image of the size the frame claims, wholly transparent.
func (f *StaticFrame) RGBA() *image.RGBA {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return image.NewRGBA(image.Rectangle{})
	}
	dst := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	BlitStatic(dst, f, 0, 0)
	return dst
}

// RGBALit is RGBA through the sprite shading ramp: this package's own lit
// blit run onto a fresh, wholly transparent canvas the size of the frame.
//
// It is BlitStaticLit written over itself for the reason RGBA is BlitStatic
// written over itself, and the reason is stronger here rather than weaker: the
// window's texture and the raster tool's pixels must now agree about a SHADED
// colour as well as a palette index, and neither has any ground truth on screen.
// A second lit walk could diverge per pixel with both looking plausible.
//
// Transparent frame pixels are left at the canvas's own zero — alpha 0, and RGB
// 0 beneath it — at every row: the ramp never reaches them, because the loop
// never writes them.
//
// A nil receiver or a frame with no area yields an empty image, exactly as RGBA
// does, and for the same reason: it is the canvas this blit would have written
// into, and it is exactly as empty.
func (f *StaticFrame) RGBALit(tint [3]uint8, row int) *image.RGBA {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return image.NewRGBA(image.Rectangle{})
	}
	dst := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	BlitStaticLit(dst, f, 0, 0, tint, row)
	return dst
}
