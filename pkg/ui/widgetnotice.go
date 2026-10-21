package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/text"
)

// A menu notice window is one kind with one builder: a window framed like the
// page under it, a title, a list of rows that scrolls in the shared list well
// with the shared bar, a hint below the list and one OK button. A row is a
// sequence of runs; an alert run is drawn in the alert ink and every other
// run in the page's ink. A long row wraps inside the well; the list scrolls
// by wrapped lines, so no row runs off the window.

// inkRun is one run of a notice row; Alert draws it in the alert ink.
type inkRun struct {
	Text  string
	Alert bool
}

// noticeRow is one row of a notice window.
type noticeRow []inkRun

// noticeWindow is one open notice window: its words and rows, the first
// visible line, the OK press latch and the bar's held gesture.
type noticeWindow struct {
	Title, Hint, OK string
	Rows            []noticeRow
	top             int
	press           buttonLatch
	bar             scrollBarInput
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

// noticeWindowLayout is a notice window's geometry for one font.
type noticeWindowLayout struct {
	Body, Title, Hint, OK image.Rectangle
	List                  listBox
}

// newNoticeWindowLayout lays a window out over panel, the page under it: the
// body is the panel less noticeWindowInset, so the page's own buttons lie
// under the window. The list takes the rows the body leaves between the title
// and the hint.
func newNoticeWindowLayout(f *text.Font, panel image.Rectangle) noticeWindowLayout {
	b := panel.Inset(noticeWindowInset)
	m := noticeWindowMargin
	cx := (b.Min.X + b.Max.X) / 2
	ok := image.Rect(cx-noticeOKWidth/2, b.Max.Y-12-noticeOKHeight, cx+noticeOKWidth/2, b.Max.Y-12)
	hintHeight := noticeHintLines * (listFallbackPitch - listPitchExtra + 1)
	if f != nil && f.Height() > 0 {
		hintHeight = noticeHintLines * (f.Height() + 1)
	}
	top := b.Min.Y + 40
	probe := newListBox(image.Rect(0, 0, 1, 0), 1, f)
	rows := max((ok.Min.Y-8-hintHeight-8-listBottomExtra-top)/probe.Pitch, 1)
	list := newListBox(image.Rect(b.Min.X+m, top, b.Max.X-m-widgetSpriteSize, top), rows, f)
	return noticeWindowLayout{
		Body:  b,
		Title: image.Rect(b.Min.X+8, b.Min.Y+10, b.Max.X-8, b.Min.Y+34),
		List:  list,
		Hint:  image.Rect(b.Min.X+m, list.Rect.Max.Y+8, b.Max.X-m, ok.Min.Y-4),
		OK:    ok,
	}
}

// noticeLines wraps every row to the list's text width. display turns a run's
// text into the bytes the font draws.
func (w *noticeWindow) noticeLines(f *text.Font, l noticeWindowLayout, display func(string) string) []noticeLine {
	width := l.List.Rect.Dx() - 2*listTextX
	var out []noticeLine
	for _, row := range w.Rows {
		out = append(out, wrapNoticeRow(f, row, width, display)...)
	}
	return out
}

// wrapNoticeRow breaks a row at spaces into lines no wider than width; the
// lines after the first are indented. A word wider than a line is broken
// where it reaches the edge.
func wrapNoticeRow(f *text.Font, row noticeRow, width int, display func(string) string) []noticeLine {
	type piece struct {
		text         string
		alert, space bool
	}
	var pieces []piece
	for _, r := range row {
		s := display(r.Text)
		for s != "" {
			space := s[0] == ' '
			n := strings.IndexFunc(s, func(c rune) bool { return (c == ' ') != space })
			if n < 0 {
				n = len(s)
			}
			pieces = append(pieces, piece{s[:n], r.Alert, space})
			s = s[n:]
		}
	}
	var out []noticeLine
	cur := noticeLine{}
	x := 0
	var gap []piece
	add := func(p piece) {
		if n := len(cur.runs); n > 0 && cur.runs[n-1].Alert == p.alert {
			cur.runs[n-1].Text += p.text
		} else {
			cur.runs = append(cur.runs, inkRun{Text: p.text, Alert: p.alert})
		}
		x += f.Advance(p.text)
	}
	flush := func() {
		out = append(out, cur)
		cur, x, gap = noticeLine{indent: noticeWrapIndent}, 0, nil
	}
	for _, p := range pieces {
		if p.space {
			if len(cur.runs) > 0 {
				gap = append(gap, p)
			}
			continue
		}
		avail := width - cur.indent
		gapWidth := 0
		for _, g := range gap {
			gapWidth += f.Advance(g.text)
		}
		if len(cur.runs) > 0 && x+gapWidth+f.Advance(p.text) > avail {
			flush()
			avail = width - cur.indent
		} else {
			for _, g := range gap {
				add(g)
			}
			gap = nil
		}
		for len(cur.runs) == 0 && f.Advance(p.text) > avail-x {
			n := 1
			for n < len(p.text) && f.Advance(p.text[:n+1]) <= avail {
				n++
			}
			add(piece{p.text[:n], p.alert, false})
			p.text = p.text[n:]
			flush()
			avail = width - cur.indent
		}
		if p.text != "" {
			add(p)
		}
	}
	if len(cur.runs) > 0 || len(out) == 0 {
		out = append(out, cur)
	}
	return out
}

// noticeMaxTop is the last first line that still fills a list of rows.
func noticeMaxTop(lines, rows int) int { return max(lines-rows, 0) }

// scroll moves the first visible line by d, inside the list.
func (w *noticeWindow) scroll(d, lines, rows int) {
	w.top = min(max(w.top+d, 0), noticeMaxTop(lines, rows))
}

// noticeBar is the window's bar: its position is the first visible line.
func noticeBar(l noticeWindowLayout, top, lines int) vScrollBar {
	last := noticeMaxTop(lines, l.List.Rows)
	return vScrollBar{Rect: l.List.Bar(), Pos: top, Count: last + 1, Disabled: last == 0}
}

// step runs one tick of the window's input and reports whether it closes:
// Enter or a released OK press closes it; the arrow, page, Home and End keys,
// the wheel and the bar scroll the list. Escape is the caller's.
func (w *noticeWindow) step(in appInput, p image.Point, ok bool, l noticeWindowLayout, lines int) bool {
	if in.Unfocused {
		w.press.Clear()
		w.bar.reset()
		return false
	}
	if in.Enter {
		return true
	}
	rows := l.List.Rows
	switch {
	case in.Up:
		w.scroll(-1, lines, rows)
	case in.Down:
		w.scroll(1, lines, rows)
	case in.PageUp:
		w.scroll(-rows, lines, rows)
	case in.PageDown:
		w.scroll(rows, lines, rows)
	case in.Home:
		w.top = 0
	case in.End:
		w.top = noticeMaxTop(lines, rows)
	case in.WheelY > 0:
		w.scroll(-3, lines, rows)
	case in.WheelY < 0:
		w.scroll(3, lines, rows)
	}
	if req, pos := w.bar.step(noticeBar(l, w.top, lines), p, ok, in); req != barNone {
		switch req {
		case barLineUp:
			w.scroll(-1, lines, rows)
		case barLineDown:
			w.scroll(1, lines, rows)
		case barPageUp:
			w.scroll(-rows, lines, rows)
		case barPageDown:
			w.scroll(rows, lines, rows)
		case barSetPos:
			w.top = 0
			w.scroll(pos, lines, rows)
		}
		return false
	}
	if w.bar.active() {
		return false
	}
	if in.PrimaryPressed {
		w.press.Press(0, ok && p.In(l.OK))
		return false
	}
	if in.PrimaryReleased {
		_, activated := w.press.Release(0, ok && p.In(l.OK))
		return activated
	}
	return false
}

// compose draws the window laid out by l over dst: frame, title, the rows in the list well
// with the bar, the hint and the OK button.
func (w *noticeWindow) compose(dst *image.RGBA, f *text.Font, l noticeWindowLayout, art *DialogFrame, frames []*image.RGBA, display func(string) string, pointer image.Point, pointerOK bool) {
	lines := w.noticeLines(f, l, display)
	w.scroll(0, len(lines), l.List.Rows)
	drawFrame(dst, windowFrame(l.Body, art))
	drawTownShellText(dst, f, display(w.Title), l.Title, loadSelectedText)
	drawEditField(dst, listWell(l.List), nil)
	for i := 0; i < l.List.Rows && w.top+i < len(lines); i++ {
		ln := lines[w.top+i]
		r := l.List.Row(i)
		clip := dst.SubImage(r.Inset(1)).(*image.RGBA)
		x := r.Min.X + listTextX + ln.indent
		for _, run := range ln.runs {
			ink := listRowInk
			if run.Alert {
				ink = noticeAlertInk
			}
			f.Draw(clip, run.Text, x, r.Min.Y+listTextY, ink)
			x += f.Advance(run.Text)
		}
	}
	drawVScrollBar(dst, frames, noticeBar(l, w.top, len(lines)).withPointer(pointer, pointerOK))
	for i, line := range NoticeLines(f, display(w.Hint), l.Hint.Dx()) {
		if i >= noticeHintLines {
			break
		}
		f.Draw(dst, line, l.Hint.Min.X, l.Hint.Min.Y+i*(f.Height()+1), townShellText)
	}
	inside := pointerOK && pointer.In(l.OK)
	drawPushButton(dst, f, pushButton{Rect: l.OK, Label: w.OK, Hover: inside, Inside: inside, Pressed: w.press.Pressed(0)})
}
