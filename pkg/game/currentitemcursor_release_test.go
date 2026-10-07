package game

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type currentCursorProof struct {
	Runtime uint32
	Start   int
	Before  []currentItemOwner
	After   [3][]currentItemOwner
}

func legacyCurrentItemCursorContinuation(t *testing.T) {
	if input := os.Getenv("AGAINROM_CURRENT_CURSOR_INPUT"); input != "" {
		raw, err := os.ReadFile(input + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var proof currentCursorProof
		if err = json.Unmarshal(raw, &proof); err != nil {
			t.Fatal(err)
		}
		f := loadAreaContinuation(t, input)
		if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.Before) {
			t.Fatal("initial cold values", firstObjectDifference(proof.Before, got))
		}
		for i, code := range []uint16{0x777, 0x778, 0x779} {
			if i < proof.Start {
				continue
			}
			currentCursorAcquire(t, f, proof.Runtime, code)
			if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.After[i]) {
				t.Errorf("next acquisition %d differs: %s", i+1, firstObjectDifference(proof.After[i], got))
			}
			if i == 0 {
				second := saveCorpseMission(t, f, t.TempDir())
				proof.Start, proof.Before = 1, proof.After[0]
				data, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(second+".json", data, 0600); err != nil {
					t.Fatal(err)
				}
				emitSpellWitness(t, second, "finite-append-cursor-second", data)
				runSpellWitnessChild(t, second, "AGAINROM_CURRENT_CURSOR_INPUT")
				return
			}
		}
		return
	}
	if os.Getenv("AGAINROM_ASSETS") == "" {
		return
	}
	f, runtime := currentWideCursorSource(t, 7)
	entity := currentCursorActor(t, f, runtime)
	pack, _ := f.live.world.CarriedStacks(entity.ID)
	wide := 0
	for _, item := range pack {
		if item.Count == 65536 {
			wide++
		}
	}
	if wide != 2 {
		t.Fatalf("need two actual wide native stacks: %+v", pack)
	}
	load := entity.CurrentActorLoad()
	if len(pack) != 2 || load.Inventory.InsertIndex != 7 {
		t.Fatalf("unexpected native fixture slots/cursor: %d/%d", len(pack), load.Inventory.InsertIndex)
	}
	t.Logf("before SAVE: %d native slots, cursor %d, two wide stacks", len(pack), load.Inventory.InsertIndex)
	beforeHash := f.live.world.Hash()
	seed := saveCorpseMission(t, f, t.TempDir())
	if beforeHash != f.live.world.Hash() {
		t.Fatal("SAVE mutated current World")
	}
	proof := currentCursorProof{Runtime: runtime, Before: currentObjectOwners(t, f)}
	for i, code := range []uint16{0x777, 0x778, 0x779} {
		currentCursorAcquire(t, f, runtime, code)
		proof.After[i] = currentObjectOwners(t, f)
	}
	data, _ := json.MarshalIndent(proof, "", "  ")
	if err := os.WriteFile(seed+".json", data, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, seed, "finite-append-cursor", data)
	runSpellWitnessChild(t, seed, "AGAINROM_CURRENT_CURSOR_INPUT")
}
