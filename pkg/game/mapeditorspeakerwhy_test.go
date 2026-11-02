package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// A refused unit says why, in the cause the pick can word: a creature of a
// class no portrait record carries has no picture, a person whose keys name
// no definition has no placement, and a person a record fits but another
// unit shows first is named with that record and the cell it shows.
func TestUnitSpeakerNamesWhyNone(t *testing.T) {
	_, s := speakerPickEditorOn(t, speakerPickMapOf([]synth.ALMUnit{
		pickUnit(pickPaladinCell, 1, 25, 1, 1),
		pickUnit(pickGoblinCell, pickGoblinClass, 9, 0, 2),
		pickUnit(pickLadyCell, 999, 0, 0, 3),
		pickUnit(pickFighterCell, 1, 999, 1, 4),
		pickUnit(pickFighterCell, 3, 99, 0, 5),
	}))
	for _, tc := range []struct {
		name  string
		unit  int
		cause ui.SpeakerCause
	}{
		{"a goblin with a portrait record", 1, ui.SpeakerShown},
		{"a creature class no record carries", 2, ui.SpeakerNoPicture},
		{"keys that name no definition", 3, ui.SpeakerNoPlacement},
		{"a man shown after the paladin", 4, ui.SpeakerNotFirst},
	} {
		_, why := s.UnitSpeaker(tc.unit)
		if why.Cause != tc.cause {
			t.Errorf("%s: unit %d cause %d, want %d", tc.name, tc.unit, why.Cause, tc.cause)
		}
		if tc.cause == ui.SpeakerNotFirst && (why.Record != 3 || why.Cell != pickPaladinCell) {
			t.Errorf("%s: record %d at %v, want record 3 at %v", tc.name, why.Record, why.Cell, pickPaladinCell)
		}
	}
}
