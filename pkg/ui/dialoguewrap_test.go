package ui

import (
	"image"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

func wrapperControlFont() *text.Font {
	f := dialogueClaimFont()
	f.Spacing = 2
	for i := range f.Glyphs {
		f.Glyphs[i].Advance = 8
	}
	return f
}

func TestDialogueWrapperSelectedControlsReachLayoutAndDraw(t *testing.T) {
	f := wrapperControlFont()
	for _, tc := range []struct {
		name, source string
		width        int
		want         []string
	}{
		{"overwide word", "AAAA", 15, []string{"AAAA"}},
		{"strict full fit", "A B", 37, []string{"A ", "B \r"}},
		{"LF data", "A\nB", 100, []string{"A\nB \r"}},
		{"empty leading piece", "\r\nA", 100, []string{"", "A \r"}},
		{"spaces piece", "   ", 100, []string{""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, ln := range dialogueWrap(f, tc.source, tc.width) {
				got = append(got, ln.text)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("selected lines %q want %q", got, tc.want)
			}
			l := dialogueClaimLayout()
			l.Text = image.Rect(128, 36, 128+tc.width, 171)
			pic := RenderNotice(l, f, tc.source, nil)
			if pic == nil {
				t.Fatal("reached RenderNotice returned nil")
			}
			if n := len(NoticeLayoutOf(l, f, tc.source)); n != len(tc.want) {
				t.Fatalf("layout lines %d want %d", n, len(tc.want))
			}
			if tc.name == "overwide word" && !whiteIn(pic, 150, 180, 43, 44) {
				t.Fatal("whole overwide word was cut across lines")
			}
		})
	}
}

func TestDialogueDoubleSpillsExcludeRetainedAccumulator(t *testing.T) {
	got := justifiedStarts(214, 290, []int{10, 20, 30, 40})
	if want := []int{214, 287, 370, 463}; !reflect.DeepEqual(got, want) {
		t.Fatalf("starts %v want spilled %v", got, want)
	}
}

func TestDialogueWrapperFiniteRemainderKeepsBytes(t *testing.T) {
	f := wrapperControlFont()
	for _, s := range []string{"A", "A B", "A\rB", "A\r"} {
		width := 10
		if s == "A B" {
			width = 27
		}
		got := dialogueWrap(f, s, width)
		var joined []string
		for _, ln := range got {
			joined = append(joined, strings.TrimRight(ln.visible(), " \t"))
		}
		if strings.Join(joined, " ") != s {
			t.Fatalf("finite remainder lost %q: %#v", s, got)
		}
	}
}

func TestDialogueSuppliedPrecisionSeparatesTwoAddSpill(t *testing.T) {
	widths := []int{61, 204, 61, 52, 33, 69}
	for _, tc := range []struct {
		bits uint
		last int
	}{{53, 1558}, {64, 1559}} {
		got := justifiedStartsWith(DialogueArithmetic{Precision: tc.bits}, 159, 1469, widths)
		if got[len(got)-1] != tc.last {
			t.Fatalf("PC%d last %d want %d", tc.bits, got[len(got)-1], tc.last)
		}
	}
	if got := justifiedStartsWith(DialogueArithmetic{Precision: 24, Rounding: big.ToNearestEven}, 214, 290, []int{10, 20, 30, 40}); got[3] != 464 {
		t.Fatalf("PC24 truncation %v", got)
	}
	f := dialogueClaimFont()
	f.Spacing = 0
	for i, w := range widths {
		f.Glyphs[int('A')-text.FirstChar+i].Advance = w
		f.Glyphs[int('A')-text.FirstChar+i].Width = 1
		f.Glyphs[int('A')-text.FirstChar+i].Pixels = make([]text.Pixel, 15)
		for j := range f.Glyphs[int('A')-text.FirstChar+i].Pixels {
			f.Glyphs[int('A')-text.FirstChar+i].Pixels[j] = text.Pixel{Level: 15, Painted: true}
		}
	}
	f.Glyphs[int('Z')-text.FirstChar].Advance = 2000
	for _, tc := range []struct {
		bits uint
		last int
	}{{53, 1399}, {64, 1400}} {
		l := dialogueClaimLayout()
		l.Box = image.Rect(159, 0, 1800, 100)
		l.Text = image.Rect(-10, 0, 1469, 34)
		l.Portrait = image.Rectangle{}
		l.Button = image.Rectangle{}
		l.DialogueArithmetic = DialogueArithmetic{Precision: tc.bits}
		img := RenderNotice(l, f, "A B C D E F Z", nil)
		if !isFullWhite(img.RGBAAt(tc.last, 7)) || isFullWhite(img.RGBAAt(tc.last-1, 7)) {
			t.Fatalf("PC%d reached last glyph at%d got %v previous%v", tc.bits, tc.last, img.RGBAAt(tc.last, 7), img.RGBAAt(tc.last-1, 7))
		}
	}
}

func TestDialogueDeclaredTrimAndSuffixReachWrapper(t *testing.T) {
	f := wrapperControlFont()
	for _, tc := range []struct {
		input string
		want  []string
	}{
		{" \t\v\fA\nB\v\f ", []string{"A\nB \r"}},
		{"A\r\n\r\n \t\v\fB\r\n", []string{"A \r", "B \r"}},
		{"\r\nA\r\n", []string{"", "A \r"}},
	} {
		var got []string
		for _, line := range dialogueWrap(f, tc.input, 100) {
			got = append(got, line.text)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("input %q lines %q want declared trim %q", tc.input, got, tc.want)
		}
	}
}
