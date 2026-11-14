package game

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/herokind"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// routeChapter is a chapter section whose tavern hands out inn, one visitor
// a mission, and whose shop hands out shop.
func routeChapter(n int, inn, shop []int32) synth.RegNode {
	npcs := make([]int32, len(inn))
	for i := range inn {
		npcs[i] = int32(n + i)
	}
	ch := []synth.RegNode{
		{Name: "InnNPC", Kind: kindIntArray, Ints: npcs},
		{Name: "InnMission", Kind: kindIntArray, Ints: inn},
	}
	if len(shop) > 0 {
		ch = append(ch, synth.RegNode{Name: "ShopMission", Kind: kindIntArray, Ints: shop})
	}
	return synth.RegNode{Name: fmt.Sprintf("Mission%d", n), Kind: kindDir, Children: ch}
}

// routeSide is the section of a side mission.
func routeSide(n int) synth.RegNode {
	return synth.RegNode{Name: fmt.Sprintf("Mission%d", n), Kind: kindDir,
		Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 700}}}
}

// routeSections is a scenario of six main missions: 10 leads to 20, and the
// towns of chapters 30 to 60 each hand out their main mission; the shops of
// chapters 30 and 40 hand out side missions 31 and 41.
func routeSections() []synth.RegNode {
	return []synth.RegNode{
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: 20}}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{{Name: "Mercenaries", Kind: kindInt, Int: 1}}},
		routeChapter(30, []int32{30}, []int32{31}),
		routeSide(31),
		routeChapter(40, []int32{40}, []int32{41}),
		routeSide(41),
		routeChapter(50, []int32{50}, nil),
		routeChapter(60, []int32{60}, nil),
	}
}

func routeCampaign(t *testing.T, sections []synth.RegNode) Campaign {
	t.Helper()
	r, err := reg.Parse(synth.Reg(kindRoot, sections))
	if err != nil {
		t.Fatal(err)
	}
	return ReadCampaign(r)
}

// routeMod is campaign stock with the mod missions of src, as the game
// admits them.
func routeMod(t *testing.T, stock Campaign, src string) (Campaign, mod.MissionData) {
	t.Helper()
	if src == "" {
		return stock, mod.MissionData{}
	}
	d, err := mod.ParseMissions("route", mod.MissionsFile, []byte(src), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	c, err := stock.withModMissions(d)
	if err != nil {
		t.Fatal(err)
	}
	return c, d
}

const (
	// routeSideToMain sends side mission 31's win to main mission 50.
	routeSideToMain = "[[mission]]\nkind = \"campaign\"\nkey = \"s\"\nnumber = 31\nAutoGetMission = 50\n"
	// routeLeadOff ends mission 10's lead to mission 20.
	routeLeadOff = "[[mission]]\nkind = \"campaign\"\nkey = \"first\"\nnumber = 10\nAutoGetMission = -1\n"
	// routeLastSide sets side mission 61's win to open the town.
	routeLastSide = "[[mission]]\nkind = \"campaign\"\nkey = \"last\"\nnumber = 61\nAutoGetMission = -1\n"
	// routeTwoGuests plays guest mission 25 after mission 20 for male heroes
	// and 26 for female heroes.
	routeTwoGuests = `[[mission]]
key = "a"
number = 25
map = "mission/{lang}/map.alm"
after = 20
chapter = 30
guest = 22
sex = "male"
[[mission]]
key = "b"
number = 26
map = "mission/{lang}/map.alm"
after = 20
chapter = 30
guest = 22
sex = "female"
`
	// routeOneGuest plays guest mission 25 after mission 20 for mages only.
	routeOneGuest = `[[mission]]
key = "a"
number = 25
map = "mission/{lang}/map.alm"
after = 20
chapter = 30
guest = 22
class = "mage"
`
)

// routeArrows is the arrows out of each mission, by the mission's number:
// the heroes each holds for, by its end, "mission N" or "town N".
type routeArrows map[int]map[string]ui.HeroAudience

func (a routeArrows) add(n int, end string, heroes ui.HeroAudience) {
	if a[n] == nil {
		a[n] = map[string]ui.HeroAudience{}
	}
	a[n][end] |= heroes
}

// lines is the arrows out of each mission, sorted, each with the word of
// its heroes where the mission's arrows hold for different heroes.
func (a routeArrows) lines() map[int][]string {
	out := map[int][]string{}
	for n, ends := range a {
		var reach ui.HeroAudience
		for _, heroes := range ends {
			reach |= heroes
		}
		out[n] = []string{}
		for _, end := range slices.Sorted(maps.Keys(ends)) {
			if ends[end] != reach {
				end += " for " + ui.HeroWord(ends[end])
			}
			out[n] = append(out[n], end)
		}
	}
	return out
}

// graphAfter is the arrows out of each mission node of a graph, sorted,
// each with the word of the heroes it is labelled with.
func graphAfter(page ui.CampaignGraphPage) map[int][]string {
	name := func(n ui.CampaignGraphNode) string {
		if n.Kind == ui.GraphTown {
			return fmt.Sprintf("town %d", n.Number)
		}
		return fmt.Sprintf("mission %d", n.Number)
	}
	out := map[int][]string{}
	for _, n := range page.Nodes {
		if n.Kind == ui.GraphMission {
			out[n.Number] = []string{}
		}
	}
	for _, e := range page.Edges {
		from := page.Nodes[e.From]
		if from.Kind != ui.GraphMission {
			continue
		}
		end := name(page.Nodes[e.To])
		if e.Heroes != 0 {
			end += " for " + ui.HeroWord(e.Heroes)
		}
		out[from.Number] = append(out[from.Number], end)
	}
	for n := range out {
		slices.Sort(out[n])
	}
	return out
}

// playedAfter is what follows the win of each mission of campaign c with
// the mod missions d, read mission by mission from what the game reads: the
// section's successor (AutoAdvance); else the towns the campaign's own walk
// of the wins gives the mission (missionTowns), through the guest missions
// each starting hero plays (AfterWin); else, for a main mission, the next
// one (NextMission). A side mission a building hands out, with what its
// sections lead to short of a main mission, returns to the building's town;
// set to open the town (TownAfter), it opens the next chapter's.
func playedAfter(t *testing.T, c Campaign, d mod.MissionData) map[int][]string {
	t.Helper()
	towns := c.missionTowns
	if towns == nil {
		walked, err := c.withModMissions(mod.MissionData{})
		if err != nil {
			t.Fatal(err)
		}
		towns = walked.missionTowns
	}
	out := routeArrows{}
	toTowns := func(n int, chapters []int) {
		for _, k := range herokind.PictureOrder {
			won, hero := n, ui.HeroOf(k.Mage(), k.Female())
			for range len(d.Missions) + 1 {
				m, ok := d.AfterWin(won, k)
				if !ok {
					break
				}
				out.add(won, fmt.Sprintf("mission %d", m.Number), hero)
				won = m.Number
			}
			for _, chapter := range chapters {
				out.add(won, fmt.Sprintf("town %d", chapter), hero)
			}
		}
	}
	for _, n := range slices.Concat(c.Main, c.Side) {
		out[n] = map[string]ui.HeroAudience{}
		next, auto := c.AutoAdvance(n)
		switch {
		case auto:
			out.add(n, fmt.Sprintf("mission %d", next), ui.EveryHero)
		case len(towns[n]) > 0:
			toTowns(n, towns[n])
		case slices.Contains(c.Main, n):
			if offer, ok := c.NextMission(n); ok && !offer.Town {
				out.add(n, fmt.Sprintf("mission %d", offer.Mission), ui.EveryHero)
			}
		}
	}
	for chapter, ch := range c.Chapters {
		for _, handed := range slices.Concat(ch.Inn, ch.Shop, ch.School) {
			if handed == 0 || slices.Contains(c.Main, handed) {
				continue
			}
			last, main := handed, false
			for range len(c.Main) + len(c.Side) + 1 {
				next, auto := c.AutoAdvance(last)
				if !auto {
					break
				}
				if last, main = next, slices.Contains(c.Main, next); main {
					break
				}
			}
			if _, cut := c.AutoAdvance(last); main || cut {
				continue
			}
			to := chapter
			if offer, ok := c.NextMission(chapter); ok && c.TownAfter[last] {
				to = offer.Mission
			}
			toTowns(last, []int{to})
		}
	}
	return out.lines()
}

// A mod's change of the route is drawn as the game plays it. A side mission
// sent to a later main mission leads there, and that mission's win opens
// its own next town, not the side mission's. A main mission whose lead the
// mod ends opens the town of the next one, which offers that mission itself
// and goes on from it. Guest missions that between them play for every hero
// leave no arrow straight to the town; an arrow that holds for some heroes
// only names them.
func TestCampaignGraphDrawsTheRouteAModChanges(t *testing.T) {
	stock := routeCampaign(t, routeSections())
	lines := func(src string) ([]string, map[int][]string) {
		c, d := routeMod(t, stock, src)
		page := campaignGraphOf(stock, c, d, nil)
		return graphLines(page), graphAfter(page)
	}
	check := func(name string, after map[int][]string, n int, want ...string) {
		t.Helper()
		if !slices.Equal(after[n], want) {
			t.Errorf("%s: mission %d leads to %q, want %q", name, n, after[n], want)
		}
	}
	holds := func(name string, all []string, want ...string) {
		t.Helper()
		for _, line := range want {
			if !slices.Contains(all, line) {
				t.Errorf("%s: no line %q in\n%s", name, line, strings.Join(all, "\n"))
			}
		}
	}

	_, after := lines(routeSideToMain)
	check("side to main", after, 31, "mission 50")
	check("side to main", after, 50, "town 60")
	check("side to main", after, 30, "town 40")
	check("side to main", after, 41, "town 40")

	all, after := lines(routeLeadOff)
	check("lead off", after, 10, "town 20")
	check("lead off", after, 20, "town 30")
	holds("lead off", all, "town 20 > mission 20")

	all, after = lines(routeTwoGuests)
	check("two guests", after, 20, "mission 25 for editor.message.hero.man", "mission 26 for editor.message.hero.woman")
	check("two guests", after, 25, "town 30")
	check("two guests", after, 26, "town 30")
	holds("two guests", all, "mission 20 > mission 25 (mod) for editor.message.hero.man", "mission 25 > town 30 (mod)")

	_, after = lines(routeOneGuest)
	check("one guest", after, 20, "mission 25 for editor.message.hero.mage", "town 30 for editor.message.hero.fighter")
	check("one guest", after, 25, "town 30")
}

// Over each fixture campaign and mod, every mission node of the graph leads
// to what the campaign plays after its win, for the same heroes, and the
// graph holds a node for every mission the campaign plays.
func TestCampaignGraphFollowsTheCampaignMissionByMission(t *testing.T) {
	twice := routeSections()
	twice[4] = routeChapter(40, []int32{40}, []int32{41, 31})
	unhanded := routeSections()
	unhanded[2] = routeChapter(30, []int32{0}, []int32{31})
	ends := append(routeSections()[:7], routeChapter(60, []int32{60}, []int32{61}), routeSide(61))
	route := routeCampaign(t, routeSections())
	for _, tc := range []struct {
		name  string
		stock Campaign
		mod   string
	}{
		{"marks fixture", graphCampaign(t), ""},
		{"marks fixture with its mod", graphCampaign(t), graphModMissions},
		{"route fixture", route, ""},
		{"side mission sent to a main one", route, routeSideToMain},
		{"lead ended", route, routeLeadOff},
		{"two guests", route, routeTwoGuests},
		{"one guest", route, routeOneGuest},
		{"side mission handed out by two towns", routeCampaign(t, twice), ""},
		{"main mission no building hands out", routeCampaign(t, unhanded), ""},
		{"last town's side mission set to open the town", routeCampaign(t, ends), routeLastSide},
	} {
		c, d := routeMod(t, tc.stock, tc.mod)
		got, want := graphAfter(campaignGraphOf(tc.stock, c, d, nil)), playedAfter(t, c, d)
		for _, n := range slices.Sorted(maps.Keys(want)) {
			if _, drawn := got[n]; !drawn {
				t.Errorf("%s: the graph has no node of mission %d", tc.name, n)
			} else if !slices.Equal(got[n], want[n]) {
				t.Errorf("%s: mission %d leads to %q in the graph; the campaign plays %q", tc.name, n, got[n], want[n])
			}
		}
		for n := range got {
			if _, played := want[n]; !played {
				t.Errorf("%s: the graph draws mission %d, which the campaign does not hold", tc.name, n)
			}
		}
	}
}
