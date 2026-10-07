package ui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// The dialogue panel as `DLG-PANEL-035`, `DLG-PORTRAIT-036`, `DLG-RECT-037`,
// `DLG-LINE-038` and `DLG-BUTTON-039` state it at 640x480. Every fixture here is
// built in this file and no assertion reads an install. Coordinates are
// panel-relative: the panel's origin is (76,124) on the screen, so the text
// control's (204,160) is (128,36) here.
//
// Nothing below names a symbol added with the dialogue style, so the same tests
// compile against the notice as it was drawn before, and fail there.

const (
	// claimTextLeft and claimTextTop are the text control's left and top beside
	// the portrait, claimTextWidth its width, claimIndent the first line's indent
	// and claimPitch the distance between line tops.
	claimTextLeft, claimTextTop = 128, 36
	claimTextWidth              = 300
	claimIndent                 = 10
	claimPitch                  = 17
)

// dialogueClaimFont is a 224-record font of solid letters: a letter is a 4x15 bar
// followed by one blank column, so the ink of a word is a row of bars whose gaps
// are one column wide, and the gap between two words is wider. Its height is 15,
// which with the layout's extra 2 is the line pitch of 17 the claims state.
func dialogueClaimFont() *text.Font {
	f := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 4, Height: 15, Advance: 4, Pixels: make([]text.Pixel, 4*15)}
		if k == 0 {
			g.Advance = 3
		} else {
			for i := range g.Pixels {
				g.Pixels[i] = text.Pixel{Level: text.MaxLevel, Painted: true}
			}
		}
		f.Glyphs[k] = g
	}
	return f
}

// dialogueClaimLayout is the authored dialogue layout resolved for a speaker,
// with every colour that is not text or the shadow blacked out, so a pixel of
// full white is text and a pixel of (8,8,8) is its shadow.
func dialogueClaimLayout() NoticeLayout {
	l := AuthoredDialogueLayout().WithPortrait(true)
	black := color.RGBA{A: 0xff}
	white := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	l.Fill, l.Border, l.ButtonFill, l.ButtonBorder = black, black, black, black
	l.PortraitFill, l.PortraitBorder = black, black
	l.TextColor, l.ButtonText = white, white
	return l
}

// dialogueClaimPiece is the colour of frame piece i in dialogueClaimFrame.
func dialogueClaimPiece(i int) color.RGBA {
	return color.RGBA{R: uint8(30 + 20*i), G: uint8(200 - 10*i), B: uint8(90 + 5*i), A: 0xff}
}

// dialogueClaimFrame is nine solid frame pieces of the sizes `DLG-PANEL-035`
// measures for the shipped `lm.256`: 96x64, 48x48, 96x48, 48x48, 48x64, 48x64,
// 48x48, 96x48, 48x48.
func dialogueClaimFrame() *DialogFrame {
	sizes := [9]image.Point{{96, 64}, {48, 48}, {96, 48}, {48, 48}, {48, 64}, {48, 64}, {48, 48}, {96, 48}, {48, 48}}
	art := &DialogFrame{}
	for i, s := range sizes {
		p := image.NewRGBA(image.Rectangle{Max: s})
		draw.Draw(p, p.Bounds(), &image.Uniform{C: dialogueClaimPiece(i)}, image.Point{}, draw.Src)
		art.Pieces[i] = p
	}
	return art
}

func isFullWhite(c color.RGBA) bool { return c.R == 255 && c.G == 255 && c.B == 255 }

// inkRuns are the words drawn on row y between x0 and x1, as first and last
// white column: white columns within two of each other belong to one word.
func inkRuns(img *image.RGBA, y, x0, x1 int) [][2]int {
	var runs [][2]int
	last := -10
	for x := x0; x < x1; x++ {
		if !isFullWhite(img.RGBAAt(x, y)) {
			continue
		}
		if x-last > 2 {
			runs = append(runs, [2]int{x, x})
		} else {
			runs[len(runs)-1][1] = x
		}
		last = x
	}
	return runs
}

// whiteIn reports whether any pixel of img in rows [y0,y1) between x0 and x1 is
// full white.
func whiteIn(img *image.RGBA, x0, x1, y0, y1 int) bool {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if isFullWhite(img.RGBAAt(x, y)) {
				return true
			}
		}
	}
	return false
}

func TestDialogueLayoutRectanglesAreThePanelClaims(t *testing.T) {
	l := AuthoredDialogueLayout()
	for _, c := range []struct {
		name      string
		got, want image.Rectangle
	}{
		{"panel drawn (76,124)-(564,356)", l.Box, image.Rect(76, 124, 564, 356)},
		{"text control, no portrait (124,160)-(504,295)", l.Text, image.Rect(48, 36, 428, 171)},
		{"text control beside the portrait (204,160)-(504,295)", l.TextBesidePortrait, image.Rect(128, 36, 428, 171)},
		{"portrait child (106,178)-(194,292)", l.Portrait, image.Rect(30, 54, 118, 168)},
		{"button (276,296)-(356,322)", l.Button, image.Rect(200, 172, 280, 198)},
	} {
		if c.got != c.want {
			t.Errorf("%s: panel-relative %v, want %v", c.name, c.got, c.want)
		}
	}
	if got, want := l.WithPortrait(true).Text, image.Rect(128, 36, 428, 171); got != want {
		t.Errorf("text control of a panel with a portrait = %v, want %v", got, want)
	}
	if got := l.Text.Dy() / (dialogueClaimFont().Height() + l.Pitch); got != 7 {
		t.Errorf("the control holds %d lines of the shipped font's pitch, want 7", got)
	}
}

// A paragraph's lines are placed as `DLG-LINE-038` states: line tops 17 apart from
// the control's top, the first line of a paragraph indented 10 and justified over
// the width less the indent, every other line but a paragraph's last justified
// over the full width, and a paragraph's last line and a one-word line left
// aligned at the control's left with the font's own space between words.
func TestDialogueTextIsJustifiedAndIndentedAsTheClaimStates(t *testing.T) {
	f := dialogueClaimFont()
	word := strings.Repeat("a", 16)
	words := func(n int) string { return strings.TrimSpace(strings.Repeat(word+" ", n)) }
	img := RenderNotice(dialogueClaimLayout(), f, words(8)+"\r\n"+words(4), nil)
	if img == nil {
		t.Fatal("RenderNotice composed nothing")
	}
	w, space := f.Advance(word), f.Advance(" ")
	if w != 80 || space != 11 {
		t.Fatalf("test font advances: word %d space %d, want 80 and 11", w, space)
	}
	// Each word is 80 wide and each line holds three, so the gaps are whole
	// numbers: (290-240)/2 = 25 on an indented line and (300-240)/2 = 30 on the
	// others.
	x0, indented := claimTextLeft, claimTextLeft+claimIndent
	for i, want := range [][]int{
		{indented, indented + w + 25, indented + 2*(w+25)},
		{x0, x0 + w + 30, x0 + 2*(w+30)},
		{x0, x0 + w + space},
		{indented, indented + w + 25, indented + 2*(w+25)},
		{x0},
	} {
		top := claimTextTop + claimPitch*i
		var got []int
		for _, run := range inkRuns(img, top+7, 120, 470) {
			got = append(got, run[0])
			if run[1]-run[0] != w-2 {
				t.Errorf("line %d: a word's ink spans %d columns, want %d", i, run[1]-run[0]+1, w-1)
			}
		}
		if len(got) != len(want) {
			t.Errorf("line %d: words start at %v, want %v", i, got, want)
			continue
		}
		for k := range want {
			if got[k] != want[k] {
				t.Errorf("line %d: words start at %v, want %v", i, got, want)
				break
			}
		}
		if whiteIn(img, 120, 470, top+15, top+claimPitch) {
			t.Errorf("line %d: ink below the line's 15 rows", i)
		}
	}
	// The last justified word of a line ends where the control does.
	if got, want := inkRuns(img, claimTextTop+7, 120, 470)[2][1]+2, claimTextLeft+claimTextWidth; got != want {
		t.Errorf("line 0: last word's pen ends at %d, want the control's right edge %d", got, want)
	}
	if whiteIn(img, 120, 470, claimTextTop+claimPitch*5, claimTextTop+claimPitch*8) {
		t.Error("ink below the fifth line, which the text does not have")
	}
}

// The text control shows min(7, lines) lines: 135 rows at a pitch of 17.
func TestDialogueTextShowsAtMostSevenLines(t *testing.T) {
	img := RenderNotice(dialogueClaimLayout(), dialogueClaimFont(), strings.Repeat("aaaa\r\n", 9), nil)
	for i := 0; i < 9; i++ {
		top := claimTextTop + claimPitch*i
		if got, want := whiteIn(img, 120, 470, top+7, top+8), i < 7; got != want {
			t.Errorf("line %d drawn = %v, want %v: the control shows seven lines of nine", i, got, want)
		}
	}
}

// Every run of text is drawn twice: a flat (8,8,8) shadow one pixel right and
// down, then the ink over it.
func TestDialogueTextCarriesAFlatShadowOnePixelDownAndRight(t *testing.T) {
	word := strings.Repeat("a", 16)
	img := RenderNotice(dialogueClaimLayout(), dialogueClaimFont(), strings.Repeat(word+" ", 8), nil)
	// Line 1 starts at (128,53): its first letter's ink is x 128..131, y 53..67.
	shadow, black, white := color.RGBA{8, 8, 8, 255}, color.RGBA{A: 255}, color.RGBA{255, 255, 255, 255}
	for _, c := range []struct {
		at   image.Point
		want color.RGBA
		why  string
	}{
		{image.Pt(128, 53), white, "ink at the run's origin"},
		{image.Pt(129, 54), white, "the ink covers the shadow it overlaps"},
		{image.Pt(132, 60), shadow, "the shadow's right column, one past the letter"},
		{image.Pt(130, 68), shadow, "the shadow's bottom row, one below the letter"},
		{image.Pt(127, 60), black, "nothing left of the run"},
		{image.Pt(128, 52), black, "nothing above the run"},
	} {
		if got := img.RGBAAt(c.at.X, c.at.Y); got != c.want {
			t.Errorf("pixel %v = %v, want %v: %s", c.at, got, c.want, c.why)
		}
	}
}

// The panel is a nine-piece frame over a 480x224 body with an 8 px shadow band
// right of and below it: corners in place, four tiles of the wide edges, two of
// the tall ones, and the interior in 4 by 2 tiles from (48,48).
func TestDialoguePanelIsTiledFromNinePiecesWithAShadowBand(t *testing.T) {
	art := dialogueClaimFrame()
	l := dialogueClaimLayout()
	l.Frame = art
	l.Portrait, l.Button = image.Rectangle{}, image.Rectangle{}
	img := RenderNotice(l, dialogueClaimFont(), "", nil)
	if img == nil {
		t.Fatal("RenderNotice composed nothing")
	}
	if got, want := img.Bounds().Size(), image.Pt(488, 232); got != want {
		t.Fatalf("panel picture is %v, want %v: the 480x224 body and the 8 px band", got, want)
	}
	tile := func(piece int, at image.Point) {
		t.Helper()
		size := art.Pieces[piece].Bounds().Size()
		want := dialogueClaimPiece(piece)
		for _, p := range []image.Point{at, at.Add(size).Sub(image.Pt(1, 1))} {
			if got := img.RGBAAt(p.X, p.Y); got != want {
				t.Errorf("piece %d tile at %v: pixel %v = %v, want %v", piece, at, p, got, want)
			}
		}
	}
	tile(1, image.Pt(0, 0))
	tile(3, image.Pt(480-48, 0))
	tile(6, image.Pt(0, 224-48))
	tile(8, image.Pt(480-48, 224-48))
	for i := 0; i < 4; i++ {
		tile(2, image.Pt(48+96*i, 0))
		tile(7, image.Pt(48+96*i, 224-48))
	}
	for j := 0; j < 2; j++ {
		tile(4, image.Pt(0, 48+64*j))
		tile(5, image.Pt(480-48, 48+64*j))
		for i := 0; i < 4; i++ {
			tile(0, image.Pt(48+96*i, 48+64*j))
		}
	}
	for y := 0; y < 224; y++ {
		for x := 0; x < 480; x++ {
			if img.RGBAAt(x, y).A != 0xff {
				t.Fatalf("body pixel (%d,%d) is not covered by a piece: %v", x, y, img.RGBAAt(x, y))
			}
		}
	}
	// Packed shadows operate on the reached scene, not this transparent image.
	shade := func(x, y int) {
		t.Helper()
		c := img.RGBAAt(x, y)
		if c != (color.RGBA{}) {
			t.Errorf("standalone band pixel (%d,%d) = %v, want clear", x, y, c)
		}
	}
	for _, p := range []image.Point{{480, 8}, {487, 8}, {483, 100}, {487, 231}, {8, 224}, {8, 231}, {300, 227}, {440, 231}} {
		shade(p.X, p.Y)
	}
	for _, p := range []image.Point{{483, 4}, {487, 7}, {4, 227}, {7, 231}} {
		shade(p.X, p.Y)
	}
}

// The button has no art and no fill: a light bevel on its top and left and a dark
// one on its bottom and right, drawn over whatever the panel shows there.
func TestDialogueButtonIsABevelWithNoFill(t *testing.T) {
	l := dialogueClaimLayout()
	blue, red := color.RGBA{B: 200, A: 255}, color.RGBA{R: 255, A: 255}
	l.Fill, l.Border = blue, blue
	l.ButtonFill, l.ButtonBorder = red, red
	l.Portrait = image.Rectangle{}
	l.ButtonLabel = ""
	img := RenderNotice(l, dialogueClaimFont(), "", nil)
	light, dark := color.RGBA{41, 68, 57, 255}, color.RGBA{0, 12, 8, 255}
	// The button is (200,172)-(280,198): with R' = 279 and B' = 197 the dark
	// lines are x = 279 for y 174..195, x = 278 for y 173..196, y = 197 for
	// x 202..277 and y = 196 for x 201..278, and the pixel (277,195); the light
	// ones are y = 172 for x 202..277, x = 200 for y 174..195, and (201,173).
	for _, c := range []struct {
		at   image.Point
		want color.RGBA
	}{
		{image.Pt(240, 185), blue},
		{image.Pt(200, 172), blue}, {image.Pt(201, 172), blue}, {image.Pt(200, 173), blue},
		{image.Pt(279, 172), blue}, {image.Pt(279, 173), blue}, {image.Pt(279, 196), blue}, {image.Pt(278, 197), blue},
		{image.Pt(202, 172), light}, {image.Pt(277, 172), light}, {image.Pt(200, 174), light}, {image.Pt(200, 195), light}, {image.Pt(201, 173), light},
		{image.Pt(279, 174), dark}, {image.Pt(279, 195), dark}, {image.Pt(278, 173), dark}, {image.Pt(278, 196), dark},
		{image.Pt(202, 197), dark}, {image.Pt(277, 197), dark}, {image.Pt(201, 196), dark}, {image.Pt(277, 195), dark},
	} {
		if got := img.RGBAAt(c.at.X, c.at.Y); got != c.want {
			t.Errorf("button pixel %v = %v, want %v", c.at, got, c.want)
		}
	}
}

// The selected engine canvas policy uses top-down physical source rows and
// descends both the window copy and final surface copy. Native exposure is
// Unknown; the final rows below identify this supplied-canvas policy only.
func TestDialoguePortraitSurfaceIsBottomUpOverABlackFill(t *testing.T) {
	art := dialogueClaimFrame()
	green, magenta := color.RGBA{G: 198, A: 255}, color.RGBA{R: 197, B: 197, A: 255}
	back := image.NewRGBA(image.Rect(0, 0, 160, 240))
	draw.Draw(back, back.Bounds(), &image.Uniform{C: green}, image.Point{}, draw.Src)
	art.Portrait = image.NewRGBA(image.Rect(0, 0, 88, 108))
	art.Portrait.SetRGBA(0, 0, magenta)
	l := dialogueClaimLayout()
	l.Frame = art
	l.Button = image.Rectangle{}
	f := dialogueClaimFont()
	origin := image.Pt(30, 54)
	at := func(img *image.RGBA, x, y int) color.RGBA { return img.RGBAAt(origin.X+x, origin.Y+y) }

	art.PortraitBack = back
	for _, w := range []struct {
		name string
		l    NoticeLayout
		top  int
	}{
		{"96 rows", l.WithFaceWindow(image.Rect(36, 0, 108, 96)), 5},
		{"92 rows", l, 9},
	} {
		img := RenderNotice(w.l, f, "", nil)
		for _, c := range []struct {
			x, y int
			want bool
		}{
			{8, w.top, true}, {79, w.top, true}, {8, 100, true}, {79, 100, true},
			{8, w.top - 1, false}, {7, w.top, false}, {80, w.top, false}, {8, 101, false},
		} {
			if got := at(img, c.x, c.y) == green; got != c.want {
				t.Errorf("%s: surface (%d,%d) is the backdrop = %v, want %v: the window covers rows %d..100", w.name, c.x, c.y, got, c.want, w.top)
			}
		}
		if got := at(img, 0, 107); got != magenta {
			t.Errorf("%s: the border's first stored pixel shows %v at the surface's bottom-left, want %v", w.name, got, magenta)
		}
		if got := at(img, 0, 0); got == magenta {
			t.Errorf("%s: the border's first stored pixel is at the surface's top-left: its rows are counted from the top", w.name)
		}
	}

	art.PortraitBack = nil
	img := RenderNotice(l, f, "", nil)
	black := color.RGBA{A: 255}
	for _, c := range []struct {
		x, y int
		want bool
	}{{8, 7, true}, {79, 7, true}, {8, 100, true}, {79, 100, true}, {8, 6, false}, {7, 7, false}, {80, 7, false}, {8, 101, false}} {
		if got := at(img, c.x, c.y) == black; got != c.want {
			t.Errorf("surface (%d,%d) is the black fill = %v, want %v: the fill is 72x94 at (8,7)", c.x, c.y, got, c.want)
		}
	}
}
