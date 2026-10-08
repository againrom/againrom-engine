//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Loss controls for the World-built SAVE. Each loaded save is changed in six
// places the player can see: the hero's health and position, the gold, an
// item count, a kill, and that body's later corpse state. Orders go through
// the player's command queue and the World plays on. The SAVE and a cold
// LOAD of it must carry every change. The whole-World hash is not compared:
// after play the actors' native basis scalars and block bytes of the live
// World differ from the restored ones, with and without this writer.
func TestSAVWriterCensusLossControls(t *testing.T) {
	for _, name := range []string{"2026-10-07/saveorcsdontgo.sav", "2026-10-06/game0007-original-m70-boltcoming.sav"} {
		t.Run(filepath.Base(name), func(t *testing.T) {
			f := censusMissionFront(t, name)
			w := f.live.world
			hero, ok := f.live.primaryPartyID()
			if !ok {
				t.Fatal("the loaded mission names no primary hero")
			}
			h, _ := w.Entity(hero)
			var victim sim.EntityID
			found := false
			for _, e := range w.Entities() {
				if e.Alive() && e.MaxHP > 0 && e.Owner != h.Owner {
					victim, found = e.ID, true
					break
				}
			}
			if !found {
				t.Fatal("the loaded mission has no hostile actor")
			}
			items, _ := w.CarriedItems(hero)
			if len(items) == 0 {
				t.Fatal("the hero carries no item")
			}
			code := items[0].Code
			count := func(w *sim.World) int64 {
				var n int64
				stacks, _ := w.CarriedStacks(hero)
				for _, s := range stacks {
					if s.Code == code {
						n += int64(s.Count)
					}
				}
				return n
			}
			startCount, start := count(w), h
			f.live.pending = append(f.live.pending, sim.Kill(victim), sim.DropCarried(hero, 0, sim.CellPoint{X: h.X, Y: h.Y}))
			for i := 0; i < 10; i++ {
				f.live.tick()
			}
			f.live.pending = append(f.live.pending, sim.MoveTo(hero, sim.CellPoint{X: h.X + 2, Y: h.Y}))
			for i := 0; i < 390; i++ {
				f.live.tick()
			}
			if err := w.HeadlessDamage(hero, 3); err != nil {
				t.Fatal(err)
			}
			slot := h.Owner
			if !w.SetPurse(slot, w.Purse(slot)+137) {
				t.Fatal("purse slot is out of range")
			}
			read := func(w *sim.World) map[string]int64 {
				out := map[string]int64{}
				e, _ := w.Entity(hero)
				out["hero HP"], out["hero X"], out["hero Y"] = int64(e.HP), int64(e.X), int64(e.Y)
				out["gold"] = int64(w.Purse(slot))
				out["item count"] = count(w)
				e, held := w.Entity(victim)
				out["kill"] = 0
				if !held || !e.Alive() {
					out["kill"] = 1
				}
				if held {
					out["corpse HP"], out["corpse decay"] = int64(e.HP), int64(e.Decay)
				}
				return out
			}
			want := read(w)
			if want["kill"] != 1 || want["item count"] != startCount-1 || want["hero X"] == int64(start.X) && want["hero Y"] == int64(start.Y) {
				t.Fatalf("the orders did not change the World: %v (item count was %d)", want, startCount)
			}
			raw, err := censusMissionSave(t, f)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "loss.sav")
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			cold := loadAreaContinuation(t, path)
			got := read(cold.live.world)
			for k, v := range want {
				if got[k] != v {
					t.Errorf("%s: changed %d, cold LOAD %d", k, v, got[k])
				}
			}
			t.Logf("carried through SAVE and cold LOAD: %v", got)
		})
	}
}

// The World holds the dead units of an engine-written mission: every Unit and
// Weapon record and every dead root of the loaded file is written again, and
// a cold LOAD of the SAVE restores the same World.
func TestSAVWriterCensusDeadUnits(t *testing.T) {
	name := "2026-10-07/saveorcsdontgo.sav"
	census := func(raw []byte) [3]int {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		var out [3]int
		for _, r := range doc.Objects {
			switch r.Class {
			case "Unit":
				out[0]++
			case "Weapon":
				out[1]++
			}
		}
		for _, root := range doc.DeadActors {
			if root != 0 {
				out[2]++
			}
		}
		return out
	}
	loaded := census(censusCorpusFile(t, name))
	f := censusMissionFront(t, name)
	hash := f.live.world.Hash()
	raw, err := censusMissionSave(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if got := census(raw); got != loaded {
		t.Fatalf("Unit, Weapon and dead-root counts %v, loaded %v", got, loaded)
	}
	path := filepath.Join(t.TempDir(), "dead.sav")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if h := loadAreaContinuation(t, path).live.world.Hash(); h != hash {
		t.Fatalf("cold LOAD World hash %x, saved World %x", h, hash)
	}
	t.Logf("Unit, Weapon and dead roots written: %v", loaded)
}
