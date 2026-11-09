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
	savePanelRect    = image.Rect(24, 12, 616, 468)
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
	saveListRect   = saveControlRect(saveListControl)
	saveListArg    = image.Rect(saveListRect.Min.X, saveListRect.Min.Y, saveListRect.Max.X-24, saveListRect.Max.Y)
	saveFocusColor = color.RGBA{R: 0xc8, G: 0xa2, B: 0x56, A: 0xff}
)

func saveControlRect(c saveControl) image.Rectangle { return saveDialogLayout.rect(int(c)) }

type saveTextBlock struct {
	word               string
	at                 image.Point
	width, rows, pitch int
}

// firstSavePage describes the file browser with the same page and widget kit.
func firstSavePage() SavePage {
	layout := AuthoredDialogueLayout()
	layout.Fill.A = 255
	label := func(word string, x, y int) PageText {
		return PageText{word: word, At: image.Pt(x, y), Ink: layout.TextColor}
	}
	button := func(c saveControl, word string) PageButton { return PageButton{Rect: saveControlRect(c), word: word} }
	return SavePage{
		Panel: savePanelRect, List: saveListArg, Rows: saveVisibleRows, NameBox: saveControlRect(saveNameControl),
		DirectoryBox: saveControlRect(saveDirectoryControl), NameLimit: 4096,
		Texts: []PageText{label("title", 38, 23),
			{word: "directory", At: image.Pt(38, 58), Ink: layout.TextColor, Listing: true},
			{word: "name", At: image.Pt(38, 272), Ink: layout.TextColor, Listing: true}},
		Buttons:    [3]PageButton{button(saveWriteControl, "save"), button(saveDeleteControl, "delete"), button(saveCancelControl, "cancel")},
		Navigation: [2]PageButton{button(saveOpenControl, "open"), button(saveUpControl, "up")},
		fileNames:  true, directories: true, overwriteQuestion: true, keyboardFocus: true,
		activateRows: true, retainDeletedRow: true, clearDeleteAttempt: true,
		clearListMessage: true, preserveListPointer: true, stopListError: true,
		initialName: "save", selection: TextReplaceAllSelection,
		focusOrder: []saveControl{saveDirectoryControl, saveOpenControl, saveUpControl, saveListControl, saveNameControl, saveWriteControl, saveDeleteControl, saveCancelControl},
		background: color.RGBA{R: 8, G: 10, B: 14, A: 255}, fill: layout.Fill, border: layout.Border,
		frame: image.Rect(8, 0, 632, 480), artOnly: true, deferredText: true,
		blocks: []saveTextBlock{{word: "context", at: image.Pt(38, 307), width: 560, rows: 3, pitch: 16},
			{word: "message-hint", at: image.Pt(38, 388), width: 560, rows: 2, pitch: 16}},
		empty: label("empty", 44, 96), detail: label("", 40, 241),
		confirmHeading: label("confirm", 38, 58), deleteHeading: label("delete-confirm", 38, 58),
		scrollHint: label("scroll-hint", 38, 408),
		confirmAt:  image.Pt(42, 92), confirmWidth: 548, confirmRows: saveConfirmationRows, confirmPitch: 18,
		confirmButtons: [3]PageButton{button(saveWriteControl, "overwrite"), button(saveDeleteControl, "delete"), button(saveCancelControl, "back")},
		wheel:          saveListFocused, hoverWheel: saveListHover, controlSound: true, standalone: true, errorName: "save dialog",
		latchIDs: []int{0, 1, 2, 3, 4, 5, 6, 7}, ink: layout.TextColor,
		fitButtons: true, selectNameOnOpen: true, reportDeletePath: true,
	}
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
func (a *App) saveLines(value string, width int) []string {
	return saveRule.plain(lineBox{Measure: a.saveTextWidth, Width: width}, value)
}

func (f *flow) saveCaption(word, fallback string) string {
	w := f.saveWords()
	switch word {
	case "title":
		return w.Title
	case "directory":
		return w.Directory
	case "name":
		return w.Name
	case "open":
		return w.Open
	case "up":
		return w.Up
	case "save":
		return w.Save
	case "delete":
		return w.Delete
	case "cancel":
		return w.Cancel
	case "back":
		return w.Back
	case "overwrite":
		return w.Overwrite
	case "confirm":
		return w.Confirm
	case "delete-confirm":
		return w.DeleteConfirm
	case "empty":
		return w.Empty
	case "scroll-hint":
		return w.ScrollHint
	case "context":
		if f.saveScreen.request.OnMap {
			return w.MapSAVDetail
		}
		return w.TownDetail
	case "message-hint":
		if f.msg != "" {
			return f.msg
		}
		return w.KeyHint
	}
	return fallback
}

func (a *App) saveConfirmationLines() []string {
	s := a.flow.saveScreen
	if s == nil {
		return nil
	}
	if s.ui.remove != nil {
		return a.saveLines(s.removePath, s.page.confirmWidth)
	}
	if s.prepared == nil {
		return nil
	}
	w := a.flow.saveWords()
	var lines []string
	for _, path := range s.prepared.Paths {
		verb := w.Create
		for _, existing := range s.prepared.Existing {
			if path == existing {
				verb = w.Replace
				break
			}
		}
		lines = append(lines, verb)
		lines = append(lines, a.saveLines(path, s.page.confirmWidth)...)
		lines = append(lines, "")
	}
	return lines
}

func (a *App) clampSaveConfirmationScroll() {
	s := a.flow.saveScreen
	if s == nil || s.page.confirmRows == 0 {
		return
	}
	s.confirmTop = max(0, min(s.confirmTop, len(a.saveConfirmationLines())-s.page.confirmRows))
}

func (a *App) savePagePaint() (*savePaint, error) {
	state := a.flow.saveScreen
	if state == nil {
		return nil, fmt.Errorf("save dialog is closed")
	}
	v := state.view()
	page, font := v.page, a.flow.menuFont
	if font == nil && !page.deferredText {
		return nil, fmt.Errorf("load list has no installed font")
	}
	s := &savePaint{pix: image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))}
	if page.background.A != 0 {
		draw.Draw(s.pix, s.pix.Bounds(), image.NewUniform(page.background), image.Point{}, draw.Src)
	} else {
		a.flow.menuPlacement().backdrop.paint(a, s.pix)
	}
	frames := []dialogFrame{}
	if page.fill.A != 0 {
		frames = append(frames, dialogFrame{Spec: panelFrame(page.Panel, page.fill, page.border)})
	}
	r := page.frame
	if r.Empty() {
		r = page.Panel
	}
	if !page.artOnly || a.flow.menuArt != nil {
		frames = append(frames, dialogFrame{Spec: windowFrame(r, a.flow.menuArt)})
	}
	drawDialog(s.pix, font, Dialog{Box: s.pix.Bounds(), Frames: frames})
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
	hover := pointerOK && !v.asking()
	button := func(c saveControl, b PageButton) {
		r := page.saveLayout().rect(int(c))
		value := a.flow.saveCaption(b.word, b.Caption)
		if page.fitButtons {
			value = a.flow.menuDisplayText(a.saveTextFit(value, r.Dx()-10))
		}
		inside := hover && pointer.In(r)
		disabled := a.flow.saveControlDisabled(state, c)
		bstate := pushButton{Rect: r, Label: value, Hover: inside, Inside: inside, Disabled: disabled,
			Focus: page.keyboardFocus && state.focus == c, Pressed: state.ui.press.Pressed(state.latchID(c, 0))}
		if font != nil {
			drawDialog(s.pix, font, Dialog{Box: s.pix.Bounds(), Frames: []dialogFrame{}, Pushes: []pushButton{bstate}})
			return
		}
		bstate.Label = ""
		drawDialog(s.pix, nil, Dialog{Box: s.pix.Bounds(), Frames: []dialogFrame{}, Pushes: []pushButton{bstate}})
		cink := ink
		if disabled {
			cink = loadDisabledText
		}
		s.label(value, r.Min.X+(r.Dx()-a.saveTextWidth(value))/2, r.Min.Y+5, cink)
	}
	if v.confirmingInPage() {
		if page.confirmRows > 0 {
			heading := page.confirmHeading
			if state.ui.remove != nil {
				heading = page.deleteHeading
			}
			caption(heading)
			a.clampSaveConfirmationScroll()
			lines := a.saveConfirmationLines()
			for i := 0; i < page.confirmRows && state.confirmTop+i < len(lines); i++ {
				label(lines[state.confirmTop+i], page.confirmAt.Add(image.Pt(0, i*page.confirmPitch)), ink)
			}
			if len(lines) > page.confirmRows {
				caption(page.scrollHint)
			}
		} else {
			a.drawPageQuestion(s.pix, font, v)
		}
	} else {
		for _, c := range []saveControl{saveDirectoryControl, saveNameControl} {
			r := page.saveLayout().rect(int(c))
			if r.Empty() {
				continue
			}
			value := *state.nameField()
			if c == saveDirectoryControl {
				value = state.request.Directory
			}
			runes := []rune(value)
			start := 0
			textH := listFallbackPitch - listPitchExtra
			if font != nil {
				textH = font.Height()
			}
			e := editField{Rect: r, TextH: textH, Focus: !page.keyboardFocus || state.focus == c, Phase: a.blink.on()}
			if e.Focus {
				caret := min(state.caret, len(runes))
				for start < caret && a.saveTextWidth(string(runes[start:caret])) > r.Dx()-16 {
					start++
				}
				e.Caret = a.saveTextWidth(string(runes[start:caret]))
				if state.selectedText {
					e.SelTo = a.saveTextWidth(string(runes[start:]))
				}
			}
			shown := a.saveTextFit(string(runes[start:]), r.Dx()-12)
			drawEditField(s.pix, e, func(at image.Point) {
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
		if state.list != nil {
			box := a.pageListBox(v)
			if font != nil {
				a.drawPageList(s.pix, font, v, func(row, width int) string {
					value := state.list.Rows()[row].Text
					if page.fileNames {
						return a.flow.menuDisplayText(a.saveTextFit(value, width))
					}
					return a.flow.menuDisplayText(a.fitLoadText(value, width))
				})
			} else {
				drawEditField(s.pix, listWell(box), nil)
				top, count := state.list.Visible()
				for i := 0; i < count; i++ {
					s.label(a.saveTextFit(state.list.Rows()[top+i].Text, box.Rect.Dx()-2*listTextX),
						box.Row(i).Min.X+listTextX, box.Row(i).Min.Y+listTextY, townShellText)
				}
				drawVScrollBar(s.pix, a.media.scroll, listBar(box, state.list).withPointer(pointer, pointerOK))
			}
			if state.list.Len() == 0 && page.empty.word != "" {
				caption(page.empty)
			}
			if entry, ok := state.selectedSave(); ok && page.detail.At != (image.Point{}) {
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
			if c == saveWriteControl && state.ui.remove != nil || c == saveDeleteControl && state.ui.remove == nil {
				continue
			}
			b = page.confirmButtons[i]
		}
		button(c, b)
	}
	if v.asking() {
		w := &state.ui.question
		w.compose(s.pix, font, page.Panel, a.flow.menuArt, a.media.scroll, func(s string) string { return s }, pointer, pointerOK)
	}
	return s, nil
}
