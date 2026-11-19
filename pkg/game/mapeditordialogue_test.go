package game

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func dlgField(t *testing.T, e *ui.MapEditor, key string) int {
	t.Helper()
	n := e.DialogueField(key)
	if n < 0 {
		t.Fatalf("the dialogue window shows no field %s:\n%s", key, trigLines(e))
	}
	return n
}

// dlgType replaces the text of the dialogue window's field key with s.
func dlgType(t *testing.T, e *ui.MapEditor, key, s string) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: dlgField(t, e, key)})
	r.Min.X = max(r.Min.X, r.Max.X-4)
	trigClick(t, e, r, ok, key)
	for i := 0; i < 80; i++ {
		_ = e.Key("backspace")
	}
	e.TextInput(s)
}

// dlgChoose picks the choice of the dialogue window's field key that has want.
func dlgChoose(t *testing.T, e *ui.MapEditor, key, want string) {
	t.Helper()
	trigChoose(t, e, dlgField(t, e, key), want)
}

// dlgPress presses the first shown button that does act.
func dlgPress(t *testing.T, e *ui.MapEditor, act string) {
	t.Helper()
	r, ok := e.DialogueButton(act)
	trigClick(t, e, r, ok, act)
}

// dlgAdd presses the Add condition (action false) or Add outcome button.
func dlgAdd(t *testing.T, e *ui.MapEditor, action bool) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 0, Flag: action})
	trigClick(t, e, r, ok, "add row")
}

func dlgApply(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	trigClick(t, e, r, ok, "Apply")
}

// dlgCancel presses the window's Cancel.
func dlgCancel(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
	trigClick(t, e, r, ok, "Cancel")
}

// dlgSelect shows the line at place rank by a click on its list entry.
func dlgSelect(t *testing.T, e *ui.MapEditor, rank int) {
	t.Helper()
	r, ok := e.DialogueListRow(rank)
	trigClick(t, e, r, ok, "list entry")
}

// dlgReply opens reply i of the line on show by a click on its header.
func dlgReply(t *testing.T, e *ui.MapEditor, i int) {
	t.Helper()
	r, ok := e.DialogueReplyRow(i)
	trigClick(t, e, r, ok, "reply header")
}

// dlgAddLine presses the list's + button and types text into the new line.
func dlgAddLine(t *testing.T, e *ui.MapEditor, text string) {
	t.Helper()
	dlgPress(t, e, "addline")
	dlgType(t, e, ui.DialogueLineText, text)
}

// startDialogueRow opens a trigger with a Start dialog action and opens the
// dialog window through the action's Configure button.
func startDialogueRow(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	wizNewTrigger(t, e, "Mill")
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Start dialog")
	wizPress(t, e, "a0.item", 0)
}

// A dialog made of lines that only follow one another is written as message
// pages: no dialogue file is made and the map has no interactive action, so
// a map in the script form is asked to fold back to .alm.
func TestPlainDialogMadeInTheWindowIsWrittenAsPages(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	if e.DialogueLines() != 1 {
		t.Fatalf("a new dialog lists %d lines:\n%s", e.DialogueLines(), trigLines(e))
	}
	dlgType(t, e, ui.DialogueLineText, "First")
	dlgAddLine(t, e, "Second")
	dlgApply(t, e)
	if len(s.DialogueNames()) != 0 {
		t.Fatalf("a plain dialog made a dialogue file: %v", s.DialogueNames())
	}
	sc := wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the action is not a plain message: %+v", sc.Actions)
	}
	found := false
	for page := uint32(1); page < 8 && !found; page++ {
		parts := s.MessageParts(page)
		found = len(parts) == 2 && parts[0].Body == "First" && parts[1].Body == "Second"
	}
	if !found {
		t.Fatalf("no dialog page holds the two lines")
	}
	if _, err := s.SaveInPlace(); err == nil || !errors.As(err, new(*ui.ScriptFormNeeded)) {
		t.Fatalf("a map with no interactive action did not ask to fold: %v", err)
	}
}

// A reply added to a line writes the dialogue file and makes the action
// interactive, so the map needs .almx; taking the reply away folds the
// dialog back to pages and removes the file.
func TestAReplyMakesTheDialogueAFileAndRemovingItFoldsBack(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	dlgType(t, e, ui.DialogueLineText, "Hello")
	dlgChoose(t, e, "d.after", "Reply choice")
	dlgType(t, e, "d.rt", "Hi")
	dlgApply(t, e)
	names := s.DialogueNames()
	if len(names) != 1 {
		t.Fatalf("a reply choice wrote no dialogue file:\n%s", trigLines(e))
	}
	sc := wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || !sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the action is not interactive: %+v", sc.Actions)
	}
	d, err := s.ReadDialogue(names[0])
	if err != nil || len(d.Nodes) != 1 || len(d.Nodes[0].Replies) != 1 || d.Nodes[0].Replies[0].Text["en"] != "Hi" {
		t.Fatalf("the file holds %+v (%v)", d, err)
	}
	if !e.DialogueOpen() {
		t.Fatalf("the window closed:\n%s", trigLines(e))
	}
	dlgPress(t, e, "removereply")
	dlgChoose(t, e, "d.after", "End dialogue")
	dlgApply(t, e)
	if got := s.DialogueNames(); len(got) != 0 {
		t.Fatalf("the file stayed after the last reply went: %v", got)
	}
	sc = wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the action stayed interactive: %+v", sc.Actions)
	}
	if _, err := s.SaveInPlace(); err == nil || !errors.As(err, new(*ui.ScriptFormNeeded)) {
		t.Fatalf("a map with no interactive action did not ask to fold: %v", err)
	}
}

// replyFileWindow makes a dialogue file with one reply through the window,
// leaves the window and returns the host and the file's name.
func replyFileWindow(t *testing.T) (*ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	dlgType(t, e, ui.DialogueLineText, "Hello")
	dlgChoose(t, e, "d.after", "Reply choice")
	dlgType(t, e, "d.rt", "Hi")
	dlgApply(t, e)
	names := s.DialogueNames()
	if len(names) != 1 {
		t.Fatalf("a reply choice wrote no dialogue file:\n%s", trigLines(e))
	}
	dlgCancel(t, e)
	return e, s, names[0]
}

// A file that holds a text in another language stays a file when its last
// reply is removed, with every text in it.
func TestDialogueFileWithOtherLanguageSurvivesRemovingEveryReply(t *testing.T) {
	e, s, name := replyFileWindow(t)
	d, err := s.ReadDialogue(name)
	if err != nil {
		t.Fatal(err)
	}
	d.Nodes[0].Text["ru"] = "Привет"
	d.Nodes[0].Replies[0].Text["ru"] = "Пока"
	if _, err := s.WriteDialogue(name, d); err != nil {
		t.Fatal(err)
	}
	wizPress(t, e, "a0.item", 0)
	dlgPress(t, e, "removereply")
	dlgChoose(t, e, "d.after", "End dialogue")
	dlgApply(t, e)
	if got := s.DialogueNames(); len(got) != 1 || got[0] != name {
		t.Fatalf("the file did not stay: %v", got)
	}
	got, err := s.ReadDialogue(name)
	if err != nil || got.Nodes[0].Text["ru"] != "Привет" || got.Nodes[0].Text["en"] != "Hello" {
		t.Fatalf("the texts were lost: %+v (%v)", got, err)
	}
	sc := wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || !sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the action stopped starting the file: %+v", sc.Actions)
	}
}

// A file written by hand with no reply, only a next chain, is kept by Apply
// as it stands.
func TestHandWrittenReplylessDialogueFileSurvivesApply(t *testing.T) {
	e, s, name := replyFileWindow(t)
	d := ui.DialogueData{Nodes: []ui.DialogueNodeData{
		{ID: 1, Text: map[string]string{"en": "One", "ru": "Раз"}, Next: 2},
		{ID: 2, Text: map[string]string{"en": "Two", "ru": "Два"}},
	}}
	if _, err := s.WriteDialogue(name, d); err != nil {
		t.Fatal(err)
	}
	want, err := s.ReadDialogue(name)
	if err != nil {
		t.Fatal(err)
	}
	wizPress(t, e, "a0.item", 0)
	dlgApply(t, e)
	if got := s.DialogueNames(); len(got) != 1 || got[0] != name {
		t.Fatalf("the file did not stay: %v", got)
	}
	got, err := s.ReadDialogue(name)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Apply changed the file:\n%+v\nwas\n%+v (%v)", got, want, err)
	}
}

// On a full HD screen the window's buttons and the fields of a reply with a
// condition and an outcome all lie on the screen.
func TestDialogueWindowFitsAFullHdScreen(t *testing.T) {
	e, _ := openGoalMap(t, millMap(t))
	if _, err := formHost(e).ExtendScript(); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	buildMill(t, e)
	screen := image.Rect(0, 0, 1920, 1080)
	for n := 0; n < 2; n++ {
		if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n}); !ok || !r.In(screen) {
			t.Fatalf("window button %d is %v, found %v", n, r, ok)
		}
	}
	for _, key := range []string{ui.DialogueLineText, "d.after", "d.rt", "d.rnext", "d.ronce", "d.runavail", "c0.item"} {
		if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: dlgField(t, e, key)}); !ok || !r.In(screen) {
			t.Fatalf("field %s is %v, found %v", key, r, ok)
		}
	}
}

func TestDialogueWindowOpensFromTheStartDialogRow(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e)
	if !strings.Contains(trigLines(e), "After the line") {
		t.Fatalf("the Configure button opened no dialog window:\n%s", trigLines(e))
	}
}

// typeAt replaces the text of the window's field n with s.
func typeAt(t *testing.T, e *ui.MapEditor, n int, s string) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
	r.Min.X = max(r.Min.X, r.Max.X-4)
	trigClick(t, e, r, ok, "field")
	for i := 0; i < 80; i++ {
		_ = e.Key("backspace")
	}
	e.TextInput(s)
}

// A speaker and a sex split made in the window live in the dialogue file and
// come back on opening it again; the same split on a plain dialog lives in
// the message page's tags (the pages store).
func TestSpeakerAndHeroVariantsSurviveBothStores(t *testing.T) {
	open := func(t *testing.T) (*ui.MapEditor, *mapInspectionEdits) {
		e, _ := openGoalMap(t, emptyScriptMap(t, 40))
		s := formHost(e)
		s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{{ID: 21, Name: "Hero"}, {ID: 51, Name: "Elder"}}}}
		if _, err := s.ExtendScript(); err != nil {
			t.Fatal(err)
		}
		startDialogueRow(t, e)
		dlgType(t, e, ui.DialogueLineText, "Hello")
		dlgChoose(t, e, ui.DialogueLineSpeaker, "Elder")
		dlgPress(t, e, "splitsex")
		typeAt(t, e, 2, "Hey, sir")
		return e, s
	}
	t.Run("file", func(t *testing.T) {
		e, s := open(t)
		dlgChoose(t, e, "d.after", "Reply choice")
		dlgType(t, e, "d.rt", "Hi")
		dlgApply(t, e)
		names := s.DialogueNames()
		if len(names) != 1 {
			t.Fatalf("a reply choice wrote no dialogue file:\n%s", trigLines(e))
		}
		d, err := s.ReadDialogue(names[0])
		if err != nil || len(d.Nodes) != 1 {
			t.Fatalf("the file holds %+v (%v)", d, err)
		}
		n := d.Nodes[0]
		if !n.Speaks || n.Speaker != 51 || n.Variants["woman"]["en"] != "Hello" || n.Variants["man"]["en"] != "Hey, sir" {
			t.Fatalf("the node holds speaker %d (%v) and variants %v", n.Speaker, n.Speaks, n.Variants)
		}
		dlgCancel(t, e)
		wizPress(t, e, "a0.item", 0)
		lines := trigLines(e)
		if !strings.Contains(lines, "Hello") || !strings.Contains(lines, "Hey, sir") || !strings.Contains(lines, "Elder") {
			t.Fatalf("the reopened window lost the speaker or a variant:\n%s", lines)
		}
	})
	t.Run("pages", func(t *testing.T) {
		e, s := open(t)
		dlgApply(t, e)
		if len(s.DialogueNames()) != 0 {
			t.Fatalf("a plain dialog made a dialogue file: %v", s.DialogueNames())
		}
		found := false
		for page := uint32(1); page < 8 && !found; page++ {
			parts := s.MessageParts(page)
			found = len(parts) == 2 && parts[0].Body == "Hello" && parts[1].Body == "Hey, sir" && parts[0].Speaker == 51 && parts[1].Speaker == 51
		}
		if !found {
			t.Fatalf("no page holds the woman's and the man's cell with speaker 51")
		}
	})
}
