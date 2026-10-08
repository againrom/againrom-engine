package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"fmt"
	"strings"
)

// CloneParty copies every mutable reference below a member. Identity values are
// preserved exactly; callers that establish a new ownership boundary use
// OwnParty below to fill legacy empty identities.
func CloneParty(in []PartyMember) []PartyMember {
	if len(in) == 0 {
		return nil
	}
	out := make([]PartyMember, len(in))
	for i := range in {
		out[i] = clonePartyMember(in[i])
	}
	return out
}

// OwnParty transfers a party into a canonical owner and guarantees that every
// member has one unique stable identity. A rider class also takes the rider
// speed term, repairing a member stored before Profile carried it.
func OwnParty(in []PartyMember) []PartyMember {
	out := CloneParty(in)
	NameParty(out)
	for i := range out {
		if data.RiderTypeID(out[i].Class) {
			out[i].Profile.Rider = true
		}
	}
	return out
}

// NameParty is OwnParty's identity half applied IN PLACE, for a caller that
// must keep the slice it was handed rather than a copy of it (1005 round 2,
// seventh pass, R1). A member that already carries a unique identity keeps it,
// so calling this twice on the same slice changes nothing the second time.
func NameParty(party []PartyMember) {
	used := make(map[string]bool, len(party))
	for i := range party {
		base := party[i].ID
		if base == "" {
			base = inferredPartyID(party[i], i)
		}
		id := base
		for suffix := 2; used[id]; suffix++ {
			id = fmt.Sprintf("%s:%d", base, suffix)
		}
		party[i].ID = id
		used[id] = true
	}
}

// DropHired removes every hired mercenary from a party, keeping the order of
// what is left. It is the party side of the end-of-mission cull.
//
// PARTY-MERC-007 is High that mercenaries do not cross a mission boundary, and
// that the reason is structural rather than a rule: they hang off the Player's
// groups, the Player is not serialized into the carried blob, and the client
// drops them by PARTY-CULL-004 because they carry +0x18c bit 4 and not bit 0.
// What crosses instead is the count per type, which is Town.mercenaryBoundary.
//
// THIS IS HALF OF ONE RULE AND MUST NOT BE CALLED ALONE. Culling without the
// per-type merge strands the pool at zero for every hired type -- the hire
// zeroes it and only the merge writes it again -- so the squad would be lost
// permanently rather than discharged.
//
// A JOINER IS NOT HIRED, so an actor a script handed to the player is kept
// here: Hired's own doc separates the two populations.
func DropHired(in []PartyMember) []PartyMember {
	out := make([]PartyMember, 0, len(in))
	for _, p := range in {
		if p.Hired() {
			continue
		}
		out = append(out, p)
	}
	return out
}

func clonePartyMember(in PartyMember) PartyMember {
	out := in
	if in.OriginalHuman != nil {
		h := *in.OriginalHuman
		h.Equipment = cloneItemEquipment(h.Equipment)
		h.Inventory = cloneItemInstances(h.Inventory)
		if h.Weapon != nil {
			weapon := *h.Weapon
			h.Weapon = &weapon
		}
		out.OriginalHuman = &h
	}
	out.Carried = append([]uint16(nil), in.Carried...)
	out.CarriedItems = cloneItemInstances(in.CarriedItems)
	out.Layers = append([]uint16(nil), in.Layers...)
	out.WornItems = cloneItemEquipment(in.WornItems)
	if in.Weapon != nil {
		weapon := *in.Weapon
		out.Weapon = &weapon
	}
	if in.Carry != nil {
		carry := *in.Carry
		carry.Items = append([]uint16(nil), in.Carry.Items...)
		carry.ItemInstances = cloneItemInstances(in.Carry.ItemInstances)
		carry.EquippedItems = cloneItemEquipment(in.Carry.EquippedItems)
		if in.Carry.NativeHistory != nil {
			history := *in.Carry.NativeHistory
			carry.NativeHistory = &history
		}
		if in.Carry.LiveLoad != nil {
			state := *in.Carry.LiveLoad
			carry.LiveLoad = &state
		}
		if in.Carry.OrderedStacks != nil {
			carry.OrderedStacks = make([]sim.ItemStack, len(in.Carry.OrderedStacks))
		}
		for i, stack := range in.Carry.OrderedStacks {
			carry.OrderedStacks[i] = stack.Clone()
		}
		out.Carry = &carry
	}
	if in.Saved != nil {
		saved := *in.Saved
		out.Saved = &saved
	}
	if in.PotionEffect != nil {
		effect := *in.PotionEffect
		out.PotionEffect = &effect
	}
	return out
}

func inferredPartyID(p PartyMember, index int) string {
	switch {
	case p.StartingHero:
		return "hero"
	case p.CompanionNPC != 0:
		return fmt.Sprintf("npc:%d", p.CompanionNPC)
	case p.MercenaryType != 0:
		return fmt.Sprintf("mercenary:%d:%d", p.MercenaryType, index+1)
	case p.PlayerCharacter && p.Name != "":
		return "player:" + stablePartyName(p.Name)
	default:
		return fmt.Sprintf("temporary:%d", index+1)
	}
}

func stablePartyName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.Join(strings.Fields(name), "-")
	if name == "" {
		return "unnamed"
	}
	return name
}
