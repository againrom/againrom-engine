package game

import (
	"fmt"
	"image"
)

// worldMapNodeGraph is the route-topology graph read from `PathMap.bmp`
// (`TOWN-116`, High): the mask is 640x480 indexed; index 1 is corridor,
// index 2 is a graph node, every other sampled value is impassable. The
// shipped mask holds 81 node pixels, and every registry `MapPoint` lands
// exactly on one of them.
type worldMapNodeGraph struct {
	nodes []image.Point
	index map[image.Point]int
	edges [][]worldMapEdge
}

// worldMapEdge is one directly-walkable corridor between two nodes: no other
// node pixel lies on it. Points holds the corridor pixels strictly between
// the two node pixels, in travel order away from the edge's own source.
type worldMapEdge struct {
	to     int
	points []image.Point
}

var worldMapDirs = [8]image.Point{
	{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
	{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1},
}

// buildWorldMapNodeGraph builds the graph unconditionally from a decoded
// mask (`TOWN-117`, High: the original's own construction is unconditional
// at world-map enter). It scans every pixel for index 2 to collect nodes,
// then walks eight-neighbour index-1 corridor pixels outward from each node
// until a different node pixel is reached, recording that corridor as one
// directed edge (`TOWN-116`/`TOWN-117`, High). A nil or nodeless mask yields
// a nil graph.
func buildWorldMapNodeGraph(mask *image.Paletted) *worldMapNodeGraph {
	if mask == nil {
		return nil
	}
	b := mask.Bounds()
	g := &worldMapNodeGraph{index: make(map[image.Point]int)}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if mask.ColorIndexAt(x, y) == 2 {
				p := image.Pt(x, y)
				g.index[p] = len(g.nodes)
				g.nodes = append(g.nodes, p)
			}
		}
	}
	if len(g.nodes) == 0 {
		return nil
	}
	g.edges = make([][]worldMapEdge, len(g.nodes))
	passable := func(p image.Point) (uint8, bool) {
		if !p.In(b) {
			return 0, false
		}
		return mask.ColorIndexAt(p.X, p.Y), true
	}
	for id, src := range g.nodes {
		visited := map[image.Point]bool{src: true}
		var walk func(p image.Point, path []image.Point)
		walk = func(p image.Point, path []image.Point) {
			for _, d := range worldMapDirs {
				q := p.Add(d)
				idx, ok := passable(q)
				if !ok {
					continue
				}
				switch idx {
				case 1:
					if visited[q] {
						continue
					}
					visited[q] = true
					next := make([]image.Point, len(path)+1)
					copy(next, path)
					next[len(path)] = q
					walk(q, next)
				case 2:
					if q == src {
						continue
					}
					g.edges[id] = append(g.edges[id], worldMapEdge{to: g.index[q], points: path})
				}
			}
		}
		walk(src, nil)
	}
	return g
}

// Route returns the pixel-by-pixel path from start to end: the node chain of
// least accumulated segment length, expanded by concatenating each chosen
// edge's corridor points between the two node pixels (`TOWN-119`, High). ok
// is false when either point is not itself a graph node, or when no chain
// connects the two nodes — both DIV-129, authored: no claim describes this
// case, and it does not arise for any shipped registry MapPoint, all of
// which land on a node (`TOWN-116`).
func (g *worldMapNodeGraph) Route(start, end image.Point) (path []image.Point, ok bool) {
	if g == nil {
		return nil, false
	}
	s, sok := g.index[start]
	e, eok := g.index[end]
	if !sok || !eok {
		return nil, false
	}
	if s == e {
		return []image.Point{g.nodes[s]}, true
	}
	chain, ok := g.chain(s, e, shippedSegmentWeight)
	if !ok {
		return nil, false
	}
	out := []image.Point{g.nodes[chain[0]]}
	for k := 1; k < len(chain); k++ {
		a, b := chain[k-1], chain[k]
		for _, edge := range g.edges[a] {
			if edge.to == b {
				out = append(out, edge.points...)
				break
			}
		}
		out = append(out, g.nodes[b])
	}
	return out, true
}

// worldMapGraphDigest reports the shipped route graph's own corpus counts:
// mask pixels by class, nodes, directed edges and whether any (source,
// destination) pair repeats, connectivity, and how many registry MapPoints
// land on a node. The numbers a story's closure quotes for this graph come
// from here, so they are reproducible by a committed command
// (`cmd/worldmapcheck`) rather than by a probe written once and deleted.
func worldMapGraphDigest(mask *image.Paletted, points []image.Point) string {
	if mask == nil {
		return "world map graph: no path mask"
	}
	corridor, node, impassable := 0, 0, 0
	b := mask.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			switch mask.ColorIndexAt(x, y) {
			case 1:
				corridor++
			case 2:
				node++
			default:
				impassable++
			}
		}
	}
	g := buildWorldMapNodeGraph(mask)
	if g == nil {
		return fmt.Sprintf("world map graph: %d corridor, %d node, %d impassable pixel(s); no graph", corridor, node, impassable)
	}
	edges, collisions := 0, 0
	for _, list := range g.edges {
		seen := make(map[int]bool, len(list))
		for _, e := range list {
			edges++
			if seen[e.to] {
				collisions++
			}
			seen[e.to] = true
		}
	}
	// Breadth-first reachability from node 0, over the directed edges.
	reached := make([]bool, len(g.nodes))
	reached[0] = true
	queue := []int{0}
	count := 1
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, e := range g.edges[u] {
			if !reached[e.to] {
				reached[e.to] = true
				count++
				queue = append(queue, e.to)
			}
		}
	}
	onNode := 0
	for _, p := range points {
		if _, ok := g.index[p]; ok {
			onNode++
		}
	}
	return fmt.Sprintf("world map graph: %d corridor, %d node, %d impassable pixel(s); %d node(s), %d directed edge(s), %d duplicate edge(s); %d of %d node(s) reachable from node 0; %d of %d MapPoint(s) on a node",
		corridor, node, impassable, len(g.nodes), edges, collisions, count, len(g.nodes), onNode, len(points))
}

// shippedSegmentWeight is the per-segment cost this build accumulates: the
// corridor's own pixel count plus one for the destination node pixel.
func shippedSegmentWeight(edgePoints int) int { return edgePoints + 1 }

// hopWeight charges one per segment traversed regardless of its length.
func hopWeight(int) int { return 1 }

// chain returns the node chain from s to e of least accumulated weight, and
// whether one exists. Route and worldMapCriterionDigest share it, so the
// measurement measures the production search rather than a copy of it: an
// equivalent implementation is a different implementation, and a digest that
// reproduces a search it does not use witnesses nothing about the build.
func (g *worldMapNodeGraph) chain(s, e int, weight func(edgePoints int) int) ([]int, bool) {
	const inf = int(^uint(0) >> 1)
	dist := make([]int, len(g.nodes))
	from := make([]int, len(g.nodes))
	visited := make([]bool, len(g.nodes))
	for i := range dist {
		dist[i] = inf
		from[i] = -1
	}
	dist[s] = 0
	for {
		u, best := -1, inf
		for i, d := range dist {
			if !visited[i] && d < best {
				u, best = i, d
			}
		}
		if u == -1 || u == e {
			break
		}
		visited[u] = true
		for _, edge := range g.edges[u] {
			alt := dist[u] + weight(len(edge.points))
			if alt < dist[edge.to] {
				dist[edge.to] = alt
				from[edge.to] = u
			}
		}
	}
	if dist[e] == inf {
		return nil, false
	}
	var chain []int
	for n := e; n != -1; n = from[n] {
		chain = append(chain, n)
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, true
}

// worldMapCriterionDigest reports how many shipped MapPoint pairs this build's
// own accumulated-segment-length search routes differently from a
// one-per-segment search, over the graph this build actually builds.
//
// It exists because the figure was quoted in a divergence row and in a story
// closure with no committed command that printed it, and was then
// contradicted by a research lane's independently derived figure over its
// own reproduction of the original's segmentation. Print it; do not quote a
// remembered value.
func worldMapCriterionDigest(mask *image.Paletted, points []image.Point) string {
	g := buildWorldMapNodeGraph(mask)
	if g == nil {
		return "world map criterion: no graph"
	}
	var idx []int
	for _, p := range points {
		if i, ok := g.index[p]; ok {
			idx = append(idx, i)
		}
	}
	same := func(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	ordered, pairs, unordered := 0, 0, map[[2]int]bool{}
	for i := range idx {
		for j := range idx {
			if i == j {
				continue
			}
			pairs++
			byLength, lok := g.chain(idx[i], idx[j], shippedSegmentWeight)
			byHops, hok := g.chain(idx[i], idx[j], hopWeight)
			if !lok || !hok {
				continue
			}
			if !same(byLength, byHops) {
				ordered++
				k := [2]int{idx[i], idx[j]}
				if k[0] > k[1] {
					k[0], k[1] = k[1], k[0]
				}
				unordered[k] = true
			}
		}
	}
	return fmt.Sprintf("world map criterion: %d MapPoint(s) on nodes, %d ordered pair(s); accumulated segment length and one-per-segment select different chains for %d ordered pair(s), %d unordered",
		len(idx), pairs, ordered, len(unordered))
}
