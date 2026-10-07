package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"
	"time"
)

func TestItemStarFieldAndThirteenPixelKernelAreDecoded(t *testing.T) {
	// These coordinates were evaluated independently from the published CRT
	// recurrence starting at this build's disclosed normalised state. They do
	// not call itemStarRand or makeItemStarField to construct the expectation.
	want := []image.Point{{8, 23}, {49, 12}, {25, 31}, {24, 71}, {28, 67}, {19, 62}, {10, 8}}
	for i, p := range want {
		if got := itemStarFields[0].points[i]; got != p {
			t.Fatalf("point %d = %v, want %v", i, got, p)
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			dst.SetRGBA(x, y, color.RGBA{A: 0xff})
		}
	}
	drawItemStarTrail(dst, image.Point{}, 0, &itemStarFields[0])
	checks := map[image.Point]color.RGBA{
		{8, 23}:  {R: 63, B: 63, A: 0xff},
		{9, 23}:  {R: 47, B: 47, A: 0xff},
		{10, 23}: {R: 31, B: 31, A: 0xff},
		{9, 24}:  {R: 15, B: 15, A: 0xff},
		{11, 23}: {A: 0xff},
	}
	for p, want := range checks {
		if got := dst.RGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v = %+v, want %+v", p, got, want)
		}
	}
}

func TestItemStarTrailIntroducesOneCentreEveryTwoPhasesAndKeepsSeven(t *testing.T) {
	frame := func(phase uint32) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				img.SetRGBA(x, y, color.RGBA{A: 0xff})
			}
		}
		drawItemStarTrail(img, image.Point{}, phase, &itemStarFields[0])
		return img
	}

	zero, one, two := frame(0), frame(1), frame(2)
	if zero.RGBAAt(8, 23) != one.RGBAAt(8, 23) {
		t.Fatal("odd phase introduced a new centre; coordinates must advance every two increments")
	}
	if got := two.RGBAAt(49, 12); got != (color.RGBA{R: 63, B: 63, A: 0xff}) {
		t.Fatalf("phase 2 new centre = %+v, want alpha 63 magenta", got)
	}
	if got := frame(6).RGBAAt(8, 23); got != (color.RGBA{R: 255, B: 255, A: 0xff}) {
		t.Fatalf("the fourth age did not reach full RGB(255,0,255): %+v", got)
	}
	if got := frame(12).RGBAAt(10, 8); got != (color.RGBA{R: 63, B: 63, A: 0xff}) {
		t.Fatalf("the seventh introduced centre = %+v, want alpha 63 magenta", got)
	}
	if got := frame(14).RGBAAt(8, 23); got != (color.RGBA{A: 0xff}) {
		t.Fatalf("an eighth centre left the oldest point alive: %+v", got)
	}
}

func TestItemStarScrollMovesVisibleSlotPhasesAsymmetrically(t *testing.T) {
	down := []uint32{11, 22, 33, 44}
	shiftItemStarPhases(down, 1)
	wantDown := []uint32{22, 33, 44, 0}
	for i := range down {
		if down[i] != wantDown[i] {
			t.Fatalf("scroll down phases = %v, want %v", down, wantDown)
		}
	}

	up := []uint32{11, 22, 33, 44}
	shiftItemStarPhases(up, -1)
	wantUp := []uint32{11, 11, 22, 33}
	for i := range up {
		if up[i] != wantUp[i] {
			t.Fatalf("scroll up phases = %v, want %v", up, wantUp)
		}
	}
}

func TestMissionPaintHoldsPhaseAndDrawsTrailAfterQuantity(t *testing.T) {
	bar, cols, cells := packBarFixture(t)
	icon := solidPic(80, 80, color.RGBA{R: 7, G: 11, B: 13, A: 0xff})
	baseSubject := InventorySubject{Pack: []*image.RGBA{icon}, PackCount: []uint32{1}, PackStars: []bool{false}}
	countSubject := baseSubject
	countSubject.PackCount = []uint32{222}
	starSubject := baseSubject
	starSubject.PackStars = []bool{true}
	bothSubject := countSubject
	bothSubject.PackStars = []bool{true}
	font := numeralFont()

	base := renderPackBarStars(baseSubject, 0, cols, bar, font, make([]uint32, cols), -1)
	count := renderPackBarStars(countSubject, 0, cols, bar, font, make([]uint32, cols), -1)
	origin := cells[0].Min
	phase, target, alpha := phaseWhoseAgeThreeKernelHitsInk(t, base, count, origin, &itemStarFields[0], shopPriceInk)
	phases := make([]uint32, cols)
	phases[0] = phase
	star := renderPackBarStars(starSubject, 0, cols, bar, font, phases, -1)
	both := renderPackBarStars(bothSubject, 0, cols, bar, font, phases, -1)
	want := starBlendExpected(count.RGBAAt(target.X, target.Y), alpha)
	if got := both.RGBAAt(target.X, target.Y); got != want {
		t.Fatalf("mission overlap at %v = %+v, want trail over quantity %+v (star-only %+v)", target, got, want, star.RGBAAt(target.X, target.Y))
	}
	again := renderPackBarStars(starSubject, 0, cols, bar, font, phases, -1)
	if !bytes.Equal(star.Pix, again.Pix) {
		t.Fatal("a mission repaint advanced the trail; only the world-tick clock may advance it")
	}
}

func TestViewerMissionStarClockMovesOnlyOnTheTickSeam(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	v.sel = selection{5}
	v.SetInventorySubject(InventorySubject{
		ID:        5,
		Pack:      []*image.RGBA{solidPic(80, 80, color.RGBA{R: 7, G: 11, B: 13, A: 0xff})},
		PackCount: []uint32{1},
		PackStars: []bool{true},
	})

	// Paint initialises the visible-slot array but is not its clock.
	if _, _, ok := v.packBarPresent(); !ok {
		t.Fatal("setup: eligible inventory has no pack bar")
	}
	if got := append([]uint32(nil), v.packStarPhases...); len(got) == 0 || got[0] != 0 {
		t.Fatalf("initial phases = %v, want visible slot zero at phase 0", got)
	}
	if _, _, ok := v.packBarPresent(); !ok || v.packStarPhases[0] != 0 {
		t.Fatalf("second paint moved phase to %v; a repaint is not mission message 0x401", v.packStarPhases)
	}

	v.AdvanceInventoryStars()
	if got := v.packStarPhases[0]; got != 1 {
		t.Fatalf("one actual tick-seam advance = %d, want 1", got)
	}
	if _, _, ok := v.packBarPresent(); !ok || v.packStarPhases[0] != 1 {
		t.Fatalf("paint after one tick moved phase to %v, want it held at 1", v.packStarPhases)
	}
	v.AdvanceInventoryStars()
	if got := v.packStarPhases[0]; got != 2 {
		t.Fatalf("second actual tick-seam advance = %d, want 2", got)
	}
}

func TestShopPaintDrawsQuantityAfterTrailAndAdvancesVisibleSlots(t *testing.T) {
	icon := solidPic(80, 80, color.RGBA{R: 7, G: 11, B: 13, A: 0xff})
	cell := ShopCell{Icon: icon, Count: 1, Back: ShopBackItem}
	baseView := ShopScreenView{Font: numeralFont()}
	baseView.Table[0] = cell
	countView := baseView
	countView.Table[0].Count = 222
	base := ComposeShopScreen(baseView, image.Point{}, false, nil, false)
	count := ComposeShopScreen(countView, image.Point{}, false, nil, false)
	origin := image.Pt(ShopTableCellRect(0).Min.X, ShopTableCellRect(0).Min.Y+1)
	phase, target, _ := phaseWhoseAgeThreeKernelHitsInk(t, base, count, origin, &itemStarFields[2], shopPriceInk)
	starView := baseView
	starView.Table[0].Star, starView.Table[0].StarPhase = true, phase
	bothView := countView
	bothView.Table[0].Star, bothView.Table[0].StarPhase = true, phase
	star := ComposeShopScreen(starView, image.Point{}, false, nil, false)
	both := ComposeShopScreen(bothView, image.Point{}, false, nil, false)
	if got := both.RGBAAt(target.X, target.Y); got != count.RGBAAt(target.X, target.Y) || got == star.RGBAAt(target.X, target.Y) {
		t.Fatalf("shop overlap at %v = %+v, count-only %+v star-only %+v; quantity must paint after trail", target, got, count.RGBAAt(target.X, target.Y), star.RGBAAt(target.X, target.Y))
	}

	var state shopStarState
	view := ShopScreenView{Chosen: 1, Member: 2}
	view.Shelf[0] = ShopCell{Icon: icon, Star: true, Count: 1}
	view.Shelf[1] = ShopCell{Icon: icon, Star: false, Count: 1}
	state.prepare(&view, ShopControl{}, false)
	state.advance(&view)
	next := ShopScreenView{Chosen: 1, Member: 2}
	next.Shelf[0] = ShopCell{Icon: icon, Star: true, Count: 1}
	next.Shelf[1] = ShopCell{Icon: icon, Star: false, Count: 1}
	state.prepare(&next, ShopControl{}, false)
	if next.Shelf[0].StarPhase != 1 || next.Shelf[1].StarPhase != 0 {
		t.Fatalf("paint phases = enchanted %d plain %d, want 1/0", next.Shelf[0].StarPhase, next.Shelf[1].StarPhase)
	}

	selected := next
	state.prepare(&selected, ShopControl{Kind: ShopControlShelfCell, Index: 0}, true)
	if selected.Shelf[0].Icon != nil || selected.Shelf[0].Star {
		t.Fatal("selected singleton remained in the shop grid")
	}
	remaining := next
	remaining.Shelf[0].Count = 2
	state.prepare(&remaining, ShopControl{Kind: ShopControlShelfCell, Index: 0}, true)
	if remaining.Shelf[0].Icon == nil || !remaining.Shelf[0].Star {
		t.Fatal("a selected stack with a remaining unit lost its grid icon or trail")
	}
}

type itemStarShopTown struct {
	fakeShopTown
	icon *image.RGBA
}

func (f *itemStarShopTown) ShopScreen() ShopScreenView {
	v := f.fakeShopTown.ShopScreen()
	v.Table[0] = ShopCell{Icon: f.icon, Count: 1, Back: ShopBackItem, Star: true}
	return v
}

func TestAppShopCompositionAdvancesBeforeEachPaint(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	town := &itemStarShopTown{icon: solidPic(80, 80, color.RGBA{R: 7, G: 11, B: 13, A: 0xff})}
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("flow.showTown refused the star-shop fixture")
	}

	first, err := a.composeTownRoom() // advances and paints q=1
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.composeTownRoom() // advances and paints q=2
	if err != nil {
		t.Fatal(err)
	}
	// Independently evaluated table-field point 1 is (31,38); the decoded
	// table origin is (32,304), so q=2 first reaches this exact pixel.
	target := image.Pt(63, 342)
	if got := first.RGBAAt(target.X, target.Y); got != (color.RGBA{R: 7, G: 11, B: 13, A: 0xff}) {
		t.Fatalf("first paint pixel %v = %+v, want untouched base before point 1 is introduced", target, got)
	}
	if got := second.RGBAAt(target.X, target.Y); got != (color.RGBA{R: 68, G: 8, B: 72, A: 0xff}) {
		t.Fatalf("second paint pixel %v = %+v, want independently blended q=2 point", target, got)
	}
	if got := a.shopStars.table[0]; got != 2 {
		t.Fatalf("two successful App paints stored phase %d, want 2", got)
	}
}

type scrollingItemStarShopTown struct {
	fakeShopTown
	shelfOffset, packOffset int
	shelf, pack             []ShopCell
	absoluteOrigins         []ShopControl
}

func (f *scrollingItemStarShopTown) ShopScreen() ShopScreenView {
	v := f.fakeShopTown.ShopScreen()
	v.Chosen, v.Member = 0, 0
	v.ShelfOffset, v.PackOffset = f.shelfOffset, f.packOffset
	for i := range v.Shelf {
		if k := f.shelfOffset + i; k >= 0 && k < len(f.shelf) {
			v.Shelf[i] = f.shelf[k]
		}
	}
	for i := range v.Pack {
		if k := f.packOffset + i; k >= 0 && k < len(f.pack) {
			v.Pack[i] = f.pack[k]
		}
	}
	return v
}

func (f *scrollingItemStarShopTown) ShopScroll(region ShopWheelRegion, rows int) {
	f.fakeShopTown.ShopScroll(region, rows)
	switch region {
	case ShopWheelShelf:
		f.shelfOffset += rows * shopShelfCols
	case ShopWheelPack:
		f.packOffset += rows
	}
}

func (f *scrollingItemStarShopTown) ShopDrag(from, to ShopControl) TownAction {
	abs := from
	switch abs.Kind {
	case ShopControlShelfCell:
		abs.Index += f.shelfOffset
	case ShopControlPackCell:
		abs.Index += f.packOffset
	}
	f.absoluteOrigins = append(f.absoluteOrigins, abs)
	return f.fakeShopTown.ShopDrag(from, to)
}

func TestShopWheelDuringDragKeepsSourceAliasAndOmitsOnlyItsAbsoluteRecord(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tablePoint := ShopTableCellRect(0).Min.Add(image.Pt(4, 4))

	t.Run("shelf source scrolls offscreen", func(t *testing.T) {
		original := solidPic(80, 80, color.RGBA{R: 11, G: 13, B: 17, A: 0xff})
		replacement := solidPic(80, 80, color.RGBA{R: 19, G: 23, B: 29, A: 0xff})
		town := &scrollingItemStarShopTown{shelf: make([]ShopCell, shopShelfN+2)}
		town.shelf[0] = ShopCell{Icon: original, Count: 1, Star: true, Back: ShopBackItem}
		town.shelf[2] = ShopCell{Icon: replacement, Count: 1, Star: true, Back: ShopBackItem}
		a := newTestApp(t, appRows(1), nil)
		a.SetTown(town)
		if !a.flow.showTown("") {
			t.Fatal("showTown refused the scrolling shelf fixture")
		}
		press := ShopShelfCellRect(0).Min.Add(image.Pt(4, 4))

		a.step(appInput{CursorX: press.X, CursorY: press.Y, PrimaryPressed: true}, now)
		// Scroll before crossing TapSlop: a late icon lookup would now read
		// the replacement at the same visible cell.
		a.step(appInput{CursorX: press.X, CursorY: press.Y, WheelY: -1}, now)
		a.step(appInput{CursorX: tablePoint.X, CursorY: tablePoint.Y}, now)
		if got, ok := a.shopDragItemPresent(); !ok || got != original {
			t.Fatalf("held alias after shelf scroll = (%p,%v), want original %p", got, ok, original)
		}

		view := town.ShopScreen()
		origin := shopDragOriginAt(view, a.shopDragOrigin, a.shopDragOriginBase)
		var stars shopStarState
		stars.prepare(&view, origin, true)
		if origin.Index != -2 || view.Shelf[0].Icon != replacement {
			t.Fatalf("scrolled shelf origin=%+v replacement=%p, want offscreen -2 and visible %p", origin, view.Shelf[0].Icon, replacement)
		}

		a.step(appInput{CursorX: tablePoint.X, CursorY: tablePoint.Y, PrimaryReleased: true}, now)
		if len(town.absoluteOrigins) != 1 || town.absoluteOrigins[0] != (ShopControl{Kind: ShopControlShelfCell, Index: 0}) {
			t.Fatalf("absolute shelf origins = %+v, want source record 0", town.absoluteOrigins)
		}
	})

	t.Run("pack source shifts while stale cell becomes another item", func(t *testing.T) {
		original := solidPic(80, 80, color.RGBA{R: 31, G: 37, B: 41, A: 0xff})
		replacement := solidPic(80, 80, color.RGBA{R: 43, G: 47, B: 53, A: 0xff})
		town := &scrollingItemStarShopTown{pack: make([]ShopCell, shopStripN+1)}
		town.pack[0] = ShopCell{Money: true, Count: 100}
		town.pack[1] = ShopCell{Icon: original, Count: 1, Star: true, Back: ShopBackItem}
		town.pack[2] = ShopCell{Icon: replacement, Count: 1, Star: true, Back: ShopBackItem}
		a := newTestApp(t, appRows(1), nil)
		a.SetTown(town)
		if !a.flow.showTown("") {
			t.Fatal("showTown refused the scrolling pack fixture")
		}
		press := ShopPackCellRect(1).Min.Add(image.Pt(4, 4))

		a.step(appInput{CursorX: press.X, CursorY: press.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: press.X, CursorY: press.Y, WheelY: -1}, now)
		a.step(appInput{CursorX: tablePoint.X, CursorY: tablePoint.Y}, now)
		if got, ok := a.shopDragItemPresent(); !ok || got != original {
			t.Fatalf("held alias after pack scroll = (%p,%v), want original %p", got, ok, original)
		}

		view := town.ShopScreen()
		origin := shopDragOriginAt(view, a.shopDragOrigin, a.shopDragOriginBase)
		var stars shopStarState
		stars.prepare(&view, origin, true)
		if origin.Index != 0 || view.Pack[0].Icon != nil || view.Pack[1].Icon != replacement {
			t.Fatalf("scrolled pack origin=%+v cells=(%p,%p), want original omitted at 0 and replacement %p at stale index 1", origin, view.Pack[0].Icon, view.Pack[1].Icon, replacement)
		}

		a.step(appInput{CursorX: tablePoint.X, CursorY: tablePoint.Y, PrimaryReleased: true}, now)
		if len(town.absoluteOrigins) != 1 || town.absoluteOrigins[0] != (ShopControl{Kind: ShopControlPackCell, Index: 1}) {
			t.Fatalf("absolute pack origins = %+v, want source record 1", town.absoluteOrigins)
		}
	})
}

func phaseWhoseAgeThreeKernelHitsInk(t *testing.T, base, ink *image.RGBA, origin image.Point, field *itemStarField, wantInk color.RGBA) (uint32, image.Point, uint8) {
	t.Helper()
	for y := base.Bounds().Min.Y; y < base.Bounds().Max.Y; y++ {
		for x := base.Bounds().Min.X; x < base.Bounds().Max.X; x++ {
			p := image.Pt(x, y)
			if ink.RGBAAt(x, y) != wantInk || ink.RGBAAt(x, y) == base.RGBAAt(x, y) {
				continue
			}
			rel := p.Sub(origin)
			for i := 3; i < len(field.points)-3; i++ {
				alpha := ageThreeKernelAlpha(rel.Sub(field.points[i]))
				if alpha == 0 {
					continue
				}
				// Keep the expected blend independent: reject a point touched
				// by another one of this phase's six centres.
				unique := true
				for age := 0; age < len(itemStarAlpha); age++ {
					if age == 3 {
						continue
					}
					centre := field.points[i+3-age]
					if itemStarKernelAlpha(rel.Sub(centre), itemStarAlpha[age]) != 0 {
						unique = false
						break
					}
				}
				if unique {
					return uint32(2 * (i + 3)), p, alpha
				}
			}
		}
	}
	t.Fatal("fixture has no quantity-ink pixel reached by a decoded star centre")
	return 0, image.Point{}, 0
}

func ageThreeKernelAlpha(d image.Point) uint8 {
	return itemStarKernelAlpha(d, 0xff)
}

func itemStarKernelAlpha(d image.Point, centre uint8) uint8 {
	switch {
	case d.X == 0 && d.Y == 0:
		return centre
	case (d.X == 0 && (d.Y == 1 || d.Y == -1)) || (d.Y == 0 && (d.X == 1 || d.X == -1)):
		return uint8(uint16(centre) * 3 / 4)
	case (d.X == 0 && (d.Y == 2 || d.Y == -2)) || (d.Y == 0 && (d.X == 2 || d.X == -2)):
		return centre / 2
	case (d.X == 1 || d.X == -1) && (d.Y == 1 || d.Y == -1):
		return centre / 4
	default:
		return 0
	}
}

func starBlendExpected(dst color.RGBA, alpha uint8) color.RGBA {
	a, inv := uint32(alpha), uint32(0xff-alpha)
	return color.RGBA{
		R: uint8(a + uint32(dst.R)*inv/0xff),
		G: uint8(uint32(dst.G) * inv / 0xff),
		B: uint8(a + uint32(dst.B)*inv/0xff),
		A: uint8(a + uint32(dst.A)*inv/0xff),
	}
}

func TestHeldSingletonOmissionKeepsARemainingMissionStack(t *testing.T) {
	v := &Viewer{dragActive: true, dragCandKind: dragFromPack, dragCandIdx: 0}
	v.invSubject.PackCount = []uint32{1}
	if got := v.packStarOmit(); got != 0 {
		t.Fatalf("single selected unit omit = %d, want 0", got)
	}
	v.invSubject.PackCount[0] = 2
	if got := v.packStarOmit(); got != -1 {
		t.Fatalf("remaining stack omit = %d, want -1", got)
	}
}
