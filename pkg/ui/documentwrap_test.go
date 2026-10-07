package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// TestDocumentTextColourMatchesTheOriginalReferenceScreenshots pins
// docTextColor at the ink colour measured off the owner's own screenshots of
// the licensed EN original (review/docs-original/orig-en-edict-{a,b}.png,
// outside this repository, checked by seat hotfix
// docs/HOTFIXES.md). A regression back toward white would read as pale
// letters with a dark antialiasing fringe again.
func TestDocumentTextColourMatchesTheOriginalReferenceScreenshots(t *testing.T) {
	want := color.RGBA{0x42, 0x2c, 0x10, 0xff}
	if docTextColor != want {
		t.Fatalf("docTextColor = %#v, want %#v (0x42,0x2c,0x10)", docTextColor, want)
	}
}

// TestWrapDocumentTextDropsBlankSeparatorsAndIndentsEachParagraphsFirstLine
// is the layout witness for the Royal Edict text's own shape: a plain blank
// line between two paragraphs and a line holding a single space between two
// others both carry no output line of their own, and every paragraph's own
// first line is marked to draw indented.
//
// THE INPUT MIRRORS THE ROYAL EDICT'S OWN RAW BYTES (dumped once against the
// EN root and not committed): "...Islands\r\n\r\nI, Emperor..." between its
// first two paragraphs, "...state.\r\n \r\nLet them..." between its last two.
func TestWrapDocumentTextDropsBlankSeparatorsAndIndentsEachParagraphsFirstLine(t *testing.T) {
	font := documentsTestFont()
	text := "AB CD\n\nEF GH IJ\n \nKL"
	lines := wrapDocumentText(font, text, 1000, 20) // width 1000: nothing wraps within a paragraph
	if len(lines) != 3 {
		t.Fatalf("wrapDocumentText produced %d lines, want 3 -- a blank separator must not draw its own line",
			len(lines))
	}
	wantText := []string{"AB CD", "EF GH IJ", "KL"}
	for i, l := range lines {
		if l.text != wantText[i] {
			t.Errorf("line %d text = %q, want %q", i, l.text, wantText[i])
		}
		if !l.indent {
			t.Errorf("line %d: indent=false, want true -- every paragraph's own first (and here only) line is indented", i)
		}
		if l.justify {
			t.Errorf("line %d: justify=true, want false -- a paragraph's own last line is never justified", i)
		}
	}
}

// TestWrapDocumentParagraphIndentsOnlyTheFirstLineAndNarrowsItsBudget is the
// exact mechanism the Royal Edict reference screenshots forced
// (docParagraphIndent's own header): the SAME words, wrapped at the SAME
// width, break in a DIFFERENT place once the first line's own budget is
// narrowed by the indent -- which is what pushes a word off the original's
// first line while every later line, never narrowed, keeps its own word.
func TestWrapDocumentParagraphIndentsOnlyTheFirstLineAndNarrowsItsBudget(t *testing.T) {
	font := documentsTestFont()
	words := []string{"AB", "CD", "EF"}
	// This fixture's own arithmetic, pinned so the numbers below are not
	// silently reading a different fixture: every two-letter word measures
	// 16px and every inter-word space advances 12px (documentsTestFont's own
	// header: an 8px cell plus 2px spacing per glyph, and record 0's own
	// space additionally carries half its cell height).
	if w, _ := font.Measure("AB"); w != 16 {
		t.Fatalf("fixture word width = %d, want 16 (this test's own arithmetic below assumes it)", w)
	}
	if adv := font.Advance(" "); adv != 12 {
		t.Fatalf("fixture space advance = %d, want 12 (this test's own arithmetic below assumes it)", adv)
	}

	noIndent := wrapDocumentParagraph(font, words, 60, 0)
	if len(noIndent) != 2 || noIndent[0].text != "AB CD" || noIndent[1].text != "EF" {
		t.Fatalf(`indent 0: got %+v, want ["AB CD" "EF"]`, noIndent)
	}

	indented := wrapDocumentParagraph(font, words, 60, 20)
	if len(indented) != 2 || indented[0].text != "AB" || indented[1].text != "CD EF" {
		t.Fatalf(`indent 20: got %+v, want ["AB" "CD EF"] -- the indent must narrow only the FIRST `+
			`line's own budget`, indented)
	}
	if !indented[0].indent || !indented[0].justify {
		t.Errorf("first group: indent=%v justify=%v, want both true (opens the paragraph, is not its last line)",
			indented[0].indent, indented[0].justify)
	}
	if indented[1].indent || indented[1].justify {
		t.Errorf("second group: indent=%v justify=%v, want both false (not the first line, and the paragraph's own last)",
			indented[1].indent, indented[1].justify)
	}
}

// TestDrawDocumentLineJustifiesNonLastLinesToTheGivenWidth is the pixel
// witness for justification: a line marked justify=true places each word by
// hand, its own natural gaps stretched evenly to land the last word's own
// right edge at the given width; a line marked justify=false is one plain
// font.Draw at its own natural advance, exactly as an unjustified line always
// drew.
func TestDrawDocumentLineJustifiesNonLastLinesToTheGivenWidth(t *testing.T) {
	font := documentsTestFont()
	words := []string{"AB", "CD", "EF"}
	line := docLine{words: words, text: "AB CD EF", justify: true}

	// natural = 16+12+16+12+16 = 72; width 90 leaves 18px extra over 2 gaps,
	// 9px each, with nothing left over.
	const width = 90
	dst := image.NewRGBA(image.Rect(0, 0, 200, 20))
	drawDocumentLine(dst, font, line, 10, 5, width, docTextColor)

	want := image.NewRGBA(image.Rect(0, 0, 200, 20))
	x := 10
	for i, w := range words {
		font.Draw(want, w, x, 5, docTextColor)
		x += font.Advance(w)
		if i < len(words)-1 {
			x += font.Advance(" ") + 9
		}
	}
	if !bytes.Equal(dst.Pix, want.Pix) {
		t.Fatal("drawDocumentLine(justify=true) did not place each word at the evenly stretched offsets")
	}

	last := docLine{words: []string{"AB", "CD"}, text: "AB CD", justify: false}
	dst2 := image.NewRGBA(image.Rect(0, 0, 200, 20))
	drawDocumentLine(dst2, font, last, 10, 5, width, docTextColor)
	want2 := image.NewRGBA(image.Rect(0, 0, 200, 20))
	font.Draw(want2, "AB CD", 10, 5, docTextColor)
	if !bytes.Equal(dst2.Pix, want2.Pix) {
		t.Fatal("drawDocumentLine(justify=false) stretched a line the wrap marked as a paragraph's own last")
	}
}
