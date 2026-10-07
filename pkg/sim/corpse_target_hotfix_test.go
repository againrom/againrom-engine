package sim

import (
	"bytes"
	"strconv"
	"testing"
)

func TestOrdinaryTargetBoundaryIsStrictlyAboveMinusTen(t *testing.T) {
	for _, tc := range []struct {
		hp   int32
		want bool
	}{{0, true}, {-1, true}, {-9, true}, {-10, false}, {-11, false}} {
		t.Run(stringHP(tc.hp), func(t *testing.T) {
			attacker := cbFighter(1, 0, 0, 30, 0)
			target := cbEnt(2, 8, 0)
			target.HP = tc.hp
			w := cbWorld(t, 244, attacker, target)
			Step(w, []Command{cbOrder(1, 2)})
			if got := cbAt(t, w, 1).HasAttackTarget; got != tc.want {
				t.Fatalf("attack target at HP %d: held=%v, want %v", tc.hp, got, tc.want)
			}

			rule := SpellRule{ID: 1, School: 1, MaxRange: 12, TargetsUnit: true,
				Damaging: true, DamageMin: 1, DamageMax: 1}
			mage := effectMage(1, 0, 0, 1<<1)
			spellTarget := spEnt(2, 2, 0)
			spellTarget.HP = tc.hp
			sw := spWorld(t, 244, []SpellRule{rule}, mage, spellTarget)
			refused := sw.bookSpellRefusal(0, 2, 1, false, true) != ""
			if got := !refused; got != tc.want {
				t.Fatalf("spell target at HP %d: admitted=%v, want %v", tc.hp, got, tc.want)
			}
		})
	}
}

func TestFallenSpellTargetPopulationAdmitsHealButNotSupport(t *testing.T) {
	damage := SpellRule{ID: 1, Damaging: true}
	drain := SpellRule{ID: 11}
	poison := SpellRule{ID: 8, EffectKind: EffectHealth, EffectMagnitude: -1}
	heal := SpellRule{ID: 6, Restorative: true}
	buff := SpellRule{ID: 5, Defensive: true, EffectKind: EffectProtectionFire, EffectMagnitude: 5}
	control := SpellRule{ID: controlSpiritSpellID}

	for _, tc := range []struct {
		name string
		hp   int32
		rule SpellRule
		want bool
	}{
		{"living heal", 1, heal, true},
		{"downed damage", 0, damage, true},
		{"fallen damage", -9, damage, true},
		{"fallen drain", -9, drain, true},
		{"fallen poison", -9, poison, true},
		{"downed heal", 0, heal, true},
		{"fallen heal", -1, heal, true},
		{"fallen buff", -9, buff, false},
		{"threshold damage", -10, damage, false},
		{"threshold heal", -10, heal, false},
		{"past-threshold poison", -11, poison, false},
		{"control spirit exception", -10, control, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := spEnt(2, 3, 3)
			target.HP = tc.hp
			if got := spellTargetable(target, tc.rule); got != tc.want {
				t.Fatalf("spellTargetable(HP %d, rule %+v) = %v, want %v", tc.hp, tc.rule, got, tc.want)
			}
		})
	}
}

func TestAnAttachedHealthEffectCannotReviveAfterItsTargetFalls(t *testing.T) {
	t.Run("positive continuous tick", func(t *testing.T) {
		target := spEnt(2, 3, 3)
		target.HP = 50
		w := spWorld(t, 0x250, []SpellRule{{ID: 30}}, target)
		if !w.attachEffect(2, 0, SpellRule{ID: 30}, EffectHealth, 10, 9, EffectContinuous) {
			t.Fatal("setup health effect did not attach")
		}
		w.entities[0].HP = -1
		w.clearFelled(0)
		Step(w, nil) // remaining 9 -> 8, the continuous application tick
		if got := cbAt(t, w, 2); got.HP != -1 || got.Decay != DecayFallen {
			t.Fatalf("continuous support revived to HP %d decay %d", got.HP, got.Decay)
		}
	})

	t.Run("expiry reversal", func(t *testing.T) {
		target := spEnt(2, 3, 3)
		target.HP = 5
		w := spWorld(t, 0x251, []SpellRule{{ID: 31}}, target)
		if !w.attachEffect(2, 0, SpellRule{ID: 31}, EffectHealth, -10, 1, EffectDuration) {
			t.Fatal("setup health penalty did not attach")
		}
		if got := cbAt(t, w, 2); got.HP != -5 || got.Decay != DecayFallen {
			t.Fatalf("setup fall = HP %d decay %d, want -5/%d", got.HP, got.Decay, DecayFallen)
		}
		Step(w, nil) // expiry attempts to give the ten points back
		if got := cbAt(t, w, 2); got.HP != -5 || got.Decay != DecayFallen {
			t.Fatalf("expiry reversal revived to HP %d decay %d", got.HP, got.Decay)
		}
	})
}

func TestScriptCastAdmissionNeverHashesAnInvalidCorpseTarget(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hp    int32
		spell uint16
		want  bool
	}{
		{"living support", 1, scInertSpell, true},
		{"finish with damage", -9, scDamageSpell, true},
		{"heal at a fallen body", -1, scHealSpell, true},
		{"no support at a fallen body", -9, scInertSpell, false},
		{"no damage at the threshold", -10, scDamageSpell, false},
		{"control spirit exception", -10, controlSpiritSpellID, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := scWorld(t, mustScript(t, nil, nil, nil))
			ti := indexOfEntity(w.entities, scTarget)
			w.entities[ti].HP = tc.hp
			if tc.hp <= 0 {
				w.entities[ti].Decay = DecayFallen
			}
			if tc.hp <= decayBonesHP {
				w.entities[ti].Decay = DecayBones
			}
			before := w.Hash()
			w.castAtUnit(unitNode(20, 15, tc.spell, 99, scTarget))
			got := len(w.casts) == 1
			if got != tc.want {
				t.Fatalf("pending script cast at HP %d = %v (%+v), want %v", tc.hp, got, w.casts, tc.want)
			}
			if (w.Hash() != before) != tc.want {
				t.Fatalf("hash movement at HP %d = %v, want %v", tc.hp, w.Hash() != before, tc.want)
			}
		})
	}
}

func TestFallingClearsSupportReferencesButKeepsFinishingDamage(t *testing.T) {
	damage := SpellRule{ID: 1, Damaging: true, TargetsUnit: true}
	heal := SpellRule{ID: 6, Restorative: true, TargetsUnit: true}
	buff := SpellRule{ID: 5, Defensive: true, EffectKind: EffectProtectionFire, EffectMagnitude: 5}
	control := SpellRule{ID: controlSpiritSpellID}
	body := spEnt(2, 3, 3)
	w := spWorld(t, 0x249, []SpellRule{damage, heal, buff, control}, body)
	w.bookCasts = []bookCast{
		{Target: 2, Spell: 1}, {Target: 2, Spell: 6}, {Target: 2, Spell: 5}, {Target: 2, Spell: controlSpiritSpellID},
	}
	w.casts = []scriptCast{
		{Target: 2, Spell: 1, AtUnit: true}, {Target: 2, Spell: 6, AtUnit: true},
		{Target: 2, Spell: 5, AtUnit: true}, {Target: 2, Spell: controlSpiritSpellID, AtUnit: true},
	}

	w.entities[0].HP = -1
	w.clearFelled(0)
	if len(w.bookCasts) != 3 || w.bookCasts[0].Spell != 1 || w.bookCasts[1].Spell != 6 ||
		w.bookCasts[2].Spell != controlSpiritSpellID {
		t.Fatalf("book references after fall = %+v, want damage, Heal and Control Spirit", w.bookCasts)
	}
	if len(w.casts) != 3 || w.casts[0].Spell != 1 || w.casts[1].Spell != 6 ||
		w.casts[2].Spell != controlSpiritSpellID {
		t.Fatalf("script references after fall = %+v, want damage, Heal and Control Spirit", w.casts)
	}
}

func stringHP(hp int32) string {
	return strconv.FormatInt(int64(hp), 10)
}

func TestFirstAttackerCrossingMinusTenClearsEveryAttackerWithoutExtraRNG(t *testing.T) {
	build := func(armSecond bool) *World {
		a1 := cbFighter(1, 0, 0, 1, 20)
		a2 := cbFighter(2, 2, 0, 1, 20)
		victim := cbEnt(3, 1, 0)
		victim.HP = 0
		w := cbWorld(t, 245, a1, a2, victim)
		for _, id := range []EntityID{1, 2} {
			if id == 2 && !armSecond {
				continue
			}
			i := indexOfEntity(w.entities, id)
			w.entities[i].AttackTarget, w.entities[i].HasAttackTarget = 3, true
			w.entities[i].AttackPhase, w.entities[i].AttackCountdown = AttackCharging, 1
		}
		return w
	}

	withSecond := build(true)
	withoutSecond := build(false)
	Step(withSecond, nil)
	Step(withoutSecond, nil)

	if hp := cbAt(t, withSecond, 3).HP; hp != -10 {
		t.Fatalf("two armed attackers left the target at %d, want -10", hp)
	}
	for _, id := range []EntityID{1, 2} {
		if e := cbAt(t, withSecond, id); e.HasAttackTarget || e.HasTarget {
			t.Errorf("attacker %d retained attack=%v route-target=%v", id, e.HasAttackTarget, e.HasTarget)
		}
	}
	if got, want := withSecond.rng.state, withoutSecond.rng.state; got != want {
		t.Fatalf("the cleared higher-id attack consumed RNG: %016x != %016x", got, want)
	}
}

func TestDecayFromMinusNineClearsRoutesAndOrdinaryCastsButKeepsControlSpirit(t *testing.T) {
	attacker := cbFighter(1, 0, 0, 10, 0)
	attacker.AttackTarget, attacker.HasAttackTarget = 2, true
	attacker.TargetX, attacker.TargetY, attacker.HasTarget = 3, 0, true
	body := cbEnt(2, 3, 0)
	body.HP, body.Decay, body.Dwell = -9, DecayFallen, 0
	caster1, caster2 := effectMage(3, 0, 1, 1<<1), effectMage(4, 0, 2, 1<<25)
	w := spWorld(t, 246, []SpellRule{{ID: 1, Damaging: true, TargetsUnit: true}, {ID: 25}},
		attacker, body, caster1, caster2)
	w.routes[0] = []cell{{x: 1, y: 0}, {x: 2, y: 0}, {x: 3, y: 0}}
	w.bookCasts = []bookCast{
		{Caster: 3, Target: 2, Spell: 1, Phase: bookPending, Retained: true},
		{Caster: 4, Target: 2, Spell: controlSpiritSpellID, Phase: bookPending, Retained: true},
	}
	w.casts = []scriptCast{
		{Spell: 1, Power: 1, Target: 2, AtUnit: true},
		{Spell: controlSpiritSpellID, Power: 1, Target: 2, AtUnit: true},
	}
	w.tick = decayPhase
	w.decayPass()

	if hp := cbAt(t, w, 2).HP; hp != -10 {
		t.Fatalf("decay left body at %d, want -10", hp)
	}
	if e := cbAt(t, w, 1); e.HasAttackTarget || e.HasTarget || len(w.routes[0]) != 0 {
		t.Fatalf("decay left attack=%v target=%v route=%v", e.HasAttackTarget, e.HasTarget, w.routes[0])
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Spell != controlSpiritSpellID {
		t.Fatalf("pending book casts after cleanup: %+v", w.bookCasts)
	}
	if len(w.casts) != 1 || w.casts[0].Spell != controlSpiritSpellID {
		t.Fatalf("pending script casts after cleanup: %+v", w.casts)
	}
}

func TestAIAndScriptAttackDoNotAcquirePastMinusTen(t *testing.T) {
	for _, tc := range []struct {
		hp   int32
		want bool
	}{{-9, true}, {-10, false}} {
		t.Run(stringHP(tc.hp), func(t *testing.T) {
			actor := engFighter(1, 1, 5, 5)
			target := engFighter(2, 2, 6, 5)
			target.HP = tc.hp
			w := engWorld(t, engRel(t, [3]uint32{1, 2, relationHostile}), actor, target)
			got := w.candidates(aiGroup{owner: 1, members: []int{0}})
			if (len(got) == 1 && got[0] == 1) != tc.want {
				t.Fatalf("AI candidates at HP %d: %v", tc.hp, got)
			}

			named := laFighter(1, 2, 7, 5, 5)
			named.HP = tc.hp
			member := laFighter(2, 2, 7, 6, 5)
			member.TargetX, member.TargetY, member.HasTarget = 10, 10, true
			sw := engWorld(t, engRel(t), named, member)
			sw.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
				Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandAttack}})
			gotMember := laEnt(t, sw, 2)
			if tc.want {
				if !gotMember.HasAttackTarget || gotMember.AttackTarget != 1 {
					t.Fatalf("script did not attack finishable HP %d: %+v", tc.hp, gotMember)
				}
			} else if gotMember.HasAttackTarget || !gotMember.HasTarget {
				t.Fatalf("refused script target disrupted the existing order: %+v", gotMember)
			}
		})
	}
}

func TestLoadNormalisesStaleCorpseReferencesAndRoundTrips(t *testing.T) {
	attacker := cbFighter(1, 0, 0, 10, 0)
	attacker.AttackTarget, attacker.HasAttackTarget = 2, true
	body := cbEnt(2, 3, 0)
	body.HP = -9
	caster1, caster2 := effectMage(3, 0, 1, 1<<1), effectMage(4, 0, 2, 1<<25)
	w := spWorld(t, 247, []SpellRule{{ID: 1, Damaging: true, TargetsUnit: true}, {ID: 25}},
		attacker, body, caster1, caster2)
	w.bookCasts = []bookCast{
		{Caster: 3, Target: 2, Spell: 1, Phase: bookPending, Retained: true},
		{Caster: 4, Target: 2, Spell: controlSpiritSpellID, Phase: bookPending, Retained: true},
	}
	w.casts = []scriptCast{
		{Spell: 1, Power: 1, Target: 2, AtUnit: true},
		{Spell: controlSpiritSpellID, Power: 1, Target: 2, AtUnit: true},
	}
	vi := indexOfEntity(w.entities, 2)
	w.entities[vi].HP, w.entities[vi].Decay, w.entities[vi].Dwell = -10, DecayBones, 0

	stale, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary stale form: %v", err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(stale); err != nil {
		t.Fatalf("UnmarshalBinary rejected a stale target: %v", err)
	}
	if e := cbAt(t, &loaded, 1); e.HasAttackTarget || e.HasTarget {
		t.Fatalf("loaded attacker retained attack=%v target=%v", e.HasAttackTarget, e.HasTarget)
	}
	if len(loaded.bookCasts) != 1 || loaded.bookCasts[0].Spell != controlSpiritSpellID ||
		len(loaded.casts) != 1 || loaded.casts[0].Spell != controlSpiritSpellID {
		t.Fatalf("loaded casts were not normalised: books=%+v scripts=%+v", loaded.bookCasts, loaded.casts)
	}

	first, err := loaded.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary normalised form: %v", err)
	}
	var again World
	if err := again.UnmarshalBinary(first); err != nil {
		t.Fatalf("second UnmarshalBinary: %v", err)
	}
	second, err := again.MarshalBinary()
	if err != nil {
		t.Fatalf("second MarshalBinary: %v", err)
	}
	if !bytes.Equal(first, second) || loaded.Hash() != again.Hash() {
		t.Fatalf("normalised roundtrip drifted: bytes=%v hash=%016x/%016x",
			bytes.Equal(first, second), loaded.Hash(), again.Hash())
	}
}

func TestLoadKeepsHealAndFinishersAtAFallenBody(t *testing.T) {
	damage := SpellRule{ID: 1, Damaging: true, TargetsUnit: true}
	heal := SpellRule{ID: 6, Restorative: true, TargetsUnit: true}
	control := SpellRule{ID: controlSpiritSpellID}
	body := spEnt(2, 3, 0)
	body.HP, body.Decay = -1, DecayFallen
	damageCaster := effectMage(3, 0, 1, 1<<1)
	healCaster := effectMage(4, 0, 2, 1<<6)
	controlCaster := effectMage(5, 0, 3, 1<<controlSpiritSpellID)
	w := spWorld(t, 0x252, []SpellRule{damage, heal, control}, body, damageCaster, healCaster, controlCaster)
	w.bookCasts = []bookCast{
		{Caster: 3, Target: 2, Spell: 1, Phase: bookPending, Retained: true},
		{Caster: 4, Target: 2, Spell: 6, Phase: bookPending, Retained: true},
		{Caster: 5, Target: 2, Spell: controlSpiritSpellID, Phase: bookPending, Retained: true},
	}
	w.casts = []scriptCast{
		{Target: 2, Spell: 1, Power: 1, AtUnit: true},
		{Target: 2, Spell: 6, Power: 1, AtUnit: true},
		{Target: 2, Spell: controlSpiritSpellID, Power: 1, AtUnit: true},
	}

	stale, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary stale fallen casts: %v", err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(stale); err != nil {
		t.Fatalf("UnmarshalBinary stale fallen casts: %v", err)
	}
	if len(loaded.bookCasts) != 3 || loaded.bookCasts[0].Spell != 1 || loaded.bookCasts[1].Spell != 6 ||
		loaded.bookCasts[2].Spell != controlSpiritSpellID {
		t.Fatalf("loaded book casts = %+v, want damage, Heal and Control Spirit", loaded.bookCasts)
	}
	if len(loaded.casts) != 3 || loaded.casts[0].Spell != 1 || loaded.casts[1].Spell != 6 ||
		loaded.casts[2].Spell != controlSpiritSpellID {
		t.Fatalf("loaded script casts = %+v, want damage, Heal and Control Spirit", loaded.casts)
	}

	first, err := loaded.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary normalized fallen casts: %v", err)
	}
	var again World
	if err := again.UnmarshalBinary(first); err != nil {
		t.Fatalf("second UnmarshalBinary: %v", err)
	}
	second, err := again.MarshalBinary()
	if err != nil {
		t.Fatalf("second MarshalBinary: %v", err)
	}
	if !bytes.Equal(first, second) || loaded.Hash() != again.Hash() {
		t.Fatalf("fallen-cast normalization drifted: bytes=%v hash=%016x/%016x",
			bytes.Equal(first, second), loaded.Hash(), again.Hash())
	}
}
