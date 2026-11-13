package game

import (
	"os"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// The graph W draws of the first game's install equals the census of
// REG-SCN-064: 13 town chapters whose buildings name 26 distinct missions,
// each by exactly one building: the tavern 19, with three visitors that
// hand out nothing at the head of chapters 50, 110 and 130, the shop 4 and
// the school 3.
func TestReleaseCampaignGraphEqualsTheHandoutCensus(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the census needs a lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	page := x.modHost().(ui.ModCampaignGraph).CampaignGraph()
	if page.Mod || page.Refused || page.Unread {
		t.Fatalf("the install's graph is flagged: mod %t, refused %t, unread %t", page.Mod, page.Refused, page.Unread)
	}
	var towns, empty []int
	places := map[ui.GraphPlace]int{}
	handed := map[int]int{} // a mission's number to the buildings that hand it out
	for i, n := range page.Nodes {
		switch n.Kind {
		case ui.GraphTown:
			towns = append(towns, n.Number)
		case ui.GraphHandout:
			var town, mission int
			for _, e := range page.Edges {
				switch {
				case e.To == i && page.Nodes[e.From].Kind == ui.GraphTown:
					town = page.Nodes[e.From].Number
				case e.From == i && page.Nodes[e.To].Kind == ui.GraphMission:
					mission = page.Nodes[e.To].Number
				}
			}
			if town == 0 {
				t.Fatalf("a hand-out on row %d hangs from no town", n.Row)
			}
			if n.Empty != (mission == 0) {
				t.Fatalf("the hand-out on row %d of town %d: empty %t, mission %d", n.Row, town, n.Empty, mission)
			}
			if n.Empty {
				if n.Place != ui.GraphInn {
					t.Fatalf("an empty hand-out of town %d is no tavern visitor", town)
				}
				empty = append(empty, town)
				continue
			}
			places[n.Place]++
			handed[mission]++
		}
	}
	slices.Sort(towns)
	slices.Sort(empty)
	if len(towns) != 13 {
		t.Fatalf("%d town chapters %v, want 13", len(towns), towns)
	}
	if want := []int{50, 110, 130}; !slices.Equal(empty, want) {
		t.Fatalf("tavern visitors with nothing in chapters %v, want %v", empty, want)
	}
	if places[ui.GraphInn] != 19 || places[ui.GraphShop] != 4 || places[ui.GraphSchool] != 3 {
		t.Fatalf("hand-outs: tavern %d, shop %d, school %d; want 19, 4, 3", places[ui.GraphInn], places[ui.GraphShop], places[ui.GraphSchool])
	}
	if len(handed) != 26 {
		t.Fatalf("%d missions handed out, want 26", len(handed))
	}
	for mission, count := range handed {
		if count != 1 {
			t.Fatalf("mission %d is handed out by %d buildings", mission, count)
		}
	}
	missions, open := 0, 0
	for _, n := range page.Nodes {
		if n.Kind == ui.GraphMission {
			missions++
			if n.Source != "" {
				open++
			}
		}
	}
	t.Logf("graph: %d nodes, %d edges; %d missions, %d with an install map to open; towns %v", len(page.Nodes), len(page.Edges), missions, open, towns)
}
