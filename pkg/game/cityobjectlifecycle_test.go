package game

import (
	"reflect"
	"testing"

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
	town := NewTown(Campaign{})
	f := &FrontEnd{CampaignSession: CampaignSession{Town: town,
		Carried: []mapload.PartyMember{{ID: "live"}}}}
	for _, bad := range []*cityObjectTopology{
		{Version: 0, NextID: 1},
		{Version: cityObjectTopologyVersion, NextID: 2, Items: []cityItemTopology{{ID: 1}, {ID: 1}}},
	} {
		before := mapload.CloneParty(f.Carried)
		if _, err := f.prepareRestore(Snapshot{CityObjects: bad}); err == nil {
			t.Fatal("malformed topology admitted")
		}
		if f.Town != town || !reflect.DeepEqual(f.Carried, before) {
			t.Fatal("rejected candidate changed the live session")
		}
	}
}
