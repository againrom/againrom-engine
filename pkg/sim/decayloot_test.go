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

// A body felled to exactly -10 drops loot once in both games and does not roll
// again after a binary save and load, even while its stage stays DecayFallen.
func TestBothGamesBodyAtMinusTenDropsLootOnceAcrossLoad(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := "first"
		if second {
			name = "second"
		}
		t.Run(name, func(t *testing.T) {
			var w *World
			var d *drDraws
			if second {
				var err error
				w, err = NewStockedWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{drDragon(1, 4)}, secondGameScript(t, nil, nil, nil), Relations{}, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				w.rng = rng{state: 0x2468ace1}
				d = &drDraws{}
				w.SetDeathItemDraw(d.draw)
			} else {
				var err error
				w, err = NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
					ID: 1, X: 4, Y: 5, HP: 10, MaxHP: 10, TypeID: 0x41, GoldChance: 101, TreasureMin: 37,
				}}, nil, Relations{}, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			Step(w, []Command{Damage(1, 20)})
			got := w.Entities()[0]
			if got.HP != -10 || got.Decay != DecayFallen || !got.CorpseDecayed {
				t.Fatalf("after lethal hit to -10: hp=%d decay=%d decayed=%v, want -10/1/true", got.HP, got.Decay, got.CorpseDecayed)
			}
			goldAfterDrop := decayLootGold(w)
			if second {
				if goldAfterDrop < 3000 || goldAfterDrop > 9000 || len(d.rolls) != 1 {
					t.Fatalf("second game initial drop: gold=%d rolls=%d", goldAfterDrop, len(d.rolls))
				}
			} else if goldAfterDrop != 37 {
				t.Fatalf("first game initial drop: gold=%d, want 37", goldAfterDrop)
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			if second {
				cold.SetDeathItemDraw(d.draw)
			}
			if cold.Hash() != w.Hash() {
				t.Fatalf("hash changed across load: %#x != %#x", cold.Hash(), w.Hash())
			}
			afterDrop := w.RandomState()
			for range 64 {
				Step(&cold, nil)
			}
			if got := cold.RandomState(); got != afterDrop {
				t.Errorf("loaded corpse drew again: random state %#x, want %#x", got, afterDrop)
			}
			if body, ok := cold.Entity(1); !ok || body.HP >= -10 || body.Decay != DecayBones || !body.CorpseDecayed {
				t.Fatal("loaded body did not advance to bones")
			}
			if decayLootGold(&cold) != goldAfterDrop {
				t.Errorf("sack gold after load = %d, want %d (no second roll)", decayLootGold(&cold), goldAfterDrop)
			}
			if second && len(d.rolls) != 1 {
				t.Errorf("second game death item rolls after load = %d, want 1", len(d.rolls))
			}
		})
	}
}

// A body an older second-game save holds at -10 after its drop, with a duration
// attachment, is torn down at load as a fresh body is: the attachment expires
// and the later ticks match the fresh run.
func TestLegacyMinusTenBodyLoadExpiresItsAttachments(t *testing.T) {
	build := func(legacy bool) *World {
		w, err := NewStockedWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{drDragon(1, 4)}, secondGameScript(t, nil, nil, nil), Relations{}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.rng = rng{state: 0x2468ace1}
		d := &drDraws{}
		w.SetDeathItemDraw(d.draw)
		Step(w, []Command{Damage(1, 20)})
		if legacy {
			e := &w.entities[0]
			e.CorpseDecayed, e.deadRank = false, 0
			w.attached = append(w.attached, attachedEffect{Target: 1, Spell: 27, Remaining: 3000, Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: 3})
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			cold.SetDeathItemDraw(d.draw)
			return &cold
		}
		return w
	}
	fresh, loaded := build(false), build(true)
	for _, a := range loaded.attached {
		if a.Target == 1 {
			t.Fatal("attachment kept on the body at load")
		}
	}
	for range 1000 {
		Step(fresh, nil)
		Step(loaded, nil)
	}
	f, _ := fresh.Entity(1)
	l, _ := loaded.Entity(1)
	if f.HP != l.HP || f.Decay != l.Decay || fresh.RandomState() != loaded.RandomState() || decayLootGold(fresh) != decayLootGold(loaded) {
		t.Fatalf("loaded run differs: hp %d/%d stage %d/%d", f.HP, l.HP, f.Decay, l.Decay)
	}
}
