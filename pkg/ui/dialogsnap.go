package ui

import (
	"image"

	"againrom/pkg/render/frame"
)

// The shared dialog base (MENU-077). A dialog is built with an argument
// rectangle; the base snaps its size to whole frame tiles and centres it on the
// screen. The painted frame is the lm.256 nine-piece art with an 8 pixel
// shadow band along the right and bottom, so the frame body is eight pixels
// smaller than the snapped size on both axes.
const (
	dialogSnapTileW  = 96
	dialogSnapTileH  = 64
	dialogSnapBaseW  = 8
	dialogSnapBaseH  = 104
	dialogShadowBand = 8
)

// snapDialogSize is the base's width and height rule, integer division
// truncating toward zero.
func snapDialogSize(w, h int) (int, int) {
	return (w-dialogSnapBaseW)/dialogSnapTileW*dialogSnapTileW + dialogSnapBaseW,
		(h-dialogSnapBaseH)/dialogSnapTileH*dialogSnapTileH + dialogSnapBaseH
}

// dialogGeometry is a snapped dialog's rectangle in frame coordinates,
// shadow included. Every control of a dialog is placed relative to its
// origin, with W the snapped width.
type dialogGeometry struct{ image.Rectangle }

// newDialogGeometry snaps the argument size and centres it on the 640x480
// frame.
func newDialogGeometry(w, h int) dialogGeometry {
	sw, sh := snapDialogSize(w, h)
	x, y := (frame.W-sw)/2, (frame.H-sh)/2
	return dialogGeometry{image.Rect(x, y, x+sw, y+sh)}
}

// W is the snapped width, the value the builders' right-edge offsets use.
func (g dialogGeometry) W() int { return g.Dx() }

// Rect places a control rectangle given relative to the dialog origin.
func (g dialogGeometry) Rect(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rect(g.Min.X+x0, g.Min.Y+y0, g.Min.X+x1, g.Min.Y+y1)
}

// Body is the nine-piece frame's rectangle: the snapped size without the
// shadow band.
func (g dialogGeometry) Body() image.Rectangle {
	return image.Rect(g.Min.X, g.Min.Y, g.Max.X-dialogShadowBand, g.Max.Y-dialogShadowBand)
}

// frame is the dialog's window frame: the base's snap rule over the
// argument rectangle, which the geometry already holds snapped.
func (g dialogGeometry) frame(art *DialogFrame) frameSpec {
	return frameSpec{Kind: frameWindow, Rect: g.Rectangle, Snap: true, Art: art}
}

// The three argument rectangles that reach the base from the options builders
// (MENU-073, MENU-075).
var (
	gameOptionsDialog  = newDialogGeometry(560, 480)
	soundOptionsDialog = newDialogGeometry(540, 420)
)
