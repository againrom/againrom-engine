package game

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func triggerMessageFixture(t *testing.T, shared bool) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	other := uint32(2)
	if shared {
		other = 1
	}
	acts := []scriptNode{{label: "Talk", opcode: 2, id: 1, tags: []uint32{1}, values: []uint32{7}}, {label: "Win", opcode: 4, id: 2}}
	checks := []scriptNode{{label: "One", opcode: 0x10002, id: 1, tags: []uint32{1}, values: []uint32{1}}}
	triggers := []goalTrigger{{name: "Talk here", left: 1, right: 1, actions: []uint32{1}}, {name: "Other", left: 1, right: 1, actions: []uint32{other}}}
	_, e, _ := openTestMapEditor(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(acts, checks, triggers)}))
	s := e.Document().Edits.(*mapInspectionEdits)
	s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{{ID: 21, Name: "Speaker"}}}}
	workspacePage(t, e, "event02.txt", "<part=1,npc=21>\nExisting message")
	return e, s
}

func messageChoiceClick(t *testing.T, e *ui.MapEditor, want string) {
	t.Helper()
	choices := e.DropdownChoices()
	e.ScrollDropdown(-len(choices))
	for row, label := range choices {
		if !strings.Contains(label, want) {
			continue
		}
		for range len(choices) + 1 {
			if r, ok := e.DropdownRowRect(row); ok {
				messageClick(t, e, r, ok)
				return
			}
			e.ScrollDropdown(1)
		}
	}
	t.Fatalf("no message choice %q in %q", want, choices)
}

// The Details action's message field reuses a page in two clicks; the
// trigger wizard reuses one through the dialog editor's choice.
func TestMessageReuseNeedsTwoClicksAndOneUndo(t *testing.T) {
	e, s := triggerMessageFixture(t, false)
	e.Select(recordIndex(t, e, "Action", 0), false)
	before, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
	messageField(t, e, 2)
	messageChoiceClick(t, e, "2: Existing message")
	sc := wizScript(t, s.model.Bytes())
	if sc.Actions[0].Value[0] != 2 || len(s.undoWorkspace) != steps+1 || !reflect.DeepEqual(pages, s.workspace.Pages) {
		t.Fatal("two clicks did not commit exactly one page-reference edit")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps {
		t.Fatal("one Undo did not restore the map and pages", err)
	}
	t.Log("before: field, choice, Apply = 3 clicks; after: field, choice = 2 clicks")
}

func TestMessageReuseKeepsOtherPendingFields(t *testing.T) {
	for _, apply := range []bool{false, true} {
		t.Run(map[bool]string{false: "action cancel", true: "action apply"}[apply], func(t *testing.T) {
			e, s := triggerMessageFixture(t, false)
			e.Select(recordIndex(t, e, "Action", 0), false)
			before, steps := bytes.Clone(s.model.Bytes()), len(s.undoWorkspace)
			messageField(t, e, 0)
			_ = e.Key("selectall")
			e.TextInput("Pending label")
			messageField(t, e, 2)
			messageChoiceClick(t, e, "2: Existing message")
			if !bytes.Equal(before, s.model.Bytes()) || len(s.undoWorkspace) != steps {
				t.Fatal("message choice committed another pending field")
			}
			if apply {
				messageButton(t, e, 0)
				sc := wizScript(t, s.model.Bytes())
				if sc.Actions[0].Value[0] != 2 || len(s.undoWorkspace) != steps+1 {
					t.Fatal("Apply did not commit the draft as one edit")
				}
				if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
					t.Fatal("draft Undo", err)
				}
			} else {
				messageButton(t, e, 1)
				if !bytes.Equal(before, s.model.Bytes()) || len(s.undoWorkspace) != steps {
					t.Fatal("Cancel committed the draft")
				}
			}
		})
	}
}

func TestTriggerWizardNewMessageUsesFourClicksAndOneUndo(t *testing.T) {
	for _, route := range []struct {
		name            string
		shared, details bool
		clicks          int
	}{{"wizard", false, false, 4}, {"wizard shared action", true, false, 4}, {"Details navigation control", false, true, 7}} {
		t.Run(route.name, func(t *testing.T) {
			e, s := triggerMessageFixture(t, route.shared)
			e.Select(recordIndex(t, e, "Trigger", 0), false)
			before, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			clicks := 0
			if route.details {
				messageButton(t, e, 4)
				r, ok := e.RowButtonRect(11, 0, false)
				messageClick(t, e, r, ok)
				clicks += 2
				messageField(t, e, 2)
				clicks++
				messageChoiceClick(t, e, "New message")
				clicks++
			} else {
				// Configure on an action whose page is missing opens a new dialog.
				configure(t, e, "a0.item")
				clicks++
			}
			e.TextInput("New dialogue")
			messageField(t, e, 1)
			clicks++
			messageChoiceClick(t, e, "21: Main hero")
			clicks++
			messageButton(t, e, 0)
			clicks++
			if clicks != route.clicks {
				t.Fatal("click count", clicks, route.clicks)
			}
			if string(s.workspace.Pages["event01.txt"]) != "<part=1,npc=21>\nNew dialogue" || len(s.undoWorkspace) != steps+1 {
				t.Fatal("five-click wizard route did not create one attached page")
			}
			sc := wizScript(t, s.model.Bytes())
			if wizNode(t, sc.Actions, sc.Triggers[0].Acts[0]).Value[0] != 1 {
				t.Fatal("new page is not attached to the trigger")
			}
			if route.shared && (sc.Triggers[0].Acts[0] == sc.Triggers[1].Acts[0] || wizNode(t, sc.Actions, sc.Triggers[1].Acts[0]).Value[0] != 7) {
				t.Fatal("wizard creation changed another trigger's action")
			}
			changed := bytes.Clone(s.model.Bytes())
			if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) {
				t.Fatal("one Undo did not remove page and attachment", err)
			}
			if err := e.Key("redo"); err != nil || !bytes.Equal(changed, s.model.Bytes()) || len(s.workspace.Pages) != len(pages)+1 {
				t.Fatal("Redo did not restore page and attachment", err)
			}
			path := filepath.Join(t.TempDir(), "dialogue.alm")
			if _, err := s.SaveAs(path); err != nil {
				t.Fatal(err)
			}
			if err := e.Open(path); err != nil {
				t.Fatal(err)
			}
			s = e.Document().Edits.(*mapInspectionEdits)
			if !bytes.Equal(changed, s.model.Bytes()) || string(s.workspace.Pages["event01.txt"]) != "<part=1,npc=21>\nNew dialogue" {
				t.Fatal("cold Open lost page, speaker or attachment")
			}
			t.Logf("%s: %d pointer clicks, one body input, one Undo", route.name, clicks)
		})
	}
}

func TestTriggerMessageReuseKeepsSharedActionPrivate(t *testing.T) {
	e, s := triggerMessageFixture(t, true)
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	before := bytes.Clone(s.model.Bytes())
	configure(t, e, "a0.item")
	messageField(t, e, 2) // the dialog choice
	messageChoiceClick(t, e, "2: Existing message")
	messageButton(t, e, 0)
	sc := wizScript(t, s.model.Bytes())
	if sc.Triggers[0].Acts[0] == sc.Triggers[1].Acts[0] || wizNode(t, sc.Actions, sc.Triggers[0].Acts[0]).Value[0] != 2 || wizNode(t, sc.Actions, sc.Triggers[1].Acts[0]).Value[0] != 7 {
		t.Fatal("reuse through the dialog choice changed another trigger's shared action")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("shared reuse Undo", err)
	}
}

func TestTriggerWizardNewMessageCancelAndRefusalKeepHistory(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel and invalid text", true: "pending name"}[pending], func(t *testing.T) {
			e, s := triggerMessageFixture(t, true)
			e.Select(recordIndex(t, e, "Trigger", 0), false)
			before, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			if pending {
				messageField(t, e, 0)
				e.TextInput(" pending")
			}
			configure(t, e, "a0.item")
			if pending {
				// Configure applies the pending trigger first, as its own entry.
				sc := wizScript(t, s.model.Bytes())
				if sc.Triggers[0].Name != "Talk here pending" || len(s.undoWorkspace) != steps+1 {
					t.Fatal("Configure did not apply the pending trigger as one entry", sc.Triggers[0].Name)
				}
				before, pages, steps = bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			} else {
				e.TextInput("bad\x00body")
				messageButton(t, e, 0)
				if len(e.PopupLines()) <= 2 {
					t.Fatal("invalid page did not report its refusal")
				}
			}
			if !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps {
				t.Fatal("refused message changed map, pages or history")
			}
			messageButton(t, e, 1)
			if !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps {
				t.Fatal("Cancel changed map, pages or history")
			}
		})
	}
}

func TestTriggerWizardOpensMessageInOneClick(t *testing.T) {
	e, s := triggerMessageFixture(t, false)
	if _, err := s.SetField("Action", 0, "value0", 2); err != nil {
		t.Fatal(err)
	}
	e, _ = openGoalMap(t, s.model.Bytes())
	s = e.Document().Edits.(*mapInspectionEdits)
	workspacePage(t, e, "event02.txt", "<part=1,npc=21>\nExisting message")
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	before, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
	configure(t, e, "a0.item")
	if !strings.Contains(trigLines(e), "Existing message") {
		t.Fatal("one click did not open the page", e.PopupLines())
	}
	messageButton(t, e, 2)
	if !e.WizardShown() || !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps {
		t.Fatal("Back changed the map, pages or history")
	}
	t.Log("before: Details, action Edit, Open = 3 clicks; after: Configure = 1 click")
}
