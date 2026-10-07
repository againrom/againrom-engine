package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/text"
)

// The floating tip panel (1018 spec behaviour 1), drawn over a screen's own
// content and reused unchanged by all five rooms that show one.
//
// THREE SHIPPED NODES COMPOSE IT: interface/t_back.bmp (160x240, a fill
// texture with no border of its own), interface/t_border.bmp (88x108, an
// ornate gold frame on a keyed black field) and interface/radiob.256 (six
// gem-toggle frames). Pairing these three as a panel's fill, frame and
// checkbox is inference from the contract's own reading of the owner's
// photographs, not a decoded fact: no claim names any of the three nodes.
// pkg/game's LoadTipPanelArt resolves and keys them; this file only composes
// and hit-tests whatever it is handed, exactly as every other room's art is
// composed here and loaded on the far side of the seam.
//
// THE FRAME'S OWN NINE-PATCH SLICE IS AUTHORED. TOWN-185 gives three child
// controls' ids, sizes and rectangles inside the original popup and nothing
// about how any of them — or the popup's own border — is painted
// ("Still open" in the contract's Research section). tipPanelBorderCornerW/H
// below quarter t_border.bmp's own 88x108 size; no claim or photograph pins
// the real corner size, and this is recorded as a divergence rather than
// left silent.
//
// THE CLOSE CONTROL'S CAPTION AND THE TOGGLE'S OWN LABEL COME FROM THE
// INSTALL. TOWN-206 resolves ids 0x7f and 0x80 to main.txt lines 127 and 128;
// pkg/game reads those bytes once into Words and carries them in each view.
// Empty fields use the EN install's exact strings so a hand-built diagnostic
// view still draws visible controls.
type TipPanelArt struct {
	// Fill is interface/t_back.bmp, tiled opaquely across the panel's own
	// interior.
	Fill image.Image
	// Border is interface/t_border.bmp, keyed so its own centre is
	// transparent, drawn as a nine-patch over the tiled fill.
	Border image.Image
	// GemOff and GemOn are two of interface/radiob.256's six frames: the
	// toggle's own unchecked and checked pictures. WHICH TWO OF THE SIX IS
	// AUTHORED (pkg/game's LoadTipPanelArt doc): the file ships three
	// shapes (round, square, small square) each as a dark/light pair, and
	// this build takes the small-square pair, frames 4 and 5, as the
	// closest fit to a checkbox next to a line of text.
	GemOff, GemOn image.Image
}

// TipControlKind is which part of a showing tip panel a point stands on.
type TipControlKind uint8

const (
	TipControlNone TipControlKind = iota
	// TipControlClose dismisses the current constructed popup.
	TipControlClose
	// TipControlToggle flips the permanent suppression every screen reads
	// at its own next entry (spec behaviour 4).
	TipControlToggle
)

// TipPanelView is one screen's own floating tip panel for one frame. The
// same type is drawn and hit-tested identically on all five rooms (spec:
// "one mechanism over five screens").
type TipPanelView struct {
	// Rect is where the panel sits on its own 640x480 frame. Chosen per
	// screen from the researched popup rectangles (TOWN-165, TOWN-015,
	// TOWN-021/TOWN-025, SHOP-TIP-045, TOWN-187) rather than authored from
	// nothing, though the panel this story draws is this build's own
	// widget and not a literal reproduction of the original's three child
	// controls.
	Rect     image.Rectangle
	Revision uint64
	// Text is the screen's own shipped tip text, already selected by the
	// game side (which install node, which of chrgen1f/chrgen1m). Empty
	// means nothing to show: a missing node, a closed panel, or a
	// suppressed one all reach this the same way, on ReadShopTip's own
	// "a widget with nothing to show is a widget that is not there" rule.
	Text string
	// ToggleOn is the checkbox's own current drawn state: true draws the
	// lit gem. It mirrors the permanent suppression store (on = tips
	// shown), read once at the screen's own entry and updated immediately
	// when the player presses the toggle, independently of Text — toggling
	// it does not blank an already-open panel, matching TOWN-186's own gate
	// being tested at construction and not against an existing popup.
	ToggleOn bool
	// CloseLabel and ToggleLabel are main.txt[127] and main.txt[128], already
	// selected by the game side. Empty retains the authored EN fallback for a
	// diagnostic view assembled without an install.
	CloseLabel, ToggleLabel string
	Art                     *TipPanelArt
	Font                    *text.Font
}

// Showing reports whether there is anything to draw or hit-test.
func (v TipPanelView) Showing() bool {
	return v.Art != nil && v.Font != nil && v.Text != "" && !v.Rect.Empty()
}

// Covers excludes only controls actually hidden by the floating panel.
func (v TipPanelView) Covers(p image.Point) bool { return v.Showing() && p.In(v.Rect) }

// The panel's own authored layout constants. None of these is decoded;
// TOWN-185's own three rectangles describe the ORIGINAL popup's controls,
// not this widget's — see the file doc.
const (
	tipPanelBorderCornerW = 22 // one quarter of t_border.bmp's 88px width
	tipPanelBorderCornerH = 27 // one quarter of t_border.bmp's 108px height

	tipPanelInset      = 10
	tipPanelButtonRowH = 18
	tipPanelCloseW     = 56
	tipPanelGemSize    = 16
	tipPanelGemGap     = 4

	// tipPanelChromeH is the close/toggle row's own total footprint below the
	// text area: top inset, bottom inset, the row itself and the 4px gap
	// TipPanelTextRect subtracts ahead of it. DIV-162 (measured at this
	// story's own landing): the four non-shop rects below, and SHOP-TIP-045's
	// own decoded 136-row shop rect (shopscreen.go's ShopTipRect), are each
	// grown by exactly this many rows past their sourced height, so the
	// close/toggle row is ADDITIVE floor space and never carves into the
	// text capacity DIV-133 already measured as exact for the shop (seven
	// wrapped lines, the shipped EN text's own count, with no spare row) —
	// carving it out of the sourced rect instead left only 4 of those 7
	// lines fitting, a truncation with no indicator this build does not
	// otherwise draw.
	tipPanelChromeH = 2*tipPanelInset + tipPanelButtonRowH + 4
)

// The four rooms whose panel is not the shop's (shopscreen.go's own
// shopTipRect covers the fifth).
//
// THE FIX RESTORES UNIFORM SWALLOW SEMANTICS: every one of app.go's five
// call sites returns on bare consumed again (the shop's own inShop call
// site never carried the narrowed guard). A showing panel consumes every
// press and release inside its own rect, on every screen, matching a
// floating window's own convention: the panel is dismissible per visit, and
// no control is permanently unreachable behind it. Each rect below is
// restored to its OWN RESEARCHED WIDTH AND TOP-LEFT CORNER
// (TOWN-165/TOWN-184, TOWN-015, TOWN-021/TOWN-188, TOWN-187) and grown only
// in HEIGHT, to the minimum that draws that screen's own shipped text whole
// at the researched width, on the worse of the two preserved roots, plus a
// small margin — cmd/tippanelcheck (committed, replacing round 2's own
// uncommitted scratch measurements) searches and prints this per screen and
// per root. Since a swallowed click costs the player nothing while the
// panel is showing, the deviation this story owns is HEIGHT ALONE
// (DIV-162): a rect that matches its researched size and still visually
// covers a control — the generator's own first fighter portrait,
// substantially covered even at TOWN-187's own researched rect — is the
// original's own layout, not this build's regression, and every rect below
// covers more of its own screen's live controls than round 2's own widened-
// but-shorter rects did, which is the deviation this round accepts in
// exchange for a correct hit test.
// TestReleaseTipPanelTextsDrawWholeOnBothClasses (pkg/game) still asserts
// every shipped text draws whole at the sizes below; tippanel_overlap_test.go
// (pkg/ui) is the round-3 witness that no press or release anywhere inside a
// showing panel's rect reaches a control the room's own hit test names, on
// any of the five screens, and that Close and Toggle still do.
var (
	// TownTipRect: researched (TOWN-165/TOWN-184) top-left and width restored
	// exactly, (328,0)-(640,*). cmd/tippanelcheck's own minimum fit at this
	// width is 295 rows (en) / 363 rows (ru); height set to 373, a 10-row
	// margin over the worse root. Covers 11394 of townmask.bmp's own live
	// pixels (door and statue), unchanged by root; the swallow at all five
	// app.go call sites is what keeps a click on that painted area from
	// reaching the door.
	TownTipRect = image.Rect(328, 0, 640, 373)
	// TavernTipRect: (160,0)-(472,*). TOWN-329 (High for every attach
	// instruction) resolves the tavern popup's ABSOLUTE rect to
	// (160,0)-(472,200).
	//
	// Height is the disclosed deviation (DIV-162), not the origin.
	TavernTipRect = image.Rect(160, 0, 472, 322)
	// SchoolTipRect: researched (TOWN-021/TOWN-188) top-left and width
	// restored exactly, (0,0)-(456,*). cmd/tippanelcheck's own minimum fit
	// is 261 (en) / 210 (ru); height set to 271. Covers 5381 of the skill
	// panel's own live pixels (schoolPanelRects/schoolSkillRects,
	// townshell.go), which start at y=188/192 regardless of width.
	SchoolTipRect = image.Rect(0, 0, 456, 271)
	// ChargenTipRect: (160,280)-(472,480).
	//
	// It is still the one rect of the five whose researched height already
	// holds its own shipped text:
	// cmd/tippanelcheck's minimum fit, worst of the fighter and mage bodies
	// on either root, is 160, so the researched 200 needs no growth and none
	// is taken. Round 3 shipped (0,310)-(312,480), 30 rows lower and 30
	// shorter than researched, while this comment and three documents said
	// the top-left was restored exactly (round-3 adversarial review, D-1);
	// the rect was corrected to the researched one at the story's landing,
	// which removes the deviation rather than disclosing it. At the
	// researched rect cmd/tippanelcheck measures two covered controls on
	// both roots: choice 0 at 97.9% (17431 of its own 17812 hit-testable
	// pixels) and choice 2 at 13.1% (2162 of 16458). Choice 2 was not
	// covered at all at round 3's own (0,310) rect, which is the correction
	// this landing accepts in exchange for the researched geometry; choices
	// 1 and 3 are uncovered at either rect.
	ChargenTipRect = image.Rect(160, 280, 472, 480)
)

// TipPanelTextRect is where the wrapped tip body draws, inset from the
// panel's own edges and stopping above the close/toggle row.
func TipPanelTextRect(r image.Rectangle) image.Rectangle {
	return image.Rect(
		r.Min.X+tipPanelInset, r.Min.Y+tipPanelInset,
		r.Max.X-tipPanelInset, r.Max.Y-tipPanelInset-tipPanelButtonRowH-4,
	)
}

// TipPanelCloseRect is the close control's own draw position and hit
// rectangle — one RECT for both, on every other room control's own
// convention (townSurfaceButtonRect's doc).
func TipPanelCloseRect(r image.Rectangle) image.Rectangle {
	return image.Rect(
		r.Max.X-tipPanelInset-tipPanelCloseW, r.Max.Y-tipPanelInset-tipPanelButtonRowH,
		r.Max.X-tipPanelInset, r.Max.Y-tipPanelInset,
	)
}

// TipPanelToggleRect is the checkbox's own draw position and hit rectangle:
// the gem plus its label, one clickable band on the same row as Close,
// starting at the panel's own left inset. Inside the band the gem is drawn
// at the band's left edge and the label is CENTRED in what remains, because
// ComposeTipPanel draws it through drawTownShellText, which centres
// (townShellTextLayout, townshell.go). This doc read "left-aligned" until
// the story's landing (round-3 adversarial review, D-6).
func TipPanelToggleRect(r image.Rectangle) image.Rectangle {
	return image.Rect(
		r.Min.X+tipPanelInset, r.Max.Y-tipPanelInset-tipPanelButtonRowH,
		TipPanelCloseRect(r).Min.X-8, r.Max.Y-tipPanelInset,
	)
}

// TipPanelControlAt is which control a point on a showing panel stands on,
// and whether the point is inside the panel at all. THE PANEL TAKES INPUT
// BEFORE THE SCREEN BENEATH IT (spec behaviour 1): a caller that gets
// consumed=true must not also run its own screen's hit test for this
// point, whether or not kind names a specific control.
func TipPanelControlAt(v TipPanelView, p image.Point) (kind TipControlKind, consumed bool) {
	if !v.Showing() || !p.In(v.Rect) {
		return TipControlNone, false
	}
	if p.In(TipPanelCloseRect(v.Rect)) {
		return TipControlClose, true
	}
	if p.In(TipPanelToggleRect(v.Rect)) {
		return TipControlToggle, true
	}
	return TipControlNone, true
}

// TipPanelListRect includes the list's overlap with the control row.
func TipPanelListRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+20, r.Min.Y+24, r.Max.X-28, TipPanelCloseRect(r).Min.Y+4)
}

// TipPanelEventAt applies the room popup's event-specific child order.
func TipPanelEventAt(v TipPanelView, p image.Point, down bool) (TipControlKind, bool) {
	if !v.Showing() || !p.In(v.Rect) {
		return TipControlNone, false
	}
	if !down && p.In(TipPanelListRect(v.Rect)) {
		return TipControlNone, true
	}
	if p.In(TipPanelCloseRect(v.Rect)) {
		return TipControlClose, true
	}
	if p.In(TipPanelToggleRect(v.Rect)) {
		return TipControlToggle, down
	}
	return TipControlNone, false
}

// TipPanelFits reports whether v's own wrapped text is drawn WHOLE inside
// TipPanelTextRect(v.Rect) — every wrapped line reached by ComposeTipPanel's
// own loop, none dropped past its bottom edge. A panel that is not Showing
// reports true (there is no text to lose). DIV-162 (this story's own
// landing): a rect too short to hold its own screen's shipped text silently
// drops trailing lines with no indicator, which is what carving the
// close/toggle row out of the shop's own decoded 136-row rect did to the
// shipped EN shop tip before the rects below were widened — this is the
// query ComposeTipPanel's draw loop answers implicitly, exposed so a test
// can assert it without duplicating the wrap.
//
// IT MEASURES THE GLYPH, NOT THE SHADOW (round-3 adversarial review, D-5).
// ComposeTipPanel draws each line twice, the shadow at (x+1,y+1) and the
// glyph at (x,y), and the condition below is the draw loop's own, so the
// two agree exactly on which lines are drawn. The last line's shadow row
// therefore sits one pixel below textRect.Max.Y. That row is inside the
// panel and inside TipPanelTextRect's own 4-pixel gap above the close and
// toggle band (TipPanelTextRect subtracts tipPanelInset+tipPanelButtonRowH+4
// from the bottom edge), so it overlaps nothing and no shipped text loses a
// row to it on either root. A caller wanting "the shadow fits too" must ask
// for Font.Height()+1, not this.
func TipPanelFits(v TipPanelView) bool {
	if !v.Showing() {
		return true
	}
	return tipPanelTextFits(v.Font, v.Text, v.Rect)
}

// tipPanelTextFits is TipPanelFits' own test, taken directly rather than
// through a TipPanelView so 1021's per-root height search (below) can ask it
// of a font and text with no Art and no showing panel yet to build. TipPanelFits
// and ComposeTipPanel's own draw loop both reduce to this.
func tipPanelTextFits(font *text.Font, s string, r image.Rectangle) bool {
	if font.Height()+2 <= 0 {
		return false
	}
	return tipPanelLinesFit(font, wrapShopTip(font, s, TipPanelTextRect(r).Dx()), r)
}

// tipPanelLinesFit is tipPanelTextFits' own test with the wrap already done.
//
// THE WRAP DEPENDS ON THE RECTANGLE'S WIDTH ALONE. TipPanelTextRect insets
// both vertical edges by the same tipPanelInset and both horizontal edges by
// tipPanelInset, so its Dx() is r.Dx() minus a constant and only its Dy()
// moves when a height search moves r.Max.Y. Splitting the test here is what
// lets TipPanelFitHeight wrap once for a whole search instead of once per
// candidate height; nothing else about the test changes.
func tipPanelLinesFit(font *text.Font, lines []string, r image.Rectangle) bool {
	textRect := TipPanelTextRect(r)
	pitch := font.Height() + 2
	if pitch <= 0 {
		return false
	}
	y := textRect.Min.Y
	for range lines {
		if y+font.Height() > textRect.Max.Y {
			return false
		}
		y += pitch
	}
	return true
}

// TipPanelFitHeight is the minimum whole-panel height, from a 20-row floor
// up to the 480-row frame, at which a panel of the given width draws text
// whole under font — tipPanelTextFits' own test, searched linearly on
// cmd/tippanelcheck's own precedent (searchHeight). Position does not enter
// the search: TipPanelTextRect's own insets are symmetric, so a candidate
// rect's Dx()/Dy() alone decide the answer. A nil font or empty text answers
// the floor with no search: nothing will be drawn regardless (Showing's own
// gate), so there is nothing to size for.
func TipPanelFitHeight(font *text.Font, s string, width int) int {
	const floor, ceiling = 20, 480
	if font == nil || s == "" {
		return floor
	}
	// One wrap for the whole search. Every candidate rect below has the same
	// width, so wrapShopTip returned an identical result on every iteration
	// of this loop and the search paid for up to 461 of them -- 76% of the
	// town composer's allocations and 42% of its time, both measured.
	lines := wrapShopTip(font, s, TipPanelTextRect(image.Rect(0, 0, width, ceiling)).Dx())
	for h := floor; h <= ceiling; h++ {
		if tipPanelLinesFit(font, lines, image.Rect(0, 0, width, h)) {
			return h
		}
	}
	return ceiling
}

// TipPanelShrinkRect returns max with its height reduced to the minimum that
// draws s whole under font (TipPanelFitHeight) plus margin rows, never
// larger than max's own height and never smaller than max's own chrome
// footprint plus one row of text (1021 spec B2: "the minimum that draws
// that root's own text whole plus a margin, measured per root rather than
// shared across both"). max's own left edge and width are unchanged; its
// top edge is unchanged and its bottom edge moves up. A nil font or empty
// text answers max unchanged, on TipPanelFitHeight's own floor-with-no-
// search rule: Showing() already gates drawing on Text being non-empty, so
// there is no text to measure a smaller panel against.
func TipPanelShrinkRect(max image.Rectangle, font *text.Font, s string) image.Rectangle {
	if font == nil || s == "" {
		return max
	}
	h := TipPanelFitHeight(font, s, max.Dx()) + tipPanelHeightMargin
	floor := tipPanelChromeH + font.Height() + 2
	if h < floor {
		h = floor
	}
	if h >= max.Dy() {
		return max
	}
	return image.Rect(max.Min.X, max.Min.Y, max.Max.X, max.Min.Y+h)
}

// tipPanelHeightMargin is the fixed number of rows of headroom
// TipPanelShrinkRect carries past the minimum height that draws its own text
// whole (1021 spec B2).
const tipPanelHeightMargin = 10

// ComposeTipPanel draws a showing panel's fill, frame, wrapped text, close
// control and toggle over dst. A panel that is not Showing draws nothing.
func ComposeTipPanel(dst *image.RGBA, v TipPanelView) {
	if !v.Showing() {
		return
	}
	r := v.Rect
	tileBlitSrc(dst, r, v.Art.Fill)
	drawNinePatchBorder(dst, r, v.Art.Border, tipPanelBorderCornerW, tipPanelBorderCornerH)

	textRect := TipPanelTextRect(r)
	y := textRect.Min.Y
	pitch := v.Font.Height() + 2
	for _, line := range wrapShopTip(v.Font, v.Text, textRect.Dx()) {
		if pitch <= 0 || y+v.Font.Height() > textRect.Max.Y {
			break
		}
		v.Font.Draw(dst, line, textRect.Min.X+1, y+1, shopShadowColor)
		v.Font.Draw(dst, line, textRect.Min.X, y, shopTextColor)
		y += pitch
	}

	closeRect := TipPanelCloseRect(r)
	drawTownShellBox(dst, closeRect, false)
	closeLabel := v.CloseLabel
	if closeLabel == "" {
		closeLabel = AuthoredTipClose
	}
	drawTownShellText(dst, v.Font, closeLabel, closeRect, townShellText)

	toggleRect := TipPanelToggleRect(r)
	gem := v.Art.GemOff
	if v.ToggleOn {
		gem = v.Art.GemOn
	}
	gy := toggleRect.Min.Y + (toggleRect.Dy()-tipPanelGemSize)/2
	if gem != nil {
		gb := gem.Bounds()
		at := image.Pt(toggleRect.Min.X, gy)
		draw.Draw(dst, gb.Add(at.Sub(gb.Min)), gem, gb.Min, draw.Over)
	}
	labelRect := image.Rect(toggleRect.Min.X+tipPanelGemSize+tipPanelGemGap, toggleRect.Min.Y, toggleRect.Max.X, toggleRect.Max.Y)
	toggleLabel := v.ToggleLabel
	if toggleLabel == "" {
		toggleLabel = AuthoredTipShowNext
	}
	drawTownShellText(dst, v.Font, toggleLabel, labelRect, townShellText)
}

// tileBlitSrc repeats src opaquely across target, one draw.Src blit per
// tile, clipped at the target's own edges — the fill's own "tiles" half of
// the contract's "the fill tiles and the frame is used as a nine-patch".
func tileBlitSrc(dst *image.RGBA, target image.Rectangle, src image.Image) {
	if src == nil {
		return
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return
	}
	for y := target.Min.Y; y < target.Max.Y; y += sh {
		for x := target.Min.X; x < target.Max.X; x += sw {
			tile := image.Rect(x, y, x+sw, y+sh).Intersect(target)
			if tile.Empty() {
				continue
			}
			draw.Draw(dst, tile, src, b.Min, draw.Src)
		}
	}
}

// tileBlitOver is tileBlitSrc's draw.Over counterpart, used for the
// border's own edge strips: unlike the fill, the border carries a keyed
// transparent field and must not overwrite what tileBlitSrc already put
// down beneath it.
func tileBlitOver(dst *image.RGBA, target image.Rectangle, src image.Image, srcRect image.Rectangle) {
	sw, sh := srcRect.Dx(), srcRect.Dy()
	if sw <= 0 || sh <= 0 {
		return
	}
	for y := target.Min.Y; y < target.Max.Y; y += sh {
		for x := target.Min.X; x < target.Max.X; x += sw {
			tile := image.Rect(x, y, x+sw, y+sh).Intersect(target)
			if tile.Empty() {
				continue
			}
			draw.Draw(dst, tile, src, srcRect.Min, draw.Over)
		}
	}
}

// drawNinePatchBorder draws border as a nine-patch over panel: the four
// corners unscaled at panel's own corners, the four edges tiled between
// them, and no centre patch — t_border.bmp's own centre is keyed
// transparent (the file doc), so there is nothing there to draw.
func drawNinePatchBorder(dst *image.RGBA, panel image.Rectangle, border image.Image, cw, ch int) {
	if border == nil {
		return
	}
	b := border.Bounds()
	if cw <= 0 || ch <= 0 || 2*cw >= b.Dx() || 2*ch >= b.Dy() || panel.Dx() < 2*cw || panel.Dy() < 2*ch {
		return
	}
	tl := image.Rect(b.Min.X, b.Min.Y, b.Min.X+cw, b.Min.Y+ch)
	tr := image.Rect(b.Max.X-cw, b.Min.Y, b.Max.X, b.Min.Y+ch)
	bl := image.Rect(b.Min.X, b.Max.Y-ch, b.Min.X+cw, b.Max.Y)
	br := image.Rect(b.Max.X-cw, b.Max.Y-ch, b.Max.X, b.Max.Y)
	top := image.Rect(b.Min.X+cw, b.Min.Y, b.Max.X-cw, b.Min.Y+ch)
	bottom := image.Rect(b.Min.X+cw, b.Max.Y-ch, b.Max.X-cw, b.Max.Y)
	left := image.Rect(b.Min.X, b.Min.Y+ch, b.Min.X+cw, b.Max.Y-ch)
	right := image.Rect(b.Max.X-cw, b.Min.Y+ch, b.Max.X, b.Max.Y-ch)

	draw.Draw(dst, image.Rect(panel.Min.X, panel.Min.Y, panel.Min.X+cw, panel.Min.Y+ch), border, tl.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Max.X-cw, panel.Min.Y, panel.Max.X, panel.Min.Y+ch), border, tr.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Min.X, panel.Max.Y-ch, panel.Min.X+cw, panel.Max.Y), border, bl.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Max.X-cw, panel.Max.Y-ch, panel.Max.X, panel.Max.Y), border, br.Min, draw.Over)

	tileBlitOver(dst, image.Rect(panel.Min.X+cw, panel.Min.Y, panel.Max.X-cw, panel.Min.Y+ch), border, top)
	tileBlitOver(dst, image.Rect(panel.Min.X+cw, panel.Max.Y-ch, panel.Max.X-cw, panel.Max.Y), border, bottom)
	tileBlitOver(dst, image.Rect(panel.Min.X, panel.Min.Y+ch, panel.Min.X+cw, panel.Max.Y-ch), border, left)
	tileBlitOver(dst, image.Rect(panel.Max.X-cw, panel.Min.Y+ch, panel.Max.X, panel.Max.Y-ch), border, right)
}
