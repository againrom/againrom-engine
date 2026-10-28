// Package game corpus proof that a resave writes each named character's
// actor+0x130 (U130, HERO-XP-077's six-slot experience aggregate) as the
// LOAD left it. An original SAV's record carries a source basis, so
// currentActorSource and applyCurrentCityHuman keep the stored aggregate. An
// Againrom-written SAV's members load without a Human basis (DIV-2885), and
// the SAVE builds their aggregate from the six skill counters (DIV-2472).
package game

import (
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestSAVU130CorpusResaveKeepsTheLoadedAggregate(t *testing.T) {
	sessionCorpusAudit(t)
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}

	total, checked, comparedChars, mismatched, written := 0, 0, 0, 0, 0
	var refused []milestone2ResumeRefusal

	milestone2Corpus(t, func(t *testing.T, mf milestone2File, _ *FrontEnd) {
		total++
		loadedChars, _, err := mf.f.PartyWalk()
		if err != nil || len(loadedChars) == 0 {
			// Not every corpus file carries a party (a pure city or map
			// record might not); nothing to compare here.
			return
		}
		// No WriterCarriers bit carries a party Human basis, so only an
		// original's stored aggregate is what LOAD keeps.
		original, err := carrierHeldForAudit(mf.f, mf.raw, 0)
		if err != nil {
			t.Fatalf("%s: writer admission: %v", mf.rel, err)
		}

		before, err := NewFrontEnd(assets)
		if err != nil {
			t.Fatalf("NewFrontEnd(%q): %v", assets, err)
		}
		before.SetDeterministicFrames(true)
		open, town, err := before.RestoreOriginal(mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{rel: mf.rel, err: err})
			return
		}
		if !town {
			if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
				refused = append(refused, milestone2ResumeRefusal{rel: mf.rel, err: err})
				return
			}
		}
		snapshot, label, err := before.Snapshot(!town)
		if err != nil {
			t.Fatalf("%s: Snapshot: %v", mf.rel, err)
		}
		raw, err := before.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatalf("%s: ExportCurrentSave: %v", mf.rel, err)
		}
		doc, err := sav.Open(raw)
		if err != nil {
			t.Fatalf("%s: our own resave did not decode: %v", mf.rel, err)
		}
		writtenChars, _, err := doc.PartyWalk()
		if err != nil {
			t.Fatalf("%s: resave PartyWalk: %v", mf.rel, err)
		}
		checked++
		if !original {
			written++
		}
		compared, misses := u130MismatchesForAudit(loadedChars, writtenChars, original)
		comparedChars += compared
		mismatched += len(misses)
		for _, m := range misses {
			t.Errorf("%s: %s", mf.rel, m)
		}
	})

	if total == 0 {
		t.Fatal("save corpus is empty; AGAINROM_SAVE_CORPUS is misconfigured")
	}
	milestone2LogRefusals(t, "U130 resave corpus", refused)
	t.Logf("SAV-U130-RESAVE-CENSUS discovered=%d checked=%d againrom-written=%d compared-characters=%d mismatched=%d refused=%d", total, checked, written, comparedChars, mismatched, len(refused))
	if comparedChars == 0 {
		t.Fatal("compared zero characters across the whole corpus; the name-matched comparison found nothing to check")
	}
}

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
