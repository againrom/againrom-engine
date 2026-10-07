package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The save dialog must not refuse or encode a name one way on one branch and
// another way on another. Before this test's own predecessor,
// playerMissionSave reached ExportCurrentWorldSave -- which assigns the
// label straight into the document and has no rule of its own -- before
// anything consulted the ASCII rule, so a player who typed a non-ASCII save
// name inside a running mission got a file whose slot name neither this
// build nor the original can display, while the same name taken in town was
// refused. The rule lives in exactly one function, encodeSaveLabel, and
// every producer the dialog can reach calls it first (DIV-1230).
//
// This is the guard's own test. The dialog-level proof that the rule is
// reached before any producer runs is
// TestReleaseSaveLabel1198MissionBranchRefusesBeforeProducing, which needs the
// owner's corpus and is gated with the rest of the release set.
func TestSaveLabel1198RefusalIsOnePlace(t *testing.T) {
	// A name the selected install's own page cannot represent at all -- a
	// Latin letter Windows-1251 has no byte for, a control byte, a script the
	// page never carries, a code point past the Basic Multilingual Plane --
	// is still a real refusal, at either selector.
	const want = "SAV label cannot represent"
	for _, label := range []string{
		"café",        // Latin-1 beyond ASCII: Windows-1251 has no 'é'
		"ok\x01",      // control byte
		"مرحبا",       // Arabic
		"a\U0001f600", // astral plane
	} {
		for _, selector := range []int{0, 1} {
			if _, err := encodeSaveLabel(label, selector); err == nil {
				t.Fatalf("label %q must be refused for a SAV slot at selector %d", label, selector)
			} else if !strings.Contains(err.Error(), want) {
				t.Fatalf("label %q refused with the wrong reason: %v", label, err)
			}
		}
	}

	// A name this build CAN write must not be caught. Over-refusing is the same
	// defect pointed the other way: the player loses a save he could have had.
	for _, label := range []string{
		"Second city",
		"Mission20 return",
		" ", // 0x20, the low bound, inclusive
		"~", // 0x7e, the high bound, inclusive
		"a-b_c.1 (2)",
	} {
		for _, selector := range []int{0, 1} {
			got, err := encodeSaveLabel(label, selector)
			if err != nil {
				t.Fatalf("label %q is writable and must not be refused: %v", label, err)
			}
			if got != label {
				t.Fatalf("an ASCII label must round-trip unchanged: encodeSaveLabel(%q, %d) = %q", label, selector, got)
			}
		}
	}

	// A Cyrillic name is now WRITTEN, not refused: Windows-1251 covers it at
	// either selector. It must come back re-encoded (not passed through as
	// UTF-8), with no NUL byte, and the two selectors must disagree with each
	// other -- textinput.EncodeRune's own doc says selector 1 undoes
	// render/text's glyph shift, so a byte written at one selector is wrong
	// at the other.
	const cyrillic = "Моя игра"
	encodedBySelector := map[int]string{}
	for _, selector := range []int{0, 1} {
		encoded, err := encodeSaveLabel(cyrillic, selector)
		if err != nil {
			t.Fatalf("a Cyrillic label is writable in Windows-1251 and must not be refused at selector %d: %v", selector, err)
		}
		if encoded == cyrillic {
			t.Fatalf("a Cyrillic label must be re-encoded, not passed through unchanged: selector %d", selector)
		}
		if strings.ContainsRune(encoded, 0) {
			t.Fatalf("an encoded label must never contain a NUL byte: %q", encoded)
		}
		encodedBySelector[selector] = encoded
	}
	if encodedBySelector[0] == encodedBySelector[1] {
		t.Fatalf("selector 0 and selector 1 produced the same bytes for %q; the shift is not exercised", cyrillic)
	}

	// The empty name is not this rule's business. Whatever rejects it, it is
	// not the code page.
	if _, err := encodeSaveLabel("", 0); err != nil {
		t.Fatalf("the empty label is not a code-page refusal: %v", err)
	}
}

// saveLabel1198MissionBranchRefusesBeforeProducing is the dialog-level half:
// on a mission whose battlefield producer SUCCEEDS, so that the detached-city
// fallback -- which carries the rule inside playerCitySave -- is never
// reached, a name the install's own page cannot represent must still be
// refused, and a name it CAN represent must reach the file encoded, not
// refused and not passed through as UTF-8. Without the call at the top of
// playerMissionSave this subtest writes a file with the wrong bytes (or
// none) instead.
//
// It uses mission20, the same fixture saveDialogPreTownBattlefield1198 uses,
// precisely because its battlefield branch is the one that succeeds.
func saveLabel1198MissionBranchRefusesBeforeProducing(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e")
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	app := f.App("1198 label refusal")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != 20 {
		t.Fatal("fixture is not the mission20 battlefield this subtest needs")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if s.Mission == 0 {
		t.Fatal("snapshot is not from a running mission; this subtest cannot discriminate")
	}

	// The control: this exact snapshot and an ASCII name produce a file
	// through the battlefield branch. That is what makes the checks below
	// statements about the label and not about the state.
	produced, notice, err := f.playerMissionSave(s, "Mission20 return")
	if err != nil || len(produced) == 0 {
		t.Fatal("the battlefield branch must succeed here for this subtest to mean anything", err)
	}
	if notice != "" {
		t.Fatalf("the battlefield branch fell back and this subtest no longer discriminates: %q", notice)
	}

	// A name outside the install's own page is still a real refusal --
	// Arabic is outside Windows-1251 at both selectors, so this holds
	// regardless of which install produced this fixture.
	const arabic = "مرحبا"
	refused, notice, err := f.playerMissionSave(s, arabic)
	if err == nil {
		t.Fatalf("a save name outside the install's own page produced %d bytes instead of a refusal", len(refused))
	}
	if !strings.Contains(err.Error(), "SAV label cannot represent") {
		t.Fatalf("the mission branch refused for the wrong reason: %v", err)
	}
	if len(refused) != 0 || notice != "" {
		t.Fatalf("a refusal returned bytes or a notice: %d bytes, notice %q", len(refused), notice)
	}

	// A Cyrillic name IS in the install's own page and must produce a file
	// through the same battlefield branch, encoded rather than refused or
	// passed through as UTF-8.
	const cyrillic = "Моя игра"
	written, notice, err := f.playerMissionSave(s, cyrillic)
	if err != nil {
		t.Fatalf("a Cyrillic mission save name must be written, not refused: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("a Cyrillic mission save name produced no bytes")
	}
	if notice != "" {
		t.Fatalf("the battlefield branch fell back for a name it should have written: %q", notice)
	}
	if strings.Contains(string(written), cyrillic) {
		t.Fatalf("the SAV bytes must carry the encoded label, not the typed UTF-8 text")
	}
	// The round trip closes end to end, not only at the codec's own unit
	// tests: OriginalSaveLabel -- SaveStore/OriginalStore's own chooser-row
	// reader -- recovers the exact typed name from the bytes this dialog
	// actually wrote, read at the SAME install's own selector.
	decoded, err := OriginalSaveLabel(written, f.textSelector())
	if err != nil {
		t.Fatalf("the written file must be a readable chooser row: %v", err)
	}
	// OriginalSaveLabel appends a known suffix (originalsave.go): a prefix
	// check would still pass on trailing garbage, so assert the exact string
	// this snapshot's own mission number produces.
	want := fmt.Sprintf("%s - mission %d", cyrillic, s.Mission)
	if decoded != want {
		t.Fatalf("OriginalSaveLabel(written, %d) = %q; want %q", f.textSelector(), decoded, want)
	}
}

// saveLabel1198TownShortCircuitCarriesNoNotice pins the s.Mission == 0 branch.
// Removing that short-circuit lets a town SAV carry the DIV-1334 disclosure it
// did not earn -- the file did not fall back from any battlefield, because
// there was no battlefield -- and nothing in the suite caught that (review
// note 3).
func saveLabel1198TownShortCircuitCarriesNoNotice(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: this subtest needs a preserved town save")
	}
	// Discovered, not named: the corpus grows, and the first file that
	// restores as a town is as good as any other for this branch.
	var f *FrontEnd
	found := ""
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found != "" || !strings.EqualFold(filepath.Ext(path), ".sav") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		g := releaseFront(t)
		if _, town, err := g.RestoreOriginal(raw); err == nil && town {
			f, found = g, path
		}
		return nil
	})
	if found == "" {
		t.Skip("no preserved save in the corpus restores as a town")
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Mission != 0 {
		t.Fatal("town snapshot reports a mission; this subtest cannot discriminate")
	}
	_, notice, err := f.playerMissionSave(s, "Town save")
	if err != nil {
		t.Fatal("town SAV through the dialog's mission producer", err)
	}
	if strings.Contains(notice, "DIV-1334") {
		t.Fatalf("a town save carried the battlefield-fallback disclosure it did not earn: %q", notice)
	}
}

// The two subtests are handed to t.Run as literal closures containing a
// bare-name call, not as function values. internal/gatedtests.Scan is a
// lexical scan and says so in its own package comment: it follows bare-name
// calls through the package's _test.go files and does not resolve function
// values. Written as `t.Run("x", helperName)` this test is invisible to that
// scan while the asset-free runtime census still finds it, so the source scan
// and the checked-in population disagree and
// TestScanMatchesTheCheckedInPopulationList fails. That is what happened here.
func TestReleaseSaveLabel1198MissionBranchRefusesBeforeProducing(t *testing.T) {
	t.Run("mission_branch_refuses", func(t *testing.T) {
		saveLabel1198MissionBranchRefusesBeforeProducing(t)
	})
	t.Run("town_short_circuit", func(t *testing.T) {
		saveLabel1198TownShortCircuitCarriesNoNotice(t)
	})
}
