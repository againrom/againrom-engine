package ui

import "fmt"

type WorkspaceConfirmation struct{ Action, Message, Confirm string }

func (e *WorkspaceConfirmation) Error() string { return e.Message }

type WorkspaceLeave struct{ Folder string }

func (e *WorkspaceLeave) Error() string { return "save, discard or cancel before opening another mod" }

type InspectionWorkspace interface {
	WorkspaceFields(scope string) []InspectionField
	SetWorkspaceField(scope, key, value string) (*InspectionDocument, error)
}

// ModOverviewSection is one heading of the overview page of the mod W has
// open, with its lines.
type ModOverviewSection struct {
	Heading string
	Lines   []string
}

// ModOverviewPage is that page: its sections, and whether W shows it first,
// for a mod no mission of which opens.
type ModOverviewPage struct {
	Sections []ModOverviewSection
	First    bool
}

// ModOverview is a document host that gives W the overview page of the mod
// open in it; a page with no section is none.
type ModOverview interface {
	ModOverview() ModOverviewPage
}

// ModWindowHost is what W works through with no map open: the mod's fields
// and overview, and the open of a mod. A map's own host answers the same.
type ModWindowHost interface {
	InspectionWorkspace
	ModOverview
}

// SetModHost gives the editor the host W works through with no map open.
func (e *MapEditor) SetModHost(host func() ModWindowHost) { e.modHost = host }

// modEdits is the host W works through: the open map's, else with no map
// the mod host's.
func (e *MapEditor) modEdits() any {
	if e.doc != nil {
		return e.doc.Edits
	}
	if e.modHost != nil {
		return e.modHost()
	}
	return nil
}

// modOverCatalog reports the mod window open over the map list, the state
// of W with no map open.
func (e *MapEditor) modOverCatalog() bool { return e.modWindow && e.catalog && e.doc == nil }

// modOverview is the overview page of the mod W has open, false with none.
func (e *MapEditor) modOverview() (ModOverviewPage, bool) {
	host, ok := e.modEdits().(ModOverview)
	if !ok {
		return ModOverviewPage{}, false
	}
	p := host.ModOverview()
	return p, len(p.Sections) > 0
}

// showModPage shows W's page n, 1 the overview: its rows from the top, the
// keyboard on its first control.
func (e *MapEditor) showModPage(n int) {
	e.modPage, e.objectTop = n, 0
	e.obj.focus = 0
	if n == 1 {
		e.obj.focus = len(e.obj.fields)
	}
}

// openModPage shows the page W opens a mod on: the overview when the host
// says so, else the mod's fields.
func (e *MapEditor) openModPage() {
	if p, ok := e.modOverview(); ok && p.First {
		e.showModPage(1)
		return
	}
	e.modPage = 0
}

func (e *MapEditor) workspace() (InspectionWorkspace, bool) {
	if e.doc == nil && e.modWindow {
		host, ok := e.modEdits().(InspectionWorkspace)
		return host, ok
	}
	if !e.editable() {
		return nil, false
	}
	host, ok := e.doc.Edits.(InspectionWorkspace)
	return host, ok
}

func (e *MapEditor) keyWorkspace(key string) error {
	host, ok := e.workspace()
	noMap := e.doc == nil && key == "package"
	if noMap {
		host, ok = e.modEdits().(InspectionWorkspace)
	}
	if !ok || e.catalog && !noMap {
		e.message = "Open a map to edit its metadata, pages or mod package."
		return nil
	}
	fields := host.WorkspaceFields(key)
	if len(fields) == 0 {
		e.message = "This map has no editable workspace fields."
		return nil
	}
	e.leave(leaveDrag | leavePick | leaveReport)
	if key == "package" {
		e.message = ""
		e.modWindow = true
		e.modConfirm = ""
		e.objectHidden = false
		e.dropObject()
		e.loadObject()
		e.objectSpec()
		e.FocusModField("folder")
		e.openModPage()
		return nil
	}
	e.openPrompt(editorPrompt{kind: promptWorkspace, scope: key, path: fieldInput(fields[0])})
	return nil
}

func (e *MapEditor) setWorkspaceField(value string) error {
	_, ok := e.workspace()
	fields := e.promptFields()
	if !ok || e.prompt.field >= len(fields) {
		return fmt.Errorf("workspace field is unavailable")
	}
	if _, err := e.commandSetMapField(mapField{Scope: e.prompt.scope, Key: fields[e.prompt.field].Key, Value: value}); err != nil {
		e.message = err.Error()
		return err
	}
	if e.prompt.scope == "pages" && fields[e.prompt.field].Key == "page" {
		fields = e.promptFields()
		if len(fields) > 1 {
			e.prompt.field, e.prompt.path = 1, fieldInput(fields[1])
			e.input = editorInputState{}
			return nil
		}
	}
	e.leave(leavePrompt)
	e.cancelRestore = nil
	return nil
}

func (e *MapEditor) multilinePrompt() bool {
	if e.prompt.kind != promptWorkspace {
		return false
	}
	fields := e.promptFields()
	return e.prompt.field < len(fields) && (fields[e.prompt.field].Key == "body" || e.prompt.scope == "metadata" && fields[e.prompt.field].Key == "description")
}

func (e *MapEditor) keyNewline(string) error {
	if e.multilinePrompt() || e.objectTyping() && e.messageTextField(e.obj.focus) {
		e.TextInput("\n")
	}
	return nil
}

func (e *MapEditor) keyClear(string) error {
	if e.prompt.open() || (e.messagePage != 0 || e.modWindow) && e.objectTyping() {
		if m, _ := e.inputModel(); m != nil {
			m.SelectAll()
			m.Replace("")
			e.input.edited()
			*e.input.target = m.Text
		}
	}
	return nil
}

func (e *MapEditor) keyHelp(string) error {
	title := e.word("editor.keys.title")
	if e.report.open && e.report.note == title {
		e.leave(leaveReport)
		return nil
	}
	var rows []InspectionFinding
	seen := map[string]bool{}
	for _, line := range e.allHelpLines() {
		if seen[line] {
			continue
		}
		seen[line] = true
		rows = append(rows, InspectionFinding{Rule: keyListRule, Text: line})
	}
	rows = append(rows, InspectionFinding{Rule: keyListRule, Text: e.saveHelp()})
	e.leave(leaveDrag | leavePick)
	e.openReport(rows, title, nil)
	return nil
}
