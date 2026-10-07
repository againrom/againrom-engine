package ui

import (
	"image"
	"strings"
	"testing"
)

// paintedLines is the bounding rectangle of each run of consecutive rows on
// which got holds a pixel that bare does not, top to bottom: the rectangle each
// line of text was drawn in over bare.
func paintedLines(got, bare *image.RGBA) []image.Rectangle {
	var out []image.Rectangle
	b := got.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		minX, maxX := b.Max.X, b.Min.X-1
		for x := b.Min.X; x < b.Max.X; x++ {
			if got.RGBAAt(x, y) != bare.RGBAAt(x, y) {
				minX, maxX = min(minX, x), max(maxX, x)
			}
		}
		if maxX < minX {
			continue
		}
		row := image.Rect(minX, y, maxX+1, y+1)
		if n := len(out); n > 0 && out[n-1].Max.Y == y {
			out[n-1] = out[n-1].Union(row)
		} else {
			out = append(out, row)
		}
	}
	return out
}

// TestSelectionLinesStandWhereTheOriginalCaptureHasThem reads the drawn
// rectangle of each selection line off the picture the upper pane composes, at
// no character selected and at two and three, and requires the owner's capture
// of the original (`DIV-1795`): each line centred on the middle of the
// 176-pixel column, its cell 54 pixels below the pane's top and 12 pixels above
// the next line's, the count on a line of its own. The font is solid, every
// glyph a filled cell as wide as its advance, so the ink of a line is exactly
// its measured box plus the one-pixel shadow. The lower statistics card draws
// no line in any state. The numbers come from the composed pictures and not
// from the drawing code.
func TestSelectionLinesStandWhereTheOriginalCaptureHasThem(t *testing.T) {
	v := missionPaneViewer(t)
	font := townShellCardTestFont()
	v.SetFont(font)
	v.SetEntities([]MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", 10, 10, 2, 2),
		panelEntity(3, "three", 10, 10, 3, 3),
	})
	for _, tc := range []struct {
		what  string
		sel   selection
		lines []string
	}{
		{"nothing selected", nil, []string{"No units", "selected"}},
		{"two selected", selection{1, 2}, []string{"Units", "selected:", "2"}},
		{"three selected", selection{1, 2, 3}, []string{"Units", "selected:", "3"}},
	} {
		v.sel = tc.sel
		upper := missionPaneCompose(t, v)
		card, _, ok := v.missionCardPresent()
		if !ok {
			t.Fatalf("%s: the lower box composed no card", tc.what)
		}
		view := v.characterPaneView(true)
		view.SelectionStatus = [3]string{}
		bareUpper := image.NewRGBA(upper.Bounds())
		DrawTownCharacterRegion(bareUpper, view)
		view.Statistics, view.CornerArt, view.Figure = true, nil, nil
		bareCard := image.NewRGBA(card.Bounds())
		DrawCharacterPaneBody(bareCard, view)

		if at, differs := firstDifference(card, bareCard, card.Bounds()); differs {
			t.Errorf("%s: the lower card differs from the bare page at %v", tc.what, at)
		}
		rects := paintedLines(upper, bareUpper)
		if len(rects) != len(tc.lines) {
			t.Fatalf("%s: %d drawn lines %v, want %d", tc.what, len(rects), rects, len(tc.lines))
		}
		for i, r := range rects {
			w, h := font.Measure(tc.lines[i])
			left := (characterPaneSeamW + 160 - w + 1) / 2
			want := image.Rect(left, 54+12*i, left+w+1, 54+12*i+h+1)
			if r != want {
				t.Errorf("%s, line %d %q: drawn in %v, want %v", tc.what, i+1, tc.lines[i], r, want)
			}
		}
	}
}

// TestSelectionLineWiderThanTheColumnStartsAtItsLeftEdge draws a line the
// column cannot hold on a canvas wide enough to show it. It starts at the
// seam's left edge and runs on to the right, and the narrower line beside it
// is still centred.
func TestSelectionLineWiderThanTheColumnStartsAtItsLeftEdge(t *testing.T) {
	font := townShellCardTestFont()
	body := image.Rect(characterPaneSeamW, 0, characterPaneSeamW+160, 242)
	pic := image.NewRGBA(image.Rect(0, 0, 400, 242))
	drawSelectionStatus(pic, font, body, [3]string{strings.Repeat("W", 30), "narrow"})
	rects := paintedLines(pic, image.NewRGBA(pic.Bounds()))
	if len(rects) != 2 {
		t.Fatalf("%d drawn lines %v, want 2", len(rects), rects)
	}
	wide, narrow := rects[0], rects[1]
	wideW, _ := font.Measure(strings.Repeat("W", 30))
	if wideW <= 176 || wide.Min.X != 0 || wide.Dx() != wideW+1 {
		t.Errorf("the wide line is drawn in %v, want it %d wide from the column's left edge", wide, wideW+1)
	}
	narrowW, _ := font.Measure("narrow")
	if want := (176 - narrowW + 1) / 2; narrow.Min.X != want {
		t.Errorf("the narrow line is drawn in %v, want it to start at %d", narrow, want)
	}
}
