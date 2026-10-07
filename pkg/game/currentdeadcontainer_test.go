package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentDeadContainerFixture(t *testing.T, present bool) *FrontEnd {
	t.Helper()
	runtime := uint32(0)
	body := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(65536 - 10001), maxHP: 30, stage: 5, runtime: &runtime}
	if present {
		body.holdings = &holdingFixture{}
	}
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	f := currentRetainedRuntimeFront(t)
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{body}))
	return openCurrentRetainedRuntime(t, f, completeCurrentDeadDocument(t, f, raw))
}

func requireCurrentDeadContainer(t *testing.T, world *sim.World, present bool, tail [2]uint32) sim.OriginalDeadRecord {
	t.Helper()
	dead := world.OriginalDeadActors()
	if len(dead) != 1 || dead[0].Source.ContainerPresent != present || dead[0].Source.ContainerTail != tail {
		t.Fatalf("current dead container = %+v, want %t/%v", dead, present, tail)
	}
	return dead[0]
}

func requireWrittenDeadContainer(t *testing.T, raw []byte, identity uint32, present bool, tail [2]uint32) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) != 1 {
		t.Fatalf("written dead actors = %+v: %v", dead, err)
	}
	if dead[0].Identity != identity || dead[0].ContainerPresent != present || dead[0].ContainerTail != tail || dead[0].Carried != 0 {
		t.Fatalf("written dead container = %+v, want identity %#x present %t tail %v", dead[0], identity, present, tail)
	}
}

func TestCurrentDeadContainerOverridesRetainedDocumentAcrossSaves(t *testing.T) {
	for _, tc := range []struct {
		name      string
		present   bool
		tail      [2]uint32
		stale     [2]uint32
		staleItem bool
	}{
		{name: "absent", stale: [2]uint32{10000, 0}},
		{name: "absent with stale item", staleItem: true},
		{name: "present", present: true, tail: [2]uint32{10000, 0}, stale: [2]uint32{7, 9}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := currentDeadContainerFixture(t, tc.present)
			body := requireCurrentDeadContainer(t, f.live.world, tc.present, tc.tail)
			doc := f.live.mission.state.savedDocument.Document
			if len(doc.DeadActors) != 1 {
				t.Fatalf("fixture dead roots = %d", len(doc.DeadActors))
			}
			var staleRefs []uint16
			if tc.staleItem {
				item := mustNewRecord("Item")
				savedObjectSetValue(&item, "Identity", 0x7f000001)
				doc.Objects = append(doc.Objects, item)
				staleRefs = []uint16{uint16(len(doc.Objects))}
			}
			retained := &doc.Objects[doc.DeadActors[0]-1]
			savedObjectSetValue(retained, "HasInventory", 1)
			savedObjectSetValue(retained, "Inventory1C", tc.stale[0])
			savedObjectSetValue(retained, "Inventory20", tc.stale[1])
			savedObjectSetRefs(retained, "Inventory", staleRefs, true)
			if _, err := sav.EncodeDocumentData(*doc); err != nil {
				t.Fatal("stale retained document is not a valid SAV", err)
			}
			requireCurrentDeadContainer(t, f.live.world, tc.present, tc.tail)
			for cycle := 0; cycle < 2; cycle++ {
				raw := currentRuntimeSave(t, f)
				requireWrittenDeadContainer(t, raw, body.Source.Identity, tc.present, tc.tail)
				if tc.staleItem {
					written, err := sav.DecodeDocumentData(raw)
					if err != nil {
						t.Fatal(err)
					}
					for _, object := range written.Objects {
						if key, err := savedStructureValue(&object, "Identity"); err == nil && key == 0x7f000001 {
							t.Fatal("stale Item survived the absent container")
						}
					}
				}
				f = openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				requireCurrentDeadContainer(t, f.live.world, tc.present, tc.tail)
			}
		})
	}
}
