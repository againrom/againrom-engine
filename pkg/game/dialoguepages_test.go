package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func openMessagePopup(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Open Message") || strings.HasPrefix(line, "Открыть сообщение") {
			r, ok := e.PopupRowRect(i)
			messageClick(t, e, r, ok)
			return
		}
	}
	t.Fatal("message link absent", e.PopupLines())
}

func dialogueField(t *testing.T, e *ui.MapEditor, field int) {
	t.Helper()
	e.ScrollDetails(-100)
	for range 30 {
		if r, ok := e.FieldRect(field); ok {
			messageClick(t, e, r, ok)
			return
		}
		e.ScrollDetails(1)
	}
	t.Fatal("field absent", field, e.PopupLines())
}

func dialogueButton(t *testing.T, e *ui.MapEditor, part, button int) {
	t.Helper()
	e.ScrollDetails(-100)
	for range 30 {
		if r, ok := e.RowButtonRect(2*part+1, button, false); ok {
			messageClick(t, e, r, ok)
			return
		}
		e.ScrollDetails(1)
	}
	t.Fatal("card button absent", part, button)
}

func TestDialoguePopupUntaggedApply(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "text edit"}[edit], func(t *testing.T) {
			e, s := messagePopupFixture(t, 1)
			raw := "Plain message\r\nSecond line"
			workspacePage(t, e, "event01.txt", raw)
			before := bytes.Clone(s.model.Bytes())
			workspaceBefore := bytes.Clone(s.workspaceImage)
			undoBefore := len(s.undoWorkspace)
			e.Select(recordIndex(t, e, "Action", 0), false)
			openMessagePopup(t, e)
			if edit {
				messageField(t, e, 0)
				if err := e.Key("ctrl+end"); err != nil {
					t.Fatal(err)
				}
				e.TextInput("!")
			}
			messageButton(t, e, 0)
			want, steps := raw, undoBefore
			if edit {
				want, steps = "<part=1>\n"+raw+"!", undoBefore+1
			}
			if got := string(s.workspace.Pages["event01.txt"]); got != want {
				t.Errorf("Apply bytes = %q, want %q", got, want)
			}
			if len(s.undoWorkspace) != steps || !bytes.Equal(before, s.model.Bytes()) {
				t.Errorf("Apply history = %d, want %d, or changed map bytes", len(s.undoWorkspace), steps)
			}
			if !edit && !bytes.Equal(workspaceBefore, s.workspaceImage) {
				t.Error("unchanged Apply changed workspace bytes")
			}
			if edit {
				if err := e.Key("undo"); err != nil || string(s.workspace.Pages["event01.txt"]) != raw || !bytes.Equal(workspaceBefore, s.workspaceImage) {
					t.Fatal("one Undo did not restore untagged bytes", err)
				}
			}
		})
	}
}

func TestDialoguePopupEditsEveryPageAndKeepsOtherBytes(t *testing.T) {
	e, s := messagePopupFixture(t, 1)
	raw := "<part=1,npc=21,tips=7>\r\nOne\r\n<part=2,npc=51,iamfemale,unknown=x>\nTwo\r\n<part=3,npc=22>\r\nThree"
	workspacePage(t, e, "event01.txt", raw)
	e.Select(recordIndex(t, e, "Action", 0), false)
	openMessagePopup(t, e)
	for _, word := range []string{"One", "Two", "Three", "51", "21"} {
		if !strings.Contains(strings.Join(e.PopupLines(), " "), word) {
			t.Fatal("page or speaker hidden", word, e.PopupLines())
		}
	}
	dialogueField(t, e, 2)
	_ = e.Key("selectall")
	e.TextInput("Changed")
	_ = e.Key("enter")
	e.TextInput("Second line")
	messageButton(t, e, 0)
	want := strings.Replace(raw, "Two", "Changed\nSecond line", 1)
	if string(s.workspace.Pages["event01.txt"]) != want {
		t.Fatal("page 2 changed other bytes", string(s.workspace.Pages["event01.txt"]))
	}
	messageButton(t, e, 2)
	if err := e.Key("undo"); err != nil || string(s.workspace.Pages["event01.txt"]) != raw {
		t.Fatal("one Undo did not restore all bytes", err)
	}
}

func TestDialoguePopupAddsRemovesAndReordersBeforeApply(t *testing.T) {
	e, s := messagePopupFixture(t, 1)
	raw := "<part=1 npc=21 tips=7>\nOne<part=2 npc=22>\nTwo<part=3 npc=23>\nThree"
	workspacePage(t, e, "event01.txt", raw)
	before := bytes.Clone(s.model.Bytes())
	e.Select(recordIndex(t, e, "Action", 0), false)
	openMessagePopup(t, e)
	dialogueButton(t, e, -1, 0) // "+"
	e.TextInput("Last")
	dialogueButton(t, e, 1, 0) // the cross of Two
	dialogueField(t, e, 4)
	_ = e.Key("cardup")
	_ = e.Key("cardup")
	if string(s.workspace.Pages["event01.txt"]) != raw || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("draft changed persistent bytes")
	}
	messageButton(t, e, 0)
	want := "<part=1 npc=23>\nThree\r\n<part=2 npc=21 tips=7>\nOne\r\n<part=3>\nLast"
	if string(s.workspace.Pages["event01.txt"]) != want || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("structure or renumber", string(s.workspace.Pages["event01.txt"]))
	}
	dialogueField(t, e, 2)
	_ = e.Key("cardup")
	messageButton(t, e, 1)
	if string(s.workspace.Pages["event01.txt"]) != want {
		t.Fatal("Cancel kept draft reorder")
	}
}

func TestDialogueMessageOverlapWarningIncludesRepeatedActionSlots(t *testing.T) {
	e, s := messagePopupFixture(t, 1)
	s.inspector.rules = []mapRule{mapRules[len(mapRules)-1]}
	for range 2 {
		if _, err := s.AddScript("Condition"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AddScript("Trigger"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFields("Trigger", 0, []ui.InspectionFieldValue{{Key: "left0", Value: 1}, {Key: "right0", Value: 2}, {Key: "action0", Value: 1}, {Key: "action1", Value: 1}}); err != nil {
		t.Fatal(err)
	}
	hits := ofRule(s.Report(), "message-overlap")
	if len(hits) != 1 || hits[0].Error || hits[0].Kind != "Trigger" || hits[0].Index != 0 || !strings.Contains(hits[0].Text, "will not see the second") {
		t.Fatal("missing plain warning", hits)
	}
	r, ok := e.BarRect("map")
	messageClick(t, e, r, ok)
	r, ok = e.MenuRowRect(6)
	messageClick(t, e, r, ok)
	if lines := e.PanelLines(); len(lines) == 0 || !strings.Contains(strings.Join(lines, " "), "second") {
		t.Fatal("Map panel hides message warning", e.PanelLines())
	}
	if _, err := s.SetFields("Trigger", 0, []ui.InspectionFieldValue{{Key: "action1", Value: 0}}); err != nil || len(ofRule(s.Report(), "message-overlap")) != 0 {
		t.Fatal("single message warned", err)
	}
}

func TestDialoguePopupMovesConditionalAlternativesTogether(t *testing.T) {
	e, s := messagePopupFixture(t, 1)
	raw := "<part=1 iamfemale npc=21 tips=7>\nHer<part=1 iammale npc=21>\nHis<part=2 npc=22>\nNext"
	workspacePage(t, e, "event01.txt", raw)
	e.Select(recordIndex(t, e, "Action", 0), false)
	openMessagePopup(t, e)
	dialogueField(t, e, 0)
	_ = e.Key("carddown")
	messageButton(t, e, 0)
	want := "<part=1 npc=22>\nNext\r\n<part=2 iamfemale npc=21 tips=7>\nHer\r\n<part=2 iammale npc=21>\nHis"
	if string(s.workspace.Pages["event01.txt"]) != want {
		t.Fatal("conditional alternatives separated or lost attributes", string(s.workspace.Pages["event01.txt"]))
	}
}
