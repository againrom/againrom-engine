package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// tailWorld builds fixtureMap at diff with tail on placement i.
func tailWorld(t *testing.T, diff mapload.Difficulty, i int, tail alm.UnitTail) (sim.Entity, sim.Entity) {
	t.Helper()
	plain, err := mapload.FromALMWith(fixtureMap(), fixtureTable(), diff)
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureMap()
	m.Units[i].Tail = tail
	w, err := mapload.FromALMWith(m, fixtureTable(), diff)
	if err != nil {
		t.Fatal(err)
	}
	return plain.Entities()[i], w.Entities()[i]
}

// TestPlacementTailStoresFollowTheDerivationOnAHuman: the stat stores reach
// the Human's derivation, and the live health stays at the lesser of the two
// maxima; defence, absorption, skills and protections overwrite what the
// derivation gave (UNIT-PLACE-034, HERO-ORDER-014).
func TestPlacementTailStoresFollowTheDerivationOnAHuman(t *testing.T) {
	base, up := tailWorld(t, mapload.DifficultyNormal, 0, alm.UnitTail{Body: 45, Mana: -1})
	if !base.Humanoid || up.MaxHP <= base.MaxHP || up.HP != base.MaxHP || up.NativeBasis.Body != 45 {
		t.Fatalf("Body 45: health %d/%d body %d; template %d/%d: want a higher maximum and the template's live health",
			up.HP, up.MaxHP, up.NativeBasis.Body, base.HP, base.MaxHP)
	}
	// The derivation's first step holds a stat at 50 (HERO-CAP-015).
	if _, capped := tailWorld(t, mapload.DifficultyNormal, 0, alm.UnitTail{Body: 60, Mana: -1}); capped.NativeBasis.Body != 50 || capped.MaxHP <= up.MaxHP {
		t.Fatalf("Body 60: body %d health %d, want 50 and above Body 45's %d", capped.NativeBasis.Body, capped.MaxHP, up.MaxHP)
	}
	_, down := tailWorld(t, mapload.DifficultyNormal, 0, alm.UnitTail{Body: 2, Mana: -1})
	if down.MaxHP >= base.MaxHP || down.HP != down.MaxHP {
		t.Fatalf("Body 2: health %d/%d, template maximum %d: want a lower maximum, the live health clamped to it", down.HP, down.MaxHP, base.MaxHP)
	}
	_, spirit := tailWorld(t, mapload.DifficultyNormal, 0, alm.UnitTail{Spirit: 44, Mana: -1})
	if spirit.Spirit != 44 || spirit.Protection == base.Protection {
		t.Fatalf("Spirit 44: spirit %d protection %v, template %v: the derivation must read the stored Spirit", spirit.Spirit, spirit.Protection, base.Protection)
	}
	tail := alm.UnitTail{Spirit: 44, Mana: 7, HasMana: true, Defence: 33, Absorption: 44,
		Skill: [5]uint8{1, 0, 3, 0, 5}, Protection: [5]uint8{0, 91, 0, 93, 0}}
	_, got := tailWorld(t, mapload.DifficultyNormal, 0, tail)
	want := spirit
	want.Mana, want.Defence, want.Absorption = 7, 33, 44
	want.Skill[1], want.Skill[3], want.Skill[5] = 1, 3, 5
	want.Protection[1], want.Protection[3] = 91, 93
	if tailFields(got) != tailFields(want) || got.SkillXP != base.SkillXP {
		t.Fatalf("stores after the derivation:\n got %+v\nwant %+v (experience %v, template %v)", tailFields(got), tailFields(want), got.SkillXP, base.SkillXP)
	}
}

// TestPlacementTailOverridesTheCreaturesDifficultyBlock: on a creature the
// stores follow the difficulty block, so Hard's defence +50 is replaced and
// its health maximum stands.
func TestPlacementTailOverridesTheCreaturesDifficultyBlock(t *testing.T) {
	tail := alm.UnitTail{Body: 9, Mind: 8, Spirit: 7, Reaction: 6, Mana: 0, HasMana: true, Defence: 12, Absorption: 13,
		Skill: [5]uint8{21, 22, 23, 24, 25}, Protection: [5]uint8{31, 32, 33, 34, 35}}
	base, got := tailWorld(t, mapload.DifficultyHard, 2, tail)
	if base.Humanoid || base.Defence < 50 || base.MaxHP != 148 {
		t.Fatalf("fixture creature at Hard: defence %d health %d, want Hard's +50 and 148", base.Defence, base.MaxHP)
	}
	want := base
	want.NativeBasis = want.NativeBasis.WithBody(9)
	want.Mind, want.Spirit, want.Reaction, want.Mana, want.Defence, want.Absorption = 8, 7, 6, 0, 12, 13
	want.Skill = [6]int32{base.Skill[0], 21, 22, 23, 24, 25}
	want.Protection = [5]int32{31, 32, 33, 34, 35}
	if tailFields(got) != tailFields(want) {
		t.Fatalf("creature:\n got %+v\nwant %+v", tailFields(got), tailFields(want))
	}
}

// TestPlacementTailLoneShootingByteKeepsTheDerivedShooting: a Shooting byte
// that is the record's only skill override leaves the derived Shooting on a
// Human and a creature; beside another skill byte it is stored (DIV-3581).
func TestPlacementTailLoneShootingByteKeepsTheDerivedShooting(t *testing.T) {
	for _, i := range []int{0, 2} {
		base, lone := tailWorld(t, mapload.DifficultyHard, i, alm.UnitTail{Mana: -1, Skill: [5]uint8{0, 0, 0, 0, 3}})
		if tailFields(lone) != tailFields(base) || base.Skill[5] == 3 {
			t.Fatalf("placement %d, lone Shooting 3: %+v, template %+v", i, tailFields(lone), tailFields(base))
		}
		_, pair := tailWorld(t, mapload.DifficultyHard, i, alm.UnitTail{Mana: -1, Skill: [5]uint8{0, 0, 0, 9, 3}})
		want := base
		want.Skill[4], want.Skill[5] = 9, 3
		if tailFields(pair) != tailFields(want) {
			t.Fatalf("placement %d, Pike 9 and Shooting 3: %+v, want %+v", i, tailFields(pair), tailFields(want))
		}
	}
}

// TestPlacementTailReachesAnUnresolvedPlacement: the stores are common to
// every arm.
func TestPlacementTailReachesAnUnresolvedPlacement(t *testing.T) {
	base, got := tailWorld(t, mapload.DifficultyNormal, 1, alm.UnitTail{Mind: 77, Mana: -1, Protection: [5]uint8{0, 0, 0, 0, 5}})
	want := base
	want.Mind, want.Protection[4] = 77, 5
	if tailFields(got) != tailFields(want) {
		t.Fatalf("unresolved:\n got %+v\nwant %+v", tailFields(got), tailFields(want))
	}
}

// TestPlacementWithoutTailSpawnsUnchanged: an absent tail, in either
// spelling of the mana word, changes no actor.
func TestPlacementWithoutTailSpawnsUnchanged(t *testing.T) {
	for i := range fixtureMap().Units {
		base, got := tailWorld(t, mapload.DifficultyHard, i, alm.UnitTail{Mana: -1})
		if tailFields(got) != tailFields(base) {
			t.Fatalf("placement %d moved: %+v vs %+v", i, tailFields(got), tailFields(base))
		}
	}
}

type tailView struct {
	HP, MaxHP, Mana, MaxMana, Mind, Spirit, Reaction, Defence, Absorption int32
	Body                                                                  uint16
	Skill                                                                 [6]int32
	Protection                                                            [5]int32
}

func tailFields(e sim.Entity) tailView {
	return tailView{e.HP, e.MaxHP, e.Mana, e.MaxMana, e.Mind, e.Spirit, e.Reaction, e.Defence, e.Absorption,
		e.NativeBasis.Body, e.Skill, e.Protection}
}
