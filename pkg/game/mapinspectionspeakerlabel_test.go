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

// The speaker list names every npc record, and each name asks which Humans
// row the record reaches. The rows are worked out once for a table and
// mission, not once per record and per list built; a new mission rebuilds.
func TestNpcRowNamesAreBuiltOncePerTableAndMission(t *testing.T) {
	_, s, _ := nativeIdentityEditor(t)
	x := s.inspector
	first := x.npcChoices(255, true)
	for range 5 {
		x.npcChoices(255, true)
	}
	if x.npcRows.builds != 1 {
		t.Fatalf("six speaker lists built the row names %d times, want 1", x.npcRows.builds)
	}
	x.identityMission++
	again := x.npcChoices(255, true)
	if x.npcRows.builds != 2 {
		t.Fatalf("a new mission built the row names %d times in all, want 2", x.npcRows.builds)
	}
	if len(again) != len(first) {
		t.Fatalf("the list has %d choices after the rebuild, %d before", len(again), len(first))
	}
}
