package mapload

import (
	"encoding/binary"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// MemberWeapon projects current equipment. Code-only values may retain an
// authored attachment; a materialized empty slot remains bare.
func MemberWeapon(p PartyMember, t *Table) *data.Weapon {
	return currentWeapon(MemberItemEquipment(p, t)[0], p.Weapon, p.WeaponMaterialized, t)
}

func currentWeapon(item sim.ItemInstance, fallback *data.Weapon, materialized bool, t *Table) *data.Weapon {
	if !item.Empty() {
		weapon := CurrentItemWeapon(item, t)
		if weapon != nil {
			weapon.SpellName, weapon.SpellPower = currentItemWeaponSpell(item, fallback, t)
			if fallback != nil && fallback.Code == data.ItemCode(item.Code) && fallback.Name != "" {
				weapon.Name = fallback.Name
			} else if t != nil {
				if name, ok := t.Names.NameFor(weapon.Code); ok {
					weapon.Name = name
				}
			}
		}
		return weapon
	}
	if materialized || fallback == nil {
		return nil
	}
	out := *fallback
	return &out
}

// CurrentItemWeapon projects the held object without reusing a prior weapon of
// the same code. An unavailable definition leaves this derived view absent;
// the owned item and its concrete saved fields are unchanged.
func CurrentItemWeapon(item sim.ItemInstance, t *Table) *data.Weapon {
	if item.Empty() {
		return nil
	}
	var out data.Weapon
	resolved := false
	if t != nil {
		var err error
		out, err = data.WeaponFromCode(data.ItemCode(item.Code), t.Shapes, t.Materials, t.Weapons)
		resolved = err == nil
	}
	s := item.SourceEquipment
	if s.Class == sim.SourceWeapon {
		if !s.Definition.Present {
			return nil
		}
		out.Code, out.Row = data.ItemCode(item.Code), int32(s.DefinitionRow)
		out.ToHit = int32(binary.LittleEndian.Uint16(s.Attack[:]))
		out.DamageBase, out.DamageSpread = int32(s.Attack[14]), int32(s.Attack[15])
		out.Defence = int32(binary.LittleEndian.Uint16(s.Defence[:]))
		out.Range = int32(s.OwnKind)
		out.AttackType, out.ChargeTime, out.RelaxTime = s.Definition.AttackType, s.Definition.Charge, s.Definition.Relax
	} else if s.Class != 0 || !resolved {
		return nil
	}
	out.Code = data.ItemCode(item.Code)
	if item.WeightPresent {
		out.Weight = int32(item.Weight)
	}
	out.SpellName, out.SpellPower = currentItemWeaponSpell(item, nil, t)
	return &out
}

func currentItemWeaponSpell(item sim.ItemInstance, fallback *data.Weapon, t *Table) (string, int32) {
	if id, power, present := item.CastSpell(); present {
		name := ""
		if t != nil && t.Spells != nil && id != 0 && int(id) < t.Spells.Len() {
			name = strings.ReplaceAll(t.Spells.EntryName(int(id)), " ", "_")
		}
		return name, power
	}
	code := item.Code
	item.Code = 0
	// Only the code-only representation lacks a current attachment value.
	if code != 0 && item.Empty() && fallback != nil && fallback.Code == data.ItemCode(code) {
		return fallback.SpellName, fallback.SpellPower
	}
	return "", 0
}
