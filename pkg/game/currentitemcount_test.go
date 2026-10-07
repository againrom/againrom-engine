package game

import (
	"bytes"
	"encoding/json"
	"testing"

	"againrom/pkg/sim"
)

func TestCurrentBoundItemCountKeepsOrdinaryEditsAndAliases(t *testing.T) {
	for _, edit := range []string{"count", "price", "repeat"} {
		t.Run(edit, func(t *testing.T) {
			f, _ := currentWideCursorSource(t, 1)
			original := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001)
			doc, actions := currentRootSAVDocument(t, f)
			var object uint16
			for _, row := range actions.Ownership {
				if row.ID == original.ID {
					if row.Object == 0 || row.Item != nil || row.CountLift == nil || *row.CountLift != (currentItemCountLift{Wire: 65535, Lift: 1}) {
						t.Fatal("wide Item has no single ordinary value authority", row)
					}
					object = row.Object
				}
			}
			if object == 0 {
				t.Fatal("wide Item lost its exact ordinary binding")
			}
			wantCount, wantPrice, occurrences := uint32(65536), int32(31), 1
			switch edit {
			case "count":
				mustSetValue(&doc.Objects[object-1], "F42", 7)
				wantCount = 7
			case "price":
				mustSetValue(&doc.Objects[object-1], "T1C", 991)
				wantPrice = 991
			case "repeat":
				for _, actor := range actions.Bindings {
					if actor.ID == original.Owner.Entity && actor.Object != 0 && !actor.Structure {
						r := &doc.Objects[actor.Object-1]
						refs, _ := savedObjectRefs(r, "Inventory")
						savedObjectSetRefs(r, "Inventory", append(refs, object), true)
						occurrences = 2
					}
				}
			}
			policy, _ := json.Marshal(actions)
			cold := currentCursorCold(t, doc)
			after, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _ := json.Marshal(after)
			if !bytes.Equal(policy, unchanged) {
				t.Fatal("ordinary edit changed native count policy")
			}
			for cycle := 0; cycle < 2; cycle++ {
				r := cold.live.world.SavedObjects()
				item, ok := r.Item(original.ID)
				if !ok || item.Value.Count != wantCount || item.Value.Price != wantPrice || item.Token.Identity != original.Token.Identity || len(r.Locations(original.ID)) != occurrences {
					t.Fatal("ordinary count/value/alias edit did not remain current", item, r.Locations(original.ID))
				}
				pack, _ := cold.live.world.CarriedStacks(original.Owner.Entity)
				seen := 0
				for _, value := range pack {
					if value.ObjectID == original.ID {
						seen++
						if !sim.StackStateEqual(item.Value, value) {
							t.Fatal("bound occurrence missed current Item value")
						}
					}
				}
				if seen != occurrences {
					t.Fatal("ordinary repeated edge was collapsed")
				}
				cold = currentCursorRoundTrip(t, cold)
			}
		})
	}
}

func TestCurrentBoundItemCountRejectsMalformedWidth(t *testing.T) {
	for _, lift := range []currentItemCountLift{{}, {Wire: 0, Lift: 65536}, {Wire: 65535}, {Wire: 1, Lift: 1}, {Wire: 65535, Lift: ^uint32(0)}} {
		row := currentOwnedObject{Kind: 1, ID: 1, Object: 1, CountLift: &lift}
		if validateCurrentItemCount(row) == nil {
			t.Fatal("invalid native count operand accepted", lift)
		}
	}
	valid := currentOwnedObject{Kind: 1, ID: 1, Object: 1, CountLift: &currentItemCountLift{Wire: 65535, Lift: ^uint32(0) - 65535}}
	if err := validateCurrentItemCount(valid); err != nil {
		t.Fatal("full uint32 native count refused", err)
	}
	for _, invalid := range []currentOwnedObject{
		{Kind: 1, Object: 1, CountLift: valid.CountLift},
		{Kind: 1, ID: 1, CountLift: valid.CountLift},
		{Kind: 2, ID: 1, Object: 1, CountLift: valid.CountLift},
		{Kind: 1, ID: 1, Object: 1, Item: &sim.SavedItemObject{}, CountLift: valid.CountLift},
	} {
		if validateCurrentItemCount(invalid) == nil {
			t.Fatal("count operand accepted without a sole ordinary bound Item")
		}
	}
}
