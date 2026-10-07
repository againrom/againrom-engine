package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCitySpellConstructorReservesOrdinaryAndRetainedKeys(t *testing.T) {
	value := sim.SourceItemSpell{Present: true, ID: 1, Range: 5, ManaCost: 7}
	p := &cityObjectProjection{
		graph: &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 7, Spells: []sim.SavedObjectID{4, 5, 6},
			ItemRecords:   map[sim.SavedObjectID]sim.SavedItemObject{1: {ID: 1, Token: sim.SavedObjectToken{Identity: 0x01000001}}},
			EffectRecords: map[sim.SavedObjectID]sim.SavedEffectObject{2: {ID: 2, Token: sim.SavedObjectToken{Identity: 0x01000002}}},
			SpellRecords:  map[sim.SavedObjectID]sim.SavedSpellObject{4: {ID: 4, This: 0x01000003, Value: value}},
			Books:         []cityBookTopology{{PartyID: []byte("first"), Slots: [28]sim.SavedObjectID{4, 5}}, {PartyID: []byte("second"), Slots: [28]sim.SavedObjectID{4, 6}}}},
		spells: map[sim.SavedObjectID]sim.SourceItemSpell{4: value, 5: value, 6: value},
	}
	raw := make([]byte, 9)
	binary.LittleEndian.PutUint32(raw[1:], 0x01000004)
	namespace := sav.DocumentData{Objects: []sav.DocumentRecordData{{
		Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x01000000}},
		Raw:    []sav.DocumentRawData{{Name: "unresolved", Bytes: raw}},
	}}}
	books := p.graph.Clone().Books
	if err := p.constructSpellRecords(namespace); err != nil {
		t.Fatal(err)
	}
	if p.graph.SpellRecords[4].This != 0x01000003 || !reflect.DeepEqual(p.graph.Books, books) {
		t.Fatal("constructor changed retained identity or shared book edges")
	}
	seen := map[uint32]bool{}
	for _, id := range []sim.SavedObjectID{5, 6} {
		row := p.graph.SpellRecords[id]
		if row.ID != id || row.Value != value || row.This <= 0x01000004 || seen[row.This] {
			t.Fatal("new equal-valued Spells alias or collide with ordinary/current graph keys", row)
		}
		seen[row.This] = true
	}
	before := p.graph.Clone()
	if err := p.constructSpellRecords(namespace); err != nil || !reflect.DeepEqual(p.graph, before) {
		t.Fatal("second constructor changed current identities", err)
	}
	reserved, err := reserveCurrentCityDocumentKeys(namespace, p.graph, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range reserved {
		if seen[key] || key <= 0x01000004 {
			t.Fatal("ordinary writer allocated a retained or constructed Spell key", key)
		}
	}
}

func TestCitySpellNamespaceConstructionLeavesCurrentStateUntouched(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	party, graph, source := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.originalCity.snapshot()
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	first, firstOrder, err := f.currentCityBase(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, secondOrder, err := f.currentCityBase(snapshot)
	if err != nil || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstOrder, secondOrder) {
		t.Fatal("current namespace construction is not deterministic", err)
	}
	if !reflect.DeepEqual(f.Carried, party) || !reflect.DeepEqual(f.Town.cityObjects, graph) || !reflect.DeepEqual(f.originalCity.snapshot(), source) {
		t.Fatal("current namespace construction mutated the party, city graph or retained source")
	}
}
