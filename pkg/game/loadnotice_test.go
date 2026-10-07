package game

import (
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// The load disclosure as the window draws it (1032 return 1).
//
// THE DEFECT THIS WITNESSES. The disclosure was pushed as ONE string joined
// with newline bytes. ui.noticeBreak treats a newline as a wrap opportunity
// rather than a hard break, so the bullet-per-sentence layout never happened
// and literal newline bytes landed inside drawn lines; the string then produced
// more wrapped lines than the window's text area draws, and ui.NoticeLayoutOf
// dropped the rest with nothing on screen saying so. openLoadNotice set the
// event payload to nil, so the first dismiss closed the window and the dropped
// half was unreachable. Measured on the real EN font: 16 lines needed and 8
// drawn on a version-53 save, on 51 of the owner's 52 loadable saves.
//
// WHAT THIS FILE CANNOT SEE is the shipped fonts' own metrics. `go test ./...`
// is green with no install, so the font here is synthetic and what is proved is
// that the pager holds for whatever the font measures to. cmd/savecheck reads
// the real numbers off a real save, and closure.md carries them.

// noticeFontCellW, noticeFontCellH and the two advances describe the synthetic
// font below: 224 records in the shipped arrangement, record k standing for
// byte 32+k, every record the same size.
//
// THE HEIGHT IS THE SHIPPED ONE. Both installed fonts measure 15 pixels tall,
// so the dialogue window's text area draws eight lines of them, and eight is
// the number the review measured a disclosure being cut to. The widths here are
// this file's own and are narrower than either install's, which makes this
// fixture the optimistic case: a disclosure that needs more than one page under
// a narrow font needs at least as many under a wide one.
const (
	noticeFontCellW   = 5
	noticeFontCellH   = 15
	noticeFontAdvance = 3
	noticeFontSpace   = 2
	noticeFontSpacing = 1
	noticeFontRecords = 224
)

// loadNoticeFont is a measurable font with no install behind it. Its records
// are distinguishable from each other so that a wrap measured through it is
// arithmetic rather than an accident of one glyph's width.
func loadNoticeFont() *text.Font {
	f := &text.Font{Spacing: noticeFontSpacing, Glyphs: make([]text.Glyph, noticeFontRecords)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: noticeFontCellW, Height: noticeFontCellH,
			Pixels: make([]text.Pixel, noticeFontCellW*noticeFontCellH), Advance: noticeFontAdvance}
		if k == 0 {
			g.Advance = noticeFontSpace
		} else {
			g.Pixels[(k%noticeFontCellH)*noticeFontCellW+(k/noticeFontCellH)%noticeFontCellW] =
				text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// loadNoticeViewer is a viewer that can measure the dialogue notice: the
// authored layouts the game ships and the font above.
func loadNoticeViewer() *ui.Viewer {
	v := &ui.Viewer{}
	v.SetFont(loadNoticeFont())
	v.SetNoticeLayouts(ui.AuthoredDialogueLayout(), ui.AuthoredOutcomeLayout())
	return v
}

// TestLiveNoticePagesReportsEveryPageOfAnOpenLoadNotice is the instrument
// cmd/savecheck reads. LiveNotice reports the string that was pushed, which is
// complete whatever the window drew; this reports the whole notice and how each
// page measures against the window.
func TestLiveNoticePagesReportsEveryPageOfAnOpenLoadNotice(t *testing.T) {
	v := loadNoticeViewer()
	mw := &mapWorld{mission: &missionNotices{}, view: v}
	f := &FrontEnd{}
	f.live = mw

	if got := f.LiveNoticePages(); len(got) != 0 {
		t.Errorf("no notice open: LiveNoticePages = %v, want none", got)
	}

	paras := []string{"First paragraph of a notice.", "Second paragraph of a notice, long enough to wrap onto more than one line of the dialogue window so that the measured pages are not trivial.", "Third paragraph."}
	pages := v.DialoguePages(paras)
	mw.openLoadNotice(pages)

	got := f.LiveNoticePages()
	if len(got) != len(pages) {
		t.Fatalf("LiveNoticePages reports %d page(s), want %d", len(got), len(pages))
	}
	for i, p := range got {
		if p.Text != pages[i] {
			t.Errorf("page %d text = %q, want %q", i+1, p.Text, pages[i])
		}
		if p.Produced == 0 {
			t.Errorf("page %d produces no line", i+1)
		}
		if p.Produced != p.Drawn {
			t.Errorf("page %d: %d produced, %d drawn", i+1, p.Produced, p.Drawn)
		}
	}
}
