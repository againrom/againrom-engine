package data

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// sentinelRow is a row of empty cells: every slot holds the sentinel, so a
// definition built from it must be the constructor's defaults and nothing else.
func sentinelRow(n int) []int32 {
	p := make([]int32, n)
	for i := range p {
		p[i] = -1
	}
	return p
}

// fieldsOf takes a definition apart into its named fields, expanding the two
// fixed arrays element by element so a difference names Protection[2] and not
// merely Protection.
//
// It walks the type rather than listing the fields, which is what makes the
// tests below cover a field added later without being edited: a new field with
// no case in the slot switch shows up here holding its default, and the
// per-slot outcome table is what would then have to say which slot writes it.
func fieldsOf(d UnitDef) map[string]any {
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

// changedFields names every field of got that differs from want.
func changedFields(want, got UnitDef) []string {
	a, b := fieldsOf(want), fieldsOf(got)
	var out []string
	for k, av := range a {
		if b[k] != av {
			out = append(out, k)
		}
	}
	return out
}

// TestAllSentinelRowIsExactlyTheDefaults is AC-3's first half and SC-2's: a row
// of empty cells leaves every default standing.
//
// It is compared FIELD BY FIELD rather than by one equality, so a failure says
// which field the sentinel reached instead of only that the values differ —
// which is the whole diagnostic value when the -1 test goes missing.
func TestAllSentinelRowIsExactlyTheDefaults(t *testing.T) {
	got, err := NewUnitDef("AllEmpty", sentinelRow(41))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	want := fieldsOf(UnitDefaults())
	for name, v := range fieldsOf(got) {
		if v != want[name] {
			t.Errorf("%s = %v, want the default %v — a sentinel cell may only leave a default standing", name, v, want[name])
		}
	}
}

// distinctRow gives every slot its own value: 100+slot everywhere but the damage
// selector, which takes the one modelled arm that does something (3) and is
// distinct from every other cell by being below 100.
func distinctRow(n int) []int32 {
	p := make([]int32, n)
	for i := range p {
		p[i] = int32(100 + i)
	}
	p[13] = 3
	return p
}

// wantDistinct is what distinctRow must load as, WRITTEN OUT BY HAND.
//
// Computing it from the row would assert the streamer against itself. The three
// numbers that are not 100+slot are the point of the table: the spread is
// slot 12 minus slot 11 rather than slot 12; sight and reach keep the
// constructor's values because no column writes them; and the experience value
// is 137, slot 37's, which it can be ONLY if slots 33 to 36 were consumed and
// dropped rather than skipped.
var wantDistinct = UnitDef{
	Body: 100, Reaction: 101, Mind: 102, Spirit: 103,
	Health: 104, HealthMax: 104, HealthRegenPeriod: 105,
	Mana: 106, ManaMax: 106, ManaRegenPeriod: 107,
	Speed: 108, RotationSpeed: 109, ScanRange: 110, Withdraw: 134, Wimpy: 135, SeeInvisible: 136,
	Sight: 0, Reach: 1,
	DamageBase: 111, DamageSpread: 1, AlwaysHits: true,
	ToHit: 114, Defence: 115, Absorption: 116,
	AttackChargeTime: 117, AttackRelaxTime: 118,
	Protection: [5]int32{119, 120, 121, 122, 123},
	Resistance: [5]int32{124, 125, 126, 127, 128},
	TypeID:     129, Face: 130, TokenSize: 131, MovementType: 132,
	DyingTime:  133,
	XPValue:    137,
	GoldChance: 138, TreasureMin: 139, TreasureMax: 140,
}

// TestEveryMappedFieldHoldsItsOwnSlot is AC-3's second half and SC-2/SC-3: every
// field takes its own slot's value, health follows the maximum, mana follows the
// maximum, and the experience value lands on slot 37.
func TestEveryMappedFieldHoldsItsOwnSlot(t *testing.T) {
	got, err := NewUnitDef("Distinct", distinctRow(41))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	want := fieldsOf(wantDistinct)
	for name, v := range fieldsOf(got) {
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

func TestNoLoadedFieldHoldsTheSentinel(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  []int32
	}{
		{"every cell empty", sentinelRow(41)},
		{"every cell written", distinctRow(41)},
		{"the sentinel only where it hurts", func() []int32 {
			// A row that would put a -1 in every array element and in the two
			// keys a search reads, if a store ever ran on one.
			p := distinctRow(41)
			for _, s := range []int{19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32} {
				p[s] = -1
			}
			return p
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewUnitDef("P1", tc.row)
			if err != nil {
				t.Fatalf("NewUnitDef: %v", err)
			}
			for name, v := range fieldsOf(d) {
				if n, ok := v.(int32); ok && n == -1 {
					t.Errorf("%s holds the sentinel", name)
				}
			}
		})
	}
}

// slotOutcomes says, for each streamed slot, exactly which fields a value in
// that slot alone may move.
var slotOutcomes = map[int][]string{
	0: {"Body"}, 1: {"Reaction"}, 2: {"Mind"}, 3: {"Spirit"},
	4:  {"HealthMax", "Health"},
	5:  {"HealthRegenPeriod"},
	6:  {"ManaMax", "Mana"},
	7:  {"ManaRegenPeriod"},
	8:  {"Speed"},
	9:  {"RotationSpeed"},
	10: {"ScanRange"},
	// The two damage columns reach the pair through the locals, so a minimum on
	// its own moves both halves and a maximum on its own moves the spread.
	11: {"DamageBase", "DamageSpread"},
	12: {"DamageSpread"},
	13: {"AlwaysHits"},
	14: {"ToHit"}, 15: {"Defence"}, 16: {"Absorption"},
	17: {"AttackChargeTime"}, 18: {"AttackRelaxTime"},
	19: {"Protection[0]"}, 20: {"Protection[1]"}, 21: {"Protection[2]"},
	22: {"Protection[3]"}, 23: {"Protection[4]"},
	24: {"Resistance[0]"}, 25: {"Resistance[1]"}, 26: {"Resistance[2]"},
	27: {"Resistance[3]"}, 28: {"Resistance[4]"},
	29: {"TypeID"}, 30: {"Face"}, 31: {"TokenSize"}, 32: {"MovementType"},
	// Withdraw and Wimpy are distinct absolute-health thresholds. See invisible
	// has its own target-filter consumer.
	33: {"DyingTime"}, 34: {"Withdraw"}, 35: {"Wimpy"}, 36: {"SeeInvisible"},
	37: {"XPValue"}, 38: {"GoldChance"}, 39: {"TreasureMin"}, 40: {"TreasureMax"},
}

func TestEachSlotHasExactlyOneOutcome(t *testing.T) {
	if got := len(slotOutcomes); got != lastUnitSlot+1 {
		t.Fatalf("the outcome table names %d slots, want %d", got, lastUnitSlot+1)
	}
	base := UnitDefaults()
	for slot := 0; slot <= lastUnitSlot; slot++ {
		row := sentinelRow(41)
		// The selector's only value that leaves a mark of its own.
		row[slot] = 3
		if slot != 13 {
			row[slot] = int32(1000 + slot)
		}
		d, err := NewUnitDef("P3", row)
		if err != nil {
			t.Fatalf("slot %d: NewUnitDef: %v", slot, err)
		}
		got := changedFields(base, d)
		want := slotOutcomes[slot]
		if !sameSet(got, want) {
			t.Errorf("slot %d moved %v, want exactly %v", slot, got, want)
		}
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

// TestTheDamageSelectorRoutesAndRefuses is AC-4 and SC-4.
func TestTheDamageSelectorRoutesAndRefuses(t *testing.T) {
	row := func(selector int32) []int32 {
		p := sentinelRow(38)
		p[11], p[12], p[13] = 40, 70, selector
		return p
	}
	for _, tc := range []struct {
		name       string
		selector   int32
		alwaysHits bool
	}{
		// An ABSENT selector: the cell is empty, so nothing is stored and the
		// local keeps the zero it was pre-set to. That is why -1 takes arm 0,
		// and not because -1 is less than zero.
		{"absent", -1, false},
		{"zero", 0, false},
		{"always-hit", 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewUnitDef("Beast", row(tc.selector))
			if err != nil {
				t.Fatalf("NewUnitDef: %v", err)
			}
			if d.DamageBase != 40 || d.DamageSpread != 30 {
				t.Errorf("damage pair = (%d, %d), want (40, 30) — base is the minimum and spread the difference",
					d.DamageBase, d.DamageSpread)
			}
			if d.AlwaysHits != tc.alwaysHits {
				t.Errorf("AlwaysHits = %v, want %v", d.AlwaysHits, tc.alwaysHits)
			}
		})
	}

	for _, selector := range []int32{1, 2} {
		t.Run(fmt.Sprintf("arm %d is refused", selector), func(t *testing.T) {
			d, err := NewUnitDef("Beast", row(selector))
			if err == nil {
				t.Fatalf("selector %d was accepted; it names a field pair this contract does not model", selector)
			}
			if !strings.Contains(err.Error(), "Beast") {
				t.Errorf("error %q does not name the offending entry", err)
			}
			if d != (UnitDef{}) {
				t.Errorf("a refused entry yielded %+v, want the zero value", d)
			}
		})
	}
}

// TestNothingPastSlot40IsRead pins the streamed window: later cells belong to
// other consumers.
func TestNothingPastSlot40IsRead(t *testing.T) {
	narrow, err := NewUnitDef("Narrow", distinctRow(41))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	wide, err := NewUnitDef("Wide", distinctRow(55))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	if narrow != wide {
		t.Errorf("a 55-cell row loaded differently from its 41-cell prefix:\n %+v\n %+v", wide, narrow)
	}
}

// TestARowShorterThanTheStreamIsRefused: a Units row with no parameters at all —
// 62 of the shipped 118 are exactly that — cannot yield a definition.
func TestARowShorterThanTheStreamIsRefused(t *testing.T) {
	for _, n := range []int{0, 1, 37} {
		d, err := NewUnitDef("Short", sentinelRow(n))
		if err == nil {
			t.Errorf("a %d-cell row was accepted", n)
		}
		if d != (UnitDef{}) {
			t.Errorf("a refused row yielded %+v, want the zero value", d)
		}
	}
	if _, err := NewUnitDef("Exact", sentinelRow(38)); err != nil {
		t.Errorf("a row of exactly the streamed width was refused: %v", err)
	}
}

// TestTheDefaultsValueIsReadOnly: UnitDefaults hands out a copy, so a caller
// cannot move the value every definition starts from.
func TestTheDefaultsValueIsReadOnly(t *testing.T) {
	d := UnitDefaults()
	d.HealthMax = 1
	if UnitDefaults().HealthMax != 30 {
		t.Error("the defaults value moved through the copy handed out")
	}
}

func TestABareUnitDefinitionsCombatReachIsOne(t *testing.T) {
	d, err := NewUnitDef("Bare", sentinelRow(38))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	if got := d.Combat().Reach; got != 1 {
		t.Errorf("Combat().Reach = %d, want the constructor's floor of 1", got)
	}
}

func TestUnitDefCombatCopiesReach(t *testing.T) {
	d, err := NewUnitDef("Bare", sentinelRow(38))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	d.Reach = 6
	if got := d.Combat().Reach; got != 6 {
		t.Errorf("Combat().Reach = %d, want the field's own 6", got)
	}
}

// TestProtectionNamesMatchTheColumnsTheyTitle pins the one property a caller
// relies on: names[i] titles the column UnitDef.Protection[i] was filled from.
// The binding is UNIT-COMBAT-015's — columns 19…23, `prot Fire..Astral` — and
// NewUnitDef's own case arms store them in that order, so this asserts the two
// agree rather than restating either.
func TestProtectionNamesMatchTheColumnsTheyTitle(t *testing.T) {
	var d UnitDef
	if got, want := len(ProtectionNames()), len(d.Protection); got != want {
		t.Fatalf("ProtectionNames() has %d entries, Protection holds %d — a caller pairing them "+
			"by index would title a column that is not there", got, want)
	}
	for i, n := range ProtectionNames() {
		if n == "" {
			t.Errorf("ProtectionNames()[%d] is empty", i)
		}
	}
	// It is a COPY. A caller that sorted or relabelled the result must not move
	// the package's own list, which every later caller reads.
	got := ProtectionNames()
	first := got[0]
	got[0] = "clobbered"
	if again := ProtectionNames(); again[0] != first {
		t.Errorf("ProtectionNames()[0] became %q after a caller wrote to a previous result", again[0])
	}
}

// TestUnitDefCombatCopiesTheSpellPair mirrors TestUnitDefCombatCopiesReach
// for 0139's own pair: Combat carries exactly this definition's own
// SpellName and SpellPower, whatever a caller — the placement loader that
// resolves an equipped weapon, a later task's job — has put there.
// NewUnitDef never writes either field (no Units column states an
// attachment), which is why they are moved by hand here rather than
// through a row.
func TestUnitDefCombatCopiesTheSpellPair(t *testing.T) {
	d, err := NewUnitDef("Bare", sentinelRow(38))
	if err != nil {
		t.Fatalf("NewUnitDef: %v", err)
	}
	d.SpellName, d.SpellPower = "Fire_Arrow", 10
	c := d.Combat()
	if c.SpellName != "Fire_Arrow" || c.SpellPower != 10 {
		t.Errorf("Combat() spell = (%q, %d), want (\"Fire_Arrow\", 10)", c.SpellName, c.SpellPower)
	}
}
