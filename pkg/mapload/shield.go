package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// WeaponAllowsShield reports whether a worn weapon leaves the second hand
// free for a shield. It does when the weapon resolves, is not ranged, and its
// authored Hands cell is not the decoded two-handed value 2; that population
// includes every bow and crossbow. A shield needs no weapon at all, so this is
// asked only about a weapon that is present. A deliberately incomplete
// synthetic table preserves existing loadouts; a complete production table
// refuses an unresolved code.
func WeaponAllowsShield(code data.ItemCode, t *Table) bool {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Weapons == nil {
		return true
	}
	w, err := data.WeaponFromCode(code, t.Shapes, t.Materials, t.Weapons)
	if err != nil {
		return false
	}
	if w.Ranged() {
		return false
	}
	hands, ok := weaponHands(&w, t)
	return ok && hands != 2
}

// NormalizeShieldLoadout moves a slot-2 shield that shares the hands with a
// weapon leaving no free hand to the same owner's unbounded container. A shield
// worn alone is valid. The complete instance is retained. It returns whether
// either collection changed.
func NormalizeShieldLoadout(worn *[sim.EquipSlots]sim.ItemInstance, carried *[]sim.ItemInstance, t *Table) bool {
	if worn == nil || carried == nil || worn[1].Empty() || worn[0].Empty() ||
		WeaponAllowsShield(data.ItemCode(worn[0].Code), t) {
		return false
	}
	*carried = append(*carried, worn[1].Clone())
	worn[1] = sim.ItemInstance{}
	return true
}

// NormalizePartyShieldLoadouts owns and normalizes a party at a construction
// or restore boundary. Returned members share no item slice with the input.
func NormalizePartyShieldLoadouts(party []PartyMember, t *Table) []PartyMember {
	out := OwnParty(party)
	for i := range out {
		NormalizePartyShieldLoadout(&out[i], t)
	}
	return out
}

// NormalizePartyShieldLoadout applies the invariant to one already-owned
// member record.
func NormalizePartyShieldLoadout(member *PartyMember, t *Table) bool {
	if member == nil {
		return false
	}
	equipped := MemberItemEquipment(*member, t)
	carried := MemberCarriedItems(*member, t)
	if !NormalizeShieldLoadout(&equipped, &carried, t) {
		return false
	}
	writePartyStock(member, equipped, carried)
	return true
}

func writePartyStock(member *PartyMember, equipped [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance) {
	if member == nil {
		return
	}
	var wornCodes [sim.EquipSlots]uint16
	for i := range equipped {
		wornCodes[i] = equipped[i].Code
	}
	carriedItems := cloneItemInstances(carried)
	carriedCodes := make([]uint16, len(carriedItems))
	for i := range carriedItems {
		carriedCodes[i] = carriedItems[i].Code
	}
	if member.Carry != nil {
		member.Carry.EquippedItems = cloneItemEquipment(equipped)
		member.Carry.Equipped = wornCodes
		member.Carry.ItemInstances = carriedItems
		member.Carry.Items = carriedCodes
		return
	}
	member.WornItems = cloneItemEquipment(equipped)
	member.Worn = wornCodes
	member.CarriedItems = carriedItems
	member.Carried = carriedCodes
}
