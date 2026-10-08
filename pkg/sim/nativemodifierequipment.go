package sim

import "fmt"

type NativeModifierEquipmentOperation uint8

const (
	NativeModifierAttach NativeModifierEquipmentOperation = 1
	NativeModifierRemove NativeModifierEquipmentOperation = 2
)

// Known masks describe independently supplied raw operands, including zero.
// A concrete Class or Definition.Present never fills the raw operand masks.
type NativeModifierEquipmentOperands struct {
	Equipment       SourceEquipment
	AttackKnown     [24]bool
	DefenceKnown    [22]bool
	AttackTypeKnown bool
	General         uint16
	GeneralKnown    bool
}

// These optional literals select the existing engine removal stores.
// They do not establish an original constructor or a later lifetime rule.
type NativeModifierEquipmentLiterals struct {
	MeleeRemovalClear  bool
	RangedRemovalClear bool
}

// Touched includes destinations whose result is unavailable.
// Invalidated includes each touched byte with an unavailable result, even
// if it was already unknown. Such bytes retain their old payload.
type NativeModifierEquipmentUpdate struct {
	Value                 [64]byte
	Known                 [64]bool
	Touched               [64]bool
	Invalidated           [64]bool
	DefinitionUnavailable bool
	DefinitionUnsupported bool
	RemovalLiteralDebt    bool
}

// UpdateNativeModifierEquipment applies only one item's local raw stores.
// It does not derive, walk Effects, choose a slot or change native admission.
// The caller supplies the exact ordered event and independent current inputs.
func UpdateNativeModifierEquipment(value [64]byte, known [64]bool, operation NativeModifierEquipmentOperation, operands NativeModifierEquipmentOperands, literals NativeModifierEquipmentLiterals) (NativeModifierEquipmentUpdate, error) {
	r := NativeModifierEquipmentUpdate{Value: value, Known: known}
	if operation != NativeModifierAttach && operation != NativeModifierRemove {
		return r, fmt.Errorf("sim: invalid native modifier equipment operation")
	}
	if operands.Equipment.Class < SourceWeapon || operands.Equipment.Class > SourceShield {
		return r, fmt.Errorf("sim: invalid native modifier equipment class")
	}
	sign := 1
	if operation == NativeModifierRemove {
		sign = -1
	}
	switch operands.Equipment.Class {
	case SourceWeapon:
		r.weapon(operation, operands, literals, sign)
	case SourceArmor, SourceShield:
		for off := 0; off < 16; off += 2 {
			r.wordDelta(42+off, [2]byte{operands.Equipment.Defence[off], operands.Equipment.Defence[off+1]}, [2]bool{operands.DefenceKnown[off], operands.DefenceKnown[off+1]}, sign)
		}
		for off := 16; off < 22; off++ {
			r.byteDelta(42+off, operands.Equipment.Defence[off], operands.DefenceKnown[off], sign)
		}
	}
	return r, nil
}

func (r *NativeModifierEquipmentUpdate) weapon(operation NativeModifierEquipmentOperation, operands NativeModifierEquipmentOperands, literals NativeModifierEquipmentLiterals, sign int) {
	d := operands.Equipment.Definition
	if !d.Present || !operands.AttackTypeKnown {
		r.DefinitionUnavailable = true
		r.invalidate(32, 34)
		r.invalidate(37, 40)
		r.invalidate(42, 44)
		if operation == NativeModifierAttach {
			r.invalidate(18, 20)
		} else {
			r.weaponToHit(operands, sign)
		}
		return
	}
	melee := d.AttackType < 10
	supported := d.AttackType >= 0 && d.AttackType <= 5 || d.AttackType == 11 || d.AttackType == 12
	if !supported {
		r.DefinitionUnsupported = true
		if operation == NativeModifierRemove && d.AttackType >= 10 {
			r.weaponToHit(operands, sign)
			return
		}
		// A known custom type still selects the existing body's destinations.
		// No unavailable branch result erases an unrelated raw member.
		if melee {
			r.invalidate(32, 34)
			r.invalidate(42, 44)
			if operation == NativeModifierRemove && literals.MeleeRemovalClear {
				r.clear(37, 40)
			} else {
				r.invalidate(37, 40)
				r.RemovalLiteralDebt = operation == NativeModifierRemove
			}
		} else {
			r.invalidate(37, 39)
			if operation == NativeModifierRemove && literals.RangedRemovalClear {
				r.clear(39, 40)
			} else {
				r.invalidate(39, 40)
				r.RemovalLiteralDebt = operation == NativeModifierRemove
			}
		}
		if operation == NativeModifierRemove {
			r.weaponToHit(operands, sign)
		} else {
			r.invalidate(18, 20)
		}
		return
	}
	if melee {
		r.byteDelta(32, operands.Equipment.Attack[14], operands.AttackKnown[14], sign)
		r.byteDelta(33, operands.Equipment.Attack[15], operands.AttackKnown[15], sign)
		r.wordDelta(42, [2]byte{operands.Equipment.Defence[0], operands.Equipment.Defence[1]}, [2]bool{operands.DefenceKnown[0], operands.DefenceKnown[1]}, sign)
		if operation == NativeModifierAttach {
			for off := 0; off < 3; off++ {
				r.store(37+off, operands.Equipment.Attack[19+off], operands.AttackKnown[19+off])
			}
		} else if literals.MeleeRemovalClear {
			r.clear(37, 40)
		} else {
			r.invalidate(37, 40)
			r.RemovalLiteralDebt = true
		}
	} else {
		r.byteDelta(37, operands.Equipment.Attack[14], operands.AttackKnown[14], sign)
		r.byteDelta(38, operands.Equipment.Attack[15], operands.AttackKnown[15], sign)
		if operation == NativeModifierAttach {
			r.store(39, uint8(d.AttackType-10), true)
		} else if literals.RangedRemovalClear {
			r.clear(39, 40)
		} else {
			r.invalidate(39, 40)
			r.RemovalLiteralDebt = true
		}
	}
	if operation == NativeModifierAttach && !melee {
		r.store(18, byte(operands.General), operands.GeneralKnown)
		r.store(19, byte(operands.General>>8), operands.GeneralKnown)
	} else {
		r.weaponToHit(operands, sign)
	}
}

func (r *NativeModifierEquipmentUpdate) weaponToHit(operands NativeModifierEquipmentOperands, sign int) {
	r.wordDelta(18, [2]byte{operands.Equipment.Attack[0], operands.Equipment.Attack[1]}, [2]bool{operands.AttackKnown[0], operands.AttackKnown[1]}, sign)
}

func (r *NativeModifierEquipmentUpdate) store(off int, value byte, known bool) {
	r.Touched[off] = true
	r.Invalidated[off] = !known
	r.Known[off] = known
	if known {
		r.Value[off] = value
	}
}

func (r *NativeModifierEquipmentUpdate) invalidate(first, end int) {
	for off := first; off < end; off++ {
		r.store(off, 0, false)
	}
}

func (r *NativeModifierEquipmentUpdate) clear(first, end int) {
	for off := first; off < end; off++ {
		r.store(off, 0, true)
	}
}

func (r *NativeModifierEquipmentUpdate) byteDelta(off int, operand byte, operandKnown bool, sign int) {
	known := r.Known[off] && operandKnown
	r.store(off, byte(int(r.Value[off])+sign*int(operand)), known)
}

// A high byte needs a known carry or borrow, not necessarily a known low
// result. A zero operand low byte always contributes no carry or borrow.
func (r *NativeModifierEquipmentUpdate) wordDelta(off int, operand [2]byte, operandKnown [2]bool, sign int) {
	low, high := r.Value[off], r.Value[off+1]
	lowKnown, highKnown := r.Known[off], r.Known[off+1]
	carry, carryKnown := 0, false
	if lowKnown && operandKnown[0] {
		if sign > 0 && int(low)+int(operand[0]) > 255 || sign < 0 && int(low)-int(operand[0]) < 0 {
			carry = 1
		}
		carryKnown = true
	} else if operandKnown[0] && operand[0] == 0 || lowKnown && (sign > 0 && low == 0 || sign < 0 && low == 255) {
		carryKnown = true
	}
	r.store(off, byte(int(low)+sign*int(operand[0])), lowKnown && operandKnown[0])
	r.store(off+1, byte(int(high)+sign*int(operand[1])+sign*carry), highKnown && operandKnown[1] && carryKnown)
}
