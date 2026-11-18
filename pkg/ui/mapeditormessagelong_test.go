package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// A dialog whose first cell ends the dialog for its hero opens with the caret
// in the first text the window shows, and a typed key writes no text the
// window does not show; a window with no text takes no typed text.
func TestMessageWindowTypesOnlyIntoShownText(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			cells := HeroCells(true, false)
			e := messageWindowEditor(t, lang, []InspectionMessagePart{
				{Source: 0, Number: 1, Heroes: cells[0], Ends: true},
				{Source: 1, Number: 1, Body: "Only him", Heroes: cells[1], Speaker: 5},
			})
			if f := e.obj.focus; f != 2 {
				t.Fatalf("the window opens with the caret in field %d, want 2 (the first text shown)", f)
			}
			e.TextInput("zzz")
			if e.obj.buf[0] != "" || !strings.Contains(e.obj.buf[2], "zzz") {
				t.Errorf("typed text went to the end cell %q and the shown text %q", e.obj.buf[0], e.obj.buf[2])
			}
			e = messageWindowEditor(t, lang, []InspectionMessagePart{{Source: 0, Number: 1, Heroes: cells[0], Ends: true}})
			e.TextInput("zzz")
			if e.obj.buf[0] != "" {
				t.Errorf("a window with no text took the typed text %q", e.obj.buf[0])
			}
		})
	}
}

// longMessageCases are texts that were taller than the room the window's
// other rows leave: one long message, one long paragraph, ten long messages.
func longMessageCases(lang string) map[string][]InspectionMessagePart {
	var lines []string
	for k := range 60 {
		lines = append(lines, fmt.Sprintf("Line %d", k))
	}
	sentence := "A sentence of the message that goes on. "
	if lang == "ru" {
		sentence = "Предложение сообщения, которое идёт дальше. "
	}
	var ten []InspectionMessagePart
	for k := range 10 {
		ten = append(ten, InspectionMessagePart{Source: k, Number: k + 1, Body: strings.Repeat(sentence, 1700/len([]rune(sentence))+1), Speaker: 5})
	}
	return map[string][]InspectionMessagePart{
		"one of 60 lines":     {{Source: 0, Number: 1, Body: strings.Join(lines, "\n"), Speaker: 5}},
		"one of 8000 letters": {{Source: 0, Number: 1, Body: strings.Repeat(sentence, 8000/len([]rune(sentence))+1), Speaker: 5}},
		"ten of 1700 letters": ten,
	}
}

// However long its texts, the message window and its Apply and Cancel stand
// inside 1920x1080, and every line of every message is reached: the text
// scrolls in its field by the wheel over it and by its own bar.
func TestMessageWindowLongTextsStayInsideAndScroll(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		for name, parts := range longMessageCases(lang) {
			t.Run(lang+" "+name, func(t *testing.T) {
				e := messageWindowEditor(t, lang, parts)
				s, _, ok := e.messageSpec()
				if !ok {
					t.Fatal("the message window is not built")
				}
				checkWindowProperties(t, e, s, screen.Size())
				l := layoutPopup(s, e.font, e.bodyFont())
				if !l.Rect.In(screen) {
					t.Fatalf("the window stands at %v", l.Rect)
				}
				for k := range parts {
					field := 2 * k
					e.obj.focus = -1
					e.objectTop = max(0, e.fieldRow(field)-1)
					s, _, _ = e.messageSpec()
					l = layoutPopup(s, e.font, e.bodyFont())
					at := e.fieldRow(field)
					var rr popupRowRect
					for _, r := range l.Rows {
						if r.Index == at {
							rr = r
						}
					}
					if rr.Control.Empty() {
						t.Fatalf("message %d (row %d) is not laid out at top %d", k, at, e.objectTop)
					}
					body := e.bodyFont()
					visible := (rr.Control.Dy() - 8) / fontHeight(body, 15)
					total := 0
					for _, part := range strings.Split(s.Rows[at].Value, "\n") {
						total += max(1, len(editorTextLines(body, part, rr.Control.Dx()-8)))
					}
					covered := visible
					for step := 0; step < 4000 && covered < total; step++ {
						before := s.Rows[at].First
						p := center(rr.Control)
						e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -3}, time.Unix(1, 0))
						s, _, _ = e.messageSpec()
						if got := s.Rows[at].First; got == before {
							break
						} else {
							covered = got + visible
						}
					}
					if covered < total {
						t.Errorf("message %d: %d of %d lines reached", k, covered, total)
					}
				}
			})
		}
	}
}
