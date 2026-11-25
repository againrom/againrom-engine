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

// mapChoiceRun is the map role's open state: the caller to answer.
type mapChoiceRun struct {
	onDone func(MapChoiceResult)
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

// mapRole is the list page as the multiplayer map choice: the engine's own
// panel with a title, the ruled list and the OK and Cancel buttons; it has no
// Delete, and Ok is off while no map is listed.
func mapRole(w MapChoiceWords) SavePage {
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
	f := a.flow
	run := &mapChoiceRun{onDone: onDone}
	entries := make([]SaveEntry, len(items))
	rows := make([]PickerRow, len(items))
	for i, item := range items {
		entries[i] = SaveEntry{Name: item.Source, Label: item.Row.Text}
		rows[i] = item.Row
	}
	f.openListRows(back, entries, rows, func() { f.loadUI.maps = run })
	f.msg = ""
	if len(items) == 0 {
		f.msg = f.mapChoiceWords().Empty
	}
}

// takeMapChoice ends the map role and hands back the caller to answer.
func (w *loadWindow) takeMapChoice() func(MapChoiceResult) {
	run := w.maps
	w.maps = nil
	if run == nil {
		return nil
	}
	return run.onDone
}

// chooseMap answers the caller with the selected map and leaves the page.
func (f *flow) chooseMap() {
	i, ok := f.loadList.Choose()
	if !ok || i < 0 || i >= len(f.saves) {
		return
	}
	f.leaveLoad(MapChoiceResult{Chosen: f.saves[i].Name, OK: true})
}
