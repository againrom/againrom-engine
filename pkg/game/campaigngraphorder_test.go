package game

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// Side hand-outs lead, then main, then none.
func TestGraphHandoutsStandSideThenMainThenNone(t *testing.T) {
	ch := Chapter{
		InnNPC: []int{5, 6, 7, 8},
		Inn:    []int{0, 30, 31, 0},
		Shop:   []int{0},
		School: []int{40},
	}
	want := []graphHandout{
		{place: ui.GraphInn, npc: 7, mission: 31},
		{place: ui.GraphInn, npc: 6, mission: 30},
		{place: ui.GraphSchool, mission: 40},
		{place: ui.GraphInn, npc: 5, mission: 0},
		{place: ui.GraphInn, npc: 8, mission: 0},
		{place: ui.GraphShop, mission: 0},
	}
	if got := graphHandouts(ch); !reflect.DeepEqual(got, want) {
		t.Fatalf("hand-outs\n got %+v\nwant %+v", got, want)
	}
}

// A group keeps the scenario's order.
func TestGraphHandoutsKeepTheOrderWithinOneGroup(t *testing.T) {
	ch := Chapter{InnNPC: []int{1, 2}, Inn: []int{71, 72}, Shop: []int{0}}
	want := []graphHandout{
		{place: ui.GraphInn, npc: 1, mission: 71},
		{place: ui.GraphInn, npc: 2, mission: 72},
		{place: ui.GraphShop, mission: 0},
	}
	if got := graphHandouts(ch); !reflect.DeepEqual(got, want) {
		t.Fatalf("hand-outs\n got %+v\nwant %+v", got, want)
	}
}
