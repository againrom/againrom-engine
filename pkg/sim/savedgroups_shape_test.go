package sim

import (
	"reflect"
	"testing"
)

// A zero-length Group sequence has one representation. The binary decoder
// produces nil, so an imported empty slice and a decoded group read the same.
func TestSavedGroupsEmptySequencesAreNilWhetherImportedOrDecoded(t *testing.T) {
	for _, tc := range []struct {
		name    string
		members []SavedGroupMember
		words   []uint16
	}{
		{"nil", nil, nil},
		{"empty", []SavedGroupMember{}, []uint16{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWorld(1, Bounds{16, 16}, ModeCanonical, nil, []Entity{{ID: 1, Owner: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}})
			if err != nil {
				t.Fatal(err)
			}
			g := SavedGroup{ID: 3, Selector: 17, Members: tc.members, Words: tc.words, Path: tc.words}
			if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{{Entity: 1, Patrol: tc.words}}); err != nil {
				t.Fatal(err)
			}
			groups, orders, _ := w.SavedGroups()
			if groups[0].Members != nil || groups[0].Words != nil || groups[0].Path != nil || orders[0].Patrol != nil {
				t.Fatalf("imported empty sequence kept a non-nil shape: %#v %#v", groups[0], orders[0])
			}
			hash := w.Hash()
			encoded, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(encoded); err != nil {
				t.Fatal(err)
			}
			coldGroups, coldOrders, _ := cold.SavedGroups()
			if cold.Hash() != hash || !reflect.DeepEqual(groups, coldGroups) || !reflect.DeepEqual(orders, coldOrders) {
				t.Fatal("decoded Group registry differs from the imported one")
			}
		})
	}
}

// Detaching the last member leaves the registry in the decoded shape.
func TestSavedGroupsDetachingTheLastMemberLeavesNilMembers(t *testing.T) {
	w, err := NewWorld(1, Bounds{16, 16}, ModeCanonical, nil, []Entity{{ID: 1, Owner: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	g := SavedGroup{ID: 3, Selector: 17, Members: []SavedGroupMember{{Archive: 1, Entity: 1, Bound: true}}}
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	w.detachSavedMember(1)
	groups, _, _ := w.SavedGroups()
	if groups[0].Members != nil {
		t.Fatalf("emptied membership has shape %#v", groups[0].Members)
	}
}
