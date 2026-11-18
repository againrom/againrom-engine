package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// playerNameInput gives the keyboard to the selected player's name field
// in the Players window and replaces its text with value.
func playerNameInput(t *testing.T, e *MapEditor, value string) {
	t.Helper()
	if err := e.Key("field"); err != nil {
		t.Fatal(err)
	}
	if focus, _ := e.playerTyping(); focus < 0 || e.prompt.open() {
		t.Fatal("E gave the keyboard to no player name", focus, e.prompt.kind)
	}
	if m, _ := e.inputModel(); m != nil {
		m.SelectAll()
	}
	e.TextInput(value)
}

func TestMapEditorPlayersUseRecordFieldsAndOneEditPerKey(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	if rows := e.rows(); len(rows) != 2 || e.doc.Records[e.selected].Kind != "Player" || e.doc.Records[e.selected].Index != 0 {
		t.Fatal("players key did not select the first roster row", rows, e.selected)
	}
	_ = e.Key("down")
	if e.doc.Records[e.selected].Index != 1 {
		t.Fatal("Down did not select the next player")
	}
	_ = e.Key("up")
	if e.doc.Records[e.selected].Index != 0 {
		t.Fatal("Up did not select the first player")
	}
	if err := e.Key("player"); err != nil {
		t.Fatal(err)
	}
	if len(calls.lines) != 1 || !strings.HasPrefix(calls.lines[0], "AddRecord Player") || e.prompt.open() || len(e.rows()) != 3 || e.doc.Records[e.selected].Index != 2 {
		t.Fatal("add did not make one edit and select the appended player", calls.lines, e.selected, e.prompt.kind)
	}
	before := len(calls.lines)
	playerNameInput(t, e, "  Northern Clans  ")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(calls.lines) != before+1 || calls.lines[before] != "SetTextField Player 2 name \"  Northern Clans  \"" || e.doc.Records[e.selected].Index != 2 {
		t.Fatal("the name did not write its table key exactly once", calls.lines[before:])
	}
	before = len(calls.lines)
	if err := e.Key("clone"); err != nil || e.pick != pickNone || len(calls.lines) != before {
		t.Fatal("a player was armed for cloning", err, e.pick, calls.lines)
	}
	if err := e.Key("delete"); err != nil || e.prompt.kind != promptDelete || len(calls.lines) != before {
		t.Fatal("Del on a player did not ask first", err, e.prompt.kind, calls.lines[before:])
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(calls.lines) != before+1 || calls.lines[before] != "Delete Player 2" || e.prompt.open() || len(e.rows()) != 2 || e.doc.Records[e.selected].Index != 1 {
		t.Fatal("delete did not make one edit and select the surviving neighbour", calls.lines, e.selected)
	}
	_ = e.Key("players")
	if e.filter != 0 {
		t.Fatal("players key did not return to all records")
	}
}

// The Players page shows the relations as a players x players matrix: row
// i, column j a picture of player i's relation toward player j, the
// diagonal inert. A click on a cell sets its next value as one host edit,
// the SetField the MCP set_field tool writes; a value the claims do not name
// shows as its number and a click starts it over at 0.
func TestPlayersPageMatrixCyclesARelationPerClick(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	// The chosen player's window floats clear of the palette's matrix.
	e.places[popObject].at = image.Pt(0, 0)
	s, ok := e.listSpec()
	g := s.Grid
	if !ok || g == nil || g.Cols != 2 || len(g.Cells) != 4 || !g.Cells[0].Inert || !g.Cells[3].Inert || g.Cells[1].Inert || g.Cells[1].Icon != relationIcon(1) ||
		!slices.Equal(g.ColHeads, []string{"Self", "Enemy"}) || !slices.Equal(g.RowHeads, []string{"", ""}) {
		t.Fatalf("the Players page holds no 2x2 relation matrix: %+v", g)
	}
	legend := 0
	for _, row := range s.Rows {
		for v, r := range relationValues {
			if row.Label == e.word(r.nameWord) && row.Help == e.word(r.helpWord) && len(row.Icons) == 1 && row.Icons[0] == relationIcon(int64(v)) {
				legend++
			}
		}
	}
	if legend != 3 {
		t.Fatalf("the matrix's legend names %d of the 3 values: %+v", legend, s.Rows)
	}
	cell, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 1})
	if !ok {
		t.Fatal("matrix cell 0,1 is not shown")
	}
	for _, want := range []int64{2, 0, 1} {
		before := len(calls.lines)
		chromeClick(e, center(cell))
		if len(calls.lines) != before+1 || calls.lines[before] != fmt.Sprintf("SetField Player 0 relation:1 %d", want) {
			t.Fatalf("a click on cell 0,1 made %q, want one SetField to %d", calls.lines[before:], want)
		}
		if s, _ := e.listSpec(); s.Grid == nil || s.Grid.Cells[1].Icon != relationIcon(want) {
			t.Fatalf("cell 0,1 does not show %d after the click", want)
		}
	}
	diagonal, _ := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 3})
	before := len(calls.lines)
	chromeClick(e, center(diagonal))
	if len(calls.lines) != before {
		t.Fatal("a click on the diagonal made an edit", calls.lines[before:])
	}
	e.doc.Edits.(*modesFull).relations[[2]int{1, 0}] = 5
	e.relations = relationMatrix{}
	if s, _ := e.listSpec(); s.Grid.Cells[2].Icon != nil || s.Grid.Cells[2].Label != "5" {
		t.Fatalf("an unnamed value shows as %+v", s.Grid.Cells[2])
	}
	other, _ := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 2})
	chromeClick(e, center(other))
	if calls.lines[len(calls.lines)-1] != "SetField Player 1 relation:0 0" {
		t.Fatal("a click on an unnamed value did not set 0", calls.lines)
	}
}

type refusedPlayerFields struct {
	InspectionEdits
	problem error
}

func (p refusedPlayerFields) SetTextField(RecordKind, int, string, string) (*InspectionDocument, error) {
	return nil, p.problem
}

func (p refusedPlayerFields) Delete(RecordKind, int) (*InspectionDocument, error) {
	return nil, p.problem
}

func TestMapEditorPlayerPromptKeepsTextAndErrors(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	_ = e.Key("down")
	name := strings.Repeat("Я", 32)
	playerNameInput(t, e, name)
	if focus, typed := e.playerTyping(); focus != 1 || typed != name {
		t.Fatal("the name field lost a full-width UTF-8 player name", focus, typed)
	}
	problem := fmt.Errorf("player name exceeds 32 encoded bytes")
	e.doc.Edits = refusedPlayerFields{InspectionEdits: e.doc.Edits, problem: problem}
	selected := e.selected
	_ = e.Key("enter")
	if focus, typed := e.playerTyping(); focus != 1 || typed != name || e.selected != selected || len(calls.lines) != 0 {
		t.Fatal("text refusal lost the typed name", focus, typed, calls.lines)
	}
	if !slices.Contains(listRowLabels(t, e), problem.Error()) {
		t.Fatal("the Players window does not say why the name was refused")
	}
	_ = e.Key("escape")
	problem = fmt.Errorf("Player 2 still owns Unit #0")
	e.doc.Edits = refusedPlayerFields{InspectionEdits: e.doc.Edits, problem: problem}
	_ = e.Key("delete")
	if e.prompt.kind != promptDelete {
		t.Fatal("Del on a player did not ask", e.prompt.kind)
	}
	if _ = e.Key("enter"); e.Message() != problem.Error() || e.selected != selected || len(calls.lines) != 0 {
		t.Fatal("delete refusal did not retain the referring-player message", e.Message())
	}
}

// listRowLabels are the labels of the shown list window's rows.
func listRowLabels(t *testing.T, e *MapEditor) []string {
	t.Helper()
	s, ok := e.listSpec()
	if !ok {
		t.Fatal("no list window is shown")
	}
	var out []string
	for _, row := range s.Rows {
		out = append(out, row.Label)
	}
	return out
}

func TestMapEditorPlayersNeedOptionalAddAndTextHosts(t *testing.T) {
	e, calls := newModesEditor(t, "plain")
	_ = e.Key("players")
	_ = e.Key("player")
	if len(e.rows()) != 2 || len(calls.lines) != 0 {
		t.Fatal("a host without the Add window received an edit", calls.lines)
	}
	playerNameInput(t, e, "Rival")
	_ = e.Key("enter")
	if focus, typed := e.playerTyping(); focus != 0 || typed != "Rival" || len(calls.lines) != 0 {
		t.Fatal("text editing silently succeeded without its optional host", focus, typed, calls.lines)
	}
	if !slices.Contains(listRowLabels(t, e), "This host cannot edit text fields.") {
		t.Fatal("the Players window does not say the host cannot edit text")
	}
}

func TestMapEditorPlayerSelectionCannotFinishAClonePick(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Select(0, false)
	_ = e.Key("clone")
	for i, r := range e.doc.Records {
		if r.Kind == "Player" {
			e.Select(i, false)
			break
		}
	}
	e.pickAt(e.CellPoint(image.Pt(25, 20)))
	if len(calls.lines) != 0 || e.pick != pickNone || e.Message() != "Clone needs a selected unit, structure or sack." {
		t.Fatal("an armed spatial clone accepted a subsequently selected player", calls.lines, e.pick, e.Message())
	}
}
