package sim

import "encoding/binary"

// The admitted item model has stable, ordered state0 Effect values, not an
// arbitrary original callback graph. ITEM-EFFDISP-075 and SAV-EQUIPEFFECT-553
// require the common derive tail even for kind41 and other empty local arms.
func (w *World) sourceEquipmentEffects(i int, item ItemInstance, sign int32) bool {
	for k := 0; k < len(item.Effects); k++ {
		if !w.sourceItemEffect(i, item.Effects[k], sign) {
			return false
		}
	}
	return true
}

func (w *World) sourceItemEffect(i int, effect ItemEffect, sign int32) bool {
	before := w.entities[i]
	s := w.entities[i].SourceNow()
	if s.Class == 0 || effect.Kind > 49 {
		return false
	}
	amount := itemEffectScalar(effect) * sign
	if int16(binary.LittleEndian.Uint16(s.Modifier[4:])) > 24 {
		binary.LittleEndian.PutUint16(s.Modifier[4:], 0)
	}
	word := func(off int) { sourceWordAdd(s.Modifier[:], off, amount) }
	floor := func(off int) {
		if int16(binary.LittleEndian.Uint16(s.Modifier[off:])) < 0 {
			binary.LittleEndian.PutUint16(s.Modifier[off:], 0)
		}
	}
	switch kind := effect.Kind; {
	case kind == 0 || kind >= 38 && kind <= 41:
		// Empty general arm; a derive is still reached below.
	case kind == 1:
		return false // actor Token value is outside this retained producer
	case kind >= 2 && kind <= 5:
		slot := [...]int{0, 0, 0, 2, 1, 3}[kind]
		s.Stats[slot] += uint16(amount)
		if effect.Mode&8 == 0 {
			s.Modifier[slot] += byte(amount)
		}
		s.Stats[slot] = uint16(min(int32(int16(s.Stats[slot])), 100))
	case kind == 6:
		s.Stats[8] = uint16(min(int32(int16(s.Stats[8]))+amount, int32(int16(s.Stats[9]))))
	case kind == 7:
		s.Stats[8] += uint16(amount)
		word(8)
	case kind == 8:
		word(10)
	case kind == 9:
		s.Stats[11] = uint16(min(int32(int16(s.Stats[11]))+amount, int32(int16(s.Stats[12]))))
	case kind == 10:
		if !s.Fighter {
			s.Stats[11] += uint16(amount)
			word(12)
		}
	case kind == 11:
		if !s.Fighter {
			word(14)
		}
	case kind == 12:
		word(18)
	case kind == 13 || kind == 43 || kind == 49:
		s.Modifier[32] += byte(amount)
	case kind == 14:
		s.Modifier[33] += byte(amount)
	case kind == 15:
		if s.Class == 2 {
			word(42)
		} else {
			sourceWordAdd(s.Defence[:], 0, amount)
			binary.LittleEndian.PutUint16(s.Defence[:], uint16(max(0, int32(int16(binary.LittleEndian.Uint16(s.Defence[:]))))))
		}
	case kind == 16:
		if s.Class == 2 {
			word(44)
			floor(44)
		} else {
			sourceWordAdd(s.Defence[:], 2, amount)
		}
	case kind == 17:
		if s.Class == 2 {
			word(4)
		} else {
			s.Stats[4] += uint16(amount)
		}
	case kind == 18:
		s.MoverSpeed += uint8(amount)
	case kind == 19:
		if s.Class == 2 {
			sourceWordAdd(s.Modifier[:], 16, amount<<8)
		} else {
			s.Sight += uint16(amount << 8)
		}
	case kind >= 20 && kind <= 25:
		off := 46 + 2*int(kind-20)
		word(off)
		floor(off)
	case kind >= 26 && kind <= 37:
		if kind <= 31 && s.Fighter {
			word(20 + 2*int(kind-26))
		}
		if kind >= 32 && !s.Fighter {
			word(20 + 2*int(kind-32))
		}
	case kind == 42:
		if s.HasSpellbook {
			if amount < 1 || amount > 28 {
				return false
			}
			if _, ok := w.findSpell(uint32(amount)); !ok {
				return false
			}
			LearnBookSpell(&w.entities[i], uint16(amount), w.spells)
		}
	case kind >= 44 && kind <= 48:
		// Both apply and remove COPY the packed original bytes. The signed
		// scalar computed above is not the source of these stores.
		s.Modifier[37], s.Modifier[38], s.Modifier[39] = byte(effect.Operand), byte(effect.Operand>>8), kind-43
	default:
		return false
	}
	w.publishSourceDerived(i, s)
	if !w.deriveSource(i) {
		return false
	}
	// Only the direct health arm is damage. A lower maximum that clamps the
	// current health, as a removed Body or maximum-health lift does, is not.
	if effect.Kind == 6 && amount < 0 {
		w.reportHealthLoss(before, i)
	}
	return true
}
