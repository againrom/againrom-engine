package game

import (
	"fmt"
	"image"
	"path/filepath"
	"testing"
)

// MAGIC-267: SAV holds the current point, not a launch point.
func TestAProjectileInFlightKeepsItsCurrentPointAndLightAcrossSAV(t *testing.T) {
	f := restoredFacingFront(t)
	app, path := openOriginalSAVApp(t, f, restoredFacingSource(t, f), "flight.sav")
	start := map[uint16]image.Point{}
	for _, p := range f.live.world.SavedProjectiles().Items {
		start[p.ID] = image.Pt(int(p.X), int(p.Y))
	}
	f.live.tick()
	f.live.tick()
	state := func(front *FrontEnd) (string, string) {
		var points string
		for _, p := range front.live.world.SavedProjectiles().Items {
			points += fmt.Sprintf("%d:%d,%d;", p.ID, p.X, p.Y)
		}
		return points, fmt.Sprint(front.live.objectLightStamps(nil))
	}
	moved := 0
	stamps := lightGrid(f.live.objectLightStamps(nil))
	for _, p := range f.live.world.SavedProjectiles().Items {
		if image.Pt(int(p.X), int(p.Y)) != start[p.ID] {
			moved++
		}
		if p.Picture != restoredBoltPicture {
			continue
		}
		cell := image.Pt(int(p.X)/256, int(p.Y)/256)
		if stamps[cell] != flightLightLevel {
			t.Errorf("Fire Arrow %d at %v leaves vertex %v at %d, want %d", p.ID, cell, cell, stamps[cell], flightLightLevel)
		}
	}
	if moved == 0 {
		t.Fatal("no restored projectile moved; the SAV cannot tell a current point from a launch point")
	}
	wantPoints, wantLight := state(f)
	_, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	fresh := restoredFacingColdLoad(t, written, name)
	gotPoints, gotLight := state(fresh)
	if gotPoints != wantPoints {
		t.Fatalf("LOAD of the SAVE holds points\n%s\nwant\n%s", gotPoints, wantPoints)
	}
	if gotLight != wantLight {
		t.Fatalf("LOAD of the SAVE stamps different light")
	}
}
