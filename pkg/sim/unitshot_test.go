package sim

import (
	"slices"
	"testing"
)

func shotWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func shotPair(distanceCells int32) []Entity {
	return []Entity{
		{ID: 1, X: 1, Y: 4, HP: 100, MaxHP: 100, Owner: 1, Reach: 6, HasAttackTarget: true, AttackTarget: 2, AttackTargetKind: AttackTargetUnit},
		{ID: 2, X: 1 + distanceCells, Y: 4, HP: 100, MaxHP: 100, Owner: 2},
	}
}

func TestReleaseUnitShotBuildsTheRecordTheClaimsDescribe(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 3, Phases: 4, OffsetX: 16, OffsetY: -24,
		Dir: func(dx, dy int) int32 { return int32(dx/100 + dy) }}) {
		t.Fatal("no record built")
	}
	got := w.SavedProjectiles()
	if got.FreeIndex != 1 || !slices.Equal(got.IDs, []uint16{0}) || len(got.Items) != 1 {
		t.Fatalf("allocator %d, IDs %v, %d items", got.FreeIndex, got.IDs, len(got.Items))
	}
	// Start: the shooter's point (cell 1,4 centre) plus the offset. Distance:
	// 4 cells = 1024 units, 1024/200 = 5 segments. The first driver call moves
	// one fifth of the way to the target's point and leaves 4 segments.
	startX, startY := int32(1*256+128+16), int32(4*256+128-24)
	targetX, targetY := int32(5*256+128), int32(4*256+128)
	want := SavedProjectile{ID: 0, X: startX + (targetX-startX)/5, Y: startY + (targetY-startY)/5, Picture: 3,
		Dir: int32((targetX-startX)/100 + targetY - startY), Phase: 0, LastAction: 1, Action: 1, ActionTarget: 2,
		ActionX: targetX, ActionY: targetY, ActionPhase: 1, ActionSegments: 4}
	want.ActionDir = want.Dir
	want.Phase = (want.ActionPhase / 2) % 4
	if got.Items[0] != want {
		t.Fatalf("record %+v, want %+v", got.Items[0], want)
	}
	d := w.SavedWorldEffectDrivers()
	if d == nil || len(d.Projectiles) != 1 || d.Projectiles[0] != (SavedProjectileDriver{ID: 0, Phases: 4, Target: 2, HasTarget: true}) {
		t.Fatalf("driver %+v", d)
	}
	for _, e := range w.Entities() {
		if e.HP != 100 {
			t.Fatalf("the shot changed entity %d's hit points to %d", e.ID, e.HP)
		}
	}
}

func TestReleaseUnitShotFliesToTheTargetAndRetires(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 1, Phases: 1})
	for call := 1; call <= 5; call++ {
		p, ok := savedProjectileOf(w, 0)
		if !ok || p.ActionSegments != int32(5-call) {
			t.Fatalf("after call %d: %+v (present %v)", call, p, ok)
		}
		if call < 5 {
			Step(w, nil)
		}
	}
	p, _ := savedProjectileOf(w, 0)
	if p.X != p.ActionX || p.Y != p.ActionY {
		t.Fatalf("the final segment ends at %d,%d, not on the target's point %d,%d", p.X, p.Y, p.ActionX, p.ActionY)
	}
	Step(w, nil)
	if len(w.SavedProjectiles().Items) != 0 || len(w.SavedProjectiles().IDs) != 0 || w.SavedProjectiles().FreeIndex != 1 {
		t.Fatalf("the record was not collected: %+v", w.SavedProjectiles())
	}
}

func savedProjectileOf(w *World, id uint16) (SavedProjectile, bool) {
	for _, p := range w.SavedProjectiles().Items {
		if p.ID == id {
			return p, true
		}
	}
	return SavedProjectile{}, false
}

func TestReleaseUnitShotAllocatesFromTheCounterIntoBucketOrder(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	// Ids 272 and 0 hash to bucket 0 and 16 to bucket 1: a new id 0 goes to
	// the head of bucket 0, ahead of the older 272, and the counter wraps.
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 0, IDs: []uint16{272, 16}, Items: []SavedProjectile{{ID: 272}, {ID: 16}}})
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 2, Phases: 1}) {
		t.Fatal("no record built")
	}
	got := w.SavedProjectiles()
	if !slices.Equal(got.IDs, []uint16{0, 272, 16}) || got.FreeIndex != 1 {
		t.Fatalf("IDs %v, allocator %d", got.IDs, got.FreeIndex)
	}
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 65535})
	w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 2, Phases: 1})
	if got := w.SavedProjectiles(); got.FreeIndex != 0 || !slices.Equal(got.IDs, []uint16{65535}) {
		t.Fatalf("wrap: IDs %v, allocator %d", got.IDs, got.FreeIndex)
	}
}

func TestReleaseUnitShotRefusals(t *testing.T) {
	structure := shotPair(4)
	structure[0].AttackTargetKind = AttackTargetStructure
	idle := shotPair(4)
	idle[0].HasAttackTarget = false
	for name, c := range map[string]struct {
		ents    []Entity
		picture int32
	}{
		"picture 0":        {shotPair(4), 0},
		"picture above 12": {shotPair(4), 13},
		"structure target": {structure, 1},
		"no target":        {idle, 1},
	} {
		w := shotWorld(t, c.ents...)
		if w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: c.picture, Phases: 1}) || len(w.SavedProjectiles().Items) != 0 || w.SavedProjectiles().FreeIndex != 0 {
			t.Errorf("%s built a record", name)
		}
	}
	if shotWorld(t, shotPair(4)...).ReleaseUnitShot(UnitShot{Shooter: 9, Picture: 1}) {
		t.Error("an absent shooter built a record")
	}
}

func TestReleaseUnitShotWithinOneSegmentLivesOneCall(t *testing.T) {
	w := shotWorld(t, shotPair(0)...)
	w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 1, Phases: 1})
	if got := w.SavedProjectiles(); len(got.Items) != 0 || len(got.IDs) != 0 || got.FreeIndex != 1 {
		t.Fatalf("a shot under 200 units is collected on its first call: %+v", got)
	}
}

func TestReleasedShotSurvivesTheWorldBinaryForm(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 4, Phases: 2})
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("the binary form changed the hash")
	}
	for range 3 {
		Step(w, nil)
		Step(&back, nil)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("the restored world flies the record differently")
	}
}

func TestProjectileTargetKey(t *testing.T) {
	native := Entity{ID: 7}
	bound := Entity{ID: 7, SourceBinding: SourceBinding{Class: 1, RuntimeID: 91}}
	boundNoID := Entity{ID: 7, SourceBinding: SourceBinding{Class: 1}}
	if ProjectileTargetKey(native) != 7 || ProjectileTargetKey(bound) != 91 || ProjectileTargetKey(boundNoID) != 7 {
		t.Fatal("target keys differ")
	}
}

func TestReleaseUnitShotKeepsItemsInIDOrder(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 16, IDs: []uint16{3}, Items: []SavedProjectile{{ID: 3}}})
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 2, Phases: 1}) {
		t.Fatal("no record built")
	}
	got := w.SavedProjectiles()
	// Id 16 hashes to bucket 1 and id 3 to bucket 0: the saved order is 3, 16.
	if !slices.Equal(got.IDs, []uint16{3, 16}) || got.Items[0].ID != 3 || got.Items[1].ID != 16 {
		t.Fatalf("IDs %v, items %d then %d", got.IDs, got.Items[0].ID, got.Items[1].ID)
	}
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 0, IDs: []uint16{16}, Items: []SavedProjectile{{ID: 16}}})
	w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 2, Phases: 1})
	got = w.SavedProjectiles()
	if !slices.Equal(got.IDs, []uint16{0, 16}) || got.Items[0].ID != 0 || got.Items[1].ID != 16 {
		t.Fatalf("IDs %v, items %d then %d", got.IDs, got.Items[0].ID, got.Items[1].ID)
	}
}
