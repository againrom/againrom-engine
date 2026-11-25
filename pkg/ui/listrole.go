package ui

// listRole is one role of the list page on the load screen: its page, where
// its rows come from, what Ok does, whom closing answers and which rows
// Delete may remove. The page reads the open role's fields and never asks
// which role is open; a new role is a new row.
type listRole struct {
	// roleName names the row; CharacterList is its one reader.
	roleName string
	// page is the role's page data.
	page func(f *flow) SavePage
	// list answers the listed entries, their rows and the message line.
	list func(f *flow) ([]SaveEntry, []PickerRow, string)
	// accept is Ok on the open page; nil does nothing.
	accept func(f *flow)
	// answer hears the role's result once the page has closed: the chosen
	// entry's name and true from Ok, nothing and false from Cancel; nil
	// hears nothing.
	answer func(chosen string, ok bool)
	// deletes are the role's delete seams; nil lets Delete remove no row.
	deletes func(f *flow) (func(string) bool, func(string) (func() error, error))
}

const characterRoleName = "characters"

// The rows are built by functions, not held in package variables: a row's
// fields reach openLoad, which names the load row, and a variable would
// initialize through itself.

// loadListRole is the Load page: the save list, Ok loads the chosen save and
// Delete removes it through the saves' seams.
func loadListRole() *listRole {
	return &listRole{
		roleName: "load",
		page:     (*flow).installedLoadPage,
		list:     (*flow).saveRows,
		accept:   (*flow).loadChosen,
		deletes: func(f *flow) (func(string) bool, func(string) (func() error, error)) {
			return f.loadUI.canDelete, f.loadUI.prepareDelete
		},
	}
}

// characterListRole is the character list the menu's Multiplayer button
// opens: the store's character rows, then the New row. Ok does nothing: no
// character is played and no creation runs from the New row yet (DIV-2369).
func characterListRole() *listRole {
	return &listRole{
		roleName: characterRoleName,
		page:     func(f *flow) SavePage { return f.installedLoadPage().asCharacterList(f.characterPage()) },
		list:     (*flow).characterRows,
		deletes:  (*flow).characterDeletes,
	}
}

// mapListRole is the multiplayer map choice over items: Ok and Cancel close
// the page and answer onDone; no row is deleted.
func mapListRole(items []MapChoiceItem, onDone func(MapChoiceResult)) *listRole {
	return &listRole{
		roleName: "maps",
		page:     func(f *flow) SavePage { return mapChoicePage(f.mapChoiceWords()) },
		list: func(f *flow) ([]SaveEntry, []PickerRow, string) {
			entries := make([]SaveEntry, len(items))
			rows := make([]PickerRow, len(items))
			for i, item := range items {
				entries[i] = SaveEntry{Name: item.Source, Label: item.Row.Text}
				rows[i] = item.Row
			}
			message := ""
			if len(items) == 0 {
				message = f.mapChoiceWords().Empty
			}
			return entries, rows, message
		},
		accept: (*flow).chooseMap,
		answer: func(chosen string, ok bool) {
			if onDone != nil {
				onDone(MapChoiceResult{Chosen: chosen, OK: ok})
			}
		},
	}
}

// openRole is the role the list page shows; with none opened it is the
// Load page.
func (w *loadWindow) openRole() *listRole {
	if w.role == nil {
		return loadListRole()
	}
	return w.role
}

// labelRows are the choosable rows of entries' labels.
func labelRows(entries []SaveEntry) []PickerRow {
	rows := make([]PickerRow, len(entries))
	for i, e := range entries {
		rows[i] = PickerRow{Text: e.Label, Choosable: true}
	}
	return rows
}

// saveRows are the Load page's rows: the listed saves, or the line saying
// why none is listed.
func (f *flow) saveRows() ([]SaveEntry, []PickerRow, string) {
	if f.saveList == nil {
		return nil, labelRows(nil), "this build has nowhere to load from"
	}
	saves := f.saveList()
	message := ""
	if len(saves) == 0 {
		message = "no saved games"
	}
	return saves, labelRows(saves), message
}

// characterRows are the character store's rows, then the New row, with the
// store's message line.
func (f *flow) characterRows() ([]SaveEntry, []PickerRow, string) {
	var saves []SaveEntry
	message := ""
	if list := f.pages.characterFiles.List; list != nil {
		saves, message = list()
	}
	rows := append(labelRows(saves), PickerRow{Text: f.characterPage().New, Choosable: true})
	return saves, rows, message
}

// characterDeletes are the character store's delete seams where the game's
// character page shows Delete, else none.
func (f *flow) characterDeletes() (func(string) bool, func(string) (func() error, error)) {
	if !f.characterPage().Delete {
		return nil, nil
	}
	c := f.pages.characterFiles
	return c.CanDelete, c.PrepareDelete
}
