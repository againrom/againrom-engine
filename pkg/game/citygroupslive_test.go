package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func liveGroupIDs(groups []cityLiveGroup) [][]string {
	var out [][]string
	for _, g := range groups {
		out = append(out, g.Members)
	}
	return out
}

func wantGroupIDs(t *testing.T, got []cityLiveGroup, want ...[]string) {
	t.Helper()
	have := liveGroupIDs(got)
	if len(have) != len(want) {
		t.Fatalf("groups %v, want %v", have, want)
	}
	for g := range want {
		if !slices.Equal(have[g], want[g]) {
			t.Fatalf("groups %v, want %v", have, want)
		}
	}
}

// A live group keeps its members and its order while the party changes around
// it: a member who left is dropped, an emptied group is not kept, a joiner
// enters the first group and a hire enters a group of its type.
func TestReconcileCityGroupsFollowsPartyChanges(t *testing.T) {
	marked := func(b byte) cityLiveGroup {
		g := cityLiveGroup{Payload: cityConstructedGroup()}
		g.Payload.Raw80[0] = b
		return g
	}
	first, second, third := marked(1), marked(2), marked(3)
	first.Members, second.Members, third.Members = []string{"b", "a"}, []string{"gone", "h4"}, []string{"c"}
	live := []cityLiveGroup{first, second, third}
	party := []mapload.PartyMember{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "h4", MercenaryType: 4}, {ID: "join"},
		{ID: "n5", MercenaryType: 5}, {ID: "n3a", MercenaryType: 3}, {ID: "n3b", MercenaryType: 3}}
	groups, order := reconcileCityGroups(live, party)
	wantGroupIDs(t, groups, []string{"b", "a", "join"}, []string{"h4"}, []string{"c"}, []string{"n5"}, []string{"n3a", "n3b"})
	if got := slices.Concat(order...); !slices.Equal(got, []int{1, 0, 4, 3, 2, 5, 6, 7}) {
		t.Fatalf("party order %v", got)
	}
	if groups[0].Payload.Raw80[0] != 1 || groups[1].Payload.Raw80[0] != 2 || groups[2].Payload.Raw80[0] != 3 {
		t.Fatal("a kept group lost its own field values")
	}
	if !slices.Equal(groups[3].Payload.Raw80, cityHireGroup().Raw80) {
		t.Fatal("a hire group does not carry the hire fields")
	}
	// The hired groups leave with their members, and a second reconcile of the
	// result changes nothing.
	again, _ := reconcileCityGroups(groups, party[:5])
	wantGroupIDs(t, again, []string{"b", "a", "join"}, []string{"h4"}, []string{"c"})
	same, _ := reconcileCityGroups(again, party[:5])
	wantGroupIDs(t, same, []string{"b", "a", "join"}, []string{"h4"}, []string{"c"})
}

// A group whose only members left is gone, so a later hire of the same type
// opens a new group at the end rather than reviving it.
func TestReconcileCityGroupsDropsAnEmptiedGroup(t *testing.T) {
	var town Town
	party := []mapload.PartyMember{{ID: "hero"}, {ID: "m6a", MercenaryType: 6}}
	town.settleCityGroups(party)
	wantGroupIDs(t, town.cityGroups, []string{"hero"}, []string{"m6a"})
	party = party[:1]
	town.settleCityGroups(party)
	wantGroupIDs(t, town.cityGroups, []string{"hero"})
	party = append(party, mapload.PartyMember{ID: "m14", MercenaryType: 14}, mapload.PartyMember{ID: "m6a", MercenaryType: 6})
	town.settleCityGroups(party)
	wantGroupIDs(t, town.cityGroups, []string{"hero"}, []string{"m14"}, []string{"m6a"})
}

// The saved document names each group's actors by object index; the current
// supplement binds those objects to party members, so the groups are rebuilt
// without any original provenance.
func TestCityGroupsFromCurrentBindActorsThroughTheDocument(t *testing.T) {
	fixed := make([]byte, 51)
	fixed[47] = 0x70
	raw := func(b byte) []byte { r := make([]byte, 80); r[0] = b; return r }
	city := sav.CityData{Players: []uint16{1}, Objects: []sav.CityObjectData{
		{Class: "Player", Player: &sav.CityPlayerData{Fixed: fixed, Groups: []sav.CityGroupData{
			{Raw80: raw(1), F1c: 3, F44: 0x70, Actors: []uint16{3, 2}},
			{Raw80: raw(2), F44: 0x70, Actors: []uint16{4, 5}},
			{Raw80: raw(9), F44: 0x70, Actors: []uint16{6}},
		}}},
	}}
	a := &currentActionData{
		Party: []currentPartyMember{{Entity: 10, ID: []byte("hero")}, {Entity: 11, ID: []byte("fergard")}, {Entity: 12, ID: []byte("h1")}},
		Bindings: []currentActionBinding{{ID: 10, Object: 2}, {ID: 11, Object: 3}, {ID: 12, Object: 4},
			{ID: 12, Object: 5, Structure: true}, {ID: 99, Object: 6}},
	}
	groups := cityGroupsFromCurrent(city, a)
	wantGroupIDs(t, groups, []string{"fergard", "hero"}, []string{"h1"})
	if groups[0].Payload.F1c != 3 || groups[0].Payload.Raw80[0] != 1 || groups[0].Payload.F40 != 0 || groups[0].Payload.F44 != nativeCityPlayerIdentity {
		t.Fatalf("first group fields %+v", groups[0].Payload)
	}
	if cityGroupsFromCurrent(city, nil) != nil || cityGroupsFromCurrent(sav.CityData{}, a) != nil {
		t.Fatal("groups from an absent supplement or Player")
	}
}

// The retained counterexample: a source companion that matches no installed
// definition leaves the loaded town's original provenance unavailable, and the
// loaded group order [companion, leader] used to fall back to the constructed
// order. The groups now come from live membership, so the second SAVE keeps
// the first SAVE's groups. The control drops the live membership and shows
// the fallback order the witness exists to exclude.
func TestCityGroupsSurviveUnavailableProvenance(t *testing.T) {
	f, newFront := cityRosterFixtureWith(t, []int{3}, []int{10}, false)
	total := len(f.Carried)
	var groups [][]string
	var last *FrontEnd
	for cycle := 0; cycle < 3; cycle++ {
		raw := currentTownSave(t, f)
		got := cityRosterOrdinary(t, raw, total)
		if cycle == 0 {
			groups = got
		} else if !reflect.DeepEqual(groups, got) {
			t.Fatalf("SAVE %d moved actors between groups: %v, then %v", cycle+1, groups, got)
		}
		cold := cityProjectionLoad(t, raw, newFront)
		if cold.originalCity == nil || cold.originalCity.unavailable == nil {
			t.Fatal("the input no longer leaves the loaded provenance unavailable")
		}
		cityRosterSame(t, f, cold)
		f, last = cold, cold
	}
	if len(groups) != 2 || len(groups[0]) != 2 || len(groups[1]) != 10 {
		t.Fatalf("groups %v, want the two unhired members and the ten hires", groups)
	}
	last.Town.cityGroups = nil
	if broken := cityRosterOrdinary(t, currentTownSave(t, last), total); reflect.DeepEqual(broken, groups) {
		t.Fatal("dropping live membership leaves the groups unchanged, so the input does not exercise it")
	}
}
