package ui

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/words"
)

// modesGraphFirst is the node of modesGraph's first mission, whose map is
// map1; modesGraphSide the node of a mission with no map to open.
const (
	modesGraphFirst = 1
	modesGraphSide  = 7
)

// modesGraph is a campaign as a graph: the start, two missions one after the
// other, then six towns, each with a tavern visitor's main mission, a shop's
// side mission that returns to the town, and a visitor with nothing.
func modesGraph() CampaignGraphPage {
	var p CampaignGraphPage
	add := func(n CampaignGraphNode) int {
		p.Nodes = append(p.Nodes, n)
		return len(p.Nodes) - 1
	}
	link := func(from, to int) { p.Edges = append(p.Edges, CampaignGraphEdge{From: from, To: to}) }
	start := add(CampaignGraphNode{Kind: GraphStart})
	first := add(CampaignGraphNode{Kind: GraphMission, Number: 10, Title: "Crossing", Column: 2, Source: "map1"})
	second := add(CampaignGraphNode{Kind: GraphMission, Number: 20, Title: "Watch", Column: 2, Row: 1, Source: "map2"})
	link(start, first)
	link(first, second)
	last, row := second, 2
	for chapter := 30; chapter <= 80; chapter += 10 {
		town := add(CampaignGraphNode{Kind: GraphTown, Number: chapter, Row: row})
		inn := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphInn, Number: 100 + chapter, Column: 1, Row: row})
		main := add(CampaignGraphNode{Kind: GraphMission, Number: chapter, Title: fmt.Sprintf("Main %d", chapter), Column: 2, Row: row, Source: "map3"})
		shop := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphShop, Column: 1, Row: row + 1})
		side := add(CampaignGraphNode{Kind: GraphMission, Number: chapter + 1, Title: "Side", Column: 2, Row: row + 1})
		none := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphInn, Number: 200 + chapter, Column: 1, Row: row + 2, Empty: true})
		for _, e := range [][2]int{{last, town}, {town, inn}, {inn, main}, {town, shop}, {shop, side}, {side, town}, {town, none}} {
			link(e[0], e[1])
		}
		last, row = main, row+3
	}
	return p
}

// graphEditor is an editor whose W shows the graph page of page.
func graphEditor(t *testing.T, page CampaignGraphPage) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := newModesEditor(t, "workspace")
	e.doc.Edits.(*modesWorkspace).graph = page
	_ = e.Key("package")
	_ = e.Key("package")
	if !e.modWindow || e.modPage != modPageGraph {
		t.Fatalf("W shows page %d, open %v", e.modPage, e.modWindow)
	}
	return e, calls
}

// graphLayout is W's spec and layout on its graph page.
func graphLayout(t *testing.T, e *MapEditor) (popupSpec, popupLayout) {
	t.Helper()
	s, _, ok := e.objectSpec()
	if !ok || s.Canvas == nil {
		t.Fatalf("W has no graph: ok %v page %d", ok, e.modPage)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	// A press is read against the window as it is drawn.
	if raw, _, _ := e.modSpec(); !reflect.DeepEqual(layoutPopup(raw, e.font, e.bodyFont()).CanvasBoxes, l.CanvasBoxes) {
		t.Fatal("the pressed layout is not the drawn one")
	}
	return s, l
}

// The canvas is an element of the popup builder: any spec holds one. Its
// view is cut to its height, a sheet taller than the view takes a bar, a
// press lands on the box that takes one and elsewhere on the ground, and the
// one painter draws it.
func TestPopupCanvasIsLaidHitAndPainted(t *testing.T) {
	f := panelFont()
	c := &popupCanvas{Sheet: image.Pt(200, 300), Height: 100, Boxes: []canvasBox{
		{Area: image.Rect(10, 10, 90, 40), Lines: []string{"frame"}, Inert: true, Fill: editorPanel, Edge: editorGold},
		{Area: image.Rect(10, 50, 90, 80), Lines: []string{"press"}, Marks: []canvasMark{{Text: "new", Ink: editorLight}}, Tip: "a tip"},
		{Area: image.Rect(10, 250, 90, 280), Lines: []string{"below"}},
	}, Lines: []canvasLine{{Points: []image.Point{{50, 40}, {50, 49}}, Ink: editorPaper, Arrow: true}}}
	s := popupSpec{Title: "Any window", Width: 300, At: image.Pt(20, 20), Canvas: c, Buttons: []string{"Close"}}
	l := layoutPopup(s, f, f)
	if l.Canvas.Dy() != 100 || l.CanvasBar.Empty() || !l.Canvas.In(l.Rect) || l.Canvas.Overlaps(l.CanvasBar) {
		t.Fatalf("view %v bar %v in %v", l.Canvas, l.CanvasBar, l.Rect)
	}
	for _, tc := range []struct {
		at   image.Point
		part popupPart
		n    int
	}{
		{center(l.CanvasBoxes.rect(0)), popupCanvasGround, 0},
		{center(l.CanvasBoxes.rect(1)), popupCanvasBox, 1},
		{l.Canvas.Max.Sub(image.Pt(3, 3)), popupCanvasGround, 0},
		{center(l.CanvasBar), popupCanvasBar, 0},
	} {
		if part, n := l.hit(tc.at); part != tc.part || n != tc.n {
			t.Errorf("hit at %v is part %d box %d, want %d %d", tc.at, part, n, tc.part, tc.n)
		}
	}
	// The box below the view is no part of the window until the sheet moves.
	if p := center(l.CanvasBoxes.rect(2)); p.In(l.Canvas) {
		t.Fatalf("the third box %v shows in %v", l.CanvasBoxes.rect(2), l.Canvas)
	}
	moved := *c
	moved.At = image.Pt(0, 1000)
	s.Canvas = &moved
	low := layoutPopup(s, f, f)
	if low.CanvasAt != image.Pt(0, 200) {
		t.Fatalf("the sheet stands at %v, want its end 0,200", low.CanvasAt)
	}
	if part, n := low.hit(center(low.CanvasBoxes.rect(2))); part != popupCanvasBox || n != 2 {
		t.Fatalf("the third box is part %d box %d", part, n)
	}
	if bar := low.canvasBar(&moved); bar.Pos != bar.Count-1 {
		t.Fatalf("bar at %d of %d at the sheet's end", bar.Pos, bar.Count)
	}
	pic := image.NewRGBA(image.Rect(0, 0, 400, 300))
	drawPopup(pic, f, f, nil, popupSpec{Title: "Any window", Width: 300, At: image.Pt(20, 20), Canvas: c, Buttons: []string{"Close"}}, l, editorPopupInk, image.Pt(-1, -1), false)
	edge := l.CanvasBoxes.rect(0).Min
	if got := pic.RGBAAt(edge.X, edge.Y); got != editorGold {
		t.Fatalf("the frame's edge is %v, want %v", got, editorGold)
	}
	line := l.Canvas.Min.Add(image.Pt(50, 44))
	if got := pic.RGBAAt(line.X, line.Y); got != editorPaper {
		t.Fatalf("the line's pixel is %v, want %v", got, editorPaper)
	}
	if tip, ok := popupTipAt(pic.Bounds(), f, f, s, l, center(l.CanvasBoxes.rect(1)), true); !ok || len(tip.Lines) != 1 || tip.Lines[0] != "a tip" {
		t.Fatalf("tip %v %v", tip, ok)
	}
	if got := c.texts(); !slices.Equal(got, []string{"frame", "press new", "below"}) {
		t.Fatalf("texts %q", got)
	}
}

// W gains the graph as a page after its others: a tab, and the window's key
// turns through the pages in tab order. The page states the campaign in the
// editor's words, in English and in Russian, and has no mod legend with no
// mod open.
func TestModWindowTurnsToTheCampaignGraph(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	host := e.doc.Edits.(*modesWorkspace)
	host.overview, host.graph = modesOverview, modesGraph()
	_ = e.Key("package")
	s, _, _ := e.modSpec()
	if e.modPage != modPageOverview || len(s.Tabs) != 4 || s.Tabs[2] != "Campaign graph" {
		t.Fatalf("page %d tabs %q", e.modPage, s.Tabs)
	}
	for _, want := range []int{modPageGraph, modPageMaps, modPageFields, modPageOverview} {
		if _ = e.Key("package"); e.modPage != want || !e.modWindow {
			t.Fatalf("the window's key shows page %d, want %d", e.modPage, want)
		}
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	modesClick(e, center(l.Tabs[2]))
	if s, _, _ = e.modSpec(); e.modPage != modPageGraph || s.ActiveTab != 2 || s.Canvas == nil {
		t.Fatalf("the graph's tab shows page %d, tab %d", e.modPage, s.ActiveTab)
	}
	before := len(e.doc.Edits.(*modesWorkspace).calls.lines)
	lines := e.PopupLines()
	for _, want := range []string{
		"The campaign of the install. A click on a mission opens its map.",
		"Start", "Mission 10 Crossing", "Mission 20 Watch", "Town 30", "Tavern, NPC 130", "Mission 30 Main 30",
		"Shop", "Mission 31 Side", "Tavern, NPC 230: nothing", "Town 80",
	} {
		if !slices.Contains(lines, want) {
			t.Errorf("the graph lacks %q", want)
		}
	}
	if joined := strings.Join(lines, "\n"); strings.Contains(joined, "the mod") {
		t.Errorf("a graph with no mod states mod marks:\n%s", joined)
	}
	_ = e.ChromeImage()
	e.SetWords(words.For("ru"))
	lines = e.PopupLines()
	for _, want := range []string{"Кампания установленной игры. Щелчок по миссии открывает ее карту.", "Начало", "Миссия 10 Crossing", "Город 30", "Таверна, NPC 130", "Лавка", "Таверна, NPC 230: ничего"} {
		if !slices.Contains(lines, want) {
			t.Errorf("the Russian graph lacks %q", want)
		}
	}
	if s, _, _ = e.modSpec(); s.Tabs[2] != "Граф кампании" {
		t.Errorf("the Russian tab is %q", s.Tabs[2])
	}
	if got := len(e.doc.Edits.(*modesWorkspace).calls.lines); got != before {
		t.Errorf("showing the graph called the host %d times", got-before)
	}
	// A host with a graph alone gives W its page beside the two a mod has.
	two, _ := graphEditor(t, modesGraph())
	if s, _, _ := two.modSpec(); len(s.Tabs) != 3 || s.ActiveTab != 1 {
		t.Fatalf("with a graph alone: tabs %q active %d", s.Tabs, s.ActiveTab)
	}
	two.doc.Edits.(*modesWorkspace).graph = CampaignGraphPage{}
	if s, _, _ := two.modSpec(); len(s.Tabs) != 2 || two.modPage != modPageFields {
		t.Fatalf("with the graph gone: tabs %q page %d", s.Tabs, two.modPage)
	}
}

// Each mod mark stands on its node as a word in its ink, and a caption line
// says each in words. A mod the game refuses, and an install whose campaign
// the graph does not read, say so in the window.
func TestCampaignGraphStatesTheModMarksInWords(t *testing.T) {
	page := modesGraph()
	page.Mod = true
	page.Nodes[modesGraphFirst].MapMark = true
	page.Nodes[modesGraphFirst+1].FlowMark = true
	page.Nodes[modesGraphSide].AddedMark, page.Nodes[modesGraphSide].PartyMark = true, true
	page.Edges[1].Marked = true
	e, _ := graphEditor(t, page)
	lines := e.PopupLines()
	for _, want := range []string{
		"The campaign with the open mod's changes. A click on a mission opens its map.",
		"map the mod replaces this mission's map",
		"flow the mod changes what follows it; its arrow has this colour",
		"new the mod adds this mission",
		"party a guest party of the mod plays this mission",
		"Mission 10 Crossing map", "Mission 20 Watch flow", "Mission 31 Side new party",
	} {
		if !slices.Contains(lines, want) {
			t.Errorf("the graph lacks %q in\n%s", want, strings.Join(lines, "\n"))
		}
	}
	s, _ := graphLayout(t, e)
	if got := s.Canvas.Boxes[modesGraphSide].Marks; len(got) != 2 || got[0].Ink != graphAddedInk || got[1].Ink != graphPartyInk {
		t.Errorf("marks of the added mission: %v", got)
	}
	if s.Canvas.Lines[1].Ink != graphFlowInk || s.Canvas.Lines[0].Ink == graphFlowInk {
		t.Errorf("edge inks %v %v", s.Canvas.Lines[0].Ink, s.Canvas.Lines[1].Ink)
	}
	e.SetWords(words.For("ru"))
	lines = e.PopupLines()
	for _, want := range []string{"карта мод заменяет карту этой миссии", "порядок мод меняет то, что идет после нее; ее стрелка этого цвета",
		"новая мод добавляет эту миссию", "отряд эту миссию играет гостевой отряд мода", "Миссия 31 Side новая отряд"} {
		if !slices.Contains(lines, want) {
			t.Errorf("the Russian graph lacks %q", want)
		}
	}
	refused := modesGraph()
	refused.Mod, refused.Refused = true, true
	e, _ = graphEditor(t, refused)
	if lines := e.PopupLines(); !strings.HasPrefix(lines[0], "The game refuses this mod's missions") {
		t.Errorf("a refused mod's graph starts %q", lines[0])
	}
	unbuilt := modesGraph()
	unbuilt.Mod, unbuilt.Unbuilt = true, true
	e, _ = graphEditor(t, unbuilt)
	if lines := e.PopupLines(); !strings.HasPrefix(lines[0], "The open mod's campaign could not be built") {
		t.Errorf("the graph of a mod whose campaign was not built starts %q", lines[0])
	}
	e.SetWords(words.For("ru"))
	if lines := e.PopupLines(); !strings.HasPrefix(lines[0], "Кампанию открытого мода собрать не удалось") {
		t.Errorf("the Russian graph of such a mod starts %q", lines[0])
	}
	e, _ = graphEditor(t, CampaignGraphPage{Unread: true})
	s, _, _ = e.modSpec()
	if lines := strings.Join(e.PopupLines(), " "); s.Canvas != nil || !strings.Contains(lines, "No campaign graph for this game") {
		t.Errorf("an install the graph does not read: canvas %v, lines %q", s.Canvas != nil, lines)
	}
	e.SetWords(words.For("ru"))
	if lines := strings.Join(e.PopupLines(), " "); !strings.Contains(lines, "Для этой игры графа кампании нет") {
		t.Errorf("the Russian page of such an install: %q", lines)
	}
}

// A graph larger than the window moves under its view: by the arrow and
// page keys, the wheel, a drag of the ground and the bar, and never past
// the sheet's edge.
func TestCampaignGraphMovesUnderItsView(t *testing.T) {
	e, _ := graphEditor(t, modesGraph())
	s, l := graphLayout(t, e)
	room := s.Canvas.Sheet.Sub(l.Canvas.Size())
	if room.Y <= 0 || l.CanvasBar.Empty() {
		t.Fatalf("the sheet %v fits the view %v", s.Canvas.Sheet, l.Canvas)
	}
	at := func() image.Point { _, l := graphLayout(t, e); return l.CanvasAt }
	for _, tc := range []struct {
		key  string
		want image.Point
	}{
		{"down", image.Pt(0, l.Pitch)}, {"down", image.Pt(0, 2*l.Pitch)}, {"up", image.Pt(0, l.Pitch)},
		{"end", image.Pt(0, room.Y)}, {"down", image.Pt(0, room.Y)}, {"pageup", image.Pt(0, max(0, room.Y-(l.Canvas.Dy()-l.Pitch)))},
		{"home", image.Pt(0, 0)}, {"up", image.Pt(0, 0)}, {"pagedown", image.Pt(0, min(room.Y, l.Canvas.Dy()-l.Pitch))},
		{"right", image.Pt(max(0, min(room.X, 2*graphGapX)), min(room.Y, l.Canvas.Dy()-l.Pitch))}, {"home", image.Pt(max(0, min(room.X, 2*graphGapX)), 0)},
		{"left", image.Pt(0, 0)},
	} {
		if err := e.Key(tc.key); err != nil || at() != tc.want {
			t.Fatalf("%s: the sheet stands at %v, want %v (%v)", tc.key, at(), tc.want, err)
		}
	}
	if !e.modWindow || e.modPage != modPageGraph {
		t.Fatalf("a key left the page: window %v page %d", e.modWindow, e.modPage)
	}
	ground := center(l.CanvasBoxes.rect(0))
	e.Pointer(Input{CursorX: ground.X, CursorY: ground.Y, WheelY: -1}, time.Unix(1, 0))
	wheeled := at()
	if wheeled.Y <= 0 {
		t.Fatalf("the wheel left the sheet at %v", wheeled)
	}
	modesPress(e, ground, true)
	modesPress(e, ground.Sub(image.Pt(0, 30)), true)
	if got := at(); got != wheeled.Add(image.Pt(0, 30)) {
		t.Fatalf("a drag of 30 moved the sheet from %v to %v", wheeled, got)
	}
	modesPress(e, ground.Sub(image.Pt(0, 30)), false)
	modesPress(e, ground, false)
	if got := at(); got != wheeled.Add(image.Pt(0, 30)) || e.graphHeld {
		t.Fatalf("the sheet moved after the release: %v held %v", got, e.graphHeld)
	}
	_ = e.Key("home")
	foot := image.Pt(center(l.CanvasBar).X, l.CanvasBar.Max.Y-3)
	modesClick(e, foot)
	if got := at(); got.Y <= 0 {
		t.Fatalf("the bar's lower arrow left the sheet at %v", got)
	}
	if e.Dirty() || !e.modWindow {
		t.Fatalf("moving the graph: dirty %v window %v", e.Dirty(), e.modWindow)
	}
}

// A press on a mission's node opens the mission's map by the route the map
// list opens one, with its question about unsaved changes; a mission the
// open mod holds opens as its row of W's mission list does. Every other
// node, and a mission with no map, takes no press.
func TestCampaignGraphNodeOpensItsMissionsMap(t *testing.T) {
	e, calls := graphEditor(t, modesGraph())
	_, l := graphLayout(t, e)
	for _, n := range []int{0, 3, 4, modesGraphSide} {
		modesClick(e, center(l.CanvasBoxes.rect(n)))
		if e.doc.Source != "map0" || asked(e) || len(calls.lines) != 0 {
			t.Fatalf("node %d: map %q asked %v calls %q", n, e.doc.Source, asked(e), calls.lines)
		}
	}
	modesClick(e, center(l.CanvasBoxes.rect(modesGraphFirst)))
	if e.doc.Source != "map1" {
		t.Fatalf("the first mission's node opened %q", e.doc.Source)
	}

	e, _ = graphEditor(t, modesGraph())
	e.doc.Edits.(*modesWorkspace).dirty = true
	_, l = graphLayout(t, e)
	modesClick(e, center(l.CanvasBoxes.rect(modesGraphFirst+1)))
	if !asked(e) || e.doc.Source != "map0" {
		t.Fatalf("unsaved changes: asked %v map %q", asked(e), e.doc.Source)
	}
	leaveButton(t, e, leaveStay)
	if asked(e) || e.doc.Source != "map0" {
		t.Fatalf("Cancel: asked %v map %q", asked(e), e.doc.Source)
	}
	modesClick(e, center(l.CanvasBoxes.rect(modesGraphFirst+1)))
	leaveButton(t, e, leaveGo)
	if e.doc.Source != "map2" {
		t.Fatalf("Discard opened %q", e.doc.Source)
	}

	held := modesGraph()
	held.Mod = true
	held.Nodes[modesGraphSide].ModMission, held.Nodes[modesGraphSide].AddedMark = 2, true
	e, calls = graphEditor(t, held)
	_, l = graphLayout(t, e)
	modesClick(e, center(l.CanvasBoxes.rect(modesGraphSide)))
	if want := `Workspace package source "2"`; !slices.Contains(calls.lines, want) {
		t.Fatalf("the mod's mission did not open as its row does: %q", calls.lines)
	}
}

// The graph's layout is its data alone: the same page lays the same canvas,
// a node stands at its column and row, no two boxes overlap, and every
// line runs level or upright from its node's box to its target's.
func TestCampaignGraphLayoutComesFromItsData(t *testing.T) {
	e, _ := graphEditor(t, modesGraph())
	page := modesGraph()
	c := e.graphCanvas(page)
	if again := e.graphCanvas(modesGraph()); !reflect.DeepEqual(c, again) {
		t.Fatal("two builds of one page differ")
	}
	sheet := image.Rectangle{Max: c.Sheet}
	for i, b := range c.Boxes {
		if !b.Area.In(sheet) {
			t.Errorf("box %d %v leaves the sheet %v", i, b.Area, c.Sheet)
		}
		for k, other := range c.Boxes[:i] {
			a, o := page.Nodes[i], page.Nodes[k]
			if b.Area.Overlaps(other.Area) {
				t.Errorf("boxes %d and %d overlap", k, i)
			}
			if a.Column == o.Column && b.Area.Min.X != other.Area.Min.X || a.Column > o.Column && b.Area.Min.X < other.Area.Max.X {
				t.Errorf("boxes %d and %d leave their columns: %v %v", k, i, other.Area, b.Area)
			}
			if a.Row > o.Row && b.Area.Min.Y < other.Area.Max.Y {
				t.Errorf("boxes %d and %d leave their rows: %v %v", k, i, other.Area, b.Area)
			}
		}
	}
	touches := func(p image.Point, r image.Rectangle) bool { return p.In(r.Inset(-2)) }
	for k, line := range c.Lines {
		edge := page.Edges[k]
		if len(line.Points) < 2 || !line.Arrow {
			t.Fatalf("line %d has points %v", k, line.Points)
		}
		for i := 1; i < len(line.Points); i++ {
			if a, b := line.Points[i-1], line.Points[i]; a.X != b.X && a.Y != b.Y {
				t.Errorf("line %d turns askew between %v and %v", k, a, b)
			}
		}
		if from, to := line.Points[0], line.Points[len(line.Points)-1]; !touches(from, c.Boxes[edge.From].Area) || !touches(to, c.Boxes[edge.To].Area) {
			t.Errorf("line %d runs %v to %v, its boxes are %v and %v", k, from, to, c.Boxes[edge.From].Area, c.Boxes[edge.To].Area)
		}
		// A line crosses no box but its own two.
		for i := 1; i < len(line.Points); i++ {
			seg := image.Rectangle{Min: line.Points[i-1], Max: line.Points[i]}.Canon()
			seg.Max = seg.Max.Add(image.Pt(1, 1))
			for n, b := range c.Boxes {
				if n != edge.From && n != edge.To && seg.Overlaps(b.Area) {
					t.Errorf("line %d crosses box %d", k, n)
				}
			}
		}
	}
}

// An arrow that holds for some heroes only carries their word, in English
// and in Russian. It leaves its mission's side at a level of its own, runs
// to a lane past the word and ends in its target's top; the page's lines
// hold the word.
func TestCampaignGraphNamesAnArrowsHeroes(t *testing.T) {
	page := CampaignGraphPage{Nodes: []CampaignGraphNode{
		{Kind: GraphMission, Number: 20, Column: 2},
		{Kind: GraphMission, Number: 25, Column: 2, Row: 1},
		{Kind: GraphMission, Number: 26, Column: 2, Row: 2},
		{Kind: GraphTown, Number: 30, Row: 3},
	}, Edges: []CampaignGraphEdge{
		{From: 0, To: 1, Heroes: HeroManMage | HeroManFighter},
		{From: 0, To: 2, Heroes: HeroWomanMage},
		{From: 0, To: 3, Heroes: HeroWomanFighter, Marked: true},
		{From: 1, To: 3},
		{From: 2, To: 3},
	}}
	e, _ := graphEditor(t, page)
	s, _ := graphLayout(t, e)
	c, body := s.Canvas, e.bodyFont()
	advance := func(word string) int {
		if body == nil {
			return 7 * len(word)
		}
		return body.Advance(word)
	}
	side := c.Boxes[0].Area
	var levels []int
	for k, want := range []string{"Man", "Woman mage", "Woman fighter", "", ""} {
		line := c.Lines[k]
		if line.Label != want || line.Over != page.Edges[k].Marked {
			t.Errorf("line %d: label %q, want %q; over %t", k, line.Label, want, line.Over)
		}
		if want == "" {
			continue
		}
		from, lane, to := line.Points[0], line.Points[1], line.Points[len(line.Points)-1]
		if from.X != side.Max.X || from.Y < side.Min.Y || from.Y >= side.Max.Y {
			t.Errorf("line %d leaves at %v, its mission's box is %v", k, from, side)
		}
		if lane.Y != from.Y || lane.X-from.X < advance(want)+2*canvasBoxPad {
			t.Errorf("line %d runs %v to %v: no room for %q", k, from, lane, want)
		}
		if target := c.Boxes[page.Edges[k].To].Area; to.Y != target.Min.Y-1 || to.X < target.Min.X || to.X >= target.Max.X {
			t.Errorf("line %d ends at %v, its target's box is %v", k, to, target)
		}
		levels = append(levels, from.Y)
	}
	if h := fontHeight(body, 15); len(levels) != 3 || levels[1]-levels[0] < h || levels[2]-levels[1] < h {
		t.Errorf("the labelled lines leave at %v, less than a text line apart", levels)
	}
	lines := e.PopupLines()
	for _, want := range []string{"Man", "Woman mage", "Woman fighter"} {
		if !slices.Contains(lines, want) {
			t.Errorf("the graph lacks the label %q in\n%s", want, strings.Join(lines, "\n"))
		}
	}
	if got := e.graphHeroes(HeroWomanMage | HeroManFighter); got != "Woman mage, Man fighter" {
		t.Errorf("two heroes of no one sex or class read %q", got)
	}
	e.SetWords(words.For("ru"))
	lines = e.PopupLines()
	for _, want := range []string{"Мужчина", "Женщина-маг", "Женщина-воин"} {
		if !slices.Contains(lines, want) {
			t.Errorf("the Russian graph lacks the label %q in\n%s", want, strings.Join(lines, "\n"))
		}
	}
}

// A canvas draws a line set Over over the others, whatever their order, and
// a line's label on cleared ground over the line, in the line's ink.
func TestPopupCanvasDrawsMarkedLinesOverAndLabels(t *testing.T) {
	f := panelFont()
	c := &popupCanvas{Sheet: image.Pt(200, 120), Height: 120, Lines: []canvasLine{
		{Points: []image.Point{{20, 30}, {150, 30}}, Ink: graphFlowInk, Over: true},
		{Points: []image.Point{{20, 30}, {150, 30}}, Ink: editorPaper},
		{Points: []image.Point{{20, 80}, {150, 80}}, Ink: editorGold, Label: "Mage"},
	}}
	s := popupSpec{Title: "Any window", Width: 300, At: image.Pt(20, 20), Canvas: c, Buttons: []string{"Close"}}
	l := layoutPopup(s, f, f)
	pic := image.NewRGBA(image.Rect(0, 0, 400, 300))
	drawPopup(pic, f, f, nil, s, l, editorPopupInk, image.Pt(-1, -1), false)
	at := func(x, y int) color.RGBA {
		p := l.Canvas.Min.Sub(l.CanvasAt).Add(image.Pt(x, y))
		return pic.RGBAAt(p.X, p.Y)
	}
	if got := at(100, 30); got != graphFlowInk {
		t.Errorf("the shared line is %v, want the ink of the line set over, %v", got, graphFlowInk)
	}
	if got := at(140, 80); got != editorGold {
		t.Errorf("the labelled line past its label is %v, want %v", got, editorGold)
	}
	h := fontHeight(f, 15)
	label := image.Rect(20+canvasBoxPad, 80-h/2, 20+canvasBoxPad+f.Advance("Mage"), 80-h/2+h)
	// The word is written off the line's level, and the line is cut under it.
	written, uncut := 0, 0
	for y := label.Min.Y; y < label.Max.Y; y++ {
		for x := label.Min.X; x < label.Max.X; x++ {
			switch got := at(x, y); {
			case y != 80 && got != editorPopupInk.TitleGround:
				written++
			case y == 80 && got == editorGold:
				uncut++
			}
		}
	}
	if written == 0 || uncut == label.Dx() {
		t.Errorf("the label's place %v holds %d written pixels; %d of %d on the line's level are the line's", label, written, uncut, label.Dx())
	}
	if got := c.labels(); !slices.Equal(got, []string{"Mage"}) {
		t.Errorf("labels %q", got)
	}
}
