package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"

	"againrom/pkg/render/text"
)

// widgetKitFont paints every glyph as a solid 3x5 block, so a label's ink
// lands on known pixels.
func widgetKitFont() *text.Font {
	font := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range font.Glyphs {
		g := text.Glyph{Width: 3, Height: 5, Pixels: make([]text.Pixel, 15), Advance: 3}
		for i := range g.Pixels {
			g.Pixels[i] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		font.Glyphs[k] = g
	}
	return font
}

// inkAt is the first pixel font paints for s drawn at (x, y) in ink, the
// independent reference a builder's label is compared against.
func inkAt(t *testing.T, f *text.Font, s string, x, y int, ink color.RGBA) (image.Point, color.RGBA) {
	t.Helper()
	ref := image.NewRGBA(image.Rect(0, 0, 640, 480))
	f.Draw(ref, s, x, y, ink)
	for py := 0; py < 480; py++ {
		for px := 0; px < 640; px++ {
			if c := ref.RGBAAt(px, py); c.A != 0 {
				return image.Pt(px, py), c
			}
		}
	}
	t.Fatal("reference label painted nothing")
	return image.Point{}, color.RGBA{}
}

func solidCanvas(c color.RGBA) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 320, 240))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return dst
}

// TestHSliderValueMapping checks MENU-118 on a 240x24 slider: the knob's hit
// left K=L+H-4+trunc(p*(W-2H-4)/N), the pointer map
// clamp(trunc(N*(x-L-H-2)/(W-2H-4)),0,N) and the step max(trunc(N/16),1),
// for the speed range 9 and the volume range 5000.
func TestHSliderValueMapping(t *testing.T) {
	r := image.Rect(10, 10, 250, 34)
	for _, tc := range []struct{ max, step int }{{9, 1}, {5000, 312}} {
		s := hSlider{Rect: r, Max: tc.max}
		if s.step() != tc.step {
			t.Fatalf("N=%d step %d, want %d", tc.max, s.step(), tc.step)
		}
		for p := 0; p <= tc.max; p += max(tc.max/9, 1) {
			s.Pos = p
			if want := 10 + 24 - 4 + p*188/tc.max; s.Knob().Min.X != want || s.Knob().Dx() != 16 {
				t.Fatalf("N=%d p=%d knob %v, want left %d width 16", tc.max, p, s.Knob(), want)
			}
		}
		for _, x := range []int{0, 36, 37, 100, 223, 224, 400} {
			want := min(max(tc.max*(x-36)/188, 0), tc.max)
			if got := s.posAt(x); got != want {
				t.Fatalf("N=%d posAt(%d)=%d, want %d", tc.max, x, got, want)
			}
		}
		if s.posAt(36) != 0 || s.posAt(224) != tc.max {
			t.Fatalf("N=%d: the track ends do not reach 0 and N", tc.max)
		}
	}
}

// TestHSliderPaintsCapsTrackAndKnob checks the part frames: left cap 0 (hot
// 3), right cap 8 (hot 11), track 7 and knob 10 at (K+1,T).
func TestHSliderPaintsCapsTrackAndKnob(t *testing.T) {
	frames := widgetTestFrames()
	r := image.Rect(10, 10, 250, 34)
	for _, hot := range []bool{false, true} {
		dst := solidCanvas(color.RGBA{120, 140, 160, 255})
		drawHSlider(dst, frames, hSlider{Rect: r, Pos: 0, Max: 9, LeftHot: hot, RightHot: hot})
		left, right := sliderLeftFrame, sliderRightFrame
		if hot {
			left, right = sliderLeftHotFrame, sliderRightHotFrame
		}
		for _, c := range []struct {
			at    image.Point
			frame int
		}{{image.Pt(10, 10), left}, {image.Pt(226, 10), right}, {image.Pt(31, 10), sliderKnobFrame}, {image.Pt(58, 10), sliderTrackFrame}} {
			if got, want := dst.RGBAAt(c.at.X, c.at.Y), frames[c.frame].RGBAAt(0, 0); got != want {
				t.Fatalf("hot=%v %v = %v, want frame %d %v", hot, c.at, got, c.frame, want)
			}
		}
	}
}

// TestSliderInputEndcapsTrackAndDrag drives sliderInput: an endcap press
// steps once, a track press maps x, and a press on the moved knob follows the
// pointer until release.
func TestSliderInputEndcapsTrackAndDrag(t *testing.T) {
	s := hSlider{Rect: image.Rect(10, 10, 250, 34), Pos: 4, Max: 9}
	var in sliderInput
	press := appInput{PrimaryPressed: true}
	press.Viewer.PrimaryDown = true
	if pos, ok := in.step(s, image.Pt(12, 20), true, press); !ok || pos != 3 {
		t.Fatalf("left endcap gave %d %v, want 3", pos, ok)
	}
	in = sliderInput{}
	if pos, ok := in.step(s, image.Pt(248, 20), true, press); !ok || pos != 5 {
		t.Fatalf("right endcap gave %d %v, want 5", pos, ok)
	}
	in = sliderInput{}
	pos, ok := in.step(s, image.Pt(224, 20), true, press)
	if !ok || pos != 9 || !in.active() {
		t.Fatalf("track press gave %d %v drag %v, want 9 and a grabbed knob", pos, ok, in.active())
	}
	s.Pos = pos
	held := appInput{CursorX: 1}
	held.Viewer.PrimaryDown = true
	if pos, ok := in.step(s, image.Pt(36, 20), true, held); !ok || pos != 0 {
		t.Fatalf("drag gave %d %v, want 0", pos, ok)
	}
	if _, ok := in.step(s, image.Pt(36, 20), true, appInput{PrimaryReleased: true}); ok || in.active() {
		t.Fatal("release kept the drag")
	}
}

func choiceTestArt(c uint8) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 20; x++ {
			pic.SetRGBA(x, y, color.RGBA{c, uint8(x * 9), uint8(y * 9), 255})
		}
	}
	return pic
}

// TestChoiceGroupRadioPixels checks MENU-123: frames 0/1 at a 24-pixel pitch
// with the body at (L+1,T+24i), the level-4 shadow at (+4,+4), the label at
// (L+30,T+24i+5) in the selected row's focus ramp, and the disabled level-3
// remap over the rectangle grown by one.
func TestChoiceGroupRadioPixels(t *testing.T) {
	bg := color.RGBA{120, 140, 160, 255}
	off, on := choiceTestArt(40), choiceTestArt(200)
	f := widgetKitFont()
	r := image.Rect(20, 20, 200, 92)
	g := choiceGroup{Kind: choiceRadio, Rect: r, Labels: []string{"A", "B", "C"}, Selected: 1, Focus: true, Off: off, On: on}
	dst := solidCanvas(bg)
	drawChoiceGroup(dst, f, g)
	if dst.RGBAAt(21, 20) != off.RGBAAt(0, 0) || dst.RGBAAt(21, 44) != on.RGBAAt(0, 0) || dst.RGBAAt(21, 68) != off.RGBAAt(0, 0) {
		t.Fatal("radio frames are not at (L+1,T+24i) with the selected row on")
	}
	if got, want := dst.RGBAAt(41, 24), mustLevel(t, widgetShadowLevel).Color(bg); got != want {
		t.Fatalf("shadow pixel %v, want %v", got, want)
	}
	for i, ink := range []color.RGBA{choiceIdleInk, choiceHotInk, choiceIdleInk} {
		at, c := inkAt(t, f, "A", 50, 20+24*i+5, ink)
		if dst.RGBAAt(at.X, at.Y) != c {
			t.Fatalf("row %d label is not at (L+30,T+24i+5) in %v", i, ink)
		}
	}
	if row, ok := g.RowAt(image.Pt(30, 70)); !ok || row != 2 {
		t.Fatalf("RowAt gave %d %v, want 2", row, ok)
	}
	if _, ok := g.RowAt(image.Pt(30, 95)); ok {
		t.Fatal("RowAt answered outside the group")
	}
	g.Disabled = true
	dst = solidCanvas(bg)
	drawChoiceGroup(dst, f, g)
	if got, want := dst.RGBAAt(19, 19), mustLevel(t, 3).Color(bg); got != want {
		t.Fatalf("disabled corner %v, want %v", got, want)
	}
}

// TestChoiceGroupChecksAndTipChecks checks MENU-124 and MENU-125: a
// standard checkbox's mask at the 24-pixel pitch, and the tip checkbox's
// 16-pixel pitch with its label at (L+22,T+3) in the pointer's ramp.
func TestChoiceGroupChecksAndTipChecks(t *testing.T) {
	bg := color.RGBA{120, 140, 160, 255}
	off, on := choiceTestArt(40), choiceTestArt(200)
	f := widgetKitFont()
	dst := solidCanvas(bg)
	calls := recordWidgets(t, func() {
		drawChoiceGroup(dst, f, choiceGroup{Kind: choiceCheck, Rect: image.Rect(20, 20, 200, 68), Labels: []string{"A", "B"},
			Mask: 2, Off: off, On: on})
	})
	if len(calls) != 1 || calls[0].kind != widgetCheck {
		t.Fatalf("check group recorded %v", calls)
	}
	if dst.RGBAAt(21, 20) != off.RGBAAt(0, 0) || dst.RGBAAt(21, 44) != on.RGBAAt(0, 0) {
		t.Fatal("checkbox frames do not follow the mask")
	}
	dst = solidCanvas(bg)
	tip := choiceGroup{Kind: choiceTipCheck, Rect: image.Rect(20, 20, 200, 52), Labels: []string{"A", "B"},
		Pointer: image.Pt(100, 40), PointerOK: true, Off: off, On: on}
	drawChoiceGroup(dst, f, tip)
	for i, ink := range []color.RGBA{choiceIdleInk, choiceHotInk} {
		at, c := inkAt(t, f, "A", 42, 20+16*i+3, ink)
		if dst.RGBAAt(at.X, at.Y) != c {
			t.Fatalf("tip row %d label is not at (L+22,T+16i+3) in %v", i, ink)
		}
	}
}

// TestEditFieldPixels checks MENU-126: the four bevel lines, the level-12
// selection remap, and the two-pixel white caret only while focused in the
// visible phase.
func TestEditFieldPixels(t *testing.T) {
	bg := color.RGBA{60, 70, 80, 255}
	r := image.Rect(10, 10, 110, 34)
	for _, tc := range []struct{ focus, phase, caret bool }{{true, true, true}, {true, false, false}, {false, true, false}} {
		dst := solidCanvas(bg)
		drawEditField(dst, editField{Rect: r, SelFrom: 0, SelTo: 10, Caret: 20, TextH: 8, Focus: tc.focus, Phase: tc.phase}, nil)
		if dst.RGBAAt(11, 10) != editShade || dst.RGBAAt(10, 11) != editShade || dst.RGBAAt(11, 33) != editLight || dst.RGBAAt(109, 11) != editLight {
			t.Fatal("bevel lines are not the requested colours")
		}
		if dst.RGBAAt(10, 10) != bg || dst.RGBAAt(60, 20) != bg {
			t.Fatal("the field filled its corner or interior")
		}
		if got, want := dst.RGBAAt(16, 20), mustLevel(t, editSelectionLevel).Color(bg); got != want {
			t.Fatalf("selection pixel %v, want %v", got, want)
		}
		if caret := dst.RGBAAt(34, 20) == editCaret && dst.RGBAAt(35, 20) == editCaret; caret != tc.caret {
			t.Fatalf("focus %v phase %v: caret %v", tc.focus, tc.phase, caret)
		}
	}
	if got := (editField{Rect: r, TextH: 8}).TextOrigin(); got != image.Pt(14, 17) {
		t.Fatalf("text origin %v", got)
	}
}

// TestCaretBlinkHalfSecond checks the 500 ms toggle and the reset to visible.
func TestCaretBlinkHalfSecond(t *testing.T) {
	t0 := time.Unix(100, 0)
	var c caretBlink
	c.reset(t0)
	for _, step := range []struct {
		at time.Duration
		on bool
	}{{400 * time.Millisecond, true}, {501 * time.Millisecond, false}, {900 * time.Millisecond, false}, {1002 * time.Millisecond, true}} {
		c.tick(t0.Add(step.at))
		if c.on() != step.on {
			t.Fatalf("at %v on=%v, want %v", step.at, c.on(), step.on)
		}
	}
	c.reset(t0.Add(2 * time.Second))
	if !c.on() {
		t.Fatal("reset left the caret hidden")
	}
}

// TestHoverBoxPixels checks MENU-128: size W+11 by 14n+5, the fill, both
// bevel colours on each side, Ball.bmp at the four corners and the label at
// (L+5,T+4).
func TestHoverBoxPixels(t *testing.T) {
	f := widgetKitFont()
	ball := image.NewRGBA(image.Rect(0, 0, 4, 4))
	red := color.RGBA{255, 0, 0, 255}
	draw.Draw(ball, ball.Bounds(), &image.Uniform{C: red}, image.Point{}, draw.Src)
	pic := composeHoverBox([]string{"AB", "C"}, f, ball)
	w, _ := f.Measure("AB")
	if pic.Bounds().Size() != image.Pt(max(w, f.Advance("AB"))+11, 2*14+5) {
		t.Fatalf("size %v", pic.Bounds().Size())
	}
	r, b := pic.Bounds().Dx()-1, pic.Bounds().Dy()
	for _, c := range []struct {
		at   image.Point
		want color.RGBA
	}{
		{image.Pt(4, 1), hoverLight}, {image.Pt(4, 2), hoverDark}, {image.Pt(4, b-3), hoverLight}, {image.Pt(4, b-2), hoverDark},
		{image.Pt(1, 5), hoverLight}, {image.Pt(2, 5), hoverDark}, {image.Pt(r-2, 5), hoverLight}, {image.Pt(r-1, 5), hoverDark},
		{image.Pt(4, 4), hoverFill}, {image.Pt(0, 0), red}, {image.Pt(r, 0), red}, {image.Pt(0, b-1), red}, {image.Pt(r, b-1), red},
		{image.Pt(r, 8), color.RGBA{}},
	} {
		if got := pic.RGBAAt(c.at.X, c.at.Y); got != c.want {
			t.Fatalf("%v = %v, want %v", c.at, got, c.want)
		}
	}
	at, ink := inkAt(t, f, "C", 5, 4+14, popupTextColor)
	if pic.RGBAAt(at.X, at.Y) != ink {
		t.Fatal("second line is not at (L+5,T+18)")
	}
}
