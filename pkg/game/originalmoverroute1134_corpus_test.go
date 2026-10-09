//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestMoverRouteCorpusAudit1134(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	f.SetDeterministicFrames(true)

	total, worlds, cities, unreadable, nonEmptyRoute, mismatches, formRefusals := 0, 0, 0, 0, 0, 0, 0
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
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source, err := sav.Open(raw)
			if err != nil {
				unreadable++
				t.Logf("unreadable, skipped: %v", err)
				return
			}
			if source.World == nil {
				cities++
				t.Log("between-mission save: no mover/route state, nothing to compare")
				return
			}
			worlds++
			graph, err := source.ActorGraph()
			if err != nil {
				t.Fatalf("ActorGraph: %v", err)
			}
			want := archiveActorsForAudit(graph)
			for _, a := range want {
				if len(a.StaticRoute) > 0 || len(a.DynamicRoute) > 0 {
					nonEmptyRoute++
					break
				}
			}

			ms, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatalf("RestoreOriginal: %v", err)
			}
			if _, _, _, present := ms.World.SavedActorMotions(); !present {
				mismatches++
				t.Error("resumed world carries no saved motion state")
				return
			}

			// The live comparison value is the world's own carried state before
			// any tick: RestoreOriginal advances none. This walk never calls
			// exportOriginalMoverRoutes, so a bug shared with the production
			// join would not silently pass both this and the byte-level check
			// below.
			live := archiveMotionsForAudit(ms.World)
			if msg, ok := sameMoverRouteForAudit(want, live); !ok {
				mismatches++
				t.Errorf("file vs live mover/route: %s", msg)
			}

			// The world's OWN byte form (pkg/sim's MarshalBinary/UnmarshalBinary,
			// the same shape a native in-game Save/.ags reload uses) is a
			// different form than the original-SAV-format native export below,
			// and this story's own review (F-1) found a continuation could be
			// minted that this form's own decodeRoutes then refused, which
			// surfaced only at the player's next in-game Save with no file
			// written at all rather than here at admission. This is the
			// corpus-wide, repeatable form of that same measurement.
			if form, err := ms.World.MarshalBinary(); err != nil {
				formRefusals++
				mismatches++
				t.Errorf("world MarshalBinary: %v", err)
			} else {
				var back sim.World
				if err := back.UnmarshalBinary(form); err != nil {
					formRefusals++
					mismatches++
					t.Errorf("world byte-form round trip: %v", err)
				} else if back.Hash() != ms.World.Hash() {
					formRefusals++
					mismatches++
					t.Error("world byte-form round trip changed Hash()")
				}
			}

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes, then compare the complete body, on
			// TestCellRecordCorpusAudit1131's own Marshal/re-Open precedent.
			// exportOriginalMoverRoutes only ever patches scalar bytes inside
			// each actor's own mover/route spans
			// (TestSetActorMoverRouteLeavesUnrelatedBytesAlone), so a
			// whole-body comparison against the untouched source is exactly as
			// strong as isolating those spans would be.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalMoverRoutes(target, ms.World); err != nil {
				t.Fatalf("exportOriginalMoverRoutes: %v", err)
			}
			written, err := sav.Open(target.Marshal())
			if err != nil {
				t.Fatalf("sav.Open (re-decode written): %v", err)
			}
			if len(written.Body) != len(source.Body) {
				mismatches++
				t.Errorf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
			} else {
				diff := 0
				for i := range source.Body {
					if source.Body[i] != written.Body[i] {
						diff++
					}
				}
				if diff > 0 {
					mismatches++
					t.Errorf("native export of the mover/route state does not reproduce the source file: %d byte(s) differ", diff)
				}
			}
			t.Logf("actors=%d", len(want))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d with a non-empty route, %d mismatching, %d failing the world's own byte-form round trip",
		total, worlds, cities, unreadable, nonEmptyRoute, mismatches, formRefusals)
}
