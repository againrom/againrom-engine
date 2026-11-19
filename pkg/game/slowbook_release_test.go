package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// slowBookCorpus is the second game's original save corpus: the EN files
// game0005 and game0006 hold Slow (spell 29) in the Player-1 hero's book.
const slowBookCorpus = "../../../gameversions/saves/2026-10-30/rom2-gog-original"

// A LOAD of each original save whose hero's book holds Slow keeps spell 29
// in the book with the cost and range the file stores, and the hero casts it
// in headless play. EN and RU installs read the same file.
func TestReleaseSecondOriginalBookKeepsAndCastsSlow(t *testing.T) {
	if _, err := os.Stat(slowBookCorpus); err != nil {
		t.Skipf("no save corpus: %v", err)
	}
	for _, name := range []string{"game0005.sav", "game0006.sav"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(slowBookCorpus, name))
			if err != nil {
				t.Skipf("no corpus file: %v", err)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
				t.Fatal(err)
			}
			f := secondGameFront(t)
			app := f.App("second-mission")
			app.Layout(1024, 768)
			f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
			_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
			groundAppLoad(t, app, list, name)
			if f.live == nil {
				t.Fatalf("the LOAD opened no mission")
			}
			w := f.live.world
			var hero sim.Entity
			for _, e := range w.Entities() {
				if e.Owner == 1 && e.KnownSpells&(1<<29) != 0 && e.Book.HasInstances() {
					hero = e
					break
				}
			}
			if hero.KnownSpells == 0 {
				t.Fatal("no Player-1 actor holds Slow in a book")
			}
			if hero.Book.State != sim.BookPresent {
				t.Fatalf("the hero's book state %d, want an ordinary present book", hero.Book.State)
			}
			slot := *hero.Book.Slot(29)
			if slot.ManaCost == 0 || slot.Range == 0 {
				t.Fatalf("Slow's book slot %+v holds no cost or range", slot)
			}
			t.Logf("%s: hero %d Slow range %d cost %d defensive %d", name, hero.ID, slot.Range, slot.ManaCost, slot.Defensive)
			if why := w.BookSpellRefusal(hero.ID, hero.ID, 29); why != "" {
				t.Fatalf("Slow on the hero is refused: %s", why)
			}
			f.live.pending = append(f.live.pending, sim.Cast(hero.ID, hero.ID, 29))
			slowed := false
			for range 40 {
				f.live.tick()
				for _, e := range f.live.world.ActiveEffects() {
					slowed = slowed || e.Spell == 29
				}
			}
			if !slowed {
				t.Fatal("the cast of Slow left no effect on the hero")
			}
		})
	}
}
