package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/installinfo"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// dlgReplyAt selects the reply whose head starts with n.
func dlgReplyAt(t *testing.T, e *ui.MapEditor, n string) {
	t.Helper()
	dlgChoose(t, e, "d.reply", n+". ")
}

// millMap is a 40 by 40 map with a drop location whose script holds four
// constants and two actions that set variables 10 and 12, so both are
// variables the map names.
func millMap(t *testing.T) []byte {
	t.Helper()
	conditions := make([]scriptNode, 4)
	for i := range conditions {
		conditions[i] = scriptNode{label: "Constant", opcode: 0x10002, id: uint32(i + 1), tags: []uint32{kInt}}
	}
	actions := []scriptNode{
		{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "ten", opcode: 3, id: 2, tags: []uint32{kInt, kInt}, values: []uint32{10, 0}},
		{label: "twelve", opcode: 3, id: 3, tags: []uint32{kInt, kInt}, values: []uint32{12, 0}},
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: goalScript(actions, conditions, []goalTrigger{{name: "T", actions: []uint32{3}}})}
	return built(t, o, nil, nil).Bytes()
}

// buildMill drives the dialogue window to the peasant who asks the player to
// drive the goblins from the mill, and applies it.
func buildMill(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	startDialogueRow(t, e, "mill")
	dlgType(t, e, "d.name", "start")
	dlgType(t, e, "d.text.en", "Goblins hold the old mill. Will you drive them out?")
	dlgType(t, e, "d.text.ru", "Гоблины заняли старую мельницу. Прогоните их?")
	dlgPress(t, e, "d.node", "addnode")
	dlgType(t, e, "d.name", "thanks")
	dlgType(t, e, "d.text.en", "Thank you, the mill is ours again.")
	dlgType(t, e, "d.text.ru", "Спасибо, мельница снова наша.")
	dlgChoose(t, e, "d.node", "start")

	reply := func(n int, en, ru string) {
		dlgPress(t, e, "d.reply", "addreply")
		dlgType(t, e, "d.r.en", en)
		dlgType(t, e, "d.r.ru", ru)
	}
	reply(1, "I will drive them out.", "Я прогоню их.")
	dlgChoose(t, e, "d.next", "thanks")
	dlgChoose(t, e, "d.once", "Yes")
	dlgAdd(t, e, true)
	dlgChoose(t, e, "a0.item", "Set a variable")
	dlgChoose(t, e, "a0.p0", "10")
	dlgType(t, e, "a0.p1", "1")

	reply(2, "Pay me first.", "Сначала заплатите.")
	dlgChoose(t, e, "d.once", "Yes")
	dlgAdd(t, e, true)
	dlgChoose(t, e, "a0.item", "Give gold")

	reply(3, "Not now.", "Не сейчас.")

	reply(4, "Tell me about the mill again.", "Расскажите о мельнице еще раз.")
	dlgChoose(t, e, "d.unavailable", "Disabled")
	dlgAdd(t, e, false)
	dlgChoose(t, e, "c0.item", "Variable compared with a number")
	dlgChoose(t, e, "c0.p0", "12")
}

// millMod is an editor mod with one guest mission on millMap, opened in the
// map editor's mod window: the editor, its fixture and the mod's folder.
func millMod(t *testing.T, book string) (*testPlayFixture, *ui.MapEditor, string) {
	t.Helper()
	f, e := workspaceFixture(t)
	f.x.archives.Base.Profile.Language = map[string]string{"": "english", "ru": "russian"}[book]
	e.SetWords(words.For(book))
	e.Layout(1280, 800)
	raw := millMap(t)
	f.x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return raw, nil })
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='mill'\nnumber=21\nmap='mill.alm'\nafter=20\nparty=[{character='npc22'}]\n")
	writeTestPlayInput(t, filepath.Join(dir, "mill.alm"), raw)
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	for _, folder := range []string{dir, own} {
		if err := workspaceField(e, "package", 0, folder); err != nil {
			t.Fatal(err)
		}
	}
	if err := modMapsOpen(t, e, 1); err != nil {
		t.Fatal(err)
	}
	return f, e, own
}

func TestMillModSavesItsDialogue(t *testing.T) {
	_, e, own := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	buildMill(t, e)
	dlgApply(t, e)
	saveMod(t, e)
	saved := tree(t, own)
	for _, rel := range millModFiles {
		if _, ok := saved[rel]; !ok {
			t.Fatalf("the saved mod holds no %s: %v", rel, saved)
		}
	}
	if _, ok := saved["maps/mill.alm"]; ok {
		t.Fatal("the map with an interactive dialogue stayed an .alm")
	}
	if !strings.Contains(saved[millDialogueFile], "Гоблины заняли старую мельницу") {
		t.Fatalf("the saved dialogue:\n%s", saved[millDialogueFile])
	}
	if out := os.Getenv("AGAINROM_MILL_OUT"); out != "" {
		for _, rel := range millModFiles {
			p := filepath.Join(out, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(saved[rel]), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for _, rel := range millModFiles {
		b, err := os.ReadFile(filepath.Join(millModFolder, filepath.FromSlash(rel)))
		if err != nil || string(b) != saved[rel] {
			t.Errorf("the committed fixture's %s is not what the editor saves (%v)", rel, err)
		}
	}
}

// millModFolder is the committed mod the editor test saves and the playing
// test loads; millModFiles are its files.
const (
	millModFolder    = "testdata/dialogue-mill"
	millDialogueFile = "maps/mill.dialogs/mill.toml"
)

var millModFiles = []string{"mod.toml", "main.star", "data/missions.toml", "maps/mill.almx", millDialogueFile}

// The committed mill mod is loaded by the game on each language: the
// peasant's first node offers the four replies in that language and a pick
// of the first sets variable 10 and pages to the thanks node.
func TestMillModPlaysOnBothLanguages(t *testing.T) {
	read := func(rel string) []byte {
		b, err := os.ReadFile(filepath.Join(millModFolder, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, lang := range []struct {
		name, selector, node, first string
		reply                       string
	}{
		{"en", "0", "Goblins hold the old mill. Will you drive them out?", "I will drive them out.", "Thank you"},
		{"ru", "1", "Гоблины заняли старую мельницу. Прогоните их?", "Я прогоню их.", "Спасибо"},
	} {
		t.Run(lang.name, func(t *testing.T) {
			f := extendedScriptFront(t, read("maps/mill.almx"))
			f.Archives.Containers.Overlay(dialogueFilePath(10, "mill"), func() ([]byte, error) { return read(millDialogueFile), nil })
			f.Archives.Containers.Overlay(installinfo.LanguagePath, func() ([]byte, error) { return []byte("id" + lang.selector), nil })
			selector := map[string]int{"en": 0, "ru": 1}[lang.name]
			w := openExtendedMission(t, f, 40)
			mw := f.live
			if mw.mission.dialogue == nil || len(mw.mission.replies) != 4 {
				t.Fatalf("the mill dialogue did not open with four replies: %+v", mw.mission.replies)
			}
			v := mw.view
			if text, _, _ := v.NoticeState(); text != EncodeModText(lang.node, selector) {
				t.Fatalf("the first node shows %q", text)
			}
			if mw.mission.replies[0].text != EncodeModText(lang.first, selector) {
				t.Fatalf("the first reply shows %q", mw.mission.replies[0].text)
			}
			mw.advanceNotice(ui.NoticeReply1)
			missionSteps(mw, 1)
			if got := w.ScriptRegister(10); got != 1 {
				t.Fatalf("the pick left variable 10 at %d", got)
			}
			if text, _, open := v.NoticeState(); !open || !strings.HasPrefix(text, EncodeModText(lang.reply, selector)) {
				t.Fatalf("the pick did not page to the thanks node: %q open %v", text, open)
			}
		})
	}
}

// Publish carries the dialogue file: the published mod loads with it.
func TestMillModPublishCarriesTheDialogueFile(t *testing.T) {
	_, e, own := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	buildMill(t, e)
	dlgApply(t, e)
	saveMod(t, e)
	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{filepath.Base(own)})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	res, err := modrt.Load(entries, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatalf("the published mod is refused: %v", err)
	}
	if len(res.Missions.Missions) != 1 || len(res.Missions.Missions[0].DialogueFiles["mill"]) == 0 {
		t.Fatalf("the published mission holds no dialogue file: %+v", res.Missions.Missions)
	}
}

// The mod window's Dialogues page lists the map's dialogues with their nodes
// and users, Open shows one and New dialogue starts one named in its window.
func TestDialoguesPageListsOpensAndStartsDialogues(t *testing.T) {
	_, e, _ := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	if !s.WorkspaceHolds(ui.ModDialoguesScope) {
		t.Fatal("a mod's map shows no Dialogues page")
	}
	if fields := s.WorkspaceFields(ui.ModDialoguesScope); len(fields) != 1 || fields[0].Key != ui.DialogueNewKey {
		t.Fatalf("an empty page holds %+v", fields)
	}
	buildMill(t, e)
	dlgApply(t, e)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
	trigClick(t, e, r, ok, "Cancel")
	var lines []string
	for _, f := range s.WorkspaceFields(ui.ModDialoguesScope) {
		lines = append(lines, f.Key+" "+f.Label+" "+f.TextValue)
	}
	want := "info.mill mill 2 nodes. Started by Mill"
	if len(lines) != 3 || lines[0] != want || lines[1] != "open.mill Open dialogue " || lines[2] != "new New dialogue " {
		t.Fatalf("the Dialogues page:\n%q", lines)
	}
	if err := e.OpenDialogue("mill"); err != nil || !e.DialogueOpen() {
		t.Fatalf("Open showed no dialogue: %v", err)
	}
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
	trigClick(t, e, r, ok, "Cancel")
	if err := e.NewDialogue(); err != nil || !e.DialogueOpen() {
		t.Fatalf("New dialogue showed no window: %v", err)
	}
	dlgType(t, e, "d.file", "second")
	dlgType(t, e, "d.text.en", "Hello")
	dlgApply(t, e)
	if got := s.DialogueNames(); len(got) != 2 || got[1] != "second" {
		t.Fatalf("the new dialogue was not written:\n%s", trigLines(e))
	}
}

// Save mod refuses a dialogue that a trigger starts and no file holds, with
// the dialogue's name.
func TestSaveModRefusesAnInteractiveActionWithNoDialogueFile(t *testing.T) {
	_, e, _ := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e, "mill")
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
	trigClick(t, e, r, ok, "Cancel")
	_ = e.Key("escape")
	if _, err := s.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), "dialogue mill, which has no file") {
		t.Fatalf("Save mod took a dialogue with no file: %v", err)
	}
}

// The second reply pays gold once, and the once mark keeps it from being
// offered again.
func TestMillModSecondReplyPaysGoldOnce(t *testing.T) {
	read := func(rel string) []byte {
		b, err := os.ReadFile(filepath.Join(millModFolder, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := extendedScriptFront(t, read("maps/mill.almx"))
	f.Archives.Containers.Overlay(dialogueFilePath(10, "mill"), func() ([]byte, error) { return read(millDialogueFile), nil })
	w := openExtendedMission(t, f, 40)
	mw := f.live
	gold := w.Purse(1)
	mw.advanceNotice(ui.NoticeReply2)
	missionSteps(mw, 1)
	if w.Purse(1) <= gold {
		t.Fatalf("the gold reply paid nothing: %d then %d", gold, w.Purse(1))
	}
	if !mw.openInteractive(0) || len(mw.mission.replies) != 3 {
		t.Fatalf("the paid reply is offered again: %+v", mw.mission.replies)
	}
}

func TestMillDialogueBuiltThroughTheEditor(t *testing.T) {
	e, _ := openGoalMap(t, millMap(t))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	buildMill(t, e)
	if got := e.DialogueChecks(); len(got) != 0 {
		t.Fatalf("the finished dialogue is refused: %v", got)
	}
	dlgApply(t, e)
	if e.DialogueOpen() && !strings.Contains(trigLines(e), "start") {
		t.Fatalf("Apply left the window in a strange state:\n%s", trigLines(e))
	}
	d, err := s.ReadDialogue("mill")
	if err != nil {
		t.Fatalf("%v (open %v, names %v):\n%s", err, e.DialogueOpen(), s.DialogueNames(), trigLines(e))
	}
	if len(d.Nodes) != 2 || len(d.Nodes[0].Replies) != 4 {
		t.Fatalf("the saved dialogue is not the built one: %+v", d)
	}
}

// A once reply read from a dialogue file with mark 0 keeps that mark through
// another edit and Apply.
func TestDialogueOnceMarkZeroSurvivesAnEdit(t *testing.T) {
	_, e, _ := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	data := ui.DialogueData{Nodes: []ui.DialogueNodeData{
		{ID: 1, Name: "start", Text: map[string]string{"en": "Hello"}, Replies: []ui.DialogueReplyData{
			{Text: map[string]string{"en": "Once"}, Once: true, Mark: 0},
		}},
	}}
	if _, err := s.WriteDialogue("zero", data); err != nil {
		t.Fatal(err)
	}
	if err := e.OpenDialogue("zero"); err != nil {
		t.Fatal(err)
	}
	dlgType(t, e, "d.text.en", "Hello again")
	dlgApply(t, e)
	got, err := s.ReadDialogue("zero")
	if err != nil || len(got.Nodes) != 1 || got.Nodes[0].Text["en"] != "Hello again" {
		t.Fatalf("read back %+v %v", got, err)
	}
	if r := got.Nodes[0].Replies[0]; !r.Once || r.Mark != 0 {
		t.Fatalf("the once mark was rewritten: %+v", r)
	}
}
