package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type currentPartyWeapon struct {
	Fallback  *data.Weapon `json:",omitempty"`
	NameCode  uint16       `json:",omitempty"`
	Name      []byte       `json:",omitempty"`
	SpellName []byte       `json:",omitempty"`
}

func capturePartyWeapon(p mapload.PartyMember, item sim.ItemInstance) *currentPartyWeapon {
	out := &currentPartyWeapon{}
	if item.Empty() && !p.WeaponMaterialized && p.Weapon != nil {
		v := *p.Weapon
		out.Name, out.SpellName = []byte(v.Name), []byte(v.SpellName)
		v.Name, v.SpellName = "", ""
		out.Fallback = &v
	}
	return out
}

func captureNamedPartyWeapon(p mapload.PartyMember, item sim.ItemInstance) *currentPartyWeapon {
	out := capturePartyWeapon(p, item)
	if !item.Empty() && p.Weapon != nil && uint16(p.Weapon.Code) == item.Code && p.Weapon.Name != "" {
		out.NameCode, out.Name = item.Code, []byte(p.Weapon.Name)
	}
	return out
}

func canonicalMageWeapon(weapon *data.Weapon, mage bool, t *mapload.Table) *data.Weapon {
	if !mage || weapon == nil || t == nil {
		return weapon
	}
	staff, err := data.ResolveWeapon(mageWeaponName, t.Shapes, t.Materials, t.Weapons)
	if err == nil && staff.Code == weapon.Code && staff.SpellName == weapon.SpellName && staff.SpellPower == weapon.SpellPower {
		return &staff
	}
	return weapon
}
func (p currentPartyWeapon) restore(item sim.ItemInstance, materialized bool, t *mapload.Table) *data.Weapon {
	if !item.Empty() {
		out := mapload.CurrentItemWeapon(item, t)
		if out != nil {
			if p.NameCode != 0 && p.NameCode == item.Code && len(p.Name) != 0 {
				out.Name = string(p.Name)
			} else if t != nil {
				if name, ok := t.Names.NameFor(out.Code); ok {
					out.Name = name
				}
			}
		}
		return out
	}
	if materialized || p.Fallback == nil {
		return nil
	}
	out := *p.Fallback
	out.Name, out.SpellName = string(p.Name), string(p.SpellName)
	return &out
}
