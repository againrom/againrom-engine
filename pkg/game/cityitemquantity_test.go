package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"testing"
)

func TestCurrentCitySharedWornQuantitySurvivesTwoSAVLoads(t *testing.T) {
	for _, pack := range []bool{true, false} {
		name := "worn-only"
		if pack {
			name = "pack-and-worn"
		}
		t.Run(name, func(t *testing.T) {
			raw, newFront := cityProjectionSource(t)
			f := cityProjectionLoad(t, raw, newFront)
			id := f.Town.cityObjects.Roots[0].Worn[0]
			item := mapload.MemberItemEquipment(f.Carried[0], f.Table)[0]
			screen := f.bindTown(&townScreen{})
			if err := screen.cityShopSetNode(id, item, 3, false); err != nil {
				t.Fatal(err)
			}
			if !pack {
				for i, member := range f.Carried {
					root, err := cityMutationParty(f.Town.cityObjects, member.ID)
					if err != nil {
						t.Fatal(err)
					}
					var stocks []sim.ItemStack
					var ids []sim.SavedObjectID
					for at, stack := range cityMemberStacks(member, f.Table) {
						if root.Pack[at] != id {
							stocks, ids = append(stocks, stack), append(ids, root.Pack[at])
						}
					}
					if err := screen.cityShopSetPack(i, stocks, false); err != nil {
						t.Fatal(err)
					}
					root.Pack = ids
				}
				if err := screen.cityShopSetNode(id, item, 3, false); err != nil {
					t.Fatal(err)
				}
			}
			for cycle := 0; cycle < 2; cycle++ {
				quantity, err := (f.bindTown(&townScreen{})).cityShopNodeCount(id)
				if err != nil || quantity != 3 {
					t.Fatal(cycle, quantity, err)
				}
				raw = cityProjectionSave(t, f)
				doc, _, indices := cityProjectionWire(t, raw)
				count, err := savedStructureValue(&doc.Objects[indices[id]-1], "F42")
				if err != nil || count != 3 {
					t.Fatal("ordinary quantity", count, err)
				}
				f = cityProjectionLoad(t, raw, newFront)
				for _, root := range f.Town.cityObjects.Roots {
					if root.Worn[0] != id {
						t.Fatal("Worn alias identity lost", root)
					}
				}
			}
			// An ordinary edit changes the sole quantity; the city graph does
			// not replay the preceding Worn-only amount over that value.
			doc, _, indices := cityProjectionWire(t, raw)
			savedObjectSetValue(&doc.Objects[indices[id]-1], "F42", 5)
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			f = cityProjectionLoad(t, raw, newFront)
			quantity, err := (f.bindTown(&townScreen{})).cityShopNodeCount(id)
			if err != nil || quantity != 5 {
				t.Fatal("ordinary Count edit was lost", quantity, err)
			}
		})
	}
}
