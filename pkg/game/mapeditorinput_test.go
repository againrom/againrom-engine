package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func TestEditorMessageStartsTypingWithoutFocusClick(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{{ID: 21, Name: "hero"}}}}
	before := bytes.Clone(s.model.Bytes())
	messageField(t, e, 2)
	r, ok := e.DropdownRowRect(0)
	messageClick(t, e, r, ok)
	e.TextInput("Привет")
	_ = e.Key("enter")
	e.TextInput("World")
	messageField(t, e, 1)
	r, ok = e.DropdownRowRect(1)
	messageClick(t, e, r, ok)
	messageButton(t, e, 0)
	if got := string(s.workspace.Pages["event01.txt"]); got != "<part=1,npc=21>\nПривет\nWorld" {
		t.Fatal("no-focus path", got)
	}
	model, _ := s.model.Map()
	script, _ := model.Script()
	if script.Actions[0].Value[0] != 1 {
		t.Fatal("new page was not attached")
	}
	if err := e.Key("undo"); err != nil || len(s.workspace.Pages) != 0 || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("one undo did not restore map and pages", err)
	}
}

func TestEditorInputRulesUseStoredCodecs(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	_ = e
	for _, c := range []struct {
		scope, kind, key string
		codec            alm.TextField
	}{
		{"metadata", "", "name", alm.MapNameText}, {"metadata", "", "description", alm.DescriptionText},
		{"", "Player", "name", alm.PlayerNameText}, {"", "Trigger", "name", alm.ScriptLabelText},
		{"", "Condition", "name", alm.ScriptLabelText}, {"", "Action", "name", alm.ScriptLabelText},
	} {
		t.Run(c.scope+c.kind+c.key, func(t *testing.T) {
			limit, size := s.InputRule(c.scope, c.kind, c.key)
			want := c.codec.Width
			if c.codec.Terminated {
				want--
			}
			if limit != want {
				t.Fatal("limit differs from field", limit, want)
			}
			for _, r := range "LatinЯяЁ🙂\x00" {
				b, err := c.codec.Encode(string(r))
				n := len(b)
				if err != nil {
					n = 0
				}
				if size(r) != n {
					t.Fatalf("%U: model accepts %d bytes, codec %d", r, size(r), n)
				}
			}
		})
	}
	for _, selector := range []int{0, text.SelectorConverting} {
		s.inspector.font = &text.Font{Selector: selector}
		limit, size := s.InputRule("pages", "", "body")
		if limit != mod.MaxFileBytes || size('Я') != 2 || size('🙂') != 0 || size(0) != 0 {
			t.Fatal("page codec policy", limit, size('Я'))
		}
	}
}

func TestEditorTextInstalledFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := os.Getenv("AGAINROM_EDITOR_TEXT_FRAMES")
	x := installedInspector(t)
	if dir == "" {
		dir = t.TempDir()
	}
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	index := -1
	for i, a := range e.Document().Script.Actions {
		if a.Opcode == 2 && a.Types[0] == 1 {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("no installed message action")
	}
	e.Select(recordIndex(t, e, "Action", index), false)
	before := bytes.Clone(s.model.Bytes())
	pages := cloneWorkspace(s.workspace).Pages
	messageField(t, e, 2)
	row := 0
	for i, target := range e.Document().Script.Targets[ui.ScriptMessageParameter] {
		if target.NewMessage {
			row = i
		}
	}
	r, ok := e.DropdownRowRect(row)
	for attempt := 0; !ok && attempt <= row; attempt++ {
		_ = e.Key("pagedown")
		r, ok = e.DropdownRowRect(row)
	}
	messageClick(t, e, r, ok)
	body := "Write this dialogue here"
	if x.pageLanguage() == "ru" {
		body = "Здесь можно написать диалог"
	}
	e.TextInput(body)
	_ = e.Key("ctrl+shift+left")
	t.Setenv("AGAINROM_MESSAGE_PICKER_FRAMES", dir)
	messageFrame(t, e, x.pageLanguage(), "text-selection")
	_ = e.Key("end")
	messageFrame(t, e, x.pageLanguage(), "message-editor")
	messageField(t, e, 1)
	messageFrame(t, e, x.pageLanguage(), "speaker-list")
	if !bytes.Equal(before, s.model.Bytes()) || len(pages) != len(s.workspace.Pages) {
		t.Fatal("draft frame changed map/pages")
	}
	if _, err := os.Stat(filepath.Join(dir, x.pageLanguage()+"-text-selection.png")); err != nil {
		t.Fatal(err)
	}
	messageButton(t, e, 1)
}

func TestEditorExistingTextOperationKeepsMapBytes(t *testing.T) {
	e, s := messagePopupFixture(t, 77)
	before := bytes.Clone(s.model.Bytes())
	e.Select(recordIndex(t, e, "Action", 0), false)
	messageField(t, e, 0)
	_ = e.Key("selectall")
	e.TextInput("Renamed")
	messageButton(t, e, 0)
	want, err := mapedit.New(before)
	if err != nil {
		t.Fatal(err)
	}
	if err := want.SetTextField(mapedit.KindAction, 0, "name", "Renamed"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want.Bytes(), s.model.Bytes()) {
		t.Fatal("text model changed existing ALM operation")
	}
}

func TestEditorPageFilenameContinuesInBody(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := bytes.Clone(s.model.Bytes())
	pages := cloneWorkspace(s.workspace).Pages
	_ = e.Key("pages")
	_ = e.Key("selectall")
	e.TextInput("event09.txt")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	e.TextInput("<part=1,npc=21>First")
	_ = e.Key("newline")
	e.TextInput("Second")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if string(s.workspace.Pages["event09.txt"]) != "<part=1,npc=21>First\nSecond" {
		t.Fatal("filename did not continue directly into body", s.workspace.Pages)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(before, s.model.Bytes()) || len(pages) != len(s.workspace.Pages) {
		t.Fatal("page write undo", err)
	}
}
