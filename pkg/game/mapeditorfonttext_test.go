package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// fontEditor opens b in an editor whose face has selector and whose words are
// lang's, the encoder the game wires.
func fontEditor(t *testing.T, b []byte, selector int, lang string) (*ui.MapEditor, *MapInspector) {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}, rules: []mapRule{}, font: &text.Font{Selector: selector}}
	path := filepath.Join(t.TempDir(), "goal.alm")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	e.SetWords(words.For(lang))
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	closeMapPanel(t, e)
	return e, x
}

// noQuestions fails when a line the editor draws has a run of replacement
// marks: text kept in the font's bytes was encoded a second time.
func noQuestions(t *testing.T, e *ui.MapEditor, x *MapInspector, where string) {
	t.Helper()
	for _, l := range e.PopupLines() {
		if s := x.messageUTF8([]byte(l)); strings.Contains(s, "??") {
			t.Errorf("%s shows replacement marks: %q", where, s)
		}
	}
}

// Notes, refusals and default names the editor writes reach the screen as
// the words say, on a Russian face and on an English one.
func TestEditorNotesAndDefaultNamesDrawWithoutReplacementMarks(t *testing.T) {
	for _, tc := range []struct {
		selector int
		lang     string
	}{{1, "ru"}, {0, ""}, {0, "ru"}} {
		book := words.For(tc.lang)
		enc := func(w string) string { return EncodeHostText(w, tc.selector) }
		e, x := fontEditor(t, triggerMap(t), tc.selector, tc.lang)
		where := book.Lang() + "/" + string(rune('0'+tc.selector))
		if _, err := formHost(e).ExtendScript(); err != nil {
			t.Fatal(err)
		}
		wizNewTrigger(t, e, "Mill")
		wizPress(t, e, "+action", 0)
		trigChoose(t, e, wizField(t, e, "a0.item"), enc(book.Text("editor.wizard.a.dialog")))
		wizPress(t, e, "a0.item", 0)
		noQuestions(t, e, x, where+" dialogue window")
		trigChoose(t, e, dlgField(t, e, "d.after"), enc(book.Text("editor.dialogue.one.replychoice")))
		dlgPress(t, e, "removereply")
		dlgApply(t, e)
		if tc.selector == 1 && !strings.Contains(strings.Join(e.PopupLines(), "\n"), enc(book.Text("editor.dialogue.one.after"))) {
			t.Errorf("%s: the window is not drawn in the editor's words:\n%s", where, x.messageUTF8([]byte(trigLines(e))))
		}
		noQuestions(t, e, x, where+" refusal")
	}
}

// A refused trigger detail row says why in the editor's words on a Russian
// face.
func TestTriggerDetailRefusalDrawsWithoutReplacementMarks(t *testing.T) {
	e, x := fontEditor(t, triggerMap(t), 1, "ru")
	book := words.For("ru")
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	trigDetails(t, e)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 11, K: 0, Flag: false})
	trigClick(t, e, r, ok, "Edit action 1")
	trigChoose(t, e, 1, EncodeHostText(book.Text("editor.action.op23"), 1))
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	trigClick(t, e, r, ok, "Apply")
	if !strings.Contains(x.messageUTF8([]byte(trigLines(e))), "Игрок: выберите значение из списка.") {
		t.Fatalf("the refusal is not drawn:\n%s", x.messageUTF8([]byte(trigLines(e))))
	}
	noQuestions(t, e, x, "detail refusal")
}
