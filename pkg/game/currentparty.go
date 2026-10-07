package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Text from the installed locale is byte text, not necessarily UTF-8. JSON's
// string replacement must not change a current character or weapon name.
type currentPartyMember struct {
	Entity                                   sim.EntityID
	Member                                   *mapload.PartyMember    `json:",omitempty"`
	ID, Name, WeaponName, OriginalWeaponName []byte                  `json:",omitempty"`
	Policy                                   *currentPartyPolicy     `json:",omitempty"`
	Base                                     *currentPartyBase       `json:",omitempty"`
	Human                                    *currentPartyHuman      `json:",omitempty"`
	Weapon                                   *currentPartyWeapon     `json:",omitempty"`
	City                                     *currentCityPartyPolicy `json:",omitempty"`
	Template                                 *currentPartyTemplate   `json:",omitempty"`
	ordinary                                 *sav.DocumentCharacter
}

type currentPartyHuman struct {
	Retired bool
}

func captureCurrentParty(id sim.EntityID, p mapload.PartyMember) currentPartyMember {
	p = mapload.CloneParty([]mapload.PartyMember{p})[0]
	out := currentPartyMember{Entity: id, Member: &p, ID: []byte(p.ID), Name: []byte(p.Name)}
	out.Member.ID, out.Member.Name = "", ""
	if out.Member.Weapon != nil {
		out.WeaponName = []byte(out.Member.Weapon.Name)
		out.Member.Weapon.Name = ""
	}
	if out.Member.OriginalHuman != nil && out.Member.OriginalHuman.Weapon != nil {
		out.OriginalWeaponName = []byte(out.Member.OriginalHuman.Weapon.Name)
		out.Member.OriginalHuman.Weapon.Name = ""
	}
	return out
}

func (p currentPartyMember) restore() mapload.PartyMember {
	var out mapload.PartyMember
	if p.Member != nil {
		out = mapload.CloneParty([]mapload.PartyMember{*p.Member})[0]
	} else if p.Policy != nil {
		out = p.Policy.member()
	}
	out.ID, out.Name = string(p.ID), string(p.Name)
	if out.Weapon != nil {
		out.Weapon.Name = string(p.WeaponName)
	}
	if out.OriginalHuman != nil && out.OriginalHuman.Weapon != nil {
		out.OriginalHuman.Weapon.Name = string(p.OriginalWeaponName)
	}
	return out
}
