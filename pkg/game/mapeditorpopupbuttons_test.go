package game

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// promptButtonCases adds the prompt-place windows popupCases does not open: the
// single or network map question and the test-play notice.
func promptButtonCases() []popupCase {
	cases := popupCases()
	cases = append(cases,
		popupCase{"new map mode question", func(t *testing.T) *ui.MapEditor {
			e := scriptMap77(t)
			_ = e.Key("new")
			_ = e.Key("enter")
			return e
		}},
		popupCase{"test-play notice", func(t *testing.T) *ui.MapEditor {
			e := scriptMap77(t)
			_ = e.Key("testplay")
			return e
		}})
	return cases
}

func scriptMap77(t *testing.T) *ui.MapEditor {
	e, _ := messagePopupFixture(t, 77)
	return e
}

// promptSign is what a press on a prompt window can change: whether it is
// open, its note lines and the Add step.
func promptSign(e *ui.MapEditor) string {
	_, open := e.PromptRect()
	key, _, _ := e.AddStep()
	return fmt.Sprint(open, strings.Join(e.PromptNotes(), "|"), key)
}

// pressPromptButton presses the drawn button n and reports whether the press
// changed the window or its state.
func pressPromptButton(t *testing.T, e *ui.MapEditor, n int) bool {
	t.Helper()
	r, ok := e.PromptButtonRect(n)
	if !ok {
		t.Fatalf("no drawn button %d", n)
	}
	before := promptSign(e)
	p := r.Min.Add(r.Size().Div(2))
	now := time.Now()
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now.Add(time.Millisecond))
	return promptSign(e) != before
}

func promptButtonCount(e *ui.MapEditor) int {
	n := 0
	for {
		if _, ok := e.PromptButtonRect(n); !ok {
			return n
		}
		n++
	}
}

// Every editor window of the prompt place, resized by its bottom edge and by
// its bottom-right corner, acts on each drawn button as it does at its natural
// size, and its last button (Cancel or OK) closes it. The press layout is the
// layout drawn.
func TestEditorPromptButtonsActAfterResize(t *testing.T) {
	resizes := map[string]image.Point{"edge": {0, 40}, "corner": {60, 40}}
	for _, c := range promptButtonCases() {
		probe := c.open(t)
		if _, ok := probe.PromptRect(); !ok {
			continue
		}
		count := promptButtonCount(probe)
		for n := 0; n < count; n++ {
			want := pressPromptButton(t, c.open(t), n)
			for name, d := range resizes {
				t.Run(fmt.Sprintf("%s/%s/button %d", c.name, name, n), func(t *testing.T) {
					e := c.open(t)
					pick := func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-2, r.Max.Y-2) }
					if name == "edge" {
						pick = func(r image.Rectangle) image.Point { return image.Pt(r.Min.X+r.Dx()/2, r.Max.Y-1) }
					}
					before, after := borderDrag(t, e, pick, d)
					if after.Dy() <= before.Dy() {
						t.Fatalf("the window did not grow: %v -> %v", before, after)
					}
					if got := pressPromptButton(t, e, n); got != want {
						t.Fatalf("pressing drawn button %d after the resize changed the window: %v, at natural size: %v", n, got, want)
					}
					if n == count-1 {
						if _, open := e.PromptRect(); open {
							t.Fatalf("the last drawn button left the window open")
						}
					}
				})
			}
		}
	}
}
