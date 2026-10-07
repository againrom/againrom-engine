package sim

import (
	"reflect"
	"testing"
)

func TestCurrentTerminalActorRetainsTupleAcrossCompactionAndColdLoad(t *testing.T) {
	companion := spEnt(3, 10, 11)
	w := spWorld(t, 1177, nil, spEnt(1, 1, 1), companion)
	w.entities[1].HP, w.entities[1].Decay = decayGoneHP-1, decayLast
	Step(w, nil)
	want := CurrentTerminalActor{ID: 3, Cell: uint16(11)<<8 | 10, HP: decayGoneHP - 1, Stage: uint8(decayLast)}
	if indexOfEntity(w.entities, want.ID) >= 0 {
		t.Fatal("fixture did not compact the companion")
	}
	if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) {
		t.Fatalf("current terminal actors = %+v, want %+v", got, want)
	}
	if len(w.OriginalDeadActors()) != 0 {
		t.Fatal("native terminal actor acquired original archive provenance")
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := cold.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) {
		t.Fatalf("cold current terminal actors = %+v, want %+v", got, want)
	}
	refs := cold.ActorIdentityReferences()
	ids := make(map[EntityID]EntityID, len(refs))
	for _, id := range refs {
		ids[id] = id + 100
	}
	if err := cold.RestoreActorIdentities(ids, nil); err != nil {
		t.Fatal(err)
	}
	want.ID = 103
	if got := cold.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) {
		t.Fatalf("remapped current terminal actors = %+v, want %+v", got, want)
	}
	if next, ok := cold.NextEntityID(); !ok || next <= want.ID {
		t.Fatalf("NextEntityID = %d,%t after terminal remap", next, ok)
	}

	values := map[EntityID]ActorValues{}
	for _, e := range cold.Entities() {
		values[e.ID] = e.Values()
	}
	terminal := want
	values[terminal.ID] = ActorValues{CurrentTerminal: &terminal}
	before := cold.Hash()
	bad := terminal
	bad.HP++
	values[terminal.ID] = ActorValues{CurrentTerminal: &bad}
	if err := cold.restoreActorValues(values); err == nil || cold.Hash() != before {
		t.Fatal("mismatched terminal tuple was accepted or changed the world")
	}
	values[terminal.ID] = ActorValues{CurrentTerminal: &terminal}
	if err := cold.restoreActorValues(values); err != nil {
		t.Fatal(err)
	}
}

func TestImportCurrentTerminalActorRemovesOnlyExactDeadNativeBody(t *testing.T) {
	w := spWorld(t, 1177, nil, spEnt(1, 1, 1), spEnt(3, 10, 11))
	e := &w.entities[1]
	e.HP, e.Decay = decayGoneHP-1, decayLast
	// A loaded record can carry a reconstructed source binding though native.
	e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 8, Identity: 0x62000020}
	e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	want := CurrentTerminalActor{ID: e.ID, Cell: uint16(e.Y)<<8 | uint16(e.X), HP: e.HP, Stage: uint8(e.Decay)}
	if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{want}); err != nil {
		t.Fatal(err)
	}
	if indexOfEntity(w.entities, want.ID) >= 0 || !w.hasCurrentTerminalActor(want) {
		t.Fatal("exact terminal body was not converted into a retained native tuple")
	}
	if len(w.OriginalDeadActors()) != 0 {
		t.Fatal("current terminal tuple acquired reconstructed original-dead provenance")
	}
	before := w.Hash()
	badWorld := spWorld(t, 1177, nil, spEnt(1, 1, 1), spEnt(3, 10, 11))
	badWorld.entities[1].HP, badWorld.entities[1].Decay = decayGoneHP-2, decayLast
	if err := badWorld.ImportCurrentTerminalActors([]CurrentTerminalActor{want}); err == nil || badWorld.Hash() == before {
		t.Fatal("mismatched terminal body was accepted")
	}
}

func TestRestoreCurrentTerminalActorReplacesOnlyUnboundConstruction(t *testing.T) {
	w := spWorld(t, 1177, nil, spEnt(1, 1, 1), spEnt(3, 10, 11))
	want := CurrentTerminalActor{ID: 3, Cell: uint16(11)<<8 | 10, HP: decayGoneHP - 1, Stage: uint8(decayLast)}
	before := w.Hash()
	if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{want}); err == nil || w.Hash() != before {
		t.Fatal("strict terminal import accepted a mismatched constructor body")
	}
	if err := w.RestoreCurrentTerminalActors([]CurrentTerminalActor{want}); err != nil {
		t.Fatal(err)
	}
	if indexOfEntity(w.entities, want.ID) >= 0 || !w.hasCurrentTerminalActor(want) || len(w.OriginalDeadActors()) != 0 {
		t.Fatal("SAV terminal restore revived or misclassified the retired actor")
	}
	if len(w.entities) != 1 || w.entities[0].ID != 1 {
		t.Fatalf("terminal restore removed an unrelated actor: %+v", w.entities)
	}

	bound := spWorld(t, 1177, nil, spEnt(3, 10, 11))
	bound.entities[0].SourceBinding = SourceBinding{Class: 1, Identity: 8}
	boundBefore := bound.Hash()
	if err := bound.RestoreCurrentTerminalActors([]CurrentTerminalActor{want}); err == nil || bound.Hash() != boundBefore {
		t.Fatal("terminal restore removed a source-bound actor or changed a refused candidate")
	}
}
