//go:build sessioncorpusaudit

// Package game corpus proof for this story's own sole-review correction
// (its returning finding's item 1): currentActorSource's s.Class == 0
// fresh-mint branch now builds actor+0x130 (U130, HERO-XP-077's
// six-slot experience aggregate) from the live SkillXP slots. A Human or
// Humanoid record loaded FROM an original SAV already carries a source
// basis (Class != 0), so currentActorSource returns e.SourceNow() unchanged
// before that branch ever runs -- the fresh-mint fix must not touch a
// resaved original's own aggregate. This walks the discovered save corpus
// and proves that directly: for every readable original file this engine
// resumes and resaves, each named character's written U130 equals what the
// original file itself carried.
package game

import (
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestSAVU130CorpusResaveKeepsTheLoadedAggregate(t *testing.T) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}

	total, checked, comparedChars, mismatched := 0, 0, 0, 0
	var refused []milestone2ResumeRefusal

	milestone2Corpus(t, func(t *testing.T, mf milestone2File, _ *FrontEnd) {
		total++
		loadedChars, _, err := mf.f.PartyWalk()
		if err != nil || len(loadedChars) == 0 {
			// Not every corpus file carries a party (a pure city or map
			// record might not); nothing to compare here.
			return
		}
		loaded := map[string]uint32{}
		for _, c := range loadedChars {
			if c.Name != "" {
				loaded[c.Name] = c.Experience
			}
		}
		if len(loaded) == 0 {
			return
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
		for _, c := range writtenChars {
			want, ok := loaded[c.Name]
			if !ok {
				continue
			}
			comparedChars++
			if c.Experience != want {
				mismatched++
				t.Errorf("%s: %q resaved U130 = %d, want %d (the original's own loaded aggregate, unchanged by a resave)", mf.rel, c.Name, c.Experience, want)
			}
		}
	})

	if total == 0 {
		t.Fatal("save corpus is empty; AGAINROM_SAVE_CORPUS is misconfigured")
	}
	milestone2LogRefusals(t, "U130 resave corpus", refused)
	t.Logf("SAV-U130-RESAVE-CENSUS discovered=%d checked=%d compared-characters=%d mismatched=%d refused=%d", total, checked, comparedChars, mismatched, len(refused))
	if comparedChars == 0 {
		t.Fatal("compared zero characters across the whole corpus; the name-matched comparison found nothing to check")
	}
}
