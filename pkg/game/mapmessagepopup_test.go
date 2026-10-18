package game

import (
	"bytes"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

func messageClick(t *testing.T, e *ui.MapEditor, r image.Rectangle, ok bool) {
	t.Helper()
	if !ok || r.Empty() {
		t.Fatal("missing popup control")
	}
	p := r.Min.Add(r.Size().Div(2))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
}

func messageField(t *testing.T, e *ui.MapEditor, field int) {
	t.Helper()
	r, ok := e.FieldRect(field)
	messageClick(t, e, r, ok)
}

func messageButton(t *testing.T, e *ui.MapEditor, button int) {
	t.Helper()
	r, ok := e.PopupButtonRect(button)
	messageClick(t, e, r, ok)
}

func messagePopupFixture(t *testing.T, value uint32) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	script := scriptPayload([]scriptNode{{label: "Message", opcode: 2, id: 1, values: []uint32{value}, tags: []uint32{1}}}, nil)
	_, e, _ := openTestMapEditor(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: script}))
	e.Select(recordIndex(t, e, "Action", 0), false)
	return e, e.Document().Edits.(*mapInspectionEdits)
}

func TestMessagePopupNewApplyCancelAndBack(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	before := bytes.Clone(s.model.Bytes())
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(0)
	messageClick(t, e, r, ok)
	if string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" || len(e.PopupLines()) != 2 {
		t.Fatal("New did not open page", e.PopupLines())
	}
	messageField(t, e, 0)
	e.TextInput("First")
	_ = e.Key("enter")
	e.TextInput("Second")
	messageButton(t, e, 1)
	if string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" {
		t.Fatal("Cancel changed page")
	}
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Open Message") {
			r, ok := e.PopupRowRect(i)
			messageClick(t, e, r, ok)
			break
		}
	}
	messageField(t, e, 0)
	e.TextInput("First")
	_ = e.Key("newline")
	e.TextInput("Second")
	messageButton(t, e, 0)
	if string(s.workspace.Pages["event01.txt"]) != "<part=1>\nFirst\nSecond" {
		t.Fatal("Apply", string(s.workspace.Pages["event01.txt"]))
	}
	if !strings.Contains(e.PopupLines()[0], "First\nSecond") {
		t.Fatal("multi-line display flattened or substituted newline", e.PopupLines())
	}
	messageButton(t, e, 2)
	if !strings.Contains(strings.Join(e.PopupLines(), " "), "1: First") {
		t.Fatal("Back lost node or refreshed choices", e.PopupLines())
	}
	opened := false
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Open Message") {
			r, ok := e.PopupRowRect(i)
			messageClick(t, e, r, ok)
			opened = true
			break
		}
	}
	if !opened || len(e.PopupLines()) != 2 {
		t.Fatal("Open link did not open page", e.PopupLines())
	}
	messageButton(t, e, 2)
	if err := e.Key("undo"); err != nil || string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" {
		t.Fatal("page Undo", err)
	}
	if err := e.Key("undo"); err != nil || len(s.workspace.Pages) != 0 || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("New Undo", err)
	}
}

func TestMessagePopupMissingValueAndEPrompt(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	before := bytes.Clone(s.model.Bytes())
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(1)
	messageClick(t, e, r, ok)
	messageButton(t, e, 0)
	if !bytes.Equal(before, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("missing page changed value/history")
	}
	if _, ok := e.RowButtonRect(2, 0, false); ok {
		t.Fatal("missing page has Open")
	}
	if strings.Contains(strings.Join(e.PopupLines(), "\n"), "Open Message") {
		t.Fatal("missing page has Open link")
	}
	_ = e.Key("escape")
	if err := e.Key("field"); err != nil {
		t.Fatal(err)
	}
	for i, f := range s.Fields("Action", 0) {
		if f.Key == "value0" {
			if len(f.Choices) != 2 || f.Choices[0].Value != ui.NewMessageChoice || f.Choices[1].Value != 77 {
				t.Fatal("E choices", f.Choices)
			}
			for range i {
				_ = e.Key("tab")
			}
			_ = e.Key("clear")
			e.TextInput("77")
			_ = e.Key("enter")
			if !bytes.Equal(before, s.model.Bytes()) {
				t.Fatal("E current value changed ALM")
			}
			return
		}
	}
	t.Fatal("no message field")
}

func TestMessageChoiceKeepsNegativeValue(t *testing.T) {
	e, s := messagePopupFixture(t, ^uint32(0))
	before := bytes.Clone(s.model.Bytes())
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(1)
	messageClick(t, e, r, ok)
	messageButton(t, e, 0)
	if !bytes.Equal(before, s.model.Bytes()) || len(s.workspace.Pages) != 0 || s.CanUndo() {
		t.Fatal("raw all-ones value became New message")
	}
}

func TestMessageWizardWindowOpensPage(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	_ = e.Key("scripts")
	_ = e.Key("player")
	for range 3 {
		_ = e.Key("down")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(e.PopupLines()) != 2 || string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" {
		t.Fatal("wizard did not open new page", e.PopupLines())
	}
	messageButton(t, e, 2)
	if len(e.Document().Script.Actions) != 2 {
		t.Fatal("wizard omitted message action")
	}
	if err := e.Key("undo"); err != nil || len(e.Document().Script.Actions) != 1 || len(s.workspace.Pages) != 0 {
		t.Fatal("wizard Undo", err)
	}
}

func TestMessagePopupRemovedPageReturnsToNode(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	before := bytes.Clone(s.model.Bytes())
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(0)
	messageClick(t, e, r, ok)
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if len(s.workspace.Pages) != 0 || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("Undo lost map/page state")
	}
	lines := strings.Join(e.PopupLines(), " ")
	if strings.Contains(lines, "Page text") || !strings.Contains(lines, "77") {
		t.Fatal("removed page did not return to its node", e.PopupLines())
	}
	if _, err := s.SetMessage(1, "Stale page", -1); err == nil {
		t.Fatal("removed page accepted an edit")
	}
}

func TestMessagePopupOpenMapDropsPage(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	source := e.Document().Source
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(0)
	messageClick(t, e, r, ok)
	if _, err := s.SaveAs(t.TempDir() + "/saved.alm"); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(source); err != nil {
		t.Fatal(err)
	}
	if lines := e.PopupLines(); len(lines) != 0 {
		t.Fatal("another map retained the page popup", lines)
	}
}
