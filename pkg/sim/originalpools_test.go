package sim

import (
	"reflect"
	"testing"
)

func poolImportWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(1, Bounds{Width: 20, Height: 20}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 5, Y: 5, MapUnitID: 71, HP: 20, MaxHP: 30, Mana: 10, MaxMana: 20,
			HealthRegenPeriod: 100, ManaRegenPeriod: 50},
		{ID: 2, X: 6, Y: 5, MapUnitID: 72, HP: 21, MaxHP: 31},
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestOriginalPools1094ChangesExactlyFourPersistedFields(t *testing.T) {
	w := poolImportWorld(t)
	// Nonzero regeneration remainders already belong to canonical state.
	w.entities[0].HealthHundredths, w.entities[0].ManaHundredths = 17, 19
	before := w.Entities()
	if err := w.ImportOriginalActorPools([]OriginalActorPools{{ID: 1, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}}); err != nil {
		t.Fatal(err)
	}
	want := before
	want[0].HP, want[0].MaxHP, want[0].Mana, want[0].MaxMana = 7, 31, 5, 23
	if !reflect.DeepEqual(w.Entities(), want) {
		t.Fatal("pool import changed fields outside its four-value contract")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil || back.Hash() != w.Hash() {
		t.Fatalf("native pool roundtrip failed: %v", err)
	}
	for range 64 {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("pool/regen continuation changed at tick %d", w.Tick())
		}
	}
}

func TestOriginalPools1094ValidatesWholeBatchBeforeMutation(t *testing.T) {
	good := OriginalActorPools{ID: 1, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}
	for _, bad := range []OriginalActorPools{
		{ID: 2, HP: 0, MaxHP: 10},
		{ID: 2, HP: -1, MaxHP: 10},
		{ID: 2, HP: 11, MaxHP: 10},
		{ID: 2, HP: 1, MaxHP: 0},
		{ID: 2, HP: 1, MaxHP: 10, Mana: -1, MaxMana: 10},
		{ID: 2, HP: 1, MaxHP: 10, Mana: 11, MaxMana: 10},
		{ID: 99, HP: 1, MaxHP: 10},
		good,
	} {
		w := poolImportWorld(t)
		before := w.Hash()
		if err := w.ImportOriginalActorPools([]OriginalActorPools{good, bad}); err == nil || w.Hash() != before {
			t.Fatalf("bad final entry %+v accepted/partly written: %v", bad, err)
		}
	}
	for _, offMap := range []bool{false, true} {
		w := poolImportWorld(t)
		if offMap {
			w.entities[1].OffMap = true
		} else {
			w.entities[1].HP = 0
		}
		before := w.Hash()
		err := w.ImportOriginalActorPools([]OriginalActorPools{good, {ID: 2, HP: 1, MaxHP: 10}})
		if offMap {
			if err != nil || !w.entities[1].OffMap || w.entities[1].HP != 1 || w.entities[0].HP != good.HP {
				t.Fatal("current detached pools were not imported", err)
			}
		} else if err == nil || w.Hash() != before {
			t.Fatalf("dead target accepted or first entry written: %v", err)
		}
	}
	var nilWorld *World
	if err := nilWorld.ImportOriginalActorPools(nil); err == nil {
		t.Fatal("nil world accepted pool import")
	}
}
