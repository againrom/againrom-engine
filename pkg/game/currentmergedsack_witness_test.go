package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentMergedSackFromLegacyFixture(t *testing.T) {
	f, _, _ := sackObjectsOpen1115(t, func(doc *sav.DocumentData) {
		r := doc.Objects[doc.World.Sacks[0]-1]
		r.Values = slices.Clone(r.Values)
		newGroupSetValue1115(t, &r, "Identity", 0x12345678)
		newGroupSetValue1115(t, &r, "S3C", 31)
		doc.Objects = append(doc.Objects, r)
		doc.World.Sacks = append(doc.World.Sacks, uint16(len(doc.Objects)))
	})
	if sacks := f.live.world.Sacks(); len(sacks) != 1 || sacks[0].Gold != 54 || sacks[0].ObjectID != 0 {
		t.Fatal("legacy fixture did not yield the measured current unbound Sack", sacks)
	}
	var comparison *FrontEnd
	for cycle := range 2 {
		before := f.live.world.Hash()
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal("current merged Snapshot", cycle, err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		after := f.live.world.Hash()
		if after != before {
			t.Fatal("current merged SAV mutated the live World")
		}
		if err != nil || len(raw) == 0 {
			t.Fatal("current merged SAV", cycle, err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.World.Sacks) != 1 {
			t.Fatal("current merged SAV retained more than one ordinary Sack root", doc.World.Sacks)
		}
		cold := cellStateFront(t)
		cold.Table, cold.Humans, cold.Campaign = f.Table, f.Humans, f.Campaign
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("current merged Sack witness").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("current merged cold LOAD", cycle, err)
		}
		if cold.live.world.Hash() != before {
			currentMenuWorldDiagnostics(t, f.live.world, cold.live.world)
			t.Fatalf("current merged full World %x -> %x", before, cold.live.world.Hash())
		}
		for tick := range 20 {
			f.live.tick()
			cold.live.tick()
			if f.live.world.Hash() != cold.live.world.Hash() {
				currentMenuWorldDiagnostics(t, f.live.world, cold.live.world)
				t.Fatal("current merged successor", cycle, tick)
			}
		}
		comparison, f = f, cold
	}
	for _, current := range []*FrontEnd{comparison, f} {
		e := newGroupActors1115(t, current.live.world)[newGroupA]
		purse := current.live.world.Purse(e.Owner)
		if err := current.live.world.TakeSack(e.ID, 15, 16); err != nil {
			t.Fatal("actual current merged pickup", err)
		}
		if current.live.world.Purse(e.Owner) != purse+54 || len(current.live.world.Sacks()) != 0 {
			t.Fatal("current merged pickup changed gold or left a Sack")
		}
	}
	if f.live.world.Hash() != comparison.live.world.Hash() || !reflect.DeepEqual(f.live.world.SavedObjects(), comparison.live.world.SavedObjects()) {
		t.Fatal("current merged pickup continuation differs")
	}
}
