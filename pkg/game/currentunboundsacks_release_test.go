package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCurrentUnboundSackItemsKeepAbsenceAndOrdinaryEdits(t *testing.T) {
	for _, edited := range []bool{false, true} {
		f := releaseFront(t)
		if err := f.App("native ground values").OpenMission(f.NewGameOpener(20, ui.ChargenResult{Name: "Ground values", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})); err != nil {
			t.Fatal(err)
		}
		want := f.live.world.Sacks()
		if len(want) != 4 || want[0].ObjectID != 0 || len(want[0].ItemInstances) != 1 || want[0].ItemInstances[0].WeightPresent {
			t.Fatalf("native mission no longer has four unregistered Sack roots: %+v", want)
		}
		weights := f.live.world.ItemWeights()
		for cycle := 0; cycle < 2; cycle++ {
			before := f.live.world.Hash()
			constructedBefore := worldHashWithConstructedCurrentSessionHead1115(t, f.live.world)
			doc, a := currentRootSAVDocument(t, f)
			if edited && cycle == 0 {
				var item uint16
				for _, index := range doc.World.Sacks {
					token, _, _, err := savedSackRecord(&doc.Objects[index-1])
					if err != nil {
						t.Fatal(err)
					}
					if int32(token.Position[2]) == want[0].X && int32(token.Position[3]) == want[0].Y {
						refs, _ := savedObjectRefs(&doc.Objects[index-1], "Contents")
						item = refs[0]
					}
				}
				anchored := false
				for _, row := range a.Ownership {
					if row.Kind == 1 && row.Object == item {
						anchored = !row.WeightKnown && !row.EquipmentKnown && row.WeightAnchor != nil && row.EquipmentAnchor != nil
					}
				}
				if item == 0 || doc.Objects[item-1].Class != "Weapon" || !anchored {
					t.Fatal("ground Item is not anchored to its ordinary constructor")
				}
				r := &doc.Objects[item-1]
				savedObjectSetValue(r, "F4A", 123)
				savedObjectSetValue(r, "F42", 2)
				raw, err := savedObjectRaw(r, "W52", 24)
				if err != nil {
					t.Fatal(err)
				}
				raw[0] = 73
				mustSetRaw(r, "W52", raw)
			}
			f = loadCurrentRootSAV(t, doc, releaseFront)
			got := f.live.world.Sacks()
			if !reflect.DeepEqual(f.live.world.ItemWeights(), weights) {
				t.Fatal("ordinary instance edits changed the global future-item constructor table")
			}
			if edited {
				if len(got) != 4 || got[0].ObjectID != 0 || len(got[0].ItemInstances) != 2 || !reflect.DeepEqual(got[1:], want[1:]) {
					t.Fatal("ordinary Item count edit changed other ground roots")
				}
				for _, item := range got[0].ItemInstances {
					if item.ObjectID != 0 || item.Weight != 123 || !item.WeightPresent || item.SourceEquipment.Class != sim.SourceWeapon || item.SourceEquipment.Attack[0] != 73 {
						t.Fatal("unchanged absence policy swallowed ordinary Item values", item)
					}
				}
			} else if !reflect.DeepEqual(got, want) {
				t.Fatal("ordinary constructor replaced absent native Item fields")
			}
			if (!edited || cycle > 0) && f.live.world.Hash() != before && f.live.world.Hash() != constructedBefore {
				t.Fatal("unchanged second SAV cycle changed exact World.Hash")
			}
		}
	}
}
