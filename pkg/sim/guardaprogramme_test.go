package sim

import (
	"fmt"
	"testing"
)

// Cases 1, 2 and 4 and the stamp belong to programme 18. A Group at
// programme 1 decides as the first game's guard does: a hurt member struck
// from outside its sight holds its post, and the player's own unit in a
// Guard Group does not walk off.
func TestSecondGameGuardAProgrammeRunsNoStrikeCases(t *testing.T) {
	away := func(o *SavedActorOrder) (int32, int32) {
		return int32(o.Strike.Away & 0xff), int32(o.Strike.Away >> 8)
	}
	for _, prog := range []uint8{1, 18} {
		t.Run(fmt.Sprintf("pursuit programme %d", prog), func(t *testing.T) {
			w := strikeFixture(t)
			for gi := range w.savedGroups.Groups {
				if g := &w.savedGroups.Groups[gi]; g.Owner.Owner == leashSlot && g.Selector == 1 {
					g.AI[0x20] = prog
				}
			}
			strikeLeashed(t, w, 2*leashSight)
			m := leashAt(w, 2)
			o := w.savedOrder(2)
			if o == nil || o.Strike.Stamped != (prog == 18) {
				t.Fatalf("programme %d: pursuit stamp = %+v", prog, o)
			}
			if _, held := engVictim(w, 2); held {
				t.Fatal("fixture: member still holds a target beyond its sight")
			}
			m.HP = m.MaxHP / 4
			w.flipOnBlow(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
			leashRun(w, 1)
			m = leashAt(w, 2)
			o = w.savedOrder(2)
			ax, ay := away(o)
			walks := m.HasTarget && m.TargetX == ax && m.TargetY == ay && (ax != m.PostX || ay != m.PostY)
			if want := prog == 18; walks != want {
				t.Fatalf("programme %d: walks to +0x9c (%d,%d) = %v, want %v (at (%d,%d) target (%d,%d) %v post (%d,%d))",
					prog, ax, ay, walks, want, m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget, m.PostX, m.PostY)
			}
			if prog == 1 && !walksTo(w, cell{m.PostX, m.PostY}) {
				t.Fatal("guard A did not keep walking to its post")
			}
		})
		t.Run(fmt.Sprintf("recent blow programme %d", prog), func(t *testing.T) {
			w := strikeFixture(t)
			g := w.savedGroupFor(2)
			g.AI[0x20] = prog
			leashRun(w, 1)
			for range 10 {
				Step(w, nil)
			}
			m := leashAt(w, 2)
			leashMove(t, w, leashHero, min(m.X+2*leashSight, engBounds.Width-1), m.Y)
			if m.X != m.PostX || m.Y != m.PostY || m.HasAttackTarget {
				t.Fatal("fixture: member is not at its post without a target")
			}
			m.HP = m.MaxHP / 4
			w.flipOnBlow(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
			leashRun(w, 1)
			m = leashAt(w, 2)
			o := w.savedOrder(2)
			if o == nil || o.Strike.Stamped || !w.strikeRecent(indexOfEntity(w.entities, 2), o) || m.HasAttackTarget {
				t.Fatalf("fixture: recent blow with no pursuit or target: order %+v member %+v", o, m)
			}
			ax, ay := away(o)
			want := cell{m.PostX, m.PostY}
			if prog == 18 {
				want = cell{ax, ay}
			}
			if !walksTo(w, want) || prog == 1 && (m.X != m.PostX || m.Y != m.PostY) {
				t.Fatalf("programme %d: at (%d,%d), walk %v to (%d,%d), want %v", prog, m.X, m.Y, m.HasTarget, m.TargetX, m.TargetY, want)
			}
		})
	}
	t.Run("player's Guard Group", func(t *testing.T) {
		w := strikeFixture(t)
		leashMove(t, w, leashHero, 36, 40)
		Step(w, []Command{GroupStance(leashHero, int32(orderGuard), 1)})
		leashRun(w, 2)
		g := w.savedGroupFor(leashHero)
		if g == nil || g.AI[0x20] != 1 || w.savedRoute(g) != orderGuard || !w.humanParticipantUnit(*leashAt(w, leashHero)) {
			t.Fatalf("hero's Guard Group: %+v", g)
		}
		for range 10 {
			Step(w, nil)
		}
		h := leashAt(w, leashHero)
		h.HP = h.MaxHP / 2
		leashMove(t, w, 3, 44, 40)
		if h.HasAttackTarget || cellOf(h).chebyshevTo(cellOf(leashAt(w, 3))) <= int64(h.ScanRange) {
			t.Fatal("fixture: hero holds a target or sees the striker")
		}
		w.flipOnBlow(indexOfEntity(w.entities, 3), indexOfEntity(w.entities, leashHero))
		for n := range 3 {
			leashRun(w, 1)
			e := *leashAt(w, leashHero)
			if e.HasTarget && (e.TargetX != e.PostX || e.TargetY != e.PostY) || e.X != e.PostX || e.Y != e.PostY {
				t.Fatalf("group tick %d: hero at (%d,%d) walk %v to (%d,%d), post (%d,%d)", n+1, e.X, e.Y, e.HasTarget, e.TargetX, e.TargetY, e.PostX, e.PostY)
			}
			leashMove(t, w, 3, 44, 40)
		}
	})
}
