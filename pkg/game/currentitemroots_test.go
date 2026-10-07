package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentSharedItemFront(t *testing.T) *FrontEnd {
	t.Helper()
	f, _ := openCurrentItemFixtureApp(t, true, func(doc *sav.DocumentData) {
		var shared, stack uint16
		for i, r := range doc.Objects {
			if savedItemClass(r.Class) {
				switch actorProjectionValue(t, r, "Identity") {
				case 0x410001:
					shared = uint16(i + 1)
				case 0x420001:
					stack = uint16(i + 1)
				}
			}
		}
		if shared == 0 || stack == 0 {
			t.Fatal("missing independent shared Item fixture")
		}
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
				continue
			}
			key := actorProjectionValue(t, *r, "Identity")
			if key != newGroupA && key != newGroupB1115 {
				continue
			}
			pack, _ := savedObjectRefs(r, "Inventory")
			held, _ := savedObjectRefs(r, "HeldWeapon")
			for _, id := range held {
				if id != 0 {
					pack = append(pack, id)
				}
			}
			pack = append(pack, shared, shared)
			if key == newGroupB1115 {
				pack = append(pack, stack)
			}
			savedObjectSetValue(r, "HasInventory", 1)
			savedObjectSetValue(r, "Inventory1C", 0)
			savedObjectSetValue(r, "Inventory20", 17)
			savedObjectSetRefs(r, "Inventory", pack, true)
			if key == newGroupA {
				savedObjectSetRefs(r, "HeldWeapon", []uint16{shared}, false)
			}
		}
	})
	return f
}

func TestCurrentItemRootsTwoOrdinarySAVCycles(t *testing.T) {
	f := currentSharedItemFront(t)
	r := f.live.world.SavedObjects()
	var id sim.SavedObjectID
	for _, row := range r.Items {
		if row.Token.Identity == 0x410001 {
			id = row.ID
		}
	}
	if id == 0 || len(r.Locations(id)) != 6 {
		t.Fatal("original ordinary Item roots were collapsed", id, r.Locations(id))
	}
	wantLocations := r.Locations(id)
	for cycle := 0; cycle < 2; cycle++ {
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, "shared Item roots")
		if err != nil {
			t.Fatal("current multi-root SAVE", err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil || a.Inventory == nil {
			t.Fatal("missing current roots", err)
		}
		indices := map[sim.SavedObjectID]uint16{}
		for _, binding := range a.Ownership {
			if binding.ID == 0 || binding.Object == 0 {
				continue
			}
			if binding.ID == 0 || indices[binding.ID] != 0 {
				t.Fatal("duplicate object binding")
			}
			indices[binding.ID] = binding.Object
		}
		if a.Inventory.Version != sim.SavedObjectsVersion {
			t.Fatal("current save emitted legacy ownership")
		}
		for _, row := range a.Ownership {
			if row.Kind == 1 && row.Owner != (sim.SavedObjectOwner{}) {
				t.Fatal("current save selected a primary Item owner")
			}
		}
		wire := indices[id]
		if wire == 0 {
			t.Fatal("shared Item lost ordinary record", id, a.Objects, a.Ownership)
		}
		if cycle == 0 {
			savedObjectSetValue(&doc.Objects[wire-1], "T1C", 901)
			refs, _ := savedObjectRefs(&doc.Objects[wire-1], "Effects")
			savedObjectSetValue(&doc.Objects[refs[0]-1], "E40", 321)
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
		}
		cold := cellStateFront(t)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("current multi-root LOAD", town, err)
		}
		app := cold.App("shared Item reload")
		if err := app.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		r = cold.live.world.SavedObjects()
		row, ok := r.Item(id)
		if !ok || row.Value.Price != 901 || row.Value.Effects[0].Operand != 321 || !reflect.DeepEqual(wantLocations, r.Locations(id)) {
			t.Fatal("ordinary value edit or surviving roots changed", row, r.Locations(id))
		}
		for _, at := range r.Locations(id) {
			if at.Owner.Kind == sim.SavedOwnerActorPack {
				pack, _ := cold.live.world.CarriedStacks(at.Owner.Entity)
				if !sim.StackStateEqual(pack[at.Index], row.Value) {
					t.Fatal("pack alias missed ordinary edit")
				}
			}
			if at.Owner.Kind == sim.SavedOwnerActorWorn {
				worn, _ := cold.live.world.EquippedItems(at.Owner.Entity)
				if !sim.StackStateEqual(sim.StackItem(worn[at.Owner.Slot-1], 1), row.Value) {
					t.Fatal("worn alias missed ordinary edit")
				}
			}
		}
		f = cold
	}
}

func currentRootSAVDocument(t *testing.T, f *FrontEnd) (sav.DocumentData, *currentActionData) {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "Item root mutation")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("missing root continuation", err)
	}
	return doc, a
}

func TestCurrentItemRootsKeepOrdinaryNullPackPositions(t *testing.T) {
	f := currentSharedItemFront(t)
	doc, a := currentRootSAVDocument(t, f)
	var actor sim.EntityID
	var expected []sim.SavedObjectID
	for _, binding := range a.Bindings {
		if binding.Structure || binding.Object == 0 {
			continue
		}
		pack, ok := savedObjectRefs(&doc.Objects[binding.Object-1], "Inventory")
		if !ok || len(pack) == 0 {
			continue
		}
		actor = binding.ID
		native, _ := f.live.world.CarriedStacks(actor)
		expected = append(expected, 0)
		for _, stack := range native {
			expected = append(expected, stack.ObjectID)
		}
		expected = append(expected, 0)
		pack = append([]uint16{0}, pack...)
		pack = append(pack, 0)
		savedObjectSetRefs(&doc.Objects[binding.Object-1], "Inventory", pack, true)
		break
	}
	if len(expected) < 3 {
		t.Fatal("no ordinary Pack fixture")
	}
	for cycle := 0; cycle < 2; cycle++ {
		f = loadCurrentRootSAV(t, doc)
		pack, _ := f.live.world.CarriedStacks(actor)
		if len(pack) != len(expected) {
			t.Fatal("null Pack position disappeared", cycle, pack)
		}
		for index, id := range expected {
			if pack[index].ObjectID != id || id == 0 && !sim.StackStateEqual(pack[index], sim.ItemStack{}) {
				t.Fatal("ordinary null Pack changed surrounding identity", cycle, index)
			}
		}
		if cycle == 0 {
			doc, _ = currentRootSAVDocument(t, f)
		}
	}
}

func loadCurrentRootSAV(t *testing.T, doc sav.DocumentData, fronts ...func(*testing.T) *FrontEnd) *FrontEnd {
	t.Helper()
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	front := cellStateFront
	if len(fronts) != 0 {
		front = fronts[0]
	}
	f := front(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("current root reload", town, err)
	}
	if err := f.App("Item root mutation reload").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCurrentItemRootsNativeMoveSplitAndSackThroughSAV(t *testing.T) {
	f := currentSharedItemFront(t)
	actors := newGroupActors1115(t, f.live.world)
	from, to := actors[newGroupA].ID, actors[newGroupB1115].ID
	r := f.live.world.SavedObjects()
	var stack, shared sim.SavedObjectID
	for _, row := range r.Items {
		if row.Token.Identity == 0x420001 {
			stack = row.ID
		}
		if row.Token.Identity == 0x410001 {
			shared = row.ID
		}
	}
	if stack == 0 || shared == 0 || len(r.Locations(stack)) != 2 {
		t.Fatal("missing explicit cross-actor stack fixture")
	}
	value, _ := r.Item(stack)
	if err := f.live.world.MoveCarried(from, to, value.Value.Code, 1); err != nil {
		t.Fatal(err)
	}
	r = f.live.world.SavedObjects()
	value, _ = r.Item(stack)
	var split sim.SavedObjectID
	for _, row := range r.Items {
		if row.Origin == (sim.SavedObjectOrigin{Kind: sim.SavedObjectSplit, Parent: stack}) && !row.Retired {
			split = row.ID
		}
	}
	if value.Value.Count != 3 || split == 0 || len(r.Locations(stack)) != 2 || len(r.Locations(split)) != 1 {
		t.Fatal("partial alias move did not create an independent node", value, split)
	}
	for cycle := 0; cycle < 2; cycle++ {
		doc, _ := currentRootSAVDocument(t, f)
		f = loadCurrentRootSAV(t, doc)
		r = f.live.world.SavedObjects()
		value, _ = r.Item(stack)
		copy, ok := r.Item(split)
		if value.Value.Count != 3 || !ok || copy.Value.Count != 1 || len(r.Locations(stack)) != 2 || len(r.Locations(shared)) != 6 {
			t.Fatal("SAV lost a surviving split or alias", cycle)
		}
	}
	if err := f.live.world.TakeSack(from, 15, 16); err != nil {
		t.Fatal(err)
	}
	r = f.live.world.SavedObjects()
	if len(r.Locations(shared)) != 6 || len(f.live.world.Sacks()) != 0 || !r.Sacks[0].Retired {
		t.Fatal("whole Sack occurrence transfer lost an actor alias")
	}
	doc, _ := currentRootSAVDocument(t, f)
	f = loadCurrentRootSAV(t, doc)
	if len(f.live.world.SavedObjects().Locations(shared)) != 6 {
		t.Fatal("SAV lost aliases after Sack retirement")
	}
}

func TestCurrentItemRootsMalformedSAVIsAtomic(t *testing.T) {
	f := currentSharedItemFront(t)
	hash := f.live.world.Hash()
	for _, edit := range []func(*currentActionData){
		func(a *currentActionData) { a.Ownership = append(a.Ownership, a.Ownership[0]) },
		func(a *currentActionData) {
			a.Ownership[0].Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 1}
		},
		func(a *currentActionData) {
			a.Inventory.ItemRoots = append(a.Inventory.ItemRoots, sim.SavedItemRoot{ID: a.Ownership[0].ID, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 1, Slot: 1}})
		},
	} {
		doc, a := currentRootSAVDocument(t, f)
		edit(a)
		payload, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, payload); err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := f.RestoreOriginal(raw); err == nil || f.live.world.Hash() != hash {
			t.Fatal("malformed current root graph changed a live mission")
		}
	}
}

func TestCurrentItemRootsDetachedAndRetiredChildUsesOrdinaryValue(t *testing.T) {
	f, _ := itemScrollOpen1115(t, 1, true)
	original := itemObjectByKey(t, f.live.world.SavedObjects(), 0x820001)
	actor := original.Owner.Entity
	for tick := 0; tick < 20 && f.live.world.ActorMotionActive(actor); tick++ {
		f.live.tick()
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindUseScroll, Entity: actor, X: int32(actor)})
	f.live.tick()
	casts := f.live.world.ScrollCasts()
	if len(casts) != 1 {
		t.Fatal("fixture did not reserve the shared-child Item")
	}
	reserved := casts[0].Item.ObjectID
	want := uint32(1 | 55<<16)
	for cycle := 0; cycle < 4; cycle++ {
		doc, a := currentRootSAVDocument(t, f)
		var child sim.SavedObjectID
		for _, row := range a.Ownership {
			if row.Kind == 1 && row.ID == reserved {
				if row.Object != 0 || row.Item == nil || len(row.Item.Value.Effects) != 0 || row.Item.Value.SourceEquipment.Spell != (sim.SourceItemSpell{}) {
					t.Fatal("private Item repeated child scalars")
				}
				child = row.Item.Effects[0]
			}
		}
		var wire uint16
		for _, row := range a.Ownership {
			if row.Kind == 2 && row.ID == child {
				wire = row.Object
			}
		}
		if wire == 0 {
			t.Fatal("shared child lost its ordinary node")
		}
		if cycle == 2 {
			want = 1 | 77<<16
		}
		if cycle == 0 || cycle == 2 {
			savedObjectSetValue(&doc.Objects[wire-1], "E40", want)
		}
		f = loadCurrentRootSAV(t, doc, itemScrollFront1115)
		r := f.live.world.SavedObjects()
		row, ok := r.Item(reserved)
		if !ok || row.Value.Effects[0].Operand != want || row.Effects[0] != row.Effects[1] || row.Retired != (cycle >= 2) {
			t.Fatal("detached or retired Item overrode its shared ordinary child", cycle, row)
		}
		if cycle < 2 {
			casts := f.live.world.ScrollCasts()
			if len(casts) != 1 || casts[0].Item.Effects[0].Operand != want {
				t.Fatal("active reservation missed the ordinary child edit")
			}
		}
		if cycle == 1 {
			for range 50 {
				f.live.tick()
			}
			if len(f.live.world.ScrollCasts()) != 0 {
				t.Fatal("shared child prevented actual reservation completion")
			}
		}
	}
}

func currentItemWorldDiagnostics(t *testing.T, before, after []byte) {
	t.Helper()
	var a, b sim.World
	if a.UnmarshalBinary(before) != nil || b.UnmarshalBinary(after) != nil {
		return
	}
	t.Logf("World hashes %x/%x; equal Items=%v Actions=%v Policy=%v Sacks=%v", a.Hash(), b.Hash(), reflect.DeepEqual(a.SavedObjects(), b.SavedObjects()), reflect.DeepEqual(a.Actions(), b.Actions()), reflect.DeepEqual(a.CurrentPolicy(), b.CurrentPolicy()), reflect.DeepEqual(a.Sacks(), b.Sacks()))
	currentItemFieldDiagnostics(t, "Objects", reflect.ValueOf(a.SavedObjects()), reflect.ValueOf(b.SavedObjects()))
	currentItemFieldDiagnostics(t, "Actions", reflect.ValueOf(a.Actions()), reflect.ValueOf(b.Actions()))
	left, right := a.Entities(), b.Entities()
	if len(left) != len(right) {
		t.Log("different actor population", len(left), len(right))
		return
	}
	for i := range left {
		x, y := reflect.ValueOf(left[i]), reflect.ValueOf(right[i])
		for j := 0; j < x.NumField(); j++ {
			if x.Field(j).CanInterface() && !reflect.DeepEqual(x.Field(j).Interface(), y.Field(j).Interface()) {
				t.Logf("Entity %d field %s: %v -> %v", left[i].ID, x.Type().Field(j).Name, x.Field(j).Interface(), y.Field(j).Interface())
			}
		}
	}
}

func currentItemFieldDiagnostics(t *testing.T, path string, a, b reflect.Value) {
	t.Helper()
	if !a.IsValid() || !b.IsValid() || !a.CanInterface() || !b.CanInterface() || reflect.DeepEqual(a.Interface(), b.Interface()) {
		return
	}
	if a.Kind() == reflect.Pointer && !a.IsNil() && !b.IsNil() {
		currentItemFieldDiagnostics(t, path, a.Elem(), b.Elem())
		return
	}
	if a.Kind() == reflect.Struct {
		for i := range a.NumField() {
			currentItemFieldDiagnostics(t, path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
		}
		return
	}
	if a.Kind() == reflect.Slice && a.Len() == b.Len() && a.Len() != 0 {
		for i := range a.Len() {
			currentItemFieldDiagnostics(t, fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i))
		}
		return
	}
	t.Logf("%s: %#v -> %#v", path, a.Interface(), b.Interface())
}
