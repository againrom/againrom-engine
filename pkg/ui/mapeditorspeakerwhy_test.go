package ui

import (
	"image"
	"slices"
	"strings"
	"testing"
)

// feedClick is one click the way the host feeds the editor: Update frames
// that read a press, a one pixel move with the button held and the release,
// each through the editor's input route into Pointer.
func feedClick(t *testing.T, e *MapEditor, p image.Point) {
	t.Helper()
	for _, f := range []struct {
		dx   int
		down bool
	}{{0, true}, {1, true}, {1, false}} {
		in := Input{CursorX: p.X + f.dx, CursorY: p.Y, PrimaryDown: f.down}
		e.nativeInput = holdNative(func() Input { return in })
		if err := e.Update(); err != nil {
			t.Fatal(err)
		}
	}
	e.nativeInput = nil
}

// noteLines is the popup's notes joined, as the message popup shows them.
func noteLines(t *testing.T, e *MapEditor) string {
	t.Helper()
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("the message popup is not shown")
	}
	return strings.Join(strings.Fields(strings.Join(s.Notes, " ")), " ")
}

// A refused click on a unit shows its reason in the message popup beside the
// armed card, in the language's words, and the status line keeps its line.
// The pick stays armed: the next click, fed through the host's route, answers
// the pick again and neither selects a unit nor opens its window.
func TestMessageSpeakerPickSaysWhyAndStaysArmed(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := speakerPickEditor(t, lang)
			why := e.word("editor.speaker.why.notfirst")
			if strings.HasPrefix(why, "editor.") {
				t.Fatalf("the %s reason has no words", lang)
			}
			prefix := why[:strings.Index(why, "%")]
			if noteLines(t, e) != "" && strings.Contains(noteLines(t, e), prefix) {
				t.Fatal("the reason shows before any click")
			}
			selected := e.Selected()
			pressSpeakerPick(t, e, 0)

			feedClick(t, e, e.CellPoint(cellUnshownUnit))
			if e.pick != pickParamUnit || e.Selected() != selected || e.obj.buf[1] != "-1" {
				t.Fatalf("a refused click left pick %q, selected %d, speaker %q", e.pick, e.Selected(), e.obj.buf[1])
			}
			if notes := noteLines(t, e); !strings.Contains(notes, prefix) || !strings.Contains(notes, "9,8") {
				t.Fatalf("the popup notes read %q, want the reason beginning %q with the cell", notes, prefix)
			}
			if status := e.statusText(); !strings.Contains(status, e.word("editor.param.nospeakerunit")) {
				t.Fatalf("the status line lost its line: %q", status)
			}

			feedClick(t, e, e.CellPoint(cellOpenGround))
			if notes := noteLines(t, e); strings.Contains(notes, prefix) || !strings.Contains(notes, e.word("editor.speaker.why.nounit")) {
				t.Fatalf("a click on open ground left the notes %q", notes)
			}

			feedClick(t, e, e.CellPoint(cellUnshownUnit))
			if e.pick != pickParamUnit || e.Selected() != selected || e.messagePage != 1 {
				t.Fatalf("the second click left pick %q, selected %d, page %d", e.pick, e.Selected(), e.messagePage)
			}
			feedClick(t, e, e.CellPoint(cellShownUnit))
			if e.pick != pickNone || e.obj.buf[1] != "5" || e.Selected() != selected {
				t.Fatalf("a click on a shown unit left pick %q, speaker %q, selected %d", e.pick, e.obj.buf[1], e.Selected())
			}
			if notes := noteLines(t, e); strings.Contains(notes, prefix) {
				t.Fatalf("the reason stays after the pick ends: %q", notes)
			}
		})
	}
}

// The button that armed the speaker pick is drawn armed while the pick is: it
// stays armed through a refused click, and a second press of the button
// disarms the pick and the button. The next click then selects the unit as an
// ordinary click does, which is why a refused pick must show it is armed.
func TestMessageSpeakerPickButtonShowsItIsArmed(t *testing.T) {
	e, _ := speakerPickEditor(t, "en")
	armed := func() bool {
		s, _, _ := e.objectSpec()
		return slices.Contains(s.Rows[e.fieldRow(1)].Armed, true)
	}
	if armed() {
		t.Fatal("the button is drawn armed before it is pressed")
	}
	pressSpeakerPick(t, e, 0)
	if !armed() {
		t.Fatal("the button is not drawn armed after it armed the pick")
	}
	feedClick(t, e, e.CellPoint(cellUnshownUnit))
	if e.pick != pickParamUnit || !armed() {
		t.Fatalf("a refused click left pick %q, armed button %v", e.pick, armed())
	}
	pressSpeakerPick(t, e, 0)
	if e.pick != pickNone || armed() {
		t.Fatalf("a second press left pick %q, armed button %v", e.pick, armed())
	}
	feedClick(t, e, e.CellPoint(cellUnshownUnit))
	if e.Selected() != 1 {
		t.Fatalf("the click after the second press selected record %d, want the unit 1", e.Selected())
	}
	e.Select(0, false)
	e.openMessage(1)
	pressSpeakerPick(t, e, 0)
	_ = e.Key("escape")
	if e.pick != pickNone || armed() {
		t.Fatalf("Esc left pick %q, armed button %v", e.pick, armed())
	}
	pressSpeakerPick(t, e, 0)
	feedClick(t, e, e.CellPoint(cellShownUnit))
	if e.pick != pickNone || armed() || e.obj.buf[1] != "5" {
		t.Fatalf("a successful pick left pick %q, armed button %v, speaker %q", e.pick, armed(), e.obj.buf[1])
	}
}
