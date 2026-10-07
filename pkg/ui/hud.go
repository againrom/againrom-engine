package ui

import "image"

// MENU-COMBAT-017, TOWN-091, DIV-213

const (
	// Town panels retain their own margin; mission bars meet the frame edges.
	hudMargin = 12
	hudBarPad = 6

	// ITEM-STARCOMP-100's inventory rectangle and icon origins.
	packCellSize    = 80
	packGridH       = 90
	packGridTop     = 6
	packGridXBase   = 32
	packGridReserve = 240
	packScrollCueW  = 22
	packScrollCueH  = 48

	// MAGIC-ICON-024: 36-pixel masks on a 38-pixel pitch, in two rows.
	bookCellSize = 36
	bookCellGap  = 2
	bookRows     = 2
	bookColumns  = 12
	bookBarW     = 480
	bookBarH     = 85

	// hudMinimapReserve and hudToggleBarH are `SHOP-FIGURE-041`'s own ids 5
	// and 6, 158 and 80 (round 2 of adversarial review, `DIV-201`'s own
	// amendment): the minimap's own reserved height and the control panel's,
	// replacing the authored 160 and 42 no owner directive ever produced.
	// hudMinimapReserve is a RESERVATION and not a measurement of the
	// minimap's own drawn box — see this file's header for the one exception,
	// the owner's separate "always square" directive, which minimapGeometry
	// (minimap.go) applies by clamping the square to this same constant on
	// its shorter axis rather than drawing the full 160-wide rect.
	//
	// THE TWO NOW SUM TO hudPanelTopY (238) EXACTLY, because both come from
	// the same decoded division `SHOP-FIGURE-041` gives for id 7's own top
	// edge. TestHudColumnSlotsSumToThePanelsOwnTop (hud_test.go) pins that
	// invariant so a future change to either constant that breaks it fails
	// loudly rather than reopening the four-pixel gap a stale version of this
	// comment used to describe.
	hudMinimapReserve = 158
	hudToggleBarH     = 80
)

// hudStackTops joins the 85-pixel book directly to the 90-pixel pack.
// spellbookBar moves the book to the floor when the pack is hidden.
func hudStackTops(area image.Point) (pack, book int) {
	pack = area.Y - packGridH
	book = pack - bookBarH
	return pack, book
}

// characterPaneFillerBGH is the mission column's fourth child's own
// background strip height: `extra1024r.bmp`'s own decoded size, measured
// 160x46 by a standalone probe against the lawful install through the
// production vfs/bmp path (`TOWN-356`'s loader binding, `TOWN-093`'s own
// shared blit). It is the only one of the two resolution-keyed backgrounds
// this build loads: the mission's own frame is fixed at (MissionFrameW,
// MissionFrameH) = (1024, 768) regardless of the player's chosen window
// resolution (viewer.go), so `extra800r.bmp`'s own 800x600 case is never
// reachable from this front end's own mission composition.
const characterPaneFillerBGH = 46

// hudLowerStack is the right column's lower half, TOP TO BOTTOM from
// hudFloorY: the character panel's top, the fourth child's own background
// top, then the worn box's — SHOP-FIGURE-041's own decoded order for ids 7
// and 8 (contract 1023 B3), continued by authored choice for the worn set
// (B5).
//
// worn IS WHERE THE WORN BOX STARTS, and it is offered for the same reason
// wornBoxRect always was: rightColumnBox refuses a box with no room. It
// starts immediately past the fourth child's own full height at 1024x768
// (`characterPaneFillerBGH` + `compactPanelH`, the background strip plus the
// borrowed panel height `TOWN-093`'s own readout occupies), which is exactly
// the mission's own fixed frame height — 480+46+242 = 768 — so it is refused
// at every shipped resolution, as it always was. `DIV-200` carries the worn
// set itself as an authored feature with no decoded basis in this column.
// THE ARITHMETIC IS UNCHANGED by the card's own name arriving between the
// two: `card + compactPanelH` is the same 768 that
// `filler + characterPaneFillerBGH + compactPanelH` was.
func hudLowerStack(area image.Point) (panel, filler, card, worn int) {
	panel = hudFloorY()
	filler = panel + compactPanelH
	card = filler + characterPaneFillerBGH
	worn = card + compactPanelH
	return panel, filler, card, worn
}

// rightColumnBox places a box of height h with its top at y in the right-hand
// column, and refuses a window with no room for the column at all.
//
// EVERY BOX IN THAT COLUMN GOES THROUGH IT, which is what makes "one column,
// one width, one right edge" a property of one statement rather than of four
// that happen to agree today.
//
// THE COLUMN IS FLUSH TO THE FRAME'S OWN EDGES, TOP, RIGHT AND BOTTOM. Round
// 1 of this story placed the column `hudMargin` inside the frame on all three
// — a leftover of the authored overlay this story replaced — where
// `SESS-VIEW-028` gives the strip as the screen minus its own 160 pixels,
// with nothing between the strip and the frame's own right edge, and
// `SHOP-FIGURE-041` gives the four ids' own slots running from y=0 to
// y=screenH with no margin at either end. Round 2 of adversarial review
// found the mismatch (a press or a picture at the column's own edges landing
// on the map instead) and removed `hudMargin` from every edge this function
// computes; nothing else in the column reads it.
//
// hudFloorY IS WHERE THE COLUMN'S LOWER HALF MAY NOT REACH: the control panel's
// own bottom edge. A box that would climb over the control panel is refused
// instead, because the panel is what switches it back on.
func rightColumnBox(area image.Point, y, h int) (image.Rectangle, bool) {
	if area.X < sidebarWidth || y < 0 || y+h > area.Y {
		return image.Rectangle{}, false
	}
	return image.Rect(area.X-sidebarWidth, y, area.X, y+h), true
}

// hudPanelTopY is SHOP-FIGURE-041's own id-7 top boundary at 1024x768: the
// local rect (0,238,160,480) offset onto the right column gives an absolute
// top of 238 regardless of screenW, and the panel's own height
// (`compactPanelH`, 242) is that same slot's height, so a panel placed at
// this top edge occupies exactly the decoded slot, top and bottom both
// (238..480). This is the one boundary in the column this build reproduces
// as a literal rather than an authored arrangement, because it is the one
// boundary with decoded content on one side of it — the character panel,
// contract 1023 B3. Ids 5 and 6's own heights are decoded too, since round 2
// (`hudMinimapReserve`, `hudToggleBarH`); which widget the owner assigned to
// each stays his own directive, not a reading of any claim (`DIV-201`).
// `DIV-200`'s own doll placement (below this line) stays authored.
const hudPanelTopY = 238

// hudFloorY is the highest a box in the right column's LOWER half may reach.
//
// THE VALUE IS THE LITERAL hudPanelTopY, NOT DERIVED from the minimap
// reservation or the control panel's own height above it, though since round
// 2 the two now sum to it exactly: `hudMinimapReserve` (158) plus
// `hudToggleBarH` (80) is 238, because both are decoded from the same
// `SHOP-FIGURE-041` division this literal reads directly. Before round 2 the
// function summed hudMargin + hudMinimapReserve + hudMargin + hudToggleBarH
// + hudBarGap, using the two authored sizes that preceded the decode, and
// reached 234 — four pixels off the literal this comment already carried,
// caught only because the boundary was pinned by name rather than derived
// from the sum. Deriving from the literal is kept for the same reason:
// hudPanelTopY is the decoded value itself, not a computation that could
// drift from it, and TestHudColumnSlotsSumToThePanelsOwnTop (hud_test.go)
// pins the sum as an invariant so a future change to either constant above
// that breaks it fails a test rather than silently reopening the gap.
func hudFloorY() int {
	return hudPanelTopY
}

// packBarRect is ITEM-STARCOMP-100's mission inventory rectangle
// (0,H-90,W-160,H). The visible count is the complete 80-pixel run between
// the two end strips; at 640x480 it is five cells.
func packBarRect(area image.Point) (image.Rectangle, int, bool) {
	if area.X < packGridReserve+packCellSize || area.Y < packGridH {
		return image.Rectangle{}, 0, false
	}
	cols := (area.X - packGridReserve) / packCellSize
	if cols < 1 || area.X <= sidebarWidth {
		return image.Rectangle{}, 0, false
	}
	return image.Rect(0, area.Y-packGridH, area.X-sidebarWidth, area.Y), cols, true
}

// packCellRects applies ITEM-STARCOMP-100's exact origin formula. The returned
// rectangles are shared by paint, hit-test, drag and double-click paths.
func packCellRects(bar image.Rectangle, cols int) []image.Rectangle {
	if cols < 0 {
		cols = 0
	}
	out := make([]image.Rectangle, cols)
	screenW := bar.Dx() + sidebarWidth
	x := bar.Min.X + packGridXBase + ((screenW-packGridReserve)%packCellSize)/2
	y := bar.Min.Y + packGridTop
	for i := range out {
		out[i] = image.Rect(x, y, x+packCellSize, y+packCellSize)
		x += packCellSize
	}
	return out
}

// packScrollRects are SESS-INPUT-037's two end strips: everything before the
// first cell scrolls back and everything after the last cell scrolls forward.
func packScrollRects(bar image.Rectangle, cols int) (back, fwd image.Rectangle) {
	cells := packCellRects(bar, cols)
	if len(cells) == 0 {
		return image.Rectangle{}, image.Rectangle{}
	}
	return image.Rect(bar.Min.X, bar.Min.Y, cells[0].Min.X, bar.Max.Y),
		image.Rect(cells[len(cells)-1].Max.X, bar.Min.Y, bar.Max.X, bar.Max.Y)
}

// bookBarRect spans the map band, with a centred native atlas and gold wings.
func bookBarRect(area image.Point) (image.Rectangle, int, bool) {
	_, y := hudStackTops(area)
	w := area.X - sidebarWidth
	if w < bookBarW || y < 0 {
		return image.Rectangle{}, 0, false
	}
	return image.Rect(0, y, w, y+bookBarH), bookColumns, true
}

// bookCellRects shares MAGIC-ICON-024's row-major order between painting,
// hover, quick-key assignment and casting. Compact shop books use the same
// pitch with their own column count.
func bookCellRects(bar image.Rectangle, cols int) []image.Rectangle {
	x0 := bar.Min.X + hudBarPad
	if cols == bookColumns && bar.Dx() >= bookBarW {
		x0 = bookAtlasX(bar) + hudBarPad
	}
	out := make([]image.Rectangle, 0, cols*bookRows)
	for row := 0; row < bookRows; row++ {
		y := bar.Min.Y + hudBarPad + row*(bookCellSize+bookCellGap)
		for col := 0; col < cols; col++ {
			x := x0 + col*(bookCellSize+bookCellGap)
			out = append(out, image.Rect(x, y, x+bookCellSize, y+bookCellSize))
		}
	}
	return out
}

func hudToggleBarRect(area image.Point) (image.Rectangle, bool) {
	return rightColumnBox(area, hudMinimapReserve, hudToggleBarH)
}

// characterPanelBoxRect is where the CHARACTER PANEL stands: the right-hand
// column, directly under the control panel — SHOP-FIGURE-041's own id 7,
// reused rather than composed a second time (contract 1023 B3).
func characterPanelBoxRect(area image.Point) (image.Rectangle, bool) {
	y, _, _, _ := hudLowerStack(area)
	return rightColumnBox(area, y, compactPanelH)
}

func columnFillerRect(area image.Point) (image.Rectangle, bool) {
	_, y, _, _ := hudLowerStack(area)
	return rightColumnBox(area, y, characterPaneFillerBGH)
}

// missionCardBoxRect is where the mission's own statistics card stands, under
// the fourth child's own background strip and flush with the frame's bottom
// edge, or false for a frame with no room for it.
//
// ITS HEIGHT IS THE PANEL'S OWN, so the card composes through exactly the
// layout the pane's own statistics mode uses and needs no second set of row
// metrics. At the mission family's fixed height 768 that is 526..768, flush at
// both ends: the strip above it ends at 526 and the frame ends at 768; expanded
// width only moves the whole right column horizontally.
func missionCardBoxRect(area image.Point) (image.Rectangle, bool) {
	_, _, y, _ := hudLowerStack(area)
	return rightColumnBox(area, y, compactPanelH)
}

// wornBoxRect is where the WORN SET would stand, or false: rightColumnBox
// refuses a box with no room, which is every shipped resolution — the fourth
// child's own full height at 1024x768 (`characterPaneFillerBGH` +
// `compactPanelH`) already reaches the mission's own fixed frame bottom
// (`DIV-200`). The function stays rather than being deleted so a caller's own
// gate (inventory.go's wornBox) reads the SAME refusal shape every other
// box's rect function gives a caller with no room, and so a taller frame this
// project never ships would still get an answer rather than a compile-time
// assumption.
//
// THE SLOT'S OWN TENANT IS DECODED, AND IT IS NOT A DOLL. The readout's two
// source fields are still Unknown, so it is not buildable from the pin as it
// stands, and this slot stays the worn set's own authored placement rather
// than the readout's.
func wornBoxRect(area image.Point) (image.Rectangle, bool) {
	_, _, _, y := hudLowerStack(area)
	return rightColumnBox(area, y, wornBoxSize().Y)
}
