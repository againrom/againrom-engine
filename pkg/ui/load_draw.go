package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
)

// LOAD rows remain UTF-8 until drawing. Its installed font uses the same
// presentation encoder as the save dialog; names passed to LoadGame stay exact.
func (a *App) composeLoadList(header string, list *Picker) (*image.RGBA, error) {
	return a.composePage(a.flow.loadView())
}

// composePage draws a page's frame, captions, name box, list or confirmation,
// message line and buttons. Its host decides whether the backing is transparent.
func (a *App) composePage(v pageView) (*image.RGBA, error) {
	if v.save != nil {
		return a.composeSaveScreen()
	}
	font := a.flow.menuFont
	if font == nil {
		return nil, fmt.Errorf("load list has no installed font")
	}
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	a.flow.menuPlacement().backdrop.paint(a, pix)
	drawFrame(pix, windowFrame(v.page.Panel, a.flow.menuArt))
	for _, t := range v.captions() {
		if t.Box != (image.Rectangle{}) {
			drawTownShellText(pix, font, t.Text, t.Box, t.Ink)
		} else {
			font.Draw(pix, t.Text, t.At.X, t.At.Y, t.Ink)
		}
	}
	if v.confirmingInPage() {
		a.drawPageQuestion(pix, font, v)
	} else {
		list := v.list
		a.drawPageList(pix, font, v, func(row, width int) string {
			return a.flow.menuDisplayText(a.fitLoadText(list.Rows()[row].Text, width))
		})
	}
	a.drawPageMessage(pix, font, v)
	pointer, pointerOK := a.pointerFrame()
	window := v.save == nil && v.ui.mismatch != nil
	pageHover := pointerOK && !window && !v.asking()
	layout := v.page.layout()
	for i, b := range v.page.Buttons {
		r := layout.rect(i)
		inside := pageHover && pointer.In(r)
		drawPushButton(pix, font, pushButton{Rect: r, Label: b.Caption, Hover: inside, Inside: inside,
			Pressed: v.ui.press.Pressed(i), Disabled: a.flow.pageButtonDisabled(v, i)})
	}
	if window {
		w := v.ui.mismatch
		w.compose(pix, font, w.layout(font, v.page.Panel), a.flow.menuArt, a.media.scroll, a.modMismatchDisplay, pointer, pointerOK)
	} else if v.asking() {
		w := &v.ui.question
		w.compose(pix, font, w.layout(font, v.page.Panel), a.flow.menuArt, a.media.scroll, func(s string) string { return s }, pointer, pointerOK)
	}
	return pix, nil
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

func (a *App) composeLoadScreen() (*image.RGBA, error) {
	return a.composeLoadList(a.loadHeader(), a.flow.loadList)
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
