package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCityTopologySnapshotAndRestoreOwnTheirGraph(t *testing.T) {
	g := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 4,
		Items:   []cityItemTopology{{ID: 1, Effects: []sim.SavedObjectID{2}, Spell: 3}},
		Effects: []sim.SavedObjectID{2}, Spells: []sim.SavedObjectID{3},
		Roots: []cityPartyObjectRoots{{PartyID: []byte("hero"), Pack: []sim.SavedObjectID{1}}},
		Books: []cityBookTopology{{PartyID: []byte("hero"), Slots: [28]sim.SavedObjectID{3}}}}
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	original := g.Clone()
	town := NewTown(Campaign{})
	town.cityObjects = g
	var snapshot Snapshot
	snapshotTown(town, &snapshot)
	restored := restoreTown(Campaign{}, snapshot)
	if !reflect.DeepEqual(restored.cityObjects, original) {
		t.Fatal("city topology lost through Snapshot")
	}
	snapshot.CityObjects.Roots[0].Pack[0] = 0
	snapshot.CityObjects.Items[0].Effects[0] = 0
	snapshot.CityObjects.Books[0].PartyID[0] = 'x'
	if !reflect.DeepEqual(restored.cityObjects, original) || !reflect.DeepEqual(town.cityObjects, original) {
		t.Fatal("snapshot shared mutable topology with a city")
	}
	restored.cityObjects.Roots[0].PartyID[0] = 'y'
	if !reflect.DeepEqual(town.cityObjects, original) {
		t.Fatal("restored candidate shared live topology")
	}
	if restoreTown(Campaign{}, Snapshot{}).cityObjects != nil {
		t.Fatal("legacy empty snapshot invented city object identities")
	}
}

func TestCityTopologyInvalidRestoreKeepsLiveSession(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	raw = cityProjectionSave(t, f)
	town := f.Town
	for _, duplicate := range []bool{false, true} {
		doc, actions, _ := cityProjectionWire(t, raw)
		if duplicate {
			actions.Objects[1].ID = actions.Objects[0].ID
		} else {
			actions.Inventory.NextID = actions.Objects[0].ID
		}
		leaf, err := json.Marshal(actions)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		bad, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		before := mapload.CloneParty(f.Carried)
		graph := town.cityObjects.Clone()
		if _, _, err := f.RestoreOriginal(bad); err == nil {
			t.Fatal("malformed topology admitted")
		}
		if f.Town != town || !reflect.DeepEqual(f.Carried, before) || !reflect.DeepEqual(town.cityObjects, graph) {
			t.Fatal("rejected candidate changed the live session")
		}
	}
}
