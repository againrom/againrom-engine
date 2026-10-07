//go:build sessioncorpusaudit

// Package game permanent round-trip regression instrument: load, resave,
// compare semantic state, reload, over the discovered corpus and our own save
// points. It runs beside the tagged milestone-2 gate and names every refusal
// instead of failing on the first one, so a growing corpus stays honest.
//
// It follows the milestone2Corpus family's own conventions
// (milestone2_acceptance_reader_test.go) rather than inventing new ones: the
// corpus is DISCOVERED by walking a directory, never a fixed file list; every
// file gets its own t.Run subtest; and a file that cannot complete the cycle
// is a named, logged REFUSAL, never a t.Fatal that silently drops the rest of
// the population.
package game

import (
	"maps"
	"os"
	"slices"
	"testing"

	"againrom/pkg/mapload"
)

// sav1195Refusal names one corpus file this instrument's round trip could not
// complete: which file, which stage refused, and that stage's own error.
// Modeled on milestone2ResumeRefusal (milestone2_acceptance_reader_test.go):
// a refusal is a recorded fact about today's population, never a reason to
// drop the rest of it.
type sav1195Refusal struct {
	rel   string
	stage string
	err   error
}

// sav1195LogRefusals prints one line per refusal so a reader can see exactly
// which file and stage excluded it, on milestone2LogRefusals' own shape.
func sav1195LogRefusals(t *testing.T, label string, refused []sav1195Refusal) {
	t.Helper()
	for _, r := range refused {
		t.Logf("%s: REFUSED %s stage=%s: %v", label, r.rel, r.stage, r.err)
	}
}

// savTripCompare reports every semantic mismatch between the state before the
// round trip and the state after it, each as its own t.Errorf so one field's
// drift does not hide another's. See the file doc comment for the exact field
// list, why it is Snapshot-to-Snapshot rather than city1166Check's
// FrontEnd-to-Snapshot shape, and why Offered/Fame are included. It returns
// whether it reported an undisclosed mismatch, a disclosed field or a migration.
// corpusLossCeilings names the fields one corpus half is PERMITTED to lose,
// each against the divergence row that discloses it and a ceiling on how many
// files may show it.
//
// THIS CANNOT QUIETLY ABSORB A REGRESSION, which is the only reason it may
// exist at all. A field that is not named still fails the moment it differs. A
// field that IS named is logged with its row, counted, and checked against its
// ceiling after the walk, so the same disclosed loss spreading to one more
// file fails exactly as an undisclosed field would. The per-field counts print
// on every run, passing or not, so the population stays visible instead of
// being summarized into one number.
//
// A file whose every difference is disclosed counts as DISCLOSED. The
// bounded hired-pool representation change counts as MIGRATED. Only a file
// identical on every compared field counts as exact.
//
// Valuable Documents appear in no disclosure set. Under the owner's own ruling
// that is the state which must survive, so it is not a field anything may be
// permitted to lose.
type sav1195Disclosure struct {
	row     string
	ceiling int
}

type corpusLossCeilings map[string]sav1195Disclosure

// originalRoundTripDisclosed is empty on purpose. The original corpus round
// trips with no disclosed loss at all, and an entry appearing here would be a
// fidelity regression against the original game's own saves.
var originalRoundTripDisclosed = corpusLossCeilings{}

// sav1195CheckDisclosed fails when a disclosed field reaches more files than
// its ceiling admits, and prints the whole tally either way.
func sav1195CheckDisclosed(t *testing.T, label string, disclosed corpusLossCeilings, tally map[string]int) {
	t.Helper()
	for _, field := range slices.Sorted(maps.Keys(disclosed)) {
		n, row := tally[field], disclosed[field]
		// SAV-ROUNDTRIP- must lead: check-milestone2-acceptance.sh selects
		// this file's lines with `:[0-9]+: SAV-ROUNDTRIP-`, so a label in
		// front of it drops the whole tally out of the gate's own output.
		t.Logf("SAV-ROUNDTRIP-DISCLOSED %s %s=%d (ceiling %d, %s)", label, field, n, row.ceiling, row.row)
		if n > row.ceiling {
			t.Errorf("%s: disclosed field %q reached %d files, ceiling %d (%s): a disclosed loss that spreads is still a regression",
				label, field, n, row.ceiling, row.row)
		}
	}
}

func TestSAVHiredPoolMigrationBound(t *testing.T) {
	member := mapload.PartyMember{ID: "mercenary:3:1", MercenaryType: 3}
	before := Snapshot{Mission: 50, Party: []mapload.PartyMember{member}}
	before.MercenaryHired[3] = true
	after := before
	after.CampaignState = true
	after.Campaign.MercenaryWorking = make([]uint16, 15)
	after.Campaign.MercenaryWorking[2] = 1
	after.MercenaryPool[3] = 1
	if !isHiredPoolMigration(before, after, 3) {
		t.Fatal("the measured legacy-mission-to-SAV representation was rejected")
	}
	ok, disclosed, migrated := savTripCompare(t, "synthetic hired-pool migration", before, after, corpusLossCeilings{}, map[string]int{})
	if !ok || disclosed || !migrated {
		t.Fatalf("migration classification = %t/%t/%t, want accepted / not disclosed / migrated", ok, disclosed, migrated)
	}
	for _, tc := range []struct {
		name string
		edit func(*Snapshot, *Snapshot)
	}{
		{"lost squad member", func(_, a *Snapshot) { a.Party = nil }},
		{"changed mercenary type", func(_, a *Snapshot) { a.Party[0].MercenaryType = 4 }},
		{"wrong campaign working count", func(_, a *Snapshot) { a.Campaign.MercenaryWorking[2] = 0 }},
		{"wrong restored stock", func(_, a *Snapshot) { a.MercenaryPool[3] = 2 }},
		{"existing campaign source", func(b, _ *Snapshot) { b.CampaignState = true }},
		{"town stock", func(b, a *Snapshot) { b.Mission, a.Mission = 0, 0 }},
		{"hire flag lost", func(_, a *Snapshot) { a.MercenaryHired[3] = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, a := before, after
			b.Party, a.Party = slices.Clone(before.Party), slices.Clone(after.Party)
			a.Campaign.MercenaryWorking = slices.Clone(after.Campaign.MercenaryWorking)
			tc.edit(&b, &a)
			if isHiredPoolMigration(b, a, 3) {
				t.Fatal("a real loss was accepted as a representation change")
			}
		})
	}
}

func savTripCompare(t *testing.T, rel string, before, after Snapshot, disclosed corpusLossCeilings, tally map[string]int) (ok, wasDisclosed, wasMigrated bool) {
	t.Helper()
	ok = true
	// counted keeps the tally a FILE count. The roster loop reports once per
	// differing member, so without this a save whose two members both lose the
	// hero record would bill two against a ceiling that means "files" and would
	// eat a second save's headroom while reading as if it had not.
	counted := map[string]bool{}
	diffs, migrated := CompareConvertedStates(before, after)
	for _, typ := range migrated {
		wasMigrated = true
		t.Logf("SAV-ROUNDTRIP-MIGRATED %s hired type %d: mission Town stock 0 -> campaign working %d with %d party members retained",
			rel, typ, after.Campaign.MercenaryWorking[typ-1], nativeMercenaryPartyCount(after.Party, typ))
	}
	for _, diff := range diffs {
		if row, named := disclosed[diff.Field]; named {
			if !counted[diff.Field] {
				counted[diff.Field] = true
				tally[diff.Field]++
			}
			wasDisclosed = true
			t.Logf("%s: DISCLOSED %s (%s): %s", rel, diff.Field, row.row, diff.Message)
			continue
		}
		ok = false
		t.Errorf("%s: %s", rel, diff.Message)
	}
	return ok, wasDisclosed, wasMigrated
}

// sav1195Baseline bounds the accepted comparison count (exact, disclosed and
// migrated), migration spread and refusal reasons. Its aggregate floor cannot
// detect swaps between files, so owner save names stay out of the baseline.
// Update it with production changes and EN/RU corpus measurements.
type sav1195Baseline struct {
	discovered      int
	accepted        int
	migratedCeiling int
	refusals        map[string]int
}

// The original corpus keeps its accepted floor across the direct SAV reload.
// No refusal or semantic loss is admitted for a newly exercised writer path.
var sav1195OriginalBaseline = sav1195Baseline{
	discovered: 111, accepted: 110, migratedCeiling: 0,
	refusals: map[string]int{},
}

// sav1195CheckBaseline enforces the accepted floor and the migration and
// refusal ceilings for one corpus.
func sav1195CheckBaseline(t *testing.T, label string, baseline sav1195Baseline, discovered, accepted, migrated int, refusals map[string]int) {
	t.Helper()
	if discovered < baseline.discovered {
		t.Errorf("%s: discovered corpus shrank: %d, baseline %d", label, discovered, baseline.discovered)
	}
	if accepted < baseline.accepted {
		t.Errorf("%s: accepted comparison count regressed: %d, baseline %d", label, accepted, baseline.accepted)
	}
	if migrated > baseline.migratedCeiling {
		t.Errorf("%s: hired-pool migration spread to %d files, ceiling %d", label, migrated, baseline.migratedCeiling)
	}
	for reason, n := range refusals {
		if allowed := baseline.refusals[reason]; n > allowed {
			t.Errorf("%s: refusal reason rose above its baseline: %q now %d, baseline %d", label, reason, n, allowed)
		}
	}
}

func TestSAVRoundTrip1195OriginalCorpus(t *testing.T) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}

	readable, exactFiles, mismatches, disclosedFiles, migratedFiles := 0, 0, 0, 0, 0
	tally := map[string]int{}
	byReason := map[string]int{}
	var refused []sav1195Refusal

	discovered := milestone2Corpus(t, func(t *testing.T, mf milestone2File, _ *FrontEnd) {
		readable++
		refuse := func(stage string, err error) {
			refused = append(refused, sav1195Refusal{rel: mf.rel, stage: stage, err: err})
			byReason[stage+": "+err.Error()]++
			t.Logf("REFUSED %s stage=%s: %v", mf.rel, stage, err)
		}

		// Independent fresh FrontEnds on both sides of the cycle, matching
		// TestSAVConvertedCorpusContinuation and TestNativeResumeCorpusCycle1139's
		// own "no reused live receiver" discipline -- this measures a genuine
		// cross-process resume, not continuity inside one FrontEnd.
		before, err := NewFrontEnd(assets)
		if err != nil {
			t.Fatalf("NewFrontEnd(%q): %v", assets, err)
		}
		cleanupFrontAudio(t, before)
		before.SetDeterministicFrames(true)
		open, town, err := before.RestoreOriginal(mf.raw)
		if err != nil {
			refuse("LOAD", err)
			return
		}
		if !town {
			if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
				refuse("LOAD-COMMIT", err)
				return
			}
		}
		want, _, err := currentRoundTripSnapshot(before, !town)
		if err != nil {
			refuse("SNAPSHOT", err)
			return
		}

		captured, _, err := before.Snapshot(!town)
		if err != nil {
			refuse("SAVE-SNAPSHOT", err)
			return
		}
		savBytes, err := before.ExportCurrentSave(captured, string(mf.f.Label))
		if err != nil {
			refuse("EXPORT-SAV", err)
			return
		}

		after, err := NewFrontEnd(assets)
		if err != nil {
			t.Fatalf("NewFrontEnd(%q): %v", assets, err)
		}
		cleanupFrontAudio(t, after)
		after.SetDeterministicFrames(true)
		open2, town2, err := after.RestoreOriginal(savBytes)
		if err != nil {
			refuse("RELOAD", err)
			return
		}
		if !town2 {
			if _, _, _, _, _, _, _, _, _, _, err = open2(); err != nil {
				refuse("RELOAD-COMMIT", err)
				return
			}
		}
		got, _, err := currentRoundTripSnapshot(after, !town2)
		if err != nil {
			refuse("RELOAD-SNAPSHOT", err)
			return
		}

		ok, wasDisclosed, wasMigrated := savTripCompare(t, mf.rel, want, got, originalRoundTripDisclosed, tally)
		switch {
		case !ok:
			mismatches++
		case wasMigrated:
			migratedFiles++
		case wasDisclosed:
			disclosedFiles++
		default:
			exactFiles++
		}
	})

	if discovered == 0 {
		t.Fatal("original SAV round-trip corpus is empty; AGAINROM_SAVE_CORPUS is misconfigured")
	}
	sav1195LogRefusals(t, "original-corpus", refused)
	for reason, n := range byReason {
		t.Logf("SAV-ROUNDTRIP-ORIGINAL-REASON %d: %s", n, reason)
	}
	accepted := exactFiles + disclosedFiles + migratedFiles
	exercised := accepted + mismatches
	if exercised == 0 {
		t.Logf("SAV-ROUNDTRIP-ORIGINAL-NOTE: the comparator ran on zero files this pass -- no readable file reached the reload/compare step, so mismatched=0 is an empty-set result, not a measurement of zero semantic loss")
	}
	t.Logf("SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=%d readable=%d exact=%d disclosed=%d migrated=%d accepted=%d refused=%d mismatched=%d comparator-exercised=%d",
		discovered, readable, exactFiles, disclosedFiles, migratedFiles, accepted, len(refused), mismatches, exercised)
	sav1195CheckDisclosed(t, "original-corpus", originalRoundTripDisclosed, tally)
	sav1195CheckBaseline(t, "original-corpus", sav1195OriginalBaseline, discovered, accepted, migratedFiles, byReason)
}
