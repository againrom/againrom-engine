package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// ObserveLoadedState captures the state a LOAD installed, for comparison across
// a conversion or a reload. A mission snapshot's Party is the entry roster used
// to reconstruct entity IDs and its Gold the town purse, so both are replaced by
// the running World's own current holdings.
func (f *FrontEnd) ObserveLoadedState(onMap bool) (Snapshot, error) {
	s, _, err := f.Snapshot(onMap)
	if err != nil || s.Mission == 0 {
		return s, err
	}
	if f.live == nil || f.live.world == nil || f.live.mission == nil || f.live.mission.state == nil {
		return Snapshot{}, fmt.Errorf("mission comparison requires the current World and roster bindings")
	}
	m := f.live.mission
	s.Gold = int(f.live.world.Purse(sim.SelfSlot))
	s.Party = mapload.CarryRoster(m.party, f.live.world, m.ids, m.state.Start.Roster)
	return s, nil
}

// roundTripMemberDiff compares the active holdings through the same precedence
// as gameplay: Carry before legacy Worn/Carried arrays. The chargen preview's
// direct Worn access is outside this observation of the loaded state.
func roundTripMemberDiff(b, a mapload.PartyMember) []string {
	var changed []string
	add := func(differs bool, field string) {
		if differs {
			changed = append(changed, field)
		}
	}
	add(a.Name != b.Name, "name")
	add(a.StartingHero != b.StartingHero, "starting-hero flag")
	add(a.CompanionNPC != b.CompanionNPC, "companion npc")
	add(a.MercenaryType != b.MercenaryType, "mercenary type")
	add(a.Hero != b.Hero, "hero record")
	add(a.KnownSpells != b.KnownSpells, "known spells")
	add(mapload.EquipmentFromParty(a) != mapload.EquipmentFromParty(b), "worn set")
	add(!reflect.DeepEqual(roundTripPack(b), roundTripPack(a)), "carried pack")
	return changed
}

// roundTripPack is the pack this member actually carries, reduced to the item
// itself: Code, Kind, Effects and Price.
//
// FOUR FIELDS ARE CLEARED, AND EACH FOR A MEASURED REASON. ObjectID is a live
// object handle rather than an item the player owns, and returnItems
// (originalcity_return.go) clears it for that same reason before its own
// comparison. Weight, WeightPresent and SourceEquipment are the shipped
// table's answer FOR A CODE, not state the save carries: a member loaded from
// a legacy snapshot reaches this comparison with Weight 0, WeightPresent false
// and a zero SourceEquipment, while the same member after a SAV reload carries
// the resolved weight and definition for the identical Code, Kind, Effects and
// Price. Measured on the legacy corpus: every one of the differing slots had an
// identical code and identical effects and differed in exactly those three
// fields, so the conversion ADDS a resolved description rather than losing an
// item. Comparing them made a gain read as a loss on 6 members.
//
// What remains compared is what the player owns. A code that changes, an
// enchantment that disappears, a price that moves or an item that vanishes
// from the pack all still fail.
func roundTripPack(p mapload.PartyMember) []sim.ItemInstance {
	items := mapload.MemberCarriedItems(p, nil)
	if len(items) == 0 {
		return nil
	}
	out := make([]sim.ItemInstance, len(items))
	for i, item := range items {
		out[i] = item.Clone()
		out[i].ObjectID = 0
		out[i].Weight, out[i].WeightPresent = 0, false
		out[i].SourceEquipment = sim.SourceEquipment{}
	}
	return out
}
