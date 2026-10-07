package sim

import "testing"

func manualCastWorld(t *testing.T) *World {
	t.Helper()
	e := spMage(1, 3, 3, 60, 1000, 1000, 1<<1|1<<26)
	e.Owner, e.AttackCharge, e.AttackRelax = 2, 8, 12
	return spWorld(t, 42, []SpellRule{
		{ID: 1, ManaCost: 5, MaxRange: 10, DamageMin: 4, DamageMax: 4, TargetsUnit: true, Damaging: true},
		{ID: 26, ManaCost: 7, MaxRange: 10},
	}, e, spEnt(2, 4, 3))
}

func TestManualCastPreemptsCurrentAction(t *testing.T) {
	for _, target := range []string{"unit", "cell"} {
		for _, action := range []string{"attack", "recovery", "turn", "move", "retreat", "pickup", "book-release"} {
			t.Run(target+"/"+action, func(t *testing.T) {
				w := manualCastWorld(t)
				e := &w.entities[0]
				switch action {
				case "attack":
					w.orderAttack(0, 2)
					e.AttackPhase, e.AttackCountdown = AttackCharging, 1
				case "recovery":
					e.CastWait = 200
				case "turn":
					e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 128, 200, 200
				case "move":
					Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 10, Y: 3}})
				case "retreat":
					Step(w, retreat1089(1))
				case "pickup":
					if !w.CompleteSackPickup(1) {
						t.Fatal("pickup completion refused")
					}
				case "book-release":
					if !w.beginBookSpellOnce(0, 2, 1) {
						t.Fatal("predecessor cast refused")
					}
					w.bookCasts[0].Remaining = 1
					e.clearTurn()
				}
				// A stale guard post must not become a walk-home order after casting.
				w.entities[0].PostX, w.entities[0].PostY = 1, 1
				cmd := spCast(1, 2, 1)
				if target == "cell" {
					cmd = Command{Kind: KindCastAt, Entity: 1, Spell: 26, X: 6, Y: 3}
				}
				beforeMana, beforeHP := w.entities[0].Mana, w.entities[1].HP
				if why := w.manualCastRefusal(cmd); why != "" {
					t.Fatalf("manual admission still refuses the busy actor: %s", why)
				}
				if events := StepObserved(w, []Command{cmd}); len(events) != 0 {
					t.Fatalf("old action released on replacement tick: %+v", events)
				}
				if action == "attack" {
					if !w.entities[0].HasAttackTarget || w.entities[0].AttackTarget != 2 || w.entities[0].PendingOrder.Kind == PendingNone || len(w.bookCasts) != 0 {
						t.Fatal("manual setter lost active fields before progress-zero dispatch")
					}
				} else if len(w.bookCasts) != 1 || w.bookCasts[0].AtCell != (target == "cell") ||
					w.entities[0].Mana != beforeMana || w.entities[1].HP != beforeHP {
					t.Fatalf("manual cast did not take over: casts=%+v actor=%+v", w.bookCasts, w.entities[0])
				}
				if w.entities[0].HasTarget || action != "attack" && w.entities[0].HasAttackTarget || w.entities[0].ActorState != actorStateAcquire {
					t.Fatal("old movement, attack or guard order survived")
				}
				back := retreatRoundTrip1089(t, w)
				casts := 0
				for tick := 0; tick < 64; tick++ {
					casts += len(StepObserved(w, nil))
					Step(back, nil)
					if w.Hash() != back.Hash() {
						t.Fatalf("native reload diverged at tick %d", tick)
					}
				}
				cost := int32(5)
				if target == "cell" {
					cost = 7
					if w.entities[0].X != 6 || w.entities[0].Y != 3 {
						t.Fatalf("Teleport did not stay at its destination: %+v", w.entities[0])
					}
				}
				if casts != 1 || w.entities[0].Mana != beforeMana-cost {
					t.Fatalf("casts=%d mana=%d want%d", casts, w.entities[0].Mana, beforeMana-cost)
				}
			})
		}
	}
}

func TestManualCastInterruptsReservedScroll(t *testing.T) {
	for _, stage := range []string{"approach", "charge", "release"} {
		t.Run(stage, func(t *testing.T) {
			w := scrollWorld1090(t, 12, 2)
			w.entities[0].Owner = 2
			w.entities[0].Mana, w.entities[0].MaxMana, w.entities[0].Mind = 100, 100, 60
			w.entities[0].KnownSpells = 1 << 26
			w.spells = append(w.spells, SpellRule{ID: 26, ManaCost: 7, MaxRange: 10})
			Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
			if len(w.scrollCasts) != 1 {
				t.Fatal("scroll not reserved")
			}
			if stage != "approach" {
				for tick := 0; tick < 200 && !w.scrollCasts[0].Started; tick++ {
					Step(w, nil)
				}
				if !w.scrollCasts[0].Started {
					t.Fatal("scroll did not start")
				}
				if stage == "release" {
					w.scrollCasts[0].Remaining = 1
					w.entities[0].clearTurn()
				}
			}
			if events := StepObserved(w, []Command{{Kind: KindCastAt, Entity: 1, Spell: 26, X: 4, Y: 4}}); len(events) != 0 {
				t.Fatalf("cancelled scroll applied: %+v", events)
			}
			if len(w.scrollCasts) != 0 || len(w.carried[0]) != 1 || w.carried[0][0].Count != 2 || len(w.bookCasts) != 1 {
				t.Fatalf("scroll refund or manual admission failed: casts=%+v stock=%+v", w.scrollCasts, w.carried[0])
			}
			back := retreatRoundTrip1089(t, w)
			for tick := 0; tick < 64; tick++ {
				Step(w, nil)
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatal("refunded scroll continuation differs after reload")
				}
			}
			if w.entities[0].X != 4 || w.entities[0].Y != 4 || w.entities[0].Mana != 93 || w.carried[0][0].Count != 2 {
				t.Fatal("manual Teleport did not replace the scroll exactly once")
			}
		})
	}
}

func TestManualTeleportInterruptsImportedCrossing(t *testing.T) {
	w := importedRelocationWorld1115(t)
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, Spell: 26, X: 7, Y: 5}})
	if len(w.bookCasts) != 1 || w.motionFor(1).Active || w.motionFor(1).Current {
		t.Fatal("imported movement still owns the caster")
	}
	back := retreatRoundTrip1089(t, w)
	for tick := 0; tick < 64; tick++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("imported continuation diverged after reload at %d", tick)
		}
	}
	if w.entities[0].X != 7 || w.entities[0].Y != 5 {
		t.Fatal("imported caster did not teleport")
	}
}

func TestInvalidManualCastPreservesCurrentAction(t *testing.T) {
	for _, bad := range []string{"mana", "unknown", "bounds", "stone", "off-map"} {
		t.Run(bad, func(t *testing.T) {
			w := manualCastWorld(t)
			if !w.beginBookSpellOnce(0, 2, 1) {
				t.Fatal("predecessor refused")
			}
			cmd := Command{Kind: KindCastAt, Entity: 1, Spell: 26, X: 6, Y: 3}
			switch bad {
			case "mana":
				w.entities[0].Mana = 5
			case "unknown":
				cmd.Spell = 29
			case "bounds":
				cmd.X = -1
			case "stone":
				w.attached = []attachedEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption, Mode: EffectDuration, Remaining: 200}}
			case "off-map":
				w.entities[0].OffMap = true
			}
			back := retreatRoundTrip1089(t, w)
			Step(w, []Command{cmd})
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("invalid manual cast changed its predecessor")
			}
		})
	}
}
