package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// heroLook is a hero's drawn body, its directory and his class.
type heroLook struct {
	Body  data.HeroBody
	Dir   string
	Class int32
	// Matched: the body name and the directory both resolved.
	Matched bool
	// Held: the World holds the actor, and his slots were read.
	Held bool
}

// heroAppearance is the one derivation of a hero's drawn body, body directory
// and class (HERO-APPEAR-042, ANIM-107). It reads the World's slots of id when
// w holds that actor, and member's record otherwise. Slot 1 is widened by the
// member's starting weapon while it has not materialized (DIV-111); the mage
// arm is the member's. The class the World stores for a hero is written from
// this answer, and the flight class rule reads that stored class.
func heroAppearance(w *sim.World, id sim.EntityID, member mapload.PartyMember, list data.BodyList) heroLook {
	var look heroLook
	var eq data.Equipment
	if w != nil {
		if slots, ok := w.Equipped(id); ok {
			eq, look.Held = equipmentFromSlots(slots), true
		}
	}
	if !look.Held {
		eq = mapload.EquipmentFromParty(member)
	}
	if occupied, _ := eq.Occupied(1); !occupied && member.Weapon != nil && !member.WeaponMaterialized {
		eq.SetCode(1, member.Weapon.Code)
	}
	look.Body, look.Dir, look.Class, look.Matched = data.HeroAppearance(list, eq, member.Mage, false)
	return look
}
