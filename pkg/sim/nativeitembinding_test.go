package sim

import "testing"

func TestNativeItemBindingUsesRepresentedRowAndKeepsAbsentRowAtomically(t *testing.T) {
	for _, represented := range []bool{false, true} {
		for _, corrupt := range []string{"valid", "row", "Position", "service"} {
			name := "absent/" + corrupt
			if represented {
				name = "represented/" + corrupt
			}
			t.Run(name, func(t *testing.T) {
				item := PlainItem(0x0e01)
				item.Price, item.WeightPresent, item.Weight = 17, true, 7
				item.NativeRecord = &NativeItemRecord{Token: SavedObjectToken{Identity: 0x12345678, RuntimeID: 19, Reference: 23, T0C: 99}, F45: 73, F47: 91, F48: 0x9876}
				item.NativeRecord.Token.Position[0], item.NativeRecord.Token.Position[11] = 0xa5, 0x5a
				if represented {
					item.Code = 0x0106
					item.SourceEquipment = SourceEquipment{Class: SourceWeapon, DefinitionRow: 6}
					item.NativeRecord.Class = SourceWeapon
				}
				var worn [EquipSlots]ItemInstance
				worn[0] = item
				w, err := NewStockedWorld(17, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
					[]Entity{{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10}}, nil, Relations{}, nil, []Stock{{ID: 7, EquippedItems: worn}})
				if err != nil {
					t.Fatal(err)
				}
				value := StackItem(item, 1)
				value.ObjectID, value.NativeRecord = 1, nil
				token := item.NativeRecord.Token
				token.T1C = uint32(item.Price)
				if represented {
					token.T0C = 6
				}
				owner := SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 7, Slot: 1}
				r := &SavedObjects{Version: SavedObjectsVersion, NextID: 2, Items: []SavedItemObject{{ID: 1, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Value: value, Token: token, F45: 73, F47: 91, F48: 0x9876}}}
				if err := r.AddItemRoot(1, owner); err != nil {
					t.Fatal(err)
				}
				switch corrupt {
				case "row":
					r.Items[0].Token.T0C++
				case "Position":
					r.Items[0].Token.Position[0]++
				case "service":
					r.Items[0].F47++
				}
				before := w.Hash()
				err = w.ImportSavedObjects(r, nil, SavedObjectBinding{ID: 1, Owner: owner, Value: value})
				if corrupt != "valid" {
					if err == nil || w.Hash() != before || w.SavedObjects() != nil {
						t.Fatal("conflicting current history was accepted or changed World", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				bound, _ := w.EquippedItems(7)
				row, ok := w.SavedObjects().Item(1)
				if !ok || bound[0].ObjectID != 1 || bound[0].NativeRecord != nil || row.Token != token || row.F47 != 91 || row.F48 != 0x9876 {
					t.Fatal("binding lost current native fields or retained an unregistered record")
				}
				raw, err := w.MarshalBinary()
				var cold World
				if err != nil || cold.UnmarshalBinary(raw) != nil || cold.Hash() != w.Hash() {
					t.Fatal("bound current record did not survive cold native LOAD", err)
				}
			})
		}
	}
}
