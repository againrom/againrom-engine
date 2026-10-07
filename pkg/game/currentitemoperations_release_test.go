package game

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func legacyCurrentItemMutationSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_ITEM_MUTATION_SAV_INPUT"); path != "" {
		var proof currentObjectProof
		data, err := os.ReadFile(path + ".json")
		if err != nil || json.Unmarshal(data, &proof) != nil {
			t.Fatal("item mutation proof", err)
		}
		f := loadAreaContinuation(t, path)
		if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.Owners) {
			t.Fatal("cold current items", firstObjectDifference(proof.Owners, got))
		}
		mutateCurrentObject(t, f, &proof)
		if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.After) {
			t.Fatal("next current item mutation", firstObjectDifference(proof.After, got))
		}
		second := saveCorpseMission(t, f, t.TempDir())
		requireCurrentObjectSAV(t, second, proof)
		cold := loadAreaContinuation(t, second)
		if got := currentObjectOwners(t, cold); !reflect.DeepEqual(got, proof.After) {
			t.Fatal("second current items", firstObjectDifference(proof.After, got))
		}
		return
	}
	if os.Getenv("AGAINROM_ASSETS") == "" {
		return
	}
	for _, scenario := range []string{"merge-values", "wide-unbound-consumed"} {
		t.Run(scenario, func(t *testing.T) {
			f := spellWitnessSource(t)
			var owner sim.Entity
			for _, actor := range f.live.world.Entities() {
				if actor.Owner == sim.SelfSlot && actor.Alive() && actor.SourceBinding.RuntimeID != 0 {
					owner = actor
					break
				}
			}
			if owner.SourceBinding.RuntimeID == 0 {
				t.Fatal("current item owner missing")
			}
			if scenario == "wide-unbound-consumed" {
				f = wideCurrentItemFixture(t, f, owner.SourceBinding.RuntimeID)
				owner = world1170Entity(t, f, owner.SourceBinding.RuntimeID)
				const wideCode = uint16(0x1c1e)
				castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: owner.ID, HasUnit: true, Item: wideCode, HasItem: true}, func() bool { return true })
				castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: owner.ID, HasUnit: true, Item: 0x777, HasItem: true}, func() bool { return true })
				pack, _ := f.live.world.CarriedStacks(owner.ID)
				wide, unbound, potion := false, false, -1
				for i, item := range pack {
					wide = wide || item.Code == wideCode && item.Count == 65536
					unbound = unbound || item.Code == 0x777 && item.ObjectID == 0
					if item.Code == 0x0e07 && item.Kind == 3 && item.Count == 2 {
						potion = i
					}
				}
				if !wide || !unbound || potion < 0 {
					t.Fatal("discriminating wide/unbound/potion setup absent", wide, unbound, potion)
				}
				if !f.live.world.UseCarriedPotion(owner.ID, potion) {
					t.Fatal("current potion was not consumed")
				}
				weapon, ok := f.live.world.UnequipSource(owner.ID, 1, true)
				if !ok {
					t.Fatal("ordinary source unequip failed")
				}
				pack, _ = f.live.world.CarriedStacks(owner.ID)
				equipped := false
				for i, item := range pack {
					if item.ObjectID == weapon.ObjectID {
						equipped = f.live.world.EquipSourceCarried(owner.ID, i)
						break
					}
				}
				if !equipped {
					t.Fatal("ordinary source equip failed")
				}
			} else {
				f = loadAreaContinuation(t, saveCorpseMission(t, f, t.TempDir()))
				for range 16 {
					f.live.tick()
				}
				found := false
				for _, item := range f.live.world.SavedObjects().Items {
					if item.Coverage.Unknown&sim.SavedUnknownMergePolicy != 0 && item.Value.Code == 3614 && item.Value.Price == 0 && item.Value.Weight == 0 {
						for _, actor := range f.live.world.Entities() {
							if actor.ID == item.Owner.Entity {
								owner = actor
							}
						}
						found = true
					}
				}
				if !found {
					t.Fatal("configured tick16 merge-policy witness missing")
				}
			}
			proof := currentObjectProof{Owners: currentObjectOwners(t, f), NextOwner: owner.SourceBinding.RuntimeID}
			pack, _ := f.live.world.CarriedStacks(owner.ID)
			for _, item := range pack {
				if scenario == "merge-values" && item.Code == 3614 && item.Price == 0 && item.Weight == 0 || scenario == "wide-unbound-consumed" && item.Count == 65536 {
					proof.NextCode, proof.NextCount, proof.NextPrice, proof.NextWeight = item.Code, min(item.Count, uint32(65535)), item.Price, item.Weight
				}
			}
			if proof.NextCode == 0 {
				t.Fatal("required next item missing", pack, owner.ID, owner.SourceBinding.RuntimeID)
			}
			before := f.live.world.Hash()
			seed := saveCorpseMission(t, f, t.TempDir())
			if f.live.world.Hash() != before {
				t.Fatal("SAVE changed native object flags or registry")
			}
			mutateCurrentObject(t, f, &proof)
			proof.After = currentObjectOwners(t, f)
			data, _ := json.MarshalIndent(proof, "", "  ")
			if err := os.WriteFile(seed+".json", data, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, seed, "items-"+scenario, data)
			runSpellWitnessChild(t, seed, "AGAINROM_ITEM_MUTATION_SAV_INPUT")
		})
	}
}

// This new boundary fixture retains the installed mission and appends explicit
// current operands. It is a configured engine witness, not an original save.
func wideCurrentItemFixture(t *testing.T, f *FrontEnd, runtime uint32) *FrontEnd {
	t.Helper()
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actorIndex := -1
	for i, record := range doc.Objects {
		if record.Class == "Human" {
			if id, _ := savedStructureValue(&record, "RuntimeID"); id == runtime {
				actorIndex = i
				break
			}
		}
	}
	if actorIndex < 0 {
		t.Fatal("boundary owner missing")
	}
	keys, err := sav.ReserveDocumentKeys(doc, 16)
	if err != nil {
		t.Fatal(err)
	}
	b := generatedDocumentBuilder{doc: doc, reservedKeys: keys, table: f.Table}
	owner, _ := savedStructureValue(&doc.Objects[actorIndex], "Reference")
	wideValue := sim.ItemInstance{Code: 0x1c1e, WeightPresent: true}
	wide, err := b.item(wideValue, 65535, owner)
	if err != nil {
		t.Fatal(err)
	}
	potionValue := mapload.ItemInstanceFromCode(0x0e07, f.Table)
	potionValue.WeightPresent = true
	for _, weight := range f.live.world.ItemWeights() {
		if weight.Code == potionValue.Code {
			potionValue.Weight = int16(weight.Weight)
		}
	}
	potion, err := b.item(potionValue, 2, owner)
	if err != nil {
		t.Fatal(err)
	}
	refs, _ := savedObjectRefs(&b.doc.Objects[actorIndex], "Inventory")
	savedObjectSetRefs(&b.doc.Objects[actorIndex], "Inventory", append(refs, wide, potion), true)
	savedObjectSetValue(&b.doc.Objects[actorIndex], "HasInventory", 1)
	doc, _, err = sav.ReindexDocumentData(b.doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/configured-word-boundary.sav"
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return loadAreaContinuation(t, path)
}
