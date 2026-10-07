package ui

import (
	"strings"

	"againrom/pkg/render/text"
)

// NoticeMaxLines is how many wrapped lines the layout's text area draws, which
// is NoticeLayoutOf's own clamp stated on its own (`DLG-WRAP-009`:
// `min(height/pitch, line count)`).
//
// It is exported because a caller that has to fit text into this window needs
// the number BEFORE it composes the text, and reading it back off
// NoticeLayoutOf means composing something and seeing what survived.
//
// Zero means nothing can be drawn at all: no font, a font with no records, or
// a pitch the area does not hold one line of.
func NoticeMaxLines(l NoticeLayout, f *text.Font) int {
	if f == nil || len(f.Glyphs) == 0 {
		return 0
	}
	pitch := f.Height() + l.Pitch
	if pitch <= 0 {
		return 0
	}
	n := l.Text.Dy() / pitch
	if n < 0 {
		return 0
	}
	return n
}

// NoticePagesOf packs paras into the fewest notice pages this layout draws
// WHOLE — every page it returns has at most NoticeMaxLines wrapped lines,
// so NoticeLayoutOf clamps none of them away (1032 return 1).
//
// THE CLAMP IS NOT A DEFECT AND PAGING IS THE ANSWER TO IT. The original's
// control scrolls and this window has no scrollbar, so NoticeLayoutOf simply
// does not draw the lines past the area, with nothing on screen saying so. A
// notice longer than the area's lines therefore ends mid-sentence and the rest
// is unreachable: measured on the real EN font at the earlier dialogue geometry,
// a version-53 save's load disclosure needed 16 lines and 8 were drawn. Text
// that has to be read in full is paged through the window's own dismiss
// instead, which is the gesture a multi-part mission dialogue already teaches.
//
// A PARAGRAPH IS THE UNIT AND A PAGE HOLDS WHOLE ONES where it can. Paragraphs
// are packed in order and joined with a single space, so a page reads as
// ordinary prose; the caller writes each paragraph as a complete sentence for
// that reason. A paragraph too long for one page on its own is split at its own
// wrapped line boundaries, which is the only remaining way to show it.
//
// It never returns an empty page and never drops a paragraph. With nothing
// measurable — no font, or an area that draws no line — it returns the whole
// text as ONE page rather than splitting blindly: a caller with no font draws
// no notice at all (Viewer.NoticeOpen), so there is nothing to page.
func NoticePagesOf(l NoticeLayout, f *text.Font, paras []string) []string {
	var kept []string
	for _, p := range paras {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	max := NoticeMaxLines(l, f)
	if max <= 0 {
		return []string{strings.Join(kept, " ")}
	}

	// Each paragraph becomes one or more CHUNKS, each of which fits a page on
	// its own. A chunk's text is its own wrapped lines joined back with single
	// spaces: the wrap is greedy, so re-wrapping that text at the same width
	// reproduces those lines, or fewer where the dialogue wrap measures the last
	// word of a paragraph without its trailing space.
	type chunk struct {
		text  string
		lines int
	}
	var chunks []chunk
	for _, p := range kept {
		lines := noticeWrap(l, f, p)
		if len(lines) == 0 {
			continue
		}
		for i := 0; i < len(lines); i += max {
			j := i + max
			if j > len(lines) {
				j = len(lines)
			}
			chunks = append(chunks, chunk{strings.Join(lines[i:j], " "), j - i})
		}
	}
	if len(chunks) == 0 {
		return nil
	}

	// GREEDY PACKING IS SAFE HERE and is not merely close enough for the plain
	// wrap. Wrapping the concatenation of two chunks never produces more lines
	// than wrapping each alone, because the wrap takes the longest prefix that
	// fits at every line: the joined text reaches each of the first chunk's own
	// break points with at least as much room. So a page whose chunk line counts
	// sum to at most max draws in at most max lines.
	//
	// THE DIALOGUE WRAP IS NOT SYMMETRIC: it measures a paragraph's last word
	// without its trailing space, so joining a chunk can move that word down a
	// line and the sum is one short. Every join is therefore also checked by
	// wrapping the page as it will be drawn.
	var pages []string
	var cur []string
	used := 0
	for _, c := range chunks {
		if used > 0 && (used+c.lines > max || len(noticeWrap(l, f, strings.Join(cur, " ")+" "+c.text)) > max) {
			pages = append(pages, strings.Join(cur, " "))
			cur, used = nil, 0
		}
		cur = append(cur, c.text)
		used += c.lines
	}
	if len(cur) > 0 {
		pages = append(pages, strings.Join(cur, " "))
	}
	return pages
}

// DialoguePages packs paras into pages of THIS viewer's dialogue notice,
// with its own font and its own layout (1032 return 1).
//
// It exists because pkg/ui keeps its font and its two layouts unexported, and
// the tier that composes a load disclosure is pkg/game. Handing that tier a
// getter for the font would make the layout a caller's business; handing it the
// pages keeps the geometry here, where NoticeLayoutOf's clamp already lives.
//
// The layout is resolved WITHOUT a portrait, which is the shape every notice
// this build pages through opens in: openLoadNotice pushes no speaker and
// SetDialogue writes no face for one.
func (v *Viewer) DialoguePages(paras []string) []string {
	return NoticePagesOf(v.noticeLayouts[NoticeDialogue].WithPortrait(false), v.font, paras)
}

// DialogueClip measures s against the dialogue notice this viewer draws: how
// many wrapped lines the text produces, and how many of those the window's own
// text area draws.
//
// IT IS THE INSTRUMENT THAT SEES A CLIPPED NOTICE (1032 return 1). The two
// numbers are equal for text that fits and differ for text that does not,
// and nothing else in this tree reports the second one: NoticeState hands
// back the raw string that was pushed, which is complete whatever the window
// drew.
//
// It resolves the layout WITHOUT a portrait, the shape openLoadNotice pushes.
func (v *Viewer) DialogueClip(s string) (produced, drawn int) {
	l := v.noticeLayouts[NoticeDialogue].WithPortrait(false)
	if v.font == nil || len(v.font.Glyphs) == 0 {
		return 0, 0
	}
	produced = len(noticeWrap(l, v.font, s))
	drawn = produced
	if n := NoticeMaxLines(l, v.font); n < drawn {
		drawn = n
	}
	return produced, drawn
}
