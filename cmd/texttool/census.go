package main

import (
	"fmt"
	"io"

	"againrom/pkg/render/text"
)

// Census is what a font measures against the files it was loaded from.
//
// Three of these figures exist because the contract leans on them and no unit
// test can falsify them — a test can only confirm that our code does what our
// own contract says. Levels is one: level 0 is specified as a PAINTED pixel, and
// if the shipped atlases turned out to carry any, glyphs would stamp opaque
// black over their neighbours' overhang. Record0Blank is the second: the
// missing-byte rule falls back to record 0 and is only invisible because that
// record is empty. PastAdvance is the third: whether cells really do overlap.
type Census struct {
	Records int

	MinW, MaxW int
	MinH, MaxH int

	MinAdvance, MaxAdvance int
	SumAdvance             int

	// Inked is how many records carry at least one painted pixel.
	Inked int

	// Levels counts painted pixels by level, index 0..15.
	Levels [text.MaxLevel + 1]int

	// Record0Blank reports whether record 0 — the space, and the fallback for a
	// byte with no record — paints nothing.
	Record0Blank bool

	// PastAdvance is how many records paint past where the pen leaves them, and
	// AdvanceGECell how many carry an advance not smaller than their own cell.
	// The second is the claim a consumer must not invert; a non-zero count is a
	// font this contract does not describe.
	PastAdvance   int
	AdvanceGECell int
}

// TakeCensus measures f. Ink extent is read through the font's own measurement
// rather than by a second scan of the pixels, so the figure it reports is the
// one the drawing tier acts on.
func TakeCensus(f *text.Font) Census {
	c := Census{Records: len(f.Glyphs), Record0Blank: true}
	if c.Records == 0 {
		return c
	}
	c.MinW, c.MaxW = f.Glyphs[0].Width, f.Glyphs[0].Width
	c.MinH, c.MaxH = f.Glyphs[0].Height, f.Glyphs[0].Height
	c.MinAdvance, c.MaxAdvance = f.Glyphs[0].Advance, f.Glyphs[0].Advance

	for i := range f.Glyphs {
		g := &f.Glyphs[i]
		c.MinW, c.MaxW = min(c.MinW, g.Width), max(c.MaxW, g.Width)
		c.MinH, c.MaxH = min(c.MinH, g.Height), max(c.MaxH, g.Height)
		c.MinAdvance, c.MaxAdvance = min(c.MinAdvance, g.Advance), max(c.MaxAdvance, g.Advance)
		c.SumAdvance += g.Advance
		if g.Advance >= g.Width {
			c.AdvanceGECell++
		}

		inked := false
		for _, p := range g.Pixels {
			if !p.Painted {
				continue
			}
			inked = true
			if int(p.Level) < len(c.Levels) {
				c.Levels[p.Level]++
			}
		}
		if inked {
			c.Inked++
			if i == 0 {
				c.Record0Blank = false
			}
		}

		// Whether this record's ink runs past its own advance, asked of the
		// font itself: for a one-byte string the measured width is the larger of
		// the pen and the ink, so the two differing IS the overhang.
		if i < 256-text.FirstChar {
			s := string([]byte{byte(text.FirstChar + i)})
			if w, _ := f.Measure(s); w > f.Advance(s) {
				c.PastAdvance++
			}
		}
	}
	return c
}

// Write prints the census as figures only — no bytes of any glyph, and nothing
// rendered.
func (c Census) Write(w io.Writer, name string) {
	fmt.Fprintf(w, "font           %s\n", name)
	fmt.Fprintf(w, "records        %d\n", c.Records)
	if c.Records == 0 {
		return
	}
	fmt.Fprintf(w, "cell           %dx%d", c.MinW, c.MinH)
	if c.MinW != c.MaxW || c.MinH != c.MaxH {
		fmt.Fprintf(w, " .. %dx%d (records differ)", c.MaxW, c.MaxH)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "advance        min %d  max %d  mean %.2f\n",
		c.MinAdvance, c.MaxAdvance, float64(c.SumAdvance)/float64(c.Records))
	fmt.Fprintf(w, "advance>=cell  %d of %d\n", c.AdvanceGECell, c.Records)
	fmt.Fprintf(w, "inked records  %d of %d\n", c.Inked, c.Records)
	fmt.Fprintf(w, "record 0 blank %v\n", c.Record0Blank)
	fmt.Fprintf(w, "ink past adv   %d of %d\n", c.PastAdvance, c.Records)
	fmt.Fprint(w, "levels        ")
	total := 0
	for lv, n := range c.Levels {
		total += n
		if n > 0 {
			fmt.Fprintf(w, " %d:%d", lv, n)
		}
	}
	fmt.Fprintf(w, "  (painted %d, level0 %d)\n", total, c.Levels[0])
}
