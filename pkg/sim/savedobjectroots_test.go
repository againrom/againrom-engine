package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func sharedItemWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 42, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 10}, {ID: 8, X: 2, Y: 2, HP: 10}})
	makeItem := func(id SavedObjectID, code uint16, count uint32, effects []SavedObjectID) SavedItemObject {
		v := ItemStack{ObjectID: id, Code: code, Count: count, WeightPresent: true, Weight: 2, Price: 17}
		for range effects {
			v.Effects = append(v.Effects, ItemEffect{Kind: 8, Mode: 1, Operand: 17})
		}
		return SavedItemObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Value: v, Effects: effects, Token: SavedObjectToken{Identity: uint32(id) + 100, T1C: 17}}
	}
	a, b := makeItem(10, 0x101, 3, []SavedObjectID{30, 30}), makeItem(20, 0x201, 1, []SavedObjectID{30})
	pack := func(id EntityID) SavedObjectOwner { return SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: id} }
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: 100, Items: []SavedItemObject{a, b}, Effects: []SavedEffectObject{{ID: 30, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Value: a.Value.Effects[0]}}, Containers: []SavedObjectContainer{{Owner: pack(7), Present: true, InsertIndex: 2, Accumulator: 12, Items: []SavedObjectID{10, 10}}, {Owner: pack(8), Present: true, InsertIndex: 2, Accumulator: 8, Items: []SavedObjectID{10, 20}}}}
	for _, id := range []EntityID{7, 8} {
		if err := r.AddItemRoot(20, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: id, Slot: 1}); err != nil {
			t.Fatal(err)
		}
	}
	w.carried[0], w.carried[1] = []ItemStack{a.Value.Clone(), a.Value.Clone()}, []ItemStack{a.Value.Clone(), b.Value.Clone()}
	w.equipment[0][0], w.equipment[1][0] = b.Value.Instance(), b.Value.Instance()
	var bindings []SavedObjectBinding
	for _, row := range r.Items {
		for _, at := range r.Locations(row.ID) {
			bindings = append(bindings, SavedObjectBinding{ID: row.ID, Owner: at.Owner, Index: at.Index, Value: row.Value.Clone()})
		}
	}
	for i := range w.carried {
		for j := range w.carried[i] {
			w.carried[i][j].ObjectID = 0
		}
		w.equipment[i][0].ObjectID = 0
		w.recomputeLoad(i)
	}
	if err := w.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal(err)
	}
	return w
}

func roundTripSharedItems(t *testing.T, w *World) *World {
	t.Helper()
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(before); err != nil {
		t.Fatal(err)
	}
	after, err := cold.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || w.Hash() != cold.Hash() || !reflect.DeepEqual(w.savedObjects, cold.savedObjects) {
		t.Fatalf("shared roots or values changed: bytes=%v hashes=%x/%x registry=%v before=%+v after=%+v", bytes.Equal(before, after), w.Hash(), cold.Hash(), reflect.DeepEqual(w.savedObjects, cold.savedObjects), w.savedObjects, cold.savedObjects)
	}
	return &cold
}

func TestSharedItemMergeIntoPackPreservesWornAliases(t *testing.T) {
	w := sharedItemWorld(t)
	r := w.savedObjects.Clone()
	incoming := *r.item(20)
	incoming.ID, incoming.Value.ObjectID, incoming.Token.Identity = 40, 40, 140
	incoming.InFlight = 1
	r.Items = append(r.Items, incoming)
	if err := r.Validate(); err != nil {
		t.Fatal("merge input is invalid", err)
	}
	before := r.Locations(20)
	id, err := r.Insert(40, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}, SavedMergeNativeRetention)
	if err != nil {
		t.Fatal("normal merge into a Pack node shared with worn roots", err)
	}
	if id != 20 || r.item(20).Value.Count != 2 || !reflect.DeepEqual(r.Locations(20), before) || !r.item(40).Retired || r.effect(30).Retired {
		t.Fatal("merge changed surviving identity, roots or child lifetime")
	}
}

func TestSharedItemRootsMoveSplitRetireAndBinary(t *testing.T) {
	w := roundTripSharedItems(t, sharedItemWorld(t))
	n := w.sourceMutationCopy(0)
	loads := []loadMutation{n.beginLoadMutation(0), n.beginLoadMutation(1)}
	moved, ok := n.takeCarriedObject(0, 1, true)
	if !ok || !n.putCarriedObject(1, moved) {
		t.Fatal("exact occurrence move failed")
	}
	for i := range loads {
		if !n.finishLoadMutation(i, loads[i]) {
			t.Fatal("move load failed")
		}
	}
	if n.carried[0][0].ObjectID != 10 || len(n.carried[1]) != 3 || n.carried[1][2].ObjectID != 10 || len(n.savedObjects.Locations(10)) != 3 {
		t.Fatal("move lost another occurrence or chose first index")
	}
	w = roundTripSharedItems(t, &n)
	n = w.sourceMutationCopy(0)
	before := n.beginLoadMutation(0)
	split, ok := n.takeCarriedObject(0, 0, false)
	if !ok || split.ObjectID == 10 || !n.putWornObject(0, 3, split.Instance()) {
		t.Fatal("explicit one-unit split failed")
	}
	n.equipment[0][2] = split.Instance()
	if !n.finishLoadMutation(0, before) {
		t.Fatal("split load failed")
	}
	for _, pack := range n.carried {
		for _, row := range pack {
			if row.ObjectID == 10 && row.Count != 2 {
				t.Fatal("shared Count did not reach a surviving view")
			}
		}
	}
	child := n.savedObjects.item(split.ObjectID)
	if child.Effects[0] == child.Effects[1] || child.Effects[0] == 30 || len(n.savedObjects.Locations(10)) != 3 {
		t.Fatal("split collapsed child occurrences or detached original aliases")
	}
	w = roundTripSharedItems(t, &n)
	for _, id := range []SavedObjectID{10, 20} {
		for len(w.savedObjects.Locations(id)) != 0 {
			at := w.savedObjects.Locations(id)[0]
			i := indexOfEntity(w.entities, at.Owner.Entity)
			n = w.sourceMutationCopy(i)
			load := n.beginLoadMutation(i)
			if at.Owner.Kind == SavedOwnerActorPack {
				if _, ok := n.takeCarriedObject(i, int(at.Index), true); !ok {
					t.Fatal("retire take")
				}
			} else {
				item := n.equipment[i][at.Owner.Slot-1]
				if !n.takeWornObject(i, int(at.Owner.Slot), item) {
					t.Fatal("retire worn")
				}
				n.equipment[i][at.Owner.Slot-1] = ItemInstance{}
			}
			if err := n.savedObjects.Dispose(id); err != nil {
				t.Fatal(err)
			}
			if !n.finishLoadMutation(i, load) {
				t.Fatal("retire load")
			}
			if len(n.savedObjects.Locations(id)) != 0 && n.savedObjects.item(id).Retired {
				t.Fatal("retired Item still has surviving roots")
			}
			if id == 10 && n.savedObjects.effect(30).Retired {
				t.Fatal("shared child retired while the second Item survives")
			}
			w = roundTripSharedItems(t, &n)
		}
	}
	if !w.savedObjects.effect(30).Retired || w.savedObjects.item(split.ObjectID).Retired || w.equipment[0][2].ObjectID != split.ObjectID {
		t.Fatal("last-reference retirement affected independent split")
	}
}

func TestSharedItemRootsRejectConflictsAtomically(t *testing.T) {
	w := sharedItemWorld(t)
	raw := mustMarshal(t, w)
	hash := w.Hash()
	for _, edit := range []func(*SavedObjects){
		func(r *SavedObjects) { r.ItemRoots = append(r.ItemRoots, r.ItemRoots[0]) },
		func(r *SavedObjects) { r.ItemRoots[0].ID = 999 },
		func(r *SavedObjects) { r.Items[0].Owner = SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7} },
		func(r *SavedObjects) { r.Items[0].Retired = true },
	} {
		r := w.savedObjects.Clone()
		edit(r)
		n := *w
		n.savedObjects = r
		if _, err := n.MarshalBinary(); err == nil {
			t.Fatal("corrupt current roots serialized")
		}
		if w.Hash() != hash || !bytes.Equal(raw, mustMarshal(t, w)) {
			t.Fatal("rejection changed live world")
		}
	}
	r := w.savedObjects.Clone()
	before := r.Clone()
	row := r.item(10)
	if _, err := r.TakeWholeAt(10, SavedItemLocation{SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, 9}, row.Value); err == nil || !reflect.DeepEqual(r, before) {
		t.Fatal("invalid occurrence was not atomic")
	}
	if _, err := r.TakeWhole(10, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, row.Value); err == nil {
		t.Fatal("ambiguous legacy take selected a winner")
	}
	bad := append([]byte(nil), raw...)
	bad[0] = 0xff
	if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != hash {
		t.Fatal("failed decode changed receiver")
	}
}

func sharedItemSackWorld(t *testing.T) *World {
	t.Helper()
	w := sharedItemWorld(t)
	row := w.savedObjects.item(10)
	w.savedObjects.Sacks = []SavedSackObject{{ID: 40, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Token: SavedObjectToken{Position: [12]byte{0, 0, 3, 3}}}}
	w.savedObjects.SackRoots = []SavedObjectID{40}
	w.savedObjects.Containers = append(w.savedObjects.Containers, SavedObjectContainer{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: 40}, Present: true, InsertIndex: 2, Accumulator: 12, Items: []SavedObjectID{10, 10}})
	w.sacks = []Sack{{X: 3, Y: 3, ObjectID: 40}}
	for range 2 {
		for range row.Value.Count {
			w.sacks[0].ItemInstances = append(w.sacks[0].ItemInstances, row.Value.Instance())
		}
	}
	w.sacks[0].Items = itemCodes(w.sacks[0].ItemInstances)
	if !w.syncSavedSackSlots(0) || len(w.savedObjects.Containers[2].Items) != 2 {
		t.Fatal("adjacent same-node Sack occurrences collapsed")
	}
	return roundTripSharedItems(t, w)
}

func TestSharedItemSackDrainAndGroundMerge(t *testing.T) {
	t.Run("drain cannot merge a split into its parent", func(t *testing.T) {
		w := sharedItemSackWorld(t)
		if err := w.TakeSack(7, 3, 3); err != nil {
			t.Fatal(err)
		}
		if len(w.sacks) != 0 || w.savedObjects.item(10).Value.Count != 1 || len(w.savedObjects.Locations(10)) != 3 || !w.savedObjects.sack(40).Retired {
			t.Fatal("Sack drain lost surviving roots or failed to remove the Sack", w.savedObjects.Locations(10))
		}
		for _, pack := range w.carried {
			for _, row := range pack {
				if row.ObjectID == 10 && row.Count != 1 {
					t.Fatal("Sack split missed a surviving actor view")
				}
			}
		}
		roundTripSharedItems(t, w)
	})
	t.Run("ground merge updates every alias", func(t *testing.T) {
		w := sharedItemSackWorld(t)
		row := w.savedObjects.Clone().Items[0]
		row.ID, row.Value.ObjectID, row.Value.Count = 50, 50, 1
		row.InFlight = 1
		w.savedObjects.Items = append(w.savedObjects.Items, row)
		if !w.putGroundObject(3, 3, 0, row.Value) {
			t.Fatal("ground merge failed")
		}
		if !w.savedObjects.item(50).Retired || w.savedObjects.item(10).Value.Count != 4 || len(w.sacks[0].ItemInstances) != 8 || len(w.savedObjects.Containers[2].Items) != 2 {
			t.Fatal("ground merge collapsed roots or lost node count")
		}
		for _, pack := range w.carried {
			for _, row := range pack {
				if row.ObjectID == 10 && row.Count != 4 {
					t.Fatal("ground merge missed an actor alias")
				}
			}
		}
		roundTripSharedItems(t, w)
	})
}

func TestSharedItemConcurrentScrollReservations(t *testing.T) {
	w := sharedItemWorld(t)
	row := w.savedObjects.Items[1]
	row.Value.Code, row.Value.Kind = 0x0e01, 4
	row.Value.Effects = []ItemEffect{{Kind: 41, Operand: 0x0001000b}}
	row.Effects = []SavedObjectID{30}
	w.savedObjects.Items = []SavedItemObject{row}
	w.savedObjects.Effects[0].Value = row.Value.Effects[0]
	w.savedObjects.ItemRoots = nil
	for i := range w.entities {
		w.entities[i].HP, w.entities[i].MaxHP = 10, 10
		w.carried[i] = []ItemStack{row.Value.Clone()}
		w.equipment[i] = [EquipSlots]ItemInstance{}
		w.savedObjects.Containers[i].Items = []SavedObjectID{row.ID}
		w.savedObjects.Containers[i].InsertIndex = 1
		w.savedObjects.Containers[i].Accumulator = 2
		w.recomputeLoad(i)
	}
	w.spells = []SpellRule{{ID: 11}}
	if !w.beginScroll(0, 0, 0, 3, 3, true) || !w.beginScroll(1, 0, 0, 3, 3, true) {
		t.Fatal("two actors could not reserve one aliased Item")
	}
	if len(w.scrollCasts) != 2 || w.scrollCasts[0].Reservation == w.scrollCasts[1].Reservation || w.scrollCasts[0].Item.ObjectID != w.scrollCasts[1].Item.ObjectID {
		t.Fatal("reservation identity collapsed")
	}
	w = roundTripSharedItems(t, w)
	raw := mustMarshal(t, w)
	suffix := w.appendSavedObjects(nil)
	at := bytes.Index(raw, suffix)
	if at < 0 {
		t.Fatal("missing object suffix")
	}
	bad := bytes.Clone(raw)
	clear(bad[at+len(suffix)-12 : at+len(suffix)-4])
	hash := w.Hash()
	if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != hash || !bytes.Equal(raw, mustMarshal(t, w)) {
		t.Fatal("corrupt reservation mapping was not atomic")
	}
	if !w.refundScroll(0, 0) {
		t.Fatal("first reservation refund failed")
	}
	if len(w.scrollCasts) != 1 || !w.retireSessionObject(w.scrollCasts[0].Item, w.scrollCasts[0].Reservation) {
		t.Fatal("second reservation disposal failed")
	}
	w.scrollCasts = nil
	if w.savedObjects.item(row.ID).Retired || w.carried[0][0].ObjectID != row.ID {
		t.Fatal("disposing one reservation destroyed refunded alias")
	}
	roundTripSharedItems(t, w)
}

// This writer is confined to the independent predecessor control. Production
// always writes the current root schema and never branches on input origin.
func legacyItemSuffix(row SavedItemObject, container SavedObjectContainer) []byte {
	b := binary.LittleEndian.AppendUint32(nil, 1)
	b = append(b, 1)
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = binary.LittleEndian.AppendUint32(b, 0)
	b = binary.LittleEndian.AppendUint32(b, 0)
	b = binary.LittleEndian.AppendUint64(b, uint64(row.ID))
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = binary.LittleEndian.AppendUint64(b, 2)
	for _, n := range []uint32{1, 0, 0, 0, 0, 1} {
		b = binary.LittleEndian.AppendUint32(b, n)
	}
	b = binary.LittleEndian.AppendUint64(b, uint64(row.ID))
	b = appendSavedObjectFixed(b, row.Origin)
	b = appendSavedObjectFixed(b, row.Owner)
	b = appendSavedObjectStack(b, row.Value)
	b = appendSavedObjectFixed(b, row.Token)
	b = append(b, row.F45, row.F46, row.F47)
	b = binary.LittleEndian.AppendUint16(b, row.F48)
	b = appendSavedObjectIDs(b, nil)
	b = binary.LittleEndian.AppendUint64(b, 0)
	b = appendSavedObjectCoverage(b, row.Coverage)
	b = appendSavedObjectFixed(b, container.Owner)
	b = appendSavedObjectFixed(b, container.Present)
	b = binary.LittleEndian.AppendUint32(b, container.InsertIndex)
	b = binary.LittleEndian.AppendUint32(b, uint32(container.Accumulator))
	b = appendSavedObjectIDs(b, container.Items)
	b = appendSavedObjectCoverage(b, container.Coverage)
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)))
}

func TestSharedItemLegacyOwnerBinaryMigrationAndAtomicCorruption(t *testing.T) {
	w := mustWorld(t, 23, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1}})
	owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}
	row := SavedItemObject{ID: 1, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Owner: owner, Value: ItemStack{ObjectID: 1, Code: 0x101, Count: 1, Price: 17, WeightPresent: true, Weight: 2}, Token: SavedObjectToken{T1C: 17, Identity: 31}}
	c := SavedObjectContainer{Owner: owner, Present: true, InsertIndex: 1, Accumulator: 2, Items: []SavedObjectID{1}}
	r := &SavedObjects{Version: 1, NextID: 2, Items: []SavedItemObject{row}, Containers: []SavedObjectContainer{c}}
	if err := r.MigrateLegacyOwners(); err != nil {
		t.Fatal(err)
	}
	w.savedObjects = r
	w.carried[0] = []ItemStack{row.Value.Clone()}
	w.recomputeLoad(0)
	current := mustMarshal(t, w)
	suffix := w.appendSavedObjects(nil)
	at := bytes.Index(current, suffix)
	old := legacyItemSuffix(row, c)
	legacy := append(bytes.Clone(current[:at]), old...)
	legacy = append(legacy, current[at+len(suffix):]...)
	var cold World
	if err := cold.UnmarshalBinary(legacy); err != nil {
		t.Fatal(err)
	}
	if cold.savedObjects.Items[0].Owner != (SavedObjectOwner{}) || cold.savedObjects.Version != SavedObjectsVersion || !bytes.Equal(current, mustMarshal(t, &cold)) || w.Hash() != cold.Hash() {
		t.Fatal("legacy owner did not migrate to the one current model")
	}
	bad := bytes.Clone(legacy)
	bad[at+61+17] = 99
	before := cold.Hash()
	if err := cold.UnmarshalBinary(bad); err == nil || cold.Hash() != before || !bytes.Equal(current, mustMarshal(t, &cold)) {
		t.Fatal("malformed legacy owner partially adopted")
	}
}
