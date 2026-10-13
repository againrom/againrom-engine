package ui

import (
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
	screenSack                      // a sack is armed
	screenUnit                      // a unit is armed
	screenDrop                      // the drop cell is armed
	screenGoal                      // a script arm's target is armed
	screenGoals                     // the script panel is open
	screenTerrain                   // the terrain brush
	screenHeight                    // the height brush
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
	promptFields                   // while the prompt asks for a number
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
	line0, line1, line2, line3 := "Drag: move / C: clone / Del", "E: fields / S: sack / R: report", "D: drop / G: arms / T, H: brush", "N: new map / U: unit"
	arms := "Up, Down: select / G: close"
	editorKeyRows = []keyRow{
		{key: "enter", label: "Enter", live: promptHears, act: (*MapEditor).keyEnter, help: []helpEntry{{screenGoals, "Enter: pick a target"}}},
		{key: "clone", label: "C", act: (*MapEditor).recordKey, help: keys(line0)},
		{key: "delete", label: "Del", act: (*MapEditor).recordKey, help: keys(line0)},
		{key: "field", label: "E", act: (*MapEditor).recordKey, help: keys(line1)},
		{key: "sack", label: "S", act: (*MapEditor).recordKey, help: keys(line1)},
		{key: "report", label: "R", act: (*MapEditor).keyReport, help: keys(line1)},
		{key: "drop", label: "D", act: (*MapEditor).recordKey, help: keys(line2)},
		{key: "goals", label: "G", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, line2}, {screenGoals, arms}}},
		{key: "terrain", label: "T", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, line2}, {screenTerrain, "T: off / Drag: paint / [ ]: size"}}},
		{key: "height", label: "H", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, line2}, {screenHeight, "H: off / Drag: shape / [ ]: size"}}},
		{key: "radiusdown", label: "[", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenTerrain, "T: off / Drag: paint / [ ]: size"}, {screenHeight, "H: off / Drag: shape / [ ]: size"}}},
		{key: "radiusup", label: "]", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenTerrain, "T: off / Drag: paint / [ ]: size"}, {screenHeight, "H: off / Drag: shape / [ ]: size"}}},
		{key: "new", label: "N", act: (*MapEditor).keyNew, help: keys(line3)},
		{key: "unit", label: "U", act: (*MapEditor).keyUnit, help: keys(line3)},
		{label: "click", help: []helpEntry{{screenClone, "Clone: click a cell"}, {screenSack, "Sack: click a cell"}, {screenUnit, "Unit: click a cell"}, {screenDrop, "Drop cell: click a cell"}, {screenGoal, "Target: click a unit or"}}},
		{key: "escape", label: "Esc", live: promptHears, act: (*MapEditor).keyEscape, help: []helpEntry{{screenClone, "Esc: cancel"}, {screenSack, "Esc: cancel"}, {screenUnit, "Esc: cancel"}, {screenDrop, "Esc: cancel"}, {screenGoal, "structure / Esc: cancel"}}},
		{key: "up", label: "Up", act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, arms}}},
		{key: "down", label: "Down", act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, arms}}},
		{key: "close", live: promptHears, act: (*MapEditor).keyClose, unlisted: unlistedWindow},
		{key: "open", act: (*MapEditor).keyOpen, unlisted: unlistedRail},
		{key: "catalog", act: (*MapEditor).keyCatalog, unlisted: unlistedRail},
		{key: "saveas", act: (*MapEditor).keySaveAs, unlisted: unlistedSaveLine},
		{key: "undo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "redo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "tab", live: promptFields, act: (*MapEditor).recordKey, unlisted: unlistedPrompt},
		{key: "backspace", live: promptHears, act: (*MapEditor).keyBackspace, unlisted: unlistedPrompt},
		{key: "fit", act: (*MapEditor).keyFit, unlisted: unlistedRail},
		{key: "triggers", act: (*MapEditor).keyTriggers, unlisted: "the rail has a filter for it"},
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

// pickScreens is the help screen of each armed pick.
var pickScreens = map[pickKind]helpScreen{pickClone: screenClone, pickSack: screenSack, pickUnit: screenUnit, pickDrop: screenDrop, pickGoal: screenGoal}

// helpScreen is the help for the window's mode: an active brush, the armed
// pick, the script panel, or the keys.
func (e *MapEditor) helpScreen() helpScreen {
	if e.brushActive() {
		if e.brush.mode == brushHeight {
			return screenHeight
		}
		return screenTerrain
	}
	if s, ok := pickScreens[e.pick]; ok {
		return s
	}
	if len(e.report.goals) > 0 {
		return screenGoals
	}
	return screenKeys
}

func (e *MapEditor) helpLines() []string { return editorHelp[e.helpScreen()] }

// Key and Pointer are the same input doors used by Update and headless proof.
// A key with no row, or one the open prompt does not hear, does nothing.
func (e *MapEditor) Key(key string) error {
	row := editorKeyByName[key]
	if row == nil {
		return nil
	}
	if e.prompt.open() {
		switch row.live {
		case promptDeaf:
			return nil
		case promptFields:
			if !e.prompt.fields() {
				return nil
			}
		}
	}
	return row.act(e, key)
}

func (e *MapEditor) keyClose(string) error {
	if e.Dirty() {
		e.message = "Unsaved changes. Save As or undo changes before closing."
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
	case e.prompt.open():
		e.leave(leavePrompt)
		e.message = ""
	case e.report.open && e.message == "":
		e.leave(leaveReport)
	case e.message != "":
		e.message = ""
	case e.brush.mode != brushOff:
		e.leave(leaveBrush)
	case e.catalog && e.doc != nil:
		e.leave(leaveCatalog)
	case e.Dirty():
		e.message = "Unsaved changes. Save As or undo changes before closing."
	default:
		return ebiten.Termination
	}
	return nil
}

func (e *MapEditor) keyOpen(string) error {
	e.leave(leaveDrag | leaveReport | leavePick)
	e.openPrompt(editorPrompt{kind: promptOpen})
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
	switch e.prompt.kind {
	case promptField, promptNewSack, promptNewUnit:
		return e.setField(strings.TrimSpace(e.prompt.path))
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
	if e.brushKey(key) {
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
		return e.saveIntoMod(m)
	}
	if e.editable() {
		e.leave(leaveDrag | leaveReport | leavePick)
		e.openPrompt(editorPrompt{kind: promptSave})
	}
	return nil
}

func (e *MapEditor) keyNew(string) error {
	e.leave(leaveDrag | leaveReport | leavePick)
	e.enterCatalog()
	e.message = ""
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
	if e.prompt.open() {
		r := []rune(e.prompt.path)
		if len(r) > 0 {
			e.prompt.path = string(r[:len(r)-1])
		}
	}
	return nil
}

func (e *MapEditor) keyFit(string) error {
	e.Fit()
	return nil
}

func (e *MapEditor) keyTriggers(string) error {
	e.filter = 7
	e.resetList(0)
	if e.selected >= 0 {
		e.Select(e.selected, false)
	}
	return nil
}

func (e *MapEditor) keyPage(key string) error {
	switch key {
	case "pageup":
		e.resetList(max(0, e.offset-e.visibleRows()))
	case "pagedown":
		e.resetList(min(max(0, e.rowCount()-e.visibleRows()), e.offset+e.visibleRows()))
	case "home":
		e.resetList(0)
	case "end":
		e.resetList(max(0, e.rowCount()-e.visibleRows()))
	}
	return nil
}
