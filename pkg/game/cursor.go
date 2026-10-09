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
	return attackPointer(loadCursorSlot(src, attackRegistration))
}

// attackPointer is frame attackFrame0 of the attack sheet's decoded frames.
func attackPointer(frames []*image.RGBA, err error) (*image.RGBA, error) {
	if err != nil {
		return nil, err
	}
	if attackFrame0 >= len(frames) {
		return nil, fmt.Errorf("%s holds %d frames, with no frame %d to draw", AttackCursorPath, len(frames), attackFrame0)
	}
	return frames[attackFrame0], nil
}

// cursorPixel is spr16.Resolve under its former name, kept only for the
// world-map loader until the town composer moves it onto the frame converter.
func cursorPixel(palette []spr16.Color, p spr16.PixelA) color.RGBA { return spr16.Resolve(palette, p) }
