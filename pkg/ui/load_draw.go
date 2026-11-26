package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
)

// savePaint is a screen's picture and the captions held back to be drawn by
// the font once there is one.
type savePaint struct {
	pix   *image.RGBA
	texts []savePaintText
}

type savePaintText struct {
	text  string
	at    image.Point
	color color.RGBA
}

func (s *savePaint) label(value string, x, y int, c color.RGBA) {
	s.texts = append(s.texts, savePaintText{text: value, at: image.Pt(x, y), color: c})
}

// composeListPage is the open list page's picture.
func (a *App) composeListPage() (*image.RGBA, error) {
	v, ok := a.flow.openPage()
	if !ok {
		return nil, fmt.Errorf("no list page is open")
	}
	return a.composePage(v)
}

// composePage draws a page's frame, captions, name box, list or confirmation,
// message line and buttons. Its host decides whether the backing is transparent.
// LOAD rows remain UTF-8 until drawing. Its installed font uses the same
// presentation encoder as the save dialog; names passed to LoadGame stay exact.
func (a *App) composePage(v pageView) (*image.RGBA, error) {
	s, err := a.paintPage(v)
	if err != nil {
		return nil, err
	}
	if a.flow.menuFont == nil {
		return nil, fmt.Errorf("list page has no installed font")
	}
	for _, label := range s.texts {
		a.flow.menuFont.Draw(s.pix, a.flow.menuDisplayText(label.text), label.at.X, label.at.Y, label.color)
	}
	return s.pix, nil
}

// paintPage paints the page: the backing and frames, the captions, then the
// name and directory fields, the list or the question, the message line and
// the buttons, and the window that owns the input.
func (a *App) paintPage(v pageView) (*savePaint, error) {
	page, font, e := v.page, a.flow.menuFont, v.edit
	if font == nil && !page.deferredText {
		return nil, fmt.Errorf("list page has no installed font")
	}
	s := &savePaint{pix: image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))}
	if page.background.A != 0 {
		draw.Draw(s.pix, s.pix.Bounds(), image.NewUniform(page.background), image.Point{}, draw.Src)
	} else {
		a.flow.menuPlacement().backdrop.paint(a, s.pix)
	}
	if page.fill.A != 0 {
		drawFrame(s.pix, panelFrame(page.Panel, page.fill, page.border))
	}
	r := page.frame
	if r.Empty() {
		r = page.Panel
	}
	if !page.artOnly || a.flow.menuArt != nil {
		drawFrame(s.pix, windowFrame(r, a.flow.menuArt))
	}
	ink := townShellText
	if page.ink != (color.RGBA{}) {
		ink = page.ink
	}
	label := func(value string, at image.Point, c color.RGBA) {
		if page.deferredText || font == nil {
			s.label(value, at.X, at.Y, c)
		} else {
			font.Draw(s.pix, value, at.X, at.Y, c)
		}
	}
	caption := func(t PageText) {
		t = v.captionInk(t)
		value := a.flow.saveCaption(t.word, t.Text)
		if !t.Box.Empty() {
			drawTownShellText(s.pix, font, value, t.Box, t.Ink)
		} else {
			label(value, t.At, t.Ink)
		}
	}
	for _, t := range v.captions() {
		caption(t)
	}
	pointer, pointerOK := a.pointerFrame()
	window := v.ui.mismatch != nil
	hover := pointerOK && !window && !v.asking()
	layout := page.saveLayout()
	button := func(c saveControl, b PageButton) {
		r := layout.rect(int(c))
		if r.Empty() {
			return
		}
		value := a.flow.saveCaption(b.word, b.Caption)
		if page.fitButtons {
			value = a.flow.menuDisplayText(a.saveTextFit(value, r.Dx()-10))
		}
		inside := hover && pointer.In(r)
		disabled := a.flow.controlDisabled(v, c)
		bstate := pushButton{Rect: r, Label: value, Hover: inside, Inside: inside, Disabled: disabled,
			Focus: page.keyboardFocus && e.focus == c, Pressed: v.ui.press.Pressed(v.latchID(c, 0))}
		if font != nil {
			drawPushButton(s.pix, font, bstate)
			return
		}
		bstate.Label = ""
		drawPushButton(s.pix, nil, bstate)
		cink := ink
		if disabled {
			cink = loadDisabledText
		}
		s.label(value, r.Min.X+(r.Dx()-a.saveTextWidth(value))/2, r.Min.Y+5, cink)
	}
	if v.confirmingInPage() {
		if page.confirmRows > 0 {
			heading := page.confirmHeading
			if v.ui.remove != nil {
				heading = page.deleteHeading
			}
			caption(heading)
			a.clampConfirmationScroll(v)
			lines := a.confirmationLines(v)
			for i := 0; i < page.confirmRows && e.confirmTop+i < len(lines); i++ {
				label(lines[e.confirmTop+i], page.confirmAt.Add(image.Pt(0, i*page.confirmPitch)), ink)
			}
			if len(lines) > page.confirmRows {
				caption(page.scrollHint)
			}
		} else {
			a.drawPageQuestion(s.pix, font, v)
		}
	} else {
		for _, c := range []saveControl{saveDirectoryControl, saveNameControl} {
			r := layout.rect(int(c))
			if r.Empty() {
				continue
			}
			value := *v.nameField()
			if c == saveDirectoryControl {
				value = e.request.Directory
			}
			runes := []rune(value)
			start := 0
			textH := listFallbackPitch - listPitchExtra
			if font != nil {
				textH = font.Height()
			}
			field := editField{Rect: r, TextH: textH, Focus: !page.keyboardFocus || e.focus == c, Phase: a.blink.on()}
			if field.Focus {
				caret := min(e.caret, len(runes))
				for start < caret && a.saveTextWidth(string(runes[start:caret])) > r.Dx()-16 {
					start++
				}
				field.Caret = a.saveTextWidth(string(runes[start:caret]))
				if e.selectedText {
					field.SelTo = a.saveTextWidth(string(runes[start:]))
				}
			}
			shown := a.saveTextFit(string(runes[start:]), r.Dx()-12)
			drawEditField(s.pix, field, func(at image.Point) {
				if font != nil {
					font.Draw(s.pix, a.flow.menuDisplayText(shown), at.X, at.Y, ink)
				} else {
					s.label(shown, at.X, at.Y, ink)
				}
			})
		}
		for i, c := range []saveControl{saveOpenControl, saveUpControl} {
			if !page.Navigation[i].Rect.Empty() {
				button(c, page.Navigation[i])
			}
		}
		if v.list != nil {
			box := a.pageListBox(v)
			if font != nil {
				a.drawPageList(s.pix, font, v, func(row, width int) string {
					value := v.list.Rows()[row].Text
					if page.fileNames {
						return a.flow.menuDisplayText(a.saveTextFit(value, width))
					}
					return a.flow.menuDisplayText(a.fitLoadText(value, width))
				})
			} else {
				drawEditField(s.pix, listWell(box), nil)
				top, count := v.list.Visible()
				for i := 0; i < count; i++ {
					s.label(a.saveTextFit(v.list.Rows()[top+i].Text, box.Rect.Dx()-2*listTextX),
						box.Row(i).Min.X+listTextX, box.Row(i).Min.Y+listTextY, townShellText)
				}
				drawVScrollBar(s.pix, a.media.scroll, listBar(box, v.list).withPointer(pointer, pointerOK))
			}
			if v.list.Len() == 0 && page.empty.word != "" {
				caption(page.empty)
			}
			if entry, _, ok := v.selected(); ok && page.detail.At != (image.Point{}) {
				t := page.detail
				t.Text = a.saveTextFit(entry.Label, 560)
				caption(t)
			}
		}
		for _, block := range page.blocks {
			for i, line := range a.saveLines(a.flow.saveCaption(block.word, ""), block.width) {
				if i >= block.rows {
					break
				}
				label(line, block.at.Add(image.Pt(0, i*block.pitch)), ink)
			}
		}
	}
	if !page.Message.Empty() {
		a.drawPageMessage(s.pix, font, v)
	}
	for i, c := range []saveControl{saveWriteControl, saveDeleteControl, saveCancelControl} {
		b := page.Buttons[i]
		if v.confirmingInPage() && page.confirmRows > 0 {
			if c == saveWriteControl && v.ui.remove != nil || c == saveDeleteControl && v.ui.remove == nil {
				continue
			}
			b = page.confirmButtons[i]
		}
		button(c, b)
	}
	if window {
		w := v.ui.mismatch
		w.compose(s.pix, font, page.Panel, a.flow.menuArt, a.media.scroll, a.modMismatchDisplay, pointer, pointerOK)
	} else if v.asking() {
		w := &v.ui.question
		w.compose(s.pix, font, page.Panel, a.flow.menuArt, a.media.scroll, func(s string) string { return s }, pointer, pointerOK)
	}
	return s, nil
}

// confirmationLines are the lines the page's confirmation lists: the file a
// deletion removes, or the files a write creates or replaces.
func (a *App) confirmationLines(v pageView) []string {
	e := v.edit
	if v.ui.remove != nil {
		return a.saveLines(e.removePath, v.page.confirmWidth)
	}
	if e.prepared == nil {
		return nil
	}
	w := a.flow.saveWords()
	var lines []string
	for _, path := range e.prepared.Paths {
		verb := w.Create
		for _, existing := range e.prepared.Existing {
			if path == existing {
				verb = w.Replace
				break
			}
		}
		lines = append(lines, verb)
		lines = append(lines, a.saveLines(path, v.page.confirmWidth)...)
		lines = append(lines, "")
	}
	return lines
}

func (a *App) clampConfirmationScroll(v pageView) {
	if v.page.confirmRows == 0 {
		return
	}
	v.edit.confirmTop = max(0, min(v.edit.confirmTop, len(a.confirmationLines(v))-v.page.confirmRows))
}

// captionInk gives a caption its default or focus ink.
func (v pageView) captionInk(t PageText) PageText {
	if t.Ink == (color.RGBA{}) {
		t.Ink = townShellText
	}
	if t.Focus && v.listFocus() {
		t.Ink = loadSelectedText
	}
	return t
}

// captions are the texts that show now; the list heading hides while asking.
func (v pageView) captions() []PageText {
	var texts []PageText
	for _, t := range v.page.Texts {
		if !t.Listing || !v.confirmingInPage() {
			texts = append(texts, v.captionInk(t))
		}
	}
	return texts
}

// drawPageQuestion draws the in-page delete question.
func (a *App) drawPageQuestion(pix *image.RGBA, font *text.Font, v pageView) {
	c := v.page.Confirm
	font.Draw(pix, strings.ReplaceAll(c.Before, "#", " "), c.At.X, c.At.Y, townShellText)
	if entry, _, ok := v.selected(); ok {
		font.Draw(pix.SubImage(c.Label).(*image.RGBA), a.flow.menuDisplayText(a.fitLoadText(entry.Label, c.Label.Dx()-6)),
			c.Label.Min.X+3, c.Label.Min.Y+2, loadSelectedText)
	}
}

// drawPageList draws the list.
func (a *App) drawPageList(pix *image.RGBA, font *text.Font, v pageView, rowText listRowText) {
	pointer, pointerOK := a.pointerFrame()
	pointerOK = pointerOK && v.ui.mismatch == nil && !v.asking()
	drawListBox(pix, font, a.media.scroll, a.pageListBox(v), v.list, rowText, pointer, pointerOK)
}

// drawPageMessage wraps the refusal or detail below the list.
func (a *App) drawPageMessage(pix *image.RGBA, font *text.Font, v pageView) {
	message := a.flow.pageMessage(v)
	if message == "" {
		return
	}
	y := v.page.Message.Min.Y
	if !v.confirmingInPage() {
		y = max(y, a.pageListBox(v).Rect.Max.Y+2)
	}
	for i, line := range noticeFlowLines(font, a.flow.menuDisplayText(message), v.page.Message.Dx()) {
		if i >= 2 {
			break
		}
		font.Draw(pix, string(line), v.page.Message.Min.X, y+i*(font.Height()+1), townShellText)
	}
}

func (a *App) loadMessage() string {
	if a.flow.msg != "" {
		return a.flow.msg
	}
	if a.flow.screen != ScreenLoad {
		return ""
	}
	return a.flow.pageMessage(a.flow.loadView())
}

// pageMessage is the refusal, else the selected save's detail, else its
// label.
func (f *flow) pageMessage(v pageView) string {
	if f.msg != "" {
		return f.msg
	}
	entry, _, ok := v.selected()
	if !ok || v.page.RowHint {
		return ""
	}
	if entry.Detail != "" {
		return entry.Detail
	}
	return entry.Label
}

// The installed font is proportional, so a row is clipped by measured pixels
// inside the shared row rectangles. The disk token is never clipped.
func (a *App) fitLoadText(value string, width int) string {
	value = a.drawableLoadText(value)
	measure := func(s string) int {
		w, _ := a.flow.menuFont.Measure(a.flow.menuDisplayText(s))
		return w
	}
	return textfit.Spec{Measure: measure, Unit: textfit.Rune, Side: textfit.End}.Fit(value, width)
}

// drawableLoadText is value as the menu font can show it. The EN font's
// non-ASCII records do not reproduce Unicode Cyrillic, so it shows '?' for
// them; the exact UTF-8 disk token is kept apart.
func (a *App) drawableLoadText(value string) string {
	if locale.FontRemaps(a.flow.menuFont.Selector) {
		return value
	}
	var ascii strings.Builder
	for _, r := range value {
		if r > 0x7e {
			r = '?'
		}
		ascii.WriteRune(r)
	}
	return ascii.String()
}
