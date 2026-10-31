package ui

import (
	"againrom/pkg/pathdialog"
	"againrom/pkg/words"
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
	screenKeys        helpScreen = iota // no pick, panel or brush: the keys
	screenClone                         // a clone is armed
	screenDrop                          // the drop cell is armed
	screenGoal                          // a script arm's target is armed
	screenGoals                         // the script panel is open
	screenTerrain                       // the terrain brush
	screenHeight                        // the height brush
	screenObjects                       // the objects brush
	screenPassability                   // the passability brush
	screenPlayers                       // the roster list
	screenPrompt
	screenMove
	screenAdd       // the Add window
	screenAddCell   // the Add window's cell is armed
	screenSelection // several records are selected
	screenCount
)

// helpLine is a help line: the message id of its words.
type helpLine struct{ Word string }

// helpEntry is a line of a help screen.
type helpEntry struct {
	screen helpScreen
	line   helpLine
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
	unlistedGroups   = "the 1..9 and Ctrl+1..9 help lines name it"
)

var (
	editorKeyRows   []keyRow
	editorKeyByName map[string]*keyRow
	editorHelp      [screenCount][]helpLine
)

// The help lines, each once.
var (
	lineMove          = helpLine{Word: "editor.keys.move"}
	lineFields        = helpLine{Word: "editor.keys.fields"}
	lineBrush         = helpLine{Word: "editor.keys.brush"}
	lineNew           = helpLine{Word: "editor.keys.new"}
	linePlayers       = helpLine{Word: "editor.keys.players"}
	lineMap           = helpLine{Word: "editor.keys.map"}
	lineStructure     = helpLine{Word: "editor.keys.structure"}
	lineHelp          = helpLine{Word: "editor.keys.help"}
	lineArea          = helpLine{Word: "editor.keys.area"}
	linePassability   = helpLine{Word: "editor.keys.passability"}
	lineSelect        = helpLine{Word: "editor.keys.select"}
	lineSelectClick   = helpLine{Word: "editor.keys.selectclick"}
	lineCopy          = helpLine{Word: "editor.keys.copy"}
	linePrompt        = helpLine{Word: "editor.keys.prompt"}
	lineAddNext       = helpLine{Word: "editor.keys.addnext"}
	lineAddChoose     = helpLine{Word: "editor.keys.addchoose"}
	lineArms          = helpLine{Word: "editor.keys.arms"}
	lineRoster        = helpLine{Word: "editor.keys.roster"}
	lineRosterFields  = helpLine{Word: "editor.keys.rosterfields"}
	lineTarget        = helpLine{Word: "editor.keys.target"}
	lineTerrain       = helpLine{Word: "editor.keys.terrain"}
	lineHeight        = helpLine{Word: "editor.keys.height"}
	lineObjects       = helpLine{Word: "editor.keys.objects"}
	linePassBrush     = helpLine{Word: "editor.keys.passbrush"}
	lineBrushSize     = helpLine{Word: "editor.keys.brushsize"}
	lineCloneCell     = helpLine{Word: "editor.keys.clonecell"}
	lineDropCell      = helpLine{Word: "editor.keys.dropcell"}
	lineTargetCell    = helpLine{Word: "editor.keys.targetcell"}
	lineTargetCancel  = helpLine{Word: "editor.keys.targetcancel"}
	lineMoveCell      = helpLine{Word: "editor.keys.movecell"}
	lineAddCell       = helpLine{Word: "editor.keys.addcell"}
	lineCancel        = helpLine{Word: "editor.keys.cancel"}
	lineRecords       = helpLine{Word: "editor.keys.records"}
	lineBack          = helpLine{Word: "editor.keys.back"}
	lineSelectionMove = helpLine{Word: "editor.keys.selectionmove"}
	lineSelectionEsc  = helpLine{Word: "editor.keys.selectionclear"}
	lineSelectBox     = helpLine{Word: "editor.keys.selectbox"}
	lineBoxMove       = helpLine{Word: "editor.keys.boxmove"}
	lineGroups        = helpLine{Word: "editor.keys.groups"}
	lineToGroup       = helpLine{Word: "editor.keys.togroup"}
	linePlace         = helpLine{Word: "editor.keys.place"}
	lineRandomTerrain = helpLine{Word: "editor.keys.randomterrain"}
	lineRandomHeights = helpLine{Word: "editor.keys.randomheights"}
	lineHeightLabels  = helpLine{Word: "editor.keys.heightlabels"}
	linePassView      = helpLine{Word: "editor.keys.passview"}
)

func init() {
	keys := func(line helpLine) []helpEntry { return []helpEntry{{screenKeys, line}} }
	brushes := []helpEntry{{screenTerrain, lineTerrain}, {screenHeight, lineHeight}, {screenObjects, lineObjects}, {screenPassability, linePassBrush}}
	editorKeyRows = []keyRow{
		{key: "enter", label: "Enter", live: promptHears, act: (*MapEditor).keyEnter, help: []helpEntry{{screenGoals, lineTarget}, {screenPrompt, linePrompt}, {screenAdd, lineAddNext}}},
		{key: "clone", label: "C", act: (*MapEditor).recordKey, help: keys(lineMove)},
		{key: "delete", label: "Del", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, lineMove}, {screenPlayers, lineRoster}, {screenSelection, lineSelectionMove}}},
		{key: "field", label: "E", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, lineFields}, {screenPlayers, lineRosterFields}}},
		{key: "sack", label: "S", act: (*MapEditor).recordKey, help: keys(lineFields)},
		{key: "report", label: "R", act: (*MapEditor).keyReport, help: keys(lineFields)},
		{key: "drop", label: "D", act: (*MapEditor).recordKey, help: keys(lineBrush)},
		{key: "goals", label: "G", act: (*MapEditor).recordKey, help: []helpEntry{{screenKeys, lineBrush}, {screenGoals, lineArms}}},
		// The size line leads each brush screen, so a status strip too narrow
		// for both drops it and keeps the brush's own line.
		{label: "Alt+wheel", help: []helpEntry{{screenTerrain, lineBrushSize}, {screenHeight, lineBrushSize}, {screenObjects, lineBrushSize}, {screenPassability, lineBrushSize}}},
		{key: "terrain", label: "T", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, lineBrush}, {screenTerrain, lineTerrain}}},
		{key: "height", label: "H", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, lineBrush}, {screenHeight, lineHeight}}},
		{key: "objects", label: "B", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, lineBrush}, {screenObjects, lineObjects}}},
		{key: "passability", label: "Q", act: (*MapEditor).brushKeyRow, help: []helpEntry{{screenKeys, linePassability}, {screenPassability, linePassBrush}}},
		{key: "radiusdown", label: "[", act: (*MapEditor).brushKeyRow, help: brushes},
		{key: "radiusup", label: "]", act: (*MapEditor).brushKeyRow, help: brushes},
		{key: "new", label: "N", act: (*MapEditor).keyNew, help: keys(lineNew)},
		{key: "unit", label: "U", act: (*MapEditor).keyUnit, help: keys(lineNew)},
		{key: "structure", label: "J", act: (*MapEditor).keyRecords, help: keys(lineStructure)},
		{key: "stock", label: "K", act: (*MapEditor).keyRecords, help: keys(lineStructure)},
		{key: "recipe", label: "I", act: (*MapEditor).keyRecords, help: keys(lineStructure)},
		{key: "casts", label: "F3", act: func(e *MapEditor, _ string) error { e.openAdd("Enchantment"); return nil }, help: keys(helpLine{Word: "editor.keys.casts"})},
		{key: "players", label: "P", act: (*MapEditor).keyPlayers, help: []helpEntry{{screenKeys, linePlayers}, {screenPlayers, lineRoster}}},
		{key: "player", label: "A", act: (*MapEditor).keyAdd, help: []helpEntry{{screenPlayers, lineRoster}}},
		{key: "testplay", label: "F5", act: (*MapEditor).keyTestPlay, help: keys(linePlayers)},
		{key: "scripts", label: "F2", act: (*MapEditor).keyScripts, help: keys(lineHelp)},
		{key: "metadata", label: "M", act: (*MapEditor).keyWorkspace, help: keys(lineMap)},
		{key: "pages", label: "X", live: promptHears, act: (*MapEditor).keyWorkspace, help: keys(lineMap)},
		{key: "package", label: "W", act: (*MapEditor).keyWorkspace, help: keys(lineMap)},
		{key: "help", label: "F1", act: (*MapEditor).keyHelp, help: keys(lineHelp)},
		{key: "drawarea", label: "L", act: (*MapEditor).keyDrawArea, help: keys(lineArea)},
		{label: "click", help: []helpEntry{{screenClone, lineCloneCell}, {screenDrop, lineDropCell}, {screenGoal, lineTargetCell}, {screenMove, lineMoveCell}, {screenAddCell, lineAddCell}}},
		{label: "Drag", help: []helpEntry{{screenKeys, lineSelectBox}, {screenKeys, lineBoxMove}, {screenAddCell, linePlace}}},
		{label: "1..9", help: []helpEntry{{screenKeys, lineGroups}, {screenSelection, lineGroups}}},
		{label: "Ctrl+1..9", help: []helpEntry{{screenKeys, lineToGroup}, {screenSelection, lineToGroup}}},
		{label: "Shift", help: []helpEntry{{screenKeys, lineSelect}, {screenKeys, lineSelectClick}}},
		{key: "copy", label: "Ctrl+C", act: (*MapEditor).keyCopy, help: []helpEntry{{screenKeys, lineCopy}, {screenSelection, lineCopy}}},
		{key: "paste", label: "Ctrl+V", act: (*MapEditor).selectionKey, help: []helpEntry{{screenKeys, lineCopy}, {screenSelection, lineCopy}}},
		{key: "randomterrain", label: "Ctrl+E", act: (*MapEditor).keyRandom, help: keys(lineRandomTerrain)},
		{key: "randomheights", label: "Ctrl+Shift+R", act: (*MapEditor).keyRandom, help: keys(lineRandomHeights)},
		{key: "passview", label: "Z", act: (*MapEditor).keyPassView, help: keys(linePassView)},
		{key: "heightlabels", label: "V", act: (*MapEditor).keyHeightLabels, help: keys(lineHeightLabels)},
		{key: "escape", label: "Esc", live: promptHears, act: (*MapEditor).keyEscape, help: []helpEntry{{screenClone, lineCancel}, {screenDrop, lineCancel}, {screenGoal, lineTargetCancel}, {screenPlayers, lineRecords}, {screenPrompt, linePrompt}, {screenMove, lineCancel}, {screenAdd, lineAddNext}, {screenAddCell, lineBack}, {screenSelection, lineSelectionEsc}}},
		{key: "up", label: "Up", live: promptFields, act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, lineArms}, {screenPlayers, lineRosterFields}, {screenAdd, lineAddChoose}}},
		{key: "down", label: "Down", live: promptFields, act: (*MapEditor).keyStep, help: []helpEntry{{screenGoals, lineArms}, {screenPlayers, lineRosterFields}, {screenAdd, lineAddChoose}}},
		{key: "close", live: promptHears, act: (*MapEditor).keyClose, unlisted: unlistedWindow},
		{key: "open", act: (*MapEditor).keyOpen, unlisted: unlistedRail},
		{key: "catalog", act: (*MapEditor).keyCatalog, unlisted: unlistedRail},
		{key: "saveas", act: (*MapEditor).keySaveAs, unlisted: unlistedSaveLine},
		{key: "undo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "redo", act: (*MapEditor).keyHistory, unlisted: unlistedSaveLine},
		{key: "tab", label: "Tab", live: promptFields, act: (*MapEditor).recordKey, help: []helpEntry{{screenPrompt, linePrompt}}},
		{key: "backtab", label: "Shift+Tab", live: promptHears, act: (*MapEditor).keyModBacktab, unlisted: unlistedWindow},
		{key: "applyobject", live: promptFields, act: (*MapEditor).keyObjectApply, unlisted: unlistedWindow},
		{key: "newline", live: promptHears, act: (*MapEditor).keyNewline, unlisted: unlistedPrompt},
		{key: "clear", live: promptHears, act: (*MapEditor).keyClear, unlisted: unlistedPrompt},
		{key: "left", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{key: "right", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{key: "selectall", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{key: "textundo", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{key: "textredo", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{key: "cut", live: promptHears, act: (*MapEditor).keyText, unlisted: unlistedPrompt},
		{label: "Ctrl+C", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.clipboard"}}}},
		{label: "Ctrl+A", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.select"}}}},
		{label: "Ctrl+Z", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.undo"}}}},
		{label: "Ctrl+Left", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.words"}}}},
		{label: "Backspace", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.repeat"}}}},
		{label: "Popup", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.popup"}}}},
		{label: "Unfocus", help: []helpEntry{{screenPrompt, helpLine{Word: "editor.keys.text.unfocus"}}}},
		{key: "backspace", live: promptHears, act: (*MapEditor).keyBackspace, unlisted: unlistedPrompt},
		{key: "fit", act: (*MapEditor).keyFit, unlisted: unlistedRail},
		{key: "triggers", act: (*MapEditor).keyScripts, unlisted: "the Scripts entry opens it"},
		{key: "pageup", live: promptFields, act: (*MapEditor).keyPage, unlisted: "the list line names PgUp and PgDn"},
		{key: "pagedown", live: promptFields, act: (*MapEditor).keyPage, unlisted: "the list line names PgUp and PgDn"},
		{key: "home", live: promptFields, act: (*MapEditor).keyPage, unlisted: "list travel"},
		{key: "end", live: promptFields, act: (*MapEditor).keyPage, unlisted: "list travel"},
		{label: "Dialogue", help: keys(helpLine{Word: "editor.keys.dialogue"})},
		{label: "+", help: keys(helpLine{Word: "editor.keys.dialogue.add"})},
		{label: "x", help: keys(helpLine{Word: "editor.keys.dialogue.remove"})},
		{label: "Restore", help: keys(helpLine{Word: "editor.keys.dialogue.restore"})},
		{label: "Drag", help: keys(helpLine{Word: "editor.keys.dialogue.move"})},
		{label: "Alt+Up", help: keys(helpLine{Word: "editor.keys.dialogue.keys"})},
		{key: "cardup", live: promptHears, act: (*MapEditor).keyCard, unlisted: "the Alt+Up line names Alt+Up and Alt+Down"},
		{key: "carddown", live: promptHears, act: (*MapEditor).keyCard, unlisted: "the Alt+Up line names Alt+Up and Alt+Down"},
		{label: "Apply", help: keys(helpLine{Word: "editor.keys.dialogue.apply"})},
	}
	// 1..9 choose a group and frame it; Ctrl+1..9 put the selection in it.
	for n := '1'; n <= '9'; n++ {
		editorKeyRows = append(editorKeyRows,
			keyRow{key: "group" + string(n), act: (*MapEditor).groupKey, unlisted: unlistedGroups},
			keyRow{key: "togroup" + string(n), act: (*MapEditor).groupKey, unlisted: unlistedGroups})
	}
	editorKeyByName = map[string]*keyRow{}
	for i := range editorKeyRows {
		r := &editorKeyRows[i]
		if r.key != "" {
			editorKeyByName[r.key] = r
		}
		for _, h := range r.help {
			if !containsLine(editorHelp[h.screen], h.line) {
				editorHelp[h.screen] = append(editorHelp[h.screen], h.line)
			}
		}
	}
}

func containsLine(list []helpLine, l helpLine) bool {
	for _, x := range list {
		if x == l {
			return true
		}
	}
	return false
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// EditorHelpLines is every help line the editor window shows, in English.
func EditorHelpLines() []string { return EditorHelpLinesIn("") }

// EditorHelpLinesIn is every help line the editor window shows, in UTF-8, in
// the words of the install language lang.
func EditorHelpLinesIn(lang string) []string {
	var out []string
	for _, lines := range editorHelp {
		out = append(out, helpWords(words.For(lang).Text, lines)...)
	}
	return out
}

// helpWords is the words of lines read through text.
func helpWords(text func(string) string, lines []helpLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = text(l.Word)
	}
	return out
}

// allHelpLines is every help line in the window's words, each once: the F1
// list.
func (e *MapEditor) allHelpLines() []string {
	var out []string
	for _, lines := range editorHelp {
		for _, line := range helpWords(e.word, lines) {
			if !containsString(out, line) {
				out = append(out, line)
			}
		}
	}
	return out
}

// brushScreens is the help screen of each brush mode.
var brushScreens = map[brushMode]helpScreen{brushTerrain: screenTerrain, brushHeight: screenHeight, brushObjects: screenObjects, brushPassability: screenPassability}

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
	if e.selectionShown() {
		return screenSelection
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
		return helpWords(e.word, editorHelp[e.helpScreen()])
	}
	var out []helpLine
	for _, key := range []string{"new", "players", "metadata", "help"} {
		for _, h := range editorKeyByName[key].help {
			if h.screen == screenKeys && !containsLine(out, h.line) {
				out = append(out, h.line)
			}
		}
	}
	return helpWords(e.word, out)
}

// Key and Pointer are the same input doors used by Update and headless proof.
// A key with no row, or one the open prompt does not hear, does nothing.
func (e *MapEditor) Key(key string) error {
	if e.pathDialog.pending != nil {
		return nil
	}
	if e.textKey(key) {
		return nil
	}
	if e.modWindow && !e.prompt.open() {
		if handled, err := e.modKey(key); handled {
			return err
		}
	}
	if confirm := e.report.confirm; confirm != nil && !e.prompt.open() && (key == "enter" || key == "sack") {
		e.report.confirm = nil
		return confirm()
	}
	row := editorKeyByName[key]
	if row == nil {
		return nil
	}
	if !e.keyRowHeard(row) {
		return nil
	}
	switch key {
	case "open", "saveas", "new", "unit", "structure", "sack", "player", "recipe", "metadata", "pages", "package", "field", "close", "testplay":
		if !e.prompt.open() {
			e.rememberCancel()
		}
	}
	wasPrompt := e.prompt.open()
	restore, document := e.cancelRestore, e.doc
	err := row.act(e, key)
	if e.prompt.open() && e.doc == document && restore != nil {
		e.cancelRestore = restore
	}
	if !wasPrompt && !e.prompt.open() && e.pathDialog.pending == nil {
		e.cancelRestore = nil
	}
	return err
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

// keyEscape cancels the foremost gesture, popup or mode.
func (e *MapEditor) keyEscape(string) error {
	switch {
	case e.brush.set != nil:
		e.brush.endStroke()
	case e.recordDrag != nil:
		e.leave(leaveDrag)
	case e.pick != pickNone:
		e.leave(leavePick)
		e.message = ""
	case e.prompt.kind == promptAreaTrigger && e.prompt.area.open:
		e.prompt.area.open = false
	case e.prompt.open():
		e.cancelPopup()
	case e.menu != "":
		e.menu = ""
	case e.dropdownOpen():
		e.obj.open = -1
	case e.objectTyping():
		e.cancelPopup()
	case e.report.open && e.message == "":
		e.cancelPopup(popList)
	case e.message != "":
		e.message = ""
	case e.catalog && e.doc != nil:
		e.cancelPopup(popList)
	case e.brush.mode != brushOff:
		e.cancelPopup(popList)
	case e.selectionShown():
		e.clearSelection()
	case e.groupSel.set && !e.objectHidden || e.selected != -1 && !e.objectHidden:
		e.cancelPopup()
	case e.listShown:
		e.cancelPopup(popList)
	case e.panel:
		e.cancelPopup(popList)
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
	if e.nativeUnitWindow() && e.dropdownOpen() && e.obj.choice != nil && !e.prompt.open() {
		e.chooseObjectChoice(e.obj.choice.Selection())
		return nil
	}
	if e.nativeUnitWindow() && e.objectTyping() {
		if len(e.obj.fields[e.obj.focus].Choices) > 0 {
			e.objectRowPress(popupRowControl, e.fieldRow(e.obj.focus))
		} else {
			e.unitSettings()
			e.obj.focus = -1
		}
		return nil
	}
	if e.objectTyping() {
		if e.messageTextField(e.obj.focus) {
			e.TextInput("\n")
			return nil
		}
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
	case promptToGroup:
		e.toGroupChosen()
		return nil
	case promptGroupWarn:
		e.groupWarnGo()
		return nil
	case promptAdd:
		e.addNext()
		return nil
	case promptField, promptWorkspace:
		return e.setField(e.prompt.path)
	case promptItemRecipe:
		return e.setItemRecipe(e.prompt.path)
	case promptRectangle:
		e.askNewMode(MapSourceKey(SourceNew, strings.TrimSpace(e.prompt.path)))
		return nil
	case promptForestShare:
		e.setForestShare(e.prompt.path)
		return nil
	case promptSeed:
		return e.applyRandom(e.prompt.path)
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

func (e *MapEditor) keyObjectApply(string) error {
	if !e.prompt.open() && !e.objectHidden && e.doc != nil {
		e.loadObject()
		if e.nativeUnitWindow() {
			e.applyObject()
		}
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
			e.obj.choice.MoveChoice(delta)
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
	if !e.editable() {
		return nil
	}
	if p, ok := e.doc.Edits.(InspectionInPlace); ok {
		if _, own := p.OwnFile(); own {
			return e.saveWith(p.SaveInPlace)
		}
	}
	e.askPath(pathdialog.SaveFile)
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
	if _, err := e.commandHistory(key == "redo"); err != nil {
		e.message = err.Error()
		return err
	}
	return nil
}

func (e *MapEditor) keyBackspace(string) error {
	if !e.prompt.open() && !e.objectHidden && e.searchableObjectChoices() && e.dropdownOpen() {
		query := []rune(e.obj.search)
		if len(query) > 0 {
			e.obj.search = string(query[:len(query)-1])
			e.obj.choice.Find(e.obj.search, e.editorSearchText)
		}
		return nil
	}
	e.textKey("backspace")
	return nil
}

func (e *MapEditor) keyFit(string) error {
	e.Fit()
	return nil
}

// keyPage pages the list popup: the map picker's highlight, else the list's
// cursor.
func (e *MapEditor) keyPage(key string) error {
	m := e.model
	add := false
	if r, ok := e.addRunning(); ok {
		m, add = r.list, true
	} else if e.dropdownOpen() {
		m = e.obj.choice
	} else if e.prompt.kind == promptAreaTrigger {
		m = e.prompt.area.choice
	}
	if m == nil {
		return nil
	}
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
	if add {
		e.addTake(m.Selection())
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
