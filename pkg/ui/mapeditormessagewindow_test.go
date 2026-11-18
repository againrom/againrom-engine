package ui

import (
	"image"
	"regexp"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// messageWindowEdits is the two-message page of speakerPortraitEdits with
// the messages parts says.
type messageWindowEdits struct {
	*speakerPortraitEdits
	parts []InspectionMessagePart
}

func (m *messageWindowEdits) MessageParts(uint32) []InspectionMessagePart { return m.parts }

// messageWindowEditor opens dialog 1 holding parts in lang at 1920x1080;
// nil parts keep the page of two messages.
func messageWindowEditor(t *testing.T, lang string, parts []InspectionMessagePart) *MapEditor {
	t.Helper()
	e, _ := speakerPortraitEditor(t, lang)
	if parts != nil {
		e.doc.Edits = &messageWindowEdits{speakerPortraitEdits: e.doc.Edits.(*speakerPortraitEdits), parts: parts}
		e.openMessage(1)
	}
	return e
}

// rowRectOf is the laid-out rectangle of spec row index.
func rowRectOf(t *testing.T, l popupLayout, index int) popupRowRect {
	t.Helper()
	for _, rr := range l.Rows {
		if rr.Index == index {
			return rr
		}
	}
	t.Fatalf("row %d is not laid out", index)
	return popupRowRect{}
}

// The message window at 1920x1080 holds the layout properties in both
// languages; a click on "+" adds a message and one on Cancel closes the
// window, both through the editor's input.
func TestMessageWindowHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := messageWindowEditor(t, lang, nil)
			s, _, ok := e.messageSpec()
			if !ok {
				t.Fatal("the message window is not built")
			}
			checkWindowProperties(t, e, s, image.Pt(1920, 1080))
			l := layoutPopup(s, e.font, e.bodyFont())
			if !l.TitleHelp.In(l.Rect) {
				t.Errorf("the title's (?) at %v leaves the window %v", l.TitleHelp, l.Rect)
			}
			add := rowRectOf(t, l, len(s.Rows)-1)
			if len(add.Buttons) != 1 || s.Rows[len(s.Rows)-1].Buttons[0] != e.word("editor.message.mark.add") {
				t.Fatalf("the last row is not the + row: %+v", s.Rows[len(s.Rows)-1])
			}
			modesClick(e, center(add.Buttons[0]))
			if n := e.messageLive(); n != 3 {
				t.Fatalf("a click on + leaves %d messages, want 3", n)
			}
			s, _, _ = e.messageSpec()
			l = layoutPopup(s, e.font, e.bodyFont())
			cancel := -1
			for k, b := range s.Buttons {
				if b == e.word("editor.button.cancel") {
					cancel = k
				}
			}
			modesClick(e, center(l.Buttons[cancel]))
			if e.messagePage != 0 {
				t.Fatal("a click on Cancel left the message window open")
			}
		})
	}
}

// The message window is as wide as its parts ask, not a fixed 660: a page
// of short messages stands narrower, and a speaker's long name widens it
// until the name and the head's buttons show whole.
func TestMessageWindowTakesItsNaturalWidth(t *testing.T) {
	e := messageWindowEditor(t, "en", nil)
	s, _, _ := e.messageSpec()
	if s.Width >= 660 {
		t.Errorf("a page of two short messages is %d wide, want its natural width under 660", s.Width)
	}
	e = messageWindowEditor(t, "en", nil)
	e.doc.Edits.(*speakerPortraitEdits).labels[1] = "5: " + strings.Repeat("Anna Maria ", 9)
	e.openMessage(1)
	s, _, _ = e.messageSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	head := rowRectOf(t, l, e.fieldRow(1))
	if l.Rect.Dx() <= 660 {
		t.Errorf("a long speaker leaves the window %d wide", l.Rect.Dx())
	}
	if need := e.bodyFont().Advance(s.Rows[e.fieldRow(1)].Value) + editTextX; head.Control.Dx() < need {
		t.Errorf("the speaker %q is %d wide in a %d wide field", s.Rows[e.fieldRow(1)].Value, need, head.Control.Dx())
	}
	for k, b := range head.Buttons {
		if !b.In(l.Rect) {
			t.Errorf("head button %d at %v leaves the window %v", k, b, l.Rect)
		}
	}
}

// Every message's text shows whole, not only a page's single message: the
// box of a long text holds each of its wrapped lines. A page taller than
// the window pages inside the screen.
func TestMessageWindowShowsEveryTextWhole(t *testing.T) {
	long := strings.TrimSuffix(strings.Repeat("A line of the first message.\n", 9), "\n")
	e := messageWindowEditor(t, "en", []InspectionMessagePart{{Source: 0, Number: 1, Body: long, Speaker: 5}, {Source: 1, Number: 2, Body: "Second", Speaker: 6}})
	s, _, _ := e.messageSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	body := e.bodyFont()
	text := rowRectOf(t, l, e.fieldRow(0))
	count := 0
	for _, part := range strings.Split(s.Rows[e.fieldRow(0)].Value, "\n") {
		count += max(1, len(editorTextLines(body, part, text.Control.Dx()-8)))
	}
	if shown := (text.Control.Dy() + 2 - 10) / fontHeight(body, 15); shown < count {
		t.Errorf("the first message's box shows %d of its %d lines", shown, count)
	}
	var parts []InspectionMessagePart
	for k := range 30 {
		parts = append(parts, InspectionMessagePart{Source: k, Number: k + 1, Body: "One\nTwo\nThree\nFour", Speaker: 5})
	}
	e = messageWindowEditor(t, "en", parts)
	s, _, _ = e.messageSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	if !l.Rect.In(image.Rect(0, 0, 1920, 1080)) || s.MaxRows <= 0 || s.MaxRows >= len(s.Rows) {
		t.Errorf("a page of 30 messages stands at %v with %d of %d rows shown", l.Rect, s.MaxRows, len(s.Rows))
	}
}

// A message's x and the page's + keep their marks from the words table, and
// each says in its tip what it does.
func TestMessageWindowButtonsSayWhatTheyDo(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := messageWindowEditor(t, lang, nil)
			s, _, _ := e.messageSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			bounds := image.Rect(0, 0, e.window.X, e.window.Y)
			for _, tc := range []struct {
				row       int
				mark, tip string
			}{
				{e.fieldRow(1), "editor.message.mark.remove", "editor.message.remove.tip"},
				{len(s.Rows) - 1, "editor.message.mark.add", "editor.message.add.tip"},
			} {
				if got := s.Rows[tc.row].Buttons[0]; got != e.word(tc.mark) {
					t.Errorf("row %d's first button is %q, want %q", tc.row, got, e.word(tc.mark))
				}
				b := rowRectOf(t, l, tc.row).Buttons[0]
				tip, ok := popupTipAt(bounds, e.font, e.bodyFont(), s, l, center(b), true)
				if !ok || strings.Join(tip.Lines, "") != e.word(tc.tip) {
					t.Errorf("the tip of %q is %q, want %q", e.word(tc.mark), tip.Lines, e.word(tc.tip))
				}
			}
		})
	}
}

// A cell where the hero's dialog ends shows its hero and the line that says
// so, with no speaker and no text box; the cell beside keeps both.
func TestMessageWindowEndCellHasNoSpeakerAndNoText(t *testing.T) {
	cells := HeroCells(true, false)
	e := messageWindowEditor(t, "en", []InspectionMessagePart{
		{Source: 0, Number: 1, Heroes: cells[0], Ends: true},
		{Source: 1, Number: 1, Body: "Only him", Speaker: 5, Heroes: cells[1]},
	})
	s, _, _ := e.messageSpec()
	ended, ends, texts := false, false, 0
	for _, r := range s.Rows {
		switch {
		case r.Label == e.word(HeroWord(cells[0])):
			ended = true
			if r.Kind != popupText || r.Value != "" || len(r.Buttons) != 0 {
				t.Errorf("the ended hero's row has a speaker: %+v", r)
			}
		case r.Label == e.word("editor.message.ends"):
			ends = true
		case r.Kind == popupMultiline:
			texts++
		}
	}
	if !ended || !ends || texts != 1 {
		t.Errorf("hero row %v, ends line %v, %d text boxes; want the hero, the line and one text box", ended, ends, texts)
	}
}

// One term in the words: a message is "message", the page a "dialog"; no
// message window, dialog keys or speaker word says card or dialogue, in
// English or Russian.
func TestMessageWordsSayMessageAndDialog(t *testing.T) {
	en, ru := words.For("en"), words.For("ru")
	if got := en.Text("editor.speaker.why.card"); got != "Message %s: %s" {
		t.Errorf("editor.speaker.why.card = %q, want %q", got, "Message %s: %s")
	}
	ids, err := words.IDs(words.Engine())
	if err != nil {
		t.Fatal(err)
	}
	english, russian := regexp.MustCompile(`(?i)\bcards?\b|dialogue|\bpages?\b`), regexp.MustCompile(`(?i)карточк|страниц`)
	for _, id := range ids {
		if !strings.HasPrefix(id, "editor.message.") && !strings.HasPrefix(id, "editor.keys.dialogue") && !strings.HasPrefix(id, "editor.speaker.") && !strings.HasPrefix(id, "editor.unit.speaker") {
			continue
		}
		if english.MatchString(en.Text(id)) {
			t.Errorf("en %s = %q", id, en.Text(id))
		}
		if russian.MatchString(ru.Text(id)) {
			t.Errorf("ru %s = %q", id, ru.Text(id))
		}
	}
	// The words that name a dialog say dialog, never message: the trigger
	// field that points to one, its new choice and the add menu.
	enMessage, ruMessage := regexp.MustCompile(`(?i)\bmessages?\b`), regexp.MustCompile(`(?i)сообщени`)
	for _, id := range []string{"editor.message.parameter", "editor.message.new", "editor.message.unavailable", "editor.message.full"} {
		if enMessage.MatchString(en.Text(id)) || !strings.Contains(strings.ToLower(en.Text(id)), "dialog") {
			t.Errorf("en %s = %q, want dialog and no message", id, en.Text(id))
		}
		if ruMessage.MatchString(ru.Text(id)) || !strings.Contains(strings.ToLower(ru.Text(id)), "диалог") {
			t.Errorf("ru %s = %q, want диалог and no сообщение", id, ru.Text(id))
		}
	}
	if got := en.Text("editor.message.overlap"); !strings.Contains(got, "more than one dialog") || !strings.Contains(got, "messages in one dialog") {
		t.Errorf("en editor.message.overlap = %q", got)
	}
	if got := ru.Text("editor.message.overlap"); !strings.Contains(got, "несколько диалогов") || !strings.Contains(got, "сообщения разговора в один диалог") {
		t.Errorf("ru editor.message.overlap = %q", got)
	}
}

// variableWindowEditor is the scripts fixture at 1920x1080 with variable 0
// shared with condition 1 and register 5 labelled "Spare", addressed by no
// node.
func variableWindowEditor(t *testing.T) *MapEditor {
	t.Helper()
	e := scriptsListEditor(t)
	e.Layout(1920, 1080)
	m := e.doc.Script
	m.Conditions[0].AddressesVariable, m.Conditions[0].VariableRegister = true, 67
	m.Actions[1].AddressesVariable, m.Actions[1].VariableRegister = true, 0
	m.Triggers[0].Acts[1] = 11
	m.Labels = map[uint32]string{5: "Spare"}
	return e
}

// The variable window at 1920x1080 holds the layout properties for a
// variable shared with a condition, whose link row opens the condition
// through the editor's input.
func TestVariableWindowHoldsTheLayoutProperties(t *testing.T) {
	e := variableWindowEditor(t)
	e.Select(-2, false)
	s, links, ok := e.objectSpec()
	if !ok {
		t.Fatal("the variable window is not built")
	}
	checkWindowProperties(t, e, s, image.Pt(1920, 1080))
	shared := len(s.Rows) - 1
	if s.Rows[shared].Label != "Shared with Condition 1" || s.Rows[shared].Help == "" {
		t.Fatalf("the last row is not the shared register's link: %+v", s.Rows[shared])
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	modesClick(e, center(rowRectOf(t, l, shared).Rect))
	if e.Selected() != -links[shared]-1 {
		t.Fatalf("a click on the shared link selects %d, want %d", e.Selected(), -links[shared]-1)
	}
}

// A variable no trigger reads or changes says so in a note at the bottom
// of its window, in the label ink, and not in a row; the window holds the
// layout properties at 1920x1080.
func TestVariableWindowSaysUnusedInANote(t *testing.T) {
	e := variableWindowEditor(t)
	for id := -2; ; id-- {
		v, ok := e.scriptVariableRow(id)
		if !ok {
			t.Fatal("the labelled register is not a variable")
		}
		if v.register != 5 {
			continue
		}
		e.Select(id, false)
		break
	}
	s, _, _ := e.objectSpec()
	checkWindowProperties(t, e, s, image.Pt(1920, 1080))
	if s.Title != "Variable 5: Spare" || strings.Join(s.Notes, " ") != e.word("editor.labels.unused") || s.NoteInk != editorPaper {
		t.Errorf("unaddressed variable: title %q, notes %q, ink %v", s.Title, s.Notes, s.NoteInk)
	}
	for _, r := range s.Rows {
		if r.Label == e.word("editor.labels.unused") {
			t.Errorf("the unused line is a row: %+v", r)
		}
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.Notes) == 0 {
		t.Error("the note is not laid out")
	}
	for _, n := range l.Notes {
		if !n.In(l.Rect) {
			t.Errorf("the note at %v leaves the window %v", n, l.Rect)
		}
	}
}
