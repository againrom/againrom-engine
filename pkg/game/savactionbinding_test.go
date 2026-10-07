package game

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentActionRestoreKeepsExecutableWorld(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	mapload.BindSourceDerive(world)
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc, Actors: []SnapshotSAVActor{binding}}
	if err := projectItems(state, world); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ActorManifest: &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: binding.EntityID, Constructed: true}}}}
	for _, text := range doc.Objects[binding.ObjectIndex-1].Texts {
		if text.Name == "Name" {
			snapshot.ActorManifest.Actors[0].Name = text.Value
		}
	}
	if err := projectCurrentActions(&doc, state, world, snapshot, nil); err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	// This transaction fixture has no separately imported item graph.
	a.Inventory, a.Ownership = nil, nil
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: world, savedDocument: state}
	if err := restoreOriginalActions(ms, nil); err != nil {
		t.Fatal(err)
	}
	if ms.World != world {
		t.Fatal("restore replaced the mission's live World pointer")
	}
	before := world.Tick()
	sim.Step(world, nil)
	if world.Tick() != before+1 {
		t.Fatal("restored current World lost its executable arithmetic binding")
	}
}

func TestCurrentActionBindingsKeepConstructorZeroKeys(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	doc.World.TerrainIdentity = 0xaabbaaff
	state := &SnapshotSAVDocument{Document: &doc, Actors: []SnapshotSAVActor{binding}}
	if err := projectItems(state, world); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{ActorManifest: &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: binding.EntityID, Constructed: true}}}}
	for _, text := range doc.Objects[binding.ObjectIndex-1].Texts {
		if text.Name == "Name" {
			snapshot.ActorManifest.Actors[0].Name = text.Value
		}
	}
	out := doc
	before, _ := json.Marshal(state)
	hash := world.Hash()
	if err := projectCurrentActions(&out, state, world, snapshot, nil); err != nil {
		t.Fatal("unminted current objects refused", err)
	}
	actions, err := readCurrentActions(&out)
	if err != nil || len(actions.Bindings) != 1 || actions.Bindings[0].Object != binding.ObjectIndex || !reflect.DeepEqual(actions.Actions, world.Actions()) {
		t.Fatal("exact archive identity or current values lost", actions, err)
	}
	if key, _ := savedStructureValue(&doc.Objects[binding.ObjectIndex-1], "Identity"); key != 0 {
		t.Fatal("zero-key control no longer discriminates")
	}
	after, _ := json.Marshal(state)
	if !reflect.DeepEqual(before, after) || hash != world.Hash() {
		t.Fatal("action projection changed source or World")
	}
	var retired []uint16
	for i, record := range out.Objects {
		if record.Class == "Item" {
			retired = append(retired, uint16(i+1))
		}
	}
	if len(retired) != 1 {
		t.Fatal("fixture must retire the one original Item absent from the current World")
	}
	ordered, _, err := sav.RetireDocumentData(out, retired)
	if err != nil {
		t.Fatal(err)
	}
	minted, err := sav.CompleteDocumentKeys(ordered)
	if err != nil {
		t.Fatal(err)
	}
	if key, _ := savedStructureValue(&minted.Objects[binding.ObjectIndex-1], "Identity"); key == 0 {
		t.Fatal("final writer did not mint constructor identity")
	}
	raw, err := sav.EncodeDocumentData(minted)
	if err != nil {
		t.Fatal(err)
	}
	cold, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readCurrentActions(&cold)
	if err != nil || !reflect.DeepEqual(got, actions) {
		t.Fatal("key remint changed archive action identity", err)
	}
	binding.EntityID = 71
	ms := &Mission{World: world, savedDocument: &SnapshotSAVDocument{Document: &cold, Actors: []SnapshotSAVActor{binding}}}
	if _, err := resolveCurrentActions(ms, got); err != nil || len(got.Actions.Actors) != 1 || got.Actions.Actors[0].Entity != 71 {
		t.Fatal("cold local entity remap lost exact object", err)
	}
	for _, kind := range []string{"absent object", "duplicate object", "duplicate label", "null object"} {
		t.Run(kind, func(t *testing.T) {
			data, _ := json.Marshal(actions)
			var invalid currentActionData
			if err := json.Unmarshal(data, &invalid); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "absent object":
				invalid.Bindings[0].Object = uint16(len(cold.Objects) + 1)
			case "duplicate object":
				b := invalid.Bindings[0]
				b.ID++
				invalid.Bindings = append(invalid.Bindings, b)
			case "duplicate label":
				invalid.Bindings = append(invalid.Bindings, invalid.Bindings[0])
			case "null object":
				invalid.Bindings[0].Object = 0
			}
			if _, err := resolveCurrentActions(ms, &invalid); err == nil {
				t.Fatal("cold LOAD admitted an unresolved or duplicate binding")
			}
		})
	}
	for _, kind := range []string{"duplicate key", "missing actor", "out of range"} {
		t.Run(kind, func(t *testing.T) {
			broken := out
			broken.Objects = slices.Clone(out.Objects)
			metadata := *state
			metadata.Actors = slices.Clone(state.Actors)
			switch kind {
			case "duplicate key":
				found := 0
				for i := range broken.Objects {
					if _, err := savedStructureValue(&broken.Objects[i], "Identity"); err == nil {
						broken.Objects[i].Values = slices.Clone(broken.Objects[i].Values)
						savedObjectSetValue(&broken.Objects[i], "Identity", 0x1234)
						found++
						if found == 2 {
							break
						}
					}
				}
				if found != 2 {
					t.Fatal("duplicate control lacks two objects")
				}
			case "missing actor":
				metadata.Actors = nil
			case "out of range":
				metadata.Actors[0].ObjectIndex = uint16(len(out.Objects) + 1)
			}
			before, _ := json.Marshal(broken)
			if err := projectCurrentActions(&broken, &metadata, world, snapshot, nil); err == nil {
				t.Fatal("accepted malformed current binding")
			}
			after, _ := json.Marshal(broken)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("failed projection changed output")
			}
		})
	}
}

func TestCurrentTypedIdentityIndexScopesAliasesBySAVClass(t *testing.T) {
	const key = 0x01020304 // shared by two record classes
	doc := &sav.DocumentData{Objects: []sav.DocumentRecordData{
		{Class: "Armor", Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}},
		{Class: "Effect", Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}},
	}}
	for _, tc := range []struct {
		class string
		want  uint16
	}{{"Armor", 1}, {"Effect", 2}} {
		got, err := currentTypedIdentityIndex(doc, tc.class, key)
		if err != nil || got != tc.want {
			t.Fatalf("typed identity %s/%#x = %d, %v; want object %d", tc.class, key, got, err, tc.want)
		}
	}
	if _, err := currentTypedIdentityIndex(doc, "", key); err == nil {
		t.Fatal("untyped action identity selected one of two record classes")
	}
	doc.Objects = append(doc.Objects, sav.DocumentRecordData{Class: "Armor", Values: []sav.DocumentValueData{{Name: "Identity", Value: key}}})
	if _, err := currentTypedIdentityIndex(doc, "Armor", key); err == nil {
		t.Fatal("same-class duplicate identity was accepted")
	}
}
