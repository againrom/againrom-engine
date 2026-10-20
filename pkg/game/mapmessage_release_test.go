package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func messageValueBytes(t *testing.T, source []byte, index int, value uint32) []byte {
	t.Helper()
	d, err := alm.OpenDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	for i, kind := range d.RecordTypeIDs() {
		if kind != 7 {
			continue
		}
		payload := d.RecordPayload(i)
		if bytes.Count(source, payload) != 1 {
			t.Fatal("script payload is not unique")
		}
		out := bytes.Clone(source)
		offset := bytes.Index(source, payload) + 4 + index*796 + 0x4c
		binary.LittleEndian.PutUint32(out[offset:], value)
		return out
	}
	t.Fatal("no script payload")
	return nil
}

func messageFrame(t *testing.T, e *ui.MapEditor, lang, name string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_MESSAGE_PICKER_FRAMES")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	pic, _, err := e.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	for _, crop := range []bool{false, true} {
		var img image.Image = pic
		suffix := ""
		if crop {
			img, suffix = pic.SubImage(image.Rect(580, 28, 1280, 650)), "-crop"
		}
		f, err := os.Create(filepath.Join(dir, lang+"-"+name+suffix+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
}

func TestReleaseMessagePickerPopulation(t *testing.T) {
	x := installedInspector(t)
	maps, values, missing, pages := 0, 0, 0, 0
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		before := bytes.Clone(s.model.Bytes())
		originalPages := cloneWorkspace(s.workspace).Pages
		maps++
		for name := range originalPages {
			if strings.HasPrefix(name, "event") {
				pages++
			}
		}
		for i, n := range e.Document().Script.Actions {
			if n.Opcode != 2 || n.Types[0] != 1 {
				continue
			}
			e.Select(recordIndex(t, e, "Action", i), false)
			messageField(t, e, 2)
			row, found := 0, false
			for j, target := range e.Document().Script.Targets[ui.ScriptMessageParameter] {
				if !target.NewMessage && target.Value == n.Values[0] {
					row, found = j, true
					break
				}
			}
			if !found {
				row = len(e.Document().Script.Targets[ui.ScriptMessageParameter])
				missing++
			}
			r, ok := e.DropdownRowRect(row)
			messageClick(t, e, r, ok)
			messageButton(t, e, 0)
			if !bytes.Equal(before, s.model.Bytes()) || s.Dirty() || !reflect.DeepEqual(originalPages, s.workspace.Pages) {
				t.Fatalf("%s action %d current value %d changed bytes/history/pages", key, i, n.Values[0])
			}
			values++
		}
	}
	if values == 0 || missing == 0 || pages == 0 {
		t.Fatal("empty population/loss control", values, missing, pages)
	}
	t.Logf("lang=%s maps=%d message_values=%d missing_values=%d event_pages=%d; every current choice writes zero ALM/page bytes", x.pageLanguage(), maps, values, missing, pages)
}

func TestReleaseMessagePickerCampaignPage(t *testing.T) {
	x := installedInspector(t)
	lang := x.pageLanguage()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	index := -1
	for i, n := range e.Document().Script.Actions {
		if n.Opcode == 2 && n.Types[0] == 1 {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("campaign map has no message")
	}
	e.Select(recordIndex(t, e, "Action", index), false)
	before := bytes.Clone(s.model.Bytes())
	number := freeMessage(s.workspace.Pages)
	messageField(t, e, 2)
	messageFrame(t, e, lang, "message-list")
	_ = e.Key("escape")
	_ = e.Key("escape")
	if err := e.Key("field"); err != nil {
		t.Fatal(err)
	}
	for i, f := range s.Fields("Action", index) {
		if f.Key == "value0" {
			for range i {
				_ = e.Key("tab")
			}
			break
		}
	}
	_ = e.Key("clear")
	e.TextInput(strconv.FormatInt(ui.NewMessageChoice, 10))
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(e.PopupLines()) != 2 || len(s.workspace.Pages[messageFilename(number)]) != 0 || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("E New did not open an unmade page", e.PopupLines())
	}
	fields := s.MessageFields(ui.NewMessagePage)
	if len(fields[1].Choices) < 2 {
		t.Fatal("npc.reg has no speakers")
	}
	body := map[string]string{"en": "A message for this map.\nThe speaker has a second line.", "ru": "Сообщение для этой карты.\nУ говорящего есть вторая строка."}[lang]
	messageField(t, e, 0)
	e.TextInput(body)
	messageField(t, e, 1)
	r, ok := e.DropdownRowRect(1)
	messageClick(t, e, r, ok)
	messageButton(t, e, 0)
	speaker := fields[1].Choices[1].Value
	want := []byte(fmt.Sprintf("<part=1,npc=%d>\n%s", speaker, body))
	if !bytes.Equal(want, s.workspace.Pages[messageFilename(number)]) {
		t.Fatal("page bytes/speaker")
	}
	messageFrame(t, e, lang, "page-popup")
	messageButton(t, e, 2)
	m, _ := s.model.Map()
	sc, _ := m.Script()
	if sc.Actions[index].Value[0] != number {
		t.Fatal("node value differs from page number")
	}
	if !bytes.Equal(messageValueBytes(t, before, index, number), s.model.Bytes()) {
		t.Fatal("ALM changes exceed the chosen value word")
	}
	root := t.TempDir()
	output := filepath.Join(root, "saved", "saved.alm")
	if err := os.Mkdir(filepath.Dir(output), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAs(output); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(output); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if !bytes.Equal(want, s.workspace.Pages[messageFilename(number)]) {
		t.Fatal("cold Open lost page")
	}
	for range 2 {
		if _, err := s.AddScript("Condition"); err != nil {
			t.Fatal(err)
		}
	}
	m, _ = s.model.Map()
	sc, _ = m.Script()
	a, b := sc.Conditions[len(sc.Conditions)-2].ID, sc.Conditions[len(sc.Conditions)-1].ID
	if _, err := s.AddScript("Trigger"); err != nil {
		t.Fatal(err)
	}
	_, err := s.SetFields("Trigger", s.model.Count(mapedit.KindTrigger)-1, []ui.InspectionFieldValue{{Key: "left0", Value: int64(a)}, {Key: "right0", Value: int64(b)}, {Key: "action0", Value: int64(sc.Actions[index].ID)}})
	if err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(root, "recording.exe")
	writeTestPlayInput(t, program, []byte("recording host"))
	if err := os.Mkdir(filepath.Join(root, "profile"), 0700); err != nil {
		t.Fatal(err)
	}
	starts := 0
	x.SetTestPlay(TestPlayHost{Game: program, Profile: func() (RuntimeProfile, error) {
		return ResolveRuntimeProfile(filepath.Join(root, "profile", "saves"), "", "", x.archives.Root)
	}, Start: func(string, []string) error { starts++; return nil }})
	if _, err := s.TestPlay(); err != nil || starts != 1 {
		t.Fatal("TestPlay staging", starts, err)
	}
	mods := filepath.Join(root, "profile", TestPlayModID)
	loaded := loadWorkspaceMod(t, x.archives.Root, filepath.Join(mods, TestPlayModID))
	f := guestFront(t, mods, TestPlayModID)
	p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: loaded.Number})
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: loaded.Number})
	if err != nil {
		t.Fatal(err)
	}
	control.ann = nil
	for range 64 {
		p.step(nil)
		control.step(nil)
		if p.World.Hash() != control.World.Hash() {
			t.Fatal("announcement changed hashed state")
		}
	}
	found := false
	for _, ann := range p.announced {
		if ann.Message == int32(number) && ann.Shown && strings.Contains(strings.Join(ann.Parts, ""), strings.Split(body, "\n")[0]) {
			found = true
		}
	}
	if !found {
		t.Fatalf("headless test play did not show new page: %+v", p.announced)
	}
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != loaded.Number {
		t.Fatal("guest route", next)
	}
	if err := app.OpenMission(f.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
	savBefore := guestMissionSave(t, f)
	if !f.live.openDialogue(int(number)) {
		t.Fatal("dialogue consumer did not show page")
	}
	if !bytes.Equal(savBefore, guestMissionSave(t, f)) {
		t.Fatal("dialogue changed SAV bytes")
	}
	export := filepath.Join(root, s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(export); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(export, "mission", "text", lang, messageFilename(number)))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("W page bytes", err)
	}
	t.Logf("lang=%s campaign=10 page=%d speaker=%d bytes=%d SAV=%d unchanged; only chosen ALM value word changed; SaveAs/cold Open/W exact page bytes; guaranteed test trigger shows authored page headlessly; consumer-free hash control", lang, number, speaker, len(want), len(savBefore))
}
