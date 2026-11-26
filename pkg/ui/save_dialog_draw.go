package ui

import (
	"image"
	"image/color"

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
