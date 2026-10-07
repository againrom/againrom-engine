package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

// The original lets the player re-create hired units while the loaded units
// survive, so one saved MapUnitID can be claimed by several actors. Such a
// save must resume with every Entity-owning actor present, each under its own
// source binding, and the ALM unit bound to exactly one claimant.
func TestReleaseResumeStaleDuplicateMapUnitID(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: stale duplicate witness requires owner saves")
	}
	f := releaseFront(t)
	kits := filepath.Join(corpus, "..", "..", "review", "owner-sav-m10-kits", "original-results", "en")
	for _, tc := range []struct {
		path      string
		claimants int
	}{
		{filepath.Join(corpus, "2026-10-02", "projectiles-original-en", "game0022.sav"), 3},
		{filepath.Join(corpus, "2026-10-02", "projectiles-original-en", "game0023.sav"), 3},
		{filepath.Join(corpus, "2026-10-02", "projectiles-original-en", "game0024.sav"), 3},
		{filepath.Join(kits, "game0016.sav"), 0},
	} {
		name := filepath.Base(tc.path)
		raw, err := os.ReadFile(tc.path)
		if err != nil {
			t.Log("corpus file missing:", err)
			continue
		}
		ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		document, origins := decodeSavedDocument(raw)
		graph, err := currentActorGraph(file, document, origins)
		if err != nil {
			t.Fatal(err)
		}
		want, claimants := 0, 0
		for _, a := range graph.Actors {
			if ownsOriginalEntity(a) {
				want++
				if a.MapUnitID == 128 {
					claimants++
				}
			}
		}
		got, owned := map[uint16]bool{}, 0
		for _, e := range ms.World.Entities() {
			if e.SourceBinding.ArchiveIndex == 0 || got[e.SourceBinding.ArchiveIndex] {
				continue
			}
			got[e.SourceBinding.ArchiveIndex] = true
			if e.MapUnitID == 128 {
				owned++
			}
		}
		if len(got) != want {
			t.Fatalf("%s: %d source actors present, SAV has %d", name, len(got), want)
		}
		if tc.claimants != 0 && (claimants != tc.claimants || owned != tc.claimants) {
			t.Fatalf("%s: MapUnitID 128 claimants %d present %d, want %d", name, claimants, owned, tc.claimants)
		}
	}
}
