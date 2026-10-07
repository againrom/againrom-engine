package main

import (
	"io"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// censusFixture is a font whose every censused property is a value this test
// chose: a blank record 0, records of two cell sizes, known levels, one record
// whose ink runs past its advance and one whose advance is not smaller than its
// own cell.
func censusFixture() *text.Font {
	glyph := func(w, h, adv int, ink func(x, y int) (uint8, bool)) text.Glyph {
		g := text.Glyph{Width: w, Height: h, Advance: adv, Pixels: make([]text.Pixel, w*h)}
		if ink != nil {
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if lv, on := ink(x, y); on {
						g.Pixels[y*w+x] = text.Pixel{Level: lv, Painted: true}
					}
				}
			}
		}
		return g
	}
	return &text.Font{Spacing: 2, Glyphs: []text.Glyph{
		glyph(8, 6, 1, nil), // record 0: the space, blank
		glyph(8, 6, 3, func(x, y int) (uint8, bool) { return 15, x == 0 && y == 0 }),      // 1 px, level 15
		glyph(8, 6, 4, func(x, y int) (uint8, bool) { return 0, x < 2 && y == 0 }),        // 2 px, level 0
		glyph(8, 6, 2, func(x, y int) (uint8, bool) { return 7, x == 7 && y == 5 }),       // ink past the advance
		glyph(4, 9, 9, func(x, y int) (uint8, bool) { return uint8(8), x == 1 && y < 3 }), // advance >= cell
	}}
}

// AC-11: every figure the census reports equals the fixture's own.
func TestCensus(t *testing.T) {
	c := TakeCensus(censusFixture())

	if c.Records != 5 {
		t.Fatalf("records %d, want 5", c.Records)
	}
	if c.MinW != 4 || c.MaxW != 8 || c.MinH != 6 || c.MaxH != 9 {
		t.Fatalf("cells %dx%d..%dx%d, want 4x6..8x9", c.MinW, c.MinH, c.MaxW, c.MaxH)
	}
	if c.MinAdvance != 1 || c.MaxAdvance != 9 || c.SumAdvance != 1+3+4+2+9 {
		t.Fatalf("advances min %d max %d sum %d, want 1/9/19", c.MinAdvance, c.MaxAdvance, c.SumAdvance)
	}
	if c.Inked != 4 {
		t.Fatalf("inked records %d, want 4", c.Inked)
	}
	if !c.Record0Blank {
		t.Fatal("record 0 reported as inked; the fixture leaves it blank")
	}
	if c.AdvanceGECell != 1 {
		t.Fatalf("advance>=cell on %d records, want 1", c.AdvanceGECell)
	}

	// Levels, counted per painted pixel — level 0 among them, which is the whole
	// reason this figure exists.
	want := map[int]int{0: 2, 7: 1, 8: 3, 15: 1}
	for lv, n := range c.Levels {
		if n != want[lv] {
			t.Fatalf("level %d counted %d painted pixels, want %d", lv, n, want[lv])
		}
	}

	// Ink past the advance: record 1 (ink at column 0, advance 3) does not;
	// record 3 (ink at column 7, advance 2) does; record 4 (ink at column 1,
	// advance 9) does not.
	if c.PastAdvance != 1 {
		t.Fatalf("ink past the advance on %d records, want 1", c.PastAdvance)
	}

	var b strings.Builder
	c.Write(&b, "fixture")
	for _, want := range []string{"records        5", "record 0 blank true", "level0 2", "ink past adv   1"} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("census output does not carry %q:\n%s", want, b.String())
		}
	}
}

func TestCensusOfAnEmptyFont(t *testing.T) {
	c := TakeCensus(&text.Font{})
	if c.Records != 0 || c.Inked != 0 || c.PastAdvance != 0 {
		t.Fatalf("an empty font censused as %+v", c)
	}
	var b strings.Builder
	c.Write(&b, "empty")
	if !strings.Contains(b.String(), "records        0") {
		t.Fatalf("census output: %s", b.String())
	}
}

// AC-12: with no asset root available the tool says so and reads no built-in
// path. Both verbs go through the same resolution, so both are checked.
func TestNoAssetRoot(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")

	err := run([]string{"census"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "asset root") {
		t.Fatalf("census with no root gave %v; want an error naming the asset root", err)
	}

	err = run([]string{"render", "-out", "unused.png", "-text", "x"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "asset root") {
		t.Fatalf("render with no root gave %v; want an error naming the asset root", err)
	}
}

// render writes nothing without an output path, and takes exactly one of the two
// ways of naming a string.
func TestRenderRefusals(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"render", "-text", "x"}, "no default output path"},
		{[]string{"render", "-out", "a.png"}, "exactly one of -text or -hex"},
		{[]string{"render", "-out", "a.png", "-text", "x", "-hex", "41"}, "exactly one of -text or -hex"},
		{[]string{"render", "-out", "a.png", "-hex", "zz"}, "-hex"},
		{[]string{"render", "-out", "a.png", "-text", "x", "-scale", "0"}, "-scale"},
		{[]string{"render", "-out", "a.png", "-text", "x", "-color", "nope"}, "-color"},
		{[]string{"nonsense"}, "unknown verb"},
		{nil, "usage"},
	} {
		err := run(tc.args, io.Discard)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("run(%v) gave %v; want an error naming %q", tc.args, err, tc.want)
		}
	}
}
