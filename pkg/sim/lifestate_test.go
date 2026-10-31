package sim

import (
	"math"
	"testing"
)

func TestLifeStatesAnswerTheRawTests(t *testing.T) {
	healths := []int32{math.MinInt32, -32768, -10002, -10001, -10000, -601, -600, -599, -11, -10, -9, -1, 0, 1, 9, 32767, math.MaxInt32}
	for _, hp := range healths {
		for _, maxHP := range []int32{0, 10} {
			for _, undying := range []bool{false, true} {
				e := Entity{HP: hp, MaxHP: maxHP, Undying: undying}
				want := map[LifeState]bool{
					LifeAlive:     undying || hp >= 0 && (hp != 0 || maxHP <= 0),
					LifeLiving:    hp > 0 || undying,
					LifeDowned:    hp == 0 && maxHP > 0 && !undying,
					LifeDead:      hp < 0 && !undying,
					LifeDecaying:  hp <= 0,
					LifeStanding:  hp > 0,
					LifeDecayWalk: hp < 0 && hp > math.MinInt32,

					LifeBones:       hp <= -10,
					LifeGone:        hp < -600,
					LifeConsumed:    hp < -10000,
					LifeNonNegative: hp >= 0,
				}
				for state, w := range want {
					if got := e.InLife(state); got != w {
						t.Errorf("state %d hp %d max %d undying %v: InLife = %v, want %v", state, hp, maxHP, undying, got, w)
					}
					if got := state.Holds(int64(hp), int64(maxHP), undying); got != w {
						t.Errorf("state %d hp %d max %d undying %v: Holds = %v, want %v", state, hp, maxHP, undying, got, w)
					}
				}
				if e.OrdinaryTargetable() != (hp > -10 || undying) {
					t.Errorf("hp %d undying %v: OrdinaryTargetable = %v", hp, undying, e.OrdinaryTargetable())
				}
				if e.Alive() != want[LifeAlive] || e.Living() != want[LifeLiving] ||
					e.Downed() != want[LifeDowned] || e.Dead() != want[LifeDead] {
					t.Errorf("hp %d max %d undying %v: a predicate disagrees with its state", hp, maxHP, undying)
				}
				if !undying {
					n := 0
					for _, s := range []bool{e.Alive(), e.Downed(), e.Dead()} {
						if s {
							n++
						}
					}
					if n != 1 {
						t.Errorf("hp %d max %d: Alive, Downed and Dead hold %d times, want once", hp, maxHP, n)
					}
				}
			}
		}
	}
}

// Each row is one state at its edge: the threshold, one below it, one above
// it, with the maximum and undying flag the state reads.
func TestLifeStateHoldsAtEachEdge(t *testing.T) {
	rows := []struct {
		state   LifeState
		hp      int64
		maxHP   int64
		undying bool
		want    bool
	}{
		{LifeAlive, -1, 10, false, false}, {LifeAlive, 0, 10, false, false}, {LifeAlive, 0, 0, false, true},
		{LifeAlive, 1, 10, false, true}, {LifeAlive, -1, 10, true, true}, {LifeAlive, 0, 10, true, true},
		{LifeLiving, -1, 10, false, false}, {LifeLiving, 0, 10, false, false}, {LifeLiving, 1, 10, false, true},
		{LifeLiving, -1, 10, true, true}, {LifeLiving, 0, 10, true, true},
		{LifeDowned, -1, 10, false, false}, {LifeDowned, 0, 10, false, true}, {LifeDowned, 1, 10, false, false},
		{LifeDowned, 0, 0, false, false}, {LifeDowned, 0, 10, true, false},
		{LifeDead, -1, 10, false, true}, {LifeDead, 0, 10, false, false}, {LifeDead, 1, 10, false, false},
		{LifeDead, -1, 10, true, false}, {LifeDead, -100, 10, true, false},
		{LifeDecaying, -1, 0, false, true}, {LifeDecaying, 0, 0, false, true}, {LifeDecaying, 1, 0, false, false},
		{LifeDecaying, 0, 0, true, true},
		{LifeStanding, -1, 0, false, false}, {LifeStanding, 0, 0, false, false}, {LifeStanding, 1, 0, false, true},
		{LifeStanding, 0, 0, true, false},
		{LifeDecayWalk, math.MinInt32, 0, false, false}, {LifeDecayWalk, math.MinInt32 + 1, 0, false, true},
		{LifeDecayWalk, -1, 0, false, true}, {LifeDecayWalk, 0, 0, false, false},
		{LifeBones, -11, 0, false, true}, {LifeBones, -10, 0, false, true}, {LifeBones, -9, 0, false, false},
		{LifeBones, -10, 10, true, true}, {LifeBones, -9, 10, true, false},
		{LifeGone, -601, 0, false, true}, {LifeGone, -600, 0, false, false}, {LifeGone, -599, 0, false, false},
		{LifeGone, -601, 10, true, true},
		{LifeConsumed, -10002, 0, false, true}, {LifeConsumed, -10001, 0, false, true},
		{LifeConsumed, -10000, 0, false, false}, {LifeConsumed, -10001, 10, true, true},
		{LifeNonNegative, -1, 0, false, false}, {LifeNonNegative, 0, 0, false, true}, {LifeNonNegative, 1, 0, false, true},
		{LifeNonNegative, -1, 10, true, false},
	}
	for _, r := range rows {
		if got := r.state.Holds(r.hp, r.maxHP, r.undying); got != r.want {
			t.Errorf("state %d hp %d max %d undying %v: Holds = %v, want %v", r.state, r.hp, r.maxHP, r.undying, got, r.want)
		}
	}
}

func TestLoseHealthSaturatesAtTheFloor(t *testing.T) {
	e := Entity{HP: -5}
	e.loseHealth(int64(math.MaxInt32) * 4)
	if e.HP != math.MinInt32 {
		t.Fatalf("HP = %d, want the floor", e.HP)
	}
	e = Entity{HP: 7}
	e.loseHealth(10)
	if e.HP != -3 {
		t.Fatalf("HP = %d, want -3", e.HP)
	}
	e = Entity{HP: 7, Undying: true}
	e.loseHealth(1 << 40)
	if e.HP != undyingHealthFloor {
		t.Fatalf("undying HP = %d, want %d", e.HP, undyingHealthFloor)
	}
}
