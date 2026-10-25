package game

import (
	"math"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// castDamageRecord is the spell record pair in float64, computed here and not
// in the simulation package: f = power/30 + 1, base the low byte of
// trunc(dmin*f), spread the low byte of trunc(dmax*f - base) (TEXT-096).
func castDamageRecord(dmin, dmax, power int32) (int64, int64) {
	f := float64(float64(power)/30.0) + 1.0
	var base, spread uint8
	if dmin > 0 {
		base = uint8(int64(float64(float64(dmin) * f)))
	}
	if dmax > 0 {
		spread = uint8(int64(float64(float64(float64(dmax)*f) - float64(base))))
	}
	return int64(base), int64(spread)
}

// castDamageResisted is MAGIC-301's elemental resistance step in float64.
func castDamageResisted(v int64, r int32) int64 {
	if r == 0 {
		return v
	}
	return max(int64(int32(float64(float64(float64(v*(100-int64(r)))/100.0)+0.75))), 0)
}

// TestReleaseCastDamageFollowsTheSpellRecord casts the installed harmful unit
// spells from mission 10's generated mage at a fixed power on a target with a
// negative resistance, over 96 seeds per spell. Every landed amount lies in
// the resisted record interval and both ends are reached: the payload is the
// record pair, the spread is inclusive and the resistance is signed
// (MAGIC-300, MAGIC-301). Stone Curse's lifetime on a target with negative
// Earth resistance is D*(100-r)/100 (MAGIC-303).
func TestReleaseCastDamageFollowsTheSpellRecord(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	installed := mapload.SpellRules(f.Table)
	if len(installed) != 28 {
		t.Fatalf("installed row census has %d rows, want 28", len(installed))
	}
	for _, rule := range installed {
		if rule.Second {
			t.Fatal("first-game row census contains a second-game rule")
		}
		for power := int32(0); power <= 255; power++ {
			b, s := sim.SpellColumnDamage(rule.DamageMin, rule.DamageMax, power)
			wb, ws := castDamageRecord(rule.DamageMin, rule.DamageMax, power)
			if b != wb || s != ws {
				t.Fatalf("spell %d power %d: pair %d/%d differs from former %d/%d", rule.ID, power, b, s, wb, ws)
			}
		}
	}
	t.Logf("%s: all %d installed rows x 256 powers = %d unchanged pairs", os.Getenv("AGAINROM_ASSETS"), len(installed), len(installed)*256)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	mission, err := StartMission(f.Archives.Containers, 10, f.Table, openDifficulty, party)
	if err != nil {
		t.Fatalf("StartMission(10): %v", err)
	}
	if len(mission.Start.IDs) != 1 {
		t.Fatalf("mission-10 party ids = %v, want one generated mage", mission.Start.IDs)
	}
	mage := entityByID(t, mission.World, mission.Start.IDs[0])
	spells := mission.World.Spells()
	byID := map[uint16]sim.SpellRule{}
	for _, rule := range spells {
		byID[rule.ID] = rule
	}

	worn, _ := mission.World.EquippedItems(mage.ID)
	const power = 20
	const resistance = -20
	mind := int32(30)
	world := func(seed uint64, rule sim.SpellRule, protection [5]int32) *sim.World {
		caster := cleanResistanceWitnessEntity(mage, 1, 5, 5)
		caster.Owner = sim.SelfSlot
		caster.KnownSpells = 1 << rule.ID
		caster.Mind = mind
		caster.Skill[rule.School] = power
		caster.Mana, caster.MaxMana = 1000, 1000
		caster.ScanRange = 20
		target := sim.Entity{ID: 2, X: 7, Y: 5, HP: 1 << 19, MaxHP: 1 << 20, Owner: 3,
			Domain: sim.DomainGround, ScanRange: 5, Reach: 1, TokenSize: 1, Protection: protection}
		var relations sim.Relations
		relations.Set(caster.Owner, 3, 1)
		relations.Set(3, caster.Owner, 1)
		w, err := sim.NewStockedSpelledWorld(seed, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical,
			sim.Terrain{}, []sim.Entity{caster, target}, nil, relations, nil,
			[]sim.Stock{{ID: 1, EquippedItems: worn}}, spells)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}

	for _, id := range []uint16{1, 2, 3, 9, 13, 14, 21} {
		rule := byID[id]
		base, spread := castDamageRecord(rule.DamageMin, rule.DamageMax, power)
		differ := 0
		for p := int32(0); p <= 100; p++ {
			b, s := castDamageRecord(rule.DamageMin, rule.DamageMax, p)
			ob := int64(rule.DamageMin) * int64(p+30) / 30
			if ob != b || int64(rule.DamageMax)*int64(p+30)/30-ob != s {
				differ++
			}
		}
		t.Logf("spell %d school %d columns %d/%d unit=%v area=%v: record %d..%d at power %d; integer formula differs at %d of powers 0..100",
			id, rule.School, rule.DamageMin, rule.DamageMax, rule.TargetsUnit, rule.Area, base, base+spread, power, differ)
	}

	for _, id := range []uint16{1, 13} {
		rule := byID[id]
		if !rule.Damaging || !rule.TargetsUnit || rule.Area || rule.School < 1 || rule.School > 5 {
			t.Fatalf("installed spell %d is not a unit-targeted harmful row: %+v", id, rule)
		}
		var protection [5]int32
		protection[rule.School-1] = resistance
		base, spread := castDamageRecord(rule.DamageMin, rule.DamageMax, power)
		lo, hi := castDamageResisted(base, resistance), castDamageResisted(base+spread, resistance)
		seen := map[int64]int{}
		for seed := uint64(1); seed <= 96; seed++ {
			w := world(seed, rule, protection)
			commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(id))}
			landed := int64(0)
			for tick := 0; tick < 256 && landed == 0; tick++ {
				sim.Step(w, commands)
				commands = nil
				landed = int64(1<<19) - int64(entityByID(t, w, 2).HP)
			}
			if landed < lo || landed > hi {
				t.Fatalf("spell %d seed %d landed %d outside the resisted record interval %d..%d", id, seed, landed, lo, hi)
			}
			seen[landed]++
		}
		if seen[lo] == 0 || seen[hi] == 0 {
			t.Fatalf("spell %d: landed amounts %v do not reach both ends %d and %d", id, seen, lo, hi)
		}
		// Loss controls: the former 0..100 clamp kept a negative resistance
		// at zero, so it never landed above the unresisted top; the former
		// integer formula gave Lightning 5/15 at power 20 the top 25, where
		// binary64 gives 15 * 1.6666666666666665 = 24.999999999999996.
		if hi <= base+spread {
			t.Fatalf("spell %d: resisted top %d does not exceed the unresisted %d", id, hi, base+spread)
		}
		oldBase := int64(rule.DamageMin) * (power + 30) / 30
		oldTop := int64(rule.DamageMax) * (power + 30) / 30
		if id == 13 && (oldBase+(oldTop-oldBase) == base+spread || seen[castDamageResisted(oldTop, resistance)] != 0) {
			t.Fatalf("spell 13: the integer top %d is reached or equals the record top %d", oldTop, base+spread)
		}
		t.Logf("spell %d at power %d, %s resistance %d: record %d..%d, landed %d..%d over 96 casts: %v",
			id, power, os.Getenv("AGAINROM_ASSETS"), resistance, base, base+spread, lo, hi, seen)
	}

	// A caster whose skill plus Mind is below 30 casts at power 0, and the
	// spellbook popup states that cast's pair: every landed amount lies in it
	// and both ends are reached (DIV-1882). Loss control: the original's record
	// level 100 states a top no cast reaches.
	mind = 5
	for _, id := range []uint16{1, 13} {
		rule := byID[id]
		w := world(1, rule, [5]int32{})
		c := sim.SpellCharacteristicsFor(w.Rules(), entityByID(t, w, 1), rule)
		if c.Power != 0 {
			t.Fatalf("spell %d: the weak caster's power is %d, want 0", id, c.Power)
		}
		seen := map[int64]int{}
		for seed := uint64(1); seed <= 96; seed++ {
			w := world(seed, rule, [5]int32{})
			commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(id))}
			landed := int64(0)
			for tick := 0; tick < 256 && landed == 0; tick++ {
				sim.Step(w, commands)
				commands = nil
				landed = int64(1<<19) - int64(entityByID(t, w, 2).HP)
			}
			if landed < c.DamageMin || landed > c.DamageMax {
				t.Fatalf("spell %d seed %d: the weak caster landed %d outside the popup's %d-%d", id, seed, landed, c.DamageMin, c.DamageMax)
			}
			seen[landed]++
		}
		if seen[c.DamageMin] == 0 || seen[c.DamageMax] == 0 {
			t.Fatalf("spell %d: the weak caster's amounts %v do not reach the popup's ends %d and %d", id, seen, c.DamageMin, c.DamageMax)
		}
		fb, fs := castDamageRecord(rule.DamageMin, rule.DamageMax, 100)
		if fb+fs <= c.DamageMax {
			t.Fatalf("spell %d: loss control: the record level 100 top %d does not exceed the cast's %d", id, fb+fs, c.DamageMax)
		}
		t.Logf("spell %d, skill plus Mind %d: popup %d-%d, landed %v over 96 casts; record level 100 would state %d-%d",
			id, int32(power)+mind, c.DamageMin, c.DamageMax, seen, fb, fb+fs)
	}
	mind = 30

	stone := byID[20]
	var earth [5]int32
	earth[3] = -40
	w := world(1, stone, earth)
	commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(20))}
	var first uint16
	for tick := 0; tick < 256 && first == 0; tick++ {
		sim.Step(w, commands)
		commands = nil
		for _, e := range w.ActiveEffects() {
			if e.Target == 2 && e.Spell == 20 {
				first = e.Remaining
			}
		}
	}
	d := uint16(int64(math.Pow(1.025, power) * float64(stone.SpellDuration) * 16))
	want := uint16(int32(d) * (100 + 40) / 100)
	t.Logf("Stone Curse at power %d: column %d, word %d, Earth resistance -40 gives %d; first observed %d", power, stone.SpellDuration, d, want, first)
	if first != want {
		t.Fatalf("Stone Curse lifetime %d, want %d", first, want)
	}
	for id, counter := range map[uint16]uint16{
		8:  128, // MAGIC-POISONINPUT-157
		7:  320, // MAGIC-398
		12: 16,  // MAGIC-398
		17: 16,  // MAGIC-398
	} {
		if got := byID[id].EffectDuration; got != counter {
			t.Errorf("spell %d template counter %d, want %d", id, got, counter)
		}
	}
}
