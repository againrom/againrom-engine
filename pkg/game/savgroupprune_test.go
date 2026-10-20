package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

// A mission SAVE drops the empty Groups of a Player whose participant word
// +28 is zero and keeps those of a Player whose word is one (SAV-1251,
// SAV-1234). The left Player holds empty Groups 1 and 2 and Group 3 with two
// actors; the right Player holds empty Group 4 and Group 5 with one actor.
func TestMissionSaveDropsOnlyParticipantZeroEmptyGroups(t *testing.T) {
	for _, tc := range []struct {
		name     string
		words    []uint32
		live     []uint32
		inPlayer []int
	}{
		{"left word zero", []uint32{0, 1}, []uint32{3, 4, 5}, []int{1, 2}},
		// Loss control: the word, not the Player's place, decides. A session
		// needs exactly one human Player, so these two are the cases.
		{"right word zero", []uint32{1, 0}, []uint32{1, 2, 3, 5}, []int{3, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newGroupFront(t, -1)
			raw := newGroupLiteral(t, f, false, tc.words...)
			open, town, err := f.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatal("synthetic LOAD", town, err)
			}
			if err := f.App("SAVE Group pruning").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if groups, _, _ := f.live.world.SavedGroups(); len(groups) != 5 {
				t.Fatal("LOAD did not restore all five Groups", groups)
			}
			var s Snapshot
			err = f.captureSave(captureCommitted, true, func(c Snapshot, _ string) error {
				s = c
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			groups, _, _ := f.live.world.SavedGroups()
			var ids []uint32
			for _, g := range groups {
				ids = append(ids, g.ID)
			}
			if !slices.Equal(ids, tc.live) {
				t.Fatalf("live Groups after SAVE %v, want %v", ids, tc.live)
			}
			out, err := f.ExportCurrentSave(s, "SAVE Group pruning")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sav.DecodeDocumentData(out); err != nil {
				t.Fatal(err)
			}
			doc := s.SavedDocument.Document
			// An actor in no saved Group gets a root-only Group of its own
			// beside the listed ones; only the listed Groups are counted.
			var counts []int
			for i, index := range doc.Players {
				if index == 0 || slices.Contains(doc.Players[:i], index) {
					continue
				}
				listed, written := 0, 0
				for _, b := range s.SavedDocument.GroupBindings.Groups {
					if b.PlayerObject == index {
						written++
						if !b.RootOnly {
							listed++
						}
					}
				}
				if written != len(doc.Objects[index-1].Groups) {
					t.Fatal("written Groups and bindings disagree", index, written, len(doc.Objects[index-1].Groups))
				}
				counts = append(counts, listed)
			}
			if !slices.Equal(counts, tc.inPlayer) {
				t.Fatalf("written Groups per Player %v, want %v", counts, tc.inPlayer)
			}
		})
	}
}
