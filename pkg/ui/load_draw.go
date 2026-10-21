package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/textfit"
)

// LOAD rows remain UTF-8 until drawing. Its installed font uses the same
// presentation encoder as the save dialog; names passed to LoadGame stay exact.
func (a *App) composeLoadList(header string, list *Picker) (*image.RGBA, error) {
	return a.composePage(a.flow.loadView())
}

// composePage draws one page: the main menu behind it, its frame and
// captions, the save page's name box, the list or the delete confirmation,
// the message line and the three buttons.
func (a *App) composePage(v pageView) (*image.RGBA, error) {
	font := a.flow.menuFont
	if font == nil {
		return nil, fmt.Errorf("load list has no installed font")
	}
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	if a.assets != nil {
		draw.Draw(pix, pix.Bounds(), a.assets.Compose(a.sel.State()), image.Point{}, draw.Src)
	}
	drawFrame(pix, windowFrame(v.page.Panel, a.flow.menuArt))
	label := func(value string, r image.Rectangle, selected bool) {
		c := townShellText
		if selected {
			c = loadSelectedText
		}
		font.Draw(pix.SubImage(r).(*image.RGBA), a.flow.menuDisplayText(a.fitLoadText(value, r.Dx()-6)), r.Min.X+3, r.Min.Y+2, c)
	}
	for _, t := range v.page.Texts {
		if t.Ink == (color.RGBA{}) {
			t.Ink = townShellText
		}
		switch {
		case t.Listing && v.ui.confirm:
		case t.Box != (image.Rectangle{}):
			drawTownShellText(pix, font, t.Text, t.Box, t.Ink)
		default:
			font.Draw(pix, t.Text, t.At.X, t.At.Y, t.Ink)
		}
	}
	if c := v.page.Confirm; v.ui.confirm && c.Inline {
		entry, _, _ := v.selected()
		s := strings.ReplaceAll(c.Before, "#", " ") + a.flow.menuDisplayText(entry.Label) + strings.ReplaceAll(c.After, "#", " ")
		for i, line := range NoticeLines(font, s, c.Label.Dx()) {
			font.Draw(pix, string(line), c.At.X, c.At.Y+i*(font.Height()+1), townShellText)
		}
	} else if v.ui.confirm {
		font.Draw(pix, strings.ReplaceAll(c.Before, "#", " "), c.At.X, c.At.Y, townShellText)
		if entry, _, ok := v.selected(); ok {
			label(entry.Label, c.Label, true)
		}
	} else {
		if v.save != nil {
			a.drawPageName(pix, v)
		}
		pointer, pointerOK := a.pointerFrame()
		pointerOK = pointerOK && v.ui.mismatch == nil
		list := v.list
		drawListBox(pix, font, a.media.scroll, a.pageListBox(v), list, func(row, width int) string {
			return a.flow.menuDisplayText(a.fitLoadText(list.Rows()[row].Text, width))
		}, pointer, pointerOK)
	}
	if message := a.flow.pageMessage(v); message != "" {
		messageY := v.page.Message.Min.Y
		if !v.ui.confirm {
			messageY = max(messageY, a.pageListBox(v).Rect.Max.Y+2)
		}
		// Wrap the detail or refusal below the list; the row retains its full text.
		for i, line := range NoticeLines(font, a.flow.menuDisplayText(message), v.page.Message.Dx()) {
			if i >= 2 {
				break
			}
			font.Draw(pix, string(line), v.page.Message.Min.X, messageY+i*(font.Height()+1), townShellText)
		}
	}
	pointer, pointerOK := a.pointerFrame()
	window := v.save == nil && v.ui.mismatch != nil
	pageHover := pointerOK && !window
	for i, b := range v.page.Buttons {
		inside := pageHover && pointer.In(b.Rect)
		drawPushButton(pix, font, pushButton{Rect: b.Rect, Label: b.Caption, Hover: inside, Inside: inside,
			Pressed: v.ui.press.Pressed(i), Disabled: a.flow.pageButtonDisabled(v, i)})
	}
	if window {
		l := newNoticeWindowLayout(font, v.page.Panel)
		v.ui.mismatch.compose(pix, font, l, a.flow.menuArt, a.media.scroll, a.modMismatchDisplay, pointer, pointerOK)
	}
	return pix, nil
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
	if !ok {
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
