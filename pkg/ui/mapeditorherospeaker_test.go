package ui

import (
	"image"
	"testing"
)

// Resting the pointer on the portrait of a card whose speaker carries a note
// shows the note; the portrait of another speaker, and the card's text, show
// none.
func TestPortraitTipShowsOverAHeroSpeakersPortraitOnly(t *testing.T) {
	e, _ := speakerPortraitEditor(t, "en")
	s, _, ok := e.messageSpec()
	if !ok {
		t.Fatal("the message editor builds no popup")
	}
	const note = "The face depends on the player's hero"
	row := e.fieldRow(0)
	l := layoutPopup(s, e.font, e.bodyFont())
	bounds := image.Rect(0, 0, 1920, 1080)
	for _, rr := range l.Rows {
		if rr.Index != row {
			continue
		}
		box := rowPortraitBox(rr)
		tip, ok := popupTipAt(bounds, e.font, e.bodyFont(), s, l, box.Min.Add(image.Pt(2, 2)), true)
		if !ok || len(tip.Lines) == 0 || tip.Lines[0] == "" {
			t.Fatalf("the portrait shows no tip: %v", ok)
		}
		if tip, ok := popupTipAt(bounds, e.font, e.bodyFont(), s, l, rr.Control.Min.Add(image.Pt(8, 8)), true); ok && len(tip.Lines) > 0 && tip.Lines[0] == note {
			t.Error("the card's text shows the portrait's tip")
		}
	}
	if s.Rows[row].PortraitTip != note {
		t.Fatalf("speaker 5 carries the tip %q", s.Rows[row].PortraitTip)
	}
	other := e.fieldRow(2)
	if s.Rows[other].PortraitTip != "" {
		t.Errorf("another speaker's card carries the tip %q", s.Rows[other].PortraitTip)
	}
}

// The card's tip comes from the speaker field's note for the chosen speaker.
func TestChoiceNoteIsTheChosenSpeakersNote(t *testing.T) {
	f := InspectionField{IconNotes: map[int64]string{21: "note"}}
	if got := choiceNote(f, "21"); got != "note" {
		t.Errorf("speaker 21: %q", got)
	}
	if got := choiceNote(f, "5"); got != "" {
		t.Errorf("speaker 5: %q", got)
	}
}
