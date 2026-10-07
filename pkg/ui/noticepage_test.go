package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// The notice pager: NoticeMaxLines, NoticePagesOf and the two viewer methods
// over them (1032 return 1).
//
// Every fixture here is built in this file. The font is panelFont, shared with
// the panel's and the notice's own tests, so a line count below is arithmetic
// this file can state rather than a number read off a real atlas. Nothing here
// reads a game install, and no assertion is about a shipped string.
//
// WHAT THESE TESTS CANNOT SEE is the shipped EN and RU font's own metrics.
// `go test ./...` is green with no install, so the real line counts are read
// by cmd/savecheck against a real save instead, and closure.md carries those
// numbers.

// noticePageLayout is the authored dialogue layout resolved for the shape a
// paged notice opens in: no portrait, which is what openLoadNotice pushes.
func noticePageLayout() NoticeLayout {
	return AuthoredDialogueLayout().WithPortrait(false)
}

// noticePageLayouts are the two styles the pager has to hold for: the dialogue
// panel's own wrap, which measures a paragraph's last word without its trailing
// space, and the plain wrap.
func noticePageLayouts() map[string]NoticeLayout {
	plain := noticePageLayout()
	plain.Style = NoticeStylePlain
	return map[string]NoticeLayout{"dialogue": noticePageLayout(), "plain": plain}
}

// noticePageParas is a five-paragraph fixture in the shape a load disclosure
// has: complete sentences, each longer than one line of the window.
var noticePageParas = []string{
	"This save was written by an older build and some things could not be restored.",
	"There are no item weights and so no carried load for anyone in the party.",
	"There is no carry capacity for anyone, so nobody is slowed by a heavy pack.",
	"No mission script test can ask whether a character is carrying a named item.",
	"No spell this mission knows has an area radius or a lingering effect on a cell.",
}

// TestNoticeMaxLinesIsTheWindowsOwnClamp pins NoticeMaxLines to what
// NoticeLayoutOf actually draws, by handing NoticeLayoutOf text far longer
// than the window and counting what comes back.
//
// The expectation is INDEPENDENT of NoticeMaxLines' own arithmetic: it is the
// production clip's own result, not a second copy of height/pitch.
func TestNoticeMaxLinesIsTheWindowsOwnClamp(t *testing.T) {
	l, f := noticePageLayout(), panelFont()
	long := strings.TrimSpace(strings.Repeat("wrapping words that keep going ", 200))

	drawn := len(NoticeLayoutOf(l, f, long))
	if drawn == 0 {
		t.Fatal("the fixture text draws no line at all")
	}
	if got := NoticeMaxLines(l, f); got != drawn {
		t.Errorf("NoticeMaxLines = %d, but NoticeLayoutOf draws %d lines of an overlong string", got, drawn)
	}
	// The clamp must bite for this to mean anything.
	if produced := len(noticeWrap(l, f, long)); produced <= drawn {
		t.Fatalf("the fixture produces %d lines and %d are drawn; it is not long enough to clip", produced, drawn)
	}
}

// TestNoticeMaxLinesIsZeroWithNothingMeasurable covers the three ways there is
// no answer: no font, a font with no records, and an area shorter than one
// line.
func TestNoticeMaxLinesIsZeroWithNothingMeasurable(t *testing.T) {
	l := noticePageLayout()
	if got := NoticeMaxLines(l, nil); got != 0 {
		t.Errorf("no font: NoticeMaxLines = %d, want 0", got)
	}
	if got := NoticeMaxLines(l, &text.Font{}); got != 0 {
		t.Errorf("empty font: NoticeMaxLines = %d, want 0", got)
	}
	short := l
	short.Text = image.Rect(l.Text.Min.X, l.Text.Min.Y, l.Text.Max.X, l.Text.Min.Y+1)
	if got := NoticeMaxLines(short, panelFont()); got != 0 {
		t.Errorf("one-pixel-tall area: NoticeMaxLines = %d, want 0", got)
	}
}

func TestNoticePagesOfNeverProducesAPageTheWindowClips(t *testing.T) {
	f := panelFont()
	// A RANGE OF WINDOW HEIGHTS, not only the shipped one. The pager has to
	// hold for whatever the font and the layout measure to, and the real fonts
	// this build ships are not measurable from a test.
	for style, base := range noticePageLayouts() {
		for _, lines := range []int{1, 2, 3, 5, 8, 17} {
			l := base
			pitch := f.Height() + l.Pitch
			l.Text = image.Rect(l.Text.Min.X, l.Text.Min.Y, l.Text.Max.X, l.Text.Min.Y+lines*pitch)
			if got := NoticeMaxLines(l, f); got != lines {
				t.Fatalf("%s: fixture for %d lines measures %d", style, lines, got)
			}

			pages := NoticePagesOf(l, f, noticePageParas)
			if len(pages) == 0 {
				t.Fatalf("%s: %d-line window: no pages", style, lines)
			}
			for i, p := range pages {
				produced := len(noticeWrap(l, f, p))
				drawn := len(NoticeLayoutOf(l, f, p))
				if produced != drawn {
					t.Errorf("%s: %d-line window, page %d of %d: %d lines produced, %d drawn — clipped\n%q",
						style, lines, i+1, len(pages), produced, drawn, p)
				}
				if strings.TrimSpace(p) == "" {
					t.Errorf("%s: %d-line window, page %d of %d is empty", style, lines, i+1, len(pages))
				}
			}
		}
	}
}

// TestNoticePagesOfKeepsEveryWordInOrder is the other half: a pager that fits
// by dropping text would pass the test above.
func TestNoticePagesOfKeepsEveryWordInOrder(t *testing.T) {
	f := panelFont()
	for style, base := range noticePageLayouts() {
		for _, lines := range []int{1, 2, 3, 5, 8, 17} {
			l := base
			pitch := f.Height() + l.Pitch
			l.Text = image.Rect(l.Text.Min.X, l.Text.Min.Y, l.Text.Max.X, l.Text.Min.Y+lines*pitch)

			got := strings.Fields(strings.Join(NoticePagesOf(l, f, noticePageParas), " "))
			want := strings.Fields(strings.Join(noticePageParas, " "))
			if len(got) != len(want) {
				t.Fatalf("%s: %d-line window: %d words paged, want %d", style, lines, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s: %d-line window: word %d = %q, want %q", style, lines, i, got[i], want[i])
				}
			}
		}
	}
}

// TestNoticePagesOfPacksWholeParagraphsWhereItCan checks the packing is not one
// paragraph per page: at the shipped geometry the five-sentence fixture must
// come back as fewer than five pages, or the pager is a splitter.
func TestNoticePagesOfPacksWholeParagraphsWhereItCan(t *testing.T) {
	l, f := noticePageLayout(), panelFont()
	pages := NoticePagesOf(l, f, noticePageParas)
	if len(pages) >= len(noticePageParas) {
		t.Errorf("%d paragraphs came back as %d pages; the pager packs nothing",
			len(noticePageParas), len(pages))
	}
}

// TestNoticePagesOfDegenerateInputs covers the edges: nothing in, nothing out;
// blank paragraphs dropped; and no measurable font yielding ONE page holding
// everything rather than a blind split.
func TestNoticePagesOfDegenerateInputs(t *testing.T) {
	l, f := noticePageLayout(), panelFont()
	if got := NoticePagesOf(l, f, nil); got != nil {
		t.Errorf("nil paragraphs = %v, want nil", got)
	}
	if got := NoticePagesOf(l, f, []string{"", "   ", "\t"}); got != nil {
		t.Errorf("blank paragraphs = %v, want nil", got)
	}
	got := NoticePagesOf(l, nil, noticePageParas)
	if len(got) != 1 {
		t.Fatalf("no font: %d pages, want 1", len(got))
	}
	if !strings.Contains(got[0], noticePageParas[len(noticePageParas)-1]) {
		t.Error("no font: the single page dropped the last paragraph")
	}
}

// TestNoticePagesOfSplitsAParagraphTooLongForOnePage is the case a page cannot
// hold whole. It must still be shown, in order, split at line boundaries.
func TestNoticePagesOfSplitsAParagraphTooLongForOnePage(t *testing.T) {
	f := panelFont()
	for style, l := range noticePageLayouts() {
		pitch := f.Height() + l.Pitch
		l.Text = image.Rect(l.Text.Min.X, l.Text.Min.Y, l.Text.Max.X, l.Text.Min.Y+2*pitch)

		one := strings.TrimSpace(strings.Repeat("alpha bravo charlie delta ", 20))
		pages := NoticePagesOf(l, f, []string{one})
		if len(pages) < 2 {
			t.Fatalf("%s: a paragraph of %d lines came back as %d page(s)",
				style, len(noticeWrap(l, f, one)), len(pages))
		}
		for i, p := range pages {
			if produced, drawn := len(noticeWrap(l, f, p)), len(NoticeLayoutOf(l, f, p)); produced != drawn {
				t.Errorf("%s: split page %d of %d: %d produced, %d drawn", style, i+1, len(pages), produced, drawn)
			}
		}
		if got, want := strings.Join(strings.Fields(strings.Join(pages, " ")), " "), one; got != want {
			t.Errorf("%s: the split lost or reordered words:\ngot  %q\nwant %q", style, got, want)
		}
	}
}

func TestDialogueClipReportsTheCutTheWindowMakes(t *testing.T) {
	v := &Viewer{}
	v.SetFont(panelFont())
	v.SetNoticeLayouts(AuthoredDialogueLayout(), AuthoredOutcomeLayout())

	short := "one short line"
	produced, drawn := v.DialogueClip(short)
	if produced == 0 {
		t.Fatal("a short string produced no line")
	}
	if produced != drawn {
		t.Errorf("short string: %d produced, %d drawn — a fitting string must not be reported clipped", produced, drawn)
	}

	long := strings.TrimSpace(strings.Repeat("wrapping words that keep going ", 200))
	produced, drawn = v.DialogueClip(long)
	if produced <= drawn {
		t.Fatalf("overlong string: %d produced, %d drawn — the clip is not reported", produced, drawn)
	}
	if want := NoticeMaxLines(AuthoredDialogueLayout().WithPortrait(false), panelFont()); drawn != want {
		t.Errorf("overlong string draws %d lines, want the window's own clamp %d", drawn, want)
	}
}

// TestDialogueClipWithNoFontReportsNothing: a viewer that cannot draw a notice
// reports no lines rather than a count the screen will never show.
func TestDialogueClipWithNoFontReportsNothing(t *testing.T) {
	v := &Viewer{}
	v.SetNoticeLayouts(AuthoredDialogueLayout(), AuthoredOutcomeLayout())
	if produced, drawn := v.DialogueClip("anything at all"); produced != 0 || drawn != 0 {
		t.Errorf("no font: %d produced, %d drawn, want 0 and 0", produced, drawn)
	}
}

// TestDialoguePagesUsesThisViewersOwnDialogueWindow ties the two viewer methods
// together: every page DialoguePages returns must measure unclipped through
// DialogueClip, which is the pair the load path relies on.
func TestDialoguePagesUsesThisViewersOwnDialogueWindow(t *testing.T) {
	v := &Viewer{}
	v.SetFont(panelFont())
	v.SetNoticeLayouts(AuthoredDialogueLayout(), AuthoredOutcomeLayout())

	pages := v.DialoguePages(noticePageParas)
	if len(pages) == 0 {
		t.Fatal("no pages")
	}
	for i, p := range pages {
		if produced, drawn := v.DialogueClip(p); produced != drawn {
			t.Errorf("page %d of %d: %d produced, %d drawn — clipped\n%q", i+1, len(pages), produced, drawn, p)
		}
	}
}
