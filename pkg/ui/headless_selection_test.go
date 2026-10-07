package ui

import (
	"reflect"
	"testing"
)

func TestHeadlessSelectAllMatchesProductionKeyAndReturnsDetachedSelection(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	in := a.headlessIdleInput()
	in.AnyKey, in.SelectAll = true, true
	a.step(in, a.headlessAt())
	want := append([]uint32(nil), a.flow.viewer.sel...)
	a.flow.viewer.sel = nil
	if err := a.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.HeadlessSelection(), want) {
		t.Fatalf("E selection=%v want%v", a.HeadlessSelection(), want)
	}
	// Check copy semantics even when the generic opener has no local actors.
	a.flow.viewer.sel = []uint32{7, 9}
	got := a.HeadlessSelection()
	got[0] = 42
	if a.flow.viewer.sel[0] != 7 {
		t.Fatal("headless observer mutated live selection")
	}
}
