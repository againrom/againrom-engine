package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
)

var (
	tipTextColour      = color.RGBA{R: 185, G: 159, B: 73, A: 0xff}
	tipShellTextColour = color.RGBA{R: 185, G: 159, B: 73, A: 0xff}
	tipCloseInk        = color.RGBA{R: 189, G: 157, B: 74, A: 0xff}
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

func assertTipPanelWitnessed(t *testing.T, screen string, v ui.TipPanelView, words ui.Words, with, without *image.RGBA) {
	t.Helper()
	if !v.Showing() {
		t.Fatalf("%s: TipPanelView is not Showing(): installed art, font, text or rectangle did not resolve", screen)
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

func TestReleaseShopTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	s.Choose(1)
	v := s.ShopScreen()
	without := v
	without.TipPanel = ui.TipPanelView{}
	compose := func(vv ui.ShopScreenView) *image.RGBA {
		return ui.ComposeShopScreen(vv, image.Point{}, false, nil, false)
	}
	assertTipPanelWitnessed(t, "shop", v.TipPanel, f.Words, compose(v), compose(without))
}

func TestReleaseSchoolTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	s.Choose(2)
	v := s.TownSurface()
	without := v
	without.Tip = ui.TipPanelView{}
	assertTipPanelWitnessed(t, "school", v.Tip, f.Words,
		ui.ComposeTownSurface(v), ui.ComposeTownSurface(without))
}

func TestReleaseTavernTipPanelIsWitnessedAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}
	s.Choose(0)
	if s.room != roomTavern {
		t.Fatalf("Choose(0) left room = %v, want roomTavern", s.room)
	}
	v := s.TownSurface()
	without := v
	without.Tip = ui.TipPanelView{}
	assertTipPanelWitnessed(t, "tavern", v.Tip, f.Words,
		ui.ComposeTownSurface(v), ui.ComposeTownSurface(without))
}

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

// TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt: the
// detailed page's enter shows chrgen1m for a mage and chrgen1f otherwise
// (TOWN-522).
func TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if setup.TipTextMage == "" || setup.TipTextMage == setup.TipText {
		t.Fatal("the fighter and mage nodes did not resolve as two texts on this install")
	}
	frames := map[int]*image.RGBA{}
	for _, choice := range []int{0, 1} {
		c := ui.NewChargen(setup)
		c.SelectPreChoice(choice)
		c.Forward()
		want := map[int]string{0: setup.TipText, 1: setup.TipTextMage}[choice]
		if v := c.TipPanel(); v.Text != want || v.Rect != chargenTipRect {
			t.Fatalf("choice %d detailed popup = %q at %v", choice, v.Text, v.Rect)
		}
		frames[choice] = ui.ComposeChargenFrame(c)
	}
	if n := diffPixelCount(frames[0], frames[1], ui.TipPanelTextRect(chargenTipRect)); n == 0 {
		t.Error("the mage and fighter detailed popups composed the same text pixels")
	}
}

func TestReleaseTipPanelTextsDrawWholeOnBothClasses(t *testing.T) {
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen() did not return *townScreen")
	}

	s.Choose(1)
	if v := s.ShopScreen().TipPanel; !ui.TipPanelFits(v) {
		t.Errorf("shop tip does not fit its own panel rect %v", v.Rect)
	}

	f2 := releaseFront(t)
	s2 := f2.TownScreen().(*townScreen)
	s2.Choose(0)
	if v := s2.TownSurface().Tip; !ui.TipPanelFits(v) {
		t.Errorf("tavern tip does not fit its own panel rect %v", v.Rect)
	}

	f3 := releaseFront(t)
	s3 := f3.TownScreen().(*townScreen)
	s3.Choose(2)
	if v := s3.TownSurface().Tip; !ui.TipPanelFits(v) {
		t.Errorf("school tip does not fit its own panel rect %v", v.Rect)
	}
	if v := s3.TownSquareView().Tip; !ui.TipPanelFits(v) {
		t.Errorf("town square tip does not fit its own panel rect %v", v.Rect)
	}

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
	if f3.ChargenAssets == nil || f3.ChargenAssets.Presentation == nil || f3.ChargenAssets.Presentation.Font == nil {
		t.Fatal("pre-create page's own font did not resolve from this install; ui.Chargen.TipPanel draws with it, so there is nothing to check the fit against")
	}
	font := f3.ChargenAssets.Presentation.Font
	for name, text := range map[string]string{"fighter": fighterText, "mage": mageText} {
		v := ui.TipPanelView{Rect: chargenTipRect, Text: text, Art: art, Font: font}
		if !ui.TipPanelFits(v) {
			t.Errorf("chargen %s tip does not fit ChargenTipRect %v", name, v.Rect)
		}
	}
	for i, addr := range ChargenSelectTipPaths {
		text, ok := ReadShopTip(src, addr)
		if !ok {
			t.Fatalf("%s is not shipped", addr)
		}
		if v := (ui.TipPanelView{Rect: preCreateTipRect, Text: text, Art: art, Font: font}); !ui.TipPanelFits(v) {
			t.Errorf("pre-create step %d tip does not fit PreCreateTipRect %v", i, v.Rect)
		}
	}
}
