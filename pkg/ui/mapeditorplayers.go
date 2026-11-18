package ui

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"sync"
)

type InspectionTextFields interface {
	SetTextField(kind RecordKind, index int, key, value string) (*InspectionDocument, error)
}

const editorPlayersFilter = 8

func (e *MapEditor) keyPlayers(string) error {
	if e.doc == nil {
		return nil
	}
	if e.barWindow() == "players" {
		e.closeBarWindow()
		return nil
	}
	e.openRecords(editorPlayersFilter)
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
	e.selectUnfollowed(rows[0])
	return nil
}

// keyAdd adds a player on the Players list and does nothing on any other.
func (e *MapEditor) keyAdd(string) error {
	if !e.listView() || e.brushActive() {
		return nil
	}
	if row, ok := e.listAdd(); ok {
		e.addNow(row.Kind)
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

// relationMatrix is the Players page's relation matrix as one document's
// host gives it: the players in index order, their list labels, and the
// relation field each row holds toward each column.
type relationMatrix struct {
	doc     *InspectionDocument
	players []int
	labels  []string
	fields  [][]InspectionField
	// names and control are each player's name and control fields; a
	// player the host offers no such field for holds the zero field.
	names, control []InspectionField
	// records are the players' document indices.
	records []int
	// top is the first matrix row shown while the window holds fewer rows
	// than the matrix has; an edit's new document keeps it.
	top int
}

// relationValue is a value's short name and what its (?) says.
type relationValue struct {
	nameWord string
	helpWord string
}

// relationValues name the values the claims give a meaning, by value: bit 0
// is hostile, an attack sets it while the low two bits are clear, and bit 1
// blocks that (AI-DIPLO-004); the shipped maps hold 0, 1 and 2
// (AI-DIPLO-005).
var relationValues = []relationValue{
	{nameWord: "editor.relation.name.neutral", helpWord: "editor.relation.neutral.help"},
	{nameWord: "editor.relation.name.hostile", helpWord: "editor.relation.hostile.help"},
	{nameWord: "editor.relation.name.ally", helpWord: "editor.relation.ally.help"},
}

// relationName is value v's name, its number for a value the claims do not
// name.
func (e *MapEditor) relationName(v int64) string {
	if v >= 0 && v < int64(len(relationValues)) {
		return e.word(relationValues[v].nameWord)
	}
	return strconv.FormatInt(v, 10)
}

// relationMatrixOf reads the matrix of the open document once.
func (e *MapEditor) relationMatrixOf() *relationMatrix {
	m := &e.relations
	if e.doc == nil || m.doc == e.doc {
		return m
	}
	*m = relationMatrix{doc: e.doc, top: m.top}
	for i, r := range e.doc.Records {
		if r.Kind == KindPlayer {
			m.players, m.labels, m.records = append(m.players, r.Index), append(m.labels, r.Label), append(m.records, i)
		}
	}
	n := len(m.players)
	for _, p := range m.players {
		row := make([]InspectionField, n)
		var name, control InspectionField
		for _, f := range e.doc.Edits.Fields(KindPlayer, p) {
			switch {
			case f.Toward >= 1 && f.Toward <= n && !f.Text:
				row[f.Toward-1] = f
			case f.Key == playerNameField && f.Text:
				name = f
			case f.Key == playerControlField && !f.Text:
				control = f
			}
		}
		m.fields, m.names, m.control = append(m.fields, row), append(m.names, name), append(m.control, control)
	}
	return m
}

// playerNameField, playerColourField and playerControlField are the keys
// of a player's name, its colour slot and its control, 0 human and 1
// computer.
const (
	playerNameField    = "name"
	playerColourField  = "colour"
	playerControlField = "control"
)

// playerHeadRows are the controls left of each matrix row, as rows that
// stand in the row's head box: the human/computer switch, the player's
// number, its name field and the cross that deletes it. placeHeadColumn
// stands them in the grid's head column at their natural widths.
func (e *MapEditor) playerHeadRows(m *relationMatrix, g *popupGrid) []popupRow {
	focus, typed := e.playerTyping()
	var rows []popupRow
	var cols []int
	add := func(r popupRow, col int) {
		rows, cols = append(rows, r), append(cols, col)
	}
	for i := range m.players {
		head := i + 1
		if f := m.control[i]; f.Key != "" {
			add(popupRow{Kind: popupSwitch, Value: e.controlName(f.Value), HeadOf: head}, 0)
		}
		add(popupRow{Kind: popupText, Value: strconv.Itoa(m.players[i] + 1), HeadOf: head}, 1)
		if f := m.names[i]; f.Key != "" {
			name := popupRow{Kind: popupEdit, Value: e.editorHostText(f.TextValue), HeadOf: head}
			if focus == i {
				name.Value, name.Focus = e.editorHostText(typed), true
				name.Edit = e.inputView(editorTextSpan{0, len([]rune(typed))}, false)
			}
			add(name, 2)
		}
		// The first player is the one the map cannot lose.
		add(popupRow{Kind: popupText, Buttons: []string{e.word("editor.players.delete")}, Disabled: []bool{m.players[i] == 0}, Tip: e.word("editor.players.delete.tip"), HeadOf: head}, 3)
	}
	placeHeadColumn(g, rows, cols, e.bodyFont())
	return rows
}

// playerFieldRow is field f's row in a player's own window, which only
// reads: the name, colour and control under their words, the control's
// value named, and a relation under the other player's label with its
// value's name and picture.
func (e *MapEditor) playerFieldRow(f InspectionField) popupRow {
	row := popupRow{Kind: popupText, Label: e.editorHostText(f.Label), Value: fieldInput(f)}
	if len(f.Choices) > 0 {
		row.Value = choiceText(f, row.Value)
	} else if f.Text {
		row.Value = e.editorHostText(row.Value)
	}
	switch {
	case f.Key == playerNameField && f.Text:
		row.Label = e.word("editor.add.name")
	case f.Key == playerColourField && !f.Text:
		row.Label = e.word("editor.player.colour")
	case f.Key == playerControlField && !f.Text:
		row.Label, row.Value = e.word("editor.add.control"), e.controlName(f.Value)
	case f.Toward > 0 && !f.Text:
		m := e.relationMatrixOf()
		if k := slices.Index(m.players, f.Toward-1); k >= 0 {
			row.Label = m.labels[k]
		}
		row.Value = e.relationName(f.Value)
		if icon := relationIcon(f.Value); icon != nil {
			row.Icons = []*image.RGBA{icon}
		}
	}
	return row
}

// playerTitle is player record r's window title: its number and name.
func (e *MapEditor) playerTitle(r InspectionRecord) string {
	name := r.Label
	m := e.relationMatrixOf()
	if k := slices.Index(m.players, r.Index); k >= 0 && m.names[k].Key != "" {
		name = e.editorHostText(m.names[k].TextValue)
	}
	return fmt.Sprintf(e.word("editor.player.title"), r.Index+1, name)
}

// controlWords name a player's control values: 0 human, 1 computer.
var controlWords = []string{"editor.add.control.human", "editor.add.control.computer"}

// controlName is control value v's name; any value but 0 is the computer.
func (e *MapEditor) controlName(v int64) string {
	if v == 0 {
		return e.word(controlWords[0])
	}
	return e.word(controlWords[1])
}

// playerTyping is the matrix row whose name the keyboard edits and the text
// typed so far; -1 while none is.
func (e *MapEditor) playerTyping() (int, string) {
	kind, index := e.selectedRecord()
	if kind != KindPlayer || !e.objectTyping() || e.obj.fields[e.obj.focus].Key != playerNameField {
		return -1, ""
	}
	m := e.relationMatrixOf()
	for i, p := range m.players {
		if p == index {
			return i, e.obj.buf[e.obj.focus]
		}
	}
	return -1, ""
}

// playerHeadPress is a press on a matrix row's own control: the switch
// sets the other control as one edit, the name field selects the player
// and takes the keyboard, Enter writing the name as one edit, and the
// cross asks before the delete.
func (e *MapEditor) playerHeadPress(row popupRow) {
	m := e.relationMatrixOf()
	i := row.HeadOf - 1
	if i < 0 || i >= len(m.players) {
		return
	}
	switch {
	case row.Kind == popupSwitch:
		f := m.control[i]
		if _, err := e.commandSetField(fieldEdit{Kind: KindPlayer, Index: m.players[i], Value: InspectionFieldValue{Key: f.Key, Value: 1 - min(1, max(0, f.Value))}}); err != nil {
			e.message = err.Error()
		}
	case row.Kind == popupEdit:
		if focus, _ := e.playerTyping(); focus != i {
			e.typePlayerName(m.players[i])
		}
	case len(row.Buttons) > 0:
		e.askPlayerDelete(m.players[i])
	}
}

// typePlayerName selects player index and gives the keyboard to its name
// field in the Players window; Enter writes the name as one edit.
func (e *MapEditor) typePlayerName(index int) {
	m := e.relationMatrixOf()
	for i, p := range m.players {
		if p != index {
			continue
		}
		e.Select(m.records[i], false)
		e.loadObject()
		for k, f := range e.obj.fields {
			if f.Key == playerNameField {
				e.focusObjectField(k)
			}
		}
	}
}

// askPlayerDelete opens the question before player index is deleted.
func (e *MapEditor) askPlayerDelete(index int) {
	e.leave(leaveDrag | leavePick | leavePrompt)
	e.openPrompt(editorPrompt{kind: promptDelete, del: RecordRef{Kind: KindPlayer, Index: index}})
}

// playerDeleteSpec is the question: the player's number and name as its
// note, and behind the title's (?) that the later players' numbers drop by
// one with the records naming them.
func (e *MapEditor) playerDeleteSpec() popupSpec {
	m, index, name := e.relationMatrixOf(), e.prompt.del.Index, ""
	for i, p := range m.players {
		if p == index {
			name = e.editorHostText(m.names[i].TextValue)
		}
	}
	note := fmt.Sprintf(e.word("editor.players.deletetext"), index+1, name)
	return e.questionSpec(question{title: e.word("editor.players.deletetitle"), help: e.word("editor.players.deletetitle.help"),
		notes: []string{note}, buttons: []string{e.word("editor.players.deletego"), e.word("editor.button.cancel")}})
}

// playerDeletePress is a press on the question.
func (e *MapEditor) playerDeletePress(p image.Point) {
	s, _ := e.promptSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); {
	case part == popupClose || part == popupButton && n == 1:
		e.cancelPopup()
	case part == popupButton && n == 0:
		e.playerDeleteGo()
	}
}

// playerDeleteGo deletes the player the question holds, as one edit.
func (e *MapEditor) playerDeleteGo() {
	ref := e.prompt.del
	e.leave(leavePrompt)
	_ = e.deleteSelected(ref.Kind, ref.Index)
}

// relationGridSpec is the Players page's matrix as popup data: row i,
// column j is player i's relation toward player j, a picture for a value
// the claims name and its number for any other; the diagonal and a cell
// the host offers no field for are inert. Each column is headed upright
// and whole with its row's head; a cell's hover line names both players
// and the value. The cells are the palette's side. The rows are the legend,
// a row a named value with its picture and its (?).
func (e *MapEditor) relationGridSpec() (*popupGrid, []popupRow, bool) {
	if e.listKind() != "records" || e.filter != editorPlayersFilter {
		return nil, nil, false
	}
	m := e.relationMatrixOf()
	n, offered := len(m.players), false
	g := &popupGrid{Cols: n, RoomTop: true}
	g.HeadH = g.side()
	for i := range n {
		// A row's head box holds the player's own controls, not its label.
		g.RowHeads = append(g.RowHeads, "")
		g.ColHeads = append(g.ColHeads, m.labels[i])
		if body := e.bodyFont(); body != nil {
			g.HeadH = max(g.HeadH, body.Advance(m.labels[i])+8)
		}
		for j := range n {
			f := m.fields[i][j]
			cell := popupGridCell{Inert: i == j || f.Key == ""}
			if !cell.Inert {
				offered = true
				if cell.Icon = relationIcon(f.Value); cell.Icon == nil {
					cell.Label = strconv.FormatInt(f.Value, 10)
				}
				cell.Tip = fmt.Sprintf(e.word("editor.relation.cell"), m.labels[i], m.labels[j], e.relationName(f.Value))
			}
			g.Cells = append(g.Cells, cell)
		}
	}
	var rows []popupRow
	if offered {
		for v, r := range relationValues {
			rows = append(rows, popupRow{Kind: popupText, Label: e.word(r.nameWord), Value: strconv.Itoa(v), Icons: []*image.RGBA{relationIcon(int64(v))}, Help: e.word(r.helpWord)})
		}
	} else if n > 0 {
		// No relation to set (one player): the rows stay for the players'
		// own controls, with no column heads and no legend.
		g.ColHeads, g.HeadH = nil, 0
	} else {
		return nil, nil, false
	}
	if focus, _ := e.playerTyping(); focus >= 0 && e.obj.note != "" {
		// Why the typed name was not written; the name stays typed.
		rows = append(rows, popupRow{Kind: popupText, Label: e.obj.note})
	}
	return g, append(rows, e.playerHeadRows(m, g)...), true
}

// relationFit gives the Players window s its natural width through listFit: what the
// matrix's head column and cells ask for, the legend's label column with
// its value and picture, or the button row, whichever is widest. The label
// column is as wide as the widest legend label and its (?). The matrix
// shows as many rows as the window's bounds hold, from the kept top row;
// the wheel scrolls the rest (relationGridWheel).
func (e *MapEditor) relationFit(s *popupSpec) {
	e.listFit(s, listFitIn{
		Counts: func(r popupRow) bool { return r.HeadOf == 0 && len(r.Icons) > 0 },
		Value: func(r popupRow, m listFitMeasure) int {
			return m.Advance(r.Value) + 8 + (m.Pitch+2)*len(r.Icons)
		},
	})
	g := s.Grid
	m := e.relationMatrixOf()
	g.Top = m.top
	shown := layoutPopup(*s, e.font, e.bodyFont()).GridRows
	m.top = max(0, min(m.top, len(m.players)-shown))
	g.Top = m.top
}

// relationPress is a press on matrix cell n: it sets the next relation
// value of that row toward that column as one edit.
func (e *MapEditor) relationPress(n int) {
	m := e.relationMatrixOf()
	k := len(m.players)
	if k == 0 || n < 0 || n >= k*k || n/k == n%k {
		return
	}
	f := m.fields[n/k][n%k]
	if f.Key == "" {
		return
	}
	if _, err := e.commandSetField(fieldEdit{Kind: KindPlayer, Index: m.players[n/k], Value: InspectionFieldValue{Key: f.Key, Value: relationNext(f.Value)}}); err != nil {
		e.message = err.Error()
	}
}

// relationNext is the value a press on a cell holding v sets: the named
// values in turn, and the first after any other.
func relationNext(v int64) int64 {
	if v >= 0 && v < int64(len(relationValues)-1) {
		return v + 1
	}
	return 0
}

// relationIcons are the matrix's pictures, drawn here: a grey ring for
// neutral, a red cross for hostile and a green disc for ally.
var relationIcons = sync.OnceValue(func() []*image.RGBA {
	var out []*image.RGBA
	for v, ink := range relationInks {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y := range 16 {
			for x := range 16 {
				dx, dy := float64(x)-7.5, float64(y)-7.5
				d := math.Hypot(dx, dy)
				on := []bool{d >= 4.5 && d <= 6.5, d <= 7 && (math.Abs(dx-dy) <= 1.5 || math.Abs(dx+dy) <= 1.5), d <= 6.5}[v]
				if on {
					img.SetRGBA(x, y, ink)
				}
			}
		}
		out = append(out, img)
	}
	return out
})

// relationIcon is value v's picture, nil for a value the claims do not name.
func relationIcon(v int64) *image.RGBA {
	if pics := relationIcons(); v >= 0 && v < int64(len(pics)) {
		return pics[v]
	}
	return nil
}
