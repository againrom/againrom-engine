package sim

import (
	"math"
	"testing"
)

func TestLifeStatesAnswerTheRawTests(t *testing.T) {
	healths := []int32{math.MinInt32, -32768, -10, -9, -1, 0, 1, 9, 32767, math.MaxInt32}
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
				}
				for state, w := range want {
					if got := e.InLife(state); got != w {
						t.Errorf("state %d hp %d max %d undying %v: InLife = %v, want %v", state, hp, maxHP, undying, got, w)
					}
					if got := state.Holds(int64(hp), int64(maxHP), undying); got != w {
						t.Errorf("state %d hp %d max %d undying %v: Holds = %v, want %v", state, hp, maxHP, undying, got, w)
					}
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
