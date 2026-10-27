package game

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/town"
)

// secondTownDescription is the second game's description of town id, read
// through its catalog.
func secondTownDescription(id int) *town.Description {
	return townBooks["rom2"].description(id, townHooksDestinations)
}

func findProfile(t *testing.T, id string) base.Profile {
	t.Helper()
	p, ok := base.Find(id)
	if !ok {
		t.Fatalf("no profile %s", id)
	}
	return p
}

// Each first-game profile's catalog lists exactly the one town: town 1,
// composed from the rom1 description under the chapters hook set, its offers
// from the chapter keys, standing at the world map's first registry object.
// The town screen's description is that entry's.
func TestFirstGameCatalogListsItsOneTown(t *testing.T) {
	want := []town.Entry{{ID: 1, Description: "rom1", Hooks: townHooksChapters, Offers: townOffersChapterKeys,
		MapPoint: town.MapPoint{Object: 1}}}
	for _, id := range []string{base.ROM1EN, base.ROM1RU} {
		p := findProfile(t, id)
		c := TownCatalog(p)
		if c == nil {
			t.Fatalf("%s: no town catalog", id)
		}
		got := slices.Clone(c.Towns)
		for i := range got {
			got[i].Cite = nil
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: towns %+v, want %+v", id, got, want)
		}
		if d := TownDescription(p); d == nil || d != townDescriptions["rom1"] {
			t.Errorf("%s: town screen description is not the catalog's town 1", id)
		}
		e, _ := c.Town(chapterTown)
		if e.ID != chapterTownEntry.ID || e.Offers != chapterTownEntry.Offers {
			t.Errorf("%s: catalog town %+v disagrees with the chapter town entry NewTown builds %+v", id, e, chapterTownEntry)
		}
	}
	if TownDescription(base.Profile{}) != townDescriptions["rom1"] {
		t.Error("the zero profile does not compose the first game's town")
	}
}

// Each second-game profile's catalog lists exactly its three towns, each
// composed from its own description under the destinations hook set, its
// offers from the scenario bank and its map point the world-map
// description's record of that town. Each town ID addresses its own shop
// assortment record, and its state holds its ID and no tavern pool. The
// first game's town screen composes none of them.
func TestSecondGameCatalogListsItsThreeTowns(t *testing.T) {
	want := []town.Entry{
		{ID: 1, Description: "rom2-town1", Hooks: townHooksDestinations, Offers: townOffersScenarioBank, MapPoint: town.MapPoint{Record: true}},
		{ID: 2, Description: "rom2-town2", Hooks: townHooksDestinations, Offers: townOffersScenarioBank, MapPoint: town.MapPoint{Record: true}},
		{ID: 3, Description: "rom2-town3", Hooks: townHooksDestinations, Offers: townOffersScenarioBank, MapPoint: town.MapPoint{Record: true}},
	}
	for _, id := range []string{base.ROM2EN, base.ROM2RU} {
		p := findProfile(t, id)
		c := TownCatalog(p)
		if c == nil {
			t.Fatalf("%s: no town catalog", id)
		}
		got := slices.Clone(c.Towns)
		for i := range got {
			got[i].Cite = nil
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: towns %+v, want %+v", id, got, want)
		}
		if TownDescription(p) != nil {
			t.Errorf("%s: the first game's town screen composes a second-game town", id)
		}
		wm := WorldMapDescriptionFor(p)
		shop := newSecondShop()
		for _, e := range c.Towns {
			if d := townDescriptionOf(c, e.ID, townHooksDestinations); d == nil || d != townDescriptions[e.Description] {
				t.Errorf("%s: town %d does not compose %s", id, e.ID, e.Description)
			}
			if townDescriptionOf(c, e.ID, townHooksChapters) != nil {
				t.Errorf("%s: town %d answers the chapters hook set", id, e.ID)
			}
			if _, ok := wm.point(placeTown, e.ID); !ok {
				t.Errorf("%s: town %d has no world-map record", id, e.ID)
			}
			if e.ID >= len(shop.records) {
				t.Errorf("%s: town %d addresses no shop assortment record", id, e.ID)
			}
			if st := newTownState(e, Campaign{MercenaryCount: []int{4}}); st.id != e.ID || st.mercPool != ([16]int{}) {
				t.Errorf("%s: town %d state %+v; want its ID and no pool", id, e.ID, st)
			}
		}
	}
}

// Every shipped catalog decodes, and every description it names is the one
// the composer registry holds.
func TestEveryEditionNamesAShippedCatalog(t *testing.T) {
	for _, p := range base.Profiles {
		c := TownCatalog(p)
		if c == nil || c.Catalog != p.Edition().Towns {
			t.Errorf("%s: edition names catalog %q, found %v", p.ID, p.Edition().Towns, c)
			continue
		}
		for _, e := range c.Towns {
			if townDescriptions[e.Description] == nil {
				t.Errorf("%s: town %d names description %s the registry does not hold", p.ID, e.ID, e.Description)
			}
		}
	}
}

func TestTownCatalogUsesOneVocabulary(t *testing.T) {
	v := townVocabulary()
	for _, name := range []string{townHooksChapters, townHooksDestinations} {
		got, ok := townHookSetVocabulary(name)
		if !ok || !reflect.DeepEqual(got, v) {
			t.Fatalf("hook set %q uses another vocabulary: %+v", name, got)
		}
	}
	for name, raw := range townDescriptionData {
		if _, err := town.Decode(raw, v); err != nil {
			t.Errorf("description %s: %v", name, err)
		}
	}
	if _, ok := townHookSetVocabulary("unknown"); ok {
		t.Fatal("unknown hook set accepted")
	}
}

func TestTownHostRefusesMissingCampaignState(t *testing.T) {
	for _, change := range []struct {
		name, want string
		apply      func(map[string]any)
	}{
		{"shop binding", "host state binding", func(d map[string]any) { d["host"].(map[string]any)["shop"] = "bank" }},
		{"inn page", "requires offer source scenario-bank", func(d map[string]any) { d["rooms"].([]any)[0].(map[string]any)["page"] = "inn" }},
		{"shop hook", "requires offer source scenario-bank", func(d map[string]any) {
			q := d["square"].(map[string]any)
			q["enter"] = append(q["enter"].([]any), map[string]any{"hook": "shop-open", "cite": []any{"owner"}})
		}},
	} {
		t.Run(change.name, func(t *testing.T) {
			var d map[string]any
			if err := json.Unmarshal(rom1TownJSON, &d); err != nil {
				t.Fatal(err)
			}
			change.apply(d)
			raw, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			data := maps.Clone(townDescriptionData)
			data["rom1"] = raw
			if _, err := decodeTownBook(rom1TownCatalogJSON, data); err == nil || !strings.Contains(err.Error(), change.want) {
				t.Fatalf("bad state binding: %v, want %q", err, change.want)
			}
		})
	}
}

// A description named under two hook sets is rejected: one decoded
// description answers one campaign.
func TestCatalogRejectsADescriptionUnderTwoHookSets(t *testing.T) {
	raw := []byte(`{"catalog": "fixture", "cite": [], "towns": [
  {"id": 1, "description": "rom1", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 1}, "cite": []},
  {"id": 2, "description": "rom1", "hooks": "destinations", "offers": "chapter-keys", "map-point": {"object": 2}, "cite": []}]}`)
	if _, err := decodeTownCatalog(raw, townDescriptionData, map[string]*town.Description{}, map[string]string{}); err == nil {
		t.Fatal("a description under two hook sets decoded")
	}
	missing := []byte(`{"catalog": "fixture", "cite": [], "towns": [
  {"id": 1, "description": "nowhere", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 1}, "cite": []}]}`)
	if _, err := decodeTownCatalog(missing, townDescriptionData, map[string]*town.Description{}, map[string]string{}); err == nil {
		t.Fatal("a catalog naming an unshipped description decoded")
	}
}

// buildTownStates builds the state of every town of catalog cat over
// campaign c through the one builder, keyed by town ID.
func buildTownStates(cat *town.Catalog, c Campaign) map[int]townState {
	states := make(map[int]townState, len(cat.IDs()))
	for _, id := range cat.IDs() {
		e, _ := cat.Town(id)
		states[id] = newTownState(e, c)
	}
	return states
}

// twoChapterTowns is a fixture catalog of two first-game towns.
const twoChapterTowns = `{"catalog": "fixture", "cite": [], "towns": [
  {"id": 1, "description": "rom1", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 1}, "cite": []},
  {"id": 2, "description": "rom1", "hooks": "chapters", "offers": "chapter-keys", "map-point": {"object": 2}, "cite": []}]}`

// stateLeaks runs a visit to town two (it accepts the tavern's first offer,
// hires mercenary type 1 and ends a mission in which all of them fell) and
// names what of it town one sees.
func stateLeaks(one, two *Town) []string {
	offers, pool, hired := one.Offers(TownTavern), one.MercenaryPool(1), one.MercenaryHired(1)
	two.gold = 1 << 20
	two.Take(TownTavern, 0)
	two.mercenaryHire(1, 0)
	var leaks []string
	if one.MercenaryHired(1) != hired {
		leaks = append(leaks, "hire")
	}
	two.mercenaryBoundary([16]int{})
	if !reflect.DeepEqual(one.Offers(TownTavern), offers) {
		leaks = append(leaks, "offers")
	}
	if one.MercenaryPool(1) != pool {
		leaks = append(leaks, "mercenaries")
	}
	return leaks
}

// A catalog of two first-game towns builds two independent states through
// the one builder: what the second town's tavern hands out and loses does
// not reach the first town's offers or mercenaries. The loss controls are
// the shapes a keyed state must not take: one Town for both IDs, and two
// Towns sharing one state's offer record.
func TestTwoChapterTownsKeepTwoStates(t *testing.T) {
	cat, err := decodeTownCatalog([]byte(twoChapterTowns), townDescriptionData, map[string]*town.Description{}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	c := townCampaign(t)
	c.MercenaryCount = []int{4}
	states := buildTownStates(cat, c)
	if got := len(states); got != 2 {
		t.Fatalf("built %d states, want 2", got)
	}
	at := func(s townState) *Town {
		tw := NewTown(c)
		tw.townState = s
		tw.Arrive()
		return tw
	}
	one, two := at(states[1]), at(states[2])
	if one.townID() != 1 || two.townID() != 2 {
		t.Fatalf("town IDs %d and %d, want 1 and 2", one.townID(), two.townID())
	}
	if len(one.Offers(TownTavern)) == 0 || one.MercenaryPool(1) != 4 {
		t.Fatalf("fixture town 1 offers %v pool %d; want offers and a pool of 4", one.Offers(TownTavern), one.MercenaryPool(1))
	}
	if leaks := stateLeaks(one, two); len(leaks) != 0 {
		t.Errorf("town 2's visit reached town 1: %v", leaks)
	}
	if two.MercenaryPool(1) != 0 || len(two.Offers(TownTavern)) == len(one.Offers(TownTavern)) {
		t.Errorf("town 2's visit did not change town 2: pool %d offers %v", two.MercenaryPool(1), two.Offers(TownTavern))
	}

	shared := at(states[1])
	if leaks := stateLeaks(shared, shared); !slices.Equal(leaks, []string{"hire", "offers", "mercenaries"}) {
		t.Errorf("loss control: one Town for both towns leaked %v, want hire, offers and mercenaries", leaks)
	}
	fresh := buildTownStates(cat, c)
	first := at(fresh[1])
	sharing := at(fresh[2])
	sharing.taken = first.taken
	if leaks := stateLeaks(first, sharing); !slices.Contains(leaks, "offers") {
		t.Errorf("loss control: a shared offer record leaked %v, want offers", leaks)
	}
}

// The second game's places and the first game's are one model: the second
// campaign's current place is its own, and the first game stands in town 1
// once its campaign reached the town, in a mission while one is live.
func TestOnePlaceModelForBothGames(t *testing.T) {
	first, second := campaignOf(base.GameROM1), campaignOf(base.GameROM2)
	s := &CampaignSession{Town: NewTown(townCampaign(t))}
	if got := first.place(s); got != (partyPlace{}) {
		t.Errorf("first game before its town: place %+v, want none", got)
	}
	s.Town.Arrive()
	if got := first.place(s); got != (partyPlace{kind: placeTown, id: 1}) {
		t.Errorf("first game in town: place %+v, want town 1", got)
	}
	if id, ok := first.place(s).town(); !ok || id != 1 {
		t.Errorf("first game town: %d %v", id, ok)
	}
	s.live, s.liveMission = &mapWorld{}, 30
	if got := first.place(s); got != (partyPlace{kind: placeMission, id: 30}) {
		t.Errorf("first game in a mission: place %+v, want mission 30", got)
	}
	s.live, s.liveMission = nil, 0
	s.Town.second = newSecondCampaign()
	if got := second.place(s); got != (partyPlace{kind: placeTown, id: 1}) {
		t.Errorf("second game at start: place %+v, want town 1", got)
	}
	s.Town.second.current = partyPlace{kind: placeTown, id: 3}
	if got := second.place(s); got != (partyPlace{kind: placeTown, id: 3}) {
		t.Errorf("second game in town 3: place %+v", got)
	}
	var none *CampaignSession
	if first.place(none) != (partyPlace{}) || second.place(none) != (partyPlace{}) {
		t.Error("a nil session stands somewhere")
	}
	s.Town.second = nil
	if got := second.place(s); got != (partyPlace{kind: placeTown, id: 1}) {
		t.Errorf("second game without its campaign state: place %+v, want the first game's answer, town 1", got)
	}
}
