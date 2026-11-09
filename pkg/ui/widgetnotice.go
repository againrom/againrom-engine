package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// A menu notice window is one description (the popup spec's window fields):
// a window framed like the page under it, its rows, answers and shape all
// data. A row is a sequence of runs; an alert run is drawn in the alert ink,
// every other run in the page's ink. A long row wraps, so no row runs off the
// window.

// inkRun is one run of a notice row; Alert draws it in the alert ink.
type inkRun struct {
	Text  string
	Alert bool
}

// noticeRow is one row of a notice window.
type noticeRow []inkRun

// dialogRun is one open notice window: its description, the first visible
// line, the answer press latch and the bar's held gesture.
type dialogRun struct {
	popupSpec
	top   int
	press buttonLatch
	bar   scrollBarInput
}

// noticeLine is one wrapped line of a row: its runs, as the font draws them,
// and its indent from the row's start.
type noticeLine struct {
	runs   []inkRun
	indent int
}

const (
	noticeHintLines    = 3
	noticeWrapIndent   = 12
	noticeWindowMargin = 16
	noticeWindowInset  = 12
	noticeOKWidth      = 96
	noticeOKHeight     = 24
)

var noticeAlertInk = color.RGBA{236, 76, 60, 255}

// spec is the window's description as drawn: over panel, in art, with the
// display and the first line the state holds.
func (w *dialogRun) spec(panel image.Rectangle, art *DialogFrame, display func(string) string) popupSpec {
	s := w.popupSpec
	s.Panel, s.Frame, s.Display, s.Top, s.Latch = panel, art, display, w.top, &w.press
	return s
}

// layout lays w out over panel, the page under it, by its shape.
func (w *dialogRun) layout(f *text.Font, panel image.Rectangle) popupLayout {
	return layoutPopup(w.spec(panel, nil, nil), f, f)
}

// noticeAnswers places n answers of size, gap apart, centred in b and bottom
// pixels above its bottom.
func noticeAnswers(b image.Rectangle, size image.Point, gap, bottom, n int) []image.Rectangle {
	x := (b.Min.X+b.Max.X)/2 - (n*size.X+(n-1)*gap)/2
	y := b.Max.Y - bottom - size.Y
	out := make([]image.Rectangle, n)
	for i := range out {
		out[i] = image.Rect(x, y, x+size.X, y+size.Y)
		x += size.X + gap
	}
	return out
}

// answerAt is the answer under p.
func (l popupLayout) answerAt(p image.Point) (int, bool) {
	for i, r := range l.Answers {
		if p.In(r) {
			return i, true
		}
	}
	return 0, false
}

// newNoticeWindowLayout lays a list window with answers answers out over
// panel, the page under it: the body is the panel less noticeWindowInset, so
// the page's own buttons lie under the window. The list takes the rows the
// body leaves between the title and the hint.
func newNoticeWindowLayout(f *text.Font, panel image.Rectangle, answers int) popupLayout {
	b := panel.Inset(noticeWindowInset)
	m := noticeWindowMargin
	buttons := noticeAnswers(b, image.Pt(noticeOKWidth, noticeOKHeight), m, 12, max(answers, 1))
	ok := buttons[0]
	hintHeight := noticeHintLines * (listFallbackPitch - listPitchExtra + 1)
	if f != nil && f.Height() > 0 {
		hintHeight = noticeHintLines * (f.Height() + 1)
	}
	top := b.Min.Y + 40
	probe := newListBox(image.Rect(0, 0, 1, 0), 1, f)
	rows := max((ok.Min.Y-8-hintHeight-8-listBottomExtra-top)/probe.Pitch, 1)
	list := newListBox(image.Rect(b.Min.X+m, top, b.Max.X-m-widgetSpriteSize, top), rows, f)
	return popupLayout{
		Rect:       b,
		Body:       b,
		HeadingBox: image.Rect(b.Min.X+8, b.Min.Y+10, b.Max.X-8, b.Min.Y+34),
		List:       list,
		Hint:       image.Rect(b.Min.X+m, list.Rect.Max.Y+8, b.Max.X-m, ok.Min.Y-4),
		Answers:    buttons,
	}
}

// noticeLines wraps every row to the list's text width, or a text window's
// width with no indent. display turns a run's text into the bytes the font
// draws.
func (w *dialogRun) noticeLines(f *text.Font, l popupLayout, display func(string) string) []noticeLine {
	s := w.popupSpec
	s.Display = display
	return noticeWindowLines(s, f, l)
}

// noticeRowLines breaks a row at spaces into lines no wider than width; the
// lines after the first are indented by indent. A word wider than a line is
// broken where it reaches the edge. display turns a run's text into the bytes
// the font draws.
func noticeRowLines(f *text.Font, row noticeRow, width, indent int, display func(string) string) []noticeLine {
	runs := make([]inkRun, len(row))
	for i, r := range row {
		runs[i] = inkRun{Text: display(r.Text), Alert: r.Alert}
	}
	lines := noticeRowRule.lines(lineBox{Measure: advanceOf(f), Width: width, Indent: indent}, runs...)
	out := make([]noticeLine, len(lines))
	for i, l := range lines {
		out[i] = noticeLine{runs: l.Runs, indent: l.Indent}
	}
	return out
}

// noticeMaxTop is the last first line that still fills a list of rows.
func noticeMaxTop(lines, rows int) int { return max(lines-rows, 0) }

// scroll moves the first visible line by d, inside the list.
func (w *dialogRun) scroll(d, lines, rows int) {
	w.top = min(max(w.top+d, 0), noticeMaxTop(lines, rows))
}

// noticeBar is the window's bar: its position is the first visible line.
func noticeBar(l popupLayout, top, lines int) vScrollBar {
	last := noticeMaxTop(lines, l.List.Rows)
	return vScrollBar{Rect: l.List.Bar(), Pos: top, Count: last + 1, Disabled: last == 0}
}

// step runs one tick of the window's input and reports the answer that
// closes it: Enter takes the first, a released press the answer it began on.
// In a list window the arrow, page, Home and End keys, the wheel and the bar
// scroll the list. Escape is the caller's.
func (w *dialogRun) step(in appInput, p image.Point, ok bool, l popupLayout, lines int) (int, bool) {
	if in.Unfocused {
		w.press.Clear()
		w.bar.reset()
		return 0, false
	}
	if in.Enter {
		return 0, true
	}
	if l.LinePitch == 0 && w.scrollStep(in, p, ok, l, lines) {
		return 0, false
	}
	at, on := l.answerAt(p)
	if in.PrimaryPressed {
		w.press.Press(at, ok && on)
		return 0, false
	}
	if in.PrimaryReleased {
		return w.press.Release(at, ok && on)
	}
	return 0, false
}

// scrollStep scrolls a list window by keys, wheel and bar and reports whether
// the bar took the input.
func (w *dialogRun) scrollStep(in appInput, p image.Point, ok bool, l popupLayout, lines int) bool {
	rows := l.List.Rows
	wheel := noticeList.wheelStep(in)
	switch {
	case in.Up || in.Down:
		w.scroll(noticeList.keyStep(in.Up, in.Down), lines, rows)
	case in.PageUp:
		w.scroll(-noticeList.pageUp(rows), lines, rows)
	case in.PageDown:
		w.scroll(noticeList.pageDown(rows), lines, rows)
	case in.Home:
		w.top = 0
	case in.End:
		w.top = noticeMaxTop(lines, rows)
	case wheel != 0:
		w.scroll(wheel, lines, rows)
	}
	if req, pos := w.bar.step(noticeBar(l, w.top, lines), p, ok, in); req != barNone {
		if req == barSetPos {
			w.top = 0
			w.scroll(pos, lines, rows)
		} else if step, ok := noticeList.barStep(req, rows); ok {
			w.scroll(step, lines, rows)
		}
		return true
	}
	return w.bar.active()
}

// compose draws the window over dst, through the one dialog painter, and
// keeps a scrolled list's first line inside its rows.
func (w *dialogRun) compose(dst *image.RGBA, f *text.Font, panel image.Rectangle, art *DialogFrame, frames []*image.RGBA, display func(string) string, pointer image.Point, pointerOK bool) {
	s := w.spec(panel, art, display)
	l := layoutPopup(s, f, f)
	if l.LinePitch == 0 {
		w.top = l.TopClamped
	}
	drawPopup(dst, f, f, frames, s, l, popupInk{}, pointer, pointerOK)
}
