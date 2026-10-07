package sim

// The decay ladder: the death transition, the dwell, the walk, the rungs and the
// removal at the bottom (0089 AC-1, AC-2, AC-6 to AC-9).
//
// The occupancy half is downed_test.go's, where the rule it replaces was already
// measured; the byte form's half is decayform_test.go's.

import "testing"

// dcEnt is a living unit with a dwell of its own, on open ground.
func dcEnt(id EntityID, x, y int32, dying int32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: dying}
}

func dcWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	return mustWorld(t, 5, Bounds{Width: 8, Height: 8}, ents)
}

// TestTheConstructorPairsTheStageWithBeingNotAlive is AC-1: the two directions it
// normalises and the one it refuses.
func TestTheConstructorPairsTheStageWithBeingNotAlive(t *testing.T) {
	t.Parallel()

	// A living unit handed a stage, and a dwell with it: both are residue of a
	// state it is not in, and both go.
	w := dcWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 50, MaxHP: 50, Decay: 3, Dwell: 9, DyingTime: 12})
	if got := occEntity(t, w, 1); got.Decay != DecayNone || got.Dwell != 0 {
		t.Errorf("a living unit came out at stage %d owing %d, want no stage and no dwell",
			got.Decay, got.Dwell)
	}
	// A body handed no stage at all is put where a death would have put it, owing
	// its own dying time — and its DEFENCE is untouched, because this is a
	// pairing fix and not a death.
	w = dcWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: -3, MaxHP: 50, DyingTime: 12, Defence: 40})
	if got := occEntity(t, w, 1); got.Decay != DecayFallen || got.Dwell != 12 || got.Defence != 40 {
		t.Errorf("a body came out at stage %d owing %d with defence %d, want stage 1 owing 12 at 40",
			got.Decay, got.Dwell, got.Defence)
	}
	// A dwell at any stage but the first is residue of a dwell that ran out.
	w = dcWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: -30, MaxHP: 50, Decay: 3, Dwell: 5})
	if got := occEntity(t, w, 1); got.Dwell != 0 {
		t.Errorf("a body at stage 3 came out owing %d, want none", got.Dwell)
	}
	// And the stage that is not a value, with the two bytes above it.
	for _, stage := range []DecayStage{5, 6, 255} {
		if _, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil,
			[]Entity{{ID: 1, X: 1, Y: 1, HP: -1, MaxHP: 10, Decay: stage}}); err == nil {
			t.Errorf("a decay stage of %d was accepted", stage)
		}
	}
}

// TestFallingSetsTheStageHalvesTheDefenceAndStartsTheDwell is AC-2, over all
// three ways a blow reaches a unit, and over the dying times whose dwell is none.
func TestFallingSetsTheStageHalvesTheDefenceAndStartsTheDwell(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cmds []Command
	}{
		{"killed outright", []Command{{Kind: KindKill, Entity: 1}}},
		{"damaged to death", []Command{{Kind: KindDamage, Entity: 1, X: 150}}},
		{"damaged to exactly zero", []Command{{Kind: KindDamage, Entity: 1, X: 100}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := dcEnt(1, 1, 1, 12)
			e.Defence, e.TargetX, e.TargetY, e.HasTarget = 41, 5, 5, true
			w := dcWorld(t, e)
			Step(w, tc.cmds)
			got := occEntity(t, w, 1)
			if got.Decay != DecayFallen {
				t.Errorf("stage %d on the tick it fell, want 1", got.Decay)
			}
			// 41 >> 1 is 20: the shift is what is asserted, not a division that
			// happens to agree on this number.
			if got.Defence != 20 {
				t.Errorf("defence %d, want 41 shifted right by one", got.Defence)
			}
			// ELEVEN and not twelve: the tick a unit falls on is the first tick
			// of its dwell, so the pass at the end of that same tick has already
			// spent one. Twelve here would be a body that dwells for thirteen.
			if got.Dwell != 11 {
				t.Errorf("dwell %d after the tick it fell, want 11 of the 12 its dying time names",
					got.Dwell)
			}
			if got.HasTarget {
				t.Error("a felled unit kept its order")
			}
		})
	}

	// The dwell over the short and the absent dying times, read after the tick of
	// the fall. A dying time of one and one of none are the same body: torn down
	// where it fell, which is what an absent column and a zero column already are
	// to every other consumer.
	for _, tc := range []struct {
		dying int32
		want  uint16
	}{{8, 7}, {1, 0}, {0, 0}, {-1, 0}} {
		w := dcWorld(t, dcEnt(1, 1, 1, tc.dying))
		Step(w, []Command{{Kind: KindKill, Entity: 1}})
		if got := occEntity(t, w, 1); got.Dwell != tc.want {
			t.Errorf("a dying time of %d gives a dwell of %d, want %d", tc.dying, got.Dwell, tc.want)
		}
	}

	// And it fires ONCE. A body at exactly zero can still be struck, and a second
	// blow must not halve a defence that is already halved.
	e := dcEnt(1, 1, 1, 12)
	e.Defence = 41
	w := dcWorld(t, e)
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 100}})
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 3}})
	if got := occEntity(t, w, 1); got.Defence != 20 {
		t.Errorf("defence %d after a second blow, want the 20 the first left", got.Defence)
	}
}

// TestTheWalkTakesOneHealthPerPeriodAndSaturates is AC-6: the cadence, the phase,
// and the representation floor.
func TestTheWalkTakesOneHealthPerPeriodAndSaturates(t *testing.T) {
	t.Parallel()

	w := dcWorld(t, dcEnt(1, 1, 1, 0))
	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	// The kill landed on tick 0 and the dwell was none, so the body is torn down
	// and every later tick is either a walk tick or not.
	hp := occEntity(t, w, 1).HP
	falls, walks := 0, 0
	// THE PERIOD AND THE PHASE ARE LITERALS HERE, not the constants the pass
	// reads: a test computing what it expects from the same two numbers the code
	// uses cannot notice either of them moving. Thirty-two is a full tick taken
	// twice and twelve is the slot within it.
	const wantCycle, wantPhase = 32, 12
	if decayCycle != wantCycle || decayPhase != wantPhase {
		t.Fatalf("the walk runs at phase %d of %d, want %d of %d",
			decayPhase, decayCycle, wantPhase, wantCycle)
	}
	for tick := uint64(1); tick < 3*wantCycle; tick++ {
		want := tick%wantCycle == wantPhase
		if want {
			walks++
		}
		Step(w, nil)
		got := occEntity(t, w, 1).HP
		if (got != hp) != want {
			t.Errorf("tick %d: health went %d to %d, want a fall on this tick: %v",
				tick, hp, got, want)
		}
		if got != hp {
			if hp-got != 1 {
				t.Errorf("tick %d: health fell by %d, want 1", tick, hp-got)
			}
			falls++
		}
		hp = got
	}
	if falls != walks || falls < 2 {
		t.Errorf("%d fall(s) over %d walk tick(s) in three periods", falls, walks)
	}

	// The floor, on a body already at the least health there is: a decrement that
	// wrapped would put it at the TOP of the range and alive, which is the one
	// outcome worse than a wrong number.
	e := dcEnt(2, 2, 2, 0)
	e.HP, e.MaxHP = minHP, 100
	w2 := mustWorld(t, 5, Bounds{Width: 8, Height: 8}, []Entity{e})
	for k := 0; k <= decayCycle; k++ {
		Step(w2, nil)
		for _, got := range w2.Entities() {
			if got.HP > 0 {
				t.Fatalf("a body at the least health there is came back at %d", got.HP)
			}
		}
	}
}

// TestTheStageIsTheLadderOfHealthAndNeverFalls is AC-7: a body climbs rung by
// rung as its health passes each threshold, an overshot one lands on the rung its
// health names rather than climbing to it, and no stage ever falls.
func TestTheStageIsTheLadderOfHealthAndNeverFalls(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		dmg   int32
		stage DecayStage
	}{
		// The four rungs are asked AT their thresholds and one short of each, so
		// a threshold moved by one is a case here rather than a number nothing
		// measures. Damage d leaves health 100-d, and the walk takes one more on
		// the tick of the fall only when that tick is a walk tick — it is not
		// here, the kill landing on tick 0.
		{"felled shallow, still at the first stage", 101, DecayFallen},
		{"one short of the first threshold", 109, DecayFallen},
		{"at the first threshold exactly", 110, DecayBones},
		{"one short of the second", 119, DecayBones},
		{"at the second exactly", 120, 3},
		{"one short of the third", 139, 3},
		{"at the third exactly", 140, decayLast},
		{"far past the third", 300, decayLast},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := dcWorld(t, dcEnt(1, 1, 1, 0))
			Step(w, []Command{{Kind: KindDamage, Entity: 1, X: tc.dmg}})
			if got := occEntity(t, w, 1); got.Decay != tc.stage {
				t.Errorf("health %d puts the body at stage %d, want %d", got.HP, got.Decay, tc.stage)
			}
		})
	}

	// And the climb, rung by rung, on a body felled shallow: the stage is
	// monotone and reaches every rung in order.
	w := dcWorld(t, dcEnt(1, 1, 1, 0))
	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	last := occEntity(t, w, 1).Decay
	reached := []DecayStage{last}
	for k := 0; k < 45*decayCycle; k++ {
		Step(w, nil)
		if len(w.Entities()) == 0 {
			break
		}
		got := occEntity(t, w, 1).Decay
		if got < last {
			t.Fatalf("the stage fell from %d to %d", last, got)
		}
		if got != last {
			reached = append(reached, got)
			last = got
		}
	}
	want := []DecayStage{DecayFallen, DecayBones, 3, decayLast}
	if len(reached) != len(want) {
		t.Fatalf("the body reached stages %v, want %v", reached, want)
	}
	for i := range want {
		if reached[i] != want[i] {
			t.Fatalf("the body reached stages %v, want %v", reached, want)
		}
	}
}

// TestTheBottomOfTheLadderTakesTheBodyOutOfTheWorld is AC-8: the last health a
// body exists at, the first it does not, and what goes with the record.
func TestTheBottomOfTheLadderTakesTheBodyOutOfTheWorld(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		hp   int32
		gone bool
	}{
		{"at the last health a body exists at", decayGoneHP, false},
		{"one past it", decayGoneHP - 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := dcEnt(1, 1, 1, 0)
			body.HP = -9
			// A second unit holding an ATTACK ORDER on the body, built through the
			// constructor because no command can set one on a unit that is
			// already dead — so what the removal sweeps is measured rather than
			// assumed.
			killer := dcEnt(2, 2, 1, 0)
			killer.AttackTarget, killer.HasAttackTarget = 1, true
			w := dcWorld(t, body, killer)
			if got := occEntity(t, w, 2); !got.HasAttackTarget {
				t.Fatal("the fixture's attacker holds no order")
			}
			w.entities[indexOfEntity(w.entities, 1)].HP = tc.hp
			Step(w, nil)
			if got := indexOfEntity(w.entities, 1); (got < 0) != tc.gone {
				t.Fatalf("the body is present: %v, want gone: %v", got >= 0, tc.gone)
			}
			if !tc.gone {
				if got := occEntity(t, w, 1); got.Decay != decayLast {
					t.Errorf("the body is at stage %d, want %d", got.Decay, decayLast)
				}
				return
			}
			if len(w.routes) != len(w.entities) {
				t.Errorf("%d route slot(s) for %d entities", len(w.routes), len(w.entities))
			}
			if got := occEntity(t, w, 2); got.HasAttackTarget {
				t.Errorf("a survivor still holds an order on entity %d, which the world no longer has",
					got.AttackTarget)
			}
		})
	}
}

// TestANonGroundMoverLeavesNoBody is AC-9: the two domains that are not the
// ground one are pinned below the ladder when their dwell runs out, so they play
// their fall, hold it, and go — leaving no body and no bones.
func TestANonGroundMoverLeavesNoBody(t *testing.T) {
	t.Parallel()

	ents := []Entity{dcEnt(1, 1, 1, 4), dcEnt(2, 3, 1, 4), dcEnt(3, 5, 1, 4)}
	ents[1].Domain, ents[2].Domain = DomainGhost, DomainAir
	w := dcWorld(t, ents...)
	Step(w, []Command{
		{Kind: KindKill, Entity: 1}, {Kind: KindKill, Entity: 2}, {Kind: KindKill, Entity: 3},
	})
	// Ticks of dwell still owed: all three are still bodies, and the two that
	// will leave none are indistinguishable from the one that will.
	for k := 0; k < 2; k++ {
		if len(w.Entities()) != 3 {
			t.Fatalf("tick %d: %d entities, want 3 — the dwell is still owed", k, len(w.Entities()))
		}
		Step(w, nil)
	}
	// And the teardown, which is where they part.
	Step(w, nil)
	got := w.Entities()
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("after the dwell the world holds %+v, want the ground mover alone", got)
	}
	if got[0].Decay != DecayFallen {
		t.Errorf("the ground body is at stage %d, want 1", got[0].Decay)
	}
}
