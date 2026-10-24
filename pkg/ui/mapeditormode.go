package ui

// The map editor window's modes are held in one value, editorMode. Its layers
// are the ones an author can have at once: the catalogue instead of the record
// list, a text prompt, an armed pick, a brush, and the report or script panel.
// This file is the only place that enters or leaves a layer; a test scans the
// package for an assignment anywhere else.

import "image"

// pickKind is what the next click on the canvas picks.
type pickKind string

const (
	pickNone  pickKind = ""
	pickClone pickKind = "clone" // a cell to clone the selected record onto
	pickDrop  pickKind = "drop"  // the cell the party drops on
	pickGoal  pickKind = "goal"  // a unit or structure for the selected script arm
	pickMove  pickKind = "move"  // a cell to move the selected record to
	// pickAddCell is the cell step of the Add window.
	pickAddCell pickKind = "addcell"
	// pickScriptCell is a cell for the open node popup's cell parameter.
	pickScriptCell pickKind = "scriptcell"
	// pickArea is the draw tool: the next drag on the map draws a new
	// trigger area.
	pickArea pickKind = "area"
	// pickWizardUnit is a unit on the map for the trigger wizard's row.
	pickWizardUnit pickKind = "wizardunit"
)

// promptKind is the text prompt in the rail.
type promptKind uint8

const (
	promptNone  promptKind = iota
	promptOpen             // an ALM path to open
	promptSave             // a new ALM path for Save As
	promptField            // a value for the selected record's field
	promptItemRecipe
	promptRectangle
	promptWorkspace
	promptAdd // the Add window; its text is the current step's
	// promptLeave asks what to do with unsaved changes before a step runs.
	promptLeave
	// promptNotice warns before test play or a save into a mod and lets
	// the author continue past it.
	promptNotice
	// promptNewMode asks whether a new map is single-player or multiplayer.
	promptNewMode
	// promptAreaTrigger asks which trigger takes a box drawn on the map.
	promptAreaTrigger
	// promptAreaShared warns that an area drag changes a node other
	// triggers or pairs share, before it writes.
	promptAreaShared
	// promptToGroup asks which group the selection's units go to.
	promptToGroup
	// promptGroupWarn warns before an edit empties a group a trigger names.
	promptGroupWarn
	// promptForestShare asks for the Forest brush's density in per cent.
	promptForestShare
	// promptSeed asks for the seed of a whole-map command; scope is the
	// command's key.
	promptSeed
)

// editorPrompt is a prompt and the data it owns: the typed text, the field it
// cycles to and, for the Add window, its run.
type editorPrompt struct {
	kind  promptKind
	field int
	path  string
	scope string
	add   *addRun
	then  leaveStep // the step a promptLeave holds, or the new map a promptNewMode opens
	// notice is what a promptNotice continues to: test play or the save
	// into the mod.
	notice noticeStep
	// area is what an area prompt holds: the drawn box or the held edit.
	area *areaPrompt
	// group is what a group prompt holds: the units and their group.
	group *groupPrompt
}

func (p editorPrompt) open() bool { return p.kind != promptNone }
func (p editorPrompt) fields() bool {
	return p.kind == promptField || p.kind == promptItemRecipe || p.kind == promptWorkspace || p.kind == promptAdd
}

// editorMode is every layer the window can be in.
type editorMode struct {
	catalog bool
	prompt  editorPrompt
	pick    pickKind
	brush   brushState
	report  reportState
	panel   bool // the Map panel stands in the list popup under any list
}

// free reports that no layer takes the canvas: the catalogue, a prompt and an
// armed pick each do.
func (m *editorMode) free() bool {
	return !m.catalog && !m.prompt.open() && m.pick == pickNone
}

// listView reports that the record list is what the rail shows.
func (m *editorMode) listView() bool { return !m.catalog && !m.report.open }

// exits names the layers one change of mode leaves.
type exits uint8

const (
	leaveDrag    exits = 1 << iota // a record drag in progress
	leavePick                      // an armed pick
	leavePrompt                    // the prompt, with its text and cell
	leaveReport                    // the report or script panel
	leaveCatalog                   // the catalogue
	leaveBrush                     // the brush mode and its stroke
	leavePanel                     // the Map panel
)

// leave clears what the named layers own, the list's fractional wheel
// included where the layer owns the list.
func (e *MapEditor) leave(x exits) {
	if x&leaveDrag != 0 {
		e.cancelRecordDrag()
	}
	if x&leavePick != 0 {
		e.pick = pickNone
	}
	if x&leavePrompt != 0 {
		e.prompt = editorPrompt{}
		e.cancelRestore = nil
	}
	if x&leaveReport != 0 {
		e.closeReport()
	}
	if x&leaveCatalog != 0 {
		e.catalog = false
		e.resetList(0)
	}
	if x&leaveBrush != 0 {
		e.brush.mode = brushOff
		e.brush.endStroke()
	}
	if x&leavePanel != 0 {
		e.panel = false
	}
}

func (e *MapEditor) restoreMode(mode editorMode) { e.editorMode = mode }

// openPanel shows the Map panel in the list popup: the records list it
// stands under closes.
func (e *MapEditor) openPanel() {
	e.rememberListCancel("panel")
	e.leave(leaveCatalog | leaveReport | leaveBrush)
	e.panel = true
	e.listShown, e.filter = false, 0
}

// resetList puts the list popup back at its first row and drops the wheel's
// remainder.
func (e *MapEditor) resetList(int) {
	e.model, e.modelKey, e.listWheel = nil, "", 0
}

// enterCatalog shows the catalogue at its first row.
func (e *MapEditor) enterCatalog() {
	e.rememberListCancel("catalog")
	e.catalog = true
	e.resetList(0)
}

// armPick arms the next canvas click to pick kind.
func (e *MapEditor) armPick(kind pickKind) {
	e.pick, e.message = kind, ""
}

// keepPick keeps kind armed after a pick that does not end it, the message
// the pick left standing.
func (e *MapEditor) keepPick(kind pickKind) { e.pick = kind }

// openPrompt replaces the prompt.
func (e *MapEditor) openPrompt(p editorPrompt) {
	e.rememberCancel()
	e.input = editorInputState{}
	e.prompt, e.message, e.promptShift = p, "", image.Point{}
}

// toggleBrush switches the brush mode on, or off when it already is mode.
func (e *MapEditor) toggleBrush(mode brushMode) {
	if !e.catalog {
		if e.brush.mode == mode {
			e.listRestore = nil
		} else {
			e.rememberListCancel("brush")
		}
	}
	e.leave(leaveDrag | leaveReport | leavePick | leavePrompt)
	e.message = ""
	e.brush.endStroke()
	if e.brush.mode == mode {
		mode = brushOff
	}
	e.brush.mode = mode
}

// openReport shows rows under a note, leaving the record list's offset to be
// restored when the panel closes. goals is set while the rows are the script
// panel's arms.
func (e *MapEditor) openReport(rows []InspectionFinding, note string, goals []InspectionGoal) {
	e.rememberListCancel("report")
	e.report.open, e.report.rows, e.report.note, e.report.sel = true, rows, note, 0
	e.report.goals = goals
	e.report.flagged = map[string]bool{}
	for _, f := range rows {
		if f.Kind != "" {
			e.report.flagged[findingKey(f.Kind, f.Index)] = true
		}
	}
	e.leave(leaveCatalog)
}

func (e *MapEditor) closeReport() {
	if !e.report.open {
		return
	}
	e.report = reportState{}
	e.resetList(0)
}

// blocksCanvas reports that the map takes no pointer: the catalogue or a
// prompt covers it. The Add window's cell pick leaves the map to the click.
func (m *editorMode) blocksCanvas() bool { return m.catalog || m.prompt.open() && !m.addPicking() }

// addPicking reports that the Add window waits for a cell click on the map.
func (m *editorMode) addPicking() bool { return m.prompt.kind == promptAdd && m.pick == pickAddCell }
