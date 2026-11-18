package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// questionEdits is the party host whose player 2 is named Monsters.
type questionEdits struct{ *partyEdits }

func (q questionEdits) Fields(kind RecordKind, index int) []InspectionField {
	if kind == KindPlayer {
		return []InspectionField{{Key: playerNameField, Text: true, TextValue: "Monsters"}}
	}
	return q.partyEdits.Fields(kind, index)
}

// questionsEditor is an editor of lang at 1920x1080 in the editor words'
// font, over a map with a second player, no start point and no victory.
func questionsEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	f, rows := endless()
	p := questionEdits{&partyEdits{mapPanelEdits: mapPanelEdits{facts: f, rows: rows}, cards: partyCards()}}
	open := func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		players := []InspectionRecord{{Kind: KindPlayer, Index: 0, Label: "Player 1"}, {Kind: KindPlayer, Index: 1, Label: "Player 2"}}
		return &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: p, Records: players}, nil
	}
	e := NewMapEditor(wordsFont(lang), nil, open)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.Layout(1920, 1080)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	return e
}

// questionWindow opens one question window.
type questionWindow struct {
	name string
	open func(e *MapEditor)
}

// questionWindows open each question window as the editor opens it, with
// notes as long as the windows meet.
func questionWindows() []questionWindow {
	picker := func(rows ...string) *Picker {
		var r []PickerRow
		for _, s := range rows {
			r = append(r, PickerRow{Text: s, Choosable: true})
		}
		p := NewPicker(r).SetWindow(min(10, len(r)))
		p.Select(0)
		return p
	}
	return []questionWindow{
		{"leave", func(e *MapEditor) { e.openPrompt(editorPrompt{kind: promptLeave, then: leaveStep{closeMap: true}}) }},
		{"notice", func(e *MapEditor) { e.openPrompt(editorPrompt{kind: promptNotice, notice: noticeTestPlay}) }},
		{"newmode", func(e *MapEditor) {
			e.openPrompt(editorPrompt{kind: promptNewMode, path: "130"})
			e.message = fmt.Sprintf(e.plainWord("editor.newmap.badheight"), "130")
		}},
		{"scriptform", func(e *MapEditor) {
			e.openPrompt(editorPrompt{kind: promptScriptForm, form: &formPrompt{need: &ScriptFormNeeded{Why: e.plainWord("editor.wizard.checksfull")}}})
		}},
		{"scriptfold", func(e *MapEditor) {
			e.openPrompt(editorPrompt{kind: promptScriptForm, form: &formPrompt{need: &ScriptFormNeeded{Fold: true, Why: e.plainWord("editor.scriptform.fits")}}})
		}},
		{"areatrigger", func(e *MapEditor) {
			choice := picker(e.editorHostText("New trigger"), e.editorHostText("Trigger 12: the party reaches the far bridge over the river"))
			e.openPrompt(editorPrompt{kind: promptAreaTrigger, area: &areaPrompt{box: image.Rect(3, 4, 10, 12), choice: choice}})
		}},
		{"areashared", func(e *MapEditor) {
			note := fmt.Sprintf(e.word("editor.area.sharedconst"), 7, "1, 4, 9", 3)
			e.openPrompt(editorPrompt{kind: promptAreaShared, area: &areaPrompt{kind: KindTrigger, notes: []string{note}}})
		}},
		{"togroup", func(e *MapEditor) {
			var rows []string
			for g := range 20 {
				rows = append(rows, fmt.Sprintf(e.word("editor.group.row"), g+1, g%4+1))
			}
			skips := []string{fmt.Sprintf(e.word("editor.group.skipped"), 2, 1)}
			e.openPrompt(editorPrompt{kind: promptToGroup, group: &groupPrompt{units: make([]RecordRef, 3), player: 1, skips: skips, list: picker(rows...)}})
		}},
		{"groupwarn", func(e *MapEditor) {
			warn := []groupWarning{
				{group: 2, users: []string{e.editorHostText("Trigger 1: Bridge"), e.editorHostText("Action 4: Move group 2")}},
				{group: 5, users: []string{e.editorHostText("Trigger 3: Gate")}},
			}
			e.openPrompt(editorPrompt{kind: promptGroupWarn, group: &groupPrompt{warn: warn}})
		}},
		{"testparty", func(e *MapEditor) { _ = e.Key("testplay") }},
		{"playerdelete", func(e *MapEditor) {
			e.openPrompt(editorPrompt{kind: promptDelete, del: RecordRef{Kind: KindPlayer, Index: 1}})
		}},
	}
}

// eachQuestion runs check on every question window open in an editor of
// each language.
func eachQuestion(t *testing.T, check func(t *testing.T, e *MapEditor, name string, s popupSpec)) {
	for _, lang := range []string{"en", "ru"} {
		for _, w := range questionWindows() {
			t.Run(lang+"/"+w.name, func(t *testing.T) {
				e := questionsEditor(t, lang)
				w.open(e)
				s, ok := e.promptSpec()
				if !ok {
					t.Fatal("the window did not open")
				}
				check(t, e, w.name, s)
			})
		}
	}
}

// A question is as wide as its text: a short one stays under the old 560,
// a long note reads at most 60 average letters a line, and the window is
// wider than 560 to hold that line.
func TestQuestionWidthIsItsOwnText(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		body := e.bodyFont()
		most := 60 * (body.Advance("abcdefghijklmnopqrstuvwxyz") / 26)
		for _, n := range s.Notes {
			if body.Advance(n) > most {
				t.Errorf("note line %q is %d wide, past %d", n, body.Advance(n), most)
			}
		}
		if s.Width > 1920-16 {
			t.Errorf("width %d past the screen less 16", s.Width)
		}
		switch name {
		case "playerdelete":
			if s.Width >= 560 {
				t.Errorf("a short question is %d wide", s.Width)
			}
		case "areashared":
			if s.Width <= 560 || len(s.Notes) < 2 {
				t.Errorf("a long note in %d lines makes the window %d wide", len(s.Notes), s.Width)
			}
		}
	})
}

// Each window's notes take one ink: label ink where they only state, the
// warning ink where they warn or refuse.
func TestQuestionNotesTakeOneInk(t *testing.T) {
	state := map[string]bool{"scriptfold": true, "areatrigger": true, "togroup": true}
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		want := editorWarning
		if state[name] {
			want = editorPaper
		}
		if s.NoteInk != want {
			t.Errorf("note ink %v, want %v", s.NoteInk, want)
		}
	})
}

// The keys line ends the help of a question that has help; a question
// without help has no (?) for the keys alone.
func TestQuestionKeysLineOnlyBehindHelp(t *testing.T) {
	helped := map[string]bool{"notice": true, "newmode": true, "scriptform": true, "scriptfold": true, "areatrigger": true, "testparty": true, "playerdelete": true}
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		keys := fmt.Sprintf(e.word("editor.question.keys"), s.Buttons[0])
		if helped[name] != (s.TitleHelp != "") {
			t.Fatalf("help %q", s.TitleHelp)
		}
		if helped[name] && (!strings.HasSuffix(s.TitleHelp, " "+keys) || strings.TrimSuffix(s.TitleHelp, " "+keys) == "") {
			t.Errorf("help %q does not end with the keys line %q", s.TitleHelp, keys)
		}
	})
}

// A title names the window; none asks.
func TestQuestionTitlesNameTheWindow(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if strings.Contains(s.Title, "?") {
			t.Errorf("title %q asks", s.Title)
		}
	})
}

// The notice window shows the warnings only; what they mean stands behind
// the (?).
func TestNoticeShowsTheWarningsOnly(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "notice" && name != "testparty" {
			return
		}
		lines := e.panelWarnings()
		if len(lines) == 0 || len(s.Notes) == 0 || !strings.HasPrefix(s.Notes[0], lines[0].label+": ") {
			t.Errorf("notes %q, want the warnings first", s.Notes)
		}
		if !strings.Contains(s.TitleHelp, e.word("editor.notice.testplay.help")) {
			t.Errorf("help %q lacks the notice's meaning", s.TitleHelp)
		}
	})
}

// The height caption keeps the range and the row has no help of its own.
func TestNewMapHeightCaptionKeepsTheRange(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "newmode" {
			return
		}
		if len(s.Rows) != 1 || s.Rows[0].Label != e.word("editor.newmap.height") || !strings.Contains(s.Rows[0].Label, "(0-126)") || s.Rows[0].Help != "" {
			t.Errorf("rows %+v", s.Rows)
		}
	})
}

// The fold window shows the host's reason as a state line.
func TestFoldWindowShowsTheReason(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "scriptfold" {
			return
		}
		if strings.Join(s.Notes, "") != e.word("editor.scriptform.fits") || s.NoteInk != editorPaper {
			t.Errorf("notes %q ink %v", s.Notes, s.NoteInk)
		}
	})
}

// The area window's note is the box only; its meaning stands behind the (?).
func TestAreaWindowNoteIsTheBox(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "areatrigger" {
			return
		}
		if box := fmt.Sprintf(e.word("editor.area.newtext"), 3, 4, 9, 11); len(s.Notes) != 1 || s.Notes[0] != box {
			t.Errorf("notes %q, want %q", s.Notes, box)
		}
		if !strings.HasPrefix(s.TitleHelp, e.word("editor.area.newtitle.help")) {
			t.Errorf("help %q", s.TitleHelp)
		}
	})
}

// The group warning has a head line per group and a line per trigger or
// action that names it.
func TestGroupWarningHasALinePerUser(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "groupwarn" {
			return
		}
		head := func(g int) string { return fmt.Sprintf(e.word("editor.group.warnhead"), g) }
		want := []string{head(2), e.editorHostText("Trigger 1: Bridge"), e.editorHostText("Action 4: Move group 2"), head(5), e.editorHostText("Trigger 3: Gate")}
		if strings.Join(s.Notes, "|") != strings.Join(want, "|") {
			t.Errorf("notes %q, want %q", s.Notes, want)
		}
	})
}

// The players' delete question names the player; the consequences stand
// behind the (?).
func TestPlayerDeleteNamesThePlayer(t *testing.T) {
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		if name != "playerdelete" {
			return
		}
		if note := fmt.Sprintf(e.word("editor.players.deletetext"), 2, "Monsters"); len(s.Notes) != 1 || s.Notes[0] != note {
			t.Errorf("notes %q, want %q", s.Notes, note)
		}
		if !strings.HasPrefix(s.TitleHelp, e.word("editor.players.deletetitle.help")) {
			t.Errorf("help %q", s.TitleHelp)
		}
	})
}
