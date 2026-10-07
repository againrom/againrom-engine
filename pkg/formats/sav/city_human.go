package sav

import (
	"encoding/binary"
	"fmt"
)

// CityHumanFields are the fixed Human members reached by school training.
// Raw block names denote their source member, not an entire object or SAV.
// The writer requires exact arrays; the game owns permission to change them.
type CityHumanFields struct {
	Attack           [24]byte
	Base             [24]byte
	Defence          [22]byte
	Modifier         [64]byte
	ManaFloor, Sight uint16
	MoverSpeed       byte
	// nil preserves these independent fields for existing school/sale callers.
	Runtime *CityHumanRuntime
}

// CityHumanRuntime carries stored scalar values, not mover/action history.
type CityHumanRuntime struct {
	Reach, AttackCharge, AttackRelax uint8
	HealthHundredths, ManaHundredths uint8
}

// CityHuman supplies the independent inputs to a later derive. In particular
// inventory weight is the saved container total, not money or an item sum.
type CityHuman struct {
	Fields                CityHumanFields
	Fighter, HasSpellbook bool
	TypeID                uint16
	InventoryWeight       int32
	HasOwner              bool
	ManaReservePercent    uint32
	AttachedEffects       bool
}

func (p *CityProvenance) Human(identity uint32) (CityHuman, error) {
	if p == nil || p.document == nil {
		return CityHuman{}, fmt.Errorf("sav: missing city document")
	}
	actor := p.document.objects[p.characterSourceIndex[identity]]
	if actor == nil || actor.unit == nil || actor.class != "Human" {
		return CityHuman{}, fmt.Errorf("sav: city identity %#x is not a Human", identity)
	}
	u := actor.unit
	out := CityHuman{Fighter: u.scalar1[3]&4 == 0, HasSpellbook: u.spellbookFlag != 0,
		TypeID: binary.LittleEndian.Uint16(u.token[17:]), InventoryWeight: int32(u.containerTails[1]), AttachedEffects: len(u.effects) != 0}
	copy(out.Fields.Attack[:], u.rawA6)
	copy(out.Fields.Base[:], u.raw114)
	copy(out.Fields.Defence[:], u.rawBE)
	copy(out.Fields.Modifier[:], u.rawD4)
	out.Fields.ManaFloor = binary.LittleEndian.Uint16(u.scalar2[30:])
	out.Fields.Sight = binary.LittleEndian.Uint16(u.scalar2[32:])
	out.Fields.MoverSpeed = u.raw154[10]
	owner := binary.LittleEndian.Uint32(u.token[33:])
	if owner != 0 {
		for _, player := range p.document.players {
			if cityObjectIdentity(player) == owner {
				out.HasOwner = true
				out.ManaReservePercent = binary.LittleEndian.Uint32(player.player.fixed[39:43])
				break
			}
		}
		if !out.HasOwner {
			return CityHuman{}, fmt.Errorf("sav: city Human owner %#x does not resolve", owner)
		}
	}
	return out, nil
}

func applyCityHumanFields(u *cityUnit, fields CityHumanFields) {
	copy(u.rawA6, fields.Attack[:])
	copy(u.raw114, fields.Base[:])
	copy(u.rawBE, fields.Defence[:])
	copy(u.rawD4, fields.Modifier[:])
	binary.LittleEndian.PutUint16(u.scalar2[30:], fields.ManaFloor)
	binary.LittleEndian.PutUint16(u.scalar2[32:], fields.Sight)
	u.raw154[10] = fields.MoverSpeed
	if r := fields.Runtime; r != nil {
		u.scalar2[28], u.scalar2[29] = r.HealthHundredths, r.ManaHundredths
		u.scalar2[34], u.scalar2[39], u.scalar2[40] = r.Reach, r.AttackCharge, r.AttackRelax
	}
}
