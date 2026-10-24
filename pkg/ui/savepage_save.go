package ui

import (
	"image"
)

// savePageState is an open save page: its directory, the listed saves behind
// rows 1.., and the name box.
type savePageState struct {
	page      SavePage
	ui        loadWindow
	list      *Picker
	saves     []SaveEntry
	directory string
	onMap     bool
	name      []rune
	caret     int
	// selectAll makes the next edit replace the whole name.
	selectAll bool
}

func (s *savePageState) view() pageView {
	return pageView{page: s.page, ui: &s.ui, list: s.list, saves: s.saves, save: s}
}

// openSavePage shows the configured save page with row 0 selected and an
// empty name box.
func (f *flow) openSavePage() {
	if !f.atGameSavePoint() || f.pages.saveSet == nil {
		return
	}
	f.pages.save = &savePageState{page: *f.pages.saveSet, directory: f.saveDialogSeams.Directory, onMap: f.menuBack == ScreenMap}
	f.msg = ""
	f.setMenuUp(true)
	f.setScreen(ScreenSave)
	f.refreshSavePage()
}

// refreshSavePage reads the directory again: row 0 is the Empty Slot and
// rows 1.. the saves the page does not hide, in the list's order.
func (f *flow) refreshSavePage() {
	s := f.pages.save
	if s == nil {
		return
	}
	s.ui.confirm, s.ui.remove = false, nil
	s.ui.resetPointer()
	s.saves = nil
	if f.saveDialogSeams.List != nil {
		dir, err := f.saveDialogSeams.List(s.directory)
		if err != nil {
			f.msg = err.Error()
		}
		if dir.Path != "" {
			s.directory = dir.Path
		}
		for _, e := range dir.Entries {
			if !s.page.hidden(e.Name) {
				s.saves = append(s.saves, e)
			}
		}
	} else {
		f.msg = f.saveWords().Unavailable
	}
	rows := []PickerRow{{Text: s.page.EmptySlot, Choosable: true}}
	for _, e := range s.saves {
		rows = append(rows, PickerRow{Text: e.Label, Choosable: true})
	}
	s.list = NewPicker(rows).SetWindow(s.page.Rows)
}

// pick puts the selected row's label in the name box; row 0 empties it.
func (s *savePageState) pick() {
	s.name, s.selectAll = nil, false
	if i := s.list.Selection() - 1; i >= 0 && i < len(s.saves) {
		s.name = []rune(s.saves[i].Label)
	}
	s.caret = len(s.name)
}

// target is the file Ok writes. A chosen row whose label equals the name is
// the file. Otherwise the first listed row with that label is taken and
// selected; row 0 writes to the file the page's slot rule names.
func (s *savePageState) target() (string, bool) {
	label := string(s.name)
	if i := s.list.Selection() - 1; i < 0 || i >= len(s.saves) || s.saves[i].Label != label {
		for i, e := range s.saves {
			if e.Label == label {
				s.list.Select(i + 1)
				break
			}
		}
	}
	if i := s.list.Selection() - 1; i >= 0 && i < len(s.saves) {
		return s.saves[i].Name, true
	}
	if s.page.Slot == nil {
		return "", false
	}
	listed := make([]string, len(s.saves))
	for i, e := range s.saves {
		listed[i] = e.Name
	}
	name := s.page.Slot(s.directory, listed)
	return name, name != ""
}

// submitSavePage writes the current state under the typed label. An existing
// file is overwritten without confirmation.
func (f *flow) submitSavePage() {
	s := f.pages.save
	if s == nil || len(s.name) == 0 {
		return
	}
	f.msg = ""
	name, ok := s.target()
	if !ok || f.saveDialogSeams.Prepare == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	prepared, err := f.saveDialogSeams.Prepare(SaveRequest{OnMap: s.onMap, Directory: s.directory, Name: name,
		Label: string(s.name), Format: SaveSAV})
	if err != nil {
		f.msg = err.Error()
		return
	}
	if prepared.Commit == nil || len(prepared.Paths) == 0 {
		f.msg = f.saveWords().Unavailable
		return
	}
	if _, err := prepared.Commit(len(prepared.Existing) != 0); err != nil {
		f.msg = err.Error()
		return
	}
	f.saveDialogSeams.Directory = s.directory
	message := f.words.SaveAcknowledgement
	if message == "" {
		message = AuthoredWords().SaveAcknowledgement
	}
	if prepared.Notice != "" {
		message = message + " " + prepared.Notice
	}
	f.closeSavePage(message)
}

func (f *flow) closeSavePage(message string) {
	f.pages.save = nil
	f.setScreen(ScreenGameMenu)
	f.rebuildGameMenu(gameMenuRoot, 0)
	f.setMenuUp(true)
	f.msg = message
}

// edit applies the name box's keys: caret moves, deletion and typed text, up
// to the page's limit.
func (s *savePageState) edit(in appInput) {
	state := EditText(TextInputState{Text: string(s.name), Caret: s.caret, SelectedAll: s.selectAll}, textInputKeys(in),
		TextInputRules{Limit: s.page.NameLimit, Selection: TextStickyAllSelection, SplitLineBreaks: true, LiteralArrows: true})
	s.name, s.caret, s.selectAll = []rune(state.Text), state.Caret, state.SelectedAll
}

// composeSaveScreen draws the open save page, else the save dialog.
func (a *App) composeSaveScreen() (*image.RGBA, error) {
	if s := a.flow.pages.save; s != nil {
		return a.composePage(s.view())
	}
	return a.composeSaveDialogScreen()
}

func (a *App) drawSaveScreen() {
	if s := a.flow.pages.save; s != nil {
		a.hasMenu = false
		if pix, err := a.composePage(s.view()); err == nil {
			a.writeCanvas(pix)
		}
		return
	}
	a.drawSaveDialog()
}

// drawPageName draws the save page's name box with its caret.
func (a *App) drawPageName(pix *image.RGBA, v pageView) {
	s, font := v.save, a.flow.menuFont
	r := v.page.NameBox
	start := 0
	caret := min(s.caret, len(s.name))
	for start < caret && a.saveTextWidth(string(s.name[start:caret])) > r.Dx()-16 {
		start++
	}
	e := editField{Rect: r, TextH: font.Height(), Focus: true, Phase: a.blink.on(),
		Caret: a.saveTextWidth(string(s.name[start:caret]))}
	if s.selectAll {
		e.SelTo = a.saveTextWidth(string(s.name[start:]))
	}
	shown := a.saveTextFit(string(s.name[start:]), r.Dx()-12)
	drawEditField(pix, e, func(at image.Point) {
		font.Draw(pix, a.flow.menuDisplayText(shown), at.X, at.Y, townShellText)
	})
}
