package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentBuildingPublicationAndActorOffMapStayIndependent(t *testing.T) {
	doc, world, sources := structureProjectionFixture(t)
	live := world.Structures()
	_, cells, _ := world.SavedStructures()
	actor := world.Entities()[0]
	actorObject := -1
	for i, r := range doc.Objects {
		if r.Class == "Unit" || r.Class == "Human" {
			actorObject = i
		}
	}
	if actorObject < 0 {
		t.Fatal("fixture has no exact actor")
	}
	for _, mask := range []uint16{0x0100, 0} {
		sources[0].Token18 = mask
		if err := world.ImportOriginalStructures(live, sources, cells, make([]byte, 32*32)); err != nil {
			t.Fatal(err)
		}
		if err := projectSavedStructures(&doc, world); err != nil {
			t.Fatal(err)
		}
		var building sav.DocumentRecordData
		for _, r := range doc.Objects {
			if r.Class == "Building" && actorProjectionValue(t, r, "Identity") == sources[0].SourceKey {
				building = r
			}
		}
		if building.Class == "" || actorProjectionValue(t, building, "T18") != uint32(mask) {
			t.Fatal("literal current publication mask did not reach exact Building", mask)
		}
		for _, off := range []bool{false, true, false} {
			actor.OffMap = off
			next, err := savedActorValueRecord(doc.Objects[actorObject], actor, false)
			if err != nil {
				t.Fatal(err)
			}
			doc.Objects[actorObject] = next
			flags := actorProjectionValue(t, next, "U4C")
			if (flags&sav.ActorOffMapFlag != 0) != off {
				t.Fatal("actor OffMap did not change its independent bit", flags, off)
			}
			if err := projectSavedStructures(&doc, world); err != nil {
				t.Fatal(err)
			}
			for _, r := range doc.Objects {
				if r.Class == "Building" && actorProjectionValue(t, r, "Identity") == sources[0].SourceKey && actorProjectionValue(t, r, "T18") != uint32(mask) {
					t.Fatal("actor OffMap changed Building publication", off, mask)
				}
			}
		}
	}
	if _, _, present := world.SavedStructures(); !present || len(live) != len(world.Structures()) || actor.Owner != sim.SelfSlot {
		t.Fatal("publication control changed structure population or actor owner")
	}
}
