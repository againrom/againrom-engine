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
	if e.prompt.open() || e.messagePage != 0 && e.objectTyping() {
		if m, _ := e.inputModel(); m != nil {
			m.selectAll()
			m.replace("")
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
