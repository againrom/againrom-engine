package game

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func cityTopologyDocument() (*sav.DocumentData, []cityActorObjectBinding, []currentOwnedObject) {
	doc := &sav.DocumentData{Version: sav.DocumentDataVersion, Objects: []sav.DocumentRecordData{
		mustNewRecord("Human", "HasInventory", "HasSpellbook"),
		mustNewRecord("Unit", "HasInventory", "HasSpellbook"),
		mustNewRecord("Weapon"), mustNewRecord("Weapon"),
		mustNewRecord("Spell"), mustNewRecord("Spell"), mustNewRecord("Armor"),
		mustNewRecord("Effect"), mustNewRecord("Effect"),
		mustNewRecord("Item"), mustNewRecord("Spell"), mustNewRecord("Sack"),
	}}
	mustSetRefs(&doc.Objects[0], "Inventory", []uint16{3, 4, 0, 3})
	mustSetRefs(&doc.Objects[0], "HeldWeapon", []uint16{3})
	worn := make([]uint16, sim.EquipSlots)
	worn[2] = 7
	mustSetRefs(&doc.Objects[0], "Worn", worn)
	mustSetRefs(&doc.Objects[0], "Spells", []uint16{5, 0, 6, 5})
	mustSetRefs(&doc.Objects[1], "Inventory", []uint16{4})
	mustSetRefs(&doc.Objects[1], "HeldWeapon", []uint16{4})
	mustSetRefs(&doc.Objects[1], "Spells", []uint16{5})
	for _, i := range []int{2, 3} {
		mustSetValue(&doc.Objects[i], "F42", 5)
		mustSetRefs(&doc.Objects[i], "Effects", []uint16{8, 8, 0, 9})
		mustSetRefs(&doc.Objects[i], "WeaponSpell", []uint16{5})
	}
	mustSetValue(&doc.Objects[6], "F42", 9)
	mustSetValue(&doc.Objects[9], "F42", 1)
	mustSetRefs(&doc.Objects[6], "Effects", []uint16{8})
	actors := []cityActorObjectBinding{{Object: 2, PartyID: string([]byte{0xff, 'b'})}, {Object: 1, PartyID: "alpha"}}
	owned := []currentOwnedObject{
		{Object: 3, ID: 10, Kind: 1}, {Object: 4, Kind: 1},
		{Object: 5, ID: 20, Kind: 3}, {Object: 7, Kind: 1}, {Object: 8, ID: 30, Kind: 2},
		{Object: 12, ID: 900, Kind: 4}, {ID: 1000, Kind: 1},
	}
	return doc, actors, owned
}

func mustCaptureCityTopology(t *testing.T) (*cityObjectTopology, map[uint16]sim.SavedObjectID) {
	t.Helper()
	doc, actors, owned := cityTopologyDocument()
	g, indices, err := captureCityObjectTopology(doc, actors, owned, 3)
	if err != nil {
		t.Fatal(err)
	}
	return g, indices
}

func TestCityObjectTopologyCapturesOrdinaryAliasesAndSeparateEqualNodes(t *testing.T) {
	doc, actors, owned := cityTopologyDocument()
	before, _, _ := cityTopologyDocument()
	g, indices, err := captureCityObjectTopology(doc, actors, owned, 3)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := map[uint16]sim.SavedObjectID{3: 10, 4: 1001, 5: 20, 6: 1002, 7: 1003, 8: 30, 9: 1004}
	if !reflect.DeepEqual(indices, wantIDs) || g.NextID != 1005 {
		t.Fatal("bindings or allocator lost the full reserved population", indices, g.NextID)
	}
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("capture mutated ordinary values or references")
	}
	if !reflect.DeepEqual(g.Items, []cityItemTopology{
		{ID: 10, Effects: []sim.SavedObjectID{30, 30, 0, 1004}, Spell: 20},
		{ID: 1001, Effects: []sim.SavedObjectID{30, 30, 0, 1004}, Spell: 20},
		{ID: 1003, Effects: []sim.SavedObjectID{30}, WornCount: 9},
	}) || !slices.Equal(g.Effects, []sim.SavedObjectID{30, 1004}) || !slices.Equal(g.Spells, []sim.SavedObjectID{20, 1002}) {
		t.Fatal("shared/repeated children or distinct equal nodes were folded", g)
	}
	if string(g.Roots[0].PartyID) != "alpha" || !slices.Equal(g.Roots[0].Pack, []sim.SavedObjectID{10, 1001, 0, 10}) ||
		g.Roots[0].Worn[0] != 10 || g.Roots[0].Worn[2] != 1003 || g.Roots[1].Worn[0] != 1001 ||
		!slices.Equal(g.Roots[1].Pack, []sim.SavedObjectID{1001}) || string(g.Roots[1].PartyID) != actors[0].PartyID {
		t.Fatal("party locations, nulls, aliases or byte identity changed", g.Roots)
	}
	if g.Books[0].Slots[0] != 20 || g.Books[0].Slots[1] != 0 || g.Books[0].Slots[2] != 1002 ||
		g.Books[0].Slots[3] != 20 || g.Books[1].Slots[0] != 20 {
		t.Fatal("book and weapon Spell alias was split", g.Books)
	}
	// Ordinary values cannot create, merge or repair topology nodes. Quantity
	// without a Pack view remains an explicit current Worn operand.
	for i := range doc.Objects {
		for k := range doc.Objects[i].Values {
			doc.Objects[i].Values[k].Value = uint32(500 + i + k)
		}
	}
	mustSetValue(&doc.Objects[6], "F42", 17)
	wantChanged := g.Clone()
	wantChanged.Items[2].WornCount = 17
	slices.Reverse(actors)
	slices.Reverse(owned)
	again, againIDs, err := captureCityObjectTopology(doc, actors, owned, 3)
	if err != nil || !reflect.DeepEqual(wantChanged, again) || !reflect.DeepEqual(indices, againIDs) {
		t.Fatal("ordinary edits changed topology or lost current Worn quantity", err, again)
	}
	// Fresh capture uses the exact ordinary edge order, including repeated nodes.
	mustSetRefs(&doc.Objects[2], "Effects", []uint16{9, 8, 8, 0})
	changed, _, err := captureCityObjectTopology(doc, actors, owned, 3)
	if err != nil || !slices.Equal(changed.Items[0].Effects, []sim.SavedObjectID{1004, 30, 30, 0}) {
		t.Fatal("ordinary child references lost authority", err, changed)
	}
}

func TestCityObjectTopologyCaptureReservesAllocatorFloor(t *testing.T) {
	doc, actors, owned := cityTopologyDocument()
	g, indices, err := captureCityObjectTopology(doc, actors, owned, 2000)
	if err != nil || indices[4] != 2000 || g.NextID != 2004 {
		t.Fatal("retired allocator space was reused", err, indices, g)
	}
	// Two unbound records with equal values remain two concrete nodes.
	g, indices, err = captureCityObjectTopology(doc, actors, nil, 0)
	if err != nil || indices[3] == indices[4] || indices[5] == indices[6] || indices[8] == indices[9] || g.NextID != 8 {
		t.Fatal("zero/unbound identity collapsed ordinary nodes", err, indices, g)
	}
}

func TestCityObjectTopologyCloneAndPersistenceDetachEverySlice(t *testing.T) {
	g, _ := mustCaptureCityTopology(t)
	want := g.Clone()
	clone := g.Clone()
	clone.Items[0].ID = 77
	clone.Items[0].Effects[0] = 78
	clone.Items[2].WornCount = 17
	clone.Effects[0] = 79
	clone.Spells[0] = 80
	clone.Roots[0].PartyID[0] = 'x'
	clone.Roots[0].Pack[0] = 81
	clone.Roots[0].Worn[0] = 82
	clone.Books[0].PartyID[0] = 'y'
	clone.Books[0].Slots[0] = 83
	if !reflect.DeepEqual(g, want) {
		t.Fatal("cancelled clone mutated live topology")
	}
	for _, format := range []string{"gob", "json"} {
		t.Run(format, func(t *testing.T) {
			var raw bytes.Buffer
			var restored cityObjectTopology
			if format == "gob" {
				if err := gob.NewEncoder(&raw).Encode(g); err != nil {
					t.Fatal(err)
				}
				if err := gob.NewDecoder(&raw).Decode(&restored); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := json.NewEncoder(&raw).Encode(g); err != nil {
					t.Fatal(err)
				}
				if err := json.NewDecoder(&raw).Decode(&restored); err != nil {
					t.Fatal(err)
				}
			}
			if err := restored.Validate(); err != nil || !reflect.DeepEqual(g, &restored) {
				t.Fatal("topology persistence lost aliases or party bytes", err, restored)
			}
		})
	}
	var absent *cityObjectTopology
	if absent.Clone() != nil || absent.Validate() != nil {
		t.Fatal("absent topology acquired a graph")
	}
}

func TestCityObjectTopologyRejectsMalformedDetachedGraph(t *testing.T) {
	tests := []struct {
		name string
		edit func(*cityObjectTopology)
	}{
		{"version", func(g *cityObjectTopology) { g.Version++ }},
		{"allocator zero", func(g *cityObjectTopology) { g.NextID = 0 }},
		{"allocator occupied", func(g *cityObjectTopology) { g.NextID = g.Items[0].ID }},
		{"zero identity", func(g *cityObjectTopology) { g.Items[0].ID = 0 }},
		{"duplicate item", func(g *cityObjectTopology) { g.Items = append(g.Items, g.Items[0]) }},
		{"duplicate child", func(g *cityObjectTopology) { g.Effects = append(g.Effects, g.Effects[0]) }},
		{"kind collision", func(g *cityObjectTopology) { g.Spells[0] = g.Items[0].ID }},
		{"missing Effect", func(g *cityObjectTopology) { g.Items[0].Effects[0] = 999 }},
		{"wrong Effect kind", func(g *cityObjectTopology) { g.Items[0].Effects[0] = g.Spells[0] }},
		{"wrong Spell kind", func(g *cityObjectTopology) { g.Items[0].Spell = g.Effects[0] }},
		{"wrong pack kind", func(g *cityObjectTopology) { g.Roots[0].Pack[0] = g.Effects[0] }},
		{"missing worn", func(g *cityObjectTopology) { g.Roots[0].Worn[4] = 999 }},
		{"empty party", func(g *cityObjectTopology) { g.Roots[0].PartyID = nil }},
		{"repeated roots", func(g *cityObjectTopology) { g.Roots = append(g.Roots, g.Roots[0]) }},
		{"repeated book", func(g *cityObjectTopology) { g.Books = append(g.Books, g.Books[0]) }},
		{"unbound book", func(g *cityObjectTopology) { g.Books[0].PartyID = []byte("missing") }},
		{"wrong book kind", func(g *cityObjectTopology) { g.Books[0].Slots[0] = g.Items[0].ID }},
		{"too many nodes", func(g *cityObjectTopology) { g.Spells = make([]sim.SavedObjectID, sim.MaxSavedObjects+1) }},
		{"too many edges", func(g *cityObjectTopology) { g.Items[0].Effects = make([]sim.SavedObjectID, sim.MaxSavedObjects) }},
	}
	g, _ := mustCaptureCityTopology(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bad := g.Clone()
			tc.edit(bad)
			before := bad.Clone()
			if err := bad.Validate(); err == nil {
				t.Fatal("malformed topology accepted")
			}
			if !reflect.DeepEqual(bad, before) {
				t.Fatal("validation repaired its input")
			}
		})
	}
	partyBytes := (64 << 20) - 1
	if err := cityTopologyPartyBytes(&partyBytes, []byte("ab")); err == nil {
		t.Fatal("party byte budget was not bounded")
	}
}

func TestCityObjectTopologyAllocatorRejectsFailureWithoutMutation(t *testing.T) {
	g, _ := mustCaptureCityTopology(t)
	first := g.NextID
	for offset := sim.SavedObjectID(0); offset < 3; offset++ {
		id, err := g.allocateID()
		if err != nil || id != first+offset || g.NextID != id+1 {
			t.Fatal("allocator reused or skipped identity", err, id, g.NextID)
		}
	}
	g.NextID = ^sim.SavedObjectID(0)
	want := g.Clone()
	if id, err := g.allocateID(); err == nil || id != 0 || !reflect.DeepEqual(g, want) {
		t.Fatal("exhaustion mutated graph", id, err)
	}
	g.NextID = g.Items[0].ID
	want = g.Clone()
	if id, err := g.allocateID(); err == nil || id != 0 || !reflect.DeepEqual(g, want) {
		t.Fatal("malformed graph acquired identity", id, err)
	}
	var absent *cityObjectTopology
	if _, err := absent.allocateID(); err == nil {
		t.Fatal("absent graph allocated an identity")
	}
}

func TestCityObjectTopologyRejectsMalformedOrdinaryCaptureAtomically(t *testing.T) {
	tests := []struct {
		name string
		edit func(*sav.DocumentData, *[]cityActorObjectBinding, *[]currentOwnedObject)
	}{
		{"version", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) { d.Version++ }},
		{"absent actor", func(_ *sav.DocumentData, a *[]cityActorObjectBinding, _ *[]currentOwnedObject) { (*a)[0].Object = 0 }},
		{"wrong actor", func(_ *sav.DocumentData, a *[]cityActorObjectBinding, _ *[]currentOwnedObject) { (*a)[0].Object = 3 }},
		{"repeated actor", func(_ *sav.DocumentData, a *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			(*a)[0].Object = (*a)[1].Object
		}},
		{"repeated party", func(_ *sav.DocumentData, a *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			(*a)[0].PartyID = (*a)[1].PartyID
		}},
		{"missing child", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			mustSetRefs(&d.Objects[2], "Effects", []uint16{65535})
		}},
		{"zero quantity", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			mustSetValue(&d.Objects[6], "F42", 0)
		}},
		{"missing quantity", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			for i, value := range d.Objects[6].Values {
				if value.Name == "F42" {
					d.Objects[6].Values = slices.Delete(d.Objects[6].Values, i, i+1)
					break
				}
			}
		}},
		{"wrong child", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			mustSetRefs(&d.Objects[2], "WeaponSpell", []uint16{8})
		}},
		{"repeated site", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			d.Objects[2].RefSlots = append(d.Objects[2].RefSlots, sav.DocumentRefsData{Name: "Effects"})
		}},
		{"wide book", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			mustSetRefs(&d.Objects[0], "Spells", make([]uint16, 29))
		}},
		{"held overlap", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			refs, _ := savedObjectRefs(&d.Objects[0], "Worn")
			refs[0] = 3
		}},
		{"short worn", func(d *sav.DocumentData, _ *[]cityActorObjectBinding, _ *[]currentOwnedObject) {
			mustSetRefs(&d.Objects[0], "Worn", []uint16{0})
		}},
		{"repeated ID", func(_ *sav.DocumentData, _ *[]cityActorObjectBinding, o *[]currentOwnedObject) {
			(*o)[1].ID = (*o)[0].ID
		}},
		{"repeated record", func(_ *sav.DocumentData, _ *[]cityActorObjectBinding, o *[]currentOwnedObject) {
			*o = append(*o, (*o)[1])
		}},
		{"wrong bound kind", func(_ *sav.DocumentData, _ *[]cityActorObjectBinding, o *[]currentOwnedObject) { (*o)[0].Kind = 3 }},
		{"absent bound record", func(_ *sav.DocumentData, _ *[]cityActorObjectBinding, o *[]currentOwnedObject) {
			(*o)[0].Object = 65535
		}},
		{"reserved maximum", func(_ *sav.DocumentData, _ *[]cityActorObjectBinding, o *[]currentOwnedObject) {
			(*o)[0].ID = ^sim.SavedObjectID(0)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, actors, owned := cityTopologyDocument()
			tc.edit(doc, &actors, &owned)
			before, _ := json.Marshal([]any{doc, actors, owned})
			g, indices, err := captureCityObjectTopology(doc, actors, owned, 0)
			if err == nil || g != nil || indices != nil {
				t.Fatal("malformed capture returned a partial graph", err, g, indices)
			}
			after, _ := json.Marshal([]any{doc, actors, owned})
			if !bytes.Equal(before, after) {
				t.Fatal("failed capture mutated input")
			}
		})
	}
	doc, actors, owned := cityTopologyDocument()
	if g, indices, err := captureCityObjectTopology(doc, actors, owned, ^sim.SavedObjectID(0)); err == nil || g != nil || indices != nil {
		t.Fatal("exhausted allocation returned a partial graph", err, g, indices)
	}
}

func cityTopologyRegistry() (*sim.SavedObjects, []cityActorEntityBinding) {
	pack := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 10}
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: 150,
		Items: []sim.SavedItemObject{
			{ID: 1, Value: sim.ItemStack{Count: 5}, Effects: []sim.SavedObjectID{4, 4}, Spell: 6},
			{ID: 2, Value: sim.ItemStack{Count: 5}, Effects: []sim.SavedObjectID{4}, Spell: 6},
			{ID: 3, Value: sim.ItemStack{Count: 9}, Effects: []sim.SavedObjectID{4}},
			{ID: 9, Value: sim.ItemStack{Count: 2}, Effects: []sim.SavedObjectID{4, 5}},
		},
		ItemRoots:  []sim.SavedItemRoot{{ID: 3, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 20, Slot: 4}}, {ID: 9, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 99, Slot: 4}}},
		Effects:    []sim.SavedEffectObject{{ID: 4}, {ID: 5}},
		Spells:     []sim.SavedSpellObject{{ID: 6}, {ID: 7}, {ID: 8, Retired: true}},
		Sacks:      []sim.SavedSackObject{{ID: 100}},
		Containers: []sim.SavedObjectContainer{{Owner: pack, Present: true, Items: []sim.SavedObjectID{2, 1, 0, 2}}},
	}
	actors := []cityActorEntityBinding{
		{Entity: 20, PartyID: "joined", Books: [28]sim.SavedObjectID{6}},
		{Entity: 10, PartyID: "hero", Books: [28]sim.SavedObjectID{6, 0, 7}},
	}
	return r, actors
}

func TestCityObjectTopologyCapturesSurvivorRegistryAndExplicitBooks(t *testing.T) {
	r, actors := cityTopologyRegistry()
	before := r.Clone()
	g, err := captureCityObjectTopologyFromSaved(r, actors)
	if err != nil {
		t.Fatal(err)
	}
	if g.NextID != 150 || !slices.Equal(g.Effects, []sim.SavedObjectID{4}) || !slices.Equal(g.Spells, []sim.SavedObjectID{6, 7}) ||
		!reflect.DeepEqual(g.Items, []cityItemTopology{{ID: 1, Effects: []sim.SavedObjectID{4, 4}, Spell: 6}, {ID: 2, Effects: []sim.SavedObjectID{4}, Spell: 6}, {ID: 3, Effects: []sim.SavedObjectID{4}, WornCount: 9}}) {
		t.Fatal("survivor graph lost shared children or retained dropped-only nodes", g)
	}
	if string(g.Roots[0].PartyID) != "hero" || !slices.Equal(g.Roots[0].Pack, []sim.SavedObjectID{2, 1, 0, 2}) ||
		string(g.Roots[1].PartyID) != "joined" || g.Roots[1].Worn[3] != 3 ||
		g.Books[0].Slots[0] != 6 || g.Books[0].Slots[2] != 7 || g.Books[1].Slots[0] != 6 {
		t.Fatal("explicit pack/worn/book locations changed", g.Roots, g.Books)
	}
	if !reflect.DeepEqual(r, before) {
		t.Fatal("capture changed the live registry")
	}
	for i := range r.Items {
		r.Items[i].Value = sim.ItemStack{Code: uint16(300 + i), Count: 17, Price: -53}
		r.Items[i].Token.RuntimeID = uint32(100 + i)
	}
	for i := range r.Effects {
		r.Effects[i].Value = sim.ItemEffect{Kind: 3, Operand: uint32(i + 57)}
	}
	for i := range r.Spells {
		r.Spells[i].Value = sim.SourceItemSpell{Present: true, ID: uint8(i + 1), ManaCost: 500}
	}
	slices.Reverse(actors)
	wantChanged := g.Clone()
	wantChanged.Items[2].WornCount = 17
	again, err := captureCityObjectTopologyFromSaved(r, actors)
	if err != nil || again.NextID != wantChanged.NextID || !reflect.DeepEqual(again.Items, wantChanged.Items) ||
		!reflect.DeepEqual(again.Roots, wantChanged.Roots) || !reflect.DeepEqual(again.Books, wantChanged.Books) ||
		!slices.Equal(again.Effects, wantChanged.Effects) || !slices.Equal(again.Spells, wantChanged.Spells) {
		t.Fatal("registry edits changed topology or lost current Worn quantity", err, again)
	}
	if again.ItemRecords[1].Value.Code != 300 || again.ItemRecords[1].Token.RuntimeID != 100 ||
		again.EffectRecords[4].Value.Operand != 57 || again.SpellRecords[6].Value.ManaCost != 500 {
		t.Fatal("registry edits lost current record supplement", again.ItemRecords, again.EffectRecords, again.SpellRecords)
	}
	g.Items[0].Effects[0] = 999
	g.Roots[0].Pack[0] = 999
	if r.Items[0].Effects[0] != 4 || r.Containers[0].Items[0] != 2 {
		t.Fatal("returned graph shares a registry slice")
	}
}

func TestCityObjectTopologyRejectsMalformedRegistryCaptureAtomically(t *testing.T) {
	tests := []struct {
		name string
		edit func(*sim.SavedObjects, *[]cityActorEntityBinding)
	}{
		{"invalid allocator", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.NextID = 100 }},
		{"kind collision outside selection", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Sacks[0].ID = 8 }},
		{"duplicate entity", func(_ *sim.SavedObjects, a *[]cityActorEntityBinding) { (*a)[1].Entity = (*a)[0].Entity }},
		{"duplicate party", func(_ *sim.SavedObjects, a *[]cityActorEntityBinding) { (*a)[1].PartyID = (*a)[0].PartyID }},
		{"wrong pack owner", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Items[0].Owner.Entity++ }},
		{"missing pack item", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Containers[0].Items[0] = 75 }},
		{"missing pack edge", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Containers[0].Items[1] = 0 }},
		{"duplicate pack", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) {
			r.Containers = append(r.Containers, r.Containers[0])
		}},
		{"absent populated pack", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Containers[0].Present = false }},
		{"invalid worn slot", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.ItemRoots[0].Owner.Slot = sim.EquipSlots + 1 }},
		{"duplicate worn slot", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.ItemRoots[1].Owner = r.ItemRoots[0].Owner }},
		{"retired child", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Effects[0].Retired = true }},
		{"wrong child kind", func(r *sim.SavedObjects, _ *[]cityActorEntityBinding) { r.Items[0].Effects[0] = 6 }},
		{"missing book", func(_ *sim.SavedObjects, a *[]cityActorEntityBinding) { (*a)[0].Books[0] = 75 }},
		{"wrong book kind", func(_ *sim.SavedObjects, a *[]cityActorEntityBinding) { (*a)[0].Books[0] = 4 }},
		{"retired book", func(_ *sim.SavedObjects, a *[]cityActorEntityBinding) { (*a)[0].Books[0] = 8 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, actors := cityTopologyRegistry()
			tc.edit(r, &actors)
			before := r.Clone()
			g, err := captureCityObjectTopologyFromSaved(r, actors)
			if err == nil || g != nil {
				t.Fatal("malformed registry returned partial topology", err, g)
			}
			if !reflect.DeepEqual(r, before) {
				t.Fatal("failed capture changed live objects")
			}
		})
	}
}
