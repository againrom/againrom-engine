package sim

import (
	"bytes"
	"testing"
)

// A body's loaded order words are World state: they survive the byte form,
// become its dead record's References when it is retired, and end when the
// actor lives again.
func TestBodyReferencesSurviveBinaryAndBecomeTheDeadRecords(t *testing.T) {
	words := [3]uint32{0x2000, 0x3000, 0x5000}
	newWorld := func() *World {
		return mustWorld(t, 3, Bounds{Width: 8, Height: 8}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20},
			{ID: 2, X: 2, Y: 1, HP: -20, MaxHP: 20, Decay: DecayBones},
		})
	}
	w := newWorld()
	before := w.Hash()
	if err := w.RestoreBodyReferences([]BodyReferenceRecord{{ID: 1, References: words}}); err == nil || w.Hash() != before {
		t.Fatal("a living actor took body references")
	}
	if err := w.RestoreBodyReferences([]BodyReferenceRecord{{ID: 2, References: words}}); err != nil {
		t.Fatal(err)
	}
	if w.Hash() == before {
		t.Fatal("body references are not hashed World state")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != bodyReferenceFormVersion || CheckSaveForm(form) != nil {
		t.Fatal("the body reference form is refused at the save boundary")
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	again, err := cold.MarshalBinary()
	if err != nil || !bytes.Equal(again, form) || cold.entities[1].BodyReferences != words {
		t.Fatal("body references did not survive the byte form", err)
	}

	// Retirement keeps a source-backed body; the binding alone selects that path.
	source := SourceBinding{Class: 1, ArchiveIndex: 4, Identity: 0x1000}
	cold.entities[1].SourceBinding = source
	cold.retireOriginalDead(2)
	dead := cold.OriginalDeadActors()
	if want := [5]uint32{0x2000, 0x3000, 0, 0x5000}; len(dead) != 1 || dead[0].Source.References != want {
		t.Fatalf("the retired body's dead record holds %+v, want References %x", dead, want)
	}
	plain := newWorld()
	plain.entities[1].SourceBinding = source
	plain.retireOriginalDead(2)
	if got := plain.OriginalDeadActors(); len(got) != 1 || got[0].Source.References != ([5]uint32{}) {
		t.Fatalf("a body without loaded words retired with %+v", got)
	}

	raised := newWorld()
	if err := raised.RestoreBodyReferences([]BodyReferenceRecord{{ID: 2, References: words}}); err != nil {
		t.Fatal(err)
	}
	raised.entities[1].HP, raised.entities[1].Decay = 5, DecayNone
	raised.releaseBodyReferences()
	if raised.entities[1].BodyReferences != ([3]uint32{}) {
		t.Fatal("a body that lives again kept its loaded words")
	}
}
