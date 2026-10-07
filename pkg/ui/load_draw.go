package ui

import (
	"fmt"
	"image"
	"image/draw"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// LOAD rows remain UTF-8 until drawing. Its installed font uses the same
// presentation encoder as the save dialog; names passed to LoadGame stay exact.
func (a *App) composeLoadList(header string, list *Picker) (*image.RGBA, error) {
	font := a.flow.menuFont
	if font == nil {
		return nil, fmt.Errorf("load list has no installed font")
	}
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	if a.assets != nil {
		draw.Draw(pix, pix.Bounds(), a.assets.Compose(a.sel.State()), image.Point{}, draw.Src)
	}
	a.flow.menuArt.Draw(pix, loadPanel)
	w := a.flow.loadUI.words
	if w.Title == "" {
		w = defaultLoadWindowWords()
	}
	label := func(value string, r image.Rectangle, selected bool) {
		c := townShellText
		if selected {
			c = loadSelectedText
		}
		font.Draw(pix.SubImage(r).(*image.RGBA), a.flow.menuDisplayText(a.fitLoadText(value, r.Dx()-6)), r.Min.X+3, r.Min.Y+2, c)
	}
	drawTownShellText(pix, font, w.Title, image.Rect(128, 88, 512, 116), townShellText)
	if a.flow.loadUI.confirm {
		font.Draw(pix, strings.ReplaceAll(w.Confirm, "#", " "), 125, 146, townShellText)
		if list != nil && list.Selection() < len(a.flow.saves) {
			label(a.flow.saves[list.Selection()].Label, image.Rect(122, 180, 520, 210), true)
		}
	} else {
		font.Draw(pix, w.Subtitle, 125, 130, loadSelectedText)
		top, count := 0, 0
		if list != nil {
			top, count = list.Visible()
		}
		for i := 0; i < loadVisibleRows; i++ {
			r := loadRowRect(i)
			selected := i < count && top+i == list.Selection()
			drawMovieBox(pix, r, selected)
			if i < count {
				label(list.Rows()[top+i].Text, r, selected)
			}
		}
		a.drawLoadScroll(pix, list)
	}
	message := a.loadMessage()
	if message != "" {
		// Wrap refusals below the list; the row model retains its full text.
		for i, line := range NoticeLines(font, a.flow.menuDisplayText(message), loadMessageBox.Dx()) {
			if i >= 2 {
				break
			}
			font.Draw(pix, string(line), loadMessageBox.Min.X, loadMessageBox.Min.Y+i*(font.Height()+1), townShellText)
		}
	}
	for i, caption := range []string{w.OK, w.Delete, w.Cancel} {
		r := loadButtonRect(i)
		drawMovieBox(pix, r, a.flow.loadUI.press == (loadTarget{kind: loadHitButton, index: i}))
		c := townShellText
		if i == 1 && (!a.flow.canDeleteLoad() || a.flow.loadUI.confirm) {
			c = loadDisabledText
		}
		drawTownShellText(pix, font, caption, r, c)
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
	return a.flow.loadNote()
}

// The installed font is proportional, so a row is clipped by measured pixels
// inside the shared row rectangles. The disk token is never clipped.
func (a *App) fitLoadText(value string, width int) string {
	// The EN font's non-ASCII records do not reproduce Unicode Cyrillic.
	// Keep its visible fallback while preserving the exact UTF-8 disk token.
	if a.flow.menuFont.Selector != text.SelectorConverting {
		var ascii strings.Builder
		for _, r := range value {
			if r > 0x7e {
				r = '?'
			}
			ascii.WriteRune(r)
		}
		value = ascii.String()
	}
	measure := func(s string) int {
		w, _ := a.flow.menuFont.Measure(a.flow.menuDisplayText(s))
		return w
	}
	if measure(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && measure(string(runes)+clipMark) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + clipMark
}
