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
