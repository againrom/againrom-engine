//go:build sessioncorpusaudit

package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"againrom/pkg/formats/sav"
)

// The converted corpus is the SAV output of the one-time AGS conversion
// (a retired command), kept with the manifest that records each input's identity
// and every difference the conversion named as debt. This instrument is the
// SAV-to-SAV continuation over it: each converted file is loaded cold, saved
// again through the ordinary producer, loaded cold a second time and compared
// field by field, with no loss admitted. It reads no AGS.
//
// AGAINROM_CONVERTED_CORPUS names the directory holding the .sav outputs and
// conversion-manifest.json.

type convertedManifestFile struct {
	Source       string `json:"source"`
	SourceSHA256 string `json:"sourceSHA256"`
	SourceBytes  int    `json:"sourceBytes"`
	Output       string `json:"output"`
	OutputSHA256 string `json:"outputSHA256"`
	Status       string `json:"status"`
	Differences  []struct {
		Field string `json:"field"`
		Row   string `json:"row"`
	} `json:"differences"`
}

type convertedManifestDocument struct {
	Summary struct {
		Discovered  int `json:"discovered"`
		Converted   int `json:"converted"`
		Failed      int `json:"failed"`
		Undisclosed int `json:"undisclosed"`
	} `json:"summary"`
	Files []convertedManifestFile `json:"files"`
}

// The converted corpus measured on both roots: 114 inputs, every one converted.
var savConvertedBaseline = sav1195Baseline{discovered: 114, accepted: 114, migratedCeiling: 0, refusals: map[string]int{}}

// The converted corpus part tests share its files by size, savConvertedParts
// ways. The first part checks the manifest against the whole corpus; the part
// that completes the set checks the census of the whole.
func TestSAVConvertedCorpusContinuationPart1(t *testing.T) { savConvertedContinuation(t, 0) }
func TestSAVConvertedCorpusContinuationPart2(t *testing.T) { savConvertedContinuation(t, 1) }

const savConvertedParts = 2

// savConvertedRefusal is a refusal as a part summary carries it.
type savConvertedRefusal struct{ Rel, Stage, Err string }

// savConvertedSummary is one part's census.
type savConvertedSummary struct {
	Exact, Mismatches int
	Refused           []savConvertedRefusal
	ByReason          map[string]int
}

func savConvertedContinuation(t *testing.T, part int) {
	corpus := os.Getenv("AGAINROM_CONVERTED_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_CONVERTED_CORPUS must name the directory of converted SAV files and conversion-manifest.json")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	rawManifest, err := os.ReadFile(filepath.Join(corpus, "conversion-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest convertedManifestDocument
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	byOutput := map[string]convertedManifestFile{}
	for _, file := range manifest.Files {
		if file.Output == "" || file.Status == "failed" {
			if part == 0 {
				t.Errorf("manifest input %s has no converted output (status %q)", file.Source, file.Status)
			}
			continue
		}
		byOutput[strings.ToLower(file.Output)] = file
	}
	if part == 0 && (manifest.Summary.Failed != 0 || manifest.Summary.Undisclosed != 0) {
		t.Errorf("manifest records %d failed and %d undisclosed conversions", manifest.Summary.Failed, manifest.Summary.Undisclosed)
	}

	type convertedFile struct{ rel, path string }
	var files []convertedFile
	var names []string
	var sizes []int64
	seen := map[string]bool{}
	walkErr := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".sav") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		seen[strings.ToLower(rel)] = true
		files = append(files, convertedFile{rel, path})
		names, sizes = append(names, rel), append(sizes, info.Size())
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	total := len(files)
	if total == 0 {
		t.Fatal("converted corpus is empty; AGAINROM_CONVERTED_CORPUS is misconfigured")
	}
	if part == 0 {
		for output := range byOutput {
			if !seen[output] {
				t.Errorf("manifest output %s is missing from the corpus", output)
			}
		}
	}
	share := corpusPartOf(names, sizes, savConvertedParts)

	exactFiles, mismatches := 0, 0
	tally := map[string]int{}
	byReason := map[string]int{}
	var refused []sav1195Refusal

	// Two files at a time: a file is loaded, saved and reloaded alone, and the
	// shared counters below change under the lock. The group ends only when
	// every file has, so the census after it reads the whole share.
	var mu sync.Mutex
	workers := make(chan struct{}, 2)
	t.Run("files", func(t *testing.T) {
		for i, file := range files {
			if share[i] != part {
				continue
			}
			rel, path := file.rel, file.path
			t.Run(rel, func(t *testing.T) {
				t.Parallel()
				workers <- struct{}{}
				defer func() { <-workers }()
				refuse := func(stage string, err error) {
					mu.Lock()
					refused = append(refused, sav1195Refusal{rel: rel, stage: stage, err: err})
					byReason[stage+": "+err.Error()]++
					mu.Unlock()
					t.Logf("REFUSED %s stage=%s: %v", rel, stage, err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading a converted save must not fail: %v", err)
				}
				entry, listed := byOutput[strings.ToLower(rel)]
				sum := sha256.Sum256(raw)
				if !listed {
					t.Errorf("%s: converted file has no manifest entry", rel)
				} else if hex.EncodeToString(sum[:]) != entry.OutputSHA256 {
					t.Errorf("%s: converted bytes differ from the manifest identity", rel)
				}

				before, err := NewFrontEnd(assets)
				if err != nil {
					t.Fatalf("NewFrontEnd(%q): %v", assets, err)
				}
				cleanupFrontAudio(t, before)
				before.SetDeterministicFrames(true)
				open, town, err := before.RestoreOriginal(raw)
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
				want, err := before.ObserveLoadedState(!town)
				if err != nil {
					refuse("SNAPSHOT", err)
					return
				}
				captured, _, err := before.Snapshot(!town)
				if err != nil {
					refuse("SAVE-SNAPSHOT", err)
					return
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					refuse("SOURCE-DOCUMENT", err)
					return
				}
				resaved, err := before.ExportCurrentSave(captured, string(doc.Label))
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
				got, err := after.LoadObservedSAV(resaved)
				if err != nil {
					refuse("RELOAD", err)
					return
				}
				fileTally := map[string]int{}
				ok, _, _ := savTripCompare(t, rel, want, got, originalRoundTripDisclosed, fileTally)
				mu.Lock()
				defer mu.Unlock()
				for field, n := range fileTally {
					tally[field] += n
				}
				if !ok {
					mismatches++
				} else {
					exactFiles++
				}
			})
		}
	})
	mine := savConvertedSummary{Exact: exactFiles, Mismatches: mismatches, ByReason: byReason}
	for _, r := range refused {
		mine.Refused = append(mine.Refused, savConvertedRefusal{Rel: r.rel, Stage: r.stage, Err: r.err.Error()})
	}
	set := corpusPartsCollect(t, "converted", part, savConvertedParts, mine)
	if set == nil {
		return
	}
	whole := savConvertedSummary{ByReason: map[string]int{}}
	for _, p := range corpusPartsDecode[savConvertedSummary](t, set) {
		whole.Exact += p.Exact
		whole.Mismatches += p.Mismatches
		whole.Refused = append(whole.Refused, p.Refused...)
		for reason, n := range p.ByReason {
			whole.ByReason[reason] += n
		}
	}
	refused = refused[:0]
	for _, r := range whole.Refused {
		refused = append(refused, sav1195Refusal{rel: r.Rel, stage: r.Stage, err: errors.New(r.Err)})
	}
	sort.Slice(refused, func(i, j int) bool { return refused[i].rel < refused[j].rel })
	sav1195LogRefusals(t, "converted-corpus", refused)
	for reason, n := range whole.ByReason {
		t.Logf("SAV-ROUNDTRIP-CONVERTED-REASON %d: %s", n, reason)
	}
	t.Logf("SAV-ROUNDTRIP-CONVERTED-CENSUS discovered=%d manifest=%d exact=%d refused=%d mismatched=%d",
		total, len(manifest.Files), whole.Exact, len(refused), whole.Mismatches)
	sav1195CheckBaseline(t, "converted-corpus", savConvertedBaseline, total, whole.Exact, 0, whole.ByReason)
}
