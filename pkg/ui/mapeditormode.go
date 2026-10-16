package ui

import "image"

// The map editor window's modes are held in one value, editorMode. Its layers
// are the ones an author can have at once: the catalogue instead of the record
// list, a text prompt, an armed pick, a brush, and the report or script panel.
// This file is the only place that enters or leaves a layer; a test scans the
// package for an assignment anywhere else.

// pickKind is what the next click on the canvas picks.
type pickKind string

const (
	pickNone  pickKind = ""
	pickClone pickKind = "clone" // a cell to clone the selected record onto
	pickSack  pickKind = "sack"  // a cell to add a sack on
	pickUnit  pickKind = "unit"  // a cell to add a unit on
	pickDrop  pickKind = "drop"  // the cell the party drops on
	pickGoal  pickKind = "goal"  // a unit or structure for the selected script arm
)

// promptKind is the text prompt in the rail.
type promptKind uint8

const (
	promptNone    promptKind = iota
	promptOpen               // an ALM path to open
	promptSave               // a new ALM path for Save As
	promptField              // a value for the selected record's field
	promptNewSack            // the item code of a sack added on the picked cell
	promptNewUnit            // the class key of a unit added on the picked cell
	promptRectangle
	promptWorkspace
)

// editorPrompt is a prompt and the data it owns: the typed text, the field it
// cycles to and, for a new record, the cell the click picked.
type editorPrompt struct {
	kind  promptKind
	cell  image.Point
	field int
	path  string
	scope string
}

func (p editorPrompt) open() bool { return p.kind != promptNone }
func (p editorPrompt) adds() bool { return p.kind == promptNewSack || p.kind == promptNewUnit }
func (p editorPrompt) fields() bool {
	return p.kind == promptField || p.adds() || p.kind == promptWorkspace
}

// editorMode is every layer the window can be in.
type editorMode struct {
	catalog bool
	prompt  editorPrompt
	pick    pickKind
	brush   brushState
	report  reportState
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
}

// resetList scrolls the record list to offset and drops the wheel's remainder.
func (e *MapEditor) resetList(offset int) {
	e.offset, e.listWheel = offset, 0
}

// enterCatalog shows the catalogue at its first row.
func (e *MapEditor) enterCatalog() {
	e.catalog = true
	e.resetList(0)
}

// armPick arms the next canvas click to pick kind.
func (e *MapEditor) armPick(kind pickKind) {
	e.pick, e.message = kind, ""
}

// openPrompt replaces the prompt.
func (e *MapEditor) openPrompt(p editorPrompt) {
	e.prompt, e.message = p, ""
}

// toggleBrush switches the brush mode on, or off when it already is mode.
func (e *MapEditor) toggleBrush(mode brushMode) {
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
	if !e.report.open {
		e.report.offset = e.offset
	}
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
	e.offset = e.report.offset
	e.report = reportState{}
	e.listWheel = 0
}

// blocksCanvas reports that the map takes no pointer: the catalogue or a
// prompt covers it.
func (m *editorMode) blocksCanvas() bool { return m.catalog || m.prompt.open() }
