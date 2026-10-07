package data_test

import (
	"math"
	"testing"

	"againrom/pkg/data"
)

// shortSword is the shipped `Iron Short Sword` as the resolver builds it from
// the installed table: base 5, spread 3, to-hit 5, no defence, blade kind,
// reach 1. It is written out here rather than resolved, because these tests are
// about the FOLD and the resolution has its own file.
//
// Its numbers are the ones the resolver's own tests derive from a synthetic
// table, and the ones the build measures against the lawful install.
var shortSword = data.Weapon{
	Name: "Iron Short Sword", DamageBase: 5, DamageSpread: 3,
	ToHit: 5, Defence: 0, AttackType: data.SkillBlade,
	ChargeTime: 9, RelaxTime: 5, Range: 1,
}

// bladeHero is a hero at the given Body, trained in the blade slot at the
// chargen level, with Reaction at the chargen start.
func bladeHero(body int32) data.Hero {
	h := data.NewCampaignHero(data.SkillBlade)
	h.Body = body
	return h
}

// THE OWNER'S FOUR MEASURED BAND EDGES. Against a character's Strength he read
// 7-10 at 15, 8-12 at 32, 9-14 at 39 and 10-16 at 43, with the sword listed as
// 5-8 — band edges, not samples.
//
// This is the check that DISCRIMINATES rather than fits. Under the retracted
// ladder the sword carries (23, 17) and the same four rows read 25-42, 26-44,
// 27-46 and 28-48, so agreement here is evidence about the ladder and not a
// restatement of it. The band is `[base, base + spread]`, which is what the
// character sheet composes and therefore what he read.
func TestTheFourMeasuredBandEdgesReproduce(t *testing.T) {
	for _, tc := range []struct{ body, lo, hi int32 }{
		{15, 7, 10},
		{32, 8, 12},
		{39, 9, 14},
		{43, 10, 16},
	} {
		c := bladeHero(tc.body).Derive(&shortSword)
		lo, hi := c.DamageBase, c.DamageBase+c.DamageSpread
		if lo != tc.lo || hi != tc.hi {
			t.Errorf("Body %d: band %d-%d, want %d-%d", tc.body, lo, hi, tc.lo, tc.hi)
		}
	}
}

// THE BAND IS CONSTANT BETWEEN THOSE POINTS and steps where the exponential
// crosses the divisor. He stated the band is constant between his readings; the
// step points are this tree's prediction against a run of the same game.
func TestTheBandStepsAtTheStatedBodies(t *testing.T) {
	steps := []int32{}
	prev := int32(-1)
	for body := int32(15); body <= 50; body++ {
		if v := bladeHero(body).Derive(&shortSword).DamageSpread - shortSword.DamageSpread; v != prev {
			steps = append(steps, body)
			prev = v
		}
	}
	want := []int32{15, 32, 39, 43, 46, 49}
	if len(steps) != len(want) {
		t.Fatalf("steps at %v, want %v", steps, want)
	}
	for i := range want {
		if steps[i] != want[i] {
			t.Fatalf("steps at %v, want %v", steps, want)
		}
	}
}

// A BARE HERO IS ZERO BELOW BODY 32, and that is the original's own answer
// rather than a hole in this one — 1.1^Body reaches the divisor first at 32.
func TestABareHeroDoesNothingBelowBodyThirtyTwo(t *testing.T) {
	for _, tc := range []struct{ body, lo, hi int32 }{
		{15, 0, 0},
		{31, 0, 0},
		{32, 1, 2},
		{39, 2, 4},
		{43, 3, 6},
	} {
		c := bladeHero(tc.body).Derive(nil)
		lo, hi := c.DamageBase, c.DamageBase+c.DamageSpread
		if lo != tc.lo || hi != tc.hi {
			t.Errorf("bare at Body %d: band %d-%d, want %d-%d", tc.body, lo, hi, tc.lo, tc.hi)
		}
		if c.AttackChargeTime != data.BareChargeTime || c.AttackRelaxTime != data.BareRelaxTime {
			t.Errorf("bare at Body %d: cadence %d/%d, want %d/%d", tc.body,
				c.AttackChargeTime, c.AttackRelaxTime, data.BareChargeTime, data.BareRelaxTime)
		}
	}
}

// The generation START with the sword. It is no longer what the front end's
// party carries — that hero is generated — and it is pinned anyway, because the
// start is a decoded fact and a later story is entitled to build on it.
func TestTheChargenStartWithItsStartingWeapon(t *testing.T) {
	got := data.NewCampaignHero(data.SkillBlade).Derive(&shortSword)
	want := data.Combat{
		DamageBase: 7, DamageSpread: 3, ToHit: 39, Defence: 8, Absorption: 0,
		AlwaysHits: false, AttackChargeTime: 9, AttackRelaxTime: 5,
		// The sword's own Range: Derive now sets Reach from the weapon it is
		// handed, same as every other number here.
		Reach: shortSword.Range,
		// The sword's own kind: this hero trained in SkillBlade too, but SkillSlot
		// follows the WEAPON in hand, not the training, so the two agreeing here
		// is a fact about the fixture and not about the field.
		SkillSlot: data.SkillBlade,
	}
	if got != want {
		t.Fatalf("Derive = %+v\nwant %+v", got, want)
	}
}

// THE SKILL REACHES THE BASE AND THE TO-HIT AND NOT THE SPREAD. The asymmetry
// is the reason the pair's two ends do not carry the same terms, and a fold
// that added the skill to both would pass every band check at one Body.
func TestTheSkillMovesTheBaseAndTheToHitAndNotTheSpread(t *testing.T) {
	base := data.NewCampaignHero(data.SkillBlade)
	trained := base
	trained.Skill[data.SkillBlade] = 40

	a := base.Derive(&shortSword)
	b := trained.Derive(&shortSword)

	if b.DamageBase-a.DamageBase != 40/5-10/5 {
		t.Errorf("base moved by %d, want %d", b.DamageBase-a.DamageBase, 40/5-10/5)
	}
	if b.ToHit-a.ToHit != 3*(40-10) {
		t.Errorf("toHit moved by %d, want %d", b.ToHit-a.ToHit, 3*(40-10))
	}
	if b.DamageSpread != a.DamageSpread {
		t.Errorf("spread moved from %d to %d — the skill must not reach it",
			a.DamageSpread, b.DamageSpread)
	}
}

// A hero trained in a slot his weapon does not name gets NEITHER skill term: the
// active skill is the weapon's own kind and nothing else.
func TestTheActiveSkillIsTheWeaponsKind(t *testing.T) {
	h := data.NewCampaignHero(data.SkillAxe) // trained in axe...
	got := h.Derive(&shortSword)             // ...holding a blade
	bare := data.Hero{Body: data.ChargenStat, Reaction: data.ChargenStat}.Derive(&shortSword)
	if got != bare {
		t.Fatalf("an axe-trained hero holding a blade derived %+v, want the untrained %+v", got, bare)
	}
}

// MIND AND SPIRIT REACH NONE OF THE EIGHT NUMBERS. Moving either alone, over
// their whole range, moves nothing.
func TestMindAndSpiritReachNothing(t *testing.T) {
	want := data.NewCampaignHero(data.SkillBlade).Derive(&shortSword)
	for v := int32(0); v <= 100; v++ {
		m := data.NewCampaignHero(data.SkillBlade)
		m.Mind = v
		s := data.NewCampaignHero(data.SkillBlade)
		s.Spirit = v
		if got := m.Derive(&shortSword); got != want {
			t.Fatalf("Mind %d moved the derive to %+v", v, got)
		}
		if got := s.Derive(&shortSword); got != want {
			t.Fatalf("Spirit %d moved the derive to %+v", v, got)
		}
	}
}

// THE CAP BINDS AT 50 and is destructive: a stat above it derives as 50 exactly,
// so a to-hit built from Body 80 and one built from Body 50 are the same number.
func TestAStatAboveFiftyDerivesAsFifty(t *testing.T) {
	at50 := data.Hero{Body: 50, Reaction: 50}.Derive(nil)
	for _, v := range []int32{51, 60, 100, 1 << 20} {
		if got := (data.Hero{Body: v, Reaction: v}).Derive(nil); got != at50 {
			t.Errorf("Body/Reaction %d derived %+v, want the capped %+v", v, got, at50)
		}
	}
	// And at the cap itself nothing is clamped away.
	if (data.Hero{Body: 49, Reaction: 49}).Derive(nil) == at50 {
		t.Error("Body 49 and Body 50 derive the same numbers; the cap is binding early")
	}
}

// THE ZERO HERO WITH NO WEAPON derives what the party carried before this
// story: five zeroes, no auto-hit, and the bare cadence. It is produced by
// the same arithmetic every other hero takes, with no branch of its own —
// which is what makes every world this tree already builds unchanged. Reach
// joins them at its own bare floor of 1, the one new field this story adds
// to the graph.
func TestTheZeroHeroDerivesTheNumbersThePartyCarriedBefore(t *testing.T) {
	got := data.Hero{}.Derive(nil)
	want := data.Combat{
		DamageBase: 0, DamageSpread: 0, ToHit: 0, Defence: 0, Absorption: 0,
		AlwaysHits: false, AttackChargeTime: 8, AttackRelaxTime: 4, Reach: 1,
	}
	if got != want {
		t.Fatalf("Derive = %+v\nwant %+v", got, want)
	}
	// Those two are the Units constructor's own cadence, which is what the party
	// took while it resolved to no definition.
	d := data.UnitDefaults()
	if want.AttackChargeTime != d.AttackChargeTime || want.AttackRelaxTime != d.AttackRelaxTime {
		t.Errorf("bare cadence %d/%d against the constructor's %d/%d",
			want.AttackChargeTime, want.AttackRelaxTime, d.AttackChargeTime, d.AttackRelaxTime)
	}
}

// A weapon ASSIGNS the cadence rather than adding to it, so an armed hero's
// cadence has no trace of the bare pair in it.
func TestAWeaponAssignsTheCadence(t *testing.T) {
	got := data.NewCampaignHero(data.SkillBlade).Derive(&shortSword)
	if got.AttackChargeTime != shortSword.ChargeTime || got.AttackRelaxTime != shortSword.RelaxTime {
		t.Fatalf("cadence %d/%d, want the weapon's %d/%d",
			got.AttackChargeTime, got.AttackRelaxTime, shortSword.ChargeTime, shortSword.RelaxTime)
	}
}

// ABSORPTION HAS NO SOURCE BUT ARMOUR, so no hero and no weapon can move it.
func TestAbsorptionIsAlwaysZero(t *testing.T) {
	armed := shortSword
	armed.Defence = 99
	for _, w := range []*data.Weapon{nil, &shortSword, &armed} {
		for _, body := range []int32{0, 25, 50} {
			if got := (data.Hero{Body: body, Reaction: body}).Derive(w); got.Absorption != 0 {
				t.Fatalf("absorption %d at Body %d", got.Absorption, body)
			}
		}
	}
	// A weapon's defence column DOES reach defence, which is what shows the
	// zero above is a fact about absorption and not about the fold.
	plain := data.Hero{Reaction: 30}.Derive(&shortSword)
	withDef := data.Hero{Reaction: 30}.Derive(&armed)
	if withDef.Defence-plain.Defence != 99 {
		t.Fatalf("a weapon's defence moved defence by %d, want 99", withDef.Defence-plain.Defence)
	}
}

// THE TRUNCATION MARGIN, measured rather than assumed. These values are computed
// through a float and then hashed, so a `pow` differing by an ULP from the one
// the original calls could change an integer at a boundary. It cannot here: no
// term stands within 1e-6 of one over the whole reachable stat range.
func TestNoTermOfTheDeriveSitsOnATruncationBoundary(t *testing.T) {
	const floor = 1e-6
	margin := func(v float64) float64 {
		d := v - math.Floor(v)
		return math.Min(d, 1-d)
	}

	worstDamage, atDamage := 1.0, int32(0)
	for body := int32(0); body <= 100; body++ {
		if m := margin(math.Pow(1.1, float64(body)) / 20); m < worstDamage {
			worstDamage, atDamage = m, body
		}
	}
	if worstDamage < floor {
		t.Errorf("damage term stands %g from a boundary at Body %d", worstDamage, atDamage)
	}

	worstToHit := 1.0
	var atBody, atReaction int32
	for body := int32(0); body <= 100; body++ {
		for reaction := int32(0); reaction <= 100; reaction++ {
			v := (math.Pow(1.1, float64(body)) + math.Pow(1.1, float64(reaction))) / 5
			if m := margin(v); m < worstToHit {
				worstToHit, atBody, atReaction = m, body, reaction
			}
		}
	}
	if worstToHit < floor {
		t.Errorf("to-hit term stands %g from a boundary at Body %d, Reaction %d",
			worstToHit, atBody, atReaction)
	}
	t.Logf("worst margins: damage %g (Body %d), to-hit %g (Body %d, Reaction %d)",
		worstDamage, atDamage, worstToHit, atBody, atReaction)
}

// NewCampaignHero writes exactly one slot, and a slot outside 1..5 trains
// nothing — the hero the mage arm and an untrained caller both produce.
func TestNewCampaignHeroWritesExactlyOneSlot(t *testing.T) {
	h := data.NewCampaignHero(data.SkillPike)
	for i, v := range h.Skill {
		want := int32(0)
		if i == data.SkillPike {
			want = data.ChargenSkill
		}
		if v != want {
			t.Errorf("slot %d = %d, want %d", i, v, want)
		}
	}
	if h.Body != 25 || h.Reaction != 25 || h.Mind != 25 || h.Spirit != 25 {
		t.Errorf("stats %d/%d/%d/%d, want four 25s", h.Body, h.Reaction, h.Mind, h.Spirit)
	}
	for _, slot := range []int32{-1, data.SkillGeneral, data.SkillSlots, 99} {
		if got := data.NewCampaignHero(slot); got.Skill != [data.SkillSlots]int32{} {
			t.Errorf("slot %d trained %v, want nothing", slot, got.Skill)
		}
	}
}

func TestAWeaponsEmptyCadenceCellLeavesTheBarePairStanding(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30}
	for _, tc := range []struct {
		name          string
		charge, relax int32
		wantCharge    int32
		wantRelax     int32
	}{
		{"both stated", 9, 5, 9, 5},
		{"charge empty", -1, 5, data.BareChargeTime, 5},
		{"relax empty", 9, -1, 9, data.BareRelaxTime},
		{"both empty", -1, -1, data.BareChargeTime, data.BareRelaxTime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := data.Weapon{Name: "W", AttackType: data.SkillBlade, ChargeTime: tc.charge, RelaxTime: tc.relax}
			c := h.Derive(&w)
			if c.AttackChargeTime != tc.wantCharge || c.AttackRelaxTime != tc.wantRelax {
				t.Errorf("cadence %d/%d, want %d/%d",
					c.AttackChargeTime, c.AttackRelaxTime, tc.wantCharge, tc.wantRelax)
			}
			// The empty value must never reach the field: it is what the
			// simulation's floor turns into the fastest possible attacker.
			if c.AttackChargeTime < 0 || c.AttackRelaxTime < 0 {
				t.Errorf("cadence %d/%d carries the empty cell", c.AttackChargeTime, c.AttackRelaxTime)
			}
		})
	}
	// The stated values must differ from the bare pair, or every case above
	// passes for the wrong reason.
	if data.BareChargeTime == 9 || data.BareRelaxTime == 5 {
		t.Fatal("the fixture's cadence collides with the bare pair")
	}
}

func TestDeriveSetsReachFromItsOwnWeaponArgument(t *testing.T) {
	if got := (data.Hero{Body: 30, Reaction: 30}).Derive(nil).Reach; got != 1 {
		t.Errorf("Derive(nil).Reach = %d, want 1", got)
	}
	bow := data.Weapon{Name: "W", AttackType: data.SkillBlade, Range: 6}
	if got := (data.Hero{Body: 30, Reaction: 30}).Derive(&bow).Reach; got != 6 {
		t.Errorf("Derive(&bow).Reach = %d, want the weapon's own 6", got)
	}
}

func TestCombatReportsTheWieldedWeaponsSkillSlot(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30}
	if got := h.Derive(nil).SkillSlot; got != data.SkillGeneral {
		t.Errorf("bare hero: SkillSlot = %d, want %d (SkillGeneral)", got, data.SkillGeneral)
	}
	if got := h.Derive(&shortSword).SkillSlot; got != data.SkillBlade {
		t.Errorf("blade-armed hero: SkillSlot = %d, want %d (SkillBlade)", got, data.SkillBlade)
	}

	// Trained in an off-hand slot, holding the blade anyway: the credited
	// slot follows the WEAPON, not the training.
	axeTrained := data.NewCampaignHero(data.SkillAxe)
	if got := axeTrained.Derive(&shortSword).SkillSlot; got != data.SkillBlade {
		t.Errorf("axe-trained hero holding a blade: SkillSlot = %d, want %d (SkillBlade)", got, data.SkillBlade)
	}
}

func TestDeriveCarriesTheWeaponsSpellPair(t *testing.T) {
	staff := shortSword
	staff.SpellName, staff.SpellPower = "Fire_Arrow", 10

	h := data.Hero{Body: 30, Reaction: 30}
	got := h.Derive(&staff)
	if got.SpellName != "Fire_Arrow" || got.SpellPower != 10 {
		t.Errorf("Derive spell = (%q, %d), want (\"Fire_Arrow\", 10)", got.SpellName, got.SpellPower)
	}

	// SC-2: a bare hero, and a hero holding a weapon that carries no spell,
	// both derive neither half — nothing about an unrelated hero changes.
	if bare := h.Derive(nil); bare.SpellName != "" || bare.SpellPower != 0 {
		t.Errorf("a bare hero derived a spell: (%q, %d)", bare.SpellName, bare.SpellPower)
	}
	if plain := h.Derive(&shortSword); plain.SpellName != "" || plain.SpellPower != 0 {
		t.Errorf("a spell-less weapon derived a spell: (%q, %d)", plain.SpellName, plain.SpellPower)
	}
}

func TestRewardAgreesWithRecompute(t *testing.T) {
	h := data.Hero{Mind: 37}
	h.Skill[data.SkillBlade] = 40
	h.Skill[data.SkillPike] = 12
	h.Skill[data.SkillGeneral] = 5

	want := h.Recompute(data.Profile{}, data.Loadout{})
	got := h.Reward()

	if got.Mind != want.Mind {
		t.Errorf("Reward.Mind = %d, want %d", got.Mind, want.Mind)
	}
	if got.SkillXP != want.SkillXP {
		t.Errorf("Reward.SkillXP = %v, want %v", got.SkillXP, want.SkillXP)
	}

	// The cap reaches Reward the same way it reaches every other statistic:
	// Reward is not a bypass of step 1.
	over := data.Hero{Mind: 999}
	if got := over.Reward().Mind; got != data.StatCap {
		t.Errorf("Reward.Mind at Mind 999 = %d, want the capped %d", got, data.StatCap)
	}
}

// TestSightSubCellsIsHeroSightWord: HERO-SIGHT-007's word for the owner's
// three tavern mages (mind+reaction 63, 59, 59).
func TestSightSubCellsIsHeroSightWord(t *testing.T) {
	if got := data.SightSubCells(37, 26); got != 1669 {
		t.Fatalf("SightSubCells(37, 26) = %d, want 1669", got)
	}
	if got := data.SightSubCells(39, 20); got != 1628 {
		t.Fatalf("SightSubCells(39, 20) = %d, want 1628", got)
	}
}
