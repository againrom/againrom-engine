package game

import (
	"strings"
	"testing"
)

// A speaker is labelled with the name the game gives a person placed from the
// same record: line n of the name table, counted from 1, for record n. The
// speaker list, the dialogue rows and the placed-person list read one label.
func TestSpeakerLabelIsTheNameOfTheRecordsLine(t *testing.T) {
	_, s, _ := nativeIdentityEditor(t)
	x := s.inspector
	for id, want := range map[int64]string{7: "Guard native name", 8: "Courier native name"} {
		label, ok := x.speakerLabel(id)
		if !ok || label != want {
			t.Errorf("speaker %d is labelled %q, %v; want %q", id, label, ok, want)
		}
	}
	for _, roles := range []bool{false, true} {
		got := map[int64]string{}
		for _, c := range x.npcChoices(255, roles) {
			got[c.Value] = c.Label
		}
		for id, want := range map[int64]string{8: "8: Courier native name"} {
			if !strings.Contains(got[id], want) {
				t.Errorf("roles=%v: choice %d reads %q, want %q", roles, id, got[id], want)
			}
		}
	}
	if label, _ := x.speakerLabel(9); strings.Contains(label, "Courier native name") {
		t.Errorf("speaker 9 takes the name of line 8: %q", label)
	}
}
