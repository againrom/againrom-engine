package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseActionCadenceUsesShippedHumanoidWeaponAndSpellInputs is story
// 1045's lawful-install witness. The release gate runs it once on each root.
// Mission 90 supplies a placed Humanoid and his equipped weapon; mission 10
// supplies the generated mage, his spellbook and the installed spell table.
// Both actors then run through ordinary canonical commands in isolated worlds
// so mission AI cannot replace the action under measurement.
func TestReleaseActionCadenceUsesShippedHumanoidWeaponAndSpellInputs(t *testing.T) {
	f := releaseFront(t)

	t.Run("physical Humanoid", func(t *testing.T) {
		mission, err := StartMission(f.Archives.Containers, 90, f.Table, openDifficulty, nil)
		if err != nil {
			t.Fatalf("StartMission(90): %v", err)
		}
		loaded := entityByMapUnitID(t, mission.World, 42)
		if !loaded.Humanoid {
			t.Fatalf("mission-90 u42 is not classified Humanoid: %+v", loaded)
		}
		worn, ok := mission.World.EquippedItems(loaded.ID)
		if !ok || worn[0].Empty() {
			t.Fatalf("mission-90 u42 has no runtime weapon: %v, ok=%v", worn, ok)
		}
		weight, ok := itemWeightFor(mission.World.ItemWeights(), worn[0].Code)
		if !ok {
			t.Fatalf("mission-90 u42 weapon %#04x has no declared runtime weight", worn[0].Code)
		}

		attacker := cleanResistanceWitnessEntity(loaded, 1, 5, 5)
		attacker.Owner = sim.SelfSlot
		attacker.AlwaysHits, attacker.DamageBase, attacker.DamageSpread = true, 1, 0
		loadedTarget := entityByMapUnitID(t, mission.World, 290)
		targetWorn, ok := mission.World.EquippedItems(loadedTarget.ID)
		if !ok || targetWorn[0].Empty() {
			t.Fatalf("mission-90 u290 has no runtime weapon: %v, ok=%v", targetWorn, ok)
		}
		target := cleanResistanceWitnessEntity(loadedTarget, 2, 6, 5)
		target.Owner = 0
		target.HP, target.MaxHP = 1<<20, 1<<20
		w, err := sim.NewStockedWorld(1045, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical,
			sim.Terrain{}, []sim.Entity{attacker, target}, nil, sim.Relations{}, nil,
			[]sim.Stock{{ID: 1, EquippedItems: worn}, {ID: 2, EquippedItems: targetWorn}})
		if err != nil {
			t.Fatal(err)
		}
		if err := w.DeclareItemWeights(mission.World.ItemWeights()); err != nil {
			t.Fatalf("declare shipped item weights: %v", err)
		}

		penalty := (weight + 5*(30-attacker.Reaction)) / 12
		if penalty < 0 {
			penalty = 0
		} else if penalty > 12 {
			penalty = 12
		}
		minimum := uint64(attacker.AttackCharge + attacker.AttackRelax + penalty + 2)
		var applications []uint64
		lastHP := target.HP
		commands := []sim.Command{{Kind: sim.KindAttack, Entity: 1, X: 2}}
		for len(applications) < 3 && w.Tick() < 512 {
			sim.Step(w, commands)
			commands = nil
			if hp := entityByID(t, w, 2).HP; hp != lastHP {
				applications = append(applications, w.Tick())
				lastHP = hp
			}
		}
		if len(applications) != 3 {
			t.Fatalf("mission-90 loaded fighter produced %d applications in 512 ticks", len(applications))
		}
		for i := 1; i < len(applications); i++ {
			if gap := applications[i] - applications[i-1]; gap < minimum || gap > minimum+3 {
				t.Fatalf("loaded physical gap = %d, want charge %d + relax %d + penalty %d + U[0,3] + 2",
					gap, attacker.AttackCharge, attacker.AttackRelax, penalty)
			}
		}
	})

	t.Run("book command is one-shot", func(t *testing.T) {
		party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
		mission, err := StartMission(f.Archives.Containers, 10, f.Table, openDifficulty, party)
		if err != nil {
			t.Fatalf("StartMission(10): %v", err)
		}
		if len(mission.Start.IDs) != 1 {
			t.Fatalf("mission-10 mage ids = %v, want one generated party actor", mission.Start.IDs)
		}
		loaded := entityByID(t, mission.World, mission.Start.IDs[0])
		if !loaded.Humanoid || loaded.MaxMana <= 0 || loaded.KnownSpells == 0 {
			t.Fatalf("mission-10 generated mage lacks cadence inputs: %+v", loaded)
		}
		worn, _ := mission.World.EquippedItems(loaded.ID)
		caster := cleanResistanceWitnessEntity(loaded, 1, 5, 5)
		caster.ScanRange = 20
		target := sim.Entity{ID: 2, X: 6, Y: 5, HP: 1 << 19, MaxHP: 1 << 20, Owner: caster.Owner,
			Domain: sim.DomainGround, ScanRange: 5, Reach: 1}

		build := func(mana, manaRegen int32) *sim.World {
			c := caster
			c.Mana, c.MaxMana, c.ManaRegenPeriod = mana, max32(caster.MaxMana, mana), manaRegen
			// This cadence probe attacks an ally. Preserve that relation so a
			// damaging book cast cannot start an unrelated automatic staff fight.
			var relations sim.Relations
			relations.Set(c.Owner, c.Owner, 2)
			w, err := sim.NewStockedSpelledWorld(1045, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical,
				sim.Terrain{}, []sim.Entity{c, target}, nil, relations, nil,
				[]sim.Stock{{ID: 1, EquippedItems: worn}}, mission.World.Spells())
			if err != nil {
				t.Fatal(err)
			}
			if err := w.DeclareItemWeights(mission.World.ItemWeights()); err != nil {
				t.Fatalf("declare shipped item weights: %v", err)
			}
			return w
		}

		probe := build(caster.Mana, 0)
		var selected sim.SpellRule
		for _, rule := range probe.Spells() {
			if rule.ID >= 32 || caster.KnownSpells&(uint32(1)<<rule.ID) == 0 || rule.ManaCost == 0 {
				continue
			}
			if probe.BookSpellRefusal(1, 2, uint32(rule.ID)) == "" {
				selected = rule
				break
			}
		}
		if selected.ID == 0 {
			t.Fatalf("mission-10 generated mage has no admissible positive-cost unit spell; known=%#x", caster.KnownSpells)
		}

		oneShort := build(int32(selected.ManaCost)-1, 0)
		beforeShort := entityByID(t, oneShort, 1).Mana
		if events := sim.StepObserved(oneShort, []sim.Command{{Kind: sim.KindCast, Entity: 1, X: 2, Y: int32(selected.ID)}}); len(events) != 0 {
			t.Fatalf("one-short manual command released %+v", events)
		}
		if _, _, casting := oneShort.CastingSpell(1); casting || entityByID(t, oneShort, 1).Mana != beforeShort {
			t.Fatalf("one-short manual command retained state or spent mana: casting=%v mana=%d before=%d",
				casting, entityByID(t, oneShort, 1).Mana, beforeShort)
		}

		w := build(int32(selected.ManaCost), 0)
		commands := []sim.Command{{Kind: sim.KindCast, Entity: 1, X: 2, Y: int32(selected.ID)}}
		released := false
		for tick := 1; tick <= 512; tick++ {
			events := sim.StepObserved(w, commands)
			commands = nil
			if len(events) == 0 {
				continue
			}
			if len(events) != 1 || events[0].Spell != selected.ID || events[0].Weapon {
				t.Fatalf("loaded book release events = %+v, want one ordinary spell %d", events, selected.ID)
			}
			released = true
			break
		}
		if !released {
			t.Fatalf("loaded spell %d never released", selected.ID)
		}

		for tick := 1; tick <= 512; tick++ {
			events := sim.StepObserved(w, nil)
			if len(events) != 0 {
				t.Fatalf("one manual command repeated at tick %d: %+v", tick, events)
			}
		}
		if _, _, casting := w.CastingSpell(1); casting {
			t.Fatalf("loaded spell %d retained a manual order after 512 ticks", selected.ID)
		}
	})
}

func itemWeightFor(weights []sim.ItemWeight, code uint16) (int32, bool) {
	for _, weight := range weights {
		if weight.Code == code {
			return weight.Weight, true
		}
	}
	return 0, false
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
