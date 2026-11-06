package ui

import (
	"fmt"
	"testing"
)

func TestEditorWordKeyTable(t *testing.T) {
	for _, row := range []struct {
		text                       string
		left, right, back, forward []int
	}{
		{"foo_bar baz",
			[]int{0, 0, 0, 0, 3, 4, 4, 4, 4, 8, 8, 8},
			[]int{3, 3, 3, 4, 7, 7, 7, 11, 11, 11, 11, 11},
			[]int{0, 0, 0, 0, 3, 4, 4, 4, 4, 8, 8, 8},
			[]int{3, 3, 3, 4, 7, 7, 7, 11, 11, 11, 11, 11}},
		{"a.b,c",
			[]int{0, 0, 1, 2, 3, 4}, []int{1, 2, 3, 4, 5, 5},
			[]int{0, 0, 1, 2, 3, 4}, []int{1, 2, 3, 4, 5, 5}},
		{"слово-слово",
			[]int{0, 0, 0, 0, 0, 0, 5, 6, 6, 6, 6, 6},
			[]int{5, 5, 5, 5, 5, 6, 11, 11, 11, 11, 11, 11},
			[]int{0, 0, 0, 0, 0, 0, 5, 6, 6, 6, 6, 6},
			[]int{5, 5, 5, 5, 5, 6, 11, 11, 11, 11, 11, 11}},
		{"  foo  ",
			[]int{0, 0, 0, 2, 2, 2, 2, 2}, []int{5, 5, 5, 5, 5, 7, 7, 7},
			[]int{0, 0, 0, 2, 2, 2, 2, 2}, []int{5, 5, 5, 5, 5, 7, 7, 7}},
		{"  слово  ",
			[]int{0, 0, 0, 2, 2, 2, 2, 2, 2, 2}, []int{7, 7, 7, 7, 7, 7, 7, 9, 9, 9},
			[]int{0, 0, 0, 2, 2, 2, 2, 2, 2, 2}, []int{7, 7, 7, 7, 7, 7, 7, 9, 9, 9}},
		{"a²b",
			[]int{0, 0, 1, 2}, []int{1, 2, 3, 3},
			[]int{0, 0, 1, 2}, []int{1, 2, 3, 3}},
		{" \t\u00a0",
			[]int{0, 0, 0, 0}, []int{3, 3, 3, 3},
			[]int{0, 0, 0, 0}, []int{3, 3, 3, 3}},
		{"", []int{0}, []int{0}, []int{0}, []int{0}},
		{"a_!?b",
			[]int{0, 0, 1, 1, 1, 4}, []int{1, 4, 4, 4, 5, 5},
			[]int{0, 0, 1, 1, 1, 4}, []int{1, 4, 4, 4, 5, 5}},
	} {
		for _, key := range []struct {
			name   string
			stops  []int
			shift  bool
			delete bool
		}{
			{"ctrl+left", row.left, false, false},
			{"ctrl+right", row.right, false, false},
			{"ctrl+shift+left", row.left, true, false},
			{"ctrl+shift+right", row.right, true, false},
			{"ctrl+backspace", row.back, false, true},
			{"ctrl+delete", row.forward, false, true},
		} {
			runes := []rune(row.text)
			if len(key.stops) != len(runes)+1 {
				t.Fatalf("%q %s: missing caret positions", row.text, key.name)
			}
			for at, stop := range key.stops {
				t.Run(fmt.Sprintf("%q/%s/%d", row.text, key.name, at), func(t *testing.T) {
					e := NewMapEditor(nil, nil, nil)
					e.openPrompt(editorPrompt{kind: promptWorkspace, scope: "pages", path: row.text})
					m, _ := e.inputModel()
					m.drag(at, false)
					if err := e.Key(key.name); err != nil {
						t.Fatal(err)
					}
					text, caret, anchor, selected := row.text, stop, stop, ""
					lo, hi := min(at, stop), max(at, stop)
					if key.delete {
						text, caret, anchor = string(runes[:lo])+string(runes[hi:]), lo, lo
					} else if key.shift {
						anchor, selected = at, string(runes[lo:hi])
					}
					if e.prompt.path != text || m.Caret != caret || m.Anchor != anchor || m.selected() != selected {
						t.Fatalf("text %q caret %d anchor %d selected %q; want %q caret %d anchor %d selected %q",
							e.prompt.path, m.Caret, m.Anchor, m.selected(), text, caret, anchor, selected)
					}
				})
			}
		}
	}
}
