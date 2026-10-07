package ui

import (
	"fmt"
	"image"
	"strings"
)

// This file is the scenario vocabulary's POINTER half (1005 round 2, seventh
// pass): the three primitive edges a mouse gesture is made of, and the hit-test
// oracles that turn a named surface into the window pixel one of those edges is
// sent at.
//
// IT INVENTS NO GEOMETRY. Every oracle below finds its pixel by asking the
// production hit test the running game asks — dollFigureSlotAt for a doll slot,
// inventoryPackCellAt for a pack cell, shopGridControlAt for a shop surface,
// inventoryCaptures for "off every inventory box" — in headlessMenuButton's own
// shape, one function up in headless.go: scan, ask, take the answer. A mask that
// moved is therefore still found, and a surface the production build no longer
// answers for is an error rather than a press landing on the background.
//
// THE THREE EDGES ARE SEPARATE STEPS rather than one drag call, because the
// gesture this story is about is decided by what happens BETWEEN them:
// v.dragMoved accumulates over every frame the button is down (dragIntent,
// viewer.go), and TapSlop is what tells a tap from a drag. A scenario that could
// only say "drag" could not express a press that never moved, which is half the
// behaviour under test.

// frameToWindow converts a mission-frame position, which is what every hit test
// in this package answers in, into the WINDOW pixel a scenario must send an
// event at (1026 B4). It is the mirror of the mapping App.step's map arm applies
// on the way in, and it is the same conversion the town's own shop oracles
// already do through a.place.
//
// A frame pixel with no window pixel is an error rather than a nearby pixel:
// that happens only below 1:1, where sending the event at a neighbour would
// resolve to a different surface than the one the scenario named.
func (v *Viewer) frameToWindow(p image.Point, what string) (int, int, error) {
	wx, wy, ok := v.place.FrameToWindow(p)
	if !ok {
		return 0, 0, fmt.Errorf("%s: frame pixel (%d,%d) has no window pixel at this placement", what, p.X, p.Y)
	}
	return wx, wy, nil
}

// HeadlessPointer dispatches one pointer edge at one window pixel through
// App.step, the same dispatch a windowed session's Update produces.
//
// action is `press`, `move` or `release` for the primary button and
// `right-press`, `right-move` or `right-release` for the secondary. A press and
// a move both carry that button's Down flag on the viewer half, so the viewer's
// own drag accumulator sees a held button; a release carries neither, which is
// what ends the gesture. Dispatch uses App's shared headless clock.
//
// A `shift-` prefix sends the same edge with either Shift key held on that
// frame: readAppInput's ShiftHeld and readInput's Shift, the key's two reads.
//
// A `ctrl-` prefix, after any `shift-`, holds Ctrl on the map's read only.
//
// `wheel-up` and `wheel-down` send one wheel notch away from and toward the
// user at that pixel, on the picker's read and the map's read alike.
func (a *App) HeadlessPointer(action string, x, y int) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless pointer: nil application")
	}
	in := appInput{CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}, Unfocused: a.headlessUnfocused}
	edge, shifted := strings.CutPrefix(action, "shift-")
	in.ShiftHeld, in.Viewer.Shift = shifted, shifted
	// A `ctrl-` prefix holds Ctrl on the map's read, the cursor cascade's Ctrl.
	edge, ctrled := strings.CutPrefix(edge, "ctrl-")
	in.Viewer.Ctrl = ctrled
	switch edge {
	case "hover":
		// A pointer update with neither button held, through the live map door.
	case "press":
		in.PrimaryPressed = true
		in.Viewer.PrimaryDown = true
	case "move":
		in.Viewer.PrimaryDown = true
	case "release":
		in.PrimaryReleased = true
	case "right-press":
		in.SecondaryPressed = true
		in.Viewer.SecondaryDown = true
	case "right-move":
		in.Viewer.SecondaryDown = true
	case "right-release":
		in.SecondaryReleased = true
	case "wheel-up":
		in.WheelY, in.Viewer.WheelY = 1, 1
	case "wheel-down":
		in.WheelY, in.Viewer.WheelY = -1, -1
	default:
		return fmt.Errorf("headless pointer: unknown action %q; use hover, press, move, release, right-press, right-move, right-release, wheel-up or wheel-down, each with an optional shift- prefix", action)
	}
	a.headlessPointer, a.hasHeadlessPointer = image.Pt(x, y), true
	if a.step(in, a.headlessAt()) {
		return fmt.Errorf("headless pointer %s requested application exit", action)
	}
	return nil
}

// HeadlessDollSlotPoint is a window pixel the map screen's own doll figure
// answers for equipment slot (1..12), or an error naming what the doll is
// showing instead.
//
// THE PIXEL IS THE MIDDLE ONE OF THE SLOT'S OWN RUN, in raster order, rather
// than the first: a layer's first pixel is on its boundary, where a one-pixel
// error in either direction lands on the neighbouring layer, and the middle of
// the run is the furthest a cheap choice gets from both edges.
func (a *App) HeadlessDollSlotPoint(slot int) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	if slot < 1 || slot > 12 {
		return 0, 0, fmt.Errorf("headless doll slot %d: the doll has slots 1 to 12", slot)
	}
	box, ok := v.dollBox()
	if !ok {
		// THE REFUSAL NAMES WHICH GATE CLOSED, because the box has two and a
		// scenario author cannot see either of them: the display switch and the
		// selection the doll is drawn for.
		_, subject := v.dollSubject()
		return 0, 0, fmt.Errorf("headless doll slot %d: this frame draws no doll box "+
			"(switch on: %v, subject: %v, frame %dx%d, selected: %d of %d entities)",
			slot, v.hudShown(hudPanelDoll), subject, v.frameW, v.frameH,
			len(presentSelected(v.sel, v.entities)), len(v.entities))
	}
	var hits []image.Point
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if got, ok := v.dollFigureSlotAt(x, y); ok && got == slot-1 {
				hits = append(hits, image.Pt(x, y))
			}
		}
	}
	if len(hits) == 0 {
		return 0, 0, fmt.Errorf("headless doll slot %d: the figure's mask answers no pixel for it", slot)
	}
	p := hits[len(hits)/2]
	return v.frameToWindow(p, "headless doll slot")
}

// HeadlessPackCellPoint is a window pixel the map screen's own pack bar answers
// for visible cell index cell (0-based, the bar's own left-to-right order under
// the current scroll).
func (a *App) HeadlessPackCellPoint(cell int) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	bar, ok := v.packBarArea()
	if !ok {
		return 0, 0, fmt.Errorf("headless pack cell %d: this frame draws no pack bar", cell)
	}
	var hits []image.Point
	for y := bar.Min.Y; y < bar.Max.Y; y++ {
		for x := bar.Min.X; x < bar.Max.X; x++ {
			if got, ok := v.inventoryPackCellAt(x, y); ok && got == cell {
				hits = append(hits, image.Pt(x, y))
			}
		}
	}
	if len(hits) == 0 {
		return 0, 0, fmt.Errorf("headless pack cell %d: the bar answers no pixel for it", cell)
	}
	p := hits[len(hits)/2]
	return v.frameToWindow(p, "headless pack cell")
}

// HeadlessDollBoxPoint is a window pixel inside the doll BOX but not on any
// slot of the figure — the destination a pack-origin drag is released on
// (command.go's own `onPane && p.In(box)` arm), which is the pane's whole
// rectangle in figure and in statistics mode alike.
func (a *App) HeadlessDollBoxPoint() (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	box, ok := v.heldItemBox()
	if !ok {
		return 0, 0, fmt.Errorf("headless doll box: this frame draws none")
	}
	p := image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2)
	return v.frameToWindow(p, "headless doll box")
}

// HeadlessGroundPoint is a window pixel the map screen owns outright: inside
// no inventory box and no other HUD surface — groundSurfaceCaptures' own
// single question (inventory.go), which is the release `ITEM-DROP-008`'s
// ground drop is judged by (command.go's own two ground-drop arms).
//
// groundSurfaceCaptures, NOT its own three named disjuncts (round-2
// adversarial review, tenth pass): this scan used to ask inventoryCaptures,
// hudToggleCaptures and spellbookCaptures by name and left out
// minimapCaptures and panelCaptures, so a scenario asking for ground COULD
// have landed inside the minimap or the unit panel — the same two surfaces
// production's own ground-drop gate was missing until this pass. Reading the
// one shared function keeps the two in step: a fourth HUD surface added to
// groundSurfaceCaptures widens this scan for free, rather than needing its
// own name added here again.
//
// The scan excludes the window's edge-scroll bands. A generic ground gesture
// must not pan the camera after release and hide the next actor to inspect.
func (a *App) HeadlessGroundPoint() (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	for y := 4; y < v.cam.ViewH; y += 4 {
		for x := 4; x < v.cam.ViewW; x += 4 {
			if v.groundSurfaceCaptures(x, y) {
				continue
			}
			wx, wy, err := v.frameToWindow(image.Pt(x, y), "headless ground point")
			if err != nil {
				// This frame pixel has no window pixel of its own, which
				// happens only below 1:1. Another one further along may.
				continue
			}
			win := v.place.WindowSize()
			left, right, top, bottom := missionEdgeBands(wx, wy, win.X, win.Y, EdgeMargin)
			if left || right || top || bottom {
				continue
			}
			return wx, wy, nil
		}
	}
	return 0, 0, fmt.Errorf("headless ground point: no ground pixel outside HUD and edge-scroll bands")
}

// HeadlessDropCell is the world cell a release at a window pixel resolves to,
// through the same dropCellAt the ground drop itself reads.
func (a *App) HeadlessDropCell(x, y int) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	fx, fy := v.windowToFrame(x, y)
	dx, dy := v.dropCellAt(fx, fy)
	return int(dx), int(dy), nil
}

// HeadlessSpellPoint locates a visible spellbook entry. It does not select it;
// callers send the ordinary pointer press/release through App afterwards.
func (a *App) HeadlessSpellPoint(spell uint32) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	bar, cols, shown := v.spellbookBar()
	if shown {
		cells := bookCellRects(bar, cols)
		for i, entry := range v.spellbook {
			if entry.ID == spell && i < len(cells) {
				p := cells[i].Min.Add(cells[i].Max).Div(2)
				if x, y, err := v.frameToWindow(p, "headless spell point"); err == nil {
					return x, y, nil
				}
				// A downscaled window may not represent the exact centre pixel.
				// Find another actual pixel in this same cell, not a neighbour.
				for y := cells[i].Min.Y + 1; y < cells[i].Max.Y-1; y++ {
					for x := cells[i].Min.X + 1; x < cells[i].Max.X-1; x++ {
						if wx, wy, err := v.frameToWindow(image.Pt(x, y), "headless spell point"); err == nil {
							return wx, wy, nil
						}
					}
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("headless spell point: spell %d is not visible", spell)
}

// HeadlessShopPoint is a window pixel the OPEN SHOP ROOM answers for one of its
// four drag surfaces: `doll` slot 1..12, `shelf` cell, `pack` cell or `table`
// place, each index 0-based except the doll's, which is the equipment slot's own
// 1..12 numbering. It also answers for the member picker's own two buttons,
// `picker_prev` and `picker_next`, which are not drag surfaces at all: they are
// ordinary controls, so a press and release on one is a click. `shelf_pick`
// and `button` address the ordinary selector and footer. `doll_box` names
// no slot — the doll's own drawn area, for a release the mask-driven
// scan below cannot answer because no slot is marked at the point chosen.
//
// It asks shopGridControlAt, the production hit test app.go's own press arm
// asks, over the shop's own frame rectangle, and converts the answer back into
// window pixels through the same placement the press will be resolved by. The
// picker's two buttons come from ShopControlAt, which is the hit test the
// release arm asks for a control that is not a grid cell. `doll_box` asks
// neither: it is the shared character figure's own middle, unconditionally.
func (a *App) HeadlessShopPoint(kind string, index int) (int, int, error) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenTown {
		return 0, 0, fmt.Errorf("headless shop point: the town screen is not showing")
	}
	view, inShop := townShopScreen(a.flow.town)
	if !inShop {
		return 0, 0, fmt.Errorf("headless shop point: no shop room is open")
	}
	if kind == "doll_box" {
		// THE DOLL'S OWN DRAWN AREA, NO PARTICULAR SLOT — the shared character
		// figure's middle, shopDollAreaAt's own geometry with no mask consulted
		// (shopscreen.go), HeadlessDollBoxPoint's own reasoning restated for the
		// shop (round-2 adversarial review, tenth pass): the destination a pack-
		// or shelf-origin drag resolves to when the release names no marked slot
		// at all — a slot this member wears nothing in, or a member wearing
		// nothing anywhere, which the mask- driven scan below can never answer
		// for.
		p := image.Pt((townCharacterFigure.Min.X+townCharacterFigure.Max.X)/2,
			(townCharacterFigure.Min.Y+townCharacterFigure.Max.Y)/2)
		wx, wy, ok := a.nativeFrameToWindow(p)
		if !ok {
			return 0, 0, fmt.Errorf("headless shop doll box: stands outside the placed frame")
		}
		return wx, wy, nil
	}
	want := ShopControl{Index: index}
	grid := true
	dollSlot := -1
	switch kind {
	case "doll":
		if index < 1 || index > 12 {
			return 0, 0, fmt.Errorf("headless shop doll slot %d: the doll has slots 1 to 12", index)
		}
		want = ShopControl{Kind: ShopControlDoll, Index: index - 1}
		dollSlot = index - 1
	case "shelf":
		want.Kind = ShopControlShelfCell
	case "pack":
		want.Kind = ShopControlPackCell
	case "table":
		want.Kind = ShopControlTableCell
	case "picker_prev":
		want, grid = ShopControl{Kind: ShopControlPickerPrev}, false
	case "picker_next":
		want, grid = ShopControl{Kind: ShopControlPickerNext}, false
	case "shelf_pick":
		want, grid = ShopControl{Kind: ShopControlShelfPick, Index: index}, false
	case "button":
		want, grid = ShopControl{Kind: ShopControlButton, Index: index}, false
	case "merchant":
		want, grid = ShopControl{Kind: ShopControlMerchant}, false
	default:
		return 0, 0, fmt.Errorf("headless shop point: unknown surface %q; doll, doll_box, shelf, "+
			"pack, table, picker_prev, picker_next, shelf_pick, button and merchant are supported", kind)
	}
	region := shopWantRegion(want)
	hits := shopHitPixels(view, want, grid, dollSlot, region)
	if len(hits) == 0 {
		return 0, 0, fmt.Errorf("headless shop point: the open room answers no pixel for %s %d", kind, index)
	}
	p := hits[len(hits)/2]
	wx, wy, ok := a.nativeFrameToWindow(p)
	if !ok {
		return 0, 0, fmt.Errorf("headless shop point: %s %d stands outside the placed frame", kind, index)
	}
	return wx, wy, nil
}

func shopWantRegion(want ShopControl) image.Rectangle {
	frame := image.Rect(0, 0, shopScreenW, shopScreenH)
	region := frame
	switch want.Kind {
	case ShopControlDoll:
		region = frame.Intersect(townCharacterFigure)
	case ShopControlShelfCell:
		region = frame.Intersect(ShopShelfCellRect(want.Index))
	case ShopControlPackCell:
		region = frame.Intersect(ShopPackCellRect(want.Index))
	case ShopControlTableCell:
		region = frame.Intersect(ShopTableCellRect(want.Index))
	case ShopControlPickerPrev:
		region = frame.Intersect(shopPickerPrevRect)
	case ShopControlPickerNext:
		region = frame.Intersect(shopPickerNextRect)
	case ShopControlMerchant:
		region = frame.Intersect(shopMerchantRect)
	case ShopControlButton:
		region = frame.Intersect(ShopButtonRect(want.Index))
	case ShopControlShelfPick:
		region = image.Rectangle{}
		if want.Index >= 0 && want.Index < len(shopShelfPickRects) {
			region = frame.Intersect(shopShelfPickRects[want.Index])
		}
	}
	return region
}

func shopHitPixels(view ShopScreenView, want ShopControl, grid bool, dollSlot int, region image.Rectangle) []image.Point {
	var hits []image.Point
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			p := image.Pt(x, y)
			var got ShopControl
			var ok bool
			switch {
			case dollSlot >= 0:
				// THE DOLL SURFACE IS ASKED OF OrdinaryDollMask, NEVER THE
				// POSSIBLY-SUPPRESSED SlotMask shopGridControlAt reads (round-2
				// adversarial review, eleventh pass, counterexample C1): a scenario that
				// names "doll slot N" while a drag on that same slot is already armed
				// and past TapSlop would otherwise ask a mask this build's own
				// suppression has already cleared at that slot, and get back "no pixel
				// answers" for the origin it is trying to release ON — the exact
				// production identity shopReleaseIsOrigin exists to preserve. Resolving
				// against OrdinaryDollMask directly, exactly as shopReleaseIsOrigin
				// does, answers the slot's true location regardless of any drag this
				// scenario step is itself mid-way through.
				n, hit := shopDollSlotAt(view, view.OrdinaryDollMask, p)
				got, ok = ShopControl{Kind: ShopControlDoll, Index: n}, hit
			case grid:
				got, ok = shopGridControlAt(view, p)
			default:
				got, ok = shopScreenControlAt(view, p)
			}
			if ok && got == want {
				hits = append(hits, p)
			}
		}
	}
	return hits
}

// HeadlessShopIdlePoint is a window pixel inside the shop frame that names no
// grid surface at all — the release a drag ends on when it lands on nothing,
// and the point a move step passes through without arming anything.
func (a *App) HeadlessShopIdlePoint() (int, int, error) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenTown {
		return 0, 0, fmt.Errorf("headless shop idle point: the town screen is not showing")
	}
	view, inShop := townShopScreen(a.flow.town)
	if !inShop {
		return 0, 0, fmt.Errorf("headless shop idle point: no shop room is open")
	}
	for y := 0; y < shopScreenH; y += 2 {
		for x := 0; x < shopScreenW; x += 2 {
			p := image.Pt(x, y)
			if _, ok := shopGridControlAt(view, p); ok {
				continue
			}
			if _, ok := shopScreenControlAt(view, p); ok {
				continue
			}
			if wx, wy, ok := a.nativeFrameToWindow(p); ok {
				return wx, wy, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("headless shop idle point: every pixel of the room names a control")
}

func (a *App) headlessViewer() (*Viewer, error) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return nil, fmt.Errorf("headless pointer: no map screen is open")
	}
	if a.flow.screen != ScreenMap {
		return nil, fmt.Errorf("headless pointer: screen is %s, not the map", a.flow.screen)
	}
	return a.flow.viewer, nil
}

// headlessWorldMap is the open world map screen's own presentation snapshot,
// the same one WorldMapClick and WorldMapHover are dispatched against
// (stepTownAt, app.go). An error names why there is none: the town screen is
// not showing, or it is showing something other than the world map.
func (a *App) headlessWorldMap() (WorldMapView, error) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenTown {
		return WorldMapView{}, fmt.Errorf("headless world map point: the town screen is not showing")
	}
	world, onMap := townWorldMapScreen(a.flow.town)
	if !onMap {
		return WorldMapView{}, fmt.Errorf("headless world map point: the world map is not open")
	}
	return world.WorldMapView(), nil
}

// HeadlessWorldMapMissionPoint is a window pixel the world map screen's own
// scroll cards answer for mission number number, the surface WorldMapClick
// reads to select a mission and start travel to it (`TOWN-118`). Only a
// mission on the CURRENT card page can be reached, exactly as a real cursor
// is limited by WorldMapCardAt itself; a mission that exists but sits on
// another page is refused rather than silently resolved against the wrong
// card.
func (a *App) HeadlessWorldMapMissionPoint(number int) (int, int, error) {
	view, err := a.headlessWorldMap()
	if err != nil {
		return 0, 0, err
	}
	for i, m := range view.Missions {
		if m.Number != number {
			continue
		}
		if i < view.CardBase || i >= view.CardBase+worldMapCardPage {
			return 0, 0, fmt.Errorf("headless world map mission %d: not on the current card page (base %d)",
				number, view.CardBase)
		}
		r := WorldMapCardRect(i - view.CardBase)
		wx, wy, ok := a.nativeFrameToWindow(image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2))
		if !ok {
			return 0, 0, fmt.Errorf("headless world map mission %d: stands outside the placed frame", number)
		}
		return wx, wy, nil
	}
	return 0, 0, fmt.Errorf("headless world map mission %d: no such mission is offered", number)
}

// HeadlessWorldMapTownPoint is a window pixel the world map screen's own
// separate return-to-town card answers for — WorldMapCardAt's slot -1.
func (a *App) HeadlessWorldMapTownPoint() (int, int, error) {
	if _, err := a.headlessWorldMap(); err != nil {
		return 0, 0, err
	}
	r := WorldMapCardRect(-1)
	wx, wy, ok := a.nativeFrameToWindow(image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2))
	if !ok {
		return 0, 0, fmt.Errorf("headless world map town card: stands outside the placed frame")
	}
	return wx, wy, nil
}

// HeadlessWorldMapMissPoint is a window pixel the world map screen answers
// for neither a scroll card nor a mission region — WorldMapClick's own skip
// arm, the click a scenario sends to fast-forward the route reveal to its
// own end while travel is in progress (`TOWN-121`). It is searched over the
// view's own 640x480 frame, the size ComposeWorldMap always draws.
func (a *App) HeadlessWorldMapMissPoint() (int, int, error) {
	view, err := a.headlessWorldMap()
	if err != nil {
		return 0, 0, err
	}
	for y := 4; y < 480; y += 4 {
		for x := 4; x < 640; x += 4 {
			p := image.Pt(x, y)
			if _, ok := WorldMapCardAt(view, p); ok {
				continue
			}
			if _, ok := WorldMapRegionAt(view, p); ok {
				continue
			}
			if wx, wy, ok := a.nativeFrameToWindow(p); ok {
				return wx, wy, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("headless world map miss point: every pixel of this frame names a card or a region")
}

// HeadlessDocumentPoint is a window pixel the campaign documents panel's own
// hit test answers for one of its three controls: `left`, `right` or `ok`.
//
// IT ASKS THE PRODUCTION HIT TEST, this file's own rule: the centre of the
// named control's rectangle is offered to documentControlAt, and a rectangle
// that moved is therefore still found. The mapping out is a.place's, the same
// 640x480 town-family placement stepDocuments maps the cursor in through, so a
// scenario's pixel and the production read of it are one conversion.
func (a *App) HeadlessDocumentPoint(control string) (int, int, error) {
	if a == nil || a.flow == nil {
		return 0, 0, fmt.Errorf("headless document point: nil application")
	}
	if a.flow.screen != ScreenDocuments {
		return 0, 0, fmt.Errorf("headless document point %q: the documents panel is not open (screen is %s)",
			control, a.flow.screen)
	}
	var want int
	switch strings.ToLower(strings.TrimSpace(control)) {
	case "left":
		want = docControlLeft
	case "right":
		want = docControlRight
	case "ok":
		want = docControlOK
	default:
		return 0, 0, fmt.Errorf("headless document point %q: left, right and ok are the three", control)
	}
	rect := docControlRect(want)
	mid := image.Pt((rect.Min.X+rect.Max.X)/2, (rect.Min.Y+rect.Max.Y)/2)
	if got, ok := documentControlAt(mid.X, mid.Y); !ok || got != want {
		return 0, 0, fmt.Errorf("headless document point %q: the panel's hit test does not answer for its own rectangle's middle", control)
	}
	wx, wy, ok := a.nativeFrameToWindow(mid)
	if !ok {
		return 0, 0, fmt.Errorf("headless document point %q: frame pixel (%d,%d) has no window pixel at this placement",
			control, mid.X, mid.Y)
	}
	return wx, wy, nil
}

// HeadlessHelpBarPoint is a window pixel on the open help panel's scroll bar:
// the middle of the `up` or `down` arrow, of the `thumb`, or the bottom of the
// `track`, at the scroll the panel stands at.
func (a *App) HeadlessHelpBarPoint(part string) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	if !v.HelpOpen() {
		return 0, 0, fmt.Errorf("headless help bar: no help panel is open")
	}
	l := v.noticeLayout()
	if l.Scrollbar.Empty() {
		return 0, 0, fmt.Errorf("headless help bar: the help text fits and draws no scroll bar")
	}
	g := v.helpGeo()
	up, down, track, thumb := helpBarParts(l.Scrollbar, v.help.scroll, g.lines, g.visible)
	var r image.Rectangle
	switch part {
	case "up":
		r = up
	case "down":
		r = down
	case "thumb":
		r = thumb
	case "track":
		r = image.Rect(track.Min.X, track.Max.Y-4, track.Max.X, track.Max.Y)
	default:
		return 0, 0, fmt.Errorf("headless help bar: unknown part %q; use up, down, thumb or track", part)
	}
	_, at, scale, ok := v.noticePresent()
	if !ok {
		return 0, 0, fmt.Errorf("headless help bar: the help panel draws no picture")
	}
	// Below 1:1 some panel pixels have no window pixel; take the one of the
	// part nearest its middle that has one.
	mid := r.Min.Add(r.Max).Div(2)
	for d := 0; d < max(r.Dx(), r.Dy()); d++ {
		for _, p := range []image.Point{mid.Add(image.Pt(0, d)), mid.Sub(image.Pt(0, d)), mid.Add(image.Pt(d, 0)), mid.Sub(image.Pt(d, 0))} {
			if !p.In(r) {
				continue
			}
			fp := image.Pt(at.X+int(float64(p.X)*scale), at.Y+int(float64(p.Y)*scale))
			if x, y, ok := v.place.FrameToWindow(fp); ok {
				return x, y, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("headless help bar: no window pixel on the %s", part)
}
