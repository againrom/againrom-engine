package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func TestReleaseMessageCodecCensus(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	maps, files, pages, edits := 0, 0, 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		maps++
		s := e.Document().Edits.(*mapInspectionEdits)
		var names []string
		for name := range s.workspace.Pages {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			raw := string(s.workspace.Pages[name])
			d := decodeMessageMarkup(raw)
			if d.encode() != raw {
				t.Fatalf("%s/%s codec changed bytes", key, name)
			}
			files++
			if !strings.HasPrefix(name, "event") || len(d.pages()) == 0 {
				continue
			}
			var number uint32
			if _, err := fmt.Sscanf(name, "event%d.txt", &number); err != nil {
				t.Fatal(err)
			}
			parts := s.MessageParts(number)
			before := bytes.Clone(s.workspace.Pages[name])
			alm := bytes.Clone(s.model.Bytes())
			if doc, err := s.SetMessageParts(number, parts); err != nil || doc != nil || !bytes.Equal(before, s.workspace.Pages[name]) {
				t.Fatalf("%s/%s unchanged Apply: %v", key, name, err)
			}
			for i := range parts {
				pages++
				next := slices.Clone(parts)
				next[i].Body += "!"
				if _, err := s.SetMessageParts(number, next); err != nil {
					t.Fatal(key, name, i, err)
				}
				end := d.Tags[parts[i].Source].End
				end = messageTextEnd(raw, end)
				want := raw[:end] + "!" + raw[end:]
				if string(s.workspace.Pages[name]) != want || !bytes.Equal(alm, s.model.Bytes()) {
					t.Fatalf("%s/%s page %d changed another byte", key, name, i)
				}
				if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.workspace.Pages[name]) {
					t.Fatal("Undo", err)
				}
				edits++
			}
		}
	}
	if files == 0 || pages == 0 || edits != pages {
		t.Fatal("empty census", files, pages, edits)
	}
	t.Logf("lang=%s maps=%d text_files=%d page_arms=%d exact_single_page_edits=%d; no-edit codec and popup byte-equal", x.pageLanguage(), maps, files, pages, edits)
}

func TestReleaseDialogueShippedPagePopup(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	for i, action := range e.Document().Script.Actions {
		parts := s.MessageParts(action.Values[0])
		if action.Opcode != 2 || len(parts) < 3 {
			continue
		}
		e.Select(recordIndex(t, e, "Action", i), false)
		for row, line := range e.PopupLines() {
			if strings.HasPrefix(line, x.inspectionUTF8(x.word("editor.param.open"))) {
				r, ok := e.PopupRowRect(row)
				messageClick(t, e, r, ok)
				break
			}
		}
		before := bytes.Clone(s.workspace.Pages[messageFilename(action.Values[0])])
		dialogueField(t, e, 2)
		_ = e.Key("ctrl+end")
		e.TextInput("!")
		messageFrame(t, e, x.pageLanguage(), "shipped-dialogue-pages")
		messageButton(t, e, 0)
		d := decodeMessageMarkup(string(before))
		end := d.Tags[parts[1].Source].End
		end = messageTextEnd(string(before), end)
		if got := string(s.workspace.Pages[messageFilename(action.Values[0])]); got != string(before[:end])+"!"+string(before[end:]) {
			t.Fatal("editing shipped page 2 changed pages 1 or 3")
		}
		path := filepath.Join(t.TempDir(), "dialogue.alm")
		if _, err := s.SaveAs(path); err != nil {
			t.Fatal(err)
		}
		if err := e.Open(path); err != nil {
			t.Fatal(err)
		}
		cold := e.Document().Edits.(*mapInspectionEdits)
		if !bytes.Equal(cold.workspace.Pages[messageFilename(action.Values[0])], s.workspace.Pages[messageFilename(action.Values[0])]) {
			t.Fatal("cold Open lost page 2 edit")
		}
		t.Logf("lang=%s mission=10 event=%d arms=%d page2 edit only, SaveAs/cold Open exact", x.pageLanguage(), action.Values[0], len(parts))
		return
	}
	t.Fatal("mission 10 has no linked message with three pages")
}

func TestReleaseDialogueThreeSpeakersTestPlay(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	index := -1
	for i, action := range e.Document().Script.Actions {
		if action.Opcode == 2 && action.Types[0] == 1 {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("no message action")
	}
	e.Select(recordIndex(t, e, "Action", index), false)
	messageField(t, e, 2)
	e.ScrollDropdown(1000)
	row := len(e.DropdownChoices()) - 1
	r, ok := e.DropdownRowRect(row)
	messageClick(t, e, r, ok)
	number := freeMessage(s.workspace.Pages)
	for part := 0; part < 3; part++ {
		if part > 0 {
			dialogueButton(t, e, part-1, 1)
		}
		dialogueField(t, e, 2*part)
		e.TextInput(fmt.Sprintf("Dialogue line %d", part+1))
		dialogueField(t, e, 2*part+1)
		chosen := -1
		for i, choice := range e.DropdownChoices() {
			if strings.HasPrefix(choice, fmt.Sprintf("%d:", part+21)) {
				chosen = i
			}
		}
		if chosen < 0 {
			t.Fatal("role speaker absent")
		}
		e.ScrollDropdown(-1000)
		for {
			if r, ok := e.DropdownRowRect(chosen); ok {
				messageClick(t, e, r, ok)
				break
			}
			e.ScrollDropdown(1)
		}
	}
	messageButton(t, e, 0)
	messageFrame(t, e, x.pageLanguage(), "three-speakers-dialogue")
	root := t.TempDir()
	program := filepath.Join(root, "recording.exe")
	writeTestPlayInput(t, program, []byte("recording host"))
	if err := os.Mkdir(filepath.Join(root, "profile"), 0700); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := s.AddScript("Condition"); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := s.model.Map()
	sc, _ := m.Script()
	if _, err := s.AddScript("Trigger"); err != nil {
		t.Fatal(err)
	}
	_, err := s.SetFields("Trigger", s.model.Count(mapedit.KindTrigger)-1, []ui.InspectionFieldValue{{Key: "left0", Value: int64(sc.Conditions[len(sc.Conditions)-2].ID)}, {Key: "right0", Value: int64(sc.Conditions[len(sc.Conditions)-1].ID)}, {Key: "action0", Value: int64(sc.Actions[index].ID)}})
	if err != nil {
		t.Fatal(err)
	}
	x.SetTestPlay(TestPlayHost{Game: program, Profile: func() (RuntimeProfile, error) {
		return ResolveRuntimeProfile(filepath.Join(root, "profile", "saves"), "", "", x.archives.Root)
	}, Start: func(string, []string) error { return nil }})
	if _, err := s.TestPlay(); err != nil {
		t.Fatal(err)
	}
	mods := filepath.Join(root, "profile", TestPlayModID)
	loaded := loadWorkspaceMod(t, x.archives.Root, filepath.Join(mods, TestPlayModID))
	if !bytes.Equal(loaded.TextFiles[messageFilename(number)], s.workspace.Pages[messageFilename(number)]) {
		t.Fatal("test play staging changed the dialogue bytes")
	}
	f := guestFront(t, mods, TestPlayModID)
	p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: loaded.Number})
	if err != nil {
		t.Fatal(err)
	}
	for range 64 {
		p.step(nil)
	}
	found := false
	for _, ann := range p.announced {
		if ann.Message == int32(number) && ann.Shown && len(ann.Parts) == 3 {
			found = true
		}
	}
	if !found {
		t.Fatal("test play lost the three-page message", p.announced)
	}
	payload := EncodeInstallTextFile(s.workspace.Pages[messageFilename(number)], x.font.Selector)
	for part := 1; part <= 3; part++ {
		body, ok := EventPart(payload, part, EventAudience{})
		speaker, named := EventPartSpeaker(payload, part, EventAudience{})
		if !ok || !strings.Contains(body, fmt.Sprintf("Dialogue line %d", part)) || !named || speaker != part+20 {
			t.Fatal("test play text or speaker order", part, body, speaker)
		}
	}
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != loaded.Number {
		t.Fatal("test play guest route", next)
	}
	if err := app.OpenMission(f.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
	beforeSave := guestMissionSave(t, f)
	if !f.live.openDialogue(int(number)) {
		t.Fatal("staged dialogue did not open")
	}
	for part := 1; part <= 3; part++ {
		body, _, open := f.LiveNotice()
		portrait, face := f.live.view.NoticeSpeaker()
		expected, _ := f.live.mission.speakerFace(part + 20)
		if !open || f.live.mission.part != part || !strings.Contains(body, fmt.Sprintf("Dialogue line %d", part)) || !portrait || face == nil || expected == nil || !bytes.Equal(face.Pix, expected.Pix) {
			t.Fatal("live test play lost page or speaker order", part, body, portrait)
		}
		if !bytes.Equal(beforeSave, guestMissionSave(t, f)) {
			t.Fatal("dialogue paging changed SAV bytes", part)
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	export := filepath.Join(root, s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(export); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(filepath.Join(export, "mission", "text", x.pageLanguage(), messageFilename(number)))
	if err != nil || !bytes.Equal(saved, s.workspace.Pages[messageFilename(number)]) {
		t.Fatal("mod export changed dialogue", err)
	}
	t.Logf("lang=%s new_message=%d pages=3 speakers=21,22,23; recording test-play package, headless announcement, live pages/faces and unchanged SAV; mod export exact", x.pageLanguage(), number)
}
