package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func cityProjectionSource(t *testing.T) ([]byte, func() *FrontEnd) {
	t.Helper()
	_, newFront := spellbookCitySource(t)
	model := cityfixture.City(false)
	token := func(key uint32) []byte {
		v := make([]byte, 37)
		binary.LittleEndian.PutUint32(v[29:], key)
		binary.LittleEndian.PutUint32(v[25:], 150)
		return v
	}
	for i := 1; i <= 2; i++ {
		u := model.Objects[i].Unit
		u.SpellbookFlag, u.SpellbookCount, u.Spells = 1, 27, make([]uint16, 26)
		u.Spells[25] = 6
		u.Container = []uint16{7, uint16(7 + i)}
		u.Reference74 = 7
	}
	model.Objects = append(model.Objects,
		sav.CityObjectData{Class: "Effect", Effect: &sav.CityEffectData{Token: token(0x4000), Fields: []byte{44, 0, 5, 0, 0, 0, 0}}},
		sav.CityObjectData{Class: "Effect", Effect: &sav.CityEffectData{Token: token(0x5000), Fields: []byte{44, 0, 5, 0, 0, 0, 0}}},
		sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: []byte{26, 7, 2, 93, 0, 0, 0x60, 0, 0}}})
	for i := 0; i < 3; i++ {
		fields, derived := make([]byte, 12), make([]byte, 47)
		binary.LittleEndian.PutUint16(fields, 0x101)
		binary.LittleEndian.PutUint16(fields[2:], 1)
		fields[4] = 2
		binary.LittleEndian.PutUint16(fields[10:], 9)
		derived[0] = 1
		children := []uint16{4, 4}
		if i == 1 {
			children = []uint16{4, 5}
		}
		model.Objects = append(model.Objects, sav.CityObjectData{Class: "Weapon", Item: &sav.CityItemData{
			Token: token(uint32(0x7000 + i*0x1000)), Fields: fields, Derived: derived, Effects: children, WeaponExtra: 6}})
	}
	city, err := sav.CityFromData(model)
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 123}
	for _, actor := range city.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(actor))
	}
	raw, err := city.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	return raw, newFront
}

func cityProjectionLoad(t *testing.T, raw []byte, newFront func() *FrontEnd) *FrontEnd {
	t.Helper()
	f := newFront()
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("city topology LOAD", town, err)
	}
	if f.Town.cityObjects == nil {
		t.Fatal("city topology was not installed")
	}
	return f
}

func cityProjectionSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	beforeParty, beforeGraph := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone()
	raw, err := f.ExportCurrentSave(s, "city topology")
	if err != nil {
		t.Fatal("city topology SAVE", err)
	}
	if !reflect.DeepEqual(beforeParty, f.Carried) || !reflect.DeepEqual(beforeGraph, f.Town.cityObjects) {
		t.Fatal("SAVE changed current city values or graph")
	}
	return raw
}

func cityProjectionWire(t *testing.T, raw []byte) (sav.DocumentData, *currentActionData, map[sim.SavedObjectID]uint16) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.Inventory == nil {
		t.Fatal("current city topology metadata absent", err)
	}
	indices := map[sim.SavedObjectID]uint16{}
	for _, row := range a.Objects {
		if row.ID == 0 || indices[row.ID] != 0 {
			t.Fatal("city repeated/absent topology ID", row)
		}
		indices[row.ID] = row.Object
	}
	for _, row := range a.Ownership {
		if row.Kind != 1 || row.Item != nil || row.Effect != nil || row.Spell != nil || row.Sack != nil {
			t.Fatal("city kept private object values", row)
		}
	}
	return doc, a, indices
}

func cityObjectIdentityTopology(g *cityObjectTopology) *cityObjectTopology {
	n := g.Clone()
	if n != nil {
		n.ItemRecords = nil
		n.EffectRecords = nil
		n.SpellRecords = nil
	}
	return n
}

func cityObjectNativeIdentities(g *cityObjectTopology) (map[sim.SavedObjectID]uint32, map[sim.SavedObjectID]uint32, map[sim.SavedObjectID]uint32) {
	items, effects, spells := map[sim.SavedObjectID]uint32{}, map[sim.SavedObjectID]uint32{}, map[sim.SavedObjectID]uint32{}
	if g == nil {
		return items, effects, spells
	}
	for id, row := range g.ItemRecords {
		items[id] = row.Token.Identity
	}
	for id, row := range g.EffectRecords {
		effects[id] = row.Token.Identity
	}
	for id, row := range g.SpellRecords {
		spells[id] = row.This
	}
	return items, effects, spells
}

func TestCityObjectProjectionTwoSAVCyclesPreserveAliasesAndNativeAbsence(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	wantGraph, wantParty := f.Town.cityObjects.Clone(), mapload.CloneParty(f.Carried)
	if len(wantGraph.Items) != 3 || len(wantGraph.Effects) != 2 || len(wantGraph.Spells) != 1 {
		t.Fatal("source did not carry distinct nodes and shared children", wantGraph)
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw = cityProjectionSave(t, f)
		doc, a, indices := cityProjectionWire(t, raw)
		if len(indices) != 6 || a.Inventory.NextID != wantGraph.NextID || len(doc.DeadActors) != 0 || doc.World != nil {
			t.Fatal("SAVE changed topology population or roots", cycle, len(indices), a.Inventory.NextID)
		}
		for _, item := range wantGraph.Items {
			record := &doc.Objects[indices[item.ID]-1]
			effects, _ := savedObjectRefs(record, "Effects")
			if len(effects) != len(item.Effects) {
				t.Fatal("Effect edge count changed", item, effects)
			}
			for slot, child := range item.Effects {
				if effects[slot] != indices[child] {
					t.Fatal("shared or repeated Effect copied", cycle, item, effects)
				}
			}
			spell, _ := savedObjectRefs(record, "WeaponSpell")
			if len(spell) != 1 || spell[0] != indices[wantGraph.Spells[0]] {
				t.Fatal("shared weapon/book Spell copied", spell)
			}
		}
		for _, policy := range a.Ownership {
			if policy.ID != 0 {
				t.Fatal("SAVE invented an absent native handle", policy)
			}
		}
		f = cityProjectionLoad(t, raw, newFront)
		if !reflect.DeepEqual(f.Town.cityObjects, wantGraph) {
			t.Fatal("city graph changed through cold LOAD", cycle, f.Town.cityObjects, wantGraph)
		}
		for i, want := range wantParty {
			got := f.Carried[i]
			if !reflect.DeepEqual(mapload.MemberItemEquipment(got, f.Table), mapload.MemberItemEquipment(want, f.Table)) ||
				!reflect.DeepEqual(cityMemberStacks(got, f.Table), cityMemberStacks(want, f.Table)) || got.Book != want.Book {
				t.Fatal("current Item/Effect/Spell values changed", cycle, i, got.Carry, want.Carry, got.Book, want.Book)
			}
		}
	}
}

func TestCityObjectProjectionOrdinaryItemEffectSpellEditsWinForTwoCycles(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	raw = cityProjectionSave(t, f)
	doc, _, indices := cityProjectionWire(t, raw)
	graph := f.Town.cityObjects.Clone()
	sharedItem := graph.Roots[0].Worn[0]
	sharedEffect, sharedSpell := graph.Items[0].Effects[0], graph.Spells[0]
	var separate sim.SavedObjectID
	for _, root := range graph.Roots {
		for _, id := range root.Pack {
			if id != sharedItem {
				separate = id
				break
			}
		}
		if separate != 0 {
			break
		}
	}
	leaf, _, _ := sav.NativeActions(doc.State)
	mustSetValue(&doc.Objects[indices[sharedItem]-1], "T1C", 777)
	mustSetValue(&doc.Objects[indices[sharedItem]-1], "F4A", 11)
	mustSetValue(&doc.Objects[indices[separate]-1], "F40", 0x102)
	mustSetValue(&doc.Objects[indices[separate]-1], "F42", 4)
	mustSetValue(&doc.Objects[indices[sharedEffect]-1], "E3D", 2)
	mustSetValue(&doc.Objects[indices[sharedEffect]-1], "E40", 0x12340009)
	mustSetValue(&doc.Objects[indices[sharedSpell]-1], "S09", 13)
	mustSetValue(&doc.Objects[indices[sharedSpell]-1], "S0A", 3)
	mustSetValue(&doc.Objects[indices[sharedSpell]-1], "S0C", 513)
	var err error
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		f = cityProjectionLoad(t, raw, newFront)
		gotGraph := f.Town.cityObjects
		wantItems, wantEffects, wantSpells := cityObjectNativeIdentities(graph)
		gotItems, gotEffects, gotSpells := cityObjectNativeIdentities(gotGraph)
		if !reflect.DeepEqual(cityObjectIdentityTopology(graph), cityObjectIdentityTopology(gotGraph)) ||
			!reflect.DeepEqual(wantItems, gotItems) || !reflect.DeepEqual(wantEffects, gotEffects) || !reflect.DeepEqual(wantSpells, gotSpells) {
			t.Fatal("ordinary scalar edit changed identity graph", cycle, gotGraph, graph)
		}
		for _, member := range f.Carried {
			if member.Book.Slots[25] != (sim.BookSpell{Range: 13, Defensive: 3, ManaCost: 513}) {
				t.Fatal("shared ordinary Spell lost book view", cycle, member.Book)
			}
			weapon := mapload.MemberItemEquipment(member, f.Table)[0]
			if weapon.Price != 777 || weapon.Weight != 11 || weapon.Effects[0].Mode != 2 || weapon.Effects[0].Operand != 0x12340009 ||
				weapon.SourceEquipment.Spell.Range != 13 || weapon.SourceEquipment.Spell.Defensive != 3 || weapon.SourceEquipment.Spell.ManaCost != 513 || weapon.ObjectID != 0 {
				t.Fatal("ordinary shared Item/Effect/Spell values lost", cycle, weapon)
			}
		}
		raw = cityProjectionSave(t, f)
		written, _, mapped := cityProjectionWire(t, raw)
		for _, tc := range []struct {
			id    sim.SavedObjectID
			field string
			value uint32
		}{{sharedItem, "T1C", 777}, {sharedItem, "F4A", 11}, {separate, "F40", 0x102}, {separate, "F42", 4},
			{sharedEffect, "E3D", 2}, {sharedEffect, "E40", 0x12340009}, {sharedSpell, "S09", 13}, {sharedSpell, "S0A", 3}, {sharedSpell, "S0C", 513}} {
			value, err := savedStructureValue(&written.Objects[mapped[tc.id]-1], tc.field)
			if err != nil || value != tc.value {
				t.Fatal("ordinary mutation did not survive next SAVE", cycle, tc, value, err)
			}
		}
	}
	unchanged, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(leaf, unchanged) {
		t.Fatal("test changed native metadata along with ordinary scalars")
	}
}

func TestCityObjectProjectionMalformedBindingsRejectAtomicCityLoad(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	raw = cityProjectionSave(t, f)
	for _, tc := range []struct {
		name string
		edit func(*currentActionData)
	}{
		{"duplicate ID", func(a *currentActionData) { a.Objects[1].ID = a.Objects[0].ID }},
		{"duplicate ordinary record", func(a *currentActionData) { a.Objects[1].Object = a.Objects[0].Object }},
		{"missing ordinary record", func(a *currentActionData) { a.Objects[0].Object = 65535 }},
		{"actor as item node", func(a *currentActionData) { a.Objects[0].Object = a.Bindings[0].Object }},
		{"zero node ID", func(a *currentActionData) { a.Objects[0].ID = 0 }},
		{"occupied allocator", func(a *currentActionData) { a.Inventory.NextID = a.Objects[0].ID }},
		{"container payload", func(a *currentActionData) { a.Inventory.Containers = []currentObjectContainer{{}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, a, _ := cityProjectionWire(t, raw)
			tc.edit(a)
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			beforeParty, beforeTown, beforeGraph := mapload.CloneParty(f.Carried), f.Town, f.Town.cityObjects.Clone()
			if _, _, err := f.RestoreOriginal(bad); err == nil {
				t.Fatal("malformed city topology installed")
			}
			if f.Town != beforeTown || !reflect.DeepEqual(f.Carried, beforeParty) || !reflect.DeepEqual(f.Town.cityObjects, beforeGraph) {
				t.Fatal("failed LOAD changed active city")
			}
		})
	}
}

func TestCityObjectProjectionChecksAllSharedCurrentViewsBeforeRecords(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	for _, kind := range []string{"Item", "Effect", "Spell"} {
		t.Run(kind, func(t *testing.T) {
			party := mapload.CloneParty(f.Carried)
			switch kind {
			case "Item":
				party[0].Carry.EquippedItems[0].Price++
			case "Effect":
				// This node is distinct from the worn Item but shares its child.
				party[0].Carry.OrderedStacks[1].Effects[0].Operand++
			case "Spell":
				party[0].Book.Slots[25].ManaCost++
			}
			before := f.Town.cityObjects.Clone()
			if p, err := newCityObjectProjection(f.Town.cityObjects, party, f.Table); err == nil || p != nil {
				t.Fatal("conflicting shared scalar views were arbitrated", p, err)
			}
			if !reflect.DeepEqual(f.Town.cityObjects, before) {
				t.Fatal("failed preflight changed graph")
			}
		})
	}
	// A newly added equal-valued item has its own explicit location and ID.
	party := mapload.CloneParty(f.Carried)
	added := party[0].Carry.OrderedStacks[1].Clone()
	added.ObjectID = 0
	stacks := append(slices.Clone(party[0].Carry.OrderedStacks), added)
	if err := mapload.UpdatePartyLoadOrdered(party[0], &party[0], f.Table, false, false, stacks); err != nil {
		t.Fatal(err)
	}
	p, err := newCityObjectProjection(f.Town.cityObjects, party, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range p.graph.Roots {
		if string(root.PartyID) == party[0].ID && (len(root.Pack) != 3 || slices.Contains(root.Pack[:2], root.Pack[2])) {
			t.Fatal("new equal item borrowed an old identity", root)
		}
	}
}

func TestCityObjectProjectionOrdinaryReferenceEditsRetainNodeIdentities(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	raw = cityProjectionSave(t, f)
	doc, a, indices := cityProjectionWire(t, raw)
	want := f.Town.cityObjects.Clone()
	root := &want.Roots[0]
	var actor uint16
	for _, member := range a.Party {
		if string(member.ID) != string(root.PartyID) {
			continue
		}
		for _, binding := range a.Bindings {
			if !binding.Structure && binding.ID == member.Entity {
				actor = binding.Object
			}
		}
	}
	if actor == 0 || len(root.Pack) != 2 || root.Pack[0] == root.Pack[1] {
		t.Fatal("no distinct ordered item roots")
	}
	slices.Reverse(root.Pack)
	mustSetRefs(&doc.Objects[actor-1], "Inventory", []uint16{indices[root.Pack[0]], indices[root.Pack[1]]})
	shared := root.Worn[0]
	for i := range want.Items {
		if want.Items[i].ID == shared {
			want.Items[i].Effects = slices.Clone(want.Effects)
			record := want.ItemRecords[shared]
			record.Effects = slices.Clone(want.Items[i].Effects)
			want.ItemRecords[shared] = record
			var refs []uint16
			for _, id := range want.Items[i].Effects {
				refs = append(refs, indices[id])
			}
			mustSetRefs(&doc.Objects[indices[shared]-1], "Effects", refs)
		}
	}
	doc, _, err := sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		f = cityProjectionLoad(t, raw, newFront)
		if !reflect.DeepEqual(want, f.Town.cityObjects) {
			t.Fatal("ordinary reference edit lost IDs or was replayed from metadata", cycle, f.Town.cityObjects, want)
		}
		raw = cityProjectionSave(t, f)
	}
}

func TestCityObjectProjectionCurrentNativeIDsSurviveTwoCycles(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	roots := map[string]cityPartyObjectRoots{}
	for _, root := range f.Town.cityObjects.Roots {
		roots[string(root.PartyID)] = root
	}
	for i := range f.Carried {
		member := &f.Carried[i]
		root := roots[member.ID]
		for slot, id := range root.Worn {
			member.Carry.EquippedItems[slot].ObjectID = id
			member.WornItems[slot].ObjectID = id
		}
		var items []sim.ItemInstance
		for index, id := range root.Pack {
			member.Carry.OrderedStacks[index].ObjectID = id
			v := member.Carry.OrderedStacks[index]
			for range v.Count {
				items = append(items, v.Instance())
			}
		}
		member.Carry.ItemInstances, member.CarriedItems = cloneOriginalItems(items), cloneOriginalItems(items)
	}
	wantGraph, wantParty := f.Town.cityObjects.Clone(), mapload.CloneParty(f.Carried)
	for cycle := 0; cycle < 2; cycle++ {
		raw = cityProjectionSave(t, f)
		_, a, indices := cityProjectionWire(t, raw)
		for _, row := range a.Ownership {
			if row.ID == 0 || indices[row.ID] != row.Object {
				t.Fatal("current native ID lost its ordinary object", row, indices)
			}
		}
		f = cityProjectionLoad(t, raw, newFront)
		if !reflect.DeepEqual(wantGraph, f.Town.cityObjects) {
			t.Fatal("native handles altered the topology namespace", cycle)
		}
		for i, member := range f.Carried {
			if !reflect.DeepEqual(member.Carry.EquippedItems, wantParty[i].Carry.EquippedItems) ||
				!reflect.DeepEqual(member.Carry.OrderedStacks, wantParty[i].Carry.OrderedStacks) {
				t.Fatal("current Item identity/value changed", cycle, i)
			}
		}
	}
}
