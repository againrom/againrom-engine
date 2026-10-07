package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/render/text"
)

// namedPreCreate opens a pre-create page whose pictures carry their own names,
// in picture order male fighter, male mage, female fighter, female mage, and
// whose field is seeded with the no-name text, as the game side seeds it.
func namedPreCreate() *Chargen {
	return NewChargen(ChargenSetup{
		Name: "Unnamed",
		PreCreate: &ChargenPreCreate{
			HeroNames: [4]string{"Danath", "Fergard", "Naira", "Reniesta"},
			Unnamed:   "Unnamed",
		},
		EncodeName: func(r rune) (byte, bool) { return byte(r), r < 0x80 },
		Choices:    []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}, {Options: []string{"fighter", "mage"}, Parent: -1}},
		Stats:      []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 1000,
	})
}

func erase(c *Chargen) {
	for range len(c.NameText()) {
		c.EditName("", true)
	}
}

// TEXT-073: the page opens with the first picture lit and the no-name seed
// turned into that picture's name.
func TestPreCreateOpensAtTheFirstPicturesName(t *testing.T) {
	c := namedPreCreate()
	if c.NameText() != "Danath" || c.PreChoice() != 0 {
		t.Fatalf("opened at %q, picture %d; want Danath, picture 0", c.NameText(), c.PreChoice())
	}
}

// TEXT-074: a press writes the picture's name only over a default text and
// only when the picture differs from the last pressed one.
func TestPreCreateHeroPressWritesOnlyOverADefault(t *testing.T) {
	c := namedPreCreate()
	steps := []struct {
		press  int
		typed  string
		back   bool
		erase  bool
		want   string
		reason string
	}{
		{press: 2, want: "Naira", reason: "a default takes the pressed picture's name"},
		{press: 1, want: "Fergard", reason: "a default takes the pressed picture's name"},
		{press: -1, typed: "x", want: "Fergardx", reason: "typing appends"},
		{press: 3, want: "Fergardx", reason: "a typed name survives a press"},
		{press: -1, back: true, want: "Fergard", reason: "backspace restores a default text"},
		{press: 3, want: "Fergard", reason: "the last pressed picture writes nothing"},
		{press: 0, want: "Danath", reason: "another picture writes over the default"},
		{press: -1, erase: true, typed: "Unnamed", want: "Unnamed", reason: "the no-name text typed by hand"},
		{press: 2, want: "Naira", reason: "the no-name text is a default too"},
		{press: -1, erase: true, want: "", reason: "an erased field"},
		{press: 1, want: "", reason: "an empty field is not a default"},
	}
	for i, s := range steps {
		if s.erase {
			erase(c)
		}
		if s.back {
			c.EditName("", true)
		}
		if s.typed != "" {
			c.EditName(s.typed, false)
		}
		if s.press >= 0 {
			c.SelectPreChoice(s.press)
			if c.PreChoice() != s.press {
				t.Fatalf("step %d: picture %d, want %d", i, c.PreChoice(), s.press)
			}
		}
		if c.NameText() != s.want {
			t.Fatalf("step %d (%s): name %q, want %q", i, s.reason, c.NameText(), s.want)
		}
	}
}

// TEXT-073: Back enters the page again: the first picture is lit and counts
// as pressed, a default becomes its name, a typed name and the difficulty stay.
func TestPreCreateBackEntersThePageAgain(t *testing.T) {
	c := namedPreCreate()
	c.SelectDifficulty(2)
	c.SelectPreChoice(3)
	c.Forward()
	if !c.Back() || c.PreChoice() != 0 || c.NameText() != "Danath" || c.Difficulty() != 3 {
		t.Fatalf("Back from a default = %q, picture %d, difficulty %d; want Danath, 0, 3",
			c.NameText(), c.PreChoice(), c.Difficulty())
	}
	c.SelectPreChoice(0)
	if c.NameText() != "Danath" {
		t.Fatalf("picture 0 after Back wrote %q", c.NameText())
	}
	c.SelectPreChoice(3)
	c.EditName("s", false)
	c.Forward()
	if !c.Back() || c.PreChoice() != 0 || c.NameText() != "Reniestas" {
		t.Fatalf("Back from a typed name = %q, picture %d; want Reniestas, 0", c.NameText(), c.PreChoice())
	}
}

// TEXT-075: the field appends under the ten-byte cap, drops a byte below
// 0x20, and backspace removes the last byte.
func TestPreCreateNameOnlyAppendsUnderTheCap(t *testing.T) {
	c := namedPreCreate()
	c.EditName("ab", false)
	if c.NameText() != "Danathab" {
		t.Fatalf("appended = %q, want Danathab", c.NameText())
	}
	c.EditName("cdef", false)
	if c.NameText() != "Danathabcd" {
		t.Fatalf("at the cap = %q, want Danathabcd", c.NameText())
	}
	c.EditName("", true)
	c.EditName("\a", false)
	if c.NameText() != "Danathabc" {
		t.Fatalf("after backspace and a control byte = %q, want Danathabc", c.NameText())
	}
}

// TEXT-076: the caret shows in the opening frame, flips at the first frame
// more than 500 ms after the last flip, restarts with every character under
// the cap, and reads no focus. Each App tick here is 20 ms.
func TestPreCreateCaretFlipsInTicks(t *testing.T) {
	c := namedPreCreate()
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, nil); err != nil {
		t.Fatal(err)
	}
	t0 := time.Unix(1_700_000_000, 0)
	tick := 0
	step := func(in appInput) bool {
		t.Helper()
		a.step(in, t0.Add(time.Duration(tick)*20*time.Millisecond))
		tick++
		return c.CaretVisible()
	}
	expect := func(from, to int, in appInput, want bool) {
		t.Helper()
		for tick <= to {
			if tick < from {
				t.Fatalf("tick %d before %d", tick, from)
			}
			if got := step(in); got != want {
				t.Fatalf("tick %d: caret %v, want %v", tick-1, got, want)
			}
			in = appInput{}
		}
	}
	expect(0, 0, appInput{}, true)
	expect(1, 26, appInput{}, false)
	expect(27, 52, appInput{}, true)
	expect(53, 59, appInput{}, false)
	expect(60, 86, appInput{Typed: "x"}, true)
	expect(87, 90, appInput{}, false)
	// Backspace does not restart the caret.
	expect(91, 91, appInput{Backspace: true}, false)
	// The focus leaving the field does not stop the blink.
	expect(92, 92, appInput{Down: true}, false)
	if c.Focus() == 0 {
		t.Fatal("the focus did not leave the field")
	}
	expect(93, 112, appInput{}, false)
	expect(113, 113, appInput{}, true)
	expect(114, 114, appInput{Up: true}, true)
	// Characters under the cap restart the phase: without them the caret would
	// hide at tick 139; the last one, at tick 118, holds it to tick 144.
	for i, r := range "abcd" {
		expect(115+i, 115+i, appInput{Typed: string(r)}, true)
	}
	if c.NameText() != "Danathabcd" {
		t.Fatalf("name %q, want Danathabcd at the cap", c.NameText())
	}
	expect(119, 144, appInput{}, true)
	expect(145, 149, appInput{}, false)
	// A character at the cap does not restart it.
	expect(150, 150, appInput{Typed: "e"}, false)
	expect(151, 170, appInput{}, false)
	expect(171, 171, appInput{}, true)
	if c.NameText() != "Danathabcd" {
		t.Fatalf("name %q changed at the cap", c.NameText())
	}
}

// TEXT-076: the detailed page draws no caret; the first frame after Back
// shows the phase the page was left in, and the frame after that settles the
// flip that is due.
func TestPreCreateCaretAcrossTheDetailedPage(t *testing.T) {
	c := namedPreCreate()
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, nil); err != nil {
		t.Fatal(err)
	}
	t0 := time.Unix(1_700_000_000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	a.step(appInput{}, at(0))
	a.step(appInput{}, at(20))
	left := c.CaretVisible()
	c.Forward()
	for ms := 40; ms <= 1000; ms += 20 {
		a.step(appInput{}, at(ms))
		if c.CaretVisible() {
			t.Fatalf("detailed page at %d ms shows a caret", ms)
		}
	}
	c.Back()
	a.step(appInput{}, at(1020))
	if c.CaretVisible() != left {
		t.Fatalf("first frame after Back = %v, want the phase left, %v", c.CaretVisible(), left)
	}
	a.step(appInput{}, at(1040))
	if c.CaretVisible() == left {
		t.Fatal("the flip due after Back did not happen")
	}
}

// TEXT-077: the prompt and the name are left-aligned draws in the name font
// from (224,305) and (224,321), in their own inks; the caret follows the name
// in the name's ink.
func TestPreCreatePromptAndNameDrawLeftAlignedInTheirInks(t *testing.T) {
	narrow := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range narrow.Glyphs {
		narrow.Glyphs[i] = text.Glyph{Width: 1, Height: 1, Advance: 1, Pixels: []text.Pixel{{Level: text.MaxLevel, Painted: true}}}
	}
	art := &ChargenPresentation{Font: narrow, NameFont: chargenTestFont()}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Prompt: "P", Art: art}})
	prompt := color.RGBA{65, 47, 20, 255}
	name := color.RGBA{101, 39, 61, 255}
	check := func(frame *image.RGBA, at image.Point, want color.RGBA, what string) {
		t.Helper()
		if got := frame.RGBAAt(at.X, at.Y); got != want {
			t.Errorf("%s at %v = %v, want %v", what, at, got, want)
		}
	}
	for _, typed := range []string{"N", "NNNN"} {
		erase(c)
		c.EditName(typed, false)
		frame := composeChargenPage(c, chargenNone, chargenNone)
		check(frame, image.Pt(224, 305), prompt, "prompt's first cell")
		check(frame, image.Pt(226, 309), prompt, "prompt's first cell, name font")
		check(frame, image.Pt(224, 321), name, typed+"'s first cell")
		if got := frame.RGBAAt(223, 321); got == name {
			t.Errorf("%s draws left of its origin", typed)
		}
		caretX := 224 + chargenTestFont().Advance(typed)
		check(frame, image.Pt(caretX, 321), name, typed+"'s caret")
	}
}
