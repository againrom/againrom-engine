package sim

import "testing"

// bodyNearestCells is the smallest Chebyshev distance between any cell of the
// square body anchored at (ax, ay) with side as and any cell of the one anchored
// at (tx, ty) with side ts. Two bodies that touch measure 1. It is built by
// listing cells, independent of the production distance expression.
func bodyNearestCells(ax, ay, as, tx, ty, ts int32) int32 {
	best := int32(1 << 30)
	for i := range as {
		for j := range as {
			for k := range ts {
				for l := range ts {
					dx, dy := ax+i-(tx+k), ay+j-(ty+l)
					if dx < 0 {
						dx = -dx
					}
					if dy < 0 {
						dy = -dy
					}
					best = min(best, max(dx, dy))
				}
			}
		}
	}
	return best
}

// TestStrikeDistanceIsTheGapBetweenTheTwoBodies pins the strike distance to the
// physical gap between two square bodies for every pair of sides up to 3 and
// every offset from -6 to 6 that does not overlap them: touching bodies measure
// 1 on all four sides and each further cell of gap adds 1. A body wider than
// one cell used to reach a unit one cell past its own edge on its top and left
// sides only.
func TestStrikeDistanceIsTheGapBetweenTheTwoBodies(t *testing.T) {
	for as := int32(1); as <= 3; as++ {
		for ts := int32(1); ts <= 3; ts++ {
			for dx := int32(-6); dx <= 6; dx++ {
				for dy := int32(-6); dy <= 6; dy++ {
					want := bodyNearestCells(8, 8, as, 8+dx, 8+dy, ts)
					if want == 0 {
						continue
					}
					a, target := cbEnt(1, 8, 8), cbEnt(2, 8+dx, 8+dy)
					a.TokenSize, target.TokenSize = uint8(as), uint8(ts)
					if got := strikeDistance(a, target); got != want {
						t.Errorf("sides %d and %d at offset (%d,%d): strike distance %d, bodies are %d cells apart", as, ts, dx, dy, got, want)
					}
					if back := strikeDistance(target, a); back != want {
						t.Errorf("sides %d and %d at offset (%d,%d): reversed strike distance %d, want %d", ts, as, dx, dy, back, want)
					}
				}
			}
		}
	}
}

// TestALargeBodyStrikesOnlyWhatItsBodyTouches runs the Ogre-sized body against a
// one-cell unit that stands still, from every start in sight, and records where
// each one stood at the first blow it landed. The Ogre's blow lands only with the
// bodies touching, and the participant's unit answers a blow that lands.
func TestALargeBodyStrikesOnlyWhatItsBodyTouches(t *testing.T) {
	for dy := int32(-4); dy <= 5; dy++ {
		for dx := int32(-4); dx <= 5; dx++ {
			if dx >= 0 && dx <= 1 && dy >= 0 && dy <= 1 {
				continue
			}
			ogre := engFighter(1, 2, 8, 8)
			ogre.TokenSize, ogre.Reach = 2, 1
			unit := engFighter(2, SelfSlot, 8+dx, 8+dy)
			unit.TokenSize, unit.Reach = 1, 1
			w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 1}), ogre, unit)
			unitHP, ogreHP := cbAt(t, w, 2).HP, cbAt(t, w, 1).HP
			firstOgre, firstUnit := -1, -1
			for tick := range 300 {
				Step(w, nil)
				o, u := cbAt(t, w, 1), cbAt(t, w, 2)
				if firstOgre < 0 && u.HP < unitHP {
					firstOgre = tick
					if gap := bodyNearestCells(o.X, o.Y, 2, u.X, u.Y, 1); gap != 1 {
						t.Errorf("start (%d,%d): the Ogre at (%d,%d) landed a blow on the unit at (%d,%d), %d cells from its body", dx, dy, o.X, o.Y, u.X, u.Y, gap)
					}
				}
				if firstUnit < 0 && o.HP < ogreHP {
					firstUnit = tick
					if gap := bodyNearestCells(o.X, o.Y, 2, u.X, u.Y, 1); gap != 1 {
						t.Errorf("start (%d,%d): the unit at (%d,%d) landed a blow on the Ogre at (%d,%d), %d cells from its body", dx, dy, u.X, u.Y, o.X, o.Y, gap)
					}
				}
			}
			if firstOgre >= 0 && (firstUnit < 0 || firstUnit-firstOgre > 40) {
				t.Errorf("start (%d,%d): the Ogre's first blow landed on tick %d and the unit's answer on tick %d", dx, dy, firstOgre, firstUnit)
			}
			if firstOgre < 0 && firstUnit < 0 && dx > -4 && dx < 4 && dy > -4 && dy < 4 {
				t.Errorf("start (%d,%d): no blow landed in 300 ticks", dx, dy)
			}
		}
	}
}
