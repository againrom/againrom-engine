package ui

import (
	"image"
)

// MapChoiceWords holds the multiplayer map choice page's title, button
// captions and empty-list text, encoded for the installed font.
type MapChoiceWords struct {
	Title, OK, Cancel, Empty string
}

// MapChoiceItem is one row of the multiplayer map choice list: Source is the
// map file name returned to the caller on Ok, Row is the PickerRow shown.
type MapChoiceItem struct {
	Source string
	Row    PickerRow
}

// MapChoiceResult is what the multiplayer map choice page returns when Ok or
// Cancel closes it.
type MapChoiceResult struct {
	Chosen string
	OK     bool
}

var (
	mapChoicePanel    = image.Rect(82, 48, 558, 392)
	mapChoiceTitleBox = image.Rect(102, 56, 538, 84)
	mapChoiceListArg  = image.Rect(120, 108, 503, 108)
	mapChoiceMessage  = image.Rect(120, 314, 520, 334)
	mapChoiceOKRect   = image.Rect(154, 338, 250, 362)
	mapChoiceNoRect   = image.Rect(398, 338, 494, 362)
)

// SetMapChoiceWords installs the multiplayer map choice page's words.
func (a *App) SetMapChoiceWords(w MapChoiceWords) { a.flow.pages.mapWords = w }

// mapChoiceWords returns the installed words, defaulting empty fields.
func (f *flow) mapChoiceWords() MapChoiceWords {
	w := f.pages.mapWords
	if w.Title == "" {
		w.Title = "Select Map"
	}
	if w.OK == "" {
		w.OK = "OK"
	}
	if w.Cancel == "" {
		w.Cancel = "Cancel"
	}
	if w.Empty == "" {
		w.Empty = "No maps"
	}
	return w
}

// mapChoicePage is the list page as the multiplayer map choice: the engine's
// own panel with a title, the ruled list and the OK and Cancel buttons; it
// has no Delete, and Ok is off while no map is listed.
func mapChoicePage(w MapChoiceWords) SavePage {
	p := SavePage{
		Panel:       mapChoicePanel,
		Texts:       []PageText{{Text: w.Title, Box: mapChoiceTitleBox, Ink: townShellText}},
		List:        mapChoiceListArg,
		Rows:        loadVisibleRows,
		Message:     mapChoiceMessage,
		Ruled:       true,
		RowHint:     true,
		OKNeedsSave: true,
	}
	p.Buttons[loadOKButton] = PageButton{Rect: mapChoiceOKRect, Caption: w.OK}
	p.Buttons[loadCancelButton] = PageButton{Rect: mapChoiceNoRect, Caption: w.Cancel}
	return p
}

// OpenMapChoice shows the multiplayer map choice page over items, returning
// the player's choice (or cancellation) to onDone and restoring back.
func (a *App) OpenMapChoice(items []MapChoiceItem, back Screen, onDone func(MapChoiceResult)) {
	a.flow.openListPage(back, mapListRole(items, onDone))
}

// chooseMap answers the caller with the selected map and leaves the page.
func (f *flow) chooseMap() {
	i, ok := f.loadList.Choose()
	if !ok || i < 0 || i >= len(f.saves) {
		return
	}
	f.leaveLoad(f.saves[i].Name, true)
}
