package sim

import "testing"

// strikeGuard is slot 3's Group 1 of a missionStartWorld.
func strikeGuard(t *testing.T, w *World) SavedGroup {
	t.Helper()
	groups, _, _ := w.SavedGroups()
	for _, g := range groups {
		if g.Owner.Owner == leashSlot && g.Selector == 1 {
			return g
		}
	}
	t.Fatal("the guard Group left the registry")
	return SavedGroup{}
}

// strikeStart is a second-game mission start with one guard member at
// (20,20) and a hero that outlasts every fight here. It returns the World
// after the first group tick and the radius +0x2d that tick rolled.
func strikeStart(t *testing.T) (*World, int32) {
	t.Helper()
	w := missionStartWorld(t, []cell{{20, 20}})
	h := leashAt(w, leashHero)
	h.HP, h.MaxHP = 4000, 4000
	leashRun(w, 1)
	return w, int32(strikeGuard(t, w).AI[0x2d])
}

// strikePursuit leads the member after the hero for three group ticks with
// the hero three cells ahead, beyond reach and inside the circle, then
// carries the hero beyond the leash. It returns on the group tick the leash
// ended the pursuit: the member's last tick in pursuit of a target beyond
// reach.
func strikePursuit(t *testing.T) (*World, int32) {
	t.Helper()
	w, r := strikeStart(t)
	for n := range 3 {
		m := leashAt(w, 2)
		leashMove(t, w, leashHero, m.X+3, m.Y)
		leashRun(w, 1)
		if v, ok := engVictim(w, 2); !ok || v != leashHero {
			t.Fatalf("pursuit tick %d: the member holds victim %v/%v, want the hero", n, v, ok)
		}
	}
	m := leashAt(w, 2)
	x := min(m.X+r+5, engBounds.Width-1)
	if x-m.X <= r {
		t.Fatalf("fixture: the hero cannot stand beyond the leash %d", r)
	}
	leashMove(t, w, leashHero, x, m.Y)
	leashRun(w, 1)
	if v, ok := engVictim(w, 2); ok {
		t.Fatalf("the member still chases %v with the hero beyond the leash", v)
	}
	if m := leashAt(w, 2); m.X <= 20 {
		t.Fatalf("fixture: the member at (%d,%d) never left its post", m.X, m.Y)
	}
	return w, r
}

// strikeFrom lands a blow on the member from dx cells east of it: the member
// loses three quarters of its health and the blow's record is written. It
// returns the cell ten cells beyond the member on the line from the striker.
func strikeFrom(t *testing.T, w *World, dx int32) cell {
	t.Helper()
	m := leashAt(w, 2)
	at := cell{m.X, m.Y}
	leashMove(t, w, leashHero, at.x+dx, at.y)
	m = leashAt(w, 2)
	m.HP = m.MaxHP / 4
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
	return cell{at.x - 10, at.y}
}

// walksTo reports whether the member walks to c or stands on it.
func walksTo(w *World, c cell) bool {
	m := leashAt(w, 2)
	return m.HasTarget && m.TargetX == c.x && m.TargetY == c.y || !m.HasTarget && m.X == c.x && m.Y == c.y
}

// The owner's case at a mission start: a guard B member led beyond its leash
// walks home; struck on the way from inside its circle, it walks to cmd
// +0x9c and takes no target while |cmd +0x16 - server word +0| < 30 (hurt),
// the stamp of its last group tick in pursuit beyond reach; on the 30th
// group tick it takes the striker (R2-ENGINE-844 cases 1 to 3,
// R2-ENGINE-845, R2-ENGINE-846).
func TestSecondGuardStruckOnTheWayHomeWalksAwayInsideTheStampWindow(t *testing.T) {
	w, r := strikePursuit(t)
	if !walksTo(w, cell{20, 20}) {
		m := leashAt(w, 2)
		t.Fatalf("the member at (%d,%d) walks to (%d,%d)/%v, want its post (20,20)", m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget)
	}
	away := strikeFrom(t, w, 2)
	for n := 1; n < 30; n++ {
		m := leashAt(w, 2)
		leashMove(t, w, leashHero, m.X+2, m.Y)
		leashRun(w, 1)
		if v, ok := engVictim(w, 2); ok {
			t.Fatalf("group tick %d after the stamp: the member took victim %v inside the window", n, v)
		}
		if !walksTo(w, away) {
			m := leashAt(w, 2)
			t.Fatalf("group tick %d after the stamp: the member at (%d,%d) walks to (%d,%d)/%v, want +0x9c %v", n, m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget, away)
		}
		if m := leashAt(w, 2); m.HP == m.MaxHP {
			t.Fatal("fixture: the member healed inside the window")
		}
	}
	m := leashAt(w, 2)
	if 2 > r {
		t.Fatalf("fixture: the striker two cells off stands beyond the leash %d", r)
	}
	leashMove(t, w, leashHero, m.X+2, m.Y)
	leashRun(w, 1)
	if v, ok := engVictim(w, 2); !ok || v != leashHero {
		t.Fatalf("on the 30th group tick after the stamp the member holds victim %v/%v, want the striker", v, ok)
	}
}

// Case 4: a hurt member with no target walks to cmd +0x9c while the fine
// clock is within 10 of its blow's +0x98; struck from beyond its circle
// longer ago than that, it walks on to its post (R2-ENGINE-844 cases 4 and
// 6, R2-ENGINE-845).
func TestSecondGuardHurtMemberWalksAwayWithinTenFineTicksOfItsBlow(t *testing.T) {
	for _, c := range []struct {
		name string
		wait int
		away bool
	}{{"blow five fine ticks before the group tick", 10, true}, {"blow fifteen fine ticks before the group tick", 0, false}} {
		t.Run(c.name, func(t *testing.T) {
			w, r := strikeStart(t)
			// Carried off its post with no pursuit, the member walks home;
			// the hero beyond the leash keeps the Group active.
			leashMove(t, w, leashHero, min(26+r+5, engBounds.Width-1), 20)
			leashMove(t, w, 2, 26, 20)
			leashRun(w, 1)
			if !walksTo(w, cell{20, 20}) {
				t.Fatal("fixture: the member does not walk home")
			}
			for range c.wait {
				Step(w, nil)
			}
			m := leashAt(w, 2)
			dx := min(r+5, engBounds.Width-1-m.X)
			if dx <= r {
				t.Fatalf("fixture: the striker cannot stand beyond the leash %d", r)
			}
			away := strikeFrom(t, w, dx)
			leashRun(w, 1)
			if v, ok := engVictim(w, 2); ok {
				t.Fatalf("the member took victim %v from beyond its circle", v)
			}
			want := cell{20, 20}
			if c.away {
				want = away
			}
			if !walksTo(w, want) {
				m := leashAt(w, 2)
				t.Fatalf("the member at (%d,%d) walks to (%d,%d)/%v, want %v", m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget, want)
			}
		})
	}
}

// A byte-form round trip inside the stamp window keeps the window, the
// blow's record and each next group tick's decision.
func TestSecondGuardStampWindowSurvivesAColdLoad(t *testing.T) {
	w, _ := strikePursuit(t)
	strikeFrom(t, w, 2)
	leashRun(w, 1)
	cold := cloneWorld(t, w)
	for n := 2; n <= 30; n++ {
		for _, x := range []*World{w, cold} {
			m := leashAt(x, 2)
			leashMove(t, x, leashHero, m.X+2, m.Y)
			leashRun(x, 1)
		}
		if w.Hash() != cold.Hash() {
			t.Fatalf("group tick %d after the stamp: the cold World departs from the live one", n)
		}
		_, held := engVictim(cold, 2)
		if n < 30 && held || n == 30 && !held {
			t.Fatalf("group tick %d after the stamp: the cold World's member holds a victim %v", n, held)
		}
	}
}
