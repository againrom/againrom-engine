package ui

import (
	"reflect"
	"testing"
)

func TestCollectKeyAddressesTheSelectedMembers(t *testing.T) {
	a, v := mkOnMap(t)
	a.SetMissionKeys("pickup-p")
	v.sel = selection{4, 9}
	var got []uint32
	var places []int
	a.flow.grab = func(id uint32, place, _ int, action GrabAction) {
		if action != GrabCollect {
			t.Errorf("P action %v, want collect", action)
		}
		got, places = append(got, id), append(places, place)
	}
	if err := a.HeadlessKey("p"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []uint32{4, 9}) || !reflect.DeepEqual(places, []int{0, 1}) {
		t.Fatalf("P addressed %v at places %v, want selected members [4 9] at [0 1]", got, places)
	}
}
