package game

import (
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/render/terrain"
)

// AttackCursorPath is the entry the attack cursor's art lives at, carrying
// graphics.res's identity segment like every other address this package reads.
//
// It is READ OUT OF THE IMAGE and not chosen: the routine that registers the
// engine's 28 cursors pairs each art path with the global it stores the handle
// in, and this path is the one paired with the global the hover routine sets the
// attack cursor from. So the picture below is the game's own attack pointer and
// not a picture that resembles it.
const AttackCursorPath = graphicsPrefix + "cursors/attack/sprites.16a"

// attackFrame0 is the single frame LoadAttackPointer resolves: the sheet's
// own frame 0, used as the still picture handed to a caller that wants one
// picture rather than the animated set (docs/1030-cursor-lifecycle B1, B3).
// Animation itself is the cursor registry's (cursorregistry.go) and the
// shared manager's (pkg/ui), not this function's.
const attackFrame0 = 0

// The .16a literal pixel's own two fields, as the resolved pixel model gives
// them: a 4-bit level whose source multiplier is level+1 out of 16.
//
// COVERAGE IS APPLIED HERE, AT THE LOAD, and not at the draw. The decoded
// model states the multiplier at the blit; expressing it as a premultiplied
// alpha in the picture is that same composite performed once per load
// instead of once per frame, and it is what lets the result reach the engine
// through the upload path the front-end's three boxes already use, with no
// draw-time option to keep in step.
const (
	cursorLevels   = 16
	cursorFullByte = 0xff
)

// LoadAttackPointer reads the attack cursor's art out of src and resolves it to
// one premultiplied picture.
//
// Every failure is an error and none is a partial picture. A read error comes
// back UNWRAPPED, so the source's own path error still names the address a
// caller can test with errors.Is; a decode error is wrapped with the address,
// since the decoder does not know which entry it was handed. A sheet with no
// frame at the index is its own error naming both the index and the count — it
// is a property of the sheet rather than of the bytes, exactly as the font's
// count mismatch is.
//
// THE PALETTE IS DECLARED PRESENT, never inferred. That declaration is the
// decoder's contract and it is correct here for a reason rather than by trial:
// the container's own trailer carries a has-palette flag which is set on every
// palette-bearing sheet in the archive, and the loader the engine sends a .16a
// through is the one that gates a 1024-byte palette read on it.
//
// A ZERO-AREA FRAME IS REFUSED. The container accepts one, and it would resolve
// to a picture with no pixels — a pointer that draws nothing, which is
// indistinguishable at the draw from the mode being down and is exactly the
// failure this whole story exists to remove.
func LoadAttackPointer(src terrain.EntrySource) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", AttackCursorPath)
	}

	b, err := src.ReadFile(AttackCursorPath)
	if err != nil {
		return nil, err
	}
	sprite, err := spr16.DecodeA(b, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", AttackCursorPath, err)
	}
	if attackFrame0 >= len(sprite.Frames) {
		return nil, fmt.Errorf("%s holds %d frames, with no frame %d to draw",
			AttackCursorPath, len(sprite.Frames), attackFrame0)
	}

	f := sprite.Frames[attackFrame0]
	if f.Width <= 0 || f.Height <= 0 {
		return nil, fmt.Errorf("%s frame %d is %dx%d and would draw nothing",
			AttackCursorPath, attackFrame0, f.Width, f.Height)
	}

	pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i, p := range f.Pixels {
		if !p.Painted {
			// The zero value, and it is the RIGHT absence: a painted level of 0
			// is a written pixel that is nearly transparent, and an unpainted
			// cell is no pixel at all. The decoder keeps the two apart and so
			// does this.
			continue
		}
		c := cursorPixel(sprite.Palette, p)
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, c.A
	}
	return pic, nil
}

// cursorPixel resolves one painted .16a cell to a premultiplied RGBA.
//
// The colour is the file's own palette entry at the pixel's index, and the
// coverage is (level+1) parts in 16 — the source multiplier the pixel model
// gives, carried straight into alpha. Premultiplied, because that is what the
// engine's upload wants and what Go's own RGBA means.
//
// An index past the palette is BLACK AT THE PIXEL'S OWN COVERAGE rather than an
// error: the index is eight bits and a declared palette is 256 entries, so the
// case is unreachable from a decoded stream, and the guard is here so that a
// hand-built one cannot panic in a loop over a whole frame.
func cursorPixel(pal []spr16.Color, p spr16.PixelA) color.RGBA {
	a := uint32((int(p.Level) + 1) * cursorFullByte / cursorLevels)
	var e spr16.Color
	if int(p.Index) < len(pal) {
		e = pal[p.Index]
	}
	return color.RGBA{
		R: uint8(uint32(e.R) * a / cursorFullByte),
		G: uint8(uint32(e.G) * a / cursorFullByte),
		B: uint8(uint32(e.B) * a / cursorFullByte),
		A: uint8(a),
	}
}
