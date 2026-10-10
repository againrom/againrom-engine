package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// SAV-1188: the release offset is 8 × (ShootOffset pair − Center), the pair
// ((dir − 8) & 14) / 2 for the shooter's sixteen-way dir. The unit shot and the
// cast producer read it through one helper.
func TestUnitShotStartPairFollowsTheSixteenWayDir(t *testing.T) {
	c := &terrain.UnitClass{CenterX: 8, CenterY: 14,
		ShootOffset: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
	// The pair each sixteen-way dir 0..15 selects, written out: S (8) and the
	// facing after it use pair 0, and the wheel runs on from there.
	pairOf := [16]int{4, 4, 5, 5, 6, 6, 7, 7, 0, 0, 1, 1, 2, 2, 3, 3}
	mw := &mapWorld{units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: c}}}
	for dir := range 16 {
		p := pairOf[dir]
		want := image.Pt(8*(c.ShootOffset[2*p]-8), 8*(c.ShootOffset[2*p+1]-14))
		for _, low := range []uint8{0, 15} {
			facing := uint8(dir<<4) | low
			got, ok := classShootOffset(c, facing)
			if !ok || got != want {
				t.Errorf("dir %d (facing byte %d): offset %v, want %v (pair %d)", dir, facing, got, want, p)
			}
			if dx, dy := mw.shotOffset(1, facing); image.Pt(dx, dy) != want {
				t.Errorf("dir %d: unit shot offset %d,%d, want %v", dir, dx, dy, want)
			}
		}
	}
	none := &terrain.UnitClass{CenterX: 8, CenterY: 14}
	mw.units.Classes[2] = none
	if off, ok := classShootOffset(none, 128); ok || off != (image.Point{}) {
		t.Errorf("a class with no array gives %v %v", off, ok)
	}
	if dx, dy := mw.shotOffset(2, 128); dx != 0 || dy != 0 {
		t.Errorf("a class with no array releases %d,%d from the shooter's point", dx, dy)
	}
}

// ANIM-140, SAV-1193: a record of a smoke-leaving picture holds no trail when
// it is loaded; each driver call appends the point it started from, oldest
// first, at most six, the oldest dropped.
func TestASmokeTrailIsThePreMovePointsOldestFirst(t *testing.T) {
	units, set := shotArchive(t)
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil,
		[]sim.Entity{shotVictim(2, 5, 4)}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	const picture = 10
	if data.CastTrailSlot(picture) < 0 {
		t.Fatal("picture 10 leaves no trail")
	}
	w.SetSavedProjectiles(sim.SavedProjectiles{FreeIndex: 1, IDs: []uint16{0}, Items: []sim.SavedProjectile{{
		ID: 0, X: 256, Y: 1024, Picture: picture, Action: 1, ActionX: 256 + 12*256, ActionY: 1024, ActionSegments: 12,
	}}})
	if err := w.ImportOriginalWorldEffectDrivers(&sim.SavedWorldEffects{Projectiles: []sim.SavedProjectileDriver{{ID: 0, Phases: 1}}}); err != nil {
		t.Fatal(err)
	}
	mw := shotMapWorld(t, w, units, set)
	if len(mw.shots.trail[0]) != 0 {
		t.Fatalf("a loaded record starts with trail %v", mw.shots.trail[0])
	}
	var starts []image.Point
	for range 9 {
		p := mw.world.SavedProjectiles().Items[0]
		starts = append(starts, image.Pt(int(p.X), int(p.Y)))
		mw.tick()
		want := starts[max(len(starts)-data.CastTrailLength, 0):]
		if got := mw.shots.trail[0]; !slices.Equal(got, want) {
			t.Fatalf("after %d calls: trail %v, want %v", len(starts), got, want)
		}
	}
	if starts[1] == starts[0] {
		t.Fatal("the record did not move; the witness proves nothing")
	}
}
