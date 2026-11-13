package game

import (
	"testing"

	"againrom/pkg/herokind"
)

func TestCampaignWinStoresKeepTheirRoutes(t *testing.T) {
	stock := routeCampaign(t, routeSections())
	for _, tc := range []struct {
		name, source              string
		won, main, next, selected int
		announced                 bool
	}{
		{"main chain", "", 10, 10, 20, 20, true},
		{"town boundary", "", 20, 20, 30, 30, false},
		{"side from town", "", 31, 30, 30, 30, false},
		{"earlier main", "", 20, 40, 40, 40, false},
		{"removed lead", routeLeadOff, 10, 10, 20, 20, true},
		{"mod side successor", routeSideToMain, 31, 30, 50, 50, true},
		{"last main", "", 60, 60, 60, 60, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := routeMod(t, stock, tc.source)
			for _, restored := range []bool{false, true} {
				town := NewTown(c)
				town.main, town.selected = tc.main, tc.main
				town.records = newCampaignRecords(c, tc.main)
				if restored {
					town.progress = &campaignProgress{campaignRecords: newCampaignRecords(c, tc.main), selected: tc.main, autoGet: 0xffffffff}
				}
				_, accepted := town.Won(tc.won)
				if restored && tc.name == "earlier main" {
					if accepted {
						t.Fatal("restored earlier main accepted")
					}
					continue
				}
				r := town.currentRecords()
				if !accepted || r.main.mission != tc.next || town.selectedMission() != tc.selected || r.main.announced != tc.announced {
					t.Fatalf("restored=%t: accepted=%t main=%d selected=%d announced=%t; want %d/%d/%t", restored, accepted, r.main.mission, town.selectedMission(), r.main.announced, tc.next, tc.selected, tc.announced)
				}
			}
		})
	}
	for _, src := range []string{routeTwoGuests, routeOneGuest} {
		_, d := routeMod(t, stock, src)
		for _, kind := range herokind.PictureOrder {
			want := 25
			if kind.Female() {
				want = 26
			}
			if src == routeOneGuest {
				want = 25
				if !kind.Mage() {
					want = 0
				}
			}
			m, ok := d.AfterWin(20, kind)
			if ok != (want != 0) || ok && m.Number != want {
				t.Fatalf("guest by hero: %+v %t, want %d", m, ok, want)
			}
		}
	}
}
