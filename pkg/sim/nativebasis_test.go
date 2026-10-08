package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func nativeBasisWorld(t *testing.T) *World {
	t.Helper()
	return mustWorldGrid(t, 17, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
		[]Entity{{ID: 0, HP: 20, MaxHP: 20, Speed: 10, Humanoid: true}, {ID: 3, X: 2, HP: 20, MaxHP: 20, Speed: 10}})
}

func TestNativeBasisOptionalFormAndExactCurrentAction(t *testing.T) {
	w := nativeBasisWorld(t)
	w.entities[1].AdmittedBookSpell = 6
	base, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var value NativeActorBasis
	value = value.WithBase([24]byte{1, 2, 3, 4}).WithModifier([64]byte{9, 8, 7, 6}).WithBody(0)
	value.ModifierKnown &^= uint64(7) << 37
	rows := []NativeActorBasisRecord{{0, value}, {3, (NativeActorBasis{}).WithBody(77)}}
	if err := w.RestoreNativeActorBases(rows); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	expected := bytes.Clone(base)
	previous := expected[0]
	expected[0] = 109
	expected = binary.LittleEndian.AppendUint32(expected, 2)
	for _, r := range rows {
		flag := byte(12)
		if r.ID == 0 {
			flag = 15
		}
		expected = binary.LittleEndian.AppendUint32(expected, uint32(r.ID))
		expected = append(expected, flag)
		expected = binary.LittleEndian.AppendUint32(expected, r.Basis.BaseKnown)
		expected = append(expected, r.Basis.Base[:]...)
		expected = binary.LittleEndian.AppendUint64(expected, r.Basis.ModifierKnown)
		expected = append(expected, r.Basis.Modifier[:]...)
		expected = binary.LittleEndian.AppendUint16(expected, r.Basis.Body)
	}
	expected = binary.LittleEndian.AppendUint32(expected, 4+2*107)
	expected = append(expected, previous, 'N', 'A', 'B', '1')
	if !bytes.Equal(raw, expected) {
		t.Fatal("native basis did not use the literal optional form")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatal(err, "native cold hash")
	}
	if cold.entities[0].NativeBasis != value || cold.entities[1].AdmittedBookSpell != 6 {
		t.Fatal("independent fields lost")
	}
	actions := w.Actions()
	if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != value {
		t.Fatal("current action basis missing")
	}
	plain := nativeBasisWorld(t)
	if err := plain.RestoreActions(actions, nil); err != nil || plain.entities[0].NativeBasis != value {
		t.Fatal(err, "action restore")
	}
	if err := cold.UnmarshalBinary(base); err != nil || cold.entities[0].NativeBasis.HasValues() {
		t.Fatal(err, "old absence default")
	}
}

func TestNativeBasisBatchAndMalformedFormAreAtomic(t *testing.T) {
	w := nativeBasisWorld(t)
	value := (NativeActorBasis{}).WithModifier([64]byte{1})
	if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{0, value}}); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	bad := value
	bad.BaseKnown = 1 << 24
	for _, rows := range [][]NativeActorBasisRecord{{{0, value}, {99, value}}, {{0, value}, {0, value}}, {{0, bad}}} {
		if err := w.RestoreNativeActorBases(rows); err == nil || w.Hash() != before {
			t.Fatal("bad batch changed current state", err)
		}
	}
	start := len(raw) - 9 - (4 + 107)
	for _, mutation := range []func([]byte){
		func(b []byte) { b[len(b)-1] = '2' },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 99) },
		func(b []byte) { b[start+8] = 16 },
		func(b []byte) { b[start+9+3] = 1 },
		func(b []byte) { b[len(b)-5] = 109 },
	} {
		wrong := bytes.Clone(raw)
		mutation(wrong)
		if err := w.UnmarshalBinary(wrong); err == nil || w.Hash() != before {
			t.Fatal("bad form changed state", err)
		}
	}
	actions := w.Actions()
	broken := value
	broken.ModifierPresent = false
	actions.Actors[0].Current.NativeBasis = &broken
	entities := w.Entities()
	if err := w.RestoreActions(actions, nil); err == nil || !reflect.DeepEqual(w.Entities(), entities) {
		t.Fatal("bad action changed state", err)
	}
}

func TestNativeBasisTracksNativeEffectHistoryWithoutAnInverse(t *testing.T) {
	w := nativeBasisWorld(t)
	var raw [64]byte
	binary.LittleEndian.PutUint16(raw[4:], 12)
	raw[40] = 99
	w.entities[0].NativeBasis = (NativeActorBasis{}).WithModifier(raw)
	landed, ok := w.applyEffectDelta(0, EffectSpeed, -20)
	if !ok || landed >= 0 {
		t.Fatal("clamped effect fixture did not land", landed)
	}
	got := binary.LittleEndian.Uint16(w.entities[0].NativeBasis.Modifier[4:])
	if got != uint16(12+landed) {
		t.Fatalf("current raw remainder did not follow actual native landing: %d want%d", got, uint16(12+landed))
	}
	if w.entities[0].NativeBasis.Modifier[40] != 99 {
		t.Fatal("unrelated history moved")
	}
	if !w.attachEffect(0, 3, SpellRule{ID: 1}, EffectSpeed, 7, 1, 1) {
		t.Fatal("attach fixture refused")
	}
	if !w.removeAttachedSpell(0, 1) {
		t.Fatal("remove fixture refused")
	}
	if binary.LittleEndian.Uint16(w.entities[0].NativeBasis.Modifier[4:]) != got {
		t.Fatal("effect removal lost remainder")
	}
	w.entities[0].NativeBasis.ModifierKnown &^= uint64(1) << 4
	w.applyEffectDelta(0, EffectSpeed, 3)
	if w.entities[0].NativeBasis.ModifierByteKnown(4) || w.entities[0].NativeBasis.ModifierByteKnown(5) {
		t.Fatal("unknown remainder was fabricated")
	}
}

func TestNativeBasisBodyFollowsActualCappedPermanentGain(t *testing.T) {
	w := nativeBasisWorld(t)
	w.entities[0].NativeBasis = (NativeActorBasis{}).WithBody(17)
	w.entities[0].PotionHeadroom[0] = 5
	w.carried[0] = []ItemStack{{Code: 1, Count: 1, Kind: 3, Effects: []ItemEffect{{Kind: 2, Operand: 7}}}}
	if !w.UseCarriedPotion(0, 0) {
		t.Fatal("potion fixture refused")
	}
	if w.entities[0].NativeBasis.Body != 22 || w.entities[0].PotionStats[0] != 5 {
		t.Fatal("native Body did not follow capped current gain", w.entities[0].NativeBasis.Body, w.entities[0].PotionStats)
	}
}

func TestNativeBasisEquipmentEventsUseCurrentRawInputs(t *testing.T) {
	w := nativeBasisWorld(t)
	var raw [64]byte
	raw[18], raw[32], raw[37], raw[38], raw[39], raw[42], raw[40] = 4, 5, 91, 92, 93, 6, 99
	w.entities[0].NativeBasis = (NativeActorBasis{}).WithModifier(raw)
	item := PlainItem(0x1101)
	item.SourceEquipment = SourceEquipment{Class: SourceWeapon, Definition: SourceWeaponDefinition{Present: true, AttackType: 2}}
	item.SourceEquipment.Attack[0], item.SourceEquipment.Attack[14], item.SourceEquipment.Defence[0] = 3, 2, 5
	item.SourceEquipment.Attack[19], item.SourceEquipment.Attack[20], item.SourceEquipment.Attack[21] = 8, 9, 10
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	w.equip(0, 0, 1, 0)
	b := w.entities[0].NativeBasis
	if b.Modifier[18] != 7 || b.Modifier[32] != 7 || b.Modifier[42] != 11 || b.Modifier[40] != 99 {
		t.Fatal("current raw equipment inputs did not follow equip", b)
	}
	if b.ModifierKnown != ^uint64(0) || b.Modifier[37] != 8 || b.Modifier[38] != 9 || b.Modifier[39] != 10 {
		t.Fatal("actual current triple did not replace old history", b)
	}
	w.unequip(0, 1)
	b = w.entities[0].NativeBasis
	if b.Modifier[18] != 4 || b.Modifier[32] != 5 || b.Modifier[42] != 6 || b.Modifier[37] != 0 || b.Modifier[38] != 0 || b.Modifier[39] != 0 || !b.ModifierByteKnown(37) {
		t.Fatal("bounded melee removal did not clear actor triple", b)
	}
}

func TestNativeBasisLiteralItemAndSpellOnlyUpdate(t *testing.T) {
	w := nativeBasisWorld(t)
	item := PlainItem(0x1101)
	item.ObjectID = 1
	item.SourceEquipment = SourceEquipment{Class: SourceWeapon, Definition: SourceWeaponDefinition{Present: true, AttackType: 2}}
	item.SourceEquipment.Attack[19], item.SourceEquipment.Attack[20], item.SourceEquipment.Attack[21] = 3, 4, 5
	w.savedObjects = &SavedObjects{Items: []SavedItemObject{{ID: 1, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: StackItem(item, 1)}}}
	o := w.nativeEquipmentOperands(item, 0)
	if !o.AttackKnown[19] || !o.AttackKnown[20] || !o.AttackKnown[21] {
		t.Fatal("literal current operands were hidden as constructor debt")
	}
	w.entities[0].NativeBasis = (NativeActorBasis{}).WithModifier([64]byte{})
	w.applyNativeEquipmentBasis(&w.entities[0], item, NativeModifierAttach)
	before := w.entities[0].NativeBasis
	changed := item.Clone()
	changed.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 4}
	w.applyEquipmentItemState(&w.entities[0], item, changed, nil)
	if w.entities[0].NativeBasis != before {
		t.Fatal("spell-only update replayed raw equipment events")
	}
}

func TestNativeModifierEquipmentWidthsAndKnowledgeDependencies(t *testing.T) {
	for _, class := range []uint8{SourceArmor, SourceShield} {
		var raw [64]byte
		var known [64]bool
		for n := range raw {
			raw[n], known[n] = byte(n*7), true
		}
		o := NativeModifierEquipmentOperands{Equipment: SourceEquipment{Class: class}}
		for n := range o.DefenceKnown {
			o.DefenceKnown[n], o.Equipment.Defence[n] = true, byte(n*9+1)
		}
		got, err := UpdateNativeModifierEquipment(raw, known, NativeModifierAttach, o, NativeModifierEquipmentLiterals{})
		if err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 16; n += 2 {
			want := binary.LittleEndian.Uint16(raw[42+n:]) + binary.LittleEndian.Uint16(o.Equipment.Defence[n:])
			if binary.LittleEndian.Uint16(got.Value[42+n:]) != want {
				t.Fatal("word width", class, n)
			}
		}
		for n := 16; n < 22; n++ {
			if got.Value[42+n] != raw[42+n]+o.Equipment.Defence[n] {
				t.Fatal("byte width", class, n)
			}
		}
		for n := 0; n < 42; n++ {
			if got.Value[n] != raw[n] || got.Touched[n] {
				t.Fatal("unrelated byte", n)
			}
		}
		back, err := UpdateNativeModifierEquipment(got.Value, got.Known, NativeModifierRemove, o, NativeModifierEquipmentLiterals{})
		if err != nil || back.Value != raw {
			t.Fatal("inverse full operand", err)
		}
		o.DefenceKnown[0] = false
		raw[42] = 0
		partial, err := UpdateNativeModifierEquipment(raw, known, NativeModifierAttach, o, NativeModifierEquipmentLiterals{})
		if err != nil || partial.Known[42] || !partial.Known[43] || partial.Value[42] != raw[42] {
			t.Fatal("independent high zero-carry knowledge", err, partial)
		}
	}
	var raw [64]byte
	var known [64]bool
	for n := range known {
		known[n], raw[n] = true, byte(n)
	}
	o := NativeModifierEquipmentOperands{Equipment: SourceEquipment{Class: SourceWeapon, Definition: SourceWeaponDefinition{Present: true, AttackType: 2}}, AttackTypeKnown: true}
	for n := range o.AttackKnown {
		o.AttackKnown[n] = true
	}
	for n := range o.DefenceKnown {
		o.DefenceKnown[n] = true
	}
	for n := 19; n < 22; n++ {
		o.AttackKnown[n] = false
	}
	partial, err := UpdateNativeModifierEquipment(raw, known, NativeModifierAttach, o, NativeModifierEquipmentLiterals{})
	if err != nil {
		t.Fatal(err)
	}
	for n := range known {
		if partial.Known[n] != (n < 37 || n >= 40) || n >= 37 && n < 40 && partial.Value[n] != raw[n] {
			t.Fatal("three absent bytes widened debt", n, partial)
		}
	}
}
