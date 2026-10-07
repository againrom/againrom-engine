package ui

import "image"

// The character figure widget's six corner rectangles, their gates and their
// art, shared by every screen that draws the pane.
//
// ONE OBJECT, TWO PARENTS. `TOWN-353` (High) establishes that the original has
// exactly ONE instance of this widget: `campaign+0xe0`, id 7, constructed once
// at `L09790` with the parent-local rect `(0,238,160,480)`, re-parented from
// the mission frame to the shop view for a visit and back again
// (`SHOP-FIGURE-041`). Its three constructor overloads are one live and two
// dead. So the six rectangles and the per-pixel slot map are IDENTICAL on both
// screens and only the gates differ, and this file is the one place they are
// written down.
//
// EVERY RECTANGLE IS PANEL-RELATIVE, never absolute. `B = T + 0xf2` at every
// resolution because the constructed vertical pair is literal. That is why
// the functions below take the pane rectangle rather than reading a
// constant: the town's pane stands at TownCharacterRegion and the mission's
// at characterPanelBoxRect, which coincide at 640x480 and do not at 800x600
// or 1024x768.
//
// THE SESSION FIELD IS A BITMASK AND NOT AN ENUMERATION (`TOWN-352`, Medium
// for the population of writers, High for bits 0 and 1): an OR of 2 applied to
// the value 1 produces 3, which no enumeration would define. Bit 0 is the
// mission frame, bit 1 is the shop. A shop opened from a mission carries both.
type CharacterPaneSession uint32

const (
	CharacterPaneMission CharacterPaneSession = 1

	// CharacterPaneShop is the shop bit alone.
	//
	// EVERY TOWN-SIDE SCREEN IN THIS BUILD PASSES THIS VALUE, including the
	// tavern, the school and the character generator, and that is a deliberate
	// divergence rather than a reading of any claim (`DIV-311`). The paint
	// routine's own first act is a zero test of `sess+0x3dc` against mask 0x627
	// (`L03384`), so with the field at 0 the original draws NOTHING here at all: this
	// widget is never on a mask-0 screen. This build draws the pane on four
	// town screens by authored reuse (stories 1021, 1022), and the shop's own
	// mask is the value the original gives the widget on a town-reachable
	// view, so it is the one that keeps the party picker live where it is the
	// only member control the screen has.
	//
	// SHOP-TOWN-023
	CharacterPaneShop CharacterPaneSession = 2

	// CharacterPaneShopInMission is `TOWN-352`'s own value for a shop opened
	// during a mission: bits 0 and 1 together. This build has no such
	// transition today (`pkg/game/shopview.go` builds one shop view and the
	// town is its only entry), so the constant exists to be passed rather than
	// to be guessed at when one is added.
	CharacterPaneShopInMission CharacterPaneSession = 3
)

type CharacterPaneCorner int

const (
	// CharacterPaneBackpack is rect A, the pane's bottom-left corner, 28x40 at
	// `(L, B-0x28, L+0x1c, B)`, gate `sess & 1`. It posts `0x40e` to itself and
	// to the map view (`TOWN-347`) and inverts `this+0x68` (`TOWN-351`). Its
	// art is `BackPackOp.bmp` or `BackPackCl.bmp` (`TOWN-356`) chosen by
	// `FindChildById(mapview, 2) != 0` (`TOWN-354`), and `SESS-VIEW-028` names
	// the four callers of the map view's own row recompute as the open/close
	// pairs the WindowProc runs for `0x40e`/`0x40f`, with mission entry
	// invoking them to restore `Inventory/IsOpen` and `SpellBook/IsOpen`. The
	// art names and the restore names agree: child 2 is the carried pack.
	CharacterPaneBackpack CharacterPaneCorner = iota

	// CharacterPaneBook is rect B, the pane's top-left corner, 28x36 at
	// `(L, T, L+0x1c, T+0x24)`, gate `sess & 3`. It posts `0x40f` to itself, to
	// the map view, and to the shop at `sess+0xf0` when `sess & 2`
	// (`TOWN-347`), and inverts `this+0x6c`. Its art is `BookOpened.bmp` or
	// `BookClosed.bmp` chosen by `FindChildById(host, 3) != 0`, `host` being
	// the shop when the shop bit is set and the map view otherwise.
	CharacterPaneBook

	// CharacterPaneMode is rect C, the pane's top-right corner, 32x36 at
	// `(L+0x80, T, R, T+0x24)`, gate `!(sess & 0x600)` and `flag == 0`. It
	// posts `0x412` to itself and then writes `[mapview+0xe0] = 1`
	// (`TOWN-347`); the arm plays fixed slot 1 (`VIDEO-SFX-016`) with priority
	// `0xdc` (`VIDEO-SFX-013`), inverts `this+0x70` and sets `this+0x60 = 1`
	// (`TOWN-351`). `this+0x70` is the pane's presentation mode, read four
	// times by the paint routine.
	CharacterPaneMode

	// CharacterPanePrev is rect D, 32x32 at `(L+1, T+0xcd, L+0x21, T+0xed)`,
	// gate `sess & 0x226`. It plays fixed slot 1 with priority `0xdc` and posts
	// `0x414`, which
	// `SHOP-PICKER-043` resolves at the shop view's own message table as the
	// previous party member. Its art is `ar1.bmp`.
	CharacterPanePrev

	// CharacterPaneNext is rect E, 32x32 at
	// `(L+0x77, T+0xcd, L+0x97, T+0xed)`, same gate, posting `0x415`, the next
	// party member. Its art is `ar2.bmp`.
	CharacterPaneNext

	// CharacterPaneMenu is rect F, 32x32 at
	// `(L+0x7e, T+0xce, L+0x9e, T+0xee)`, gate `sess & 1`. It plays fixed slot
	// 1 with priority `0xdc` and calls `PostMessageA([sess+0x1c], 0x416, 0, 0)`, the same
	// `0x416` `MENU-ESC-010` reads on the frame window's own `VK_ESCAPE` arm
	// under `campaign+0x3dc == 1`. Its tooltip resolves to `text/main.txt`
	// line 15, `Main Menu <ESC>`. Its art is `diskette.bmp`.
	CharacterPaneMenu

	characterPaneCornerCount
)

// characterPaneRects is the six rectangles' own panel-relative geometry, in
// `TOWN-346`'s test order and with its own displacements written as the
// hexadecimal the listing carries.
//
// EVERY ONE OF THE TWENTY-FOUR BOUNDS IS A CONSTANT DISPLACEMENT FROM THE
// ACCUMULATED RECT, formed by register moves, address computations or adds between `L11872` and
// `L11873`, with no table and no data operand anywhere in the window — so
// the population cannot vary with content (`TOWN-346`, High).
//
// `B` IS `T + 0xf2` AT EVERY RESOLUTION (`TOWN-345`), so rect A's own
// `B - 0x28` (line below, `CharacterPaneBackpack`'s `Min.Y`) equals
// `T + 0xca`, which is what `TOWN-355`'s overlap arithmetic uses -- but it
// is written here as `b-0x28`, off the accumulated rect's own bottom, to
// keep this function a pure offset table with no derived constant of its
// own.
func characterPaneRects(pane image.Rectangle) [characterPaneCornerCount]image.Rectangle {
	l, t := pane.Min.X, pane.Min.Y
	r, b := pane.Max.X, pane.Max.Y
	return [characterPaneCornerCount]image.Rectangle{
		CharacterPaneBackpack: image.Rect(l, b-0x28, l+0x1c, b),
		CharacterPaneBook:     image.Rect(l, t, l+0x1c, t+0x24),
		CharacterPaneMode:     image.Rect(l+0x80, t, r, t+0x24),
		CharacterPanePrev:     image.Rect(l+1, t+0xcd, l+0x21, t+0xed),
		CharacterPaneNext:     image.Rect(l+0x77, t+0xcd, l+0x97, t+0xed),
		CharacterPaneMenu:     image.Rect(l+0x7e, t+0xce, l+0x9e, t+0xee),
	}
}

// CharacterPaneCornerRect is one corner's rectangle in the pane's own screen
// coordinates. Exported so a release test can name the rectangle it is
// comparing without copying the displacements, which is the drift
// TownCharacterPersistentControls' own doc already records once.
func CharacterPaneCornerRect(pane image.Rectangle, c CharacterPaneCorner) image.Rectangle {
	if c < 0 || c >= characterPaneCornerCount {
		return image.Rectangle{}
	}
	return characterPaneRects(pane)[c]
}

// characterPaneModeFlag is the original's own `flag`, computed at
// `L11877`..`L11878`: 1 when `screenH - B > B - T` and `sess == 1`
// (`TOWN-346`). It gates rect C's hit test and rect C's art alike.
//
// `B - T` IS ALWAYS `0xf2`, so on the mission screen with `B = 480` the
// condition reduces to `screenH > 722`, which of the three resolutions
// `SESS-VIEW-028` enumerates only 1024x768 satisfies. `TOWN-346` states
// plainly that this reduction assumes the containers above `campaign+0xd4`
// contribute no vertical origin, which was not read; this function computes
// the unreduced expression from the pane it is handed instead, so the
// assumption is not baked in (`DIV-312`).
//
// A screenH of zero or less is read as "the caller does not know", and the
// flag is 0. That is the value every shipped resolution but one produces, and
// it is the value that keeps rect C live rather than silently removing the
// pane's own mode control from a caller that forgot to fill the field in.
//
// PANE MUST BE THE PANE'S OWN ABSOLUTE SCREEN RECTANGLE, never a
// canvas-relative one. `pane.Max.Y` is read directly and `B - T` alone is not
// enough to reproduce `TOWN-346`'s own accumulated-origin assumption: a local
// rect `(0,0)-(160,242)` and a screen rect `(864,238)-(1024,480)` share the
// same height but different `Max.Y`, and this function's threshold moves with
// it (`screenH > 484` against `screenH > 722`). Round-2 adversarial review
// found the two callers passing different rects and agreeing only because
// 768 clears both thresholds; `characterPaneModeFlag` is now called from
// exactly one place, `(*Viewer).characterPaneModeFlag` (missionpane.go),
// which always resolves the pane's own screen rectangle first, and every
// other function in this file takes the resulting bool rather than a pane.
func characterPaneModeFlag(pane image.Rectangle, s CharacterPaneSession, screenH int) bool {
	if s != CharacterPaneMission || screenH <= 0 {
		return false
	}
	return screenH-pane.Max.Y > pane.Max.Y-pane.Min.Y
}

// CharacterPaneCornerLive reports whether corner c answers a click, which is
// `TOWN-346`'s own gate for it and nothing else. It says nothing about whether
// the corner is DRAWN: three of `TOWN-355`'s findings are exactly that a
// rectangle answers clicks while nothing is painted there, and
// characterPaneCornerDrawn is the separate question.
//
// modeFlag IS THE CALLER'S OWN, precomputed value of characterPaneModeFlag —
// see that function's own doc for why this takes the bool rather than the
// rect and the screen height it used to be computed from.
func CharacterPaneCornerLive(modeFlag bool, s CharacterPaneSession, c CharacterPaneCorner) bool {
	switch c {
	case CharacterPaneBackpack, CharacterPaneMenu:
		return s&1 != 0
	case CharacterPaneBook:
		return s&3 != 0
	case CharacterPaneMode:
		return s&0x600 == 0 && !modeFlag
	case CharacterPanePrev, CharacterPaneNext:
		return s&0x226 != 0
	}
	return false
}

// CharacterPaneCornersAt is every corner a left-button-up at p reaches, in the
// original's own test order.
//
// IT RETURNS A SLICE AND NOT ONE ANSWER, and that is the whole point of the
// function. So a click at panel-local `(5,206)` with both gates open posts
// `0x40e` twice and `0x414` once, and a click inside the second overlap
// posts `0x415` and `0x416`. Reproducing that is what makes `TOWN-355`
// observable here rather than hidden behind a first-match hit test
// (`DIV-310`).
//
// A caller that wants one control asks for element 0. A caller that wants the
// original's behaviour walks the slice.
//
// pane IS THE HIT TEST'S OWN COORDINATE SYSTEM, which every caller of this
// function passes as the pane's absolute screen rectangle already — p is a
// window pixel, so it can be nothing else. That makes it safe to compute the
// mode flag from the same pane right here, once, rather than threading a
// third value through every caller.
func CharacterPaneCornersAt(pane image.Rectangle, s CharacterPaneSession, screenH int, p image.Point) []CharacterPaneCorner {
	rects := characterPaneRects(pane)
	modeFlag := characterPaneModeFlag(pane, s, screenH)
	var out []CharacterPaneCorner
	for c := CharacterPaneCorner(0); c < characterPaneCornerCount; c++ {
		if !CharacterPaneCornerLive(modeFlag, s, c) {
			continue
		}
		if p.In(rects[c]) {
			out = append(out, c)
		}
	}
	return out
}

// characterPaneCornerDrawn is whether corner c's own art is painted, which is
// a DIFFERENT expression from its hit gate for three of the six
// (`TOWN-354`, `TOWN-355`).
//
// pickerSuppressed is the original's `sess+0x6bc == 2` term, which this build
// has no field for: `TOWN-354` reads the exclusion at
// `L11968`..`L11969` and nothing decoded says what `sess+0x6bc` counts, so
// this build passes false and records the gap (`DIV-313`).
//
// modeFlag IS THE CALLER'S OWN PRECOMPUTED VALUE, `CharacterPaneCornerLive`'s
// own doc applies here too: this function never resolves a pane rectangle of
// its own, so it cannot repeat the local-vs-screen mismatch the composer and
// the hit test used to disagree about.
func characterPaneCornerDrawn(modeFlag bool, s CharacterPaneSession, c CharacterPaneCorner, pickerSuppressed bool) bool {
	switch c {
	case CharacterPaneBackpack, CharacterPaneMenu:
		// `!(sess & 0x226)` AND `!(sess & 0x400)`, against a hit gate of
		// `sess & 1`. With `sess == 3` — `TOWN-352`'s value for a shop opened
		// from a mission — both rects are live and neither is drawn
		// (`TOWN-355`, first gate mismatch).
		return s&0x226 == 0 && s&0x400 == 0
	case CharacterPaneBook:
		return s&3 != 0
	case CharacterPaneMode:
		return s&0x600 == 0 && !modeFlag
	case CharacterPanePrev, CharacterPaneNext:
		if s&0x200 != 0 && pickerSuppressed {
			return false
		}
		return s&0x226 != 0
	}
	return false
}

// characterPaneArtRects is where each corner's own bitmap lands, which is NOT
// its hit rectangle for four of the six (`TOWN-354`, every blit's destination
// and size an operand of a listing that tiled exactly).
//
// The two picker blits ARE pixel-exact with their hit rectangles, which is the
// clause `SHOP-PICKER-043` recorded as missing and `TOWN-354` answers. The
// other four are not, and reproducing the offsets rather than reusing the hit
// rectangles is what keeps the drawn corner where the original draws it.
//
// Each entry is the destination of the blit under whichever of that corner's
// bitmaps is selected. Rect A and rect B each have TWO destinations, one per
// bitmap, so this function takes the selector.
func characterPaneArtRect(pane image.Rectangle, c CharacterPaneCorner, alt bool) image.Rectangle {
	l, t := pane.Min.X, pane.Min.Y
	switch c {
	case CharacterPaneBackpack:
		if alt {
			// BackPackOp.bmp at `(L, T+0xd0)` 32x31.
			return image.Rect(l, t+0xd0, l+32, t+0xd0+31)
		}
		// BackPackCl.bmp at `(L+1, T+0xc9)` 28x30.
		return image.Rect(l+1, t+0xc9, l+1+28, t+0xc9+30)
	case CharacterPaneBook:
		if alt {
			// BookOpened.bmp at `(L, T)` 28x38.
			return image.Rect(l, t, l+28, t+38)
		}
		// BookClosed.bmp at `(L, T+4)` 28x37.
		return image.Rect(l, t+4, l+28, t+4+37)
	case CharacterPaneMode:
		// HumanMode.bmp or TextMode.bmp at `(L+0x80, T+4)` 28x32.
		return image.Rect(l+0x80, t+4, l+0x80+28, t+4+32)
	case CharacterPanePrev:
		// ar1.bmp at `(L+1, T+0xcd)` 32x32 — pixel-exact with rect D.
		return image.Rect(l+1, t+0xcd, l+1+32, t+0xcd+32)
	case CharacterPaneNext:
		// ar2.bmp at `(L+0x77, T+0xcd)` 32x32 — pixel-exact with rect E.
		return image.Rect(l+0x77, t+0xcd, l+0x77+32, t+0xcd+32)
	case CharacterPaneMenu:
		// diskette.bmp at `(L+0x7e, T+0xce)` 32x32 — pixel-exact with rect F.
		return image.Rect(l+0x7e, t+0xce, l+0x7e+32, t+0xce+32)
	}
	return image.Rectangle{}
}

// CharacterPaneCornerArt is the nine shipped bitmaps the six corners blit,
// resolved once at startup and handed to the composer.
//
// EVERY PATH IS A CLAIM'S OWN.
//
// THE PAIR NAMES SETTLE WHAT THE TWO OPEN MESSAGES OPEN. `TOWN-347` grades
// Medium the inference that `0x40e` and `0x40f` toggle the presence of the map
// view's child id 2 and child id 3, taken from this widget's own paint
// predicate rather than from the map view's handler. `SESS-VIEW-028` (High for
// the enumeration) names the four callers of the map view's row recompute as
// the open/close pairs the WindowProc runs for those two messages, and names
// mission entry invoking them to restore `Inventory/IsOpen` and
// `SpellBook/IsOpen`. A backpack bitmap under the child-2 predicate and a book
// bitmap under the child-3 predicate agree with both, in the order both give.
//
// A nil field draws nothing for that corner and never fails: the pane's own
// authored fallback is DrawTownCharacterRegion's rule for the body and it is
// this file's rule for the corners.
type CharacterPaneCornerArt struct {
	// BackpackOpen and BackpackClosed are rect A's pair, selected by whether
	// the carried pack is on screen.
	BackpackOpen, BackpackClosed image.Image
	// BookOpened and BookClosed are rect B's pair, selected by whether the
	// spellbook is on screen.
	BookOpened, BookClosed image.Image
	// HumanMode and TextMode are rect C's pair, selected by the pane's own
	// presentation mode. `L11935`..`L13189` reads `this+0x70` and takes
	// HumanMode when it is NON-ZERO, TextMode when it is zero; the same
	// polarity at `L11934`..`L13190` takes HumanBackR when it is non-zero
	// and TextBackR when it is zero, and the block at `L11936` skips the
	// statistics text entirely when it is non-zero. So non-zero is the doll
	// and zero is the statistics card, and the constructor forces 1
	// (`TOWN-351`, `L11938`), which is why the pane opens on the doll.
	HumanMode, TextMode image.Image
	// Diskette is rect F's single bitmap, the in-mission menu control.
	Diskette image.Image
	// Ar1 and Ar2 are rects D and E, the party picker's two arrows. Their
	// blits are pixel-exact with their own hit rectangles (`TOWN-354`), which
	// is the clause `SHOP-PICKER-043` recorded as missing.
	Ar1, Ar2 image.Image
}

// characterPaneCornerPicture is the bitmap corner c blits given the pane's own
// two open flags and its presentation mode, and whether that bitmap is the
// `alt` one characterPaneArtRect must be asked about.
//
// `alt` IS THE PREDICATE'S TRUE SIDE, which for rects A and B is the OPEN
// child and for rect C is the doll. The selection has the same shape in all
// three places: load the first global, test it, and skip the second load when
// it is set.
func characterPaneCornerPicture(a *CharacterPaneCornerArt, c CharacterPaneCorner, packOpen, bookOpen, figureMode bool) (image.Image, bool) {
	if a == nil {
		return nil, false
	}
	switch c {
	case CharacterPaneBackpack:
		if packOpen {
			return a.BackpackOpen, true
		}
		return a.BackpackClosed, false
	case CharacterPaneBook:
		if bookOpen {
			return a.BookOpened, true
		}
		return a.BookClosed, false
	case CharacterPaneMode:
		if figureMode {
			return a.HumanMode, true
		}
		return a.TextMode, false
	case CharacterPanePrev:
		return a.Ar1, false
	case CharacterPaneNext:
		return a.Ar2, false
	case CharacterPaneMenu:
		return a.Diskette, false
	}
	return nil, false
}

// characterPaneFigureOrigin is where the composed 160x240 figure's own (0,0)
// pixel lands, pane-relative.
//
// IT IS THE SHOP'S OWN shopFigureRect ORIGIN GIVEN A PANE, and the two are
// written here once rather than twice: at 640x480 this returns
// `(480,240)`, which is `shopFigureRect.Min` to the pixel. The mask is read
// from this origin WITHOUT CENTRING, on both screens, which is what makes the
// slot map the same map the contract's G1 asks for.
func characterPaneFigureOrigin(pane image.Rectangle) image.Point {
	return image.Pt(pane.Min.X, pane.Min.Y+2)
}

// characterPaneFigureRect includes all 240 source rows. The former 200-row
// crop cut off feet on both bitmap portraits and composed human figures.
// Corner controls draw over the figure after it, without reserving a full
// strip across the bottom of the pane.
func characterPaneFigureRect(pane image.Rectangle) image.Rectangle {
	o := characterPaneFigureOrigin(pane)
	return image.Rect(o.X, o.Y, o.X+160, o.Y+240).Intersect(pane)
}

// WHERE A CORNER AND THE FIGURE SHARE A PIXEL, THE GESTURE DECIDES — there is
// no geometric exclusion between the two (`DIV-308`).
//
// Corner controls overlap the figure at the top and bottom. A single left
// click on an overlapping pixel is the corner in the original and nothing
// else.
//
// This build reaches both from one button, so it splits them by gesture and
// keeps the original's answer for the single click: a press and release on an
// overlapping pixel runs the CORNER, and a press that travels past TapSlop
// arms the FIGURE's own drag (command.go's corner arm). The cost is that a
// tap-to-unequip on a marked slot under rect B or rect C now flips that corner
// instead; the drag remains, and this build's tap-to-unequip is its own
// gesture rather than a decoded one. Nothing here excludes a pixel from the
// slot map, so shopDollSlotAt and dollFigureSlotAt answer over the whole crop
// exactly as they did before this story.
