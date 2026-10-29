package sim

import "testing"

// darkWorldSight is fogWalkSight for either game: the same world, the second
// game's script state present or absent.
func darkWorldSight(t *testing.T, second bool, owner uint32, es ...Entity) []byte {
	t.Helper()
	w, err := NewTerrainWorld(1, sightBounds, ModeCanonical, Terrain{}, es, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	return w.Sight(owner)
}

// darkHuman is a viewer's Human whose whole-cell sight has fallen to 0: the
// word is the fraction byte its Mind and Reaction give.
func darkHuman(mind, reaction int32) Entity {
	e := fogWalkHero(22, 22, 1)
	e.Mind, e.Reaction, e.ScanRange = mind, reaction, 0
	return e
}

// The fog stamp's gate is the 16-bit sight word (TERR-TILE-079, TERR-FOG-275,
// HERO-MOD-016), in both games. A body whose whole-cell sight is 0 and whose
// word is nonzero stamps; a word of 0 stamps nothing.
func TestAWordNotAWholeCellOpensTheFog(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		// Mind 25 and Reaction 36 give a fraction of 112/256 at ScanRange 0.
		hero := darkHuman(25, 36)
		if w := hero.SightWord(true); w != 112 {
			t.Fatalf("sight word %d, want 112", w)
		}
		got := darkWorldSight(t, second, 1, hero)
		want := claimWalkStamp(nil, sightBounds, 22, 22, 112, 1)
		stampDiff(t, "darkened Human", got, want)
		if litCount(got) != 4 {
			t.Errorf("second game %v: a darkened Human lights %d cells, want his 2x2 block", second, litCount(got))
		}

		// Mind+Reaction a multiple of 25 leaves a word of 0.
		zero := darkHuman(25, 25)
		if w := zero.SightWord(true); w != 0 {
			t.Fatalf("sight word %d, want 0", w)
		}
		if n := litCount(darkWorldSight(t, second, 1, zero)); n != 0 {
			t.Errorf("second game %v: a body with word 0 lights %d cells", second, n)
		}

		// A loaded body keeps its saved low byte under a high byte of 0.
		loaded := fogWalkHero(22, 22, 1)
		loaded.ScanRange = 0
		loaded.NativeBasis.ScalarsPresent = true
		loaded.NativeBasis.ScalarKnown = 1 << ScalarUA4
		loaded.NativeBasis.Scalars[ScalarUA4] = 0x80
		if w := loaded.SightWord(true); w != 0x80 {
			t.Fatalf("loaded sight word %d, want 128", w)
		}
		stampDiff(t, "loaded body", darkWorldSight(t, second, 1, loaded),
			claimWalkStamp(nil, sightBounds, 22, 22, 0x80, 1))
		loaded.NativeBasis.Scalars[ScalarUA4] = 0
		if n := litCount(darkWorldSight(t, second, 1, loaded)); n != 0 {
			t.Errorf("second game %v: a loaded body with word 0 lights %d cells", second, n)
		}
	}
}
