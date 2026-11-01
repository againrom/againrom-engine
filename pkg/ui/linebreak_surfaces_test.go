package ui

import (
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"os"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// breakFixtureFont is a font with a different pen advance for almost every
// record and ink that overhangs the advance on some, so that the pen
// (Advance) and the box (Measure) disagree the way an installed atlas does.
// selector 1 is the converting selector the Russian install uses.
func breakFixtureFont(selector int) *text.Font {
	f := &text.Font{Spacing: 1, Selector: selector, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		adv := 3 + (k*7)%5
		g := text.Glyph{Width: adv + 2, Height: 9, Advance: adv}
		if k == 0 {
			g.Advance = 4
		}
		g.Pixels = make([]text.Pixel, g.Width*g.Height)
		if k != 0 {
			col := adv - 1
			if k%6 == 0 {
				col = g.Width - 1
			}
			g.Pixels[(k%g.Height)*g.Width+col] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// breakTexts are the inputs every surface is asked. Bytes at or above 0x80 are
// the Russian install's font bytes; the UTF-8 entries are what the surfaces
// that take display text are given.
var breakTexts = []string{
	"",
	" ",
	"a",
	"abc",
	"hello world",
	"The quick brown fox jumps over the lazy dog",
	"  leading and   inner   spaces  and trailing  ",
	"one\ntwo\n\nthree",
	"one\r\ntwo\r\n\r\nthree four five six seven",
	"tab\tseparated\twords and\rcarriage\rreturns",
	"supercalifragilisticexpialidocious and antidisestablishmentarianism",
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	"short words a b c d e f g h i j k l m n o p q r s t u v w x y z",
	"Wide WWWW MMMM iiii llll mixed WiMi",
	"first-class pre- cooked well- known sub- division",
	"Royal Edict. We decree  that  the  lands  be  held in trust.\n\nSigned.",
	"\x8f\xe0\xa8\xa2\xa5\xe2 \xac\xa8\xe0 \xa8 \xe2\xe3 \xa6\xa5 \xa4\xae\xab\xa3\xa8\xa5 \xe1\xab\xae\xa2\xa0 \xa4\xab\xef \xaf\xe0\xae\xa2\xa5\xe0\xaa\xa8",
	"\xe0\xa5\xa6\xa8\xac \xaf\xe0\xa8\xaa\xab\xee\xe7\xa5\xad\xa8\xef \xa2\xe1\xa5\xa3\xae\xa4\xa0\xe8\xad\xa8\xe5 \xe1\xab\xae\xa2",
	"Привет мир и все добрые люди этого города",
	"Длинное слово: сверхъестественнопротивоестественный и короткие",
	"C:\\Users\\Someone\\Documents\\Againrom\\Saves\\a_very_long_directory_name\\file.sav",
	"line one\n\nline three is a little longer than line one\nlast",
	"# hash # separated #",
	"a  b   c    d",
	"word-\nbreak and hyphen- ated text with soft- ware parts",
}

// breakWidths include the degenerate ones: none, negative and one pixel.
var breakWidths = []int{-5, 0, 1, 7, 18, 33, 47, 60, 95, 140, 400}

// breakCase is one question put to every surface.
type breakCase struct {
	font  *text.Font
	nilF  bool
	text  string
	width int
}

type breakSurface struct {
	name string
	ask  func(c breakCase) string
}

func breakDump(v any) string { return fmt.Sprintf("%q", v) }

// breakBackground is a card face whose flat span differs row by row, so the
// two rows of a split name have different extents.
func breakBackground() (PanelLayout, *image.RGBA) {
	l := PanelLayout{Size: image.Pt(140, 60)}
	bg := image.NewRGBA(image.Rect(0, 0, 140, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 140; x++ {
			bg.SetRGBA(x, y, characterCardFill)
		}
	}
	for y := 0; y < 60; y++ {
		cut := 10 + (y%7)*3
		for x := 0; x < cut; x++ {
			bg.SetRGBA(x, y, color.RGBA{9, 9, 9, 255})
		}
		for x := 140 - cut/2; x < 140; x++ {
			bg.SetRGBA(x, y, color.RGBA{9, 9, 9, 255})
		}
	}
	return l, bg
}

func breakSurfaces() []breakSurface {
	l, bg := breakBackground()
	dialogueLayout := AuthoredDialogueLayout()
	plain := noticeInkLayout()
	menuApp := func(c breakCase) *App {
		var f *text.Font
		if !c.nilF {
			f = c.font
		}
		return &App{flow: &flow{menuFont: f}}
	}
	font := func(c breakCase) *text.Font {
		if c.nilF {
			return nil
		}
		return c.font
	}
	return []breakSurface{
		{"cardname", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			a, b, ok := splitCardName(l, bg, c.font, c.text, c.width, 20, 11)
			return breakDump([]any{a, b, ok})
		}},
		{"document", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			var out []string
			for _, ln := range documentLines(c.font, c.text, c.width, 9) {
				out = append(out, fmt.Sprintf("%q|%q|%v|%v", ln.words, ln.text, ln.indent, ln.justify))
			}
			return breakDump(out)
		}},
		{"gamemenu", func(c breakCase) string { return breakDump(gameMenuTextLines(c.text)) }},
		{"editor", func(c breakCase) string { return breakDump(editorTextLines(font(c), c.text, c.width)) }},
		{"message", func(c breakCase) string { return breakDump(messageTextLines(font(c), c.text, c.width)) }},
		{"mod", func(c breakCase) string {
			if c.nilF || c.width < 0 {
				// A negative width never terminates in the screen's breaker.
				return "-"
			}
			return breakDump(menuApp(c).modTextLines(c.text, c.width))
		}},
		{"noticelines", func(c breakCase) string { return breakDump(noticeFlowLines(font(c), c.text, c.width)) }},
		{"noticewrap", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			var out [][]string
			for _, lay := range []NoticeLayout{dialogueLayout, plain} {
				lay.Text = image.Rect(0, 0, max(c.width, 0), 400)
				out = append(out, noticeBodyLines(lay, c.font, c.text, 0))
			}
			return breakDump(out)
		}},
		{"dialogue", func(c breakCase) string {
			var out []string
			for _, ln := range dialogueFlow(font(c), c.text, c.width) {
				out = append(out, fmt.Sprintf("%q|%v|%v|%v", ln.text, ln.indent, ln.justify, ln.paragraphEnd))
			}
			return breakDump(out)
		}},
		{"save", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			return breakDump(menuApp(c).saveLines(c.text, c.width))
		}},
		{"shoptip", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			return breakDump(shopTipLines(c.font, c.text, c.width))
		}},
		{"tooltip", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			return breakDump(tooltipFlowLines(c.text, c.font, c.width))
		}},
		{"noticerow", func(c breakCase) string {
			if c.nilF || c.width <= 0 {
				// The row breaker slices past its text at a width it cannot fit a byte in.
				return "-"
			}
			half := len(c.text) / 2
			row := noticeRow{{Text: c.text[:half]}, {Text: c.text[half:], Alert: true}}
			var out []string
			for _, indent := range []int{0, 12} {
				if c.width <= indent && indent > 0 {
					continue
				}
				for _, ln := range noticeRowLines(c.font, row, c.width, indent, func(s string) string { return s }) {
					out = append(out, fmt.Sprintf("%d:%+v", ln.indent, ln.runs))
				}
				out = append(out, "--")
			}
			return breakDump(out)
		}},
		{"tiptext", func(c breakCase) string {
			var out []string
			for _, ln := range tipTextLines(font(c), c.text, c.width) {
				out = append(out, fmt.Sprintf("%q|%v|%v|%v", ln.text, ln.indent, ln.justify, ln.paragraphEnd))
			}
			return breakDump(out)
		}},
		{"editorinput", func(c breakCase) string {
			if c.nilF {
				return "-"
			}
			spans := editorTextSpans(c.text, c.width, func(s string) int { return c.font.Advance(s) })
			return breakDump(spans)
		}},
	}
}

func breakFonts() []struct {
	name string
	f    *text.Font
	nilF bool
} {
	return []struct {
		name string
		f    *text.Font
		nilF bool
	}{
		{"en", breakFixtureFont(0), false},
		{"ru", breakFixtureFont(text.SelectorConverting), false},
		{"nil", nil, true},
	}
}

// breakAnswers is every surface's answer over every input, one checksum per
// (surface, font, text, width), keyed by those four.
func breakAnswers() map[string]string {
	out := map[string]string{}
	for _, s := range breakSurfaces() {
		for _, fnt := range breakFonts() {
			for ti, txt := range breakTexts {
				for _, w := range breakWidths {
					got := s.ask(breakCase{font: fnt.f, nilF: fnt.nilF, text: txt, width: w})
					key := fmt.Sprintf("%s/%s/t%02d/w%d", s.name, fnt.name, ti, w)
					out[key] = fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(got)))
				}
			}
		}
	}
	return out
}

const breakGolden = "testdata/linebreak_surfaces.golden"

// TestLineBreakSurfacesKeepTheirBreaks pins what every multi-line text surface
// answers over fixture data, EN and RU fonts and the degenerate inputs. The
// golden file was recorded before the surfaces were rebuilt on the one line
// breaker and is not rewritten by the suite; AGAINROM_RECORD_BREAKS=1 writes it.
func TestLineBreakSurfacesKeepTheirBreaks(t *testing.T) {
	got := breakAnswers()
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if os.Getenv("AGAINROM_RECORD_BREAKS") == "1" {
		var b strings.Builder
		for _, k := range keys {
			b.WriteString(k + " " + got[k] + "\n")
		}
		if err := os.WriteFile(breakGolden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(breakGolden)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		k, v, _ := strings.Cut(line, " ")
		want[k] = v
	}
	if len(want) != len(got) {
		t.Fatalf("golden holds %d answers, the surfaces give %d", len(want), len(got))
	}
	bad := 0
	shown := map[string]int{}
	for _, k := range keys {
		if want[k] != got[k] {
			bad++
			surface, _, _ := strings.Cut(k, "/")
			if shown[surface]++; shown[surface] <= 3 {
				t.Errorf("%s: answer %s, recorded %s", k, got[k], want[k])
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d of %d answers changed", bad, len(keys))
	}
}
