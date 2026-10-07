package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// cursorTexture is the upload cache both cursor draw sites share: App.drawCursor
// for every non-map screen, and Viewer.Draw for the mission map's attack
// pointer. It rewrites the engine texture when, and only when, the source
// picture changes.
//
// ONE CACHE FOR BOTH SITES, because the two had different rules and one of them
// was wrong. The viewer's upload was gated on a "fresh" flag set once per
// viewer, at map open, and cleared by the first upload; the cursor manager then
// advanced the frame index and handed the draw path frames 1..9 of the attack
// sheet, none of which was ever written to the texture. The player saw frame 0
// standing still for the whole session. App.drawCursor already compared the
// source picture, twenty lines away, which is the rule here.
//
// THE DECISION IS SEPARATED FROM THE UPLOAD (stale, below) so a test with no
// window can assert that a new animation frame reaches the screen. An engine
// texture cannot be constructed headless, so an upload path that decides
// anything inside itself decides it where nothing can look.
type cursorTexture struct {
	img *ebiten.Image
	src *image.RGBA
}

// stale reports whether src must be written to the texture: it is not the
// picture the texture currently holds. The zero value holds none, so the first
// picture of a session is always written.
func (t *cursorTexture) stale(src *image.RGBA) bool {
	return src != nil && t.src != src
}

// upload returns the engine texture holding src, writing it first if stale. It
// allocates a new texture whenever the frame's dimensions differ from the one
// in hand, which the 28 registrations need: they are 32x32 except swarm 44x44,
// cantput 64x64 and the six 16x16 slots.
func (t *cursorTexture) upload(src *image.RGBA) *ebiten.Image {
	if !t.stale(src) {
		return t.img
	}
	b := src.Bounds()
	if t.img == nil || t.img.Bounds().Dx() != b.Dx() || t.img.Bounds().Dy() != b.Dy() {
		t.img = ebiten.NewImage(b.Dx(), b.Dy())
	}
	t.img.WritePixels(src.Pix)
	t.src = src
	return t.img
}
