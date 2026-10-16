package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// flightStep runs one world step around the record memories the map world
// keeps: the pre-move points before it and the trails and looks after it.
func flightStep(mw *mapWorld) {
	mw.noteShotPreMoves()
	sim.StepReported(mw.world, nil)
	mw.advanceShotTrails()
}

// flightRecords is the World's armed records in id order.
func flightRecords(mw *mapWorld) []sim.SavedProjectile {
	records, _ := mw.armedRecords()
	slices.SortFunc(records, func(a, b sim.SavedProjectile) int { return int(a.ID) - int(b.ID) })
	return records
}

// A cast aimed at a cell is drawn along its line of flight: the record's dir
// is the sixteen-way direction from the launch cell to the aim cell, not the
// caster's eight-way facing, so a Fire Ball off the compass lines is not
// turned away from where it flies.
func TestCastRecordDirFollowsTheLineOfFlight(t *testing.T) {
	for _, d := range []image.Point{{5, 0}, {5, 1}, {-5, 2}, {5, -2}, {3, -4}, {-4, -3}} {
		mw := aoWorld(t)
		from := image.Pt(6, 6)
		to := from.Add(d)
		mw.releaseCast(sim.CastEvent{Spell: 2, AtCell: true, Facing: 32,
			FromX: int32(from.X), FromY: int32(from.Y), ToX: int32(to.X), ToY: int32(to.Y)}, 0)
		items := mw.world.SavedProjectiles().Items
		if len(items) != 1 {
			t.Fatalf("delta %v: %d records, want 1", d, len(items))
		}
		want := sim.ProjectileDirection(int32(d.X), int32(d.Y))
		if got := items[0]; got.Dir != want || got.ActionDir != want {
			t.Errorf("delta %v: dir %d actiondir %d, want the flight's direction %d", d, got.Dir, got.ActionDir, want)
		}
		if got := savedProjectileFacing(items[0].Dir); got != EffectFacing(d.X, d.Y) {
			t.Errorf("delta %v: drawn facing %d, flight facing %d", d, got, EffectFacing(d.X, d.Y))
		}
	}
}

func TestCastAdapterActorZeroTargetUsesItsRuntimeKey(t *testing.T) {
	for _, bound := range []bool{false, true} {
		for _, atCell := range []bool{false, true} {
			target := sim.Entity{ID: 0, X: 5, Y: 4, HP: 100, MaxHP: 100}
			if bound {
				target.SourceBinding = sim.SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 101, RuntimeID: 91}
				target.ActorLoad = sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1}}
			}
			w, err := sim.NewWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
				[]sim.Entity{target, {ID: 1, X: 1, Y: 4, HP: 100, MaxHP: 100}})
			if err != nil {
				t.Fatal(err)
			}
			mw := sbWorld(t)
			mw.world, mw.projectiles.Homing = w, map[int]bool{10: true}
			mw.releaseCast(sim.CastEvent{Caster: 1, Target: 0, AtCell: atCell, Spell: 1, FromX: 1, FromY: 4, ToX: 5, ToY: 4}, 0)
			items := w.SavedProjectiles().Items
			if len(items) != 1 || w.SavedProjectiles().FreeIndex != 1 {
				t.Fatalf("bound %v cell %v: record or allocator lost: %+v", bound, atCell, w.SavedProjectiles())
			}
			wantKey := int32(0)
			if bound && !atCell {
				wantKey = 91
			}
			if p := items[0]; p.ActionTarget != wantKey || p.ActionX != 5*256+128 || p.ActionY != 4*256+128 {
				t.Fatalf("bound %v cell %v: record %+v, want target key %d", bound, atCell, p, wantKey)
			}
			if d := w.SavedWorldEffectDrivers().Projectiles[0]; d.HasTarget != (bound && !atCell) || d.Target != 0 {
				t.Fatalf("bound %v cell %v: driver %+v", bound, atCell, d)
			}
			if err := w.HeadlessPlace(0, 7, 4); err != nil {
				t.Fatal(err)
			}
			wantX, wantY := int32(5*256+128), int32(4*256+128)
			if bound && !atCell {
				wantX, wantY, _ = w.ProjectilePoint(0)
				if wantX == items[0].ActionX && wantY == items[0].ActionY {
					t.Fatal("headless placement did not move the target point")
				}
			}
			flightStep(mw)
			if p := w.SavedProjectiles().Items[0]; p.ActionX != wantX || p.ActionY != wantY || p.ActionTarget != wantKey {
				t.Fatalf("bound %v cell %v: moved-target record %+v, want aim %d,%d key %d", bound, atCell, p, wantX, wantY, wantKey)
			}
		}
	}
}
