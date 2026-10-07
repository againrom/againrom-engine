package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A literal preserved file carries this turn and both routes. The expected
// sequence is stated independently of the turn producer and save decoder.
func TestReleaseOriginalSavedTurnContinues1147(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("saved turn continuation")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0009.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0009.sav")
	motion := func() sim.SavedActorMotion {
		motions, _, _, _ := f.live.world.SavedActorMotions()
		m, ok := findMotion1134(motions, 3)
		if !ok {
			t.Fatal("missing original turn witness")
		}
		return m
	}
	before := motion()
	if before.Mover[0] != 28 || before.Mover[1] != 96 || before.Mover[10] != 20 || before.Mover[0x9d] != 3 || binary.LittleEndian.Uint32(before.Mover[0xa0:]) != 1 || before.Issue != "" || !before.Current {
		t.Fatalf("file's pending turn was not restored: %+v", before)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	coldApp := cold.App("uninterrupted turn control")
	coldApp.Layout(1024, 768)
	coldSave, coldList, coldLoad := nativeContinuationSeams1170(t, cold, SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	coldApp.SetSaveSeams(coldSave, coldList, coldLoad)
	groundAppLoad(t, coldApp, coldList, "game0009.sav")
	for n, facing := range []byte{48, 68, 88, 96} {
		f.live.tick()
		cold.live.tick()
		if cold.live.world.Hash() != f.live.world.Hash() {
			t.Fatal("ordinary SAVE/LOAD continuation differs from the uninterrupted game")
		}
		m := motion()
		if m.Mover[0] != facing || m.Mover[0xa4] != byte(4-n) || m.Mover[0x9d] != byte(4+n) || m.Position != before.Position || !reflect.DeepEqual(m.DynamicRoute, before.DynamicRoute) || !reflect.DeepEqual(m.StaticRoute, before.StaticRoute) {
			t.Fatalf("body update %d: facing=%d estimate=%d counter=%d", n+1, m.Mover[0], m.Mover[0xa4], m.Mover[0x9d])
		}
		if n == 1 {
			hash := f.live.world.Hash()
			for tries := 0; app.Screen() != ui.ScreenGameMenu && tries < 3; tries++ {
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := listAGS(store)
			if err != nil || len(entries) != 1 {
				t.Fatalf("ordinary SAVE: %v %v", entries, err)
			}
			groundAppLoad(t, app, list, entries[0].Name)
			if f.live.world.Hash() != hash {
				t.Fatal("ordinary mid-turn SAVE/LOAD changed the world")
			}
		}
	}
	finished := motion()
	if binary.LittleEndian.Uint32(finished.Mover[0xa0:]) != 0 || f.live.world.ActorMotionActive(3) {
		t.Fatal("turn did not finish")
	}
	e, ok := f.live.entity(3)
	if !ok || !e.HasTarget {
		t.Fatal("completed move turn has no continuation target")
	}
	x, y := e.X, e.Y
	moved := false
	for range 100 {
		f.live.tick()
		e, _ = f.live.entity(3)
		if e.X != x || e.Y != y {
			moved = true
			break
		}
	}
	if !moved {
		t.Fatal("original actor still stands after its turn completed")
	}
	t.Logf("original actor3: 28->48->68->88->96; ordinary SAVE/LOAD at68 preserves hash; route resumes at %d,%d", e.X, e.Y)
}
