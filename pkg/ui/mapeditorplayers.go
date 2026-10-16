package ui

type InspectionPlayers interface {
	AddPlayer() (*InspectionDocument, error)
}

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
	e.filter = 0
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

func (e *MapEditor) keyPlayer(string) error {
	if e.addScriptKey() {
		return nil
	}
	if !e.editable() || !e.listView() || e.brushActive() || e.filter != editorPlayersFilter {
		return nil
	}
	p, ok := e.doc.Edits.(InspectionPlayers)
	if !ok {
		return nil
	}
	e.leave(leaveDrag | leavePick | leaveBrush)
	next, err := p.AddPlayer()
	if err := e.adoptEdit(next, err); err != nil {
		return err
	}
	if next != nil {
		e.selectNewest("Player")
		e.Select(e.selected, false)
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
