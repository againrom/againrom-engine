package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/textfit"
)

const (
	saveVisibleRows      = 7
	saveConfirmationRows = 17
)

var (
	savePanelRect = image.Rect(24, 12, 616, 468)
	// saveDialogLayout is the dialog's control geometry; draw and hit read it.
	saveDialogLayout = controlLayout{
		{int(saveDirectoryControl), image.Rect(106, 52, 436, 79)},
		{int(saveOpenControl), image.Rect(444, 52, 530, 79)},
		{int(saveUpControl), image.Rect(538, 52, 602, 79)},
		{int(saveListControl), image.Rect(38, 88, 602, 235)},
		{int(saveNameControl), image.Rect(106, 266, 602, 293)},
		{int(saveDeleteControl), image.Rect(138, 432, 254, 458)},
		{int(saveWriteControl), image.Rect(262, 432, 378, 458)},
		{int(saveCancelControl), image.Rect(386, 432, 502, 458)},
	}
	saveListRect = saveControlRect(saveListControl)
	// saveListArg is the list's argument: the list control less the bar's
	// 24 pixels.
	saveListArg    = image.Rect(saveListRect.Min.X, saveListRect.Min.Y, saveListRect.Max.X-24, saveListRect.Max.Y)
	saveFocusColor = color.RGBA{R: 0xc8, G: 0xa2, B: 0x56, A: 0xff}
)

func saveControlRect(c saveControl) image.Rectangle { return saveDialogLayout.rect(int(c)) }

// saveListBox is the save dialog's shared list; its bar fills the list
// rectangle's right 24 pixels.
func (a *App) saveListBox() listBox { return newListBox(saveListArg, saveVisibleRows, a.flow.menuFont) }

type savePaintText struct {
	text  string
	at    image.Point
	color color.RGBA
}

type savePaint struct {
	pix   *image.RGBA
	texts []savePaintText
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
	return textfit.Spec{Measure: a.saveTextWidth, Unit: textfit.Rune, Side: textfit.End}.Fit(value, width)
}

// saveLines breaks value by character, keeping every character of an exact
// target path, spaces included. Paths are scrollable, never shortened to a
// basename or ellipsis.
func (a *App) saveLines(value string, width int) []string {
	return saveRule.plain(lineBox{Measure: a.saveTextWidth, Width: width}, value)
}

func (a *App) saveConfirmationLines() []string {
	d := a.flow.saveDialog
	if d == nil {
		return nil
	}
	if d.remove != nil {
		return a.saveLines(d.removePath, 548)
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
		lines = append(lines, a.saveLines(path, 548)...)
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
	drawFrame(s.pix, panelFrame(savePanelRect, layout.Fill, layout.Border))
	if a.flow.menuArt != nil {
		drawFrame(s.pix, windowFrame(image.Rect(8, 0, 632, 480), a.flow.menuArt))
	}
	s.label(w.Title, 38, 23, layout.TextColor)
	pointer, pointerOK := a.pointerFrame()
	// Without the install font the label is queued for the debug font.
	button := func(c saveControl, label string) {
		r := saveControlRect(c)
		disabled := c == saveDeleteControl && d.remove == nil && !a.flow.canDeleteSave()
		label = a.saveTextFit(label, r.Dx()-10)
		inside := pointerOK && pointer.In(r)
		b := pushButton{Rect: r, Hover: inside, Focus: d.focus == c, Pressed: d.press.Pressed(saveLatchID(c, 0)),
			Inside: inside, Disabled: disabled}
		font := a.flow.menuFont
		if font == nil {
			drawPushButton(s.pix, nil, b)
			textColor := layout.TextColor
			if disabled {
				textColor = loadDisabledText
			}
			s.label(label, r.Min.X+(r.Dx()-a.saveTextWidth(label))/2, r.Min.Y+5, textColor)
			return
		}
		b.Label = a.flow.menuDisplayText(label)
		drawPushButton(s.pix, font, b)
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
			button(saveDeleteControl, w.Delete)
		} else {
			button(saveWriteControl, w.Overwrite)
		}
		button(saveCancelControl, w.Back)
		return s, nil
	}
	s.label(w.Directory, 38, 58, layout.TextColor)
	s.label(w.Name, 38, 272, layout.TextColor)
	textH := listFallbackPitch - listPitchExtra
	if font := a.flow.menuFont; font != nil {
		textH = font.Height()
	}
	for _, c := range []saveControl{saveDirectoryControl, saveNameControl} {
		r := saveControlRect(c)
		value := d.request.Directory
		if c == saveNameControl {
			value = d.request.Name
		}
		runes := []rune(value)
		start := 0
		e := editField{Rect: r, TextH: textH, Focus: d.focus == c, Phase: a.blink.on()}
		if e.Focus {
			caret := min(d.caret, len(runes))
			for start < caret && a.saveTextWidth(string(runes[start:caret])) > r.Dx()-16 {
				start++
			}
			e.Caret = a.saveTextWidth(string(runes[start:caret]))
			if d.selectedText {
				e.SelTo = a.saveTextWidth(string(runes[start:]))
			}
		}
		shown := a.saveTextFit(string(runes[start:]), r.Dx()-12)
		drawEditField(s.pix, e, func(at image.Point) {
			// The text paints before the caret, which draws over it
			// (MENU-126); only a fontless debug app queues it.
			if font := a.flow.menuFont; font != nil {
				font.Draw(s.pix, a.flow.menuDisplayText(shown), at.X, at.Y, layout.TextColor)
				return
			}
			s.label(shown, at.X, at.Y, layout.TextColor)
		})
	}
	button(saveOpenControl, w.Open)
	button(saveUpControl, w.Up)
	if d.list != nil {
		box := a.saveListBox()
		pointer, pointerOK := a.pointerFrame()
		if font := a.flow.menuFont; font != nil {
			drawListBox(s.pix, font, a.media.scroll, box, d.list, func(row, width int) string {
				return a.flow.menuDisplayText(a.saveTextFit(d.list.Rows()[row].Text, width))
			}, pointer, pointerOK)
		} else {
			drawEditField(s.pix, listWell(box), nil)
			top, count := d.list.Visible()
			for i := 0; i < count; i++ {
				s.label(a.saveTextFit(d.list.Rows()[top+i].Text, box.Rect.Dx()-2*listTextX), box.Row(i).Min.X+listTextX, box.Row(i).Min.Y+listTextY, townShellText)
			}
			drawVScrollBar(s.pix, a.media.scroll, listBar(box, d.list).withPointer(pointer, pointerOK))
		}
		if d.list.Len() == 0 {
			s.label(w.Empty, 44, 96, layout.TextColor)
		}
		i := d.list.Selection() - len(d.directory.Directories)
		if i >= 0 && i < len(d.directory.Entries) {
			entry := d.directory.Entries[i]
			s.label(a.saveTextFit(entry.Label, 560), 40, 241, layout.TextColor)
		}
	}
	detail := w.TownDetail
	if d.request.OnMap {
		detail = w.MapSAVDetail
	}
	for i, line := range a.saveLines(detail, 560) {
		if i >= 3 {
			break
		}
		s.label(line, 38, 307+i*16, layout.TextColor)
	}
	message := a.flow.msg
	if message == "" {
		message = w.KeyHint
	}
	for i, line := range a.saveLines(message, 560) {
		if i >= 2 {
			break
		}
		s.label(line, 38, 388+i*16, layout.TextColor)
	}
	button(saveWriteControl, w.Save)
	button(saveDeleteControl, w.Delete)
	button(saveCancelControl, w.Cancel)
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
