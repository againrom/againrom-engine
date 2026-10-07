package data

import "testing"

// testScaleTable is a ScaleTable built directly in test code, the same way
// defsearch_test.go's testCollection builds a Collection: a slice of names,
// with EntryDoubles answering nil because no test in this file reads a
// scale factor — takePrefix and impliedShapePrefix read only names.
type testScaleTable []string

func (t testScaleTable) Len() int                     { return len(t) }
func (t testScaleTable) EntryName(i int) string       { return t[i] }
func (t testScaleTable) EntryDoubles(i int) []float64 { return nil }

// TestTakePrefixDescendingWalkPicksTheLastEntryThatMatches is FR-2a: the
// table is walked from its LAST entry down to its first, and the first one
// that occurs anywhere in the subject wins — not the longest, not the one
// that occurs earliest in the subject, and not the one lowest in the table.
// Both "Steel" (index 0) and "Iron" (index 1) occur in the subject below;
// the walk starts at index 1 and stops there, so "Steel" — which the OLD
// ascending longest-match walk would also have found — is never even
// reached.
func TestTakePrefixDescendingWalkPicksTheLastEntryThatMatches(t *testing.T) {
	table := testScaleTable{"Steel", "Iron"}

	index, name, rest := takePrefix("Iron Steel Blade", table)
	if index != 1 || name != "Iron" {
		t.Fatalf("takePrefix = (%d, %q, %q), want index 1, name \"Iron\" — the LAST entry that matches",
			index, name, rest)
	}
}

// TestTakePrefixPositionalRebuildMangles is FR-2a's own arithmetic: the
// second slice is s[len(name)+1:], with NO found term added to it, so a
// match that does not sit at the front of the subject leaves the rebuild
// cutting from the WRONG offset. "Steel" is found at offset 5 in "Fine
// Steel Blade"; a plain trim would answer "Fine Blade", but the positional
// rebuild answers "Fine teel Blade" — it drops "Fine " (five bytes, offset
// 0 to 5) and then len("Steel")+1 == 6 bytes measured from the FRONT of the
// original string, not from where "Steel" was actually found, so the 'S' at
// offset 5 survives as a lone 't' short of it and the six bytes actually
// removed are "Fine S". This is the "mangled rather than refused" contract
// FR-2a states, demonstrated on a name authored out of order.
func TestTakePrefixPositionalRebuildMangles(t *testing.T) {
	table := testScaleTable{"Steel"}

	index, name, rest := takePrefix("Fine Steel Blade", table)
	if index != 0 || name != "Steel" {
		t.Fatalf("takePrefix matched (%d, %q), want (0, \"Steel\")", index, name)
	}
	if rest != "Fine teel Blade" {
		t.Fatalf("takePrefix rest = %q, want the mangled \"Fine teel Blade\" — "+
			"\"Fine Blade\" would be a TRIM, which is not this contract", rest)
	}
}

func TestTakePrefixGuardsTheSecondSliceAgainstOverrun(t *testing.T) {
	table := testScaleTable{"Blade"}

	index, name, rest := takePrefix("Blade", table)
	if index != 0 || name != "Blade" || rest != "" {
		t.Fatalf("takePrefix(%q) = (%d, %q, %q), want (0, \"Blade\", \"\")", "Blade", index, name, rest)
	}
}

func TestTakePrefixNoMatchAnswersIndexZeroUnchanged(t *testing.T) {
	for _, tc := range []struct {
		label string
		table ScaleTable
	}{
		{"a nil table", nil},
		{"an empty table", testScaleTable{}},
		{"a table naming nothing in the subject", testScaleTable{"Bronze", "Copper"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			index, name, rest := takePrefix("Axe", tc.table)
			if index != 0 || name != "" || rest != "Axe" {
				t.Errorf("takePrefix = (%d, %q, %q), want (0, \"\", \"Axe\")", index, name, rest)
			}
		})
	}
}

// TestTakePrefixSkipsAnEmptyNamedEntry: a table sized past its last written
// entry carries empty-named cells the same way a Collection does, and the
// walk must step past one rather than "matching" it as an empty substring —
// every string contains "", so a naive strings.Index would hit index 3 first
// and answer nothing was ever taken.
func TestTakePrefixSkipsAnEmptyNamedEntry(t *testing.T) {
	table := testScaleTable{"Iron", "", "", ""}

	index, name, rest := takePrefix("Iron Axe", table)
	if index != 0 || name != "Iron" || rest != "Axe" {
		t.Fatalf("takePrefix = (%d, %q, %q), want (0, \"Iron\", \"Axe\") — the empty-named entries must be skipped",
			index, name, rest)
	}
}

// TestImpliedShapePrefixBothArms is FR-2b: a material name containing
// "Leather" puts "Soft " back on the front of what is left, and one
// containing "Wood" puts "Wooden " back. Neither is a whole-word test —
// impliedShapePrefix reads strings.Contains, the same as the rest of this
// walk reads a substring and not a word.
func TestImpliedShapePrefixBothArms(t *testing.T) {
	for _, tc := range []struct {
		material, subject, want string
	}{
		{"Aged Leather", "Jerkin", "Soft Jerkin"},
		{"Pale Wood", "Buckler", "Wooden Buckler"},
		{"Bronze", "Jerkin", "Jerkin"}, // neither word: unchanged
	} {
		if got := impliedShapePrefix(tc.material, tc.subject); got != tc.want {
			t.Errorf("impliedShapePrefix(%q, %q) = %q, want %q", tc.material, tc.subject, got, tc.want)
		}
	}
}

func TestTakeCastSpellParsesTheWellFormedAttachment(t *testing.T) {
	for _, tc := range []struct {
		suffix    string
		wantToken string
		wantLevel int32
	}{
		{"{castSpell=Fire_Arrow:10}", "Fire_Arrow", 10},
		{"{castSpell=Fire_Ball:70}", "Fire_Ball", 70},
		{"{castSpell=Lightning:1}", "Lightning", 1},
		{"{castSpell=Prismatic_Spray:99}", "Prismatic_Spray", 99},
	} {
		token, level, ok := takeCastSpell(tc.suffix)
		if !ok {
			t.Fatalf("takeCastSpell(%q): ok = false, want true", tc.suffix)
		}
		if token != tc.wantToken || level != tc.wantLevel {
			t.Errorf("takeCastSpell(%q) = (%q, %d), want (%q, %d)",
				tc.suffix, token, level, tc.wantToken, tc.wantLevel)
		}
	}
}

// TestTakeCastSpellRefusesEveryMalformedShape is FR-1a: none of these leaves
// half a spell — ok is false and both other results are the zero value,
// for the two failures FR-1a names within a suffix that does open on
// `castSpell=` (no `:`; a LEVEL that is not a bare run of digits, tried with
// a sign, a separator, surrounding space and letters), plus the shapes that
// never reach that grammar at all: no suffix, and a `{…}` that is not this
// attachment (the shipped `{fire}` this package has never parsed).
func TestTakeCastSpellRefusesEveryMalformedShape(t *testing.T) {
	for _, tc := range []struct{ label, suffix string }{
		{"no suffix at all", ""},
		{"a suffix that is not this attachment", "{fire}"},
		{"a bracketed form with a colon but the wrong keyword", "{castsound=Fire_Arrow:10}"},
		{"missing the closing brace", "{castSpell=Fire_Arrow:10"},
		{"missing the colon", "{castSpell=Fire_Arrow10}"},
		{"an empty level", "{castSpell=Fire_Arrow:}"},
		{"a signed level", "{castSpell=Fire_Arrow:-10}"},
		{"a level with a separator", "{castSpell=Fire_Arrow:1,0}"},
		{"a level with surrounding space", "{castSpell=Fire_Arrow: 10}"},
		{"a non-digit level", "{castSpell=Fire_Arrow:ten}"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			token, level, ok := takeCastSpell(tc.suffix)
			if ok {
				t.Fatalf("takeCastSpell(%q): ok = true, want false", tc.suffix)
			}
			if token != "" || level != 0 {
				t.Errorf("takeCastSpell(%q) = (%q, %d), want (\"\", 0) — a malformed attachment carries no half of one",
					tc.suffix, token, level)
			}
		})
	}
}

func TestImpliedShapePrefixLeftTrimsLeadingSpaces(t *testing.T) {
	for _, tc := range []struct {
		label, material, subject, want string
	}{
		{"leading run of spaces, leather arm", "Aged Leather", "   Tunic", "Soft Tunic"},
		{"no leading space, leather arm", "Aged Leather", "Tunic", "Soft Tunic"},
		{"leading run of spaces, wood arm", "Pale Wood", "  Targe", "Wooden Targe"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			if got := impliedShapePrefix(tc.material, tc.subject); got != tc.want {
				t.Errorf("impliedShapePrefix(%q, %q) = %q, want %q", tc.material, tc.subject, got, tc.want)
			}
		})
	}
}
