package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
)

// The floating tip panel's own install-gated witness (1018 spec behaviours
// 1, 2; DIV-162), on townsquare_release_test.go's own pattern: compose the
// real install through the production path and require the result to
// change against a baseline, rather than trusting a synthetic fixture that
// never installs real art.
//
// WHY THIS FILE EXISTS. This story's own panel is exactly that shape — a
// bordered frame this build blits from three shipped nodes, a wrapped tip
// body, a Close label and a "Show tips" label — so the same failure (art
// paints, text does not) is checked directly rather than assumed absent
// because the synthetic tests in tippanel_test.go pass.
//
// tipTextColour and tipShellTextColour restate pkg/ui's own unexported
// shopTextColor and townShellText, on chargen_release_test.go's own
// "label := color.RGBA{...}" precedent: this package cannot reach an
// unexported constant a package away, and the panel's own body text and its
// two control labels are drawn in exactly these two colours (tippanel.go).
var (
	tipTextColour      = color.RGBA{R: 0xe8, G: 0xdc, B: 0xc0, A: 0xff}
	tipShellTextColour = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}
	// tipCloseInk is the shared push button's grey caption (210,210,210)
	// packed to 16-bit RGB565 and expanded (MENU-115).
	tipCloseInk = color.RGBA{R: 213, G: 210, B: 213, A: 0xff}
)

func countPixels(img *image.RGBA, r image.Rectangle, want color.RGBA) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.RGBAAt(x, y) == want {
				n++
			}
		}
	}
	return n
}

func diffPixelCount(a, b *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

// assertTipPanelWitnessed is the body every room's own test below shares.
// v must be Showing() — real Fill, Border, Font, Text and a non-empty Rect
// all resolved from the install, not a fixture. with and without are the
// SAME frame composed once with the panel in place and once with it forced
// off; they must differ inside v.Rect (the art painted something, not a
// same-coloured no-op). The tip body, the Close label and the toggle's own
// "Show tips" label must each show at least one pixel in their own drawn
// colour — the specific surface a diff-only check cannot see, since a diff
// already passes once the fill and border alone paint and the labels stay
// blank.
func assertTipPanelWitnessed(t *testing.T, screen string, v ui.TipPanelView, words ui.Words, with, without *image.RGBA) {
	t.Helper()
	if !v.Showing() {
		t.Fatalf("%s: TipPanelView is not Showing() -- real Fill/Border/Font/Text/Rect did not all resolve from the install", screen)
	}
	if n := diffPixelCount(with, without, v.Rect); n == 0 {
		t.Errorf("%s: composing the panel changed 0 pixels inside its own Rect %v against the same frame with no panel", screen, v.Rect)
	}
	if v.CloseLabel != words.TipClose || v.ToggleLabel != words.TipShowNext {
		t.Errorf("%s: tip captions = %q / %q, want install words %q / %q",
			screen, v.CloseLabel, v.ToggleLabel, words.TipClose, words.TipShowNext)
	}
	if n := countPixels(with, ui.TipPanelTextRect(v.Rect), tipTextColour); n == 0 {
		t.Errorf("%s: no pixel in the tip body's own text colour inside %v", screen, ui.TipPanelTextRect(v.Rect))
	}
	if n := countPixels(with, ui.TipPanelCloseRect(v.Rect), tipCloseInk); n == 0 {
		t.Errorf("%s: no pixel in the Close label's own text colour inside %v", screen, ui.TipPanelCloseRect(v.Rect))
	}
	if n := countPixels(with, ui.TipPanelToggleRect(v.Rect), tipShellTextColour); n == 0 {
		t.Errorf("%s: no pixel in the toggle label's own text colour inside %v", screen, ui.TipPanelToggleRect(v.Rect))
	}
}

// The town square's own tip, at construction (spec behaviour 2).
func TestReleaseTownSquareTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	v := s.TownSquareView()
	without := v
	without.Tip = ui.TipPanelView{}
	assertTipPanelWitnessed(t, "town square", v.Tip, f.Words,
		ui.ComposeTownSquare(v), ui.ComposeTownSquare(without))
}

// The shop's own tip, now the bordered panel rather than 1011's bare text
// (spec behaviour 5).
func TestReleaseShopTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	// The shop door, townDoors[1] (tips_test.go's own comment). Choose loads
	// the tip before it opens the room's own head offer dialogue, if one is
	// on offer, which a fresh campaign's shop has; ShopScreen's own TipPanel
	// is keyed to roomShop unconditionally (shopview.go), independently of
	// t.room, so it is unaffected either way and this test does not assert
	// t.room itself.
	s.Choose(1)
	v := s.ShopScreen()
	without := v
	without.TipPanel = ui.TipPanelView{}
	compose := func(vv ui.ShopScreenView) *image.RGBA {
		return ui.ComposeShopScreen(vv, image.Point{}, false, nil, false)
	}
	assertTipPanelWitnessed(t, "shop", v.TipPanel, f.Words, compose(v), compose(without))
}

// The school's own tip, sharing TownSurfaceView.Tip with the tavern
// (townshell.go's own doc).
func TestReleaseSchoolTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	// The school door, townDoors[2]. Choose loads the tip before it opens
	// the school's own head offer dialogue, if one is on offer; TownSurface
	// still builds the school's own view while a school dialogue is open
	// (its own room-or-roomTalk-with-this-building branch, townshell.go),
	// so this test does not assert t.room itself.
	s.Choose(2)
	v := s.TownSurface()
	without := v
	without.Tip = ui.TipPanelView{}
	assertTipPanelWitnessed(t, "school", v.Tip, f.Words,
		ui.ComposeTownSurface(v), ui.ComposeTownSurface(without))
}

// The tavern's own tip, the other TownSurfaceView.Tip caller.
func TestReleaseTavernTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	s.Choose(0) // the tavern door, townDoors[0]
	if s.room != roomTavern {
		t.Fatalf("Choose(0) left room = %v, want roomTavern", s.room)
	}
	v := s.TownSurface()
	without := v
	without.Tip = ui.TipPanelView{}
	assertTipPanelWitnessed(t, "tavern", v.Tip, f.Words,
		ui.ComposeTownSurface(v), ui.ComposeTownSurface(without))
}

// The generator's own tip, on the pre-create page only (TOWN-187's own
// "class bit read once, at open"; spec behaviours 1, 2). The baseline here
// is CloseTip's own real effect rather than a hand-built zero view, so this
// test also witnesses that a close genuinely removes the panel from the
// composed frame (spec behaviour 3) against real art, not only against the
// synthetic fixture in chargen_tip_test.go.
func TestReleaseChargenTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	c := ui.NewChargen(setup)
	if c.Stage() != ui.PreCreateStage {
		t.Fatal("a fresh Chargen is not at PreCreateStage")
	}
	v := c.TipPanel()
	with := ui.ComposeChargenFrame(c)
	c.CloseTip()
	without := ui.ComposeChargenFrame(c)
	assertTipPanelWitnessed(t, "chargen", v, f.Words, with, without)
}

// TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt is round
// 2's Finding B witness (DIV-162), against a real install rather than
// chargen_tip_test.go's own synthetic fixture: f.ChargenSetup() resolves
// both shipped chrgen1f.txt/chrgen1m.txt nodes, and selecting a mage
// portrait (choice 1, male mage) through the real ui.Chargen state machine
// composes the panel with the mage text's own pixels where the fighter
// text's stood — a real install regression of the same shape the synthetic
// test cannot see (real art, real font, real wrapping), on this file's own
// "why this file exists" rationale.
func TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if setup.TipTextMage == "" {
		t.Fatal("f.ChargenSetup().TipTextMage is empty: the mage tip node did not resolve from this install")
	}
	if setup.TipTextMage == setup.TipText {
		t.Fatal("TipTextMage == TipText: the fighter and mage nodes read identically on this install, the fixture cannot discriminate")
	}
	c := ui.NewChargen(setup)
	fighter := ui.ComposeChargenFrame(c)
	c.SelectPreChoice(1) // male mage
	mage := ui.ComposeChargenFrame(c)

	v := c.TipPanel()
	if v.Text != setup.TipTextMage {
		t.Fatalf("TipPanel().Text after SelectPreChoice(1) = %q, want the mage node", v.Text)
	}
	if n := diffPixelCount(fighter, mage, ui.TipPanelTextRect(v.Rect)); n == 0 {
		t.Error("selecting the mage portrait changed 0 pixels inside the tip body's own text rect: " +
			"the composed frame still shows the fighter text")
	}
}

// Every one of the six shipped tip texts this story reads draws WHOLE, with
// no line silently dropped past the panel's own text rect (DIV-162): a
// mis-sized rect answers Showing() true and paints a partial tip with no
// indicator, which TestReleaseTownSquareTipPanelIsWitnessedAgainstRealArt's
// own kind of test cannot see, since it only checks that SOME pixel of the
// tip body's own colour appears, not that every wrapped line did. This is
// the release-side companion to TestShopTipRectExtendsPastTheMessageStrip
// and its own package's synthetic tests: those prove the rects have the
// SHAPE this story intends; this proves that shape holds the six real
// shipped payloads on this root, not only a fixture built to fit.
func TestReleaseTipPanelTextsDrawWholeOnBothClasses(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}

	s.Choose(1) // shop
	if v := s.ShopScreen().TipPanel; !ui.TipPanelFits(v) {
		t.Errorf("shop tip does not fit its own panel rect %v", v.Rect)
	}

	f2 := releaseFront(t)
	s2 := f2.TownScreen().(*townScreen)
	s2.Choose(0) // tavern
	if v := s2.TownSurface().Tip; !ui.TipPanelFits(v) {
		t.Errorf("tavern tip does not fit its own panel rect %v", v.Rect)
	}

	f3 := releaseFront(t)
	s3 := f3.TownScreen().(*townScreen)
	s3.Choose(2) // school
	if v := s3.TownSurface().Tip; !ui.TipPanelFits(v) {
		t.Errorf("school tip does not fit its own panel rect %v", v.Rect)
	}
	if v := s3.TownSquareView().Tip; !ui.TipPanelFits(v) {
		t.Errorf("town square tip does not fit its own panel rect %v", v.Rect)
	}

	// Both chargen texts, fighter and mage (TOWN-187's class branch,
	// chargenTipPath): ChargenSetup itself now resolves both (round 2 /
	// DIV-162 correction — ui.Chargen.TipPanel picks between them by the
	// player's own live pre-create selection, chargen_tip_test.go's own
	// TestPreCreateTipTextFollowsTheLiveClassChoice). Both are read directly
	// here as well so this test proves the fit against ChargenTipRect for
	// each shipped node independently of which one a given setup happens to
	// pick.
	var src entrySource
	if f3.Archives != nil {
		src = f3.Archives.Containers
	}
	fighterText, ok := ReadShopTip(src, ChargenFighterTipPath)
	if !ok {
		t.Fatal("ReadShopTip(ChargenFighterTipPath) reported no shipped node")
	}
	mageText, ok := ReadShopTip(src, ChargenMageTipPath)
	if !ok {
		t.Fatal("ReadShopTip(ChargenMageTipPath) reported no shipped node")
	}
	art := f3.tipArt()
	// THE FONT IS THE PRE-CREATE PAGE'S OWN (round 3, D-3): ui.Chargen.TipPanel
	// (pkg/ui/chargen.go) resolves its font from c.setup.PreCreate.Art.Font,
	// which LoadChargenAssets (chargenassets.go) loads as "font2" — a
	// different resource from f.Font ("font1"), the shared font every other
	// room's own tipView (tips.go) draws with. Before this fix the check
	// below built its view with f3.Font, the wrong font for this screen; it
	// passed only because round 2's own taller rect (height 250) still fit
	// font1's wider glyphs, and reddened the instant ChargenTipRect's height
	// dropped to this round's own 170 — a false failure against a rect that
	// fits the real font (cmd/tippanelcheck) with a 10-row margin.
	//
	// THERE IS NO FALLBACK TO f3.Font (round-3 adversarial review, pass 3).
	// This resolved font2 when it could and font1 otherwise, and font1 is
	// the exact defect above: a witness that silently degrades into the
	// instrument it was written to replace reports a pass either way. An
	// install that does not resolve the pre-create page's own font fails
	// here instead.
	if f3.ChargenAssets == nil || f3.ChargenAssets.Presentation == nil || f3.ChargenAssets.Presentation.Font == nil {
		t.Fatal("pre-create page's own font did not resolve from this install; ui.Chargen.TipPanel draws with it, so there is nothing to check the fit against")
	}
	font := f3.ChargenAssets.Presentation.Font
	for name, text := range map[string]string{"fighter": fighterText, "mage": mageText} {
		v := ui.TipPanelView{Rect: ui.ChargenTipRect, Text: text, Art: art, Font: font}
		if !ui.TipPanelFits(v) {
			t.Errorf("chargen %s tip does not fit ChargenTipRect %v", name, v.Rect)
		}
	}
}
