// Package menu is the main-menu composition contract: which main.res bitmaps
// make up the brooch screen, what the hit mask's byte values mean, where each
// button overlay is placed, and how a frame is composed from them.
//
// That contract is decoded by the research submodule from the game's own files
// and executable, not re-derived here. This package holds it as data plus the
// behaviour the specification defines over it, and nothing else: it decodes no
// container (the archive is 0001's), invents no format fact, and reads no game
// path — its input is a one-method entry source the caller supplies.
//
// It imports only the bitmap decoder and is engine-free, so the whole selection/compositing contract
// is unit-testable with no window.
//
// # Two things a reader should know before changing anything here
//
// The hit mask is consumed as RAW 8-BIT INDICES, never through palette colours.
// The mask's 256-entry palette is an identity grayscale ramp, so it carries no
// meaning of its own and the index value is the semantic; a colour-based reading
// is wrong by construction rather than merely slower. bmp.DecodePaletted keeps
// the indices and ButtonAt reads them straight out of Pix.
//
// The mask's stored ROW ORDER is *inferred* by the research, not read directly:
// it was concluded from each button's placement rectangle bracketing its mask
// region, and the brooch layout is nearly symmetric about the horizontal midline,
// so a vertically flipped read would very nearly satisfy that same check. This is
// why NewGameButton is computed from the placement table and never from a mask
// index literal — under one row order the visually top-left button answers to the
// first index, under the other to the fourth, and no layout evidence can tell
// them apart. Anchoring the action to geometry makes the question falsifiable the
// first time anyone hovers the top-left button in the real game.
package menu

import (
	"fmt"
	"image"

	"againrom/pkg/formats/bmp"
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

// EntryPrefix is the address prefix the menu assets live under: main.res's own
// identity segment, then the subtree inside it, in the folded form every address
// is compared in (lower case, '/' separators). The prefix is part of the key:
// lookup folds case but not directories.
//
// The identity segment is part of the prefix BECAUSE it is part of the name:
// one address identifies an entry across every container an install ships,
// so this package's constants say which container the menu art comes from
// instead of leaving the caller to know it out of band.
const EntryPrefix = "main/graphics/mainmenu/"

// Asset entry names, relative to EntryPrefix. Eighteen in total: the base brooch
// screen, the hit mask, and a hover and a pressed overlay per button.
const (
	BaseEntry = "menu_.bmp"
	MaskEntry = "menumask.bmp"
)

// HoverEntry and PressedEntry name button n's two overlays, n in [1, ButtonCount].
func HoverEntry(n int) string   { return fmt.Sprintf("button%d.bmp", n) }
func PressedEntry(n int) string { return fmt.Sprintf("button%dp.bmp", n) }

// Entries returns every address the menu requires, in the order Load validates
// them: the base, the mask, then each button's hover and pressed overlay in
// button order.
func Entries() []string {
	out := make([]string, 0, 2+2*ButtonCount)
	out = append(out, EntryPrefix+BaseEntry, EntryPrefix+MaskEntry)
	for n := 1; n <= ButtonCount; n++ {
		out = append(out, EntryPrefix+HoverEntry(n), EntryPrefix+PressedEntry(n))
	}
	return out
}

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
// It is COMPUTED from the placement table, never written down as a literal, and
// that is the point. The research labels the mask's stored row order as
// inferred, so a requirement anchored to a mask index would be one the layout
// evidence cannot falsify: under one row order the visually top-left button
// answers to index 0x80, under the other to 0xb0, and both satisfy every check
// the research ran. Anchored to the rectangle instead, a wrong row order changes
// only which art lights up — never which action a rectangle performs — and is
// falsified the first time anyone hovers the top-left button in the real game.
//
// Note the rule is not "the topmost entry": the right column's top placement
// begins four pixels higher than the left column's, so ranking by y alone picks
// the wrong button. The columns are told apart by x first.
var NewGameButton = newGameButton()

// LoadGameButton is the 1-based index of the brooch's top-RIGHT button — the one
// the front-end treats as LOAD GAME.
//
// Same provenance as NewGameButton, and computed the same way for the same
// reason. "The top-right corner of the brooch is where LOAD GAME lives" is
// the owner's knowledge of the game, which is a claim about a POSITION on
// screen, so it is anchored to the placement rectangle and never to a mask
// index. If the inferred mask row order is wrong, a wrong row order changes
// only which art lights up, never which rectangle performs which action.
//
// This is the third bound button of the eight. The other five carry commands the
// research still reports as undecoded, and nothing here binds them.
var LoadGameButton = loadGameButton()

// ExitButton is the 1-based index of the brooch button that quits the game.
//
// Unlike NewGameButton this one is a LITERAL, and the difference is provenance,
// not taste. "The top-left button is NEW GAME" is the owner's knowledge of the
// game — a claim about a position on screen — so it is anchored to a position and
// computed from the placement table. "Button 8 quits" is decoded: the original's
// click dispatcher posts WM_CLOSE for button 8, and the research names the
// number, not a corner. Deriving this one geometrically would be inventing a
// placement claim nobody made.
//
// The rectangle is a consequence, not the definition: placement entry 8 is
// {324, 276, 208, 136}, the bottom entry of the right column, which is where a
// human looks when checking this against the real brooch. It is exactly one of
// the eight buttons; the other six carry commands the research reports as
// undecoded, so nothing here binds them.
//
// This binding runs through the same inferred mask row order as everything else
// (see the package doc): ButtonAt answers with a mask-derived number, so if that
// inference is wrong the gem that quits is the top-right one instead. No design
// avoids that, because the mask is the only thing that says what the cursor is
// over — only a human at the real menu can settle it.
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

// EntrySource supplies raw entry bytes by address. It is the whole of this
// package's dependency on the container tier: a single archive reader and a
// filesystem over a set of them both satisfy it, so nothing below the render
// tier has to cross into it.
type EntrySource interface {
	ReadFile(name string) ([]byte, error)
}

// Assets is a validated main-menu asset set: the base screen, the hit mask, and
// both overlays for every button.
//
// A value of this type only ever exists fully populated and dimension-consistent
// — Load returns nil alongside any error rather than a partial set — so no caller
// can show a menu whose hit regions or overlays are only partly defined.
type Assets struct {
	Base    *image.RGBA              // 640x480 brooch background, drawn first
	Mask    *image.Paletted          // 640x480 hit mask, raw indices
	Hover   [ButtonCount]*image.RGBA // per-button hover overlay
	Pressed [ButtonCount]*image.RGBA // per-button pressed overlay

	// place is nil for the first game, whose rectangles are the tables above.
	place *secondPlacement
}

// Load reads and validates the eighteen menu bitmaps from src.
//
// Validation is in a fixed order — base, mask, then each button's hover and
// pressed overlay — and stops at the FIRST defect, so a broken install produces
// one message naming one entry, and the windowed and headless modes cannot
// disagree about which failure to report. The checks are: the entry is present,
// it decodes, and its dimensions are right — 640x480 for the base and the mask,
// and for each overlay exactly the (w, h) of its own row of the corresponding
// placement table. That last check is the decoded 1:1 draw rule turned into a
// startup assertion: the tables' sizes are the bitmaps' own sizes, so a
// disagreement means the asset set and the geometry are not the pair the contract
// describes.
func Load(src EntrySource) (*Assets, error) {
	return load(src, HoverRects, PressedRects)
}

func load(src EntrySource, hover, pressed [ButtonCount]image.Rectangle) (*Assets, error) {
	a := &Assets{}

	base, err := loadRGBA(src, EntryPrefix+BaseEntry)
	if err != nil {
		return nil, err
	}
	if err := checkSize(EntryPrefix+BaseEntry, base.Bounds(), FrameW, FrameH); err != nil {
		return nil, err
	}
	a.Base = base

	maskName := EntryPrefix + MaskEntry
	maskData, err := src.ReadFile(maskName)
	if err != nil {
		return nil, fmt.Errorf("menu: %s: %w", maskName, err)
	}
	mask, err := bmp.DecodePaletted(maskData)
	if err != nil {
		return nil, fmt.Errorf("menu: %s: %w", maskName, err)
	}
	if err := checkSize(maskName, mask.Bounds(), FrameW, FrameH); err != nil {
		return nil, err
	}
	a.Mask = mask

	for i := 0; i < ButtonCount; i++ {
		n := i + 1
		for _, o := range []struct {
			name string
			want image.Rectangle
			dst  *[ButtonCount]*image.RGBA
		}{
			{EntryPrefix + HoverEntry(n), hover[i], &a.Hover},
			{EntryPrefix + PressedEntry(n), pressed[i], &a.Pressed},
		} {
			img, err := loadRGBA(src, o.name)
			if err != nil {
				return nil, err
			}
			if err := checkSize(o.name, img.Bounds(), o.want.Dx(), o.want.Dy()); err != nil {
				return nil, err
			}
			o.dst[i] = img
		}
	}
	return a, nil
}

func loadRGBA(src EntrySource, name string) (*image.RGBA, error) {
	data, err := src.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("menu: %s: %w", name, err)
	}
	img, err := bmp.DecodeRGBA(data)
	if err != nil {
		return nil, fmt.Errorf("menu: %s: %w", name, err)
	}
	return img, nil
}

func checkSize(name string, b image.Rectangle, w, h int) error {
	if b.Dx() != w || b.Dy() != h {
		return fmt.Errorf("menu: %s is %dx%d, want %dx%d", name, b.Dx(), b.Dy(), w, h)
	}
	return nil
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
