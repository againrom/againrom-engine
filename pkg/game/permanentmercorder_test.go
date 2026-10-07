package game

import (
	"slices"
	"testing"
)

func TestPermanentMercenaryListKeepsDeclarationOrder(t *testing.T) {
	c := Campaign{Chapters: map[int]Chapter{
		10: {Mission: 10, EnableMercenary: []int{14, 6, 13, 4, 7, 3, 2, 9, 12}},
		41: {Mission: 41, EnableMercenary: []int{10}},
		71: {Mission: 71, EnableMercenary: []int{8}},
	}}
	var all [16]bool
	for _, typ := range []int{14, 6, 13, 4, 7, 3, 2, 9, 12, 10, 8} {
		all[typ] = true
	}
	got := mercEnabledU16s(c, nil, all)
	want := []uint16{14, 6, 13, 4, 7, 3, 2, 9, 12, 10, 8}
	if !slices.Equal(got, want) {
		t.Fatalf("declaration order = %v, want %v", got, want)
	}
	// The loaded order wins.
	got = mercEnabledU16s(c, []int{12, 14}, all)
	want = []uint16{12, 14, 6, 13, 4, 7, 3, 2, 9, 10, 8}
	if !slices.Equal(got, want) {
		t.Fatalf("loaded order = %v, want %v", got, want)
	}
	// Only enabled types, never repeated; undeclared ones ascend last.
	var some [16]bool
	some[9], some[14], some[5], some[1] = true, true, true, true
	got = mercEnabledU16s(c, []int{9, 9}, some)
	want = []uint16{9, 14, 1, 5}
	if !slices.Equal(got, want) {
		t.Fatalf("partial set = %v, want %v", got, want)
	}
}

func TestPermanentMercenaryListRoundTripKeepsLoadedOrder(t *testing.T) {
	c := tavernAvailabilityCampaign()
	p := campaignProjectionAt(c, 30)
	p.PermanentMercenaries = []uint16{14, 6, 13, 4}
	progress, err := campaignProgressFromSAV(c, p)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(c, progress)
	got := mercEnabledU16s(c, town.permanentLoaded(), town.mercEnabled)
	if !slices.Equal(got, p.PermanentMercenaries) {
		t.Fatalf("round trip = %v, want %v", got, p.PermanentMercenaries)
	}
}
