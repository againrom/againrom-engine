package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseMilestone2Buildings1145(t *testing.T) {
	for _, variant := range []string{"original", "synthetic retained fields"} {
		t.Run(variant, func(t *testing.T) {
			f := releaseFront(t)
			path, raw := groundCorpusFile(t, "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0")
			source, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			const body = 46677 // independently transcribed first ruined Building
			if binary.LittleEndian.Uint32(source.Body[body+19:]) != 7 || binary.LittleEndian.Uint16(source.Body[body+60:]) != 0 || binary.LittleEndian.Uint16(source.Body[body+62:]) != 2000 {
				t.Fatal("original Building anchor changed")
			}
			if variant != "original" {
				// Only a temporary copy changes. Distinct dormant values prevent
				// all-default retention from satisfying the acceptance witness.
				source.Body[body+4] = 91
				source.Body[body+16] = 7
				binary.LittleEndian.PutUint16(source.Body[body+23:], 0x1234)
				source.Body[body+37] = 0xab
				binary.LittleEndian.PutUint16(source.Body[body+60:], 7)
				binary.LittleEndian.PutUint16(source.Body[body+62:], 31)
				binary.LittleEndian.PutUint16(source.Body[body+64:], 0x5678)
				source.Body[body+66] = 0x9a
				raw = source.Marshal()
				path = filepath.Join(t.TempDir(), filepath.Base(path))
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			wants, links, err := buildings1145Expected(source)
			if err != nil {
				t.Fatal(err)
			}
			if len(wants) != 11 || len(links) == 0 {
				t.Fatalf("empty or changed witness population: %d/%d", len(wants), len(links))
			}
			check := func(w *sim.World) {
				t.Helper()
				sources, cells, present := w.SavedStructures()
				if differences := buildings1145Differences(wants, links, w.Structures(), sources, cells, present); len(differences) != 0 {
					t.Fatal(differences)
				}
			}
			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			check(ms.World)
			f.SetDeterministicFrames(true)
			app := f.App("1145 Building acceptance")
			app.Layout(1024, 768)
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			check(f.live.world)
			driver := f.live
			fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
			check(fresh.live.world)
			initial := driver.world.Tick()
			for i := 0; i < 64; i++ {
				driver.tick()
				fresh.live.tick()
				if driver.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("native continuation differs at step %d", i)
				}
			}
			if driver.world.Tick() == initial {
				t.Fatal("continuation did not advance")
			}
			check(driver.world)
			check(fresh.live.world)
			t.Logf("11 Building records and %d cell links agree on both original LOAD doors, ordinary SAVE/fresh LOAD and 64 driver ticks", len(links))
		})
	}
}
