package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// diaryPlayersDocument is two Player roots, the local Player and a computer
// Player, each with its own inline Diary; diaries[i] is root i's content.
func diaryPlayersDocument(t *testing.T, diaries ...sav.Diary) *SnapshotSAVDocument {
	t.Helper()
	doc := &sav.DocumentData{}
	for i, d := range diaries {
		r := mustNewRecord("Player")
		mustSetValue(&r, "Slot", uint32(i+1))
		mustSetValue(&r, "Participant", min(uint32(i), 1))
		diary, err := sav.ProjectDocumentDiary(mustNewDiaryRecord(119, uint32(0x4c6f+i)), d)
		if err != nil {
			t.Fatal(err)
		}
		r.Inline[0].Record = diary
		doc.Objects = append(doc.Objects, r)
		doc.Players = append(doc.Players, uint16(i+1))
	}
	state := &SnapshotSAVDocument{Document: doc}
	if err := importCurrentPlayerRoots(state); err != nil {
		t.Fatal(err)
	}
	return state
}

var (
	diaryLocalPlayer = sav.Diary{Length: 119, Entries: []sav.DiaryEntry{{Index: 64, Count: 2, Remaining: 1022}, {Index: 72, Count: 4, Remaining: 1020}}}
	diaryOtherPlayer = sav.Diary{Length: 119, Entries: []sav.DiaryEntry{{Index: 72, Count: 1, Remaining: 1023}}}
)

// loadDiaryPlayers is LOAD of a two-Player document; saved is the document
// the SAVE builder makes, every Diary at its construction default.
func loadDiaryPlayers(t *testing.T) (w *sim.World, loaded, saved *SnapshotSAVDocument) {
	t.Helper()
	loaded = diaryPlayersDocument(t, diaryLocalPlayer, diaryOtherPlayer)
	w, err := sim.NewWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, make([]byte, 4), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := importSavedDiaries(&Mission{World: w}, loaded); err != nil {
		t.Fatal(err)
	}
	return w, loaded, diaryPlayersDocument(t, sav.Diary{Length: 119}, sav.Diary{Length: 119})
}

func TestComputerPlayerDiaryLoadThenSaveKeepsBytes(t *testing.T) {
	w, loaded, saved := loadDiaryPlayers(t)
	if err := projectSavedDiaries(saved, w); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Document.Objects[1].Inline[0].Record, loaded.Document.Objects[1].Inline[0].Record) {
		got, _ := sav.ReadDocumentDiary(saved.Document.Objects[1].Inline[0].Record)
		t.Fatalf("computer Player Diary changed across LOAD then SAVE: %+v", got)
	}
}

func TestComputerPlayerDiaryChangeReachesSave(t *testing.T) {
	w, _, saved := loadDiaryPlayers(t)
	diaries := w.SavedDiaries()
	at := slices.IndexFunc(diaries, func(d sim.SavedDiary) bool { return !d.Owner.Player && d.Owner.Actor == 0 })
	if at < 0 {
		t.Fatal("LOAD did not carry the computer Player's Diary")
	}
	diaries[at].Entries[0].Count, diaries[at].Entries[0].Remaining = 2, 1022
	w.SetSavedDiaries(diaries)
	if err := projectSavedDiaries(saved, w); err != nil {
		t.Fatal(err)
	}
	got, err := sav.ReadDocumentDiary(saved.Document.Objects[1].Inline[0].Record)
	if err != nil || !reflect.DeepEqual(got.Entries, []sav.DiaryEntry{{Index: 72, Count: 2, Remaining: 1022}}) {
		t.Fatal("computer Player Diary change did not reach SAVE", got, err)
	}
}

func TestLocalPlayerDiaryLoadThenSaveUnchanged(t *testing.T) {
	w, loaded, saved := loadDiaryPlayers(t)
	if d := w.SavedDiaries(); len(d) == 0 || d[0].Owner != (sim.SavedDiaryOwner{Player: true}) || !reflect.DeepEqual(d[0].Entries, savDiaryEntriesToSaved(diaryLocalPlayer.Entries)) {
		t.Fatalf("local Player Diary import changed: %+v", d)
	}
	if err := projectSavedDiaries(saved, w); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Document.Objects[0].Inline[0].Record, loaded.Document.Objects[0].Inline[0].Record) {
		t.Fatal("local Player Diary changed across LOAD then SAVE")
	}
}

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
