package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

const messageGrid = 24

// messageFont is a 224-record font in the shipped arrangement, distinguishable
// record from record.
func messageFont() *text.Font {
	f := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 5, Height: 6, Pixels: make([]text.Pixel, 30), Advance: 3}
		if k == 0 {
			g.Advance = 2
		} else {
			g.Pixels[(k%6)*5+(k/6)%5] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// shippedShapeFont has the shipped map font's cells, 16 by 15, and its letter
// spacing of 2. Every record advances 10 pixels, the space record included
// (its own height-over-two term is folded into its advance), so a string
// advances 10 pixels a byte with no spacing left over. Record 'A' paints its
// first column at level 15 and record 'B' the same column at level 8; no
// other record paints anything.
func shippedShapeFont() *text.Font {
	f := &text.Font{Spacing: 2, Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = text.Glyph{Width: 16, Height: 15, Advance: 8, Pixels: make([]text.Pixel, 16*15)}
	}
	f.Glyphs[0].Advance = 1
	column := func(record int, level uint8) {
		for y := 0; y < 15; y++ {
			f.Glyphs[record].Pixels[y*16] = text.Pixel{Level: level, Painted: true}
		}
	}
	column('A'-text.FirstChar, 15)
	column('B'-text.FirstChar, 8)
	return f
}

// messageViewer is a lit viewer over a flat grid with the distinguishable
// font, whose view is 640 by 480 pixels.
func messageViewer(t *testing.T) *Viewer {
	t.Helper()
	return messageViewerSized(t, 640, 480, messageFont())
}

// messageViewerSized is a lit viewer over a flat grid with a font, whose view
// is viewW by viewH pixels.
func messageViewerSized(t *testing.T, viewW, viewH int, f *text.Font) *Viewer {
	t.Helper()
	v, err := NewViewer("message", grid(messageGrid, messageGrid), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, viewW, viewH)
	v.SetFont(f)
	return v
}

// messageClock delivers one clock reading per call, ms milliseconds after the
// last, the way the map screen's tick does.
type messageClock struct {
	v   *Viewer
	now time.Time
}

func (c *messageClock) tick(ms int) {
	c.now = c.now.Add(time.Duration(ms) * time.Millisecond)
	c.v.stepMessages(c.now)
}

func messageTexts(v *Viewer) []string {
	var out []string
	for _, ln := range v.MessageLines() {
		out = append(out, ln.Text)
	}
	return out
}

// The capacity is half the lines the view's height holds at the font's pitch,
// after the view's bottom snaps down to whole 32-pixel rows: 14, 16 and 22 for
// the three views the original has, 480, 576 and 768 pixels high, with the
// 800 by 600 screen's 600 snapping to 576 (MISSION-MSGLINE-057).
func TestMessageCapacityIsHalfTheLinesTheViewHolds(t *testing.T) {
	f := shippedShapeFont()
	for _, tc := range []struct{ viewH, want int }{
		{480, 14}, {576, 16}, {600, 16}, {768, 22}, {31, 0},
	} {
		if got := messageCapacity(f, tc.viewH); got != tc.want {
			t.Errorf("view %d high: capacity %d, want %d", tc.viewH, got, tc.want)
		}
	}
}

// A post appends at the bottom, and past the capacity the oldest lines go.
func TestMessagePostAppendsAndDropsTheOldestPastCapacity(t *testing.T) {
	v := messageViewerSized(t, 864, 768, shippedShapeFont())
	var want []string
	for i := 0; i < 25; i++ {
		s := "line " + string(rune('a'+i))
		v.PostMessage(s, MessageWhite, 3*time.Second)
		want = append(want, s)
		if len(want) > 22 {
			want = want[1:]
		}
		if got := messageTexts(v); !slices.Equal(got, want) {
			t.Fatalf("after post %d the lines are %q, want %q", i, got, want)
		}
	}
	if len(want) != 22 || want[0] != "line d" {
		t.Fatalf("setup: the oldest kept line is %q of %d", want[0], len(want))
	}
}

// A text narrower than the view stays one line; one at least as wide breaks at
// spaces into lines each narrower than the view, every piece with the post's
// ink and life. A word too wide for the view stands alone (DIV-1500).
func TestMessagePostWrapsAtSpacesAgainstTheViewWidth(t *testing.T) {
	f := shippedShapeFont()
	if f.Advance("aaaa aaaa") != 90 {
		t.Fatalf("setup: the font advances %d for nine bytes, want 90", f.Advance("aaaa aaaa"))
	}
	word := strings.Repeat("a", 9)
	for _, tc := range []struct {
		name string
		text string
		want []string
	}{
		{"one byte short of the width", strings.Repeat("a", 47), []string{strings.Repeat("a", 47)}},
		{"exactly the width", strings.Repeat("a", 23) + " " + strings.Repeat("a", 24),
			[]string{strings.Repeat("a", 23), strings.Repeat("a", 24)}},
		{"ten words of nine", strings.Join(slices.Repeat([]string{word}, 10), " "),
			[]string{
				strings.Join(slices.Repeat([]string{word}, 4), " "),
				strings.Join(slices.Repeat([]string{word}, 4), " "),
				strings.Join(slices.Repeat([]string{word}, 2), " "),
			}},
		{"one word wider than the view", strings.Repeat("a", 60), []string{strings.Repeat("a", 60)}},
		{"a long word among short ones", "aa " + strings.Repeat("b", 60) + " cc",
			[]string{"aa", strings.Repeat("b", 60), "cc"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := messageViewerSized(t, 480, 480, f)
			v.PostMessage(tc.text, MessageGrey, 7*time.Second)
			lines := v.MessageLines()
			var got []string
			for _, ln := range lines {
				got = append(got, ln.Text)
				if ln.Ink != MessageGrey || ln.Life != 7*time.Second {
					t.Errorf("piece %q has ink %d life %v, want the post's", ln.Text, ln.Ink, ln.Life)
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("pieces %q, want %q", got, tc.want)
			}
		})
	}
}

// A carriage return in a post is a space, so it wraps like one.
func TestMessagePostTurnsACarriageReturnIntoASpace(t *testing.T) {
	v := messageViewerSized(t, 480, 480, shippedShapeFont())
	v.PostMessage("Picked up\rSword", MessageWhite, time.Second)
	if got := messageTexts(v); !slices.Equal(got, []string{"Picked up Sword"}) {
		t.Fatalf("lines %q, want the carriage return as a space", got)
	}
	v.PostMessage("", MessageWhite, time.Second)
	if len(v.MessageLines()) != 1 {
		t.Fatalf("an empty post added %d lines", len(v.MessageLines())-1)
	}
}

// Lifetimes follow one another: the oldest line goes once the time counted
// against it exceeds its life, one line per tick, and the line below it is
// counted from the tick that removed the one above (MISSION-MSGLINE-057).
func TestMessageLinesExpireOldestFirstOnePerTick(t *testing.T) {
	v := messageViewerSized(t, 864, 768, shippedShapeFont())
	clock := &messageClock{v: v, now: time.Unix(100, 0)}
	clock.tick(20)
	v.PostMessage("one", MessageWhite, 3*time.Second)
	v.PostMessage("two", MessageWhite, 3*time.Second)
	v.PostMessage("three", MessageGrey, 1*time.Second)

	var gone []int
	seen := 3
	for ms := 20; ms <= 20000 && seen > 0; ms += 20 {
		clock.tick(20)
		if n := len(v.MessageLines()); n != seen {
			if n != seen-1 {
				t.Fatalf("%d ms after the post the list fell from %d to %d lines in one tick", ms, seen, n)
			}
			gone = append(gone, ms)
			seen = n
		}
	}
	// Each line's time counts from the post or the removal above it, and
	// passes 3000, 3000 and 1000 ms on the first tick past each.
	if want := []int{3020, 6040, 7060}; !slices.Equal(gone, want) {
		t.Fatalf("lines went at %v ms after the first tick, want %v", gone, want)
	}
}

// A line dropped past the capacity leaves the time already counted where it
// was: the next oldest line is measured from what its predecessor had.
func TestMessageOverflowLeavesTheTimeCountedUnchanged(t *testing.T) {
	v := messageViewerSized(t, 864, 768, shippedShapeFont())
	clock := &messageClock{v: v, now: time.Unix(100, 0)}
	clock.tick(20)
	v.PostMessage("first", MessageWhite, 3*time.Second)
	for i := 0; i < 100; i++ {
		clock.tick(20)
	}
	for i := 0; i < 22; i++ {
		v.PostMessage("filler", MessageWhite, 3*time.Second)
	}
	if lines := v.MessageLines(); len(lines) != 22 || lines[0].Text != "filler" {
		t.Fatalf("setup: %d lines after the overflow, oldest %q", len(lines), lines[0].Text)
	}
	// 2000 ms are counted. The oldest filler goes when the count passes 3000.
	for i := 0; i < 50; i++ {
		clock.tick(20)
	}
	if got := len(v.MessageLines()); got != 22 {
		t.Fatalf("a line went at 3000 ms counted, %d left", got)
	}
	clock.tick(20)
	if got := len(v.MessageLines()); got != 21 {
		t.Fatalf("no line went once the counted time passed its life: %d left", got)
	}
}

// A post that leaves one line restarts the clock. A post of several pieces
// into an empty list does not, so its first piece is measured from the last
// tick before the list emptied and goes on the first tick after the post
// (DIV-1499).
func TestMessagePostIntoAnEmptyListRestartsTheClockForOneLineOnly(t *testing.T) {
	f := shippedShapeFont()
	emptied := func(t *testing.T) (*Viewer, *messageClock) {
		v := messageViewerSized(t, 480, 480, f)
		clock := &messageClock{v: v, now: time.Unix(100, 0)}
		clock.tick(20)
		v.PostMessage("first", MessageWhite, 3*time.Second)
		for len(v.MessageLines()) > 0 {
			clock.tick(20)
		}
		for i := 0; i < 500; i++ {
			clock.tick(20)
		}
		return v, clock
	}

	t.Run("one piece", func(t *testing.T) {
		v, clock := emptied(t)
		v.PostMessage("short", MessageWhite, 3*time.Second)
		clock.tick(20)
		if got := messageTexts(v); !slices.Equal(got, []string{"short"}) {
			t.Fatalf("a one-line post left %q a tick later, want it standing", got)
		}
	})

	t.Run("two pieces", func(t *testing.T) {
		v, clock := emptied(t)
		word := strings.Repeat("a", 9)
		v.PostMessage(strings.Join(slices.Repeat([]string{word}, 6), " "), MessageWhite, 3*time.Second)
		pieces := len(v.MessageLines())
		clock.tick(20)
		if pieces != 2 || len(v.MessageLines()) != 1 {
			t.Fatalf("a %d-piece post held %d lines a tick later, want its first piece gone", pieces, len(v.MessageLines()))
		}
	})
}

// A tick with nothing standing changes no time, and the first line posted
// before any clock reading is counted from the first tick.
func TestMessageTickNeedsALineAndAReading(t *testing.T) {
	v := messageViewerSized(t, 864, 768, shippedShapeFont())
	v.PostMessage("early", MessageWhite, 3*time.Second)
	clock := &messageClock{v: v, now: time.Unix(5000, 0)}
	clock.tick(20)
	if got := messageTexts(v); !slices.Equal(got, []string{"early"}) {
		t.Fatalf("a line posted before any reading went at the first tick: %q", got)
	}
	for i := 0; i < 150; i++ {
		clock.tick(20)
	}
	if got := len(v.MessageLines()); got != 1 {
		t.Fatalf("the line went before 3000 ms had passed")
	}
	clock.tick(20)
	if got := len(v.MessageLines()); got != 0 {
		t.Fatalf("the line stands past 3000 ms")
	}
}

// The map screen's own step delivers the clock: a line posted on a stepping
// viewer stands through 3000 ms of steps and goes on the first step past them.
func TestViewerStepAgesTheMessageLine(t *testing.T) {
	v := messageViewer(t)
	at := time.Unix(1000, 0)
	v.step(Input{}, at)
	v.PostMessage("one", MessageWhite, 3*time.Second)
	for i := 0; i < 150; i++ {
		at = at.Add(20 * time.Millisecond)
		v.step(Input{}, at)
	}
	if got := len(v.MessageLines()); got != 1 {
		t.Fatalf("the line went within 3000 ms of steps: %d lines", got)
	}
	v.step(Input{}, at.Add(20*time.Millisecond))
	if got := len(v.MessageLines()); got != 0 {
		t.Fatalf("the line stands after the step past 3000 ms: %d lines", got)
	}
}

// A message line is drawn from (8,8), one pitch of the font's height and 2
// lower for each next line, every line starting at the same x, the text in
// the line's ramp and a 1-pixel shadow of 8 in every channel behind it
// (MISSION-MSGLINE-056).
func TestMessageLogDrawsTextAndShadowAtTheOriginalGeometry(t *testing.T) {
	f := shippedShapeFont()
	v := messageViewerSized(t, 864, 768, f)
	v.PostMessage("AB", MessageWhite, 3*time.Second)
	v.PostMessage("BA", MessageGrey, 3*time.Second)
	pic, at, lines, ok := v.MessageLog()
	if !ok || at != image.Pt(8, 8) || len(lines) != 2 {
		t.Fatalf("MessageLog = %v, %v, %+v, want the picture at (8,8) and two lines", pic != nil, at, lines)
	}
	wantPens := []image.Point{image.Pt(8, 8), image.Pt(8, 25)}
	for i, ln := range lines {
		if ln.Pen != wantPens[i] {
			t.Errorf("line %d pen %v, want %v", i, ln.Pen, wantPens[i])
		}
	}
	shadow := color.RGBA{R: 8, G: 8, B: 8, A: 255}
	at0 := func(x, y int) color.RGBA { return pic.RGBAAt(x, y) }
	for _, tc := range []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"white A face", 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"white A face, last row", 0, 14, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"white B face at level 8", 10, 0, color.RGBA{R: 136, G: 136, B: 136, A: 255}},
		{"grey B face at level 8", 0, 17, color.RGBA{R: 112, G: 112, B: 112, A: 255}},
		{"grey A face at level 15", 10, 17, color.RGBA{R: 210, G: 210, B: 210, A: 255}},
		{"shadow of the first A", 1, 1, shadow},
		{"shadow of the first A, last row", 1, 15, shadow},
		{"shadow of the first B", 11, 1, shadow},
		{"shadow of the second line's B", 1, 18, shadow},
		{"shadow of the second line's A", 11, 32, shadow},
		{"nothing right of the shadow", 2, 1, color.RGBA{}},
		{"nothing below the line", 0, 15, color.RGBA{}},
	} {
		if got := at0(tc.x, tc.y); got != tc.want {
			t.Errorf("%s: pixel (%d,%d) is %v, want %v", tc.name, tc.x, tc.y, got, tc.want)
		}
	}
	painted := 0
	b := pic.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if pic.RGBAAt(x, y).A != 0 {
				painted++
			}
		}
	}
	// Two lines of two glyphs, each a 15-pixel face column and a 15-pixel shadow column.
	if painted != 2*2*(15+15) {
		t.Errorf("%d painted pixels, want %d: nothing beyond faces and shadows", painted, 2*2*(15+15))
	}
}

// The picture is composed again when the lines or the font change, and only
// then.
func TestMessagePictureFollowsTheLinesAndTheFont(t *testing.T) {
	v := messageViewerSized(t, 864, 768, shippedShapeFont())
	if _, _, ok := v.messagePresent(); ok {
		t.Fatal("a picture with nothing posted")
	}
	v.PostMessage("A", MessageWhite, 3*time.Second)
	first, _, ok := v.messagePresent()
	if !ok {
		t.Fatal("no picture for a posted line")
	}
	again, _, _ := v.messagePresent()
	if first != again {
		t.Error("the picture was composed again with nothing changed")
	}
	v.PostMessage("B", MessageWhite, 3*time.Second)
	if second, _, _ := v.messagePresent(); second == first || second.Bounds().Dy() <= first.Bounds().Dy() {
		t.Error("a second line did not make a taller picture")
	}
	v.SetFont(shippedShapeFont())
	if third, _, _ := v.messagePresent(); third == nil || v.messageFresh != true {
		t.Error("a new font did not compose the picture again")
	}
	v.SetFont(nil)
	if _, _, ok := v.messagePresent(); ok {
		t.Error("a picture with no font")
	}
	if len(v.MessageLines()) != 2 {
		t.Error("the lines did not survive a missing font")
	}
}

// A line posted before the viewer has a font is kept and drawn once it does.
func TestMessageLinePostedWithoutAFontStandsUntilOneIsSet(t *testing.T) {
	v, err := NewViewer("message-nofont", grid(messageGrid, messageGrid), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 864, 768)
	v.PostMessage("Sword", MessageWhite, 3*time.Second)
	if _, _, ok := v.messagePresent(); ok {
		t.Fatal("a picture with no font")
	}
	v.SetFont(messageFont())
	if _, _, ok := v.messagePresent(); !ok {
		t.Fatal("no picture once the font is set")
	}
}

// A posted line moves neither selection nor orders.
func TestMessageLinesDoNotAffectSelectionOrOrders(t *testing.T) {
	v := messageViewer(t)
	v.SetEntities([]MapEntity{{ID: 9, Cell: image.Pt(2, 2)}})
	x, y := cellPoint(v, 2, 2)
	v.PostMessage("Sword", MessageWhite, 3*time.Second)
	if _, ok := tapAt(v, x, y); ok {
		t.Errorf("a tap with a message line standing issued an order")
	}
	if !slices.Equal(v.sel, selection{9}) {
		t.Errorf("selection with a message line standing = %v, want [9]", v.sel)
	}
}
