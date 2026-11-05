package sim

import (
	"fmt"
	"testing"
)

// routeWallCell is a cell no route reaches: the eight cells round it are
// blocked to the ground. It lies four cells from the guard's post (20,20),
// inside the guard's circle and sight, and off the line from the post to
// the blow's cell (30,20).
var routeWallCell = cell{24, 24}

func routeWallGrid() []byte {
	g := make([]byte, engBounds.Width*engBounds.Height)
	for dy := int32(-1); dy <= 1; dy++ {
		for dx := int32(-1); dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				g[(routeWallCell.y+dy)*engBounds.Width+routeWallCell.x+dx] = blockGround
			}
		}
	}
	return g
}

// routePass is member 2 as one group tick's decisions leave it.
type routePass struct {
	clock, stamp uint16
	stamped      bool
	at           cell
	// window: the member holds the hero as its target, runs no attack and
	// walks to the blow's cell, or holds its cell with no blow.
	window bool
	// attacks: the member holds the hero and its attack runs.
	attacks bool
}

func (p routePass) String() string {
	return fmt.Sprintf("{clock %d stamp %d/%v window %v attacks %v}", p.clock, p.stamp, p.stamped, p.window, p.attacks)
}

// routePasses runs n group ticks and records each.
func routePasses(w *World, n int, away cell, held bool) []routePass {
	var out []routePass
	for range n {
		leashRun(w, 1)
		m := *leashAt(w, 2)
		o := w.savedOrder(2)
		v, target := engVictim(w, 2)
		p := routePass{clock: w.groupClock(), stamp: o.Strike.Stamp, stamped: o.Strike.Stamped, at: cellOf(&m)}
		hero := target && v == leashHero
		walks := walksTo(w, away)
		if !held {
			walks = !m.HasTarget
		}
		p.window = hero && m.PursuitIdle && walks
		p.attacks = hero && !m.PursuitIdle && o.Raw[8] == 5
		out = append(out, p)
	}
	return out
}

// A second-game guard B member that cannot reach the unit it attacks steps
// away for the stamp's window and then attacks again: the approach's route
// search fails and stamps cmd +0x16 with server word +0; for 10 group ticks
// (30 while hurt) the member keeps its target and walks to the blow's cell
// +0x9c, or holds its cell with no blow; then it attacks, the search fails
// and stamps again (R2-ENGINE-873, R2-ENGINE-874, R2-ENGINE-844 cases 1
// and 2, DIV-4503, DIV-4544).
func TestSecondGuardThatCannotReachItsTargetStepsAwayAndAttacksAgain(t *testing.T) {
	for _, c := range []struct {
		name       string
		blow, hurt bool
		window     uint16
	}{
		{"blow, hurt", true, true, 30},
		{"blow, whole", true, false, 10},
		{"no blow, hurt", false, true, 30},
		{"no blow, whole", false, false, 10},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := missionStartWorld(t, []cell{{20, 20}}, Terrain{Block: routeWallGrid()})
			leashAt(w, leashHero).HP, leashAt(w, leashHero).MaxHP = 4000, 4000
			leashRun(w, 1)
			m := leashAt(w, 2)
			if m.X != 20 || m.Y != 20 || m.HasAttackTarget {
				t.Fatal("fixture: the member is not at its post")
			}
			away := cell{30, 20}
			if c.blow {
				leashMove(t, w, leashHero, 19, 20)
				w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
				if o := w.savedOrder(2); !o.Strike.Held || int32(o.Strike.Away&0xff) != away.x || int32(o.Strike.Away>>8) != away.y {
					t.Fatalf("fixture: blow record %+v, want +0x9c %v", o.Strike, away)
				}
			}
			if c.hurt {
				m.HP = m.MaxHP / 4
			}
			leashMove(t, w, leashHero, routeWallCell.x, routeWallCell.y)
			passes := routePasses(w, int(3*c.window), away, c.blow)
			open := -1
			for k, p := range passes {
				if p.window {
					open = k
					break
				}
			}
			if open < 0 {
				t.Fatalf("the member never stepped away from a target no route reaches: %v", passes[:6])
			}
			s := passes[open].stamp
			if !passes[open].stamped || passes[open].clock-s > 1 || open == 0 || !passes[open-1].attacks {
				t.Fatalf("window opened at pass %d without the approach's fresh stamp: %v", open, passes[:open+1])
			}
			// The window holds the stamp; the decision that ends it attacks,
			// and the approach may search, miss and stamp in the same tick.
			k := open
			for ; k < len(passes) && passes[k].window && passes[k].stamp == s; k++ {
			}
			if k == len(passes) || passes[k-1].clock != s+c.window-1 || passes[k].clock != s+c.window {
				t.Fatalf("window from stamp %d: want %d group ticks; passes %v", s, c.window, passes[open:min(k+2, len(passes))])
			}
			if !passes[k].attacks && passes[k].stamp != passes[k].clock {
				t.Fatalf("the member did not attack after the window: %v", passes[k-1:min(k+2, len(passes))])
			}
			again := k
			for ; again < len(passes) && passes[again].stamp == s; again++ {
			}
			if again+1 >= len(passes) || passes[again].stamp < s+c.window || !passes[again+1].window {
				t.Fatalf("the attack after the window did not fail and stamp again: %v", passes[k:min(again+2, len(passes))])
			}
			for j := open + 2; !c.blow && j < k; j++ {
				if passes[j].at != passes[open+1].at {
					t.Fatalf("pass %d: the member with no blow left its cell inside the window: %v", j, passes[open:k])
				}
			}
			m = leashAt(w, 2)
			if c.blow && (m.X <= 20 || cellOf(m).chebyshevTo(routeWallCell) <= 2) {
				t.Fatalf("the member at (%d,%d) did not walk toward +0x9c %v", m.X, m.Y, away)
			}
		})
	}
}

// A cold load inside the window keeps the stamp, the kept target and the
// walk, so the loaded World steps away and attacks again as the live one.
func TestSecondGuardStepAwayWindowSurvivesAColdLoad(t *testing.T) {
	w := missionStartWorld(t, []cell{{20, 20}}, Terrain{Block: routeWallGrid()})
	leashAt(w, leashHero).HP, leashAt(w, leashHero).MaxHP = 4000, 4000
	leashRun(w, 1)
	leashMove(t, w, leashHero, 19, 20)
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
	leashAt(w, 2).HP = leashAt(w, 2).MaxHP / 4
	leashMove(t, w, leashHero, routeWallCell.x, routeWallCell.y)
	var opened bool
	for range 4 {
		if p := routePasses(w, 1, cell{30, 20}, true); p[0].window {
			opened = true
			break
		}
	}
	if !opened {
		t.Fatal("the member never stepped away from a target no route reaches")
	}
	cold := cloneWorld(t, w)
	live := routePasses(w, 40, cell{30, 20}, true)
	loaded := routePasses(cold, 40, cell{30, 20}, true)
	for k := range live {
		if live[k] != loaded[k] {
			t.Fatalf("group tick %d after the load: live %v, loaded %v", k+1, live[k], loaded[k])
		}
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("the loaded World departs from the live one")
	}
}

// A member that reaches its target is never stamped: open ground, the same
// blow and the same striker cell.
func TestSecondGuardThatReachesItsTargetIsNotStamped(t *testing.T) {
	w := missionStartWorld(t, []cell{{20, 20}})
	leashAt(w, leashHero).HP, leashAt(w, leashHero).MaxHP = 4000, 4000
	leashRun(w, 1)
	leashMove(t, w, leashHero, 19, 20)
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
	leashAt(w, 2).HP = leashAt(w, 2).MaxHP / 4
	leashMove(t, w, leashHero, routeWallCell.x, routeWallCell.y)
	heroHP := leashAt(w, leashHero).HP
	for k, p := range routePasses(w, 12, cell{30, 20}, true) {
		if p.stamped || p.window || !p.attacks {
			t.Fatalf("group tick %d: %v; want the attack and no stamp", k+1, p)
		}
	}
	if leashAt(w, leashHero).HP >= heroHP {
		t.Fatal("the member dealt no damage to a target it reaches")
	}
}

// A loaded stamp with no blow builds the member's missing order and opens
// case 2, which tests no blow (R2-ENGINE-883).
func TestSecondGuardLoadedStampAloneOpensTheWindow(t *testing.T) {
	w := leashWorld(t, cell{5, 5}, []cell{{20, 20}}, nil)
	if w.savedOrder(2) != nil {
		t.Fatal("fixture: the LOAD holds a member order")
	}
	stamp := w.groupClock()
	if err := w.ImportLoadedStrikes([]LoadedStrike{{Entity: 2, Stamp: stamp}}); err != nil {
		t.Fatal(err)
	}
	o := w.savedOrder(2)
	if o == nil || !o.Strike.Stamped || o.Strike.Stamp != stamp || o.Strike.Held {
		t.Fatalf("member order after the LOAD: %+v, want the stamp %d and no blow", o, stamp)
	}
	if !w.strikeWindow(indexOfEntity(w.entities, 2), o) {
		t.Fatal("the loaded stamp does not open case 2")
	}
}
