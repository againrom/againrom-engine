package sim

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func prismaticRaysWorld(t *testing.T, rule SpellRule, foes int) *World {
	t.Helper()
	caster := effectMage(1, 1, 1, 1<<14)
	caster.Skill[1] = 100
	rel := engRel(t, [3]uint32{SelfSlot, prismaticFoe, relationHostile})
	ents := []Entity{caster, prismaticFoeAt(2, 14, 1)}
	for id := EntityID(3); id < EntityID(2+foes); id++ {
		ents = append(ents, prismaticFoeAt(id, int32(id)-1, 1))
	}
	return hlWorld(t, 0x5e1ec9, rel, []SpellRule{rule}, ents...)
}

func TestRayLimitIsTheOneRule(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		power int32
		want  int
	}{{-100, 0}, {0, 2}, {20, 3}, {99, 6}, {100, 7}, {1000, 7}} {
		if got := (SpellRule{}).RayLimit(c.power); got != c.want {
			t.Errorf("power %d: %d rays, want %d", c.power, got, c.want)
		}
	}
	if got := (SpellRule{Rays: 100}).RayLimit(0); got != 100 {
		t.Errorf("a cap of 100 gave %d", got)
	}
	for _, power := range []int32{0, 60, 100} {
		got, ok := weaponSpellCharacteristics(Rules{}, SpellRule{ID: 14, Damaging: true, TargetsUnit: true, Rays: 50}, power)
		if !ok || got.RayCount != 50 {
			t.Errorf("tooltip at power %d: %+v %v, want 50 rays", power, got, ok)
		}
		plain, _ := weaponSpellCharacteristics(Rules{}, SpellRule{ID: 14, Damaging: true, TargetsUnit: true}, power)
		if int(plain.RayCount) != (SpellRule{}).RayLimit(power) {
			t.Errorf("unmodded tooltip at power %d: %d", power, plain.RayCount)
		}
	}
}

func TestRayCapAboveTenSelectsEveryCandidateInScoreOrder(t *testing.T) {
	t.Parallel()

	rule := prismaticTestRule
	rule.Rays = 100
	w := prismaticRaysWorld(t, rule, 13)
	cells, ids := prismaticCast(t, w, 2)
	want := []EntityID{2}
	for id := EntityID(3); id <= 14; id++ {
		want = append(want, id)
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("victims %v, want the primary then all 12 other foes in distance order %v", ids, want)
	}
	if len(cells) != len(ids) {
		t.Fatalf("%d rays for %d victims", len(cells), len(ids))
	}
}

func TestRayCapIsAMaximumAndNeverRepeatsAVictim(t *testing.T) {
	t.Parallel()

	rule := prismaticTestRule
	rule.Rays = 100
	w := prismaticRaysWorld(t, rule, 3)
	cells, ids := prismaticCast(t, w, 2)
	if !reflect.DeepEqual(ids, []EntityID{2, 3, 4}) || len(cells) != 3 {
		t.Fatalf("victims %v rays %d, want the 3 foes once each", ids, len(cells))
	}
	rule.Rays = 5
	w = prismaticRaysWorld(t, rule, 13)
	if _, ids := prismaticCast(t, w, 2); len(ids) != 5 || ids[0] != 2 {
		t.Fatalf("a cap of 5 gave victims %v", ids)
	}
}

func TestRayCapIgnoresPowerAndTheOriginalStopsAtSeven(t *testing.T) {
	t.Parallel()

	rule := prismaticTestRule
	w := prismaticRaysWorld(t, rule, 13)
	if _, ids := prismaticCast(t, w, 2); len(ids) != 7 {
		t.Fatalf("the unmodded spell at power 100 chose %d victims, want 7", len(ids))
	}
	rule.Rays = 12
	w = prismaticRaysWorld(t, rule, 13)
	w.entities[0].Skill[1] = 0
	if _, ids := prismaticCast(t, w, 2); len(ids) != 12 {
		t.Fatalf("a cap of 12 at power 0 chose %d victims, want 12", len(ids))
	}
}

func TestRayCapRoundTripsTheByteFormAndDeliveryJSON(t *testing.T) {
	t.Parallel()

	rule := prismaticTestRule
	rule.Rays = 100
	w := hlWorld(t, 3, Relations{}, []SpellRule{rule}, spEnt(1, 1, 1))
	back := requireSpellGraphBinary(t, w)
	if !slices.Equal(back.Spells(), w.Spells()) || back.Spells()[0].Rays != 100 {
		t.Fatalf("cap lost: %+v", back.Spells())
	}
	plain := hlWorld(t, 3, Relations{}, []SpellRule{prismaticTestRule}, spEnt(1, 1, 1))
	if slices.Equal(hlBytes(t, plain), hlBytes(t, w)) {
		t.Error("a world with a cap encodes as the world without")
	}
	if got, _ := json.Marshal(prismaticTestRule); strings.Contains(string(got), "Rays") {
		t.Errorf("an uncapped rule writes Rays: %s", got)
	}
	set, _ := json.Marshal(rule)
	var again SpellRule
	if err := json.Unmarshal(set, &again); err != nil || again.Rays != 100 {
		t.Errorf("delivery rule %s -> %+v %v", set, again, err)
	}

	for name, bad := range map[string]SpellRule{
		"cap on another row": {ID: 1, Damaging: true, DamageMax: 4, Rays: 3},
		"cap above the max":  {ID: 14, Damaging: true, DamageMax: 4, Rays: 101},
	} {
		if _, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical,
			Terrain{}, []Entity{spEnt(1, 1, 1)}, nil, Relations{}, nil, nil, []SpellRule{bad}); err == nil {
			t.Errorf("%s: the table was accepted", name)
		}
	}

	record := func(id uint16, flags, radius uint8) []byte {
		b := make([]byte, spellCountLen+spellRecordLen)
		b[0] = 1
		b[spellCountLen] = byte(id)
		b[spellCountLen+16] = flags
		b[spellCountLen+22] = radius
		return b
	}
	damaging := spellFlagDamaging | spellFlagTargetsUnit
	for name, c := range map[string]struct {
		data []byte
		ok   bool
	}{
		"cap form":           {record(14, damaging|spellFlagAreaHitsHi, 100), true},
		"zero cap":           {record(14, damaging|spellFlagAreaHitsHi, 0), false},
		"cap above the max":  {record(14, damaging|spellFlagAreaHitsHi, 101), false},
		"cap on another row": {record(1, damaging|spellFlagAreaHitsHi, 3), false},
		"low bit on the row": {record(14, damaging|spellFlagAreaHitsLo, 3), false},
	} {
		if _, _, err := decodeSpells(c.data); (err == nil) != c.ok {
			t.Errorf("%s: err %v, want ok=%v", name, err, c.ok)
		}
	}
}
