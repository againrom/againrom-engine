package game

import (
	"testing"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A reply's text stays as typed while the other reply fields change, in the
// window and in the file, on a Russian face and an English one.
func TestReplyTextSurvivesEditingTheOtherReplyFields(t *testing.T) {
	for _, tc := range []struct {
		lang, reply string
		selector    int
	}{
		{"ru", "Попросить золото", 1},
		{"en", "Ask for gold", 0},
	} {
		t.Run(tc.lang, func(t *testing.T) {
			e, x := fontEditor(t, emptyScriptMap(t, 40), tc.selector, tc.lang)
			s := formHost(e)
			if _, err := s.ExtendScript(); err != nil {
				t.Fatal(err)
			}
			w := words.For(tc.lang)
			label := func(key string) string { return x.inspectionUTF8(w.Text(key)) }
			wizNewTrigger(t, e, "Mill")
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "a0.item"), label("editor.wizard.a.dialog"))
			wizPress(t, e, "a0.item", 0)
			dlgType(t, e, ui.DialogueLineText, "Hello")
			dlgChoose(t, e, "d.after", label("editor.dialogue.one.replychoice"))
			dlgType(t, e, "d.rt", tc.reply)
			want := tc.reply
			same := func(step string) {
				t.Helper()
				if got := e.DialogueText("d.rt"); got != want {
					t.Fatalf("after %s the reply text is %q, want %q", step, got, want)
				}
			}
			same("typing")
			dlgChoose(t, e, "d.runavail", label("editor.dialogue.disabled"))
			same("When unavailable")
			dlgChoose(t, e, "d.ronce", label("editor.dialogue.once.no"))
			same("Only once")
			dlgChoose(t, e, "d.rnext", label("editor.dialogue.one.end"))
			same("Transition")
			dlgApply(t, e)
			names := s.DialogueNames()
			if len(names) != 1 {
				t.Fatalf("no dialogue file:\n%s", trigLines(e))
			}
			d, err := s.ReadDialogue(names[0])
			if err != nil || len(d.Nodes) != 1 || len(d.Nodes[0].Replies) != 1 {
				t.Fatalf("the file holds %+v (%v)", d, err)
			}
			if got := d.Nodes[0].Replies[0].Text[w.Lang()]; got != tc.reply {
				t.Fatalf("the file holds the reply %q, want %q", got, tc.reply)
			}
			dlgCancel(t, e)
			wizPress(t, e, "a0.item", 0)
			if e.DialogueField("d.rt") < 0 {
				dlgReply(t, e, 0)
			}
			same("reopening")
			dlgChoose(t, e, "d.runavail", label("editor.dialogue.hidden"))
			same("When unavailable on the reopened reply")
			dlgChoose(t, e, "d.ronce", label("editor.dialogue.once.yes"))
			same("Only once on the reopened reply")
		})
	}
}
