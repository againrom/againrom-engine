package game

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Natural original input, two installed consumers. This verifies this game's
// original LOAD/native persistence; it does not launch or claim a ROM1 runtime.
// The synthetic App witness separately exercises real GiveMoney and u32 wrap.
func TestReleaseOriginalPlayerPurses1115NativeContinuation(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(raw)
	if err != nil || source.Head.Mission != 10 {
		t.Fatal("natural source changed", err)
	}
	type purse struct{ slot, money uint32 }
	want := make(map[uint32]purse)
	for _, player := range source.Players {
		key, ok := player.Field("This")
		at, haveOffset := player.Fields.Off["Money"]
		if !ok || key == 0 || !haveOffset || at < 0 || at+4 > len(source.Body) {
			t.Fatal("natural source lacks exact Player identity/raw money")
		}
		money := binary.LittleEndian.Uint32(source.Body[at:]) ^ 0x5c073f4d
		if money != player.Money {
			t.Fatal("raw unsigned money and independent field view disagree")
		}
		if old, exists := want[key]; exists && old != (purse{uint32(player.Slot), money}) {
			t.Fatal("natural source aliases different Player values")
		}
		want[key] = purse{uint32(player.Slot), money}
	}
	if len(want) < 2 {
		t.Fatal("natural input no longer exercises scenario-owned purses")
	}
	check := func(world *sim.World, state *SnapshotSAVDocument, initial bool) {
		t.Helper()
		if state == nil || state.PlayerPurses == nil || len(state.PlayerPurses.Players) != len(want) {
			t.Fatal("natural Player purse population disappeared")
		}
		seen := make(map[uint32]bool)
		for _, record := range state.Document.Objects {
			if record.Class != "Player" {
				continue
			}
			key := actorProjectionValue(t, record, "This")
			p, exists := want[key]
			if !exists || seen[key] || actorProjectionValue(t, record, "Slot") != p.slot {
				t.Fatal("natural Player identity/slot changed", key)
			}
			seen[key] = true
			money := world.Purse(p.slot)
			if initial && money != p.money || actorProjectionValue(t, record, "Money") != money {
				t.Fatal("natural current Money was not restored/projected", key, money, p.money)
			}
		}
		if len(seen) != len(want) {
			t.Fatal("natural document lost Player objects")
		}
		for _, row := range state.PlayerPurses.Players {
			if row.Unavailable != "" {
				t.Fatal("natural distinct Player purse unexpectedly uncovered", row)
			}
		}
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	check(ms.World, ms.savedDocument, true)
	f.SetDeterministicFrames(true)
	app := f.App("natural Player purses")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	check(f.live.world, before.SavedDocument, true)
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	after, _, err := fresh.Snapshot(true)
	if err != nil || !bytes.Equal(before.World, after.World) || !reflect.DeepEqual(before.SavedDocument, after.SavedDocument) {
		t.Fatal("ordinary SAVE/fresh LOAD changed complete Player purse state", err)
	}
	check(fresh.live.world, after.SavedDocument, true)
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("natural native continuation diverged")
		}
	}
	after, _, err = fresh.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	check(fresh.live.world, after.SavedDocument, false)
	t.Logf("natural M10: %d exact Player purses, raw XOR baseline, ordinary native SAVE/fresh LOAD and next20 driver ticks", len(want))
}
