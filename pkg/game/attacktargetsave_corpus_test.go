package game

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestAttackTargetSaveCensus(t *testing.T) {
	sessionCorpusAudit(t)
	assets, output := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_ATTACK_TARGET_WITNESS")
	original, converted, engine := os.Getenv("AGAINROM_SAVE_CORPUS"), os.Getenv("AGAINROM_CONVERTED_CORPUS"), os.Getenv("AGAINROM_ATTACK_TARGET_ENGINE_SAVES")
	if assets == "" || output == "" || original == "" || converted == "" || engine == "" {
		t.Skip("AGAINROM_ASSETS, AGAINROM_ATTACK_TARGET_WITNESS, AGAINROM_SAVE_CORPUS, AGAINROM_CONVERTED_CORPUS and AGAINROM_ATTACK_TARGET_ENGINE_SAVES are required")
	}
	for _, source := range []struct {
		name, directory   string
		marker, recursive bool
	}{
		{"milestone-original-engine", original, true, true},
		{"milestone-converted-engine", converted, false, true},
		{"engine-saves", engine, false, false},
	} {
		t.Run(source.name, func(t *testing.T) {
			discovered, selected, resaved, excluded, before, after, failed := 0, 0, 0, 0, 0, 0, 0
			livingBefore, livingAfter, readingBefore, readingAfter := 0, 0, 0, 0
			beforeResaved, livingBeforeResaved, readingBeforeResaved, refused := 0, 0, 0, 0
			if _, err := os.Stat(source.directory); err != nil {
				t.Fatal(err)
			}
			err := filepath.WalkDir(source.directory, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					if path != source.directory && !source.recursive {
						return filepath.SkipDir
					}
					return corpusDirSkip(entry)
				}
				if filepath.Ext(entry.Name()) != ".sav" {
					return nil
				}
				discovered++
				rel, err := filepath.Rel(source.directory, path)
				if err != nil {
					return err
				}
				t.Run(filepath.ToSlash(rel), func(t *testing.T) {
					directory := filepath.Join(output, source.name, "input")
					raw := attackTargetCopiedInput(t, path, directory)
					doc, err := sav.DecodeDocumentData(raw)
					if err != nil {
						failed++
						if source.marker {
							t.Logf("DECODE refused, engine provenance unclassified %s: %v", path, err)
						} else {
							t.Errorf("DECODE refused %s: %v", path, err)
						}
						return
					}
					if source.marker {
						_, present, err := sav.NativeActions(doc.State)
						if err != nil {
							failed++
							t.Error(err)
							return
						}
						if !present {
							excluded++
							return
						}
					}
					selected++
					living, reading, faults := attackTargetWireCensus(doc)
					livingBefore += living
					readingBefore += reading
					before += len(faults)
					t.Logf("TARGET-CENSUS BEFORE source=%s sha256=%x living=%d reading=%d unresolved=%d faults=%+v", path, sha256.Sum256(raw), living, reading, len(faults), faults)
					written, err := attackTargetHeadlessResaveResult(assets, raw, filepath.Join(output, source.name), "output", "attack target census")
					if err != nil {
						refused++
						t.Logf("TARGET-CENSUS REFUSED source=%s sha256=%x living=%d reading=%d unresolved=%d reason=%v", path, sha256.Sum256(raw), living, reading, len(faults), err)
						return
					}
					next, err := sav.DecodeDocumentData(written)
					if err != nil {
						failed++
						t.Error(err)
						return
					}
					resaved++
					livingBeforeResaved += living
					readingBeforeResaved += reading
					beforeResaved += len(faults)
					living, reading, faults = attackTargetWireCensus(next)
					livingAfter += living
					readingAfter += reading
					after += len(faults)
					t.Logf("TARGET-CENSUS AFTER source=%s sha256=%x living=%d reading=%d unresolved=%d faults=%+v", path, sha256.Sum256(written), living, reading, len(faults), faults)
					if len(faults) != 0 {
						t.Errorf("resave has unresolved living target orders: %+v", faults)
					}
				})
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("TARGET-CENSUS TOTAL root=%s population=%s discovered=%d selected=%d resaved=%d excluded-no-native-actions=%d decode-failed=%d refused=%d unresaved=%d living-before-all=%d reading-before-all=%d unresolved-before-all=%d living-resaved=%d/%d reading-resaved=%d/%d unresolved-resaved=%d/%d living-unresaved=%d reading-unresaved=%d unresolved-unresaved=%d", assets, source.name, discovered, selected, resaved, excluded, failed, refused, selected-resaved, livingBefore, readingBefore, before, livingBeforeResaved, livingAfter, readingBeforeResaved, readingAfter, beforeResaved, after, livingBefore-livingBeforeResaved, readingBefore-readingBeforeResaved, before-beforeResaved)
			if discovered == 0 && !source.marker {
				t.Errorf("empty target census population %s", source.directory)
			}
			if selected > 0 && resaved == 0 {
				t.Errorf("no selected SAV was resaved in population %s: selected=%d refused=%d", source.directory, selected, refused)
			}
		})
	}
}
