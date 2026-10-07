package sim

import (
	"bytes"
	"reflect"
	"sort"
	"testing"
)

func effectMage(id EntityID, x, y int32, known uint32) Entity {
	e := spMage(id, x, y, 30, 100, 100, known)
	e.Owner = SelfSlot
	e.TypeID = HumanTypeID
	e.TokenSize = 1
	return e
}

func TestLastingPointEffectsRefreshAnnihilateExpireAndUnapply(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<24|1<<23|1<<27)
	target := spEnt(2, 3, 2)
	target.Speed = 10
	w := hlWorld(t, 1, Relations{}, nil, caster, target)

	haste := SpellRule{ID: 24, SpellDuration: 1, EffectKind: EffectSpeed, EffectMode: EffectDuration}
	if !w.ordinaryEffect(0, 1, haste, 30) {
		t.Fatal("haste did not attach")
	}
	if got := w.entities[1].Speed; got != 13 {
		t.Fatalf("haste speed = %d, want 13", got)
	}
	first := w.attached[0].Remaining
	Step(w, nil)
	if !w.ordinaryEffect(0, 1, haste, 30) || w.entities[1].Speed != 13 || w.attached[0].Remaining != first {
		t.Fatalf("refresh stacked or failed to restore duration: entity=%+v effects=%+v", w.entities[1], w.attached)
	}
	for len(w.attached) != 0 {
		Step(w, nil)
	}
	if got := w.entities[1].Speed; got != 10 {
		t.Fatalf("expired haste left speed %d, want original 10", got)
	}

	bless := SpellRule{ID: 23, SpellDuration: 1}
	curse := SpellRule{ID: 27, SpellDuration: 1}
	if !w.ordinaryEffect(0, 1, bless, 20) || len(w.attached) != 1 {
		t.Fatal("bless did not attach")
	}
	if !w.ordinaryEffect(0, 1, curse, 20) || len(w.attached) != 0 {
		t.Fatalf("curse did not annihilate bless: %+v", w.attached)
	}
}

func TestBlessAndCurseConsumeTheirAttachedProbabilityInThePhysicalDamageRoll(t *testing.T) {
	for _, tc := range []struct {
		name, spell string
		id          uint16
		maximum     bool
	}{
		{name: "bless selects the maximum", spell: "bless", id: 23, maximum: true},
		{name: "curse selects the minimum", spell: "curse", id: 27},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := spEnt(1, 2, 2)
			a.DamageBase, a.DamageSpread, a.AlwaysHits = 10, 20, true
			v := spEnt(2, 3, 2)
			w := hlWorld(t, 100, Relations{}, nil, a, v)
			if !w.ordinaryEffect(0, 0, SpellRule{ID: tc.id, SpellDuration: 1}, 100) {
				t.Fatalf("%s did not attach", tc.spell)
			}

			const state = uint64(0x1001)
			w.rng.state = state
			oracle := rng{state: state}
			normal := oracle.uniform(a.DamageSpread)
			chance := oracle.uniform(100)
			_ = oracle.uniform(hitRollSpan)
			want := int32(a.DamageBase) + normal
			if chance < 100 {
				if tc.maximum {
					want = a.DamageBase + a.DamageSpread
				} else {
					want = a.DamageBase
				}
			}
			w.resolveBlow(0, 1, nil)
			if got := int32(100) - w.entities[1].HP; got != want {
				t.Fatalf("%s damage = %d, want %d (inclusive chance draw %d)", tc.spell, got, want, chance)
			}
		})
	}
}

func TestInvisibilityRemovesAnActorFromNewEntityTargetAdmissionUntilItIsCancelled(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<1)
	target := spEnt(2, 3, 2)
	rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	w := hlWorld(t, 101, Relations{}, []SpellRule{rule}, caster, target)
	if !w.attachEffect(2, 2, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	before := w.entities[0]
	Step(w, []Command{spCast(1, 2, 1)})
	after := w.entities[0]
	if after.Mana != before.Mana || after.Facing != before.Facing || after.CastWait != before.CastWait || len(w.bookCasts) != 0 {
		t.Fatalf("a cast at an invisible actor changed caster/action state: before=%+v after=%+v pending=%d",
			before, after, len(w.bookCasts))
	}
	if !w.removeAttachedSpell(2, 15) {
		t.Fatal("invisibility did not cancel")
	}
	if !w.beginBookSpell(0, 2, 1) {
		t.Fatal("the same visible actor remained untargetable after invisibility ended")
	}
}

func TestSeeInvisibleUsesTheObserversOwnChebyshevRadiusAtTheBoundary(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<1)
	caster.ScanRange, caster.SeeInvisible = 8, 1
	target := spEnt(2, 4, 2)
	rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	w := hlWorld(t, 101, Relations{}, []SpellRule{rule}, caster, target)
	if !w.attachEffect(2, 2, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	if w.beginBookSpell(0, 2, 1) {
		t.Fatal("detector radius one admitted an invisible target two cells away")
	}
	w.entities[0].SeeInvisible = 2
	if !w.beginBookSpell(0, 2, 1) {
		t.Fatal("detector did not admit an invisible target exactly on its boundary")
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := back.entities[0].SeeInvisible; got != 2 || back.Hash() != w.Hash() {
		t.Fatalf("detector round trip = %d, hashes %#x/%#x", got, back.Hash(), w.Hash())
	}
}

func TestAGroupKeepsAnInvisibleCandidateDetectedByAnyMember(t *testing.T) {
	members := []Entity{
		{ID: 1, X: 2, Y: 2, Owner: 1, Group: 7, HP: 10, MaxHP: 10, ScanRange: 8},
		{ID: 2, X: 5, Y: 2, Owner: 1, Group: 7, HP: 10, MaxHP: 10, ScanRange: 8},
	}
	target := Entity{ID: 3, X: 6, Y: 2, Owner: 2, HP: 10, MaxHP: 10}
	w := engWorld(t, engRel(t, [3]uint32{1, 2, 1}), append(members, target)...)
	if !w.attachEffect(3, 3, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	g := aiGroup{owner: 1, group: 7, members: []int{0, 1}}
	if got := w.candidates(g); len(got) != 0 {
		t.Fatalf("group without a detector kept candidates %v", got)
	}
	w.entities[1].SeeInvisible = 1
	if got := w.candidates(g); len(got) != 1 || got[0] != 2 {
		t.Fatalf("second member's detector produced candidates %v, want [2]", got)
	}
}

func TestProtectionUsesTheDecodedRoundedResistanceAndDrainBypassesIt(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<1|1<<11)
	caster.HP, caster.MaxHP = 50, 100
	target := spEnt(2, 3, 2)
	target.Protection[0] = 25
	// The parallel weapon-kind family must not enter spell arithmetic. All
	// five bytes are maximal so any accidental index visibly changes this
	// Fire-school result.
	target.Resistance = [5]uint8{255, 255, 255, 255, 255}
	w := hlWorld(t, 2, Relations{}, nil, caster, target)
	rule := SpellRule{ID: 1, School: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	w.applySpellDamage(1, rule, 0)
	if got := w.entities[1].HP; got != 92 { // ftol(10*0.75+0.75) = 8
		t.Fatalf("protected damage left %d health, want 92", got)
	}
	w.entities[1].HP = 100
	drain := rule
	drain.ID, drain.Damaging = 11, false
	if !w.ordinaryEffect(0, 1, drain, 0) {
		t.Fatal("drain did not apply")
	}
	if w.entities[1].HP != 90 || w.entities[0].HP != 60 {
		t.Fatalf("Drain should bypass the direct-damage resolver: caster=%d target=%d, want 60/90", w.entities[0].HP, w.entities[1].HP)
	}
}

// TestACloudPulsesOnItsOwnCounterAndNotOnElapsedTicks is `MAGIC-AREAPULSE-037`'s
// phase. `effect+0x4c` starts at `V0 = (AreaDuration << 4) + (power << 4)/10`,
// decrements once per tick, and pulses when the NEW value is a positive multiple
// of 16.
//
// THE FIXTURE PICKS A POWER WHERE THE TWO READINGS DISAGREE. They agree only
// when `V0` is itself a multiple of 16, which is why a power-0 fixture cannot
// see this: Mind 35 at skill level 0 gives power 5, so `V0 = 16 + 8 = 24`, and
// the one pulse falls on tick 8 where counting elapsed ticks puts it on tick 16.
// The pulse count is `floor((V0 - 1)/16) = 1` over `V0 + 1 = 25` ticks.
//
// It also witnesses which occupant slot a cloud reads: only `+0x4`
// (`MAGIC-AREACELL-039`), so the flyer over a painted cell takes nothing, and
// the pulse does not ask whose side the occupant is on.
func TestACloudPulsesOnItsOwnCounterAndNotOnElapsedTicks(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<7)
	caster.Mind = 35
	friend := spEnt(2, 5, 5)
	friend.Owner = SelfSlot
	enemy := spEnt(3, 6, 5)
	enemy.Owner = 2
	air := spEnt(4, 5, 6)
	air.Domain, air.Owner = DomainAir, 2
	rule := SpellRule{ID: 7, Area: true, Distribution: 3, Radius: 1, AreaDuration: 1,
		School: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true, MaxRange: 8}
	w := hlWorld(t, 3, acEnemies(t), []SpellRule{rule}, caster, friend, enemy, air)
	// Keep recipients in the cloud after the first pulse alerts them.
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}
	events := spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 5, Y: 5, Spell: 7})
	if len(events) != 1 || events[0].FromX != 2 || events[0].FromY != 2 ||
		events[0].ToX != 5 || events[0].ToY != 5 || events[0].Target != 0 {
		t.Fatalf("point-target area observation = %+v, want one caster-to-cell event", events)
	}
	for i := 0; i < 7; i++ {
		Step(w, nil)
	}
	if w.entities[1].HP != 100 || w.entities[2].HP != 100 {
		t.Fatalf("the cloud pulsed before counter tick 8: friend=%d enemy=%d", w.entities[1].HP, w.entities[2].HP)
	}
	Step(w, nil)
	friendHit, enemyHit := w.entities[1].HP, w.entities[2].HP
	if friendHit >= 100 || enemyHit >= 100 || w.entities[3].HP != 100 {
		t.Fatalf("counter tick 8 = friend %d enemy %d air %d; want both slot +0x4 occupants hit and the flyer untouched",
			friendHit, enemyHit, w.entities[3].HP)
	}
	for i := 0; i < 17; i++ {
		Step(w, nil)
	}
	if w.entities[1].HP != friendHit-(100-friendHit) || w.entities[2].HP != enemyHit-(100-enemyHit) {
		t.Fatalf("final-zero pulse differs: friend %d->%d enemy %d->%d; V0=24 gives two",
			friendHit, w.entities[1].HP, enemyHit, w.entities[2].HP)
	}
	if got := w.CellEffects(); len(got) != 0 {
		t.Fatalf("the cloud outlived its V0 + 1 = 25 ticks: %+v", got)
	}
}

// The per-unit cell walk pays the DAMAGE feed and no cast award (DIV-238). It
// used to pay one cast award per application, which is what left the six cloud
// and wall rows training nothing at all: a cloud reaches this walk only on a
// later pulse, and the pulse call passed the award over. The cast award is
// castSpell's and castBookAt's now, one per landed book cast.
//
// TO CONFIRM IT WITNESSES THE FIX, restore an awardSkill call inside
// applyAreaCells's occupant loop (pkg/sim/celleffect.go) and rerun: the total
// rises by two half-mana awards.
func TestTheAreaCellWalkPaysTheDamageFeedAndNoCastAward(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<2)
	caster.GainsXP = true
	a, b := spEnt(2, 4, 4), spEnt(3, 5, 4)
	a.Owner, b.Owner = 2, 3
	rule := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, ManaCost: 10,
		DamageMin: 1, DamageMax: 1, Damaging: true}
	w := hlWorld(t, 4, Relations{}, []SpellRule{rule}, caster, a, b)
	e := cellEffect{Key: cellKey(4, 4), Spell: 2, Caster: 1, HasCaster: true,
		Mode: areaModeBlast}
	w.applyAreaCells(e, rule, blastCells(4, 4, 1))
	if got := w.entities[0].SkillXP[1]; got != 2 {
		t.Fatalf("two area applications trained %d XP, want two damage events and no cast award", got)
	}
}

func TestABookCastTrainsItsSchoolOnceEvenWhenItReachesNoUnit(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule SpellRule
		want int32
	}{
		{"cloud, Light's own columns", SpellRule{ID: 12, Area: true, Distribution: 3,
			Radius: 1, AreaDuration: 20, School: 3, ManaCost: 5, MaxRange: 6}, 3},
		{"wall", SpellRule{ID: 19, Area: true, Distribution: 4,
			Radius: 1, AreaDuration: 15, School: 4, ManaCost: 20, MaxRange: 6}, 12},
		{"blast", SpellRule{ID: 2, Area: true, Distribution: 3,
			Radius: 1, School: 1, ManaCost: 10, MaxRange: 10}, 6},
		{"staged", SpellRule{ID: 21, Area: true, Distribution: 5,
			Radius: 2, School: 4, ManaCost: 20, MaxRange: 10}, 12},
		{"teleport, the one point row cast at a cell", SpellRule{ID: 26,
			School: 5, ManaCost: 20, MaxRange: 8}, 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 4, 4, 1<<uint(tc.rule.ID))
			caster.GainsXP = true
			w := hlWorld(t, 0x100d, Relations{}, []SpellRule{tc.rule}, caster)
			spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 6, Y: 4,
				Spell: tc.rule.ID})
			if got := w.entities[0].SkillXP[tc.rule.School]; got != tc.want {
				t.Fatalf("a cast reaching no unit trained %d XP in school %d, want %d",
					got, tc.rule.School, tc.want)
			}
		})
	}
}

// The unit form of the same rule (DIV-238). castSpell admits an area row aimed
// at a unit and lands it at that unit's cell, and it refused every area row its
// cast award. Light applies to nobody at its landing, a cloud reaching units on
// its later pulses only, so the cast's own award is again the only award in the
// world. Its `Spell Target` column is 2 and TargetsUnit is therefore false here,
// which is the shipped row: the admission gate takes an area row whatever that
// column says.
//
// TO CONFIRM IT WITNESSES THE FIX, restore the `!rule.Area` guard on castSpell's
// award (pkg/sim/spell.go) and rerun.
func TestAnAreaRowAimedAtAUnitTrainsTheCastersSchool(t *testing.T) {
	caster := effectMage(1, 4, 4, 1<<12)
	caster.GainsXP = true
	ally := spEnt(2, 6, 4)
	ally.Owner = 2
	rule := SpellRule{ID: 12, Area: true, Distribution: 3, Radius: 1,
		AreaDuration: 20, School: 3, ManaCost: 5, MaxRange: 6}
	w := hlWorld(t, 0x100e, Relations{}, []SpellRule{rule}, caster, ally)
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 12})
	if got := w.entities[0].SkillXP[3]; got != 3 {
		t.Fatalf("an area row aimed at a unit trained %d XP in Air, want 3", got)
	}
}

func TestTeleportTargetsAnEmptyCellAtItsDecodedPowerScaledRange(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<26)
	caster.Mind = 60 // power 30: Teleport adds 10 to its base range 1.
	rule := SpellRule{ID: 26, ManaCost: 20, School: 5, MaxRange: 1}
	w := hlWorld(t, 5, Relations{}, []SpellRule{rule}, caster)
	events := spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 11, Y: 1, Spell: 26})
	got := w.entities[0]
	if got.X != 11 || got.Y != 1 || got.Mana != 80 {
		t.Fatalf("teleport result = cell (%d,%d), mana %d; want (11,1), 80", got.X, got.Y, got.Mana)
	}
	if len(events) != 1 || events[0].FromX != 1 || events[0].FromY != 1 ||
		events[0].ToX != 11 || events[0].ToY != 1 {
		t.Fatalf("teleport observation = %+v, want one source-to-destination event", events)
	}
}

func TestFireballNormalisesAfterPowerAndByCoveredCells(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<2)
	target := spEnt(2, 4, 4)
	target.TokenSize = 2
	rule := SpellRule{ID: 2, Area: true, Radius: 2, School: 1,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	w := hlWorld(t, 7, Relations{}, []SpellRule{rule}, caster, target)
	e := cellEffect{Key: cellKey(4, 4), Spell: 2, Caster: 1, HasCaster: true,
		Power: 30, Mode: areaModeBlast}
	w.applyAreaCells(e, rule, blastCells(4, 4, 2))
	if got := w.entities[1].HP; got != 80 {
		t.Fatalf("2x2 target took %d damage, want 20: scaled 20 / 4 cells, four applications", 100-got)
	}
}

func TestAreaLayerConflictsAreDirectional(t *testing.T) {
	w := hlWorld(t, 8, Relations{}, []SpellRule{{ID: 3}, {ID: 7}, {ID: 8}, {ID: 2}}, spEnt(1, 1, 1))
	cloud := func(id uint16) SpellRule {
		return SpellRule{ID: id, Area: true, Distribution: 3, Radius: 1, AreaDuration: 2}
	}
	if !w.landArea(cloud(3), 0, 0, false, 1, 2, 3, 2, nil) {
		t.Fatal("fire layer did not land")
	}
	if !w.landArea(cloud(8), 0, 0, false, 1, 2, 3, 2, nil) {
		t.Fatal("poison layer did not register")
	}
	if len(w.effects) != 2 || len(w.effects[0].Cells) == 0 || len(w.effects[1].Cells) != 0 {
		t.Fatalf("poison cast into fire changed the wrong layer: %+v", w.effects)
	}
	if !w.landArea(cloud(7), 0, 0, false, 1, 2, 3, 2, nil) {
		t.Fatal("freezing layer did not land")
	}
	for _, e := range w.effects {
		if e.Spell == 3 && len(e.Cells) != 0 {
			t.Fatalf("freezing left fire cells standing: %+v", w.effects)
		}
	}
	// A blast is not retained as a layer, but it still clears poison and
	// freezing in every square cell it visits.
	w.landArea(SpellRule{ID: 2, Area: true, Radius: 2}, 0, 0, false, 1, 2, 3, 2, nil)
	for _, e := range w.effects {
		if (e.Spell == 7 || e.Spell == 8) && len(e.Cells) != 0 {
			t.Fatalf("fireball left conflicting cells standing: %+v", w.effects)
		}
	}
}

func TestWallOfEarthBlocksGroundAndGhostRoutesAndRemovalRestoresThem(t *testing.T) {
	for _, domain := range []Domain{DomainGround, DomainGhost, DomainAir} {
		mover := spEnt(1, 0, 2)
		mover.Domain = domain
		w, err := NewStockedSpelledWorld(uint64(10+domain), Bounds{Width: 5, Height: 5}, ModeCanonical,
			Terrain{}, []Entity{mover}, nil, Relations{}, nil, nil, []SpellRule{{ID: 19}})
		if err != nil {
			t.Fatalf("NewStockedSpelledWorld: %v", err)
		}
		wall := SpellRule{ID: 19, Area: true, Distribution: 4, AreaDuration: 1, Radius: 1}
		if !w.landArea(wall, 0, 0, false, 1, 2, 2, 2, nil) {
			t.Fatal("wall did not land")
		}
		s := newRouteScratch(w)
		_, open := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, 4, 2)
		if want := domain == DomainAir; open != want {
			t.Errorf("domain %d route through wall open=%v, want %v", domain, open, want)
		}
		for len(w.effects) != 0 {
			Step(w, nil)
		}
		s = newRouteScratch(w)
		if _, open := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, 4, 2); !open {
			t.Errorf("domain %d route stayed blocked after wall removal", domain)
		}
	}
}

func TestTheControlledRealMissionWitnessUsesTheOrdinaryEffectOwners(t *testing.T) {
	base, err := NewStockedSpelledWorld(1001, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{
			{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15, School: 1,
				MaxRange: 6, ManaCost: 20, DamageMin: 6, DamageMax: 6, Damaging: true},
			{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15, School: 4,
				MaxRange: 6, ManaCost: 20},
			{ID: 24, TargetsUnit: true, Defensive: true, School: 5, MaxRange: 5, ManaCost: 10,
				SpellDuration: 30, EffectKind: EffectSpeed, EffectMode: EffectDuration, EffectMagnitude: 2, EffectDuration: 30},
		})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ControlledSpellEffectWitness(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.PointBefore != 10 || got.PointAfter <= got.PointBefore || got.PointEffects != 1 {
		t.Errorf("point witness = speed %d->%d, effects %d", got.PointBefore, got.PointAfter, got.PointEffects)
	}
	if got.AreaFriendAt15 != got.AreaFriendBefore || got.AreaEnemyAt15 != got.AreaEnemyBefore ||
		got.AreaFriendAt16 >= got.AreaFriendBefore || got.AreaEnemyAt16 >= got.AreaEnemyBefore {
		t.Errorf("area witness = friend %d/%d/%d enemy %d/%d/%d",
			got.AreaFriendBefore, got.AreaFriendAt15, got.AreaFriendAt16,
			got.AreaEnemyBefore, got.AreaEnemyAt15, got.AreaEnemyAt16)
	}
	if got.WallBefore != [3]bool{true, true, true} || got.WallDuring != [3]bool{false, false, true} ||
		got.WallAfter != [3]bool{true, true, true} {
		t.Errorf("wall witness = before %v during %v after %v", got.WallBefore, got.WallDuring, got.WallAfter)
	}
}

func TestAttachedAndAreaStateRoundTripByteIdentically(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<24)
	target := spEnt(2, 2, 1)
	w := hlWorld(t, 20, Relations{}, []SpellRule{{ID: 24}, {ID: 19}}, caster, target)
	if !w.attachEffect(2, 1, SpellRule{ID: 24}, EffectSpeed, 3, 29, EffectDuration) {
		t.Fatal("effect did not attach")
	}
	if !w.landArea(SpellRule{ID: 19, Area: true, Distribution: 4, AreaDuration: 2}, 0, 1, true, 1, 1, 4, 4, nil) {
		t.Fatal("wall did not land")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("second MarshalBinary: %v", err)
	}
	if !bytes.Equal(form, again) || w.Hash() != back.Hash() {
		t.Fatal("attached/area state did not round-trip byte-identically")
	}
	for _, k := range back.effects[0].Cells {
		x, y := keyCell(k)
		if back.terrainOpen(DomainGround, x, y) {
			t.Fatalf("decoded wall cell (%d,%d) is open", x, y)
		}
	}
}

func TestBookCastProgressRoundTripsAndReleasesOnce(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<1)
	caster.AttackCharge, caster.AttackRelax = 12, 4
	target := spEnt(2, 5, 1)
	rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 10, DamageMax: 10,
		Damaging: true, TargetsUnit: true}
	w := hlWorld(t, 21, Relations{}, []SpellRule{rule}, caster, target)
	Step(w, []Command{spCast(1, 2, 1)})
	for range 5 {
		Step(w, nil)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(form, mustWorldBytes(t, &back)) || back.Hash() != w.Hash() {
		t.Fatal("mid-wind-up cast did not round-trip canonically")
	}
	var events int
	for range 20 {
		events += len(StepObserved(&back, nil))
	}
	if events != 1 || back.entities[1].HP != 90 || back.entities[0].Mana != 97 {
		t.Fatalf("resumed cast produced events=%d hp=%d mana=%d, want one/90/97", events, back.entities[1].HP, back.entities[0].Mana)
	}
}

func mustWorldBytes(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHealThatBecomesUnneededDuringWindupStillReleasesAndPays(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<6)
	target := spEnt(2, 2, 1)
	target.HP = 50
	w := hlWorld(t, 22, Relations{}, []SpellRule{hlHeal()}, caster, target)
	Step(w, []Command{spCast(1, 2, 6)})
	w.entities[1].HP = w.entities[1].MaxHP
	var events int
	for range castPeriod {
		events += len(StepObserved(w, nil))
	}
	if events != 1 || w.entities[1].HP != w.entities[1].MaxHP || w.entities[0].Mana >= 100 {
		t.Fatalf("pending heal at a full target emitted %d events, left health %d/%d mana %d: want one release, health at the maximum, mana paid",
			events, w.entities[1].HP, w.entities[1].MaxHP, w.entities[0].Mana)
	}
}

func TestControlSpiritConsumesBonesAndCreatesANewOwnedGhost(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 1)
	corpse.Reaction, corpse.Mind, corpse.Spirit = 41, 37, 29
	corpse.ToHit, corpse.Defence = 71, 73
	corpse.MaxHP = 86
	tmpl := hlGhostTemplate()
	tmpl.Humanoid = true
	w := hlGhostWorld(t, 30, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, tmpl, caster, corpse)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})
	if len(w.entities) != 2 || w.entities[1].ID == 2 || w.entities[1].Domain != DomainGhost || w.entities[1].Owner != 7 {
		t.Fatalf("control spirit result = %+v, want consumed corpse and a new owner-7 ghost", w.entities)
	}
	if got := w.entities[1]; got.Reaction != 21 || got.Mind != 37 || got.Spirit != 29 {
		t.Fatalf("ghost stats = reaction %d, mind %d, spirit %d; want 21, 37, 29", got.Reaction, got.Mind, got.Spirit)
	}
	// THE RAISED ACTOR IS THE TEMPLATE'S KIND. Class is what pkg/game resolves
	// art and a displayed name through; a raise that left it at zero drew
	// nothing and named nothing. Every field here is the template's own value
	// and none is the corpse's, so this fails on a build that copies the
	// corpse instead.
	if got := w.entities[1]; got.Class != tmpl.Class || got.TypeID != tmpl.TypeID ||
		got.Speed != tmpl.Speed || got.ScanRange != tmpl.ScanRange || got.Reach != tmpl.Reach ||
		got.DyingTime != tmpl.DyingTime || got.XPValue != tmpl.XPValue ||
		got.Withdraw != tmpl.Withdraw || got.Wimpy != tmpl.Wimpy ||
		got.Protection != tmpl.Protection || got.Resistance != tmpl.Resistance || got.XPSlot != tmpl.XPSlot ||
		got.Absorption != tmpl.Absorption || got.DamageBase != tmpl.DamageBase ||
		got.DamageSpread != tmpl.DamageSpread || got.AttackCharge != tmpl.AttackCharge ||
		got.AttackRelax != tmpl.AttackRelax || got.Humanoid != tmpl.Humanoid || !got.AlwaysHits {
		t.Fatalf("raised actor = %+v, want the ghost template %+v", w.entities[1], tmpl)
	}
	// The seven corpse stores: the first words of the to-hit and defence
	// blocks come off the corpse, and health and its maximum are half the
	// corpse's maximum (MAGIC-249).
	if got := w.entities[1]; got.ToHit != 71 || got.Defence != 73 || got.MaxHP != 43 || got.HP != 43 {
		t.Fatalf("ghost to-hit %d defence %d health %d/%d; want 71, 73, 43/43", got.ToHit, got.Defence, got.HP, got.MaxHP)
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatalf("resulting world is not persistent: %v", err)
	}
}

func TestControlSpiritPreservesEveryDefinedTemplateDomainThroughTheForm(t *testing.T) {
	for _, tc := range []struct {
		name   string
		domain Domain
	}{
		{"ground", DomainGround},
		{"ghost", DomainGhost},
		{"air", DomainAir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.domain
			caster := effectMage(1, 1, 1, 1<<25)
			caster.Owner = 7
			corpse := spEnt(2, 2, 1)
			tmpl := hlGhostTemplate()
			tmpl.Domain = d
			w := hlGhostWorld(t, 30,
				[]SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
				tmpl, caster, corpse)
			w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
			spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})
			if len(w.entities) != 2 || w.entities[1].Domain != d {
				t.Fatalf("raised entities = %+v, want domain %d", w.entities, uint8(d))
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var back World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
			again, err := back.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary after load: %v", err)
			}
			if !bytes.Equal(again, form) || back.Hash() != w.Hash() || back.entities[1].Domain != d {
				t.Fatalf("domain %d round trip: bytes=%v hashes=%#x/%#x domain=%d",
					uint8(d), bytes.Equal(again, form), back.Hash(), w.Hash(), uint8(back.entities[1].Domain))
			}
		})
	}
}

// TestControlSpiritWithoutATemplateIsRefusedBeforeItsCost is the other half of
// the raise: a world whose definition table shipped no `Ghost` row raises
// nothing, and it refuses at ADMISSION rather than paying and then failing.
func TestControlSpiritWithoutATemplateIsRefusedBeforeItsCost(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 1)
	w := hlWorld(t, 30, Relations{}, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, caster, corpse)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
	mana := w.entities[0].Mana
	if r := w.BookSpellRefusal(1, 2, 25); r == "" {
		t.Fatalf("BookSpellRefusal admitted a raise with no template loaded")
	}
	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})
	if len(w.entities) != 2 || w.entities[0].Mana != mana {
		t.Fatalf("templateless raise: %d entities and mana %d, want 2 and %d",
			len(w.entities), w.entities[0].Mana, mana)
	}
}

func TestMapOwnedMageChoosesAnAffordableSpellThroughOrdinaryAIApply(t *testing.T) {
	mage := effectMage(1, 2, 2, 1<<1)
	mage.Owner, mage.Mind = 2, 30
	target := spEnt(2, 3, 2)
	target.Owner = SelfSlot
	rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 5, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	w := hlWorld(t, 31, acEnemies(t), []SpellRule{rule}, mage, target)
	w.engagementPassObserved(nil)
	spRunUnbidden(w)
	if w.entities[1].HP != 90 || w.entities[0].Mana != 97 {
		t.Fatalf("AI spell result = target %d, mana %d; want 90 and 97", w.entities[1].HP, w.entities[0].Mana)
	}
}

func TestStagedAreaProgramsUseTheirDecodedOrderedCells(t *testing.T) {
	w := mustWorld(t, 0x175, Bounds{Width: 40, Height: 40}, nil)
	point := func(dx, dy int32) uint16 { return cellKey(20+dx, 20+dy) }

	fire := cellEffect{Key: cellKey(20, 20), Spell: 4}
	for stage, want := range [][]uint16{
		{point(-1, 1), point(-1, 0), point(-1, -1), point(0, 1), point(0, -1), point(1, 1), point(1, 0), point(1, -1)},
		{point(-2, 1), point(-2, 0), point(-2, -1), point(-1, 2), point(0, 2), point(1, 2), point(-1, -2), point(0, -2), point(1, -2), point(2, 1), point(2, 0), point(2, -1)},
	} {
		fire.Direction = 7 // Fire Sacrifice ignores orientation.
		if got := w.ringStageCells(fire, stage); !reflect.DeepEqual(got, want) {
			t.Fatalf("Fire Sacrifice stage %d = %v, want %v", stage, got, want)
		}
	}

	transform := func(o int, dx, dy int32) (int32, int32) {
		switch o {
		case 0, 1:
			return dx, -dy
		case 2:
			return dy, dx
		case 3:
			return dx, dy
		case 4, 5:
			return -dx, dy
		case 6:
			return -dy, dx
		default:
			return -dx, -dy
		}
	}
	acid := cellEffect{Key: cellKey(20, 20), Spell: 9}
	for o := 0; o < 8; o++ {
		acid.Direction = uint8(o)
		for stage := 0; stage < 6; stage++ {
			want := make([]uint16, 0)
			if o&1 == 0 {
				if stage < 5 {
					for x := -stage; x <= stage; x++ {
						dx, dy := transform(o, int32(x), int32(stage))
						want = append(want, point(dx, dy))
					}
				}
			} else {
				for i := 0; i <= stage; i++ {
					dx, dy := transform(o, int32(stage-i), int32(i))
					want = append(want, point(dx, dy))
				}
			}
			if got := w.ringStageCells(acid, stage); !reflect.DeepEqual(got, want) {
				t.Fatalf("Acid Stream orientation %d stage %d = %v, want %v", o, stage, got, want)
			}
		}
	}

	meteor := cellEffect{Key: cellKey(20, 20), Spell: 21}
	oracle := w.rng
	for stage := 0; stage < 32; stage++ {
		want := point(oracle.uniform(5)-2, oracle.uniform(5)-2)
		if got := w.ringStageCells(meteor, stage); len(got) != 1 || got[0] != want {
			t.Fatalf("Meteor stage %d = %v, want [%d] from x-then-y draws", stage, got, want)
		}
	}
}

// TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots is
// `TERR-CELLREC-146`'s keying. The cell record holds `+0x4` an actor of
// movement domain 1 or 2, `+0x8` an actor of domain 3, `+0xc` a structure —
// this build's Domain.layer() is the same ground/air split. So a cell reaches
// its ground-layer actors and its flyer, not the first three entities by id.
//
// THE DECODED RECORD CAPS EACH ACTOR SLOT AT ONE, because ROM1's own
// movement refuses to register an actor's footprint over a same-layer cell
// another actor holds (`TERR-FOOTPRINT-147`). Round 2 read that cap here
// before this build enforced the footprint invariant, which made entity 5
// below immune to every area effect. Constructors and canonical decode still
// accept an already-overlapping state, so the round-3 defensive read of
// every covering actor remains until DIV-055 defines that import policy.
//
// The fixture is the representable failure: three decayed bodies on one cell filled
// a three-slot cap keyed by nothing, and the living actor beneath them took
// nothing at all from a Fire Ball. It also pins the flyer to `+0x8`, which a
// blast reads and a cloud does not.
func TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots(t *testing.T) {
	body := func(id EntityID) Entity {
		return Entity{ID: id, X: 20, Y: 20, HP: -10, MaxHP: 100, TokenSize: 1, Decay: DecayBones}
	}
	ents := []Entity{
		body(1), body(2), body(3),
		{ID: 4, X: 20, Y: 20, HP: 100, MaxHP: 100, TokenSize: 1},
		{ID: 5, X: 20, Y: 20, HP: 100, MaxHP: 100, TokenSize: 1},
		{ID: 6, X: 20, Y: 20, HP: 100, MaxHP: 100, TokenSize: 1, Domain: DomainAir},
	}
	rule := SpellRule{ID: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	for _, mode := range []struct {
		name    string
		mode    uint8
		wantAir bool
	}{
		{"blast reads +0x4, +0x8 and +0xc", areaModeBlast, true},
		{"a cloud pulse reads +0x4 alone", areaModeCloud, false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			w := mustWorld(t, 8, Bounds{Width: 40, Height: 40}, ents)
			w.applyAreaCells(cellEffect{Spell: 1, Power: 30, Mode: mode.mode}, rule, []uint16{cellKey(20, 20)})
			for i, e := range w.entities {
				switch {
				case i < 3:
					// A terminal body past its dwell is in no slot at all, so it
					// neither takes the effect nor stands between the effect and 4.
					if e.HP != -10 {
						t.Errorf("decayed body %d health = %d, want -10", e.ID, e.HP)
					}
				case e.ID == 4:
					if e.HP >= 100 {
						t.Errorf("the cell's first ground-layer occupant %d was not hit", e.ID)
					}
				case e.ID == 5:
					// R3-A2: a second ground-layer actor on the same cell is no
					// longer immune. TO CONFIRM THE REVERT REDDENS IT, restore
					// cellSlotOccupants' single-int return and applyAreaCells'
					// `slots[:read]` cap and rerun: this actor reads 100 (0
					// damage) again.
					if e.HP >= 100 {
						t.Errorf("a second ground-layer actor %d on entity 4's cell was not hit", e.ID)
					}
				case e.ID == 6:
					if hit := e.HP < 100; hit != mode.wantAir {
						t.Errorf("flyer %d hit = %v, want %v", e.ID, hit, mode.wantAir)
					}
				}
			}
		})
	}
}

// TestWallGeometryIsTheShippedTablesAndNotADrawnLine is
// `MAGIC-AREACELL-039`'s distribution-4 paint.
//
// Table B is what the diagonal arms take: a doubled anti-diagonal spanning
// +/-2, 9 cells. A line drawn from the bearing spans +/-4 and is 1 thick, which
// is what all four diagonal arms produced. Eight shipped nodes lay diagonal
// walls.
func TestWallGeometryIsTheShippedTablesAndNotADrawnLine(t *testing.T) {
	tableA := [][2]int32{{-2, 1}, {-1, 1}, {0, 1}, {1, 1}, {2, 1}, {-2, 0}, {-1, 0}, {0, 0}, {1, 0}, {2, 0}}
	tableB := [][2]int32{{-2, 2}, {-1, 1}, {0, 0}, {1, -1}, {2, -2}, {-1, 2}, {0, 1}, {1, 0}, {2, -1}}
	arms := []struct {
		table        [][2]int32
		swap         bool
		signX, signY int32
	}{
		{tableA, false, 1, -1}, {tableB, false, 1, -1},
		{tableA, true, 1, 1}, {tableB, false, 1, 1},
		{tableA, false, -1, 1}, {tableB, false, -1, 1},
		{tableA, true, -1, 1}, {tableB, false, -1, -1},
	}
	w := mustWorld(t, 0x39, Bounds{Width: 40, Height: 40}, nil)
	for d, arm := range arms {
		want := make([]uint16, 0, len(arm.table))
		for _, o := range arm.table {
			dx, dy := o[0], o[1]
			if arm.swap {
				dx, dy = dy, dx
			}
			want = append(want, cellKey(20+arm.signX*dx, 20+arm.signY*dy))
		}
		sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })
		if got := w.wallCells(20, 20, uint8(d)); !reflect.DeepEqual(got, want) {
			t.Errorf("direction %d wall = %v, want %v", d, got, want)
		}
		if n := len(want); n != len(arm.table) {
			t.Errorf("direction %d produced %d distinct cells from a table of %d", d, n, len(arm.table))
		}
	}
}

// TestBothShippedDistributionFourRowsPaintAWall is A4. `MAGIC-AREACELL-039`
// puts `Distribution system == 4` — `wall_of_fire` AND `wall_of_earth` —
// through the wall tables, and the installed table carries exactly those two
// rows at that column. Forking on spell id 19 gave Wall of Fire the
// distribution-3 diamond: 25 cells of a Manhattan radius 3 where the row asks
// for 10.
func TestBothShippedDistributionFourRowsPaintAWall(t *testing.T) {
	victim := Entity{ID: 2, X: 20, Y: 20, HP: 100, MaxHP: 100, TokenSize: 1}
	for _, id := range []uint16{3, 19} {
		w := mustWorld(t, 0xa4, Bounds{Width: 40, Height: 40}, []Entity{victim})
		w.spells = []SpellRule{{ID: id, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}}
		if !w.landArea(w.spells[0], 0, 0, false, 16, 20, 20, 20, nil) {
			t.Fatalf("spell %d did not land", id)
		}
		got := w.effects[0].Cells
		// Cast from due west, so the arm is direction 2: table A with its dx
		// and dy arrays swapped, a 2x5 block in the columns 20 and 21.
		want := w.wallCells(20, 20, 2)
		if id == 19 {
			want = w.skipOccupiedGround(append([]uint16(nil), want...))
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("spell %d painted %v, want %v", id, got, want)
		}
		if id == 3 && len(got) != 10 {
			t.Errorf("Wall of Fire painted %d cells, want the table's 10", len(got))
		}
		if id == 19 && len(got) != 9 {
			t.Errorf("Wall of Earth painted %d cells, want 10 less the occupied one", len(got))
		}
	}
}

// TestTheRealMissionWitnessSeparatesTheWallFromTheDiamond is A4 on installed
// data rather than on a fixture. The mission witness's two victims both stand
// inside the wall table AND inside the distribution-3 diamond of radius 3, so
// their health cannot tell the two shapes apart. The fourth actor, two cells
// west of the target, is inside the diamond and outside the wall.
func TestTheRealMissionWitnessSeparatesTheWallFromTheDiamond(t *testing.T) {
	base, err := NewStockedSpelledWorld(0xa4a4, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{
			{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15, School: 1,
				MaxRange: 6, ManaCost: 20, DamageMin: 6, DamageMax: 6, Damaging: true},
			{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15, School: 4,
				MaxRange: 6, ManaCost: 20},
			{ID: 24, TargetsUnit: true, Defensive: true, School: 5, MaxRange: 5, ManaCost: 10,
				SpellDuration: 30, EffectKind: EffectSpeed, EffectMode: EffectDuration,
				EffectMagnitude: 2, EffectDuration: 30},
		})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ControlledSpellEffectWitness(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.AreaEnemyAt16 >= got.AreaEnemyBefore {
		t.Fatalf("the wall did not reach its own victim: enemy %d -> %d", got.AreaEnemyBefore, got.AreaEnemyAt16)
	}
	if got.AreaOutsideAfter != got.AreaOutsideBefore {
		t.Errorf("the actor two cells west took %d damage; the wall table does not reach it, "+
			"and only the distribution-3 diamond would", got.AreaOutsideBefore-got.AreaOutsideAfter)
	}
}

// TestTheCloudDiamondStopsAtTheLoopBoundNotAtTheFilter is D2. The filter
// reaches one further than the walk, so the four axis tips at distance r + 1
// are unreachable and the painted set is that diamond intersected with the
// (2r+1) square.
func TestTheCloudDiamondStopsAtTheLoopBoundNotAtTheFilter(t *testing.T) {
	w := mustWorld(t, 0xd2, Bounds{Width: 40, Height: 40}, nil)
	for _, tc := range []struct{ r, cells int32 }{{1, 9}, {2, 21}, {4, 57}} {
		got := w.diamondCells(20, 20, tc.r)
		if int32(len(got)) != tc.cells {
			t.Errorf("radius %d painted %d cells, want %d", tc.r, len(got), tc.cells)
		}
		for _, tip := range [][2]int32{{tc.r + 1, 0}, {-tc.r - 1, 0}, {0, tc.r + 1}, {0, -tc.r - 1}} {
			if containsKey(got, cellKey(20+tip[0], 20+tip[1])) {
				t.Errorf("radius %d painted the axis tip (%d,%d), which the loop bound cannot reach",
					tc.r, tip[0], tip[1])
			}
		}
		for dy := -tc.r; dy <= tc.r; dy++ {
			for dx := -tc.r; dx <= tc.r; dx++ {
				want := cellEffectAbs(dx)+cellEffectAbs(dy) <= tc.r+1
				if containsKey(got, cellKey(20+dx, 20+dy)) != want {
					t.Errorf("radius %d cell (%d,%d) painted = %v, want %v", tc.r, dx, dy, !want, want)
				}
			}
		}
	}
}

// TestWallOfEarthRefusesEveryCellOfAnOccupantsFootprint is D5.
// `MAGIC-WALLEARTH-042` skips a cell whose `+0x4` slot is non-null;
// `TERR-FOOTPRINT-147` puts an actor's pointer in every cell of its n x n
// footprint. Reading the anchor alone painted a wall inside a large unit.
//
// The second half is the slot's own membership: a body lying where it fell
// keeps its cell, so it fills `+0x4` and refuses one, where a liveness test let
// a wall paint over it.
func TestWallOfEarthRefusesEveryCellOfAnOccupantsFootprint(t *testing.T) {
	big := Entity{ID: 2, X: 20, Y: 19, HP: 100, MaxHP: 100, TokenSize: 3}
	corpse := Entity{ID: 3, X: 21, Y: 22, HP: 0, MaxHP: 100, TokenSize: 1,
		Decay: DecayFallen, Dwell: 40}
	flyer := Entity{ID: 4, X: 20, Y: 18, HP: 100, MaxHP: 100, TokenSize: 1, Domain: DomainAir}
	w := mustWorld(t, 0xd5, Bounds{Width: 40, Height: 40}, []Entity{big, corpse, flyer})
	w.spells = []SpellRule{{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}}
	if !w.landArea(w.spells[0], 0, 0, false, 16, 20, 20, 20, nil) {
		t.Fatal("wall did not land")
	}
	// Direction 2: columns 20 and 21, rows 18 to 22.
	for _, c := range [][2]int32{{20, 19}, {21, 19}, {20, 20}, {21, 20}, {20, 21}, {21, 21}} {
		if containsKey(w.effects[0].Cells, cellKey(c[0], c[1])) {
			t.Errorf("the wall painted (%d,%d), inside the 3x3 unit's own footprint", c[0], c[1])
		}
	}
	if containsKey(w.effects[0].Cells, cellKey(21, 22)) {
		t.Error("the wall painted the cell a fallen body is lying on")
	}
	if !containsKey(w.effects[0].Cells, cellKey(20, 18)) {
		t.Error("the wall skipped (20,18); a flyer is in slot +0x8 and refuses no cell")
	}
	if got, want := len(w.effects[0].Cells), 3; got != want {
		t.Errorf("the wall kept %d of 10 cells, want %d", got, want)
	}
}

// TestAStagedEffectSurvivesASaveAtEveryTickOfItsLife is B3. The ring generators
// produce their cells in the decoded visit order, which is significant at
// application and is not ascending; the byte form refuses a cell list that is
// not strictly ascending, so a save taken while Fire Sacrifice or Acid Stream
// stood could be written and not read back.
//
// It walks every tick of both programs, and Acid Stream in all eight
// orientations, because the refusal was direction-dependent.
func TestAStagedEffectSurvivesASaveAtEveryTickOfItsLife(t *testing.T) {
	caster := Entity{ID: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, TokenSize: 1}
	for _, id := range []uint16{4, 9} {
		for dir := int32(0); dir < 8; dir++ {
			w, err := NewStockedSpelledWorld(uint64(0xb3+dir), Bounds{Width: 40, Height: 40}, ModeCanonical,
				Terrain{}, []Entity{caster}, nil, Relations{}, nil, nil,
				[]SpellRule{{ID: id, Area: true, Distribution: 5, Radius: 2}})
			if err != nil {
				t.Fatal(err)
			}
			from := [8][2]int32{{0, 4}, {-4, 4}, {-4, 0}, {-4, -4}, {0, -4}, {4, -4}, {4, 0}, {4, 4}}[dir]
			if !w.landArea(w.spells[0], 0, 1, true, 20+from[0], 20+from[1], 20, 20, nil) {
				t.Fatalf("spell %d direction %d did not land", id, dir)
			}
			for tick := 0; len(w.effects) != 0; tick++ {
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatalf("spell %d direction %d tick %d: MarshalBinary: %v", id, dir, tick, err)
				}
				var back World
				if err := back.UnmarshalBinary(form); err != nil {
					t.Fatalf("spell %d direction %d tick %d: UnmarshalBinary: %v", id, dir, tick, err)
				}
				if w.Hash() != back.Hash() {
					t.Fatalf("spell %d direction %d tick %d did not round-trip", id, dir, tick)
				}
				Step(w, nil)
			}
		}
	}
}

// TestAWallAcrossAStoredRouteKeepsItAndLoads is B4. A wall landing across a
// stored route leaves the route standing, as the original does, and the byte
// form holds it. Twenty-three shipped instant-21 nodes cast Wall of Earth.
func TestAWallAcrossAStoredRouteKeepsItAndLoads(t *testing.T) {
	mover := Entity{ID: 1, X: 4, Y: 20, HP: 100, MaxHP: 100, Speed: 10, TokenSize: 1,
		ScanRange: 8, HasTarget: true, TargetX: 34, TargetY: 20}
	w, err := NewStockedSpelledWorld(0xb4, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, []Entity{mover}, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4 && len(w.routes[0]) == 0; i++ {
		Step(w, nil)
	}
	route := w.routes[0]
	if len(route) == 0 {
		t.Fatal("the mover holds no stored route to invalidate")
	}
	mid := route[len(route)/2]
	if !w.landArea(w.spells[0], 0, 0, false, mid.x-4, mid.y, mid.x, mid.y, nil) {
		t.Fatal("wall did not land")
	}
	if w.routeTerrainOpen(w.entities[0], w.routes[0]) {
		t.Fatal("the wall closed no cell of the stored route")
	}
	if len(w.routes[0]) != len(route) {
		t.Errorf("the mover holds %d route cells after the wall, had %d", len(w.routes[0]), len(route))
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("a world holding a wall over a stored route did not load: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Error("the loaded world's hash differs from the saved world's")
	}
}

// TestTheAreaModeComesFromTheRowsOwnColumns is E1. `MAGIC-AREATICK-036` gives
// the selector as `Distribution system == 5` -> staged, else
// `Area Effect Duaration > 0` -> cloud, else blast, and the distribution arm
// runs second so it also zeroes the duration counter. The shipped rows agree
// with the id switch this replaces; an edited row did not, which left
// `SpellRule.Distribution` parsed, serialized and read by nothing.
//
// The staged life is the same claim's cadence — stage ticks 0, 3, 6, ... with
// the terminal stage removing the record — so `areaLife` no longer carries an
// invented `(Radius + 2) * 2` behind a hard-coded id 21.
func TestTheAreaModeComesFromTheRowsOwnColumns(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule SpellRule
		mode uint8
		life uint16
	}{
		{"Fire Ball's own columns", SpellRule{ID: 2, Area: true, Distribution: 3, Radius: 1}, areaModeBlast, 1},
		{"Fire Ball given a duration column", SpellRule{ID: 2, Area: true, Distribution: 3, Radius: 1, AreaDuration: 1}, areaModeCloud, 17},
		{"Fire Sacrifice's own columns", SpellRule{ID: 4, Area: true, Distribution: 5, Radius: 2}, areaModeRing, 4},
		{"Fire Sacrifice without the staged column", SpellRule{ID: 4, Area: true, Distribution: 3, Radius: 2}, areaModeBlast, 1},
		{"Acid Stream", SpellRule{ID: 9, Area: true, Distribution: 5, Radius: 3}, areaModeRing, 16},
		{"Meteor Storm, whose duration column the staged arm overwrites",
			SpellRule{ID: 21, Area: true, Distribution: 5, Radius: 4, AreaDuration: 10}, areaModeRing, 94},
		{"Wall of Fire", SpellRule{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}, areaModeCloud, 241},
	} {
		if got := areaModeFor(tc.rule); got != tc.mode {
			t.Errorf("%s: mode = %d, want %d", tc.name, got, tc.mode)
		}
		if got := areaLife(tc.rule, 0); got != tc.life {
			t.Errorf("%s: life = %d, want %d", tc.name, got, tc.life)
		}
	}
}
