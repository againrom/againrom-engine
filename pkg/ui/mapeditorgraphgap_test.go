package ui

import "testing"

// Floors for the room between graph columns and rows.
func TestGraphGapsGiveRoomBetweenLineRuns(t *testing.T) {
	const minGapX, minGapY = 34, 28
	if graphGapX < minGapX {
		t.Fatalf("graphGapX = %d, want at least %d", graphGapX, minGapX)
	}
	if graphGapY < minGapY {
		t.Fatalf("graphGapY = %d, want at least %d", graphGapY, minGapY)
	}
	if top := graphRunBase + graphRunStep*(graphRuns-1); top >= graphGapY {
		t.Fatalf("the highest run stands %d over its row, in a gap of %d", top, graphGapY)
	}
}

// graphHandoutOrderPage is a town whose side hand-out stands over its main
// one, both handed to the town's row, with the side mission's way back.
func graphHandoutOrderPage() CampaignGraphPage {
	var p CampaignGraphPage
	add := func(n CampaignGraphNode) int {
		p.Nodes = append(p.Nodes, n)
		return len(p.Nodes) - 1
	}
	town := add(CampaignGraphNode{Kind: GraphTown, Number: 30, Column: 0})
	shop := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphShop, Column: 1})
	inn := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphInn, Number: 22, Column: 1, Row: 1})
	none := add(CampaignGraphNode{Kind: GraphHandout, Place: GraphInn, Number: 25, Column: 1, Row: 2, Empty: true})
	main := add(CampaignGraphNode{Kind: GraphMission, Number: 30, Title: "Main", Column: 2})
	side := add(CampaignGraphNode{Kind: GraphMission, Number: 31, Title: "Side", Column: 3})
	for _, e := range [][2]int{{town, shop}, {town, inn}, {town, none}, {shop, side}, {inn, main}, {side, town}} {
		p.Edges = append(p.Edges, CampaignGraphEdge{From: e[0], To: e[1]})
	}
	return p
}

// Lines of a town's hand-outs that run up to the town's row cross no other
// line, and two parallel runs of different lines stand at least a run step
// apart horizontally and a lane apart upright.
func TestGraphHandoutLinesCrossNothingAndStandApart(t *testing.T) {
	page := graphHandoutOrderPage()
	e, _ := graphEditor(t, page)
	c := e.graphCanvas(page)
	if found := crossingsOf(c.Lines); len(found) != 0 {
		t.Fatalf("%d crossings, first at %v", len(found), found[0].at)
	}
	for j, one := range c.Lines {
		for k := j + 1; k < len(c.Lines); k++ {
			two := c.Lines[k]
			for i := 1; i < len(one.Points); i++ {
				a, b := one.Points[i-1], one.Points[i]
				for n := 1; n < len(two.Points); n++ {
					p, q := two.Points[n-1], two.Points[n]
					switch {
					case a.Y == b.Y && p.Y == q.Y && a.X != b.X && p.X != q.X:
						d := max(a.Y, p.Y) - min(a.Y, p.Y)
						if d != 0 && d < graphRunStep && min(max(a.X, b.X), max(p.X, q.X)) > max(min(a.X, b.X), min(p.X, q.X)) {
							t.Errorf("runs of lines %d and %d are %d apart at y %d and %d", j, k, d, a.Y, p.Y)
						}
					case a.X == b.X && p.X == q.X && a.Y != b.Y && p.Y != q.Y:
						d := max(a.X, p.X) - min(a.X, p.X)
						if d != 0 && d < graphLane && min(max(a.Y, b.Y), max(p.Y, q.Y)) > max(min(a.Y, b.Y), min(p.Y, q.Y)) {
							t.Errorf("lanes of lines %d and %d are %d apart at x %d and %d", j, k, d, a.X, p.X)
						}
					}
				}
			}
		}
	}
}
