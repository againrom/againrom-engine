package ui

import (
	"image"
	"slices"
	"testing"
)

// A unit given to another player while selected leaves a selection of several
// units, on the next entity push and on a group recall; a lone foreign unit
// stays selected as a plain click leaves it.
func TestAGivenUnitLeavesAPartySelection(t *testing.T) {
	_, v := mkOnMap(t)
	v.sel = selection{4, 9}
	v.groups[3] = selection{4, 9}
	v.SetEntities([]MapEntity{
		{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1},
		{ID: 9, Cell: image.Pt(5, 3), Life: LifeAlive, Owner: 8},
	})
	if want := (selection{4}); !slices.Equal(v.sel, want) {
		t.Fatalf("selection after the handover = %v, want %v", v.sel, want)
	}
	v.sel = nil
	v.groupKey(3, false, false, false)
	if want := (selection{4}); !slices.Equal(v.sel, want) {
		t.Fatalf("group recall after the handover = %v, want %v", v.sel, want)
	}
	v.sel = selection{9}
	v.SetEntities(v.entities)
	if want := (selection{9}); !slices.Equal(v.sel, want) {
		t.Fatalf("a lone foreign selection = %v, want %v", v.sel, want)
	}
	if v.LocalOwns(8) || !v.LocalOwns(1) {
		t.Fatal("LocalOwns differs from the local participant")
	}
}
