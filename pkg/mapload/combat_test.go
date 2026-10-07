package mapload_test

import (
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The Units row slots the eight source columns occupy. They are transcribed here
// from the row's own documented order rather than imported, for the reason every
// other slot number in this suite is: a test that asserts a column reached a
// field must not read that column's position out of the code it is testing.
//
// The damage pair is THREE columns and not two — a minimum, a maximum and the
// routing switch that decides which field pair they land in — which is the whole
// reason the eight values below come from nine columns.
const (
	slotDamageMin = 11
	slotDamageMax = 12
	slotDamageArm = 13
	slotToHit     = 14
	slotDefence   = 15
	slotAbsorb    = 16
	slotCharge    = 17
	slotRelax     = 18
)

// The eight source values this file loads, and the six the definition tier's own
// constructor leaves standing.
//
// NO TWO OF THE EIGHT ARE EQUAL and none of them is a constructor default. That
// is what makes a field crossed with another field and a field left unwritten
// two DIFFERENT failures: with the cadence pair at the defaults 8 and 4, a loader
// that never wrote either would pass, and with any two equal, a loader that
// swapped them would.
//
// The damage pair is the derived one: the base is the minimum column and the
// spread is the maximum LESS the minimum, so a loader carrying the maximum
// through unchanged reads 9 where this expects 6.
const (
	srcDamageMin = 3
	srcDamageMax = 9
	srcToHit     = 40
	srcDefence   = 12
	srcAbsorb    = 2
	srcCharge    = 11
	srcRelax     = 7

	wantDamageBase   = 3
	wantDamageSpread = 6

	// The health, the rate and the domain column, set to values of their own so
	// that "this story moved none of the three" is asserted against numbers
	// rather than against zero.
	srcHealthMax = 77
	srcSpeed     = 21
	srcMovement  = 2 // the ghost code

	// hardConstant is what the hard setting adds to the to-hit and to the
	// defence. Written out here because it is the number under test.
	hardConstant = 50
)

// The routing column's arms, by the value that selects each. They are the
// column's own numbering and are spelled here for the same reason the slots are.
const (
	armFirstPair  = 0 // the pair this tree models, no mark
	armSecondPair = 1 // refused
	armThirdPair  = 2 // refused
	armAlwaysHits = 3 // the pair this tree models, and the mark
	armUnknown    = 7 // no arm names it, so the switch's own default carries it
)

// combatRow is a Units row carrying all nine source columns. armSet false leaves
// the routing cell EMPTY, which is a different input from a cell holding zero
// and has to be tested as one: an empty cell is the sentinel the row loader
// skips, and only a selector pre-set to zero makes it mean the first arm rather
// than meaning "-1".
func combatRow(arm int32, armSet bool) []int32 {
	slots := map[int]int32{
		slotUnitType: 0x40, slotUnitFace: 1,
		slotHealthMax: srcHealthMax, slotSpeed: srcSpeed, slotMovement: srcMovement,
		slotDamageMin: srcDamageMin, slotDamageMax: srcDamageMax,
		slotToHit: srcToHit, slotDefence: srcDefence, slotAbsorb: srcAbsorb,
		slotCharge: srcCharge, slotRelax: srcRelax,
	}
	if armSet {
		slots[slotDamageArm] = arm
	}
	return defRow(slots)
}

// combatMap places ONE unit, on a key at the class-key floor's upper side, so
// the placement takes the units arm and the arm is not what any test here
// varies. Its cell is inside the extent.
func combatMap() *alm.Map {
	return &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 0x40, ClassSubID: 1}},
	}
}

// combatTable wraps one row as the units collection index 1, with index 0 the
// reserved empty entry every collection here carries.
func combatTable(name string, row []int32) *mapload.Table {
	return &mapload.Table{
		Units:  defCollection{{}, {name: name, params: row}},
		Humans: defCollection{},
	}
}

// loadOne builds the one-placement world and returns its only entity.
func loadOne(t *testing.T, tbl *mapload.Table, diff mapload.Difficulty) sim.Entity {
	t.Helper()
	w, err := mapload.FromALMWith(combatMap(), tbl, diff)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	return ents[0]
}

// TestAResolvedPlacementCarriesItsClassesEight is AC-1 and SC-1.
//
// EACH FIELD IS ITS OWN ASSERTION, not one struct comparison, so a run names the
// field that is wrong instead of printing two entities and leaving the reader to
// diff them — and so a loader that filled seven of the eight fails with the
// eighth named.
func TestAResolvedPlacementCarriesItsClassesEight(t *testing.T) {
	t.Parallel()

	e := loadOne(t, combatTable("k40", combatRow(armAlwaysHits, true)), mapload.DifficultyNormal)

	for _, c := range []struct {
		field string
		got   int32
		want  int32
	}{
		{"AttackCharge", e.AttackCharge, srcCharge},
		{"AttackRelax", e.AttackRelax, srcRelax},
		{"ToHit", e.ToHit, srcToHit},
		{"Defence", e.Defence, srcDefence},
		{"Absorption", e.Absorption, srcAbsorb},
		{"DamageBase", e.DamageBase, wantDamageBase},
		{"DamageSpread", e.DamageSpread, wantDamageSpread},
	} {
		if c.got != c.want {
			t.Errorf("%s is %d, want %d", c.field, c.got, c.want)
		}
	}
	if !e.AlwaysHits {
		t.Error("AlwaysHits is false and this row's routing column selects the arm that sets it")
	}

	// The three this story must NOT have moved, off the same resolution.
	if e.HP != srcHealthMax || e.MaxHP != srcHealthMax {
		t.Errorf("the health pair is %d/%d, want %d/%d", e.HP, e.MaxHP, srcHealthMax, srcHealthMax)
	}
	if e.Speed != srcSpeed {
		t.Errorf("the rate is %d, want %d", e.Speed, srcSpeed)
	}
	if e.Domain != sim.DomainGhost {
		t.Errorf("the domain is %v, want the ghost domain this row's column names", e.Domain)
	}
}

// TestTheDamagePairAndTheMarkComeOffOneColumnGroup is AC-2 and SC-2.
//
// Every admitted value of the routing column gets a row, INCLUDING the empty
// cell and a value no arm names, and the mark is asserted set on exactly one of
// them — a mark set on two arms and a mark set on none are both failures here.
func TestTheDamagePairAndTheMarkComeOffOneColumnGroup(t *testing.T) {
	t.Parallel()

	marked := 0
	for _, tc := range []struct {
		name   string
		arm    int32
		armSet bool
		mark   bool
	}{
		{"empty cell", 0, false, false},
		{"first pair", armFirstPair, true, false},
		{"always hits", armAlwaysHits, true, true},
		{"no arm names it", armUnknown, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := loadOne(t, combatTable("k40", combatRow(tc.arm, tc.armSet)), mapload.DifficultyNormal)

			if e.DamageBase != wantDamageBase {
				t.Errorf("the damage base is %d, want the minimum column %d", e.DamageBase, wantDamageBase)
			}
			if e.DamageSpread != wantDamageSpread {
				t.Errorf("the damage spread is %d, want the maximum less the minimum, %d",
					e.DamageSpread, wantDamageSpread)
			}
			if e.AlwaysHits != tc.mark {
				t.Errorf("AlwaysHits is %v, want %v", e.AlwaysHits, tc.mark)
			}
		})
		if tc.mark {
			marked++
		}
	}
	if marked != 1 {
		t.Fatalf("%d of the rows above expect the mark; the contract sets it on exactly one", marked)
	}
}

// eight is the eight numbers under test, lifted off an entity so two of them can
// be compared as a value. It carries NOTHING ELSE: a comparison that also caught
// the id or the cell would fail for reasons this contract says nothing about.
type eight struct {
	charge, relax      int32
	toHit, defence     int32
	absorb             int32
	dmgBase, dmgSpread int32
	alwaysHits         bool
}

func eightOf(e sim.Entity) eight {
	return eight{
		charge: e.AttackCharge, relax: e.AttackRelax,
		toHit: e.ToHit, defence: e.Defence, absorb: e.Absorption,
		dmgBase: e.DamageBase, dmgSpread: e.DamageSpread, alwaysHits: e.AlwaysHits,
	}
}

// TestTheDifficultyReachesTheToHitAndTheDefenceAndNothingElse is AC-3, AC-4 and
// SC-3.
//
// It compares TWO BUILT WORLDS rather than asserting arithmetic on a definition.
// A test that added 50 to a column and checked the sum would pass by
// construction however the adjustment was wired; comparing what the loader
// actually produced at two settings fails if the adjustment ever reaches a
// number this contract fences off.
//
// THE SIX THAT MUST NOT MOVE ARE ASSERTED, not merely the two that must. That is
// the whole content of the requirement: an adjustment that also scaled the
// damage would satisfy every "hard is stronger" test ever written.
func TestTheDifficultyReachesTheToHitAndTheDefenceAndNothingElse(t *testing.T) {
	t.Parallel()

	tbl := func() *mapload.Table { return combatTable("k40", combatRow(armAlwaysHits, true)) }
	normal := eightOf(loadOne(t, tbl(), mapload.DifficultyNormal))

	for _, tc := range []struct {
		name  string
		diff  mapload.Difficulty
		delta int32
	}{
		{"hard", mapload.DifficultyHard, hardConstant},
		{"easy", mapload.DifficultyEasy, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := eightOf(loadOne(t, tbl(), tc.diff))

			if got.toHit != normal.toHit+tc.delta {
				t.Errorf("the to-hit is %d and normal's is %d — a difference of %d, want %d",
					got.toHit, normal.toHit, got.toHit-normal.toHit, tc.delta)
			}
			if got.defence != normal.defence+tc.delta {
				t.Errorf("the defence is %d and normal's is %d — a difference of %d, want %d",
					got.defence, normal.defence, got.defence-normal.defence, tc.delta)
			}

			// The other six, one at a time, so the failure names the number the
			// setting reached and should not have.
			for _, c := range []struct {
				field    string
				got, was int32
			}{
				{"AttackCharge", got.charge, normal.charge},
				{"AttackRelax", got.relax, normal.relax},
				{"Absorption", got.absorb, normal.absorb},
				{"DamageBase", got.dmgBase, normal.dmgBase},
				{"DamageSpread", got.dmgSpread, normal.dmgSpread},
			} {
				if c.got != c.was {
					t.Errorf("%s is %d at this setting and %d at normal — the setting must not reach it",
						c.field, c.got, c.was)
				}
			}
			if got.alwaysHits != normal.alwaysHits {
				t.Errorf("AlwaysHits is %v at this setting and %v at normal — the setting must not reach it",
					got.alwaysHits, normal.alwaysHits)
			}
		})
	}
}

// unresolvedShapes is the three ways a placement resolves to no unit definition,
// each on its own map so the three can be loaded independently and compared.
//
// They are DIFFERENT SHAPES and not three spellings of one: no table at all is a
// nil argument, the second is a key below the class-key floor whose type id
// names nothing in a humans collection that EXISTS, and the third is a key above
// the floor that names nothing in a table that is otherwise full.
//
// The second shape USED TO BE a humans-arm placement that reached an entry, and
// that is exactly what 0087 removed from this population: such a placement now
// carries its row's own numbers, so it is no longer a shape that resolves to
// nothing. The question this list asks is unchanged and is re-aimed rather than
// weakened — the shape kept here is a humans-band placement that reaches NO
// entry, which is still one of the ways a placement can end up with the
// constructor's eight, and it is a stronger member than the old one because the
// table it is given is populated and the search genuinely fails.
func unresolvedShapes() []struct {
	name string
	m    *alm.Map
	t    *mapload.Table
} {
	full := combatTable("k40", combatRow(armAlwaysHits, true))
	humansTable := &mapload.Table{
		Units:  full.Units,
		Humans: defCollection{{}, {name: "h9", params: defRow(map[int]int32{slotHumanType: 9})}},
	}
	at := func(class int16) *alm.Map {
		return &alm.Map{Width: 40, Height: 40,
			Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: class, ClassSubID: 1}}}
	}
	return []struct {
		name string
		m    *alm.Map
		t    *mapload.Table
	}{
		{"no table at all", combatMap(), nil},
		{"a humans-arm placement matching no entry", at(7), humansTable},
		{"a key naming nothing", at(0x41), full},
	}
}

// TestAnUnresolvedPlacementCarriesTheConstructorsEight is AC-5 and SC-4.
//
// The three shapes are asserted equal TO EACH OTHER as well as to the expected
// pair, which is the half that matters: a change that moved all three together
// would satisfy every "it carries 8 and 4" assertion and would still have split
// the population if it had reached only two of them.
func TestAnUnresolvedPlacementCarriesTheConstructorsEight(t *testing.T) {
	t.Parallel()

	want := eight{charge: 8, relax: 4}
	var first *eight
	var firstName string

	for _, tc := range unresolvedShapes() {
		w, err := mapload.FromALMWith(tc.m, tc.t, mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("%s: FromALMWith: %v", tc.name, err)
		}
		got := eightOf(w.Entities()[0])
		if got != want {
			t.Errorf("%s carries %+v, want %+v — a charge of 8, a relax of 4 and six zeros",
				tc.name, got, want)
		}
		if first == nil {
			g := got
			first, firstName = &g, tc.name
			continue
		}
		if got != *first {
			t.Errorf("%s carries %+v and %s carries %+v — the two populations that resolve to "+
				"nothing have come to differ", tc.name, got, firstName, *first)
		}
	}
	if first == nil {
		t.Fatal("no unresolved shape was loaded at all")
	}
}

func TestAnUnmodelledDamageArmIsRefusedByName(t *testing.T) {
	t.Parallel()

	for _, arm := range []int32{armSecondPair, armThirdPair} {
		w, err := mapload.FromALMWith(combatMap(),
			combatTable("k40-refused", combatRow(arm, true)), mapload.DifficultyNormal)

		if err == nil {
			t.Errorf("arm %d: a world was built; the entry names a field pair this tree does not model", arm)
			continue
		}
		if w != nil {
			t.Errorf("arm %d: a world was returned beside the error, holding %d entities",
				arm, len(w.Entities()))
		}
		if !strings.Contains(err.Error(), "k40-refused") {
			t.Errorf("arm %d: the refusal is %q and does not name the entry", arm, err)
		}
	}
}

func TestTicksDoNotMoveTheEight(t *testing.T) {
	t.Parallel()

	w, err := mapload.FromALMWith(combatMap(),
		combatTable("k40", combatRow(armAlwaysHits, true)), mapload.DifficultyHard)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	atLoad := eightOf(w.Entities()[0])

	// A walking order as well as bare ticks, so the entity is one a tick is
	// actually doing something to rather than one it steps over.
	sim.Step(w, []sim.Command{{Entity: 0, X: 20, Y: 20}})
	for i := 0; i < 24; i++ {
		sim.Step(w, nil)
	}
	if w.Tick() == 0 {
		t.Fatal("the world stands at tick 0, so nothing was advanced and this asserts nothing")
	}
	if got := eightOf(w.Entities()[0]); got != atLoad {
		t.Errorf("after %d tick(s) the eight are %+v, want the %+v the load wrote", w.Tick(), got, atLoad)
	}
}
