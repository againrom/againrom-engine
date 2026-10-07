package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func savedTierRecord(mapID, typeID, face uint32) sav.DocumentRecordData {
	return sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{
		{Name: "T08", Value: mapID}, {Name: "T0E", Value: typeID}, {Name: "U4B", Value: face},
	}}
}

func TestSavedActorTiersFollowRestoredIDs(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{
		{UnitID: 42, ClassID: 70},
		{UnitID: 1098, ClassID: 70},
		{UnitID: 903, ClassID: 73},
	}}
	placed := map[sim.EntityID]int{0: 2, 1: 4}
	entities := []sim.Entity{
		{ID: 1, MapUnitID: 903, Class: 73, TypeID: 73},
		{ID: 9, MapUnitID: 1098, Class: 70, TypeID: 70},
		{ID: 10, MapUnitID: 42, Class: 70, TypeID: 70},
	}
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		savedTierRecord(1098, 70, 4),
	}}, Actors: []SnapshotSAVActor{{EntityID: 9, ObjectIndex: 1}}}
	tiers := savedActorTiers(m, placed, entities, state)
	if _, stale := tiers[1]; stale {
		t.Fatal("removed ALM index coloured a different restored actor")
	}
	if got := markOf(tierArtClass(4).TierFrames(tiers[9])[0]); got != 4 {
		t.Errorf("restored bat drew tier mark %d, want 4", got)
	}
	if got := tiers[10]; got != 2 {
		t.Errorf("unique authored placement tier = %d, want 2", got)
	}
	if placed[1] != 4 || len(placed) != 2 {
		t.Fatal("rekey changed the placement lookup")
	}
}

func TestSavedActorTiersRejectAmbiguousOrMismatchedSources(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{
		{UnitID: 50, ClassID: 70}, {UnitID: 50, ClassID: 70},
		{UnitID: 60, ClassID: 70},
	}}
	placed := map[sim.EntityID]int{0: 1, 1: 2, 2: 3}
	entities := []sim.Entity{
		{ID: 7, MapUnitID: 50, TypeID: 70},
		{ID: 8, MapUnitID: 60, TypeID: 71},
		{ID: 9, MapUnitID: 60, TypeID: 70},
	}
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		savedTierRecord(60, 71, 4), // type does not match actor 9
	}}, Actors: []SnapshotSAVActor{{EntityID: 9, ObjectIndex: 1}}}
	tiers := savedActorTiers(m, placed, entities, state)
	if len(tiers) != 1 || tiers[9] != 3 {
		t.Fatalf("tiers=%v, want only actor 9 from the unique matching ALM placement", tiers)
	}
}
