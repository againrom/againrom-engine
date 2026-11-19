package game

import (
	"errors"
	"image"
	"slices"
	"strings"
	"testing"

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

// dlgPress presses the button of the row of field key that does act.
func dlgPress(t *testing.T, e *ui.MapEditor, key, act string) {
	t.Helper()
	k := slices.Index(e.DialogueRowActions(key), act)
	if k < 0 {
		t.Fatalf("the row of %s has no %s button: %v\n%s", key, act, e.DialogueRowActions(key), trigLines(e))
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: dlgField(t, e, key), K: k})
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

// startDialogueRow opens a trigger whose action starts the dialogue name and
// opens the dialogue window through the action's New dialogue button.
func startDialogueRow(t *testing.T, e *ui.MapEditor, name string) {
	t.Helper()
	wizNewTrigger(t, e, "Mill")
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Start dialog")
	trigChoose(t, e, wizField(t, e, "a0.p0"), "Dialogue with replies")
	wizType(t, e, "a0.p1", name)
	wizPress(t, e, "a0.p1", 0)
}

// Switching the action back to Legacy leaves the name unused, and a map with
// no interactive action left asks to fold to .alm when it is saved.
func TestSwitchingBackToLegacyKeepsTheNameUnusedAndFoldsTheMap(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	startDialogueRow(t, e, "mill")
	dlgApply(t, e)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 1})
	trigClick(t, e, r, ok, "Cancel")
	if sc := wizScript(t, s.model.Bytes()); len(sc.Actions) != 1 || !sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the interactive action is not on the map: %+v", sc.Actions)
	}
	trigChoose(t, e, wizField(t, e, "a0.p0"), "Message page")
	wizApply(t, e)
	sc := wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || sc.Actions[0].InteractiveMessage() || len(s.DialogueUsers("mill")) != 0 {
		t.Fatalf("the action is still interactive: %+v", sc.Actions)
	}
	if _, err := s.SaveInPlace(); err == nil || !errors.As(err, new(*ui.ScriptFormNeeded)) {
		t.Fatalf("a map with no interactive action did not ask to fold: %v", err)
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
	for n := 0; n < 3; n++ {
		if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n}); !ok || !r.In(screen) {
			t.Fatalf("window button %d is %v, found %v", n, r, ok)
		}
	}
	for _, key := range []string{"d.node", "d.name", "d.text.en", "d.text.ru", "d.reply", "d.r.en", "d.r.ru", "d.next", "d.once", "d.unavailable", "c0.item", "c0.p0"} {
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
	startDialogueRow(t, e, "mill")
	if !e.DialogueOpen() {
		t.Fatalf("the New dialogue button opened no window:\n%s", trigLines(e))
	}
	if !strings.Contains(trigLines(e), "Node name") {
		t.Fatalf("the window is not the dialogue's:\n%s", trigLines(e))
	}
	sc := wizScript(t, s.model.Bytes())
	if len(sc.Actions) != 1 || sc.Actions[0].Label != "mill" || !sc.Actions[0].InteractiveMessage() {
		t.Fatalf("the action was not written as an interactive message: %+v", sc.Actions)
	}
}
