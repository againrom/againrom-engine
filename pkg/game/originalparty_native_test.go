package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
)

func TestNativeOriginalPartyWithdrawsOnlySavedNonzeroPlacements(t *testing.T) {
	m := resumeMap()
	before := append(m.Units[:0:0], m.Units...)
	for _, party := range [][]mapload.PartyMember{nil, {{ID: "native"}}, {{ID: "hero", Saved: &mapload.Saved{MapUnitID: 0}}}} {
		if n := withdrawSavedPartyPlacements(m, party); n != 0 || !reflect.DeepEqual(m.Units, before) {
			t.Fatalf("no-placement party changed map: removed=%d", n)
		}
	}
	party := []mapload.PartyMember{
		{ID: "hero", Saved: &mapload.Saved{}},
		{ID: "joined", Saved: &mapload.Saved{MapUnitID: 102}},
		{ID: "repeated", Saved: &mapload.Saved{MapUnitID: 102}},
		{ID: "other-mission", Saved: &mapload.Saved{MapUnitID: 555}},
	}
	if n := withdrawSavedPartyPlacements(m, party); n != 1 {
		t.Fatalf("removed %d placements, want one", n)
	}
	want := before[:0:0]
	for _, u := range before {
		if u.UnitID != 102 {
			want = append(want, u)
		}
	}
	if !reflect.DeepEqual(m.Units, want) {
		t.Fatal("changed a retained placement or its order")
	}
	if n := withdrawSavedPartyPlacements(m, party); n != 0 {
		t.Fatalf("second withdrawal=%d", n)
	}
	if n := withdrawSavedPartyPlacements(nil, party); n != 0 {
		t.Fatalf("nil map withdrawal=%d", n)
	}
}
