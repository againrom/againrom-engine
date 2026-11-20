package ui

import (
	"image/color"

	"againrom/pkg/render/text"
)

// The hover style table: the only file that builds a hoverStyle, a
// hoverBoxInk or a hoverShape. A hover box is a description (lines, style,
// anchor) that composeHoverBox and placeHover draw; a style key names a row
// here.

// hoverStyleKey names a row of the hover style table.
type hoverStyleKey uint8

const (
	// hoverStyleShared is the shared hover help: popupTextColor, no shadow,
	// lines trimmed, blank ones dropped and wrapped. Its row is a nil style.
	hoverStyleShared hoverStyleKey = iota
	// hoverStyleGame is the game's own hover style, from the generator
	// description.
	hoverStyleGame
	// hoverStyleShop is the characteristics box beside the pointer on the
	// shop and the town screens.
	hoverStyleShop
)

// hoverStyle is how a hover box draws its text: the font when named, the
// ink, a shadow one pixel right and down when named, and, when authored,
// every '#' line as written, blank ones kept and none wrapped. A nil style
// is the shared hover help. A box set names the fill and the bevel's two
// colours in place of the hover box's own; a shape set names the box's
// geometry in place of the hover box's own.
type hoverStyle struct {
	font     *text.Font
	ink      color.RGBA
	shadow   *color.RGBA
	authored bool
	box      *hoverBoxInk
	shape    *hoverShape
}

// hoverBoxInk is a hover box's fill and its bevel's light and dark colours.
// An outlined shape draws its outline in light.
type hoverBoxInk struct{ fill, light, dark color.RGBA }

// hoverShape is a hover box's geometry: its frame, the line pitch (pitch, or
// the font's height plus pitchPad when pitch is zero), the width and height
// added to the text, the text's offset, and whether the corner picture and
// the advance width count.
type hoverShape struct {
	outlined        bool
	pitch, pitchPad int
	padW, padH      int
	textX, textY    int
	advance         bool
	corners         bool
	// clip keeps the text and the picture inside the frame's inner edge.
	clip bool
}

// hoverParams are the values a keyed style takes from its caller.
type hoverParams struct {
	ink      color.RGBA
	shadow   *color.RGBA
	authored bool
}

// The hover box's requested colours.
var (
	hoverFill  = color.RGBA{36, 44, 39, 255}
	hoverLight = color.RGBA{160, 120, 50, 255}
	hoverDark  = color.RGBA{80, 60, 24, 255}
)

// hoverSharedBox is the colours of the shared hover box.
var hoverSharedBox = hoverBoxInk{fill: hoverFill, light: hoverLight, dark: hoverDark}

// hoverSharedShape is the shape of every bevelled hover box.
var hoverSharedShape = hoverShape{
	pitch: hoverPitch, padW: hoverExtraW, padH: hoverExtraH,
	textX: hoverTextX, textY: hoverTextY, advance: true, corners: true,
}

// hoverShopShape is the shape of the characteristics box: a one-pixel
// outline, ten wide and eight tall around the text.
var hoverShopShape = hoverShape{outlined: true, pitchPad: 2, padW: 10, padH: 8, textX: 5, textY: 4}

// editorTipShape is the shape of the editor's tips in body: a one-pixel
// outline, tipPad around the lines, the lines body's height apart.
func editorTipShape(body *text.Font) hoverShape {
	return hoverShape{outlined: true, clip: true, pitch: fontHeight(body, 15),
		padW: 2 * tipPad, padH: 2 * tipPad, textX: tipPad, textY: tipPad}
}

// editorTipStyle is the style of the editor's tips in the window inks: the
// title ground framed in the edge ink, the lines in the value ink.
func editorTipStyle(body *text.Font, ink popupInk) *hoverStyle {
	shape := editorTipShape(body)
	return &hoverStyle{ink: ink.Value, box: &hoverBoxInk{fill: ink.TitleGround, light: ink.Edge}, shape: &shape}
}

// hoverStyleFor is the style of row key, filled from p where the row takes
// the caller's values. The shared row is nil.
func hoverStyleFor(key hoverStyleKey, p hoverParams) *hoverStyle {
	switch key {
	case hoverStyleGame:
		return &hoverStyle{ink: p.ink, shadow: p.shadow, authored: p.authored}
	case hoverStyleShop:
		return &hoverStyle{ink: shopTextColor, box: &hoverBoxInk{fill: shopScreenFill, light: invBorder}, shape: &hoverShopShape}
	}
	return nil
}

// shapeOf is the geometry the style names, the shared one for a nil style.
func (s *hoverStyle) shapeOf() *hoverShape {
	if s == nil || s.shape == nil {
		return &hoverSharedShape
	}
	return s.shape
}
