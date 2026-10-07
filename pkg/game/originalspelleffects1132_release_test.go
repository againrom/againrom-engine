package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseOriginalSpellEffectGraphRestoresOnLoad1132(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-15/game0018.sav",
		"1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	want, present, err := source.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	if len(want) == 0 {
		t.Fatal("fixture carries an empty SpellEffect list; this witness requires the known non-empty one")
	}
	root := want[0]
	if root.Class != "SpellTransport" || root.ST44 == nil || root.ST44.Class != "PointEffect" ||
		root.ST44.PE48 == nil || root.ST44.PE48.Class != "Effect_DirectDamage" || root.ST48 != nil {
		t.Fatalf("fixture does not carry SAV-EFFECTGRAPH-366's own witnessed shape: %+v", root)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1132 spell effects")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0018.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0018.sav")

	// Independently reconstructed live comparison: walks the file's own
	// decode against the live world state in parallel, on
	// TestSpellEffectCorpusAudit1132's own shape, rather than calling
	// exportOriginalSpellEffects/spellEffectConverter for both directions.
	live := f.live.world.SavedSpellEffects()
	if msg, ok := sameSpellEffectGraphForAudit(want, live); !ok {
		t.Fatalf("file vs live graph: %s", msg)
	}
	rootLive := live[0]
	if rootLive.Class != "SpellTransport" || rootLive.ST44 == nil || rootLive.ST44.Class != "PointEffect" ||
		rootLive.ST44.PE48 == nil || rootLive.ST44.PE48.Class != "Effect_DirectDamage" || rootLive.ST48 != nil {
		t.Fatalf("LOAD did not restore SAV-EFFECTGRAPH-366's own witnessed shape: %+v", rootLive)
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the complete body to reproduce the source
	// file byte for byte (exportOriginalSpellEffects only ever patches inside
	// the SpellEffect list's own span, TestSetSpellEffectsLeavesUnrelatedBytesAlone).
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalSpellEffects(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalSpellEffects: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	if len(written.Body) != len(source.Body) {
		t.Fatalf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
	}
	for i := range source.Body {
		if source.Body[i] != written.Body[i] {
			t.Fatalf("native export of the SpellEffect list does not reproduce the source file at byte %d", i)
		}
	}
	t.Logf("top-level records=%d: SpellTransport/PointEffect/Effect_DirectDamage chain and null AreaEffect arm restored; LOAD and native export both reproduce the source file exactly", len(want))
}

// TestReleaseEmptySpellEffectListStaysEmptyOnLoad1132 covers the shape every
// other reachable owner save carries (TestSpellEffectCorpusAudit1132: 49 of
// 51 world-half saves): an absent top-level SpellEffect list must decode as
// zero live records, not be confused with the true-empty/no-world
// distinction sav.File.SpellEffects already draws, and a native re-export
// must leave the source file byte for byte unchanged. It reuses story
// 1131's own fixture and hash.
func TestReleaseEmptySpellEffectListStaysEmptyOnLoad1132(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game9999.sav", "5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	want, present, err := source.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	if len(want) != 0 {
		t.Fatalf("fixture carries %d SpellEffect record(s); this witness requires the empty-list shape", len(want))
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1132 empty spell effects")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	if live := f.live.world.SavedSpellEffects(); len(live) != 0 {
		t.Fatalf("LOAD produced %d SpellEffect record(s) from an empty list", len(live))
	}

	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalSpellEffects(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalSpellEffects: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	if len(written.Body) != len(source.Body) {
		t.Fatalf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
	}
	for i := range source.Body {
		if source.Body[i] != written.Body[i] {
			t.Fatalf("native export of an empty SpellEffect list does not reproduce the source file at byte %d", i)
		}
	}
	t.Log("empty SpellEffect list: LOAD carried zero records and native export left the source file byte-identical")
}
