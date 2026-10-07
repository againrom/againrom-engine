package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// Natural M31 has one gold-only Sack among four source Sacks. Both installed
// consumers run this actual import/pickup/native SAVE path; no ROM1 is launched.
func TestReleaseOriginalSackObjects1115NativeContinuation(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	defer func() {
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(after) != sha256.Sum256(raw) {
			t.Fatal("read-only source changed", err)
		}
	}()
	checkInitial := func(world *sim.World, state *SnapshotSAVDocument) sim.SavedObjectID {
		t.Helper()
		r := world.SavedObjects()
		if r == nil || len(r.Sacks) != 4 || len(r.SackRoots) != 4 || state.Objects == nil || state.Objects.Version != 2 || len(state.Objects.Sacks) != 4 || len(state.Objects.Unavailable) != 0 || len(state.Document.World.Sacks) != 4 {
			t.Fatalf("natural four-Sack population not exactly owned: %+v metadata=%+v", r, state.Objects)
		}
		var goldID sim.SavedObjectID
		for _, row := range r.Sacks {
			if row.Retired || row.ID == 0 || !slices.Contains(r.SackRoots, row.ID) {
				t.Fatal("natural source Sack lacks its live exact root", row)
			}
			native := groundAt(world.Sacks(), int32(row.Token.Position[2]), int32(row.Token.Position[3]))
			if native == nil || native.ObjectID != row.ID || native.Gold != row.Gold {
				t.Fatal("natural source Sack lacks its exact native binding", row)
			}
			if row.Token.Identity == 46891536 && row.Token.Position[2] == 13 && row.Token.Position[3] == 40 {
				if goldID != 0 || row.Gold != 200 {
					t.Fatal("ambiguous natural gold Sack key/position")
				}
				goldID = row.ID
			}
		}
		sack := groundAt(world.Sacks(), 13, 40)
		if goldID == 0 || sack == nil || sack.ObjectID != goldID || sack.Gold != 200 || len(sack.Items) != 0 || len(r.Items) == 0 || len(state.Objects.Items) != len(r.Items) {
			t.Fatal("gold Sack identity or admitted item-Sack ownership changed")
		}
		if err := validateSavedObjectBindingWorld(state, world); err != nil {
			t.Fatal(err)
		}
		return goldID
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal("low-level original LOAD", err)
	}
	checkInitial(ms.World, ms.savedDocument)
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("original LOAD", town, err)
	}
	app := f.App("natural M31 Sack ownership")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	goldID := checkInitial(f.live.world, before.SavedDocument)
	originalRegistry := f.live.world.SavedObjects()
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	initialFresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	initialSnapshot, _, err := initialFresh.Snapshot(true)
	if err != nil || !bytes.Equal(before.World, initialSnapshot.World) || !reflect.DeepEqual(before.SavedDocument, initialSnapshot.SavedDocument) {
		t.Fatal("live gold Sack changed through ordinary SAVE/fresh LOAD", err)
	}
	checkInitial(initialFresh.live.world, initialSnapshot.SavedDocument)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	// Each ordinary SAVE has its own explicit destination. Wall-clock file
	// names must not decide whether this test gets one row or two.
	store = SaveStore{Dir: t.TempDir()}
	save, list, load = f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	id := f.live.mission.ids[0]
	var owner uint32
	for _, entity := range f.live.world.Entities() {
		if entity.ID == id {
			owner = entity.Owner
		}
	}
	purse := f.live.world.Purse(owner)
	livePlaceAndWalk(t, f.live, id, 13, 40)
	f.live.takeSackFor(id)
	font := f.Font.Value()
	if font == nil {
		t.Fatal("installed pickup font is absent")
	}
	mainText, err := f.Archives.Containers.ReadFile(MainTextPath)
	if err != nil {
		t.Fatal(err)
	}
	mainLines := strings.Split(string(mainText), "\r\n")
	if len(mainLines) <= 89 || mainLines[88] == "" || mainLines[89] == "" {
		t.Fatalf("main.txt holds %d lines, want the gold words 88 and 89", len(mainLines))
	}
	goldLine := mainLines[88] + " 200 " + mainLines[89]
	requireInkedNoticeGlyphs(t, font, goldLine)
	if rows := f.live.view.MessageLines(); !reflect.DeepEqual(rows, announced(goldLine)) {
		t.Fatalf("natural gold pickup posted %+v, want the one line %q", rows, goldLine)
	}
	after, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal("post-pickup projection", err)
	}
	if f.live.world.Purse(owner) != purse+200 || groundAt(f.live.world.Sacks(), 13, 40) != nil || len(after.SavedDocument.Document.World.Sacks) != 3 || len(after.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects)-1 {
		t.Fatal("natural pickup did not change purse, ground, graph and stable binding together")
	}
	registry := f.live.world.SavedObjects()
	if len(registry.Sacks) != 4 || len(registry.SackRoots) != 3 || len(f.live.world.Sacks()) != 3 || !reflect.DeepEqual(registry.Items, originalRegistry.Items) || !reflect.DeepEqual(registry.Effects, originalRegistry.Effects) || !reflect.DeepEqual(registry.Spells, originalRegistry.Spells) {
		t.Fatal("gold pickup changed another Sack or its Item/Effect/Spell ownership")
	}
	for i, row := range registry.Sacks {
		binding := after.SavedDocument.Objects.Sacks[i]
		if binding.ID != row.ID {
			t.Fatal("Sack metadata lost native identity after graph reindex")
		}
		if row.ID == goldID {
			if !row.Retired || row.Gold != 0 || binding.ObjectIndex != 0 || slices.Contains(registry.SackRoots, goldID) {
				t.Fatal("natural gold Sack was not retired by its exact key/identity")
			}
		} else if row.Retired || binding.ObjectIndex == 0 || !slices.Contains(registry.SackRoots, row.ID) || !reflect.DeepEqual(row, originalRegistry.Sacks[i]) {
			t.Fatal("one of the other three admitted Sacks lost ownership")
		}
	}
	if err := validateSavedObjectBindingWorld(after.SavedDocument, f.live.world); err != nil {
		t.Fatal("post-pickup exact object pairing", err)
	}
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	reloaded, _, err := fresh.Snapshot(true)
	if err != nil || !bytes.Equal(after.World, reloaded.World) || !reflect.DeepEqual(after.SavedDocument, reloaded.SavedDocument) {
		t.Fatal("retired Sack changed through ordinary SAVE/fresh LOAD", err)
	}
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("natural post-retirement continuation differs")
		}
	}
	if _, _, err := fresh.Snapshot(true); err != nil {
		t.Fatal("next SAVE after actual continuation", err)
	}
	t.Log("natural M31: all four Sacks owned; exact key46891536 at13,40; +200 gold; roots4->3 and one graph object retired; other three item-Sacks remain owned; ordinary SAVE/fresh LOAD before and after, next20 ticks")
}
