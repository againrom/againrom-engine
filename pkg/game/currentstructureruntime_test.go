package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentStructureRuntimeCoordinatesSurviveRemintAndOrdinaryEdits(t *testing.T) {
	for _, ids := range [][2]uint32{{73, 91}, {73, 73}, {0, 0}, {0xabcdef77, 0xabcdef77}} {
		f, snapshot, world, live, source := currentBuildingRootFixture(t, true)
		_, cells, _ := world.SavedStructures()
		for i := range source {
			source[i].RuntimeID = ids[i]
		}
		if err := world.ImportOriginalStructures(live, source, cells, make([]byte, int(world.Bounds().Width*world.Bounds().Height))); err != nil {
			t.Fatal(err)
		}
		snapshot.World, _ = world.MarshalBinary()
		raw, err := f.ExportCurrentSave(snapshot, "current structure runtime coordinates")
		if err != nil {
			t.Fatal(err)
		}
		for _, edit := range []bool{false, true} {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			want := ids
			if edit {
				want[0] = 60123
				for _, object := range doc.World.Buildings {
					r := &doc.Objects[object-1]
					if actorProjectionValue(t, *r, "Identity") == source[0].SourceKey {
						mustSetValue(r, "RuntimeID", want[0])
					}
				}
			}
			doc, _, err = sav.ReindexDocumentData(doc)
			if err == nil {
				raw, err = sav.EncodeDocumentData(doc)
			}
			if err != nil {
				t.Fatal(err)
			}
			cold := coldCurrentScript(t, f, raw)
			for cycle := 0; cycle < 2; cycle++ {
				rows, _, present := cold.live.world.SavedStructures()
				if !present || len(rows) != len(source) {
					t.Fatal("current structure roster missing")
				}
				for i, row := range rows {
					if row.SourceKey != source[i].SourceKey || row.RuntimeID != want[i] {
						t.Fatalf("exact structure runtime coordinate changed: key%x value%d want%d, inputs%v edit%t cycle%d", row.SourceKey, row.RuntimeID, want[i], ids, edit, cycle)
					}
				}
				cold.live.tick()
				next, _, err := cold.Snapshot(true)
				if err == nil {
					raw, err = cold.ExportCurrentSave(next, "next structure runtime coordinate")
				}
				if err != nil {
					t.Fatal(err)
				}
				cold = coldCurrentScript(t, cold, raw)
			}
		}
	}
}
