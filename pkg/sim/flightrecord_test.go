package sim

import "testing"

// A shot at a structure is an ordinary unit-shot record whose target key is
// the structure's id and whose aim is its anchor cell's centre (SAV-1197).
func TestReleaseUnitShotAtAStructureHomesOnItsAnchor(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	w.DeclareStructures([]Structure{{ID: 40, Col: 6, Row: 4, Width: 1, Height: 1, MaxHealth: 50}})
	w.entities[0].AttackTarget, w.entities[0].AttackTargetKind = 40, AttackTargetStructure
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 13, Phases: 1}) {
		t.Fatal("no record built for a structure target")
	}
	p, ok := savedProjectileOf(w, 0)
	if !ok || p.ActionTarget != 40 || p.ActionX != 6*256+128 || p.ActionY != 4*256+128 || p.Picture != 13 {
		t.Fatalf("record %+v, want picture 13 keyed 40 aimed at the centre of (6,4)", p)
	}
	d := w.SavedWorldEffectDrivers().Projectiles[0]
	if !d.TargetStructure || !d.HasTarget || d.Target != 40 {
		t.Fatalf("driver %+v, want a structure target 40", d)
	}
	for p.ActionSegments > 0 {
		Step(w, nil)
		p, _ = savedProjectileOf(w, 0)
	}
	if p.X != p.ActionX || p.Y != p.ActionY {
		t.Fatalf("the last segment ends at %d,%d, not on the anchor", p.X, p.Y)
	}
	// The driver row's structure flag survives the World's binary form.
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	if got := cold.SavedWorldEffectDrivers().Projectiles[0]; got != d {
		t.Fatalf("restored driver %+v, want %+v", got, d)
	}
}

// A cast record starts where the caller says, aims at the cell centre for a
// non-homing picture, takes its id from the shared counter and runs its first
// call (ANIM-147). A 0-segment record retires on that call (SAV-1204).
func TestReleaseCastBuildsTheRecordAndRunsItsFirstCall(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	id, ok := w.ReleaseCast(CastRecord{Caster: 1, Picture: 10, Phases: 4, X: 384, Y: 1152,
		AimX: 5*256 + 128, AimY: 4*256 + 128, Dir: 4, Segments: 5})
	if !ok || id != 0 {
		t.Fatalf("release answered %d %v", id, ok)
	}
	p, _ := savedProjectileOf(w, 0)
	if p.ActionPhase != 1 || p.ActionSegments != 4 || p.ActionTarget != 0 || p.Dir != 4 || p.ActionDir != 4 ||
		p.X != 384+(5*256+128-384)/5 || p.Y != 1152+(4*256+128-1152)/5 {
		t.Fatalf("record %+v after its first call", p)
	}
	if id, ok := w.ReleaseCast(CastRecord{Caster: 1, Picture: 22, Segments: 0}); !ok || id != 1 {
		t.Fatalf("a 0-segment cast answered %d %v", id, ok)
	}
	if got := w.SavedProjectiles(); got.FreeIndex != 2 || len(got.Items) != 1 {
		t.Fatalf("a 0-segment record outlived its call: %+v", got)
	}
}

// A homing cast aims at its target actor's point and carries its key; a
// homing target the World does not hold builds nothing (MAGIC-288).
func TestReleaseCastHomesOnItsTarget(t *testing.T) {
	w := shotWorld(t, shotPair(4)...)
	if _, ok := w.ReleaseCast(CastRecord{Caster: 1, Picture: 10, Target: 9, HasTarget: true, Segments: 5}); ok {
		t.Fatal("an absent homing target built a record")
	}
	if _, ok := w.ReleaseCast(CastRecord{Caster: 1, Picture: 10, Target: 2, HasTarget: true, X: 384, Y: 1152, Segments: 5}); !ok {
		t.Fatal("no record built")
	}
	p, _ := savedProjectileOf(w, 0)
	if p.ActionTarget != 2 || p.ActionX != 5*256+128 || p.ActionY != 4*256+128 {
		t.Fatalf("record %+v, want it homing on actor 2", p)
	}
	if d := w.SavedWorldEffectDrivers().Projectiles[0]; !d.HasTarget || d.Target != 2 || d.TargetStructure {
		t.Fatalf("driver %+v", d)
	}
}

// A client-arm record starts at actionphase -1 and keeps dir 0 (ANIM-144); a
// staged burst stands at its cell centre from actionphase -1 (ANIM-148).
func TestClientAndBurstRecordsStartAtActionPhaseMinusOne(t *testing.T) {
	w := shotWorld(t)
	w.ReleaseCast(CastRecord{Picture: 34, X: 640, Y: 640, AimX: 1664, AimY: 640, Dir: 5, Client: true, Segments: 5})
	p, _ := savedProjectileOf(w, 0)
	if p.ActionPhase != 0 || p.Dir != 0 || p.ActionDir != 0 || p.Phase != 0 || p.X != 640 {
		t.Fatalf("client record %+v", p)
	}
	w.ReleaseAreaBurst(AreaBurst{CellX: 3, CellY: 2, Picture: 17, Segments: 16, Phases: 8})
	b, _ := savedProjectileOf(w, 1)
	if b.X != 3*256+128 || b.Y != 2*256+128 || b.ActionPhase != 0 || b.ActionSegments != 15 || b.ActionTarget != 0 {
		t.Fatalf("burst record %+v", b)
	}
}

// ANIM-149: the Teleport picture's arm writes no position.
func TestTheTeleportRecordStandsStill(t *testing.T) {
	w := shotWorld(t)
	w.ReleaseCast(CastRecord{Picture: teleportProjectilePicture, X: 300, Y: 400, AimX: 3000, AimY: 400, Segments: 21})
	for range 20 {
		p, _ := savedProjectileOf(w, 0)
		if p.X != 300 || p.Y != 400 {
			t.Fatalf("the record moved to %d,%d", p.X, p.Y)
		}
		Step(w, nil)
	}
}
