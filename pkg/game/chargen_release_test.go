package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseChargenDetailedNavArtIsDrawnUnmodified: the command panel is
// Inn\ButtonsArea.bmp with the three Inn off buttons over it at MENU-139's
// rectangles; inside a button a pixel is its art or label ink.
func TestReleaseChargenDetailedNavArtIsDrawnUnmodified(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if setup.PreCreate == nil || setup.PreCreate.Art == nil || setup.PreCreate.Art.NavArt == nil {
		t.Fatalf("production chargen art did not resolve NavArt")
	}
	art := setup.PreCreate.Art
	c := ui.NewChargen(setup)
	c.SelectPreChoice(0)
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		t.Fatalf("Forward did not reach DetailedStage")
	}
	frame := ui.ComposeChargenFrame(c)
	labels := ui.ChargenDetailedNavLabelRects(c)
	buttons := [3]image.Rectangle{labels[2], labels[1], labels[0]} // Accept, Reset, Back
	if buttons[0] != image.Rect(484, 44, 624, 90) || buttons[1] != image.Rect(484, 91, 624, 137) || buttons[2] != image.Rect(484, 138, 624, 184) {
		t.Fatalf("command rectangles %v, want MENU-139's", buttons)
	}
	region := ui.TownUpperRegion
	ink := color.RGBA{R: 255, G: 230, B: 150, A: 255}
	mismatches := 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			p := image.Pt(x, y)
			src, at := art.NavArt, region.Min
			for i, r := range buttons {
				if p.In(r) {
					src, at = art.NavButtons[i][0], r.Min
				}
			}
			got := frame.RGBAAt(x, y)
			r, g, b, a := src.At(x-at.X, y-at.Y).RGBA()
			want := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			if src != art.NavArt && inkShade(got, ink) {
				continue
			}
			if got != want {
				mismatches++
				if mismatches <= 5 {
					t.Errorf("nav region %v: composed pixel %#v != art pixel %#v", p, got, want)
				}
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d pixel(s) in the nav region are neither the shipped art nor label ink", mismatches)
	}
}

// inkShade reports whether c is ink scaled by one glyph coverage factor.
func inkShade(c, ink color.RGBA) bool {
	if c.R == 0 {
		return false
	}
	k := float64(c.R) / float64(ink.R)
	near := func(got, full uint8) bool {
		d := float64(got) - k*float64(full)
		return d > -3 && d < 3
	}
	return c.A == 0xff && near(c.G, ink.G) && near(c.B, ink.B)
}

// TestReleaseChargenDetailedNavLabelsAreDrawn is the complement of the test
// above and closes the surface that one deliberately excludes.
//
// WHY IT EXISTS. 1017's adversarial review pass 3 (W-1) found that the
// generator's three navigation labels were drawn correctly and witnessed by
// nothing: with all three install variables set, the whole suite stayed green
// when the labels were not drawn at all, when they were drawn in the
// background colour, and when Back and Play were swapped. That is the same
// class of defect this story shipped in round 1, where the school's and the
// tavern's buttons composed as ornate empty plaques. The test above forbids
// any overpaint of the nav art OUTSIDE the three label rectangles, which
// leaves 15762 of the region's 38080 pixels — 41% — asserted by nothing.
//
// WHAT IT WITNESSES, per rectangle: at least one pixel in the label colour;
// the drawn bounding box centred in the rectangle; and the drawn width
// agreeing with what the production font measures for THAT control's own
// string, which is what distinguishes a swap from a correct draw.
//
// WHAT IT DOES NOT WITNESS. Glyph identity. The width test separates two
// labels only when their strings measure differently, which the shipped
// wording satisfies on both roots but which a customised install need not.
func TestReleaseChargenDetailedNavLabelsAreDrawn(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if setup.PreCreate == nil || setup.PreCreate.Art == nil || setup.PreCreate.Art.Font == nil {
		t.Fatalf("production chargen art did not resolve a font")
	}
	if setup.Detailed == nil {
		t.Fatalf("production chargen setup carries no detailed wording")
	}
	// The three labels draw with font4, NameFont (MENU-139).
	font := setup.PreCreate.Art.NameFont
	if font == nil {
		font = setup.PreCreate.Art.Font
	}

	c := ui.NewChargen(setup)
	c.SelectPreChoice(0)
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		t.Fatalf("Forward did not reach DetailedStage")
	}

	frame := ui.ComposeChargenFrame(c)
	rects := ui.ChargenDetailedNavLabelRects(c)
	want := [3]string{setup.Detailed.Back, setup.Detailed.Reset, setup.Detailed.Play}
	label := color.RGBA{R: 255, G: 230, B: 150, A: 255}

	for i, r := range rects {
		box := image.Rectangle{}
		count := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if frame.RGBAAt(x, y) != label {
					continue
				}
				count++
				p := image.Rect(x, y, x+1, y+1)
				if box.Empty() {
					box = p
				} else {
					box = box.Union(p)
				}
			}
		}
		if count == 0 {
			t.Errorf("control %d (%q): no pixel in the label colour anywhere in %v", i, want[i], r)
			continue
		}
		measured, _ := font.Measure(want[i])
		t.Logf("control %d (%q): %d px, box %v in rect %v, font measures %d", i, want[i], count, box, r, measured)

		// font4's measure carries its trailing advance, so the inked box is
		// narrower by at most a quarter.
		if got, exp := box.Dx(), measured; got > exp || got < exp*3/4 {
			t.Errorf("control %d (%q): drawn width %d, font measures %d for this string", i, want[i], got, exp)
		}
		leftGap, rightGap := box.Min.X-r.Min.X, r.Max.X-box.Max.X
		if d := leftGap - rightGap; d < -3 || d > 3 {
			t.Errorf("control %d (%q): not centred, %d px left and %d px right in %v", i, want[i], leftGap, rightGap, r)
		}
	}
}

func TestReleaseChargenDetailedSeamColumnsDrawShippedStrips(t *testing.T) {
	f := releaseFront(t)

	assertRegion := func(t *testing.T, name string, frame *image.RGBA, rect image.Rectangle, src image.Image, exclude ...image.Rectangle) {
		t.Helper()
		b := src.Bounds()
		if b.Dx() != rect.Dx() || b.Dy() != rect.Dy() {
			t.Fatalf("%s: source is %dx%d, the column it fills is %dx%d", name, b.Dx(), b.Dy(), rect.Dx(), rect.Dy())
		}
		mismatches, total := 0, 0
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				p := image.Pt(x, y)
				excluded := false
				for _, e := range exclude {
					if p.In(e) {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
				total++
				got := frame.RGBAAt(x, y)
				sr, sg, sb, sa := src.At(b.Min.X+x-rect.Min.X, b.Min.Y+y-rect.Min.Y).RGBA()
				want := color.RGBA{R: uint8(sr >> 8), G: uint8(sg >> 8), B: uint8(sb >> 8), A: uint8(sa >> 8)}
				if got != want {
					mismatches++
					if mismatches <= 5 {
						t.Errorf("%s %v: composed pixel %#v != source pixel %#v", name, image.Pt(x, y), got, want)
					}
				}
			}
		}
		if mismatches > 0 {
			t.Fatalf("%s: %d of %d checked pixel(s) are not the shipped source", name, mismatches, total)
		}
		if total == 0 {
			t.Fatalf("%s: the exclusion rectangles %v cover the whole region %v, nothing was checked", name, exclude, rect)
		}
	}

	assertSeamRegion := func(t *testing.T, name string, frame, before *image.RGBA, rect image.Rectangle, src image.Image, exclude ...image.Rectangle) {
		t.Helper()
		b := src.Bounds()
		if b.Dx() != rect.Dx() || b.Dy() != rect.Dy() {
			t.Fatalf("%s: source is %dx%d, the column it fills is %dx%d", name, b.Dx(), b.Dy(), rect.Dx(), rect.Dy())
		}
		mismatches, total := 0, 0
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				p := image.Pt(x, y)
				excluded := false
				for _, e := range exclude {
					if p.In(e) {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
				total++
				sr, sg, sb, sa := src.At(b.Min.X+x-rect.Min.X, b.Min.Y+y-rect.Min.Y).RGBA()
				got := frame.RGBAAt(x, y)
				var want color.RGBA
				if sa == 0 {
					want = before.RGBAAt(x, y)
				} else {
					want = color.RGBA{R: uint8(sr >> 8), G: uint8(sg >> 8), B: uint8(sb >> 8), A: uint8(sa >> 8)}
				}
				if got != want {
					mismatches++
					if mismatches <= 5 {
						t.Errorf("%s %v: composed pixel %#v != want %#v (shipped seam pixel alpha %d)", name, image.Pt(x, y), got, want, sa)
					}
				}
			}
		}
		if mismatches > 0 {
			t.Fatalf("%s: %d of %d checked pixel(s) do not follow the shipped seam's own key/opaque split", name, mismatches, total)
		}
	}

	// chargenFrame composes the detailed page for one pre-choice (0=male
	// fighter, 1=male mage; Chargen.SelectPreChoice), through mutate applied
	// to a private copy of the resolved art. mutate is nil for the
	// production frame; a non-nil mutate builds the "before" frame a keyed
	// seam's own key pixels are read from. setup.PreCreate.Art already
	// points at a fresh *ui.ChargenPresentation returned by this call alone
	// (pkg/game/chargen.go's own ChargenSetup, "setup.PreCreate = &ui.
	// ChargenPreCreate{...}" each call), so replacing artCopy.Art on it
	// cannot mutate any other call's or test's own state; the two calls this
	// test makes per class each get their own.
	chargenFrame := func(t *testing.T, choice int, mutate func(*ui.ChargenPresentation)) (*image.RGBA, *ui.ChargenPresentation, *ui.ChargenPreview) {
		t.Helper()
		setup := f.ChargenSetup()
		if setup.PreCreate == nil || setup.PreCreate.Art == nil {
			t.Fatalf("production chargen art did not resolve")
		}
		artCopy := *setup.PreCreate.Art
		if mutate != nil {
			mutate(&artCopy)
		}
		setup.PreCreate.Art = &artCopy
		c := ui.NewChargen(setup)
		c.SelectPreChoice(choice)
		c.Forward()
		if c.Stage() != ui.DetailedStage {
			t.Fatalf("Forward did not reach DetailedStage for pre-choice %d", choice)
		}
		preview := c.Preview()
		return ui.ComposeChargenFrame(c), &artCopy, &preview
	}

	upperSeam := image.Rect(ui.TownWideUpperRegion.Min.X, ui.TownWideUpperRegion.Min.Y,
		ui.TownUpperRegion.Min.X, ui.TownUpperRegion.Max.Y)
	dollSeam := image.Rect(ui.TownWideUpperRegion.Min.X, ui.TownUpperRegion.Max.Y,
		ui.TownUpperRegion.Min.X, ui.TownCharacterRegion.Max.Y)
	plateSeam := image.Rect(160, 0, 176, 238)
	// cardSeam is chargenCardSeamRegion (1022 spec B2, unexported): the
	// card's own border-column strip, closing DIV-189.
	cardSeam := image.Rect(160, 238, 176, 480)
	// cardBox is chargenCardBox (1022 spec B3, unexported); its x:[0,160)
	// no longer reaches plateSeam's x:[160,176) at all, so the two never
	// overlap and this intersection is now always empty. It stays as a
	// parameter rather than being dropped: a future geometry change that
	// reintroduces an overlap will resurrect this exclusion automatically.
	cardBox := image.Rect(0, 238, 160, 480)

	classes := []struct {
		name   string
		choice int
	}{{"fighter", 0}, {"mage", 1}}
	for _, cl := range classes {
		t.Run(cl.name, func(t *testing.T) {
			frame, art, preview := chargenFrame(t, cl.choice, nil)
			if art.NavSeam == nil || art.PlateSeam == nil || art.DollPane.Body == nil || art.DollPane.Seam == nil {
				t.Fatalf("production chargen art did not resolve every seam/pane field: %+v", art)
			}

			navBefore, _, _ := chargenFrame(t, cl.choice, func(a *ui.ChargenPresentation) { a.NavSeam = nil })
			t.Run("nav seam", func(t *testing.T) {
				assertSeamRegion(t, "nav seam", frame, navBefore, upperSeam, art.NavSeam)
			})

			plateBefore, _, _ := chargenFrame(t, cl.choice, func(a *ui.ChargenPresentation) { a.PlateSeam = nil })
			t.Run("plate seam", func(t *testing.T) {
				assertSeamRegion(t, "plate seam", frame, plateBefore, plateSeam, art.PlateSeam, plateSeam.Intersect(cardBox))
			})

			// The card's own seam column, DIV-189's own strip (round-2
			// adversarial review D-3/D-4: this subtest did not exist, and
			// DIV-189's closure text and closure.md both claimed all four
			// border-column slots were pixel-checked when only three were).
			// It overlaps ChargenTipRect exactly as the doll seam below
			// overlaps it, for the same reason: the tip panel opens by
			// default (spec B5) and draws last, over
			// x:[160,176), y:[280,480) of this column.
			cardSeamTip := cardSeam.Intersect(chargenTipRect)
			cardBefore, _, _ := chargenFrame(t, cl.choice, func(a *ui.ChargenPresentation) { a.CardSeam = nil })
			t.Run("card seam", func(t *testing.T) {
				assertSeamRegion(t, "card seam", frame, cardBefore, cardSeam, art.CardSeam, cardSeamTip)
			})

			// The doll box's own seam column (x:[464,480)) never overlaps
			// the doll figure, which draws inside chargenDollBox
			// (x:[480,640)) alone. It does overlap the tip panel's own rect
			// (1022 spec B5, chargenTipRect = (160,280)-(472,480)) at
			// x:[464,472), y:[280,480): the panel now opens on the detailed
			// page by default (not yet closed, not yet suppressed) and
			// composes last, over this column like everything else on the
			// page. The excluded rectangle below is exactly that overlap.
			dollSeamTip := dollSeam.Intersect(chargenTipRect)
			dollBefore, _, _ := chargenFrame(t, cl.choice, func(a *ui.ChargenPresentation) { a.DollPane.Seam = nil })
			t.Run("doll seam", func(t *testing.T) {
				assertSeamRegion(t, "doll seam", frame, dollBefore, dollSeam, art.DollPane.Seam, dollSeamTip)
			})

			t.Run("doll body", func(t *testing.T) {
				dollFigure := image.Rectangle{}
				if preview.Doll != nil {
					at := ui.TownCharacterRegion.Min.Add(image.Pt(0, 2))
					dollFigure = preview.Doll.Bounds().Add(at.Sub(preview.Doll.Bounds().Min)).Intersect(ui.TownCharacterRegion)
				}
				assertRegion(t, "doll body", frame, ui.TownCharacterRegion, art.DollPane.Body, dollFigure)
			})
		})
	}
}
