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
	a.flow.grab = func(id uint32, _, _ int, action GrabAction) {
		if action != GrabCollect {
			t.Errorf("P action %v, want collect", action)
		}
		got = append(got, id)
	}
	if err := a.HeadlessKey("p"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []uint32{4, 9}) {
		t.Fatalf("P addressed %v, want selected members [4 9]", got)
	}
}
