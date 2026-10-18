package ui

import "fmt"

type InspectionWorkspace interface {
	WorkspaceFields(scope string) []InspectionField
	SetWorkspaceField(scope, key, value string) (*InspectionDocument, error)
}

func (e *MapEditor) workspace() (InspectionWorkspace, bool) {
	if !e.editable() {
		return nil, false
	}
	host, ok := e.doc.Edits.(InspectionWorkspace)
	return host, ok
}

func (e *MapEditor) keyWorkspace(key string) error {
	host, ok := e.workspace()
	if !ok || e.catalog {
		e.message = "Open a map to edit its metadata, pages or mod package."
		return nil
	}
	fields := host.WorkspaceFields(key)
	if len(fields) == 0 {
		e.message = "This map has no editable workspace fields."
		return nil
	}
	e.leave(leaveDrag | leavePick | leaveReport)
	e.openPrompt(editorPrompt{kind: promptWorkspace, scope: key, path: fieldInput(fields[0])})
	return nil
}

func (e *MapEditor) setWorkspaceField(value string) error {
	host, ok := e.workspace()
	fields := e.promptFields()
	if !ok || e.prompt.field >= len(fields) {
		return fmt.Errorf("workspace field is unavailable")
	}
	next, err := host.SetWorkspaceField(e.prompt.scope, fields[e.prompt.field].Key, value)
	if err := e.adoptEdit(next, err); err != nil {
		return err
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
	if e.multilinePrompt() || e.messagePage != 0 && e.objectTyping() && e.obj.focus == 0 {
		e.TextInput("\n")
	}
	return nil
}

func (e *MapEditor) keyClear(string) error {
	if e.messagePage != 0 && e.objectTyping() {
		e.obj.buf[e.obj.focus] = ""
	}
	if e.prompt.open() {
		e.prompt.path = ""
	}
	return nil
}

func (e *MapEditor) keyHelp(string) error {
	if e.report.open && e.report.note == "Map editor keys" {
		e.leave(leaveReport)
		return nil
	}
	var rows []InspectionFinding
	seen := map[string]bool{}
	for _, line := range EditorHelpLines() {
		if seen[line] {
			continue
		}
		seen[line] = true
		rows = append(rows, InspectionFinding{Rule: "keys", Text: line})
	}
	rows = append(rows, InspectionFinding{Rule: "keys", Text: e.saveHelp()})
	e.leave(leaveDrag | leavePick)
	e.openReport(rows, "Map editor keys", nil)
	return nil
}
