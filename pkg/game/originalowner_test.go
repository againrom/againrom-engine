package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
)

func TestOriginalPositionsRestoreSavedOwnerWithoutReplayingGroups(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{
		{UnitID: 136, Owner: 5, GroupID: 8},
		{UnitID: 137, Owner: 1, GroupID: 9},
		{UnitID: 138, Owner: 5, GroupID: 10},
		{UnitID: 139, Owner: 5, GroupID: 11},
	}}
	f := &sav.File{Actors: []sav.Actor{
		{MapUnitID: 136, OwnerSlot: 1, RuntimeID: 5, HP: 10},
		{MapUnitID: 137, OwnerSlot: 0, RuntimeID: 6, HP: 10},
		{MapUnitID: 138, OwnerSlot: 1, RuntimeID: 7, HP: 0},
		{MapUnitID: 900, OwnerSlot: 1, RuntimeID: 8, HP: 10},
	}}
	var report OriginalSaveResume
	if err := applyOriginalPositions(m, f, &report); err != nil {
		t.Fatal(err)
	}
	if report.Joined != 2 || report.Reowned != 2 || len(m.Units) != 4 {
		t.Fatalf("joined=%d reowned=%d units=%d", report.Joined, report.Reowned, len(m.Units))
	}
	for i, want := range []uint32{1, 0, 5, 5} {
		if m.Units[i].Owner != want || m.Units[i].GroupID != uint32(8+i) {
			t.Fatalf("unit%d: %+v", i, m.Units[i])
		}
	}
}

func TestOriginalOwnerJoinFailureDoesNotPartiallyMutateMap(t *testing.T) {
	for _, duplicateSource := range []bool{false, true} {
		m := &alm.Map{Units: []alm.Unit{{UnitID: 136, Owner: 5, X: 300, Y: 400}, {UnitID: 137, Owner: 2}}}
		f := &sav.File{Actors: []sav.Actor{
			{MapUnitID: 136, OwnerSlot: 1, RuntimeID: 5, HP: 10},
			{MapUnitID: 137, OwnerSlot: 1, RuntimeID: 6, HP: 10},
		}}
		if duplicateSource {
			f.Actors = append(f.Actors, f.Actors[1])
		} else {
			m.Units = append(m.Units, m.Units[1])
		}
		before := append([]alm.Unit(nil), m.Units...)
		var report OriginalSaveResume
		if err := applyOriginalPositions(m, f, &report); err == nil {
			t.Fatal("ambiguous join accepted")
		}
		if !reflect.DeepEqual(before, m.Units) || report.Joined != 0 || report.Reowned != 0 {
			t.Fatal("failed join changed positions, owners or report")
		}
	}
}
