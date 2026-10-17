package ui

type InspectionTextFields interface {
	SetTextField(kind string, index int, key, value string) (*InspectionDocument, error)
}

const editorPlayersFilter = 8

func (e *MapEditor) keyPlayers(string) error {
	if e.doc == nil {
		return nil
	}
	wasPlayers := e.listView() && !e.brushActive() && e.filter == editorPlayersFilter
	e.leave(leaveDrag | leavePick | leaveReport | leaveCatalog | leaveBrush)
	e.message = ""
	e.filter, e.listShown = 0, !wasPlayers
	if !wasPlayers {
		e.filter = editorPlayersFilter
	}
	e.resetList(0)
	rows := e.rows()
	if len(rows) == 0 {
		e.selected = -1
		return nil
	}
	for _, row := range rows {
		if row == e.selected {
			e.Select(row, false)
			return nil
		}
	}
	e.Select(rows[0], false)
	return nil
}

// keyAdd opens the Add window on the kind the shown list holds.
func (e *MapEditor) keyAdd(string) error {
	if !e.listView() || e.brushActive() {
		return nil
	}
	if row, ok := addKindFor(e.filter); ok {
		e.openAdd(row.Kind)
	}
	return nil
}

func (e *MapEditor) stepPlayer(delta int) {
	rows := e.rows()
	if len(rows) == 0 {
		return
	}
	at := 0
	for i, row := range rows {
		if row == e.selected {
			at = min(len(rows)-1, max(0, i+delta))
			break
		}
	}
	e.Select(rows[at], false)
}
