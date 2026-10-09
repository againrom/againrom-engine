package ui

import (
	"image"
	"testing"
	"time"
)

// pressAt and releaseAt are clickAt's own two calls (tippanel_overlap_test.go),
// split apart so a gesture can press at one point and release at another.
func pressAt(a *App, now time.Time, p image.Point) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
}

func releaseAt(a *App, now time.Time, p image.Point) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
}

// TestSchoolTipPanelCrossingLeavesNoLatch is the inSurface call site (shared
// by tavern and school): a press on a live button and a release inside the
// panel, and the reverse. Button well 0 (townSurfaceButtonWells) sits at
// global (484,71)-(624,117), entirely past SchoolTipRect's own x=456 edge, so
// it is live regardless of the panel's own width.
func TestSchoolTipPanelCrossingLeavesNoLatch(t *testing.T) {
	tip := TipPanelView{Rect: SchoolTipRect, Text: "the school teaches skills", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	cells := make([]TownSurfaceCell, 5)
	for i := range cells {
		cells[i].Enabled = true
	}
	buttons := []TownSurfaceButton{{Enabled: true}, {Enabled: true}}
	view := TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1, Cells: cells, Buttons: buttons, Tip: tip}
	town := &fakeTipSurfaceEnumTown{view: view}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the school surface")
	}

	well := townSurfaceButtonWells[TownSurfaceSchool][0].Add(TownUpperRegion.Min)
	if well.Overlaps(SchoolTipRect) {
		t.Fatalf("fixture error: school button well 0 (%v) overlaps SchoolTipRect %v", well, SchoolTipRect)
	}
	live, ok := sampleInside(well)
	if !ok {
		t.Fatal("fixture error: school button well 0 is empty")
	}
	panelPt, ok := sampleInside(tipBackgroundZone(SchoolTipRect))
	if !ok {
		t.Fatal("fixture error: SchoolTipRect's own background zone is empty")
	}

	now := time.Unix(1_700_000_000, 0)

	// Press on the live button, release inside the panel.
	pressAt(a, now, live)
	releaseAt(a, now, panelPt)
	if len(town.surfaceClicks) != 0 {
		t.Fatalf("surfaceClicks = %v after a press on a live button and a release inside the showing panel, want none", town.surfaceClicks)
	}
	if a.townSurfacePress != (TownSurfaceControl{}) {
		t.Fatalf("townSurfacePress = %+v after the release the panel swallowed, want the zero value: the latch is still armed", a.townSurfacePress)
	}
	// The next release on that same button, with no new press between, must
	// activate nothing either.
	releaseAt(a, now, live)
	if len(town.surfaceClicks) != 0 {
		t.Fatalf("surfaceClicks = %v after a lone release on the button the earlier press named, want none: the old latch fired", town.surfaceClicks)
	}

	// Press inside the panel, release on the live button.
	pressAt(a, now, panelPt)
	releaseAt(a, now, live)
	if len(town.surfaceClicks) != 1 || town.surfaceClicks[0].Kind != TownSurfaceControlCell {
		t.Fatalf("surfaceClicks = %v; list down must select its covered class slot and outside release must not activate a button", town.surfaceClicks)
	}
}

// fakeTipShopCrossingTown is a shop room with a Showing tip and a doll mask.
// It duplicates fakeShopTown's own suppression tracking (town_test.go, which
// has no Tip field) rather than widening that shared fixture, so this file's
// own addition cannot change any other shop test's fixture shape.
type fakeTipShopCrossingTown struct {
	fakeTown
	tipCloseToggleRecorder
	tip          TipPanelView
	mask         *SlotMask
	dollIcon     *image.RGBA
	dragged      [][2]ShopControl
	clicked      []ShopControl
	suppressed   []int
	suppressSlot int
}

func (f *fakeTipShopCrossingTown) AtTownShop() bool { return true }

func (f *fakeTipShopCrossingTown) ShopScreen() ShopScreenView {
	v := ShopScreenView{Purse: 100, Buy: 40, Sell: 9, Chosen: -1, TipPanel: f.tip,
		Character: TownCharacterView{HasSubject: true, MemberCount: 1}}
	v.OrdinaryDollMask = f.mask
	v.SlotMask = f.mask
	if f.suppressSlot != 0 {
		v.SlotMask = suppressedCopy(f.mask, f.suppressSlot)
	}
	if f.dollIcon != nil {
		v.SlotIcon[3] = f.dollIcon
	}
	return v
}

func (f *fakeTipShopCrossingTown) ShopClick(c ShopControl) TownAction {
	f.clicked = append(f.clicked, c)
	return TownAction{}
}

func (f *fakeTipShopCrossingTown) ShopScroll(region ShopWheelRegion, rows int) {}

func (f *fakeTipShopCrossingTown) ShopDrag(from, to ShopControl) TownAction {
	f.dragged = append(f.dragged, [2]ShopControl{from, to})
	return TownAction{}
}

func (f *fakeTipShopCrossingTown) ShopSuppressDoll(slot int) {
	f.suppressed = append(f.suppressed, slot)
	f.suppressSlot = slot
}

// TestShopTipPanelCrossingLeavesNoLatch is the inShop call site: a drag
// begun on the doll (shopDragDollPoint, shopdrag_test.go — slot 4, one-based)
// moved into the panel and released there, then a press inside the panel
// released on a live shelf cell.
func TestShopTipPanelCrossingLeavesNoLatch(t *testing.T) {
	tip := TipPanelView{Rect: ShopTipRect(), Text: "the shop buys and sells", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	town := &fakeTipShopCrossingTown{tip: tip, mask: shopTestMask(), dollIcon: image.NewRGBA(image.Rect(0, 0, 1, 1))}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop fixture")
	}

	panelPt, ok := sampleInside(tipBackgroundZone(ShopTipRect()))
	if !ok {
		t.Fatal("fixture error: ShopTipRect's own background zone is empty")
	}
	now := time.Unix(1_700_000_000, 0)

	// Press the doll (a real origin), move into the panel past TapSlop —
	// the per-frame arm runs on every frame regardless of the panel, so this
	// step alone already arms the drag and captures its icon — then release
	// inside the panel.
	pressAt(a, now, shopDragDollPoint)
	a.step(appInput{CursorX: panelPt.X, CursorY: panelPt.Y}, now)
	if !a.shopDragArmed || a.shopDragMoved < TapSlop || a.shopDragIcon == nil {
		t.Fatalf("fixture error: armed=%v moved=%d icon=%v after the move into the panel, want armed, moved >= %d and a captured icon",
			a.shopDragArmed, a.shopDragMoved, a.shopDragIcon, TapSlop)
	}
	releaseAt(a, now, panelPt)

	if len(town.dragged) != 0 {
		t.Fatalf("dragged = %v after a release inside the showing panel, want none", town.dragged)
	}
	if a.shopDragArmed || a.shopDragMoved != 0 || a.shopDragIcon != nil {
		t.Fatalf("armed=%v moved=%d icon=%v after the swallowed release, want false/0/nil: the drag machine is still latched",
			a.shopDragArmed, a.shopDragMoved, a.shopDragIcon)
	}

	// The following idle frame (no press, no release) must push no doll
	// suppression: the per-frame arm reads a.shopDragArmed unconditionally,
	// so a latch left armed here would still suppress the doll's own slot
	// with the mouse button already up.
	town.suppressed = nil
	a.step(appInput{CursorX: panelPt.X, CursorY: panelPt.Y}, now)
	if len(town.suppressed) == 0 || town.suppressed[len(town.suppressed)-1] != 0 {
		t.Fatalf("suppressed = %v on the idle frame after the swallowed release, want a trailing 0 (no suppression)", town.suppressed)
	}

	// Press inside the panel, release on a live shelf cell, uninvolved in
	// the doll gesture above: a stale armed drag from an earlier cycle must
	// not resolve into a completed ShopDrag (the brief's own repro:
	// "ShopDrag({ShelfCell 0} -> {TableCell 0}): an item moves from a
	// gesture that began on the tip panel"). A plain ShopClick on the
	// release point is NOT asserted here: with nothing armed, the shop's own
	// release handler (`case onFrame`) resolves a tap at wherever the mouse
	// comes up regardless of where the press was — proven unrelated to the
	// panel by a press at a dead pixel producing the same click — so it is
	// pre-existing shop behaviour this hotfix's scope does not touch.
	pressAt(a, now, panelPt)
	releaseAt(a, now, shopDragShelfPoint)
	if len(town.dragged) != 0 {
		t.Fatalf("dragged = %v after a press inside the showing panel and a release on a live shelf cell, want none: a stale-armed drag fired", town.dragged)
	}
}

// TestPreCreateTipPanelCrossingLeavesNoLatch is the stepPreCreate call site.
//
// It needs one point on a choice that the showing panel does NOT swallow.
// ChargenTipRect is now the researched absolute (160,280)-(472,480)
// (TOWN-314), and at this fixture's own 160x240 portraits every one of the
// four choices intersects it, so no choice rect is clear as a whole any
// more.
//
// The precondition was always the point, not the rect: liveChoicePoint below
// searches choice 1's own rect for a pixel TipPanelControlAt does not consume
// and fails if there is none. That witnesses liveness directly rather than
// inferring it from a non-overlap, so it holds at any rect and any portrait
// size.
// liveChoicePoint returns a pixel of r that the chargen's own showing tip
// panel does not consume, scanning row by row. It answers the precondition
// "this choice can still be pressed while the tip shows" directly, from
// production's own TipPanelControlAt, rather than from a rectangle test.
func liveChoicePoint(c *Chargen, r image.Rectangle) (image.Point, bool) {
	tip := c.TipPanel()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p := image.Pt(x, y)
			if _, consumed := TipPanelControlAt(tip, p); !consumed {
				return p, true
			}
		}
	}
	return image.Point{}, false
}

func TestPreCreateTipPanelCrossingLeavesNoLatch(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Forward: image.NewRGBA(image.Rect(0, 0, 96, 74)), Font: shopTipTestFont()}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 160, 240))
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	setup.TipsOn = true
	setup.TipArt = tipTestArt()
	setup.TipSelect[0] = "pick a hero"
	a := newTestApp(t, appRows(1), okLoader(t))
	c := NewChargen(setup)
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	if !c.TipPanel().Showing() {
		t.Fatal("fixture tip is not Showing()")
	}

	liveID := chargenChoice0 + chargenControl(1)
	liveRect := preControlRect(c, liveID)
	live, ok := liveChoicePoint(c, liveRect)
	if !ok {
		t.Fatalf("fixture error: every pixel of choice 1's own rect %v is swallowed by the showing panel %v -- a press on a live choice cannot be driven", liveRect, c.TipPanel().Rect)
	}
	// Sampled from the showing panel's own shrunk rect: ChargenTipRect's
	// centre lies above it, on choice 3, which a press now selects.
	panelPt, ok := sampleInside(tipBackgroundZone(c.TipPanel().Rect))
	if !ok {
		t.Fatal("fixture error: ChargenTipRect's own background zone is empty")
	}

	before := c.PreChoice()

	// A press on a live choice chooses it at once (VIDEO-SFX-058); the
	// release inside the panel changes nothing and clears the latch.
	pressAt(a, now, live)
	if c.PreChoice() != 1 {
		t.Fatalf("PreChoice() = %d after a press on live choice 1, want 1", c.PreChoice())
	}
	releaseAt(a, now, panelPt)
	if c.PreChoice() != 1 {
		t.Fatalf("PreChoice() = %d after the release inside the showing panel, want 1 (unchanged)", c.PreChoice())
	}
	if a.chargenPress != chargenNone {
		t.Fatalf("chargenPress = %v after the release the panel swallowed, want chargenNone: the latch is still armed", a.chargenPress)
	}
	// The next release on that same choice, with no new press between, must
	// activate nothing either.
	c.SelectPreChoice(before)
	releaseAt(a, now, live)
	if c.PreChoice() != before {
		t.Fatalf("PreChoice() = %d after a lone release on the choice the earlier press named, want %d (unchanged): the old latch fired", c.PreChoice(), before)
	}

	// Press inside the panel, release on a live choice: this must not
	// complete either, and must not arm the double-click window — a later,
	// genuine click at the same point must not read as its second half.
	pressAt(a, now, panelPt)
	releaseAt(a, now, live)
	if c.PreChoice() != before {
		t.Fatalf("PreChoice() = %d after a press inside the showing panel and a release on a live choice, want %d (unchanged)", c.PreChoice(), before)
	}
	if a.chargenChoiceClick != chargenNone {
		t.Fatalf("chargenChoiceClick = %v after a press inside the panel and a release on a live choice, want chargenNone: the illegitimate click armed the double-click window", a.chargenChoiceClick)
	}
}
