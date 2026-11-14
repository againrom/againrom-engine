package ui

import (
	"errors"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/pathdialog"
)

// modButtons is W's button row in its order: the action each button runs and
// the word of its caption.
var modButtons = []struct{ action, word string }{
	{action: "open", word: "editor.mod.openmod"},
	{action: "add", word: "editor.mod.addshort"},
	{action: "remove", word: "editor.mod.remove"},
	{action: "save", word: "editor.mod.save"},
	{action: "publish", word: "editor.mod.publish"},
	{action: "close", word: "editor.mod.close"},
}

// modButtonIndex is the place of action's button in W's row.
func modButtonIndex(action string) int {
	for n, b := range modButtons {
		if b.action == action {
			return n
		}
	}
	return -1
}

func (e *MapEditor) FocusModField(key string) bool {
	if !e.modWindow {
		return false
	}
	e.loadObject()
	if n := modButtonIndex(key); n >= 0 {
		e.obj.focus = len(e.obj.fields) + n
		return true
	}
	focus := func() bool {
		for i, f := range e.obj.fields {
			if f.Key == key && !f.ReadOnly {
				if e.modPage != modPageGraph {
					e.modPage = modPageFields
				}
				e.obj.detail = f.Parent
				e.objectSpec()
				e.focusObjectField(i)
				return true
			}
		}
		return false
	}
	if focus() {
		return true
	}
	if e.modPage != modPageGraph {
		return false
	}
	// The graph page holds the fields of its mission only: a field of the
	// mod's own page is focused there.
	e.modPage = modPageFields
	e.dropObject()
	e.loadObject()
	if focus() {
		return true
	}
	e.modPage = modPageGraph
	e.dropObject()
	e.loadObject()
	return false
}

func (e *MapEditor) ModWindowOpen() bool { return e.modWindow }

// InspectionModMaps is a host that lists the maps of its open mod: the
// package field "map", one choice per map with its role, the map these
// edits hold chosen. Setting that field to a choice opens its map.
type InspectionModMaps interface {
	ModMaps() InspectionField
}

// modMapList is the open mod's list of maps as the host gives it. W's rows
// and the Open window's group of the mod's maps both read it.
func (e *MapEditor) modMapList() (InspectionField, bool) {
	host, ok := e.modEdits().(InspectionModMaps)
	if !ok {
		return InspectionField{}, false
	}
	f := host.ModMaps()
	return f, len(f.Choices) > 0
}

// OpenModMap opens row n, from 1, of the mod's list of maps as a tab: the
// route of the list's Open button and of the Open window's group.
func (e *MapEditor) OpenModMap(n int) error {
	_, err := e.commandSetMapField(mapField{Scope: "package", Key: "map", Value: strconv.Itoa(n)})
	return err
}

func (e *MapEditor) applyModField() error {
	o := &e.obj
	if o.focus < 0 || o.focus >= len(o.fields) {
		return nil
	}
	f := o.fields[o.focus]
	if f.Action {
		return nil
	}
	if o.buf[o.focus] == fieldInput(f) {
		return nil
	}
	return e.modAction(f.Key, o.buf[o.focus])
}

func (e *MapEditor) modAction(key, value string) error {
	if strings.HasPrefix(key, "graph.") {
		return e.graphAction(key)
	}
	if _, ok := e.workspace(); !ok {
		return nil
	}
	focus, detail := e.obj.focus, e.obj.detail
	button := -1
	fieldKey := ""
	if focus >= len(e.obj.fields) {
		button = focus - len(e.obj.fields)
	} else if focus >= 0 {
		fieldKey = e.obj.fields[focus].Key
	}
	scope := "package"
	if e.modPage == modPageGraph && e.graphEdit && (key == "AutoGetMission" || key == "insert" || slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key })) {
		scope = e.modFieldScope()
	}
	_, err := e.commandSetMapField(mapField{Scope: scope, Key: key, Value: value})
	if err != nil {
		var leave *WorkspaceLeave
		if errors.As(err, &leave) {
			e.askLeave(leaveStep{open: true, source: leave.Folder, workspace: true})
			return nil
		}
		var ask *WorkspaceConfirmation
		if errors.As(err, &ask) {
			e.modConfirm = ask.Action
			e.modConfirmLabel = e.editorHostText(ask.Confirm)
			e.modStatus = ask.Message
			e.obj.focus = len(e.obj.fields)
			return nil
		}
		e.obj.note = err.Error()
		e.modStatus = err.Error()
		return err
	}
	e.modConfirm = ""
	e.modStatus = ""
	e.message = ""
	e.modWindow = true
	e.objectHidden = false
	e.dropObject()
	e.loadObject()
	e.obj.detail = detail
	e.obj.focus = 0
	if button >= 0 {
		e.obj.focus = len(e.obj.fields) + min(button, e.modButtonCount()-1)
	} else {
		for i, f := range e.obj.fields {
			if f.Key == fieldKey && !f.ReadOnly {
				e.obj.focus = i
				break
			}
		}
	}
	if key == "folder" || key == "folder-open" {
		e.graphMission, e.graphLink, e.graphPicked = 0, false, false
		e.openModPage()
	}
	if status, ok := e.modEdits().(interface{ SaveNote() string }); ok {
		e.modStatus = status.SaveNote()
	}
	e.objectSpec()
	return nil
}

func (e *MapEditor) modButton(n int) error {
	if e.modConfirm != "" {
		if n == 0 {
			return e.modAction(e.modConfirm, "confirmed")
		} else {
			e.modConfirm = ""
			e.modStatus = ""
			e.obj.focus = 0
		}
		return nil
	}
	if n < 0 || n >= len(modButtons) {
		return nil
	}
	if e.modPage == modPageGraph && (modButtons[n].action == "add" || modButtons[n].action == "remove") {
		return nil
	}
	switch key := modButtons[n].action; key {
	case "close":
		e.closeObject()
		return nil
	case "open":
		// A typed field edit is applied first, as the other buttons do, so the
		// route asks about unsaved changes with it; the folder field holds a
		// path, not a mod edit, and stays as typed.
		if o := e.obj; o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Key != "folder" {
			if err := e.applyModField(); err != nil {
				return err
			}
		}
		e.openModChoice()
		return nil
	default:
		if err := e.applyModField(); err != nil {
			return err
		}
		return e.modAction(key, "")
	}
}

// openModChoice asks the host's path window for a mod to open: its mod.toml
// or a packed mod-<id>.res. The answer goes the folder field's own route,
// which asks about unsaved changes before it opens the mod. With no window
// the folder field takes the path.
func (e *MapEditor) openModChoice() {
	if !e.hasPathDialog() {
		e.modOpenTyped(nil)
		return
	}
	r := pathdialog.Request{Kind: pathdialog.OpenFile, Title: e.plainWord("editor.mod.openmod"), StartDir: e.pathDialog.baseDir,
		Filters: []pathdialog.Filter{{Name: e.plainWord("editor.mod.openfilter"), Extensions: []string{"toml", "res"}}}}
	if e.pathDialog.mapsFolder != nil {
		if dir := e.pathDialog.mapsFolder(); dir != "" {
			r.StartDir = dir
		}
	}
	for _, f := range e.obj.fields {
		if folder := strings.TrimSpace(fieldInput(f)); f.Key == "folder" && filepath.IsAbs(folder) {
			r.StartDir = filepath.Dir(filepath.Clean(folder))
		}
	}
	e.askChoice(pathChoice{
		request:   r,
		chosen:    func(path string) { _ = e.modAction("folder", path) },
		cancelled: func() {},
		failed:    e.modOpenTyped,
	})
}

// modFolderShown reports that W's folder field holds the focus as shown: a
// press on it or Enter opens the path window; a typed folder is edited.
func (e *MapEditor) modFolderShown() bool {
	o := &e.obj
	if !e.modWindow || e.prompt.open() || o.focus < 0 || o.focus >= len(o.fields) || o.focus >= len(o.buf) {
		return false
	}
	f := o.fields[o.focus]
	return f.Key == "folder" && !f.ReadOnly && o.buf[o.focus] == fieldInput(f)
}

// modOpenTyped focuses W's folder field and says to type the mod's path
// there; err is why the host showed no window, nil when it has none.
func (e *MapEditor) modOpenTyped(err error) {
	e.FocusModField("folder")
	if err == nil || errors.Is(err, pathdialog.ErrUnavailable) {
		e.modStatus = e.plainWord("editor.mod.nodialog")
		return
	}
	e.modStatus = fmt.Sprintf(e.plainWord("editor.mod.typepath"), err)
}

func (e *MapEditor) modKey(key string) (bool, error) {
	o := &e.obj
	if e.modConfirm != "" {
		switch key {
		case "package":
			if e.turnModPage(-1) {
				return true, nil
			}
		case "tab", "backtab":
			o.focus = len(o.fields) + (o.focus-len(o.fields)+1)%2
		case "enter":
			return true, e.modButton(max(0, min(1, o.focus-len(o.fields))))
		case "escape":
			e.modConfirm, e.modStatus = "", ""
			o.focus = 0
		}
		return true, nil
	}
	if e.modPage == modPageGraph && e.graphKey(key) {
		return true, nil
	}
	if e.modPage == modPageOverview && key == "enter" {
		if at := o.focus - len(o.fields) - e.modButtonCount(); at >= 0 && at < len(e.modFoldRow) {
			e.toggleModFold(at)
			return true, nil
		}
	}
	if e.modPage != modPageFields && !(e.modPage == modPageGraph && e.graphEdit) && (key == "tab" || key == "backtab") {
		n, step := e.modButtonCount(), 1
		if e.modPage == modPageOverview {
			e.modSpec()
			n += len(e.modFoldRow)
		}
		if key == "backtab" {
			step = n - 1
		}
		o.focus = len(o.fields) + (max(0, o.focus-len(o.fields))+step)%n
		return true, nil
	}
	switch key {
	case "nextmap", "prevmap":
		return true, e.keyMapStep(key)
	case "close":
		e.closeObject()
		return true, e.keyClose(key)
	case "package":
		// With more pages than one, the window's key turns its page.
		if e.turnModPage(-1) {
			return true, nil
		}
		return true, e.keyWorkspace(key)
	case "saveas":
		return true, e.modButton(modButtonIndex("save"))
	case "testplay":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.closeObject()
		return true, e.keyTestPlay(key)
	case "undo", "redo":
		err := e.keyHistory(key)
		e.modWindow = true
		e.objectHidden = false
		e.obj.doc = nil
		e.loadObject()
		e.objectSpec()
		return true, err
	case "tab":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.objectSpec()
		if o.focus >= len(o.fields) {
			o.focus++
			if o.focus >= len(o.fields)+e.modButtonCount() {
				o.focus = e.nextObjectField(-1)
			}
		} else {
			next := e.nextObjectField(o.focus)
			if next <= o.focus {
				o.focus = len(o.fields)
			} else {
				e.focusObjectField(next)
			}
		}
		return true, nil
	case "backtab":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.objectSpec()
		var visible []int
		for i := range o.fields {
			if e.fieldRow(i) >= 0 {
				visible = append(visible, i)
			}
		}
		for i := 0; i < e.modButtonCount(); i++ {
			visible = append(visible, len(o.fields)+i)
		}
		at := 0
		for i, n := range visible {
			if n == o.focus {
				at = i
				break
			}
		}
		o.focus = visible[(at+len(visible)-1)%len(visible)]
		if o.focus < len(o.fields) {
			e.focusObjectField(o.focus)
		}
		return true, nil
	case "enter":
		if o.focus >= len(o.fields) {
			return true, e.modButton(o.focus - len(o.fields))
		}
		if e.dropdownOpen() && o.choice != nil {
			e.chooseObjectChoice(o.choice.Selection())
			return true, nil
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Action {
			return true, e.modAction(o.fields[o.focus].Key, "")
		}
		if e.modFolderShown() {
			e.openModChoice()
			return true, nil
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Key == "folder" {
			return true, e.modAction("folder", o.buf[o.focus])
		}
		if o.focus >= 0 && o.focus < len(o.fields) && len(o.fields[o.focus].Choices) > 0 && o.buf[o.focus] == fieldInput(o.fields[o.focus]) {
			e.objectRowPress(popupRowControl, e.fieldRow(o.focus))
			return true, nil
		}
		return true, e.applyModField()
	case "escape":
		if e.modConfirm != "" {
			e.modConfirm = ""
			e.modStatus = ""
			o.focus = 0
			return true, nil
		}
		if e.dropdownOpen() {
			o.open = -1
			return true, nil
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.buf[o.focus] != fieldInput(o.fields[o.focus]) {
			o.buf[o.focus] = fieldInput(o.fields[o.focus])
			return true, nil
		}
		e.closeObject()
		return true, nil
	case "cardup", "carddown":
		action := "up"
		if key == "carddown" {
			action = "down"
		}
		if o.focus >= 0 && o.focus < len(o.fields) {
			f := o.fields[o.focus]
			if strings.HasPrefix(f.Key, "party.") {
				parts := strings.Split(f.Key, ".")
				action = parts[0] + "." + parts[1] + "." + action
			}
		}
		return true, e.modAction(action, "")
	}
	// A button of the window holds the keyboard; so does the window with no
	// map under it. With no field or button focused the key is the map's.
	if o.focus >= len(o.fields) || o.focus < 0 && e.doc == nil {
		return true, nil
	}
	return false, nil
}

// modYieldKeys gives the keyboard to the map at a press on it: the mod
// window's focus and open drop-down are put down. A value typed and not yet
// entered keeps the focus, so nothing typed is lost or read as a map key.
func (e *MapEditor) modYieldKeys() {
	o := &e.obj
	if !e.modWindow || e.prompt.open() || e.modConfirm != "" || o.doc != e.doc || o.record != modObjectRecord {
		return
	}
	if o.focus >= 0 && o.focus < len(o.fields) && o.focus < len(o.buf) && o.buf[o.focus] != fieldInput(o.fields[o.focus]) {
		return
	}
	o.focus, o.open = -1, -1
}

func (e *MapEditor) keyModBacktab(string) error { return nil }

func (e *MapEditor) modNativeKeys(in editorNativeInput) (bool, error) {
	if !e.modWindow || e.prompt.open() {
		return false, nil
	}
	if in.justPressed(ebiten.KeyTab) {
		key := "tab"
		if in.pressed(ebiten.KeyShift) {
			key = "backtab"
		}
		return true, e.Key(key)
	}
	if in.pressed(ebiten.KeyAlt) {
		if in.justPressed(ebiten.KeyArrowUp) {
			return true, e.Key("cardup")
		}
		if in.justPressed(ebiten.KeyArrowDown) {
			return true, e.Key("carddown")
		}
	}
	if !in.pressed(ebiten.KeyControl) {
		return false, nil
	}
	if in.justPressed(ebiten.KeyS) {
		return true, e.Key("saveas")
	}
	// Ctrl+O is W's Open mod; the File menu's Open still opens a map.
	if in.justPressed(ebiten.KeyO) {
		if e.pathDialog.pending == nil && e.modConfirm == "" {
			return true, e.modButton(modButtonIndex("open"))
		}
		return true, nil
	}
	pending := false
	o := &e.obj
	if o.focus >= 0 && o.focus < len(o.fields) {
		pending = o.buf[o.focus] != fieldInput(o.fields[o.focus])
	}
	if !pending {
		if in.justPressed(ebiten.KeyZ) {
			key := "undo"
			if in.pressed(ebiten.KeyShift) {
				key = "redo"
			}
			return true, e.Key(key)
		}
		if in.justPressed(ebiten.KeyY) {
			return true, e.Key("redo")
		}
	}
	return false, nil
}

func (e *MapEditor) modSpec() (popupSpec, []int, bool) {
	e.loadObject()
	o := &e.obj
	o.rowActs, o.modKeys = map[int][]string{}, nil
	s := popupSpec{Title: e.word("editor.mod.window"), Width: e.popupWide(popObject, min(620, e.window.X-16)), At: e.personCorner(homeMod), Bounds: e.popupBounds(), Columns: 2, NoteInk: editorPaper, LabelWidth: 80}
	var links []int
	if s.Tabs, s.ActiveTab = e.modTabs(); len(s.Tabs) > 0 {
		s.TabSwitch = true
	}
	switch e.modPage {
	case modPageOverview:
		page, _ := e.modOverview()
		e.modOverviewRows(&s, &links, page)
	case modPageGraph:
		page, _ := e.modGraph()
		e.modGraphRows(&s, &links, page)
	default:
		e.objectFieldRows(&s, &links)
	}
	for _, f := range o.fields {
		if f.Key != "source" || e.modPage != modPageFields {
			continue
		}
		// Every mission of the mod is a row; the window's scroll reaches
		// the ones the page has no room to show.
		for n := range f.Choices {
			// A choice's label is already in the font's bytes, as the
			// Mission drop-down draws it.
			row := popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), f.Choices[n].Label, s.Width/2-62), Buttons: []string{e.word("editor.mod.open")}, Column: 0, Focus: int64(n+1) == f.Value}
			index := len(s.Rows)
			s.Rows = append(s.Rows, row)
			links = append(links, 0)
			if o.rowActs == nil {
				o.rowActs = map[int][]string{}
			}
			o.rowActs[index] = []string{strconv.FormatInt(f.Choices[n].Value, 10)}
		}
	}
	if e.modPage == modPageFields {
		e.modMapRows(&s, &links)
	}
	s.Buttons = nil
	for _, b := range modButtons {
		s.Buttons = append(s.Buttons, e.word(b.word))
	}
	s.Disabled = nil
	if e.modPage == modPageGraph {
		s.Disabled = make([]bool, len(modButtons))
		s.Disabled[modButtonIndex("add")], s.Disabled[modButtonIndex("remove")] = true, true
	}
	if e.modConfirm != "" {
		s.Buttons = []string{e.word("editor.mod.confirm"), e.word("editor.button.cancel")}
		if e.modConfirmLabel != "" {
			s.Buttons[0] = e.modConfirmLabel
		}
	}
	if n := o.focus - len(o.fields); n >= 0 && n < len(s.Buttons) {
		s.ButtonFocus = &n
	}
	status := e.modStatus
	if status == "" && e.modPage == modPageGraph {
		status = e.plainWord("editor.graph.keys")
	}
	if status == "" {
		status = e.plainWord("editor.mod.keys")
	}
	s.Notes = editorTextLines(e.bodyFont(), e.editorHostText(status), s.Width-20)
	if host, ok := e.modEdits().(interface{ ModSummary() string }); ok {
		s.Notes = append([]string{editorClip(e.bodyFont(), e.editorHostText(host.ModSummary()), s.Width-20)}, s.Notes...)
	}
	if host, ok := e.modEdits().(interface{ ModOutput() string }); ok {
		if output := host.ModOutput(); output != "" {
			s.Notes = append(s.Notes, editorTextLines(e.bodyFont(), e.editorHostText(output), s.Width-20)...)
		}
	}
	s.MaxRows = max(3, (e.popupRoom().Dy()-fontHeight(e.font, 15)-len(s.Notes)*(fontHeight(e.bodyFont(), 15)+popupRowExtra)-60)/(fontHeight(e.bodyFont(), 15)+popupRowExtra))
	// Rows give way until a wrapped button row fits the editor window.
	for s.MaxRows > 3 && layoutPopup(s, e.font, e.bodyFont()).Rect.Dy() > s.Bounds.Dy() {
		s.MaxRows--
	}
	e.objectTop = max(0, min(e.objectTop, max(0, popupRowCount(s)-s.MaxRows)))
	s.Top = e.objectTop
	// Only the graph page keeps its rows still above a canvas that scrolls
	// alone; every other page is one list of rows that the wheel and the
	// bar move together, so a one-column page shows the rows it scrolled to.
	if e.modPage == modPageGraph {
		s.ColumnTops = []int{0, e.objectTop}
	}
	return s, links, true
}

// ModMapButton is one button of a map's entry on W's Missions page: its
// caption, the field key the host takes and the value it is given.
type ModMapButton struct{ Caption, Key, Value string }

// ModMapAction is a map of the open mod that a mission can be made of, and
// the buttons that make it.
type ModMapAction struct {
	Label   string
	Buttons []ModMapButton
}

// InspectionModMapActions is a host that lists the maps of its open mod that
// a mission can be made of. It answers false when no mod project is open.
type InspectionModMapActions interface {
	ModMapActions() ([]ModMapAction, bool)
}

// modMapRows adds to W's Missions page the maps no mission uses, each with
// the buttons that make it a mission, and the rows that add the open map or a
// folder of maps to the mod.
func (e *MapEditor) modMapRows(s *popupSpec, links *[]int) {
	host, ok := e.modEdits().(InspectionModMapActions)
	if !ok {
		return
	}
	actions, open := host.ModMapActions()
	if !open {
		return
	}
	o := &e.obj
	o.modKeys = map[int][]string{}
	add := func(row popupRow, keys, values []string) {
		index := len(s.Rows)
		s.Rows = append(s.Rows, row)
		*links = append(*links, 0)
		o.rowActs[index], o.modKeys[index] = values, keys
	}
	if len(actions) > 0 {
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word("editor.mod.mapactions")}), append(*links, 0)
	}
	for _, a := range actions {
		row := popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), e.editorHostText(a.Label), s.Width/2-62), Column: 0}
		var keys, values []string
		for _, b := range a.Buttons {
			row.Buttons = append(row.Buttons, e.editorHostText(b.Caption))
			keys, values = append(keys, b.Key), append(values, b.Value)
		}
		add(row, keys, values)
	}
	add(popupRow{Kind: popupText, Label: e.word("editor.mod.addmap.label"), Buttons: []string{e.word("editor.mod.addmap")}, Column: 0}, []string{"addmap"}, []string{""})
	add(popupRow{Kind: popupText, Label: e.word("editor.mod.addmaps.label"), Buttons: []string{e.word("editor.mod.addmaps")}, Column: 0}, []string{"addmaps"}, []string{""})
}

// askMapsFolder asks the host's folder window for a folder of maps to add to
// the open mod. Where the host shows none, a typed path is asked for.
func (e *MapEditor) askMapsFolder() {
	typed := func(err error) {
		e.openPrompt(editorPrompt{kind: promptMapsFolder})
		if err != nil && !errors.Is(err, pathdialog.ErrUnavailable) {
			e.message = fmt.Sprintf(e.plainWord("editor.mod.addmaps.typepath"), err)
		} else {
			e.message = e.plainWord("editor.mod.addmaps.nodialog")
		}
	}
	if !e.hasPathDialog() {
		typed(nil)
		return
	}
	r := pathdialog.Request{Kind: pathdialog.ChooseFolder, Title: e.plainWord("editor.mod.addmaps.title"), StartDir: e.pathDialog.baseDir}
	if e.pathDialog.mapsFolder != nil {
		if dir := e.pathDialog.mapsFolder(); dir != "" {
			r.StartDir = dir
		}
	}
	e.askChoice(pathChoice{
		request:   r,
		chosen:    func(path string) { _ = e.modAction("addmaps", path) },
		cancelled: func() {},
		failed:    typed,
	})
}

// modOverviewRows lays W's overview page: a heading per section and its
// lines, broken to the window's width.
func (e *MapEditor) modOverviewRows(s *popupSpec, links *[]int, page ModOverviewPage) {
	o := &e.obj
	o.rowBlock, o.usesShown, o.viewOffer, o.cell = nil, false, false, -1
	o.first, o.rowField = len(s.Rows), map[int]int{}
	s.Columns = 1
	e.modFoldRow = e.modFoldRow[:0]
	at := o.focus - len(o.fields) - e.modButtonCount()
	folds := 0
	for _, section := range page.Sections {
		head := popupRow{Kind: popupHeading, Label: e.editorHostText(section.Heading)}
		if section.Fold {
			head = blockRow(head.Label, e.modUnfolded[folds])
			head.Focus = folds == at
			e.modFoldRow = append(e.modFoldRow, len(s.Rows))
			folds++
		}
		s.Rows, *links = append(s.Rows, head), append(*links, 0)
		if section.Fold && !e.modUnfolded[folds-1] {
			continue
		}
		for _, line := range section.Lines {
			for _, part := range editorTextLines(e.bodyFont(), e.editorHostText(line), s.Width-40) {
				s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
			}
		}
	}
}

func (e *MapEditor) modPress(p image.Point) bool {
	if !e.modWindow {
		return false
	}
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	part, n := l.hit(p)
	if part == popupClose {
		e.closeObject()
		return true
	}
	if part == popupButton {
		e.modButton(n)
		return true
	}
	if part == popupTab {
		e.turnModPage(n)
		return true
	}
	if e.modConfirm != "" {
		return true
	}
	if part == popupCanvasBox || part == popupCanvasGround {
		e.graphPress(part, n, p)
		return true
	}
	if part == popupRowButton || part == popupRowControl || part == popupRowLine {
		if err := e.applyModField(); err != nil {
			return true
		}
	}
	if part == popupRowLine && e.modPage == modPageOverview {
		if k := slices.Index(e.modFoldRow, n); k >= 0 {
			e.toggleModFold(k)
			return true
		}
	}
	if part == popupRowButton {
		if row, k, ok := l.rowButtonAt(p); ok {
			if i, found := e.obj.rowField[row]; found && e.obj.fields[i].Key == "source" {
				action := "up"
				if k == 1 {
					action = "down"
				}
				_ = e.modAction(action, "")
				return true
			}
			if actions := e.obj.rowActs[row]; k < len(actions) {
				// A row's button opens its mission, or runs the key its map
				// action names.
				key := "source"
				if keys := e.obj.modKeys[row]; k < len(keys) && keys[k] != "" {
					key = keys[k]
				}
				if key == "addmaps" && actions[k] == "" {
					e.askMapsFolder()
					return true
				}
				_ = e.modAction(key, actions[k])
				return true
			}
		}
	}
	if part == popupRowButton || part == popupRowControl || part == popupRowLine {
		e.objectRowPress(part, n)
		return true
	}
	return false
}

// toggleModFold unfolds the overview's folded section k, or folds it again.
func (e *MapEditor) toggleModFold(k int) {
	if e.modUnfolded == nil {
		e.modUnfolded = map[int]bool{}
	}
	e.modUnfolded[k] = !e.modUnfolded[k]
}

func (e *MapEditor) modButtonCount() int {
	if e.modConfirm != "" {
		return 2
	}
	return len(modButtons)
}
