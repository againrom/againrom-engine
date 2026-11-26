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
	// nameField is the text the page's name field edits on s; nil for a page
	// without a name field.
	nameField func(s *savePageState) *string
	// write is Ok on a page that writes: the name field's file, after the
	// overwrite question; nil leaves Ok to accept.
	write func(f *flow)
	// buttonDisabled answers whether page button i is disabled; nil is the
	// list rule: Ok needs a listed save, Delete a deletable one.
	buttonDisabled func(f *flow, v pageView, i int) bool
	// confirmDelete is Delete on the selected row, with its overwrite and
	// delete questions; nil is the list's delete question.
	confirmDelete func(f *flow)
	// cancel is Cancel and Escape; nil is the list's: leave a question, else
	// close the page.
	cancel func(f *flow)
	// step drives one frame of a page that keeps its own controls; nil is the
	// list's step.
	step func(a *App, in appInput)
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
// opens: the store's character rows, then the New row. Ok on the New row
// opens the store's generator; Ok on a character row does nothing yet
// (DIV-2369).
func characterListRole() *listRole {
	return &listRole{
		roleName: characterRoleName,
		page:     func(f *flow) SavePage { return f.installedLoadPage().asCharacterList(f.characterPage()) },
		list:     (*flow).characterRows,
		accept:   (*flow).newCharacter,
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

// saveListRole is the Save page: the page keeps its own list, built by
// refreshSavePage, so the row has no list; its name field, its write, its
// delete seams, its Delete, its Cancel and its step are the row's.
func saveListRole() *listRole {
	return &listRole{
		roleName:  "save",
		page:      func(f *flow) SavePage { return f.saveScreen.page },
		nameField: (*savePageState).fileOrLabel,
		write:     func(f *flow) { f.activateSaveControl(saveWriteControl) },
		deletes:   (*flow).saveDeletes,
		buttonDisabled: func(f *flow, v pageView, i int) bool {
			controls := [...]saveControl{saveWriteControl, saveDeleteControl, saveCancelControl}
			return i >= 0 && i < len(controls) && f.saveControlDisabled(v.save, controls[i])
		},
		confirmDelete: (*flow).confirmSaveDelete,
		cancel:        (*flow).cancelSavePage,
		step:          (*App).stepSaveScreen,
	}
}

// saveDeletes are the save dialog's delete seams on the open directory.
func (f *flow) saveDeletes() (func(string) bool, func(string) (func() error, error)) {
	seams := f.saveDialogSeams
	if seams.CanDelete == nil || seams.PrepareDelete == nil || f.saveScreen == nil {
		return nil, nil
	}
	dir := f.saveScreen.directory.Path
	return func(name string) bool { return seams.CanDelete(dir, name) },
		func(name string) (func() error, error) { return seams.PrepareDelete(dir, name) }
}

// row is the page's role row.
func (s *savePageState) row() *listRole {
	if s.ui.role == nil {
		s.ui.role = saveListRole()
	}
	return s.ui.role
}

// fileOrLabel is the save page's name field: the file name on a page that
// names files, else the label.
func (s *savePageState) fileOrLabel() *string {
	if s.page.fileNames {
		return &s.request.Name
	}
	return &s.request.Label
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

// newCharacter is Ok on the character list: on the New row it shows the
// store's generator over the list (R2-ENGINE-1840, MENU-169). The
// generator's accept stores the character and opens the list again with
// the new file's row selected; its Back and Escape return to the list.
func (f *flow) newCharacter() {
	i, ok := f.loadList.Choose()
	create := f.pages.characterFiles.New
	if !ok || i != len(f.saves) || create == nil {
		return
	}
	e := create()
	if e == nil || e.Begin == nil {
		return
	}
	listed := make(map[string]bool, len(f.saves))
	for _, s := range f.saves {
		listed[s.Name] = true
	}
	back := f.loadBack
	begin := func(res ChargenResult) (MapOpener, error) {
		if open, err := e.Begin(res); err != nil || open != nil {
			return open, err
		}
		f.openListPage(back, characterListRole())
		for j, s := range f.saves {
			if !listed[s.Name] {
				f.loadList.Select(j)
				break
			}
		}
		return nil, nil
	}
	f.armChargen(e.Model, begin, ScreenLoad)
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
