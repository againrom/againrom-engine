package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/render/text"
)

// solidFont15 is 224 records of fully painted 5x15 cells advancing 5 with no
// spacing: the height of font 1, so the panel geometry is the claimed one.
func solidFont15() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 5, Height: 15, Pixels: make([]text.Pixel, 75), Advance: 5}
		for i := range g.Pixels {
			g.Pixels[i] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

func helpOpenApp(t *testing.T) (*App, *Viewer, time.Time) {
	t.Helper()
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 60)
	a.flow.viewer.SetFont(solidFont15())
	a.step(appInput{Help: true}, now)
	if !a.flow.viewer.HelpOpen() {
		t.Fatal("help did not open")
	}
	return a, a.flow.viewer, now
}

// The panel-local rectangles of the text control, scroll bar and OK button, the
// 12 visible lines and the 0..lines-12 range (MENU-079).
func TestHelpPanelRectangles(t *testing.T) {
	_, v, _ := helpOpenApp(t)
	l := v.noticeLayout()
	if want := image.Rect(40, 56, 422, 267); l.Text != want {
		t.Errorf("text control = %v, want %v (382x211)", l.Text, want)
	}
	if want := image.Rect(422, 56, 446, 267); l.Scrollbar != want {
		t.Errorf("scroll bar = %v, want %v (24x211)", l.Scrollbar, want)
	}
	if want := image.Rect(196, 300, 292, 324); l.Button != want {
		t.Errorf("OK button = %v, want %v (96x24)", l.Button, want)
	}
	lines, visible := v.HelpLines()
	if visible != 12 {
		t.Errorf("visible lines = %d, want 12", visible)
	}
	if first, last := v.HelpScroll(); first != 0 || last != lines-12 {
		t.Errorf("scroll %d..%d, want 0..%d", first, last, lines-12)
	}
}

// The body is grey ramp entry 15 over the 8-per-channel shadow, not the
// dialogue white (MENU-079).
func TestHelpTextInkAndShadow(t *testing.T) {
	_, v, _ := helpOpenApp(t)
	pic, _, _, ok := v.noticePresent()
	if !ok {
		t.Fatal("no picture")
	}
	l := v.noticeLayout()
	seen := map[color.RGBA]bool{}
	for y := l.Text.Min.Y; y < l.Text.Max.Y; y++ {
		for x := l.Text.Min.X; x < l.Text.Max.X; x++ {
			seen[pic.RGBAAt(x, y)] = true
		}
	}
	if !seen[color.RGBA{210, 210, 210, 255}] {
		t.Error("no grey ramp entry 15 ink in the text")
	}
	if !seen[color.RGBA{8, 8, 8, 255}] {
		t.Error("no 8-per-channel shadow in the text")
	}
	if seen[color.RGBA{255, 255, 255, 255}] {
		t.Error("white dialogue ink in the help text")
	}
}

// The first Up does nothing, the first Down only resyncs the current line, and
// the first Page Up resyncs to the top (MENU-078).
func TestHelpFirstPressResyncs(t *testing.T) {
	a, v, now := helpOpenApp(t)
	a.step(appInput{Up: true}, now)
	a.step(appInput{Down: true}, now)
	if first, _ := v.HelpScroll(); first != 0 {
		t.Fatalf("Up then Down scrolled to %d, want 0", first)
	}
	a, v, now = helpOpenApp(t)
	a.step(appInput{PageDown: true}, now)
	if first, _ := v.HelpScroll(); first != 11 {
		t.Fatalf("Page Down scrolled to %d, want 11 (visible-1)", first)
	}
	a, v, now = helpOpenApp(t)
	a.step(appInput{PageUp: true}, now)
	if first, _ := v.HelpScroll(); first != 0 || v.help.cur != 0 {
		t.Fatalf("first Page Up: scroll %d current %d, want 0 and 0", first, v.help.cur)
	}
	a.step(appInput{PageDown: true}, now)
	a.step(appInput{PageUp: true}, now)
	if first, _ := v.HelpScroll(); first != 0 {
		t.Errorf("Page Up after Page Down scrolled to %d, want 0", first)
	}
}

// With OK focused the four keys do nothing; Up returns focus to the text
// without scrolling; Enter closes from either focus (MENU-078).
func TestHelpFocusGatesTheScrollKeys(t *testing.T) {
	a, v, now := helpOpenApp(t)
	if v.help.okFocus {
		t.Fatal("OK holds the initial focus")
	}
	a.step(appInput{PaneMode: true}, now)
	if !v.help.okFocus {
		t.Fatal("Tab did not move focus to OK")
	}
	for _, in := range []appInput{{Down: true}, {PageDown: true}, {PageUp: true}} {
		a.step(in, now)
		if first, _ := v.HelpScroll(); first != 0 || !v.help.okFocus {
			t.Fatalf("%+v with OK focused: scroll %d, OK focus %v", in, first, v.help.okFocus)
		}
	}
	a.step(appInput{Up: true}, now)
	if v.help.okFocus {
		t.Fatal("Up from OK left focus on OK")
	}
	if first, _ := v.HelpScroll(); first != 0 {
		t.Fatalf("Up from OK scrolled to %d", first)
	}
	a.step(appInput{Down: true}, now)
	a.step(appInput{Down: true}, now)
	if first, _ := v.HelpScroll(); first != 1 {
		t.Errorf("Down with the text focused again: scroll %d, want 1", first)
	}
	a.step(appInput{PaneMode: true}, now)
	a.step(appInput{Enter: true}, now)
	if v.HelpOpen() {
		t.Error("Enter with OK focused did not close help")
	}
	a, v, now = helpOpenApp(t)
	a.step(appInput{Enter: true}, now)
	if v.HelpOpen() {
		t.Error("Enter with the text focused did not close help")
	}
}

// F1 opens over the spellbook, a panel that leaves the campaign word at 1
// (MENU-080).
func TestHelpGateOverSpellbook(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 60)
	v := a.flow.viewer
	if !v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
	a.step(appInput{Help: true}, now)
	if !v.HelpOpen() {
		t.Error("F1 did not open help over the spellbook")
	}
}

// The meter recomputes only after more than 1000 ms and states draw calls x
// 1000 over elapsed ms (MENU-081).
func TestFPSMeter(t *testing.T) {
	var m fpsMeter
	for i := 0; i < 50; i++ {
		if got := m.frame(int64(i) * 20); got != 0 {
			t.Fatalf("draw %d: rate %v before 1000 ms elapsed", i, got)
		}
	}
	// 50 calls, 980 ms elapsed. A call at 1000 ms makes exactly 1000 and does
	// not recompute; the next at 1040 exceeds it.
	m.frame(1000)
	if m.value != 0 {
		t.Fatalf("exactly 1000 ms recomputed: %v", m.value)
	}
	got := m.frame(1040)
	if want := 52.0 * 1000 / 1040; got != want {
		t.Errorf("rate = %v, want %v", got, want)
	}
	if m.calls != 0 || m.elapsed != 40 {
		t.Errorf("after recompute: calls %d elapsed %d, want 0 and 40", m.calls, m.elapsed)
	}
}

// The readout box is the value 8 on every channel and the text is right-aligned
// 8 px inside its right edge from y 0 over a +1,+1 shadow (MENU-081).
func TestFPSReadoutPlacement(t *testing.T) {
	_, v := mkOnMap(t)
	v.SetFont(solidFont15())
	v.ToggleFPS()
	pic, at, ok := v.fpsPresent(60)
	if !ok {
		t.Fatal("no readout")
	}
	if want := image.Pt(v.cam.ViewW-120, 0); at != want {
		t.Errorf("box origin %v, want %v", at, want)
	}
	if pic.Bounds().Dx() != 90 || pic.Bounds().Dy() != 24 {
		t.Errorf("box %v, want 90x24", pic.Bounds())
	}
	box := color.RGBA{8, 8, 8, 255}
	if pic.RGBAAt(89, 23) != box || pic.RGBAAt(0, 23) != box {
		t.Error("box is not the value 8 on every channel")
	}
	right, top := -1, 99
	for y := 0; y < 24; y++ {
		for x := 0; x < 90; x++ {
			if pic.RGBAAt(x, y) == (color.RGBA{255, 255, 255, 255}) {
				right, top = max(right, x), min(top, y)
			}
		}
	}
	if right < 0 {
		t.Fatal("no white ink drawn")
	}
	if right != 81 {
		t.Errorf("last ink column %d, want 81 (right edge at width-38 = box column 82)", right)
	}
	if top != 0 {
		t.Errorf("ink starts at row %d, want 0", top)
	}
}

// Alt+Backspace and Alt+F12 reach neither arm (MENU-082); the plain keys do.
func TestAltBackspaceAndAltF12AreInert(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetFont(panelFont())
	v.PostMessage("kept", MessageWhite, time.Minute)
	if err := a.HeadlessKey("alt-backspace"); err != nil {
		t.Fatal(err)
	}
	if len(v.MessageLines()) != 1 {
		t.Error("Alt+Backspace emptied the message line")
	}
	if err := a.HeadlessKey("alt-f12"); err != nil {
		t.Fatal(err)
	}
	if v.FPSShown() {
		t.Error("Alt+F12 toggled the readout")
	}
	if err := a.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if len(v.MessageLines()) != 0 {
		t.Error("control: plain Backspace did not empty the line")
	}
	if err := a.HeadlessKey("f12"); err != nil || !v.FPSShown() {
		t.Error("control: plain F12 did not toggle the readout")
	}
}
