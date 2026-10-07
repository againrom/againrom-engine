package sim

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func itemOperationsWorld(t *testing.T) *World {
	t.Helper()
	w, r, sacks := unboundGoldWorld1115(t)
	r.Version = 1
	owner := SavedObjectOwner{Kind: SavedOwnerSack, Object: goldObject1115}
	item := objectItem1115(10, owner, 3)
	r.Items = []SavedItemObject{item}
	r.Containers[0].Items = []SavedObjectID{10}
	value := item.Value.Instance()
	value.ObjectID = 0
	w.sacks[0].ItemInstances = []ItemInstance{value.Clone(), value.Clone(), value.Clone()}
	w.sacks[0].Items = itemCodes(w.sacks[0].ItemInstances)
	w.carried[0] = []ItemStack{StackItem(value, 2)}
	w.recomputeLoad(0)
	if err := w.ImportSavedObjects(r, sacks, SavedObjectBinding{ID: 10, Owner: owner, Value: item.Value}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSavedItemOperations1115PickupSplitsMergesMixedAndReloads(t *testing.T) {
	w := itemOperationsWorld(t)
	beforeID := w.savedObjects.NextID
	if err := w.TakeSack(7, 10, 20); err != nil {
		t.Fatal(err)
	}
	if len(w.carried[0]) != 1 || w.carried[0][0].ObjectID != 0 || w.carried[0][0].Count != 5 {
		t.Fatalf("mixed pickup into the unbound equal element: %+v", w.carried[0])
	}
	if w.savedObjects.NextID != beforeID+2 || !w.savedObjects.item(10).Retired || !w.savedObjects.item(beforeID).Retired || w.savedObjects.item(beforeID).Token.T08 != 1 {
		t.Fatal("pickup did not perform ordered one-unit split/stamp/retire")
	}
	form := mustMarshal(t, w)
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(form, mustMarshal(t, &cold)) {
		t.Fatal("ordinary native SAVE/LOAD changed bytes")
	}
	sameContainers := len(w.savedObjects.Containers) == 0 && len(cold.savedObjects.Containers) == 0 ||
		reflect.DeepEqual(w.savedObjects.Containers, cold.savedObjects.Containers)
	if !reflect.DeepEqual(w.savedObjects.Items, cold.savedObjects.Items) || !sameContainers {
		t.Fatal("ordinary native SAVE/LOAD changed current object rows")
	}
}

func TestSavedItemOperations1115ExactImportRejectsWrongOrdinalAtomically(t *testing.T) {
	w, r, sacks := unboundGoldWorld1115(t)
	owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}
	item := objectItem1115(10, owner, 1)
	r.Items = []SavedItemObject{item}
	r.Containers = append(r.Containers, SavedObjectContainer{Owner: owner, Present: true, Items: []SavedObjectID{10}})
	value := item.Value.Instance()
	value.ObjectID = 0
	w.carried[0] = []ItemStack{StackItem(value, 1)}
	w.recomputeLoad(0)
	before := mustMarshal(t, w)
	if err := w.ImportSavedObjects(r, sacks, SavedObjectBinding{ID: 10, Owner: owner, Index: 1, Value: item.Value}); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("wrong exact ordinal was recovered by equal-value lookup or changed world")
	}
}

func bindOperationsPack1115(t *testing.T, w *World, i int) *SavedObjects {
	return bindOperationsPack(t, w, i)
}

func bindOperationsPack(t *testing.T, w *World, i int) *SavedObjects {
	t.Helper()
	r := &SavedObjects{Version: 1, NextID: 100}
	owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: w.entities[i].ID}
	c := SavedObjectContainer{Owner: owner, Present: true, InsertIndex: uint32(len(w.carried[i])), Accumulator: w.containerWeight(i)}
	if a := w.entities[i].ActorLoad; a.Present {
		c.Present, c.InsertIndex, c.Accumulator = a.ContainerPresent, a.InsertIndex, a.Accumulator
	}
	var bindings []SavedObjectBinding
	for k, value := range w.carried[i] {
		id := SavedObjectID(10 + k)
		row := objectItem1115(id, owner, value.Count)
		row.Value = value.Clone()
		row.Value.ObjectID = id
		row.Token.T1C = uint32(value.Price)
		if value.SourceEquipment.Class != 0 {
			row.Token.T0C = value.SourceEquipment.DefinitionRow
		}
		for _, effect := range value.Effects {
			eid := SavedObjectID(30 + len(r.Effects))
			row.Effects = append(row.Effects, eid)
			r.Effects = append(r.Effects, SavedEffectObject{ID: eid, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: effect})
		}
		if value.SourceEquipment.Spell.Present {
			row.Spell = SavedObjectID(80 + len(r.Spells))
			r.Spells = append(r.Spells, SavedSpellObject{ID: row.Spell, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: value.SourceEquipment.Spell, This: 0xfedcba98})
		}
		r.Items = append(r.Items, row)
		c.Items = append(c.Items, id)
		bindings = append(bindings, SavedObjectBinding{ID: id, Owner: owner, Index: uint32(k), Value: row.Value})
	}
	r.Containers = []SavedObjectContainer{c}
	if err := w.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal(err)
	}
	return w.savedObjects
}

func operationsPack1115(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20}})
	w.carried[0] = []ItemStack{{Code: 0x204, Kind: 1, Price: 31, Count: 3, WeightPresent: true, Weight: 2}}
	w.recomputeLoad(0)
	bindOperationsPack1115(t, w, 0)
	return w
}

func reloadOperations1115(t *testing.T, w *World) *World {
	t.Helper()
	return reloadOperations(t, w)
}

func reloadOperations(t *testing.T, w *World) *World {
	t.Helper()
	form := mustMarshal(t, w)
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || !bytes.Equal(form, mustMarshal(t, &cold)) {
		t.Fatalf("native continuation: %v", err)
	}
	return &cold
}

func TestSavedItemOperations1115MoveEquipDropConsumeAndContinuation(t *testing.T) {
	w := operationsPack1115(t)
	if err := w.MoveCarried(7, 8, 0x204, 2); err != nil {
		t.Fatal(err)
	}
	if w.carried[0][0].ObjectID != 10 || w.carried[0][0].Count != 1 || w.carried[1][0].ObjectID == 10 || w.carried[1][0].Count != 2 {
		t.Fatal("partial transfer copied source identity")
	}
	w = reloadOperations1115(t, w)
	w.equip(1, 0, 4, 0)
	if w.equipment[1][3].ObjectID == 0 || w.equipment[1][3].ObjectID == w.carried[1][0].ObjectID {
		t.Fatal("equip split did not mint one independently owned object")
	}
	w = reloadOperations1115(t, w)
	w.dropFromEquipment(1, 4, 2, 1)
	if len(w.sacks) != 1 || w.sacks[0].ObjectID == 0 || !w.equipment[1][3].Empty() {
		t.Fatal("ground ownership was not connected")
	}
	w = reloadOperations1115(t, w)
	if err := w.TakeSack(8, 2, 1); err != nil {
		t.Fatal(err)
	}
	if len(w.carried[1]) != 1 || w.carried[1][0].Count != 2 || len(w.sacks) != 0 {
		t.Fatal("generated Sack pickup did not merge back")
	}
	w = reloadOperations1115(t, w)
	if !w.consumeCarriedUnit(1, 0) || w.carried[1][0].Count != 1 {
		t.Fatal("consumption did not split and retire")
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperations1115ScrollReserveRefundRelease(t *testing.T) {
	w := scrollWorld1090(t, 2, 3)
	bindOperationsPack1115(t, w, 0)
	if !w.beginScroll(0, 0, 2, 0, 0, false) {
		t.Fatal("bound scroll was not reserved")
	}
	id := w.scrollCasts[0].Item.ObjectID
	if id == 10 || !w.savedObjects.HasRoot(id, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: w.scrollCasts[0].Reservation}) || w.carried[0][0].Count != 2 {
		t.Fatal("reserved scroll reused pack identity")
	}
	w = reloadOperations1115(t, w)
	if !w.cancelScroll(0) || len(w.scrollCasts) != 0 || w.carried[0][0].Count != 3 || !w.savedObjects.item(id).Retired {
		t.Fatal("refund did not retain destination and retire detached object")
	}
	w = reloadOperations1115(t, w)
	if !w.beginScroll(0, 0, 2, 0, 0, false) {
		t.Fatal("second reservation failed")
	}
	id = w.scrollCasts[0].Item.ObjectID
	for n := 0; n < 100 && len(w.scrollCasts) != 0; n++ {
		Step(w, nil)
	}
	if len(w.scrollCasts) != 0 || w.carried[0][0].Count != 2 || !w.savedObjects.item(id).Retired {
		t.Fatal("release did not retire its detached scroll")
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperationsSharedChildSurvivesConsumption(t *testing.T) {
	w := operationsPack1115(t)
	w.carried[0][0].Effects = []ItemEffect{{Kind: 6, Operand: 5}}
	w.carried[0][0].Kind = 3
	w.savedObjects.item(10).Value = w.carried[0][0].Clone()
	w.savedObjects.item(10).Effects = []SavedObjectID{30}
	w.savedObjects.Effects = []SavedEffectObject{{ID: 30, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: w.carried[0][0].Effects[0], ExternalReferences: 1}}
	w.carried[0][0].Count = 1
	w.savedObjects.item(10).Value.Count = 1
	w.entities[0].HP = 1
	w.recomputeLoad(0)
	if !w.UseCarriedPotion(7, 0) || !w.savedObjects.item(10).Retired || w.savedObjects.effect(30).Retired || w.entities[0].HP <= 1 || len(w.carried[0]) != 0 {
		t.Fatal("consumption lost a surviving shared Effect or failed to apply")
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperations1115WeaponSpellExternalAndLateFailure(t *testing.T) {
	makeWorld := func() *World {
		item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
		item.Effects = []ItemEffect{{Kind: 41, Operand: 1 | 7<<16}}
		item.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 3, Range: 9, ManaCost: 5}
		w := sourceMutationWorld(t, item)
		w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
		w.entities[0].Reach = 1
		w.spells = []SpellRule{{ID: 1, MaxRange: 7, ManaCost: 3, Defensive: true}}
		bindOperationsPack1115(t, w, 0)
		return w
	}
	w := makeWorld()
	want := SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3, Defensive: 1}
	if !w.EquipSourceCarried(1, 0) || w.equipment[0][0].ObjectID != 10 || w.equipment[0][0].SourceEquipment.Spell != want {
		t.Fatal("whole Weapon equip did not retain Item and rebuild known Spell")
	}
	first := w.savedObjects.item(10).Spell
	if first == 80 || !w.savedObjects.spell(80).Retired || w.savedObjects.spell(first).Value != want || w.savedObjects.spell(first).Coverage.Unknown != SavedUnknownIdentity {
		t.Fatal("Spell lifecycle or exact constructor operands differ")
	}
	w = reloadOperations1115(t, w)
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	var receipt SourceEquipmentReceipt
	item, ok := w.UnequipSource(1, 1, false, SourceEquipmentOperation{Receipt: &receipt})
	if !ok || item.ObjectID != 10 || item.SourceEquipment.Spell.Present || !w.savedObjects.spell(first).Retired || receipt.Reservation == 0 || w.savedObjects.sessionHandle(10, receipt.Reservation) != receipt.Reservation {
		t.Fatal("external unequip lost owner or retained disposed nested Spell")
	}
	before := mustMarshal(t, w)
	wrong := item.Clone()
	wrong.Price++
	if w.EquipSourceItem(1, wrong, SourceEquipmentOperation{SessionHandle: receipt.Reservation}) || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("external owner was adopted by partial value matching")
	}
	w = reloadOperations1115(t, w)
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	if !w.EquipSourceItem(1, item, SourceEquipmentOperation{SessionHandle: receipt.Reservation}) || w.equipment[0][0].ObjectID != 10 || w.equipment[0][0].SourceEquipment.Spell != want || w.savedObjects.item(10).Spell <= first {
		t.Fatal("same exact external Item did not re-enter with a fresh Spell")
	}
	reloadOperations1115(t, w)
	for _, mode := range []string{"derive", "shared"} {
		w = makeWorld()
		if mode == "derive" {
			w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
				if s.Attack[16] != 0 {
					return s, fmt.Errorf("late equipment failure")
				}
				return s, nil
			})
		} else {
			w.savedObjects.spell(80).ExternalReferences = 1
		}
		before = mustMarshal(t, w)
		ok := w.EquipSourceCarried(1, 0)
		if mode == "derive" {
			if ok || !bytes.Equal(before, mustMarshal(t, w)) {
				t.Fatal("failed Weapon transaction moved roots, Spell, or NextID")
			}
		} else if !ok || w.savedObjects.spell(80).Retired || w.savedObjects.item(10).Spell == 80 || w.equipment[0][0].ObjectID != 10 {
			t.Fatal("Spell replacement destroyed its other live reference")
		}
	}
}

func TestSavedItemOperations1115DropAllAdoptsOrDrains(t *testing.T) {
	for _, existing := range []bool{false, true} {
		w := operationsPack1115(t)
		if existing {
			w.pourSack(1, 1, 19, []ItemInstance{PlainItem(0x401)})
		}
		w.dropAll(0)
		if len(w.carried[0]) != 0 || len(w.sacks) != 1 || w.sacks[0].ObjectID == 0 {
			t.Fatal("drop-all did not transfer whole container", existing)
		}
		c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: w.sacks[0].ObjectID})
		if !existing {
			if len(c.Items) != 1 || c.Items[0] != 10 || w.savedObjects.item(10).Value.Count != 3 || len(w.savedObjects.Items) != 1 {
				t.Fatal("new Sack did not adopt exact stack identity")
			}
		} else if len(c.Items) != 2 || c.Items[0] != 0 || c.Items[1] == 10 || !w.savedObjects.item(10).Retired || len(w.savedObjects.Items) != 3 || w.sacks[0].Gold != 19 {
			t.Fatal("existing Sack did not run repeated one-unit drain", c.Items)
		}
		w = reloadOperations1115(t, w)
		if err := w.TakeSack(7, 1, 1); err != nil {
			t.Fatal(err)
		}
		reloadOperations1115(t, w)
	}
}

func TestSavedItemOperations1115MixedSackSlotsAndBulkRestoreRefusal(t *testing.T) {
	w := itemOperationsWorld(t)
	w.pourSack(10, 20, 4, []ItemInstance{PlainItem(0x405)})
	c := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: goldObject1115})
	if !reflect.DeepEqual(c.Items, []SavedObjectID{10, 0}) {
		t.Fatal("mixed Sack slots were not recorded")
	}
	w = reloadOperations1115(t, w)
	c = w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: goldObject1115})
	c.Items = append(c.Items, 0)
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("Sack extra unbound slot was accepted")
	}
	w = operationsPack1115(t)
	before := mustMarshal(t, w)
	if w.ReplaceStock(Stock{ID: 7, Items: []uint16{0x777}}) || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("bulk restore overwrote existing object owner")
	}
	if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 7}}); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("original-only restore overwrote bound owner")
	}
}

func TestSavedItemOperations1115TerminalDropSuppressionAndFinalRemoval(t *testing.T) {
	for _, suppress := range []bool{false, true} {
		w := operationsPack1115(t)
		w.entities[0].SuppressCorpseLoot = suppress
		if !w.dropTerminalLoot(0) {
			t.Fatal("terminal item lifecycle refused", suppress)
		}
		if len(w.carried[0]) != 0 || (len(w.sacks) == 0) != suppress {
			t.Fatal("terminal owner population", suppress)
		}
		if (w.savedObjects.item(10).Retired) != suppress {
			t.Fatal("terminal disposal/adoption identity", suppress)
		}
		w = reloadOperations1115(t, w)
		if !w.remove([]EntityID{7}) || w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}) != nil {
			t.Fatal("removed actor retained container owner")
		}
		reloadOperations1115(t, w)
	}
	w := operationsPack1115(t)
	w.carried[0][0].Effects = []ItemEffect{{Kind: 6, Operand: 5}}
	w.savedObjects.item(10).Value = w.carried[0][0].Clone()
	w.savedObjects.item(10).Effects = []SavedObjectID{30}
	w.savedObjects.Effects = []SavedEffectObject{{ID: 30, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: w.carried[0][0].Effects[0], ExternalReferences: 1}}
	if !w.remove([]EntityID{7}) || !w.savedObjects.item(10).Retired || w.savedObjects.effect(30).Retired {
		t.Fatal("actor removal destroyed another owner's Effect")
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperations1115SourceWornDropAppliesLoadOnce(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	bindOperationsPack1115(t, w, 0)
	if !w.EquipSourceCarried(1, 0) {
		t.Fatal("source armor equip")
	}
	if w.entities[0].ActorLoad.OwnWeight != 2 {
		t.Fatal("equip fixture own weight")
	}
	w.dropFromEquipment(0, 7, w.entities[0].X, w.entities[0].Y)
	if !w.equipment[0][6].Empty() || len(w.sacks) != 1 || w.sacks[0].ItemInstances[0].ObjectID != 10 || w.entities[0].ActorLoad.OwnWeight != 0 || w.entities[0].ActorLoad.Accumulator != -2 {
		t.Fatal("source worn drop replayed load delta or lost item identity", w.entities[0].ActorLoad)
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperations1115ScriptMixedIngressGiveAllAndTake(t *testing.T) {
	w := operationsPack1115(t)
	w.runInstant(ScriptInstant{Op: ScriptInstantAddItem, Unit: 7, HasUnit: true, Item: 0x777, HasItem: true})
	if len(w.carried[0]) != 2 || w.carried[0][1].ObjectID != 0 {
		t.Fatal("script ingress invented identity")
	}
	w = reloadOperations1115(t, w)
	w.runInstant(ScriptInstant{Op: ScriptInstantGiveAll, Unit: 7, HasUnit: true, Unit2: 8, HasUnit2: true})
	if len(w.carried[0]) != 0 || len(w.carried[1]) != 2 || w.carried[1][0].ObjectID == 10 || w.carried[1][0].Count != 3 || w.carried[1][1].ObjectID != 0 || !w.savedObjects.item(10).Retired {
		t.Fatal("GiveAll failed repeated one-unit ownership drain")
	}
	w = reloadOperations1115(t, w)
	w.runInstant(ScriptInstant{Op: ScriptInstantTakeItem, Unit: 8, HasUnit: true, Item: 0x204, HasItem: true})
	if w.carried[1][0].Count != 2 {
		t.Fatal("script take did not consume one bound unit")
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantTakeItem, Unit: 8, HasUnit: true, Item: 0x777, HasItem: true})
	if len(w.carried[1]) != 1 {
		t.Fatal("script native slot removal failed")
	}
	reloadOperations1115(t, w)
}

func TestSavedItemOperations1115HugeUnboundTransferKeepsBulkQuantity(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{{ID: 7, HP: 20, MaxHP: 20}, {ID: 8, HP: 20, MaxHP: 20}})
	w.carried[0] = []ItemStack{StackItem(PlainItem(0x777), 1<<30)}
	if err := w.MoveCarried(7, 8, 0x777, 1<<29); err != nil {
		t.Fatal(err)
	}
	if len(w.carried[0]) != 1 || len(w.carried[1]) != 1 || w.carried[0][0].Count != 1<<29 || w.carried[1][0].Count != 1<<29 || w.carried[1][0].ObjectID != 0 || w.savedObjects != nil {
		t.Fatal("unbound bulk transfer invented ownership or lost quantity")
	}
}
