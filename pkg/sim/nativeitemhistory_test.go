package sim

import (
	"encoding/binary"
	"testing"
)

func nativeItemHistoryWorld(t *testing.T, item ItemInstance, raw [64]byte) *World {
	t.Helper()
	w := nativeBasisWorld(t)
	w.entities[0].NativeBasis = (NativeActorBasis{}).WithModifier(raw)
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	return w
}

func nativeHistoryItem(class uint8, effects ...ItemEffect) ItemInstance {
	code := uint16(0x1201)
	if class == SourceWeapon {
		code = 0x1101
	}
	item := PlainItem(code)
	item.SourceEquipment = SourceEquipment{Class: class}
	item.Effects = effects
	return item
}

func TestNativeItemEffectsUpdateKnownHistoryThroughEquipmentCommands(t *testing.T) {
	var raw [64]byte
	binary.LittleEndian.PutUint16(raw[18:], 0x1234)
	binary.LittleEndian.PutUint16(raw[4:], 12)
	raw[40] = 0x99
	item := nativeHistoryItem(SourceShield, ItemEffect{Kind: 12, Operand: 3})
	w := nativeItemHistoryWorld(t, item, raw)
	for _, step := range []struct {
		command Command
		want    uint16
	}{{Equip(0, 0, 2), 0x1237}, {Unequip(0, 2), 0x1234}} {
		Step(w, []Command{step.command})
		b := w.entities[0].NativeBasis
		if !b.ModifierByteKnown(18) || !b.ModifierByteKnown(19) || binary.LittleEndian.Uint16(b.Modifier[18:]) != step.want || b.Modifier[40] != 0x99 {
			t.Fatalf("command %d lost known raw history: word=%04x known=%016x", step.command.Kind, binary.LittleEndian.Uint16(b.Modifier[18:]), b.ModifierKnown)
		}
	}
}

func TestNativeItemHistoryFollowsEffectAndWeaponOrder(t *testing.T) {
	for _, class := range []uint8{SourceShield, SourceWeapon} {
		t.Run(string(rune('0'+class)), func(t *testing.T) {
			item := nativeHistoryItem(class, ItemEffect{Kind: 44, Operand: 0x0907}, ItemEffect{Kind: 45, Operand: 0x0b08})
			if class == SourceWeapon {
				item.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: 2}
				item.SourceEquipment.Attack[19], item.SourceEquipment.Attack[20], item.SourceEquipment.Attack[21] = 31, 32, 33
			}
			w := nativeItemHistoryWorld(t, item, [64]byte{})
			slot := EquipSlot(2)
			if class == SourceWeapon {
				slot = 1
			}
			Step(w, []Command{Equip(0, 0, slot)})
			b := w.entities[0].NativeBasis
			if b.Modifier[37] != 8 || b.Modifier[38] != 11 || b.Modifier[39] != 2 || b.ModifierKnown&(7<<37) != 7<<37 {
				t.Fatal("ordered Effects did not follow direct attachment", b)
			}
			Step(w, []Command{Unequip(0, slot)})
			b = w.entities[0].NativeBasis
			want := [3]byte{8, 11, 2}
			if class == SourceWeapon {
				want = [3]byte{}
			}
			if [3]byte{b.Modifier[37], b.Modifier[38], b.Modifier[39]} != want || b.ModifierKnown&(7<<37) != 7<<37 {
				t.Fatal("removal lost class-specific Effect/direct ordering", b, want)
			}
		})
	}
}

func TestNativeWeaponRemovalClearsSupportedElementalSelector(t *testing.T) {
	for _, attackType := range []int32{5, 11, 12} {
		item := nativeHistoryItem(SourceWeapon)
		item.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: attackType}
		item.SourceEquipment.Attack[0], item.SourceEquipment.Attack[14], item.SourceEquipment.Attack[15] = 11, 10, 7
		item.SourceEquipment.Defence[0] = 13
		var raw [64]byte
		raw[32], raw[33], raw[37], raw[38], raw[39], raw[40] = 250, 247, 3, 2, 4, 99
		binary.LittleEndian.PutUint16(raw[18:], 5)
		binary.LittleEndian.PutUint16(raw[42:], 65532)
		w := nativeItemHistoryWorld(t, item, raw)
		w.equipment[0][0], w.carried[0] = item, nil
		Step(w, []Command{Unequip(0, 1)})
		b := w.entities[0].NativeBasis
		wantPhysical, wantElemental := [3]uint16{250, 247, 65532}, [2]byte{249, 251}
		if attackType == 5 {
			wantPhysical, wantElemental = [3]uint16{240, 240, 65519}, [2]byte{}
		}
		if [3]uint16{uint16(b.Modifier[32]), uint16(b.Modifier[33]), binary.LittleEndian.Uint16(b.Modifier[42:])} != wantPhysical ||
			[2]byte{b.Modifier[37], b.Modifier[38]} != wantElemental || b.Modifier[39] != 0 || !b.ModifierByteKnown(39) ||
			binary.LittleEndian.Uint16(b.Modifier[18:]) != 65530 || b.Modifier[40] != 99 {
			t.Fatalf("attack type %d removal lost local operands/order: %+v", attackType, b)
		}
	}
}

func TestNativeItemHistoryAvailabilityAndDispatcherGates(t *testing.T) {
	item := nativeHistoryItem(SourceShield, ItemEffect{Kind: 12, Operand: 3})
	w := nativeItemHistoryWorld(t, item, [64]byte{})
	w.entities[0].NativeBasis.ModifierKnown &^= 1 << 18
	Step(w, []Command{Equip(0, 0, 2)})
	b := w.entities[0].NativeBasis
	if b.ModifierByteKnown(18) || b.ModifierByteKnown(19) || !b.ModifierByteKnown(40) {
		t.Fatal("unknown arithmetic preimage supplied a known word or erased unrelated bytes", b)
	}
	item = nativeHistoryItem(SourceShield, ItemEffect{Kind: 44, Operand: 0x0907})
	w = nativeItemHistoryWorld(t, item, [64]byte{})
	w.entities[0].NativeBasis = NativeActorBasis{}
	Step(w, []Command{Equip(0, 0, 2)})
	b = w.entities[0].NativeBasis
	if !b.ModifierPresent || b.ModifierKnown != 7<<37 || [3]byte{b.Modifier[37], b.Modifier[38], b.Modifier[39]} != [3]byte{7, 9, 1} {
		t.Fatal("literal assignment needs no earlier component, but establishes only its own bytes", b)
	}
	for _, control := range []struct {
		name    string
		class   NativeClass
		effects []ItemEffect
		known   bool
		want    uint16
	}{
		{"unknown class", NativeClass{}, []ItemEffect{{Kind: 26, Operand: 3}}, false, 0},
		{"fighter opposite gate", NativeClass{Present: true, Fighter: true}, []ItemEffect{{Kind: 32, Operand: 3}}, true, 0},
		{"mage matching gate", NativeClass{Present: true}, []ItemEffect{{Kind: 32, Operand: 3}}, true, 3},
		{"unsupported mode", NativeClass{Present: true}, []ItemEffect{{Kind: 32, Mode: 3, Operand: 3}}, false, 0},
	} {
		t.Run(control.name, func(t *testing.T) {
			w := nativeItemHistoryWorld(t, nativeHistoryItem(SourceShield, control.effects...), [64]byte{})
			w.entities[0].NativeClass = control.class
			Step(w, []Command{Equip(0, 0, 2)})
			b := w.entities[0].NativeBasis
			if b.ModifierByteKnown(20) != control.known || b.ModifierByteKnown(21) != control.known || binary.LittleEndian.Uint16(b.Modifier[20:]) != control.want || !b.ModifierByteKnown(40) {
				t.Fatal("class/mode gate fabricated history or erased an unrelated word", b)
			}
		})
	}
	item = nativeHistoryItem(SourceShield, ItemEffect{Kind: 17, Operand: 30}, ItemEffect{Kind: 41}, ItemEffect{Kind: 16, Operand: 0xffff})
	w = nativeItemHistoryWorld(t, item, [64]byte{})
	Step(w, []Command{Equip(0, 0, 2)})
	b = w.entities[0].NativeBasis
	if binary.LittleEndian.Uint16(b.Modifier[4:]) != 0 || binary.LittleEndian.Uint16(b.Modifier[44:]) != 0 || !b.ModifierByteKnown(4) || !b.ModifierByteKnown(44) {
		t.Fatal("empty arm prelude or signed absorption floor lost its literal zero", b)
	}
}

func TestNativeItemEffectOnlyChangeAndSpellOnlyChangeDoNotReplayFields(t *testing.T) {
	old := nativeHistoryItem(SourceWeapon, ItemEffect{Kind: 12, Operand: 3})
	old.ObjectID = 1
	old.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: 11}
	old.SourceEquipment.Attack[0] = 11
	newItem := old.Clone()
	newItem.Effects[0].Operand = 5
	var raw [64]byte
	binary.LittleEndian.PutUint16(raw[18:], 0x1234)
	w := nativeItemHistoryWorld(t, old, raw)
	w.entities[0].Skill[0] = 17
	w.applyEquipmentItemState(&w.entities[0], old, newItem, nil)
	b := w.entities[0].NativeBasis
	if binary.LittleEndian.Uint16(b.Modifier[18:]) != 0x1236 || !b.ModifierByteKnown(18) || !b.ModifierByteKnown(19) {
		t.Fatal("effect-only replacement replayed ranged direct assignment", b)
	}
	spellChange := newItem.Clone()
	spellChange.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 6}
	w.applyEquipmentItemState(&w.entities[0], newItem, spellChange, nil)
	if w.entities[0].NativeBasis != b {
		t.Fatal("Spell-only replacement replayed raw history")
	}
}

func TestNativeWeaponRemovalKeepsOtherKnownDefinitionDamageMembers(t *testing.T) {
	item := nativeHistoryItem(SourceWeapon)
	item.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: 13}
	item.SourceEquipment.Attack[0], item.SourceEquipment.Attack[14] = 11, 10
	var raw [64]byte
	raw[32], raw[33], raw[37], raw[38], raw[39], raw[42] = 1, 2, 3, 4, 5, 6
	binary.LittleEndian.PutUint16(raw[18:], 5)
	w := nativeItemHistoryWorld(t, item, raw)
	w.equipment[0][0], w.carried[0] = item, nil
	Step(w, []Command{Unequip(0, 1)})
	b := w.entities[0].NativeBasis
	if b.ModifierKnown != ^uint64(0) || [6]byte{b.Modifier[32], b.Modifier[33], b.Modifier[37], b.Modifier[38], b.Modifier[39], b.Modifier[42]} != [6]byte{1, 2, 3, 4, 5, 6} || binary.LittleEndian.Uint16(b.Modifier[18:]) != 65530 {
		t.Fatal("other signed kind removal changed damage members instead of bypassing them", b)
	}
}

func TestNativeItemHistoryRangedLiteralDoesNotNeedPriorSelector(t *testing.T) {
	item := nativeHistoryItem(SourceWeapon)
	item.SourceEquipment.Definition = SourceWeaponDefinition{Present: true, AttackType: 11}
	item.SourceEquipment.Attack[14], item.SourceEquipment.Attack[15] = 10, 7
	var raw [64]byte
	raw[37], raw[38], raw[39], raw[40] = 3, 2, 4, 99
	w := nativeItemHistoryWorld(t, item, raw)
	w.equipment[0][0], w.carried[0] = item, nil
	w.entities[0].NativeBasis.ModifierKnown &^= 7 << 37
	Step(w, []Command{Unequip(0, 1)})
	b := w.entities[0].NativeBasis
	if b.ModifierByteKnown(37) || b.ModifierByteKnown(38) || !b.ModifierByteKnown(39) || b.Modifier[39] != 0 || b.Modifier[37] != 3 || b.Modifier[38] != 2 || !b.ModifierByteKnown(40) {
		t.Fatal("removal literal fabricated arithmetic or required a prior selector", b)
	}
	w = nativeItemHistoryWorld(t, PlainItem(0x1101), raw)
	w.equipment[0][0], w.carried[0] = PlainItem(0x1101), nil
	Step(w, []Command{Unequip(0, 1)})
	b = w.entities[0].NativeBasis
	if b.ModifierByteKnown(37) || b.ModifierByteKnown(38) || b.ModifierByteKnown(39) || !b.ModifierByteKnown(40) {
		t.Fatal("class-only item supplied unavailable definition operands", b)
	}
	before := b
	Step(w, []Command{Unequip(0, 1)})
	if w.entities[0].NativeBasis != before {
		t.Fatal("empty current slot replayed a removal event")
	}
}

func TestNativeItemHistoryDropWornUsesActualRemovalEvent(t *testing.T) {
	item := nativeHistoryItem(SourceShield, ItemEffect{Kind: 12, Operand: 3})
	var raw [64]byte
	binary.LittleEndian.PutUint16(raw[18:], 0x1234)
	w := nativeItemHistoryWorld(t, item, raw)
	Step(w, []Command{Equip(0, 0, 2)})
	Step(w, []Command{DropWorn(0, 2, CellPoint{X: 1, Y: 1})})
	b := w.entities[0].NativeBasis
	if !w.equipment[0][1].Empty() || len(w.Sacks()) != 1 || binary.LittleEndian.Uint16(b.Modifier[18:]) != 0x1234 || !b.ModifierByteKnown(18) || !b.ModifierByteKnown(19) {
		t.Fatal("actual worn drop failed to remove one raw Effect", b, w.equipment[0][1], w.Sacks())
	}
}
