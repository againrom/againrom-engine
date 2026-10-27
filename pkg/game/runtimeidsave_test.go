package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// currentRuntimeSavedWords maps each bound actor to the runtime id its written
// record holds.
func currentRuntimeSavedWords(t *testing.T, raw []byte) map[sim.EntityID]uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal(err)
	}
	out := map[sim.EntityID]uint32{}
	for _, b := range actions.Bindings {
		if b.Structure || b.Missing {
			continue
		}
		wire, err := savedStructureValue(&doc.Objects[b.Object-1], "RuntimeID")
		if err != nil {
			t.Fatal(err)
		}
		out[b.ID] = wire
	}
	return out
}

// LOAD leaves the allocator as the file's own replay gives it (SAV-1371): the
// fixture's Sack constructor takes a bit no object carries, so the lowest
// clear bit is the replay's, not the owners'. Its SAVE loads to the same bits.
func TestCurrentRuntimeLoadReplaysTheSavedIDs(t *testing.T) {
	f, _, doc := sackObjectsOpen(t, nil)
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	records, err := runtimeIDFileRecords(mustOpenSAV(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	marks, next := runtimeIDFileReplay(records)
	w := f.live.world
	if got := w.RuntimeIDMarks(); !slices.Equal(got, marks) || w.NextRuntimeID() != next {
		t.Fatalf("LOAD marks %v next %d, the file's replay %v next %d", got, w.NextRuntimeID(), marks, next)
	}
	owned := map[uint16]bool{0: true}
	for _, r := range records {
		if runtimeIDOwnerClasses[r.class] {
			owned[uint16(r.id)] = true
		}
	}
	if len(owned) == len(marks) {
		t.Fatal("the fixture's replay marks no bit beyond its owners")
	}
	cold, err := loadOriginalFront(cellStateFront(t), currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	if got := cold.live.world.RuntimeIDMarks(); !slices.Equal(got, marks) || cold.live.world.NextRuntimeID() != next {
		t.Fatalf("cold LOAD marks %v, want %v", got, marks)
	}
}

// The word a SAVE writes for an actor is the id its World object carries: a
// live change of that id moves the word, and ids the World repeats are
// written repeated, not renumbered (SAV-1371, DIV-2504).
func TestCurrentRuntimeSaveWritesTheWorldID(t *testing.T) {
	for _, test := range []struct {
		name   string
		loaded [3]uint32
		live   []sim.ActorRuntimeIDLoad
	}{
		{"repeated as loaded", [3]uint32{77, 77, 77}, nil},
		{"live change", [3]uint32{71, 72, 73}, []sim.ActorRuntimeIDLoad{{Entity: 1, Saved: 60123}}},
		{"live repeat", [3]uint32{71, 72, 73}, []sim.ActorRuntimeIDLoad{{Entity: 1, Saved: 73}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := currentRuntimeFront(t, test.loaded)
			f.live.world.RestoreActorRuntimeIDs(test.live)
			words := currentRuntimeSavedWords(t, currentRuntimeSave(t, f))
			if len(words) == 0 {
				t.Fatal("SAVE bound no actor")
			}
			for id, word := range words {
				e, live := f.live.world.Entity(id)
				if !live || word != e.SourceBinding.RuntimeID {
					t.Fatalf("actor %d written %d, World %d", id, word, e.SourceBinding.RuntimeID)
				}
			}
			for _, change := range test.live {
				if words[change.Entity] != change.Saved {
					t.Fatalf("actor %d written %d after the World took %d", change.Entity, words[change.Entity], change.Saved)
				}
			}
		})
	}
}
