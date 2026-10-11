package sim

import (
	"bytes"
	"testing"
)

// TestAConstructedRegistryKeepsItsMarkThroughTheByteForm checks that a
// constructed registry the byte form writes, because the World's state no
// longer builds the same one, decodes as a constructed registry: when the
// state builds it again, the live World and its decoded copy both omit it and
// hash alike. A form without the mark decodes as an imported registry.
func TestAConstructedRegistryKeepsItsMarkThroughTheByteForm(t *testing.T) {
	w, err := NewWorld(17, Bounds{Width: 40, Height: 40}, ModeCanonical, nil, []Entity{
		{ID: 1, Owner: 9, Group: 19, X: 10, Y: 10, HP: 100, MaxHP: 100, Reach: 1},
		{ID: 2, Owner: SelfSlot, X: 30, Y: 30, HP: 100, MaxHP: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !w.GroupsConstructed() || !w.groupsAsConstructed() {
		t.Fatal("the mission-start registry is not the constructed one")
	}
	w.entities[0].Reach = 4
	if w.groupsAsConstructed() {
		t.Fatal("a changed reach still builds the stored registry")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(b); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("decode of the written constructed registry", err)
	}
	w.entities[0].Reach, cold.entities[0].Reach = 1, 1
	if !w.groupsAsConstructed() || !cold.groupsAsConstructed() || cold.Hash() != w.Hash() {
		t.Fatalf("live omitted %v, decoded omitted %v", w.groupsAsConstructed(), cold.groupsAsConstructed())
	}

	// Loss control: the same bytes without the mark hold an imported registry.
	w.entities[0].Reach = 4
	marked, _ := w.MarshalBinary()
	w.savedGroups.fresh = false
	unmarked, _ := w.MarshalBinary()
	w.savedGroups.fresh = true
	if len(marked) != len(unmarked)+1 || bytes.Equal(marked, unmarked) {
		t.Fatal("the mark is not one byte of the registry payload")
	}
	var imported World
	if err := imported.UnmarshalBinary(unmarked); err != nil {
		t.Fatal(err)
	}
	imported.entities[0].Reach = 1
	if imported.GroupsConstructed() || imported.groupsAsConstructed() {
		t.Fatal("a registry written without the mark decoded as the constructed one")
	}
}
