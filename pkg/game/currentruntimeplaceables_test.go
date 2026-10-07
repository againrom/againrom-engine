package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentRuntimeIDsReserveRetainedPlaceablesBeforeMinting(t *testing.T) {
	unit := mustNewRecord("Unit")
	mustSetValue(&unit, "RuntimeID", 0)
	building := mustNewRecord("Building")
	mustSetValue(&building, "RuntimeID", 1)
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{unit, building}}
	if err := projectCurrentRuntimeIDs(&doc, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	unitID, err := savedStructureValue(&doc.Objects[0], "RuntimeID")
	if err != nil {
		t.Fatal(err)
	}
	buildingID, err := savedStructureValue(&doc.Objects[1], "RuntimeID")
	if err != nil {
		t.Fatal(err)
	}
	if unitID != 2 || buildingID != 1 {
		t.Fatalf("new Unit took retained Building coordinate: Unit=%d Building=%d", unitID, buildingID)
	}
}
