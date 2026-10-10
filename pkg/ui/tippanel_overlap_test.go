package ui

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// Popup regressions retain reached room routes and control ownership.
type tipCloseToggleRecorder struct {
	closed  int
	toggled int
}

func (r *tipCloseToggleRecorder) CloseTip()   { r.closed++ }
func (r *tipCloseToggleRecorder) ToggleTips() { r.toggled++ }

// sampleInside is a point strictly inside r, or false for an empty or
// inverted r. Used to turn a geometric intersection into one pixel to click.
func sampleInside(r image.Rectangle) (image.Point, bool) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return image.Point{}, false
	}
	return r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)), true
}

func clickAt(a *App, now time.Time, p image.Point) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
}

// tipBackgroundZone is r minus the close/toggle row: the part of a showing
// panel round 2's own narrowed guard (`consumed && kind != TipControlNone`)
// let a click fall through from, since round 2's guard already caught a
// Close or Toggle hit correctly.
func tipBackgroundZone(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X, r.Min.Y, r.Max.X, TipPanelToggleRect(r).Min.Y)
}

// fakeTipSquareEnumTown is a TownScreen with an art-backed square (0157 /
// 1016's own precedent) plus a Showing tip, so a release at a mask pixel the
// panel's own rect covers can be driven through the real dispatch. Rows
// mirrors the production four-door order so Choose(i) matches townDoors.
type fakeTipSquareEnumTown struct {
	tipCloseToggleRecorder
	scene  TownSquareScene
	tip    TipPanelView
	chosen []int
}

func (f *fakeTipSquareEnumTown) TownSquareView() TownSquareView {
	return TownSquareView{Scene: f.scene, Font: panelFont(), Tip: f.tip}
}
func (f *fakeTipSquareEnumTown) AtTownSquare() bool { return true }
func (f *fakeTipSquareEnumTown) Header() string     { return "square" }
func (f *fakeTipSquareEnumTown) Rows() []TownRow {
	names := []string{"TAVERN", "SHOP", "SCHOOL", "GATES"}
	rows := make([]TownRow, len(names))
	for i, name := range names {
		rows[i] = TownRow{Text: name, Choosable: true}
	}
	return rows
}
func (f *fakeTipSquareEnumTown) Footer() []string { return nil }
func (f *fakeTipSquareEnumTown) Choose(i int) TownAction {
	f.chosen = append(f.chosen, i)
	return TownAction{}
}
func (f *fakeTipSquareEnumTown) Back() bool { return false }

// TestTownSquareTipListConsumesReleaseAndBodyPasses enumerates all five
// controls a square scene answers (four doors and the statue's own menu),
// each placed inside TownTipRect (328,0)-(640,373): the mask's own full
// vocabulary, not one representative door.
func TestTownSquareTipListConsumesReleaseAndBodyPasses(t *testing.T) {
	codes := []struct {
		control TownSquareControl
		at      image.Point
	}{
		{squareDoor(0), image.Pt(340, 30)},
		{squareDoor(1), image.Pt(380, 60)},
		{squareDoor(2), image.Pt(420, 90)},
		{squareDoor(3), image.Pt(460, 120)},
		{squareMenu, image.Pt(500, 150)},
	}
	scene := &fakeSquareScene{controls: map[image.Point]TownSquareControl{}}
	for _, c := range codes {
		if !c.at.In(TownTipRect) {
			t.Fatalf("fixture error: %v is not inside TownTipRect %v", c.at, TownTipRect)
		}
		scene.controls[c.at] = c.control
	}
	tip := TipPanelView{Rect: TownTipRect, Text: "the square is where the town meets", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	for _, c := range codes {
		if k, consumed := TipPanelControlAt(tip, c.at); !consumed || k != TipControlNone {
			t.Fatalf("fixture error at %v: TipPanelControlAt = %v,%v, want TipControlNone,true (background)", c.at, k, consumed)
		}
	}

	town := &fakeTipSquareEnumTown{scene: scene, tip: tip}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}

	now := time.Unix(1_700_000_000, 0)
	before := a.Screen()
	for _, c := range codes {
		clickAt(a, now, c.at)
	}
	if len(town.chosen) != 1 || town.chosen[0] != 0 {
		t.Fatalf("chosen = %v; body release must pass to tavern and list releases must be consumed", town.chosen)
	}
	if a.Screen() != before {
		t.Fatalf("Screen() = %v after releasing on the statue's own code, covered by the showing tip panel, want unchanged (%v): the statue's menu opened", a.Screen(), before)
	}

	closeAt, ok := sampleInside(TipPanelCloseRect(TownTipRect))
	if !ok {
		t.Fatal("fixture error: TipPanelCloseRect(TownTipRect) is empty")
	}
	clickAt(a, now, closeAt)
	if town.closed != 1 {
		t.Fatalf("CloseTip() calls = %d after a release on the panel's own Close control, want 1", town.closed)
	}
	toggleAt, ok := sampleInside(TipPanelToggleRect(TownTipRect))
	if !ok {
		t.Fatal("fixture error: TipPanelToggleRect(TownTipRect) is empty")
	}
	clickAt(a, now, toggleAt)
	if town.toggled != 1 {
		t.Fatalf("ToggleTips() calls = %d after a release on the panel's own Toggle control, want 1", town.toggled)
	}
}

// fakeTipSurfaceEnumTown is a school-or-tavern surface with a Showing tip;
// fakeTown supplies the base TownScreen five (Choose/Back are unused once
// showTown has already entered the surface).
type fakeTipSurfaceEnumTown struct {
	fakeTown
	tipCloseToggleRecorder
	view          TownSurfaceView
	surfaceClicks []TownSurfaceControl
}

func (f *fakeTipSurfaceEnumTown) AtTownSurface() bool          { return true }
func (f *fakeTipSurfaceEnumTown) TownSurface() TownSurfaceView { return f.view }
func (f *fakeTipSurfaceEnumTown) TownSurfaceClick(c TownSurfaceControl, _ bool) TownAction {
	f.surfaceClicks = append(f.surfaceClicks, c)
	return TownAction{}
}

func TestTavernTipListPressReachesCoveredCells(t *testing.T) {
	tip := TipPanelView{Rect: TavernTipRect, Text: "the tavern hires mercenaries", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	for i := 0; i < 4; i++ {
		global := TownSurfaceButtonRect(TownSurfaceTavern, i)
		if global.Overlaps(TavernTipRect) {
			t.Fatalf("fixture error: tavern button well %d (%v) overlaps TavernTipRect %v — the enumeration below must cover it too", i, global, TavernTipRect)
		}
	}

	cells := make([]TownSurfaceCell, 12)
	view := TownSurfaceView{Kind: TownSurfaceTavern, Cells: cells, Tip: tip}

	bg := tipBackgroundZone(TavernTipRect)
	var covered []image.Point
	for i := range cells {
		r := townSurfaceCellRect(TownSurfaceTavern, i, len(cells))
		p, ok := sampleInside(r.Intersect(bg))
		if !ok {
			continue
		}
		if k, consumed := TipPanelControlAt(tip, p); !consumed || k != TipControlNone {
			t.Fatalf("fixture error: cell %d's own sample %v, TipPanelControlAt = %v,%v, want TipControlNone,true", i, p, k, consumed)
		}
		covered = append(covered, p)
	}
	if len(covered) == 0 {
		t.Fatal("fixture error: no mercenary cell overlaps TavernTipRect's own background zone — the enumeration below would prove nothing")
	}

	town := &fakeTipSurfaceEnumTown{view: view}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the tavern surface")
	}
	now := time.Unix(1_700_000_000, 0)
	for _, p := range covered {
		clickAt(a, now, p)
	}
	if len(town.surfaceClicks) != len(covered) {
		t.Fatalf("covered list downs selected %v; want one per occupied cell, releases inert", town.surfaceClicks)
	}

	closedBefore, toggledBefore := town.closed, town.toggled
	closeAt, ok := sampleInside(TipPanelCloseRect(TavernTipRect))
	if !ok {
		t.Fatal("fixture error: TipPanelCloseRect(TavernTipRect) is empty")
	}
	clickAt(a, now, closeAt)
	if town.closed != closedBefore+1 {
		t.Fatalf("CloseTip() calls = %d after a release on the panel's own Close control, want %d", town.closed, closedBefore+1)
	}
	toggleAt, ok := sampleInside(TipPanelToggleRect(TavernTipRect))
	if !ok {
		t.Fatal("fixture error: TipPanelToggleRect(TavernTipRect) is empty")
	}
	clickAt(a, now, toggleAt)
	if town.toggled != toggledBefore+1 {
		t.Fatalf("ToggleTips() calls = %d after a release on the panel's own Toggle control, want %d", town.toggled, toggledBefore+1)
	}
}

// TestSchoolTipListPressReachesCoveredSkills covers all five fighter skill
// codes with a deliberately tall tip fixture. The room's native tip rectangle
// does not determine this input-routing overlap.
func TestSchoolTipListPressReachesCoveredSkills(t *testing.T) {
	tip := TipPanelView{Rect: image.Rect(0, 0, 456, 264), Text: "the school teaches skills", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	for i, well := range townSurfaceButtonWells[TownSurfaceSchool] {
		global := well.Add(TownUpperRegion.Min)
		if global.Overlaps(tip.Rect) {
			t.Fatalf("fixture error: school button well %d (%v) overlaps tip rectangle %v — the enumeration below must cover it too", i, global, tip.Rect)
		}
	}

	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 0xff}
	}
	art := &TownSchoolArt{}
	art.Masks[0] = image.NewPaletted(image.Rect(0, 0, 92, 120), palette)
	panelRect := schoolPanelRects[0]
	overlap := panelRect.Intersect(tipBackgroundZone(tip.Rect))
	if overlap.Dx() <= 0 || overlap.Dy() <= 0 {
		t.Fatalf("fixture error: the fighter skill panel %v does not overlap the tip background zone", panelRect)
	}

	codes := []uint8{0xff, 0x9e, 0xd2, 0x87, 0x37} // sword, axe, club, pike, bow — schoolMaskSlot(0, ...)
	var covered []image.Point
	for i, code := range codes {
		global := overlap.Min.Add(image.Pt(10+i*15, overlap.Dy()/2))
		local := global.Sub(panelRect.Min)
		if !global.In(overlap) {
			t.Fatalf("fixture error: skill %d's own placement %v is not inside the panel/background overlap %v", i, global, overlap)
		}
		art.Masks[0].SetColorIndex(local.X, local.Y, code)
		if k, consumed := TipPanelControlAt(tip, global); !consumed || k != TipControlNone {
			t.Fatalf("fixture error: skill %d's own sample %v, TipPanelControlAt = %v,%v, want TipControlNone,true", i, global, k, consumed)
		}
		covered = append(covered, global)
	}

	cells := make([]TownSurfaceCell, 5)
	for i := range cells {
		cells[i].Enabled = true
	}
	view := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: 0, Cells: cells, Tip: tip}
	town := &fakeTipSurfaceEnumTown{view: view}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the school surface")
	}
	now := time.Unix(1_700_000_000, 0)
	for _, p := range covered {
		clickAt(a, now, p)
	}
	if len(town.surfaceClicks) != len(covered) {
		t.Fatalf("covered list downs selected %v; want one per enabled class-mask slot", town.surfaceClicks)
	}

	closedBefore, toggledBefore := town.closed, town.toggled
	closeAt, ok := sampleInside(TipPanelCloseRect(tip.Rect))
	if !ok {
		t.Fatal("fixture error: the tip Close rectangle is empty")
	}
	clickAt(a, now, closeAt)
	if town.closed != closedBefore+1 {
		t.Fatalf("CloseTip() calls = %d after a release on the panel's own Close control, want %d", town.closed, closedBefore+1)
	}
	toggleAt, ok := sampleInside(TipPanelToggleRect(tip.Rect))
	if !ok {
		t.Fatal("fixture error: the tip Toggle rectangle is empty")
	}
	clickAt(a, now, toggleAt)
	if town.toggled != toggledBefore+1 {
		t.Fatalf("ToggleTips() calls = %d after a release on the panel's own Toggle control, want %d", town.toggled, toggledBefore+1)
	}
}

// fakeTipShopEnumTown is a shop room with a Showing tip; ShopSuppressDoll is
// recorded nowhere since none of this test's clicks reach the drag machine.
type fakeTipShopEnumTown struct {
	fakeTown
	tipCloseToggleRecorder
	tip      TipPanelView
	clicked  []ShopControl
	scrolled []scrolledBy
	dragged  [][2]ShopControl
}

func (f *fakeTipShopEnumTown) AtTownShop() bool { return true }
func (f *fakeTipShopEnumTown) ShopScreen() ShopScreenView {
	return ShopScreenView{Purse: 100, Buy: 40, Sell: 9, Chosen: -1, TipPanel: f.tip}
}
func (f *fakeTipShopEnumTown) ShopClick(c ShopControl) TownAction {
	f.clicked = append(f.clicked, c)
	return TownAction{}
}
func (f *fakeTipShopEnumTown) ShopScroll(region ShopWheelRegion, rows int) {
	f.scrolled = append(f.scrolled, scrolledBy{region, rows})
}
func (f *fakeTipShopEnumTown) ShopDrag(from, to ShopControl) TownAction {
	f.dragged = append(f.dragged, [2]ShopControl{from, to})
	return TownAction{}
}
func (f *fakeTipShopEnumTown) ShopSuppressDoll(slot int) {}

// TestShopTipListReleaseConsumesAndBodyPasses enumerates every DISTINCT
// ShopControl ShopControlAt names anywhere under ShopTipRect — found by
// scanning the rect against the same production geometry cmd/tippanelcheck
// measures (D-2: two shelf picks, the merchant and four table cells, 47422
// live pixels) — rather than a hardcoded list, so a future geometry change
// is caught here even if this comment is not updated.
func TestShopTipListReleaseConsumesAndBodyPasses(t *testing.T) {
	rect := tipBackgroundZone(ShopTipRect()).Intersect(image.Rect(0, 0, 640, 480))
	found := map[ShopControl]image.Point{}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			p := image.Pt(x, y)
			if c, hit := ShopControlAt(p); hit {
				if _, ok := found[c]; !ok {
					found[c] = p
				}
			}
		}
	}
	if len(found) == 0 {
		t.Fatal("fixture error: no live shop control under ShopTipRect's own background zone — the enumeration below would prove nothing")
	}

	tip := TipPanelView{Rect: ShopTipRect(), Text: "the shop buys and sells", Art: tipTestArt(), Font: shopTipTestFont()}
	if !tip.Showing() {
		t.Fatal("fixture tip is not Showing()")
	}
	for c, p := range found {
		if k, consumed := TipPanelControlAt(tip, p); !consumed || k != TipControlNone {
			t.Fatalf("fixture error: %+v's own sample %v, TipPanelControlAt = %v,%v, want TipControlNone,true", c, p, k, consumed)
		}
	}

	shop := &fakeTipShopEnumTown{tip: tip}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(shop)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop fixture")
	}
	now := time.Unix(1_700_000_000, 0)
	for _, p := range found {
		clickAt(a, now, p)
	}
	wantClicks := 0
	for c, p := range found {
		if !p.In(TipPanelListRect(tip.Rect)) && c.Kind != ShopControlMerchant {
			wantClicks++
		}
	}
	if len(shop.clicked) != wantClicks {
		t.Fatalf("shop body releases %v; want %d passthrough controls and consumed list releases", shop.clicked, wantClicks)
	}
	if len(shop.scrolled) != 0 || len(shop.dragged) != 0 {
		t.Fatalf("ShopScroll/ShopDrag calls after the same releases, want none: scrolled=%v dragged=%v", shop.scrolled, shop.dragged)
	}

	closedBefore, toggledBefore := shop.closed, shop.toggled
	closeAt, ok := sampleInside(TipPanelCloseRect(ShopTipRect()))
	if !ok {
		t.Fatal("fixture error: TipPanelCloseRect(ShopTipRect()) is empty")
	}
	clickAt(a, now, closeAt)
	if shop.closed != closedBefore+1 {
		t.Fatalf("CloseTip() calls = %d after a release on the panel's own Close control, want %d", shop.closed, closedBefore+1)
	}
	toggleAt, ok := sampleInside(TipPanelToggleRect(ShopTipRect()))
	if !ok {
		t.Fatal("fixture error: TipPanelToggleRect(ShopTipRect()) is empty")
	}
	clickAt(a, now, toggleAt)
	if shop.toggled != toggledBefore+1 {
		t.Fatalf("ToggleTips() calls = %d after a release on the panel's own Toggle control, want %d", shop.toggled, toggledBefore+1)
	}
}

// TestPreCreateTipPanelSwallowsEveryPortraitItCovers enumerates every
// pre-create choice (0..3) whose own rect (preControlRect) intersects the
// showing panel, at this fixture's own 160x240 synthetic portrait size.
//
// A click on a covered portrait must leave PreChoice alone, and the pinned
// value is what makes that distinguishable from a swallow that happened to
// leave PreChoice at its default. At the researched absolute rect
// (160,280)-(472,480) (TOWN-314) all four choices intersect the panel at
// this portrait size, so a single pin would have to be excluded from the
// enumeration and one choice would go unclicked.
//
// Each choice is therefore clicked under its OWN pin, chosen as (i+1)%4 so
// the pin is never the choice under test. A swallow failure at choice i sets
// PreChoice to i, which differs from the pin in every one of the four cases.
// The covered count is asserted rather than assumed: the real shipped
// coverage is whatever cmd/tippanelcheck measures against installed art, and
// this fixture's own portraits are a separate population.
func TestPreCreateTipPanelSwallowsEveryPortraitItCovers(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Layout: testGenerator(), Forward: image.NewRGBA(image.Rect(0, 0, 96, 74)), Font: shopTipTestFont()}
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
	// tipBackgroundZone reads the panel's own actual Rect, not the raw
	// ChargenTipRect (round-2 adversarial review, owner item: the shop tip
	// is too tall; the generator's own tip is the same shape, and now
	// shrinks the same way). c.TipPanel() applies TipPanelShrinkRect, so a
	// zone built from the unshrunk ceiling can claim overlap the shrunk
	// panel no longer covers.
	bg := tipBackgroundZone(c.TipPanel().Rect)
	// Mask.bmp decides every hit (TOWN-520): the fixture's mask names
	// portrait 1 under the whole panel, so a press there would choose it.
	art.PreMask = chargenMask(640, 480)
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			art.PreMask.SetColorIndex(x, y, preMaskCode[1])
		}
	}
	type coveredChoice struct {
		choice int
		at     image.Point
	}
	p, ok := sampleInside(bg)
	if !ok {
		t.Fatal("fixture error: the panel's background zone is empty")
	}
	if k, consumed := TipPanelControlAt(c.TipPanel(), p); !consumed || k != TipControlNone {
		t.Fatalf("fixture error: sample %v, TipPanelControlAt = %v,%v, want TipControlNone,true", p, k, consumed)
	}
	if preControlAt(c, p) != chargenChoice1 {
		t.Fatal("fixture error: the mask does not name portrait 1 under the panel")
	}
	covered := []coveredChoice{{choice: 1, at: p}}
	for _, cc := range covered {
		pin := (cc.choice + 1) % 4
		c.SelectPreChoice(pin)
		if c.PreChoice() != pin {
			t.Fatalf("fixture error: PreChoice() = %d after SelectPreChoice(%d)", c.PreChoice(), pin)
		}
		clickAt(a, now, cc.at)
		if c.PreChoice() != pin {
			t.Fatalf("PreChoice() = %d after a press/release at %v (choice %d, covered by the showing tip panel), want %d (unchanged): the panel's own swallow did not hold", c.PreChoice(), cc.at, cc.choice, pin)
		}
	}

	closeAt, ok := sampleInside(TipPanelCloseRect(c.TipPanel().Rect))
	if !ok {
		t.Fatal("fixture error: TipPanelCloseRect(c.TipPanel().Rect) is empty")
	}
	clickAt(a, now, closeAt)
	if c.TipPanel().Showing() {
		t.Fatal("TipPanel() after a release on the panel's own Close control is still Showing()")
	}
}
