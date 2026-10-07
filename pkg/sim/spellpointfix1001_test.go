package sim

import "testing"

// spSlowRow and spFreezeRow are the two shipped rows that both attach
// EffectSpeed at −(power/15 + 1): Slow (28) and Freezing Cloud's inner effect
// (7). They are different spell ids, so the same-id no-stacking rule of
// MAGIC-ATTACH-016 does not merge them and an actor can carry both.
func spSlowRow() SpellRule {
	return SpellRule{ID: 28, ManaCost: 1, School: 3, MaxRange: 8, TargetsUnit: true,
		SpellDuration: 30, EffectKind: EffectSpeed, EffectMode: EffectDuration}
}

func spFreezeRow() SpellRule {
	return SpellRule{ID: 7, ManaCost: 1, School: 2, MaxRange: 8, TargetsUnit: true,
		SpellDuration: 30, EffectKind: EffectSpeed, EffectMode: EffectDuration,
		EffectDuration: 400}
}

// A2. Two speed-reducing effects on one actor never take its speed to zero or
// below. At zero, moverSpeed makes the actor UNRATED and step.go gives an
// unrated mover the cell-per-tick cadence, which is the fastest the engine has:
// before the floor, Slow plus Freezing Cloud on a speed-10 actor at power 60
// produced a unit about 21 times faster than the same actor unslowed.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the minEffectSpeed clamp in
// applyEffectDelta (pkg/sim/effect.go) and rerun.
func TestTwoSlowingEffectsLeaveAnActorSlowRatherThanUnrated(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<28|1<<7)
	caster.Mind = 100
	victim := spEnt(2, 3, 2)
	victim.Speed, victim.TokenSize = 10, 1
	w := hlWorld(t, 0x1001, Relations{}, []SpellRule{spSlowRow(), spFreezeRow()}, caster, victim)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 28})
	first := spAt(t, w, 2).Speed
	if first >= 10 {
		t.Fatalf("Slow left speed at %d, want a fall from 10 — the assertion below would witness nothing", first)
	}
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 7})
	second := spAt(t, w, 2).Speed

	if second < minEffectSpeed {
		t.Fatalf("Slow and Freezing Cloud together left speed %d, below the floor of %d", second, minEffectSpeed)
	}
	if !rated(spAt(t, w, 2)) {
		t.Fatalf("the doubly slowed actor is unrated at speed %d, which is the engine's fastest cadence", second)
	}
	if second > first {
		t.Fatalf("the second slowing effect RAISED speed from %d to %d", first, second)
	}
}

// A2, the other half: what falls comes back. The floor stores what actually
// landed, so expiry restores the original speed exactly rather than adding back
// a magnitude the clamp swallowed.
func TestASlowedActorGetsExactlyItsSpeedBack(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<28|1<<7)
	caster.Mind = 100
	victim := spEnt(2, 3, 2)
	victim.Speed, victim.TokenSize = 3, 1
	w := hlWorld(t, 0x1002, Relations{}, []SpellRule{spSlowRow(), spFreezeRow()}, caster, victim)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 28})
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 7})
	if got := spAt(t, w, 2).Speed; got != minEffectSpeed {
		t.Fatalf("a speed-3 actor under both rows is at %d, want the floor %d", got, minEffectSpeed)
	}
	for i := 0; i < 20000 && len(w.attached) > 0; i++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 2).Speed; got != 3 {
		t.Fatalf("after both effects expired speed is %d, want the original 3", got)
	}
}

// B1. Bless's magnitude is not a protection. The recompute walks five
// protection slots and only four protection effect kinds exist, so
// EffectProtectionFire + 4 is EffectBless — a probability in [20,100] landing
// in the Astral row, hashed, serialized and displayed.
//
// TO CONFIRM IT WITNESSES THE FIX, restore the unbounded
// `c.Protection[k] + w.effectDelta(id, EffectProtectionFire+EffectKind(k))` in
// SetCombat (pkg/sim/rearm.go) and rerun.
func TestBlessDoesNotLandInTheAstralProtectionSlot(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<23)
	caster.Mind = 100
	target := spEnt(2, 3, 2)
	target.TokenSize = 1
	rule := SpellRule{ID: 23, ManaCost: 1, School: 5, MaxRange: 8, TargetsUnit: true,
		SpellDuration: 30, Defensive: true}
	w := hlWorld(t, 0x1003, Relations{}, []SpellRule{rule}, caster, target)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 23})
	mag, ok := w.attachedMagnitude(2, 23)
	if !ok || mag <= 0 {
		t.Fatalf("Bless did not attach (magnitude %d, ok %v) — the assertion below would witness nothing", mag, ok)
	}
	if !w.SetCombat(2, CombatBlock{}) {
		t.Fatal("SetCombat(2) was refused")
	}
	if got := spAt(t, w, 2).Protection; got != ([5]int32{}) {
		t.Fatalf("after a recompute on a blessed actor Protection = %v, want all zero — "+
			"the Bless magnitude of %d reached a protection slot", got, mag)
	}
}

// B2. Apply and expire are exact inverses under the clamps. A Protection
// effect that clamps at 100 on the way in must not subtract its whole nominal
// magnitude on the way out, and a Darkness that clamps ScanRange at 0 must not
// give back more sight than it took.
//
// TO CONFIRM IT WITNESSES THE FIX, make applyEffectDelta return the nominal
// amount instead of the landed delta, or stop attachEffect storing what landed
// (pkg/sim/effect.go), and rerun.
func TestAClampedEffectGivesBackExactlyWhatItTook(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rule  SpellRule
		spell int32
		setup func(e *Entity)
		check func(t *testing.T, e Entity)
	}{
		{
			name: "protection clamped at its ceiling",
			rule: SpellRule{ID: 5, ManaCost: 1, School: 1, MaxRange: 8, TargetsUnit: true,
				Defensive: true, SpellDuration: 4, EffectKind: EffectProtectionFire,
				EffectMode: EffectDuration},
			spell: 5,
			setup: func(e *Entity) { e.Protection[0] = 80 },
			check: func(t *testing.T, e Entity) {
				if e.Protection[0] != 80 {
					t.Fatalf("fire protection is %d after the effect expired, want the base 80", e.Protection[0])
				}
			},
		},
		{
			name: "scan range clamped at its floor",
			rule: SpellRule{ID: 17, ManaCost: 1, School: 3, MaxRange: 8, TargetsUnit: true,
				SpellDuration: 4, EffectKind: EffectScanRange, EffectMode: EffectDuration,
				EffectDuration: 20},
			spell: 17,
			setup: func(e *Entity) { e.ScanRange = 2 },
			check: func(t *testing.T, e Entity) {
				if e.ScanRange != 2 {
					t.Fatalf("scan range is %d after the effect expired, want the base 2", e.ScanRange)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 2, 2, 1<<uint(tc.spell))
			caster.Mind = 100
			target := spEnt(2, 3, 2)
			target.TokenSize = 1
			tc.setup(&target)
			w := hlWorld(t, 0x1004, Relations{}, []SpellRule{tc.rule}, caster, target)

			spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: tc.spell})
			if len(w.attached) == 0 {
				t.Fatal("nothing attached — the assertion below would witness nothing")
			}
			for i := 0; i < 20000 && len(w.attached) > 0; i++ {
				Step(w, nil)
			}
			tc.check(t, spAt(t, w, 2))
		})
	}
}

// A refused Control Spirit remains free; a blocked admitted Teleport is paid.
func TestControlSpiritRefusalAndTeleportPlacementHaveDistinctCosts(t *testing.T) {
	t.Run("control spirit at a fallen body", func(t *testing.T) {
		caster := effectMage(1, 1, 1, 1<<25)
		corpse := spEnt(2, 2, 1)
		w := hlGhostWorld(t, 0x1005, []SpellRule{{ID: 25, ManaCost: 30, TargetsUnit: true, MaxRange: 4}},
			hlGhostTemplate(), caster, corpse)
		w.entities[1].HP, w.entities[1].Decay = -10, DecayFallen
		mana := spAt(t, w, 1).Mana
		if r := w.BookSpellRefusal(1, 2, 25); r == "" {
			t.Error("BookSpellRefusal reports a fallen body as an admissible Control Spirit target")
		}
		spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})
		if got := spAt(t, w, 1).Mana; got != mana {
			t.Fatalf("a refused Control Spirit spent %d mana", mana-got)
		}
		if len(w.entities) != 2 {
			t.Fatalf("the world holds %d entities, want the untouched 2", len(w.entities))
		}
	})

	t.Run("teleport onto a blocked cell", func(t *testing.T) {
		caster := effectMage(1, 1, 1, 1<<26)
		beacon := spEnt(2, 4, 4)
		beacon.TokenSize = 1
		grid := make([]byte, 16*16)
		grid[4*16+4] = 1
		w, err := NewStockedSpelledWorld(0x1006, Bounds{Width: 16, Height: 16}, ModeCanonical,
			Terrain{Block: grid}, []Entity{caster, beacon}, nil, Relations{}, nil, nil,
			[]SpellRule{{ID: 26, ManaCost: 60, TargetsUnit: true, MaxRange: 8}})
		if err != nil {
			t.Fatalf("NewStockedSpelledWorld: %v", err)
		}
		mana := spAt(t, w, 1).Mana
		if r := w.BookSpellRefusal(1, 2, 26); r != "" {
			t.Errorf("blocked relocation refused the cast: %s", r)
		}
		spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 26})
		got := spAt(t, w, 1)
		if got.Mana != mana-60 {
			t.Fatalf("blocked Teleport spent %d mana, want 60", mana-got.Mana)
		}
		if got.X != 1 || got.Y != 1 {
			t.Fatalf("the caster stands at (%d,%d), want its own (1,1)", got.X, got.Y)
		}
	})

	t.Run("a duration row whose computed duration is zero", func(t *testing.T) {
		caster := effectMage(1, 2, 2, 1<<5)
		target := spEnt(2, 3, 2)
		target.TokenSize = 1
		// SpellDuration 0 with a duration mode: lastingTicks answers 0, so
		// attachEffect refuses and nothing lands.
		rule := SpellRule{ID: 5, ManaCost: 7, School: 1, MaxRange: 8, TargetsUnit: true,
			Defensive: true, SpellDuration: 0, EffectKind: EffectProtectionFire,
			EffectMode: EffectDuration}
		w := hlWorld(t, 0x1007, Relations{}, []SpellRule{rule}, caster, target)
		mana := spAt(t, w, 1).Mana
		if r := w.BookSpellRefusal(1, 2, 5); r == "" {
			t.Error("BookSpellRefusal admits a row whose computed duration is zero")
		}
		spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 5})
		if got := spAt(t, w, 1).Mana; got != mana {
			t.Fatalf("a cast that attached nothing spent %d mana", mana-got)
		}
		if len(w.attached) != 0 {
			t.Fatalf("%d effect(s) attached on a zero duration", len(w.attached))
		}
	})
}

// C3. Prismatic Spray walks every entity in radius. A corpse in that radius
// took a damage roll — consuming world RNG, which moves every later draw — and
// paid the caster school experience.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the liveness guard at the head of
// ordinaryEffect (pkg/sim/spell.go) and rerun.
func TestPrismaticSprayPassesOverCorpses(t *testing.T) {
	rule := SpellRule{ID: 14, ManaCost: 4, School: 1, MaxRange: 8, TargetsUnit: true,
		Damaging: true, DamageMin: 5, DamageMax: 25}
	// The control is a world with NO corpses at all. A corpse the spray passes
	// over must leave the same world behind as no corpse: the same victim
	// health, which is the generator's position, and the same banked school
	// experience.
	// DIV-072's filter reads owners now, so the victim and every corpse are
	// owned by a slot the caster's own slot is hostile toward — the fixture
	// this test measures (corpses vs. no corpses) is independent of that
	// filter, but the spray must still land on ANYTHING for the comparison
	// below to witness what it names.
	rel := engRel(t, [3]uint32{1, 2, relationHostile})
	build := func(corpses bool) *World {
		caster := effectMage(1, 2, 2, 1<<14)
		caster.GainsXP, caster.Skill[1], caster.Owner, caster.TypeID = true, 1, 1, HumanTypeID
		victim := spEnt(2, 3, 2)
		victim.TokenSize, victim.Owner = 1, 2
		ents := []Entity{caster, victim}
		if corpses {
			// Inside the spray's own radius: power is 1 here, so the radius is 2.
			for id, c := range []cell{{x: 4, y: 2}, {x: 5, y: 2}, {x: 4, y: 3}, {x: 5, y: 3}} {
				e := spEnt(EntityID(3+id), c.x, c.y)
				e.TokenSize, e.HP, e.Decay, e.Owner = 1, -10, DecayFallen, 2
				ents = append(ents, e)
			}
		}
		return hlWorld(t, 0x1008, rel, []SpellRule{rule}, ents...)
	}

	bare, dead := build(false), build(true)
	spRunCast(bare, Command{Kind: KindCast, Entity: 1, X: 2, Y: 14})
	spRunCast(dead, Command{Kind: KindCast, Entity: 1, X: 2, Y: 14})

	if spAt(t, bare, 2).HP == 100 {
		t.Fatal("the spray did not damage its living victim — the comparison below would witness nothing")
	}
	if got, want := spAt(t, dead, 2).HP, spAt(t, bare, 2).HP; got != want {
		t.Errorf("with four corpses in range the living victim is at %d health and with none at %d — "+
			"the corpses took rolls and moved the generator", got, want)
	}
	for id := EntityID(3); id <= 6; id++ {
		if got := spAt(t, dead, id).HP; got != -10 {
			t.Errorf("corpse %d is at %d health, want the untouched -10", id, got)
		}
	}
	if got, want := spAt(t, dead, 1).SkillXP[1], spAt(t, bare, 1).SkillXP[1]; got != want {
		t.Errorf("the caster banked %d school experience with four corpses in range and %d with none", got, want)
	}
}

// E2. Stone Curse's magnitude is the row's own `Effects` column
// (MAGIC-EFFECT-015: it is one of the two arms whose number is not overwritten
// by the arm), so editing the installed row changes what the spell does. It was
// a literal 5.
//
// TO CONFIRM IT WITNESSES THE FIX, restore `mag = 5` in pointEffect
// (pkg/sim/spell.go) and rerun.
func TestStoneCurseTakesItsMagnitudeFromTheInstalledRow(t *testing.T) {
	for _, mag := range []int32{5, 12} {
		caster := effectMage(1, 2, 2, 1<<20)
		target := spEnt(2, 3, 2)
		target.TokenSize = 1
		rule := SpellRule{ID: 20, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true,
			SpellDuration: 20, EffectKind: EffectAbsorption, EffectMode: EffectDuration,
			EffectMagnitude: mag}
		w := hlWorld(t, 0x1009, Relations{}, []SpellRule{rule}, caster, target)
		spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 20})
		if got := spAt(t, w, 2).Absorption; got != mag {
			t.Errorf("a Stone Curse row carrying absorbtion=%d raised absorption by %d", mag, got)
		}
	}
}

// E2, the second half: MAGIC-SING-019 (d) — the DURATION is multiplied by
// (100 − protectionEarth)/100 with a floor of one tick, the only place a
// resistance shortens an effect instead of reducing damage. Nothing exercised
// Stone Curse at all before this round.
func TestStoneCursesDurationIsShortenedByEarthProtection(t *testing.T) {
	cast := func(earth int32) uint16 {
		caster := effectMage(1, 2, 2, 1<<20)
		target := spEnt(2, 3, 2)
		target.TokenSize, target.Protection[3] = 1, earth
		rule := SpellRule{ID: 20, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true,
			SpellDuration: 20, EffectKind: EffectAbsorption, EffectMode: EffectDuration,
			EffectMagnitude: 5}
		w := hlWorld(t, 0x100a, Relations{}, []SpellRule{rule}, caster, target)
		spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 20})
		if len(w.attached) != 1 {
			t.Fatalf("earth %d: %d effect(s) attached, want 1", earth, len(w.attached))
		}
		return w.attached[0].Remaining
	}
	bare, half, full := cast(0), cast(50), cast(100)
	if half >= bare || half == 0 {
		t.Errorf("earth 50 gives %d ticks against %d unprotected", half, bare)
	}
	if full != 1 {
		t.Errorf("earth 100 gives %d ticks, want the floor of 1", full)
	}
}

// E3. MAGIC-ATTACH-016's untimed gate is `+0x3d & 7 == 0`, which covers
// `charges` (4) as well as duration and continuous. A charges row was treated
// as untimed: applied once, stored nowhere, reversible by nothing.
//
// TO CONFIRM IT WITNESSES THE FIX, narrow effectTimedModes back to
// EffectDuration|EffectContinuous (pkg/sim/spell.go) and rerun.
func TestAChargesModeRowIsStoredRatherThanAppliedAndForgotten(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<12)
	target := spEnt(2, 3, 2)
	target.TokenSize, target.ScanRange = 1, 5
	rule := SpellRule{ID: 12, ManaCost: 1, School: 3, MaxRange: 8, TargetsUnit: true,
		EffectKind: EffectScanRange, EffectMode: EffectCharges, EffectDuration: 40}
	w := hlWorld(t, 0x100b, Relations{}, []SpellRule{rule}, caster, target)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 12})
	if len(w.attached) != 1 {
		t.Fatalf("a charges-mode row attached %d effect(s), want 1", len(w.attached))
	}
	if got := spAt(t, w, 2).ScanRange; got <= 5 {
		t.Fatalf("scan range is %d, want a rise from 5", got)
	}
	remaining := w.attached[0].Remaining
	for i := 0; i < 80; i++ {
		Step(w, nil)
	}
	if len(w.attached) != 1 || w.attached[0].Remaining != remaining || spAt(t, w, 2).ScanRange <= 5 {
		t.Fatal("charges-only attachment incorrectly used a duration timer")
	}
}

// Coverage the review found missing: Shield, Light and Darkness are cast and
// their magnitudes read. Each is MAGIC-EFFECT-015's own per-arm expression.
func TestShieldLightAndDarknessLandTheirDecodedMagnitudes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    int32
		kind  EffectKind
		mind  int32
		mag   int32
		start int32
	}{
		{"shield p/10 + 3", 18, EffectAbsorption, 100, spellPower(0, 100)/10 + 3, 0},
		{"light p/30 + 1", 12, EffectScanRange, 100, spellPower(0, 100)/30 + 1, 5},
		{"darkness -1 - p/30", 17, EffectScanRange, 100, -1 - spellPower(0, 100)/30, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 2, 2, 1<<uint(tc.id))
			caster.Mind, caster.Skill[3] = tc.mind, 0
			target := spEnt(2, 3, 2)
			target.TokenSize = 1
			if tc.kind == EffectScanRange {
				target.ScanRange = uint8(tc.start)
			} else {
				target.Absorption = tc.start
			}
			rule := SpellRule{ID: uint16(tc.id), ManaCost: 1, School: 3, MaxRange: 8,
				TargetsUnit: true, SpellDuration: 20, EffectKind: tc.kind,
				EffectMode: EffectDuration, EffectDuration: 200}
			w := hlWorld(t, 0x100c, Relations{}, []SpellRule{rule}, caster, target)
			spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: tc.id})

			got := spAt(t, w, 2)
			have := got.Absorption
			if tc.kind == EffectScanRange {
				have = int32(got.ScanRange)
			}
			if have != tc.start+tc.mag {
				t.Fatalf("%s: the value is %d, want %d + %d", tc.name, have, tc.start, tc.mag)
			}
		})
	}
}

// F1. MAGIC-SING-019 (f) cancels invisibility at the owner's own melee
// APPROACH. Removing it at the landed blow left an attacker that missed its
// to-hit roll invisible, and one whose victim stood out of reach invisible for
// as long as it walked.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the removeAttachedSpell call in
// approach (pkg/sim/combat.go) and rerun.
func TestAnAttackersApproachCancelsItsOwnInvisibility(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<15)
	caster.Mind, caster.TokenSize, caster.Reach, caster.Speed = 100, 1, 1, 4
	// A victim far enough that the attacker walks for many ticks and never
	// lands a blow, and weak enough on to-hit that a landed one is not the
	// thing being measured.
	victim := spEnt(2, 12, 2)
	victim.TokenSize, victim.Owner, victim.Defence = 1, 2, 10000
	rule := SpellRule{ID: 15, ManaCost: 1, School: 5, MaxRange: 8, TargetsUnit: true,
		Defensive: true, SpellDuration: 20}
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile})
	w := hlWorld(t, 0x100d, rel, []SpellRule{rule}, caster, victim)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 1, Y: 15})
	if !w.hasAttachedSpell(1, 15) {
		t.Fatal("the caster is not invisible — the assertion below would witness nothing")
	}
	for spAt(t, w, 1).CastWait != 0 {
		Step(w, nil)
	}
	// Keep the order tick off the group-decision phase. The retained cadence
	// changes which absolute tick the helper returns on, while this witness is
	// about the melee approach rather than an unrelated group release.
	if w.tick%scriptCycle == scriptPassPhase {
		Step(w, nil)
	}
	w.orderAttack(0, 2)
	if !spAt(t, w, 1).HasAttackTarget {
		t.Fatalf("the attack order was not taken — tick=%d actor=%+v victim=%+v casts=%+v busy=%v targetInvisible=%v; the assertion below would witness nothing",
			w.tick, spAt(t, w, 1), spAt(t, w, 2), w.bookCasts, w.actorCastBusy(0), w.invisibleToActor(0, 1))
	}
	Step(w, nil)
	if w.hasAttackTarget(1) && w.hasAttachedSpell(1, 15) {
		t.Fatal("an attacker walking toward its victim is still invisible; MAGIC-SING-019 (f) " +
			"cancels it at the approach, not at a landed blow")
	}
}

// hasAttackTarget is a reader for the test above: whether the world still holds
// an attack order for id.
func (w *World) hasAttackTarget(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	return i >= 0 && w.entities[i].HasAttackTarget
}

// F3. A duration-mode health effect reverses on expiry, and the reversal can
// take its target below zero. Every other path that drives health down calls
// clearFelled; this one did not, so it could leave a felled actor holding an
// order — a state the byte form's own decoder refuses.
//
// TO CONFIRM IT WITNESSES THE FIX, delete the clearFelled call in
// removeAttachedAt (pkg/sim/effect.go) and rerun.
func TestAHealthEffectThatFellsItsTargetOnExpiryClearsTheOrder(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<9)
	caster.Mind, caster.Owner = 100, SelfSlot
	target := spEnt(2, 3, 2)
	target.TokenSize, target.HP, target.MaxHP, target.Speed = 1, 50, 100, 4
	target.Owner = SelfSlot
	// A duration-mode health row: it raises health on attach and takes the
	// same amount back on expiry, which is what fells a target this close to
	// zero.
	rule := SpellRule{ID: 9, ManaCost: 1, School: 1, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectKind: EffectHealth, EffectMode: EffectDuration,
		EffectMagnitude: 40, EffectDuration: 6}
	w := hlWorld(t, 0x100e, Relations{}, []SpellRule{rule}, caster, target)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 9})
	if spAt(t, w, 2).HP != 90 {
		t.Fatalf("the health effect did not land (health %d) — the assertion below would witness nothing",
			spAt(t, w, 2).HP)
	}
	// Wounded to one point while the effect stands, so that giving the forty
	// back on expiry is what takes the actor below zero.
	Step(w, []Command{{Kind: KindDamage, Entity: 2, X: 89}})
	if got := spAt(t, w, 2).HP; got != 1 {
		t.Fatalf("the wound left health at %d, want 1", got)
	}
	Step(w, []Command{{Kind: KindMoveTo, Entity: 2, X: 9, Y: 9}})
	if !spAt(t, w, 2).HasTarget {
		t.Fatal("the move order was not taken — the assertion below would witness nothing")
	}
	for i := 0; i < 200 && len(w.attached) > 0; i++ {
		Step(w, nil)
	}
	got := spAt(t, w, 2)
	if got.HP > 0 {
		t.Fatalf("the expiry left health at %d, want it at or below zero", got.HP)
	}
	if got.HasTarget {
		t.Fatal("a felled actor still holds a walk order after the effect expired")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatalf("the resulting world does not encode: %v", err)
	}
}

// The Teleport autocast, removed (owner): "давай телепорт
// пока не будем тогда трогать, но текущий
// автокаст его нужно убрать, потому что
// сейчас маг просто стоит на месте и
// тратит ману впустую".
//
// What an armed Teleport did, measured here rather than assumed: the shipped
// row carries `Spell Defensive` and is not restorative, so autoCastTarget's
// defensive shortcut answers with the CASTER'S OWN id. The arm then teleports
// the caster to the cell it is already standing on — an open cell, so nothing
// refuses it — and pays the row's 60 mana for a move of zero, clearing the
// caster's own walk order and route on the way.
//
// THIS IS NOT THE CAUSE OF THE STANDING-STILL SYMPTOM the owner reported
// beside it. That was measured by the orders lane and is general to every
// armed offensive row: the mover stood down for CastWait as well as for the
// wind-up, and stepAutoCasts runs before the mover in the tick. This removal
// does not fix it and must not be read as fixing it.
//
// TO CONFIRM IT WITNESSES THE FIX, make autoCastable return true for every row
// (pkg/sim/spell.go) and rerun.
func TestAnArmedTeleportNeitherFiresNorStaysArmed(t *testing.T) {
	// The shipped row's own columns: defensive, not a unit target, 60 mana.
	rule := SpellRule{ID: 26, ManaCost: 60, School: 5, MaxRange: 1, Defensive: true}
	caster := effectMage(1, 5, 5, 1<<26)
	caster.Mind, caster.MaxMana, caster.Mana, caster.TokenSize = 100, 10000, 10000, 1
	caster.AutoSpell = 26
	w := hlWorld(t, 0x100f, Relations{}, []SpellRule{rule}, caster)

	before := spAt(t, w, 1)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 9, Y: 9}})
	for i := 0; i < 400; i++ {
		Step(w, nil)
	}
	after := spAt(t, w, 1)

	if after.Mana != before.Mana {
		t.Errorf("an armed Teleport spent %d mana over 400 ticks", before.Mana-after.Mana)
	}
	if after.AutoSpell != 0 {
		t.Errorf("Teleport is still armed as spell %d, so the dashed border stands over a cell "+
			"that never fires", after.AutoSpell)
	}
}

// And the manual path is untouched: a player-issued Teleport still moves the
// caster. Only the unbidden path is removed.
func TestAPlayerIssuedTeleportStillWorks(t *testing.T) {
	rule := SpellRule{ID: 26, ManaCost: 60, School: 5, MaxRange: 8, Defensive: true}
	caster := effectMage(1, 2, 2, 1<<26)
	caster.Mind, caster.MaxMana, caster.Mana, caster.TokenSize = 100, 10000, 10000, 1
	beacon := spEnt(2, 6, 6)
	beacon.TokenSize = 1
	// Ground and air have distinct decoded occupant slots. A ground Teleport
	// may therefore share this cell with an air beacon while still refusing a
	// same-layer actor, as TestTeleportDoesNotCreateAnOccupiedCellOverlap pins.
	beacon.Domain = DomainAir
	w := hlWorld(t, 0x1010, Relations{}, []SpellRule{rule}, caster, beacon)

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 26})
	got := spAt(t, w, 1)
	if got.X != 6 || got.Y != 6 {
		t.Fatalf("the caster stands at (%d,%d) after a commanded Teleport, want the beacon's (6,6)", got.X, got.Y)
	}
	if got.Mana != 10000-60 {
		t.Fatalf("a commanded Teleport left %d mana, want %d", got.Mana, 10000-60)
	}
}
