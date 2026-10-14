package game

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/sim"
)

// teleportSeriesDir finds the owner's EN Teleport series in the save corpus:
// five saves written during one mission-110 cast admitted on sub-tick 10656
// (SAV-1254). It answers "" when the corpus does not hold it.
func teleportSeriesDir(corpus string) string {
	found := ""
	_ = filepath.WalkDir(corpus, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && d.Name() == "exp-teleport-series" && found == "" {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// TestReleaseOwnerTeleportSeriesRelocatesOnTheCountdownTick loads each save of
// the owner's series, copied out first, and steps it. From each save taken in
// the countdown the caster stands on (20,62) through sub-tick 10663 and stands
// on (12,62) from sub-tick 10664, the tick the original's last save shows the
// move (SAV-1254, SAV-1256); that last save loads with the caster moved.
func TestReleaseOwnerTeleportSeriesRelocatesOnTheCountdownTick(t *testing.T) {
	corpus, assets := os.Getenv("AGAINROM_SAVE_CORPUS"), os.Getenv("AGAINROM_ASSETS")
	if corpus == "" || assets == "" {
		t.Skip("AGAINROM_SAVE_CORPUS and AGAINROM_ASSETS are required")
	}
	dir := teleportSeriesDir(corpus)
	if dir == "" {
		t.Skip("the save corpus holds no exp-teleport-series directory")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the owner's series is a first-game mission")
	}
	f.SetDeterministicFrames(true)
	const relocation = 10664
	work := t.TempDir()
	for _, tc := range []struct {
		name  string
		tick  uint64
		moved bool
	}{
		{"game0023.sav", 10658, false},
		{"game0025.sav", 10660, false},
		{"game0026.sav", 10661, false},
		{"game0027.sav", 10662, false},
		{"game0028.sav", 10664, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			copied := filepath.Join(work, tc.name)
			if err := os.WriteFile(copied, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if raw, err = os.ReadFile(copied); err != nil {
				t.Fatal(err)
			}
			ms, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatalf("LOAD: %v", err)
			}
			w := ms.World
			if w.Tick() != tc.tick {
				t.Fatalf("loaded at sub-tick %d, want %d", w.Tick(), tc.tick)
			}
			caster := sim.EntityID(0)
			at := sim.CellPoint{X: 20, Y: 62}
			if tc.moved {
				at = sim.CellPoint{X: 12, Y: 62}
			}
			for _, e := range w.Entities() {
				if e.Alive() && e.X == at.X && e.Y == at.Y && e.KnownSpells&(1<<26) != 0 {
					caster = e.ID
				}
			}
			if caster == 0 {
				t.Fatalf("no living Teleport caster on (%d,%d) at load", at.X, at.Y)
			}
			for w.Tick() < relocation+4 {
				r := sim.StepReported(w, nil)
				e, _ := ms.World.Entity(caster)
				wantX := int32(20)
				if w.Tick() >= relocation {
					wantX = 12
				}
				if e.X != wantX || e.Y != 62 {
					t.Fatalf("sub-tick %d: caster on (%d,%d), want (%d,62)", w.Tick(), e.X, e.Y, wantX)
				}
				if moved := len(r.Relocations) == 1 && r.Relocations[0].Caster == caster; moved != (w.Tick() == relocation && !tc.moved) {
					t.Fatalf("sub-tick %d: relocations %+v", w.Tick(), r.Relocations)
				}
				if len(r.Casts) != 0 {
					t.Fatalf("sub-tick %d: a loaded cast in flight reported a second cast %+v", w.Tick(), r.Casts)
				}
			}
			// The two loaded picture-60 records keep their points and draw one
			// phase per sub-tick from the admission on 10656 (SAV-1255).
			records := 0
			for _, r := range w.SavedProjectiles().Items {
				if r.Picture != 60 {
					continue
				}
				records++
				if r.Y != 15943 || r.X != 5200 && r.X != 3152 || uint64(r.Phase) != w.Tick()-10656 || r.ActionPhase+r.ActionSegments != 21 {
					t.Errorf("sub-tick %d: record %d at (%d,%d) phase %d actionphase %d remaining %d", w.Tick(), r.ID, r.X, r.Y, r.Phase, r.ActionPhase, r.ActionSegments)
				}
			}
			if records != 2 {
				t.Errorf("sub-tick %d: %d picture-60 records, want the two the save holds", w.Tick(), records)
			}
			e, _ := ms.World.Entity(caster)
			t.Logf("%s: loaded at %d, caster %d on (%d,%d) at %d, mana %d, hash %016x", tc.name, tc.tick, caster, e.X, e.Y, w.Tick(), e.Mana, w.Hash())
		})
	}
}
