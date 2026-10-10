package game

import (
	"image"
	"os"
	"testing"

	"againrom/pkg/ui"
)

// WHY THESE TWO TESTS ARE INSTALL-GATED.
//
// The statistics card is a font budget: seventeen two-column rows inside
// 160x242 pixels. Both tests passed while the shipped screen truncated every
// right-column value away, clipped every right-column label at the card's
// edge, and placed the last two rows past the card's own height. A synthetic
// font cannot witness a font budget, so these run against a lawful install
// and skip without one (golden rule 2).
//
// THE INK IS READ OFF THE PRODUCTION COMPOSITION, not off a second render.
// Each test composes the screen through its own production composer and takes
// as the card's text every pixel that differs from the shipped background
// bitmap the card is composed on. composePanel draws that bitmap into the card
// unscaled and at the card's own origin, then paints the rows over it, so the
// difference is the text and nothing else. This is the same instrument
// TestReleaseTownSquareOverlayDoesNotPaintBlackOverTheSky uses on the square.
//
// The scan stops at card-local y=205, where the pane's own three persistent
// controls begin (townCharacterPrev/Mode/Next). Those are chrome, not card
// rows. The row count below is stated as a literal for that reason: it is
// the number of rows the owner's card has, and the card has no room for an
// eighteenth at this font.

// cardRowBand is the card-local y range the card's own rows may occupy: from
// its top edge down to the first of the three persistent controls below them.
const cardRowBand = 205

// townCardRegion composes DrawTownCharacterRegion for v on a 640x480 canvas
// and returns the frame.
func townCardRegion(v ui.TownCharacterView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	ui.DrawTownCharacterRegion(dst, v)
	return dst
}

func townCardBodyRegion(v ui.TownCharacterView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	ui.DrawCharacterPaneBody(dst, v)
	return dst
}

// checkCardNotOverpainted requires that nothing drawn after the card body
// (the six corner controls) paints over a pixel the card's own text already
// occupies. bodyOnly is full's view composed with the corners left out
// (townCardBodyRegion); a pixel is the card's own ink where bodyOnly differs
// from plain, the card's unwritten background.
//
// DIV-328
func checkCardNotOverpainted(t *testing.T, what string, full, bodyOnly, plain *image.RGBA, box image.Rectangle) {
	t.Helper()
	bad, first := 0, image.Point{}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if bodyOnly.RGBAAt(x, y) == plain.RGBAAt(x, y) {
				continue // not card ink
			}
			if full.RGBAAt(x, y) != bodyOnly.RGBAAt(x, y) {
				if bad == 0 {
					first = image.Pt(x-box.Min.X, y-box.Min.Y)
				}
				bad++
			}
		}
	}
	if bad != 0 {
		t.Errorf("%s: a later blit painted over %d card-ink pixel(s) in the row band, first at card-local %v",
			what, bad, first)
	}
}

// cardInkMask is the set of pixels in box where frame and plain differ,
// expressed in card-local coordinates (box.Min is the origin), less any pixel
// inside one of the excluded rectangles.
//
// EXCLUSION IS FOR CHROME PAINTED INSIDE THE ROW BAND. The scan already
// stops at cardRowBand because the pane's controls used to sit below the
// rows; two of the six corner rectangles now stand at the pane's own top
// corners, inside the band, and their shipped bitmaps differ from the body
// they are drawn on exactly as text does. The caller passes
// ui.TownCharacterPersistentControls, which is the corners' own gate rather
// than a second copy of their rectangles.
func cardInkMask(frame, plain *image.RGBA, box image.Rectangle, exclude []image.Rectangle) [][2]int {
	var out [][2]int
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if frame.RGBAAt(x, y) == plain.RGBAAt(x, y) {
				continue
			}
			skip := false
			for _, r := range exclude {
				if (image.Point{X: x, Y: y}).In(r) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			out = append(out, [2]int{x - box.Min.X, y - box.Min.Y})
		}
	}
	return out
}

// inkBounds is the mask's own bounding box, and reports whether it has any
// pixel at all.
func inkBounds(mask [][2]int) (image.Rectangle, bool) {
	if len(mask) == 0 {
		return image.Rectangle{}, false
	}
	r := image.Rect(mask[0][0], mask[0][1], mask[0][0]+1, mask[0][1]+1)
	for _, p := range mask[1:] {
		r = r.Union(image.Rect(p[0], p[1], p[0]+1, p[1]+1))
	}
	return r, true
}

// firstInkBand is the mask restricted to its topmost run of consecutive inked
// rows. On this card that run is the name row: it is the first row drawn, and
// the layout's own Gap leaves at least one blank scan line under every row.
func firstInkBand(mask [][2]int) [][2]int {
	full, ok := inkBounds(mask)
	if !ok {
		return nil
	}
	rows := map[int]bool{}
	for _, p := range mask {
		rows[p[1]] = true
	}
	last := full.Min.Y
	for rows[last+1] {
		last++
	}
	var out [][2]int
	for _, p := range mask {
		if p[1] <= last {
			out = append(out, p)
		}
	}
	return out
}

// checkCardFits is the whole of what "the card fits" means, applied to one
// composed card: every laid-out row keeps its stated value, seventeen rows are
// placed, and no glyph of any of them is painted outside the card's own
// padded box.
func checkCardFits(t *testing.T, what string, l ui.PanelLayout, lineH int,
	rep []ui.PanelLineReport, mask [][2]int) {
	t.Helper()
	if len(rep) != 17 {
		t.Fatalf("%s: the card laid out %d row(s), want 17", what, len(rep))
	}
	for i, ln := range rep {
		if ln.Value != ln.FullValue {
			t.Errorf("%s: row %d (%q) had its value truncated from %q to %q at the production font",
				what, i, ln.Label, ln.FullValue, ln.Value)
		}
		if ln.RightValue != ln.FullRightValue {
			t.Errorf("%s: row %d (%q) had its right value truncated from %q to %q at the production font",
				what, i, ln.RightLabel, ln.FullRightValue, ln.RightValue)
		}
		if ln.At.Y+lineH > l.Size.Y-l.Pad.Y {
			t.Errorf("%s: row %d (%q) is placed at y=%d and is %d pixels tall, past the card's "+
				"own last usable row at y=%d",
				what, i, ln.Label, ln.At.Y, lineH, l.Size.Y-l.Pad.Y)
		}
	}
	box, ok := inkBounds(mask)
	if !ok {
		t.Fatalf("%s: the composed card painted no text at all", what)
	}
	want := image.Rect(min(l.Pad.X, 6), l.Pad.Y, l.Size.X-l.Pad.X, l.Size.Y-l.Pad.Y)
	if !box.In(want) {
		t.Errorf("%s: the card's text ink spans %v, outside its own padded box %v", what, box, want)
	}
}

// checkNameCentred requires the card's first drawn row to sit on the card's
// name axis, x=72. The expected value is a literal and is not
// derived from the name, the font or the layout pass under test.
func checkNameCentred(t *testing.T, what string, l ui.PanelLayout, mask [][2]int) {
	t.Helper()
	band, ok := inkBounds(firstInkBand(mask))
	if !ok {
		t.Fatalf("%s: the composed card painted no name row", what)
	}
	// Twice the centre, so an odd ink width needs no rounding rule. The
	// tolerance is one pixel each way: a glyph's own side bearing can leave
	// the painted run a pixel off the pen's own midpoint.
	got, want := band.Min.X+band.Max.X, 144
	if got < want-2 || got > want+2 {
		t.Errorf("%s: the name row's ink spans x=[%d,%d) over y=[%d,%d), centred at %.1f against "+
			"the name axis %.1f",
			what, band.Min.X, band.Max.X, band.Min.Y, band.Max.Y, float64(got)/2, float64(want)/2)
	}
}

func TestReleaseStatisticsCardFitsAtTheProductionFont(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.arriveInTown()
	s := f.TownScreen().(*townScreen)
	s.townStats = true

	t.Run("town", func(t *testing.T) {
		v := s.townCharacterView()
		if !v.HasSubject {
			t.Fatal("the production town character pane has no subject")
		}
		bg, _ := v.StatsPane.Body.(*image.RGBA)
		if bg == nil {
			t.Fatal("the production statistics pane has no shipped body bitmap")
		}
		if v.CardFont == nil {
			t.Fatal("the production town character pane resolved no card font")
		}
		box := ui.TownCharacterRegion
		box.Max.Y = box.Min.Y + cardRowBand
		frame := townCardRegion(v)
		mask := cardInkMask(frame, cardBackgroundFrame(bg, ui.TownCharacterRegion), box,
			ui.TownCharacterPersistentControls())
		l := ui.CompactPanelLayout(bg)
		checkCardFits(t, "town statistics card", l, v.CardFont.Height(),
			ui.CharacterPanelReport(l, v.CardFont, v.Subject), mask)
		checkNameCentred(t, "town statistics card", l, mask)
		checkCardNotOverpainted(t, "town statistics card", frame, townCardBodyRegion(v),
			cardBackgroundFrame(bg, ui.TownCharacterRegion), box)
	})

	t.Run("chargen", func(t *testing.T) {
		cf := releaseFront(t)
		app := cf.App("1027-release-chargen-card")
		app.Layout(640, 480)
		if err := app.HeadlessActivate("new game"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() == ui.ScreenPicker {
			if err := app.HeadlessActivate("@first"); err != nil {
				t.Fatal(err)
			}
		}
		state, ok := app.HeadlessChargenState()
		if !ok {
			t.Fatal("the production generator answered no state")
		}
		if err := headlessChargenPress(app, &state, ui.ChargenControlForward, ""); err != nil {
			t.Fatalf("forward to the detailed page: %v", err)
		}
		if state.Stage != ui.ChargenStageDetailed {
			t.Fatalf("forward left stage %q, want the detailed page", state.Stage)
		}
		frame, note, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		if note != "" {
			t.Fatalf("the detailed page composed incompletely: %s", note)
		}
		if cf.ChargenAssets == nil || cf.ChargenAssets.Presentation == nil {
			t.Fatal("the production generator resolved no presentation")
		}
		p := cf.ChargenAssets.Presentation
		if p.CardBackground == nil {
			t.Fatal("the production generator resolved no card background")
		}
		// The card's own box on this page, its background bitmap's own size at
		// its own origin. chargenCardBox is unexported; the origin below is
		// the same (0,238) DIV-193's shared card size is anchored at.
		full := p.CardBackground.Bounds().Sub(p.CardBackground.Bounds().Min).Add(image.Pt(0, 238))
		box := full
		box.Max.Y = box.Min.Y + cardRowBand
		mask := cardInkMask(frame, cardBackgroundFrame(p.CardBackground, full), box, nil)
		lg := ui.CompactPanelLayout(p.CardBackground)
		// The card canvas stands the description's card offset right of the
		// box (the town card builder's CardOffset); the name axis is the
		// canvas's.
		off := cf.generator().Detail.CardOffset.Pt()
		for i := range mask {
			mask[i][0] -= off.X
		}
		bb, _ := inkBounds(mask)
		t.Logf("generator card: offset=%v text ink %v over %d pixel(s)", off, bb, len(mask))
		checkNameCentred(t, "generator statistics card", lg, mask)
	})
}

// cardBackgroundFrame is a 640x480 canvas carrying bg at box, so a composed
// page can be differenced against the card's own unwritten background.
func cardBackgroundFrame(bg *image.RGBA, box image.Rectangle) *image.RGBA {
	bg = ui.CompactPanelLayout(bg).Background
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	b := bg.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.SetRGBA(box.Min.X+x, box.Min.Y+y, bg.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func TestReleaseShopDrawsNothingOverTheStatisticsCard(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the shop composition witness needs a lawful install")
	}
	app, s := releaseShopApp(t)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlCharacterMode})
	v := s.ShopScreen()
	if !v.Character.Statistics {
		t.Fatal("the production shop did not enter statistics mode")
	}
	_ = app
	shop := ui.ComposeShopScreen(v, image.Point{}, false, nil, false)
	shared := townCardRegion(v.Character)
	r := ui.TownCharacterRegion
	bad, first := 0, image.Point{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if shop.RGBAAt(x, y) != shared.RGBAAt(x, y) {
				if bad == 0 {
					first = image.Pt(x-r.Min.X, y-r.Min.Y)
				}
				bad++
			}
		}
	}
	if bad != 0 {
		t.Fatalf("the composed shop differs from the shared character region at %d pixel(s), "+
			"first at card-local %v: the shop drew over the statistics card", bad, first)
	}
}

func TestReleaseChargenMaximumAllocationFitsTheCard(t *testing.T) {
	f := releaseFront(t)
	c := ui.NewChargen(f.ChargenSetup())
	c.SelectPreChoice(0)
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		t.Fatalf("Forward left stage %v, want the detailed page", c.Stage())
	}
	// Round-robin rather than one stat at a time: the point cost rises with
	// the value, so spending the whole budget on one row stops early and
	// leaves the other three at their floor.
	for spent := true; spent; {
		spent = false
		for stat := 0; stat < 4; stat++ {
			if c.Remaining() <= 0 {
				break
			}
			before := c.Remaining()
			if c.AdjustStat(stat, +1) && c.Remaining() >= 0 {
				spent = true
				continue
			}
			if c.Remaining() != before {
				c.AdjustStat(stat, -1)
			}
		}
	}
	if !c.Legal() {
		t.Fatalf("the maximised spread is not one generation could produce (remaining %d)", c.Remaining())
	}
	sub := c.Preview().Subject
	t.Logf("maximum allocation: remaining=%d rows=%v HP=%d/%d Mana=%d/%d Defence=%d ToHit=%d",
		c.Remaining(), c.DerivedText(), sub.HP, sub.MaxHP, sub.Mana, sub.MaxMana,
		sub.Combat.Defence, sub.Combat.ToHit)

	if f.ChargenAssets == nil || f.ChargenAssets.Presentation == nil {
		t.Fatal("the production generator resolved no presentation")
	}
	p := f.ChargenAssets.Presentation
	if p.Font == nil || p.CardBackground == nil {
		t.Fatal("the production generator resolved no font or card background")
	}
	l := ui.CompactPanelLayout(p.CardBackground)
	card := ui.RenderCharacterPanel(l, p.Font, sub)
	if card == nil {
		t.Fatal("the maximised subject composed no card")
	}
	box := card.Bounds()
	mask := cardInkMask(card, cardBackgroundFrameAt(p.CardBackground, box), box, nil)
	checkCardFits(t, "generator card at the maximum allocation", l, p.Font.Height(),
		ui.CharacterPanelReport(l, p.Font, sub), mask)
	checkNameCentred(t, "generator card at the maximum allocation", l, mask)
}

// cardBackgroundFrameAt is cardBackgroundFrame for a destination that is the
// card's own bounds rather than a 640x480 page.
func cardBackgroundFrameAt(bg *image.RGBA, box image.Rectangle) *image.RGBA {
	bg = ui.CompactPanelLayout(bg).Background
	dst := image.NewRGBA(box)
	b := bg.Bounds()
	for y := 0; y < b.Dy() && box.Min.Y+y < box.Max.Y; y++ {
		for x := 0; x < b.Dx() && box.Min.X+x < box.Max.X; x++ {
			dst.SetRGBA(box.Min.X+x, box.Min.Y+y, bg.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
