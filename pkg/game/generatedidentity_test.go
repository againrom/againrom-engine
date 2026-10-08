package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentKeyReservationKeepsExistingKeysAndRuntimeIDs(t *testing.T) {
	b := generatedDocumentBuilder{reservedKeys: []uint32{16, 32, 48}, currentKeys: map[uint32]bool{16: true}, runtimeIDs: map[uint32]bool{7: true}}
	b.reserveCurrentForm(binary.LittleEndian.AppendUint32(nil, 32))
	if !b.runtimeIDs[7] || !b.runtimeIDs[32] {
		t.Fatal("current reservation dropped an existing runtime ID")
	}
	if got := b.identity(); got != 48 {
		t.Fatalf("reserved identity = %d, want 48 after existing16 and current32", got)
	}
	if got := b.identity(); got != 64 {
		t.Fatalf("fallback identity = %d, want 64 after emitted48", got)
	}
}

func TestCurrentIdentityFallbackKeepsExistingDocumentKeys(t *testing.T) {
	actor, spell := mustNewRecord("Human"), mustNewRecord("Spell")
	mustSetValue(&actor, "Identity", 16)
	mustSetValue(&spell, "This", 32)
	b := generatedDocumentBuilder{doc: sav.DocumentData{Objects: []sav.DocumentRecordData{actor, spell}},
		reservedKeys: []uint32{16, 32, 64}, currentKeys: map[uint32]bool{64: true}}
	if got := b.identity(); got != 48 {
		t.Fatalf("fallback identity = %d, want 48 after document16/32 and current64", got)
	}
}

func TestCurrentNativeItemKeyReservationSeparatesEffectAndExistingRecord(t *testing.T) {
	item := sim.PlainItem(0x0e01)
	item.Price, item.WeightPresent, item.Weight = 17, true, 7
	item.Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}}
	item.NativeRecord = &sim.NativeItemRecord{Token: sim.SavedObjectToken{Identity: 32, RuntimeID: 19, Reference: 23}, F45: 73, F47: 91, F48: 0x9876}
	item.NativeRecord.Token.Position[0], item.NativeRecord.Token.Position[11] = 0xa5, 0x5a
	w, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{item}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, existing := range []bool{false, true} {
		name := "fresh"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			b := generatedDocumentBuilder{}
			if existing {
				b.doc.Objects = []sav.DocumentRecordData{savedCurrentEffectRecord(sim.SavedEffectObject{Token: sim.SavedObjectToken{Identity: 16, RuntimeID: 11}, Value: sim.ItemEffect{Kind: 8, Operand: 1}})}
				b.reservedKeys, err = sav.ReserveDocumentKeys(b.doc, 16)
				if err != nil {
					t.Fatal(err)
				}
			}
			before := w.Hash()
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			b.reserveCurrentForm(raw)
			index, err := b.item(item, 1, 23)
			if err != nil {
				t.Fatal(err)
			}
			row, err := savedItemRecord(&b.doc, index)
			want := item.NativeRecord.Token
			want.T1C = uint32(item.Price)
			if err != nil || row.Token != want || row.F45 != 73 || row.F47 != 91 || row.F48 != 0x9876 || row.Value.Code != item.Code || row.Value.Kind != item.Kind || row.Value.Price != 17 || row.Value.Count != 1 {
				t.Fatal("native current item fields changed during reserved emission", row, err)
			}
			seen := map[uint32]bool{}
			for _, record := range b.doc.Objects {
				key, err := savedStructureValue(&record, "Identity")
				if err != nil || key == 0 || seen[key] {
					t.Fatal("native item, generated effect and existing record share a key", key, err)
				}
				seen[key] = true
			}
			if !seen[32] || existing && !seen[16] || w.Hash() != before {
				t.Fatal("emission reminted retained identity or changed the current World")
			}
		})
	}
}

func TestCurrentItemGraphReservesUnregisteredItemBeforeEffectMint(t *testing.T) {
	for _, holder := range []string{"legacy_pack", "ordered_pack", "worn", "sack"} {
		t.Run(holder, func(t *testing.T) {
			actor := mustNewRecord("Human", "HasInventory")
			mustSetValue(&actor, "Identity", 16)
			mustSetValue(&actor, "Reference", 23)
			player, group := mustNewRecord("Player"), newSavedGroupRecord()
			mustSetValue(&player, "This", 64)
			mustSetRefs(&group, "Actors", []uint16{1})
			mustSetCount(&group, "Actors", 1)
			player.Groups = []sav.DocumentRecordData{group}
			mustSetCount(&player, "Groups", 1)
			mustSetCount(&player, "Actors", 1)
			application, _, _ := actorProjectionFixture(t, "Human")
			doc := sav.DocumentData{Version: sav.DocumentDataVersion, FileVersion: sav.MinVersion, Marker: sav.GeneratedCityMarker,
				Objects: []sav.DocumentRecordData{actor, player}, Players: []uint16{2}, World: &sav.DocumentWorldData{TerrainIdentity: 11},
				State: application.State, Campaign: application.Campaign, Trailer: application.Trailer}
			keys, err := sav.ReserveDocumentKeys(doc, 3)
			if err != nil {
				t.Fatal(err)
			}
			key := keys[1]
			if holder == "sack" {
				key = keys[2]
			}
			item := sim.PlainItem(0x0e01)
			item.Price, item.WeightPresent, item.Weight = 17, true, 7
			item.Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}}
			item.NativeRecord = &sim.NativeItemRecord{Token: sim.SavedObjectToken{Identity: key, RuntimeID: 19, Reference: 23}, F47: 91}
			stock := sim.Stock{ID: 7}
			var sacks []sim.Sack
			switch holder {
			case "legacy_pack":
				stock.ItemInstances = []sim.ItemInstance{item}
			case "ordered_pack":
				stock.ItemInstances = []sim.ItemInstance{item}
				stock.OrderedStacks = []sim.ItemStack{sim.StackItem(item, 1)}
			case "worn":
				stock.EquippedItems[0] = item
			case "sack":
				sacks = []sim.Sack{{X: 3, Y: 3, ItemInstances: []sim.ItemInstance{item}}}
			}
			w, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
				[]sim.Entity{{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, sacks, []sim.Stock{stock})
			if err != nil {
				t.Fatal(err)
			}
			if holder == "legacy_pack" || holder == "ordered_pack" {
				current := w.Stock()
				if len(current) != 1 || len(current[0].OrderedStacks) != 1 || current[0].OrderedStacks[0].NativeRecord == nil || current[0].OrderedStacks[0].NativeRecord.Token.Identity != key {
					t.Fatal("current Stock did not expose legacy/ordered native keys")
				}
			}
			state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc, Actors: []SnapshotSAVActor{{EntityID: 7, ObjectIndex: 1}}}
			before := w.Hash()
			if err := projectCurrentItemGraphForSave(state, w); err != nil {
				t.Fatal(err)
			}
			seen := map[uint32]bool{}
			for _, record := range state.Document.Objects {
				for _, value := range record.Values {
					if value.Name != "Identity" && value.Name != "This" {
						continue
					}
					if value.Value == 0 || seen[value.Value] {
						t.Fatal("item-graph producer reused an unregistered current identity", value.Value)
					}
					seen[value.Value] = true
				}
			}
			if !seen[16] || !seen[64] || !seen[key] || w.Hash() != before {
				t.Fatal("item-graph projection lost an existing/current key or changed World")
			}
		})
	}
}

func TestCurrentPartyTemplateReservesNativeItemBeforeEffects(t *testing.T) {
	member, table := nativeCarryMember(t)
	item := sim.PlainItem(0x0e01)
	item.Price, item.WeightPresent, item.Weight = 17, true, 7
	item.Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}}
	item.NativeRecord = &sim.NativeItemRecord{Token: sim.SavedObjectToken{Identity: 32, RuntimeID: 19, Reference: 23}, F47: 91}
	member.Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(item, 1)}
	member.Carry.ItemInstances, member.Carry.Items = []sim.ItemInstance{item}, []uint16{item.Code}
	member.CarriedItems, member.Carried = member.Carry.ItemInstances, member.Carry.Items
	p, err := captureCurrentPartyTemplate(777, member, member, table)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint32]bool{}
	for _, record := range p.Template.Records.Objects {
		for _, value := range record.Values {
			if value.Name != "Identity" && value.Name != "This" || value.Value == 0 {
				continue
			}
			if seen[value.Value] {
				t.Fatal("party-template producer reused current native Item identity", value.Value)
			}
			seen[value.Value] = true
		}
	}
	if !seen[32] || member.Carry.ItemInstances[0].NativeRecord.Token.Identity != 32 {
		t.Fatal("party-template emission reminted native identity or changed current party")
	}
}

func TestCurrentNativeItemEmissionReusesRetainedSourceObject(t *testing.T) {
	doc, _, _ := actorProjectionFixture(t, "Human")
	var retained uint16
	for i := range doc.Objects {
		if doc.Objects[i].Class == "Item" {
			retained = uint16(i + 1)
			break
		}
	}
	if retained == 0 {
		t.Fatal("source fixture has no native Item aliases")
	}
	mustSetValue(&doc.Objects[retained-1], "Identity", 32)
	if _, _, err := sav.ReindexDocumentData(doc); err != nil {
		t.Fatal("source alias graph is invalid", err)
	}
	source, err := savedItemRecord(&doc, retained)
	if err != nil {
		t.Fatal(err)
	}
	item := source.Value.Instance()
	history := nativeItemRecord(source)
	item.ObjectID, item.NativeRecord = 0, &history
	item.Price, item.Effects = 17, []sim.ItemEffect{{Kind: 12, Operand: 3}}
	if err := item.ValidateWeight(); err != nil {
		t.Fatal("unregistered current native Item is invalid", err)
	}
	b := generatedDocumentBuilder{doc: doc}
	index, err := b.item(item, source.Value.Count, source.Token.Reference)
	if err != nil || index != retained {
		t.Fatal("current native Item duplicated its retained source object", index, retained, err)
	}
	row, err := savedItemRecord(&b.doc, index)
	if err != nil || row.Token.Identity != 32 || row.Token.RuntimeID != source.Token.RuntimeID || row.Value.Price != 17 || row.Value.Count != source.Value.Count || len(row.Value.Effects) != 1 || row.Value.Effects[0] != item.Effects[0] {
		t.Fatal("source-index reuse did not write current Item operands", row, err)
	}
	rawSnapshot, err := json.Marshal(b.doc)
	if err != nil {
		t.Fatal(err)
	}
	var rawBefore sav.DocumentData
	if err := json.Unmarshal(rawSnapshot, &rawBefore); err != nil || !reflect.DeepEqual(rawBefore, b.doc) {
		t.Fatal("raw document snapshot changed the current graph", err)
	}
	beforeDoc, _, err := sav.ReindexDocumentData(rawBefore)
	if err != nil {
		t.Fatal("current native alias graph is invalid", err)
	}
	before, err := sav.EncodeDocumentData(beforeDoc)
	if err != nil {
		t.Fatal(err)
	}
	key, runtime := b.nextKey, b.nextRuntime
	for _, field := range []string{"price", "effect", "count", "native_record"} {
		conflict, count := item.Clone(), source.Value.Count
		switch field {
		case "price":
			conflict.Price++
		case "effect":
			conflict.Effects[0].Operand++
		case "count":
			count++
		case "native_record":
			conflict.NativeRecord.F47++
		}
		if _, err := b.item(conflict, count, source.Token.Reference); err == nil {
			t.Fatal("conflicting current native aliases were merged", field)
		}
		afterDoc, _, err := sav.ReindexDocumentData(b.doc)
		if err != nil {
			t.Fatal("rejected current alias invalidated the document", field, err)
		}
		after, err := sav.EncodeDocumentData(afterDoc)
		if err != nil || !reflect.DeepEqual(rawBefore, b.doc) || !bytes.Equal(before, after) || b.nextKey != key || b.nextRuntime != runtime {
			t.Fatal("rejected current alias changed the document or allocators", field, err)
		}
	}
}
