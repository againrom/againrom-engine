package sim

import (
	"reflect"
	"slices"
	"testing"
)

func objectToken1115(seed uint8, price int32) SavedObjectToken {
	t := SavedObjectToken{RuntimeID: 0x80000000 + uint32(seed), T0C: seed, T0E: 0xbeef, T08: 0x40,
		T18: 0xfefe, T1C: uint32(price), Identity: 0xfedcba98, Reference: 0xfedcba98}
	for i := range t.Position {
		t.Position[i] = seed + uint8(i)
	}
	return t
}

func objectItem1115(id SavedObjectID, owner SavedObjectOwner, count uint32) SavedItemObject {
	return SavedItemObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Owner: owner,
		Value: ItemStack{ObjectID: id, Code: 0x3185, Kind: 1, Price: -27, Count: count, WeightPresent: true, Weight: 2},
		Token: objectToken1115(13, -27), F45: 0x91, F46: 0xff, F47: 0x5e, F48: 0x8765}
}

func objectRegistry1115(t *testing.T) *SavedObjects {
	t.Helper()
	pack7 := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}
	pack8 := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}
	ground := SavedObjectOwner{Kind: SavedOwnerSack, Object: 100}
	a, b, weapon := objectItem1115(10, pack7, 3), objectItem1115(20, ground, 1), objectItem1115(30, pack8, 2)
	b.Token.T08 = 0x88
	weapon.Value.Code = 0x2369
	weapon.Value.Kind = 2
	weapon.Value.Weight = -3
	weapon.Value.SourceEquipment = SourceEquipment{Class: SourceWeapon, DefinitionRow: 13, OwnKind: 3,
		Definition: SourceWeaponDefinition{Present: true, AttackType: 2, Hands: 1, Charge: 5, Relax: 7, Suitable: 1},
		Spell:      SourceItemSpell{Present: true, ID: 7, Range: 33, Defensive: 0xfe, ManaCost: 65000}}
	weapon.Value.SourceEquipment.Attack[23], weapon.Value.SourceEquipment.Defence[21] = 0xfb, 0x87
	weapon.Value.Effects = []ItemEffect{{Kind: 41, Mode: 1, Operand: 0x9abc0007}, {Kind: 8, Mode: 0x82, Operand: 0xdeadbeef}}
	weapon.Effects, weapon.Spell = []SavedObjectID{40, 41}, 50
	r := &SavedObjects{Version: 1, NextID: 101,
		Items: []SavedItemObject{a, b, weapon},
		Effects: []SavedEffectObject{
			{ID: 40, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: objectToken1115(23, 19), E0C: 0xde, Value: weapon.Value.Effects[0]},
			{ID: 41, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: objectToken1115(21, 21), E0C: 21, Value: weapon.Value.Effects[1]},
		},
		Spells:    []SavedSpellObject{{ID: 50, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, This: 0xfedcba98, Value: weapon.Value.SourceEquipment.Spell}},
		Sacks:     []SavedSackObject{{ID: 100, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Gold: 0xf1234567, Token: objectToken1115(17, 123)}},
		SackRoots: []SavedObjectID{100},
		Containers: []SavedObjectContainer{
			{Owner: pack7, Present: true, InsertIndex: 10000, Accumulator: 106, Items: []SavedObjectID{10}},
			{Owner: pack8, Present: true, InsertIndex: 10000, Accumulator: 14, Items: []SavedObjectID{30}},
			{Owner: ground, Present: true, InsertIndex: 10000, Accumulator: 77, Items: []SavedObjectID{20}},
		},
	}
	if err := r.MigrateLegacyOwners(); err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r
}

func sameRegistry1115(t *testing.T, before, after *SavedObjects) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed operation changed registry or allocated IDs")
	}
}

func TestSavedObjects1115WholeTransferKeepsCompleteIdentity(t *testing.T) {
	r := objectRegistry1115(t)
	before, _ := singleRootItem(t, r, 10)
	id, err := r.TakeWhole(10, before.Owner, before.Value)
	if err != nil || id != 10 || r.NextID != 101 {
		t.Fatalf("whole take = %d, %v", id, err)
	}
	if err := r.ValidateNoInFlight(); err == nil {
		t.Fatal("in-flight token admitted at save boundary")
	}
	destination := SavedObjectOwner{Kind: SavedOwnerSack, Object: 100}
	if retained, err := r.Insert(id, destination, SavedMergeNone); err != nil || retained != 10 {
		t.Fatalf("insert = %d, %v", retained, err)
	}
	got, _ := singleRootItem(t, r, 10)
	if got.Token != before.Token || !StackStateEqual(got.Value, before.Value) || got.F45 != before.F45 || got.F46 != before.F46 || got.F47 != before.F47 || got.F48 != before.F48 {
		t.Fatal("whole transfer changed full item operands")
	}
	if !slices.Equal(r.Containers[2].Items, []SavedObjectID{20, 10}) || r.Containers[0].Accumulator != 100 || r.Containers[2].Accumulator != 83 || r.Containers[0].InsertIndex != 0 {
		t.Fatal("whole transfer lost order or retained container bookkeeping")
	}
	if r.NextID != 101 || r.ValidateNoInFlight() != nil {
		t.Fatal("whole transfer minted or remained detached")
	}
	beforeRegistry := r.Clone()
	if _, err := r.Insert(id, destination, SavedMergeNone); err == nil {
		t.Fatal("same detached token inserted twice")
	}
	sameRegistry1115(t, beforeRegistry, r)
}

func TestSavedObjects1115SplitMintsItemEffectsAndSpell(t *testing.T) {
	r := objectRegistry1115(t)
	original, _ := singleRootItem(t, r, 30)
	id, err := r.TakeOne(30, original.Owner, original.Value)
	if err != nil || id != 101 || r.NextID != 105 {
		t.Fatalf("split = %d next=%d, %v", id, r.NextID, err)
	}
	clone, _ := singleRootItem(t, r, id)
	source, _ := singleRootItem(t, r, 30)
	if source.Value.Count != 1 || source.Token != original.Token || !slices.Equal(source.Effects, []SavedObjectID{40, 41}) || source.Spell != 50 {
		t.Fatal("split rewrote source identity or source-only fields")
	}
	if clone.Value.Count != 1 || clone.Value.ObjectID != 101 || clone.Origin != (SavedObjectOrigin{Kind: SavedObjectSplit, Parent: 30}) || !slices.Equal(clone.Effects, []SavedObjectID{102, 103}) || clone.Spell != 104 {
		t.Fatal("split did not make independent one-unit identities")
	}
	if clone.F47 != 0 || clone.Coverage.Unknown&SavedUnknownF47 == 0 || clone.Coverage.Unknown&SavedUnknownToken != SavedUnknownToken {
		t.Fatal("split invented known constructor/F47 operands")
	}
	if clone.Value.SourceEquipment.Spell != (SourceItemSpell{Present: true, ID: 7}) || clone.Coverage.Unknown&SavedUnknownSpellInitialization == 0 || r.Spells[1].This != 0 {
		t.Fatal("split copied filled owned Spell scratch or invented identity")
	}
	for i, child := range r.Effects[2:] {
		if child.ID != SavedObjectID(102+i) || child.Value != original.Value.Effects[i] || child.E0C != r.Effects[i].E0C || child.Token.T0C != r.Effects[i].Token.T0C || child.Origin.Parent != SavedObjectID(40+i) || child.Token.Identity != 0 || child.Coverage.Unknown&SavedUnknownIdentity == 0 || child.Token.Position != r.Effects[i].Token.Position {
			t.Fatal("split Effect copy lost value/base or claimed identity")
		}
	}
	if r.Containers[1].Accumulator != 17 { // negative signed item weight
		t.Fatal("split used absolute weight or erased stored load residue")
	}
	if _, err := r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 8, Slot: 1}, SavedMergeNone); err != nil {
		t.Fatal(err)
	}
	copy := r.Clone()
	copy.Items[3].Value.Effects[0].Operand ^= 0xff
	copy.Items[3].Effects[0] = 0
	if r.Items[3].Value.Effects[0].Operand != original.Value.Effects[0].Operand || r.Items[3].Effects[0] != 102 {
		t.Fatal("snapshot clone exposed mutable effects")
	}
}

func TestSavedObjects1115PickupMergeAndSackRetirement(t *testing.T) {
	r := objectRegistry1115(t)
	beforeSack := r.Sacks[0]
	if err := r.StampPickup(100); err != nil {
		t.Fatal(err)
	}
	item, _ := singleRootItem(t, r, 20)
	if item.Token.T08 != 1 {
		t.Fatal("pickup did not replace incoming flags before drain")
	}
	id, err := r.TakeOne(20, item.Owner, item.Value)
	if err != nil || id != 20 || r.NextID != 101 {
		t.Fatalf("count-one pickup cloned: %d, %v", id, err)
	}
	if retained, err := r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, SavedMergeNativeRetention); err != nil || retained != 10 {
		t.Fatalf("merge = %d, %v", retained, err)
	}
	destination, _ := singleRootItem(t, r, 10)
	incoming, _ := singleRootItem(t, r, 20)
	if destination.Value.Count != 4 || destination.Token.T08 != 0x41 || !incoming.Retired || destination.Token.Position != objectToken1115(13, -27).Position {
		t.Fatal("pickup merge did not retain destination object/position with flags OR 1")
	}
	if r.Containers[0].Accumulator != 108 || r.Containers[2].Accumulator != 75 {
		t.Fatal("pickup overwrote container load residue")
	}
	if err := r.SetSackGold(100, beforeSack.Gold, 0x80000000); err != nil || r.Sacks[0].Token != beforeSack.Token {
		t.Fatal("gold setter fabricated a Token/cache update", err)
	}
	bad := r.Clone()
	if err := r.RetireSack(100, 7); err == nil {
		t.Fatal("retired Sack despite stale gold")
	}
	sameRegistry1115(t, bad, r)
	if err := r.RetireSack(100, 0x80000000); err != nil {
		t.Fatal(err)
	}
	if !r.Sacks[0].Retired || r.Sacks[0].Gold != 0 || len(r.SackRoots) != 0 || len(r.Containers) != 2 || r.Sacks[0].Token != beforeSack.Token {
		t.Fatal("Sack retirement retained live roots/container or erased raw identity")
	}
}

func TestSavedObjects1115ExternalOwnershipAndPairing(t *testing.T) {
	r := objectRegistry1115(t)
	item, _ := singleRootItem(t, r, 10)
	id, err := r.TakeWhole(item.ID, item.Owner, item.Value)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ExportExternal(id, 0x8000000000000032); err != nil {
		t.Fatal(err)
	}
	if err := r.ValidateNoInFlight(); err != nil {
		t.Fatal("explicit session owner confused with transient token", err)
	}
	if err := r.ValidateExternal(nil); err == nil {
		t.Fatal("unpaired game-session item admitted")
	}
	pair := SavedExternalItem{Handle: 0x8000000000000032, ID: id, Value: item.Value}
	if err := r.ValidateExternal([]SavedExternalItem{pair}); err != nil {
		t.Fatal(err)
	}
	if err := r.ValidateExternal([]SavedExternalItem{pair, pair}); err == nil {
		t.Fatal("duplicated session pair admitted")
	}
	wrong := item.Value.Clone()
	wrong.Price++
	before := r.Clone()
	if _, err := r.ImportExternal(pair.Handle, wrong); err == nil {
		t.Fatal("external return repaired mismatched value")
	}
	sameRegistry1115(t, before, r)
	if got, err := r.ImportExternal(pair.Handle, item.Value); err != nil || got != id {
		t.Fatalf("return = %d, %v", got, err)
	}
	if _, err := r.Insert(id, item.Owner, SavedMergeNativeRetention); err != nil {
		t.Fatal(err)
	}
	if r.NextID != 101 || r.ValidateExternal(nil) != nil {
		t.Fatal("external roundtrip minted or left session ownership")
	}
}

func TestSavedObjects1115DisposeUniqueChildren(t *testing.T) {
	r := objectRegistry1115(t)
	item, _ := singleRootItem(t, r, 30)
	id, err := r.TakeWhole(item.ID, item.Owner, item.Value)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Dispose(id); err != nil {
		t.Fatal(err)
	}
	if !r.Items[2].Retired || !r.Effects[0].Retired || !r.Effects[1].Retired || !r.Spells[0].Retired || r.NextID != 101 {
		t.Fatal("dispose did not retire exactly the owned objects")
	}
	before := r.Clone()
	if err := r.Dispose(id); err == nil {
		t.Fatal("already disposed token reused")
	}
	sameRegistry1115(t, before, r)
}

func TestSavedObjectsAliasImportSplitAndLastReferenceRetirement(t *testing.T) {
	for _, kind := range []string{"effect-external", "spell-external", "duplicate-effect", "effect-and-spell-shared"} {
		t.Run(kind, func(t *testing.T) {
			r := objectRegistry1115(t)
			switch kind {
			case "effect-external":
				r.Effects[0].ExternalReferences = 1
			case "spell-external":
				r.Spells[0].ExternalReferences = 1
			case "duplicate-effect":
				r.Items[2].Effects = append(r.Items[2].Effects, 40)
				r.Items[2].Value.Effects = append(r.Items[2].Value.Effects, r.Effects[0].Value)
			case "effect-and-spell-shared":
				r.Items[0].Value = r.Items[2].Value.Clone()
				r.Items[0].Value.ObjectID = 10
				r.Items[0].Effects = slices.Clone(r.Items[2].Effects)
				r.Items[0].Spell = 50
			}
			if err := r.Validate(); err != nil {
				t.Fatal("valid identity alias rejected/flattened at import", err)
			}
			original, _ := singleRootItem(t, r, 30)
			cloneID, err := r.TakeOne(30, original.Owner, original.Value)
			if err != nil {
				t.Fatal("copy cannot preserve source alias topology", err)
			}
			clone, _ := singleRootItem(t, r, cloneID)
			if len(clone.Effects) != len(original.Effects) || r.Items[2].Effects[0] != 40 {
				t.Fatal("split flattened alias multiplicity")
			}
			if kind == "duplicate-effect" && clone.Effects[0] == clone.Effects[2] {
				t.Fatal("per-node deep copy reused the old shared identity")
			}
			if err := r.Dispose(cloneID); err != nil {
				t.Fatal("fresh split children incorrectly inherited external aliases", err)
			}
			remaining, _ := singleRootItem(t, r, 30)
			id, err := r.TakeWhole(30, remaining.Owner, remaining.Value)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Dispose(id); err != nil {
				t.Fatal(err)
			}
			if !r.Items[2].Retired {
				t.Fatal("released Item was not retired")
			}
			if kind == "effect-external" && r.Effects[0].Retired || kind == "spell-external" && r.Spells[0].Retired || kind == "effect-and-spell-shared" && (r.Effects[0].Retired || r.Spells[0].Retired) {
				t.Fatal("surviving child reference was retired")
			}
			if kind == "duplicate-effect" && (!r.Effects[0].Retired || !r.Spells[0].Retired) {
				t.Fatal("last repeated child reference remained live")
			}
		})
	}
}

func TestSavedObjects1115CountWidthsAndAtomicOverflow(t *testing.T) {
	for _, tc := range []struct {
		name        string
		count, add  uint32
		want        uint32
		refuse, gap bool
	}{
		{name: "word-edge", count: 65534, add: 1, want: 65535},
		{name: "word-overflow", count: 65535, add: 1, want: 65536, gap: true},
		{name: "native-wrap", count: ^uint32(0), add: 2, want: 1, gap: true},
		{name: "invalid-native-zero", count: ^uint32(0), add: 1, refuse: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := objectRegistry1115(t)
			r.Items[0].Value.Count, r.Items[1].Value.Count = tc.count, tc.add
			if tc.count > 65535 {
				r.Items[0].Coverage.Unknown |= SavedUnknownCountWidth
			}
			item, _ := singleRootItem(t, r, 20)
			id, err := r.TakeWhole(20, item.Owner, item.Value)
			if err != nil {
				t.Fatal(err)
			}
			before := r.Clone()
			_, err = r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, SavedMergeNativeRetention)
			if tc.refuse {
				if err == nil {
					t.Fatal("zero-count overflow admitted")
				}
				sameRegistry1115(t, before, r)
				return
			}
			if err != nil || r.Items[0].Value.Count != tc.want || (r.Items[0].Coverage.Unknown&SavedUnknownCountWidth != 0) != tc.gap {
				t.Fatalf("merge count=%d coverage=%x error=%v", r.Items[0].Value.Count, r.Items[0].Coverage.Unknown, err)
			}
		})
	}
}

func TestSavedObjects1115ExhaustedIDAfterChildMintRollsBack(t *testing.T) {
	r := objectRegistry1115(t)
	r.NextID = ^SavedObjectID(0) - 2 // Item and first Effect fit; second Effect fails.
	item, _ := singleRootItem(t, r, 30)
	before := r.Clone()
	if _, err := r.TakeOne(30, item.Owner, item.Value); err == nil {
		t.Fatal("exhausted child ID allocation accepted")
	}
	sameRegistry1115(t, before, r)
}

func TestSavedObjects1115HostileStateAndLiveMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*SavedObjects)
	}{
		{"price", func(r *SavedObjects) { r.Items[0].Token.T1C++ }},
		{"handle", func(r *SavedObjects) { r.Items[0].Value.ObjectID = 20 }},
		{"definition-row", func(r *SavedObjects) { r.Items[2].Token.T0C++ }},
		{"effect-value", func(r *SavedObjects) { r.Effects[0].Value.Mode++ }},
		{"spell-value", func(r *SavedObjects) { r.Spells[0].Value.ManaCost++ }},
		{"spell-edge", func(r *SavedObjects) { r.Items[2].Spell = 0 }},
		{"duplicate-owner", func(r *SavedObjects) { r.Containers = append(r.Containers, r.Containers[0]) }},
		{"missing-owner", func(r *SavedObjects) { r.Containers[0].Items = nil }},
		{"dead-child", func(r *SavedObjects) { r.Effects[0].Retired = true }},
		{"cross-kind-id", func(r *SavedObjects) { r.Spells[0].ID = 40 }},
		{"bad-next", func(r *SavedObjects) { r.NextID = 100 }},
		{"bad-origin", func(r *SavedObjects) { r.Items[0].Origin.Parent = 5 }},
		{"unknown-mask", func(r *SavedObjects) { r.Items[0].Coverage.Unknown = 1 << 63 }},
		{"raw-zero-count", func(r *SavedObjects) { r.Items[0].Value.Count = 0 }},
		{"unmarked-wide-count", func(r *SavedObjects) { r.Items[0].Value.Count = 65536 }},
		{"retired-root", func(r *SavedObjects) { r.Sacks[0].Retired = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := objectRegistry1115(t)
			tc.edit(r)
			before := r.Clone()
			if err := r.Validate(); err == nil {
				t.Fatal("hostile registry accepted")
			}
			if err := r.StampPickup(100); err == nil {
				t.Fatal("mutation repaired hostile registry")
			}
			sameRegistry1115(t, before, r)
		})
	}
	r := objectRegistry1115(t)
	bindings := []SavedObjectBinding{
		{ID: 10, Owner: SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, Value: r.Items[0].Value.Clone()},
		{ID: 20, Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: 100}, Value: r.Items[1].Value.Clone()},
		{ID: 30, Owner: SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}, Value: r.Items[2].Value.Clone()},
	}
	if err := r.CompareLive(bindings); err != nil {
		t.Fatal(err)
	}
	bindings[0].Value.Count++
	before := r.Clone()
	if err := r.CompareLive(bindings); err == nil {
		t.Fatal("live value disagreement repaired")
	}
	sameRegistry1115(t, before, r)
	bindings[0].Value.Count--
	bindings[0].Index = 1
	if err := r.CompareLive(bindings); err == nil {
		t.Fatal("live ordered edge disagreement ignored")
	}
	if err := r.CompareLive(bindings[1:]); err == nil {
		t.Fatal("omitted live identity accepted")
	}
}

func TestSavedObjects1115AbsentLegacyIsNotAdopted(t *testing.T) {
	var absent *SavedObjects
	if absent.Clone() != nil || absent.Validate() != nil || absent.ValidateExternal(nil) != nil || absent.ValidateNoInFlight() != nil {
		t.Fatal("nil legacy registry acquired state")
	}
	legacy := SavedObjectBinding{Value: PlainStack(0x3185, 3)}
	if err := absent.CompareLive([]SavedObjectBinding{legacy}); err != nil {
		t.Fatal(err)
	}
	legacy.ID = 10
	legacy.Value.ObjectID = 10
	if err := absent.CompareLive([]SavedObjectBinding{legacy}); err == nil {
		t.Fatal("bound item silently reconstructed without owner")
	}
	r := objectRegistry1115(t)
	old := r.Clone()
	if err := r.RetireSack(100, r.Sacks[0].Gold); err == nil {
		t.Fatal("nonempty Sack retired")
	}
	sameRegistry1115(t, old, r)
}

func TestSavedObjects1115NativeAndOriginalPredicatesRemainDistinct(t *testing.T) {
	for _, policy := range []SavedMergePolicy{SavedMergeNativeRetention, SavedMergeOriginalPredicate} {
		t.Run(map[SavedMergePolicy]string{SavedMergeNativeRetention: "native-enchanted", SavedMergeOriginalPredicate: "original-enchanted"}[policy], func(t *testing.T) {
			r := objectRegistry1115(t)
			// Equal non-stackable values merge under the existing native
			// predicate, but not the literal original stackable-only branch.
			incomingEffect := r.Effects[1]
			incomingEffect.ID = 60
			r.Effects = append(r.Effects, incomingEffect)
			r.Items[0].Effects, r.Items[1].Effects = []SavedObjectID{41}, []SavedObjectID{60}
			r.Items[0].Value.Effects = []ItemEffect{incomingEffect.Value}
			r.Items[1].Value.Effects = []ItemEffect{incomingEffect.Value}
			item, _ := singleRootItem(t, r, 20)
			id, err := r.TakeWhole(20, item.Owner, item.Value)
			if err != nil {
				t.Fatal(err)
			}
			retained, err := r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, policy)
			if err != nil {
				t.Fatal(err)
			}
			if policy == SavedMergeNativeRetention {
				if retained != 10 || r.Items[0].Value.Count != 4 || r.Items[0].Coverage.Unknown&SavedUnknownMergePolicy == 0 || !r.Effects[2].Retired || r.Effects[1].Retired {
					t.Fatal("native merge hid its predicate difference or retired retained children")
				}
			} else if retained != 20 || r.Items[0].Value.Count != 3 || !slices.Equal(r.Containers[0].Items, []SavedObjectID{10, 20}) || r.Effects[2].Retired {
				t.Fatal("literal predicate was replaced by native equality")
			}
		})
	}
	for _, policy := range []SavedMergePolicy{SavedMergeNativeRetention, SavedMergeOriginalPredicate} {
		t.Run(map[SavedMergePolicy]string{SavedMergeNativeRetention: "native-potion", SavedMergeOriginalPredicate: "original-potion"}[policy], func(t *testing.T) {
			r := objectRegistry1115(t)
			r.Items[0].Value.Kind, r.Items[1].Value.Kind = 3, 3
			r.Items[1].Value.Price = -28
			r.Items[1].Token.T1C = uint32(r.Items[1].Value.Price)
			item, _ := singleRootItem(t, r, 20)
			id, err := r.TakeWhole(20, item.Owner, item.Value)
			if err != nil {
				t.Fatal(err)
			}
			retained, err := r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}, policy)
			if err != nil {
				t.Fatal(err)
			}
			if policy == SavedMergeNativeRetention {
				if retained != 20 || r.Items[1].Coverage.Unknown&SavedUnknownMergePolicy == 0 || r.Items[1].Value.Price != -28 {
					t.Fatal("native price retention was hidden or lost")
				}
			} else if retained != 10 || r.Items[0].Value.Price != -27 || r.Items[0].Value.Count != 4 {
				t.Fatal("literal merge did not preserve destination operands")
			}
		})
	}
}

func TestSavedObjectsSharedMergeAndOccupiedEquipmentAtomic(t *testing.T) {
	r := objectRegistry1115(t)
	// Incoming and destination share exact child identities. Import and a
	// whole transfer preserve those aliases; retirement retains the shared child.
	r.Items[1].Value = r.Items[2].Value.Clone()
	r.Items[1].Value.ObjectID, r.Items[1].Value.Count = 20, 1
	r.Items[1].Effects, r.Items[1].Spell = slices.Clone(r.Items[2].Effects), 50
	item, _ := singleRootItem(t, r, 20)
	id, err := r.TakeWhole(20, item.Owner, item.Value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Insert(id, SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}, SavedMergeNativeRetention); err != nil {
		t.Fatal(err)
	}
	if !r.Items[1].Retired || r.Effects[0].Retired || r.Spells[0].Retired {
		t.Fatal("merge destroyed a surviving shared child")
	}
	var before *SavedObjects

	r = objectRegistry1115(t)
	first, _ := singleRootItem(t, r, 10)
	id, err = r.TakeOne(10, first.Owner, first.Value)
	if err != nil {
		t.Fatal(err)
	}
	worn := SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 7, Slot: 1}
	if _, err := r.Insert(id, worn, SavedMergeNone); err != nil {
		t.Fatal(err)
	}
	second, _ := singleRootItem(t, r, 20)
	id, err = r.TakeOne(20, second.Owner, second.Value)
	if err != nil {
		t.Fatal(err)
	}
	before = r.Clone()
	if _, err := r.Insert(id, worn, SavedMergeNone); err == nil {
		t.Fatal("occupied worn owner accepted a second object")
	}
	sameRegistry1115(t, before, r)
}

func TestSavedObjects1115GoldOnlyZeroHighBitAndAliasedRoots(t *testing.T) {
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: 18,
		Sacks:      []SavedSackObject{{ID: 17, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: objectToken1115(0xef, -321)}},
		SackRoots:  []SavedObjectID{17, 17},
		Containers: []SavedObjectContainer{{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: 17}, Present: true, InsertIndex: 0xffffffff, Accumulator: -7}},
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	token := r.Sacks[0].Token
	var previous uint32
	for _, gold := range []uint32{0x80000000, ^uint32(0), 0, 19} {
		if err := r.SetSackGold(17, previous, gold); err != nil {
			t.Fatal(err)
		}
		if r.Sacks[0].Gold != gold || r.Sacks[0].Token != token || len(r.SackRoots) != 2 {
			t.Fatal("gold setter narrowed raw money or normalized source graph")
		}
		previous = gold
	}
	before := r.Clone()
	if err := r.SetSackGold(17, 0, 8); err == nil {
		t.Fatal("stale gold update accepted")
	}
	sameRegistry1115(t, before, r)
	if err := r.RetireSack(17, 19); err != nil {
		t.Fatal(err)
	}
	if len(r.SackRoots) != 0 || len(r.Containers) != 0 || !r.Sacks[0].Retired || r.Sacks[0].Gold != 0 || r.Sacks[0].Token != token || r.NextID != 18 {
		t.Fatal("named Sack retirement changed unrelated raw fields or kept an alias root")
	}
	if err := r.ValidateNoInFlight(); err != nil {
		t.Fatal(err)
	}
	bad := r.Clone()
	bad.Sacks[0].Gold = 1
	if err := bad.Validate(); err == nil {
		t.Fatal("retired Sack still claimed spendable gold")
	}
	before = r.Clone()
	if err := r.SetSackGold(17, 0, 3); err == nil {
		t.Fatal("retired Sack resurrected by gold setter")
	}
	sameRegistry1115(t, before, r)
}

// Old exclusive-owner controls address one root explicitly through this view.
// Shared-root controls never use it, and current production rows keep Owner zero.
func singleRootItem(t *testing.T, r *SavedObjects, id SavedObjectID) (SavedItemObject, bool) {
	t.Helper()
	row, ok := r.Item(id)
	if !ok {
		return row, false
	}
	roots := r.Locations(id)
	if len(roots) > 1 {
		t.Fatal("exclusive fixture has multiple roots")
	}
	if len(roots) == 1 {
		row.Owner = roots[0].Owner
	} else if row.Retired {
		row.Owner = SavedObjectOwner{Kind: SavedOwnerRetired}
	} else if row.InFlight != 0 {
		row.Owner = SavedObjectOwner{Kind: SavedOwnerInFlight}
	}
	return row, true
}
