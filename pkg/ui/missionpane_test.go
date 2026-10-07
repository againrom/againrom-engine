package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

// The mission screen's character pane: where its six corner controls are
// drawn, and what its own presentation shows at the shipped frame.
//
// THESE ARE FRAME COMPARISONS THROUGH THE PRODUCTION COMPOSER, not assertions
// about a rectangle-returning function's own return value. panelPresent is the
// call Viewer.Draw itself makes for this widget, and each case below reads the
// composed *image.RGBA. `AGENTS.md` rule 6: a GeometryTests entry earns its
// name by failing when the production DRAW CALL that reads the rectangle is
// mutated, not when the rectangle's own return statement is. Checked against
// two mutations at drawCharacterPaneCorners' own use site: a +1px shift of
// characterPaneArtRect's destination reddens the two geometry tests (the
// third does not check exact corner positions, only the toggle invariant, so
// a uniform shift applied to both sides of its comparison is invisible to
// it, by design); dropping characterPaneCornerDrawn's own `&& !modeFlag` term
// so rect C always paints reddens the geometry test and the toggle test
// both.

// missionPaneCornerColours is one flat opaque colour per shipped corner
// bitmap, all distinct and all distinct from the pane body below, so a pixel
// in the composed picture names exactly which bitmap painted it.
var missionPaneCornerColours = struct {
	backpackOpen, backpackClosed color.RGBA
	bookOpened, bookClosed       color.RGBA
	humanMode, textMode          color.RGBA
	diskette, ar1, ar2           color.RGBA
	body                         color.RGBA
}{
	backpackOpen:   color.RGBA{R: 0x11, A: 0xff},
	backpackClosed: color.RGBA{R: 0x22, A: 0xff},
	bookOpened:     color.RGBA{G: 0x33, A: 0xff},
	bookClosed:     color.RGBA{G: 0x44, A: 0xff},
	humanMode:      color.RGBA{B: 0x55, A: 0xff},
	textMode:       color.RGBA{B: 0x66, A: 0xff},
	diskette:       color.RGBA{R: 0x77, G: 0x77, A: 0xff},
	ar1:            color.RGBA{R: 0x88, B: 0x88, A: 0xff},
	ar2:            color.RGBA{G: 0x99, B: 0x99, A: 0xff},
	body:           color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff},
}

// missionPaneArt is a corner set of flat 40x40 blocks, each larger than the
// destination rectangle it is drawn into, so the composed picture carries the
// colour over the WHOLE of that rectangle and nothing of it outside.
func missionPaneArt() *CharacterPaneCornerArt {
	c := missionPaneCornerColours
	return &CharacterPaneCornerArt{
		BackpackOpen:   solidPic(40, 40, c.backpackOpen),
		BackpackClosed: solidPic(40, 40, c.backpackClosed),
		BookOpened:     solidPic(40, 40, c.bookOpened),
		BookClosed:     solidPic(40, 40, c.bookClosed),
		HumanMode:      solidPic(40, 40, c.humanMode),
		TextMode:       solidPic(40, 40, c.textMode),
		Diskette:       solidPic(40, 40, c.diskette),
		Ar1:            solidPic(40, 40, c.ar1),
		Ar2:            solidPic(40, 40, c.ar2),
	}
}

// missionPaneViewer is a viewer on the mission screen at the SHIPPED FRAME,
// 1024x768 — the only size v.frameH ever takes in production (viewer.go's own
// NewViewer, never reassigned by Layout) — carrying the corner art and a flat
// pane body, with no subject: the figure and the statistics card both need
// one, and leaving it out keeps the geometry case
// below about the corners alone.
func missionPaneViewer(t *testing.T) *Viewer {
	t.Helper()
	v := panelViewer(t)
	v.SetFont(panelFont())
	body := TownPane{Body: solidPic(160, 242, missionPaneCornerColours.body)}
	v.SetCharacterPaneArt(body, body, missionPaneArt())
	return v
}

// missionPaneCompose returns the pane picture panelPresent composed, and fails
// the test if it composed none.
func missionPaneCompose(t *testing.T, v *Viewer) *image.RGBA {
	t.Helper()
	pic, _, ok := v.panelPresent()
	if !ok || pic == nil {
		t.Fatal("panelPresent composed no character pane")
	}
	if got, want := pic.Bounds(), image.Rect(0, 0, 160+characterPaneSeamW, 242); got != want {
		t.Fatalf("the pane composed %v, want the id-7 slot's own %v widened by the seam", got, want)
	}
	return pic
}

// drawStatusLines puts the selection lines where the upper pane draws them,
// by arithmetic written out here and not read from the drawing code: each
// line centred on the middle of the 176-pixel column the seam starts, an odd
// remainder falling on the left, its
// cell 54 pixels below the pane's top for the first line and 12 more for each
// next, a flat shadow one pixel right and down under the gold ink.
func drawStatusLines(dst *image.RGBA, f *text.Font, lines [3]string) {
	ink := color.RGBA{R: 0xbd, G: 0x9e, B: 0x4a, A: 0xff}
	shadow := color.RGBA{R: 8, G: 8, B: 8, A: 0xff}
	for i, line := range lines {
		if line == "" {
			continue
		}
		w, _ := f.Measure(line)
		x, y := (characterPaneSeamW+160-w+1)/2, 54+12*i
		f.DrawFlat(dst, line, x+1, y+1, shadow)
		f.Draw(dst, line, x, y, ink)
	}
}

// TestMissionEmptySelectionKeepsAnEmptyDollAboveAnEmptyStatisticsPage is the
// mission column with nothing selected: the upper pane holds the empty doll,
// the lower box the empty statistics page, and each states the two lines the
// original draws for no character selected (`TEXT-UI-047`: global strings 47
// and 48 at zero selected, 49 and 50 with the count from two). The claim names
// the lines and no position, so the expected pictures put them where the
// owner's capture of the original does (`DIV-1795`) by a direct draw over the
// pane's own background. The lower statistics page states none.
func TestMissionEmptySelectionKeepsAnEmptyDollAboveAnEmptyStatisticsPage(t *testing.T) {
	v := missionPaneViewer(t)
	w := AuthoredWords()
	w.SelectionStatus = [4]string{"ZERO A", "ZERO B", "MANY A", "MANY B"}
	v.SetWords(w)
	figure := TownPane{Body: solidPic(160, 242, color.RGBA{R: 0x51, A: 0xff})}
	stats := TownPane{Body: solidPic(160, 242, color.RGBA{G: 0x62, A: 0xff})}
	v.SetCharacterPaneArt(figure, stats, nil)
	upper := missionPaneCompose(t, v)

	zero := [3]string{"ZERO A", "ZERO B"}
	view := v.characterPaneView(true)
	if view.Statistics || view.HasSubject || view.SelectionStatus != zero || view.Figure != nil {
		t.Fatalf("empty-selection upper view = statistics %v subject %v status %q figure %p; want the empty doll stating %q",
			view.Statistics, view.HasSubject, view.SelectionStatus, view.Figure, zero)
	}
	view.SelectionStatus = [3]string{}
	bareUpper := image.NewRGBA(upper.Bounds())
	DrawTownCharacterRegion(bareUpper, view)
	wantUpper := image.NewRGBA(upper.Bounds())
	DrawTownCharacterRegion(wantUpper, view)
	drawStatusLines(wantUpper, v.font, zero)
	if _, ok := firstDifference(wantUpper, bareUpper, upper.Bounds()); !ok {
		t.Fatal("the zero lines paint nothing over the empty doll, so this comparison cannot tell them from none")
	}
	if at, ok := firstDifference(upper, wantUpper, upper.Bounds()); ok {
		t.Fatalf("empty doll pane differs at %v: got %v want %v", at,
			upper.RGBAAt(at.X, at.Y), wantUpper.RGBAAt(at.X, at.Y))
	}
	if got := upper.RGBAAt(characterPaneSeamW+80, 120); got != (color.RGBA{R: 0x51, A: 0xff}) {
		t.Fatalf("upper pane center = %v, want figure-pane body", got)
	}

	lower, origin, ok := v.missionCardPresent()
	if !ok {
		t.Fatal("empty selection composed no lower statistics page")
	}
	if want := image.Pt(MissionViewportSize().X-characterPaneSeamW, 526); origin != want {
		t.Fatalf("lower statistics page origin = %v, want %v", origin, want)
	}
	cardView := view
	cardView.Statistics = true
	cardView.CornerArt = nil
	cardView.Figure = nil
	bareLower := image.NewRGBA(lower.Bounds())
	DrawCharacterPaneBody(bareLower, cardView)
	if at, ok := firstDifference(lower, bareLower, lower.Bounds()); ok {
		t.Fatalf("the empty lower statistics page states a line at %v: got %v want the bare page %v", at,
			lower.RGBAAt(at.X, at.Y), bareLower.RGBAAt(at.X, at.Y))
	}
	if got := lower.RGBAAt(characterPaneSeamW+80, 120); got != (color.RGBA{G: 0x62, A: 0xff}) {
		t.Fatalf("lower page center = %v, want statistics-pane body", got)
	}

	v.toggleHudPanel(hudPanelDoll)
	if toggled := v.characterPaneView(true); toggled.Statistics {
		t.Fatal("empty selection let the pane-mode toggle replace the empty upper doll")
	}
	if _, _, ok := v.missionCardPresent(); !ok {
		t.Fatal("empty selection let the pane-mode toggle remove the lower statistics page")
	}
}

func TestMissionPluralSelectionStatusKeysThePresentedCount(t *testing.T) {
	v := missionPaneViewer(t)
	v.SetEntities([]MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", 10, 10, 2, 2),
		panelEntity(3, "three", 10, 10, 3, 3),
	})
	v.sel = selection{1, 2}
	missionPaneCompose(t, v)
	if got := v.panelKey.selectionStatus; got != ([3]string{"Units", "selected:", "2"}) {
		t.Fatalf("two-selected key = %q", got)
	}
	builds := v.panelBuilds
	v.sel = selection{1, 2, 3}
	missionPaneCompose(t, v)
	if v.panelBuilds != builds+1 || v.panelKey.selectionStatus != ([3]string{"Units", "selected:", "3"}) {
		t.Fatalf("three-selected cache = builds %d status %q", v.panelBuilds, v.panelKey.selectionStatus)
	}
	v.sel = selection{1}
	view := v.characterPaneView(true)
	if !view.HasSubject || view.SelectionStatus != ([3]string{}) {
		t.Fatalf("one-selected view = subject %v status %q", view.HasSubject, view.SelectionStatus)
	}
}

// TestMissionBoxesRestateTheCountAsTheSelectionChanges presents both mission
// boxes as a frame does through two, three, no and two selected characters,
// and requires each to carry that state's own lines each time. A selection of
// two or more has no subject, so nothing but the lines tells the upper pane's
// cache one count from another. The expected pictures put the lines where the
// owner's capture of the original does (`DIV-1795`) by a direct draw over what
// the pane draws with no lines, so neither the cache key nor the production
// text call is the oracle. The lower card is the bare statistics page in every
// state.
func TestMissionBoxesRestateTheCountAsTheSelectionChanges(t *testing.T) {
	v := missionPaneViewer(t)
	v.SetEntities([]MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", 10, 10, 2, 2),
		panelEntity(3, "three", 10, 10, 3, 3),
	})
	var before *image.RGBA
	for _, tc := range []struct {
		what  string
		sel   selection
		lines [3]string
	}{
		{"two selected", selection{1, 2}, [3]string{"Units", "selected:", "2"}},
		{"three selected", selection{1, 2, 3}, [3]string{"Units", "selected:", "3"}},
		{"nothing selected", nil, [3]string{"No units", "selected"}},
		{"two selected again", selection{1, 2}, [3]string{"Units", "selected:", "2"}},
	} {
		v.sel = tc.sel
		upper := missionPaneCompose(t, v)
		card, _, ok := v.missionCardPresent()
		if !ok {
			t.Fatalf("%s: the lower box composed no card", tc.what)
		}
		view := v.characterPaneView(true)
		view.SelectionStatus = [3]string{}
		wantUpper := image.NewRGBA(upper.Bounds())
		DrawTownCharacterRegion(wantUpper, view)
		drawStatusLines(wantUpper, v.font, tc.lines)
		view.Statistics, view.CornerArt, view.Figure = true, nil, nil
		wantCard := image.NewRGBA(card.Bounds())
		DrawCharacterPaneBody(wantCard, view)
		if before != nil {
			if _, differs := firstDifference(before, wantUpper, before.Bounds()); !differs {
				t.Fatalf("%s: the expected pane equals the last state's, so this step cannot tell a stale pane from a fresh one", tc.what)
			}
		}
		before = wantUpper
		if at, ok := firstDifference(upper, wantUpper, upper.Bounds()); ok {
			t.Fatalf("%s: the upper pane differs at %v: got %v want %v", tc.what, at,
				upper.RGBAAt(at.X, at.Y), wantUpper.RGBAAt(at.X, at.Y))
		}
		if at, ok := firstDifference(card, wantCard, card.Bounds()); ok {
			t.Fatalf("%s: the lower card differs at %v: got %v want %v", tc.what, at,
				card.RGBAAt(at.X, at.Y), wantCard.RGBAAt(at.X, at.Y))
		}
	}
}

func TestMissionCharacterPaneCornersDrawAtTheDecodedRectangles(t *testing.T) {
	v := missionPaneViewer(t)
	if !v.characterPaneModeFlag() {
		t.Fatal("setup: characterPaneModeFlag is false at the shipped mission frame")
	}
	pic := missionPaneCompose(t, v)
	c := missionPaneCornerColours

	L := characterPaneSeamW
	for _, tc := range []struct {
		what string
		rect image.Rectangle
		col  color.RGBA
	}{
		// BackPackOp.bmp at (L, T+0xd0), 32x31 — rect A. A fresh viewer has
		// every HUD display switch SHOWN, the four being stored inverted
		// (hudtoggles.go), so the pack bar and the spellbook are both up and
		// both corners draw their OPEN bitmap.
		{"rect A, the backpack, open", image.Rect(L, 0xd0, L+32, 0xd0+31), c.backpackOpen},
		// BookOpened.bmp at (L, T), 28x38 — rect B.
		{"rect B, the spellbook, open", image.Rect(L, 0, L+28, 38), c.bookOpened},
		// diskette.bmp at (L+0x7e, T+0xce), 32x32 — rect F.
		{"rect F, the in-mission menu", image.Rect(L+0x7e, 0xce, L+0x7e+32, 0xce+32), c.diskette},
	} {
		coversExactly(t, pic, tc.rect, tc.col, tc.what)
	}

	for _, tc := range []struct {
		what string
		col  color.RGBA
	}{
		{"rect D, the party picker's previous", c.ar1},
		{"rect E, the party picker's next", c.ar2},
		{"rect A's closed bitmap, with the pack bar up", c.backpackClosed},
		{"rect B's closed bitmap, with the spellbook up", c.bookClosed},
		{"rect C's figure-mode bitmap, dead at the shipped frame", c.humanMode},
		{"rect C's statistics-mode bitmap, dead at the shipped frame", c.textMode},
	} {
		if at, ok := findColour(pic, tc.col); ok {
			t.Errorf("%s is painted at %v; the mission's own session gates it off", tc.what, at)
		}
	}
}

// TestMissionCharacterPaneCornersFollowTheirOwnState is the other half of the
// same geometry: rect A and rect B each have TWO bitmaps at two different
// rectangles, and which one is drawn is the state of the surface that corner
// opens (`TOWN-356`). Without this, an implementation that always drew the
// open pair would pass the case above.
func TestMissionCharacterPaneCornersFollowTheirOwnState(t *testing.T) {
	v := missionPaneViewer(t)
	v.toggleHudPanel(hudPanelPack)
	v.toggleHudPanel(hudPanelBook)
	pic := missionPaneCompose(t, v)
	c := missionPaneCornerColours

	// BackPackCl.bmp at (L+1, T+0xc9), 28x30 — one pixel right of the open
	// bitmap and seven rows above it.
	L := characterPaneSeamW
	coversExactly(t, pic, image.Rect(L+1, 0xc9, L+1+28, 0xc9+30), c.backpackClosed, "rect A, the backpack, closed")
	// BookClosed.bmp at (L, T+4), 28x37 — four rows below the open bitmap.
	coversExactly(t, pic, image.Rect(L, 4, L+28, 4+37), c.bookClosed, "rect B, the spellbook, closed")

	for _, tc := range []struct {
		what string
		col  color.RGBA
	}{
		{"rect A's open bitmap, with the pack bar down", c.backpackOpen},
		{"rect B's open bitmap, with the spellbook down", c.bookOpened},
	} {
		if at, ok := findColour(pic, tc.col); ok {
			t.Errorf("%s is painted at %v", tc.what, at)
		}
	}
}

func TestMissionCharacterPaneTogglesBetweenFigureAndStatistics(t *testing.T) {
	v := missionPaneViewer(t)
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	v.sel = selection{1}
	v.SetInventorySubject(InventorySubject{ID: 1, Figure: solidPic(160, 200, color.RGBA{R: 0xf0, G: 0x0f, A: 0xff})})

	if !v.characterPaneModeFlag() {
		t.Fatal("setup: characterPaneModeFlag is false at the shipped mission frame")
	}
	if v.characterPaneStatistics() {
		t.Fatal("setup: a fresh viewer is not in statistics mode (hudPanelDoll's own zero value)")
	}
	figureFrame := missionPaneCompose(t, v)

	v.toggleHudPanel(hudPanelDoll)
	if !v.characterPaneStatistics() {
		t.Fatal("toggling hudPanelDoll did not switch the interactive pane to statistics mode")
	}
	statsFrame := missionPaneCompose(t, v)

	if _, ok := firstDifference(figureFrame, statsFrame, figureFrame.Bounds()); !ok {
		t.Error("toggling hudPanelDoll produced an identical frame")
	}

	// Rect C stays dead through the toggle: neither of its own two bitmaps
	// ever paints, in either mode — `TOWN-346`'s flag is permanently true at
	// the shipped 1024x768 frame, and this gating is unchanged by round 3
	// (pass 2 confirmed it; do not re-derive it).
	c := missionPaneCornerColours
	for _, pic := range []*image.RGBA{figureFrame, statsFrame} {
		for _, col := range []color.RGBA{c.humanMode, c.textMode} {
			if at, ok := findColour(pic, col); ok {
				t.Errorf("rect C painted colour %v at %v though characterPaneModeFlag is true", col, at)
			}
		}
	}
}

func TestMissionDrawsTheFigureAndTheStatisticsCardTogether(t *testing.T) {
	v := missionPaneViewer(t)
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	v.sel = selection{1}
	figure := color.RGBA{R: 0xf0, G: 0x0f, A: 0xff}
	v.SetInventorySubject(InventorySubject{ID: 1, Figure: solidPic(160, 200, figure)})

	if v.characterPaneStatistics() {
		t.Fatal("setup: a fresh viewer opens on the figure (`TOWN-351`, the constructor forces 1)")
	}
	pane := missionPaneCompose(t, v)
	if _, ok := findColour(pane, figure); !ok {
		t.Fatal("the pane painted no figure in figure mode")
	}
	card, _, ok := v.missionCardPresent()
	if !ok {
		t.Fatal("the fourth slot composed no statistics card while the pane holds the figure (`DIV-344`)")
	}
	if got, want := card.Bounds(), pane.Bounds(); got != want {
		t.Fatalf("the card composed %v, want the pane's own %v", got, want)
	}

	v.toggleHudPanel(hudPanelDoll)
	if !v.characterPaneStatistics() {
		t.Fatal("toggling hudPanelDoll did not switch the pane to statistics mode")
	}
	statsPane := missionPaneCompose(t, v)
	if at, ok := findColour(statsPane, figure); ok {
		t.Errorf("the pane still paints the figure at %v in statistics mode", at)
	}
	if _, _, ok := v.missionCardPresent(); ok {
		t.Error("the fourth slot still composes a card while the pane itself shows one")
	}

	L := characterPaneSeamW
	corners := []image.Rectangle{
		image.Rect(L, 0xd0, L+32, 0xd0+31),
		image.Rect(L, 0, L+28, 38),
		image.Rect(L+0x7e, 0xce, L+0x7e+32, 0xce+32),
	}
	b := card.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := image.Pt(x, y)
			covered := false
			for _, r := range corners {
				if p.In(r) {
					covered = true
					break
				}
			}
			if covered {
				continue
			}
			if got, want := card.RGBAAt(x, y), statsPane.RGBAAt(x, y); got != want {
				t.Fatalf("the fourth slot's card at (%d,%d) is %v, want the pane's own statistics pixel %v",
					x, y, got, want)
			}
		}
	}
}

// coversExactly asserts that col is the colour of every pixel inside r and of
// no pixel outside it.
func coversExactly(t *testing.T, pic *image.RGBA, r image.Rectangle, col color.RGBA, what string) {
	t.Helper()
	b := pic.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			got := pic.RGBAAt(x, y) == col
			want := image.Pt(x, y).In(r)
			if got != want {
				if want {
					t.Fatalf("%s: (%d,%d) is %v, want the bitmap's own %v over all of %v",
						what, x, y, pic.RGBAAt(x, y), col, r)
				}
				t.Fatalf("%s: (%d,%d) carries the bitmap's own %v outside %v",
					what, x, y, col, r)
			}
		}
	}
}

// findColour reports the first pixel of that colour, top to bottom.
func findColour(pic *image.RGBA, col color.RGBA) (image.Point, bool) {
	b := pic.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if pic.RGBAAt(x, y) == col {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}

// firstDifference reports the first pixel inside r on which the two pictures
// disagree.
func firstDifference(a, b *image.RGBA, r image.Rectangle) (image.Point, bool) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}
