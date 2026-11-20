package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// Dialog is the game's modal dialog description for the packages that compose
// pages. A value is laid out by layoutDialog and painted by paintDialog.
// Geometry comes from given rectangles (Box and the fields beside it, or a
// notice window over Panel); the frame is art or colours by value, a dialog's
// parts are data the painter reads the same way for every value. The notice
// windows, the gold editor, the game menu, the save page's frames and
// buttons, the authored notices, the help panel and town talk are values of
// it. The map editor's windows are not: they are popupSpec trees.
type Dialog struct {
	// Title heads a notice window; Top is its first shown line.
	Title string
	Top   int

	// Box and Panel give the dialog's geometry: Box is a composed picture of
	// that size at the origin, Panel a notice window standing over that page
	// rectangle. The frame is by value.
	Box, Panel image.Rectangle
	// Frame is the window art a framed dialog draws; Frames are frames by
	// value drawn instead of it, an empty non-nil list none.
	Frame  *DialogFrame
	Frames []dialogFrame
	// Lines, Pushes and Edit are parts stated as data; Bar is a scroll bar
	// drawn over the frames argument (or BarFrames from a notice).
	Lines  []dialogLine
	Pushes []pushButton
	Edit   *dialogEdit
	Bar    *vScrollBar

	// A notice window's words and answer shape. Runs are its rows, each a
	// run of plain and alert ink; the zero ShapeSize lays out a list window
	// with a title, list, hint and 96x24 answers; a ShapeSize a text window
	// of that size, each line centred, its answers AnswerSize, AnswerGap
	// apart and AnswerBottom above the bottom. Latch holds the answer pressed.
	Runs                          []noticeRow
	Hint                          string
	Answers                       []string
	ShapeSize, AnswerSize         image.Point
	TextTop, TextWidth, LinePitch int
	AnswerGap, AnswerBottom       int
	Latch                         *buttonLatch
	// Display turns a word into the bytes the font draws; nil is as given.
	Display func(string) string

	// An authored notice's look (the rest of the Box form), read by the one
	// layout and painter. Box is its rectangle in the design space, not in
	// window pixels: the map screen draws straight to the window, so the
	// authored 640x480 rectangle would be a quarter-size box in a corner at
	// the shipped startup scale; the composition happens at the authored
	// size and the placement scales it. The rectangles below are relative
	// to Box's origin, as the window's child rectangles are. A new value of
	// this type replaces a notice's look (Viewer.SetNoticeLayouts).
	//
	// Style picks the row of noticeStyles the notice is laid out and painted
	// by. Body and Face are what one notice shows; RenderNotice sets them.
	Style              NoticeStyle
	DialogueBackdrop   DialogueBackdrop
	DialogueArithmetic DialogueArithmetic
	ButtonState        DialogueButtonState
	// Text and Button stand in the box; SecondaryButton is the campaign
	// outcome panel's second control, empty on dialogue notices, and
	// SecondaryState its pointer state.
	Text            image.Rectangle
	Button          image.Rectangle
	SecondaryButton image.Rectangle
	SecondaryState  DialogueButtonState
	// Portrait is the speaker's pane and TextBesidePortrait the text area
	// when the pane is there: both shapes of one window in one value, as
	// the original's one routine holds both text rectangles and picks on a
	// flag. WithPortrait is that pick; an empty Portrait is a layout with
	// no pane at all. PortraitAt is the request point on the dialogue
	// surface, PortraitWindow the picture's crop.
	Portrait           image.Rectangle
	TextBesidePortrait image.Rectangle
	PortraitAt         image.Point
	PortraitWindow     image.Rectangle
	// ButtonLabel is the word on the primary button (the success panel's
	// from dialogs.bin); SecondaryButtonLabel is empty where SecondaryButton
	// is empty.
	ButtonLabel, SecondaryButtonLabel string
	SecondaryDisabled                 bool
	// FirstLine is the first wrapped line drawn, Scrollbar the scroll bar
	// beside the text and the two hot flags its endcap states; only the help
	// panel sets them.
	FirstLine                           int
	Scrollbar                           image.Rectangle
	ScrollbarTopHot, ScrollbarBottomHot bool
	// Ink is the body text colour when its alpha is nonzero, the style's
	// ink otherwise; only the help panel sets it. Pitch is added to the
	// font's height to give the line pitch.
	Ink   color.RGBA
	Pitch int
	// Fill and Border are the colours of a frameless notice, TextColor the
	// plain style's body ink, ButtonFill, ButtonBorder and ButtonText the
	// button's colours; none is read when the frame art is present.
	Fill, Border, TextColor, ButtonFill, ButtonBorder, ButtonText color.RGBA
	// GrowText makes Text hold every line of its body (WithBody, DIV-4229).
	GrowText bool
	// FrameInset is a shadow band kept inside Box beyond the style's own, and
	// CenterText centres each body line on the text rectangle's middle.
	FrameInset int
	CenterText bool
	// PortraitFill and PortraitBorder paint an empty pane, so a speaker
	// whose picture was not found and a pane never built read differently.
	PortraitFill, PortraitBorder color.RGBA
	Body                         string
	Face                         *image.RGBA
	// Replies are a dialogue page's numbered answers, wrapped, and ReplyRows
	// their rows, set together by WithReplies; ReplyLit is the lit reply,
	// from 1, 0 none.
	Replies   []string
	ReplyRows []image.Rectangle
	ReplyLit  int
	// ReplyHeld is the reply a held primary press began on, from 1, 0 none.
	ReplyHeld int
	// ReplyOff marks the rows shown greyed and not choosable.
	ReplyOff []bool
}

// dialogLayout is a laid out Dialog: every rectangle its painter fills and
// its hit tests read.
type dialogLayout struct {
	Rect image.Rectangle
	// Body, HeadingBox, Hint and Text are the dialog's rectangles, Answers
	// its answer buttons and LinePitch a text window's line pitch.
	Body, HeadingBox, Hint, Text image.Rectangle
	Answers                      []image.Rectangle
	LinePitch                    int
	// List is a list window's list and TopClamped its first shown line
	// after clamping.
	List       listBox
	TopClamped int
	// Frames, Lines, Pushes, Edit, Bar and BarFrames are the painted parts;
	// BarLive makes the bar answer the pointer as a list window's does.
	Frames    []dialogFrame
	Lines     []dialogLine
	Pushes    []pushButton
	Edit      *dialogEdit
	Bar       *vScrollBar
	BarFrames []*image.RGBA
	BarLive   bool
	// Portrait is a notice's pane, painted right after the frames.
	Portrait image.Rectangle
}

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
func drawDialogParts(dst *image.RGBA, f *text.Font, frames []*image.RGBA, s Dialog, l dialogLayout, pointer image.Point, pointerOK bool) {
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
func drawDialog(dst *image.RGBA, f *text.Font, s Dialog) {
	paintDialog(dst, f, nil, s, layoutDialog(s, f), image.Point{}, false)
}

// paintDialog is the one dialog painter: the frames, the portrait pane, then
// the parts.
func paintDialog(dst *image.RGBA, f *text.Font, frames []*image.RGBA, s Dialog, l dialogLayout, pointer image.Point, pointerOK bool) {
	if dst == nil {
		return
	}
	for _, fr := range l.Frames {
		if fr.Body {
			drawFrameBody(dst, fr.Spec)
		} else {
			drawFrame(dst, fr.Spec)
		}
	}
	if !l.Portrait.Empty() {
		s.styleRow().portrait(dst, s, s.Face)
	}
	drawDialogParts(dst, f, frames, s, l, pointer, pointerOK)
}

// layoutDialog is the one dialog layout: a notice box, authored in the design
// space and composed at the picture's origin, or a notice window over its
// Panel.
func layoutDialog(s Dialog, f *text.Font) dialogLayout {
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
	portrait     func(*image.RGBA, Dialog, *image.RGBA)
}

var noticeStyles = [...]noticeStyleRow{
	NoticeStylePlain: {rule: noticeRule, secondary: true, portrait: drawPlainPortrait},
	NoticeStyleDialogue: {rule: dialogueRule, trim: " \t", paragraphs: true, indent: dialogueIndent, textShadow: dialogueTextShadow,
		ramp: pushButtonDialogue, inset: noticeShadow, ink: dialogueInk, frameShadows: true, refuseOff: true, portrait: drawDialoguePortrait},
}

// styleRow is the value's row of noticeStyles; an unknown style is plain.
func (s Dialog) styleRow() noticeStyleRow {
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
func layoutNoticeBox(s Dialog, f *text.Font) dialogLayout {
	size := s.Box.Size()
	row := s.styleRow()
	l := dialogLayout{Rect: image.Rectangle{Max: size}, Frames: s.Frames, Lines: s.Lines,
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
	for i, r := range s.ReplyRows {
		label := ""
		if i < len(s.Replies) {
			label = s.Replies[i]
		}
		off := i < len(s.ReplyOff) && s.ReplyOff[i]
		lit := s.ReplyLit == i+1 && !off
		l.Pushes = append(l.Pushes, pushButton{Rect: r, Label: label, Literal: true, LabelLeft: replyRowInset, Ramp: row.ramp,
			Hover: lit, Pressed: s.ReplyHeld == i+1 && !off, Inside: lit, Disabled: off, Policy: s.DialogueBackdrop})
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
func noticeBoxFrame(s Dialog, row noticeStyleRow, size image.Point) dialogFrame {
	inset := row.inset + s.FrameInset
	body := image.Rect(0, 0, size.X-inset, size.Y-inset)
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
func layoutNoticeBody(l *dialogLayout, s Dialog, f *text.Font, row noticeStyleRow) {
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
		if s.CenterText {
			x += (width - f.Advance(ln.visible())) / 2
		}
		l.Lines = append(l.Lines, dialogLine{Runs: []inkRun{{Text: ln.visible()}}, At: image.Pt(x, s.Text.Min.Y+i*pitch), Ink: ink,
			Shadow: row.textShadow, Justify: row.paragraphs && ln.justify, Width: width, Origin: s.Box.Min.X, Arithmetic: s.DialogueArithmetic})
	}
}

// layoutNoticeWindow lays out a notice window over Panel, the page under it:
// the zero shape a list window (the panel less noticeWindowInset), a ShapeSize
// a text window of that size centred on the panel, its lines wrapped to
// TextWidth and each centred one every LinePitch pixels.
func layoutNoticeWindow(s Dialog, f *text.Font) dialogLayout {
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
func layoutNoticeText(s Dialog, f *text.Font) dialogLayout {
	at := s.Panel.Min.Add(s.Panel.Size().Sub(s.ShapeSize).Div(2))
	b := image.Rectangle{Min: at, Max: at.Add(s.ShapeSize)}
	answers := noticeAnswers(b, s.AnswerSize, s.AnswerGap, s.AnswerBottom, len(s.Answers))
	x := b.Min.X + (s.ShapeSize.X-s.TextWidth)/2
	bottom := b.Max.Y
	if len(answers) > 0 {
		bottom = answers[0].Min.Y
	}
	l := dialogLayout{Rect: b, Body: b, Text: image.Rect(x, b.Min.Y+s.TextTop, x+s.TextWidth, bottom), Answers: answers, LinePitch: s.LinePitch}
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
func dialogDisplayOf(s Dialog) func(string) string {
	if s.Display != nil {
		return s.Display
	}
	return func(t string) string { return t }
}

// noticeWindowLines wraps every run row of a window to the list's text width,
// or a text window's width with no indent.
func noticeWindowLines(s Dialog, f *text.Font, l dialogLayout) []noticeLine {
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
