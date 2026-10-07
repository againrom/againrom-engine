package ui

import (
	"image"
	"image/draw"
)

// TownPane is one 160-wide column slot's shipped presentation: an opaque
// body and, where the neighbouring content does not already cover the 16
// columns beside it, the keyed strip that closes them (drawTownPane's own
// doc comment gives the decoded evidence for the keyed half). Every screen
// of the town family composes its columns from a TownPane instead of
// computing a body or a seam rectangle of its own (spec B1, the owner's own
// proposal: "нужно ввести LeftPane, RightPane и везде их
// использовать" — introduce a left-slot and right-slot pane
// type and use them everywhere). One shared struct, not two named side
// types: drawTownPane's own body/seam draw is identical at every call site
// regardless of which of a room's two 160-wide columns it fills, so a side
// discriminator would carry no branch and no caller would read it. A slot
// whose neighbour already covers its own 16 columns — the tavern's own
// left column, next to its Center picture — carries a TownPane with a nil
// Seam.
type TownPane struct {
	Body image.Image
	Seam image.Image
}

// drawTownPane draws p's Body opaque at bodyRect and, if p carries one, its
// Seam at seamRect. Either image may be nil: a nil Body leaves bodyRect for
// the caller's own fallback (drawTownShellBox's own authored fill, on every
// call site below); a nil Seam leaves seamRect exactly as already composed,
// which is correct where a neighbouring bitmap — not this pane — already
// covers those columns.
func drawTownPane(dst *image.RGBA, p TownPane, bodyRect, seamRect image.Rectangle) {
	if p.Body != nil {
		b := p.Body.Bounds()
		draw.Draw(dst, bodyRect, p.Body, b.Min, draw.Src)
	}
	if p.Seam != nil {
		b := p.Seam.Bounds()
		draw.Draw(dst, seamRect, p.Seam, b.Min, draw.Over)
	}
}
