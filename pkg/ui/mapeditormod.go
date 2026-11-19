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
// the word of its caption. Each acts on the window; a mission's own actions
// stand on the Missions page beside it.
var modButtons = []struct{ action, word string }{
	{action: "open", word: "editor.mod.openmod"},
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

// modWindowWidth is the mod window's least width before a remembered size or
// the editor's own width limits it; a tab row that needs more widens it to
// that, the same on every page (tabRowWidth).
const modWindowWidth = 1160

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
				if e.modPage == modPageOverview {
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
	page := e.modPage
	if page != modPageGraph && modPageScope(page) == "" {
		return false
	}
	// The graph page and the pages of a scope of their own hold their own
	// fields only: a field of the mod's own page is focused there.
	e.modPage = modPageFields
	e.dropObject()
	e.loadObject()
	if focus() {
		return true
	}
	e.modPage = page
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
	if e.modPage == modPageDialogues {
		switch name, open := strings.CutPrefix(key, DialogueOpenKey); {
		case key == DialogueNewKey:
			return e.NewDialogue()
		case open:
			return e.OpenDialogue(name)
		}
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
	if own := modPageScope(e.modPage); own != "" && slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key }) {
		scope = own
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
		// The refusal is the editor's message; the window's notes hold only
		// the save state.
		e.obj.note = e.editorHostText(err.Error())
		e.message = err.Error()
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
	e.objectSpec()
	return nil
}

// modSaveState is the one line W's notes hold: a pending confirmation's
// question while its buttons stand, else the save state the host reports,
// else "not saved yet" while the host holds unsaved changes.
func (e *MapEditor) modSaveState() string {
	if e.modConfirm != "" {
		return e.modStatus
	}
	if status, ok := e.modEdits().(interface{ SaveNote() string }); ok {
		if note := status.SaveNote(); note != "" {
			return note
		}
	}
	if host, ok := e.modEdits().(interface{ Dirty() bool }); ok && host.Dirty() {
		return e.plainWord("editor.mod.unsaved")
	}
	return ""
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
	key := modButtons[n].action
	if ask := modPathRows[key]; ask != nil {
		// A typed field edit is applied first, as the other buttons do, so the
		// route asks about unsaved changes with it; a field that asks a path
		// holds a path, not a mod edit, and stays as typed.
		if f, ok := e.modFocusField(); ok && modPathRows[f.Key] == nil {
			if err := e.applyModField(); err != nil {
				return err
			}
		}
		e.askPathQuestion(ask(e))
		return nil
	}
	switch key {
	case "close":
		e.closeObject()
		return nil
	default:
		if err := e.applyModField(); err != nil {
			return err
		}
		return e.modAction(key, "")
	}
}

// modPathRows are the path questions of W's rows by the field, button or row
// button key; the Publish folder field itself asks none: its row's Choose
// button does.
var modPathRows = map[string]func(*MapEditor) pathQuestion{
	"folder":             func(e *MapEditor) pathQuestion { return e.modOpenQuestion("folder") },
	"open":               func(e *MapEditor) pathQuestion { return e.modOpenQuestion("folder") },
	"addmaps":            (*MapEditor).mapsFolderQuestion,
	"folder" + modChoose: func(e *MapEditor) pathQuestion { return e.modFolderQuestion("folder") },
	"output" + modChoose: func(e *MapEditor) pathQuestion { return e.modFolderQuestion("output") },
}

// modChoose ends the key of a folder row's Choose button.
const modChoose = ".pick"

// modChosenOnly reports W's field i chosen, not typed: its row carries a
// Choose that asks the system folder window, and the host has that window.
// A host without one types the field.
func (e *MapEditor) modChosenOnly(i int) bool {
	if !e.modWindow || !e.hasPathDialog() || i < 0 || i >= len(e.obj.fields) {
		return false
	}
	key := e.obj.fields[i].Key
	return slices.ContainsFunc(e.obj.fields, func(f InspectionField) bool { return f.Action && f.Of == key && f.Key == key+modChoose })
}

// modFolderQuestion asks the system folder window for W's folder field:
// from the folder it holds, else the maps folder; the field's route takes
// the answer, and a system without the window asks the typed prompt.
func (e *MapEditor) modFolderQuestion(field string) pathQuestion {
	title, start := "", ""
	for _, f := range e.obj.fields {
		if f.Key == field {
			title = f.Label
			if folder := strings.TrimSpace(f.TextValue); filepath.IsAbs(folder) {
				start = filepath.Clean(folder)
			}
		}
	}
	return pathQuestion{
		Question: pathdialog.Question{Kind: pathdialog.ChooseFolder, Title: title},
		start: func() string {
			if start != "" {
				return start
			}
			return e.mapsStart()
		},
		chosen:   func(path string) { _ = e.modAction(field, path) },
		fallback: func(err error) { e.modOpenTyped(field, err) },
	}
}

// modOpenQuestion asks for a mod.toml or mod-<id>.res, from the folder of an
// absolute path in field, else the maps folder; field's route takes the answer.
func (e *MapEditor) modOpenQuestion(field string) pathQuestion {
	return pathQuestion{
		Question: pathdialog.Question{
			Kind:    pathdialog.OpenFile,
			Title:   e.plainWord("editor.mod.openmod"),
			Filters: []pathdialog.Filter{{Name: e.plainWord("editor.mod.openfilter"), Extensions: []string{"toml", "res"}}},
		},
		start: func() string {
			start := e.mapsStart()
			for _, f := range e.obj.fields {
				if folder := strings.TrimSpace(fieldInput(f)); f.Key == field && filepath.IsAbs(folder) {
					start = filepath.Dir(filepath.Clean(folder))
				}
			}
			return start
		},
		chosen:   func(path string) { _ = e.modAction(field, path) },
		fallback: func(err error) { e.modOpenTyped(field, err) },
	}
}

// modFocusField is W's field that holds the focus; false on a button.
func (e *MapEditor) modFocusField() (InspectionField, bool) {
	if o := &e.obj; o.focus >= 0 && o.focus < len(o.fields) {
		return o.fields[o.focus], true
	}
	return InspectionField{}, false
}

// modPathShown is the path question of W's focused field as shown, which a
// press or Enter asks; nil for a typed path or a field that asks none.
func (e *MapEditor) modPathShown() func(*MapEditor) pathQuestion {
	o := &e.obj
	if !e.modWindow || e.prompt.open() || o.focus < 0 || o.focus >= len(o.fields) || o.focus >= len(o.buf) {
		return nil
	}
	if f := o.fields[o.focus]; !f.ReadOnly && o.buf[o.focus] == fieldInput(f) {
		if ask := modPathRows[f.Key]; ask != nil || len(f.Pick) == 0 {
			return ask
		}
		return func(e *MapEditor) pathQuestion { return e.modPickQuestion(f) }
	}
	return nil
}

// modPickQuestion asks for a host file of one of f's Pick extensions, from
// the maps folder; f's route takes the answer, else the path typed there.
func (e *MapEditor) modPickQuestion(f InspectionField) pathQuestion {
	return pathQuestion{
		Question: pathdialog.Question{
			Kind:    pathdialog.OpenFile,
			Title:   f.Label,
			Filters: []pathdialog.Filter{{Name: "*." + strings.Join(f.Pick, ", *."), Extensions: f.Pick}},
		},
		start:    e.mapsStart,
		chosen:   func(path string) { _ = e.modAction(f.Key, path) },
		fallback: func(err error) { e.modOpenTyped(f.Key, err) },
	}
}

// modOpenTyped is the typed path prompt for W's field where the system has
// no file window: the field's value typed there, the instruction inside the
// prompt; err is why the host showed no window, nil when it has none.
func (e *MapEditor) modOpenTyped(field string, err error) {
	e.FocusModField(field)
	note := e.plainWord("editor.mod.nodialog")
	if err != nil && !errors.Is(err, pathdialog.ErrUnavailable) {
		note = fmt.Sprintf(e.plainWord("editor.mod.typepath"), err)
	}
	e.openPrompt(editorPrompt{kind: promptMapsFolder, scope: field})
	e.message = note
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
	if e.modPage != modPageFields && modPageScope(e.modPage) == "" && !(e.modPage == modPageGraph && e.graphEdit) && (key == "tab" || key == "backtab") {
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
		// A button with Pick asks for its file below, as a file field does.
		if o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Action && len(o.fields[o.focus].Pick) == 0 {
			return true, e.modAction(o.fields[o.focus].Key, "")
		}
		if ask := e.modPathShown(); ask != nil {
			e.askPathQuestion(ask(e))
			return true, nil
		}
		if f, ok := e.modFocusField(); ok && modPathRows[f.Key] != nil {
			return true, e.modAction(f.Key, o.buf[o.focus])
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
	o.rowActs, o.modKeys = map[int][]string{}, map[int][]string{}
	s := popupSpec{Title: e.word("editor.mod.window"), At: e.keptCorner(popObject), Bounds: e.popupBounds(), Columns: 2, NoteInk: editorPaper, LabelWidth: 80}
	var links []int
	if s.Tabs, s.ActiveTab = e.modTabs(); len(s.Tabs) > 0 {
		s.TabSwitch = true
	}
	s.Width = e.popupWide(popObject, min(max(modWindowWidth, tabRowWidth(e.bodyFont(), s.Tabs)), e.window.X-16))
	switch e.modPage {
	case modPageOverview:
		page, _ := e.modOverview()
		e.modOverviewRows(&s, &links, page)
	case modPageGraph:
		page, _ := e.modGraph()
		e.modGraphRows(&s, &links, page)
	default:
		e.objectFieldRows(&s, &links)
		for _, f := range o.fields {
			if f.Grid {
				s.Grid = e.modGrid(f, s.Width)
			}
			if f.Book != nil {
				s.Canvas, s.Drops = e.modBook(f.Book, s.Width)
			}
		}
	}
	for _, f := range o.fields {
		if f.Key != "source" || e.modPage != modPageFields {
			continue
		}
		// Every mission of the mod is a row under the mod's maps; the
		// window's scroll reaches the ones the page has no room to show.
		s.Rows, links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word("editor.mod.maps")}), append(links, 0)
		for n := range f.Choices {
			// A choice's label is already in the font's bytes, as the
			// Mission drop-down draws it.
			row := popupRow{Kind: popupText, Label: f.Choices[n].Label, Tip: f.Choices[n].Label, Buttons: []string{e.word("editor.mod.open")}, Column: 0, Focus: int64(n+1) == f.Value}
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
	// A page whose rows all stand in the first column is one column wide,
	// its label column as wide as its labels need (Fit).
	if e.modPage != modPageGraph && !slices.ContainsFunc(s.Rows, func(r popupRow) bool { return r.Column > 0 }) {
		s.Columns, s.Fit = 1, true
	}
	s.Buttons = nil
	for _, b := range modButtons {
		s.Buttons = append(s.Buttons, e.word(b.word))
	}
	s.Disabled = nil
	if e.modConfirm != "" {
		s.Buttons = []string{e.word("editor.mod.confirm"), e.word("editor.button.cancel")}
		if e.modConfirmLabel != "" {
			s.Buttons[0] = e.modConfirmLabel
		}
	}
	if n := o.focus - len(o.fields); n >= 0 && n < len(s.Buttons) {
		s.ButtonFocus = &n
	}
	// The title's (?) says what the page changes in the game, then its keys;
	// the notes hold only what the modder must see now.
	keys := e.plainWord("editor.mod.keys")
	if e.modPage == modPageGraph {
		keys = e.plainWord("editor.graph.keys")
	}
	s.TitleHelp = e.editorHostText(e.plainWord(modPageWords[e.modPage].helpWord) + "\n" + keys)
	s.Notes = nil
	if note := e.modSaveState(); note != "" {
		s.Notes = editorTextLines(e.bodyFont(), e.editorHostText(note), s.Width-20)
	}
	s.MaxRows = max(3, (e.popupRoom().Dy()-fontHeight(e.font, 15)-len(s.Notes)*(fontHeight(e.bodyFont(), 15)+popupRowExtra)-60)/(fontHeight(e.bodyFont(), 15)+popupRowExtra))
	// The layout pass takes rows away until a wrapped button row fits the
	// editor window, measured from the bounds' top wherever the window stands.
	want, count := e.objectTop, popupRowCount(specWindow(s))
	s.Top = max(0, min(want, count-s.MaxRows))
	s.MaxRows = e.rowsFit(s)
	e.objectTop = max(0, min(want, count-s.MaxRows))
	s.Top = e.objectTop
	// Only the graph page keeps its rows still above a canvas that scrolls
	// alone; every other page is one list of rows that the wheel and the
	// bar move together, so a one-column page shows the rows it scrolled to.
	if e.modPage == modPageGraph {
		s.ColumnTops = []int{0, e.objectTop}
	}
	// modPress hits this spec itself, so it stands where objectSpec does.
	e.keepInRoom(popObject, &s)
	return s, links, true
}

// modGrid is the grid field f of a page as the one grid of picture cells:
// each choice in order, named by its tip, its Variations its hover
// pictures, its rows fitted and scrolled as every grid popup's are.
func (e *MapEditor) modGrid(f InspectionField, width int) *popupGrid {
	g := &popupGrid{Side: gridCellSide, Top: e.modGridTop, LeastRows: 1}
	for _, c := range f.Choices {
		cell := popupGridCell{Icon: f.Icons[c.Value], Tip: c.Label, TipPictures: f.Variations[c.Value], Pressed: c.Value == f.Value,
			Ticked: slices.Contains(f.Ticked, c.Value), Inert: slices.Contains(f.Shut, c.Value)}
		if f.Small && cell.Icon == nil {
			cell.Label = f.Marks[c.Value]
		}
		g.Cells = append(g.Cells, cell)
	}
	return g
}

// ModMapButton is one button of a map's entry: caption, host key and value.
type ModMapButton struct{ Caption, Key, Value string }

// ModMapAction is a map of the open mod that a mission can be made of.
type ModMapAction struct {
	Label   string
	Buttons []ModMapButton
}

// InspectionModMapActions is a host that lists those maps; false when no mod
// project is open.
type InspectionModMapActions interface {
	ModMapActions() ([]ModMapAction, bool)
}

// modMapRows adds the unused maps with their make buttons, and the rows that
// add the open map or a folder of maps, to W's Missions page.
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
		row := popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), e.editorHostText(a.Label), s.Width-20), Column: 0}
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

// modListPages are W's pages of things laid as a list and a chosen thing
// (modListRows); a page's things are its fields' groups. modListEmpty is
// what each says while it holds none.
var (
	modListPages = []int{modPageTerrain, modPageCharacters, modPageItems, modPageTowns, modPageBodies, modPageCompanions, modPageScreens, modPageSettings, modPageSpeakers}
	modListEmpty = []struct {
		page int
		word string
	}{{page: modPageTerrain, word: "editor.terrain.empty"}, {page: modPageCharacters, word: "editor.characters.empty"}, {page: modPageItems, word: "editor.items.empty"}, {page: modPageTowns, word: "editor.towns.empty"}}
)

// modListEmptyWord is the word list page n says while it holds no thing.
func modListEmptyWord(n int) string {
	for _, p := range modListEmpty {
		if p.page == n {
			return p.word
		}
	}
	return ""
}

// modListRows lays a page of things: left, the page's adders (its fields of
// no group) and the list of its things, one row each, none where the
// page's grid lists them; right, the chosen thing's name line with its
// actions as buttons, then its fields.
func (e *MapEditor) modListRows(s *popupSpec, links *[]int, fieldRows func(int, string, bool) int) {
	o := &e.obj
	s.Columns = 2
	column := func(at, c int) {
		for k := at; k < len(s.Rows); k++ {
			s.Rows[k].Column = c
		}
	}
	var groups []string
	grid := false
	for i, f := range o.fields {
		grid = grid || f.Grid
		if f.Group != "" && !slices.Contains(groups, f.Group) {
			groups = append(groups, f.Group)
		}
		if f.Group == "" {
			at := len(s.Rows)
			fieldRows(i, "", false)
			column(at, 0)
		}
	}
	if e.modChosen == nil {
		e.modChosen = map[int]int{}
	}
	chosen := max(0, min(e.modChosen[e.modPage], len(groups)-1))
	e.modListRow = map[int]int{}
	if !grid && len(groups) > 0 {
		s.Rows, *links = append(s.Rows, popupRow{Kind: popupHeading, Label: e.word(modPageWords[e.modPage].tabWord)}), append(*links, 0)
		for k, g := range groups {
			e.modListRow[len(s.Rows)] = k
			label := e.editorHostText(g)
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: label, Tip: label, Focus: k == chosen}), append(*links, 0)
		}
	}
	if len(groups) == 0 {
		at := len(s.Rows)
		if word := modListEmptyWord(e.modPage); word != "" {
			for _, part := range editorTextLines(e.bodyFont(), e.editorHostText(e.plainWord(word)), s.Width/2-40) {
				s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
			}
		}
		column(at, 1)
		return
	}
	g := groups[chosen]
	head := popupRow{Kind: popupText, Label: e.editorHostText(g), Tip: e.editorHostText(g), Column: 1}
	var keys []string
	for _, f := range o.fields {
		if f.Group == g && f.Action {
			head.Buttons, keys = append(head.Buttons, e.editorHostText(f.Label)), append(keys, f.Key)
		}
	}
	o.rowActs[len(s.Rows)], o.modKeys[len(s.Rows)] = make([]string, len(keys)), keys
	s.Rows, *links = append(s.Rows, head), append(*links, 0)
	for i, f := range o.fields {
		if f.Group == g && !f.Action {
			at := len(s.Rows)
			fieldRows(i, "", false)
			column(at, 1)
		}
	}
}

// ModChainKey is the Missions page's read-only field of the chosen mission's
// campaign chain: its line, and a Value of 1 while the mission can move up
// and down the mod's list.
const ModChainKey = "chain"

// modChainRow is the chain field f as one line in the Mission block, cut to
// its width with the whole line in its tip, Move up and Move down on it.
func (e *MapEditor) modChainRow(s *popupSpec, links *[]int, f InspectionField) {
	line := ""
	if len(f.Lines) > 0 {
		line = f.Lines[0].Text
	}
	row := popupRow{Kind: popupText, Label: line, Tip: line}
	var keys []string
	if f.Value == 1 {
		row.Buttons, keys = []string{e.word("editor.mod.up"), e.word("editor.mod.down")}, []string{"up", "down"}
	}
	e.obj.rowActs[len(s.Rows)], e.obj.modKeys[len(s.Rows)] = make([]string, len(keys)), keys
	s.Rows, *links = append(s.Rows, row), append(*links, 0)
}

// modRowButtons puts on row n, the row of W's field key, the Action fields
// that name key as their row (Of): each a button that runs its own key. A
// row that carries a button of its own takes none.
func (e *MapEditor) modRowButtons(row *popupRow, n int, key string) {
	o := &e.obj
	if len(row.Buttons) > 0 || o.rowActs == nil || o.modKeys == nil {
		return
	}
	for _, f := range o.fields {
		if f.Action && f.Of == key {
			row.Buttons, row.ButtonTips = append(row.Buttons, e.editorHostText(f.Label)), append(row.ButtonTips, e.editorHostText(f.Meaning))
			o.rowActs[n], o.modKeys[n] = append(o.rowActs[n], ""), append(o.modKeys[n], f.Key)
		}
	}
}

// mapsFolderQuestion asks for a folder of maps to add; the fallback is the
// typed maps folder line.
func (e *MapEditor) mapsFolderQuestion() pathQuestion {
	return pathQuestion{
		Question: pathdialog.Question{Kind: pathdialog.ChooseFolder, Title: e.plainWord("editor.mod.addmaps.title")},
		start:    e.mapsStart,
		chosen:   func(path string) { _ = e.modAction("addmaps", path) },
		fallback: func(err error) {
			e.openPrompt(editorPrompt{kind: promptMapsFolder})
			if err != nil && !errors.Is(err, pathdialog.ErrUnavailable) {
				e.message = fmt.Sprintf(e.plainWord("editor.mod.addmaps.typepath"), err)
			} else {
				e.message = e.plainWord("editor.mod.addmaps.nodialog")
			}
		},
	}
}

// modOverviewRows lays W's overview page: a heading per section and its
// lines, broken to the window's width.
func (e *MapEditor) modOverviewRows(s *popupSpec, links *[]int, page ModOverviewPage) {
	o := &e.obj
	o.rowBlock, o.usesShown, o.viewOffer, o.cell, o.usesFold = nil, false, false, -1, 0
	o.first, o.rowField = len(s.Rows), map[int]int{}
	s.Columns = 1
	e.modFoldRow, e.modOverviewLinks = e.modFoldRow[:0], map[int]int{}
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
		for i, line := range section.Lines {
			// A line that opens a page is one link; a press on it shows
			// that page.
			if i < len(section.Pages) {
				if page, ok := modPageNamed(section.Pages[i]); ok {
					text := e.editorHostText(line)
					e.modOverviewLinks[len(s.Rows)] = page
					s.Rows, *links = append(s.Rows, popupRow{Kind: popupLink, Label: "> " + text, Tip: text}), append(*links, 0)
					continue
				}
			}
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
	// The press hits the window as drawn, after the room has stood it.
	s, _, _ := e.objectSpec()
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
	if e.carryPress(s, part, n, p) {
		return true
	}
	if part == popupGridPart {
		e.modGridPress(n)
		return true
	}
	if part == popupCanvasBox || part == popupCanvasGround {
		if e.modPage == modPageGraph {
			e.graphPress(part, n, p)
		}
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
		if page, ok := e.modOverviewLinks[n]; ok {
			if k := slices.Index(e.modPages(), page); k >= 0 {
				e.turnModPage(k)
			}
			return true
		}
	}
	if part == popupRowButton {
		if row, k, ok := l.rowButtonAt(p); ok {
			if i, found := e.obj.rowField[row]; found && e.obj.fields[i].Key == "source" {
				_ = e.modAction("remove", "")
				return true
			}
			if actions := e.obj.rowActs[row]; k < len(actions) {
				// A row's button opens its mission, or runs the key its map
				// action names.
				key := "source"
				if keys := e.obj.modKeys[row]; k < len(keys) && keys[k] != "" {
					key = keys[k]
				}
				if ask := modPathRows[key]; ask != nil && actions[k] == "" {
					e.askPathQuestion(ask(e))
					return true
				}
				_ = e.modAction(key, actions[k])
				return true
			}
		}
	}
	if k, ok := e.modListRow[n]; ok && part == popupRowLine && slices.Contains(modListPages, e.modPage) {
		e.modChosen[e.modPage] = k
		return true
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
