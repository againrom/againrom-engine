package sim

import (
	"fmt"
	"testing"
)

const smoothWindowStart = 13

func smoothWorld(t *testing.T, e Entity, second, original, smooth bool) *World {
	t.Helper()
	w := rgWorld(t, e)
	if second {
		asSecondGame(t, w)
	}
	if original {
		w.entities[0].CurrentProfileBasis = ProfileOriginalCurrent
	}
	w.tick = smoothWindowStart
	w.fullTick = DerivedFullTick(uint32(w.tick))
	w.SetSmoothRegen(smooth)
	return w
}

func smoothEntity(rate int32) Entity {
	e := Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 1000, HealthRegenPeriod: 600,
		Mana: 4, MaxMana: 500, ManaRegenPeriod: 300}
	e.ActionClock = ActionClock{Known: true, End: 0x01000000}
	if rate == 3 {
		e.ActionClock.End = 0xfffe7960
	}
	return e
}

func pools(w *World) [4]int64 {
	e := w.entities[0]
	return [4]int64{int64(e.HP), int64(e.Mana), int64(e.HealthHundredths), int64(e.ManaHundredths)}
}

func TestSmoothRegenWholeWindowsEqualTheWholePassGain(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, original := range []bool{false, true} {
			for _, rate := range []int32{1, 3} {
				t.Run(fmt.Sprintf("second=%t/original=%t/rate=%d", second, original, rate), func(t *testing.T) {
					off := smoothWorld(t, smoothEntity(rate), second, original, false)
					on := smoothWorld(t, smoothEntity(rate), second, original, true)
					moved := 0
					for window := 1; window <= 3; window++ {
						for i := 0; i < healthRegenCycle; i++ {
							before := pools(on)
							Step(off, nil)
							Step(on, nil)
							if window == 1 && pools(on) != before {
								moved++
							}
						}
						if got, want := pools(on), pools(off); got != want {
							t.Fatalf("window %d: smooth %v, whole-pass %v", window, got, want)
						}
					}
					if moved < 2 {
						t.Fatalf("pools moved on %d sub-ticks of the first window", moved)
					}
					if p := pools(off); p[0] <= 10 || p[1] <= 4 {
						t.Fatalf("whole-pass pools %v did not move", p)
					}
				})
			}
		}
	}
}

func TestSmoothRegenCapsAtTheMaximumInsideAWindow(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, original := range []bool{false, true} {
			t.Run(fmt.Sprintf("second=%t/original=%t", second, original), func(t *testing.T) {
				e := smoothEntity(1)
				e.HP, e.MaxHP, e.HealthRegenPeriod = 99, 100, 100
				e.Mana, e.MaxMana, e.ManaRegenPeriod = 49, 50, 100
				on := smoothWorld(t, e, second, original, true)
				off := smoothWorld(t, e, second, original, false)
				for i := 0; i < healthRegenCycle; i++ {
					Step(on, nil)
					Step(off, nil)
					if got := on.entities[0]; got.HP > 100 || got.Mana > 50 {
						t.Fatalf("sub-tick %d: pools %d/%d above the maxima", i, got.HP, got.Mana)
					}
				}
				if got := on.entities[0]; got.HP != 100 || got.Mana != 50 || got.HP != off.entities[0].HP || got.Mana != off.entities[0].Mana {
					t.Fatalf("capped pools %d/%d, whole-pass %d/%d", got.HP, got.Mana, off.entities[0].HP, off.entities[0].Mana)
				}
			})
		}
	}
}

func TestSmoothRegenKeepsTheWholeStepWhereTheGainIsNegativeOnEveryPath(t *testing.T) {
	for _, original := range []bool{false, true} {
		for _, second := range []bool{false, true} {
			t.Run(fmt.Sprintf("original=%t/second=%t", original, second), func(t *testing.T) {
				e := smoothEntity(1)
				e.HealthRegeneration, e.ManaRegeneration = -200, -200
				e.HP, e.Mana = 900, 400
				off := smoothWorld(t, e, second, original, false)
				on := smoothWorld(t, e, second, original, true)
				start := pools(on)
				for i := 0; i < healthRegenCycle; i++ {
					Step(off, nil)
					Step(on, nil)
					if got := pools(on); i < healthRegenCycle-1 && got[0] != start[0] {
						t.Fatalf("sub-tick %d: health moved to %v before the window closed", i, got)
					}
				}
				if got, want := pools(on), pools(off); got != want {
					t.Fatalf("negative gain: smooth %v, whole-pass %v", got, want)
				}
				if pools(on) == start {
					t.Fatal("negative gain never applied")
				}
			})
		}
	}
}

func TestSmoothRegenPartsSumToTheGain(t *testing.T) {
	for _, gain := range []int64{0, 1, 15, 33, 1<<62 - 1, -7, -(1 << 62)} {
		for _, k := range []int64{16, 64} {
			var sum int64
			for j := int64(0); j < k; j++ {
				part, apply := regenPart{j, k}.step(gain, 0)
				if apply {
					sum += part
				}
			}
			if sum != gain {
				t.Fatalf("gain %d over %d parts sums to %d", gain, k, sum)
			}
		}
	}
}
