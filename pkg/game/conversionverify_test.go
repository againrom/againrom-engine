package game

import (
	"fmt"
	"reflect"
	"slices"

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

type roundTripRosterDifference struct {
	field, message string
}

// Mission Groups retain their current order; the entry roster can have another
// order. Compare each persistent member to itself and account for order apart.
func roundTripRosterDiff(before, after Snapshot) []roundTripRosterDifference {
	var out []roundTripRosterDifference
	add := func(field, message string) {
		out = append(out, roundTripRosterDifference{field, message})
	}
	if len(before.Party) != len(after.Party) {
		add("roster size", fmt.Sprintf("roster size changed: %d -> %d", len(before.Party), len(after.Party)))
		return out
	}
	indices := make([]int, len(before.Party))
	for i := range indices {
		indices[i] = i
	}
	if before.Mission != 0 && after.Mission != 0 {
		beforeIDs, afterIDs := make([]string, len(indices)), make([]string, len(indices))
		positions, seen := make(map[string]int, len(indices)), make(map[string]bool, len(indices))
		for i, member := range after.Party {
			if _, exists := positions[member.ID]; member.ID == "" || exists {
				add("roster identity", fmt.Sprintf("reloaded roster has absent or repeated identity %q", member.ID))
				return out
			}
			positions[member.ID], afterIDs[i] = i, member.ID
		}
		for i, member := range before.Party {
			j, exists := positions[member.ID]
			if member.ID == "" || seen[member.ID] || !exists {
				add("roster identity", fmt.Sprintf("original roster identity %q is absent, repeated or missing after reload", member.ID))
				return out
			}
			seen[member.ID], beforeIDs[i], indices[i] = true, member.ID, j
		}
		if !slices.Equal(beforeIDs, afterIDs) {
			add("roster order", fmt.Sprintf("roster order changed: %v -> %v", beforeIDs, afterIDs))
		}
	}
	for i, j := range indices {
		b, a := before.Party[i], after.Party[j]
		for _, field := range roundTripMemberDiff(b, a) {
			add(field, fmt.Sprintf("roster member %d %q: %s changed", i, b.Name, field))
		}
	}
	return out
}

// A legacy mission snapshot keeps hired Town stock at zero while the squad is out. The
// original campaign record instead carries the live hired count in its working
// cell, and RestoreOriginal exposes that cell as Town stock until mission end.
// This is one conversion representation, not a stock gain: the same party must
// contain the squad on both sides and the emitted campaign cell must agree.
func isHiredPoolMigration(before, after Snapshot, typ int) bool {
	if typ <= 0 || typ >= len(before.MercenaryPool) || before.Mission == 0 ||
		before.Mission != after.Mission || before.CampaignState || !after.CampaignState ||
		len(after.Campaign.MercenaryWorking) != 15 || !before.MercenaryHired[typ] ||
		!after.MercenaryHired[typ] || before.MercenaryPool[typ] != 0 {
		return false
	}
	count := nativeMercenaryPartyCount(before.Party, typ)
	return count == nativeMercenaryPartyCount(after.Party, typ) &&
		after.MercenaryPool[typ] == count &&
		int(after.Campaign.MercenaryWorking[typ-1]) == count
}

// ConversionDifference is one field a conversion changed. Row names the ledger
// row or model note that discloses the field; an empty Row is an undisclosed
// difference.
type ConversionDifference struct {
	Field, Message, Row string
}

// conversionDisclosedRows names the fields a legacy-to-SAV conversion is
// permitted to change, each against the row that discloses it. Valuable
// Documents appear in no row: under the owner's ruling that is the state which
// must survive.
var conversionDisclosedRows = map[string]string{
	"offered mission": "DIV-1322",
	"fame":            "DIV-1319",
	"hero record":     "DIV-1321",
	"roster order":    "DIV-1368",
	// sav.CampaignProjection models no completed-mission set on either side,
	// so the loss is an absence in this project's model, not a writer dropping
	// a field it was holding. Whether the original's format carries such a field
	// is a ROM1 question this codebase cannot answer.
	"completed missions": "not modeled in sav.CampaignProjection; open",
}

// CompareConvertedStates reports every semantic difference between the state
// loaded from the source and the state a cold LOAD of the written SAV installed,
// each against its disclosing row. migrated lists the mercenary types whose
// hired-pool representation changed without loss.
func CompareConvertedStates(before, after Snapshot) (diffs []ConversionDifference, migrated []int) {
	report := func(field, format string, args ...any) {
		diffs = append(diffs, ConversionDifference{Field: field, Message: fmt.Sprintf(format, args...), Row: conversionDisclosedRows[field]})
	}
	if before.Mission != after.Mission {
		report("mission identity", "mission/city identity changed: %d -> %d", before.Mission, after.Mission)
	}
	if before.Gold != after.Gold {
		report("gold", "gold changed: %d -> %d", before.Gold, after.Gold)
	}
	for _, difference := range roundTripRosterDiff(before, after) {
		report(difference.field, "%s", difference.message)
	}
	if !reflect.DeepEqual(before.Won, after.Won) {
		report("completed missions", "completed missions changed: %v -> %v", before.Won, after.Won)
	}
	for typ := 1; typ < len(before.MercenaryPool); typ++ {
		if before.MercenaryPool[typ] == after.MercenaryPool[typ] {
			continue
		}
		if isHiredPoolMigration(before, after, typ) {
			migrated = append(migrated, typ)
			continue
		}
		report("mercenary pool", "type %d stock changed: %d -> %d (hired %t -> %t; campaign working %v -> %v)",
			typ, before.MercenaryPool[typ], after.MercenaryPool[typ], before.MercenaryHired[typ], after.MercenaryHired[typ],
			before.Campaign.MercenaryWorking, after.Campaign.MercenaryWorking)
	}
	if before.MercenaryEnabled != after.MercenaryEnabled {
		report("mercenary availability", "mercenary pool availability changed")
	}
	if before.MercenaryHired != after.MercenaryHired {
		report("hired mercenaries", "hired mercenary set changed")
	}
	if before.Offered != after.Offered {
		report("offered mission", "offered mission changed: %d -> %d", before.Offered, after.Offered)
	}
	if !reflect.DeepEqual(before.Fame, after.Fame) {
		report("fame", "fame changed: %+v -> %+v", before.Fame, after.Fame)
	}
	if !reflect.DeepEqual(before.Documents, after.Documents) {
		report("valuable documents", "valuable documents changed: %d -> %d records, %+v -> %+v",
			len(before.Documents), len(after.Documents), before.Documents, after.Documents)
	}
	return diffs, migrated
}

// nativeMercenaryPartyCount is the tavern's party count over a plain party
// slice. The legacy AGS-to-SAV migration comparator uses it to validate a
// hired roster; product save projection reads the campaign working pool.
func nativeMercenaryPartyCount(party []mapload.PartyMember, typ int) int {
	n := 0
	for _, member := range party {
		if int(member.MercenaryType) == typ {
			n++
		}
	}
	return n
}
