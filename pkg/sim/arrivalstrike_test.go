package sim

import "testing"

const (
	asHero EntityID = 1
	asFoeA EntityID = 2
	asFoeB EntityID = 3
	asSide uint32   = 5
)

func asWorld(t *testing.T, second bool, ents ...Entity) *World {
	t.Helper()
	rel := engRel(t, [3]uint32{asSide, SelfSlot, 1}, [3]uint32{SelfSlot, asSide, 1})
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{}, ents, s, rel, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func asFighter(id EntityID, owner uint32, x, y int32) Entity {
	e := engFighter(id, owner, x, y)
	e.HP, e.MaxHP, e.Reach, e.Speed, e.Mind = 1000, 1000, 1, 10, 20
	return e
}

func asFoe(id EntityID, x, y int32) Entity {
	e := asFighter(id, asSide, x, y)
	e.DamageBase = 0
	return e
}

func asRun(w *World, n int) {
	for k := 0; k < n; k++ {
		Step(w, nil)
	}
}

// A player's fighter under an ordinary move order walks past enemies in sight
// (owner direction), and once he has arrived he strikes an enemy beside him,
// the next one when his victim dies, and one that steps beside him later.
func TestAPlayersFighterWalksPastEnemiesAndStrikesOnArrival(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		w := asWorld(t, second, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 34, 41), asFoe(asFoeB, 34, 39))
		dest := CellPoint{X: 33, Y: 40}
		Step(w, []Command{GroupMoveTo(asHero, dest, 1)})
		asRun(w, 40)
		if e := laEnt(t, w, asHero); e.HasAttackTarget {
			t.Fatalf("second=%v: the walker took victim %d", second, e.AttackTarget)
		}
		asRun(w, 140)
		e := laEnt(t, w, asHero)
		if e.X != dest.X || e.Y != dest.Y || !e.HasAttackTarget {
			t.Fatalf("second=%v: arrived at (%d,%d) holding victim %v/%d", second, e.X, e.Y, e.HasAttackTarget, e.AttackTarget)
		}
		first := e.AttackTarget
		other := asFoeB
		if first == asFoeB {
			other = asFoeA
		}
		Step(w, []Command{Kill(first)})
		asRun(w, 120)
		if e := laEnt(t, w, asHero); !e.HasAttackTarget || e.AttackTarget != other {
			t.Fatalf("second=%v: after the kill the victim is %v/%d, want %d", second, e.HasAttackTarget, e.AttackTarget, other)
		}
	}
}

func TestAnEnemyThatStepsBesideAStandingFighterIsStruck(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		w := asWorld(t, second, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 42, 40))
		Step(w, []Command{GroupMoveTo(asHero, CellPoint{X: 33, Y: 40}, 1)})
		asRun(w, 180)
		if e := laEnt(t, w, asHero); e.HasAttackTarget {
			t.Fatalf("second=%v: the fighter holds victim %d with no enemy beside him", second, e.AttackTarget)
		}
		Step(w, []Command{MoveTo(asFoeA, CellPoint{X: 34, Y: 40})})
		asRun(w, 240)
		if e := laEnt(t, w, asHero); !e.HasAttackTarget || e.AttackTarget != asFoeA {
			t.Fatalf("second=%v: the enemy stepped beside him and his victim is %v/%d", second, e.HasAttackTarget, e.AttackTarget)
		}
	}
}
