package sim

import "testing"

func TestGuardStruckMemberTakesTheStrikerOnTheSameTickInBothGames(t *testing.T) {
	for _, state := range []string{"post", "attacking"} {
		t.Run(state, func(t *testing.T) {
			var worlds [2]*World
			for game := range worlds {
				w := gameProfileFixture(t, game == 1)
				worlds[game] = w
				leashAt(w, guardRouteHero).HP, leashAt(w, guardRouteHero).MaxHP = 4000, 4000
				leashAt(w, guardRouteFoe).HP, leashAt(w, guardRouteFoe).MaxHP = 400, 400
				leashMove(t, w, guardRouteHero, 5, 5)
				leashRun(w, 1)
				m := leashAt(w, guardRouteFoe)
				if m.X != m.PostX || m.Y != m.PostY || m.HasAttackTarget {
					t.Fatalf("game %d: fixture is not at its post", game+1)
				}
				leashMove(t, w, guardRouteHero, m.X-1, m.Y)
				if state == "attacking" {
					leashRun(w, 1)
					if v, held := engVictim(w, guardRouteFoe); !held || v != guardRouteHero {
						t.Fatalf("game %d: fixture is not attacking the hero", game+1)
					}
				}
				o := w.savedOrder(guardRouteFoe)
				o.Strike.Stamped, o.Strike.Stamp = true, w.groupClock()
				leashAt(w, guardRouteFoe).HP = 100
				w.flipOnBlow(indexOfEntity(w.entities, guardRouteHero), indexOfEntity(w.entities, guardRouteFoe))
				if game == 0 && o.Strike.Held {
					t.Fatal("first-game blow acquired a second-game strike record")
				}
			}
			var firstTick [2]uint64
			for range 96 {
				for game, w := range worlds {
					Step(w, nil)
					if v, held := engVictim(w, guardRouteFoe); held && v == guardRouteHero && firstTick[game] == 0 {
						firstTick[game] = w.tick
					}
					if w.ScriptPassJustRan() {
						if v, held := engVictim(w, guardRouteFoe); !held || v != guardRouteHero {
							t.Errorf("game %d tick %d: struck %s guard holds victim %d/%v, want hero %d", game+1, w.tick, state, v, held, guardRouteHero)
						}
					}
				}
			}
			if firstTick[0] == 0 || firstTick[1] != firstTick[0] {
				t.Errorf("first target tick: ROM1 %d, ROM2 %d; want the same nonzero tick", firstTick[0], firstTick[1])
			}
			t.Logf("first target tick: ROM1 %d, ROM2 %d", firstTick[0], firstTick[1])
		})
	}
}

func TestSecondGuardStruckMemberFightsBack(t *testing.T) {
	for _, state := range []string{"post", "post distant", "idle", "pursuing", "attacking", "edge"} {
		for _, stamped := range []bool{false, true} {
			name := state + "/fresh"
			if stamped {
				name = state + "/retained stamp"
			}
			t.Run(name, func(t *testing.T) {
				w, _ := strikeStart(t)
				m := leashAt(w, 2)
				if state == "edge" {
					leashMove(t, w, 2, 0, 20)
					m.PostX = 0
					w.savedOrder(2).Raw[0] = 0
				} else if state == "idle" {
					leashMove(t, w, 2, 22, 20)
				}
				m = leashAt(w, 2)
				dx := int32(1)
				if state == "pursuing" || state == "post distant" {
					dx = 5
				}
				leashMove(t, w, leashHero, m.X+dx, m.Y)
				if state == "pursuing" || state == "attacking" {
					leashRun(w, 1)
					if v, ok := engVictim(w, 2); !ok || v != leashHero {
						t.Fatalf("fixture: %s member has no hero target", state)
					}
					m = leashAt(w, 2)
					leashMove(t, w, leashHero, m.X+dx, m.Y)
				}
				o := w.savedOrder(2)
				o.Strike.Stamped, o.Strike.Stamp = stamped, w.groupClock()
				m.HP = m.MaxHP / 4
				startX, heroHP := m.X, leashAt(w, leashHero).HP
				w.flipOnBlow(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
				if !o.Strike.Held || o.Strike.Attacker != leashHero || o.Strike.Clock != w.fineClock() || o.Strike.Stamped != stamped || o.Raw[0x54] != 1 {
					t.Fatalf("the blow did not keep its current command fields: %+v", o)
				}
				for n := range 4 {
					leashRun(w, 1)
					m = leashAt(w, 2)
					if v, ok := engVictim(w, 2); !ok || v != leashHero {
						t.Fatalf("struck %s member, pass %d: victim %d/%v, at (%d,%d), walking %v to (%d,%d); want striker %d",
							state, n+1, v, ok, m.X, m.Y, m.HasTarget, m.TargetX, m.TargetY, leashHero)
					}
				}
				if dx == 1 && leashAt(w, leashHero).HP >= heroHP {
					t.Fatal("member took the striker but dealt no damage in reach")
				}
				if dx > 1 && m.X <= startX {
					t.Fatal("member took the distant striker but did not advance toward it")
				}
			})
		}
	}
}

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

// A returning guard takes the striker without a stamp-window delay (DIV-4503).
func TestSecondGuardStruckOnTheWayHomeTakesTheStriker(t *testing.T) {
	w, _ := strikePursuit(t)
	if !walksTo(w, cell{20, 20}) {
		m := leashAt(w, 2)
		t.Fatalf("the member at (%d,%d) walks to (%d,%d)/%v, want its post (20,20)", m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget)
	}
	strikeFrom(t, w, 2)
	leashRun(w, 1)
	if v, ok := engVictim(w, 2); !ok || v != leashHero {
		t.Fatalf("on the first group tick after the blow the member holds victim %v/%v, want the striker", v, ok)
	}
}

// A blow from outside the circle does not divert the return to the post.
func TestSecondGuardHurtMemberOutsideTheCircleKeepsItsPost(t *testing.T) {
	for _, c := range []struct {
		name string
		wait int
	}{{"blow five fine ticks before the group tick", 10}, {"blow fifteen fine ticks before the group tick", 0}} {
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
			strikeFrom(t, w, dx)
			leashRun(w, 1)
			if v, ok := engVictim(w, 2); ok {
				t.Fatalf("the member took victim %v from beyond its circle", v)
			}
			want := cell{20, 20}
			if !walksTo(w, want) {
				m := leashAt(w, 2)
				t.Fatalf("the member at (%d,%d) walks to (%d,%d)/%v, want %v", m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget, want)
			}
		})
	}
}

// A cold load keeps the blow, a retained stamp and the next combat decisions.
func TestSecondGuardBlowAndTargetSurviveAColdLoad(t *testing.T) {
	w, _ := strikePursuit(t)
	o := w.savedOrder(2)
	o.Strike.Stamped, o.Strike.Stamp = true, w.groupClock()
	stamp := o.Strike.Stamp
	strikeFrom(t, w, 2)
	leashRun(w, 1)
	cold := cloneWorld(t, w)
	for n := 2; n <= 30; n++ {
		for _, x := range []*World{w, cold} {
			leashRun(x, 1)
		}
		if w.Hash() != cold.Hash() {
			t.Fatalf("group tick %d after the stamp: the cold World departs from the live one", n)
		}
		v, held := engVictim(cold, 2)
		if !held || v != leashHero || cold.savedOrder(2).Strike.Stamp != stamp {
			t.Fatalf("group tick %d: cold member holds victim %d/%v, stamp %d; want hero and retained stamp %d", n, v, held, cold.savedOrder(2).Strike.Stamp, stamp)
		}
	}
}
