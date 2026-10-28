package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/text"
)

// A menu notice window is one kind with one builder: a window framed like the
// page under it, its rows, answers and shape all data. A row is a sequence of
// runs; an alert run is drawn in the alert ink, every other run in the page's
// ink. A long row wraps, so no row runs off the window.

// inkRun is one run of a notice row; Alert draws it in the alert ink.
type inkRun struct {
	Text  string
	Alert bool
}

// noticeRow is one row of a notice window.
type noticeRow []inkRun

// noticeWindow is one open notice window: its words, rows, answers and shape,
// the first visible line, the answer press latch and the bar's held gesture.
type noticeWindow struct {
	Title, Hint string
	// Answers are the buttons, left to right; Enter takes the first.
	Answers []string
	Rows    []noticeRow
	Shape   NoticeShape
	top     int
	press   buttonLatch
	bar     scrollBarInput
}

// NoticeShape is a notice window's geometry. The zero shape fills the page
// less noticeWindowInset, its answers 96x24 and 12 pixels above its bottom.
// A shape with a Size is that window centred on the page, with no title, list
// or hint: its rows wrap to TextWidth and each line is centred, one every
// Pitch pixels from TextTop; its answers are Answer in size, AnswerGap apart
// and centred AnswerBottom pixels above its bottom.
type NoticeShape struct {
	Size                      image.Point
	TextTop, TextWidth, Pitch int
	Answer                    image.Point
	AnswerGap, AnswerBottom   int
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

// noticeWindowLayout is a notice window's geometry for one font; Pitch is
// positive in a text window.
type noticeWindowLayout struct {
	Body, Title, Hint, Text image.Rectangle
	Answers                 []image.Rectangle
	List                    listBox
	Pitch                   int
}

// layout lays w out over panel, the page under it, by its shape.
func (w *noticeWindow) layout(f *text.Font, panel image.Rectangle) noticeWindowLayout {
	s := w.Shape
	if s.Size == (image.Point{}) {
		return newNoticeWindowLayout(f, panel, len(w.Answers))
	}
	at := panel.Min.Add(panel.Size().Sub(s.Size).Div(2))
	b := image.Rectangle{Min: at, Max: at.Add(s.Size)}
	answers := noticeAnswers(b, s.Answer, s.AnswerGap, s.AnswerBottom, len(w.Answers))
	x := b.Min.X + (s.Size.X-s.TextWidth)/2
	bottom := b.Max.Y
	if len(answers) > 0 {
		bottom = answers[0].Min.Y
	}
	return noticeWindowLayout{Body: b, Text: image.Rect(x, b.Min.Y+s.TextTop, x+s.TextWidth, bottom), Answers: answers, Pitch: s.Pitch}
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
func (l noticeWindowLayout) answerAt(p image.Point) (int, bool) {
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
func newNoticeWindowLayout(f *text.Font, panel image.Rectangle, answers int) noticeWindowLayout {
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
	return noticeWindowLayout{
		Body:    b,
		Title:   image.Rect(b.Min.X+8, b.Min.Y+10, b.Max.X-8, b.Min.Y+34),
		List:    list,
		Hint:    image.Rect(b.Min.X+m, list.Rect.Max.Y+8, b.Max.X-m, ok.Min.Y-4),
		Answers: buttons,
	}
}

// noticeLines wraps every row to the list's text width, or a text window's
// width with no indent. display turns a run's text into the bytes the font
// draws.
func (w *noticeWindow) noticeLines(f *text.Font, l noticeWindowLayout, display func(string) string) []noticeLine {
	width, indent := l.List.Rect.Dx()-2*listTextX, noticeWrapIndent
	if l.Pitch > 0 {
		width, indent = l.Text.Dx(), 0
	}
	var out []noticeLine
	for _, row := range w.Rows {
		out = append(out, wrapNoticeRow(f, row, width, indent, display)...)
	}
	return out
}

// wrapNoticeRow breaks a row at spaces into lines no wider than width; the
// lines after the first are indented by indent. A word wider than a line is
// broken where it reaches the edge.
func wrapNoticeRow(f *text.Font, row noticeRow, width, indent int, display func(string) string) []noticeLine {
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
		cur, x, gap = noticeLine{indent: indent}, 0, nil
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

// step runs one tick of the window's input and reports the answer that
// closes it: Enter takes the first, a released press the answer it began on.
// In a list window the arrow, page, Home and End keys, the wheel and the bar
// scroll the list. Escape is the caller's.
func (w *noticeWindow) step(in appInput, p image.Point, ok bool, l noticeWindowLayout, lines int) (int, bool) {
	if in.Unfocused {
		w.press.Clear()
		w.bar.reset()
		return 0, false
	}
	if in.Enter {
		return 0, true
	}
	if l.Pitch == 0 && w.scrollStep(in, p, ok, l, lines) {
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
func (w *noticeWindow) scrollStep(in appInput, p image.Point, ok bool, l noticeWindowLayout, lines int) bool {
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
		return true
	}
	return w.bar.active()
}

// compose draws the window laid out by l over dst, then its answers.
func (w *noticeWindow) compose(dst *image.RGBA, f *text.Font, l noticeWindowLayout, art *DialogFrame, frames []*image.RGBA, display func(string) string, pointer image.Point, pointerOK bool) {
	lines := w.noticeLines(f, l, display)
	if l.Pitch > 0 {
		drawFrame(dst, windowFrame(l.Body, art))
		w.composeText(dst, f, l, lines)
	} else {
		w.composeList(dst, f, l, art, frames, display, lines, pointer, pointerOK)
	}
	for i, r := range l.Answers {
		label := ""
		if i < len(w.Answers) {
			label = w.Answers[i]
		}
		inside := pointerOK && pointer.In(r)
		drawPushButton(dst, f, pushButton{Rect: r, Label: label, Hover: inside, Inside: inside, Pressed: w.press.Pressed(i)})
	}
}

// composeText draws a text window's lines, each centred, while a line fits
// above the answers.
func (w *noticeWindow) composeText(dst *image.RGBA, f *text.Font, l noticeWindowLayout, lines []noticeLine) {
	for i, ln := range lines {
		y := l.Text.Min.Y + i*l.Pitch
		if y+f.Height() > l.Text.Max.Y {
			break
		}
		width := 0
		for _, run := range ln.runs {
			width += f.Advance(run.Text)
		}
		x := (l.Text.Min.X+l.Text.Max.X)/2 - width/2
		for _, run := range ln.runs {
			ink := townShellText
			if run.Alert {
				ink = noticeAlertInk
			}
			f.Draw(dst, run.Text, x, y, ink)
			x += f.Advance(run.Text)
		}
	}
}

// composeList draws a list window's frame, title, rows in the list well with
// the bar, and hint.
func (w *noticeWindow) composeList(dst *image.RGBA, f *text.Font, l noticeWindowLayout, art *DialogFrame, frames []*image.RGBA, display func(string) string, lines []noticeLine, pointer image.Point, pointerOK bool) {
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
}
