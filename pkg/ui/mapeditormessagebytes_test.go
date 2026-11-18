package ui

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

type refusingCopy struct {
	toolAreaHost
	reason string
}

func (h refusingCopy) CopyTrigger(int) (*InspectionDocument, error) { return nil, errors.New(h.reason) }

func oneByteHost(s string) string {
	var b []byte
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		switch {
		case r == utf8.RuneError && n <= 1:
			b = append(b, '?')
		case r >= 0x80:
			b = append(b, byte(0xC0+r%0x40))
		default:
			b = append(b, byte(r))
		}
		s = s[n:]
	}
	return string(b)
}

func TestMapEditorMessageBoxShowsTextOnceConverted(t *testing.T) {
	reason := map[string]string{"": "Check register 50 is taken (the map holds 100 of them).", "russian": "Эта проверка занимает регистр 50 (на карте их 100)."}
	for _, lang := range []string{"", "russian"} {
		cases := []struct {
			name string
			run  func(e *MapEditor)
			want func(e *MapEditor) string
		}{
			{"copy refused", func(e *MapEditor) { e.doc.Edits = refusingCopy{toolAreaHost{}, reason[lang]}; e.copyTrigger(0) },
				func(*MapEditor) string { return reason[lang] }},
			{"copy with no trigger selected", func(e *MapEditor) { e.copySelectedTrigger() },
				func(e *MapEditor) string { return e.plainWord("editor.trigger.copyfirst") }},
			{"tabs full", func(e *MapEditor) { e.message = e.plainWord("editor.tabs.limit") },
				func(e *MapEditor) string { return e.plainWord("editor.tabs.limit") }},
		}
		for _, c := range cases {
			t.Run(lang+"/"+c.name, func(t *testing.T) {
				e, p, _ := menuEditor(t, lang)
				e.hostText = oneByteHost
				if c.name == "copy refused" {
					e.doc.Edits = refusingCopy{toolAreaHost{p}, reason[lang]}
				}
				c.run(e)
				s, ok := e.promptSpec()
				lines := messageLines(s)
				if !ok || len(lines) == 0 {
					t.Fatalf("no message box: %v %v", ok, s.Rows)
				}
				got := strings.Join(lines, " ")
				want := oneByteHost(c.want(e))
				if got != want || strings.Contains(got, "?") && !strings.Contains(want, "?") {
					t.Fatalf("the box shows %q, want %q", got, want)
				}
			})
		}
	}
}
