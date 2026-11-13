package ui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// GraphNodeKind is what a node of the campaign graph stands for.
type GraphNodeKind uint8

const (
	// GraphStart is the hero generator a new campaign begins at.
	GraphStart GraphNodeKind = iota
	// GraphMission is a mission; Number is its number.
	GraphMission
	// GraphTown is the town of a chapter; Number is the chapter's mission.
	GraphTown
	// GraphHandout is a building's offer of a mission; Place is the
	// building and Number the tavern's visitor.
	GraphHandout
)

// GraphPlace is the building of a hand-out.
type GraphPlace uint8

const (
	GraphInn GraphPlace = iota
	GraphShop
	GraphSchool
)

// CampaignGraphNode is one node of the campaign graph. Column and Row place
// it on the graph's grid. The marks say what the open mod changes: MapMark
// a replaced map, FlowMark an overridden successor, AddedMark a mission the
// mod adds, PartyMark a guest party. Source is the install map a mission
// node opens; ModMission, when not zero, is the row of the open mod's
// mission list that holds the mission's map instead.
type CampaignGraphNode struct {
	Kind        GraphNodeKind
	Place       GraphPlace
	Number      int
	Title       string
	Column, Row int
	MapMark     bool
	FlowMark    bool
	AddedMark   bool
	PartyMark   bool
	// Empty marks a hand-out with nothing to give.
	Empty      bool
	Source     string
	ModMission int
	// Pick, when set, is what the host selects a town or a hand-out by
	// while missions are edited; Selected marks the one it holds selected.
	Pick     string
	Selected bool
}

// CampaignGraphEdge is an arrow from node From to node To, by their indices.
// Marked says the open mod set it. Heroes, when not zero, is the starting
// heroes the arrow holds for; zero is every hero that reaches From.
type CampaignGraphEdge struct {
	From, To int
	Marked   bool
	Heroes   HeroAudience
}

// CampaignGraphPage is the campaign as a graph. Mod says a mod is open and
// the graph holds its changes; Refused that the game refuses the open mod's
// missions, and Unbuilt that the mod's campaign was not built for another
// reason, so the graph is the install's; Unread that this install's campaign
// is not one the graph reads, so it has no nodes.
type CampaignGraphPage struct {
	Nodes   []CampaignGraphNode
	Edges   []CampaignGraphEdge
	Mod     bool
	Refused bool
	Unbuilt bool
	Unread  bool
}

// ModCampaignGraph is the optional host seam of W's campaign graph page.
type ModCampaignGraph interface {
	CampaignGraph() CampaignGraphPage
}

const (
	// graphPad is the sheet's margin, graphGapX and graphGapY the room
	// between two columns and two rows of boxes.
	graphPad  = 14
	graphGapX = 22
	graphGapY = 14
	// graphNarrow and graphWide bound a column's width.
	graphNarrow = 56
	graphWide   = 230
	// graphLane is the room one upright lane takes right of the columns.
	graphLane = 6
)

// The inks of the four mod marks, on a node and in the legend.
var (
	graphMapInk   = editorCondition
	graphFlowInk  = editorWarning
	graphAddedInk = editorLight
	graphPartyInk = editorGold
)

// modGraph is the campaign graph the host draws, and whether W has the page.
func (e *MapEditor) modGraph() (CampaignGraphPage, bool) {
	host, ok := e.modEdits().(ModCampaignGraph)
	if !ok {
		return CampaignGraphPage{}, false
	}
	p := host.CampaignGraph()
	return p, len(p.Nodes) > 0 || p.Unread
}

// modGraphRows lays W's graph page: a caption, the legend of the mod marks
// when a mod is open, and the graph as the popup's canvas.
func (e *MapEditor) modGraphRows(s *popupSpec, links *[]int, page CampaignGraphPage) {
	o := &e.obj
	o.rowBlock, o.usesShown, o.viewOffer, o.cell = nil, false, false, -1
	o.first, o.rowField = len(s.Rows), map[int]int{}
	s.Columns = 1
	text := func(words string) {
		for _, part := range editorTextLines(e.bodyFont(), words, s.Width-40) {
			s.Rows, *links = append(s.Rows, popupRow{Kind: popupText, Label: part}), append(*links, 0)
		}
	}
	switch {
	case page.Unread:
		text(e.word("editor.graph.unread"))
		return
	case page.Refused:
		text(e.word("editor.graph.refused"))
	case page.Unbuilt:
		text(e.word("editor.graph.unbuilt"))
	case page.Mod:
		text(e.word("editor.graph.mod"))
	default:
		text(e.word("editor.graph.install"))
		text(e.word("editor.graph.readonly"))
	}
	if page.Mod {
		for _, m := range e.graphMarks(CampaignGraphNode{MapMark: true, FlowMark: true, AddedMark: true, PartyMark: true}) {
			row := popupRow{Kind: popupText, Label: m.Text, Value: e.word("editor.graph.legend." + m.key), Ink: m.Ink}
			s.Rows, *links = append(s.Rows, row), append(*links, 0)
		}
		if !page.Refused && !page.Unbuilt {
			if e.graphEdit {
				text(e.word("editor.graph.select"))
				if e.graphMission > 0 && !e.graphPicked {
					text(fmt.Sprintf(e.plainWord("editor.graph.mission"), e.graphMission))
				}
				if e.graphLink {
					text(e.word("editor.graph.target"))
				}
			}
			e.objectFieldRows(s, links)
		}
	}
	// The graph grows down to the status strip and scrolls the rest.
	s.Room = e.popupRoom()
	s.Canvas = e.graphCanvas(page)
}

// graphMark is a mod mark of a node: its word, its ink and its word-book key.
type graphMark struct {
	canvasMark
	key string
}

// graphMarks is the mod marks node n carries, in the legend's order.
func (e *MapEditor) graphMarks(n CampaignGraphNode) []graphMark {
	var out []graphMark
	add := func(on bool, key string, ink color.RGBA) {
		if on {
			out = append(out, graphMark{canvasMark{Text: e.word("editor.graph.mark." + key), Ink: ink}, key})
		}
	}
	add(n.MapMark, "map", graphMapInk)
	add(n.FlowMark, "flow", graphFlowInk)
	add(n.AddedMark, "added", graphAddedInk)
	add(n.PartyMark, "party", graphPartyInk)
	return out
}

// graphHeroes is the words of the starting heroes h: the one word of a sex,
// a class or one hero, else each hero's own.
func (e *MapEditor) graphHeroes(h HeroAudience) string {
	if word := HeroWord(h); word != "" {
		return e.word(word)
	}
	var parts []string
	for _, w := range heroWords {
		if one := w.Heroes&(w.Heroes-1) == 0; one && h&w.Heroes != 0 {
			parts = append(parts, e.word(w.Word))
		}
	}
	return strings.Join(parts, ", ")
}

// graphBox is node n's box without its place: its text, marks and tip. A
// mission with a map to open takes a press; every other node is a frame.
func (e *MapEditor) graphBox(n CampaignGraphNode) canvasBox {
	number := func(format string) string { return e.editorHostText(fmt.Sprintf(format, n.Number)) }
	b := canvasBox{Inert: true, Fill: editorPanel, Edge: editorPaper}
	switch n.Kind {
	case GraphStart:
		b.Lines, b.Tip, b.Edge = []string{e.word("editor.graph.start")}, e.word("editor.graph.start.tip"), editorLight
	case GraphTown:
		b.Lines, b.Tip, b.Edge = []string{number(e.plainWord("editor.graph.town"))}, number(e.plainWord("editor.graph.town.tip")), editorGold
	case GraphHandout:
		b.Tip = e.word("editor.graph.handout.tip")
		switch {
		case n.Place == GraphShop:
			b.Lines = []string{e.word("editor.graph.shop")}
		case n.Place == GraphSchool:
			b.Lines = []string{e.word("editor.graph.school")}
		case n.Empty:
			b.Lines, b.Tip = []string{number(e.plainWord("editor.graph.inn.none"))}, e.word("editor.graph.handout.none")
		default:
			b.Lines = []string{number(e.plainWord("editor.graph.inn"))}
		}
	case GraphMission:
		title := e.editorHostText(n.Title)
		b.Lines = []string{number(e.plainWord("editor.graph.mission")), title}
		for _, m := range e.graphMarks(n) {
			b.Marks = append(b.Marks, m.canvasMark)
		}
		b.Tip = b.Lines[0]
		if title != "" {
			b.Tip += ": " + title
		}
		if b.Inert = n.Source == "" && n.ModMission == 0; b.Inert {
			b.Tip += ". " + e.word("editor.graph.nomap")
			b.Fill = editorInk
		}
	}
	return b
}

// graphCanvas is the campaign graph of page as a canvas: one box a node, at
// its column and row, each column as wide as its longest text; one line an
// edge. A line runs straight to the node on its right or under it, down a
// trunk to a node of the next column, and otherwise up or down a lane right
// of the columns into its target's top. Lines to one target share a lane.
// A line that holds for some heroes only takes a lane, leaves its node at a
// level of its own and carries the heroes' words there, left of the lanes.
func (e *MapEditor) graphCanvas(page CampaignGraphPage) *popupCanvas {
	if len(page.Nodes) == 0 {
		return nil
	}
	body := e.bodyFont()
	advance := func(s string) int {
		if body == nil {
			return 7 * len(s)
		}
		return body.Advance(s)
	}
	line := fontHeight(body, 15)
	tall, low := 2*line+2*canvasBoxPad, line+2*canvasBoxPad
	pitch := tall + graphGapY
	c := &popupCanvas{At: e.graphAt, Boxes: make([]canvasBox, len(page.Nodes))}
	cols, rows := 0, 0
	for _, n := range page.Nodes {
		cols, rows = max(cols, n.Column+1), max(rows, n.Row+1)
	}
	widths, marks := make([]int, cols), make([]int, len(page.Nodes))
	for i, n := range page.Nodes {
		b := e.graphBox(n)
		if e.graphEdit && page.Mod && !page.Refused && !page.Unbuilt && (n.Kind == GraphMission || n.Pick != "") {
			b.Inert = false
			if n.Kind == GraphMission && n.Number == e.graphMission || n.Kind != GraphMission && e.graphPicked && n.Selected {
				b.Edge = editorWarning
			}
		}
		for _, m := range b.Marks {
			marks[i] += advance(" ") + advance(m.Text)
		}
		w := 0
		for k, s := range b.Lines {
			if k == 0 {
				w = max(w, advance(s)+marks[i])
			} else {
				w = max(w, advance(s))
			}
		}
		widths[n.Column] = max(widths[n.Column], min(graphWide, w+2*canvasBoxPad+2))
		c.Boxes[i] = b
	}
	left, x := make([]int, cols), graphPad
	for k := range widths {
		widths[k] = max(widths[k], graphNarrow)
		left[k] = x
		x += widths[k] + graphGapX
	}
	// said is each line's heroes' words, told the lines of each node that
	// carry any, and gutter the room the longest takes left of the lanes.
	said, told, gutter := make([]string, len(page.Edges)), map[int]int{}, 0
	for k, ed := range page.Edges {
		if ed.Heroes != 0 {
			said[k] = e.graphHeroes(ed.Heroes)
			told[ed.From]++
			gutter = max(gutter, advance(said[k])+2*canvasBoxPad)
		}
	}
	rail := x - graphGapX + graphLane + gutter
	taken := map[image.Point]bool{}
	for i, n := range page.Nodes {
		b := &c.Boxes[i]
		h := low
		if len(b.Lines) > 1 {
			h = tall
		}
		y := graphPad + n.Row*pitch + (tall-h)/2
		b.Area = image.Rect(left[n.Column], y, left[n.Column]+widths[n.Column], y+h)
		for k := range b.Lines {
			room := widths[n.Column] - 2*canvasBoxPad
			if k == 0 {
				room -= marks[i]
			}
			b.Lines[k] = editorClip(body, b.Lines[k], room)
		}
		taken[image.Pt(n.Column, n.Row)] = true
	}
	mid := func(row int) int { return graphPad + row*pitch + tall/2 }
	centre := func(i int) int { return (c.Boxes[i].Area.Min.X + c.Boxes[i].Area.Max.X) / 2 }
	// above is the level of a line's run over node i, in the gap over its
	// row; the nodes of one row take runs at different levels.
	above := func(i int) int { return graphPad + page.Nodes[i].Row*pitch - 4 - 3*(page.Nodes[i].Column%3) }
	clear := func(from, to CampaignGraphNode) bool {
		for k := from.Column + 1; k < to.Column; k++ {
			if taken[image.Pt(k, from.Row)] {
				return false
			}
		}
		return true
	}
	c.Lines = make([]canvasLine, len(page.Edges))
	lanes := map[int][2]int{}
	var railed []int
	leaves, nth := make([]int, len(page.Edges)), map[int]int{}
	for k, ed := range page.Edges {
		if ed.From < 0 || ed.From >= len(page.Nodes) || ed.To < 0 || ed.To >= len(page.Nodes) {
			continue
		}
		from, to := page.Nodes[ed.From], page.Nodes[ed.To]
		a, b := c.Boxes[ed.From].Area, c.Boxes[ed.To].Area
		l := canvasLine{Ink: editorPaper, Arrow: true, Over: ed.Marked, Label: said[k]}
		if ed.Marked {
			l.Ink = graphFlowInk
		}
		plain := l.Label == ""
		leaves[k] = mid(from.Row)
		if n := told[ed.From]; !plain && n > 1 {
			// The labelled lines of one node leave its side a text line
			// apart, or closer where its side is too short for that.
			step := min(line, (a.Dy()-2)/(n-1))
			leaves[k] += (2*nth[ed.From] - n + 1) * step / 2
			nth[ed.From]++
		}
		switch {
		case plain && from.Row == to.Row && to.Column > from.Column && clear(from, to):
			l.Points = []image.Point{{a.Max.X, mid(from.Row)}, {b.Min.X - 1, mid(to.Row)}}
		case plain && from.Column == to.Column && to.Row == from.Row+1:
			l.Points = []image.Point{{centre(ed.From), a.Max.Y}, {centre(ed.To), b.Min.Y - 1}}
		case plain && to.Column == from.Column+1 && to.Row > from.Row:
			trunk := a.Max.X + graphGapX/2
			l.Points = []image.Point{{a.Max.X, mid(from.Row)}, {trunk, mid(from.Row)}, {trunk, mid(to.Row)}, {b.Min.X - 1, mid(to.Row)}}
		default:
			span, ok := lanes[ed.To]
			if !ok {
				span = [2]int{above(ed.To), above(ed.To)}
			}
			lanes[ed.To] = [2]int{min(span[0], leaves[k]), max(span[1], leaves[k])}
			railed = append(railed, k)
		}
		c.Lines[k] = l
	}
	// The targets take lanes in the order their spans begin, each the
	// first lane no earlier span reaches.
	targets := make([]int, 0, len(lanes))
	for to := range lanes {
		targets = append(targets, to)
	}
	sort.Slice(targets, func(i, j int) bool {
		a, b := lanes[targets[i]], lanes[targets[j]]
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		return targets[i] < targets[j]
	})
	var ends []int
	lane := map[int]int{}
	for _, to := range targets {
		at := slices.IndexFunc(ends, func(end int) bool { return end+2 < lanes[to][0] })
		if at < 0 {
			at, ends = len(ends), append(ends, 0)
		}
		ends[at], lane[to] = lanes[to][1], at
	}
	for _, k := range railed {
		ed := page.Edges[k]
		x, y, from := rail+lane[ed.To]*graphLane, above(ed.To), leaves[k]
		c.Lines[k].Points = []image.Point{{c.Boxes[ed.From].Area.Max.X, from}, {x, from}, {x, y}, {centre(ed.To), y}, {centre(ed.To), c.Boxes[ed.To].Area.Min.Y - 1}}
	}
	c.Sheet = image.Pt(rail+len(ends)*graphLane+graphPad-graphLane, graphPad+rows*pitch-graphGapY+graphPad)
	return c
}

// graphPan moves the graph's sheet by d under W's view, held inside it.
func (e *MapEditor) graphPan(d image.Point) {
	s, _, ok := e.objectSpec()
	if !ok || s.Canvas == nil {
		return
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	room, at := s.Canvas.Sheet.Sub(l.Canvas.Size()), l.CanvasAt.Add(d)
	e.graphAt = image.Pt(max(0, min(at.X, room.X)), max(0, min(at.Y, room.Y)))
}

// graphShown reports W open on its graph page.
func (e *MapEditor) graphShown() bool {
	return e.modWindow && e.modPage == modPageGraph && !e.prompt.open()
}

// graphKey moves the graph by a key: the arrows a row or two columns' gap,
// the page keys a view less a row, Home and End to the top and the bottom.
func (e *MapEditor) graphKey(key string) bool {
	if key == "escape" && e.graphLink {
		e.graphLink = false
		return true
	}
	if e.graphEdit && e.obj.focus >= 0 && e.obj.focus < len(e.obj.fields) {
		return false
	}
	s, _, ok := e.objectSpec()
	if !ok || s.Canvas == nil {
		return false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	step := max(1, l.Pitch)
	page := max(step, l.Canvas.Dy()-step)
	switch key {
	case "up":
		e.graphPan(image.Pt(0, -step))
	case "down":
		e.graphPan(image.Pt(0, step))
	case "left":
		e.graphPan(image.Pt(-2*graphGapX, 0))
	case "right":
		e.graphPan(image.Pt(2*graphGapX, 0))
	case "pageup":
		e.graphPan(image.Pt(0, -page))
	case "pagedown":
		e.graphPan(image.Pt(0, page))
	case "home":
		e.graphPan(image.Pt(0, -s.Canvas.Sheet.Y))
	case "end":
		e.graphPan(image.Pt(0, s.Canvas.Sheet.Y))
	default:
		return false
	}
	return true
}

// graphWheel moves the graph by the wheel when W shows it.
func (e *MapEditor) graphWheel(wheel wheelSource) bool {
	if !e.graphShown() {
		return false
	}
	rows := editorWheelRows(&e.objectWheel, wheel)
	e.graphPan(image.Pt(0, rows*(fontHeight(e.bodyFont(), 15)+popupRowExtra)))
	return true
}

// graphPress is a press on the graph: on a mission's box it opens the
// mission's map, on the ground it takes hold of the sheet.
func (e *MapEditor) graphPress(part popupPart, n int, p image.Point) {
	if part == popupCanvasGround {
		e.graphHeld, e.graphFrom = true, p
		return
	}
	page, _ := e.modGraph()
	if n < 0 || n >= len(page.Nodes) {
		return
	}
	if e.graphEdit && page.Mod && !page.Refused && !page.Unbuilt && page.Nodes[n].Kind != GraphMission && page.Nodes[n].Pick != "" && !e.graphLink {
		if err := e.applyModField(); err != nil {
			return
		}
		if _, err := e.commandSetMapField(mapField{Scope: graphPickScope, Key: "select", Value: page.Nodes[n].Pick}); err != nil {
			e.modStatus = err.Error()
			return
		}
		e.graphMission, e.graphPicked = 0, true
		e.dropObject()
		e.loadObject()
		e.obj.focus = len(e.obj.fields)
		return
	}
	if e.graphEdit && page.Mod && !page.Refused && !page.Unbuilt && page.Nodes[n].Kind == GraphMission {
		if err := e.applyModField(); err != nil {
			return
		}
		number := page.Nodes[n].Number
		e.graphPicked = false
		if e.graphLink {
			if err := e.modAction("AutoGetMission", strconv.Itoa(number)); err != nil {
				return
			}
			e.graphLink = false
			return
		}
		e.graphMission = number
		e.dropObject()
		e.loadObject()
		e.obj.focus = len(e.obj.fields)
		return
	}
	e.openGraphNode(page.Nodes[n])
}

// graphPickScope is the workspace scope of the town or hand-out the graph
// has selected; its "select" key names a node's Pick and is no edit.
const graphPickScope = "handout"

func (e *MapEditor) modFieldScope() string {
	if e.modPage == modPageGraph {
		if e.graphEdit && e.graphPicked {
			return graphPickScope
		}
		if e.graphEdit {
			return "campaign." + strconv.Itoa(e.graphMission)
		}
		return "campaign.0"
	}
	return "package"
}

func (e *MapEditor) graphAction(key string) error {
	page, _ := e.modGraph()
	if !page.Mod || page.Unread || page.Refused || page.Unbuilt {
		return nil
	}
	switch key {
	case "graph.edit":
		if err := e.applyModField(); err != nil {
			return err
		}
		e.graphEdit, e.graphLink = !e.graphEdit, false
	case "graph.link":
		e.graphLink = true
	case "graph.unlink":
		return e.modAction("AutoGetMission", "-1")
	}
	e.dropObject()
	e.loadObject()
	e.obj.focus = len(e.obj.fields)
	return nil
}

// graphDrag moves a held sheet with the pointer and lets go on release.
func (e *MapEditor) graphDrag(p image.Point, down bool) {
	if !e.graphHeld {
		return
	}
	if !down || !e.graphShown() {
		e.graphHeld = false
		return
	}
	e.graphPan(e.graphFrom.Sub(p))
	e.graphFrom = p
}

// openGraphNode opens a mission node's map by the route its owner already
// has: a mission of the open mod as its row's Open button in W does, an
// install mission as the map list does. Both ask about unsaved changes.
func (e *MapEditor) openGraphNode(n CampaignGraphNode) {
	switch {
	case n.Kind != GraphMission:
	case n.ModMission > 0:
		_ = e.modAction("source", strconv.Itoa(n.ModMission))
	case n.Source != "":
		_ = e.Open(n.Source)
	}
}
