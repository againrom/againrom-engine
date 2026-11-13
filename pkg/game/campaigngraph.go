package game

import (
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/herokind"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// campaignFlow is a campaign and the mod missions it runs with, as the
// install or one admitted fork holds them, and the graph built of them once
// W asked for it.
type campaignFlow struct {
	campaign Campaign
	missions mod.MissionData
	graph    *ui.CampaignGraphPage
}

// The columns of the campaign graph: the start and the towns, the hand-outs
// of a town, the missions.
const (
	graphTownColumn = iota
	graphHandoutColumn
	graphMissionColumn
)

// campaignGraph lays a campaign as a graph. A node takes the next free row
// when it is reached, so the rows follow the campaign: the first mission and
// what follows it without a town, then each town in the order the wins open
// them, its hand-outs in the scenario's order and each hand-out's mission on
// the hand-out's row.
type campaignGraph struct {
	stock    Campaign // the install's campaign, which says what a mod adds
	c        Campaign
	missions mod.MissionData
	page     ui.CampaignGraphPage
	mission  map[int]int    // the node of a mission, by its number
	town     map[int]int    // the node of a chapter's town
	edges    map[[2]int]int // an arrow's place in the page, by its two nodes
	walked   map[[3]int]bool
	laid     map[int]bool // the chapters whose town has its rows
	told     map[int]bool // the chapters whose town offers its mission itself
	queue    []int        // the chapters reached and not laid, in that order
	row      int
}

// campaignGraphOf is campaign c with the mod missions it runs with as a
// graph; stock is the install's campaign. What follows a win is read from
// what the game reads: the campaign's walk of a win (winRoute), whether the
// win moves the town (advances), the guest mission a starting hero plays
// after it (AfterWin) and each chapter's building lists.
func campaignGraphOf(stock, c Campaign, missions mod.MissionData) ui.CampaignGraphPage {
	g := &campaignGraph{stock: stock, c: c, missions: missions, mission: map[int]int{}, town: map[int]int{},
		edges: map[[2]int]int{}, walked: map[[3]int]bool{}, laid: map[int]bool{}, told: map[int]bool{}}
	if first, ok := c.FirstMission(); ok {
		start := g.node(ui.CampaignGraphNode{Kind: ui.GraphStart, Column: graphTownColumn, Row: g.nextRow()})
		at, _ := g.missionNode(first, g.page.Nodes[start].Row)
		g.edge(start, at, ui.EveryHero)
		g.follow(at, first, first, 0)
	}
	g.layTowns()
	chapters := make([]int, 0, len(c.Chapters))
	for n := range c.Chapters {
		chapters = append(chapters, n)
	}
	slices.Sort(chapters)
	for _, n := range chapters {
		if !g.laid[n] && len(graphHandouts(c.chapter(n))) > 0 {
			g.townNode(n)
			g.layTowns()
		}
	}
	// A mission nothing reaches stands alone under the towns. A main one
	// still moves the campaign on; a side one stands on none.
	for _, n := range slices.Sorted(slices.Values(slices.Concat(c.Main, c.Side))) {
		if _, placed := g.mission[n]; !placed {
			stands := 0
			if slices.Contains(c.Main, n) {
				stands = n
			}
			at, _ := g.missionNode(n, g.nextRow())
			g.follow(at, n, stands, 0)
			g.layTowns()
		}
	}
	// An arrow names its heroes only where the arrows out of its node differ.
	reach := map[int]ui.HeroAudience{}
	for _, e := range g.page.Edges {
		reach[e.From] |= e.Heroes
	}
	for i := range g.page.Edges {
		if e := &g.page.Edges[i]; e.Heroes == reach[e.From] {
			e.Heroes = 0
		}
	}
	return g.page
}

func (g *campaignGraph) nextRow() int {
	g.row++
	return g.row - 1
}

func (g *campaignGraph) node(n ui.CampaignGraphNode) int {
	g.page.Nodes = append(g.page.Nodes, n)
	return len(g.page.Nodes) - 1
}

// edge draws one arrow once, for the starting heroes given; drawn again it
// holds for those heroes too. It is marked when the open mod set it: it
// leaves a mission whose successor the mod states, or ends at or leaves a
// mission the mod adds.
func (g *campaignGraph) edge(from, to int, heroes ui.HeroAudience) {
	if at, ok := g.edges[[2]int{from, to}]; ok {
		g.page.Edges[at].Heroes |= heroes
		return
	}
	g.edges[[2]int{from, to}] = len(g.page.Edges)
	a, b := g.page.Nodes[from], g.page.Nodes[to]
	marked := a.Kind == ui.GraphMission && (a.FlowMark || a.AddedMark) || b.Kind == ui.GraphMission && b.AddedMark
	g.page.Edges = append(g.page.Edges, ui.CampaignGraphEdge{From: from, To: to, Marked: marked, Heroes: heroes})
}

// missionNode is the node of mission n, made on row when the graph has none,
// and whether it was made. Its marks are what the mod's rows of that number
// state.
func (g *campaignGraph) missionNode(n, row int) (int, bool) {
	if at, ok := g.mission[n]; ok {
		return at, false
	}
	node := ui.CampaignGraphNode{Kind: ui.GraphMission, Number: n, Column: graphMissionColumn, Row: row}
	for _, m := range g.missions.Missions {
		if m.Off || m.Number != n {
			continue
		}
		switch {
		case !m.Campaign() || !g.stock.holds(n):
			node.AddedMark = true
		case m.Map != "":
			node.MapMark = true
		}
		node.FlowMark = node.FlowMark || m.Campaign() && m.AutoGetMission != nil
		node.PartyMark = node.PartyMark || m.GuestParty()
	}
	g.mission[n] = g.node(node)
	return g.mission[n], true
}

// townNode is the node of chapter n's town, made and queued for its rows
// when the graph has none.
func (g *campaignGraph) townNode(n int) int {
	if at, ok := g.town[n]; ok {
		return at
	}
	g.town[n] = g.node(ui.CampaignGraphNode{Kind: ui.GraphTown, Number: n, Column: graphTownColumn})
	g.queue = append(g.queue, n)
	return g.town[n]
}

// reach is the node of mission n, on the next free row when the graph had
// none.
func (g *campaignGraph) reach(n int) int {
	to, made := g.missionNode(n, g.row)
	if made {
		g.nextRow()
	}
	return to
}

// follow draws what the win of mission n, the node at, leads to, by the
// campaign's walk of that win. stands is the main mission the campaign
// stands on, zero for a side mission nothing reaches; home is the chapter
// of the town n was taken in, zero for none. A win that moves that town
// nowhere returns to it; any other opens the town of the walk's offer, or
// leads to the offer's mission where no town stands before it.
func (g *campaignGraph) follow(at, n, stands, home int) {
	if g.walked[[3]int{n, stands, home}] {
		return
	}
	g.walked[[3]int{n, stands, home}] = true
	path, stood, offer, ok := g.c.winRoute(n, stands)
	for _, next := range path[1:] {
		to := g.reach(next)
		g.edge(at, to, ui.EveryHero)
		at = to
	}
	last := path[len(path)-1]
	if _, more := g.c.AutoAdvance(last); more {
		return // the sections lead round without end
	}
	stays := home != 0 && stood == home && !g.c.advances(last, home)
	switch {
	case stays && !(ok && g.c.TownAfter[last]):
		g.toTown(at, last, home)
	case !ok || stood == 0:
	case offer.Town:
		g.toTown(at, last, offer.Mission)
		// The win set to open the town leaves its main mission announced.
		if g.told[offer.Mission] = g.told[offer.Mission] || g.c.TownAfter[last]; g.laid[offer.Mission] {
			g.offered(offer.Mission)
		}
	default:
		to := g.reach(offer.Mission)
		g.edge(at, to, ui.EveryHero)
		g.follow(to, offer.Mission, offer.Mission, 0)
	}
}

// toTown draws the win of mission n, the node at, opening the town of
// chapter, for each starting hero: through the guest missions the mod plays
// for that hero after the win (AfterWin), then into the town.
func (g *campaignGraph) toTown(at, n, chapter int) {
	for _, k := range herokind.PictureOrder {
		from, won, hero := at, n, ui.HeroOf(k.Mage(), k.Female())
		for range g.missions.Missions {
			m, ok := g.missions.AfterWin(won, k)
			if !ok {
				break
			}
			to := g.reach(m.Number)
			g.edge(from, to, hero)
			from, won = to, m.Number
		}
		g.edge(from, g.townNode(chapter), hero)
	}
}

// offered draws a town that offers its main mission itself, as a win set to
// open the town leaves it, and what that mission leads to; a town whose
// building hands the mission out has the arrow already. It reports that the
// mission took the town's row.
func (g *campaignGraph) offered(chapter int) bool {
	handed := func(h graphHandout) bool { return h.mission == chapter }
	if !g.told[chapter] || slices.ContainsFunc(graphHandouts(g.c.chapter(chapter)), handed) {
		return false
	}
	to, made := g.missionNode(chapter, g.row)
	if made {
		g.nextRow()
	}
	g.edge(g.town[chapter], to, ui.EveryHero)
	g.follow(to, chapter, chapter, 0)
	return made
}

// graphHandout is one offer of a town's building: the building, the
// tavern's visitor and the mission, zero for a visitor with none.
type graphHandout struct {
	place   ui.GraphPlace
	npc     int
	mission int
}

// graphHandouts is a chapter's hand-outs in the scenario's order: the
// tavern's pairs over the shorter of its two lists, the shop's missions,
// the school's.
func graphHandouts(ch Chapter) []graphHandout {
	var out []graphHandout
	for i := range min(len(ch.InnNPC), len(ch.Inn)) {
		out = append(out, graphHandout{place: ui.GraphInn, npc: ch.InnNPC[i], mission: ch.Inn[i]})
	}
	for _, n := range ch.Shop {
		out = append(out, graphHandout{place: ui.GraphShop, mission: n})
	}
	for _, n := range ch.School {
		out = append(out, graphHandout{place: ui.GraphSchool, mission: n})
	}
	return out
}

// layTowns gives every queued town its rows: the town on its first
// hand-out's row, one row a hand-out, and what each handed-out mission
// leads to. A mission whose win moves the town goes on from itself; any
// other is walked from the town that handed it out.
func (g *campaignGraph) layTowns() {
	for len(g.queue) > 0 {
		chapter := g.queue[0]
		g.queue = g.queue[1:]
		g.laid[chapter] = true
		town := g.town[chapter]
		g.page.Nodes[town].Row = g.row
		handouts := graphHandouts(g.c.chapter(chapter))
		if !g.offered(chapter) && len(handouts) == 0 {
			g.nextRow()
		}
		for _, h := range handouts {
			row := g.nextRow()
			at := g.node(ui.CampaignGraphNode{Kind: ui.GraphHandout, Place: h.place, Number: h.npc,
				Column: graphHandoutColumn, Row: row, Empty: h.mission == 0})
			g.edge(town, at, ui.EveryHero)
			if h.mission == 0 {
				continue
			}
			to, _ := g.missionNode(h.mission, row)
			g.edge(at, to, ui.EveryHero)
			if g.c.advances(h.mission, chapter) {
				g.follow(to, h.mission, h.mission, 0)
			} else {
				g.follow(to, h.mission, chapter, chapter)
			}
		}
	}
}

// installFlow is the install's campaign with no mod, with the town book the
// game's launch gives it, read once; nil with no archives open or no
// campaign to read.
func (x *MapInspector) installFlow() *campaignFlow {
	if x.archives == nil || x.archives.Containers == nil {
		return nil
	}
	if x.flow == nil {
		x.flow = &campaignFlow{}
		if c, err := LoadCampaign(x.archives.Containers); err == nil {
			c.Towns = townBookOf(x.archives.Base.Profile)
			x.flow.campaign = c
		}
	}
	return x.flow
}

// graphMission is what a mission node shows and opens: the title the map
// list gives mission n and the key of its install map, both empty for a
// mission the install lists no map of.
func (x *MapInspector) graphMission(n int) (title, source string) {
	for _, e := range x.Maps {
		if number, ok := missionNumber(e); ok && number == n {
			e.Name = alm.MapNameText.Decode([]byte(e.Name))
			return x.mapTitle(e), ui.MapSourceKey(ui.SourceScenario, e.Source)
		}
	}
	return "", ""
}

// CampaignGraph is W's campaign graph page: the install's campaign, or with
// a mod open the campaign of the mod's admitted fork, the one the game's
// launch builds. A mission the mod holds the map of opens as its row of W's
// mission list does; any other opens its install map. The graph of a mod the
// game refuses, or whose fork was not built, is the install's.
func (s *mapInspectionEdits) CampaignGraph() ui.CampaignGraphPage {
	x := s.inspector
	stock := x.installFlow()
	if stock == nil {
		return ui.CampaignGraphPage{}
	}
	page := ui.CampaignGraphPage{Mod: x.mod != nil}
	if book := townBookOf(x.archives.Base.Profile); book != nil && book.catalog != nil && !townCatalogTakesLayer(book.catalog) {
		page.Unread = true
		return page
	}
	flow := stock
	if x.mod != nil {
		page.Refused = x.mod.refusal != ""
		if page.Unbuilt = x.mod.flow == nil && !page.Refused; x.mod.flow != nil {
			flow = x.mod.flow
		}
	}
	if flow.graph == nil {
		built := campaignGraphOf(stock.campaign, flow.campaign, flow.missions)
		for i := range built.Nodes {
			n := &built.Nodes[i]
			if n.Kind != ui.GraphMission {
				continue
			}
			n.Title, n.Source = x.graphMission(n.Number)
			for _, m := range flow.missions.Missions {
				if m.Off || m.Number != n.Number {
					continue
				}
				if m.Title != "" {
					n.Title = m.Title
				}
				if m.Map != "" {
					// The mod's map is the mission's; the install's is not offered.
					n.Source, n.ModMission = "", -1
				}
			}
		}
		flow.graph = &built
	}
	page.Nodes, page.Edges = slices.Clone(flow.graph.Nodes), flow.graph.Edges
	for i := range page.Nodes {
		if n := &page.Nodes[i]; n.ModMission < 0 {
			n.ModMission = s.modMissionRow(n.Number)
		}
	}
	return page
}

// modMissionRow is the row, from one, of W's mission list that holds the map
// of mission n; zero when no row that opens does.
func (s *mapInspectionEdits) modMissionRow(n int) int {
	if s.workspace == nil || s.workspace.Project == nil {
		return 0
	}
	for i, m := range s.workspace.Project.Missions {
		if !m.Off && m.Map != "" && m.Number == n {
			return i + 1
		}
	}
	return 0
}
