package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

func playerPromptInput(t *testing.T, e *MapEditor, value string) {
	t.Helper()
	for e.prompt.path != "" {
		if err := e.Key("backspace"); err != nil {
			t.Fatal(err)
		}
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
	for _, key := range []string{"player", "enter", "enter", "enter"} {
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls.lines) != 1 || !strings.HasPrefix(calls.lines[0], "AddRecord Player") || len(e.rows()) != 3 || e.doc.Records[e.selected].Index != 2 {
		t.Fatal("add did not make one edit and select the appended player", calls.lines, e.selected)
	}
	for _, edit := range []struct {
		tabs  int
		value string
		call  string
	}{
		{0, "  Northern Clans  ", "SetTextField Player 2 name \"  Northern Clans  \""},
		{1, "15", "SetField Player 2 colour 15"},
		{2, "0", "SetField Player 2 control 0"},
		{3, "2", "SetField Player 2 relation:0 2"},
	} {
		before := len(calls.lines)
		_ = e.Key("field")
		for tab := 0; tab < edit.tabs; tab++ {
			_ = e.Key("tab")
		}
		playerPromptInput(t, e, edit.value)
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		if len(calls.lines) != before+1 || calls.lines[before] != edit.call || e.prompt.open() || e.doc.Records[e.selected].Index != 2 {
			t.Fatal("field edit did not use its table key exactly once", calls.lines, e.prompt)
		}
	}
	before := len(calls.lines)
	if err := e.Key("clone"); err != nil || e.pick != pickNone || len(calls.lines) != before {
		t.Fatal("a player was armed for cloning", err, e.pick, calls.lines)
	}
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if len(calls.lines) != before+1 || calls.lines[before] != "Delete Player 2" || len(e.rows()) != 2 || e.doc.Records[e.selected].Index != 1 {
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
// the SetField the field prompt writes; a value the claims do not name
// shows as its number and a click starts it over at 0.
func TestPlayersPageMatrixCyclesARelationPerClick(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	s, ok := e.listSpec()
	g := s.Grid
	if !ok || g == nil || g.Rows != 2 || g.Cols != 2 || len(g.Cells) != 4 || !g.Cells[0].Inert || !g.Cells[3].Inert || g.Cells[1].Inert || g.Cells[1].Icon != relationIcon(1) ||
		!slices.Equal(g.ColHeads, []string{"1", "2"}) || !slices.Equal(g.RowHeads, []string{"Self", "Enemy"}) {
		t.Fatalf("the Players page holds no 2x2 relation matrix: %+v", g)
	}
	legend := 0
	for _, row := range s.Rows {
		for v, w := range relationWords {
			if row.Label == e.word(w) && len(row.Icons) == 1 && row.Icons[0] == relationIcon(int64(v)) {
				legend++
			}
		}
	}
	if legend != 3 {
		t.Fatalf("the matrix's legend names %d of the 3 values: %+v", legend, s.Rows)
	}
	cell, ok := e.PlaceGridCellRect(1)
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
	diagonal, _ := e.PlaceGridCellRect(3)
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
	other, _ := e.PlaceGridCellRect(2)
	chromeClick(e, center(other))
	if calls.lines[len(calls.lines)-1] != "SetField Player 1 relation:0 0" {
		t.Fatal("a click on an unnamed value did not set 0", calls.lines)
	}
}

type refusedPlayerFields struct {
	InspectionEdits
	problem error
}

func (p refusedPlayerFields) SetTextField(string, int, string, string) (*InspectionDocument, error) {
	return nil, p.problem
}

func (p refusedPlayerFields) Delete(string, int) (*InspectionDocument, error) {
	return nil, p.problem
}

func TestMapEditorPlayerPromptKeepsTextAndErrors(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	_ = e.Key("down")
	_ = e.Key("field")
	name := strings.Repeat("Я", 32)
	playerPromptInput(t, e, name)
	e.SetHostTextEncoder(func(s string) string { return strings.ReplaceAll(s, "Я", "X") })
	found := false
	for _, row := range promptText(t, e) {
		found = found || row == strings.Repeat("X", 32)
	}
	if e.prompt.path != name || !found {
		t.Fatal("the common prompt lost a full-width UTF-8 player name", e.prompt.path, found)
	}
	problem := fmt.Errorf("player name exceeds 32 encoded bytes")
	e.doc.Edits = refusedPlayerFields{InspectionEdits: e.doc.Edits, problem: problem}
	selected := e.selected
	if err := e.Key("enter"); err != problem || e.Message() != problem.Error() || !e.prompt.open() || e.prompt.field != 0 || e.prompt.path != name || e.selected != selected || len(calls.lines) != 0 {
		t.Fatal("text refusal lost the selected field or entered text", err, e.prompt, calls.lines)
	}
	_ = e.Key("tab")
	playerPromptInput(t, e, "not a number")
	if err := e.Key("enter"); err == nil || e.Message() != "Enter a whole number." || !e.prompt.open() || e.prompt.field != 1 || e.prompt.path != "not a number" || len(calls.lines) != 0 {
		t.Fatal("numeric refusal lost its field or made an edit", err, e.prompt, calls.lines)
	}
	_ = e.Key("escape")
	problem = fmt.Errorf("Player 2 still owns Unit #0")
	e.doc.Edits = refusedPlayerFields{InspectionEdits: e.doc.Edits, problem: problem}
	if err := e.Key("delete"); err != problem || e.Message() != problem.Error() || e.selected != selected || len(calls.lines) != 0 {
		t.Fatal("delete refusal did not retain the referring-player message", err, e.Message())
	}
}

func TestMapEditorPlayersNeedOptionalAddAndTextHosts(t *testing.T) {
	e, calls := newModesEditor(t, "plain")
	_ = e.Key("players")
	_ = e.Key("player")
	if len(e.rows()) != 2 || len(calls.lines) != 0 {
		t.Fatal("a host without the Add window received an edit", calls.lines)
	}
	_ = e.Key("field")
	if err := e.Key("enter"); err == nil || !e.prompt.open() || e.Message() != "This host cannot edit text fields." || len(calls.lines) != 0 {
		t.Fatal("text editing silently succeeded without its optional host", err, calls.lines)
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
