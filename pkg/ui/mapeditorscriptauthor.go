package ui

type InspectionScriptAuthor interface {
	AddScript(kind string) (*InspectionDocument, error)
}

const editorScriptsFilter = 9

func scriptRecord(kind string) bool {
	return kind == "Action" || kind == "Condition" || kind == "Trigger"
}

func (e *MapEditor) keyScripts(string) error {
	if !e.editable() {
		return nil
	}
	e.leave(leaveDrag | leavePick | leaveReport | leaveCatalog | leaveBrush)
	e.filter, e.listShown = editorScriptsFilter, true
	e.resetList(0)
	e.selected = -1
	if rows := e.rows(); len(rows) > 0 {
		e.Select(rows[0], false)
	}
	e.message = ""
	return nil
}

func (e *MapEditor) addScriptKey() bool {
	if !e.editable() || !e.listView() || e.filter != editorScriptsFilter {
		return false
	}
	if _, ok := e.doc.Edits.(InspectionScriptAuthor); !ok {
		return false
	}
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptNewScript, path: "0"})
	return true
}
