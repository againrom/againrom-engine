package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func binaryObjectsWorld1115(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1}, {ID: 8, X: 2, Y: 2}})
	pack := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}
	ground := SavedObjectOwner{Kind: SavedOwnerSack, Object: 100}
	item := func(id SavedObjectID, owner SavedObjectOwner, code uint16, count uint32) SavedItemObject {
		return SavedItemObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Owner: owner,
			Value: ItemStack{ObjectID: id, Code: code, Kind: 1, Price: -27, Count: count, WeightPresent: true, Weight: -3},
			Token: SavedObjectToken{Position: [12]byte{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23},
				RuntimeID: 0x87654321, T0C: 13, T0E: 0xbeef, T08: 0xcafebabe, T18: 0xabcd, T1C: 0xffffffe5, Identity: 0xf1234567, Reference: 0x9abcdef0},
			F45: 0xf1, F46: 0xf2, F47: 0xf3, F48: 0xa987,
			Coverage: SavedObjectCoverage{Unknown: SavedUnknownPosition | SavedUnknownF46, Unsupported: "literal codec owner"}}
	}
	a := item(10, pack, 0x2369, 2)
	a.Value.Effects = []ItemEffect{{Kind: 41, Mode: 1, Operand: 0x80010007}, {Kind: 8, Mode: 0x82, Operand: 0xdeadbeef}}
	a.Value.SourceEquipment = SourceEquipment{Class: SourceWeapon, DefinitionRow: 13, OwnKind: 3,
		Attack: [24]byte{0: 0xf7, 23: 0xe9}, Defence: [22]byte{0: 0xd3, 21: 0xb1}, EffectsUnsupported: true,
		Definition: SourceWeaponDefinition{Present: true, AttackType: 2, Hands: 1, Charge: -111, Relax: 7, Suitable: 1},
		Spell:      SourceItemSpell{Present: true, ID: 7, Range: 33, Defensive: 0xfe, ManaCost: 65000}}
	a.Effects, a.Spell = []SavedObjectID{40, 41}, 50
	b := a
	b.ID, b.Value = 11, a.Value.Clone()
	b.Value.ObjectID, b.Value.Count = 11, 3
	sackItem := item(20, ground, 0x0201, 2)
	external := item(35, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: 0x123456789abcdef0}, 0x0901, 1)
	retired := item(36, SavedObjectOwner{Kind: SavedOwnerRetired}, 0x0201, 1)
	retired.Origin = SavedObjectOrigin{Kind: SavedObjectSplit, Parent: 20}
	worn := item(0x100000001, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 8, Slot: 2}, 0x0701, 1)
	r := &SavedObjects{Version: 1, NextID: 0x100000002,
		Items: []SavedItemObject{a, b, sackItem, external, retired, worn},
		Effects: []SavedEffectObject{
			{ID: 40, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: a.Token, Value: a.Value.Effects[0], E0C: 0xfd, ExternalReferences: 7},
			{ID: 41, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: b.Token, Value: a.Value.Effects[1], E0C: 3},
			{ID: 42, Origin: SavedObjectOrigin{Kind: SavedObjectSplit, Parent: 40}, Token: a.Token, Value: a.Value.Effects[0], E0C: 0x91, Retired: true,
				Coverage: SavedObjectCoverage{Unknown: SavedUnknownIdentity, Unsupported: "retired effect"}},
		},
		Spells: []SavedSpellObject{
			{ID: 50, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: a.Value.SourceEquipment.Spell, This: 0xfedcba98, ExternalReferences: 3},
			{ID: 51, Origin: SavedObjectOrigin{Kind: SavedObjectSplit, Parent: 50}, Value: SourceItemSpell{Present: true, ID: 7}, Retired: true,
				Coverage: SavedObjectCoverage{Unknown: SavedUnknownSpellInitialization, Unsupported: "unexpanded child"}},
		},
		Sacks: []SavedSackObject{
			{ID: 100, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: SavedObjectToken{Position: [12]byte{3, 3, 3, 3}, Identity: 0x89abcdef}, Gold: 0xf1234567},
			{ID: 101, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Retired: true, Coverage: SavedObjectCoverage{Unknown: SavedUnknownToken}},
		},
		SackRoots: []SavedObjectID{100, 100},
		Containers: []SavedObjectContainer{
			{Owner: pack, Present: true, InsertIndex: 123, Accumulator: -17, Items: []SavedObjectID{10, 11}, Coverage: SavedObjectCoverage{Unsupported: "container cursor"}},
			{Owner: SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}, Present: false},
			{Owner: ground, Present: true, InsertIndex: 65000, Accumulator: 2147, Items: []SavedObjectID{20}},
		},
	}
	if err := r.MigrateLegacyOwners(); err != nil {
		t.Fatal(err)
	}
	w.savedObjects = r
	w.carried[0] = []ItemStack{a.Value.Clone(), b.Value.Clone()}
	w.equipment[1][1] = worn.Value.Instance()
	s := makeSack(3, 3, 0xf1234567, []ItemInstance{sackItem.Value.Instance(), sackItem.Value.Instance()})
	s.ObjectID = 100
	w.sacks = []Sack{s}
	if err := w.validateSavedObjects(); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSavedObjectsBinary1115FullRegistryAndLiveHandlesRoundTrip(t *testing.T) {
	w := binaryObjectsWorld1115(t)
	form := mustMarshal(t, w)
	span := int(binary.LittleEndian.Uint32(form[len(form)-entityIDFloorLen-spellDeliverySpanLen-36:]))
	start := len(form) - entityIDFloorLen - spellDeliverySpanLen - 36 - span
	if form[0] != formatVersion || !bytes.Equal(form[start:start+13], []byte{2, 0, 0, 0, 1, 5, 0, 0, 0, 1, 0, 0, 0}) {
		t.Fatal("literal form84 suffix header differs")
	}
	if size, err := w.savedObjectsBinarySize(); err != nil || size != uint64(span) {
		t.Fatalf("explicit byte budget differs: %d span%d %v", size, span, err)
	}
	for i, row := range []struct {
		ordinal uint32
		id      uint64
	}{{0, 20}, {1, 20}, {2, 10}, {3, 11}, {17, 0x100000001}, {0, 100}} {
		at := start + 13 + i*12
		if binary.LittleEndian.Uint32(form[at:]) != row.ordinal || binary.LittleEndian.Uint64(form[at+4:]) != row.id {
			t.Fatalf("literal live handle row %d differs", i)
		}
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.savedObjects, cold.savedObjects) {
		t.Fatalf("registry fields, duplicate children or retired rows changed:\nwant=%+v\ngot=%+v", w.savedObjects, cold.savedObjects)
	}
	for i, stocks := range w.carried {
		if len(stocks) != len(cold.carried[i]) {
			t.Fatal("live pack population changed")
		}
		for j, stack := range stocks {
			if !StackStateEqual(stack, cold.carried[i][j]) {
				t.Fatal("live pack value or handle changed")
			}
		}
		for j, item := range w.equipment[i] {
			if !StackStateEqual(StackItem(item, 1), StackItem(cold.equipment[i][j], 1)) {
				t.Fatal("live worn value or handle changed")
			}
		}
	}
	if cold.sacks[0].ObjectID != 100 || len(cold.sacks[0].ItemInstances) != 2 {
		t.Fatal("live Sack identity/quantity changed")
	}
	for j, item := range w.sacks[0].ItemInstances {
		if !StackStateEqual(StackItem(item, 1), StackItem(cold.sacks[0].ItemInstances[j], 1)) {
			t.Fatal("live Sack item value or handle changed")
		}
	}
	if !bytes.Equal(form, mustMarshal(t, &cold)) || w.Hash() != cold.Hash() || len(cold.carried[0]) != 2 {
		t.Fatal("identical-value objects folded before handle adoption or changed native hash")
	}
	cold.savedObjects.Items[0].Effects[0] = 999
	cold.savedObjects.Items[0].Value.Effects[0].Operand++
	if w.savedObjects.Items[0].Effects[0] != 40 || w.savedObjects.Items[0].Value.Effects[0].Operand != 0x80010007 {
		t.Fatal("decoded registry aliases source World")
	}
}

func TestSavedObjectsBinary1115ReservedScrollHandleFollowsEveryWornSlot(t *testing.T) {
	w := binaryObjectsWorld1115(t)
	row := w.savedObjects.item(35)
	row.Value.Code, row.Value.Kind, row.Value.Price = 0x0e01, 4, 10
	row.Token.T1C = 10
	row.Value.Effects = []ItemEffect{{Kind: 41, Operand: 0x0001000b}}
	row.Effects = []SavedObjectID{43}
	w.savedObjects.Effects = append(w.savedObjects.Effects, SavedEffectObject{
		ID: 43, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: row.Value.Effects[0],
	})
	w.entities[0].HP, w.entities[0].MaxHP = 10, 10
	w.spells = []SpellRule{{ID: 11}}
	w.scrollCasts = []ScrollCast{{Caster: 7, X: 3, Y: 3, AtCell: true, Index: 17, Item: row.Value.Instance(), Reservation: 0x123456789abcdef0}}
	form := mustMarshal(t, w)
	span := int(binary.LittleEndian.Uint32(form[len(form)-entityIDFloorLen-spellDeliverySpanLen-36:]))
	start := len(form) - entityIDFloorLen - spellDeliverySpanLen - 36 - span
	if binary.LittleEndian.Uint32(form[start+5:]) != 6 || binary.LittleEndian.Uint32(form[start+73:]) != 28 || binary.LittleEndian.Uint64(form[start+77:]) != 35 {
		t.Fatal("reserved scroll identity ordinal skipped empty equipment slots")
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if len(cold.scrollCasts) != 1 || !StackStateEqual(StackItem(cold.scrollCasts[0].Item, 1), row.Value) || !bytes.Equal(form, mustMarshal(t, &cold)) || cold.Hash() != w.Hash() {
		t.Fatal("reserved scroll handle/value did not survive native LOAD")
	}
}

func TestSavedObjectsBinary1115CorruptSuffixRefusesAtomically(t *testing.T) {
	w := binaryObjectsWorld1115(t)
	good := mustMarshal(t, w)
	span := int(binary.LittleEndian.Uint32(good[len(good)-entityIDFloorLen-spellDeliverySpanLen-36:]))
	start := len(good) - entityIDFloorLen - spellDeliverySpanLen - 36 - span
	// Header13 + six handle rows72 + registry header48 = first Item133.
	const firstItem = 133
	spanAt := len(good) - entityIDFloorLen - spellDeliverySpanLen - 36
	if binary.LittleEndian.Uint64(good[start+firstItem:]) != 10 || binary.LittleEndian.Uint32(good[start+firstItem+179:]) != 2 || binary.LittleEndian.Uint16(good[start+firstItem+215:]) != uint16(len("literal codec owner")) {
		t.Fatal("independent malformed-field offsets no longer identify the authored Item")
	}
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"schema", func(b []byte) []byte { b[start] = 3; return b }},
		{"presence", func(b []byte) []byte { b[start+4] = 2; return b }},
		{"span-too-large", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-36:], maxSavedObjectsBytes+1)
			return b
		}},
		{"handle-count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+5:], 0xffffffff); return b }},
		{"duplicate-ordinal", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+25:], 0); return b }},
		{"zero-handle", func(b []byte) []byte { clear(b[start+17 : start+25]); return b }},
		{"outside-item", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+61:], 999999); return b }},
		{"empty-worn-item", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+61:], 16); return b }},
		{"outside-sack", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+73:], 1); return b }},
		{"live-key-mismatch", func(b []byte) []byte { binary.LittleEndian.PutUint64(b[start+17:], 10); return b }},
		{"registry-version", func(b []byte) []byte { b[start+85] = 3; return b }},
		{"registry-count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+97:], 0xffffffff); return b }},
		{"registry-item-effects", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+firstItem+41:], 0xffffffff); return b }},
		{"inflight", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[start+firstItem+18:], 1)
			return b
		}},
		{"weight-boolean", func(b []byte) []byte { b[start+firstItem+57] = 2; return b }},
		{"definition-boolean", func(b []byte) []byte { b[start+firstItem+109] = 2; return b }},
		{"spell-boolean", func(b []byte) []byte { b[start+firstItem+130] = 2; return b }},
		{"unsupported-boolean", func(b []byte) []byte { b[start+firstItem+136] = 2; return b }},
		{"edge-count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+firstItem+179:], 0xffffffff); return b }},
		{"coverage-length", func(b []byte) []byte { binary.LittleEndian.PutUint16(b[start+firstItem+215:], 513); return b }},
		{"coverage-nul", func(b []byte) []byte { b[start+firstItem+217] = 0; return b }},
		{"coverage-bits", func(b []byte) []byte { b[start+firstItem+214] = 0x80; return b }},
		{"extra-byte", func(b []byte) []byte {
			out := append(bytes.Clone(b[:spanAt]), 0)
			out = binary.LittleEndian.AppendUint32(out, uint32(span+1))
			return append(out, b[spanAt+4:]...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := tc.edit(bytes.Clone(good))
			before := w.Hash()
			if err := w.UnmarshalBinary(bad); err == nil || before != w.Hash() || !bytes.Equal(good, mustMarshal(t, w)) {
				t.Fatalf("corrupt suffix adopted: %v", err)
			}
		})
	}
	// Every truncated payload prefix keeps a correctly framed footer. This
	// exercises the record reader, not just the outer span refusal.
	for n := 1; n < span; n++ {
		bad := append(bytes.Clone(good[:start]), good[start:start+n]...)
		bad = binary.LittleEndian.AppendUint32(bad, uint32(n))
		bad = append(bad, good[spanAt+4:]...)
		if err := w.UnmarshalBinary(bad); err == nil {
			t.Fatalf("truncated suffix prefix %d admitted", n)
		}
	}
	if !bytes.Equal(good, mustMarshal(t, w)) {
		t.Fatal("truncated decode changed receiver")
	}
}

func TestSavedObjectsBinary1115PresentEmptyRegistryAndStrictWriter(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{8, 8}, nil)
	current := mustMarshal(t, w)
	if current[0] != formatVersion || !bytes.Equal(current[len(current)-entityIDFloorLen-spellDeliverySpanLen-36:len(current)-entityIDFloorLen-spellDeliverySpanLen], make([]byte, 36)) {
		t.Fatal("absent registry is not the four-byte zero suffix")
	}
	var cold World
	w.savedObjects = &SavedObjects{Version: SavedObjectsVersion, NextID: 1}
	present := mustMarshal(t, w)
	if bytes.Equal(current, present) || binary.LittleEndian.Uint32(present[len(present)-entityIDFloorLen-spellDeliverySpanLen-36:]) != 61 {
		t.Fatal("present empty registry collapsed into legacy absence")
	}
	if err := cold.UnmarshalBinary(present); err != nil || cold.savedObjects == nil || cold.savedObjects.NextID != 1 {
		t.Fatal("present empty registry did not round trip", err)
	}
	w = binaryObjectsWorld1115(t)
	w.savedObjects.Items[0].Coverage.Unsupported = strings.Repeat("x", 513)
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("writer admitted oversized coverage")
	}
	w = binaryObjectsWorld1115(t)
	w.savedObjects.Items[0].InFlight = 1
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("writer admitted in-flight registry")
	}
	w = binaryObjectsWorld1115(t)
	w.savedObjects = nil
	hash := w.Hash()
	w.carried[0][0].ObjectID++
	if _, err := w.MarshalBinary(); err == nil || hash == w.Hash() {
		t.Fatal("orphan handle was admitted or omitted from the hash")
	}
}
