package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
)

// partyNames lists the names of the party members satisfying keep, in party order.
func partyNames(party []mapload.PartyMember, keep func(mapload.PartyMember) bool) []string {
	var out []string
	for _, m := range party {
		if keep(m) {
			out = append(out, m.Name)
		}
	}
	return out
}

// Each member who joins the party between missions is written in a group of
// its own after the groups the town already holds (PARTY-039).
func TestReleaseCitySAVGivesEachJoiningCompanionItsOwnGroup(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	reachabilityWalkArrive(t, f, "Joining Hero")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.settleCityGroups(f.Carried)
	before := len(f.Carried)
	_, _, firstNames := cityGroupsByName(t, writtenTownData(t, f))
	if len(firstNames) != 1 || len(firstNames[0]) != before {
		t.Fatalf("fixture town writes groups %v, want one group of %d", firstNames, before)
	}
	// Two members join the way a script's owner change adds them to the party
	// that returns from a mission: appended to the party, then settled.
	in := f.townInstall()
	chapter := f.Town.Chapter()
	for npc := 1; npc < 60 && len(f.Carried) < before+2; npc++ {
		if member, ok := townCompanion(in, chapter, npc, f.Carried); ok && !slices.ContainsFunc(f.Carried, func(p mapload.PartyMember) bool { return p.Name == member.Name }) {
			member.CompanionNPC = npc
			f.Carried = mapload.OwnParty(append(f.Carried, member))
		}
	}
	f.Town.settleCityGroups(f.Carried)
	joined := len(f.Carried) - before
	if joined < 2 {
		t.Fatalf("fixture assumption broke: %d members joined, want at least 2", joined)
	}
	_, _, names := cityGroupsByName(t, writtenTownData(t, f))
	if len(names) != 1+joined {
		t.Fatalf("written groups %v, want the first group and %d groups of one joiner", names, joined)
	}
	if !slices.Equal(names[0], firstNames[0]) {
		t.Fatalf("first group %v, want %v", names[0], firstNames[0])
	}
	for g := 1; g < len(names); g++ {
		if len(names[g]) != 1 || names[g][0] != f.Carried[before+g-1].Name {
			t.Fatalf("group %d is %v, want the joiner %q alone", g, names[g], f.Carried[before+g-1].Name)
		}
	}
}

// Two hired types in one inn visit write one group per type, and a siege hire
// follows the same path (PARTY-040).
func TestReleaseCitySAVWritesOneGroupPerHiredTypeSiegeIncluded(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := hireSiegeSquads(t, "Siege Groups")
	hireInTown(t, f, 14)
	_, groups, names := cityGroupsByName(t, writtenTownData(t, f))
	if len(groups) < 4 {
		t.Fatalf("written groups %v, want the party groups and one group each for types 1, 2 and 14", names)
	}
	tail := groups[len(groups)-3:]
	for k, typ := range []uint8{1, 2, 14} {
		held := 0
		for _, m := range f.Carried {
			if m.MercenaryType == typ {
				held++
			}
		}
		if held == 0 || len(tail[k].Actors) != held {
			t.Fatalf("group for type %d holds %d actors, want the %d members of the type; groups %v", typ, len(tail[k].Actors), held, names)
		}
	}
}

// A hire group whose members all left stays in the live list, empty, and the
// town SAV does not write it: the town's Player has participant word zero
// (SAV-1251). A refunded return removes only its own group.
func TestReleaseCitySAVDropsAnEmptiedHireGroup(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, screen := hireForOrderTest(t, "Emptied Hero")
	hireInTown(t, f, 14, 6)
	_, _, hired := cityGroupsByName(t, writtenTownData(t, f))
	if len(hired) < 3 || len(hired[len(hired)-1]) == 0 {
		t.Fatalf("fixture hire groups %v", hired)
	}
	base := len(hired) - 2
	kept := f.Carried[:0:0]
	for _, m := range f.Carried {
		if m.MercenaryType != 6 {
			kept = append(kept, m)
		}
	}
	f.Carried = kept
	f.Town.settleCityGroups(f.Carried)
	_, _, lost := cityGroupsByName(t, writtenTownData(t, f))
	if len(lost) != base+1 || len(lost[base]) == 0 || emptyGroupCount(liveGroupIDs(f.Town.cityGroups)) != 1 {
		t.Fatalf("after the type-6 squad was lost the SAV groups are %v and the live ones %v, want the type-14 group written and one empty live group", lost, liveGroupIDs(f.Town.cityGroups))
	}
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("return of the type-14 squad refused")
	}
	_, _, returned := cityGroupsByName(t, writtenTownData(t, f))
	if len(returned) != base || emptyGroupCount(liveGroupIDs(f.Town.cityGroups)) != 1 {
		t.Fatalf("after the refunded return the SAV groups are %v and the live ones %v, want no squad group written and the loss-created empty group live", returned, liveGroupIDs(f.Town.cityGroups))
	}
}

// A town whose party had no join, hire or loss keeps the groups it was given:
// the same members in the same groups and order.
func TestReconcileCityGroupsLeavesAnUnchangedPartyAlone(t *testing.T) {
	live := []cityLiveGroup{
		{Payload: cityConstructedGroup(), Members: []string{"b", "a"}},
		{Payload: cityHireGroup(), Members: []string{"h"}},
		{Payload: cityConstructedGroup(), Members: []string{"c"}},
	}
	live[2].Payload.Raw80[3] = 9
	party := []mapload.PartyMember{{ID: "a"}, {ID: "b"}, {ID: "h", MercenaryType: 4}, {ID: "c"}}
	got, order := reconcileCityGroups(live, party)
	wantGroupIDs(t, got, []string{"b", "a"}, []string{"h"}, []string{"c"})
	if !slices.Equal(slices.Concat(order...), []int{1, 0, 2, 3}) || got[2].Payload.Raw80[3] != 9 {
		t.Fatalf("order %v, payload %v", order, got[2].Payload.Raw80)
	}
}

// The mission-start walk removes the leading groups that hold no member and
// stops at the first group that holds one; a group behind it is kept, and the
// last listed group is never removed (AI-479).
func TestJoinCityGroupsRemovesOnlyTheLeadingEmptyGroups(t *testing.T) {
	var town Town
	town.cityGroups = []cityLiveGroup{{Members: []string{"x"}}, {Members: []string{"y"}}, {Members: []string{"z"}}, {Members: []string{"a"}}, {Members: []string{"w"}}}
	party := []mapload.PartyMember{{ID: "a"}}
	town.joinCityGroups(party)
	wantGroupIDs(t, town.cityGroups, []string{"a"}, []string{"w"})
	town.cityGroups = []cityLiveGroup{{Members: []string{"x"}}, {Members: []string{"y"}}}
	town.joinCityGroups(nil)
	wantGroupIDs(t, town.cityGroups, []string{"y"})
	town.cityGroups = []cityLiveGroup{{Members: []string{"a"}}, {Members: []string{"x"}}}
	town.joinCityGroups(party)
	wantGroupIDs(t, town.cityGroups, []string{"a"}, []string{"x"})
	var none Town
	none.joinCityGroups(party)
	if none.cityGroups != nil {
		t.Fatal("the walk made groups for a town that held none")
	}
}
