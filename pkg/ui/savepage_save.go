package ui

// savePageState is the one open save page: the page's data, its window and
// list, and the edit state its name field and file browser keep.
type savePageState struct {
	page  SavePage
	ui    loadWindow
	list  *Picker
	saves []SaveEntry
	pageEdit
}

func (s *savePageState) view() pageView {
	return pageView{page: s.page, ui: &s.ui, list: s.list, saves: s.saves, edit: &s.pageEdit,
		slots: pageSlots{list: &s.list, saves: &s.saves}}
}

// saveView is the open save page, and false while none is open.
func (f *flow) saveView() (pageView, bool) {
	if f.saveScreen == nil {
		return pageView{}, false
	}
	return f.saveScreen.view(), true
}

func (f *flow) openSaveDialog() {
	if !f.savePoint(saveMenuWindow).open() {
		return
	}
	page := firstSavePage()
	if f.pages.saveSet != nil {
		page = *f.pages.saveSet
	}
	s := &savePageState{page: page, pageEdit: pageEdit{request: SaveRequest{
		OnMap: f.menuBack == ScreenMap, Directory: f.saveDialogSeams.Directory, Format: SaveSAV,
	}}}
	f.saveScreen = s
	v := s.view()
	*v.nameField() = page.initialName
	f.msg = ""
	f.setMenuUp(true)
	f.openListPage(ScreenGameMenu, saveListRole())
	s.view().setFocus(saveNameControl)
	s.selectedText = page.selectNameOnOpen
}

// target is the file the name field writes: the file the field names, or the
// listed row whose label it is, or a new slot.
func (v pageView) target() (string, bool) {
	if v.page.fileNames {
		return v.edit.request.Name, true
	}
	label := *v.nameField()
	if e, _, ok := v.selected(); !ok || e.Label != label {
		for i, e := range v.saves {
			if e.Label == label {
				v.list.Select(i + v.first())
				break
			}
		}
	}
	if e, _, ok := v.selected(); ok {
		return e.Name, true
	}
	if v.page.Slot == nil {
		return "", false
	}
	listed := make([]string, len(v.saves))
	for i, e := range v.saves {
		listed[i] = e.Name
	}
	name := v.page.Slot(v.edit.request.Directory, *v.nameField(), listed)
	return name, name != ""
}

// submitSavePage is Ok on the save page: it prepares the write, asks the
// overwrite question where the page has one, and commits.
func (f *flow) submitSavePage() {
	v, ok := f.saveView()
	if !ok || !v.page.fileNames && *v.nameField() == "" {
		return
	}
	e := v.edit
	f.msg = ""
	if e.prepared != nil {
		f.commitSavePage(true)
		return
	}
	name, ok := v.target()
	if !ok || f.saveDialogSeams.Prepare == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	request := e.request
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
	e.prepared = &prepared
	if v.page.overwriteQuestion && len(prepared.Existing) != 0 {
		e.confirmTop = 0
		v.setFocus(saveCancelControl)
		return
	}
	f.commitSavePage(len(prepared.Existing) != 0)
}

func (f *flow) commitSavePage(overwrite bool) {
	v, ok := f.saveView()
	if !ok || v.edit.prepared == nil || v.edit.prepared.Commit == nil {
		return
	}
	e := v.edit
	notice := e.prepared.Notice
	_, err := e.prepared.Commit(overwrite)
	e.prepared = nil
	if err != nil {
		f.msg = err.Error()
		if v.page.keyboardFocus {
			v.setFocus(saveNameControl)
		}
		return
	}
	f.saveDialogSeams.Directory = e.request.Directory
	message := f.words.SaveAcknowledgement
	if message == "" {
		message = AuthoredWords().SaveAcknowledgement
	}
	if notice != "" {
		message += " " + notice
	}
	f.leaveLoad(message, true)
}
