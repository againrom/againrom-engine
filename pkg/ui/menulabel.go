package ui

import (
	"image"

	"againrom/pkg/render/debugtext"
	"againrom/pkg/render/menu"
)

// menuLabelMargin keeps the label off the frame's right and bottom edges.
const menuLabelMargin = 4

// SetMenuLabel sets the small text the main menu shows in its bottom-right
// corner, where the frame holds only background and no button. An empty label
// shows nothing, which is every caller's state until it asks.
func (a *App) SetMenuLabel(label string) {
	a.menuLabel = label
	a.hasMenu = false
}

// menuLabelBounds is the rectangle the label's glyph cells occupy on the
// menu frame, or the empty rectangle when there is no label.
func menuLabelBounds(label string) image.Rectangle {
	if label == "" {
		return image.Rectangle{}
	}
	w := len([]rune(label)) * debugtext.CellWidth
	x := menu.FrameW - w - menuLabelMargin
	y := menu.FrameH - debugtext.LineHeight - menuLabelMargin/2
	return image.Rect(x, y, x+w+1, y+debugtext.LineHeight)
}

// drawMenuLabel paints the label onto a composed menu frame.
func (a *App) drawMenuLabel(pix *image.RGBA) {
	if r := menuLabelBounds(a.menuLabel); !r.Empty() {
		debugtext.Draw(pix, a.menuLabel, r.Min.X, r.Min.Y)
	}
}
