package game

import (
	"bytes"
	"slices"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/herokind"
	"againrom/pkg/mod"
	"againrom/pkg/town"
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

// The columns of the campaign graph: the films, the start and the towns, the
// hand-outs of a town, then the missions of one ten, one column each.
const (
	graphFilmColumn = iota
	graphTownColumn
	graphHandoutColumn
	graphMissionColumn
)

// filmShelf is the films of a movie directory the game finds to play, in
// the order it plays them; CutsceneBank.Films is the game's.
type filmShelf func(directory string) []string

// campaignGraph builds a campaign as a graph: the walk of the campaign makes
// the nodes and the arrows, the start's and each mission's films are added,
// and levels places them by tens.
type campaignGraph struct {
	stock    Campaign // the install's campaign, which says what a mod adds
	c        Campaign
	missions mod.MissionData
	page     ui.CampaignGraphPage
	mission  map[int]int    // the node of a mission, by its number
	town     map[int]int    // the node of a chapter's town
	handouts map[int][]int  // the hand-out nodes of a town's node, in order
	film     map[int]int    // the film node of the start's or a mission's node
	edges    map[[2]int]int // an arrow's place in the page, by its two nodes
	walked   map[[3]int]bool
	laid     map[int]bool                    // the chapters whose town has its hand-outs
	told     map[int]bool                    // the chapters whose town offers its mission itself
	queue    []int                           // the chapters reached and not laid, in that order
	base     map[town.OfferKey]town.OfferRow // the install's offer rows, layer 0
	rows     []town.OfferRow                 // the campaign's offer rows: the merged ones, else layer 0
}

// campaignGraphOf is campaign c with the mod missions it runs with as a
// graph; stock is the install's campaign and films the game's film lookup,
// nil for none. What follows a win is read from what the game reads: the
// campaign's walk of a win (winRoute), whether the win moves the town
// (advances), the guest mission a starting hero plays after it (AfterWin)
// and each chapter's building lists.
func campaignGraphOf(stock, c Campaign, missions mod.MissionData, films filmShelf) ui.CampaignGraphPage {
	g := &campaignGraph{stock: stock, c: c, missions: missions, mission: map[int]int{}, town: map[int]int{},
		handouts: map[int][]int{}, film: map[int]int{}, edges: map[[2]int]int{}, walked: map[[3]int]bool{},
		laid: map[int]bool{}, told: map[int]bool{}, base: map[town.OfferKey]town.OfferRow{}}
	base := stock.Towns.baseOffers(stock)
	for _, r := range base {
		g.base[town.TavernTable.Key(r)] = r
	}
	if rows, ok := c.Towns.tavernRows(); ok {
		base = rows
	}
	g.rows = base
	if first, ok := c.FirstMission(); ok {
		start := g.node(ui.CampaignGraphNode{Kind: ui.GraphStart, Column: graphTownColumn})
		at, _ := g.missionNode(first)
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
	// A mission nothing reaches stands alone. A main one still moves the
	// campaign on; a side one stands on none.
	for _, n := range slices.Sorted(slices.Values(slices.Concat(c.Main, c.Side))) {
		if _, placed := g.mission[n]; !placed {
			stands := 0
			if slices.Contains(c.Main, n) {
				stands = n
			}
			at, _ := g.missionNode(n)
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
	g.films(films)
	g.levels()
	return g.page
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

// missionNode is the node of mission n, made when the graph has none, and
// whether it was made. Its marks are what the mod's rows of that number
// state; it is a side mission unless the campaign counts it main.
func (g *campaignGraph) missionNode(n int) (int, bool) {
	if at, ok := g.mission[n]; ok {
		return at, false
	}
	node := ui.CampaignGraphNode{Kind: ui.GraphMission, Number: n, Side: !slices.Contains(g.c.Main, n)}
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

// townNode is the node of chapter n's town, made and queued for its
// hand-outs when the graph has none.
func (g *campaignGraph) townNode(n int) int {
	if at, ok := g.town[n]; ok {
		return at
	}
	g.town[n] = g.node(ui.CampaignGraphNode{Kind: ui.GraphTown, Number: n})
	g.queue = append(g.queue, n)
	return g.town[n]
}

// reach is the node of mission n, made when the graph had none.
func (g *campaignGraph) reach(n int) int {
	to, _ := g.missionNode(n)
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
	a := g.c.after(last, stood, false)
	if a.declared {
		return // the sections lead round without end
	}
	stays := home != 0 && stood == home && !g.c.advances(last, home)
	switch {
	case stays && !(ok && a.town):
		g.toTown(at, last, home)
	case !ok || stood == 0:
	case offer.Town:
		g.toTown(at, last, offer.Mission)
		// The win set to open the town leaves its main mission announced.
		if g.told[offer.Mission] = g.told[offer.Mission] || a.town; g.laid[offer.Mission] {
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
// building hands the mission out has the arrow already.
func (g *campaignGraph) offered(chapter int) {
	handed := func(h graphHandout) bool { return h.mission == chapter }
	if !g.told[chapter] || slices.ContainsFunc(graphHandouts(g.c.chapter(chapter)), handed) {
		return
	}
	to, _ := g.missionNode(chapter)
	g.edge(g.town[chapter], to, ui.EveryHero)
	g.follow(to, chapter, chapter, 0)
}

// graphHandout is one offer of a town's building: the building, the
// tavern's visitor and the mission, zero for a visitor with none.
type graphHandout struct {
	place   ui.GraphPlace
	npc     int
	mission int
}

// graphBuildings are the town buildings of the graph's hand-out places.
var graphBuildings = map[ui.GraphPlace]TownBuilding{ui.GraphInn: TownTavern, ui.GraphShop: TownShop, ui.GraphSchool: TownSchool}

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

// layTowns gives every queued town its hand-outs and what each handed-out
// mission leads to. A mission whose win moves the town goes on from itself;
// any other is walked from the town that handed it out.
func (g *campaignGraph) layTowns() {
	for len(g.queue) > 0 {
		chapter := g.queue[0]
		g.queue = g.queue[1:]
		g.laid[chapter] = true
		home := g.town[chapter]
		handouts := graphHandouts(g.c.chapter(chapter))
		g.offered(chapter)
		g.page.Nodes[home].Pick = townPick(chapter)
		for _, h := range handouts {
			node := ui.CampaignGraphNode{Kind: ui.GraphHandout, Place: h.place, Number: h.npc, Empty: h.mission == 0}
			// A row is picked by its key; one the install does not hold is
			// the mod's, one it holds otherwise the mod changed.
			if r, ok := chapterRowIn(g.rows, chapter, graphBuildings[h.place], h.npc, h.mission); ok {
				key := town.TavernTable.Key(r)
				node.Pick = rowPick(key)
				prior, held := g.base[key]
				node.AddedMark, node.FlowMark = !held, held && !prior.Equal(r)
			}
			at := g.node(node)
			g.handouts[home] = append(g.handouts[home], at)
			g.edge(home, at, ui.EveryHero)
			if h.mission == 0 {
				continue
			}
			to, _ := g.missionNode(h.mission)
			g.edge(at, to, ui.EveryHero)
			if g.c.advances(h.mission, chapter) {
				g.follow(to, h.mission, h.mission, 0)
			} else {
				g.follow(to, h.mission, chapter, chapter)
			}
		}
	}
}

// films adds a film node for the start and for each mission whose movie
// directory the game finds a film in, named by its files. A film has no arrow:
// its row ties it to its node.
func (g *campaignGraph) films(shelf filmShelf) {
	if shelf == nil {
		return
	}
	for i := range len(g.page.Nodes) {
		n := g.page.Nodes[i]
		directory := ui.StartCutscenes
		switch n.Kind {
		case ui.GraphStart:
		case ui.GraphMission:
			directory = ui.MissionCutscenes(n.Number)
		default:
			continue
		}
		if files := shelf(directory); len(files) > 0 {
			g.film[i] = g.node(ui.CampaignGraphNode{Kind: ui.GraphFilm, Number: n.Number, Title: strings.Join(files, ", ")})
		}
	}
}

// levels places the nodes by tens. Mission n stands on level n/10, so 10 to
// 19 share a level and a number under 10 stands on level 0; the missions of
// a level take one row, in number order, a column each from the mission
// column on. A film stands in the film column at its node's row, dropped
// below it; when a level has two films, the second takes the row under the
// first. The town of chapter n stands on n's level in the town column, its
// hand-outs one a row beside it from its row on, the towns of a level in
// chapter order. The start and its film stand above every level; a level is
// as tall as its rows.
func (g *campaignGraph) levels() {
	nodes := g.page.Nodes
	missions, towns := map[int][]int{}, map[int][]int{}
	var levels []int
	for i, n := range nodes {
		level := n.Number / mainMissionStride
		switch n.Kind {
		case ui.GraphMission:
			missions[level] = append(missions[level], i)
		case ui.GraphTown:
			towns[level] = append(towns[level], i)
		default:
			continue
		}
		if !slices.Contains(levels, level) {
			levels = append(levels, level)
		}
	}
	slices.Sort(levels)
	row := 0
	for i, n := range nodes {
		if n.Kind == ui.GraphStart {
			row = 1
			if film, ok := g.film[i]; ok {
				nodes[film].Column, nodes[film].Row, nodes[film].Dropped = graphFilmColumn, 0, true
			}
		}
	}
	byNumber := func(a, b int) int { return nodes[a].Number - nodes[b].Number }
	for _, level := range levels {
		next := row
		slices.SortFunc(missions[level], byNumber)
		films := 0
		for k, i := range missions[level] {
			nodes[i].Column, nodes[i].Row, next = graphMissionColumn+k, row, max(next, row+1)
			if film, ok := g.film[i]; ok {
				nodes[film].Column, nodes[film].Row, nodes[film].Dropped = graphFilmColumn, row+films, true
				films++
				next = max(next, row+films)
			}
		}
		at := row
		slices.SortFunc(towns[level], byNumber)
		for _, i := range towns[level] {
			nodes[i].Column, nodes[i].Row = graphTownColumn, at
			for k, h := range g.handouts[i] {
				nodes[h].Column, nodes[h].Row = graphHandoutColumn, at+k
			}
			at += max(1, len(g.handouts[i]))
		}
		row = max(next, at)
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

// films is the install's film bank on the speed route a player who names
// none plays from: the films the game finds to play are the graph's.
func (x *MapInspector) films() *CutsceneBank {
	return openCutscenes(x.archives.Root, x.archives.Base.Profile.Edition().Cutscenes, CutsceneArchive(false, false))
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

func (x *MapInspector) graphUnread() bool {
	book := townBookOf(x.archives.Base.Profile)
	return book != nil && book.catalog != nil && !townCatalogTakesLayer(book.catalog)
}

// CampaignGraph reads the admitted project draft, or the install when the
// mod is absent or refused. Missions open through the existing map routes.
func (s *mapInspectionEdits) CampaignGraph() ui.CampaignGraphPage {
	x := s.inspector
	stock := x.installFlow()
	if stock == nil {
		return ui.CampaignGraphPage{}
	}
	page := ui.CampaignGraphPage{Mod: x.mod != nil}
	if x.graphUnread() {
		page.Unread = true
		return page
	}
	flow := stock
	if x.mod != nil {
		page.Refused = x.mod.refusal != ""
		if page.Unbuilt = x.mod.flow == nil && !page.Refused; x.mod.flow != nil {
			flow = x.mod.flow
		}
		if s.workspace != nil && s.workspace.Project != nil && !page.Refused && !page.Unbuilt {
			if !bytes.Equal(s.flowImage, s.workspaceImage) || s.draftFlow == nil {
				if draft, err := s.missionDraft(s.workspace); err == nil {
					s.draftFlow, s.flowImage = draft, s.workspaceImage
				}
			}
			if s.draftFlow != nil {
				flow = s.draftFlow
			}
		}
	}
	if flow.graph == nil {
		built := campaignGraphOf(stock.campaign, flow.campaign, flow.missions, x.films().Films)
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
		n := &page.Nodes[i]
		n.Selected = n.Pick != "" && n.Pick == s.handout
		if !page.Mod || page.Refused || page.Unbuilt {
			n.Pick = ""
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
