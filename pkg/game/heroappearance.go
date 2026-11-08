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

// heroMoment is when heroAppearance answers: at a hero drawable's creation
// (mission open, the open after a LOAD) or in play.
type heroMoment uint8

const (
	heroCreated heroMoment = iota
	heroInPlay
)

// heroAppearance is the one derivation of a hero's drawn body, body directory
// and class (HERO-APPEAR-042, ANIM-107). It reads the World's slots of id when
// w holds that actor, and member's record otherwise. Slot 1 is widened by the
// member's starting weapon while it has not materialized (DIV-111); the mage
// arm is the member's. The class the World stores for a hero is written from
// this answer, and the flight class rule reads that stored class (HERO-187).
//
// A loaded list's name no arm matches keeps the class held before: in play
// the stored one, else the creation class 1, or 23 for a mage (HERO-190). A
// held body at decay stage 1 holds 1, or 24 for a mage; past it, in play he
// keeps his stored class and at creation his slots derive it (HERO-191). The
// body follows the slots throughout.
func heroAppearance(w *sim.World, id sim.EntityID, member mapload.PartyMember, list data.BodyList, at heroMoment) heroLook {
	var look heroLook
	var eq data.Equipment
	var actor sim.Entity
	if w != nil {
		if slots, ok := w.Equipped(id); ok {
			eq, look.Held = equipmentFromSlots(slots), true
			actor, _ = w.Entity(id)
		}
	}
	if !look.Held {
		eq = mapload.EquipmentFromParty(member)
	}
	if occupied, _ := eq.Occupied(1); !occupied && member.Weapon != nil && !member.WeaponMaterialized {
		eq.SetCode(1, member.Weapon.Code)
	}
	look.Body, look.Dir, look.Class, look.Matched = data.HeroAppearance(list, eq, member.Mage, false)
	held, _ := data.HeroBodyClass(data.HeroBodyName(data.BodyUnarmed, false, member.Mage, false))
	if look.Held && at == heroInPlay {
		held = actor.Class
	}
	switch {
	case look.Held && actor.Decay == sim.DecayFallen:
		look.Class, _ = data.HeroBodyClass(data.HeroBodyName(data.BodyUnarmed, false, member.Mage, true))
	case look.Held && actor.Decay != sim.DecayNone && at == heroInPlay:
		look.Class = actor.Class
	case list.Len() != 0 && !data.HeroClassMatched(list, eq, member.Mage, false):
		look.Class = held
	}
	return look
}
