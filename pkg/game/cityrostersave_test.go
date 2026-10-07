package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func cityRosterFixture(t *testing.T, total int) (*FrontEnd, func() *FrontEnd) {
	t.Helper()
	return cityRosterHiredFixture(t, []int{3}, []int{total - 2})
}

func cityRosterHiredFixture(t *testing.T, types, counts []int) (*FrontEnd, func() *FrontEnd) {
	t.Helper()
	return cityRosterFixtureWith(t, types, counts, true)
}

// cityRosterFixtureWith builds the hired-roster town. Without matching
// definitions no Humans row matches the source companion's class, which
// leaves the loaded town's original provenance unavailable.
func cityRosterFixtureWith(t *testing.T, types, counts []int, matching bool) (*FrontEnd, func() *FrontEnd) {
	t.Helper()
	raw, base := cityProjectionSource(t)
	var nodes []synth.RegNode
	for _, typ := range types {
		nodes = append(nodes, synth.RegNode{Name: fmt.Sprintf("npc%d", typ), Kind: kindDir, Children: []synth.RegNode{
			{Name: "DataBinID", Kind: kindInt, Int: int32(typ)},
			{Name: "PriceA", Kind: kindInt, Int: 1},
			{Name: "PriceB", Kind: kindInt, Int: 2},
		}})
	}
	registry, err := reg.Parse(synth.Reg(kindRoot, nodes))
	if err != nil {
		t.Fatal(err)
	}
	campaign := shellCampaign()
	before, current := campaign.Chapters[20], campaign.Chapters[30]
	before.EnableMercenary, current.Mercenaries = types, types
	campaign.Chapters[20], campaign.Chapters[30] = before, current
	newFront := func() *FrontEnd {
		f := base()
		rows := append(dbCollection(nil), f.Table.Humans.(dbCollection)...)
		if matching {
			copy(rows[1:5], fourBaseHumans()[1:])
			profile := humansParams(100, 0, 1)
			profile[16] = data.HeroUnmatchedClass
			rows[5] = dbEntry{name: "PC_roster", params: profile}
		}
		for _, typ := range types {
			params := make([]int32, data.MinHumanRow)
			params[0], params[1], params[2], params[3] = 25, 20, 15, 10
			params[4], params[16], params[17], params[18] = 100, 0x17, 1, 1
			params[21], params[22] = 1, 1
			rows = append(rows, dbEntry{name: fmt.Sprintf("NPC%02d_1", typ), params: params})
		}
		f.Table.Humans, f.Table.NPC, f.Words = rows, data.LoadNPCDefs(registry), ui.AuthoredWords()
		f.Campaign = resolved(campaign, nil)
		return f
	}
	f := cityProjectionLoad(t, raw, base)
	f.InstallResources = newFront().InstallResources
	graph, groups := f.Town.cityObjects, f.Town.cityGroups
	f.Town = NewTown(campaign)
	f.Town.cityGroups = groups
	f.Town.Won(20)
	f.Town.Arrive()
	f.Town.cityObjects = graph
	f.Town.gold = 1_000_000
	s := f.TownScreen().(*townScreen)
	total := len(f.Carried)
	for i, typ := range types {
		count := counts[i]
		f.Town.mercPool[typ], f.Town.mercCapacity[typ], f.Town.mercEnabled[typ] = count, count, true
		total += count
		if msg, ok := s.toggleMercenary(typ); !ok || len(f.Carried) != total {
			t.Fatal("production whole-squad hire", typ, msg, len(f.Carried), total)
		}
	}
	return f, newFront
}

func cityRosterSame(t *testing.T, before, after *FrontEnd) {
	t.Helper()
	if len(before.Carried) != len(after.Carried) || before.Town.Gold() != after.Town.Gold() ||
		before.Town.mercHired != after.Town.mercHired || before.Town.mercPool != after.Town.mercPool {
		t.Fatal("city roster, purse or hired pools changed", len(before.Carried), len(after.Carried), before.Town.Gold(), after.Town.Gold())
	}
	for i, want := range before.Carried {
		got := after.Carried[i]
		if want.ID != got.ID || want.Name != got.Name || want.Hero != got.Hero || want.MercenaryType != got.MercenaryType ||
			want.StartingHero != got.StartingHero || want.Hired() != got.Hired() || want.KnownSpells != got.KnownSpells {
			t.Fatalf("member %d changed identity/order/current attributes: %s/%s", i, want.ID, got.ID)
		}
		wantWorn, wantPack := memberItemCodes(want)
		gotWorn, gotPack := memberItemCodes(got)
		if wantWorn != gotWorn || !reflect.DeepEqual(wantPack, gotPack) {
			t.Fatalf("member %s lost worn/pack holdings", want.ID)
		}
	}
}

func cityRosterOrdinary(t *testing.T, raw []byte, count int) [][]string {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil || len(city.Roster()) != count {
		t.Fatal("ordinary city actor population", count, err)
	}
	seen := map[uint32]bool{}
	for _, member := range city.Roster() {
		if member.Identity == 0 || seen[member.Identity] {
			t.Fatal("ordinary city repeats or omits actor identity", member.Identity)
		}
		seen[member.Identity] = true
	}
	_, _, groups := cityGroupsByName(t, city.Data())
	return groups
}

func TestCityRosterSaveTwelveAndThirteenTotalMembers(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		types, counts []int
	}{
		{"12", []int{3}, []int{10}},
		{"13", []int{3}, []int{11}},
		{"thirteen hired actors", []int{3}, []int{13}},
		{"ascending squads", []int{3, 4}, []int{7, 6}},
		{"descending squads", []int{4, 3}, []int{6, 7}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f, newFront := cityRosterHiredFixture(t, scenario.types, scenario.counts)
			total := len(f.Carried)
			live := f
			var groups [][]string
			for cycle := 0; cycle < 2; cycle++ {
				raw := currentTownSave(t, f)
				gotGroups := cityRosterOrdinary(t, raw, total)
				if cycle == 0 {
					groups = gotGroups
				} else if !reflect.DeepEqual(groups, gotGroups) {
					t.Fatal("second city SAVE moved actors between groups", groups, gotGroups)
				}
				cold := cityProjectionLoad(t, raw, newFront)
				cityRosterSame(t, live, cold)
				if !reflect.DeepEqual(live.Town.cityObjects.ItemRecords, cold.Town.cityObjects.ItemRecords) ||
					!reflect.DeepEqual(live.Town.cityObjects.EffectRecords, cold.Town.cityObjects.EffectRecords) ||
					!reflect.DeepEqual(live.Town.cityObjects.SpellRecords, cold.Town.cityObjects.SpellRecords) {
					t.Fatal("expanded roster changed retained item, effect or spell records")
				}
				f = cold
			}
			for _, target := range []*FrontEnd{live, f} {
				s := target.TownScreen().(*townScreen)
				for _, typ := range scenario.types {
					for _, hired := range []bool{false, true} {
						if msg, ok := s.toggleMercenary(typ); !ok || target.Town.MercenaryHired(typ) != hired {
							t.Fatal("next return/hire action", typ, hired, msg)
						}
					}
				}
			}
			cityRosterSame(t, live, f)
		})
	}
}

func TestCityRosterSaveRejectsFabricatedCurrentPopulationAtomically(t *testing.T) {
	f, newFront := cityRosterFixture(t, 12)
	raw := currentTownSave(t, f)
	for _, defect := range []string{"repeated entity", "unbound member", "repeated identity"} {
		t.Run(defect, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal(err)
			}
			switch defect {
			case "repeated entity":
				a.Party = append(a.Party, a.Party[len(a.Party)-1])
			case "unbound member":
				a.Party[len(a.Party)-1].Entity = 65535
			case "repeated identity":
				a.Party[len(a.Party)-1].ID = a.Party[0].ID
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := cityProjectionLoad(t, raw, newFront)
			party, town := mapload.CloneParty(cold.Carried), cold.Town
			if _, _, err := cold.RestoreOriginal(bad); err == nil || cold.Town != town || !reflect.DeepEqual(party, cold.Carried) {
				t.Fatal("invalid city population replaced live state", err)
			}
		})
	}
}
