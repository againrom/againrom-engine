package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func secondPayHealWorld(t *testing.T, casterMana, targetHP, targetMaxHP, delivery int32) (*World, SpellRule) {
	t.Helper()
	heal := secondRow(SpellRule{
		ID: 24, School: 5, ManaCost: 15, MaxRange: 8, Delivery: delivery,
		DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true,
	})
	caster := effectMage(1, 2, 2, 1<<24)
	caster.Owner, caster.Mana, caster.MaxMana = SelfSlot, casterMana, 200
	target := spEnt(2, 4, 2)
	target.Owner, target.HP, target.MaxHP = SelfSlot, targetHP, targetMaxHP
	return secondWorld(t, 7, Bounds{Width: 16, Height: 16}, []SpellRule{heal}, caster, target), heal
}

func TestSecondGameHealRefusalChargesMana(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		hp       int32
		maxHP    int32
		delivery int32
	}{
		{"dead at minus ten direct", -10, 100, 0},
		{"dead below minus ten delivery", -15, 100, 1},
		{"no health system", 0, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, heal := secondPayHealWorld(t, 100, tc.hp, tc.maxHP, tc.delivery)
			w.entities[1].HP, w.entities[1].MaxHP = tc.hp, tc.maxHP
			if tc.hp <= -10 {
				w.entities[1].Decay = DecayBones
			}
			// Command path charges mana once and does not heal.
			spRunCast(w, Cast(1, 2, 24))
			if got := spAt(t, w, 1).Mana; got != 100-heal.ManaCost {
				t.Fatalf("command cast mana = %d, want %d", got, 100-heal.ManaCost)
			}
			if got := spAt(t, w, 2).HP; got != tc.hp {
				t.Fatalf("target HP = %d, want unchanged %d", got, tc.hp)
			}
			// Direct castSpell also charges once and returns false.
			before := spAt(t, w, 1).Mana
			if w.castSpell(0, 2, 24, nil) {
				t.Fatal("castSpell succeeded on refused Heal target")
			}
			if got := spAt(t, w, 1).Mana; got != before-heal.ManaCost {
				t.Fatalf("direct castSpell mana = %d, want %d", got, before-heal.ManaCost)
			}
		})
	}
}

func TestSecondGamePointEffectRefusalChargesMana(t *testing.T) {
	t.Parallel()
	t.Run("control spirit on living target", func(t *testing.T) {
		rule := secondRow(SpellRule{ID: 22, School: 5, ManaCost: 30, MaxRange: 8, TargetsUnit: true, Delivery: 1})
		caster := effectMage(1, 2, 2, 1<<22)
		caster.Mana, caster.MaxMana = 100, 200
		living := spEnt(2, 4, 2)
		w := secondWorld(t, 9, Bounds{Width: 16, Height: 16}, []SpellRule{rule}, caster, living)
		spRunCast(w, Cast(1, 2, 22))
		if got := spAt(t, w, 1).Mana; got != 70 {
			t.Fatalf("command cast mana = %d, want 70", got)
		}
		if len(w.entities) != 2 {
			t.Fatalf("entities = %d, want 2", len(w.entities))
		}
		if w.castSpell(0, 2, 22, nil) {
			t.Fatal("direct castSpell succeeded on living Control Spirit target")
		}
		if got := spAt(t, w, 1).Mana; got != 40 {
			t.Fatalf("direct castSpell mana = %d, want 40", got)
		}
	})

	t.Run("zero duration buff", func(t *testing.T) {
		rule := secondRow(SpellRule{
			ID: 4, School: 1, ManaCost: 12, MaxRange: 8, TargetsUnit: true, Defensive: true,
			SpellDuration: 0, EffectKind: EffectProtectionFire, EffectMode: EffectDuration,
		})
		caster := effectMage(1, 2, 2, 1<<4)
		caster.Mana, caster.MaxMana = 100, 200
		target := spEnt(2, 4, 2)
		w := secondWorld(t, 11, Bounds{Width: 16, Height: 16}, []SpellRule{rule}, caster, target)
		spRunCast(w, Cast(1, 2, 4))
		if got := spAt(t, w, 1).Mana; got != 88 || len(w.attached) != 0 {
			t.Fatalf("mana = %d attached = %d, want 88 and 0", got, len(w.attached))
		}
	})
}

func TestSecondGameSummonFailedPlacementChargesMana(t *testing.T) {
	t.Parallel()
	t.Run("all 49 cells blocked", func(t *testing.T) {
		var ring []Entity
		id := EntityID(2)
		for y := int32(13); y <= 19; y++ {
			for x := int32(13); x <= 19; x++ {
				if x != 16 || y != 16 {
					ring = append(ring, spEnt(id, x, y))
					id++
				}
			}
		}
		w := summonWorld(t, 0x25, 0, ring...)
		before := spAt(t, w, 1).Mana
		spRunCast(w, Cast(1, 1, 25))
		if got := spAt(t, w, 1).Mana; got != before-100 {
			t.Fatalf("mana = %d, want %d", got, before-100)
		}
		if len(w.entities) != len(ring)+1 {
			t.Fatalf("entities = %d, want %d", len(w.entities), len(ring)+1)
		}
	})

	t.Run("no summon creature declared", func(t *testing.T) {
		w := summonWorld(t, 0x25, 0)
		w.DeclareSpellCreatures(nil)
		before := spAt(t, w, 1).Mana
		spRunCast(w, Cast(1, 1, 25))
		if got := spAt(t, w, 1).Mana; got != before-100 || len(w.entities) != 1 {
			t.Fatalf("mana = %d entities = %d, want %d and 1", got, len(w.entities), before-100)
		}
	})
}

func TestSecondGameAreaLandingRefusalChargesMana(t *testing.T) {
	t.Parallel()
	// Staged area row with no arm program: areaLandingRefusal refuses it.
	staged := secondRow(SpellRule{
		ID: 16, School: 4, ManaCost: 25, MaxRange: 8, Area: true,
		Distribution: distributionStaged, Radius: 2, TargetsUnit: true, Damaging: true,
		DamageMin: 5, DamageMax: 10, Delivery: 1,
	})
	t.Run("cell cast", func(t *testing.T) {
		caster := effectMage(1, 2, 2, 1<<16)
		caster.Mana, caster.MaxMana = 100, 200
		w := secondWorld(t, 13, Bounds{Width: 16, Height: 16}, []SpellRule{staged}, caster)
		spRunCast(w, CastAt(1, 16, CellPoint{X: 4, Y: 2}))
		if got := spAt(t, w, 1).Mana; got != 75 || len(w.effects) != 0 {
			t.Fatalf("cell command mana = %d effects = %d, want 75 and 0", got, len(w.effects))
		}
		if w.castBookAt(0, 4, 2, 16, nil) {
			t.Fatal("direct castBookAt succeeded on refused area")
		}
		if got := spAt(t, w, 1).Mana; got != 50 {
			t.Fatalf("direct castBookAt mana = %d, want 50", got)
		}
	})

	t.Run("unit cast", func(t *testing.T) {
		caster := effectMage(1, 2, 2, 1<<16)
		caster.Mana, caster.MaxMana = 100, 200
		victim := spEnt(2, 4, 2)
		w := secondWorld(t, 14, Bounds{Width: 16, Height: 16}, []SpellRule{staged}, caster, victim)
		spRunCast(w, Cast(1, 2, 16))
		if got := spAt(t, w, 1).Mana; got != 75 || len(w.effects) != 0 {
			t.Fatalf("unit command mana = %d effects = %d, want 75 and 0", got, len(w.effects))
		}
		if w.castSpell(0, 2, 16, nil) {
			t.Fatal("direct castSpell succeeded on refused area")
		}
		if got := spAt(t, w, 1).Mana; got != 50 {
			t.Fatalf("direct castSpell mana = %d, want 50", got)
		}
	})
}

func TestSecondGameInsufficientManaChargesNothing(t *testing.T) {
	t.Parallel()
	t.Run("heal at minus ten short on mana", func(t *testing.T) {
		w, _ := secondPayHealWorld(t, 14, -10, 100, 1)
		w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
		Step(w, []Command{Cast(1, 2, 24)})
		if got := spAt(t, w, 1).Mana; got != 14 {
			t.Fatalf("command cast charged mana: got %d, want 14", got)
		}
		if w.castSpell(0, 2, 24, nil) || spAt(t, w, 1).Mana != 14 {
			t.Fatalf("direct castSpell charged mana: got %d, want 14", spAt(t, w, 1).Mana)
		}
	})

	t.Run("area landing refusal short on mana", func(t *testing.T) {
		staged := secondRow(SpellRule{
			ID: 16, School: 4, ManaCost: 25, MaxRange: 8, Area: true,
			Distribution: distributionStaged, Radius: 2, Damaging: true, Delivery: 1,
		})
		caster := effectMage(1, 2, 2, 1<<16)
		caster.Mana, caster.MaxMana = 24, 200
		w := secondWorld(t, 15, Bounds{Width: 16, Height: 16}, []SpellRule{staged}, caster)
		Step(w, []Command{CastAt(1, 16, CellPoint{X: 4, Y: 2})})
		if got := spAt(t, w, 1).Mana; got != 24 {
			t.Fatalf("command CastAt charged mana: got %d, want 24", got)
		}
		if w.castBookAt(0, 4, 2, 16, nil) || spAt(t, w, 1).Mana != 24 {
			t.Fatalf("direct castBookAt charged mana: got %d, want 24", spAt(t, w, 1).Mana)
		}
	})

	t.Run("out of range stays free", func(t *testing.T) {
		w, _ := secondPayHealWorld(t, 100, -10, 100, 1)
		w.entities[1].X, w.entities[1].HP, w.entities[1].Decay = 14, -10, DecayBones
		if got := w.BookSpellRefusal(1, 2, 24); got != refusalTargetOutOfRange {
			t.Fatalf("refusal = %q, want %q", got, refusalTargetOutOfRange)
		}
		Step(w, []Command{Cast(1, 2, 24)})
		if got := spAt(t, w, 1).Mana; got != 100 {
			t.Fatalf("out-of-range cast charged mana: got %d, want 100", got)
		}
	})
}

func TestFirstGameHealRefusalFollowsPaymentAndAreaRefusalPrecedesIt(t *testing.T) {
	t.Parallel()
	t.Run("heal at minus ten", func(t *testing.T) {
		heal := SpellRule{ID: 6, School: 5, ManaCost: 15, MaxRange: 8, DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true}
		caster := effectMage(1, 2, 2, 1<<6)
		caster.Mana, caster.MaxMana = 100, 200
		target := spEnt(2, 4, 2)
		w := spWorld(t, 16, []SpellRule{heal}, caster, target)
		w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
		spRunCast(w, Cast(1, 2, 6))
		if got := spAt(t, w, 1).Mana; got != 85 {
			t.Fatalf("first-game Heal at -10 left %d mana, want 85 after the cost 15", got)
		}
	})

	t.Run("area landing refusal", func(t *testing.T) {
		staged := SpellRule{ID: 7, School: 2, ManaCost: 25, MaxRange: 8, Area: true, Distribution: distributionStaged, Radius: 1, TargetsUnit: true}
		caster := effectMage(1, 2, 2, 1<<7)
		caster.Mana, caster.MaxMana = 100, 200
		target := spEnt(2, 4, 2)
		w := spWorld(t, 17, []SpellRule{staged}, caster, target)
		spRunCast(w, CastAt(1, 7, CellPoint{X: 4, Y: 2}))
		if w.castBookAt(0, 4, 2, 7, nil) || w.castSpell(0, 2, 7, nil) || spAt(t, w, 1).Mana != 100 {
			t.Fatalf("first-game area refusal charged mana: got %d, want 100", spAt(t, w, 1).Mana)
		}
	})
}

func areaHostilityWorld(t *testing.T, second bool, victimHP, victimMaxHP int32, rel Relations) (*World, SpellRule, SpellRule) {
	t.Helper()
	dark := SpellRule{
		ID: 17, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
		TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200,
	}
	fire := SpellRule{
		ID: 2, School: 1, Area: true, Distribution: distributionDiamond, Radius: 1,
		TargetsUnit: true, Damaging: true, DamageMin: 1, DamageMax: 1,
	}
	var script *Script
	if second {
		dark = secondRow(SpellRule{
			ID: 14, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
			TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200,
		})
		fire = secondRow(fire)
		script = secondGameScript(t, nil, nil, nil)
	}
	caster := effectMage(1, 2, 2, (1<<dark.ID)|(1<<fire.ID))
	caster.Owner = SelfSlot
	victim := engFighter(2, 2, 6, 6)
	victim.HP, victim.MaxHP = victimHP, victimMaxHP
	w, err := NewStockedSpelledWorld(21, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{caster, victim}, script, rel, nil, nil, []SpellRule{dark, fire})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w, dark, fire
}

func pulseAreaAtVictim(w *World, rule SpellRule, mode uint8) {
	key := cellKey(w.entities[1].X, w.entities[1].Y)
	w.applyAreaCells(cellEffect{Key: key, Spell: rule.ID, Mode: mode, Caster: 1, HasCaster: true}, rule, []uint16{key})
}

func TestSecondGameAreaHitDefersReverseHostilityWhenNoConditionHolds(t *testing.T) {
	t.Parallel()
	w, dark, fire := areaHostilityWorld(t, true, 100, 100, Relations{})
	pulseAreaAtVictim(w, dark, areaModeCloud)
	if !w.relations.Hostile(SelfSlot, 2) {
		t.Fatal("forward hostility M[A][B] was not set")
	}
	if w.relations.Hostile(2, SelfSlot) {
		t.Fatal("reverse hostility M[B][A] flipped when no condition held")
	}
	if got := w.entities[1].AreaHitCount; got != 1 {
		t.Fatalf("AreaHitCount = %d, want 1", got)
	}
	// A damaging area hit leaving HP >= MaxHP/2 also defers reverse hostility and increments the counter.
	pulseAreaAtVictim(w, fire, areaModeBlast)
	if w.relations.Hostile(2, SelfSlot) {
		t.Fatal("damaging area hit flipped reverse hostility while victim stayed above half health")
	}
	if got := w.entities[1].AreaHitCount; got != 2 {
		t.Fatalf("AreaHitCount after Fireball = %d, want 2", got)
	}

	// Locked reverse relation returns without flipping or incrementing the counter.
	lockedRel := engRel(t, [3]uint32{2, SelfSlot, relationLocked})
	locked, dark2, _ := areaHostilityWorld(t, true, 100, 100, lockedRel)
	pulseAreaAtVictim(locked, dark2, areaModeCloud)
	if locked.relations.Hostile(2, SelfSlot) || locked.entities[1].AreaHitCount != 0 {
		t.Fatalf("locked reverse changed: byte %#x count %d", locked.relations.Byte(2, SelfSlot), locked.entities[1].AreaHitCount)
	}
}

func TestSecondGameAreaHitFlipsReverseWhenCounterExceedsTen(t *testing.T) {
	t.Parallel()
	rel := engRel(t, [3]uint32{2, SelfSlot, 0x10})
	w, dark, _ := areaHostilityWorld(t, true, 100, 100, rel)
	for hit := int32(1); hit <= 11; hit++ {
		pulseAreaAtVictim(w, dark, areaModeCloud)
		if w.relations.Hostile(2, SelfSlot) {
			t.Fatalf("hit %d flipped reverse hostility before counter exceeded 10", hit)
		}
		if got := w.entities[1].AreaHitCount; got != hit {
			t.Fatalf("hit %d: AreaHitCount = %d, want %d", hit, got, hit)
		}
	}
	// Hit 12 reads counter 11 (> 10), ORs bit 0 preserving upper bits, and increments to 12.
	pulseAreaAtVictim(w, dark, areaModeCloud)
	if got := w.relations.Byte(2, SelfSlot); got != 0x11 {
		t.Fatalf("hit 12: reverse byte = %#x, want 0x11", got)
	}
	if got := w.entities[1].AreaHitCount; got != 12 {
		t.Fatalf("hit 12: AreaHitCount = %d, want 12", got)
	}
	// Hit 13 sees M[B][A]&3 != 0 and returns without incrementing the counter further.
	pulseAreaAtVictim(w, dark, areaModeCloud)
	if got := w.entities[1].AreaHitCount; got != 12 {
		t.Fatalf("hit 13: AreaHitCount = %d, want 12 unchanged", got)
	}
}

func TestSecondGameAreaHitFlipsReverseBelowHalfHealth(t *testing.T) {
	t.Parallel()
	// Odd MaxHP=101: trunc(101/2) = 50. HP 50 defers; HP 49 flips.
	atHalf, dark, _ := areaHostilityWorld(t, true, 50, 101, Relations{})
	pulseAreaAtVictim(atHalf, dark, areaModeCloud)
	if atHalf.relations.Hostile(2, SelfSlot) || atHalf.entities[1].AreaHitCount != 1 {
		t.Fatalf("at half health (50/101): hostile=%v count=%d, want false and 1",
			atHalf.relations.Hostile(2, SelfSlot), atHalf.entities[1].AreaHitCount)
	}

	belowHalf, dark2, _ := areaHostilityWorld(t, true, 49, 101, Relations{})
	pulseAreaAtVictim(belowHalf, dark2, areaModeCloud)
	if !belowHalf.relations.Hostile(2, SelfSlot) || belowHalf.entities[1].AreaHitCount != 1 {
		t.Fatalf("below half health (49/101): hostile=%v count=%d, want true and 1",
			belowHalf.relations.Hostile(2, SelfSlot), belowHalf.entities[1].AreaHitCount)
	}

	// Damaging area spell dropping victim below half health flips on that hit.
	damaged, _, fire := areaHostilityWorld(t, true, 50, 100, Relations{})
	pulseAreaAtVictim(damaged, fire, areaModeBlast)
	if damaged.entities[1].HP >= 50 || !damaged.relations.Hostile(2, SelfSlot) || damaged.entities[1].AreaHitCount != 1 {
		t.Fatalf("post-damage below half: hp=%d hostile=%v count=%d",
			damaged.entities[1].HP, damaged.relations.Hostile(2, SelfSlot), damaged.entities[1].AreaHitCount)
	}
}

func TestSecondGameAreaHitFlipsReverseInDeadState(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1943: downed HP=0 holds state 0; state 0x10 needs HP <= -10.
	w, _, fire := areaHostilityWorld(t, true, 1, 1, Relations{})
	pulseAreaAtVictim(w, fire, areaModeBlast)
	if w.entities[1].Alive() || w.entities[1].HP != 0 {
		t.Fatalf("fixture: victim hp=%d alive=%v, want 0 and false", w.entities[1].HP, w.entities[1].Alive())
	}
	if w.relations.Hostile(2, SelfSlot) || w.entities[1].AreaHitCount != 1 {
		t.Fatalf("downed state condition: hostile=%v count=%d, want false and 1",
			w.relations.Hostile(2, SelfSlot), w.entities[1].AreaHitCount)
	}
	dead, _, _ := areaHostilityWorld(t, true, -10, -20, Relations{})
	dead.flipOnBlow(0, 1, blowArea)
	if !dead.relations.Hostile(2, SelfSlot) || dead.entities[1].AreaHitCount != 1 {
		t.Fatalf("dead state condition: hostile=%v count=%d, want true and 1",
			dead.relations.Hostile(2, SelfSlot), dead.entities[1].AreaHitCount)
	}
}

func TestSecondGameDirectBlowStillTurnsBothDirections(t *testing.T) {
	t.Parallel()
	w, _, _ := areaHostilityWorld(t, true, 100, 100, Relations{})
	w.flipOnBlow(0, 1, blowDirect)
	if !w.relations.Hostile(SelfSlot, 2) || !w.relations.Hostile(2, SelfSlot) {
		t.Fatalf("direct blow relations: fwd=%v rev=%v, want both true",
			w.relations.Hostile(SelfSlot, 2), w.relations.Hostile(2, SelfSlot))
	}
	if got := w.entities[1].AreaHitCount; got != 0 {
		t.Fatalf("direct blow changed AreaHitCount to %d, want 0", got)
	}
}

func TestFirstGameAreaHitStillTurnsBothDirectionsImmediately(t *testing.T) {
	t.Parallel()
	w, dark, _ := areaHostilityWorld(t, false, 100, 100, Relations{})
	pulseAreaAtVictim(w, dark, areaModeCloud)
	if !w.relations.Hostile(SelfSlot, 2) || !w.relations.Hostile(2, SelfSlot) {
		t.Fatalf("first-game area hit relations: fwd=%v rev=%v, want both true",
			w.relations.Hostile(SelfSlot, 2), w.relations.Hostile(2, SelfSlot))
	}
	if got := w.entities[1].AreaHitCount; got != 0 {
		t.Fatalf("first-game area hit changed AreaHitCount to %d, want 0", got)
	}
}

func TestSecondGameAreaHitCountSurvivesBinaryAndValuesRoundTrip(t *testing.T) {
	t.Parallel()
	w, dark, _ := areaHostilityWorld(t, true, 100, 100, Relations{})
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] == areaHitFormVersion {
		t.Fatalf("zero counters wrote wrapper version %d", areaHitFormVersion)
	}
	for range 5 {
		pulseAreaAtVictim(w, dark, areaModeCloud)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != areaHitFormVersion {
		t.Fatalf("form version = %d, want %d", form[0], areaHitFormVersion)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if loaded.Hash() != w.Hash() || loaded.entities[1].AreaHitCount != 5 {
		t.Fatalf("loaded count = %d hash equal = %v, want 5 and true", loaded.entities[1].AreaHitCount, loaded.Hash() == w.Hash())
	}
	// Continuing from the loaded world flips on the 12th cumulative hit (7 more hits).
	for range 6 {
		pulseAreaAtVictim(&loaded, dark, areaModeCloud)
	}
	if loaded.relations.Hostile(2, SelfSlot) {
		t.Fatal("flipped on hit 11 after reload")
	}
	pulseAreaAtVictim(&loaded, dark, areaModeCloud)
	if !loaded.relations.Hostile(2, SelfSlot) || loaded.entities[1].AreaHitCount != 12 {
		t.Fatalf("hit 12 after reload: hostile=%v count=%d", loaded.relations.Hostile(2, SelfSlot), loaded.entities[1].AreaHitCount)
	}

	// ActorValues carries AreaHitCount.
	v := w.entities[1].Values()
	if v.AreaHitCount != 5 {
		t.Fatalf("Values().AreaHitCount = %d, want 5", v.AreaHitCount)
	}
	restored := w.entities[1]
	restored.AreaHitCount = 0
	if err := restored.restoreValues(v); err != nil || restored.AreaHitCount != 5 {
		t.Fatalf("restoreValues: err=%v count=%d, want nil and 5", err, restored.AreaHitCount)
	}

	// ImportLoadedStrikes restores AreaHits onto the entity without needing a SavedActorOrder.
	fresh, _, _ := areaHostilityWorld(t, true, 100, 100, Relations{})
	if err := fresh.ImportLoadedStrikes([]LoadedStrike{{Entity: 2, Stamp: unstampedCommand, AreaHits: 9}}); err != nil {
		t.Fatalf("ImportLoadedStrikes: %v", err)
	}
	if got := fresh.entities[1].AreaHitCount; got != 9 {
		t.Fatalf("ImportLoadedStrikes count = %d, want 9", got)
	}

	// Malformed wrapper (zero count entry) is refused.
	bad := bytes.Clone(plain)
	ws := beginWrapper(bad)
	bad = binary.LittleEndian.AppendUint32(bad, 1)
	bad = binary.LittleEndian.AppendUint32(bad, 0)
	bad = endWrapper(bad, ws, areaHitFormVersion)
	var rejected World
	if err := rejected.UnmarshalBinary(bad); err == nil {
		t.Fatal("UnmarshalBinary accepted zero-count area-hit record")
	}
}

func TestSecondGameLightAndWallOfEarthCellsRunNoHitHook(t *testing.T) {
	t.Parallel()
	for _, id := range []uint16{15, 17} {
		row := secondRow(SpellRule{
			ID: id, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
			TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200,
		})
		caster := effectMage(1, 2, 2, 1<<row.ID)
		caster.Owner = SelfSlot
		victim := engFighter(2, 2, 6, 6)
		w, err := NewStockedSpelledWorld(21, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
			[]Entity{caster, victim}, secondGameScript(t, nil, nil, nil), Relations{}, nil, nil, []SpellRule{row})
		if err != nil {
			t.Fatalf("NewStockedSpelledWorld: %v", err)
		}
		pulseAreaAtVictim(w, row, areaModeCloud)
		if w.relations.Hostile(SelfSlot, 2) || w.entities[1].AreaHitCount != 0 {
			t.Fatalf("row %d ran the hit hook: hostile %v count %d", id, w.relations.Hostile(SelfSlot, 2), w.entities[1].AreaHitCount)
		}
	}
}

func TestSecondGamePoisonTickTurnsNeutralVictimAtOnce(t *testing.T) {
	t.Parallel()
	w, _, fire := areaHostilityWorld(t, true, 100, 100, Relations{})
	w.attached = append(w.attached, attachedEffect{
		Target: 2, Caster: 1, HasCaster: true, Spell: uint16(fire.ID),
		Kind: EffectHealth, Mode: EffectContinuous, Magnitude: 2, Remaining: 100,
	})
	w.applyPoisonAttachment(0, 1)
	if !w.relations.Hostile(SelfSlot, 2) || !w.relations.Hostile(2, SelfSlot) {
		t.Fatalf("one poison tick: fwd=%v rev=%v, want both true",
			w.relations.Hostile(SelfSlot, 2), w.relations.Hostile(2, SelfSlot))
	}
	if got := w.entities[1].AreaHitCount; got != 0 {
		t.Fatalf("poison tick changed AreaHitCount to %d, want 0", got)
	}
}
