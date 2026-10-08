package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentRuntimeAllocatorReservesEveryRetainedRecord(t *testing.T) {
	weapon, actor, sack := mustNewRecord("Weapon"), mustNewRecord("Unit"), mustNewRecord("Sack")
	mustSetValue(&weapon, "RuntimeID", 1)
	mustSetValue(&actor, "RuntimeID", 2)
	mustSetValue(&sack, "RuntimeID", 1)
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{weapon, actor, sack}}
	if err := projectCurrentRuntimeIDs(&doc, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	itemID, _ := savedStructureValue(&doc.Objects[0], "RuntimeID")
	actorID, _ := savedStructureValue(&doc.Objects[1], "RuntimeID")
	sackID, _ := savedStructureValue(&doc.Objects[2], "RuntimeID")
	if itemID != 1 || actorID != 2 || sackID == 0 || sackID == itemID || sackID == actorID {
		t.Fatalf("retained item/actor/new Sack RuntimeIDs %d/%d/%d", itemID, actorID, sackID)
	}
}

func TestCurrentRuntimeAllocatorKeepsPlaceableIDsAcrossItemCollisions(t *testing.T) {
	weapon, human, unit := mustNewRecord("Weapon"), mustNewRecord("Human"), mustNewRecord("Unit")
	mustSetValue(&weapon, "RuntimeID", 1)
	mustSetValue(&human, "RuntimeID", 1)
	mustSetValue(&unit, "RuntimeID", 0)
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{weapon, human, unit}}
	if err := projectCurrentRuntimeIDs(&doc, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	itemID, _ := savedStructureValue(&doc.Objects[0], "RuntimeID")
	humanID, _ := savedStructureValue(&doc.Objects[1], "RuntimeID")
	unitID, _ := savedStructureValue(&doc.Objects[2], "RuntimeID")
	if itemID != 1 || humanID != 1 || unitID != 2 {
		t.Fatalf("retained Weapon/Human and new Unit RuntimeIDs %d/%d/%d, want 1/1/2", itemID, humanID, unitID)
	}
}
