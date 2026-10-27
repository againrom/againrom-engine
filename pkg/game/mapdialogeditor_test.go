package game

import (
	"bytes"
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// The dialog editor through a trigger's Start dialog: trigger 0 starts page
// 7, which holds three cards; trigger 1 only wins.
func dialogFixture(t *testing.T) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	e, s := triggerMessageFixture(t, false)
	workspacePage(t, e, "event07.txt", "<part=1 npc=21>\nOne<part=2>\nTwo<part=3>\nThree")
	return e, s
}

// configure presses Configure on the wizard's action row key.
func configure(t *testing.T, e *ui.MapEditor, key string) {
	t.Helper()
	r, ok := e.RowButtonRect(wizField(t, e, key), 0, false)
	trigClick(t, e, r, ok, "Configure of "+key)
	if e.WizardShown() {
		t.Fatalf("Configure did not open the dialog editor:\n%s", trigLines(e))
	}
}

// openDialog opens trigger 0's dialog in the dialog editor.
func openDialog(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	configure(t, e, "a0.item")
	for _, want := range []string{"One", "Two", "Three"} {
		if !strings.Contains(trigLines(e), want) {
			t.Fatalf("the dialog editor lacks %q:\n%s", want, trigLines(e))
		}
	}
}

// bodies are page n's card texts in order.
func bodies(s *mapInspectionEdits, n uint32) []string {
	var out []string
	for _, p := range s.MessageParts(n) {
		out = append(out, p.Body)
	}
	return out
}

// dragCard drags card from by its grip and drops it on the top edge of
// card to.
func dragCard(t *testing.T, e *ui.MapEditor, from, to int) {
	t.Helper()
	_, grip, ok := e.CardRects(from)
	if !ok || grip.Empty() {
		t.Fatalf("card %d shows no grip", from)
	}
	target, _, ok := e.CardRects(to)
	if !ok {
		t.Fatalf("card %d is not shown", to)
	}
	at := grip.Min.Add(grip.Size().Div(2))
	drop := image.Pt(at.X, target.Min.Y+2)
	e.Pointer(ui.Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: at.X, CursorY: (at.Y + drop.Y) / 2, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: drop.X, CursorY: drop.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: drop.X, CursorY: drop.Y}, time.Time{})
}

// unchanged fails when the map, the pages or the history moved.
func unchanged(t *testing.T, s *mapInspectionEdits, model []byte, pages map[string][]byte, steps int, what string) {
	t.Helper()
	if !bytes.Equal(model, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps {
		t.Fatal(what, "wrote the map, the pages or a history entry")
	}
}

func TestStartDialogIsNamedAndItsRowShowsOnlyConfigure(t *testing.T) {
	for _, lang := range []struct{ book, name, button string }{{"", "Start dialog", "Configure"}, {"russian", "Начать диалог", "Настроить"}} {
		t.Run(lang.name, func(t *testing.T) {
			e, s := dialogFixture(t)
			if got := words.For(lang.book).Text("editor.wizard.a.dialog"); got != lang.name {
				t.Fatalf("the action's word is %q", got)
			}
			if lang.book != "" {
				e.SetWords(words.For(lang.book))
			}
			lang.name, lang.button = s.inspector.inspectionUTF8(lang.name), s.inspector.inspectionUTF8(lang.button)
			e.Select(recordIndex(t, e, "Trigger", 0), false)
			lines := e.PopupLines()
			row := slices.IndexFunc(lines, func(l string) bool { return strings.HasSuffix(l, " "+lang.name) })
			if row < 0 {
				t.Fatalf("no action row %q:\n%s", lang.name, trigLines(e))
			}
			if got := e.PopupRowButtons(row); len(got) != 2 || got[0] != lang.button {
				t.Fatalf("the action row's buttons are %q, want %q first", got, lang.button)
			}
			for _, l := range lines[row+1:] {
				if strings.Contains(l, "7") || strings.Contains(l, "One") || strings.HasPrefix(strings.TrimSpace(l), "Open") || strings.HasPrefix(strings.TrimSpace(l), "Открыть") {
					t.Fatalf("the Start dialog row shows its number or text: %q\n%s", l, trigLines(e))
				}
			}
		})
	}
}

func TestStartDialogIsOnePerTrigger(t *testing.T) {
	e, s := dialogFixture(t)
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	wizPress(t, e, "+action", 0)
	field := wizField(t, e, "a1.item")
	r, ok := e.FieldRect(field)
	trigClick(t, e, r, ok, "a1.item")
	choices := e.DropdownChoices()
	_ = e.Key("escape")
	if !slices.Contains(choices, "Start dialog: one per trigger, the player sees only the first") {
		t.Fatalf("a trigger with a Start dialog offers it plainly: %q", choices)
	}
	trigChoose(t, e, field, "Start dialog")
	if n := strings.Count(trigLines(e), "Action Start dialog"); n != 1 {
		t.Fatalf("the trigger shows %d Start dialog rows:\n%s", n, trigLines(e))
	}
	// Another trigger offers it.
	e.Select(recordIndex(t, e, "Trigger", 1), false)
	if e.WizardShown() && strings.Contains(trigLines(e), "Start dialog") {
		t.Fatal("trigger 1 starts a dialog already")
	}
	wizPress(t, e, "+action", 0)
	field = wizField(t, e, "a1.item")
	r, ok = e.FieldRect(field)
	trigClick(t, e, r, ok, "a1.item")
	if choices := e.DropdownChoices(); !slices.Contains(choices, "Start dialog") {
		t.Fatalf("a trigger without a Start dialog does not offer it: %q", choices)
	}
	_ = e.Key("escape")
	trigChoose(t, e, field, "Start dialog")
	if !strings.Contains(trigLines(e), "Action Start dialog") {
		t.Fatalf("trigger 1 did not take Start dialog:\n%s", trigLines(e))
	}
	model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
	messageButton(t, e, 1)
	unchanged(t, s, model, pages, steps, "Cancel of the wizard")
}

func TestStartDialogTwoInOneTriggerLoadAndStay(t *testing.T) {
	acts := []scriptNode{{label: "Talk", opcode: 2, id: 1, tags: []uint32{1}, values: []uint32{1}}, {label: "More", opcode: 2, id: 2, tags: []uint32{1}, values: []uint32{2}}}
	checks := []scriptNode{{label: "One", opcode: 0x10002, id: 1, tags: []uint32{1}, values: []uint32{1}}}
	triggers := []goalTrigger{{name: "Twice", left: 1, right: 1, actions: []uint32{1, 2}}}
	_, e, _ := openTestMapEditor(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(acts, checks, triggers)}))
	s := e.Document().Edits.(*mapInspectionEdits)
	s.inspector.rules = []mapRule{mapRules[len(mapRules)-1]}
	if len(ofRule(s.Report(), "message-overlap")) != 1 {
		t.Fatal("two Start dialogs in one trigger give no message-overlap warning")
	}
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	if n := strings.Count(trigLines(e), "Action Start dialog"); n != 2 {
		t.Fatalf("the trigger shows %d Start dialog rows, want both:\n%s", n, trigLines(e))
	}
	wizType(t, e, "name", "Renamed")
	wizApply(t, e)
	sc := wizScript(t, s.model.Bytes())
	if sc.Triggers[0].Acts[0] != 1 || sc.Triggers[0].Acts[1] != 2 || len(ofRule(s.Report(), "message-overlap")) != 1 {
		t.Fatalf("Apply did not keep both Start dialogs: %+v", sc.Triggers[0])
	}
}

func TestDialogEditorDragMovesACard(t *testing.T) {
	for _, apply := range []bool{true, false} {
		t.Run(map[bool]string{true: "Apply", false: "Cancel"}[apply], func(t *testing.T) {
			e, s := dialogFixture(t)
			openDialog(t, e)
			model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			dragCard(t, e, 2, 0)
			unchanged(t, s, model, pages, steps, "the drag")
			lines := strings.Join(e.PopupLines()[1:], "\n") // after the dialog choice
			if strings.Index(lines, "Three") > strings.Index(lines, "One") {
				t.Fatalf("the drag did not move card 3 first:\n%s", lines)
			}
			if !apply {
				messageButton(t, e, 1)
				unchanged(t, s, model, pages, steps, "Cancel after a drag")
				return
			}
			messageButton(t, e, 0)
			if got := bodies(s, 7); !reflect.DeepEqual(got, []string{"Three", "One", "Two"}) || len(s.undoWorkspace) != steps+1 || !bytes.Equal(model, s.model.Bytes()) {
				t.Fatalf("Apply wrote %q in %d history entries", got, len(s.undoWorkspace)-steps)
			}
		})
	}
}

func TestDialogEditorKeyboardMovesACard(t *testing.T) {
	e, s := dialogFixture(t)
	openDialog(t, e)
	messageField(t, e, 0)
	if err := e.Key("carddown"); err != nil {
		t.Fatal(err)
	}
	messageButton(t, e, 0)
	if got := bodies(s, 7); !reflect.DeepEqual(got, []string{"Two", "One", "Three"}) {
		t.Fatalf("Alt+Down then Apply wrote %q", got)
	}
}

func TestDialogEditorCrossRestoreAndApply(t *testing.T) {
	for _, end := range []string{"restore", "apply", "cancel"} {
		t.Run(end, func(t *testing.T) {
			e, s := dialogFixture(t)
			openDialog(t, e)
			model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			r, ok := e.RowButtonRect(3, 0, false) // card 2's cross
			trigClick(t, e, r, ok, "cross")
			if !strings.Contains(trigLines(e), "Message removed") || strings.Contains(trigLines(e), "Two") {
				t.Fatalf("the cross did not strike card 2:\n%s", trigLines(e))
			}
			unchanged(t, s, model, pages, steps, "the cross")
			switch end {
			case "restore":
				r, ok = e.RowButtonRect(3, 0, false)
				trigClick(t, e, r, ok, "Restore")
				if strings.Contains(trigLines(e), "Message removed") || !strings.Contains(trigLines(e), "Two") {
					t.Fatalf("Restore did not bring card 2 back:\n%s", trigLines(e))
				}
				messageButton(t, e, 0)
				if got := bodies(s, 7); !reflect.DeepEqual(got, []string{"One", "Two", "Three"}) {
					t.Fatalf("Restore then Apply wrote %q", got)
				}
			case "apply":
				messageButton(t, e, 0)
				if got := bodies(s, 7); !reflect.DeepEqual(got, []string{"One", "Three"}) || len(s.undoWorkspace) != steps+1 {
					t.Fatalf("the cross then Apply wrote %q in %d history entries", got, len(s.undoWorkspace)-steps)
				}
			case "cancel":
				messageButton(t, e, 1)
				unchanged(t, s, model, pages, steps, "Cancel after the cross")
			}
		})
	}
}

func TestDialogEditorPlusAddsACardAtTheEnd(t *testing.T) {
	for _, apply := range []bool{true, false} {
		t.Run(map[bool]string{true: "Apply", false: "Cancel"}[apply], func(t *testing.T) {
			e, s := dialogFixture(t)
			openDialog(t, e)
			model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			r, ok := e.RowButtonRect(-1, 0, false)
			trigClick(t, e, r, ok, "+")
			e.TextInput("Four")
			unchanged(t, s, model, pages, steps, "the +")
			if !apply {
				messageButton(t, e, 1)
				unchanged(t, s, model, pages, steps, "Cancel after the +")
				return
			}
			messageButton(t, e, 0)
			if got := bodies(s, 7); !reflect.DeepEqual(got, []string{"One", "Two", "Three", "Four"}) || len(s.undoWorkspace) != steps+1 {
				t.Fatalf("the + then Apply wrote %q in %d history entries", got, len(s.undoWorkspace)-steps)
			}
		})
	}
}

func TestStartDialogConfigureMakesANewDialogOnApply(t *testing.T) {
	for _, apply := range []bool{true, false} {
		t.Run(map[bool]string{true: "Apply", false: "Cancel"}[apply], func(t *testing.T) {
			e, s := dialogFixture(t)
			e.Select(recordIndex(t, e, "Trigger", 1), false)
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "a1.item"), "Start dialog")
			wizApply(t, e)
			model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
			configure(t, e, "a1.item")
			if !strings.Contains(trigLines(e), "Dialog New dialog") {
				t.Fatalf("Configure on an action with no dialog did not start a new one:\n%s", trigLines(e))
			}
			e.TextInput("Hello")
			unchanged(t, s, model, pages, steps, "typing in a new dialog")
			if !apply {
				messageButton(t, e, 1)
				unchanged(t, s, model, pages, steps, "Cancel of a new dialog")
				return
			}
			messageButton(t, e, 0)
			sc := wizScript(t, s.model.Bytes())
			page := wizNode(t, sc.Actions, sc.Triggers[1].Acts[1]).Value[0]
			if page == 0 || !reflect.DeepEqual(bodies(s, page), []string{"Hello"}) || len(s.undoWorkspace) != steps+1 {
				t.Fatalf("Apply made page %d with %q in %d history entries", page, bodies(s, page), len(s.undoWorkspace)-steps)
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(model, s.model.Bytes()) || !reflect.DeepEqual(pages, s.workspace.Pages) {
				t.Fatal("one Undo did not remove the new dialog and its naming", err)
			}
		})
	}
}

func TestDialogEditorChoosesWhichDialogTheActionStarts(t *testing.T) {
	e, s := triggerMessageFixture(t, false)
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	model, pages, steps := bytes.Clone(s.model.Bytes()), cloneWorkspace(s.workspace).Pages, len(s.undoWorkspace)
	configure(t, e, "a0.item")
	choice := e.DialogChoiceField()
	if choice != 2 {
		t.Fatalf("the dialog choice is field %d, want 2 after one card", choice)
	}
	messageField(t, e, choice)
	choices := e.DropdownChoices()
	if !slices.Contains(choices, "New dialog") || !slices.ContainsFunc(choices, func(c string) bool { return strings.Contains(c, "2: Existing message") }) {
		t.Fatalf("the dialog choice offers %q", choices)
	}
	messageChoiceClick(t, e, "2: Existing message")
	if !strings.Contains(trigLines(e), "Existing message") {
		t.Fatalf("choosing dialog 2 did not show it:\n%s", trigLines(e))
	}
	unchanged(t, s, model, pages, steps, "the choice")
	messageButton(t, e, 0)
	sc := wizScript(t, s.model.Bytes())
	if wizNode(t, sc.Actions, sc.Triggers[0].Acts[0]).Value[0] != 2 || !reflect.DeepEqual(pages, s.workspace.Pages) || len(s.undoWorkspace) != steps+1 {
		t.Fatal("Apply did not make the action start dialog 2 as one history entry")
	}
	// Another trigger starting dialog 2 shows the shared note.
	messageButton(t, e, 2)
	e.Select(recordIndex(t, e, "Trigger", 1), false)
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a1.item"), "Start dialog")
	configure(t, e, "a1.item")
	messageField(t, e, choice)
	messageChoiceClick(t, e, "2: Existing message")
	if !strings.Contains(trigLines(e), "Trigger 1 also starts this dialog") {
		t.Fatalf("no shared note:\n%s", trigLines(e))
	}
}
