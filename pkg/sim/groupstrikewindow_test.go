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

func TestSecondGameStrikeWindowDoesNotBlockJoinedFoe(t *testing.T) {
	for _, fighting := range []bool{false, true} {
		name := "striker stands on its blow cell"
		if fighting {
			name = "striker keeps attacking"
		}
		t.Run(name, func(t *testing.T) {
			w := strikeFixture(t)
			leashRun(w, 1)
			m := leashAt(w, 2)
			leashMove(t, w, leashHero, m.X+1, m.Y)
			m.MaxHP, m.HP = 4000, 3000
			leashAt(w, leashHero).DamageBase = 1
			o := w.savedOrder(2)
			o.Strike.Stamped, o.Strike.Stamp = true, w.groupClock()
			w.flipOnBlow(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 2))
			w.engageDrew = nil
			if !w.strikeWindow(indexOfEntity(w.entities, 2), o) || !w.joinsFight(indexOfEntity(w.entities, 2)) {
				t.Fatal("fixture: member is not free to join inside a retained window")
			}
			w.joinSavedEngagedGroups()
			if v, held := engVictim(w, 2); !held || v != leashHero {
				t.Fatalf("join pass: victim %d/%v, want striker", v, held)
			}
			for n := range 4 {
				if fighting {
					Step(w, []Command{Attack(leashHero, 2)})
				}
				leashRun(w, 1)
				if v, held := engVictim(w, 2); !held || v != leashHero {
					t.Fatalf("group pass %d: victim %d/%v, want striker", n+1, v, held)
				}
			}
		})
	}
}
