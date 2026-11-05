package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestEditorWordDeletion(t *testing.T) {
	for _, tc := range []struct {
		name, key, before, after string
	}{
		{"Latin back", "backspace", "one two|", "one |"},
		{"Latin forward", "delete", "|one two", "| two"},
		{"Cyrillic back", "backspace", "один два42|", "один |"},
		{"Cyrillic forward", "delete", "|два42 один", "| один"},
		{"mixed digits back", "backspace", "a9Я2|!", "|!"},
		{"mixed digits forward", "delete", "!|a9Я2", "!|"},
		{"wide rune back", "backspace", "x 𝔸я٢|!", "x |!"},
		{"wide rune forward", "delete", "!|𝔸я٢ x", "!| x"},
		{"punctuation back", "backspace", "one...!?|two", "one|two"},
		{"punctuation forward", "delete", "one|...!?two", "one|two"},
		{"underscore back", "backspace", "one_two|", "one_|"},
		{"underscore forward", "delete", "|one_two", "|_two"},
		{"underscore punctuation back", "backspace", "one_!?|", "one|"},
		{"underscore punctuation forward", "delete", "|_!?one", "|one"},
		{"trailing spaces back", "backspace", "one two   |", "one |"},
		{"leading spaces forward", "delete", "|   one two", "| two"},
		{"leading spaces back", "backspace", "   |one", "|one"},
		{"trailing spaces forward", "delete", "one|   ", "one|"},
		{"spaces then punctuation back", "backspace", "one!?  |two", "one|two"},
		{"spaces then punctuation forward", "delete", "one|  !?two", "one|two"},
		{"middle back", "backspace", "one tw|o", "one |o"},
		{"middle forward", "delete", "one t|wo end", "one t| end"},
		{"start back", "backspace", "|один", "|один"},
		{"end forward", "delete", "один|", "один|"},
		{"empty back", "backspace", "|", "|"},
		{"empty forward", "delete", "|", "|"},
		{"spaces back", "backspace", "   |", "|"},
		{"spaces forward", "delete", "|   ", "|"},
		{"line break back", "backspace", "one\r\n  |two", "|two"},
		{"line break forward", "delete", "one| \r\n two", "one|"},
		{"Unicode space back", "backspace", "one\u00a0|two", "|two"},
		{"Unicode space forward", "delete", "one|\u00a0two", "one|"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := strings.Index(tc.before, "|")
			caret := len([]rune(tc.before[:at]))
			value := strings.Replace(tc.before, "|", "", 1)
			e := NewMapEditor(nil, nil, nil)
			e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "pages", path: value})
			m, _ := e.inputModel()
			m.drag(caret, false)
			if err := e.Key("ctrl+" + tc.key); err != nil {
				t.Fatal(err)
			}
			r := []rune(e.prompt.path)
			got := string(r[:m.Caret]) + "|" + string(r[m.Caret:])
			if got != tc.after || m.Anchor != m.Caret {
				t.Fatalf("Ctrl+%s: %q anchor %d, want %q", tc.key, got, m.Anchor, tc.after)
			}
			if e.prompt.path == value {
				if len(e.input.undo) != 0 {
					t.Fatal("empty deletion added history")
				}
				return
			}
			_ = e.Key("textundo")
			if e.prompt.path != value || m.Caret != caret || m.Anchor != caret {
				t.Fatal("Undo lost text or caret", e.prompt.path, m.Caret, m.Anchor)
			}
			_ = e.Key("textredo")
			if e.prompt.path != strings.Replace(tc.after, "|", "", 1) || len(e.input.undo) != 1 {
				t.Fatal("word deletion is not one redo step", e.prompt.path)
			}
		})
	}
}

func TestEditorWordDeletionUpdate(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyDelete} {
		t.Run(key.String(), func(t *testing.T) {
			e, _ := popupEditor(t)
			if err := e.Open("popups"); err != nil {
				t.Fatal(err)
			}
			e.Select(0, false)
			e.loadObject()
			e.obj.focus = 2
			e.obj.buf[2] = "one два"
			m, _ := e.inputModel()
			want := "one "
			if key == ebiten.KeyDelete {
				m.drag(0, false)
				want = " два"
			}
			updateEditorKeys(t, e, ebiten.KeyControl, key)
			if e.obj.buf[2] != want || len(e.input.undo) != 1 {
				t.Fatalf("Update must delete one word once: %q history %d", e.obj.buf[2], len(e.input.undo))
			}
		})
	}
}

func TestEditorDropdownBackspaceKeepsRuneDeletion(t *testing.T) {
	for _, tc := range []struct{ before, after string }{
		{"", ""}, {"a", ""}, {"два", "дв"}, {"one два", "one дв"}, {"a𝔸", "a"},
	} {
		e := NewMapEditor(nil, nil, nil)
		e.modWindow = true
		e.obj.open = 0
		e.obj.search = tc.before
		e.obj.choice = NewPicker([]PickerRow{{Text: "one два", Choosable: true}})
		if err := e.keyBackspace("backspace"); err != nil {
			t.Fatal(err)
		}
		if e.obj.search != tc.after {
			t.Fatalf("dropdown Backspace: %q -> %q, want %q", tc.before, e.obj.search, tc.after)
		}
	}
}

func TestEditorWordDeletionSelection(t *testing.T) {
	for _, key := range []string{"ctrl+backspace", "ctrl+delete", "ctrl+shift+backspace", "ctrl+shift+delete"} {
		for _, reverse := range []bool{false, true} {
			e := NewMapEditor(nil, nil, nil)
			e.openPrompt(editorPrompt{kind: promptOpen, path: "one два three"})
			m, _ := e.inputModel()
			m.Caret, m.Anchor = 5, 6
			if reverse {
				m.Caret, m.Anchor = m.Anchor, m.Caret
			}
			before := stepOf(m)
			_ = e.Key(key)
			if e.prompt.path != "one да three" || m.Caret != 5 || m.Anchor != 5 {
				t.Fatalf("%s expanded the selection: %q %+v", key, e.prompt.path, m)
			}
			_ = e.Key("textundo")
			if stepOf(m) != before {
				t.Fatalf("%s lost the undone selection", key)
			}
		}
	}
}

func TestEditorWordDeletionNativeInputs(t *testing.T) {
	for _, target := range []string{"dialogue-prompt", "dialogue-field", "record-field", "bar-search", "tree-search"} {
		for _, key := range []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyDelete} {
			t.Run(target+"/"+key.String(), func(t *testing.T) {
				e := NewMapEditor(nil, nil, nil)
				var value *string
				switch target {
				case "dialogue-prompt":
					e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "pages", path: "one два три"})
					value = &e.prompt.path
				case "record-field", "dialogue-field":
					e.doc = &InspectionDocument{Records: []InspectionRecord{{Kind: KindUnit}}}
					e.selected = 0
					e.obj = objectEdit{doc: e.doc, record: 0, focus: 0, open: -1,
						fields: []InspectionField{{Key: "text", Text: true}}, buf: []string{"one два три"}}
					if target == "dialogue-field" {
						e.messagePage = 1
					}
					value = &e.obj.buf[0]
				case "bar-search":
					e.doc = &InspectionDocument{}
					e.search = barSearch{text: "one два три", typing: true}
					value = &e.search.text
				case "tree-search":
					e.doc = &InspectionDocument{}
					e.panels.state[panelLeft].Restored = true
					e.tree.typing, e.tree.search = true, "one два три"
					value = &e.tree.search
				}
				m, _ := e.inputModel()
				if m == nil {
					t.Fatal("fixture did not reach shared input")
				}
				caret, want := 7, "one  три"
				if key == ebiten.KeyDelete {
					caret, want = 3, "one три"
				}
				m.drag(caret, false)
				pressed := func(k ebiten.Key) bool { return k == ebiten.KeyControl || k == key }
				now := time.Unix(100, 0)
				in := editorNativeInput{pressed: pressed, justPressed: pressed}
				if typing, _ := e.updateTextInput(in, now); !typing || *value != want {
					t.Fatalf("native Ctrl+%s: %q want %q", key, *value, want)
				}
				e.updateTextInput(in, now.Add(399*time.Millisecond))
				if *value != want {
					t.Fatal("word repeat fired before delay", *value)
				}
				e.updateTextInput(in, now.Add(400*time.Millisecond))
				want = " три"
				if key == ebiten.KeyDelete {
					want = "one"
				}
				if *value != want {
					t.Fatal("held key lost word deletion", *value, want)
				}
			})
		}
	}
}
