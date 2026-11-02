package game

import (
	"fmt"
	"testing"

	"againrom/internal/firstgame"
)

// On the installed map whose fifth dialogue the owner reported, speaker 57 is
// the record of a man mage: the speaker field names him with the game's name
// for a person of that record (line 57 of the name table, counted from 1), not
// with the next line, which belongs to record 58, a woman.
func TestReleaseDialogueSpeakerIsNamedByItsRecordsLine(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/130.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if len(x.table.NPCNames) < 58 || x.table.NPCNames[56] == "" || x.table.NPCNames[56] == x.table.NPCNames[57] {
		t.Fatalf("the install's name table does not hold lines 57 and 58: %d lines", len(x.table.NPCNames))
	}
	want := fmt.Sprintf("57: %s", x.inspectionUTF8(x.messageUTF8([]byte(x.table.NPCNames[56]))))
	other := x.inspectionUTF8(x.messageUTF8([]byte(x.table.NPCNames[57])))
	label, ok := x.speakerLabel(57)
	if !ok || label != x.messageUTF8([]byte(x.table.NPCNames[56])) {
		t.Fatalf("speaker 57 is labelled %q, %v; want the name on line 57 of the name table", label, ok)
	}
	fields := s.MessageFields(5)
	if len(fields) < 2 {
		t.Fatal("the map holds no dialogue 5")
	}
	speaker := fields[1]
	found := false
	for _, c := range speaker.Choices {
		if c.Value == 57 {
			found = true
			if c.Label != want {
				t.Errorf("the speaker choice 57 reads %q, want %q (not %q)", c.Label, want, other)
			}
		}
	}
	if !found {
		t.Fatal("the speaker list holds no speaker 57")
	}
	if speaker.Icons[57] == nil {
		t.Error("speaker 57 has no portrait on the map")
	}
}
