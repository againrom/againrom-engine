package sim

import "testing"

// secondRow is a second-game row as the loader hands it over: its arm set from
// the second game's table.
func secondRow(r SpellRule) SpellRule {
	rows := []SpellRule{r}
	AssignSecondGameArms(rows)
	return rows[0]
}

func TestSecondGameArmsFollowThePublishedTable(t *testing.T) {
	want := map[uint16]uint16{1: 1, 4: 5, 5: ArmNone, 6: 8, 7: ArmNone, 10: 13, 11: 14, 12: 15,
		15: 12, 16: ArmNone, 18: 20, 22: 25, 23: 26, 24: 6, 25: ArmNone, 26: 11, 27: 18, 28: 27, 29: 28,
		0: ArmNone, 30: ArmNone}
	for id, arm := range want {
		if got := SecondGameSpellArm(id); got != arm {
			t.Errorf("SecondGameSpellArm(%d) = %d, want %d", id, got, arm)
		}
	}
	first := SpellRule{ID: 24}
	if first.ArmID() != 24 || first.Second {
		t.Errorf("a first-game row runs arm %d, want its own id", first.ArmID())
	}
}

func TestSecondHealDrawsOneToSpread(t *testing.T) {
	heal := secondRow(SpellRule{ID: 24, School: 5, DamageMin: 8, DamageMax: 16, TargetsUnit: true, Restorative: true})
	target := spEnt(2, 1, 0)
	target.HP = 10
	w := spWorld(t, 7, []SpellRule{heal}, spEnt(1, 0, 0), target)
	base, spread := spellDamage(heal.DamageMin, heal.DamageMax, 0)
	probe := w.rng
	want := int32(10 + base + 1 + int64(probe.uniform(int32(spread-1))))
	w.applySpellHealing(1, heal, 0)
	if got := w.entities[1].HP; got != want {
		t.Fatalf("second-game Heal left health %d, want %d (base %d + U[1,%d])", got, want, base, spread)
	}

	none := secondRow(SpellRule{ID: 24, School: 5, DamageMin: 8, DamageMax: 8, TargetsUnit: true, Restorative: true})
	w = spWorld(t, 7, []SpellRule{none}, spEnt(1, 0, 0), target)
	w.applySpellHealing(1, none, 0)
	if got := w.entities[1].HP; got != 10+8+1 {
		t.Fatalf("second-game Heal with spread 0 left health %d, want 19 (a spread of 0 still draws 1)", got)
	}
}

func TestSecondDrainLifeIsCutByAstralResistance(t *testing.T) {
	if got := secondDrainAmount(4, 25, 1); got != (4+1+1)*75/100 {
		t.Errorf("secondDrainAmount(4, astral 25, draw 1) = %d, want %d", got, (4+1+1)*75/100)
	}
	if got := secondDrainAmount(4, 150, 0); got != 0 {
		t.Errorf("astral resistance above 100 left %d, want 0", got)
	}
	drain := secondRow(SpellRule{ID: 26, School: 5, DamageMin: 40, DamageMax: 60, TargetsUnit: true})
	if drain.Damaging || drain.ArmID() != 11 {
		t.Fatalf("second-game Drain Life row runs arm %d damaging=%v, want arm 11 not damaging", drain.ArmID(), drain.Damaging)
	}
	caster, victim := spEnt(1, 0, 0), spEnt(2, 1, 0)
	caster.HP = 50
	victim.Protection[4] = 50
	w := spWorld(t, 9, []SpellRule{drain}, caster, victim)
	base, spread := spellDamage(drain.DamageMin, drain.DamageMax, 0)
	probe := w.rng
	amount := secondDrainAmount(base, 50, probe.uniform(secondDrawSpan(spread)))
	if !w.ordinaryEffectPayload(0, 1, drain, 0) {
		t.Fatal("second-game Drain Life did not land")
	}
	if got := w.entities[1].HP; int64(got) != 100-amount {
		t.Errorf("victim health %d, want %d", got, 100-amount)
	}
	if got := w.entities[0].HP; int64(got) != 50+amount {
		t.Errorf("caster health %d, want %d", got, 50+amount)
	}
}

func TestSecondPoisonCloudAndInvisibilityTakeTheirOwnFormulas(t *testing.T) {
	poison := secondRow(SpellRule{ID: 6, School: 2, Area: true, Distribution: 3, SpellDuration: 5,
		EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 8})
	w := spWorld(t, 1, []SpellRule{poison}, spEnt(1, 0, 0))
	_, mag, duration, _ := w.pointEffect(0, poison, 90)
	if mag != -6 {
		t.Errorf("second-game Poison Cloud magnitude at P 90 = %d, want -6 (-2 * (90/45 + 1))", mag)
	}
	if want := lastingTicks(90, 5, durationSlow); duration != want {
		t.Errorf("second-game Poison Cloud duration at P 90 = %d, want %d (Duration 5 on the 1.025 law)", duration, want)
	}
	first := poison
	first.ID, first.Arm, first.Second = 8, 0, false
	if _, mag, duration, _ := w.pointEffect(0, first, 90); mag != -8 || duration != 8 {
		t.Errorf("first-game Poison Cloud at P 90 = %d for %d ticks, want -8 for 8", mag, duration)
	}

	invisible := secondRow(SpellRule{ID: 12, School: 3, TargetsUnit: true, SpellDuration: 10})
	if _, _, duration, _ := w.pointEffect(0, invisible, 70); duration != 70<<4 {
		t.Errorf("second-game Invisibility duration at P 70 = %d, want %d (P << 4)", duration, 70<<4)
	}
}

func TestSecondStoneCurseWithNoEffectKindLandsAndAttachesNothing(t *testing.T) {
	curse := secondRow(SpellRule{ID: 18, School: 4, TargetsUnit: true, SpellDuration: 10})
	w := spWorld(t, 3, []SpellRule{curse}, spEnt(1, 0, 0), spEnt(2, 1, 0))
	if r := w.pointEffectRefusal(0, 1, curse, 50); r != "" {
		t.Fatalf("second-game Stone Curse refused: %s", r)
	}
	before := w.entities[1]
	if !w.ordinaryEffectPayload(0, 1, curse, 50) {
		t.Fatal("second-game Stone Curse did not land")
	}
	if len(w.attached) != 0 || w.entities[1].HP != before.HP {
		t.Error("second-game Stone Curse changed its target")
	}
	first := curse
	first.ID, first.Arm, first.Second = 20, 0, false
	if r := w.pointEffectRefusal(0, 1, first, 50); r == "" {
		t.Error("a first-game Stone Curse row with no effect kind was admitted")
	}
}

func TestSpellRuleLandsRefusesAStagedAreaWithNoProgram(t *testing.T) {
	blizzard := secondRow(SpellRule{ID: 7, School: 2, Area: true, Distribution: distributionStaged, Radius: 4, DamageMin: 10, DamageMax: 30, Damaging: true})
	if SpellRuleLands(blizzard) {
		t.Error("Blizzard, a staged area with no cell program, lands")
	}
	acid := secondRow(SpellRule{ID: 9, School: 2, Area: true, Distribution: distributionStaged, Radius: 3, DamageMin: 10, DamageMax: 14, Damaging: true})
	if !SpellRuleLands(acid) {
		t.Error("Acid Stream, which runs the first-game staged program, does not land")
	}
	ice := secondRow(SpellRule{ID: 5, School: 2, TargetsUnit: true, DamageMin: 5, DamageMax: 10, Damaging: true})
	if !SpellRuleLands(ice) {
		t.Error("Ice Missile does not land")
	}
	summon := secondRow(SpellRule{ID: 25, School: 5})
	if SpellRuleLands(summon) {
		t.Error("Summon, with no arm and no effect, lands")
	}
}

func TestCreatureAimOfARowWithNoArmFollowsItsColumns(t *testing.T) {
	ice := secondRow(SpellRule{ID: 5, TargetsUnit: true, Damaging: true, DamageMin: 5, DamageMax: 10})
	blizzard := secondRow(SpellRule{ID: 7, Area: true, Distribution: distributionStaged, Damaging: true, DamageMin: 10, DamageMax: 30})
	summon := secondRow(SpellRule{ID: 25})
	heal := secondRow(SpellRule{ID: 24, Restorative: true, TargetsUnit: true})
	w := spWorld(t, 1, []SpellRule{ice, blizzard, summon, heal})
	for id, want := range map[uint32]creatureAim{5: creatureAimVictim, 7: creatureAimVictimCell, 25: creatureAimNone, 24: creatureAimCaster} {
		if got := w.creatureAim(id); got != want {
			t.Errorf("creatureAim(%d) = %d, want %d", id, got, want)
		}
	}
}
