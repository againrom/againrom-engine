package ui

import (
	"againrom/pkg/pathdialog"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The map editor window's keys. editorKeyRows is the one table: each row is a
// key, what it does and where the rail's help lists it. Key dispatches from
// it and the help lines are built from it, so a key the help names is a key
// the window acts on.

// helpScreen is one of the help the rail shows, chosen by the window's mode.
type helpScreen uint8

const (
	screenKeys    helpScreen = iota // no pick, panel or brush: the keys
	screenClone                     // a clone is armed
	screenDrop                      // the drop cell is armed
	screenGoal                      // a script arm's target is armed
	screenGoals                     // the script panel is open
	screenTerrain                   // the terrain brush
	screenHeight                    // the height brush
	screenObjects                   // the objects brush
	screenPlayers                   // the roster list
	screenPrompt
	screenMove
	screenAdd     // the Add window
	screenAddCell // the Add window's cell is armed
	screenCount
)

// helpEntry is a line of a help screen.
type helpEntry struct {
	screen helpScreen
	line   string
}

// promptLive says when a key acts while a prompt is open.
type promptLive uint8

const (
	promptDeaf   promptLive = iota // never: the prompt takes only its own keys
	promptHears                    // always
	promptFields                   // while the prompt edits a record field
)

// keyRow is one key. A row with no key is an instruction for the pointer that
// only the help carries. A row with no help lines gives the reason it has none
// in unlisted.
type keyRow struct {
	key      string
	label    string // the key as the help writes it
	live     promptLive
	act      func(e *MapEditor, key string) error
	help     []helpEntry
	unlisted string
}

const (
	unlistedSaveLine = "the bottom line names Ctrl+Z, Ctrl+Y and Ctrl+S"
	unlistedPrompt   = "the prompt names it"
	unlistedRail     = "the rail has a button for it"
	unlistedWindow   = "a window control, not a map edit"
)

var (
	editorKeyRows   []keyRow
	editorKeyByName map[string]*keyRow
	editorHelp      [screenCount][]string
)

func init() {
	keys := func(line string) []helpEntry { return []helpEntry{{screenKeys, line}} }
	line0, line1, line2, line3 := "Drag: move / C: clone / Del", "E: fields / S: sack / R: report", "D: drop / G: arms / T, H, B: brush", "N: new map / U: unit"
	line4, line5 := "P: players / F5: test play", "M: map / X: pages / W: mod"
	promptLine := "Enter: set / Tab: next / Esc: back"
	addLine, addChoose := "Enter: next / Esc: cancel", "Up, Down: choose"
	arms := "Up, Down: select / G: close"
	players, playerFields := "P: records / A: add / Del: delete", "Up, Down: select / E: fields"
	editorKeyRows = []keyRow{
		{key: "enter", label: "Enter", live: promptHears, act: (*MapEditor).keyEnter, help: []helpEntry{{screenGoals, "Enter: pick a target"}, {screenPrompt, promptLine}, {screenAdd, addLine}}},
		{key: "clone", label: "C", act: (*MapEditor).recordKey, help: keys(line0)},
		{key: "delete", label: "Del", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, line0}, {screenPlayers, players}}},
		{key: "field", label: "E", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, line1}, {screenPlayers, playerFields}}},
		{key: "sack", label: "S", act: (*MapEditor).recordKey, help: keys(line1)},
		{key: "report", label: "R", act: (*MapEditor).keyReport, help: keys(line1)},
		{key: "drop", label: "D", act: (*MapEditor).recordKey, help: keys(line2)},
		{key: "goals", label: "G", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, line2}, {screenGoals, arms}}},
		{key: "terrain", label: "T", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, line2}, {screenTerrain, "T: off / Drag: paint / [ ]: size"}}},
		{key: "height", label: "H", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, line2}, {screenHeight, "H: off / Drag: shape / [ ]: size"}}},
		{key: "objects", label: "B", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, line2}, {screenObjects, "B: off / Drag: place / [ ]: size"}}},
		{key: "radiusdown", label: "[", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenTerrain, "T: off / Drag: paint / [ ]: size"}, {screenHeight, "H: off / Drag: shape / [ ]: size"}, {screenObjects, "B: off / Drag: place / [ ]: size"}}},
		{key: "radiusup", label: "]", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenTerrain, "T: off / Drag: paint / [ ]: size"}, {screenHeight, "H: off / Drag: shape / [ ]: size"}, {screenObjects, "B: off / Drag: place / [ ]: size"}}},
		{key: "new", label: "N", act: (*MapEditor).keyNew, help: keys(line3)},
		{key: "unit", label: "U", act: (*MapEditor).keyUnit, help: keys(line3)},
		{key: "structure", label: "J", act: (*MapEditor).keyRecords, help: keys("J: structure / K: stock / I: recipe")},
		{key: "stock", label: "K", act: (*MapEditor).keyRecords, help: keys("J: structure / K: stock / I: recipe")},
		{key: "recipe", label: "I", act: (*MapEditor).keyRecords, help: keys("J: structure / K: stock / I: recipe")},
		{key: "players", label: "P", act: (*MapEditor).keyPlayers, help: []helpEntry{{screenKeys, line4}, {screenPlayers, players}}},
		{key: "player", label: "A", act: (*MapEditor).keyAdd, help: []helpEntry{{screenPlayers, players}}},
		{key: "testplay", label: "F5", act: (*MapEditor).keyTestPlay, help: keys(line4)},
		{key: "scripts", label: "F2", act: (*MapEditor).keyScripts, help: keys("F1: all keys / F2: scripts")},
		{key: "metadata", label: "M", act: (*MapEditor).keyWorkspace, help: keys(line5)},
		{key: "pages", label: "X", act: (*MapEditor).keyWorkspace, help: keys(line5)},
		{key: "package", label: "W", act: (*MapEditor).keyWorkspace, help: keys(line5)},
		{key: "help", label: "F1", act: (*MapEditor).keyHelp, help: keys("F1: all keys / F2: scripts")},
		{key: "drawarea", label: "L", act: (*MapEditor).keyDrawArea, help: keys("L: draw a trigger area")},
		{label: "click", help: []helpEntry{{screenClone, "Clone: click a cell"}, {screenDrop, "Drop cell: click a cell"}, {screenGoal, "Target: click a unit or"}, {screenMove, "Move: click a cell"}, {screenAddCell, "Cell: click a cell"}}},
		{key: "escape", label: "Esc", live: promptHears, act: (*MapEditor).keyEscape, help: []helpEntry{{screenClone, "Esc: cancel"}, {screenDrop, "Esc: cancel"}, {screenGoal, "structure / Esc: cancel"}, {screenPlayers, "Esc: map records"}, {screenPrompt, promptLine}, {screenMove, "Esc: cancel"}, {screenAdd, addLine}, {screenAddCell, "Esc: back"}}},
		{key: "up", label: "Up", live: promptFields, act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, arms}, {screenPlayers, playerFields}, {screenAdd, addChoose}}},
		{key: "down", label: "Down", live: promptFields, act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, arms}, {screenPlayers, playerFields}, {screenAdd, addChoose}}},
		{key: "close", live: promptHears, act: (*MapEditor).keyClose, unlisted: unlistedWindow},
		{key: "open", act: (*MapEditor).keyOpen, unlisted: unlistedRail},
		{key: "catalog", act: (*MapEditor).keyCatalog, unlisted: unlistedRail},
		{key: "saveas", act: (*MapEditor).keySaveAs, unlisted: unlistedSaveLine},
		{key: "undo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "redo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "tab", label: "Tab", live: promptFields, act: (*MapEditor).recordKey, help: []helpEntry{{screenPrompt, promptLine}}},
		{key: "newline", live: promptHears, act: (*MapEditor).keyNewline, unlisted: unlistedPrompt},
		{key: "clear", live: promptHears, act: (*MapEditor).keyClear, unlisted: unlistedPrompt},
		{key: "backspace", live: promptHears, act: (*MapEditor).keyBackspace, unlisted: unlistedPrompt},
		{key: "fit", act: (*MapEditor).keyFit, unlisted: unlistedRail},
		{key: "triggers", act: (*MapEditor).keyScripts, unlisted: "the Scripts entry opens it"},
		{key: "pageup", act: (*MapEditor).keyPage, unlisted: "the list line names PgUp and PgDn"},
		{key: "pagedown", act: (*MapEditor).keyPage, unlisted: "the list line names PgUp and PgDn"},
		{key: "home", act: (*MapEditor).keyPage, unlisted: "list travel"},
		{key: "end", act: (*MapEditor).keyPage, unlisted: "list travel"},
	}
	editorKeyByName = map[string]*keyRow{}
	for i := range editorKeyRows {
		r := &editorKeyRows[i]
		if r.key != "" {
			editorKeyByName[r.key] = r
		}
		for _, h := range r.help {
			if !containsString(editorHelp[h.screen], h.line) {
				editorHelp[h.screen] = append(editorHelp[h.screen], h.line)
			}
		}
	}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// EditorHelpLines is every help line the editor window shows.
func EditorHelpLines() []string {
	var out []string
	for _, lines := range editorHelp {
		out = append(out, lines...)
	}
	return out
}

// brushScreens is the help screen of each brush mode.
var brushScreens = map[brushMode]helpScreen{brushTerrain: screenTerrain, brushHeight: screenHeight, brushObjects: screenObjects}

// pickScreens is the help screen of each armed pick.
var pickScreens = map[pickKind]helpScreen{pickClone: screenClone, pickDrop: screenDrop, pickGoal: screenGoal, pickMove: screenMove, pickAddCell: screenAddCell}

// helpScreen is the help for the window's mode: an active brush, the armed
// pick, the script panel, or the keys.
func (e *MapEditor) helpScreen() helpScreen {
	if e.addPicking() {
		return screenAddCell
	}
	if e.prompt.kind == promptAdd {
		return screenAdd
	}
	if e.prompt.open() {
		return screenPrompt
	}
	if e.brushActive() {
		return brushScreens[e.brush.mode]
	}
	if s, ok := pickScreens[e.pick]; ok {
		return s
	}
	if len(e.report.goals) > 0 {
		return screenGoals
	}
	if e.listView() && e.filter == editorPlayersFilter {
		return screenPlayers
	}
	return screenKeys
}

func (e *MapEditor) helpLines() []string {
	if e.helpScreen() != screenKeys {
		return editorHelp[e.helpScreen()]
	}
	var out []string
	for _, key := range []string{"new", "players", "metadata", "help"} {
		for _, h := range editorKeyByName[key].help {
			if h.screen == screenKeys && !containsString(out, h.line) {
				out = append(out, h.line)
			}
		}
	}
	return out
}

// Key and Pointer are the same input doors used by Update and headless proof.
// A key with no row, or one the open prompt does not hear, does nothing.
func (e *MapEditor) Key(key string) error {
	if e.pathDialog.pending != nil {
		return nil
	}
	row := editorKeyByName[key]
	if row == nil {
		return nil
	}
	if e.prompt.open() || e.objectTyping() {
		switch row.live {
		case promptDeaf:
			return nil
		case promptFields:
			if !e.prompt.fields() && !e.objectTyping() && e.prompt.kind != promptAreaTrigger {
				return nil
			}
		}
	}
	return row.act(e, key)
}

func (e *MapEditor) keyClose(string) error {
	if e.prompt.kind == promptLeave {
		return e.runLeave(leaveStep{})
	}
	if e.Dirty() {
		e.askLeave(leaveStep{})
		return nil
	}
	return ebiten.Termination
}

// keyEscape leaves the foremost state: a drag, a pick, the prompt, the panel,
// the message, the brush and the catalogue, in that order, then the window.
func (e *MapEditor) keyEscape(string) error {
	switch {
	case e.recordDrag != nil:
		e.leave(leaveDrag)
	case e.pick != pickNone:
		e.leave(leavePick)
		e.message = ""
	case e.prompt.kind == promptAreaTrigger && e.prompt.area.open:
		e.prompt.area.open = false
	case e.prompt.open():
		e.leave(leavePrompt)
		e.saveThen = nil
		e.message = ""
	case e.menu != "":
		e.menu = ""
	case e.dropdownOpen():
		e.obj.open = -1
	case e.objectTyping():
		e.obj.focus = -1
	case e.report.open && e.message == "":
		e.leave(leaveReport)
	case e.message != "":
		e.message = ""
	case e.brush.mode != brushOff:
		e.leave(leaveBrush)
	case e.catalog && e.doc != nil:
		e.leave(leaveCatalog)
	case e.groupSel.set && !e.objectHidden || e.selected != -1 && !e.objectHidden:
		e.closeObject()
	case e.listShown:
		e.closeList()
	case e.panel:
		e.leave(leavePanel)
	case e.Dirty():
		e.askLeave(leaveStep{})
	default:
		return ebiten.Termination
	}
	return nil
}

func (e *MapEditor) keyOpen(string) error {
	e.askPath(pathdialog.OpenFile)
	return nil
}

func (e *MapEditor) keyCatalog(string) error {
	e.leave(leaveDrag | leaveReport)
	if e.catalog {
		e.leave(leaveCatalog)
	} else {
		e.enterCatalog()
	}
	e.message = ""
	return nil
}

func (e *MapEditor) keyEnter(string) error {
	if e.objectTyping() {
		e.applyObject()
		return nil
	}
	if e.catalog && !e.prompt.open() {
		e.catalogOpen()
		return nil
	}
	switch e.prompt.kind {
	case promptLeave:
		return e.leaveSaveAs()
	case promptNotice:
		return e.noticeContinue()
	case promptNewMode:
		return e.newModeOpen(false)
	case promptAreaTrigger:
		e.areaCreate()
		return nil
	case promptAreaShared:
		e.areaSharedContinue()
		return nil
	case promptAdd:
		if !e.addPicking() {
			e.addNext()
		}
		return nil
	case promptField, promptWorkspace:
		return e.setField(e.prompt.path)
	case promptItemRecipe:
		return e.setItemRecipe(e.prompt.path)
	case promptRectangle:
		e.askNewMode("new/" + strings.TrimSpace(e.prompt.path))
		return nil
	case promptSave:
		return e.saveAs(strings.TrimSpace(e.prompt.path))
	case promptOpen:
		return e.Open(strings.TrimSpace(e.prompt.path))
	}
	if len(e.report.goals) > 0 {
		e.armGoal()
	} else if e.report.open {
		e.selectFinding(e.report.sel)
	}
	return nil
}

func (e *MapEditor) keyReport(string) error {
	e.reportKey()
	return nil
}

func (e *MapEditor) brushKeyRow(key string) error {
	e.brushKey(key)
	return nil
}

// keyStep moves the brush palette's row, else the report's selected finding.
func (e *MapEditor) keyStep(key string) error {
	delta := 1
	if key == "up" {
		delta = -1
	}
	if e.prompt.kind == promptAdd {
		e.addMove(delta)
		return nil
	}
	if e.prompt.kind == promptAreaTrigger {
		e.areaStep(delta)
		return nil
	}
	if e.prompt.fields() {
		e.stepChoice(delta)
		return nil
	}
	if e.objectTyping() {
		if e.dropdownOpen() {
			dropdownStep(e.obj.choice, delta)
			e.chooseObjectChoiceKeep()
		} else {
			e.objectStep(delta)
		}
		return nil
	}
	if e.catalog {
		e.catalogStep(delta)
		return nil
	}
	if e.brushKey(key) {
		return nil
	}
	if e.listView() && (e.filter == editorPlayersFilter || e.filter == editorScriptsFilter) {
		delta := 1
		if key == "up" {
			delta = -1
		}
		e.stepPlayer(delta)
		return nil
	}
	if key == "up" {
		e.reportStep(-1)
	} else {
		e.reportStep(1)
	}
	return nil
}

func (e *MapEditor) keySaveAs(string) error {
	if m, ok := e.modSave(); ok {
		if e.askNotice(noticeModSave) {
			return nil
		}
		return e.saveIntoMod(m)
	}
	if e.editable() {
		e.askPath(pathdialog.SaveFile)
	}
	return nil
}

func (e *MapEditor) keyNew(string) error {
	e.leave(leaveDrag | leaveReport | leavePick)
	e.enterCatalog()
	e.openPrompt(editorPrompt{kind: promptRectangle, path: "80x80"})
	return nil
}

func (e *MapEditor) keyUnit(string) error {
	e.unitKey()
	return nil
}

// keyHistory is undo and redo: each is one history entry.
func (e *MapEditor) keyHistory(key string) error {
	e.leave(leaveDrag | leavePick)
	if !e.editable() {
		return nil
	}
	if key == "undo" && !e.doc.Edits.CanUndo() || key == "redo" && !e.doc.Edits.CanRedo() {
		return nil
	}
	var next *InspectionDocument
	var err error
	if key == "undo" {
		next, err = e.doc.Edits.Undo()
	} else {
		next, err = e.doc.Edits.Redo()
	}
	return e.adoptEdit(next, err)
}

func (e *MapEditor) keyBackspace(string) error {
	if e.objectTyping() {
		e.objectText("", true)
		return nil
	}
	if e.prompt.open() {
		r := []rune(e.prompt.path)
		if len(r) > 0 {
			e.prompt.path = string(r[:len(r)-1])
		}
		e.addFollow()
	}
	return nil
}

func (e *MapEditor) keyFit(string) error {
	e.Fit()
	return nil
}

// keyPage pages the list popup: the map picker's highlight, else the list's
// cursor.
func (e *MapEditor) keyPage(key string) error {
	if e.model == nil || e.listKind() == "" {
		return nil
	}
	m := e.model
	switch key {
	case "pageup":
		listKey(m, false, false, true, false)
	case "pagedown":
		listKey(m, false, false, false, true)
	case "home":
		m.Select(0)
	case "end":
		m.Select(m.Len() - 1)
	}
	if e.catalog && m.Len() > 0 && !m.Rows()[m.Selection()].Choosable {
		if key == "end" || key == "pagedown" {
			e.catalogStep(-1)
		} else {
			e.catalogStep(1)
		}
	}
	return nil
}
