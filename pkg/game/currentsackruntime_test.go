package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestCurrentNativeSackKeepsTokenAcrossCyclesAndOrdinaryEdits(t *testing.T) {
	f, _ := itemMutationOpen1115(t, 1, false)
	f.live.tick()
	old := map[sim.SavedObjectID]bool{}
	for _, row := range f.live.world.SavedObjects().Sacks {
		old[row.ID] = true
	}
	actor := newGroupActors1115(t, f.live.world)[newGroupA]
	f.live.pending = append(f.live.pending, sim.Damage(actor.ID, actor.HP+10))
	var id sim.SavedObjectID
	for tick := 0; tick < 100 && id == 0; tick++ {
		f.live.tick()
		for _, row := range f.live.world.SavedObjects().Sacks {
			if !old[row.ID] {
				id = row.ID
			}
		}
	}
	if id == 0 {
		t.Fatal("actual death did not construct a native Sack")
	}
	want := f.live.world.SavedObjects()
	index := -1
	for i, row := range want.Sacks {
		if row.ID == id {
			index = i
		}
	}
	if index < 0 || want.Sacks[index].Token.Identity != 0 {
		t.Fatal("native Sack fixture lost absent key")
	}
	for cycle := 0; cycle < 3; cycle++ {
		before := f.live.world.Hash()
		doc, a := currentRootSAVDocument(t, f)
		var binding currentOwnedObject
		for _, row := range a.Ownership {
			if row.Kind == 4 && row.ID == id {
				binding = row
			}
		}
		if binding.Object == 0 || before != f.live.world.Hash() {
			t.Fatal("native Sack SAVE lost its binding or changed the World")
		}
		if cycle < 2 && (binding.IdentityPresent == nil || *binding.IdentityPresent || binding.IdentityAnchor == nil) {
			t.Fatal("absent native key has no ordinary anchor")
		}
		if cycle == 1 {
			r := &doc.Objects[binding.Object-1]
			token, _, _, err := savedSackRecord(r)
			if err != nil {
				t.Fatal(err)
			}
			oldKey := token.Identity
			token.Identity = 0xfed10077
			token.Position[0], token.Position[1], token.Position[4], token.Position[5] = 7, 9, 13, 15
			savedObjectSetValue(r, "Identity", token.Identity)
			mustSetRaw(r, "Block12", token.Position[:])
			for i := range doc.World.Cells {
				if doc.World.Cells[i].Sack == oldKey {
					doc.World.Cells[i].Sack = token.Identity
				}
			}
			want.Sacks[index].Token = token
		}
		f = loadCurrentRootSAV(t, doc, func(t *testing.T) *FrontEnd { return itemMutationFront1115(t, 1) })
		if got := f.live.world.SavedObjects(); !reflect.DeepEqual(got, want) {
			currentItemFieldDiagnostics(t, "Sack registry", reflect.ValueOf(want).Elem(), reflect.ValueOf(got).Elem())
			t.Fatal("native Sack token or registry changed across ordinary SAV", cycle)
		}
	}
}

func TestCurrentSackRuntimePreservesOrdinaryValueAcrossCycles(t *testing.T) {
	f := currentSharedItemFront(t)
	initial := f.live.world.SavedObjects().Sacks[0]
	want := initial.Token.RuntimeID
	for cycle := 0; cycle < 3; cycle++ {
		doc, a := currentRootSAVDocument(t, f)
		var object uint16
		for _, row := range a.Ownership {
			if row.Kind == 4 && row.ID == initial.ID {
				object = row.Object
			}
		}
		if object == 0 {
			t.Fatal("current Sack lost explicit binding")
		}
		got, err := savedStructureValue(&doc.Objects[object-1], "RuntimeID")
		if err != nil || got != want {
			t.Fatal("current Sack runtime value was narrowed or allocated again", got, want, err)
		}
		if cycle == 0 {
			want = 0xfedcba98
			savedObjectSetValue(&doc.Objects[object-1], "RuntimeID", want)
		}
		f = loadCurrentRootSAV(t, doc)
		row := f.live.world.SavedObjects().Sacks[0]
		if row.ID != initial.ID || row.Token.RuntimeID != want {
			t.Fatal("ordinary Sack runtime edit was not current on LOAD")
		}
	}
}

func TestCurrentSackRuntimeReservesBeforeOtherAllocations(t *testing.T) {
	f := currentSharedItemFront(t)
	doc, a := currentRootSAVDocument(t, f)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	var sack uint16
	for _, row := range a.Ownership {
		if row.Kind == 4 && row.ID != 0 {
			sack = row.Object
		}
	}
	if sack == 0 {
		t.Fatal("current Sack has no ordinary binding")
	}
	for i := range doc.Objects {
		if runtimePlaceable(doc.Objects[i].Class) {
			savedObjectSetValue(&doc.Objects[i], "RuntimeID", 0)
		}
	}
	savedObjectSetValue(&doc.Objects[sack-1], "RuntimeID", 1)
	if err := projectCurrentRuntimeIDs(&doc, s); err != nil {
		t.Fatal(err)
	}
	for i, row := range doc.Objects {
		if !runtimePlaceable(row.Class) {
			continue
		}
		id, _ := savedStructureValue(&row, "RuntimeID")
		if uint16(i+1) == sack && id != 1 || uint16(i+1) != sack && id == 1 {
			t.Fatal("earlier allocation reused the current Sack runtime ID")
		}
	}
	var before sim.World
	if err := before.UnmarshalBinary(s.World); err != nil || before.Hash() != f.live.world.Hash() {
		t.Fatal("runtime projection changed current World", err)
	}
}
