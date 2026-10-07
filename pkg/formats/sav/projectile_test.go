package sav

import (
	"reflect"
	"strings"
	"testing"
)

// TestProjectileLeafNamesMatchTheWorldShape locks the reader/writer's own
// hardcoded leaf list against worldProjectileLeaves, so the two files can
// never silently drift apart (world_state.go owns the shape, this file owns
// the typed projection over it).
func TestProjectileLeafNamesMatchTheWorldShape(t *testing.T) {
	if len(projectileLeafNames) != len(worldProjectileLeaves) {
		t.Fatalf("%d leaf names, worldProjectileLeaves has %d", len(projectileLeafNames), len(worldProjectileLeaves))
	}
	for i, name := range projectileLeafNames {
		leaf := worldProjectileLeaves[i]
		if leaf.name != name {
			t.Fatalf("leaf %d is %q, worldProjectileLeaves names it %q", i, name, leaf.name)
		}
		if leaf.kind != 2 {
			t.Fatalf("leaf %q has kind %d, want 2 (this reader/writer hardcodes every leaf as an int)", name, leaf.kind)
		}
	}
}

func TestProjectilesDecodesAllocatorAndSections(t *testing.T) {
	f := &File{}
	f.splitTail(literalStateStore(literalWorldDirectories(266, 7, 266)))
	store, ok, err := f.Projectiles()
	if err != nil || !ok {
		t.Fatalf("Projectiles() = %v, %v, %v", store, ok, err)
	}
	if store.FreeIndex != 0x1234 {
		t.Fatalf("FreeIndex = %#x, want 0x1234", store.FreeIndex)
	}
	if !reflect.DeepEqual(store.IDs, []uint16{266, 7, 266}) {
		t.Fatalf("IDs = %v, want [266 7 266] (wire order and multiplicity preserved)", store.IDs)
	}
	if len(store.Items) != 2 {
		t.Fatalf("Items has %d entries, want 2 (one per distinct id)", len(store.Items))
	}
	byID := map[uint16]Projectile{}
	for _, p := range store.Items {
		byID[p.ID] = p
	}
	for _, id := range []uint16{266, 7} {
		p, ok := byID[id]
		if !ok {
			t.Fatalf("Items missing id %d", id)
		}
		base := int32(id) * 100
		want := Projectile{
			ID: id,
			X:  base, Y: base + 1, Z: base + 2, Picture: base + 3, Dir: base + 4, Phase: base + 5,
			LastAction: base + 6, Action: base + 7, ActionDir: base + 8, ActionTarget: base + 9,
			ActionX: base + 10, ActionY: base + 11, ActionZ: base + 12, ActionPhase: base + 13,
			ActionSegments: base + 14, ActionSpell: base + 15,
		}
		if p != want {
			t.Fatalf("id %d decoded %+v, want %+v", id, p, want)
		}
	}
}

// TestNoProjectilesSectionIsNotAnError mirrors TestNoFogSectionIsNotAnError:
// a store with neither leaf present is a between-mission save's own shape.
func TestNoProjectilesSectionIsNotAnError(t *testing.T) {
	dirs := literalWorldDirectories()
	kept := dirs[:0]
	for _, d := range dirs {
		if d.name != "Projectiles" {
			kept = append(kept, d)
		}
	}
	f := &File{}
	f.splitTail(literalStateStore(kept))
	store, ok, err := f.Projectiles()
	if err != nil {
		t.Fatalf("a store with no Projectiles section errored: %v", err)
	}
	if ok || len(store.Items) != 0 || len(store.IDs) != 0 {
		t.Fatalf("Projectiles() = %+v, %v; want no record", store, ok)
	}
}

// TestProjectilesAcceptsTheSingletonIDsCompatibilityArm is SAV-PROJLOAD-429's
// own compatibility case: a kind-2 IDs leaf naming one id.
func TestProjectilesAcceptsTheSingletonIDsCompatibilityArm(t *testing.T) {
	dirs := literalWorldDirectories(266)
	for i, d := range dirs {
		if d.name == "Projectiles" {
			dirs[i].leaves[1] = literalStateLeaf{name: "IDs", kind: 2, word: 266}
		}
	}
	f := &File{}
	f.splitTail(literalStateStore(dirs))
	store, ok, err := f.Projectiles()
	if err != nil || !ok {
		t.Fatalf("Projectiles() = %v, %v, %v", store, ok, err)
	}
	if !reflect.DeepEqual(store.IDs, []uint16{266}) || len(store.Items) != 1 || store.Items[0].ID != 266 {
		t.Fatalf("singleton IDs decoded %+v", store)
	}
}

// TestProjectilesMissingFreeIndexOrIDsDefaultsIndependently is
// SAV-PROJLOAD-429's own loader-tolerant absence rule for each leaf.
func TestProjectilesMissingFreeIndexOrIDsDefaultsIndependently(t *testing.T) {
	for _, name := range []string{"missing-IDs", "missing-FreeIndex"} {
		t.Run(name, func(t *testing.T) {
			dirs := literalWorldDirectories()
			for i, d := range dirs {
				if d.name != "Projectiles" {
					continue
				}
				switch name {
				case "missing-IDs":
					dirs[i].leaves = d.leaves[:1] // FreeIndex only
				case "missing-FreeIndex":
					dirs[i].leaves = d.leaves[1:] // IDs only
				}
			}
			f := &File{}
			f.splitTail(literalStateStore(dirs))
			store, ok, err := f.Projectiles()
			if err != nil || !ok {
				t.Fatalf("Projectiles() = %v, %v, %v", store, ok, err)
			}
			if name == "missing-IDs" {
				if store.FreeIndex != 0x1234 || len(store.IDs) != 0 || len(store.Items) != 0 {
					t.Fatalf("missing-IDs decoded %+v", store)
				}
			} else {
				if store.FreeIndex != 0 || len(store.IDs) != 0 {
					t.Fatalf("missing-FreeIndex decoded %+v", store)
				}
			}
		})
	}
}

// TestProjectilesMissingPerSectionLeafDefaultsToZero is the one loader arm
// SAV-PROJCORP-430's exhaustive corpus never exercises: a referenced Prj<id>
// section that does not carry all sixteen leaves.
func TestProjectilesMissingPerSectionLeafDefaultsToZero(t *testing.T) {
	dirs := literalWorldDirectories(266)
	for i, d := range dirs {
		if d.name == "Prj266" {
			dirs[i].leaves = d.leaves[:1] // x only
		}
	}
	f := &File{}
	f.splitTail(literalStateStore(dirs))
	store, ok, err := f.Projectiles()
	if err != nil || !ok || len(store.Items) != 1 {
		t.Fatalf("Projectiles() = %v, %v, %v", store, ok, err)
	}
	p := store.Items[0]
	if p.X != 266*100 {
		t.Fatalf("present leaf x = %d, want %d", p.X, 266*100)
	}
	if p.Y != 0 || p.ActionSpell != 0 {
		t.Fatalf("absent leaves did not default to zero: %+v", p)
	}
}

// TestSetProjectilesRoundTripsThroughReopen writes a store, reopens the
// marshaled file and reads it back through the ordinary loader-tolerant
// path: this is the layout test, since a decode that recovers the exact
// input locks the writer's kind and order (kind 2 for every leaf, kind 6
// canonical for IDs) against the shape that reader already accepts.
func TestSetProjectilesRoundTripsThroughReopen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		store ProjectileStore
	}{
		{"empty", ProjectileStore{FreeIndex: 0}},
		{"one item", ProjectileStore{FreeIndex: 267, IDs: []uint16{266}, Items: []Projectile{{
			ID: 266, X: 1, Y: 2, Z: 3, Picture: 4, Dir: 5, Phase: 6, LastAction: 7, Action: 8,
			ActionDir: 9, ActionTarget: 10, ActionX: 11, ActionY: 12, ActionZ: 13, ActionPhase: 14,
			ActionSegments: 15, ActionSpell: 16,
		}}}},
		{"repeated id", ProjectileStore{FreeIndex: 9, IDs: []uint16{5, 9, 5}, Items: []Projectile{
			{ID: 5, X: 50}, {ID: 9, X: 90},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := standard()
			fx.tail = literalStateStore(literalWorldDirectories())
			f := open(t, fx)
			if err := f.SetProjectiles(tc.store); err != nil {
				t.Fatalf("SetProjectiles: %v", err)
			}
			reopened, err := Open(f.Marshal())
			if err != nil {
				t.Fatalf("re-Open: %v", err)
			}
			got, ok, err := reopened.Projectiles()
			if err != nil {
				t.Fatalf("Projectiles (reopened): %v", err)
			}
			// SetProjectiles always writes the canonical shape (SAV-PROJSTORE-428):
			// FreeIndex and IDs both present even for an all-empty store.
			if !ok {
				t.Fatal("Projectiles() ok=false after SetProjectiles; the canonical producer always emits FreeIndex and IDs")
			}
			if got.FreeIndex != tc.store.FreeIndex {
				t.Fatalf("FreeIndex = %#x, want %#x", got.FreeIndex, tc.store.FreeIndex)
			}
			if !reflect.DeepEqual(got.IDs, tc.store.IDs) && !(len(got.IDs) == 0 && len(tc.store.IDs) == 0) {
				t.Fatalf("IDs = %v, want %v", got.IDs, tc.store.IDs)
			}
			if len(got.Items) != len(tc.store.Items) {
				t.Fatalf("Items has %d entries, want %d", len(got.Items), len(tc.store.Items))
			}
			want := map[uint16]Projectile{}
			for _, p := range tc.store.Items {
				want[p.ID] = p
			}
			for _, p := range got.Items {
				if want[p.ID] != p {
					t.Fatalf("id %d round-tripped %+v, want %+v", p.ID, p, want[p.ID])
				}
			}
		})
	}
}

// TestSetProjectilesLeavesOtherSectionsAlone is the leave-alone test: writing
// Projectiles must reproduce every other city/world leaf's own decoded value
// exactly, proving this writer's span is exactly the Projectiles subtree.
func TestSetProjectilesLeavesOtherSectionsAlone(t *testing.T) {
	fx := standard()
	fx.tail = literalStateStore(literalWorldDirectories(7))
	f := open(t, fx)
	before, err := parseWorldState(f.Store)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetProjectiles(ProjectileStore{FreeIndex: 1, IDs: []uint16{266}, Items: []Projectile{{ID: 266, X: 9}}}); err != nil {
		t.Fatalf("SetProjectiles: %v", err)
	}
	after, err := parseWorldState(f.Store)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range before.values {
		if path == "/Projectiles/FreeIndex" || path == "/Projectiles/IDs" || strings.HasPrefix(path, "/Prj7/") {
			continue // Prj7 is checked below (it must be gone entirely); the other two are meant to change.
		}
		got, ok := after.values[path]
		// int32 on a kind 0/6 value is only ever the transient pool offset
		// (city_state.go), which a fresh serialize is free to relocate; the
		// content that matters there is bytes, exactly TestWorldStateLiteral-
		// EmptyAndProjectileValues' own asymmetric comparison.
		if !ok || got.kind != want.kind ||
			(want.kind == 2 && got.int32 != want.int32) ||
			(want.kind != 2 && string(got.bytes) != string(want.bytes)) {
			t.Fatalf("%s changed: got %+v, want %+v", path, got, want)
		}
	}
	// Prj7 is stale (7 is not in the new IDs) and must be gone entirely.
	if _, ok := after.directoryKinds["/Prj7"]; ok {
		t.Fatal("a stale Prj<id> directory survived SetProjectiles")
	}
	for _, name := range projectileLeafNames {
		if _, ok := after.values["/Prj7/"+name]; ok {
			t.Fatalf("stale leaf /Prj7/%s survived SetProjectiles", name)
		}
	}
	if got := after.values["/Prj266/x"]; got.int32 != 9 {
		t.Fatalf("new section did not take: /Prj266/x = %+v", got)
	}
}

// TestSetProjectilesPreservesAnExistingDirectoryKindWord is the flag-bit
// preservation this file's own doc comment requires: a directory's kind word
// may carry bits validateStateShape never inspects (e.g. 0x10), and a write
// that keeps an id must not replace them with a freshly hardcoded 1.
func TestSetProjectilesPreservesAnExistingDirectoryKindWord(t *testing.T) {
	fx := standard()
	fx.tail = literalStateStore(literalWorldDirectories(266))
	f := open(t, fx)
	before, err := parseWorldState(f.Store)
	if err != nil {
		t.Fatal(err)
	}
	mutated := cloneCityState(before)
	mutated.directoryKinds["/Prj266"] |= 0x10
	mutated.directoryKinds["/Projectiles"] |= 0x10
	encoded, err := serializeWorldState(mutated)
	if err != nil {
		t.Fatal(err)
	}
	f.splitTail(append(encoded, f.TailRest...))
	if err := f.SetProjectiles(ProjectileStore{FreeIndex: 1, IDs: []uint16{266}, Items: []Projectile{{ID: 266, X: 99}}}); err != nil {
		t.Fatalf("SetProjectiles: %v", err)
	}
	after, err := parseWorldState(f.Store)
	if err != nil {
		t.Fatal(err)
	}
	if after.directoryKinds["/Prj266"]&0x10 == 0 {
		t.Fatal("SetProjectiles dropped an existing Prj<id> directory's own flag bit")
	}
	if after.directoryKinds["/Projectiles"]&0x10 == 0 {
		t.Fatal("SetProjectiles dropped the Projectiles directory's own flag bit")
	}
	if got := after.values["/Prj266/x"]; got.int32 != 99 {
		t.Fatal("value write did not take alongside the preserved kind word")
	}
}

// TestSetProjectilesRefusesABetweenMissionSave matches
// exportOriginalCellRecords' own guard: a between-mission save has no world
// half and therefore no Projectiles subtree to hold.
func TestSetProjectilesRefusesABetweenMissionSave(t *testing.T) {
	fx := standard()
	fx.noWorld = true
	f := open(t, fx)
	if err := f.SetProjectiles(ProjectileStore{}); err == nil {
		t.Fatal("SetProjectiles accepted a between-mission save")
	}
}

// TestSetProjectilesRefusesAnIDsItemsMismatch checks the free consistency
// check serializeWorldState already performs: an id named by IDs with no
// matching Items entry has no Prj<id> section to populate, and worldStateShape
// requires the section's full sixteen leaves regardless.
func TestSetProjectilesRefusesAnIDsItemsMismatch(t *testing.T) {
	fx := standard()
	fx.tail = literalStateStore(literalWorldDirectories())
	f := open(t, fx)
	if err := f.SetProjectiles(ProjectileStore{IDs: []uint16{266}}); err == nil {
		t.Fatal("SetProjectiles accepted an id named by IDs with no Items entry")
	}
}
