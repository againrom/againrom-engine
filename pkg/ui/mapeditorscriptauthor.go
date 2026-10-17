package ui

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
