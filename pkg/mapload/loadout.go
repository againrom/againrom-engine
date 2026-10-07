package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// EquipmentFromParty returns the equipment this member owns at a mission
// boundary. Restored/carried equipment wins over the class-row starting set,
// including when the carried set is deliberately empty.
func EquipmentFromParty(p PartyMember) data.Equipment {
	slots := p.Worn
	if p.Carry != nil {
		if !itemEquipmentEmpty(p.Carry.EquippedItems) {
			for i := range slots {
				slots[i] = p.Carry.EquippedItems[i].Code
			}
		} else {
			slots = p.Carry.Equipped
		}
	} else if !itemEquipmentEmpty(p.WornItems) {
		for i := range slots {
			slots[i] = p.WornItems[i].Code
		}
	}
	return equipmentFromSlots(slots)
}

// equipmentFromSlots turns a bare worn-slot array into the data.Equipment
// ResolveEquipmentLoadout reads — the one conversion EquipmentFromParty above
// and blockFor's person arm (fromalm.go) both need, written once so the two
// callers cannot come to disagree about which slot number a code belongs to.
func equipmentFromSlots(slots [sim.EquipSlots]uint16) data.Equipment {
	var eq data.Equipment
	for slot, code := range slots {
		eq.SetCode(slot+1, data.ItemCode(code))
	}
	return eq
}

// ResolveEquipmentLoadout turns one owned equipment value into the loadout
// used by the derived-stat graph. It is shared by mission construction and
// live equip/unequip re-derivation, so opening equipment and changed equipment
// cannot use different armour arithmetic.
//
// fallback stands in for slot 1's own weapon in two different cases, and
// everEquipped is what tells them apart (hotfix b51b439+1). The first case
// is unconditional: when slot 1's code equals fallback's own code, the
// resolved weapon is fallback itself rather than a fresh
// data.WeaponFromCode lookup, because a packed code alone cannot carry
// everything the original fallback object can — a castSpell attachment
// among them (0139 spec D-14) — so re-resolving from the code would lose
// it even though the two name the same weapon. The second case is gated by
// everEquipped: an EMPTY slot 1 answers fallback only when everEquipped is
// false. A mission's starting weapon is a loader value that does not
// always reach the equipment array (0124's disclosed limit), so an empty
// slot 1 the caller has never seen occupied still means "wearing the
// starting weapon, not yet written into the array" and fallback is the
// right answer. Once the caller HAS seen slot 1 occupied at least once,
// an empty slot 1 means the weapon was taken off, and the answer is a bare
// loadout (Weapon: nil) instead — passing fallback there would re-arm an
// unequipped hero with the weapon he started the mission holding, which is
// the defect this parameter exists to close.
func ResolveEquipmentLoadout(eq data.Equipment, fallback *data.Weapon, everEquipped bool, t *Table) (data.Loadout, bool) {
	weapon := fallback
	code, _ := eq.Code(1)
	switch {
	case code != 0 && (fallback == nil || fallback.Code != code):
		if t == nil {
			return data.Loadout{}, false
		}
		resolved, err := data.WeaponFromCode(code, t.Shapes, t.Materials, t.Weapons)
		if err != nil {
			return data.Loadout{}, false
		}
		weapon = &resolved
	case code == 0 && everEquipped:
		weapon = nil
	}

	var mod data.EquipMod
	if t != nil && t.Shapes != nil && t.Materials != nil {
		mod = data.FoldWear(eq, t.Shapes, t.Materials, t.Shields, t.Armors)
	}
	return data.Loadout{Weapon: weapon, Mod: mod, Rules: t.rules()}, true
}

// PartyLoadout derives the held weapon from its current object. An absent,
// never-materialized starting weapon keeps the explicit fallback.
func PartyLoadout(p PartyMember, t *Table) data.Loadout {
	items := MemberItemEquipment(p, t)
	carried := MemberCarriedItems(p, t)
	NormalizeShieldLoadout(&items, &carried, t)
	loadout, _ := ResolveItemLoadout(items, p.Weapon, p.WeaponMaterialized, t)
	AddLayers(&loadout, MemberLayers(p, t), t)
	loadout.RotationSpeed = RotationSpeedBase(p.Hired(), p.HiredRotationSpeed, p.Class, t)
	return loadout
}

// ResolveItemLoadout derives current equipment without a same-code cache.
// The bool reports whether a held weapon's definition could be resolved.
func ResolveItemLoadout(items [sim.EquipSlots]sim.ItemInstance, fallback *data.Weapon, materialized bool, t *Table) (data.Loadout, bool) {
	weapon := currentWeapon(items[0], fallback, materialized, t)
	var mod data.EquipMod
	if t != nil && t.Shapes != nil && t.Materials != nil {
		mod = data.FoldWear(equipmentFromSlots(itemEquipmentCodes(items)), t.Shapes, t.Materials, t.Shields, t.Armors)
	}
	return data.Loadout{Weapon: weapon, Mod: mod, Rules: t.rules()}, items[0].Empty() || weapon != nil
}

// DIV-437
func RotationSpeedBase(hired bool, hiredBase, class int32, t *Table) int32 {
	if hired {
		if hiredBase > 0 {
			return hiredBase
		}
		// Legacy fallback: a hired member built before HiredRotationSpeed
		// existed. Reaches the wrong row for the same 36 of 52 TypeIDs this
		// function's own doc names; kept only so a save from before this
		// round does not regress further than it already had.
		c := t.humans()
		if i := data.FindHumanByType(c, class); i != data.NotFound {
			if d, err := data.NewHumanDef(c.EntryName(i), c.EntryParams(i)); err == nil {
				return d.RotationSpeed
			}
		}
	}
	return data.UnitDefaults().RotationSpeed
}

// equipmentSlots keeps the shared width anchored to the simulation's owned
// equipment record. It is compile-time evidence that data and sim still agree.
var _ [sim.EquipSlots]uint16 = [data.EquipSlots]uint16{}
