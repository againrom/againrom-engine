package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestMessageChoicesUseWorkspaceFirstLine(t *testing.T) {
	_, e := workspaceFixture(t)
	workspacePage(t, e, "event02.txt", "Second page\nAnother line")
	workspacePage(t, e, "event04.txt", "<NPC=51,Part=1>\r\n\r\nТекст страницы\r\nДальше")
	s := e.Document().Edits.(*mapInspectionEdits)
	d, err := s.AddScript("Action")
	if err != nil {
		t.Fatal(err)
	}
	rows := d.Script.Targets[ui.ScriptMessageParameter]
	if len(rows) < 1 || rows[0].Value != 2 || rows[0].ChoiceText() != "2: Second page..." || rows[0].Text != "Second page\nAnother line" {
		t.Fatalf("message choices: %+v", rows)
	}
	if len(rows) != 3 || rows[1].Value != 4 || rows[1].ChoiceText() != s.inspector.inspectionUTF8("4: Текст страницы...") || rows[1].Text != s.inspector.inspectionUTF8("Текст страницы")+"\n"+s.inspector.inspectionUTF8("Дальше") {
		t.Fatal("registry-first header, CRLF or Cyrillic label", rows)
	}
	if fields := s.MessageFields(4); fields[1].Value != 51 {
		t.Fatal("registry-first speaker", fields)
	}
}

func TestMessageFreeNumber(t *testing.T) {
	pages := map[string][]byte{"event01.txt": {}, "event03.txt": {}, "event254.txt": {}, "event255.txt": {}}
	if got := freeMessage(pages); got != 2 {
		t.Fatal(got)
	}
	for i := 1; i <= 254; i++ {
		pages[fmt.Sprintf("event%02d.txt", i)] = nil
	}
	if got := freeMessage(pages); got != 256 {
		t.Fatal(got)
	}
	delete(pages, "event254.txt")
	if got := freeMessage(pages); got != 254 {
		t.Fatal(got)
	}
}

func TestMessageViewKindCannotComeFromWireTag(t *testing.T) {
	var types [10]uint32
	types[0] = 11
	if kind := scriptParameterKind(false, 3, types, 0); kind != 0 {
		t.Fatal("unknown wire tag became a message picker", kind)
	}
	types[0] = 1
	if kind := scriptParameterKind(false, 2, types, 0); kind != ui.ScriptMessageParameter {
		t.Fatal("message operation lost its page picker", kind)
	}
}

func TestMessageNewPageBytesAndHistory(t *testing.T) {
	for _, old := range []int64{1, 7} {
		t.Run(fmt.Sprint(old), func(t *testing.T) {
			_, e := workspaceFixture(t)
			s := e.Document().Edits.(*mapInspectionEdits)
			if _, err := s.AddScript("Action"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SetField("Action", s.model.Count(mapedit.KindAction)-1, "value0", old); err != nil {
				t.Fatal(err)
			}
			before := bytes.Clone(s.model.Bytes())
			pages := cloneWorkspace(s.workspace).Pages
			d, err := s.NewMessage(ui.InspectionMessageOwner{Kind: "Action", Index: s.model.Count(mapedit.KindAction) - 1, Key: "value0"})
			if err != nil {
				t.Fatal(err)
			}
			m, _ := s.model.Map()
			sc, _ := m.Script()
			if sc.Actions[len(sc.Actions)-1].Value[0] != 1 || string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" {
				t.Fatal("new page/node", sc.Actions, s.workspace.Pages)
			}
			changed := bytes.Clone(d.SourceBytes)
			if old == 7 {
				diff := 0
				for i := range before {
					if before[i] != changed[i] {
						diff++
					}
				}
				if diff != 1 {
					t.Fatalf("changed %d bytes, expected the value word's low byte only", diff)
				}
			} else if !bytes.Equal(before, changed) {
				t.Fatal("same-value page creation rewrote ALM")
			}
			if _, err := s.Undo(); err != nil || !bytes.Equal(s.model.Bytes(), before) || !reflect.DeepEqual(s.workspace.Pages, pages) {
				t.Fatal("Undo", err)
			}
			if _, err := s.Redo(); err != nil || !bytes.Equal(s.model.Bytes(), changed) || string(s.workspace.Pages["event01.txt"]) != "<part=1>\n" {
				t.Fatal("Redo", err)
			}
			fields := s.MessageFields(1)
			if _, err := s.SetMessage(1, "First line\nSecond line", fields[1].Value); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "message.alm")
			if _, err := s.SaveAs(path); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(filepath.Join(filepath.Dir(path), "message.text", "en", "event01.txt"))
			if err != nil || string(b) != "<part=1>\nFirst line\nSecond line" {
				t.Fatalf("page bytes %q: %v", b, err)
			}
			if _, err := s.SetMessage(1, "bad\x00text", fields[1].Value); err == nil {
				t.Fatal("NUL loss control accepted")
			}
			if !bytes.Equal(s.workspace.Pages["event01.txt"], b) {
				t.Fatal("rejection changed page")
			}
		})
	}
}

func TestMessageWizardAndRefusal(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := bytes.Clone(s.model.Bytes())
	if _, err := s.NewMessage(ui.InspectionMessageOwner{Kind: "Unit", Key: "value0"}); err == nil || !bytes.Equal(before, s.model.Bytes()) || len(s.workspace.Pages) != 0 {
		t.Fatal("invalid node changed workspace", err)
	}
	d, err := s.AddRecord(ui.InspectionNewRecord{Kind: "Message"})
	if err != nil || len(d.Script.Actions) != 2 || d.Script.Actions[1].Values[0] != 1 {
		t.Fatal("wizard", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) || len(s.workspace.Pages) != 0 {
		t.Fatal("wizard Undo", err)
	}
	for i := 1; i <= 64; i++ {
		s.workspace.Pages[messageFilename(uint32(i))] = []byte("<part=1>\n")
	}
	if _, err := s.NewMessage(ui.InspectionMessageOwner{Kind: "Message"}); err == nil || len(s.workspace.Pages) != 64 || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("page-limit refusal changed map/pages", err)
	}
}

func TestMessageSpeakerAndLossControls(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	workspacePage(t, e, "event01.txt", "<part=1,npc=51,mage>\nOne\nTwo<part=2>Three")
	fields := s.MessageFields(1)
	if fields[0].TextValue != "One\nTwo" || fields[1].Value != 51 {
		t.Fatal(fields)
	}
	before := bytes.Clone(s.workspace.Pages["event01.txt"])
	if d, err := s.SetMessage(1, fields[0].TextValue, 51); err != nil || d != nil || !bytes.Equal(before, s.workspace.Pages["event01.txt"]) {
		t.Fatal("unchanged page wrote bytes", err)
	}
	if _, err := s.SetMessage(1, "Edited\nLine", 51); err != nil || string(s.workspace.Pages["event01.txt"]) != "<part=1,npc=51,mage>\nEdited\nLine<part=2>Three" {
		t.Fatal("speaker/markup bytes", err, string(s.workspace.Pages["event01.txt"]))
	}
	before = bytes.Clone(s.workspace.Pages["event01.txt"])
	if _, err := s.SetMessage(1, "Rejected", 999); err == nil || !bytes.Equal(before, s.workspace.Pages["event01.txt"]) {
		t.Fatal("bad speaker changed page", err)
	}
}

func TestMessageSpeakerRolesLabelIDs21To24(t *testing.T) {
	roles := map[string][4]string{
		"":        {"Main hero", "Traitor mage", "Monk", "Prisoner"},
		"russian": {"Главный герой", "Маг-предатель", "Монах", "Узник"},
	}
	for language, want := range roles {
		t.Run("language "+language, func(t *testing.T) {
			_, e := workspaceFixture(t)
			s := e.Document().Edits.(*mapInspectionEdits)
			x := s.inspector
			x.archives = &Archives{}
			x.archives.Base.Profile.Language = language
			x.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{
				{ID: 21, Name: "Nyra"}, {ID: 22, Name: "A"}, {ID: 23, Name: "B"}, {ID: 24, Name: "C"}, {ID: 51, Name: "Guard"},
			}}}
			workspacePage(t, e, "event01.txt", "<part=1,npc=22>\nText")
			fields := s.MessageFields(1)
			var ids []int64
			labels := map[int64]string{}
			for _, c := range fields[1].Choices {
				ids = append(ids, c.Value)
				labels[c.Value] = c.Label
			}
			if !slices.Equal(ids, []int64{-1, 21, 22, 23, 24, 51}) {
				t.Fatal("choice order", ids)
			}
			for i, role := range want {
				id := int64(21 + i)
				if labels[id] != fmt.Sprintf("%d: %s", id, x.inspectionUTF8(role)) {
					t.Fatalf("id %d label %q", id, labels[id])
				}
			}
			if labels[51] != "51: Guard" {
				t.Fatal("other id changed", labels[51])
			}
			if fields[1].Value != 22 {
				t.Fatal("stored speaker", fields[1].Value)
			}
			if _, err := s.SetMessage(1, "Text", 23); err != nil || string(s.workspace.Pages["event01.txt"]) != "<part=1,npc=23>\nText" {
				t.Fatal("markup", err, string(s.workspace.Pages["event01.txt"]))
			}
		})
	}
}
