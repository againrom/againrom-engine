package main

import (
	"strings"
	"testing"
)

// The census verb's own arithmetic, synthetically. What it reports about a
// shipped install cannot be tested without one (golden rule 2), so what is
// tested here is the classification every count is made of: which of the five
// answers a given pair of columns falls into.

// TestGainlessIsThePassesOwnArithmetic pins the truncation test the "gain
// truncates to 0" column counts. A period above a hundred times the factored
// maximum leaves the gain at zero hundredths; at exactly a hundred times it,
// the gain is one hundredth and the unit does move.
func TestGainlessIsThePassesOwnArithmetic(t *testing.T) {
	for _, tc := range []struct {
		what           string
		max, period    int32
		factor         int32
		gainless, hold bool
	}{
		{what: "a doubled maximum of 45 at period 100 gains 90", max: 45, period: 100, factor: 2},
		{what: "a maximum of 1 at period 100 gains exactly 1", max: 1, period: 100, factor: 1},
		{what: "a maximum of 1 at period 101 gains nothing", max: 1, period: 101, factor: 1, gainless: true},
		{what: "a maximum of 45 at period 4501 gains nothing", max: 45, period: 4501, factor: 1, gainless: true},
		{what: "no maximum is a different answer", max: 0, period: 50, factor: 1},
		{what: "no period is a different answer", max: 45, period: 0, factor: 1},
	} {
		if got := gainless(tc.max, tc.period, tc.factor); got != tc.gainless {
			t.Errorf("%s: gainless = %v, want %v", tc.what, got, tc.gainless)
		}
	}
}

func TestTheTallyCountsEachColumnOnce(t *testing.T) {
	var tally regenTally
	tally.add(30, 30, 100, 0, 0, 50)  // ordinary: a health period, no pool
	tally.add(30, 30, 0, 0, 0, 50)    // no health period
	tally.add(20, 30, 100, 5, 40, 60) // wounded, and carries a pool
	tally.add(30, 30, 100, 0, 40, 0)  //
	tally.add(1, 1, 300, 1, 1, 300)   // both gains truncate to zero
	if tally.total != 5 {
		t.Errorf("total %d, want 5", tally.total)
	}
	if tally.mana != 3 {
		t.Errorf("mana maxima %d, want 3", tally.mana)
	}
	if tally.noHealthPeriod != 1 {
		t.Errorf("absent health periods %d, want 1", tally.noHealthPeriod)
	}
	if tally.noManaPeriod != 1 {
		t.Errorf("absent mana periods %d, want 1", tally.noManaPeriod)
	}
	if tally.poolWithoutPeriod != 1 {
		t.Errorf("pools with no period %d, want 1", tally.poolWithoutPeriod)
	}
	if tally.deadHealth != 1 || tally.deadMana != 1 {
		t.Errorf("truncating gains %d health and %d mana, want 1 and 1", tally.deadHealth, tally.deadMana)
	}
	if tally.wounded != 1 {
		t.Errorf("born wounded %d, want 1", tally.wounded)
	}
}

// TestTheRegenVerbIsReachableFromTheArgumentShape checks the dispatch alone:
// the verb is refused without a root, and a bad root reaches the verb rather
// than the usage arm. It opens no archive, so it needs no install.
func TestTheRegenVerbIsReachableFromTheArgumentShape(t *testing.T) {
	var out strings.Builder
	err := run([]string{regenFlag}, &out)
	if err == nil {
		t.Fatal("a bare -regen was accepted")
	}
	err = run([]string{regenFlag, t.TempDir()}, &out)
	if err == nil {
		t.Fatal("-regen against an empty directory was accepted")
	}
	if strings.Contains(err.Error(), "usage") {
		t.Errorf("-regen against a root reached the usage arm: %v", err)
	}
}
