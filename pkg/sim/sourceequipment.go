package sim

import "fmt"

const (
	SourceWeapon uint8 = 1
	SourceArmor  uint8 = 2
	SourceShield uint8 = 3
)

// SourceEquipment retains the concrete saved object's producer operands.
// Class selects the definition collection; DefinitionRow is Token+0x0c.
// The definition pointer is rebound on LOAD from that row (ITEM-DEF-002),
// never from appearance Code or OwnKind. Attack is Weapon W52; Defence is
// Weapon W6A, Armor A52 or Shield S50. OwnKind is W50/A50, zero for Shield.
type SourceEquipment struct {
	Class, DefinitionRow, OwnKind uint8
	Attack                        [24]byte
	Defence                       [22]byte
	Definition                    SourceWeaponDefinition
	Spell                         SourceItemSpell
	EffectsUnsupported            bool
}

func (s SourceEquipment) Validate() error {
	if s.Class > SourceShield || s.Class == 0 && s != (SourceEquipment{}) {
		return fmt.Errorf("sim: invalid source equipment class/residue")
	}
	if s.Class != SourceWeapon && s.Attack != ([24]byte{}) || s.Class == SourceShield && s.OwnKind != 0 {
		return fmt.Errorf("sim: source equipment has another class's members")
	}
	if s.Class != SourceWeapon && s.Definition != (SourceWeaponDefinition{}) || !s.Definition.Present && s.Definition != (SourceWeaponDefinition{}) {
		return fmt.Errorf("sim: invalid source equipment definition residue")
	}
	if s.Class != SourceWeapon && s.Spell != (SourceItemSpell{}) || !s.Spell.Present && s.Spell != (SourceItemSpell{}) {
		return fmt.Errorf("sim: invalid source equipment Spell residue")
	}
	return nil
}

// SourceWeaponDefinition contains only the reached definition columns.
// Original LOAD binds the row; native SAVE retains the reached values as
// hashed item state, not arithmetic reconstructed from an appearance code.
// Missing rows remain unavailable for mutation.
type SourceWeaponDefinition struct {
	Present                 bool
	AttackType, Hands       int32
	Charge, Relax, Suitable int32
}

// SourceItemSpell is the Weapon-owned Spell, distinct from its kind41 Effect.
// Archive identity/aliasing remain outside the canonical value boundary.
type SourceItemSpell struct {
	Present              bool
	ID, Range, Defensive uint8
	ManaCost             uint16
}
