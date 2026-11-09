package widgetkit

import (
	"image"
	"image/color"
	"testing"
)

func TestCheckMarks(t *testing.T) {
	for _, tc := range []struct {
		style   CheckStyle
		off, on string
	}{
		{CheckText, "[ ] ", "[x] "},
		{CheckRadioText, "( ) ", "(*) "},
		{CheckArt, "", ""},
	} {
		if got := (CheckSpec{Style: tc.style}).Mark(); got != tc.off {
			t.Errorf("style %d off mark = %q, want %q", tc.style, got, tc.off)
		}
		if got := (CheckSpec{On: true, Style: tc.style}).Mark(); got != tc.on {
			t.Errorf("style %d on mark = %q, want %q", tc.style, got, tc.on)
		}
	}
}

func TestDrawPanelFillsAndBorders(t *testing.T) {
	fill, border := color.RGBA{10, 20, 30, 255}, color.RGBA{200, 100, 50, 255}
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	DrawPanel(img, image.Rect(1, 1, 7, 7), fill, border)
	if got := img.RGBAAt(0, 0); got != (color.RGBA{}) {
		t.Errorf("outside the panel = %v", got)
	}
	for _, p := range []image.Point{{1, 1}, {6, 1}, {1, 6}, {6, 6}, {3, 1}, {1, 3}} {
		if got := img.RGBAAt(p.X, p.Y); got != border {
			t.Errorf("border %v = %v, want %v", p, got, border)
		}
	}
	if got := img.RGBAAt(3, 3); got != fill {
		t.Errorf("fill = %v, want %v", got, fill)
	}
}

func TestDrawFlatFrameOneSideAndWidth(t *testing.T) {
	border := color.RGBA{200, 100, 50, 255}
	img := image.NewRGBA(image.Rect(0, 0, 6, 6))
	DrawFlatFrame(img, img.Bounds(), FlatFrame{Border: border, Width: 2, Sides: EdgeTop, PaintBorder: true})
	if got := img.RGBAAt(2, 1); got != border {
		t.Errorf("top strip = %v", got)
	}
	if got := img.RGBAAt(2, 2); got != (color.RGBA{}) {
		t.Errorf("below the strip = %v", got)
	}
	if got := img.RGBAAt(0, 4); got != (color.RGBA{}) {
		t.Errorf("left edge = %v", got)
	}
}

func TestWheelStep(t *testing.T) {
	for _, tc := range []struct {
		w     float64
		step  int
		whole bool
		want  int
	}{
		{1, 3, false, -3},
		{-1, 3, false, 3},
		{0, 3, false, 0},
		{0.4, 3, true, 0},
		{2, 3, true, -6},
		{-1, 1, true, 1},
	} {
		if got := WheelStep(tc.w, tc.step, tc.whole); got != tc.want {
			t.Errorf("WheelStep(%v,%d,%v) = %d, want %d", tc.w, tc.step, tc.whole, got, tc.want)
		}
	}
	if ListWheelLines(1) != -1 || ListWheelLines(-1) != 1 || ListWheelLines(0) != 0 {
		t.Error("ListWheelLines is one line a notch, away from the user towards the first line")
	}
}

func TestEditTextAppendsAndBackspaces(t *testing.T) {
	rules := TextInputRules{AppendOnly: true, SplitLineBreaks: true}
	if got := EditText(TextInputState{Text: "ab"}, TextInputKeys{Typed: "c"}, rules).Text; got != "abc" {
		t.Errorf("typed = %q", got)
	}
	if got := EditText(TextInputState{Text: "ab"}, TextInputKeys{Backspace: true}, rules).Text; got != "a" {
		t.Errorf("backspace = %q", got)
	}
}

func TestTextInputStateMoveSelectReplace(t *testing.T) {
	var m TextInputState
	m.Reset("one two")
	m.Move("left", true, true)
	if got := m.Selected(); got != "two" {
		t.Fatalf("Ctrl+Shift+Left selection = %q", got)
	}
	m.Replace("три")
	if m.Text != "one три" {
		t.Fatalf("after replace = %q", m.Text)
	}
	m.SelectAll()
	m.Erase(true)
	if m.Text != "" {
		t.Fatalf("after erase = %q", m.Text)
	}
}

func TestClipboardMemoryRoundTrip(t *testing.T) {
	saved := TextClipboard
	TextClipboard = NewTextClipboard(true)
	t.Cleanup(func() { TextClipboard = saved })
	if err := TextClipboard.Write("Привет"); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadTextClipboard(); err != nil || got != "Привет" {
		t.Fatal(got, err)
	}
}
