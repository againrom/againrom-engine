//go:build sessioncorpusaudit

// Package game milestone-2 acceptance instrument (docs/1140/story.md), owner
// milestone 2 (pipeline/SAV-COMPLETION.md): an original ROM1 mission SAV
// reads back completely.
//
// INDEPENDENCE RULE. Every comparison in this file family instead reads the
// FILE side directly off this save's own decompressed body
// (rawU8/rawU16/rawU32/rawCString below), at an offset this file computes
// from a knowledge/claims citation — never through a pkg/formats/sav
// decoded Fields/Value/Object.Fields map, an ActorHoldings, SessionState,
// Cell, or BlockRecord. pkg/formats/sav is used only for two mechanical,
// generic facts this rule's own text allows: decompressing the container
// (sav.Open; the transport codec, pkg/formats/sav/codec.go, is a custom
// run/literal word coder this instrument does not re-derive, so f.Body is
// trusted as "the decompressed stream" and nothing past that), and locating
// a STRUCTURE'S OWN START OFFSET where a claim gives a layout relative to it
// — sav.File.World.BlocksDataOff/CellRecDataOff/SessionOff are exported by
// that package for exactly this (WorldHalf's own doc comment). Every claimed
// field byte within a structure is then read here with this file's own
// binary.LittleEndian arithmetic. The LIVE side is always this engine's own
// resumed game state (RestoreOriginal -> *Mission / sim.World accessors),
// never a second decode of the file.
//
// Selected by scripts/check-milestone2-acceptance.sh, required by the seat
// for original-resume or acceptance-instrument changes. Standalone command:
//
//	go test -tags sessioncorpusaudit -count=1 ./pkg/game/ -run 'TestMilestone2' -v
//
// AGAINROM_SAVE_CORPUS names gameversions/saves; AGAINROM_ASSETS names the EN
// or RU lawful root to resume through.
package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
)

func rawU16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off : off+2]) }
func rawU32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off : off+4]) }

// milestone2File is one opened save: its raw bytes, the decompressed body
// (sav.Open's f.Body, used only as bytes -- see the independence rule above)
// and the sav.File the World-half location fields (BlocksDataOff,
// CellRecDataOff, SessionOff) are read off. present reports whether this
// save carries a world half at all.
type milestone2File struct {
	rel     string
	raw     []byte
	body    []byte
	f       *sav.File
	present bool
}

// milestone2ResumeRefusal names one corpus file RestoreOriginal refused,
// with its own error text.
//
// This is not a claim that the discovered corpus carries zero refusals going
// forward. It grows on its own (F-1), and it DID carry a REFUSAL OF A
// DIFFERENT FAMILY -- a dead-actor identity bind (MapUnitID 0, a hired
// mercenary never an authored map placement) and a late-dead simulation
// stage (the decay ladder's Stage 2) the importer did not admit, neither one
// the item-ownership guard f1053f5 touched. That is what this type exists
// for: a future refusal, from any importer, in any family, gets named here
// instead of failing the whole population. No file list is fixed here on
// purpose -- the corpus is discovered and the list moves.
type milestone2ResumeRefusal struct {
	rel string
	err error
}

// milestone2LogRefusals prints one line per refusal under label (e.g. "map
// reopen"), so an instrument's own log names precisely which file it excluded
// and why -- never only a smaller count that does not by itself explain the
// gap.
func milestone2LogRefusals(t *testing.T, label string, refused []milestone2ResumeRefusal) {
	t.Helper()
	for _, r := range refused {
		t.Logf("%s: excluded (RestoreOriginal refused) %s: %v", label, r.rel, r.err)
	}
}

// milestone2Corpus walks AGAINROM_SAVE_CORPUS and calls fn once per readable
// .sav file, in its own t.Run subtest, on TestUnitResidualCorpusAudit1138's
// own shape. It counts and logs the same four population classes that audit
// does (total, world-half, between-mission, unreadable) so a reader can
// compare this instrument's population against that one's.
func milestone2Corpus(t *testing.T, fn func(t *testing.T, mf milestone2File, fe *FrontEnd)) int {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	fe, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	cleanupFrontAudio(t, fe)
	fe.SetDeterministicFrames(true)

	return walkOriginalSaveCorpus(t, corpus, func(t *testing.T, mf milestone2File) {
		fn(t, mf, fe)
	})
}

func walkOriginalSaveCorpus(t *testing.T, corpus string, fn func(t *testing.T, mf milestone2File)) int {
	t.Helper()
	total, worlds, cities, unreadable := 0, 0, 0, 0
	walkErr := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		total++
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		f, err := sav.Open(raw)
		if err != nil {
			unreadable++
			t.Logf("%s: unreadable, skipped: %v", rel, err)
			return nil
		}
		present := f.World != nil
		if present {
			worlds++
		} else {
			cities++
		}
		t.Run(rel, func(t *testing.T) {
			fn(t, milestone2File{rel: rel, raw: raw, body: f.Body, f: f, present: present})
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable",
		total, worlds, cities, unreadable)
	return total
}

func TestMilestone2CorpusCountsUnreadable(t *testing.T) {
	raw, err := cityfixture.Original(false)
	if err != nil {
		t.Fatal(err)
	}
	corpus := t.TempDir()
	for _, file := range []struct {
		name string
		raw  []byte
	}{
		{"good.sav", raw},
		{"bad.sav", []byte("not a SAV")},
		{"ignored.ags", raw},
		{"exp-control/hidden.sav", raw},
	} {
		path := filepath.Join(corpus, filepath.FromSlash(file.name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, file.raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	readable := 0
	discovered := walkOriginalSaveCorpus(t, corpus, func(t *testing.T, mf milestone2File) {
		readable++
		if mf.rel != "good.sav" {
			t.Errorf("callback file = %q, want good.sav", mf.rel)
		}
	})
	if discovered != 2 || readable != 1 {
		t.Fatalf("discovered=%d readable=%d, want discovered=2 readable=1", discovered, readable)
	}
}
