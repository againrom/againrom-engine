package sim

import "testing"

func cadenceEquipWeightedWeapon(t *testing.T, w *World, actor EntityID, weight int32) {
	t.Helper()
	i := indexOfEntity(w.entities, actor)
	if i < 0 {
		t.Fatalf("world holds no actor %d", actor)
	}
	w.equipment[i][slotWeapon] = ItemInstance{Code: 0x1001}
	if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x1001, Weight: weight}}); err != nil {
		t.Fatalf("DeclareItemWeights: %v", err)
	}
}

func TestCadence1045PhysicalDistanceAndSchedulerTerms(t *testing.T) {
	if got := rangedExtra(cbEnt(1, 0, 0), cbEnt(2, 4, 0)); got != 5 {
		t.Fatalf("four-cell ranged extra = %d, want 5", got)
	}
	a := cbFighter(1, 0, 0, 1, 0)
	a.Reach = 4
	v := cbEnt(2, 4, 0)
	v.HP, v.MaxHP = 1<<20, 1<<20
	w := cbWorld(t, 1045, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	if got := cbAt(t, w, 2).HP; got != 1<<20 {
		t.Fatalf("the ranged blow landed before its flight term: hp %d", got)
	}
	for tick := 2; tick <= 5; tick++ {
		Step(w, nil)
		if got := cbAt(t, w, 2).HP; got != 1<<20 {
			t.Fatalf("the ranged blow landed on advance %d, want 6", tick)
		}
	}
	Step(w, nil)
	if got := cbAt(t, w, 2).HP; got == 1<<20 {
		t.Fatal("the ranged blow did not land on charge 1 + extra 5")
	}

	adjacent := cbWorld(t, 1045, cbFighter(1, 0, 0, 1, 0), cbEnt(2, 1, 0))
	Step(adjacent, []Command{cbOrder(1, 2)})
	first := cbAt(t, adjacent, 2).HP
	for tick := 1; tick < 3; tick++ {
		Step(adjacent, nil)
		if got := cbAt(t, adjacent, 2).HP; got != first {
			t.Fatalf("a second blow landed after %d tick(s), before the two scheduler turns", tick)
		}
	}
}

func TestCadence1045PhysicalIntervalCarriesEveryTerm(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 2)
	a.Humanoid, a.Reaction, a.Reach = true, 20, 4
	v := cbEnt(2, 4, 0)
	v.HP, v.MaxHP = 1<<20, 1<<20
	w := cbWorld(t, 1045, a, v)
	cadenceEquipWeightedWeapon(t, w, 1, 70) // penalty 10

	var blows []uint64
	last := cbAt(t, w, 2).HP
	commands := []Command{cbOrder(1, 2)}
	for len(blows) < 4 && w.Tick() < 200 {
		Step(w, commands)
		commands = nil
		if hp := cbAt(t, w, 2).HP; hp != last {
			blows = append(blows, w.Tick())
			last = hp
		}
	}
	if len(blows) != 4 {
		t.Fatalf("got %d physical applications in 200 ticks", len(blows))
	}
	for i := 1; i < len(blows); i++ {
		if gap := blows[i] - blows[i-1]; gap < 20 || gap > 23 {
			t.Errorf("physical gap = %d, want charge 1 + ranged 5 + relax 2 + U[0,3] + penalty 10 + boundary 2", gap)
		}
	}
}

func TestCadence1045HumanoidPenaltyUsesTheRuntimeWeaponWeight(t *testing.T) {
	build := func(t *testing.T, humanoid bool, reaction, weight int32, equipped bool) *World {
		t.Helper()
		a := cbEnt(1, 0, 0)
		a.Humanoid, a.Reaction = humanoid, reaction
		stock := []Stock{{ID: 1}}
		if equipped {
			stock[0].Equipped[slotWeapon] = 0x1001
		}
		w, err := NewStockedWorld(1, Bounds{Width: 2, Height: 2}, ModeCanonical, Terrain{},
			[]Entity{a}, nil, Relations{}, nil, stock)
		if err != nil {
			t.Fatalf("NewStockedWorld: %v", err)
		}
		if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x1001, Weight: weight}}); err != nil {
			t.Fatalf("DeclareItemWeights: %v", err)
		}
		return w
	}
	for _, tc := range []struct {
		name                       string
		humanoid, equipped         bool
		reaction, weight, expected int32
	}{
		{"equipped humanoid", true, true, 20, 70, 10},
		{"non-humanoid", false, true, 20, 70, 0},
		{"unarmed humanoid", true, false, 20, 70, 0},
		{"negative result clamps", true, true, 50, 0, 0},
		{"large result clamps", true, true, 0, 1000, 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := build(t, tc.humanoid, tc.reaction, tc.weight, tc.equipped)
			if got := w.humanoidPenalty(0); got != int64(tc.expected) {
				t.Errorf("penalty = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestCadence1045RetainedBookIntervalIncludesComplicationAndBoundaries(t *testing.T) {
	rule := SpellRule{ID: 1, Complication: 10, ManaCost: 1, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 50, 100, 100, 1<<1)
	caster.AttackCharge, caster.AttackRelax = 1, 2
	victim := spEnt(2, 3, 0)
	victim.HP, victim.MaxHP = 1<<20, 1<<20
	w := spWorld(t, 1045, []SpellRule{rule}, caster, victim)
	w.entities[0].Humanoid, w.entities[0].Reaction = true, 20
	cadenceEquipWeightedWeapon(t, w, 1, 70) // penalty 10
	if !w.beginBookSpell(0, 2, 1) {
		t.Fatal("retained AI producer refused the fixture")
	}
	var releases []uint64
	for len(releases) < 3 && w.Tick() < 200 {
		if events := StepObserved(w, nil); len(events) != 0 {
			releases = append(releases, w.Tick())
		}
	}
	if len(releases) != 3 {
		t.Fatalf("got %d retained releases in 200 ticks", len(releases))
	}
	for i := 1; i < len(releases); i++ {
		gap := releases[i] - releases[i-1]
		if gap < 32 || gap > 35 {
			t.Errorf("release gap = %d, want charge 8 + relax 2 + U[0,3] + penalty 10 + complication 10 + boundary 2", gap)
		}
	}
}

func TestCadence1045MageWeaponDiversionOmitsFloorRangeAndComplication(t *testing.T) {
	rule := wpnRule(1, 1, 8)
	rule.Complication = 99
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 2)
	caster.Humanoid, caster.Reaction = true, 20
	victim := wpnEnt(2, 4, 0)
	victim.HP, victim.MaxHP = 1<<20, 1<<20
	w := spWorld(t, 1045, []SpellRule{rule}, caster, victim)
	cadenceEquipWeightedWeapon(t, w, 1, 70) // penalty 10

	var releases []uint64
	commands := []Command{cbOrder(1, 2)}
	for len(releases) < 4 && w.Tick() < 200 {
		if events := StepObserved(w, commands); len(events) != 0 {
			releases = append(releases, w.Tick())
		}
		commands = nil
	}
	if len(releases) != 4 {
		t.Fatalf("got %d weapon-diverted releases in 200 ticks", len(releases))
	}
	for i := 1; i < len(releases); i++ {
		if gap := releases[i] - releases[i-1]; gap < 15 || gap > 18 {
			t.Errorf("weapon-divert gap = %d, want charge 1 + relax 2 + U[0,3] + penalty 10 + boundary 2", gap)
		}
	}
}

func TestCadence1045FighterWeaponSpellIsARiderOnThePhysicalCycle(t *testing.T) {
	rule := wpnRule(1, 1, 8)
	rule.Complication = 99
	fighter := wpnCaster(1, 0, 0, 1, 30, 1, 2)
	fighter.MaxMana, fighter.Mana = 0, 0
	fighter.Humanoid, fighter.Reaction, fighter.Reach = true, 20, 4
	fighter.DamageBase, fighter.AlwaysHits = 10, true
	victim := wpnEnt(2, 4, 0)
	victim.HP, victim.MaxHP = 1<<20, 1<<20
	w := spWorld(t, 1045, []SpellRule{rule}, fighter, victim)
	cadenceEquipWeightedWeapon(t, w, 1, 70) // penalty 10

	var strikes []uint64
	commands := []Command{cbOrder(1, 2)}
	last := cbAt(t, w, 2).HP
	for len(strikes) < 4 && w.Tick() < 200 {
		events := StepObserved(w, commands)
		commands = nil
		if hp := cbAt(t, w, 2).HP; hp != last {
			if len(events) == 0 {
				t.Fatalf("physical strike on tick %d produced no rider event", w.Tick())
			}
			strikes = append(strikes, w.Tick())
			last = hp
		}
	}
	if len(strikes) != 4 {
		t.Fatalf("got %d fighter strikes in 200 ticks", len(strikes))
	}
	for i := 1; i < len(strikes); i++ {
		if gap := strikes[i] - strikes[i-1]; gap < 20 || gap > 23 {
			t.Errorf("fighter-rider gap = %d, want the physical interval without a second recovery", gap)
		}
	}
}

func TestCadence1045InsufficientManaRetainsRetryState(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 50, 10, 0, 1<<1)
	victim := spEnt(2, 3, 0)
	victim.HP, victim.MaxHP = 1<<20, 1<<20
	w := spWorld(t, 1045, []SpellRule{rule}, caster, victim)
	if !w.beginBookSpell(0, 2, 1) {
		t.Fatal("retained AI producer refused the pending fixture")
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookPending || w.bookCasts[0].Complete {
		t.Fatalf("fresh refusal state = %+v, want incomplete pending order", w.bookCasts)
	}
	for i := 0; i < 3; i++ {
		Step(w, nil)
		if w.bookCasts[0].Progress != 0 {
			t.Fatalf("incomplete retry %d advanced progress to %d", i, w.bookCasts[0].Progress)
		}
	}
	w.entities[0].Mana = 10
	Step(w, nil)
	if w.bookCasts[0].Phase != bookCharging || w.bookCasts[0].Remaining != castPeriod {
		t.Fatalf("later mana produced %+v, want the same order charging from %d", w.bookCasts[0], castPeriod)
	}

	w.bookCasts[0].Phase, w.bookCasts[0].Remaining = bookPending, 0
	w.bookCasts[0].Complete, w.bookCasts[0].Progress = true, 0
	w.entities[0].Mana = 0
	for attempt, want := range []uint8{1, 2, 3, 0} {
		Step(w, nil)
		if got := w.bookCasts[0].Progress; got != want {
			t.Fatalf("completed retry tick %d progress = %d, want %d", attempt+1, got, want)
		}
	}
}

func TestCadence1045InsufficientManaSeparatesOneShotAndRetainedProducers(t *testing.T) {
	t.Run("retained AI order", func(t *testing.T) {
		rule := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
			DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
		w := spWorld(t, 1045, []SpellRule{rule},
			spMage(1, 0, 0, 50, 10, 0, 1<<1), spEnt(2, 2, 0))
		if !w.beginBookSpell(0, 2, 1) {
			t.Fatal("retained AI producer refused the fixture")
		}
		if len(w.bookCasts) != 1 || !w.bookCasts[0].Retained || w.bookCasts[0].Target != 2 {
			t.Fatalf("AI refusal did not retain its order: %+v", w.bookCasts)
		}
	})

	t.Run("armed offensive one-shot returns to selection", func(t *testing.T) {
		caster := acCaster(1, 0, 0, 0, 1<<1, 1)
		caster.Owner = 1
		first, second := spEnt(2, 2, 0), spEnt(3, 4, 0)
		first.Owner, second.Owner = 2, 2
		w := hlWorld(t, 1045, acEnemies(t), []SpellRule{hlArrow()}, caster, first, second)
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatalf("insufficient offensive one-shot retained %+v", w.bookCasts)
		}
		w.entities[indexOfEntity(w.entities, 2)].X = 15
		w.entities[indexOfEntity(w.entities, 1)].Mana = 3
		Step(w, nil)
		if len(w.bookCasts) != 1 || w.bookCasts[0].Retained || w.bookCasts[0].Target != 3 {
			t.Fatalf("restored producer did not select the current enemy: %+v", w.bookCasts)
		}
	})

	t.Run("armed restorative one-shot returns to selection", func(t *testing.T) {
		caster := acCaster(1, 0, 0, 0, 1<<6, 6)
		first, second := spEnt(2, 2, 0), spEnt(3, 4, 0)
		first.HP, second.HP = 20, 30
		w := hlWorld(t, 1045, Relations{}, []SpellRule{hlHeal()}, caster, first, second)
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatalf("insufficient restorative one-shot retained %+v", w.bookCasts)
		}
		w.entities[indexOfEntity(w.entities, 2)].HP = 100
		w.entities[indexOfEntity(w.entities, 1)].Mana = 10
		Step(w, nil)
		if len(w.bookCasts) != 1 || w.bookCasts[0].Retained || w.bookCasts[0].Target != 3 {
			t.Fatalf("restored producer did not select the current patient: %+v", w.bookCasts)
		}
	})

	t.Run("unarmed idle Heal remains behind its affordability gate", func(t *testing.T) {
		caster := acCaster(1, 0, 0, 10, 1<<6, 0)
		patient := spEnt(2, 2, 0)
		patient.HP = 20
		w := hlWorld(t, 1045, Relations{}, []SpellRule{hlHeal()}, caster, patient)
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatalf("unarmed unaffordable Heal retained %+v", w.bookCasts)
		}
	})

	t.Run("non-retained pending residue is discarded", func(t *testing.T) {
		rule := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
			DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
		w := spWorld(t, 1045, []SpellRule{rule},
			spMage(1, 0, 0, 50, 10, 0, 1<<1), spEnt(2, 2, 0))
		w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1, Phase: bookPending}}
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatalf("one-shot pending residue survived: %+v", w.bookCasts)
		}
	})
}

func TestCadence1045ActorDeathCancelsAnInFlightCast(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 1, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	w := spWorld(t, 1045, []SpellRule{rule}, spMage(1, 0, 0, 50, 10, 10, 1<<1), spEnt(2, 3, 0))
	Step(w, []Command{spCast(1, 2, 1)})
	w.entities[0].HP = 0
	Step(w, nil)
	if len(w.bookCasts) != 0 {
		t.Fatalf("dead caster kept %+v", w.bookCasts)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("dead caster applied a spell: target hp %d", got)
	}
}

func TestCadence1045ActorDeathCancelsAnInFlightPhysicalAction(t *testing.T) {
	w := cbWorld(t, 1045, cbFighter(1, 0, 0, 8, 2), cbEnt(2, 1, 0))
	Step(w, []Command{cbOrder(1, 2)})
	w.entities[0].HP = 0
	Step(w, nil)
	if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		t.Fatalf("dead attacker kept target=%v phase=%d countdown=%d", e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
	}
	if got := cbAt(t, w, 2).HP; got != 100 {
		t.Errorf("dead attacker applied a blow: target hp %d", got)
	}
}

func TestCadence1045ActorDeathCancelsRecoveryResidue(t *testing.T) {
	t.Run("book", func(t *testing.T) {
		rule := SpellRule{ID: 1, ManaCost: 1, School: 1, MaxRange: 8,
			DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
		caster := spMage(1, 0, 0, 1, 12, 10, 1<<1)
		w := spWorld(t, 1045, []SpellRule{rule}, caster, spEnt(2, 3, 0))
		if !w.beginBookSpell(0, 2, 1) {
			t.Fatal("retained AI producer refused the fixture")
		}
		for i := 0; i < 20 && (len(w.bookCasts) == 0 || !w.bookCasts[0].Complete); i++ {
			Step(w, nil)
		}
		if len(w.bookCasts) != 1 || !w.bookCasts[0].Complete {
			t.Fatalf("book cast did not reach recovery: %+v", w.bookCasts)
		}
		w.entities[0].HP = 0
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatalf("dead caster kept recovery %+v", w.bookCasts)
		}
	})

	t.Run("physical", func(t *testing.T) {
		w := cbWorld(t, 1045, cbFighter(1, 0, 0, 1, 12), cbEnt(2, 1, 0))
		Step(w, []Command{cbOrder(1, 2)})
		if e := cbAt(t, w, 1); e.AttackPhase != AttackRelaxing {
			t.Fatalf("physical action reached phase %d, want recovery", e.AttackPhase)
		}
		w.entities[0].HP = 0
		Step(w, nil)
		if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
			t.Fatalf("dead attacker kept recovery target=%v phase=%d countdown=%d",
				e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
		}
	})
}

func TestCadence1045DeathClearsCastWaitFromEveryReleaseProducer(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 1, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	for _, tc := range []struct {
		name string
		arm  func(*World)
	}{
		{
			name: "one-shot autocast",
			arm: func(w *World) {
				w.entities[0].AutoSpell = 1
			},
		},
		{
			name: "moving retained cast",
			arm: func(w *World) {
				Step(w, []Command{spCast(1, 2, 1)})
				Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 8}})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 0, 0, 50, 10, 10, 1<<1)
			caster.AttackCharge, caster.AttackRelax = 1, 12
			caster.Owner = 1
			victim := spEnt(2, 2, 0)
			victim.Owner = 2
			w := hlWorld(t, 1045, acEnemies(t), []SpellRule{rule}, caster, victim)
			tc.arm(w)
			for n := 0; n < 32 && spAt(t, w, 1).CastWait == 0; n++ {
				Step(w, nil)
			}
			if got := spAt(t, w, 1).CastWait; got == 0 {
				t.Fatal("release did not move recovery into CastWait")
			}
			Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 100}})
			if got := spAt(t, w, 1); got.Alive() || got.CastWait != 0 {
				t.Fatalf("felled caster is alive=%v with CastWait=%d", got.Alive(), got.CastWait)
			}
		})
	}
}

func TestCadence1045PhysicalApplicationRechecksReach(t *testing.T) {
	w := cbWorld(t, 1045, cbFighter(1, 0, 0, 3, 2), cbEnt(2, 1, 0))
	Step(w, []Command{cbOrder(1, 2)})
	w.entities[1].X = 8
	for range 3 {
		Step(w, nil)
	}
	if got := cbAt(t, w, 2).HP; got != 100 {
		t.Errorf("target moved out of reach during charge and was struck to %d", got)
	}
}

func TestCadence1045PhysicalApplicationCancelsWhenTheTargetIsRemoved(t *testing.T) {
	w := cbWorld(t, 1045, cbFighter(1, 0, 0, 3, 2), cbEnt(2, 1, 0))
	Step(w, []Command{cbOrder(1, 2)})
	w.entities = w.entities[:1]
	for range 3 {
		Step(w, nil)
	}
	if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		t.Fatalf("removed target left attack target=%v phase=%d countdown=%d",
			e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
	}
}

func TestCadence1045TheStrikeCrossingMinusTenClearsTheCycle(t *testing.T) {
	victim := cbEnt(3, 1, 0)
	victim.HP, victim.MaxHP = 4, 4
	w := cbWorld(t, 1045, cbFighter(1, 0, 0, 1, 20), cbFighter(2, 2, 0, 1, 20), victim)
	Step(w, []Command{cbOrder(1, 3), cbOrder(2, 3)})
	if got := cbAt(t, w, 3).HP; got != -16 {
		t.Errorf("two ten-point strikes left target at %d, want -16", got)
	}
	if cbAt(t, w, 2).HasAttackTarget {
		t.Fatal("the strike that crossed -10 left its retained attack target")
	}
}

func TestCadence1045NegativeLinkedTargetSurvivesUntilTeardown(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rider bool
	}{
		{"physical", false},
		{"fighter rider", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attacker := cbFighter(1, 0, 0, 1, 20)
			attacker.DamageBase = 1
			attacker.AttackTarget, attacker.HasAttackTarget = 2, true
			var spells []SpellRule
			var bystander Entity
			if tc.rider {
				attacker.WeaponSpell, attacker.WeaponSpellLevel = 2, 40
				attacker.MaxMana, attacker.Mana = 0, 0
				spells = []SpellRule{{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 5,
					DamageMin: 1, DamageMax: 1, Damaging: true}}
				bystander = spEnt(3, 1, 1)
			}
			victim := spEnt(2, 1, 0)
			victim.HP, victim.Decay, victim.Dwell = -1, DecayFallen, 2
			ents := []Entity{attacker, victim}
			if tc.rider {
				ents = append(ents, bystander)
			}
			w := spWorld(t, 1045, spells, ents...)
			beforeBystander := int32(0)
			if tc.rider {
				beforeBystander = spAt(t, w, 3).HP
			}
			events := StepObserved(w, nil)
			if got := spAt(t, w, 2).HP; got >= -1 {
				t.Fatalf("negative linked target was not struck: hp %d", got)
			}
			if !spAt(t, w, 1).HasAttackTarget {
				t.Fatal("health alone cleared the retained attack before teardown")
			}
			if tc.rider {
				if len(events) == 0 || spAt(t, w, 3).HP >= beforeBystander {
					t.Fatalf("fighter rider did not apply through the negative primary target: events=%+v bystander=%d", events, spAt(t, w, 3).HP)
				}
			}

			vi := indexOfEntity(w.entities, 2)
			w.entities[vi].HP, w.entities[vi].Dwell = decayGoneHP-1, 0
			Step(w, nil)
			if indexOfEntity(w.entities, 2) >= 0 {
				t.Fatal("decay teardown kept the target entity")
			}
			if e := spAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
				t.Fatalf("next-tick teardown left target=%v phase=%d countdown=%d", e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
			}
		})
	}
}
