package sim

import "encoding/binary"

func nativeModifierUpdate(b NativeActorBasis) NativeModifierEquipmentUpdate {
	r := NativeModifierEquipmentUpdate{Value: b.Modifier}
	for n := range r.Known {
		r.Known[n] = b.ModifierByteKnown(n)
	}
	return r
}

func publishNativeModifier(b *NativeActorBasis, r NativeModifierEquipmentUpdate) {
	var known uint64
	for n, value := range r.Known {
		if value {
			known |= uint64(1) << n
		}
	}
	if b.ModifierPresent || known != 0 {
		b.ModifierPresent, b.ModifierKnown, b.Modifier = true, known, r.Value
	}
}

// ITEM-EFFDISP-075, ITEM-EFFARM-146.
func nativeItemEffects(e *Entity, item ItemInstance, sign int) {
	if e.ActorLoad.Source.Class != 0 {
		return
	}
	for _, effect := range item.Effects {
		if effect.Kind == 2 && e.NativeBasis.BodyPresent {
			e.NativeBasis.BodyKnown = false
		}
		r := nativeModifierUpdate(e.NativeBasis)
		if effect.Kind > 49 || !r.Known[4] || !r.Known[5] {
			r.invalidate(4, 6)
		} else if int16(binary.LittleEndian.Uint16(r.Value[4:])) > 24 {
			r.clear(4, 6)
		}
		admitted := effect.Mode == 0 || effect.Mode == 1 || effect.Mode == 2 || effect.Mode == 4 || effect.Mode == 8
		amount := itemEffectScalar(effect)
		word := func(at int, scale int32) {
			operand := uint16(amount * scale)
			r.wordDelta(at, [2]byte{byte(operand), byte(operand >> 8)}, [2]bool{admitted, admitted}, sign)
		}
		floor := func(at int) {
			if !r.Known[at] || !r.Known[at+1] {
				r.invalidate(at, at+2)
			} else if int16(binary.LittleEndian.Uint16(r.Value[at:])) < 0 {
				r.clear(at, at+2)
			}
		}
		gatedWord := func(at int, fighter bool) {
			if !e.NativeClass.Present {
				r.invalidate(at, at+2)
			} else if e.NativeClass.Fighter == fighter {
				word(at, 1)
			}
		}
		switch kind := effect.Kind; {
		case kind >= 2 && kind <= 5:
			if effect.Mode&8 == 0 {
				r.byteDelta([]int{0, 0, 0, 2, 1, 3}[kind], byte(amount), admitted, sign)
			}
		case kind == 7:
			word(8, 1)
		case kind == 8:
			word(10, 1)
		case kind == 10:
			gatedWord(12, false)
		case kind == 11:
			gatedWord(14, false)
		case kind == 12:
			word(18, 1)
		case kind == 13 || kind == 43 || kind == 49:
			r.byteDelta(32, byte(amount), admitted, sign)
		case kind == 14:
			r.byteDelta(33, byte(amount), admitted, sign)
		case kind == 15 && e.Humanoid:
			word(42, 1)
		case kind == 16 && e.Humanoid:
			word(44, 1)
			floor(44)
		case kind == 17 && e.Humanoid:
			word(4, 1)
		case kind == 19 && e.Humanoid:
			word(16, 256)
		case kind >= 20 && kind <= 25:
			at := 46 + 2*int(kind-20)
			word(at, 1)
			floor(at)
		case kind >= 26 && kind <= 31:
			gatedWord(20+2*int(kind-26), true)
		case kind >= 32 && kind <= 37:
			gatedWord(20+2*int(kind-32), false)
		case kind >= 44 && kind <= 48:
			r.store(37, byte(effect.Operand), admitted)
			r.store(38, byte(effect.Operand>>8), admitted)
			r.store(39, kind-43, admitted)
		}
		publishNativeModifier(&e.NativeBasis, r)
	}
}
