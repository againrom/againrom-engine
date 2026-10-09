package ui

import "image"

// MenuPanelArt carries the installed lm.256 large window pieces in source order.
// Geometry follows the owner's options and objectives references.
type DialogFrame struct {
	Pieces   [9]*image.RGBA
	Portrait *image.RGBA
	// PortraitBack is the 160x240 picture a dialogue pane shows behind every
	// speaker, cut at that speaker's own window (DIV-1485). The original draws
	// it first, opaque, then the speaker's picture keyed over it
	// (`REG-NPC-089`, `TOWN-150`).
	PortraitBack *image.RGBA
	Minimap      *image.RGBA
	MinimapSeam  *image.RGBA
	helpScroll   []*image.RGBA
}

// MenuPanelArt retains the existing loader seam. All modal windows use the
// same DialogFrame, including mission notices and town conversations.
type MenuPanelArt = DialogFrame

func (art *DialogFrame) DrawPortrait(dst *image.RGBA, r image.Rectangle) {
	if art == nil || art.Portrait == nil {
		return
	}
	// The installed border is 88x108. Keep the existing 72x96 face crop
	// and its origin; extend only the border's straight middle for tall panes.
	drawFrame(dst, frameSpec{Kind: framePortrait, Rect: r, Art: art})
}
