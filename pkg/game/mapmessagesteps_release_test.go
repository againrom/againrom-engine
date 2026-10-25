package game

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func installedTriggerMessage(t *testing.T) (*MapInspector, *ui.MapEditor, string, int) {
	t.Helper()
	x := installedInspector(t)
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		if e.Document().Script == nil {
			continue
		}
		for _, tr := range e.Document().Script.Triggers {
			e.Select(tr.Record, false)
			for action := range 4 {
				field := e.WizardField(fmt.Sprintf("a%d.p0", action))
				if field < 0 {
					continue
				}
				index := -1
				for i, n := range e.Document().Script.Actions {
					if n.ID == tr.Acts[action] && n.Opcode == 2 && n.Types[0] == 1 {
						index = i
					}
				}
				if index < 0 {
					continue
				}
				t.Logf("%s trigger=%d action=%d", key, tr.Record, index)
				return x, e, fmt.Sprintf("a%d.p0", action), index
			}
		}
	}
	t.Fatal("no installed trigger message parameter")
	return nil, nil, "", -1
}

func TestReleaseTriggerMessageFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, e, key, index := installedTriggerMessage(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before, pages := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages
	messageFrame(t, e, x.pageLanguage(), "trigger-message-popup")
	messageField(t, e, wizField(t, e, key))
	messageFrame(t, e, x.pageLanguage(), "wizard-message-list")
	e.Select(recordIndex(t, e, "Action", index), false)
	messageField(t, e, 2)
	messageFrame(t, e, x.pageLanguage(), "details-message-list")
	if !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) {
		t.Fatal("frame capture changed the map or pages")
	}
}

func TestReleaseTriggerMessageEditsAndUndo(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, e, key, _ := installedTriggerMessage(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
	page := freeMessage(pages)
	messageField(t, e, wizField(t, e, key))
	for row, target := range e.Document().Script.Targets[ui.ScriptMessageParameter] {
		if target.NewMessage {
			e.ScrollDropdown(-len(e.DropdownChoices()))
			for range len(e.DropdownChoices()) + 1 {
				if r, ok := e.DropdownRowRect(row); ok {
					messageClick(t, e, r, ok)
					break
				}
				e.ScrollDropdown(1)
			}
			break
		}
	}
	if len(e.PopupLines()) != 3 {
		t.Fatal("wizard New did not open the shared message editor", e.PopupLines())
	}
	body := map[string]string{"en": "A new wizard message.", "ru": "Новое сообщение мастера."}[x.pageLanguage()]
	e.TextInput(body)
	fields := s.MessageFields(ui.NewMessagePage)
	messageField(t, e, 1)
	r, ok := e.DropdownRowRect(1)
	messageClick(t, e, r, ok)
	messageButton(t, e, 0)
	want := fmt.Sprintf("<part=1,npc=%d>\n%s", fields[1].Choices[1].Value, body)
	if string(s.workspace.Pages[messageFilename(page)]) != want || len(s.undoWorkspace) != steps+1 {
		t.Fatal("installed wizard did not create one page/attachment edit")
	}
	messageFrame(t, e, x.pageLanguage(), "wizard-created-page")
	messageButton(t, e, 2)
	if !e.WizardShown() {
		t.Fatal("Back did not return to the trigger wizard")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) {
		t.Fatal("one installed Undo did not restore map and pages", err)
	}
	t.Logf("lang=%s page=%d: five clicks created text, speaker and trigger attachment; one Undo restored all bytes", x.pageLanguage(), page)
}
