package ui

import (
	"fmt"
	"image"
	"strconv"
	"testing"

	"againrom/pkg/words"
	"github.com/hajimehoshi/ebiten/v2"
)

type modesWorkspace struct {
	modesFull
	fields  []InspectionField
	confirm *WorkspaceConfirmation
	leave   *WorkspaceLeave
}

func newModesWorkspace() *modesWorkspace {
	return &modesWorkspace{fields: []InspectionField{
		{Key: "folder", Label: "Folder", Text: true, TextValue: "working"},
		{Key: "title", Label: "Name", Text: true, TextValue: "River"},
		{Key: "after", Label: "After win", Value: 20, Column: 1, Choices: []InspectionChoice{{Value: 20, Label: "Campaign 20"}, {Value: 25, Label: "Crossing"}}},
		{Key: "source", Label: "Mission", Value: 1, Choices: []InspectionChoice{{Value: 1, Label: "Crossing (25)"}, {Value: 2, Label: "Watch (26)"}}},
		{Key: "party.0.character", Label: "Member / Leader", Value: 0, Column: 1, Choices: []InspectionChoice{{Value: 0, Label: "Aila"}, {Value: 1, Label: "Sellsword"}}},
		{Key: "party.0.level", Label: "Level", Value: 2, Column: 1, Parent: "party.0."},
		{Key: "add", Label: "Add open map", Action: true},
	}}
}

func (p *modesWorkspace) WorkspaceFields(scope string) []InspectionField {
	if scope == "package" {
		return p.fields
	}
	return nil
}
func (p *modesWorkspace) SetWorkspaceField(scope, key, value string) (*InspectionDocument, error) {
	if key == "folder" && p.leave != nil {
		return nil, p.leave
	}
	if key == "publish" && p.confirm != nil {
		return nil, p.confirm
	}
	for i := range p.fields {
		if p.fields[i].Key == key {
			if p.fields[i].Text {
				p.fields[i].TextValue = value
			} else {
				n, e := strconv.ParseInt(value, 10, 64)
				if e != nil && !p.fields[i].Action {
					return nil, e
				}
				p.fields[i].Value = n
			}
		}
	}
	return p.next("Workspace %s %s %q", scope, key, value)
}

func TestModWindowUsesTheSharedSaveDiscardCancelQuestion(t *testing.T) {
	for _, choice := range []int{leaveSave, leaveGo, leaveStay} {
		e, calls := newModesEditor(t, "workspace")
		p := e.doc.Edits.(*modesWorkspace)
		p.leave = &WorkspaceLeave{Folder: "other-mod"}
		_ = e.Key("package")
		_ = e.Key("clear")
		e.TextInput("other-mod")
		_ = e.Key("enter")
		if !asked(e) || !e.prompt.then.workspace {
			t.Fatal("folder change did not reach the shared question")
		}
		leaveButton(t, e, choice)
		if choice == leaveSave {
			if e.prompt.kind != promptSave {
				t.Fatal("Save did not use Save As", e.prompt.kind)
			}
			_ = e.Key("clear")
			e.TextInput("copy.alm")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
		}
		opened := false
		for _, line := range calls.lines {
			opened = opened || line == "Workspace package folder-open \"other-mod\""
		}
		if opened != (choice != leaveStay) || e.ExitAsked() {
			t.Fatal("question chose the wrong continuation", choice, calls.lines)
		}
	}
}

func TestModWindowPublishConfirmationKeepsItsTargetAndShowsTheExplicitAction(t *testing.T) {
	e, calls := newModesEditor(t, "workspace")
	p := e.doc.Edits.(*modesWorkspace)
	p.confirm = &WorkspaceConfirmation{Action: "publish-replace", Message: "Replace this target?", Confirm: "Replace"}
	_ = e.Key("package")
	if !e.FocusModField("publish") {
		t.Fatal("missing Publish footer")
	}
	_ = e.Key("enter")
	s, _, _ := e.modSpec()
	if e.modConfirm != "publish-replace" || len(s.Buttons) != 2 || s.Buttons[0] != "Replace" {
		t.Fatal("missing explicit confirmation", s.Buttons)
	}
	before := len(calls.lines)
	for _, key := range []string{"cardup", "saveas", "testplay", "undo", "package"} {
		_ = e.Key(key)
	}
	for range 5 {
		_ = e.Key("tab")
		if e.obj.focus < len(e.obj.fields) {
			t.Fatal("confirmation focus entered fields")
		}
	}
	if len(calls.lines) != before {
		t.Fatal("confirmation changed the target", calls.lines)
	}
	_ = e.Key("escape")
	if e.modConfirm != "" || len(calls.lines) != before {
		t.Fatal("Cancel performed an action")
	}
}

func TestModWindowMouseMissionSwitchAcceptsTheFocusedField(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	_ = e.Key("package")
	e.FocusModField("title")
	_ = e.Key("clear")
	e.TextInput("Accepted by click")
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	clicked := false
	for _, row := range l.Rows {
		if actions := e.obj.rowActs[row.Index]; len(actions) > 0 && actions[0] == "2" && len(row.Buttons) > 0 {
			modesClick(e, center(row.Buttons[0]))
			clicked = true
			break
		}
	}
	p := e.doc.Edits.(*modesWorkspace)
	if !clicked || p.fields[1].TextValue != "Accepted by click" || p.fields[3].Value != 2 {
		t.Fatal("mission switch discarded the focused field", p.fields)
	}
}

func TestModWindowUsesSharedRowsAndBlocksCanvasInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, calls := newModesEditor(t, "workspace")
		e.SetWords(words.For(lang))
		e.Layout(640, 480)
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
		s, _, ok := e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		if !ok || s.Columns != 2 || !l.Rect.In(image.Rect(0, e.barHeight(), 640, 480)) || len(l.Buttons) != 5 {
			t.Fatalf("%s mod window layout: %v", lang, l.Rect)
		}
		before := len(calls.lines)
		for _, key := range []string{"unit", "clone", "delete", "drop", "field", "new"} {
			_ = e.Key(key)
		}
		modesClick(e, image.Pt(400, 400))
		if len(calls.lines) != before {
			t.Fatal("mod window leaked a canvas command", calls.lines)
		}
		if !e.FocusModField("after") {
			t.Fatal("missing shared choice field")
		}
		_ = e.Key("enter")
		e.TextInput("cross")
		_ = e.Key("enter")
		if e.obj.fields[e.obj.focus].Value != 25 {
			t.Fatal("named choice did not commit", calls.lines)
		}
		if !e.FocusModField("title") {
			t.Fatal("missing title")
		}
		_ = e.Key("clear")
		e.TextInput("Changed")
		_ = e.Key("enter")
		if !e.ModWindowOpen() {
			t.Fatal("accepting a value closed W")
		}
		_ = e.Key("escape")
		if e.ModWindowOpen() {
			t.Fatal("Escape did not close W")
		}
	}
}

func TestPopupColumnScrollingKeepsTheOtherColumnFixed(t *testing.T) {
	s := popupSpec{Columns: 2, Width: 620, MaxRows: 5, Top: 7, ColumnTops: []int{0, 7}}
	for col := 0; col < 2; col++ {
		for n := 0; n < 14; n++ {
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Column: col, Label: fmt.Sprint(col, n)})
		}
	}
	l := layoutPopup(s, nil, nil)
	if len(l.Rows) != 10 {
		t.Fatal("independent columns", len(l.Rows))
	}
	if l.Rows[0].Index != 0 || l.Rows[5].Index != 21 {
		t.Fatal("column scroll moved identity", l.Rows)
	}
}

func TestModWindowNativeKeysReachSaveHistoryAndReverseFocus(t *testing.T) {
	e, calls := newModesEditor(t, "workspace")
	_ = e.Key("package")
	e.FocusModField("title")
	e.TextInput("x")
	before := len(calls.lines)
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyZ)
	if len(calls.lines) != before {
		t.Fatal("text undo reached project history")
	}
	_ = e.Key("clear")
	e.TextInput("Accepted")
	_ = e.Key("enter")
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyS)
	if len(calls.lines) == 0 || calls.lines[len(calls.lines)-1] != "Workspace package save \"\"" {
		t.Fatal("Ctrl+S did not save the mod", calls.lines)
	}
	e.FocusModField("title")
	updateEditorKeys(t, e, ebiten.KeyShift, ebiten.KeyTab)
	if e.obj.fields[e.obj.focus].Key != "folder" {
		t.Fatal("Shift+Tab did not move backward")
	}
}
