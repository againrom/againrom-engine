package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func bodyReferencesUnit(identity uint32, words map[string]uint32, u68 ...uint16) sav.DocumentRecordData {
	r := sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: identity}}, Raw: []sav.DocumentRawData{
		{Name: "U54", Bytes: make([]byte, 4)}, {Name: "U58", Bytes: make([]byte, 4)}, {Name: "U50", Bytes: make([]byte, 4)}, {Name: "U158", Bytes: make([]byte, 148)},
	}, RefSlots: []sav.DocumentRefsData{{Name: "U68", Objects: u68}}}
	for _, name := range []string{"Stage", "U5C", "U64", "U44", "U6C", "U40", "U48", "U136"} {
		r.Values = append(r.Values, sav.DocumentValueData{Name: name, Value: words[name]})
	}
	return r
}

func bodyReferencesWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{
		{ID: 1, X: 1, Y: 1, Class: 1, HP: 20, MaxHP: 20},
		{ID: 2, X: 3, Y: 1, Class: 1, HP: -20, MaxHP: 20, Decay: sim.DecayBones},
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func bodyReferencesOf(t *testing.T, w *sim.World, id sim.EntityID) [3]uint32 {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e.BodyReferences
		}
	}
	t.Fatalf("entity %d is absent", id)
	return [3]uint32{}
}

// A body's order words live in the World from LOAD on. The SAV writer writes
// them from the World, not from the loaded document, and a cold LOAD of the
// written document gives the World the same words. A body made by the engine
// holds none and is written with none.
func TestABodyKeepsItsLoadedOrderWordsThroughSave(t *testing.T) {
	const standing, body, cast = 0x700001, 0x700002, 0x700003
	loaded := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		bodyReferencesUnit(standing, map[string]uint32{"U5C": 0x2100}),
		bodyReferencesUnit(body, map[string]uint32{"U5C": 0x2000, "U64": 0x3000, "U44": 0x4000, "U40": 7}, 3),
		bodyReferencesUnit(cast, nil),
	}}, Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}}}
	w := bodyReferencesWorld(t)
	if err := importSavedActorActions(&Mission{World: w}, loaded); err != nil {
		t.Fatal(err)
	}
	want := [3]uint32{0x2000, 0x3000, cast}
	if got := bodyReferencesOf(t, w, 2); got != want {
		t.Fatalf("LOAD left the body References %x, want %x", got, want)
	}
	if got := bodyReferencesOf(t, w, 1); got != ([3]uint32{}) {
		t.Fatalf("a standing actor took body References %x", got)
	}

	// The writer reads the World: poisoning the loaded document after LOAD
	// changes nothing it writes.
	for i := range loaded.Document.Objects {
		for _, name := range []string{"U5C", "U64"} {
			savedObjectSetValue(&loaded.Document.Objects[i], name, 0xdead)
		}
	}
	written := &sav.DocumentData{Objects: []sav.DocumentRecordData{
		bodyReferencesUnit(cast, nil),
		bodyReferencesUnit(standing, map[string]uint32{"U5C": 0x2200}),
		bodyReferencesUnit(body, map[string]uint32{"U40": 9}),
	}}
	if before := bodyRecordReferences(written, &written.Objects[2]); before[0] != 0 || before[2] != 0 {
		t.Fatalf("loss control: the constructed body record already holds %x", before)
	}
	writeBodyReferences(written, &written.Objects[2], bodyReferencesOf(t, w, 2))
	if got := bodyRecordReferences(written, &written.Objects[2]); got != want {
		t.Fatalf("written body References %x, want %x", got, want)
	}
	if got, _ := savedStructureValue(&written.Objects[2], "U40"); got != 9 {
		t.Fatalf("the body words overwrote the current kill credit: U40 %x", got)
	}

	engine := &sav.DocumentData{Objects: []sav.DocumentRecordData{bodyReferencesUnit(body, map[string]uint32{"U5C": 0x2400}, 1)}}
	writeBodyReferences(engine, &engine.Objects[0], [3]uint32{})
	if got := bodyRecordReferences(engine, &engine.Objects[0]); got != ([3]uint32{}) {
		t.Fatalf("an engine-made body was written with words %x", got)
	}

	cold := bodyReferencesWorld(t)
	reloaded := &SnapshotSAVDocument{Document: written, Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 2}, {EntityID: 2, ObjectIndex: 3}}}
	if err := importSavedActorActions(&Mission{World: cold}, reloaded); err != nil {
		t.Fatal(err)
	}
	if got := bodyReferencesOf(t, cold, 2); got != want {
		t.Fatalf("cold LOAD left the body References %x, want %x", got, want)
	}
}

// A body made during play holds no loaded words. Its book selection and the
// cast it held while it lived are other World state; SAVE and a cold LOAD
// of our own SAV, twice, leave its words and the whole World unchanged.
func TestABodyMadeInPlayKeepsItsWordsThroughSaveAndLoad(t *testing.T) {
	f := castOrderFront(t)
	w := f.live.world
	sim.Step(w, []sim.Command{sim.Cast(1, 2, 6)})
	for tick := 0; tick < 16 && (len(w.Actions().Books) != 1 || w.Actions().Books[0].Phase != 1); tick++ {
		sim.Step(w, nil)
	}
	if e, _ := w.Entity(1); e.AdmittedBookSpell == 0 {
		t.Fatal("the caster admitted no book cast")
	}
	f.LiveKill(1)
	for tick := 0; ; tick++ {
		if e, _ := w.Entity(1); !e.Alive() && !e.Dying() {
			break
		}
		if tick == 4000 {
			t.Fatal("the caster did not become a body")
		}
		sim.Step(w, nil)
	}
	body, _ := w.Entity(1)
	if body.AdmittedBookSpell == 0 || body.BodyReferences != ([3]uint32{}) {
		t.Fatalf("the body holds book spell %d and words %x", body.AdmittedBookSpell, body.BodyReferences)
	}
	current := f
	for cycle := range 2 {
		raw, _, _ := saveCurrentEffect(t, current)
		cold := openCurrentEffectSave(t, current, raw)
		assertCurrentWorldEqual(t, w, cold.live.world, fmt.Sprintf("body made in play, cycle %d", cycle))
		current = cold
	}
}
