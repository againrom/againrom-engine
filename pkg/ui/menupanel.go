package ui

import (
	"image"
	"image/draw"
)

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

func drawMenuPanel(dst *image.RGBA, r image.Rectangle, art *MenuPanelArt) {
	art.Draw(dst, r)
}

func (art *DialogFrame) Draw(dst *image.RGBA, r image.Rectangle) {
	if !art.valid() {
		drawTownShellBox(dst, r, false)
		return
	}
	tileBlitSrc(dst, r.Inset(16), art.Pieces[0])
	for _, edge := range []struct {
		i int
		r image.Rectangle
	}{
		{4, image.Rect(r.Min.X, r.Min.Y+48, r.Min.X+48, r.Max.Y-48)},
		{5, image.Rect(r.Max.X-48, r.Min.Y+48, r.Max.X, r.Max.Y-48)},
	} {
		pic := art.Pieces[edge.i]
		tileBlitOver(dst, edge.r, pic, pic.Bounds())
	}
	for _, edge := range []struct{ i, y int }{{2, r.Min.Y}, {7, r.Max.Y - 48}} {
		pic := art.Pieces[edge.i]
		sy := 0
		if edge.i == 7 {
			sy = 32
		}
		// Repeat only the straight border, not one vertical column of its
		// interior texture: that would produce horizontal bands behind titles.
		tileBlitOver(dst, image.Rect(r.Min.X+48, edge.y+sy, r.Max.X-48, edge.y+sy+16), pic, image.Rect(0, sy, 1, sy+16))
		x := (r.Min.X + r.Max.X - pic.Bounds().Dx()) / 2
		draw.Draw(dst, image.Rect(x, edge.y+sy, x+pic.Bounds().Dx(), edge.y+sy+16), pic, image.Pt(0, sy), draw.Over)
	}
	for _, corner := range []struct{ i, x, y int }{
		{1, r.Min.X, r.Min.Y}, {3, r.Max.X - 48, r.Min.Y},
		{6, r.Min.X, r.Max.Y - 48}, {8, r.Max.X - 48, r.Max.Y - 48},
	} {
		pic := art.Pieces[corner.i]
		draw.Draw(dst, pic.Bounds().Add(image.Pt(corner.x, corner.y)), pic, image.Point{}, draw.Over)
	}
}

func (art *DialogFrame) valid() bool {
	if art == nil {
		return false
	}
	for _, p := range art.Pieces {
		if p == nil || p.Bounds().Empty() {
			return false
		}
	}
	return true
}

func (art *DialogFrame) DrawPortrait(dst *image.RGBA, r image.Rectangle) {
	if art == nil || art.Portrait == nil {
		return
	}
	// The installed border is 88x108. Keep the existing 72x96 face crop
	// and its origin; extend only the border's straight middle for tall panes.
	drawNinePatchBorder(dst, r, art.Portrait, 22, 27)
}
