package ui

import (
	"strings"
	"testing"
)

func TestGameFieldsUseSharedTextModel(t *testing.T) {
	t.Run("encoded hero name", func(t *testing.T) {
		c := namedPreCreate()
		c.name = "\x84\xa0"
		c.setup.EncodeName = func(r rune) (byte, bool) { return 0x9f, r == 'Я' }
		c.caret.restart = false
		c.EditName("Я", true)
		if c.name != "\x84\x9f" || !c.caret.restart {
			t.Fatalf("encoded edit %x, restart %v", c.name, c.caret.restart)
		}
		c.caret.restart = false
		c.EditName("🙂", false)
		if c.name != "\x84\x9f" || !c.caret.restart {
			t.Fatal("rejected character changed the old caret restart rule")
		}
	})
	t.Run("dialog name and directory", func(t *testing.T) {
		for _, focus := range []saveControl{saveNameControl, saveDirectoryControl} {
			d := &saveDialog{focus: focus, caret: 1}
			*d.editField() = "абВ"
			f := &flow{saveDialog: d}
			f.editSaveText(appInput{SelectText: true})
			f.editSaveText(appInput{Right: true})
			f.editSaveText(appInput{Backspace: true, Typed: "Q\n"})
			if *d.editField() != "аQВ" || d.caret != 2 || d.selectedText {
				t.Fatal("dialog edit", *d.editField(), d.caret, d.selectedText)
			}
		}
	})
	t.Run("page sticky selection", func(t *testing.T) {
		s := &savePageState{name: []rune("абВ"), caret: 1, page: SavePage{NameLimit: 4}}
		s.edit(appInput{SelectText: true})
		s.edit(appInput{Right: true})
		if s.caret != 2 || !s.selectAll {
			t.Fatal("selection moved", s.caret, s.selectAll)
		}
		s.edit(appInput{Typed: "\n"})
		if len(s.name) != 0 || s.selectAll {
			t.Fatal("filtered edit did not clear selection")
		}
		s.edit(appInput{Typed: "Q12345"})
		if string(s.name) != "Q123" {
			t.Fatal("page cap", string(s.name))
		}
	})
	t.Run("detached amount selection", func(t *testing.T) {
		g := goldState{text: "1234", caret: 4, selStart: 1, selEnd: 3}
		if !g.backspace() || g.text != "123" || g.selStart != 1 || g.selEnd != 3 {
			t.Fatal("amount backspace", g)
		}
		g.move(0, false)
		g.insert("Q\nЯ")
		if g.text != "Q123" || g.caret != 1 || g.selStart != 1 || g.selEnd != 1 {
			t.Fatal("amount insert", g)
		}
	})
	t.Run("bounded chat", func(t *testing.T) {
		a, _ := mkOnMap(t)
		a.SetCheatCommands(func(string) {}, nil)
		_ = a.HeadlessKey("enter")
		_ = a.HeadlessType(strings.Repeat("я", 127)+"Q🙂", false)
		if line, _ := a.HeadlessChatState(); len(line) != 255 || !strings.HasSuffix(line, "Q") {
			t.Fatal("chat limit", len(line))
		}
		_ = a.HeadlessType("", true)
		if line, _ := a.HeadlessChatState(); line != strings.Repeat("я", 127) {
			t.Fatal("chat backspace", line)
		}
	})
}

func TestAppendOnlyPolicyRefusesEditorKeys(t *testing.T) {
	state := EditText(TextInputState{Text: "abc", Caret: 1, SelectedAll: true},
		TextInputKeys{Home: true, Left: true, Delete: true, SelectAll: true, Typed: "Q"},
		TextInputRules{AppendOnly: true, Limit: 4})
	if state.Text != "abcQ" || state.Caret != 4 || state.SelectedAll {
		t.Fatal(state)
	}
}

func TestSharedTextStateKeepsKeyboardSelection(t *testing.T) {
	state := TextInputState{Text: "ab", Caret: 2, Anchor: 2}
	state = EditText(state, TextInputKeys{Left: true, Shift: true}, TextInputRules{})
	if state.Caret != 1 || state.Anchor != 2 {
		t.Fatal(state)
	}
	state = EditText(state, TextInputKeys{Typed: "Q"}, TextInputRules{})
	if state.Text != "aQ" || state.Caret != 2 || state.Anchor != 2 {
		t.Fatal(state)
	}
}

func TestGameArrowPolicyKeepsWordKeysLiteral(t *testing.T) {
	state := TextInputState{Text: "one two", Caret: 7, Anchor: 7}
	game := EditText(state, TextInputKeys{Left: true, Ctrl: true}, TextInputRules{LiteralArrows: true})
	editor := EditText(state, TextInputKeys{Left: true, Ctrl: true}, TextInputRules{})
	if game.Caret != 6 || editor.Caret != 4 {
		t.Fatal(game.Caret, editor.Caret)
	}
}

func TestSaveNameAdaptersKeepControlArrowsLiteral(t *testing.T) {
	for _, page := range []bool{true, false} {
		name := "dialog"
		if page {
			name = "page"
		}
		t.Run(name, func(t *testing.T) {
			s := &savePageState{name: []rune("one two"), caret: 7, page: SavePage{NameLimit: 20}}
			d := &saveDialog{focus: saveNameControl, caret: 7}
			*d.editField() = "one two"
			f := &flow{saveDialog: d}
			for _, left := range []bool{true, false} {
				in := appInput{Left: left, Right: !left, Viewer: Input{Ctrl: true}}
				caret, value := 0, ""
				if page {
					s.edit(in)
					caret, value = s.caret, string(s.name)
				} else {
					f.editSaveText(in)
					caret, value = d.caret, *d.editField()
				}
				want := 7
				if left {
					want = 6
				}
				if caret != want || value != "one two" {
					t.Fatalf("Ctrl arrow: caret %d want %d, text %q", caret, want, value)
				}
			}
		})
	}
}
