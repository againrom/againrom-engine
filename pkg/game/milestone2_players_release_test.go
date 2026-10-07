package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseMilestone2Players1154(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-24/game9999.sav", "d954bd394473d0311e934b5358519eb73ca45773732b8070e6948346f6475b27")
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := players1154Expected(file)
	if err != nil {
		t.Fatal(err)
	}
	join, err := players1154Origins(raw)
	if err != nil {
		t.Fatal(err)
	}
	var identityBaseline *SnapshotSAVDocument
	check := func(front *FrontEnd) Snapshot {
		t.Helper()
		s, _, err := front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if s.SavedDocument == nil || s.SavedDocument.Document == nil || s.SavedDocument.Unavailable != "" {
			t.Fatal("complete Player Document unavailable")
		}
		currentJoin := join
		if identityBaseline != nil {
			currentJoin, err = players1154SnapshotOrigins(want, join, identityBaseline, s.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
		}
		d := players1154DocumentDifferences(want, currentJoin, s.SavedDocument.Document)
		l, p := players1154LiveDifferences(want, currentJoin, s.SavedDocument, front.live.world)
		if len(d)+len(l) != 0 {
			t.Fatal(d, l)
		}
		if p.playerDiaries != 1 || p.entries == 0 || p.purses < 2 {
			t.Fatal("natural Player/Diary/purse witness lost its subject", p)
		}
		if identityBaseline == nil {
			// Freeze source identity only after the original raw correspondence
			// passes. The mission grants a new pack Item during continuation;
			// canonical serialization then changes DTO-local object numbers.
			identityBaseline, err = cloneSavedDocument(s.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
		}
		return s
	}
	f.SetDeterministicFrames(true)
	app := f.App("Player acceptance")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	originals := t.TempDir()
	path := filepath.Join(originals, "players.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("title LOAD door absent")
	}
	groundAppLoad(t, app, list, filepath.Base(path))
	check(f)
	first := f.live
	groundAppLoad(t, app, list, filepath.Base(path))
	if first == f.live {
		t.Fatal("mission-menu original LOAD did not replace its driver")
	}
	before := check(f)
	// Both SAVE and the fresh FrontEnd must use retained semantic state.
	// Only this test's private copy is removed; preserved inputs stay read-only.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	after := check(fresh)
	if !bytes.Equal(before.World, after.World) || !reflect.DeepEqual(before.SavedDocument, after.SavedDocument) {
		t.Fatal("menu SAVE/fresh LOAD changed Player retention or World")
	}
	for step := 0; step < 20; step++ {
		before, freshBefore := f.live.world.Tick(), fresh.live.world.Tick()
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Tick() != before+1 || fresh.live.world.Tick() != freshBefore+1 {
			t.Fatalf("continuation did not advance exactly one tick at step %d", step)
		}
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("World hash differs after advancing step %d", step)
		}
	}
	check(f)
	check(fresh)
	t.Logf("%d raw Player root slots, %d distinct identities: title and mission-menu original LOAD, ordinary menu SAVE/fresh FrontEnd LOAD and 20 advancing equal World hashes; every retained Player scalar,32-byte tail and owned Diary rechecked separately", len(want.roots), len(want.players))
}
