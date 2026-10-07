package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// TestReleaseUnitShadowHeroPairFollowsTheHeroTypeID requires a hero body and
// its paired sheet for every non-party actor whose type id is in the hero
// range (UNIT-140) and a class sheet for every other one.
func TestReleaseUnitShadowHeroPairFollowsTheHeroTypeID(t *testing.T) {
	f := releaseFront(t)
	units, err := LoadUnits(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	hero, other := 0, 0
	for n := 1; n <= 160; n++ {
		addr, ok := MissionMap(n)
		if !ok {
			continue
		}
		raw, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			continue
		}
		m, err := alm.Open(raw)
		if err != nil {
			t.Fatalf("decode %s: %v", addr, err)
		}
		ms, err := StartMissionFrom(m, addr, n, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
		if err != nil {
			t.Fatalf("mission %d: %v", n, err)
		}
		art := missionAppearanceArt(ms, units, f.Bodies, f.Archives.Containers)
		party := map[sim.EntityID]bool{}
		for _, id := range ms.Start.IDs {
			party[id] = true
		}
		bodies := map[*terrain.UnitClass]bool{}
		for _, b := range units.Bodies {
			bodies[b] = true
		}
		for _, e := range ms.World.Entities() {
			if party[e.ID] {
				continue
			}
			a := art[e.ID]
			isBody := a != nil && bodies[a]
			if data.FigureIsHero(e.TypeID) {
				hero++
				if !isBody {
					t.Errorf("mission %d entity %d type %#x is in the hero range and draws no hero body", n, e.ID, e.TypeID)
					continue
				}
				if len(a.Boundary) != len(a.Frames) || len(a.Frames) == 0 {
					t.Errorf("mission %d entity %d: hero body has %d frames and %d paired frames", n, e.ID, len(a.Frames), len(a.Boundary))
				}
				continue
			}
			if isBody {
				t.Errorf("mission %d entity %d type %#x draws a hero body outside the hero range", n, e.ID, e.TypeID)
			}
			other++
		}
	}
	if hero == 0 || other == 0 {
		t.Fatalf("hero-range placements %d, others %d", hero, other)
	}
	t.Logf("%d placed actors in the hero range draw the hero pair; %d others draw their class sheets", hero, other)
}

// opaqueAt reports an opaque pixel of f at (x, y), mirrored inside its own box.
func opaqueAt(f *terrain.StaticFrame, x, y int, mirror bool) bool {
	if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
		return false
	}
	if mirror {
		x = f.Width - 1 - x
	}
	return f.Pixels[y*f.Width+x].Opaque
}

// pairOverlap counts the pixels both silhouettes stamp, b at (dx, dy) from a.
func pairOverlap(a, b *terrain.StaticFrame, dx, dy int, mirror bool) int {
	n := 0
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if opaqueAt(b, x, y, mirror) && opaqueAt(a, x+dx, y+dy, mirror) {
				n++
			}
		}
	}
	return n
}

// TestReleaseUnitShadowTranslatedPairNeverOverlapsOnTheInstalledArt requires
// that no pixel is stamped by both translated silhouettes of the Z classes
// (TERR-195); a one-pixel shift is the control.
func TestReleaseUnitShadowTranslatedPairNeverOverlapsOnTheInstalledArt(t *testing.T) {
	f := releaseFront(t)
	units, err := LoadUnits(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	classes, frames, unequal, control := 0, 0, 0, 0
	for id, c := range units.Classes {
		if c.Z == 0 || len(c.Frames) == 0 {
			continue
		}
		classes++
		for i, fr := range c.Frames {
			b := c.BoundaryOf(fr)
			if b == nil || fr.Width <= 0 || b.Width <= 0 {
				continue
			}
			own, ok1 := terrain.UnitPlace(10, 10, c, fr, false, 0, 0)
			pair, ok2 := terrain.UnitPlace(10, 10, c, b, false, 0, 0)
			if !ok1 || !ok2 {
				t.Fatalf("class %d frame %d does not place", id, i)
			}
			dx, dy := pair.TopLeft.X-own.TopLeft.X, pair.TopLeft.Y-own.TopLeft.Y
			if fr.Width != b.Width || fr.Height != b.Height {
				unequal++
			}
			frames++
			for _, mirror := range []bool{false, true} {
				if got := pairOverlap(fr, b, dx, dy, mirror); got != 0 {
					t.Errorf("class %d frame %d mirror %v: %d pixels stamped twice", id, i, mirror, got)
				}
			}
			control += pairOverlap(fr, b, dx-1, dy, false) + pairOverlap(fr, b, dx+1, dy, false)
		}
	}
	if classes != 2 || frames == 0 {
		t.Fatalf("%d translated-arm classes, %d frames; want 2 classes", classes, frames)
	}
	if control == 0 {
		t.Fatal("a one-pixel shift overlaps nothing: the probe cannot see overlap")
	}
	t.Logf("%d frames of %d classes (%d with unequal sheet sizes): no overlap; the one-pixel shift overlaps %d pixels", frames, classes, unequal, control)
}
