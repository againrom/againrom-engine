package ui

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
)

// A long kept error under the open save prompt keeps its first lines and the
// kept line in the window body at every screen size; the whole text stays in
// the title's help.
func TestLongKeptErrorUnderSavePromptStaysInTheBody(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
			for _, n := range []int{20, 25, 30} {
				t.Run(lang+"/"+strconv.Itoa(size.X)+"/"+strconv.Itoa(n), func(t *testing.T) {
					e := promptsEditor(t, lang, size)
					e.pathFallback(pathdialog.SaveFile, pathdialog.ErrUnavailable)
					e.TextInput("81.alm")
					var lines []string
					for i := range n {
						lines = append(lines, fmt.Sprintf("errline%02d %s", i, strings.Repeat("w", 70)))
					}
					e.keptError(errors.New(strings.Join(lines, "\n")))
					s, ok := e.promptSpec()
					if !ok {
						t.Fatal("no window")
					}
					l := layoutPopup(s, e.font, e.bodyFont())
					if len(l.Notes) == 0 {
						t.Fatalf("no note in the body (spec notes %d)", len(s.Notes))
					}
					joined := strings.Join(s.Notes, "\n")
					if !strings.Contains(joined, "errline00") || !strings.Contains(joined, e.word("editor.message.kept")) {
						t.Fatalf("the body lacks the first line or the kept line: %q", joined)
					}
					if !strings.Contains(s.TitleHelp+joined, "errline"+fmt.Sprintf("%02d", n-1)) {
						t.Fatalf("the last line is lost")
					}
					if r := l.Rect; r.Min.X < 0 || r.Min.Y < 0 || r.Max.X > size.X || r.Max.Y > size.Y {
						t.Fatalf("window %v outside the screen %v", r, size)
					}
					if len(l.Rows) == 0 {
						t.Fatal("the prompt's field is gone")
					}
				})
			}
		}
	}
}
