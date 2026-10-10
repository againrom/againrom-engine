package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

func chargenMask(w, h int) *image.Paletted {
	return image.NewPaletted(image.Rect(0, 0, w, h), make(color.Palette, 256))
}

func countChargenPixel(pic *image.RGBA, want color.RGBA) int {
	if pic == nil {
		return 0
	}
	n := 0
	for y := pic.Bounds().Min.Y; y < pic.Bounds().Max.Y; y++ {
		for x := pic.Bounds().Min.X; x < pic.Bounds().Max.X; x++ {
			if pic.RGBAAt(x, y) == want {
				n++
			}
		}
	}
	return n
}

func nonBlackChargenPixels(pic image.Image, at image.Point) []image.Point {
	if pic == nil {
		return nil
	}
	b := pic.Bounds()
	out := make([]image.Point, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := pic.At(x, y).RGBA()
			if r != 0 || g != 0 || bl != 0 {
				out = append(out, image.Pt(x-b.Min.X+at.X, y-b.Min.Y+at.Y))
			}
		}
	}
	return out
}

func chargenTestFont() *text.Font {
	glyphs := make([]text.Glyph, 224)
	for i := range glyphs {
		glyphs[i] = text.Glyph{Width: 3, Height: 5, Advance: 3, Pixels: []text.Pixel{
			{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
			{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
			{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
			{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
			{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
		}}
	}
	return &text.Font{Glyphs: glyphs}
}

func centeredChargenOrigin(f *text.Font, value string, box image.Rectangle) image.Point {
	w, h := f.Measure(value)
	return image.Pt(box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-h)/2)
}

func TestPreCreateControlBoundsAndNativePixels(t *testing.T) {
	art := &ChargenPresentation{}
	for i := range art.Choices {
		pic := image.NewRGBA(image.Rect(0, 0, 8+i, 9+i))
		for state := range art.Choices[i] {
			art.Choices[i][state] = pic
		}
	}
	art.Forward = image.NewRGBA(image.Rect(0, 0, 7, 11))
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: art}})
	if got := preControlAt(c, image.Pt(640, 0)); got != chargenNone {
		t.Fatalf("letterbox/outside point hit %v", got)
	}
	for i := 0; i < 4; i++ {
		id := chargenChoice0 + chargenControl(i)
		if got := preControlAt(c, preControlRect(c, id).Min); got != id {
			t.Errorf("choice %d hit = %v, want %v", i, got, id)
		}
		if got := preControlAt(c, preControlRegion(id).Min); got != chargenNone {
			t.Errorf("choice %d band padding hit %v", i, got)
		}
	}
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pic.SetRGBA(0, 0, color.RGBA{R: 201, A: 255})
	for state := range art.Choices[0] {
		art.Choices[0][state] = pic
	}
	got := composeChargenPage(c, chargenNone, chargenNone)
	if pixel := got.RGBAAt(preChoiceOrigin[0].X, preChoiceOrigin[0].Y); pixel.R != 201 {
		t.Errorf("choice pixel = %#v, want native source pixel", pixel)
	}
	if got := preControlAt(c, preForwardOrigin); got != chargenForward {
		t.Fatalf("forward pixel hit = %v, want Forward", got)
	}
	if got := preControlAt(c, preForwardOrigin.Add(image.Pt(7, 0))); got != chargenNone {
		t.Fatalf("forward padding hit = %v", got)
	}
}

// TestPreCreateNameControlRegionIsPublished pins
// preControlRegion(chargenName) against a hand-transcribed literal, not
// against a second call to preControlRect (which would read the same
// production rect this test is meant to catch a shift in —
// TestPreCreateControlBoundsAndNativePixels's own choice/forward cases
// hit-test at a point THEIR OWN production rect just produced, which is why
// the audit's mutation of this exact line (chargen_page.go:100) survived:
// chargenName was never enumerated there at all).
func TestPreCreateNameControlRegionIsPublished(t *testing.T) {
	want := image.Rect(224, 310, 362, 337) // the field's rect, TEXT-075
	if got := preControlRegion(chargenName); got != want {
		t.Errorf("preControlRegion(chargenName) = %v, want %v", got, want)
	}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{}}})
	if got := preControlAt(c, want.Min); got != chargenName {
		t.Errorf("preControlAt(region.Min) = %v, want chargenName: the hit test and the published rect must agree at its own corner", got)
	}
	if got := preControlAt(c, image.Pt(want.Min.X-1, want.Min.Y)); got == chargenName {
		t.Errorf("preControlAt one pixel left of the region's own left edge still hit chargenName")
	}
}

// TestPreCreateNameTextDrawsAtItsOwnOrigin pins preCreateNameTextOrigin
// against a hand-transcribed literal, independent of composeChargenPage's
// own Font.Draw call (round 2 adversarial review, finding 1:
// chargen_page.go:529's inline literal had no test that could fail when it
// moved — this is that test, on the pre-create page only; the chargen
// DETAILED page's own name placement is untouched and out of this story's
// scope). It also confirms composeChargenPage's real render paints ink at
// that exact origin for a non-empty name.
func TestPreCreateNameTextDrawsAtItsOwnOrigin(t *testing.T) {
	want := image.Pt(224, 321) // TEXT-077
	if got := preCreateNameTextOrigin(); got != want {
		t.Fatalf("preCreateNameTextOrigin() = %v, want %v", got, want)
	}

	art := &ChargenPresentation{Font: chargenTestFont()}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: art}})
	c.EditName("M", false)

	frame := composeChargenPage(c, chargenNone, chargenNone)
	if got := frame.RGBAAt(want.X, want.Y); got.A == 0 {
		t.Errorf("composeChargenPage drew no ink at preCreateNameTextOrigin() = %v for a non-empty name", want)
	}
}

func TestDetailedControlBoundsAndDollReplacement(t *testing.T) {
	art := &ChargenPresentation{Plate: image.NewRGBA(image.Rect(0, 0, 160, 238))}
	for class := range art.Skills {
		for skill := range art.Skills[class] {
			for state := range art.Skills[class][skill] {
				art.Skills[class][skill][state] = image.NewRGBA(image.Rect(0, 0, 11+skill, 13+skill))
			}
		}
	}
	doll := image.NewRGBA(image.Rect(0, 0, 160, 240))
	doll.SetRGBA(1, 1, color.RGBA{B: 199, A: 255})
	preview := ChargenPreview{Doll: doll}
	c := NewChargen(ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices:   []ChargenChoice{{Options: []string{"m"}, Parent: -1}, {Options: []string{"f"}, Parent: -1}, {Options: []string{"a", "b", "c", "d", "e"}, Parent: -1}},
		Stats:     []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
		Preview: func(ChargenResult) ChargenPreview { return preview },
	})
	c.Forward()
	id := chargenSkill2
	if got := detailedControlAt(c, detailedControlRect(c, id).Min); got != id {
		t.Fatalf("skill hit = %v, want %v", got, id)
	}
	if got := detailedControlAt(c, detailedControlRegion(id).Min); got != chargenNone {
		t.Fatalf("skill padding hit = %v", got)
	}
	frame := composeChargenPage(c, chargenNone, chargenNone)
	if got := frame.RGBAAt(481, 241); got.B != 199 {
		t.Fatalf("doll pixel = %#v, want native doll pixel", got)
	}
	// chargenDollBox's own frame no longer draws (round-2 adversarial review
	// item 4, owner: remove the brown rectangle around the doll); its corner
	// shows the page's own background fill instead.
	if got := frame.RGBAAt(chargenDollBox.Min.X, chargenDollBox.Min.Y); got != (color.RGBA{18, 18, 18, 255}) {
		t.Fatalf("doll box corner = %#v, want page background (no frame)", got)
	}
	preview = ChargenPreview{}
	c.SelectSkill(1)
	frame = composeChargenPage(c, chargenNone, chargenNone)
	if got := frame.RGBAAt(481, 241); got.B == 199 {
		t.Fatalf("missing Doll retained old pixel %#v", got)
	}
}

func TestDetailedStatTextUsesDecodedPlateBoxes(t *testing.T) {
	font := chargenTestFont()
	art := &ChargenPresentation{Plate: image.NewRGBA(image.Rect(0, 0, 160, 238)), Font: font}
	buttonColor := func(direction, state int) color.RGBA {
		return color.RGBA{R: uint8(30 + direction*80 + state), A: 255}
	}
	for direction := range art.StatButtons {
		for state := range art.StatButtons[direction] {
			pic := image.NewRGBA(image.Rect(0, 0, 20, 20))
			for y := 0; y < 20; y++ {
				for x := 0; x < 20; x++ {
					pic.SetRGBA(x, y, buttonColor(direction, state))
				}
			}
			art.StatButtons[direction][state] = pic
		}
	}
	c := NewChargen(ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"m"}, Parent: -1},
			{Options: []string{"fighter"}, Parent: -1},
			{Options: []string{"a", "b", "c", "d", "e"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}, {Floor: 0, Ceiling: 50, Start: 25}, {Floor: 0, Ceiling: 50, Start: 25}, {Floor: 0, Ceiling: 50, Start: 25}},
		Cost:  triangular(50), Budget: 2000,
	})
	c.Forward()
	frame := composeChargenDetailedPage(c, chargenNone, chargenNone)
	for row := range chargenStatValueBox {
		at := centeredChargenOrigin(font, "25", chargenStatValueBox[row])
		if !(image.Rectangle{Min: at, Max: at.Add(image.Pt(font.Measure("25")))}).In(chargenStatValueBox[row]) {
			t.Fatalf("row %d value at %v is outside decoded box %v", row, at, chargenStatValueBox[row])
		}
		if got := frame.RGBAAt(at.X, at.Y); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
			t.Errorf("row %d value pixel at exact centered origin = %#v, want game-font ink", row, got)
		}
		if got := frame.RGBAAt(chargenStatMinusBox[row].Min.X, chargenStatMinusBox[row].Min.Y); got != buttonColor(0, 0) {
			t.Errorf("row %d minus rest pixel = %#v, want native mnloff", row, got)
		}
		if got := frame.RGBAAt(chargenStatPlusBox[row].Min.X, chargenStatPlusBox[row].Min.Y); got != buttonColor(1, 0) {
			t.Errorf("row %d plus rest pixel = %#v, want native pnloff", row, got)
		}
	}
	remaining := centeredChargenOrigin(font, "100", chargenRemainingBox)
	if got := frame.RGBAAt(remaining.X, remaining.Y); got != (color.RGBA{R: 255, G: 230, B: 150, A: 255}) {
		t.Errorf("remaining point text at %v = %#v, want game-font ink", remaining, got)
	}
	if got := composeChargenDetailedPage(c, chargenStatMinus0, chargenNone).RGBAAt(chargenStatMinusBox[0].Min.X, chargenStatMinusBox[0].Min.Y); got != buttonColor(0, 1) {
		t.Errorf("minus hover pixel = %#v, want native mloff", got)
	}
	if got := composeChargenDetailedPage(c, chargenStatMinus0, chargenStatMinus0).RGBAAt(chargenStatMinusBox[0].Min.X, chargenStatMinusBox[0].Min.Y); got != buttonColor(0, 2) {
		t.Errorf("minus pressed pixel = %#v, want native mlon", got)
	}
	c.statValue[0] = c.setup.Stats[0].Floor
	if got := composeChargenDetailedPage(c, chargenNone, chargenNone).RGBAAt(chargenStatMinusBox[0].Min.X, chargenStatMinusBox[0].Min.Y); got != buttonColor(0, 4) {
		t.Errorf("minus disabled pixel = %#v, want native mdisable", got)
	}
}

func TestDetailedSkillUsesSelectedHoverAndPressedSourceStates(t *testing.T) {
	art := &ChargenPresentation{Plate: image.NewRGBA(image.Rect(0, 0, 160, 238))}
	for skill := 0; skill < 5; skill++ {
		for state := 0; state < 3; state++ {
			pic := image.NewRGBA(image.Rect(0, 0, 5, 5))
			for y := 0; y < 5; y++ {
				for x := 0; x < 5; x++ {
					pic.SetRGBA(x, y, color.RGBA{R: uint8(20 + 10*skill + state), A: 255})
				}
			}
			art.Skills[0][skill][state] = pic
		}
	}
	c := NewChargen(ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"m"}, Parent: -1},
			{Options: []string{"fighter"}, Parent: -1},
			{Options: []string{"a", "b", "c", "d", "e"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
	})
	c.Forward()

	pixelAt := func(id chargenControl, hover, pressed chargenControl) color.RGBA {
		r := detailedControlRect(c, id)
		frame := composeChargenDetailedPage(c, hover, pressed)
		return frame.RGBAAt(r.Min.X+(r.Dx()-1)/2, r.Min.Y+(r.Dy()-1)/2)
	}
	const background = 18 // composeChargenDetailedPage's own fill color; no column art is set in this fixture, so the rest corner's un-drawn pixel stays at this value
	if got := pixelAt(chargenSkill0, chargenNone, chargenNone); got.R != 20 {
		t.Fatalf("selected, not hovered source state pixel = %#v, want state 0 (on.bmp, pressed dark)", got)
	}
	if got := pixelAt(chargenSkill1, chargenSkill1, chargenNone); got.R != 31 {
		t.Fatalf("hover source state pixel = %#v, want state 1", got)
	}
	if got := pixelAt(chargenSkill2, chargenNone, chargenSkill2); got.R != 40 {
		t.Fatalf("captured press alone reads selected, not hovered: pixel = %#v, want state 0 (on.bmp, pressed dark)", got)
	}
	if got := pixelAt(chargenSkill0, chargenNone, chargenSkill2); got.R != background {
		t.Fatalf("prior selected skill remained selected during a different press: %#v, want background %d (rest corner draws no patch)", got, background)
	}
	c.focus = 1 // keyboard focus on the unselected second skill
	if got := pixelAt(chargenSkill1, chargenNone, chargenNone); got.R != background {
		t.Fatalf("focused unselected source state pixel = %#v, want background %d; focus alone does not select or hover", got, background)
	}
	for selected := 0; selected < 5; selected++ {
		c.SelectSkill(selected)
		for skill := 0; skill < 5; skill++ {
			got := pixelAt(chargenSkill0+chargenControl(skill), chargenNone, chargenNone).R
			if skill == selected {
				want := uint8(20 + 10*skill) // selectedAtRest: state 0, on.bmp
				if got != want {
					t.Errorf("selection %d skill %d (selected, not hovered) state pixel = %d, want %d", selected, skill, got, want)
				}
				continue
			}
			if got != background {
				t.Errorf("selection %d skill %d (rest) state pixel = %d, want background %d (no patch)", selected, skill, got, background)
			}
		}
	}
	frame := composeChargenDetailedPage(c, chargenNone, chargenNone)
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			if got := frame.RGBAAt(x, y); got == (color.RGBA{R: 255, G: 255, A: 255}) {
				t.Fatalf("unexpected yellow focus overlay at %v", image.Pt(x, y))
			}
		}
	}
}

// TestDetailedSkillPictureIsTwoIndependentBooleans discriminates all four
// (selected, hovered) combinations for one skill's picture. It first checks
// detailedSkillPictures.pick against four synthetic pictures, one per
// combination, so a collapse back to a three- or fewer-value ordinal shows as
// a repeated picture. It then checks detailedSkillArt, the adapter that
// builds that table from the three pictures the install ships per skill,
// against all four combinations: three map to a distinct shipped picture, and
// the fourth (not selected, not hovered) deliberately has none, because the
// column canvas already carries that corner as its own baked art.
func TestDetailedSkillPictureIsTwoIndependentBooleans(t *testing.T) {
	onePixel := func() image.Image { return image.NewRGBA(image.Rect(0, 0, 1, 1)) }
	rest, hoverPic, pressedHover, pressedRest := onePixel(), onePixel(), onePixel(), onePixel()
	table := detailedSkillPictures{rest: rest, hover: hoverPic, selected: pressedHover, selectedAtRest: pressedRest}

	got := map[string]image.Image{
		"not selected, not hovered": table.pick(false, false),
		"not selected, hovered":     table.pick(false, true),
		"selected, hovered":         table.pick(true, true),
		"selected, not hovered":     table.pick(true, false),
	}
	want := map[string]image.Image{
		"not selected, not hovered": rest,
		"not selected, hovered":     hoverPic,
		"selected, hovered":         pressedHover,
		"selected, not hovered":     pressedRest,
	}
	for label, want := range want {
		if got := got[label]; got != want {
			t.Errorf("%s picture = %v, want %v", label, got, want)
		}
	}
	seen := map[image.Image]bool{}
	for _, pic := range got {
		seen[pic] = true
	}
	if len(seen) != 4 {
		t.Fatalf("the four (selected, hovered) pairs chose %d distinct pictures, want 4: %v", len(seen), got)
	}

	// detailedSkillArt builds the four-picture table from the three the
	// install ships per skill: on.bmp (pressed dark), shine_off.bmp (raised
	// light), shine_on.bmp (pressed light). The fourth corner, not selected
	// and not hovered (raised dark), has no shipped picture: the column
	// canvas already carries it as its own baked art, drawn once before the
	// skill loop, so this corner draws nothing rather than reusing any of
	// the three named pictures.
	on, shineOff, shineOn := onePixel(), onePixel(), onePixel()
	real := detailedSkillArt([3]image.Image{on, shineOff, shineOn})
	cases := []struct {
		label             string
		selected, hovered bool
		want              image.Image
	}{
		{"not selected, not hovered", false, false, nil},
		{"not selected, hovered", false, true, shineOff},
		{"selected, hovered", true, true, shineOn},
		{"selected, not hovered", true, false, on},
	}
	seenReal := map[image.Image]bool{}
	for _, tc := range cases {
		got := real.pick(tc.selected, tc.hovered)
		if got != tc.want {
			t.Errorf("%s picture over real install art = %v, want %v", tc.label, got, tc.want)
		}
		if got != nil {
			seenReal[got] = true
		}
	}
	if len(seenReal) != 3 {
		t.Fatalf("the three non-rest corners chose %d distinct shipped pictures, want 3 (on, shine_off, shine_on each used exactly once): %v", len(seenReal), seenReal)
	}
}

// TestDetailedPageDrawsTheProductionCompactCard proves the detailed page's
// own card slot (chargenCardBox, 1022 spec B3) carries exactly the pixels
// RenderCharacterPanel(CompactPanelLayout(...), ...) produces for the same
// subject and background — the shared production card, not a private copy
// — and that every one of those pixels sits inside chargenCardBox. B4
// removed the old message BOX (drawChargenDetailMessage,
// chargenDetailMessageBox); round-2 adversarial review restored the copy it
// carried through a different surface, chargenMessageRect in the doll box
// — see TestDetailedPageDrawsTheProductionMessageStrip below, which is
// that surface's own independence witness and is deliberately not folded
// into this test, since the two surfaces do not overlap.
func TestDetailedPageDrawsTheProductionCompactCard(t *testing.T) {
	font := chargenTestFont()
	bg := image.NewRGBA(image.Rect(0, 0, 160, 242))
	art := &ChargenPresentation{Plate: image.NewRGBA(image.Rect(0, 0, 160, 238)), CardBackground: bg, Font: font}
	c := NewChargen(ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"m"}, Parent: -1},
			{Options: []string{"fighter"}, Parent: -1},
			{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
	})
	c.Forward()
	frame := composeChargenDetailedPage(c, chargenNone, chargenNone)
	card := RenderCharacterPanel(CompactPanelLayout(bg), font, c.Preview().Subject)
	if card.Bounds().Dx() > chargenCardBox.Dx() || card.Bounds().Dy() > chargenCardBox.Dy() {
		t.Fatalf("native production Card size=%v exceeds destination=%v", card.Bounds().Size(), chargenCardBox.Size())
	}
	for y := 0; y < card.Bounds().Dy(); y++ {
		for x := 0; x < card.Bounds().Dx(); x++ {
			if got, want := frame.RGBAAt(chargenCardBox.Min.X+x, chargenCardBox.Min.Y+y), card.RGBAAt(x, y); got != want {
				t.Fatalf("generator Card pixel at %v=%#v, shared production pixel=%#v", image.Pt(x, y), got, want)
			}
		}
	}
}

func TestDetailedPageDrawsTheProductionMessageStrip(t *testing.T) {
	font := chargenTestFont()
	newSetup := func() (*Chargen, *ChargenPresentation) {
		art := &ChargenPresentation{
			Plate:          image.NewRGBA(image.Rect(0, 0, 160, 238)),
			CardBackground: image.NewRGBA(image.Rect(0, 0, 160, 242)),
			Font:           font,
			DollPane:       TownPane{Body: image.NewRGBA(image.Rect(0, 0, 160, 242))},
		}
		c := NewChargen(ChargenSetup{
			PreCreate: &ChargenPreCreate{Art: art},
			Choices: []ChargenChoice{
				{Options: []string{"m"}, Parent: -1},
				{Options: []string{"fighter"}, Parent: -1},
				{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
			},
			Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
		})
		c.Forward()
		return c, art
	}

	before, _ := newSetup()
	frameBefore := composeChargenDetailedPage(before, chargenNone, chargenNone)

	after, _ := newSetup()
	const msg = "cannot confirm: name is empty"
	after.SetDetailMessage(msg)
	frameAfter := composeChargenDetailedPage(after, chargenNone, chargenNone)

	// The expectation is built independently of composeChargenDetailedPage:
	// start from the message-free frame and draw the message directly with
	// the font, onto a copy, rather than calling the production compose path
	// a second time. AGENTS.md rule 6 requires a geometry witness to fail
	// when the production DRAW CALL is mutated, not only when
	// chargenMessageRect's own declaration moves; a region-only inside/outside
	// scan (the prior form of this test) still passes when the draw call
	// shifts by a few pixels within the rect's own margin, since the shifted
	// text still lands inside chargenMessageRect and nothing outside moves.
	// An exact pixel comparison against a frame the test drew itself does not
	// have that blind spot: a mutated x or y in drawChargenMessage moves
	// frameAfter's pixels away from want's, which the test computed on its
	// own call stack.
	want := image.NewRGBA(frameBefore.Bounds())
	copy(want.Pix, frameBefore.Pix)
	w, h := font.Measure(msg)
	x := chargenMessageRect.Min.X + (chargenMessageRect.Dx()-w)/2
	y := chargenMessageRect.Min.Y + (chargenMessageRect.Dy()-h)/2
	font.Draw(want, msg, x+1, y+1, shopShadowColor)
	font.Draw(want, msg, x, y, shopTextColor)

	b := frameAfter.Bounds()
	diff, firstBad := 0, image.Point{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if frameAfter.RGBAAt(x, y) != want.RGBAAt(x, y) {
				if diff == 0 {
					firstBad = image.Pt(x, y)
				}
				diff++
			}
		}
	}
	if diff != 0 {
		t.Fatalf("composed frame differs from the independently drawn message strip at %d pixel(s), first at %v", diff, firstBad)
	}

	// The message strip must be the only surface SetDetailMessage can reach:
	// no pixel outside chargenMessageRect may move between before and after.
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if (image.Point{X: x, Y: y}).In(chargenMessageRect) {
				continue
			}
			if frameBefore.RGBAAt(x, y) != frameAfter.RGBAAt(x, y) {
				t.Fatalf("SetDetailMessage changed pixel %v outside chargenMessageRect %v", image.Pt(x, y), chargenMessageRect)
			}
		}
	}
}

func TestChargenSourcePlacementsMasksAndFrames(t *testing.T) {
	art := &ChargenPresentation{
		Background: image.NewUniform(color.RGBA{R: 9, A: 255}),
		Plate:      image.NewRGBA(image.Rect(0, 0, 160, 238)),
		PreMask:    chargenMask(640, 480),
		Columns: [2]image.Image{
			image.NewUniform(color.RGBA{G: 41, A: 255}),
			image.NewUniform(color.RGBA{G: 73, A: 255}),
		},
		ColumnMask: [2]*image.Paletted{chargenMask(320, 480), chargenMask(320, 480)},
	}
	// Uniform images have no finite extent; use finite source columns and
	// patches, whose distinguishable pixels make duplicate/incorrect offsets
	// observable at the destination.
	art.Columns[0] = image.NewRGBA(image.Rect(0, 0, 320, 480))
	art.Columns[1] = image.NewRGBA(image.Rect(0, 0, 320, 480))
	for class := range art.Columns {
		column := art.Columns[class].(*image.RGBA)
		for y := 0; y < 480; y++ {
			for x := 0; x < 320; x++ {
				column.SetRGBA(x, y, color.RGBA{G: uint8(41 + 32*class), A: 255})
			}
		}
	}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			pic := image.NewRGBA(image.Rect(0, 0, 3, 3))
			pic.SetRGBA(1, 1, color.RGBA{R: uint8(50 + 10*choice + state), A: 255})
			art.Choices[choice][state] = pic
		}
		art.PreMask.SetColorIndex(preChoiceOrigin[choice].X+1, preChoiceOrigin[choice].Y+1, preMaskCode[choice])
	}
	art.PreMask.SetColorIndex(536, 133, preBackMaskCode)
	art.PreMask.SetColorIndex(463, 354, preForwardMaskCode)
	art.Amulet = image.NewRGBA(image.Rect(0, 0, 112, 204))
	art.Amulet.(*image.RGBA).SetRGBA(1, 1, color.RGBA{R: 191, A: 255})
	art.Forward = image.NewRGBA(image.Rect(0, 0, 100, 56))
	art.Forward.(*image.RGBA).SetRGBA(1, 1, color.RGBA{G: 181, A: 255})
	for class := range art.Skills {
		for skill := range art.Skills[class] {
			for state := range art.Skills[class][skill] {
				markX, width := 1, 3
				if class == 1 {
					// The card reaches x=239 only over the column's blank stone
					// margin. Model the decoded mage patch's first visible pixel
					// beyond that margin, while its rectangular black-key bounds
					// still begin at the native source origin.
					markX, width = 12, 14
				}
				pic := image.NewRGBA(image.Rect(0, 0, width, 3))
				pic.SetRGBA(markX, 1, color.RGBA{B: uint8(50 + 10*skill + state), A: 255})
				art.Skills[class][skill][state] = pic
			}
			at := detailedSkillOrigin[class][skill]
			markX := 1
			if class == 1 {
				markX = 12
			}
			art.ColumnMask[class].SetColorIndex(at.X+markX, at.Y+1, detailedMaskCode[class][skill])
		}
	}
	setup := ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art}, Detailed: &ChargenDetailed{},
		Choices: []ChargenChoice{{Options: []string{"m", "f"}, Parent: -1}, {Options: []string{"fighter", "mage"}, Parent: -1}, {Options: []string{"a", "b", "c", "d", "e"}, Parent: -1}},
		Stats:   []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
	}
	c := NewChargen(setup)
	pre := composeChargenPage(c, chargenNone, chargenNone)
	for choice, at := range preChoiceOrigin {
		// Only the opening chosen portrait draws, its on state (TOWN-521).
		want, placements := uint8(50+10*choice), 0
		if choice == 0 {
			placements = 1
			if got := pre.RGBAAt(at.X+1, at.Y+1).R; got != want {
				t.Errorf("pre choice %d native patch at %v = %d, want %d", choice, at, got, want)
			}
		}
		if got := countChargenPixel(pre, color.RGBA{R: want, A: 255}); got != placements {
			t.Errorf("pre choice %d sentinel appears %d times, want %d", choice, got, placements)
		}
		if got := preControlAt(c, image.Pt(at.X+1, at.Y+1)); got != chargenChoice0+chargenControl(choice) {
			t.Errorf("pre mask choice %d hit = %v", choice, got)
		}
	}
	if got := preControlAt(c, image.Pt(0, 0)); got != chargenNone {
		t.Errorf("pre unmasked pixel hit = %v", got)
	}
	if got := preControlAt(c, image.Pt(536, 133)); got != chargenBack {
		t.Errorf("pre brooch mask hit = %v, want Back", got)
	}
	if got := preControlAt(c, image.Pt(463, 354)); got != chargenForward {
		t.Errorf("pre book mask hit = %v, want Forward", got)
	}
	if got := pre.RGBAAt(529, 141); got.R != 9 {
		t.Errorf("idle brooch overlay pixel = %#v, want mainarea restored", got)
	}
	if got := composeChargenPage(c, chargenBack, chargenNone).RGBAAt(529, 141); got.R != 191 {
		t.Errorf("hover brooch overlay pixel = %#v, want source highlight", got)
	}
	if got := pre.RGBAAt(preForwardOrigin.X+1, preForwardOrigin.Y+1); got.R != 9 {
		t.Errorf("idle book overlay pixel = %#v, want mainarea restored", got)
	}
	if got := composeChargenPage(c, chargenForward, chargenNone).RGBAAt(preForwardOrigin.X+1, preForwardOrigin.Y+1); got.G != 181 {
		t.Errorf("hover book overlay pixel = %#v, want source highlight", got)
	}

	c.Forward()
	detail := composeChargenDetailedPage(c, chargenNone, chargenNone)
	if got := detail.RGBAAt(chargenColumnDestination.Min.X, 0).G; got != 41 {
		t.Fatalf("fighter column at destination x=%d = %d, want 41", chargenColumnDestination.Min.X, got)
	}
	fighterColumnFill := color.RGBA{G: 41, A: 255}
	for skill, at := range detailedSkillOrigin[0] {
		hot := image.Pt(chargenColumnOffset.X+at.X+1, at.Y+1)
		if skill == 0 { // draft's default pick: selected, not hovered -> state 0 (on.bmp, pressed dark)
			want := uint8(50 + 10*skill)
			if got := detail.RGBAAt(hot.X, hot.Y).B; got != want {
				t.Errorf("skill %d native patch at %v = %d, want %d", skill, at, got, want)
			}
			if got := countChargenPixel(detail, color.RGBA{B: want, A: 255}); got != 1 {
				t.Errorf("fighter skill %d sentinel appears %d times, want exactly one native placement", skill, got)
			}
		} else { // rest corner: not selected, not hovered -> no patch, the column's own fill shows through
			if got := detail.RGBAAt(hot.X, hot.Y); got != fighterColumnFill {
				t.Errorf("skill %d rest corner at %v = %#v, want column fill %#v (no patch drawn)", skill, at, got, fighterColumnFill)
			}
		}
		if got := detailedControlAt(c, hot); got != chargenSkill0+chargenControl(skill) {
			t.Errorf("skill %d mask hit = %v", skill, got)
		}
	}
	c.Back()
	c.SelectPreChoice(1)
	c.Forward()
	detail = composeChargenDetailedPage(c, chargenNone, chargenNone)
	if got := detail.RGBAAt(chargenColumnDestination.Min.X, 0).G; got != 73 {
		t.Fatalf("mage column at destination x=%d = %d, want 73", chargenColumnDestination.Min.X, got)
	}
	mageColumnFill := color.RGBA{G: 73, A: 255}
	for skill, at := range detailedSkillOrigin[1] {
		markX := 12
		hot := image.Pt(chargenColumnOffset.X+at.X+markX, at.Y+1)
		if skill == 0 { // draft's default pick: selected, not hovered -> state 0 (on.bmp, pressed dark)
			want := uint8(50 + 10*skill)
			if got := detail.RGBAAt(hot.X, hot.Y).B; got != want {
				t.Errorf("mage skill %d native patch at %v = %d, want %d", skill, at, got, want)
			}
		} else { // rest corner: not selected, not hovered -> no patch, the column's own fill shows through
			if got := detail.RGBAAt(hot.X, hot.Y); got != mageColumnFill {
				t.Errorf("mage skill %d rest corner at %v = %#v, want column fill %#v (no patch drawn)", skill, at, got, mageColumnFill)
			}
		}
		if got := detailedControlAt(c, hot); got != chargenSkill0+chargenControl(skill) {
			t.Errorf("mage skill %d mask hit = %v", skill, got)
		}
	}
	if got := detail.RGBAAt(chargenNavBox.Min.X, chargenNavBox.Min.Y); got != (color.RGBA{138, 116, 70, 255}) {
		t.Errorf("nav frame at %v = %#v, want framed surface", chargenNavBox.Min, got)
	}
	// chargenDollBox's own frame no longer draws (round-2 adversarial review
	// item 4, owner: remove the brown rectangle around the doll). Its corner
	// now shows the page's own background fill, the same near-black
	// constant the DollPane-absent fallback below it also uses.
	if got := detail.RGBAAt(chargenDollBox.Min.X, chargenDollBox.Min.Y); got != (color.RGBA{18, 18, 18, 255}) {
		t.Errorf("doll box corner at %v = %#v, want page background (no frame)", chargenDollBox.Min, got)
	}
	regions := []image.Rectangle{image.Rect(0, 0, 160, 240), chargenNavBox, chargenDollBox}
	for i := range regions {
		for j := 0; j < i; j++ {
			if !regions[i].Intersect(regions[j]).Empty() {
				t.Errorf("regions %d and %d overlap: %v", i, j, regions[i].Intersect(regions[j]))
			}
		}
	}
	for skill, at := range detailedSkillOrigin[1] {
		for _, px := range nonBlackChargenPixels(art.Skills[1][skill][0], at.Add(chargenColumnOffset)) {
			if px.In(chargenCardBox) {
				t.Errorf("card %v covers mage source non-black pixel %d at %v", chargenCardBox, skill, px)
			}
		}
		if image.Pt(chargenColumnOffset.X+at.X+12, at.Y+1).In(chargenCardBox) {
			t.Errorf("card %v covers mage source hot pixel %d", chargenCardBox, skill)
		}
	}
}
