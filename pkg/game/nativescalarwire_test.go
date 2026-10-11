package game

import (
	"bytes"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A known scalar whose file word an action writer replaces (U58, U6C: the
// idle actor's phase and countdown) comes back from cold LOAD as the World
// held it, while the file keeps the action's word. An edited file word wins.
func TestNativeScalarReplacedByActionWriterSurvivesColdLOAD(t *testing.T) {
	f, doc, object := legacyRawFixture(t)
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, f, raw)
	e, _ := cold.live.world.Entity(0)
	if !e.NativeBasis.ScalarIsKnown(sim.ScalarU6C) || !e.NativeBasis.ScalarIsKnown(sim.ScalarU58) || e.HasAttackTarget {
		t.Fatal("fixture actor lacks known U6C/U58 or is attacking", e.NativeBasis)
	}
	basis := e.NativeBasis
	basis.Scalars[sim.ScalarU6C], basis.Scalars[sim.ScalarU58] = 9, 3
	if err := cold.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	before := cold.live.world.Hash()
	written, saved, a := saveCurrentEffect(t, cold)
	r := &saved.Objects[object-1]
	for slot, want := range map[int]uint32{sim.ScalarU6C: 0, sim.ScalarU58: 0} {
		if got, err := readNativeScalar(r, slot); err != nil || got != want {
			t.Fatalf("action writer word %d: got %d want %d (%v)", slot, got, want, err)
		}
	}
	replaced := 0
	for _, wire := range a.NativeBasisWires {
		replaced += len(wire.Scalars)
	}
	if replaced != 2 {
		t.Fatalf("replaced scalar rows %d, want 2: %+v", replaced, a.NativeBasisWires)
	}
	again := nativeSubjectCold(t, f, written)
	got, _ := again.live.world.Entity(0)
	if got.NativeBasis.Scalars[sim.ScalarU6C] != 9 || got.NativeBasis.Scalars[sim.ScalarU58] != 3 || again.live.world.Hash() != before {
		t.Fatalf("cold LOAD basis U6C=%d U58=%d hash equal=%t", got.NativeBasis.Scalars[sim.ScalarU6C], got.NativeBasis.Scalars[sim.ScalarU58], again.live.world.Hash() == before)
	}

	t.Run("loss control: no replaced rows", func(t *testing.T) {
		lost := withNativeActions(t, written, func(m map[string]any) {
			for _, wire := range m["NativeBasisWires"].([]any) {
				delete(wire.(map[string]any), "Scalars")
			}
		})
		cold := nativeSubjectCold(t, f, lost)
		got, _ := cold.live.world.Entity(0)
		if got.NativeBasis.Scalars[sim.ScalarU6C] != 0 || cold.live.world.Hash() == before {
			t.Fatal("without the rows LOAD still restored the World's scalar", got.NativeBasis.Scalars[sim.ScalarU6C])
		}
	})
	t.Run("an edited file word wins", func(t *testing.T) {
		edited, err := sav.DecodeDocumentData(written)
		if err != nil {
			t.Fatal(err)
		}
		next, _, err := cloneSavedGroupFieldRecord(&edited.Objects[object-1])
		if err != nil {
			t.Fatal(err)
		}
		if err := savedActorSetValue(&next, "U6C", 5); err != nil {
			t.Fatal(err)
		}
		edited.Objects[object-1] = next
		raw, err := sav.EncodeDocumentData(edited)
		if err != nil {
			t.Fatal(err)
		}
		cold := nativeSubjectCold(t, f, raw)
		got, _ := cold.live.world.Entity(0)
		if got.NativeBasis.Scalars[sim.ScalarU6C] != 5 || got.NativeBasis.Scalars[sim.ScalarU58] != 3 {
			t.Fatal("edited U6C did not win or U58 lost", got.NativeBasis.Scalars)
		}
	})
}

func withNativeActions(t *testing.T, raw []byte, edit func(map[string]any)) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("native actions", present, err)
	}
	var m map[string]any
	d := json.NewDecoder(bytes.NewReader(leaf))
	d.UseNumber()
	if err := d.Decode(&m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	next, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, next); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
