package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/render/frame"
)

const (
	saveVisibleRows      = 6
	saveRowHeight        = 23
	saveScrollX          = 580
	saveConfirmationRows = 17
)

var (
	savePanelRect      = image.Rect(24, 12, 616, 468)
	saveListRect       = image.Rect(38, 88, 602, 235)
	saveFocusColor     = color.RGBA{R: 0xc8, G: 0xa2, B: 0x56, A: 0xff}
	saveSelectionColor = color.RGBA{R: 0x48, G: 0x3b, B: 0x27, A: 0xff}
)

func saveControlRect(c saveControl) image.Rectangle {
	switch c {
	case saveDirectoryControl:
		return image.Rect(106, 52, 436, 79)
	case saveOpenControl:
		return image.Rect(444, 52, 530, 79)
	case saveUpControl:
		return image.Rect(538, 52, 602, 79)
	case saveListControl:
		return saveListRect
	case saveNameControl:
		return image.Rect(106, 266, 602, 293)
	case saveDeleteControl:
		return image.Rect(238, 432, 354, 458)
	case saveWriteControl:
		return image.Rect(362, 432, 478, 458)
	case saveCancelControl:
		return image.Rect(486, 432, 602, 458)
	}
	return image.Rectangle{}
}

type savePaintText struct {
	text  string
	at    image.Point
	color color.RGBA
}

type savePaint struct {
	pix   *image.RGBA
	texts []savePaintText
}

func (s *savePaint) box(r image.Rectangle, fill, border color.RGBA) {
	draw.Draw(s.pix, r, image.NewUniform(border), image.Point{}, draw.Src)
	draw.Draw(s.pix, r.Inset(1), image.NewUniform(fill), image.Point{}, draw.Src)
}

func (s *savePaint) label(value string, x, y int, c color.RGBA) {
	s.texts = append(s.texts, savePaintText{text: value, at: image.Pt(x, y), color: c})
}

func (a *App) saveTextWidth(value string) int {
	if font := a.flow.menuFont; font != nil {
		w, _ := font.Measure(a.flow.menuDisplayText(value))
		return w
	}
	return 6 * len([]rune(value))
}

func (a *App) saveTextFit(value string, width int) string {
	if a.saveTextWidth(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && a.saveTextWidth(string(runes)+"...") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

// Hard wrapping preserves every character of an exact target path, including
// spaces. Paths are scrollable, never shortened to a basename or ellipsis.
func (a *App) saveWrapped(value string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		line := ""
		for _, r := range paragraph {
			next := line + string(r)
			if line != "" && a.saveTextWidth(next) > width {
				lines = append(lines, line)
				line = string(r)
			} else {
				line = next
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func (a *App) saveConfirmationLines() []string {
	d := a.flow.saveDialog
	if d == nil {
		return nil
	}
	if d.remove != nil {
		return a.saveWrapped(d.removePath, 548)
	}
	if d.prepared == nil {
		return nil
	}
	w := a.flow.saveWords()
	var lines []string
	for _, path := range d.prepared.Paths {
		verb := w.Create
		for _, existing := range d.prepared.Existing {
			if path == existing {
				verb = w.Replace
				break
			}
		}
		lines = append(lines, verb)
		lines = append(lines, a.saveWrapped(path, 548)...)
		lines = append(lines, "")
	}
	return lines
}

func (a *App) clampSaveConfirmationScroll() {
	d := a.flow.saveDialog
	if d == nil {
		return
	}
	maxTop := len(a.saveConfirmationLines()) - saveConfirmationRows
	if d.confirmTop > maxTop {
		d.confirmTop = maxTop
	}
	if d.confirmTop < 0 {
		d.confirmTop = 0
	}
}

func (a *App) saveDialogPaint() (*savePaint, error) {
	d := a.flow.saveDialog
	if d == nil {
		return nil, fmt.Errorf("save dialog is closed")
	}
	layout := AuthoredDialogueLayout()
	layout.Fill.A = 255
	w := a.flow.saveWords()
	s := &savePaint{pix: image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))}
	draw.Draw(s.pix, s.pix.Bounds(), image.NewUniform(color.RGBA{R: 8, G: 10, B: 14, A: 255}), image.Point{}, draw.Src)
	s.box(savePanelRect, layout.Fill, layout.Border)
	if a.flow.menuArt != nil {
		a.flow.menuArt.Draw(s.pix, image.Rect(8, 0, 632, 480))
	}
	s.label(w.Title, 38, 23, layout.TextColor)
	button := func(c saveControl, label string, selected bool) {
		r := saveControlRect(c)
		fill, border := layout.ButtonFill, layout.ButtonBorder
		if selected {
			fill = saveSelectionColor
		}
		if d.focus == c {
			border = saveFocusColor
		}
		s.box(r, fill, border)
		label = a.saveTextFit(label, r.Dx()-10)
		x := r.Min.X + (r.Dx()-a.saveTextWidth(label))/2
		textColor := layout.TextColor
		if c == saveDeleteControl && d.remove == nil && !a.flow.canDeleteSave() {
			textColor = loadDisabledText
		}
		s.label(label, x, r.Min.Y+5, textColor)
	}
	if d.prepared != nil || d.remove != nil {
		confirm := w.Confirm
		if d.remove != nil {
			confirm = w.DeleteConfirm
		}
		s.label(confirm, 38, 58, layout.TextColor)
		a.clampSaveConfirmationScroll()
		lines := a.saveConfirmationLines()
		for i := 0; i < saveConfirmationRows && d.confirmTop+i < len(lines); i++ {
			s.label(lines[d.confirmTop+i], 42, 92+i*18, layout.TextColor)
		}
		if len(lines) > saveConfirmationRows {
			s.label(w.ScrollHint, 38, 408, layout.TextColor)
		}
		if d.remove != nil {
			button(saveDeleteControl, w.Delete, false)
		} else {
			button(saveWriteControl, w.Overwrite, false)
		}
		button(saveCancelControl, w.Back, false)
		return s, nil
	}
	s.label(w.Directory, 38, 58, layout.TextColor)
	s.label(w.Name, 38, 272, layout.TextColor)
	for _, c := range []saveControl{saveDirectoryControl, saveNameControl} {
		r := saveControlRect(c)
		fill, border := color.RGBA{R: 8, G: 10, B: 14, A: 255}, layout.Border
		value := d.request.Directory
		if c == saveNameControl {
			value = d.request.Name
		}
		if d.focus == c {
			border = saveFocusColor
			if d.selectedText {
				fill = saveSelectionColor
			}
		}
		s.box(r, fill, border)
		runes := []rune(value)
		start := 0
		if d.focus == c {
			caret := d.caret
			if caret > len(runes) {
				caret = len(runes)
			}
			for start < caret && a.saveTextWidth(string(runes[start:caret])) > r.Dx()-16 {
				start++
			}
			x := r.Min.X + 5 + a.saveTextWidth(string(runes[start:caret]))
			if !d.selectedText {
				draw.Draw(s.pix, image.Rect(x, r.Min.Y+5, x+1, r.Max.Y-5), image.NewUniform(layout.TextColor), image.Point{}, draw.Src)
			}
		}
		s.label(a.saveTextFit(string(runes[start:]), r.Dx()-12), r.Min.X+5, r.Min.Y+5, layout.TextColor)
	}
	button(saveOpenControl, w.Open, false)
	button(saveUpControl, w.Up, false)
	top, count := 0, 0
	if d.list != nil {
		top, count = d.list.Visible()
	}
	for i := 0; i < saveVisibleRows; i++ {
		y := saveListRect.Min.Y + 4 + i*saveRowHeight
		selected := i < count && top+i == d.list.Selection()
		drawMovieBox(s.pix, image.Rect(40, y-1, saveScrollX-3, y+saveRowHeight-2), selected)
		if i < count {
			ink := townShellText
			if selected {
				ink = loadSelectedText
			}
			s.label(a.saveTextFit(d.list.Rows()[top+i].Text, 525), 44, y+2, ink)
		}
	}
	if d.list != nil {
		if count == 0 {
			s.label(w.Empty, 44, 96, layout.TextColor)
		}
		thumbY := 112
		if len(d.list.Rows()) > 1 {
			thumbY += d.list.Selection() * (99 - 24) / (len(d.list.Rows()) - 1)
		}
		if !drawScrollbarSkin(s.pix, a.media.scroll, image.Rect(580, 88, 602, 112), image.Rect(580, 112, 602, 211),
			image.Rect(580, 211, 602, 235), image.Rect(580, thumbY, 602, thumbY+24)) {
			s.label("^", 585, 96, layout.TextColor)
			s.label("v", 585, 212, layout.TextColor)
		}
		i := d.list.Selection() - len(d.directory.Directories)
		if i >= 0 && i < len(d.directory.Entries) {
			entry := d.directory.Entries[i]
			detail := entry.Label
			if entry.Note != "" {
				detail = entry.Note
			}
			s.label(a.saveTextFit(detail, 560), 40, 241, layout.TextColor)
		}
	}
	detail := w.TownDetail
	if d.request.OnMap {
		detail = w.MapSAVDetail
	}
	for i, line := range a.saveWrapped(detail, 560) {
		if i >= 3 {
			break
		}
		s.label(line, 38, 307+i*16, layout.TextColor)
	}
	message := a.flow.msg
	if message == "" {
		message = w.KeyHint
	}
	for i, line := range a.saveWrapped(message, 560) {
		if i >= 2 {
			break
		}
		s.label(line, 38, 388+i*16, layout.TextColor)
	}
	button(saveWriteControl, w.Save, false)
	button(saveDeleteControl, w.Delete, false)
	button(saveCancelControl, w.Cancel, false)
	return s, nil
}

func (a *App) composeSaveDialogScreen() (*image.RGBA, error) {
	s, err := a.saveDialogPaint()
	if err != nil {
		return nil, err
	}
	if a.flow.menuFont == nil {
		return nil, fmt.Errorf("save dialog has no installed font")
	}
	for _, label := range s.texts {
		a.flow.menuFont.Draw(s.pix, a.flow.menuDisplayText(label.text), label.at.X, label.at.Y, label.color)
	}
	return s.pix, nil
}

func (a *App) drawSaveDialog() {
	a.hasMenu = false
	if pix, err := a.composeSaveDialogScreen(); err == nil {
		a.writeCanvas(pix)
		return
	}
	// Hand-assembled apps can omit the install font. They retain a readable
	// interactive screen through the same debug font as the existing load list.
	s, err := a.saveDialogPaint()
	if err != nil {
		return
	}
	a.writeCanvas(s.pix)
	for _, label := range s.texts {
		a.printCanvas(label.text, label.at.X, label.at.Y)
	}
}
