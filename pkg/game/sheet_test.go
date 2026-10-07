package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The character sheet's two halves reaching the window tier.
//
// It is an internal test because what it is about is the DRIVER's own lookup and
// its push, both unexported — and that is the seam that matters: the eight
// numbers and the character have different lifetimes and arrive on one seam, so
// the thing worth asserting is that they arrive together and for the right unit.

// family converts a recompute's [5]int32 protection or resistance family to
// the panel's own [5]int, matching partyCharacters' own field-by-field
// conversion so a test built from this helper witnesses the same widening
// the production code performs.
func family(v [5]int32) (out [5]int) {
	for i := range out {
		out[i] = int(v[i])
	}
	return out
}

// sheetWorld is a two-entity world with numbers that are all distinct, so a push
// that carried one entity's values under another's id could not pass.
func sheetWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 3, Y: 4, HP: 40, MaxHP: 40, DamageBase: 2, DamageSpread: 3,
			ToHit: 11, Defence: 5, Absorption: 1, AttackCharge: 6, AttackRelax: 2},
		{ID: 1, X: 8, Y: 9, HP: 100, MaxHP: 100, DamageBase: 10, DamageSpread: 6,
			ToHit: 49, Defence: 8, Absorption: 0, AttackCharge: 9, AttackRelax: 5,
			AlwaysHits: true},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

func sheetDriver(t *testing.T, chars map[sim.EntityID]ui.UnitCharacter) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	return newMapWorldWith(sheetWorld(t), nil, nil, nil, chars, nil, worldFixtureViewer(t, m))
}

// EVERY ENTITY'S EIGHT NUMBERS CROSS, and they are that entity's own.
func TestTheEntityPushCarriesEveryUnitsOwnEightNumbers(t *testing.T) {
	draws := sheetDriver(t, nil).entityDraws()
	if len(draws) != 2 {
		t.Fatalf("%d entries pushed, want 2", len(draws))
	}
	want := map[uint32]ui.UnitCombat{
		0: {Known: true, DamageBase: 2, DamageSpread: 3, ToHit: 11, Defence: 5,
			Absorption: 1, AttackCharge: 6, AttackRelax: 2},
		1: {Known: true, DamageBase: 10, DamageSpread: 6, ToHit: 49, Defence: 8,
			Absorption: 0, AttackCharge: 9, AttackRelax: 5, AlwaysHits: true},
	}
	for _, d := range draws {
		if d.Combat != want[d.ID] {
			t.Errorf("entity %d crossed with %+v, want %+v", d.ID, d.Combat, want[d.ID])
		}
	}
}

// AND THE CHARACTER CROSSES BESIDE THEM, for the one entity the load knew one
// for. An id with no entry states none — the tier lookup's own rule — which is
// every unit a map placed.
func TestOnlyTheEntityTheLoadKnowsACharacterForStatesOne(t *testing.T) {
	hero := ui.UnitCharacter{Known: true, Body: 43, Reaction: 26, Mind: 15, Spirit: 15,
		Skills: [ui.PanelSkillSlots]int{9, 9, 9, 9, 9, 9}, Weapon: "Iron Short Sword",
		Experience: 12345}

	// THE OVERLAY REPLACES Skills AND Experience: sheetWorld's entity 1 carries
	// no SkillXP of its own, so both read as their entity-derived zero value on
	// the crossing — deliberately NOT what hero above says, which is what
	// proves the push reads the entity and not the map it was seeded from.
	want := hero
	want.Skills = [ui.PanelSkillSlots]int{}
	want.Experience = 0

	for _, d := range sheetDriver(t, map[sim.EntityID]ui.UnitCharacter{1: hero}).entityDraws() {
		switch d.ID {
		case 1:
			if d.Char != want {
				t.Errorf("the hero crossed with %+v, want %+v", d.Char, want)
			}
		default:
			if d.Char.Known {
				t.Errorf("entity %d states a character the load never knew: %+v", d.ID, d.Char)
			}
		}
	}

	// A DRIVER HOLDING NO LOOKUP AT ALL states none for anyone, which is every
	// map the picker opens: that path places no party, so it resolves no
	// character, and it is the map screen it always was with the eight added.
	for _, d := range sheetDriver(t, nil).entityDraws() {
		if d.Char.Known || d.Char != (ui.UnitCharacter{}) {
			t.Errorf("entity %d states %+v with no lookup at all", d.ID, d.Char)
		}
	}
}

// THE LOOKUP PAIRS THE START'S OWN TWO SLICES and derives neither. The ids come
// from the start, the statistics from the member; getting either from anywhere
// else is how the panel would come to describe the wrong unit.
func TestPartyCharactersPairsTheStartsOwnSlices(t *testing.T) {
	bare := data.NewHero(data.Spread{Body: 30, Reaction: 20, Mind: 18, Spirit: 17}, data.SkillAxe)
	untrained := data.NewHero(data.Spread{Body: 21, Reaction: 22, Mind: 23, Spirit: 24}, data.SkillGeneral)
	sword := &data.Weapon{Name: "Iron Short Sword"}
	pike := &data.Weapon{Name: "Bronze Pike"}

	ms := &Mission{
		Party: []mapload.PartyMember{
			{Hero: PartyHero(), Weapon: sword},
			{Hero: bare},
			{Hero: untrained, Weapon: pike},
		},
		Start: mapload.Start{IDs: []sim.EntityID{7, 8, 9}},
	}
	got := partyCharacters(ms)
	if len(got) != 3 {
		t.Fatalf("%d character(s), want 3", len(got))
	}

	s := PartySpread()
	// THE EXPERIENCE AND THE TWO FAMILIES ARE COMPUTED HERE, off the same Hero
	// and Loadout partyCharacters itself builds for each member, rather than
	// pasted as literals that happen to match today's arithmetic and would rot
	// silently the day it moves.
	d7 := PartyHero().Recompute(data.Profile{}, data.Loadout{Weapon: sword})
	d8 := bare.Recompute(data.Profile{}, data.Loadout{})
	d9 := untrained.Recompute(data.Profile{}, data.Loadout{Weapon: pike})
	// EVERY MEMBER CARRIES THE PERSON BAND: the one field this story adds to
	// partyCharacters' own output, added to every want entry below and to no
	// other field of any of them.
	want := map[sim.EntityID]ui.UnitCharacter{
		7: {Known: true, Band: ui.CharacterBandPerson, Body: int(s.Body), Reaction: int(s.Reaction), Mind: int(s.Mind),
			Spirit: int(s.Spirit), Skills: [ui.PanelSkillSlots]int{0, 10, 0, 0, 0, 0}, Weapon: "Iron Short Sword",
			Experience: int(d7.Experience), Protection: family(d7.Protection), Resistance: family(d7.Resistance),
			Sight: int(d7.Sight)},
		// ALL SIX SLOTS ARE READ OFF THE HERO and none is assumed to be the one
		// this front end generates: an axe-trained member states its level at
		// slot 2 and zero everywhere else. A BARE member names no weapon
		// rather than an empty one.
		8: {Known: true, Band: ui.CharacterBandPerson, Body: 30, Reaction: 20, Mind: 18, Spirit: 17,
			Skills:     [ui.PanelSkillSlots]int{0, 0, 10, 0, 0, 0},
			Experience: int(d8.Experience), Protection: family(d8.Protection), Resistance: family(d8.Resistance),
			Sight: int(d8.Sight)},
		// A member who trained nothing states six zeros, slot 0 included.
		9: {Known: true, Band: ui.CharacterBandPerson, Body: 21, Reaction: 22, Mind: 23, Spirit: 24, Weapon: "Bronze Pike",
			Experience: int(d9.Experience), Protection: family(d9.Protection), Resistance: family(d9.Resistance),
			Sight: int(d9.Sight)},
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("entity %d = %+v\nwant %+v", id, got[id], w)
		}
	}

	// AND ONE INDEPENDENT ARITHMETIC PIN beside the pairing above. A want
	// computed through Recompute moves WITH Recompute, so it witnesses that
	// partyCharacters calls it and nothing about what it answers; a wrong
	// derivation would pass that comparison as readily as a right one. Member
	// 8's Spirit is 17, so each of his five protections is 17/2 = 8; he is
	// trained in one slot at level 10, so his experience is the one slot's
	// ftol((1.1^10 - 1) * 1000) = 1593; and no character re-derives a
	// damage-kind resistance, so all five of those are 0.
	if m := got[8]; m.Protection != ([5]int{8, 8, 8, 8, 8}) || m.Experience != 1593 ||
		m.Resistance != ([5]int{}) {
		t.Errorf("entity 8 states protection %v, experience %d, resistance %v; "+
			"want [8 8 8 8 8], 1593 and five zeroes", m.Protection, m.Experience, m.Resistance)
	}

	// A mission with no party pairs nothing, and so does one whose start
	// recorded no ids — the walk takes the shorter of the two rather than
	// trusting them to be the same length.
	for _, empty := range []*Mission{
		{},
		{Party: []mapload.PartyMember{{Hero: PartyHero()}}},
		{Start: mapload.Start{IDs: []sim.EntityID{7}}},
	} {
		if c := partyCharacters(empty); len(c) != 0 {
			t.Errorf("a mission with %d member(s) and %d id(s) paired %d character(s)",
				len(empty.Party), len(empty.Start.IDs), len(c))
		}
	}
}

func TestPushingTheCharacterMovesNoWorldState(t *testing.T) {
	chars := map[sim.EntityID]ui.UnitCharacter{1: {Known: true, Body: 43}}
	with, without := sheetDriver(t, chars), sheetDriver(t, nil)

	if a, b := with.world.Hash(), without.world.Hash(); a != b {
		t.Errorf("a world pushed with a character hashes %#016x and one without %#016x", a, b)
	}
	before := with.world.Hash()
	with.push()
	with.push()
	if got := with.world.Hash(); got != before {
		t.Errorf("two pushes moved the digest to %#016x from %#016x", got, before)
	}
	if a, b := with.world.Tick(), without.world.Tick(); a != b || a != 0 {
		t.Errorf("ticks %d and %d, want 0 and 0", a, b)
	}
}

// AC-8 pinned the OPPOSITE of what this test now checks — that the number
// the readout states RISES on a tick the subject lands a blow on, off
// SkillXP through data.SkillLevelFor. The fixture is
// TestAGainingAttackerEarnsInExactlyOneSlot's own, from
// pkg/sim/experiencepay_test.go — the same seed, the same attacker (Mind
// 60, credited slot 3, always hits, no spread) and the same target —
// because that combination is already proved to land exactly one blow on a
// single Step; this test adds nothing to that claim except reading the
// result through entityDraws rather than through the entity directly. Relax
// is 50, far past relaxJitter's own top of 3 (pkg/sim/combat.go), so five
// more ticks with no order at all cannot reach a second blow.
func TestTheReadoutsSkillNumberRisesOnABlowAndHoldsWithoutOne(t *testing.T) {
	const attacker, target sim.EntityID = 1, 2
	const creditedSlot = 3
	const seededLevel = 5
	a := sim.Entity{ID: attacker, X: 0, Y: 0, HP: 100, MaxHP: 100, DyingTime: 200,
		AttackCharge: 1, AttackRelax: 50, DamageBase: 10, AlwaysHits: true,
		Owner: 2, GainsXP: true, TypeID: sim.HumanTypeID, Mind: 60, XPSlot: creditedSlot}
	a.Skill[creditedSlot] = seededLevel
	v := sim.Entity{ID: target, X: 1, Y: 0, HP: 100, MaxHP: 100, DyingTime: 200,
		Owner: 3, XPValue: 10}
	w, err := sim.NewWorld(200, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{a, v})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, chars: map[sim.EntityID]ui.UnitCharacter{attacker: {Known: true}}}

	readSkill := func() (level int, total int) {
		for _, d := range mw.entityDraws() {
			if d.ID == uint32(attacker) {
				return d.Char.Skills[creditedSlot], d.Char.Experience
			}
		}
		t.Fatal("the attacker crossed no entry")
		return 0, 0
	}
	// rawXP is the entity's OWN SkillXP for the credited slot, read straight
	// off mw.world rather than through the readout — the independent value
	// readSkill's answer is checked against, so a bug that made both sides
	// of the overlay agree on a wrong number could not hide.
	rawXP := func() int32 {
		for _, e := range mw.world.Entities() {
			if e.ID == attacker {
				return e.SkillXP[creditedSlot]
			}
		}
		t.Fatal("the world holds no attacker")
		return 0
	}

	if level, total := readSkill(); level != seededLevel || total != 0 {
		t.Fatalf("before any blow the readout states level %d, total %d, want %d and 0",
			level, total, seededLevel)
	}

	// TICK 1: the attack order lands its one blow (proved by the sim-level
	// fixture this borrows).
	sim.Step(w, []sim.Command{{Kind: sim.KindAttack, Entity: attacker, X: int32(target)}})
	level1, total1 := readSkill()
	xp := rawXP()
	if xp <= 0 {
		t.Fatalf("after the landed blow the credited slot holds %d, want more than 0", xp)
	}
	if total1 != int(xp) {
		t.Errorf("the readout's total is %d, want the credited slot's own xp %d", total1, xp)
	}
	if level1 != seededLevel {
		t.Errorf("credited slot reads level %d, want the entity's own stored %d, unmoved by the blow",
			level1, seededLevel)
	}

	// TICKS 2..6: no new order, and the attacker is still relaxing — relax 50
	// against a jitter top of 3, so none of these five ticks can reach a
	// second blow. The readout must hold at exactly what tick 1 left it.
	for i := 0; i < 5; i++ {
		sim.Step(w, nil)
		if level, total := readSkill(); level != level1 || total != total1 {
			t.Errorf("tick %d with no blow moved the readout from (%d, %d) to (%d, %d)",
				i+2, level1, total1, level, total)
		}
	}
}
