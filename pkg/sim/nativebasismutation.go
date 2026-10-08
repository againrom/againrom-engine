package sim

import "encoding/binary"

// UpdateNativeEquipmentBasis shares local raw events with fresh constructors.
// It changes no live sheet, inventory, spell child or actor admission.
func UpdateNativeEquipmentBasis(basis NativeActorBasis, item ItemInstance, operation NativeModifierEquipmentOperation, general int32, humanoid bool, class NativeClass) NativeActorBasis {
	if operation != NativeModifierAttach && operation != NativeModifierRemove {
		return basis
	}
	e := Entity{NativeBasis: basis, Humanoid: humanoid, NativeClass: class}
	e.Skill[0] = general
	var w World
	w.applyNativeEquipmentBasis(&e, item, operation)
	return e.NativeBasis
}

func nativeModifierEffectDelta(e *Entity, kind EffectKind, amount int32) {
	b := &e.NativeBasis
	if !b.ModifierPresent {
		return
	}
	if kind == EffectHealthRegeneration || kind == EffectManaRegeneration {
		at, value := 10, e.HealthRegeneration
		if kind == EffectManaRegeneration {
			at, value = 14, e.ManaRegeneration
		}
		binary.LittleEndian.PutUint16(b.Modifier[at:], uint16(value))
		b.ModifierKnown |= uint64(3) << at
		return
	}
	if amount == 0 {
		return
	}
	at, scale := -1, int32(1)
	switch kind {
	case EffectSpeed:
		at = 4
	case EffectScanRange:
		at, scale = 16, 256
	case EffectAbsorption:
		at = 44
	case EffectProtectionFire, EffectProtectionWater, EffectProtectionAir, EffectProtectionEarth:
		at = 48 + 2*int(kind-EffectProtectionFire)
	}
	if at < 0 {
		return
	}
	r := NativeModifierEquipmentUpdate{Value: b.Modifier}
	for n := range r.Known {
		r.Known[n] = b.ModifierByteKnown(n)
	}
	operand := uint16(amount * scale)
	r.wordDelta(at, [2]byte{byte(operand), byte(operand >> 8)}, [2]bool{true, true}, 1)
	b.Modifier = r.Value
	for n := at; n < at+2; n++ {
		if !r.Known[n] {
			b.ModifierKnown &^= uint64(1) << n
		}
	}
}

func (w *World) nativeEquipmentOperands(item ItemInstance, general int32) NativeModifierEquipmentOperands {
	o := NativeModifierEquipmentOperands{Equipment: item.SourceEquipment, General: uint16(general), GeneralKnown: true}
	if o.Equipment.Class == 0 {
		switch (item.Code >> 8) & 15 {
		case 1:
			o.Equipment.Class = SourceWeapon
		case 2:
			o.Equipment.Class = SourceShield
		case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
			o.Equipment.Class = SourceArmor
		}
		return o
	}
	o.AttackTypeKnown = o.Equipment.Definition.Present
	// This is the typed current value, never SAVE's class-only fallback.
	for n := range o.AttackKnown {
		o.AttackKnown[n] = o.Equipment.Class == SourceWeapon
	}
	for n := range o.DefenceKnown {
		o.DefenceKnown[n] = true
	}
	return o
}

func (w *World) applyNativeEquipmentBasis(e *Entity, item ItemInstance, operation NativeModifierEquipmentOperation) {
	if item.Empty() || e.ActorLoad.Source.Class != 0 {
		return
	}
	o := w.nativeEquipmentOperands(item, e.Skill[0])
	sign := 1
	if operation == NativeModifierRemove {
		sign = -1
		if o.Equipment.Class == SourceWeapon {
			nativeItemEffects(e, item, sign)
		}
	}
	if o.Equipment.Class != 0 {
		r := nativeModifierUpdate(e.NativeBasis)
		literals := NativeModifierEquipmentLiterals{MeleeRemovalClear: true,
			RangedRemovalClear: o.AttackTypeKnown && (o.Equipment.Definition.AttackType == 11 || o.Equipment.Definition.AttackType == 12)}
		update, err := UpdateNativeModifierEquipment(r.Value, r.Known, operation, o, literals)
		if err == nil {
			publishNativeModifier(&e.NativeBasis, update)
		}
	}
	if operation != NativeModifierRemove || o.Equipment.Class != SourceWeapon {
		nativeItemEffects(e, item, sign)
	}
}

func (w *World) applyEquipmentItemState(e *Entity, removed, added ItemInstance, table []SpellRule, observation ...*damageObservation) {
	w.applyNativeEquipmentChange(e, removed, added)
	applyEquipmentItemState(e, removed, added, table, observation...)
}

func UpdateNativeEquipmentChangeBasis(basis NativeActorBasis, removed, added ItemInstance, general int32, humanoid bool, class NativeClass) NativeActorBasis {
	e := Entity{NativeBasis: basis, Humanoid: humanoid, NativeClass: class}
	e.Skill[0] = general
	var w World
	w.applyNativeEquipmentChange(&e, removed, added)
	return e.NativeBasis
}

func (w *World) applyNativeEquipmentChange(e *Entity, removed, added ItemInstance) {
	oldRaw, newRaw := removed.SourceEquipment, added.SourceEquipment
	oldRaw.Spell, newRaw.Spell = SourceItemSpell{}, SourceItemSpell{}
	if removed.ObjectID == 0 || removed.ObjectID != added.ObjectID || oldRaw != newRaw {
		w.applyNativeEquipmentBasis(e, removed, NativeModifierRemove)
		w.applyNativeEquipmentBasis(e, added, NativeModifierAttach)
	} else if !sameItemEffects(removed.Effects, added.Effects) {
		nativeItemEffects(e, removed, -1)
		nativeItemEffects(e, added, 1)
	}
}

func sameItemEffects(a, b []ItemEffect) bool {
	if len(a) != len(b) {
		return false
	}
	for n := range a {
		if a[n] != b[n] {
			return false
		}
	}
	return true
}
