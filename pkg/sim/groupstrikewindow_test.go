package sim

import (
	"testing"
)

// strikeFixture is a second-game mission start: the hero (slot self), one
// whole guard member (entity 2) of slot 3's Group 1 and a second foe.
func strikeFixture(t *testing.T) *World {
	t.Helper()
	ents := []Entity{engFighter(leashHero, SelfSlot, 5, 5)}
	m := engFighter(2, leashSlot, 20, 20)
	m.Group, m.ScanRange, m.HP, m.MaxHP = 1, leashSight, 400, 400
	other := engFighter(3, leashSlot, 30, 30)
	other.Group = 2
	ents = append(ents, m, other)
	rel := engRel(t, [3]uint32{leashSlot, SelfSlot, 1}, [3]uint32{SelfSlot, leashSlot, 2})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, ents, secondGameScript(t, nil, nil, nil), rel)
	if err != nil {
		t.Fatal(err)
	}
	if !playsSecondGame(w) {
		t.Fatal("fixture is not the second game")
	}
	h := leashAt(w, leashHero)
	h.HP, h.MaxHP = 4000, 4000
	return w
}

// strikeLeashed leads the member beyond its leash, so the hero's blow finds
// it holding no target.
func strikeLeashed(t *testing.T, w *World, standoff int32) {
	t.Helper()
	leashRun(w, 1)
	for range 3 {
		m := leashAt(w, 2)
		leashMove(t, w, leashHero, m.X+3, m.Y)
		leashRun(w, 1)
	}
	m := leashAt(w, 2)
	leashMove(t, w, leashHero, min(m.X+standoff, engBounds.Width-1), m.Y)
	leashRun(w, 1)
}

// A guard B member whose stamp window holds takes no foe from the join
// pass, whether the striker stands on its blow cell or keeps attacking, and
// may take it once the window ends.
func TestSecondGameStrikeWindowTakesNoJoinedFoe(t *testing.T) {
	check := func(t *testing.T, w *World, step func(), words int, fighting bool) {
		i := indexOfEntity(w.entities, 2)
		o := w.savedOrder(2)
		if o == nil {
			t.Fatal("fixture: member has no order")
		}
		inWindow, eligible, blows, after := 0, 0, 0, false
		was, clock := w.strikeWindow(i, o), o.Strike.Clock
		blowCell := cellOf(leashAt(w, leashHero))
		for range words {
			step()
			m := leashAt(w, 2)
			o := w.savedOrder(2)
			if o == nil {
				continue
			}
			// A blow lands after the Group's decision in its tick, so the
			// first tick the window holds can still show the chase it ends.
			now := w.strikeWindow(indexOfEntity(w.entities, 2), o)
			was, now = now, now && was
			if now {
				inWindow++
				if w.joinsFight(i) && containsIndex(w.engagedFoes(m.Owner, []int{i}), indexOfEntity(w.entities, leashHero)) {
					eligible++
				}
				if fighting && o.Strike.Clock != clock {
					blows++
				}
				if !fighting && cellOf(leashAt(w, leashHero)) != blowCell {
					t.Fatal("fixture: striker left its blow cell inside the window")
				}
				if v, held := engVictim(w, 2); held {
					t.Fatalf("tick %d word %d: member inside its window holds victim %d at (%d,%d)", w.tick, w.groupClock(), v, m.X, m.Y)
				}
			} else if _, held := engVictim(w, 2); held && !was && inWindow > 0 {
				after = true
			}
			clock = o.Strike.Clock
		}
		if inWindow < 3 {
			t.Fatalf("window held at %d group ticks, want several", inWindow)
		}
		if !after {
			t.Fatal("member never took the striker after its window ended")
		}
		if eligible == 0 {
			t.Fatal("fixture: member never arrived free to join its striker inside the window")
		}
		if fighting && blows < 2 {
			t.Fatalf("fixture: only %d further blows inside the window, want repeated attacks", blows)
		}
		t.Logf("window checked at %d group ticks, join eligible at %d, further blows %d; striker taken after expiry", inWindow, eligible, blows)
	}
	t.Run("striker stands on its blow cell", func(t *testing.T) {
		w := strikeFixture(t)
		strikeLeashed(t, w, 15)
		m := leashAt(w, 2)
		leashMove(t, w, leashHero, m.X+2, m.Y)
		w.flipOnBlow(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
		check(t, w, func() { leashRun(w, 1) }, 40, false)
	})
	t.Run("striker keeps attacking", func(t *testing.T) {
		w := strikeFixture(t)
		strikeLeashed(t, w, 15)
		m := leashAt(w, 2)
		leashMove(t, w, leashHero, m.X+1, m.Y)
		m.MaxHP, m.HP = 4000, 3000
		leashAt(w, leashHero).DamageBase = 1
		check(t, w, func() {
			for {
				if hv, ok := engVictim(w, leashHero); !ok || hv != 2 {
					Step(w, []Command{Attack(leashHero, 2)})
				} else {
					Step(w, nil)
				}
				if w.ScriptPassJustRan() {
					return
				}
			}
		}, 60, true)
	})
}
