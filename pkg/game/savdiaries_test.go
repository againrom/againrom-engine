package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestRetiredDiaryKeepsValuesAndExactDeadRoot(t *testing.T) {
	world := func(id sim.EntityID) *sim.World {
		w, err := sim.NewWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, make([]byte, 4), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.ImportOriginalDeadActors([]sim.OriginalDeadActor{{ID: id, Source: sim.OriginalDeadSource{
			Class: sim.GeneratedHumanBinding, Identity: 0x110, State: sim.DeadActorState{Stage: 5, HP: -10001, FineX: 128, FineY: 128}}}}); err != nil {
			t.Fatal(err)
		}
		return w
	}
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		{Class: "Human", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x110}},
			RefSlots: []sav.DocumentRefsData{{Name: "Diary", Objects: []uint16{2}}}}, mustNewDiaryRecord(4, 0)}},
		Actors: []SnapshotSAVActor{{EntityID: 7, ObjectIndex: 1, Retired: true}}}
	w := world(7)
	w.SetSavedDiaries([]sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Actor: 7}, Length: 4,
		Entries: []sim.SavedDiaryEntry{{Index: 1, Count: 7, Remaining: 1017}}}})
	if err := projectSavedDiaries(state, w); err != nil {
		t.Fatal(err)
	}
	state.Actors = nil
	cold := world(19)
	if err := importSavedDiaries(&Mission{World: cold}, state); err != nil {
		t.Fatal(err)
	}
	want := w.SavedDiaries()
	want[0].Owner.Actor = 19
	if !reflect.DeepEqual(cold.SavedDiaries(), want) {
		t.Fatal("retired Diary values or owner lost during import")
	}
	want[0].Entries[0].Count++
	cold.SetSavedDiaries(want)
	if err := projectSavedDiaries(state, cold); err != nil {
		t.Fatal(err)
	}
	diary, err := sav.ReadDocumentDiary(state.Document.Objects[1])
	if err != nil || diary.Length != 4 || len(diary.Entries) != 1 || diary.Entries[0].Count != 8 || diary.Entries[0].Remaining != 1017 {
		t.Fatal("next Diary mutation was not projected", diary, err)
	}
	state.Document.Objects = append(state.Document.Objects, state.Document.Objects[0])
	if err := projectSavedDiaries(state, cold); err == nil {
		t.Fatal("ambiguous dead identity was accepted")
	}
}
