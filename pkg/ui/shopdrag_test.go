package ui

import (
	"image"
	"reflect"
	"testing"
	"time"
)

// The App-level drag machine over the shop screen (1005 round 2:
// stepTown's own "THE DRAG MACHINE'S OWN ARM" comments, app.go). Every test
// here drives a.step the way ebiten's own input loop does — one appInput per
// frame — rather than calling the drag fields directly, so what is proven is
// the WIRING: a press over the doll is held, not acted on; a release that
// never crossed TapSlop is a tap; a release that did is judged by where it
// lands; a landing nowhere recognised moves nothing, exactly as a mission's
// own ground drop names ITEM-DROP-008 and a town screen has no ground for it
// (DIV-088).

// TestShopDragOriginIconReadsEachSurfacesOwnPicture is shopDragOriginIcon's
// own witness (round-2 adversarial review, item 2): each of the four drag
// surfaces answers ITS OWN field of v, and a kind naming none of them — or
// an index outside the one it does name — answers nil rather than
// panicking or reading another surface's slice.
func TestShopDragOriginIconReadsEachSurfacesOwnPicture(t *testing.T) {
	doll := image.NewRGBA(image.Rect(0, 0, 1, 1))
	shelf := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pack := image.NewRGBA(image.Rect(0, 0, 3, 3))
	table := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var v ShopScreenView
	v.SlotIcon[2] = doll
	v.Shelf[1].Icon = shelf
	v.Pack[0].Icon = pack
	v.Table[3].Icon = table

	for _, tc := range []struct {
		name   string
		origin ShopControl
		want   *image.RGBA
	}{
		{"doll", ShopControl{Kind: ShopControlDoll, Index: 2}, doll},
		{"shelf", ShopControl{Kind: ShopControlShelfCell, Index: 1}, shelf},
		{"pack", ShopControl{Kind: ShopControlPackCell, Index: 0}, pack},
		{"table", ShopControl{Kind: ShopControlTableCell, Index: 3}, table},
		{"doll, wrong index", ShopControl{Kind: ShopControlDoll, Index: 0}, nil},
		{"doll, index out of range", ShopControl{Kind: ShopControlDoll, Index: 99}, nil},
		{"a kind naming no surface", ShopControl{Kind: ShopControlMerchant}, nil},
	} {
		if got := shopDragOriginIcon(v, tc.origin); got != tc.want {
			t.Errorf("%s: shopDragOriginIcon(%+v) = %v, want %v", tc.name, tc.origin, got, tc.want)
		}
	}
}

// shopDragTestApp is one App over a fakeShopTown carrying shopTestMask
// (shopscreen_test.go), so the doll's marked pixel at figure-local (50,100) —
// slot 4, one-based — is a real press target.
func shopDragTestApp(t *testing.T) (*App, *fakeShopTown) {
	t.Helper()
	town := &fakeShopTown{mask: shopTestMask()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop town")
	}
	return a, town
}

var (
	shopDragDollPoint  = shopFigureRect.Min.Add(image.Pt(50, 100)) // the mask's own marked pixel, slot 4 (1-based)
	shopDragShelfPoint = ShopShelfCellRect(0).Min.Add(image.Pt(4, 4))
	shopDragMissPoint  = image.Pt(0, 0) // ShopControlAt's own miss, confirmed by the 5px sweep this test's comment describes
)

func shopFullDollDragTestApp(t *testing.T, mask *SlotMask) (*App, *fakeShopTown) {
	t.Helper()
	town := &fakeShopTown{mask: mask, character: shopFullDollTestView(mask).Character}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop town")
	}
	return a, town
}

// TestTheApplicationDragsFromTheDollToAShelfCell is the crossed-TapSlop
// path: a press on the doll, a move past TapSlop toward the shelf grid, and
// a release there sends exactly one ShopDrag(origin, dest) across the seam,
// origin the doll slot the mask named and dest the shelf cell the release
// landed on — never a click, which is the OTHER gesture this same press
// could have been.
func TestTheApplicationDragsFromTheDollToAShelfCell(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	// A move-only frame past TapSlop, cursor already over the destination:
	// stepTown's own per-frame arm latches shopDragMoved and pushes the
	// suppression before any press or release edge is seen this frame.
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y}, now)
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryReleased: true}, now)

	want := [][2]ShopControl{{
		{Kind: ShopControlDoll, Index: 3},
		{Kind: ShopControlShelfCell, Index: 0},
	}}
	if !reflect.DeepEqual(town.dragged, want) {
		t.Fatalf("dragged = %+v, want %+v", town.dragged, want)
	}
	if len(town.clicked) != 0 {
		t.Errorf("clicked = %v, want none: a crossed-TapSlop release is a drag, not a click", town.clicked)
	}
	// The origin slot (one-based: 4) was pushed as the live suppression on
	// at least one frame of the gesture, including the release frame
	// itself (stepTown's own doc on why the per-frame arm runs first).
	sawSlot := false
	for _, s := range town.suppressed {
		if s == 4 {
			sawSlot = true
		}
	}
	if !sawSlot {
		t.Errorf("suppressed = %v, want slot 4 pushed at least once during the drag", town.suppressed)
	}
}

// TestTheApplicationDragsFromPackToAnUnwornDollArea is the doll's own
// fallback (round-2 adversarial review, tenth pass): a release inside
// shopDollAreaAt that names no marked slot at all — shopTestMask marks
// only figure-local (50,100), slot 4 — still resolves as the doll, since
// ShopDrag's own doll-destination cases (pkg/game/shopview.go) match on Kind
// alone and never consult Index. Before this fix, shopGridControlAt answered
// !ok for such a release and the drag was silently dropped: a shop character
// wearing nothing anywhere, or nothing in the slot a release happened to
// land on, could never be dressed by a drag at all, since no pixel in
// shopDollAreaAt would ever answer for the doll again.
func TestTheApplicationDragsFromPackToAnUnwornDollArea(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)
	packPoint := ShopPackCellRect(1).Min.Add(image.Pt(4, 4))
	bareDollPoint := shopFigureRect.Min.Add(image.Pt(50, 50)) // shopTestMask's own unmarked ground

	a.step(appInput{CursorX: packPoint.X, CursorY: packPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: bareDollPoint.X, CursorY: bareDollPoint.Y}, now)
	a.step(appInput{CursorX: bareDollPoint.X, CursorY: bareDollPoint.Y, PrimaryReleased: true}, now)

	want := [][2]ShopControl{{
		{Kind: ShopControlPackCell, Index: 1},
		{Kind: ShopControlDoll},
	}}
	if !reflect.DeepEqual(town.dragged, want) {
		t.Fatalf("dragged = %+v, want %+v", town.dragged, want)
	}
}

// TestTheApplicationTapsTheDollForTheOneMutationDoor is the un-crossed path:
// a press and release at the same doll point, never reaching TapSlop, is
// round 1's own tap-to-unequip restated for the shop — ShopClick's
// ShopControlDoll arm, the same one mutation door a genuine click uses.
func TestTheApplicationTapsTheDollForTheOneMutationDoor(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryReleased: true}, now)

	if !reflect.DeepEqual(town.clicked, []int{3}) {
		t.Fatalf("clicked = %v, want [3] (the doll's own zero-based slot)", town.clicked)
	}
	if len(town.dragged) != 0 {
		t.Errorf("dragged = %+v, want none: a tap is not a drag", town.dragged)
	}
}

// TestTheApplicationBookControlWinsOverAnOccupiedDollPixel reproduces the
// 1006 landing defect through App.step, restated for rect B's own decoded hit
// rectangle since the authored Book plaque this test used to probe is deleted
// (owner finding 3, round-2 adversarial return). Rect B is painted over the
// figure, so an occupied SlotMask pixel beneath it must not arm the doll drag
// or unequip that slot on release.
func TestTheApplicationBookControlWinsOverAnOccupiedDollPixel(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)
	p := TownCharacterRegion.Min.Add(image.Pt(12, 12))
	q := p.Sub(shopFigureRect.Min)
	town.mask.Slot[q.Y*town.mask.W+q.X] = 7

	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
	if a.shopDragArmed {
		t.Fatal("Book press armed the occupied doll pixel underneath the painted control")
	}
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)

	want := []ShopControl{{Kind: ShopControlBook}}
	if !reflect.DeepEqual(town.controls, want) {
		t.Fatalf("ShopClick controls = %+v, want Book once", town.controls)
	}
	if len(town.dragged) != 0 {
		t.Fatalf("ShopDrag calls = %+v, want none", town.dragged)
	}
}

// TestTheApplicationUsesFormerDollNameRowsForEveryDollRoute exercises the
// four consumers of the shared hit answer. Both a pixel formerly covered by
// the name glyph and its neighbouring gap now expose the underlying slot.
func TestTheApplicationUsesFormerDollNameRowsForEveryDollRoute(t *testing.T) {
	const slot = 3
	glyph, gap := image.Pt(559, 424), image.Pt(558, 423)
	points := []struct {
		name string
		at   image.Point
	}{{"former glyph", glyph}, {"former gap", gap}}
	full := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	for i := range full.Slot {
		full.Slot[i] = slot + 1
	}

	t.Run("hover", func(t *testing.T) {
		v := shopFullDollTestView(full)
		v.SlotInfo[slot] = []string{"worn item"}
		for _, tc := range points {
			if lines, ok := ShopHoverLines(v, tc.at); !ok || !reflect.DeepEqual(lines, []string{"worn item"}) {
				t.Fatalf("%s hover = %v,%v, want worn item", tc.name, lines, ok)
			}
		}
	})

	t.Run("tap and drag origin", func(t *testing.T) {
		for _, tc := range points {
			t.Run(tc.name, func(t *testing.T) {
				now := time.Unix(1_700_000_000, 0)
				a, town := shopFullDollDragTestApp(t, full)
				a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryPressed: true}, now)
				a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryReleased: true}, now)
				if !reflect.DeepEqual(town.clicked, []int{slot}) {
					t.Fatalf("tap = %v, want [%d]", town.clicked, slot)
				}

				a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryPressed: true}, now)
				a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y}, now)
				a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryReleased: true}, now)
				want := [][2]ShopControl{{
					{Kind: ShopControlDoll, Index: slot},
					{Kind: ShopControlShelfCell, Index: 0},
				}}
				if !reflect.DeepEqual(town.dragged, want) {
					t.Fatalf("drag = %+v, want %+v", town.dragged, want)
				}
			})
		}
	})

	t.Run("unworn destination", func(t *testing.T) {
		now := time.Unix(1_700_000_000, 0)
		for _, tc := range points {
			t.Run(tc.name, func(t *testing.T) {
				a, town := shopFullDollDragTestApp(t, shopTestMask())
				a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryPressed: true}, now)
				a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y}, now)
				a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryReleased: true}, now)
				got := len(town.dragged) == 1 && town.dragged[0][1].Kind == ShopControlDoll
				if !got {
					t.Fatalf("unworn destination at %v routed=%v, want true; calls=%+v", tc.at, got, town.dragged)
				}
			})
		}
	})
}

// TestHeadlessShopPointUsesVisibleDollPixels gives slot 4 exactly two raw mask
// pixels in the former name row. Both are now visible and the deterministic
// midpoint selection chooses the later point.
func TestHeadlessShopPointUsesVisibleDollPixels(t *testing.T) {
	gap, glyph := image.Pt(558, 423), image.Pt(559, 424)
	mask := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	for _, p := range []image.Point{gap, glyph} {
		q := p.Sub(shopFigureRect.Min)
		mask.Slot[q.Y*mask.W+q.X] = 4
	}
	a, _ := shopFullDollDragTestApp(t, mask)
	x, y, err := a.HeadlessShopPoint("doll", 4)
	if err != nil {
		t.Fatal(err)
	}
	if got := image.Pt(x, y); got != glyph {
		t.Fatalf("HeadlessShopPoint(doll,4) = %v, want midpoint %v", got, glyph)
	}
}

func TestShopStatisticsLeftControlsStayLiveAtAppBoundary(t *testing.T) {
	a, town := shopDragTestApp(t)
	town.stats = true
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryReleased: true}, now)
	if len(town.controls) != 1 || town.controls[0].Kind != ShopControlShelfCell || town.controls[0].Index != 0 {
		t.Fatalf("shelf click during statistics = %+v, want shelf cell 0", town.controls)
	}
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, WheelY: 1}, now)
	if len(town.scrolled) != 1 || town.scrolled[0].region != ShopWheelShelf {
		t.Fatalf("wheel over the shelf during statistics = %+v, want one shelf scroll", town.scrolled)
	}

	town.controls = nil
	p := shopButtonRects[0].Min.Add(image.Pt(4, 4))
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	if len(town.controls) != 1 || town.controls[0].Kind != ShopControlButton || town.controls[0].Index != 0 {
		t.Fatalf("persistent upper button controls = %+v, want button 0", town.controls)
	}

	town.controls = nil
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryReleased: true}, now)
	for _, c := range town.controls {
		if c.Kind == ShopControlDoll {
			t.Fatalf("doll answered %+v during statistics, want a miss (the card replaces it)", c)
		}
	}
}

// TestTheApplicationATremorWithinOneShelfCellIsATapNotADrag is
// counterexample 4 (round-2 adversarial review, second pass): TapSlop is 4
// Manhattan pixels, high-water, and a shelf cell (ShopShelfCellRect) is 80
// pixels square — a hand's own tremor crosses TapSlop while the cursor
// never leaves the cell it pressed. ShopDrag's own switch
// (pkg/game/shopview.go) has no ShelfCell -> ShelfCell case, because
// dragging a cell onto its own family is not a shop action at all; before
// this fix the release reached ShopDrag(shelf 0, shelf 0), unrecognised, and
// the click the player meant was silently dropped. dest.Kind == origin.Kind
// routes it to the same ShopClick door a release under TapSlop already
// opens.
func TestTheApplicationATremorWithinOneShelfCellIsATapNotADrag(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)
	tremor := shopDragShelfPoint.Add(image.Pt(TapSlop, 0)) // still inside shelf cell 0

	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: tremor.X, CursorY: tremor.Y}, now)
	a.step(appInput{CursorX: tremor.X, CursorY: tremor.Y, PrimaryReleased: true}, now)

	if !reflect.DeepEqual(town.clicked, []int{0}) {
		t.Fatalf("clicked = %v, want [0] (shelf cell 0, the one both press and release stood on)", town.clicked)
	}
	if len(town.dragged) != 0 {
		t.Errorf("dragged = %+v, want none: a same-family release is a tap, not a drag", town.dragged)
	}
}

// TestTheApplicationADollDragReleasedBackOnTheDollCancels is the owner's
// correction to the preview gesture. Once a doll press crosses TapSlop it is
// a drag, even if it later returns almost to the press point. Releasing that
// held item back on the doll cancels the move; it must not reuse the bare-tap
// ShopClick door and unequip it. The separate un-crossed test keeps the
// intentional tap-to-unequip gesture.
func TestTheApplicationADollDragReleasedBackOnTheDollCancels(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)
	away := shopDragDollPoint.Add(image.Pt(4*TapSlop, 0)) // off the mask, well past TapSlop
	near := shopDragDollPoint.Add(image.Pt(1, 0))
	q := near.Sub(shopFigureRect.Min)
	town.mask.Slot[q.Y*town.mask.W+q.X] = 4 // same worn layer, not the exact press pixel

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: away.X, CursorY: away.Y}, now)
	a.step(appInput{CursorX: near.X, CursorY: near.Y, PrimaryReleased: true}, now)

	if len(town.clicked) != 0 {
		t.Fatalf("clicked = %v, want none: returning a held item to the doll cancels", town.clicked)
	}
	if len(town.dragged) != 0 {
		t.Errorf("dragged = %+v, want none: a release back on the doll cancels", town.dragged)
	}

	// Cancellation must restore the ordinary mask immediately. A following
	// plain tap on the same item is still the intentional tap-to-unequip path;
	// if release leaves the preview suppression latched, this press sees no
	// doll slot and silently does nothing.
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryReleased: true}, now)
	if !reflect.DeepEqual(town.clicked, []int{3}) {
		t.Fatalf("click after cancelled drag = %v, want [3] after the ordinary mask is restored", town.clicked)
	}
}

// TestTheApplicationTremorAcrossFormerNamePixelsCancelsOnTheSameSlot isolates
// the OrdinaryDollMask consumer in shopReleaseIsOrigin. Both pixels are now
// visible, but a crossed-TapSlop release on the same worn slot remains a
// cancellation rather than an unequip click.
func TestTheApplicationTremorAcrossFormerNamePixelsCancelsOnTheSameSlot(t *testing.T) {
	gap, glyph := image.Pt(558, 423), image.Pt(559, 424)
	mask := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	for _, p := range []image.Point{gap, glyph} {
		q := p.Sub(shopFigureRect.Min)
		mask.Slot[q.Y*mask.W+q.X] = 4
	}
	a, town := shopFullDollDragTestApp(t, mask)
	now := time.Unix(1_700_000_000, 0)
	away := gap.Add(image.Pt(4*TapSlop, 0))

	a.step(appInput{CursorX: gap.X, CursorY: gap.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: away.X, CursorY: away.Y}, now)
	a.step(appInput{CursorX: glyph.X, CursorY: glyph.Y, PrimaryReleased: true}, now)

	if len(town.clicked) != 0 || len(town.dragged) != 0 {
		t.Fatalf("same-slot release routed click=%v drag=%v, want neither", town.clicked, town.dragged)
	}
}

// TestTheApplicationDragsBetweenTwoShelfCellsOfTheSameFamily is
// counterexample 1 (round-2 adversarial review, third pass): a real drag
// from shelf cell 0 to shelf cell 5, both ShopControlShelfCell, crosses
// TapSlop and lands on a DIFFERENT cell of the SAME family. dest.Kind ==
// origin.Kind, the check this replaces, matches Kind alone and would route
// this release to ShopClick(dest) — buying shelf 5 for a press that
// grabbed shelf 0's item. dest == origin also compares Index, so a different
// cell of the same family falls through to dragShop(origin, dest);
// ShopDrag's own switch carries no case for a same-family pair, so the fixed
// behaviour is a no-op, not a same-family sale.
func TestTheApplicationDragsBetweenTwoShelfCellsOfTheSameFamily(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)
	dest := ShopShelfCellRect(5).Min.Add(image.Pt(4, 4))

	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: dest.X, CursorY: dest.Y}, now)
	a.step(appInput{CursorX: dest.X, CursorY: dest.Y, PrimaryReleased: true}, now)

	want := [][2]ShopControl{{
		{Kind: ShopControlShelfCell, Index: 0},
		{Kind: ShopControlShelfCell, Index: 5},
	}}
	if !reflect.DeepEqual(town.dragged, want) {
		t.Fatalf("dragged = %+v, want %+v: a cross-cell same-family release is a drag, not a click", town.dragged, want)
	}
	if len(town.clicked) != 0 {
		t.Errorf("clicked = %v, want none: dest.Kind == origin.Kind alone would wrongly resolve this as a click on cell 5", town.clicked)
	}
}

func TestTheApplicationDoesNotClickADifferentDollSlotAfterADollDrag(t *testing.T) {
	a, town := shopDragTestApp(t)
	w := town.mask.W
	town.mask.Slot[100*w+90] = 12 // a second marked slot, away from the origin
	now := time.Unix(1_700_000_000, 0)
	dest := shopFigureRect.Min.Add(image.Pt(90, 100))

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: dest.X, CursorY: dest.Y}, now)
	a.step(appInput{CursorX: dest.X, CursorY: dest.Y, PrimaryReleased: true}, now)

	want := [][2]ShopControl{{
		{Kind: ShopControlDoll, Index: 3},
		{Kind: ShopControlDoll, Index: 11},
	}}
	if !reflect.DeepEqual(town.dragged, want) {
		t.Fatalf("dragged = %+v, want %+v", town.dragged, want)
	}
	if len(town.clicked) != 0 {
		t.Fatalf("clicked = %v, want none: releasing a dragged item over another doll slot must not unequip the destination slot", town.clicked)
	}
}

// TestTheApplicationCapturesTheDragIconAtTapSlopAndClearsItOnRelease is
// Counterexample 2's own wiring witness (round-2 adversarial review): the
// icon is absent before TapSlop, present once the gesture crosses it, the
// SAME icon (not recomposed) for the rest of the gesture, and gone again the
// instant the button is released.
func TestTheApplicationCapturesTheDragIconAtTapSlopAndClearsItOnRelease(t *testing.T) {
	a, town := shopDragTestApp(t)
	town.dollIcon = image.NewRGBA(image.Rect(0, 0, 4, 4))
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	if icon, ok := a.shopDragItemPresent(); ok {
		t.Fatalf("shopDragItemPresent = (%v,true) before TapSlop, want false", icon)
	}

	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y}, now)
	icon1, ok := a.shopDragItemPresent()
	if !ok || icon1 != town.dollIcon {
		t.Fatalf("shopDragItemPresent = (%v,%v) past TapSlop, want the doll's own SlotIcon[3]", icon1, ok)
	}

	// A second move-only frame must not recompose: command.go's own
	// "captured once" precedent restated for the shop.
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y}, now)
	if icon2, _ := a.shopDragItemPresent(); icon2 != icon1 {
		t.Errorf("the icon changed identity across a second move frame, want the SAME captured picture")
	}

	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryReleased: true}, now)
	if icon3, ok := a.shopDragItemPresent(); ok {
		t.Errorf("shopDragItemPresent = (%v,true) after release, want false", icon3)
	}

	// A SECOND, UNRELATED GESTURE MUST NOT INHERIT THE FIRST ONE'S ICON: the
	// shelf cell this press starts from carries no icon in this fixture
	// (fakeShopTown never populates Shelf), so a stale a.shopDragIcon left
	// over from the doll drag above would leak the WRONG picture onto the
	// cursor rather than answering "none" — the press arm's own
	// a.shopDragIcon = nil is what a shopDragItemPresent()-only check on the
	// first gesture alone cannot witness.
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y}, now)
	if icon4, ok := a.shopDragItemPresent(); ok {
		t.Errorf("shopDragItemPresent = (%v,true) for a shelf origin with no icon, want false — not the previous gesture's doll icon", icon4)
	}
}

// TestEscapeDropsAnArmedShopDragBeforeUnwindingTheRoom is follow-up 2's own
// witness (round-2 adversarial review): pressing on the doll and then
// leaving the shop room by Escape, with the button still physically held,
// must not leave the drag machine armed — App.step's Escape branch calls
// clearShopDrag before flow.escape() unwinds the room, so the eventual
// release (which lands outside stepTown's own inShop gate once the room has
// changed) has nothing stale to act on.
func TestEscapeDropsAnArmedShopDragBeforeUnwindingTheRoom(t *testing.T) {
	a, town := shopDragTestApp(t)
	town.dollIcon = image.NewRGBA(image.Rect(0, 0, 4, 4))
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragShelfPoint.X, CursorY: shopDragShelfPoint.Y}, now)
	if _, ok := a.shopDragItemPresent(); !ok {
		t.Fatal("the drag never armed past TapSlop; the fixture does not exercise Escape's own guard")
	}

	a.step(appInput{Escape: true}, now)

	if a.shopDragArmed {
		t.Error("shopDragArmed is still true after Escape")
	}
	if a.shopDragIcon != nil {
		t.Error("shopDragIcon is still set after Escape")
	}
	if a.shopDragOrigin != (ShopControl{}) {
		t.Errorf("shopDragOrigin = %+v after Escape, want the zero value", a.shopDragOrigin)
	}

	// The eventual release — the button coming up on whatever screen Escape
	// left the player on — sends nothing across the seam: nothing here still
	// names the shop, and the town-only dispatch that would have read it is
	// stepTown's own inShop gate, which this input no longer reaches.
	a.step(appInput{PrimaryReleased: true}, now)
	if len(town.dragged) != 0 {
		t.Errorf("dragged = %+v, want none: the release followed Escape, not a live drag", town.dragged)
	}
}

// TestTheApplicationDragReleasedOutsideEverySurfaceIsANoOp is DIV-088's own
// case: a crossed-TapSlop drag whose release lands nowhere ShopControlAt or
// the doll recognises sends nothing across the seam — no ShopDrag, no
// ShopClick — so the item this gesture picked up stays exactly where it
// began, the town screen's own answer to "there is no ground here."
func TestTheApplicationDragReleasedOutsideEverySurfaceIsANoOp(t *testing.T) {
	a, town := shopDragTestApp(t)
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: shopDragMissPoint.X, CursorY: shopDragMissPoint.Y}, now)
	a.step(appInput{CursorX: shopDragMissPoint.X, CursorY: shopDragMissPoint.Y, PrimaryReleased: true}, now)

	if len(town.dragged) != 0 {
		t.Errorf("dragged = %+v, want none: the release named no recognised surface", town.dragged)
	}
	if len(town.clicked) != 0 {
		t.Errorf("clicked = %v, want none", town.clicked)
	}
}
