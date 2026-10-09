package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func spell1152Check(t *testing.T, want spell1152Graph, world *sim.World, state *SnapshotSAVDocument) {
	t.Helper()
	if differences := want.typedDifferences(world.SavedSpellEffects()); len(differences) != 0 {
		t.Fatalf("typed effect state: %v", differences)
	}
	if differences := want.documentDifferences(state); len(differences) != 0 {
		t.Fatalf("retained effect document: %v", differences)
	}
}

func spell1152CheckFront(t *testing.T, want spell1152Graph, f *FrontEnd) {
	t.Helper()
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	spell1152Check(t, want, f.live.world, snapshot.SavedDocument)
}

func spell1152AppLoad(t *testing.T, f *FrontEnd, raw []byte) (*ui.App, SaveStore) {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("SpellEffect acceptance")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0018.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0018.sav")
	return app, store
}

func spell1152MenuFresh(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, front func(*testing.T) *FrontEnd) *FrontEnd {
	t.Helper()
	hash := f.live.world.Hash()
	_, nativeList, nativeLoad := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(nativeCheckpoint1170(f, store), nativeList, nativeLoad)
	t.Log("App menu with explicit native checkpoint codec")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("ordinary menu SAVE: %v %v", entries, err)
	}
	fresh := front(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("SpellEffect fresh LOAD")
	app2.Layout(1024, 768)
	save, list, load := agsSaveSeams(fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	if fresh.live.world.Hash() != hash {
		t.Fatal("fresh native LOAD changed World hash")
	}
	return fresh
}

func spell1152Continue(t *testing.T, want spell1152Graph, driver *mapWorld, f, fresh *FrontEnd) {
	t.Helper()
	start := driver.world.Tick()
	for i := 1; i <= 20; i++ {
		driver.tick()
		fresh.live.tick()
		if driver.world.Hash() != fresh.live.world.Hash() {
			a, _ := driver.world.MarshalBinary()
			b, _ := fresh.live.world.MarshalBinary()
			at := 0
			for at < len(a) && at < len(b) && a[at] == b[at] {
				at++
			}
			t.Fatalf("native continuation hash differs at step %d: ticks %d/%d, bytes %d/%d first %d, pending %v/%v", i, driver.world.Tick(), fresh.live.world.Tick(), len(a), len(b), at, driver.pending, fresh.live.pending)
		}
	}
	if driver.world.Tick() == start {
		t.Fatal("continuation did not advance")
	}
	for _, front := range []*FrontEnd{f, fresh} {
		snapshot, _, err := front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if len(front.live.world.SavedSpellEffects()) != 0 || len(snapshot.SavedDocument.Document.World.Effects) != 0 {
			t.Fatal("finished graph retained roots")
		}
	}

}

// One untagged registered witness uses SAV-EFFECTGRAPH-366's original source.
// Both EN and RU are installed consumers of the same preserved document.
func TestReleaseMilestone2SpellEffects1152(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := spell1152Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	fields, typed, refs, aliases, classes := want.population()
	if len(want.roots) != 1 || len(want.nodes) != 3 || fields != 152 || typed != 41 || refs != 3 || aliases != 0 || want.span != 220 || classes["SpellTransport"] != 1 || classes["PointEffect"] != 1 || classes["Effect_DirectDamage"] != 1 {
		t.Fatalf("SAV-EFFECTGRAPH-366 population changed: roots=%d nodes=%d fields=%d typed=%d refs=%d aliases=%d span=%d classes=%v", len(want.roots), len(want.nodes), fields, typed, refs, aliases, want.span, classes)
	}
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	spell1152Check(t, want, ms.World, ms.savedDocument)
	app, store := spell1152AppLoad(t, f, raw)
	spell1152CheckFront(t, want, f)
	driver := f.live
	fresh := spell1152MenuFresh(t, f, app, store, releaseFront)
	spell1152CheckFront(t, want, fresh)
	spell1152Continue(t, want, driver, f, fresh)
	t.Log("spell effects: 1 root, 3 objects, 3 reference slots; 41 typed member bytes and all 152 source field bytes (111 Token bytes) in retained Document agree through both original LOAD doors, ordinary menu SAVE/fresh FrontEnd LOAD, and 20 continuation hashes")
	t.Log("spell effects: the target was already dead; delivery and graphics retire without replay. Synthetic DAG controls cover shared nodes; this original fixture has no alias")
}
