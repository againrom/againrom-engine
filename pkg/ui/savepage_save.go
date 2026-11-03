package ui

import (
	"fmt"
	"image"
)

// savePageState is the one open save screen built from SavePage data.
type savePageState struct {
	page         SavePage
	ui           loadWindow
	list         *Picker
	saves        []SaveEntry
	directory    SaveDirectory
	request      SaveRequest
	focus        saveControl
	caret        int
	selectedText bool
	prepared     *PreparedSave
	removePath   string
	confirmTop   int
	listFocus    bool
}

func (s *savePageState) view() pageView {
	return pageView{page: s.page, ui: &s.ui, list: s.list, saves: s.saves, save: s}
}

func (s *savePageState) confirming() bool {
	return s.ui.confirm || s.prepared != nil || s.ui.remove != nil
}

func (s *savePageState) first() int {
	first := 0
	if s.page.directories {
		first += len(s.directory.Directories)
	}
	if !s.page.fileNames {
		first++
	}
	return first
}

func (f *flow) openSaveDialog() {
	if !f.savePoint(saveMenuWindow).open() {
		return
	}
	page := firstSavePage()
	if f.pages.saveSet != nil {
		page = *f.pages.saveSet
	}
	s := &savePageState{page: page, request: SaveRequest{
		OnMap: f.menuBack == ScreenMap, Directory: f.saveDialogSeams.Directory, Format: SaveSAV,
	}}
	*s.nameField() = page.initialName
	f.saveScreen = s
	f.msg = ""
	f.setMenuUp(true)
	f.setScreen(ScreenSave)
	f.refreshSavePage()
	s.setFocus(saveNameControl)
	s.selectedText = page.selectNameOnOpen
}

func (f *flow) refreshSavePage() {
	s := f.saveScreen
	if s == nil {
		return
	}
	s.ui.confirm, s.ui.remove, s.prepared = false, nil, nil
	s.removePath = ""
	if !s.page.preserveListPointer {
		s.ui.resetPointer()
	}
	s.directory = SaveDirectory{Path: s.request.Directory}
	s.saves = nil
	rows := []PickerRow(nil)
	if !s.page.fileNames {
		rows = append(rows, PickerRow{Text: s.page.EmptySlot, Choosable: true, Ink: s.page.EmptyInk})
	}
	s.list = NewPicker(rows).SetWindow(s.page.Rows)
	if f.saveDialogSeams.List == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	dir, err := f.saveDialogSeams.List(s.request.Directory)
	if err != nil {
		f.msg = err.Error()
		if s.page.stopListError {
			return
		}
	}
	s.directory = dir
	if dir.Path != "" {
		s.request.Directory = dir.Path
	} else if !s.page.directories {
		s.directory.Path = s.request.Directory
	}
	if s.page.directories {
		for _, name := range dir.Directories {
			rows = append(rows, PickerRow{Text: "[" + name + "]", Choosable: true})
		}
	}
	for _, e := range dir.Entries {
		if s.page.hidden(e.Name) {
			continue
		}
		s.saves = append(s.saves, e)
		label := e.Label
		if s.page.fileNames {
			label = saveListName(e.Name)
		}
		rows = append(rows, PickerRow{Text: label, Choosable: true})
	}
	s.list = NewPicker(rows).SetWindow(s.page.Rows)
	if s.page.clearListMessage {
		f.msg = ""
	}
}

func (s *savePageState) nameField() *string {
	if s.page.fileNames {
		return &s.request.Name
	}
	return &s.request.Label
}

func (s *savePageState) pick() {
	value := ""
	if e, ok := s.selectedSave(); ok {
		value = e.Label
		if s.page.fileNames {
			value = saveFieldName(e.Name)
		}
	}
	*s.nameField() = value
	s.selectedText = s.page.activateRows
	s.caret = len([]rune(value))
}

func (s *savePageState) target() (string, bool) {
	if s.page.fileNames {
		return s.request.Name, true
	}
	label := *s.nameField()
	if e, ok := s.selectedSave(); !ok || e.Label != label {
		for i, e := range s.saves {
			if e.Label == label {
				s.list.Select(i + s.first())
				break
			}
		}
	}
	if e, ok := s.selectedSave(); ok {
		return e.Name, true
	}
	if s.page.Slot == nil {
		return "", false
	}
	listed := make([]string, len(s.saves))
	for i, e := range s.saves {
		listed[i] = e.Name
	}
	name := s.page.Slot(s.request.Directory, listed)
	return name, name != ""
}

func (f *flow) submitSavePage() {
	s := f.saveScreen
	if s == nil || !s.page.fileNames && *s.nameField() == "" {
		return
	}
	f.msg = ""
	if s.prepared != nil {
		f.commitSavePage(true)
		return
	}
	name, ok := s.target()
	if !ok || f.saveDialogSeams.Prepare == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	request := s.request
	request.Name = name
	prepared, err := f.saveDialogSeams.Prepare(request)
	if err != nil {
		f.msg = err.Error()
		return
	}
	if prepared.Commit == nil || len(prepared.Paths) == 0 {
		f.msg = f.saveWords().Unavailable
		return
	}
	s.prepared = &prepared
	if s.page.overwriteQuestion && len(prepared.Existing) != 0 {
		s.confirmTop = 0
		s.setFocus(saveCancelControl)
		return
	}
	f.commitSavePage(len(prepared.Existing) != 0)
}

func (f *flow) commitSavePage(overwrite bool) {
	s := f.saveScreen
	if s == nil || s.prepared == nil || s.prepared.Commit == nil {
		return
	}
	notice := s.prepared.Notice
	_, err := s.prepared.Commit(overwrite)
	s.prepared = nil
	if err != nil {
		f.msg = err.Error()
		if s.page.keyboardFocus {
			s.setFocus(saveNameControl)
		}
		return
	}
	f.saveDialogSeams.Directory = s.request.Directory
	message := f.words.SaveAcknowledgement
	if message == "" {
		message = AuthoredWords().SaveAcknowledgement
	}
	if notice != "" {
		message += " " + notice
	}
	f.closeSavePage(message)
}

func (f *flow) closeSavePage(message string) {
	f.saveScreen = nil
	if f.pages.menu.WindowsCloseMenu {
		f.closeGameMenu()
		return
	}
	f.setScreen(ScreenGameMenu)
	f.rebuildGameMenu(gameMenuRoot, 0)
	f.setMenuUp(true)
	f.msg = message
}

func (s *savePageState) edit(in appInput) {
	field := s.editField()
	if field == nil {
		return
	}
	selection := s.page.selection
	if selection == TextCaretSelection {
		selection = TextStickyAllSelection
	}
	state := EditText(TextInputState{Text: *field, Caret: s.caret, SelectedAll: s.selectedText}, textInputKeys(in),
		TextInputRules{Limit: s.page.NameLimit, Selection: selection, SplitLineBreaks: true, LiteralArrows: true})
	*field, s.caret, s.selectedText = state.Text, state.Caret, state.SelectedAll
}

func (a *App) composeSaveScreen() (*image.RGBA, error) {
	s, err := a.savePagePaint()
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

func (a *App) drawSaveScreen() {
	a.hasMenu = false
	if pix, err := a.composeSaveScreen(); err == nil {
		a.writeCanvas(pix)
		return
	}
	s, err := a.savePagePaint()
	if err != nil {
		return
	}
	a.writeCanvas(s.pix)
	for _, label := range s.texts {
		a.printCanvas(label.text, label.at.X, label.at.Y)
	}
}
