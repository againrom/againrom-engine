package ui

import (
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

func tipLayoutFont() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		g := text.Glyph{Width: 2, Height: 10, Advance: 80, Pixels: make([]text.Pixel, 20)}
		if i == 0 {
			g.Advance = 0
		} else {
			for p := range g.Pixels {
				g.Pixels[p] = text.Pixel{Level: uint8(7 + 8*(p%2)), Painted: true}
			}
		}
		f.Glyphs[i] = g
	}
	f.Glyphs[32].Advance = 8
	return f
}

func TestComposeTipTextUsesAuthoredListJustificationAndFlatShadow(t *testing.T) {
	f := tipLayoutFont()
	v := TipPanelView{Rect: image.Rect(0, 0, 312, 200), Text: "A B C D E", Font: f, Art: tipTestArt()}
	pic := image.NewRGBA(v.Rect)
	ComposeTipPanel(pic, v)
	r := TipPanelTextRect(v.Rect)
	ink := color.RGBA{185, 159, 73, 255}
	first := r.Min.X + 8
	gap := (r.Dx() - 8 - 3*80) / 2
	for _, x := range []int{first, first + 80 + gap, first + 2*(80+gap)} {
		if got := pic.RGBAAt(x+1, r.Min.Y); got != ink {
			t.Fatalf("justified word at (%d,%d) = %v, want %v", x+1, r.Min.Y, got, ink)
		}
	}
	if got := pic.RGBAAt(first+1, r.Min.Y+10); got != (color.RGBA{8, 8, 8, 255}) {
		t.Fatalf("low-level glyph shadow = %v, want flat (8,8,8)", got)
	}
	if got := pic.RGBAAt(r.Min.X+1, r.Min.Y+12); got != ink {
		t.Fatalf("paragraph tail starts at %v, want unindented pitch 12", got)
	}
}

func tipSmallFont() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = text.Glyph{Width: 1, Height: 2, Advance: 1}
	}
	return f
}

func TestTipTextUsesSelectedWrapperParagraphsAndStrictFit(t *testing.T) {
	f := tipSmallFont()
	for _, tc := range []struct {
		source string
		width  int
		want   []string
		indent []bool
	}{
		{"a b c d\r\ne f", 7, []string{"a b ", "c d \r", "e f \r"}, []bool{true, false, true}},
		{"a b", 4, []string{"a ", "b \r"}, []bool{true, false}},
		{"\r\na", 20, []string{"", "a \r"}, []bool{true, false}},
		{"a\nb", 20, []string{"a\nb \r"}, []bool{true}},
		{"a\r", 20, []string{"a\r"}, []bool{true}},
		{"a", 1, []string{"a"}, []bool{true}},
	} {
		var got []string
		var indent []bool
		for _, ln := range tipTextLines(f, tc.source, tc.width) {
			got = append(got, ln.text)
			indent = append(indent, ln.indent)
		}
		if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(indent, tc.indent) {
			t.Fatalf("source %q width %d = %q indent %v, want %q indent %v", tc.source, tc.width, got, indent, tc.want, tc.indent)
		}
	}
}

func TestTipTextUsesOnlyInstalledDiscretionaryHyphenation(t *testing.T) {
	f := tipSmallFont()
	for _, tc := range []struct {
		name, source string
		width        int
		want         []string
	}{
		{"joined inline", "a guard- ian b", 30, []string{"a guardian b"}},
		{"installed break", "a guard- ian b", 12, []string{"a guard-", "ian b"}},
		{"word starts line", "guard- ians", 9, []string{"guard-", "ians"}},
		{"spaced dash", "a - b", 30, []string{"a - b"}},
		{"literal hyphen", "part-time", 30, []string{"part-time"}},
		{"no invented break", "abcdefgh", 4, []string{"abcdefgh"}},
		{"multiple markers", "abc- def- ghij", 7, []string{"abc-", "def-", "ghij"}},
		{"paragraph marker", "guard- ian\r\nend", 30, []string{"guardian", "end"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			lines := tipTextLines(f, tc.source, tc.width)
			for _, ln := range lines {
				got = append(got, strings.TrimRight(ln.visible(), " \t"))
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("lines %q, want %q", got, tc.want)
			}
			for i, ln := range lines {
				if ln.justify != (i < len(lines)-1 && !ln.paragraphEnd) {
					t.Fatalf("line %d paragraph-end=%v justify=%v", i, ln.paragraphEnd, ln.justify)
				}
			}
		})
	}
}
