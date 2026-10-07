package game

import (
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// DIV-1186
func TestReleaseNewSack1200EmptyCell(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("new Sack with no prior document root")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	hero := f.live.mission.ids[0]
	pack, ok := f.live.world.CarriedStacks(hero)
	if !ok || len(pack) == 0 {
		t.Fatal("no carried pack to drop from")
	}
	const x, y = int32(10), int32(10)
	if groundAt(f.live.world.Sacks(), x, y) != nil {
		t.Fatalf("cell %d,%d already holds ground state in this corpus file; pick a different empty cell", x, y)
	}
	if err := f.live.world.HeadlessPlace(hero, x, y); err != nil {
		t.Fatal(err)
	}
	dropped := pack[0]
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDropCarried, Entity: hero, X: x, Y: y, Spell: 0})
	f.live.tick()
	live := groundAt(f.live.world.Sacks(), x, y)
	if live == nil {
		t.Fatal("drop onto an empty cell did not create a new Sack")
	}
	if len(live.Items) != 1 || live.Items[0] != dropped.Code {
		t.Fatalf("new Sack carries the wrong item: %+v want code %d", live, dropped.Code)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.ExportCurrentWorldSave(s, "new sack"); err != nil {
		t.Fatal("a Sack with no prior document root must not refuse current SAV export", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatal("ordinary drop onto an empty cell must reach ordinary SAV", name, err)
	}
	fresh, _ := loadSAVWindow(t, store, name)
	reloaded := groundAt(fresh.live.world.Sacks(), x, y)
	if reloaded == nil {
		t.Fatal("source-free LOAD lost the new Sack")
	}
	if len(reloaded.Items) != 1 || reloaded.Items[0] != dropped.Code || reloaded.Gold != live.Gold {
		t.Fatalf("reloaded Sack differs: %+v want %+v", *reloaded, *live)
	}
	// The next action: the fresh world keeps ticking, and a second SAVE from
	// it still succeeds -- the new Sack is resumable state, not a one-shot.
	beforeTick := fresh.live.world.Tick()
	for i := 0; i < 20; i++ {
		fresh.live.tick()
	}
	if fresh.live.world.Tick() <= beforeTick {
		t.Fatal("fresh world did not continue ticking after LOAD")
	}
	if groundAt(fresh.live.world.Sacks(), x, y) == nil {
		t.Fatal("new Sack did not survive continued play")
	}
	s2, _, err := fresh.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	secondSave, _, _ := fresh.SaveSeams(store, OriginalStore{}, nil)
	secondName, err := secondSave(true)
	if err != nil || !strings.HasSuffix(secondName, ".sav") {
		t.Fatal("second ordinary SAVE from the resumed world must also reach SAV", secondName, err)
	}
	if _, err := fresh.ExportCurrentWorldSave(s2, "second"); err != nil {
		t.Fatal("second direct export must also succeed", err)
	}
	t.Logf("Sack minted at %d,%d with no prior document root: changed SAVE, source-free LOAD kept item code %d and gold %d, next action (20 ticks and a second SAVE) both succeed", x, y, dropped.Code, live.Gold)
}
