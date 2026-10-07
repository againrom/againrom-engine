package sim

import "testing"

// D-1: item, effect, spell, sack and container all index a field of r with no
// check on r itself. World.EquipSourceItem calls n.savedObjects.item(...)
// with no guard of its own (sourceequipmove.go), and SourceTownEquipment
// (pkg/mapload) always builds an isolated World with a nil registry, so an
// item carrying a non-zero ObjectID into that path would dereference nil. No
// current caller passes such an ObjectID there, which is why this was latent
// rather than live; this test reaches the same five accessors directly so the
// gap is caught even if that call graph changes.
func TestNilSavedObjectsRegistryAccessorsReturnNilNotPanic(t *testing.T) {
	var r *SavedObjects
	if got := r.item(10); got != nil {
		t.Fatalf("item on nil registry = %+v, want nil", got)
	}
	if got := r.effect(10); got != nil {
		t.Fatalf("effect on nil registry = %+v, want nil", got)
	}
	if got := r.spell(10); got != nil {
		t.Fatalf("spell on nil registry = %+v, want nil", got)
	}
	if got := r.sack(10); got != nil {
		t.Fatalf("sack on nil registry = %+v, want nil", got)
	}
	if got := r.container(SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 1}); got != nil {
		t.Fatalf("container on nil registry = %+v, want nil", got)
	}
}

// TestEquipSourceItemOnARegistryLessWorldRefusesInsteadOfPanicking reproduces
// the live shape D-1 was raised against: a World built the way
// mapload.SourceTownEquipment builds one, with no saved-objects registry
// (sourceMutationWorld never imports one, so w.savedObjects is the zero
// value, nil), asked to equip an item that names a saved ObjectID. Before the
// guard, n.savedObjects.item(item.ObjectID) dereferenced that nil receiver;
// after it, the lookup returns nil like any other unknown ID and
// EquipSourceItem takes its ordinary "unknown/foreign object" refusal.
func TestEquipSourceItemOnARegistryLessWorldRefusesInsteadOfPanicking(t *testing.T) {
	w := sourceMutationWorld(t, sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 2))
	if w.savedObjects != nil {
		t.Fatal("fixture precondition: this world must start with no saved-object registry")
	}
	item := sourceEquipmentWeapon(0x0201, 3, 4, 5, 6, 1)
	item.ObjectID = 10
	if w.EquipSourceItem(1, item) {
		t.Fatal("equip of a saved-registry item on a registry-less world must refuse, not succeed")
	}
}
