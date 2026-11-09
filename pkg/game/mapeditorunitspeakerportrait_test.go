package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// The goblin's unit window carries the face the speaker list shows for the
// speaker UnitSpeaker names, and that is the face the mission's dialogue draws
// for that speaker. The woman no speaker shows carries no face and says why.
func TestUnitWindowShowsTheFaceItsSpeakerWears(t *testing.T) {
	_, s := speakerPickEditor(t)
	const goblin, lady = 3, 2

	n, miss := s.UnitSpeaker(goblin)
	if miss.Cause != ui.SpeakerShown || n != 41 {
		t.Fatalf("UnitSpeaker(goblin) = %d, %v; want speaker 41", n, miss)
	}
	field := s.speakerField(goblin)
	if field.Picture == nil {
		t.Fatal("the goblin's speaker field has no picture")
	}
	listed := s.npcPortraits([]ui.InspectionChoice{{Value: n}})[n]
	if listed == nil || !bytes.Equal(field.Picture.Pix, listed.Pix) {
		t.Fatal("the window's picture is not the speaker list's picture")
	}
	drawn := missionStartFace(t, s, int(n))
	if field.Picture.Bounds() != drawn.Bounds() || !bytes.Equal(field.Picture.Pix, drawn.Pix) {
		t.Fatal("the window's picture is not the face the message window draws")
	}

	_, miss = s.UnitSpeaker(lady)
	if miss.Cause == ui.SpeakerShown {
		t.Fatal("a speaker shows the woman; the fixture changed")
	}
	field = s.speakerField(lady)
	if field.Picture != nil {
		t.Fatal("a refused unit has a picture")
	}
	x := s.inspector
	label, _ := x.speakerLabel(miss.Record)
	want := ui.SpeakerMissText(func(id string) string { return x.inspectionUTF8(x.word(id)) }, &miss, fmt.Sprintf("%d: %s", miss.Record, x.inspectionUTF8(label)))
	if len(field.Lines) != 2 || field.Lines[1].Text != want || strings.TrimSpace(want) == "" {
		t.Fatalf("the refused unit's lines %v, want the refusal %q", field.Lines, want)
	}
}

func TestUnitWindowWithAFaceKeepsInsideTheEditor(t *testing.T) {
	e, _ := speakerPickEditor(t)
	e.Layout(1280, 800)
	e.Select(recordIndex(t, e, "Unit", 3), true)
	c := &editorClicker{t: t}
	for _, view := range []string{"first", "switched"} {
		if view == "switched" {
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlViewSwitch})
			c.click(e, r, ok, "view switch")
		}
		e.ScrollDetails(-100)
		for step := 0; step < 40; step++ {
			r, ok := e.PopupRect()
			if !ok || r.Min.Y < 0 || r.Max.Y > 800 {
				t.Fatalf("%s view, scroll %d: the unit window stands at %v %v", view, step, r, ok)
			}
			e.ScrollDetails(1)
		}
	}
}
