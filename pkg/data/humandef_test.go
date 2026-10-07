package data

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// humanSlots is the streamed width of a Humans row: slots 0 to lastHumanSlot.
const humanSlots = lastHumanSlot + 1

// humanFieldsOf takes a definition apart into its named fields, expanding the
// skill array element by element so a difference names Skill[3] and not merely
// Skill.
//
// It WALKS THE TYPE rather than listing the fields, which is what makes every
// test below cover a field added later without being edited: a field with no
// case in the slot switch shows up here holding its default, and the per-slot
// table is what would then have to say which slot writes it.
func humanFieldsOf(d HumanDef) map[string]any {
	m := make(map[string]any)
	v := reflect.ValueOf(d)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name, fv := t.Field(i).Name, v.Field(i)
		if fv.Kind() == reflect.Array {
			for j := 0; j < fv.Len(); j++ {
				m[fmt.Sprintf("%s[%d]", name, j)] = fv.Index(j).Interface()
			}
			continue
		}
		m[name] = fv.Interface()
	}
	return m
}

// humanDefaults is what an all-empty row must yield, WRITTEN OUT BY HAND rather
// than read back off UnitDefaults.
//
// Deriving it from the value the loader itself reads would assert the loader
// against itself and would go green if the two arms' defaults silently diverged.
var humanDefaults = HumanDef{
	Body: 30, Reaction: 30, Mind: 20, Spirit: 20,
	Health: 30, HealthMax: 30,
	Mana: 0, ManaMax: 0,
	Speed: 10, RotationSpeed: 16, ScanRange: 5,
	AttackChargeTime: 8, AttackRelaxTime: 4,
	TokenSize: 1, MovementType: 1, Face: 1,
	DyingTime: 8,
}

// TestAnAllEmptyHumanRowIsExactlyTheDefaults is AC-2's first half.
//
// Compared FIELD BY FIELD rather than by one equality, so a failure says which
// field the empty cell reached instead of only that the values differ — which is
// the whole diagnostic value when the sentinel test goes missing.
func TestAnAllEmptyHumanRowIsExactlyTheDefaults(t *testing.T) {
	got, err := NewHumanDef("AllEmpty", sentinelRow(humanSlots))
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	want := humanFieldsOf(humanDefaults)
	for name, v := range humanFieldsOf(got) {
		if v != want[name] {
			t.Errorf("%s = %v, want the default %v — an empty cell may only leave a default standing",
				name, v, want[name])
		}
	}
}

// humanDistinctRow gives every slot its own value, 100+slot, so a field holding
// another field's slot is visible as a number and not as a coincidence.
func humanDistinctRow(n int) []int32 {
	p := make([]int32, n)
	for i := range p {
		p[i] = int32(100 + i)
	}
	return p
}

// wantHumanDistinct is what humanDistinctRow must load as, WRITTEN OUT BY HAND.
//
// The numbers that are not 100+slot are the point of the table: the health and
// the mana follow their maxima; the defence is 109 rather than the to-hit-shaped
// value a reader expecting the other collection's order would predict; and the
// six skills run 110 to 115 and so push everything after them along by six.
// wantHumanDistinct also carries KnownSpells at 125, slot 25's — reached even
// though slot 24 between it and DyingTime is consumed and DROPPED, which is what
// the movement type being 122 rather than 123 proves the cursor survives.
//
// Slot 18 was dropped the same way until 0141 gave the gender column a consumer;
// it is now 118, and the run 116-117-118-119 is a straight one.
var wantHumanDistinct = HumanDef{
	Body: 100, Reaction: 101, Mind: 102, Spirit: 103,
	Health: 104, HealthMax: 104,
	Mana: 105, ManaMax: 105,
	Speed: 106, RotationSpeed: 107, ScanRange: 108,
	Defence: 109,
	Skill:   [SkillSlots]int32{110, 111, 112, 113, 114, 115},
	TypeID:  116, Face: 117, Gender: 118,
	AttackChargeTime: 119, AttackRelaxTime: 120,
	TokenSize: 121, MovementType: 122,
	DyingTime:   123,
	KnownSpells: 125,
}

// TestEveryHumanFieldHoldsItsOwnSlot is AC-1.
func TestEveryHumanFieldHoldsItsOwnSlot(t *testing.T) {
	got, err := NewHumanDef("Distinct", humanDistinctRow(humanSlots))
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	want := humanFieldsOf(wantHumanDistinct)
	for name, v := range humanFieldsOf(got) {
		if v != want[name] {
			t.Errorf("%s = %v, want %v", name, v, want[name])
		}
	}
	if got.Health != got.HealthMax {
		t.Errorf("health %d does not follow the maximum %d", got.Health, got.HealthMax)
	}
	if got.Mana != got.ManaMax {
		t.Errorf("mana %d does not follow the maximum %d", got.Mana, got.ManaMax)
	}
}

// TestEachHumanSlotHasExactlyOneOutcome is AC-1's own half of the coverage
// argument: for every streamed slot, a row empty everywhere BUT that slot moves
// exactly the fields the table names and no others.
//
// This is the test a "field i takes column i" loader fails and a distinct-value
// table alone might not: with one cell written, a field that took the wrong slot
// keeps its default and shows up as a field that did NOT move.
func TestEachHumanSlotHasExactlyOneOutcome(t *testing.T) {
	// The whole slot map, written out independently of the loader's switch.
	moved := map[int][]string{
		0: {"Body"}, 1: {"Reaction"}, 2: {"Mind"}, 3: {"Spirit"},
		4: {"Health", "HealthMax"},
		5: {"Mana", "ManaMax"},
		6: {"Speed"}, 7: {"RotationSpeed"}, 8: {"ScanRange"},
		9:  {"Defence"},
		10: {"Skill[0]"}, 11: {"Skill[1]"}, 12: {"Skill[2]"},
		13: {"Skill[3]"}, 14: {"Skill[4]"}, 15: {"Skill[5]"},
		16: {"TypeID"}, 17: {"Face"}, 18: {"Gender"},
		19: {"AttackChargeTime"}, 20: {"AttackRelaxTime"},
		21: {"TokenSize"}, 22: {"MovementType"},
		23: {"DyingTime"},
		24: nil, // read and dropped
		25: {"KnownSpells"},
	}
	if len(moved) != humanSlots {
		t.Fatalf("the table names %d slots, want the %d that are streamed", len(moved), humanSlots)
	}

	base := humanFieldsOf(humanDefaults)
	for slot := 0; slot <= lastHumanSlot; slot++ {
		row := sentinelRow(humanSlots)
		// A value no default holds, so "did not move" cannot pass by equality.
		row[slot] = 7777
		got, err := NewHumanDef(fmt.Sprintf("Slot%d", slot), row)
		if err != nil {
			t.Fatalf("slot %d: NewHumanDef: %v", slot, err)
		}
		want := map[string]bool{}
		for _, f := range moved[slot] {
			want[f] = true
		}
		for name, v := range humanFieldsOf(got) {
			// wantMoved is 7777 typed as v's own type: every field here is
			// int32 except KnownSpells, a uint32, so the marker the "moved"
			// branch compares against must be typed to match or the two
			// interface values can never be equal.
			wantMoved := any(int32(7777))
			if _, ok := v.(uint32); ok {
				wantMoved = uint32(7777)
			}
			switch {
			case want[name] && v != wantMoved:
				t.Errorf("slot %d: %s = %v, want the cell's own 7777", slot, name, v)
			case !want[name] && v != base[name]:
				t.Errorf("slot %d: %s = %v, want the untouched default %v", slot, name, v, base[name])
			}
		}
	}
}

func TestNoLoadedHumanFieldHoldsTheSentinel(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  []int32
	}{
		{"all empty", sentinelRow(humanSlots)},
		{"distinct", humanDistinctRow(humanSlots)},
		{"wide", humanDistinctRow(humanSlots + 2)},
	} {
		got, err := NewHumanDef(tc.name, tc.row)
		if err != nil {
			t.Fatalf("%s: NewHumanDef: %v", tc.name, err)
		}
		for name, v := range humanFieldsOf(got) {
			if v == any(int32(-1)) {
				t.Errorf("%s: %s holds the sentinel", tc.name, name)
			}
		}
	}
}

// TestAHumanRowShorterThanTheStreamIsRefused is AC-3.
func TestAHumanRowShorterThanTheStreamIsRefused(t *testing.T) {
	for _, n := range []int{0, 1, lastHumanSlot} {
		got, err := NewHumanDef("Short", sentinelRow(n))
		if err == nil {
			t.Fatalf("%d cell(s): no error", n)
		}
		if !strings.Contains(err.Error(), "Short") {
			t.Errorf("%d cell(s): %v does not name the entry", n, err)
		}
		if got != (HumanDef{}) {
			t.Errorf("%d cell(s): refusal yielded %+v, want the zero value", n, got)
		}
	}
}

// TestAHumansKnownSpellsIsTheRowsOwnBitmask is AC-5a: the sentinel loads as an
// empty book, a stated mask loads with exactly the bits the cell sets, and the
// shipped ManMage_Staff value decodes to the four spells its row names.
func TestAHumansKnownSpellsIsTheRowsOwnBitmask(t *testing.T) {
	for _, tc := range []struct {
		name string
		cell int32
		want uint32
	}{
		{"the sentinel is an empty book, not every spell", -1, 0},
		{"a stated mask loads verbatim", 42, 42},
		{"the shipped ManMage_Staff mask", 266306, 1<<1 | 1<<6 | 1<<12 | 1<<18},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := sentinelRow(humanSlots)
			row[lastHumanSlot] = tc.cell
			d, err := NewHumanDef(tc.name, row)
			if err != nil {
				t.Fatalf("NewHumanDef: %v", err)
			}
			if d.KnownSpells != tc.want {
				t.Errorf("KnownSpells = %d, want %d", d.KnownSpells, tc.want)
			}
		})
	}
}

// TestAHumansHeroIsItsStatisticsAndItsSkills is the seam Combat derives through:
// the four statistics and the six levels cross, and nothing else does.
func TestAHumansHeroIsItsStatisticsAndItsSkills(t *testing.T) {
	d, err := NewHumanDef("Distinct", humanDistinctRow(humanSlots))
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	want := Hero{Body: 100, Reaction: 101, Mind: 102, Spirit: 103,
		Skill: [SkillSlots]int32{110, 111, 112, 113, 114, 115}}
	if got := d.Hero(); got != want {
		t.Errorf("Hero() = %+v, want %+v", got, want)
	}
}

// humanFor is a definition whose four statistics and six levels are the caller's
// and whose cadence pair is a value no weapon below carries, so a cadence that
// came from the wrong place is visible as a number.
func humanFor(body, reaction int32, skills [SkillSlots]int32, charge, relax int32) HumanDef {
	return HumanDef{Body: body, Reaction: reaction, Mind: 11, Spirit: 12,
		Health: 40, HealthMax: 40, Skill: skills,
		AttackChargeTime: charge, AttackRelaxTime: relax}
}

// TestABarePersonsEightAreTheDerivationAndHisOwnCadence is AC-5 and AC-9's first
// half.
//
// The expected numbers are the DERIVATION'S, taken from the one routine that
// owns them, and only the fields Derive does not itself supply are written
// out — asserting the arithmetic again here would be a second copy of the
// graph, which is the thing this arm exists not to have. Reach joins the
// cadence pair among those: Derive leaves it standing at zero, and Combat is
// what sets it.
func TestABarePersonsEightAreTheDerivationAndHisOwnCadence(t *testing.T) {
	d := humanFor(40, 33, [SkillSlots]int32{}, 21, 6)
	want := d.Hero().Derive(nil)
	// The template's own pair stands in place of the HERO's bare fallback, and
	// the two differ, so this cannot pass by them being equal.
	if want.AttackChargeTime == 21 || want.AttackRelaxTime == 6 {
		t.Fatalf("the fixture's cadence collides with the hero's bare pair (%d/%d)",
			want.AttackChargeTime, want.AttackRelaxTime)
	}
	want.AttackChargeTime, want.AttackRelaxTime = 21, 6
	// A bare person's reach is the constructor's own floor, not the zero Derive
	// leaves standing.
	want.Reach = 1

	if got := d.Combat(nil); got != want {
		t.Errorf("Combat(nil) = %+v, want %+v", got, want)
	}
	if got := d.Combat(nil); got.Absorption != 0 || got.AlwaysHits {
		t.Errorf("absorption %d, always-hits %v; want 0 and false on this arm",
			got.Absorption, got.AlwaysHits)
	}
	if c := d.Combat(nil); c.DamageBase != c.DamageSpread {
		t.Errorf("a bare person's pair is %d/%d, want two equal halves", c.DamageBase, c.DamageSpread)
	}
	if c := d.Combat(nil); c.Defence != 33/3 {
		t.Errorf("defence %d, want a third of Reaction", c.Defence)
	}
}

// TestAnArmedPersonDiffersByExactlyHisWeapon is AC-6 and AC-9's second half.
//
// Every case is asserted as a DIFFERENCE from the bare person, so the assertion
// is about what the weapon contributes and cannot go green on the derivation
// alone.
func TestAnArmedPersonDiffersByExactlyHisWeapon(t *testing.T) {
	skills := [SkillSlots]int32{0, 0, 0, 25, 0, 0}
	d := humanFor(40, 33, skills, 21, 6)
	bare := d.Combat(nil)

	for _, tc := range []struct {
		name          string
		w             Weapon
		wantCharge    int32
		wantRelax     int32
		wantSkillSlot int32
	}{
		{"both cells stated", Weapon{Name: "W", DamageBase: 5, DamageSpread: 3, ToHit: 7,
			Defence: 2, AttackType: SkillBludgen, ChargeTime: 9, RelaxTime: 4}, 9, 4, SkillBludgen},
		{"charge empty", Weapon{Name: "W", DamageBase: 5, DamageSpread: 3, ToHit: 7,
			Defence: 2, AttackType: SkillBludgen, ChargeTime: -1, RelaxTime: 4}, 21, 4, SkillBludgen},
		{"relax empty", Weapon{Name: "W", DamageBase: 5, DamageSpread: 3, ToHit: 7,
			Defence: 2, AttackType: SkillBludgen, ChargeTime: 9, RelaxTime: -1}, 9, 6, SkillBludgen},
		{"both empty", Weapon{Name: "W", DamageBase: 5, DamageSpread: 3, ToHit: 7,
			Defence: 2, AttackType: SkillBludgen, ChargeTime: -1, RelaxTime: -1}, 21, 6, SkillBludgen},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := d.Combat(&tc.w)
			// The skill terms: three times the level onto to-hit and a fifth onto
			// the BASE alone, which is why the two halves of the pair carry
			// different terms.
			skill := skills[tc.wantSkillSlot]
			if want := bare.DamageBase + tc.w.DamageBase + skill/5; got.DamageBase != want {
				t.Errorf("damage base %d, want %d", got.DamageBase, want)
			}
			if want := bare.DamageSpread + tc.w.DamageSpread; got.DamageSpread != want {
				t.Errorf("damage spread %d, want %d — the skill must not reach the spread",
					got.DamageSpread, want)
			}
			if want := bare.ToHit + tc.w.ToHit + 3*skill; got.ToHit != want {
				t.Errorf("to-hit %d, want %d", got.ToHit, want)
			}
			if want := bare.Defence + tc.w.Defence; got.Defence != want {
				t.Errorf("defence %d, want %d", got.Defence, want)
			}
			if got.AttackChargeTime != tc.wantCharge || got.AttackRelaxTime != tc.wantRelax {
				t.Errorf("cadence %d/%d, want %d/%d", got.AttackChargeTime, got.AttackRelaxTime,
					tc.wantCharge, tc.wantRelax)
			}
			if got.Absorption != 0 || got.AlwaysHits {
				t.Errorf("absorption %d, always-hits %v; want 0 and false", got.Absorption, got.AlwaysHits)
			}
		})
	}
}

func TestAPersonsEightAreAFunctionOfTheRowAlone(t *testing.T) {
	d := humanFor(40, 33, [SkillSlots]int32{0, 9, 0, 0, 0, 0}, 21, 6)
	w := Weapon{Name: "W", DamageBase: 5, DamageSpread: 3, ToHit: 7, Defence: 2,
		AttackType: SkillBlade, ChargeTime: 9, RelaxTime: 4}
	beforeD, beforeW := d, w
	first := d.Combat(&w)
	if second := d.Combat(&w); second != first {
		t.Errorf("two calls disagree: %+v vs %+v", first, second)
	}
	if d != beforeD || w != beforeW {
		t.Errorf("the call moved its inputs")
	}
}

func TestABarePersonsReachIsOne(t *testing.T) {
	d := humanFor(40, 33, [SkillSlots]int32{}, 21, 6)
	if got := d.Combat(nil).Reach; got != 1 {
		t.Errorf("Combat(nil).Reach = %d, want 1", got)
	}
}

func TestAnArmedPersonsReachIsHisWeaponsRange(t *testing.T) {
	d := humanFor(40, 33, [SkillSlots]int32{}, 21, 6)
	w := Weapon{Name: "W", AttackType: SkillBludgen, Range: 6}
	if got := d.Combat(&w).Reach; got != 6 {
		t.Errorf("Combat(&w).Reach = %d, want the weapon's own 6", got)
	}
}

// TestAUnitsEightAreItsOwnColumns is T3's other half: the accessor on the other
// definition returns exactly what that type already carries and computes nothing.
func TestAUnitsEightAreItsOwnColumns(t *testing.T) {
	d, err := NewUnitDef("Distinct", distinctRow(38))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	want := Combat{DamageBase: d.DamageBase, DamageSpread: d.DamageSpread,
		ToHit: d.ToHit, Defence: d.Defence, Absorption: d.Absorption,
		AlwaysHits:       d.AlwaysHits,
		AttackChargeTime: d.AttackChargeTime, AttackRelaxTime: d.AttackRelaxTime,
		Reach: d.Reach}
	if got := d.Combat(); got != want {
		t.Errorf("Combat() = %+v, want the definition's own %+v", got, want)
	}
	// And the values are the row's, not zeroes that happen to agree.
	if want.DamageBase == 0 || want.ToHit == 0 || want.AlwaysHits == false {
		t.Fatalf("the fixture carries no distinguishing value: %+v", want)
	}
}
