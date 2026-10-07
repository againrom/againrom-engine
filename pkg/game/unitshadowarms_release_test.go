package game

import (
	"sort"
	"testing"

	"againrom/pkg/data"
)

// TestReleaseUnitShadowSecondSilhouetteSheetsPairWithTheInstalledArt loads the
// installed unit classes and hero bodies and requires every frame of a class
// or body whose spritesb sibling pairs with it to answer a paired frame. It
// reports the classes the sibling does not pair with, and the classes the
// translated arm draws.
func TestReleaseUnitShadowSecondSilhouetteSheetsPairWithTheInstalledArt(t *testing.T) {
	f := releaseFront(t)
	units, err := LoadUnits(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	var paired, unpaired, flat []int
	pairs := 0
	for id, c := range units.Classes {
		if len(c.Frames) == 0 {
			continue
		}
		if c.Z != 0 {
			flat = append(flat, int(id))
		}
		if len(c.Boundary) == 0 {
			unpaired = append(unpaired, int(id))
			continue
		}
		paired = append(paired, int(id))
		for i, fr := range c.Frames {
			b := c.BoundaryOf(fr)
			if b == nil {
				t.Fatalf("class %d frame %d has no paired frame", id, i)
			}
			if b.Width <= 0 && b.Height <= 0 {
				continue
			}
			pairs++
		}
		for k, tier := range c.Tiers {
			if len(tier) != len(c.Frames) {
				continue
			}
			for i, fr := range tier {
				if c.BoundaryOf(fr) == nil {
					t.Fatalf("class %d tier %d frame %d has no paired frame", id, k+1, i)
				}
			}
		}
	}
	sort.Ints(paired)
	sort.Ints(unpaired)
	sort.Ints(flat)
	if len(paired) == 0 {
		t.Fatal("no installed unit class pairs with its spritesb sibling")
	}
	heroes := 0
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		for _, body := range []data.HeroBody{data.BodySwordsman, data.BodyAxeman, data.BodyClubman, data.BodyArcher, data.BodyMage} {
			LoadHeroBody(f.Archives.Containers, units, dir, body)
			live := units.Bodies[data.HeroBodyKey(dir, body)]
			if live == nil {
				continue
			}
			for i, fr := range live.Frames {
				if live.BoundaryOf(fr) == nil {
					t.Fatalf("%s/%s frame %d has no paired frame", dir, body, i)
				}
			}
			heroes++
		}
	}
	if heroes == 0 {
		t.Fatal("no installed hero body resolved")
	}
	if len(paired) != 34 || len(unpaired) != 0 {
		t.Errorf("paired %d classes and left %d unpaired, want all 34 paired", len(paired), len(unpaired))
	}
	if len(flat) != 2 || flat[0] != 70 || flat[1] != 71 {
		t.Errorf("translated-arm classes %v, want exactly 70 and 71", flat)
	}
	t.Logf("paired classes %v; classes without a pairing %v; translated-arm classes %v; %d paired frames, %d hero bodies", paired, unpaired, flat, pairs, heroes)
}
