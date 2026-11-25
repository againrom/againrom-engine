package sim

import "testing"

// A dead body drops its terminal loot, and rolls its death gold, once: a body
// that the ladder holds at DecayFallen after the drop (the second game's strict
// ladder at exactly -10) is not dropped again on the ticks that follow.

// decayLootTicks is long enough for the ladder's whole walk, one health per
// decayCycle ticks from -10 down to below decayGoneHP.
const decayLootTicks = (-decayGoneHP + 40) * decayCycle

// decayLootGold is the total gold the world's sacks hold.
func decayLootGold(w *World) (total uint32) {
	for _, s := range w.Sacks() {
		total += s.Gold
	}
	return total
}

func TestSecondGameBodyAtTheBonesThresholdRollsItsLootOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		hp   int32
	}{{"exactly -10", -10}, {"below -10", -15}} {
		t.Run(tc.name, func(t *testing.T) {
			w, d := drWorld(t, false, []Entity{drDragon(1, 4)}, nil)
			Step(w, []Command{Damage(1, 10-tc.hp)})
			if got := w.Entities()[0].HP; got != tc.hp {
				t.Fatalf("killing blow left HP %d, want %d", got, tc.hp)
			}
			for k := range decayLootTicks {
				Step(w, nil)
				if k == 3 && tc.hp == -10 {
					if w.entities[0].SuppressRewards || len(w.appendRewardMarks([]byte{0})) != 1 {
						t.Errorf("a spent body carries the reward mark: it would pay no experience if revived")
					}
					if !w.entities[0].SuppressCorpseLoot {
						t.Errorf("a spent body at -10 lacks the corpse-loot mark")
					}
				}
				if len(w.Entities()) == 0 {
					break
				}
			}
			if len(w.Entities()) != 0 {
				t.Fatalf("body still present after %d ticks", decayLootTicks)
			}
			if n := len(d.rolls); n != 1 {
				t.Errorf("death item rolls = %d, want 1 (one deathRoll per body)", n)
			}
			if g := decayLootGold(w); g < 3000 || g > 9000 {
				t.Errorf("sack gold = %d, want one roll's gold in [3000,9000]", g)
			}
		})
	}
}

// A template-marked body with empty roll columns and empty hands draws exactly
// the death roll's two values on its first drop, as on base, and nothing after.
func TestSecondGameMarkedEmptyBodyDrawsAsOnBaseOnItsFirstDrop(t *testing.T) {
	dead := Entity{ID: 1, X: 4, Y: 5, HP: 10, MaxHP: 10, TypeID: 71, SuppressCorpseLoot: true}
	w, _ := drWorld(t, false, []Entity{dead}, nil)
	want := w.rng
	want.uniform(100)
	want.uniform(100)
	Step(w, []Command{Damage(1, 20)})
	Step(w, nil)
	if w.rng.state != want.state {
		t.Fatalf("first drop left the generator at %#x, want %#x (two draws)", w.rng.state, want.state)
	}
	for range 100 {
		Step(w, nil)
	}
	if len(w.Entities()) == 0 || w.rng.state != want.state {
		t.Fatalf("repeat visits drew: state %#x, want %#x", w.rng.state, want.state)
	}
}

func TestFirstGameBodyRollsItsLootOnce(t *testing.T) {
	for _, hp := range []int32{-10, -15} {
		w, err := NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
			ID: 1, X: 4, Y: 5, HP: 10, MaxHP: 10, TypeID: 0x41, GoldChance: 101, TreasureMin: 37,
		}}, nil, Relations{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		Step(w, []Command{Damage(1, 10-hp)})
		for range decayLootTicks {
			Step(w, nil)
			if len(w.Entities()) == 0 {
				break
			}
		}
		if g := decayLootGold(w); g != 37 {
			t.Errorf("hp %d: sack gold = %d, want 37 (one roll)", hp, g)
		}
	}
}
