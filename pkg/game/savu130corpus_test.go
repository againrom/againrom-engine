// Package game corpus proof that a resave writes each named character's
// actor+0x130 (U130, HERO-XP-077's six-slot experience aggregate) as the
// LOAD left it. An original SAV's record carries a source basis, so
// currentActorSource and applyCurrentCityHuman keep the stored aggregate. An
// Againrom-written SAV's members load without a Human basis (DIV-2885), and
// the SAVE builds their aggregate from the six skill counters (DIV-2472).
package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestU130MismatchesForAuditKeepsOriginalsAndSumsAgainromSAVs(t *testing.T) {
	// A record whose stored aggregate is not its six-slot sum.
	named := func(name string, experience uint32, slots ...uint32) sav.Character {
		c := sav.Character{Class: "Human", Name: name, Experience: experience}
		copy(c.SkillXP[:], slots)
		return c
	}
	hero := named("hero", 211433, 0, 171859, 186990, 0, 6354, 9606)
	for _, tc := range []struct {
		name     string
		loaded   []sav.Character
		written  []sav.Character
		original bool
		compared int
		misses   int
	}{
		{"original keeps its aggregate", []sav.Character{hero}, []sav.Character{named("hero", 211433)}, true, 1, 0},
		{"original rebuilt from slots", []sav.Character{hero}, []sav.Character{named("hero", 374809)}, true, 1, 1},
		{"Againrom SAV writes the slot sum", []sav.Character{hero}, []sav.Character{named("hero", 374809)}, false, 1, 0},
		{"Againrom SAV keeps the stored word", []sav.Character{hero}, []sav.Character{named("hero", 211433)}, false, 1, 1},
		{"slot sum wraps", []sav.Character{named("hero", 7, 0xffffffff, 2)}, []sav.Character{named("hero", 1)}, false, 1, 0},
		{"shared name in reverse order", []sav.Character{named("npc", 10), named("npc", 20)}, []sav.Character{named("npc", 20), named("npc", 10)}, true, 2, 0},
		{"shared name written twice", []sav.Character{named("npc", 10), named("npc", 20)}, []sav.Character{named("npc", 20), named("npc", 20)}, true, 2, 1},
		{"unnamed and unknown skipped", []sav.Character{named("", 10), hero}, []sav.Character{named("", 11), named("other", 1)}, true, 0, 0},
	} {
		compared, misses := u130MismatchesForAudit(tc.loaded, tc.written, tc.original)
		if compared != tc.compared || len(misses) != tc.misses {
			t.Errorf("%s: compared %d, misses %v; want %d compared, %d misses", tc.name, compared, misses, tc.compared, tc.misses)
		}
	}
}

// u130WantForAudit is the aggregate a resave writes for one loaded
// character: the stored U130 of an original (SAV-HEROXP-063), or for an
// Againrom-written SAV the wrapping sum of its six stored skill counters
// (DIV-2472, the sum HERO-XP-077 maintains).
func u130WantForAudit(c sav.Character, original bool) uint32 {
	if original {
		return c.Experience
	}
	var sum uint32
	for _, xp := range c.SkillXP {
		sum += xp
	}
	return sum
}

// u130MismatchesForAudit joins each written named character to a loaded
// one of the same name. A name several records carry pairs each written
// record with one unpaired loaded value it equals, so neither record's
// value stands in for the other's.
func u130MismatchesForAudit(loaded, written []sav.Character, original bool) (int, []string) {
	wants := map[string][]uint32{}
	for _, c := range loaded {
		if c.Name != "" {
			wants[c.Name] = append(wants[c.Name], u130WantForAudit(c, original))
		}
	}
	compared := 0
	var misses []string
	for _, c := range written {
		left, ok := wants[c.Name]
		if !ok {
			continue
		}
		compared++
		at := -1
		for i, want := range left {
			if want == c.Experience {
				at = i
				break
			}
		}
		if at < 0 {
			misses = append(misses, fmt.Sprintf("%q resaved U130 = %d, want one of %v (original=%v)", c.Name, c.Experience, left, original))
			continue
		}
		wants[c.Name] = append(left[:at:at], left[at+1:]...)
	}
	return compared, misses
}
