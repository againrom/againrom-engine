package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const teleportSpell = sim.SpellID(26)

// bookRanges prints every known slot's range byte of every actor with a book.
func bookRanges(w *sim.World) string {
	var b strings.Builder
	for _, e := range w.Entities() {
		if !e.Book.HasInstances() {
			continue
		}
		fmt.Fprintf(&b, "actor %d owner %d at %d,%d mind %d skills %v:", e.ID, e.Owner, e.X, e.Y, e.Mind, e.Skill)
		for slot, v := range e.Book.Slots {
			if e.KnownSpells&(1<<uint(slot+1)) != 0 {
				fmt.Fprintf(&b, " %d=%d", slot+1, v.Range)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// savedBookRanges prints every saved book's range bytes and returns the
// Teleport byte of the one book that knows it.
func savedBookRanges(t *testing.T, raw []byte) (string, uint8) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	books, err := file.ActorSpellbooks()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	teleport, found := uint8(0), 0
	for _, book := range books {
		if !book.HasSpellbook {
			continue
		}
		fmt.Fprintf(&b, "record %d:", book.Off)
		for _, s := range book.Spells {
			fmt.Fprintf(&b, " %d=%d", s.ID, s.Range)
			if s.ID == uint8(teleportSpell) {
				teleport, found = s.Range, found+1
			}
		}
		b.WriteByte('\n')
	}
	if found != 1 {
		t.Fatalf("saved Teleport slots %d, want 1", found)
	}
	return b.String(), teleport
}

func teleportHero(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	var found []sim.Entity
	for _, e := range w.Entities() {
		if e.Book.HasInstances() && e.KnownSpells&(1<<uint(teleportSpell)) != 0 && e.Alive() {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		t.Fatalf("Teleport casters %d, want 1", len(found))
	}
	return found[0]
}

func closeGameMenu(t *testing.T, app *ui.App) {
	t.Helper()
	for i := 0; i < 3 && app.Screen() != ui.ScreenMap; i++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatal("game menu did not close", app.Screen())
	}
}

// teleportReach orders a Teleport seven cells along dx and returns the length
// of the jump: the distance from the cell where the cast paid its mana to the
// cell where the hero landed.
func teleportReach(t *testing.T, f *FrontEnd, id sim.EntityID, dx int32) (int32, string) {
	t.Helper()
	w := f.live.world
	hero, _ := w.Entity(id)
	target := sim.CellPoint{X: hero.X + 7*dx, Y: hero.Y}
	f.live.pending = append(f.live.pending, sim.CastAt(id, teleportSpell, target))
	trace := fmt.Sprintf("from %d,%d to %d,%d:", hero.X, hero.Y, target.X, target.Y)
	px, py, mana, paid := hero.X, hero.Y, hero.Mana, false
	var payX int32
	for i := 0; i < 400; i++ {
		f.LiveAdvance(1)
		e, _ := w.Entity(id)
		if e.Mana < mana && !paid {
			paid, payX = true, e.X
		}
		if e.X != px || e.Y != py || e.Mana != mana {
			trace += fmt.Sprintf(" t%d %d,%d m%d", w.Tick(), e.X, e.Y, e.Mana)
		}
		if paid && e.X != payX {
			return max(e.X-payX, payX-e.X), trace
		}
		px, py, mana = e.X, e.Y, e.Mana
	}
	t.Fatal("Teleport did not land:", trace)
	return 0, trace
}

// TestReleaseTeleportReachRightAfterOriginalLOAD loads the EN install's
// game0021 (hero Fergard: Mind 26, Astral 10, stored Teleport range 1). The
// stored byte holds until the first tick, whose book walk gives the range his
// power 6 gives, 1 + 6/3 = 3 (MAGIC-420, MAGIC-421); both casts jump 3 cells.
func TestReleaseTeleportReachRightAfterOriginalLOAD(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	path := os.Getenv("AGAINROM_TELEPORT_SAV")
	if path == "" {
		t.Skip("set AGAINROM_TELEPORT_SAV to the EN install's game0021.sav")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if sum := fmt.Sprintf("%x", sha256.Sum256(raw)); !strings.HasPrefix(sum, "7acf1d56") {
		t.Fatal("game0021 SHA-256 differs:", sum)
	}
	if _, stored := savedBookRanges(t, raw); stored != 1 {
		t.Fatalf("game0021 stores Teleport range %d, want 1", stored)
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "game0021.sav")
	w := f.live.world
	hero := teleportHero(t, w)
	t.Logf("tick %d at LOAD\n%s", w.Tick(), bookRanges(w))
	if got := hero.Book.Slots[teleportSpell-1].Range; got != 1 {
		t.Fatalf("Teleport range at LOAD %d, want the stored 1", got)
	}
	_, _, loadSAV := menuSAVE(t, f, app, OriginalStore{})
	books, stored := savedBookRanges(t, loadSAV)
	t.Logf("SAVE right after LOAD at tick %d\n%s", w.Tick(), books)
	if stored != 1 {
		t.Fatalf("SAVE right after LOAD writes Teleport range %d, want the stored 1", stored)
	}
	closeGameMenu(t, app)
	f.LiveAdvance(1)
	t.Logf("tick %d after one tick\n%s", w.Tick(), bookRanges(w))
	if hero, _ = w.Entity(hero.ID); hero.Book.Slots[teleportSpell-1].Range != 3 {
		t.Fatalf("Teleport range after the first tick %d, want 3", hero.Book.Slots[teleportSpell-1].Range)
	}
	_, _, tickSAV := menuSAVE(t, f, app, OriginalStore{})
	books, stored = savedBookRanges(t, tickSAV)
	t.Logf("SAVE after one tick\n%s", books)
	if stored != 3 {
		t.Fatalf("SAVE after one tick writes Teleport range %d, want 3", stored)
	}
	closeGameMenu(t, app)
	for cast, dx := range []int32{-1, 1} {
		reach, trace := teleportReach(t, f, hero.ID, dx)
		t.Logf("cast %d %s\n%s", cast+1, trace, bookRanges(w))
		if reach != 3 {
			t.Fatalf("Teleport %d after LOAD reached %d cells, want 3", cast+1, reach)
		}
	}
}
