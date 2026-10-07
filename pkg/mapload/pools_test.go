package mapload_test

// A hero's statistics reaching his health and his mana, at the mission-start
// seam (0119-chargen T4, AC-8 to AC-10).
//
// THE OWNER'S OWN DEFECT IS HERE: a party member was minted at the spawn
// constant whatever his Body said, because nothing between the recompute and
// the entity carried a profile for it to read. These tests fix it at the seam
// the player reaches, and the last one witnesses the fix by reverting it.
//
// Every fixture is built in test code, the same as every other file in this
// package — no game install is read.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// TestAPartyMemberWithAHealthColumnMovesWithBody is AC-8: an entity's health
// maximum equals the recompute's OWN answer for that member's profile and
// weapon, and two members differing only in Body get different maxima — the
// pair is not a second, disagreeing copy of the number and it is not a
// constant wearing the recompute's clothes.
func TestAPartyMemberWithAHealthColumnMovesWithBody(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	profile := data.Profile{HealthColumn: true}
	low := mapload.PartyMember{Class: 100, Hero: data.Hero{Body: 25}, Profile: profile}
	high := mapload.PartyMember{Class: 101, Hero: data.Hero{Body: 45}, Profile: profile}

	w, _ := mustStart(t, m, []mapload.PartyMember{low, high})
	ents := w.Entities()
	if len(ents) != 2 {
		t.Fatalf("%d entit(y/ies), want 2 party members and no map placement", len(ents))
	}

	// THE EXPECTATION IS THE RECOMPUTE'S OWN, called here rather than
	// asserted as a literal — a change to the graph moves this test's want
	// alongside the code under test rather than falling out of step with it.
	wantLow := low.Hero.Recompute(low.Profile, data.Loadout{Weapon: low.Weapon}).HealthMax
	wantHigh := high.Hero.Recompute(high.Profile, data.Loadout{Weapon: high.Weapon}).HealthMax

	if ents[0].HP != wantLow || ents[0].MaxHP != wantLow {
		t.Errorf("Body 25's health pair is %d/%d, want the recompute's own %d on both",
			ents[0].HP, ents[0].MaxHP, wantLow)
	}
	if ents[1].HP != wantHigh || ents[1].MaxHP != wantHigh {
		t.Errorf("Body 45's health pair is %d/%d, want the recompute's own %d on both",
			ents[1].HP, ents[1].MaxHP, wantHigh)
	}
	if wantLow == wantHigh {
		t.Fatalf("Body 25 and Body 45 both recompute to %d — this fixture cannot show "+
			"the maximum moving with Body", wantLow)
	}

	// A CONCRETE PAIR, so a reader sees a real value and not a tautology of
	// "the test agrees with the function it calls": at Body 25, no skill
	// trained, no weapon, the recompute's own health maximum is 27; at Body
	// 45 it is 77. If the graph ever moves these two numbers, this is the
	// line that says so and asks for a deliberate update.
	if wantLow != 27 || wantHigh != 77 {
		t.Fatalf("the fixture's own figures moved: Body 25/45 now recompute to %d/%d, "+
			"want 27/77 — update this comment and the literals above before trusting "+
			"the rest of this test", wantLow, wantHigh)
	}
}

// TestAPartyMemberWithAManaColumnHasAPositiveManaPair is AC-10: a profile
// carrying the mana column gets a positive mana pair off the same recompute,
// and his health — no health column on this profile — stays at the spawn
// constant, which is what says the two pools are wired independently rather
// than one gate covering both.
func TestAPartyMemberWithAManaColumnHasAPositiveManaPair(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	member := mapload.PartyMember{
		Class: 100, Hero: data.Hero{Spirit: 25}, Profile: data.Profile{ManaColumn: true},
	}
	w, _ := mustStart(t, m, []mapload.PartyMember{member})
	got := lastEntity(t, w)

	want := member.Hero.Recompute(member.Profile, data.Loadout{Weapon: member.Weapon}).ManaMax
	if want <= 0 {
		t.Fatalf("the fixture's own recompute answered ManaMax %d, want a positive figure "+
			"— this test would pass vacuously against a member wired to zero", want)
	}
	if got.Mana != want || got.MaxMana != want {
		t.Errorf("mana pair %d/%d, want the recompute's own %d on both", got.Mana, got.MaxMana, want)
	}
	// A CONCRETE NUMBER, for the same reason the health test states one:
	// Spirit 25, the mana column present, no class flag, recomputes to 55.
	if want != 55 {
		t.Fatalf("the fixture's own figure moved: Spirit 25 now recomputes to %d, want 55 "+
			"— update this comment before trusting the rest of this test", want)
	}

	if got.HP != mapload.SpawnHP || got.MaxHP != mapload.SpawnHP {
		t.Errorf("health pair %d/%d, want the spawn constant %d on both — this profile "+
			"carries no health column, so nothing derives one", got.HP, got.MaxHP, mapload.SpawnHP)
	}
}

func TestAZeroProfileMemberIsByteForByteTodaysMember(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	hero := data.NewCampaignHero(data.SkillBlade) // trained; see doc above
	w, _ := mustStart(t, m, []mapload.PartyMember{
		{Class: 100, Hero: hero, Weapon: &heroSword}, // Profile left at its zero value
	})

	got := lastEntity(t, w)
	wantCombat := hero.Derive(&heroSword)
	wantSpeed := hero.Speed()
	wantSight := hero.Sight()

	if got.DamageBase != wantCombat.DamageBase || got.DamageSpread != wantCombat.DamageSpread ||
		got.ToHit != wantCombat.ToHit || got.Defence != wantCombat.Defence ||
		got.Absorption != wantCombat.Absorption || got.AlwaysHits != wantCombat.AlwaysHits ||
		got.AttackCharge != wantCombat.AttackChargeTime || got.AttackRelax != wantCombat.AttackRelaxTime {
		t.Errorf("combat block is %d/%d dmg, toHit %d, defence %d, absorption %d, "+
			"alwaysHits %v, cadence %d/%d — want Derive's own %+v",
			got.DamageBase, got.DamageSpread, got.ToHit, got.Defence, got.Absorption,
			got.AlwaysHits, got.AttackCharge, got.AttackRelax, wantCombat)
	}
	if got.Reach != uint8(wantCombat.Reach) {
		t.Errorf("reach %d, want Derive's own %d", got.Reach, wantCombat.Reach)
	}
	if got.Speed != wantSpeed {
		t.Errorf("speed %d, want Speed()'s own %d", got.Speed, wantSpeed)
	}
	if got.ScanRange != uint8(wantSight) {
		t.Errorf("sight %d, want Sight()'s own %d", got.ScanRange, wantSight)
	}
	if got.HP != mapload.SpawnHP || got.MaxHP != mapload.SpawnHP {
		t.Errorf("health pair %d/%d, want the spawn constant %d on both — the zero profile "+
			"states no health column, and the column is the gate, not whatever the recompute's "+
			"own maximum happens to be", got.HP, got.MaxHP, mapload.SpawnHP)
	}
	if got.Mana != 0 || got.MaxMana != 0 {
		t.Errorf("mana pair %d/%d, want 0/0 — the zero profile carries no mana column",
			got.Mana, got.MaxMana)
	}
	// And the combat block is not vacuously equal on all zeroes: this hero is
	// trained and armed, so a real number is what agrees here.
	if wantCombat.DamageBase == 0 && wantCombat.DamageSpread == 0 {
		t.Error("the oracle itself swings for nothing — this comparison would hold for " +
			"a member that carried nothing either")
	}
}

func TestAZeroProfileTrainedHeroDoesNotHealFromExperience(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	hero := data.NewCampaignHero(data.SkillBlade)

	// THE POSITIVE HALF OF THE ASYMMETRY: the same hero's own zero-profile
	// recompute, called directly rather than through a start, so this test
	// does not depend on the mint loop to demonstrate what the graph alone
	// already does.
	wantHealthMax := hero.Recompute(data.Profile{}, data.Loadout{Weapon: &heroSword}).HealthMax
	if wantHealthMax <= 0 {
		t.Fatalf("the fixture's own zero-profile recompute answered HealthMax %d, want a "+
			"positive figure — this test cannot show the asymmetry without one", wantHealthMax)
	}
	if wantHealthMax != 29 {
		t.Fatalf("the fixture's own figure moved: a trained hero's zero-profile HealthMax is "+
			"now %d, want 29 — update this comment before trusting the rest of this test",
			wantHealthMax)
	}

	// THE OTHER HALF: the start path does not use that positive number.
	w, _ := mustStart(t, m, []mapload.PartyMember{
		{Class: 100, Hero: hero, Weapon: &heroSword}, // Profile left at its zero value
	})
	got := lastEntity(t, w)
	if got.HP != mapload.SpawnHP || got.MaxHP != mapload.SpawnHP {
		t.Errorf("health pair %d/%d, want the spawn constant %d on both even though this "+
			"hero's own zero-profile recompute derives a positive HealthMax of %d — the "+
			"profile's column is the gate, not the recompute's positive maximum",
			got.HP, got.MaxHP, mapload.SpawnHP, wantHealthMax)
	}
}
