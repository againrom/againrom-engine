package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// Dialog names the modal dialog description (popupSpec) for the packages that
// compose pages. A value is laid out by layoutPopup and painted by drawPopup.
// Geometry comes from the rows, or from given rectangles (Box and the fields
// beside it); the frame is art or colours by value, a dialog's parts are data
// the painter reads the same way for every value. The editor's popups, the
// notice windows, the gold editor, the game menu, the save page's frames and
// buttons, the authored notices, the help panel and town talk are values of it.
type Dialog = popupSpec

// dialogLine is one line a dialog draws: its runs left to right from At,
// inside Clip when that is set, a flat shadow Shadow pixels right and down
// before each run. Justify spreads the words over Width from At.X, counted
// from the design space's Origin (a justified dialogue line).
type dialogLine struct {
	Runs          []inkRun
	At            image.Point
	Clip          image.Rectangle
	Ink, AlertInk color.RGBA
	Shadow        int
	Justify       bool
	Width, Origin int
	Arithmetic    DialogueArithmetic
	// Dim dims this rectangle after the line is drawn (a disabled row).
	Dim image.Rectangle
}

// dialogFrame is one frame of a dialog by value; Body draws the frame's body
// without its shadows, which the scene under the dialog receives apart.
type dialogFrame struct {
	Spec frameSpec
	Body bool
}

// dialogEdit is a dialog's edit field: the field's look and the line drawn in
// it, none when Line is nil.
type dialogEdit struct {
	Field editField
	Line  *dialogLine
}

// drawDialogLines draws lines in order: one pass reads every line the same
// way.
func drawDialogLines(dst *image.RGBA, f *text.Font, lines []dialogLine) {
	if f == nil {
		return
	}
	for _, ln := range lines {
		target := dst
		if !ln.Clip.Empty() {
			target = dst.SubImage(ln.Clip).(*image.RGBA)
		}
		since := markCapture()
		if ln.Justify && len(ln.Runs) > 0 {
			drawDialogueLine(target, f, dialogueLine{text: ln.Runs[0].Text, justify: true}, ln.Origin, ln.At.X, ln.At.Y, ln.Width, ln.Arithmetic, ln.Ink)
		} else {
			x := ln.At.X
			for _, run := range ln.Runs {
				ink := ln.Ink
				if run.Alert {
					ink = ln.AlertInk
				}
				if ln.Shadow > 0 {
					f.DrawFlat(target, run.Text, x+ln.Shadow, ln.At.Y+ln.Shadow, messageShadowColor)
				}
				f.Draw(target, run.Text, x, ln.At.Y, ink)
				x += f.Advance(run.Text)
			}
		}
		if !ln.Dim.Empty() {
			dimDisabledRow(dst, ln.Dim, since)
		}
	}
}

// drawDialogParts paints a given dialog's parts over its frames: the edit
// field, the buttons, the scroll bar, the lines and the answers, each read
// the same way for every value. The rows form has none of them.
func drawDialogParts(dst *image.RGBA, f *text.Font, frames []*image.RGBA, s popupSpec, l popupLayout, pointer image.Point, pointerOK bool) {
	if l.Edit != nil {
		var label func(image.Point)
		if line := l.Edit.Line; line != nil && f != nil {
			label = func(at image.Point) {
				ln := *line
				ln.At = at
				drawDialogLines(dst, f, []dialogLine{ln})
			}
		}
		drawEditField(dst, l.Edit.Field, label)
	}
	for _, b := range l.Pushes {
		drawPushButton(dst, f, b)
	}
	if l.Bar != nil {
		art := frames
		if l.BarFrames != nil {
			art = l.BarFrames
		}
		bar := *l.Bar
		if l.BarLive {
			bar = bar.withPointer(pointer, pointerOK)
		}
		drawVScrollBar(dst, art, bar)
	}
	drawDialogLines(dst, f, l.Lines)
	for i, r := range l.Answers {
		label := ""
		if i < len(s.Answers) {
			label = s.Answers[i]
		}
		inside := pointerOK && pointer.In(r)
		b := pushButton{Rect: r, Label: label, Hover: inside, Inside: inside}
		if s.Latch != nil {
			b.Pressed = s.Latch.Pressed(i)
		}
		drawPushButton(dst, f, b)
	}
}

// drawDialog lays s out and paints it over dst with no pointer: the one
// layout and the one painter for a value whose parts are given.
func drawDialog(dst *image.RGBA, f *text.Font, s popupSpec) {
	drawPopup(dst, f, f, nil, s, layoutPopup(s, f, f), popupInk{}, image.Point{}, false)
}

// dialogGivenForm reports a description whose geometry is given rather than
// derived from rows.
func dialogGivenForm(s popupSpec) bool { return !s.Box.Empty() || !s.Panel.Empty() }

// layoutDialogGiven is the layout of a description with given geometry: a
// notice box, authored in the design space and composed at the picture's
// origin, or a notice window over its Panel.
func layoutDialogGiven(s popupSpec, f *text.Font) popupLayout {
	if !s.Panel.Empty() {
		return layoutNoticeWindow(s, f)
	}
	return layoutNoticeBox(s, f)
}

// noticeStyleRow is what a notice style sets for the layout and the painter,
// as frameRow is for a frame kind: the line rule and what its visible lines
// trim, the justified paragraphs, indent and flat shadow of a body line, the
// button ramp, the band kept inside the box for the frame's shadows, the
// default body ink, whether the frame's shadows fall on the scene under the
// notice, whether the second button is drawn, whether a disabled first button
// refuses a press, and the portrait pane's procedure.
type noticeStyleRow struct {
	rule         lineRule
	trim         string
	paragraphs   bool
	indent       int
	textShadow   int
	ramp         pushButtonRamp
	inset        int
	ink          color.RGBA
	frameShadows bool
	secondary    bool
	refuseOff    bool
	portrait     func(*image.RGBA, popupSpec, *image.RGBA)
}

var noticeStyles = [...]noticeStyleRow{
	NoticeStylePlain: {rule: noticeRule, secondary: true, portrait: drawPlainPortrait},
	NoticeStyleDialogue: {rule: dialogueRule, trim: " \t", paragraphs: true, indent: dialogueIndent, textShadow: dialogueTextShadow,
		ramp: pushButtonDialogue, inset: noticeShadow, ink: dialogueInk, frameShadows: true, refuseOff: true, portrait: drawDialoguePortrait},
}

// styleRow is the value's row of noticeStyles; an unknown style is plain.
func (s popupSpec) styleRow() noticeStyleRow {
	if int(s.Style) < len(noticeStyles) {
		return noticeStyles[s.Style]
	}
	return noticeStyles[NoticeStylePlain]
}

// layoutNoticeBox lays out a Box description: its children are relative to
// the box's origin, so the picture's top-left corner is that origin. Frames
// given are used as they are; otherwise the frame is the art when it is
// valid and the value's colours when it is not. Button, SecondaryButton,
// Portrait, Text and Scrollbar become parts when they have area.
func layoutNoticeBox(s popupSpec, f *text.Font) popupLayout {
	size := s.Box.Size()
	row := s.styleRow()
	l := popupLayout{Rect: image.Rectangle{Max: size}, Frames: s.Frames, Lines: s.Lines,
		Edit: s.Edit, Bar: s.Bar, Portrait: s.Portrait}
	if l.Frames == nil {
		l.Frames = []dialogFrame{noticeBoxFrame(s, row, size)}
	}
	state := s.ButtonState
	if !s.Button.Empty() {
		l.Pushes = append(l.Pushes, pushButton{Rect: s.Button, Label: s.ButtonLabel, Ramp: row.ramp, Hover: state.Hover,
			Pressed: state.Pressed, Inside: state.Inside, Disabled: state.Disabled, Policy: s.DialogueBackdrop})
	}
	if second := s.SecondaryState; row.secondary && !s.SecondaryButton.Empty() {
		l.Pushes = append(l.Pushes, pushButton{Rect: s.SecondaryButton, Label: s.SecondaryButtonLabel, Ramp: row.ramp, Hover: second.Hover,
			Pressed: second.Pressed, Inside: second.Inside, Disabled: s.SecondaryDisabled, Policy: s.DialogueBackdrop})
	}
	l.Pushes = append(l.Pushes, s.Pushes...)
	if f != nil && !s.Text.Empty() {
		layoutNoticeBody(&l, s, f, row)
	}
	return l
}

// noticeBoxFrame is the frame of a notice with no given frames: the box less
// the style's shadow band, the art's body with the backdrop's pixel format
// when the style's shadows fall on the scene, and the value's colours when
// the art is not whole.
func noticeBoxFrame(s popupSpec, row noticeStyleRow, size image.Point) dialogFrame {
	body := image.Rect(0, 0, size.X-row.inset, size.Y-row.inset)
	if !s.Frame.valid() {
		return dialogFrame{Spec: panelFrame(body, s.Fill, s.Border)}
	}
	spec := windowFrame(body, s.Frame)
	if !row.frameShadows {
		return dialogFrame{Spec: spec}
	}
	spec.Policy = s.DialogueBackdrop
	spec.Policy.FrameClipped = false
	return dialogFrame{Spec: spec, Body: true}
}

// layoutNoticeBody lays out the body text and the scroll bar beside it: the
// lines of Body by the style's rule, from FirstLine, as many as Text holds.
func layoutNoticeBody(l *popupLayout, s popupSpec, f *text.Font, row noticeStyleRow) {
	pitch := f.Height() + s.Pitch
	if pitch <= 0 {
		return
	}
	lines := dialogueLinesOf(row.rule, f, s.Body, s.Text.Dx())
	total, visible := len(lines), max(s.Text.Dy()/pitch, 0)
	first := min(max(s.FirstLine, 0), max(total-visible, 0))
	lines = lines[first:]
	if visible < len(lines) {
		lines = lines[:visible]
	}
	if !s.Scrollbar.Empty() {
		bar := helpBar(s.Scrollbar, first, total, visible)
		bar.TopHot, bar.BottomHot = s.ScrollbarTopHot, s.ScrollbarBottomHot
		l.Bar = &bar
		if s.Frame != nil {
			l.BarFrames = s.Frame.helpScroll
		}
	}
	ink := s.TextColor
	if row.ink.A != 0 {
		ink = row.ink
		if s.Ink.A != 0 {
			ink = s.Ink
		}
	}
	for i, ln := range lines {
		x, width := s.Text.Min.X, s.Text.Dx()
		if ln.indent {
			x, width = x+row.indent, width-row.indent
		}
		l.Lines = append(l.Lines, dialogLine{Runs: []inkRun{{Text: ln.visible()}}, At: image.Pt(x, s.Text.Min.Y+i*pitch), Ink: ink,
			Shadow: row.textShadow, Justify: row.paragraphs && ln.justify, Width: width, Origin: s.Box.Min.X, Arithmetic: s.DialogueArithmetic})
	}
}

// layoutNoticeWindow lays out a notice window over Panel, the page under it:
// the zero shape a list window (the panel less noticeWindowInset), a ShapeSize
// a text window of that size centred on the panel, its lines wrapped to
// TextWidth and each centred one every LinePitch pixels.
func layoutNoticeWindow(s popupSpec, f *text.Font) popupLayout {
	if s.ShapeSize != (image.Point{}) {
		return layoutNoticeText(s, f)
	}
	l := newNoticeWindowLayout(f, s.Panel, len(s.Answers))
	l.Frames = []dialogFrame{{Spec: windowFrame(l.Body, s.Frame)}}
	if f == nil {
		return l
	}
	display := dialogDisplayOf(s)
	lines := noticeWindowLines(s, f, l)
	top := min(max(s.Top, 0), noticeMaxTop(len(lines), l.List.Rows))
	l.TopClamped = top
	l.Lines = append(l.Lines, townShellTitleLine(f, display(s.Title), l.HeadingBox))
	for i := 0; i < l.List.Rows && top+i < len(lines); i++ {
		ln := lines[top+i]
		r := l.List.Row(i)
		l.Lines = append(l.Lines, dialogLine{Runs: ln.runs, At: image.Pt(r.Min.X+listTextX+ln.indent, r.Min.Y+listTextY),
			Clip: r.Inset(1), Ink: listRowInk, AlertInk: noticeAlertInk})
	}
	for i, line := range noticeFlowLines(f, display(s.Hint), l.Hint.Dx()) {
		if i >= noticeHintLines {
			break
		}
		l.Lines = append(l.Lines, dialogLine{Runs: []inkRun{{Text: line}}, At: image.Pt(l.Hint.Min.X, l.Hint.Min.Y+i*(f.Height()+1)), Ink: townShellText})
	}
	well := listWell(l.List)
	l.Edit = &dialogEdit{Field: well}
	bar := noticeBar(l, top, len(lines))
	l.Bar, l.BarLive = &bar, true
	return l
}

// layoutNoticeText is the text form of a notice window.
func layoutNoticeText(s popupSpec, f *text.Font) popupLayout {
	at := s.Panel.Min.Add(s.Panel.Size().Sub(s.ShapeSize).Div(2))
	b := image.Rectangle{Min: at, Max: at.Add(s.ShapeSize)}
	answers := noticeAnswers(b, s.AnswerSize, s.AnswerGap, s.AnswerBottom, len(s.Answers))
	x := b.Min.X + (s.ShapeSize.X-s.TextWidth)/2
	bottom := b.Max.Y
	if len(answers) > 0 {
		bottom = answers[0].Min.Y
	}
	l := popupLayout{Rect: b, Body: b, Text: image.Rect(x, b.Min.Y+s.TextTop, x+s.TextWidth, bottom), Answers: answers, LinePitch: s.LinePitch}
	l.Frames = []dialogFrame{{Spec: windowFrame(b, s.Frame)}}
	if f == nil {
		return l
	}
	for i, ln := range noticeWindowLines(s, f, l) {
		y := l.Text.Min.Y + i*l.LinePitch
		if y+f.Height() > l.Text.Max.Y {
			break
		}
		width := 0
		for _, run := range ln.runs {
			width += f.Advance(run.Text)
		}
		x := (l.Text.Min.X+l.Text.Max.X)/2 - width/2
		l.Lines = append(l.Lines, dialogLine{Runs: ln.runs, At: image.Pt(x, y), Ink: townShellText, AlertInk: noticeAlertInk})
	}
	return l
}

// townShellTitleLine is a line standing where drawTownShellText stands it.
func townShellTitleLine(f *text.Font, s string, r image.Rectangle) dialogLine {
	shown, at, ok := townShellTextLayout(f, s, r)
	if !ok {
		return dialogLine{}
	}
	return dialogLine{Runs: []inkRun{{Text: shown}}, At: at, Ink: loadSelectedText}
}

// dialogDisplayOf is the description's Display, or the identity.
func dialogDisplayOf(s popupSpec) func(string) string {
	if s.Display != nil {
		return s.Display
	}
	return func(t string) string { return t }
}

// noticeWindowLines wraps every run row of a window to the list's text width,
// or a text window's width with no indent.
func noticeWindowLines(s popupSpec, f *text.Font, l popupLayout) []noticeLine {
	width, indent := l.List.Rect.Dx()-2*listTextX, noticeWrapIndent
	if l.LinePitch > 0 {
		width, indent = l.Text.Dx(), 0
	}
	display := dialogDisplayOf(s)
	var out []noticeLine
	for _, row := range s.Runs {
		out = append(out, noticeRowLines(f, row, width, indent, display)...)
	}
	return out
}
