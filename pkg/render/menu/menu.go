// Package menu holds main-menu layout, hit-mask rules and frame composition.
package menu

import (
	"fmt"
	"image"
	"image/draw"

	"againrom/pkg/screenart"
)

// Frame size. The menu is composed in the game's fixed 640x480 frame, and the
// hit mask is indexed from its origin (0,0).
const (
	FrameW = 640
	FrameH = 480
)

// ButtonCount is the number of brooch buttons. The engine's loader builds its
// overlay arrays in a loop over exactly this many, so it is a decoded constant
// rather than a count of whatever art happens to be present.
const ButtonCount = 8

// placement is one {x, y, w, h} row of a static placement table.
type placement struct{ x, y, w, h int }

func (p placement) rect() image.Rectangle {
	return image.Rect(p.x, p.y, p.x+p.w, p.y+p.h)
}

// The two static placement tables, in button order. Each (w, h) is the pixel size
// of that button's own bitmap, so an overlay draws 1:1 in frame pixels and a
// button's hover and pressed rectangles legitimately differ in both position and
// size — they are two separate images, not one image drawn twice.
//
// These are placement rectangles, not hit regions: they are bounding boxes of
// irregular brooch shapes, they overlap one another, and only the mask decides
// what the cursor is over.
var (
	hoverTable = [ButtonCount]placement{
		{112, 64, 212, 136},
		{84, 88, 236, 148},
		{84, 236, 236, 152},
		{112, 276, 212, 136},
		{320, 60, 212, 140},
		{324, 88, 236, 148},
		{324, 236, 236, 152},
		{324, 276, 208, 136},
	}
	pressedTable = [ButtonCount]placement{
		{116, 64, 208, 138},
		{88, 88, 236, 152},
		{88, 236, 236, 152},
		{116, 272, 208, 140},
		{320, 64, 212, 140},
		{324, 88, 232, 152},
		{324, 236, 236, 152},
		{320, 272, 212, 140},
	}

	// HoverRects and PressedRects are the tables as half-open frame rectangles.
	HoverRects   = rects(hoverTable)
	PressedRects = rects(pressedTable)
)

func rects(t [ButtonCount]placement) [ButtonCount]image.Rectangle {
	var out [ButtonCount]image.Rectangle
	for i, p := range t {
		out[i] = p.rect()
	}
	return out
}

// ColumnSplit divides the brooch's two columns: the left column's placements
// start at x < ColumnSplit, the right column's at x >= ColumnSplit.
const ColumnSplit = 320

// NewGameButton is the 1-based index of the brooch's top-left button — the one
// the front-end treats as NEW GAME.
//
// It is COMPUTED from the placement table, never written down as a literal. The
// research labels the mask's stored row order as inferred, so a requirement
// anchored to a mask index could not be falsified; anchored to the rectangle, a
// wrong row order changes only which art lights up, never which action a
// rectangle performs.
//
// Note the rule is not "the topmost entry": the right column's top placement
// begins four pixels higher than the left column's, so ranking by y alone picks
// the wrong button. The columns are told apart by x first.
var NewGameButton = newGameButton()

// LoadGameButton is the 1-based index of the brooch's top-RIGHT button — the one
// the front-end treats as LOAD GAME.
//
// Same provenance as NewGameButton, computed the same way: "LOAD GAME lives at
// the top-right of the brooch" is a claim about a position, so it is anchored
// to the placement rectangle and never to a mask index.
var LoadGameButton = loadGameButton()

// ExitButton is the 1-based index of the brooch button that quits the game.
//
// Unlike NewGameButton this one is a LITERAL: "button 8 quits" is decoded (the
// original's click dispatcher posts WM_CLOSE for button 8), while "the top-left
// button is NEW GAME" is a claim about a position. Its rectangle, the bottom
// entry of the right column, is a consequence. The binding runs through the
// inferred mask row order like every other button.
const ExitButton = 8

// HallOfFameButton is the third right-hand control. The installed base and
// button7 artwork carry that caption; the owner requested its main-menu door.
const HallOfFameButton = 7

// The third and fourth left plaques in the installed menu artwork.
const CutscenesButton = 3
const CreditsButton = 4

func newGameButton() int {
	best := 0
	for i, p := range hoverTable {
		if p.x >= ColumnSplit {
			continue // right column
		}
		if best == 0 || less(p, hoverTable[best-1]) {
			best = i + 1
		}
	}
	return best
}

// loadGameButton is newGameButton's mirror: the top of the RIGHT column.
//
// The column filter is REDUNDANT on the shipped table and is here anyway. The
// right column's top placement begins four pixels higher than the left column's,
// so it is also the global top: dropping the filter picks the same button, and
// removing it does not fail the test above it. That is a property of these
// numbers, not of the rule. It is the reverse for newGameButton, where the same
// filter is load-bearing — rank the whole table by y and NEW GAME gets the
// top-RIGHT gem. Stating the rule the same way in both directions costs three
// lines and means a future table change is answered by one reading, not two.
func loadGameButton() int {
	best := 0
	for i, p := range hoverTable {
		if p.x < ColumnSplit {
			continue // left column
		}
		if best == 0 || less(p, hoverTable[best-1]) {
			best = i + 1
		}
	}
	return best
}

// less orders left-column placements: smallest y, then smallest x. Both terms are
// needed only to make the rule total; the shipped table is unambiguous on y
// alone within a column.
func less(a, b placement) bool {
	if a.y != b.y {
		return a.y < b.y
	}
	return a.x < b.x
}

// maskButton maps every 8-bit mask index to the button it selects, 0 meaning
// none. Eight values are hot; index 0 (background), the anti-aliased edge ramp
// and every stray value fall through to none.
//
// It is written as eight literals rather than the equivalent idx/16-7 arithmetic
// so the rule is falsifiable by inspection and total by construction: an
// arithmetic form would silently accept values the research does not list as hot.
var maskButton = func() [256]uint8 {
	var t [256]uint8
	for i, idx := range [ButtonCount]byte{0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0} {
		t[idx] = uint8(i + 1)
	}
	return t
}()

// Layout is one game's main-menu row: where each button's hover and pressed
// overlay is drawn, the art entries of its captions, and what releasing each
// button does.
//
// Captions names the art entry of each button's caption bitmap, in button
// order, and CaptionAt is the frame rectangle every caption is drawn at; a
// game without captions leaves every entry empty.
type Layout struct {
	Hover, Pressed [ButtonCount]image.Rectangle
	Captions       [ButtonCount]string
	CaptionAt      image.Rectangle
	Actions        Actions
}

// FirstLayout is the first game's row: the two placement tables, no captions
// and its six decoded bindings.
var FirstLayout = Layout{Hover: HoverRects, Pressed: PressedRects, Actions: FirstActions}

// Assets is a validated main-menu asset set: the base screen, the hit mask, and
// both overlays for every button, with the layout row they were loaded with.
//
// Load returns nil on a missing or mistyped art entry. The screen art loader
// validates each picture and mask against the manifest before binding.
type Assets struct {
	Base    *image.RGBA              // 640x480 brooch background, drawn first
	Mask    *image.Paletted          // 640x480 hit mask, raw indices
	Hover   [ButtonCount]*image.RGBA // per-button hover overlay
	Pressed [ButtonCount]*image.RGBA // per-button pressed overlay

	// Layout is the row the assets were loaded with.
	Layout Layout
	// Captions holds the bitmaps of Layout.Captions, in button order.
	Captions [ButtonCount]*image.RGBA
}

// Load binds a game's resolved main-menu art to its layout row.
func Load(set *screenart.Set, l Layout) (*Assets, error) {
	a := &Assets{Layout: l}
	var err error
	if a.Base, err = picture(set, "base"); err != nil {
		return nil, err
	}
	a.Mask = set.Mask("mask")
	if a.Mask == nil {
		return nil, fmt.Errorf("menu: art entry %q is missing", "mask")
	}
	for i := range a.Hover {
		if a.Hover[i], err = picture(set, fmt.Sprintf("hover/%d", i)); err != nil {
			return nil, err
		}
		if a.Pressed[i], err = picture(set, fmt.Sprintf("pressed/%d", i)); err != nil {
			return nil, err
		}
	}
	for i, name := range l.Captions {
		if name == "" {
			continue
		}
		if a.Captions[i], err = picture(set, name); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func picture(set *screenart.Set, name string) (*image.RGBA, error) {
	img, ok := set.Picture(name).(*image.RGBA)
	if !ok || img == nil {
		return nil, fmt.Errorf("menu: art entry %q is not an RGBA picture", name)
	}
	return img, nil
}

// ButtonAt reports which button the frame position p selects: 1..ButtonCount, or
// 0 for none.
//
// The index is read straight out of the mask's raw pixel bytes — never through
// its palette, which is an identity grayscale ramp carrying no meaning. Every
// value outside the eight hot indices selects nothing: index 0, the anti-aliased
// edge ramp around each region, and any stray value. So does any position outside
// the frame, which is what a cursor in the window's letterbox maps to.
func (a *Assets) ButtonAt(p image.Point) int {
	if a == nil || a.Mask == nil {
		return 0
	}
	b := a.Mask.Bounds()
	if p.X < 0 || p.Y < 0 || p.X >= b.Dx() || p.Y >= b.Dy() {
		return 0
	}
	return int(maskButton[a.Mask.Pix[p.Y*a.Mask.Stride+p.X]])
}

// Overlay returns the single overlay a state asks for: the bitmap and the frame
// rectangle to draw it at, or ok == false for none.
//
// "At most one" is structural here rather than a rule the caller must keep: the
// signature cannot express two, and State cannot represent two. The rectangle is
// the button's own row of the layout's hover or pressed rectangles, whose size is
// that bitmap's own pixel size — so the overlay draws 1:1 in frame pixels and
// nothing is resampled inside the frame.
func (a *Assets) Overlay(s State) (img *image.RGBA, at image.Rectangle, ok bool) {
	if a == nil || s.Selected < 1 || s.Selected > ButtonCount {
		return nil, image.Rectangle{}, false
	}
	i := s.Selected - 1
	if s.Pressed {
		return a.Pressed[i], a.Layout.Pressed[i], a.Pressed[i] != nil
	}
	return a.Hover[i], a.Layout.Hover[i], a.Hover[i] != nil
}

// Compose renders one menu frame: the base brooch bitmap over the whole 640x480
// frame, then at most one button overlay at its own rectangle, then that
// button's caption when the layout has one.
//
// The overlay is copied OPAQUELY (draw.Src, not draw.Over): the overlays are
// opaque 24-bit rectangles and no blend is decoded, so none is invented.
//
// The result is a fresh image each call; the caller owns it.
func (a *Assets) Compose(s State) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, FrameW, FrameH))
	if a == nil || a.Base == nil {
		return dst
	}
	draw.Draw(dst, dst.Bounds(), a.Base, a.Base.Bounds().Min, draw.Src)

	if img, at, ok := a.Overlay(s); ok {
		draw.Draw(dst, at, img, img.Bounds().Min, draw.Src)
		if c := a.Captions[s.Selected-1]; c != nil {
			draw.Draw(dst, a.Layout.CaptionAt, c, c.Bounds().Min, draw.Src)
		}
	}
	return dst
}

// MaskRegions reports, for each button, how many frame pixels carry that
// button's hot index.
//
// "A non-empty hit region" is exactly a non-zero count here: at least one pixel.
// Nothing stronger is checked — not a bounding box, not contiguity — because
// neither is decoded, and either would call a real but scattered region empty.
// A button present in the asset set but with no mask region is reported, not
// rejected: it is a legitimate thing to observe about an install.
func (a *Assets) MaskRegions() [ButtonCount]int {
	var counts [ButtonCount]int
	if a == nil || a.Mask == nil {
		return counts
	}
	b := a.Mask.Bounds()
	for y := 0; y < b.Dy(); y++ {
		row := a.Mask.Pix[y*a.Mask.Stride : y*a.Mask.Stride+b.Dx()]
		for _, idx := range row {
			if n := maskButton[idx]; n != 0 {
				counts[n-1]++
			}
		}
	}
	return counts
}
