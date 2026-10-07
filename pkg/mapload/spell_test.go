package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// slotHumanKnownSpells is the Humans row's own spellbook slot (0127 FR-4a),
// spelled out here rather than imported, on this suite's own rule: a test
// asserting that a column reached a field must not read that column's
// position out of the code it is testing.
const slotHumanKnownSpells = 25

// spellRowWidth is one past the last slot a Spells row's own contract reads
// (18), so spellRow's array always reaches it — spellRow's own local shape
// of pkg/data's lastSpellRowSlots, not imported for the reason every slot
// number in this suite is spelled out rather than read off the code under
// test.
const spellRowWidth = 19

// spellRow is a Spells row carrying the established scalar slots, leaving
// every other cell at the format's own empty sentinel — pkg/data's own
// spellRow (spell_test.go), transcribed here because the two packages' tests
// do not share fixtures. A Complication witness authors slot 0 after return.
func spellRow(manaCost, school, target, maxRange, dmgMin, dmgMax, defensive int32) []int32 {
	p := make([]int32, spellRowWidth)
	for i := range p {
		p[i] = -1
	}
	p[1], p[2], p[4], p[6], p[16], p[17], p[18] = manaCost, school, target, maxRange, dmgMin, dmgMax, defensive
	// Slot 8, `Distribution system`, is authored as a POINT row rather than
	// left at the empty sentinel: LoadSpells reads "1 is a point effect,
	// anything else is an area effect", so -1 would make every row this helper
	// builds an area row, which is neither what its callers describe nor the
	// shipped table's own majority.
	p[8] = 1
	return p
}

func TestASpellTableReachesTheWorldFieldForField(t *testing.T) {
	spellDefs := defCollection{
		{}, // 0: reserved, never loaded
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)},
	}
	spellDefs[1].params[0] = 3 // Complication Level
	table := &mapload.Table{Spells: spellDefs}
	m := &alm.Map{Width: 10, Height: 10}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	want := []sim.SpellRule{{
		ID: 1, Complication: 3, ManaCost: 3, School: 1, MaxRange: 7,
		DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true,
		Distribution: 1,
	}}
	if got := w.Spells(); !reflect.DeepEqual(got, want) {
		t.Errorf("world holds spells %+v, want %+v", got, want)
	}
}

func TestATableWithNoUsableSpellsCollectionStillBuildsAWorld(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10}
	for _, tc := range []struct {
		name  string
		table *mapload.Table
	}{
		{"a nil table", nil},
		{"a table naming no Spells collection at all", &mapload.Table{}},
		{"a Spells collection holding no rows", &mapload.Table{Spells: defCollection{}}},
		{"a Spells collection whose one row is too short to load", &mapload.Table{
			Spells: defCollection{{}, {name: "short", params: []int32{0}}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := mapload.FromALMWith(m, tc.table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatalf("FromALMWith: %v", err)
			}
			if got := w.Spells(); len(got) != 0 {
				t.Errorf("world holds %d spell row(s), want none: %+v", len(got), got)
			}
		})
	}
}

// TestAPlacedHumansKnownSpellsMaskReachesHisEntity is AC-5a's own value
// carried the whole way to an entity (0127 FR-4a): a Humans row stating
// 266306 at slot 25 — the shipped ManMage_Staff mask, spells 1, 6, 12 and
// 18 — must land on the placed entity's KnownSpells exactly, and a
// placement that resolves to no human row at all must carry the empty book
// rather than any part of it.
func TestAPlacedHumansKnownSpellsMaskReachesHisEntity(t *testing.T) {
	t.Run("a resolved human row", func(t *testing.T) {
		m := &alm.Map{Width: 10, Height: 10,
			Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 7}}}
		table := &mapload.Table{
			Humans: defCollection{
				{},
				{name: "h7", params: humanRow(map[int]int32{
					slotHumanType: 7, slotHumanKnownSpells: 266306,
				})},
			},
		}
		w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("FromALMWith: %v", err)
		}
		if got := w.Entities()[0].KnownSpells; got != 266306 {
			t.Errorf("entity's book is %d, want the row's own 266306", got)
		}
	})

	// A placement whose key resolves to no row at all — a units row, or
	// nothing — is a CREATURE or UNRESOLVED, neither of which this build
	// reads a book off (fromalm.go's own doc on the field), so the entity
	// it produces carries the zero value: an empty book.
	t.Run("a placement resolving to no human row", func(t *testing.T) {
		m := &alm.Map{Width: 10, Height: 10,
			Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 12345}}}
		w := mapload.FromALM(m)
		if got := w.Entities()[0].KnownSpells; got != 0 {
			t.Errorf("entity's book is %d, want 0 (empty) for an unresolved placement", got)
		}
	})
}

// spellsFixture is a Spells collection carrying the four shipped tokens
// spec FR-1b names, PLUS a duplicate of the first at a later row — the tie
// FR-1b's own tie-break has to resolve — and one row whose name has no
// token this test ever asks for. Ids are the collection's own subscripts:
// "Fire Arrow" is 1, the LATER "Fire Arrow" is 5, "Fire Ball" is 2,
// "Lightning" is 3, "Prismatic Spray" is 4.
func spellsFixture() defCollection {
	return defCollection{
		{}, // 0: reserved
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)},
		{name: "Fire Ball", params: spellRow(5, 1, 1, 10, 6, 12, 0)},
		{name: "Lightning", params: spellRow(4, 3, 1, 9, 5, 9, 0)},
		{name: "Prismatic Spray", params: spellRow(6, 2, 0, 6, 3, 7, 0)},
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)}, // 5: the later duplicate
	}
}

// TestSpellIDByTokenMatchesTheUnderscoreSubstitution is FR-1b's own
// grammar: TOKEN with every `_` replaced by a space matched EXACTLY against
// a row's Name — measured against all four shipped tokens spec FR-1b names,
// so this test would catch a substitution applied to the wrong character or
// a match that is only a prefix or a substring.
func TestSpellIDByTokenMatchesTheUnderscoreSubstitution(t *testing.T) {
	spells := spellsFixture()
	table := &mapload.Table{Spells: spells}
	for _, tc := range []struct {
		token string
		want  uint16
	}{
		{"Fire_Arrow", 1},
		{"Fire_Ball", 2},
		{"Lightning", 3}, // no underscore at all: the substitution is a no-op, not a requirement
		{"Prismatic_Spray", 4},
	} {
		t.Run(tc.token, func(t *testing.T) {
			got, ok := mapload.SpellIDByToken(table, tc.token)
			if !ok || got != tc.want {
				t.Errorf("SpellIDByToken(%q) = (%d, %v), want (%d, true)", tc.token, got, ok, tc.want)
			}
		})
	}
}

// TestSpellIDByTokenFirstRowWins is FR-1b's own tie-break: where two rows
// carry one name, the FIRST wins — spellsFixture's row 5 repeats row 1's
// own "Fire Arrow", and the answer must stay 1.
func TestSpellIDByTokenFirstRowWins(t *testing.T) {
	table := &mapload.Table{Spells: spellsFixture()}
	got, ok := mapload.SpellIDByToken(table, "Fire_Arrow")
	if !ok || got != 1 {
		t.Errorf("SpellIDByToken(%q) = (%d, %v), want (1, true) — the FIRST Fire Arrow, not the fifth",
			"Fire_Arrow", got, ok)
	}
}

// TestSpellIDByTokenAnswersFalseForTheUnresolvable is FR-2b's own upstream
// half: every shape a token cannot resolve through answers (0, false) —
// never half of a match — so a caller downstream (start.go, fromalm.go,
// pkg/game's rearm) can treat "no id" as "this weapon carries no spell"
// uniformly, on WeaponSpell's own "zero is none" rule (pkg/sim/world.go).
func TestSpellIDByTokenAnswersFalseForTheUnresolvable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *mapload.Table
		token string
	}{
		{"a nil table", nil, "Fire_Arrow"},
		{"a table naming no Spells collection", &mapload.Table{}, "Fire_Arrow"},
		{"an empty token — FR-1a's own zero pair", &mapload.Table{Spells: spellsFixture()}, ""},
		{"a token matching no row", &mapload.Table{Spells: spellsFixture()}, "Teleport"},
		{"a token whose substitution matches no row (case differs)", &mapload.Table{Spells: spellsFixture()}, "fire_arrow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := mapload.SpellIDByToken(tc.table, tc.token); ok || got != 0 {
				t.Errorf("SpellIDByToken(%q) = (%d, %v), want (0, false)", tc.token, got, ok)
			}
		})
	}
}
