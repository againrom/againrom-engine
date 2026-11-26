package ui

import "fmt"

// listRole is one role of the list page: the screen it shows on, its page,
// where its rows come from, what Ok does, whom closing answers and which rows
// Delete may remove. The page reads the open role's fields and never asks
// which role is open; a new role is a new row.
type listRole struct {
	// roleName names the row; CharacterList is its one reader.
	roleName string
	// shownOn is the screen the role's page shows on.
	shownOn Screen
	// page is the role's page data.
	page func(f *flow) SavePage
	// list answers the listed entries, their rows and the message line.
	list func(f *flow, v pageView) listing
	// accept is Ok on the open page; nil does nothing.
	accept func(f *flow)
	// answer hears the role's result once the page has closed: the chosen
	// entry's name and true from Ok, nothing and false from Cancel; nil
	// hears nothing.
	answer func(f *flow, chosen string, ok bool)
	// deletes are the role's delete seams; nil lets Delete remove no row.
	deletes func(f *flow) (func(string) bool, func(string) (func() error, error))
	// named: the page has a name field the player edits and a row picks.
	named bool
	// chooses: Ok chooses the selected row, which sounds the common control
	// and lays the viewer out again before the row's accept runs.
	chooses bool
	// keepsMenuRow: closing restores the game menu's chosen row, not its
	// first.
	keepsMenuRow bool
	// reports: closing shows the chosen text on the menu's message line.
	reports bool
}

// listing is a role's rows: the saves behind them, the rows drawn, the
// message line and how many rows precede the first save.
type listing struct {
	saves   []SaveEntry
	rows    []PickerRow
	message string
	lead    int
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
		shownOn:  ScreenLoad,
		page:     (*flow).installedLoadPage,
		list:     (*flow).saveRows,
		accept:   (*flow).loadChosen,
		deletes: func(f *flow) (func(string) bool, func(string) (func() error, error)) {
			return f.loadUI.canDelete, f.loadUI.prepareDelete
		},
		chooses:      true,
		keepsMenuRow: true,
	}
}

// characterListRole is the character list the menu's Multiplayer button
// opens: the store's character rows, then the New row. Ok on the New row
// opens the store's generator; Ok on a character row opens the map choice
// (DIV-2369, DIV-5196).
func characterListRole() *listRole {
	return &listRole{
		roleName:     characterRoleName,
		shownOn:      ScreenLoad,
		page:         func(f *flow) SavePage { return f.installedLoadPage().asCharacterList(f.characterPage()) },
		list:         (*flow).characterRows,
		accept:       (*flow).acceptCharacter,
		deletes:      (*flow).characterDeletes,
		chooses:      true,
		keepsMenuRow: true,
	}
}

// mapListRole is the multiplayer map choice over items: Ok and Cancel close
// the page and answer onDone; no row is deleted.
func mapListRole(items []MapChoiceItem, onDone func(MapChoiceResult)) *listRole {
	return &listRole{
		roleName: "maps",
		shownOn:  ScreenLoad,
		page:     func(f *flow) SavePage { return mapChoicePage(f.mapChoiceWords()) },
		list: func(f *flow, _ pageView) listing {
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
			return listing{saves: entries, rows: rows, message: message}
		},
		accept: (*flow).chooseMap,
		answer: func(_ *flow, chosen string, ok bool) {
			if onDone != nil {
				onDone(MapChoiceResult{Chosen: chosen, OK: ok})
			}
		},
		chooses:      true,
		keepsMenuRow: true,
	}
}

// saveListRole is the Save page: the open directory's files after the empty
// slot and the directories, Ok writes the name field's file, Delete removes
// a listed file through the save dialog's seams, and closing reports the
// outcome on the menu.
func saveListRole() *listRole {
	return &listRole{
		roleName: "save",
		shownOn:  ScreenSave,
		page:     func(f *flow) SavePage { return f.saveScreen.page },
		list:     (*flow).saveFileRows,
		accept:   (*flow).submitSavePage,
		answer:   func(f *flow, _ string, _ bool) { f.saveScreen = nil },
		deletes:  (*flow).saveDeletes,
		named:    true,
		reports:  true,
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

// saveFileRows are the Save page's rows: the empty slot, the directories
// where the page browses, then the listed files the page does not hide. The
// open directory and the message line follow the listing's outcome.
func (f *flow) saveFileRows(v pageView) listing {
	e, page := v.edit, v.page
	e.directory = SaveDirectory{Path: e.request.Directory}
	var rows []PickerRow
	if !page.fileNames {
		rows = append(rows, PickerRow{Text: page.EmptySlot, Choosable: true, Ink: page.EmptyInk})
	}
	if f.saveDialogSeams.List == nil {
		return listing{rows: rows, message: f.saveWords().Unavailable, lead: len(rows)}
	}
	message := f.msg
	dir, err := f.saveDialogSeams.List(e.request.Directory)
	if err != nil {
		message = err.Error()
		if page.stopListError {
			return listing{rows: rows, message: message, lead: len(rows)}
		}
	}
	e.directory = dir
	if dir.Path != "" {
		e.request.Directory = dir.Path
	} else if !page.directories {
		e.directory.Path = e.request.Directory
	}
	if page.directories {
		for _, name := range dir.Directories {
			rows = append(rows, PickerRow{Text: "[" + name + "]", Choosable: true})
		}
	}
	out := listing{lead: len(rows)}
	for _, entry := range dir.Entries {
		if page.hidden(entry.Name) {
			continue
		}
		out.saves = append(out.saves, entry)
		label := entry.Label
		if page.fileNames {
			label = saveListName(entry.Name)
		}
		rows = append(rows, PickerRow{Text: label, Choosable: true})
	}
	out.rows = rows
	out.message = message
	if page.clearListMessage {
		out.message = ""
	}
	return out
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
func (f *flow) saveRows(pageView) listing {
	if f.saveList == nil {
		return listing{rows: labelRows(nil), message: "this build has nowhere to load from"}
	}
	saves := f.saveList()
	message := ""
	if len(saves) == 0 {
		message = "no saved games"
	}
	return listing{saves: saves, rows: labelRows(saves), message: message}
}

// characterRows are the character store's rows, then the New row, with the
// store's message line.
func (f *flow) characterRows(pageView) listing {
	var saves []SaveEntry
	message := ""
	if list := f.pages.characterFiles.List; list != nil {
		saves, message = list()
	}
	rows := append(labelRows(saves), PickerRow{Text: f.characterPage().New, Choosable: true})
	return listing{saves: saves, rows: rows, message: message}
}

// acceptCharacter is Ok on the character list: the New row runs
// newCharacter, a character row chooseCharacterMap.
func (f *flow) acceptCharacter() {
	i, ok := f.loadList.Choose()
	switch {
	case !ok:
	case i == len(f.saves):
		f.newCharacter()
	case i >= 0 && i < len(f.saves):
		f.chooseCharacterMap(f.saves[i].Name)
	}
}

// showCharacters opens the character list over back with the first row pick
// accepts selected.
func (f *flow) showCharacters(back Screen, pick func(SaveEntry) bool) {
	f.openListPage(back, characterListRole())
	for j, s := range f.saves {
		if pick(s) {
			f.loadList.Select(j)
			break
		}
	}
}

// newCharacter shows the store's generator over the list (R2-ENGINE-1840,
// MENU-169). The generator's accept stores the character and opens the list
// again with the new file's row selected; its Back and Escape return to the
// list.
func (f *flow) newCharacter() {
	create := f.pages.characterFiles.New
	if create == nil {
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
		f.showCharacters(back, func(s SaveEntry) bool { return !listed[s.Name] })
		return nil, nil
	}
	f.armChargen(e.Model, begin, ScreenLoad)
}

// chooseCharacterMap opens the map choice for the character file named
// character, with no connection step (DIV-5196). Cancel and Escape show the
// character list again with that row selected; Ok enters the chosen map with
// the character, and a refusal shows the list with the reason on its message
// line. The end of the entered map shows the list again the same way.
func (f *flow) chooseCharacterMap(character string) {
	c := f.pages.characterFiles
	if c.Maps == nil || c.Enter == nil {
		return
	}
	back := f.loadBack
	again := func(message string) {
		f.showCharacters(back, func(s SaveEntry) bool { return s.Name == character })
		if message != "" {
			f.msg = message
		}
	}
	f.openListPage(back, mapListRole(c.Maps(), func(res MapChoiceResult) {
		if !res.OK {
			again("")
			return
		}
		open, err := c.Enter(character, res.Chosen)
		if err == nil {
			err = f.enterOpened(open)
		}
		if err != nil {
			again(err.Error())
			return
		}
		f.loadUI.session = again
	}))
}

// enterOpened runs open and shows the map it opens in place of the current
// screen.
func (f *flow) enterOpened(open MapOpener) error {
	if open == nil {
		return fmt.Errorf("no map to enter")
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		return err
	}
	f.leaveMap()
	f.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	return nil
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
