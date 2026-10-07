package game

import (
	"bytes"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMilestone2Sacks1151(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	source, err := sackByteWalk(file)
	if err != nil {
		t.Fatal(err)
	}
	_, originRows, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	origins := sack1151Origins(originRows)
	check := func(front *FrontEnd, want sackByteSource, join map[uint16]uint16) Snapshot {
		t.Helper()
		snapshot, _, err := front.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		d := sacks1151DocumentDifferences(want, join, snapshot.SavedDocument.Document)
		live, gaps := sacks1151LiveDifferences(want, join, snapshot.SavedDocument.Objects, front.live.world.SavedObjects(), front.live.world.Sacks())
		if len(d)+len(live)+len(gaps) != 0 {
			t.Fatal(d, live, gaps)
		}
		return snapshot
	}
	f.SetDeterministicFrames(true)
	app := f.App("Sack acceptance")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("title LOAD door absent")
	}
	groundAppLoad(t, app, list, filepath.Base(path))
	check(f, source, origins)
	// The second ordinary App door is the in-mission game menu, not a direct
	// call to the importer. Both restore the same unmodified original file.
	groundAppLoad(t, app, list, filepath.Base(path))
	before := check(f, source, origins)
	if source.count != 4 {
		t.Fatal("natural four-Sack witness changed")
	}
	target := groundAt(f.live.world.Sacks(), 42, 39)
	if target == nil || len(target.ItemInstances) != 2 || target.Gold != 264 {
		t.Fatal("natural destination Sack changed")
	}
	var sackArchive uint16
	for _, index := range source.roots {
		token := sack1151Token(source.rows[index])
		if token.Position[2] == 42 && token.Position[3] == 39 {
			sackArchive = index
		}
	}
	if sackArchive == 0 {
		t.Fatal("raw destination Sack absent")
	}
	// Bind every original archive object to its stable native identity before
	// the action. Snapshot may reindex the DTO when the ownership edge moves.
	initialIDs := map[uint16]sim.SavedObjectID{}
	for _, group := range [][]SnapshotSAVObjectBinding{before.SavedDocument.Objects.Sacks, before.SavedDocument.Objects.Items, before.SavedDocument.Objects.Effects, before.SavedDocument.Objects.Spells} {
		for _, binding := range group {
			for archive, local := range origins {
				if local == binding.ObjectIndex {
					initialIDs[archive] = binding.ID
				}
			}
		}
	}
	currentOrigins := func(state *SnapshotSAVDocument) map[uint16]uint16 {
		byID := map[sim.SavedObjectID]uint16{}
		for _, group := range [][]SnapshotSAVObjectBinding{state.Objects.Sacks, state.Objects.Items, state.Objects.Effects, state.Objects.Spells} {
			for _, b := range group {
				byID[b.ID] = b.ObjectIndex
			}
		}
		join := map[uint16]uint16{}
		for archive, id := range initialIDs {
			join[archive] = byID[id]
		}
		return join
	}
	hero := f.live.mission.ids[0]
	pack, ok := f.live.world.CarriedStacks(hero)
	if !ok {
		t.Fatal("source party pack absent")
	}
	ordinal := -1
	var dropped sim.ItemStack
	for i, item := range pack {
		if item.Count == 1 && item.ObjectID != 0 && !slices.Contains(target.Items, item.Code) && item.Weight != 0 {
			ordinal, dropped = i, item
			break
		}
	}
	if ordinal < 0 {
		t.Fatal("natural one-unit distinct drop subject absent")
	}
	var itemArchive uint16
	for archive, id := range initialIDs {
		if id == dropped.ObjectID {
			itemArchive = archive
		}
	}
	locations, err := file.DocumentObjectLocations()
	if err != nil {
		t.Fatal(err)
	}
	r := &sack1151Reader{body: file.Body, byIndex: map[uint16]sav.DocumentObjectLocation{}, byOff: map[int]sav.DocumentObjectLocation{}, source: source}
	for _, loc := range locations {
		r.byIndex[loc.ArchiveIndex], r.byOff[loc.Off] = loc, loc
	}
	loc, ok := r.byIndex[itemArchive]
	if !ok {
		t.Fatal("dropped source Item location absent")
	}
	r.record(loc, 0)
	if r.err != nil {
		t.Fatal(r.err)
	}
	wantItem := source.item(source.rows[itemArchive], dropped.ObjectID)
	if wantItem.Count != 1 || !sack1151SameItem(dropped, wantItem) {
		t.Fatal("raw drop item values disagree before action")
	}
	// Position alone is controlled. The real command dispatcher performs the
	// item move from within its two-cell drop window into an existing Sack.
	if err := f.live.world.HeadlessPlace(hero, 41, 39); err != nil {
		t.Fatal(err)
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDropCarried, Entity: hero, X: 42, Y: 39, Spell: uint16(ordinal)})
	f.live.tick()
	sack := source.rows[sackArchive]
	oldLoad := int32(sack.values["Contents20"])
	// Current native nonmerge ground insertion records the old end index.
	// This is the engine continuation policy, not original-runtime evidence.
	sack.values["Contents1C"] = uint32(len(sack.refs["Contents"]))
	sack.values["Contents20"] = uint32(oldLoad + int32(wantItem.Weight))
	sack.refs["Contents"] = append(sack.refs["Contents"], itemArchive)
	sack.counts["Contents"]++
	// ITEM-SACK-010: the receiving Sack's own value slot is gold plus every
	// carried item's own price, recomputed on every successful mutation
	// (sim.SackTokenValue), not retained from before the drop.
	sack.values["T1C"] += uint32(wantItem.Price)
	projected, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	after := check(f, source, currentOrigins(projected.SavedDocument))
	remaining, _ := f.live.world.CarriedStacks(hero)
	for _, item := range remaining {
		if item.ObjectID == dropped.ObjectID {
			t.Fatal("drop left rival pack ownership")
		}
	}
	if len(remaining) != len(pack)-1 || len(after.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects) {
		t.Fatal("one-unit transfer changed unexpected object population")
	}
	if native := groundAt(f.live.world.Sacks(), 42, 39); native == nil || native.ObjectID != target.ObjectID {
		t.Fatal("drop reminted destination Sack")
	}
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	reloaded, _, err := fresh.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after.World, reloaded.World) || !reflect.DeepEqual(after.SavedDocument, reloaded.SavedDocument) {
		t.Fatal("menu SAVE/fresh LOAD changed Sack/item graph or World")
	}
	check(fresh, source, currentOrigins(reloaded.SavedDocument))
	initialTick := f.live.world.Tick()
	for step := 0; step < 20; step++ {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("continuation hash differs at step %d", step)
		}
	}
	if f.live.world.Tick() <= initialTick {
		t.Fatal("continuation did not advance")
	}
	continued, _, err := fresh.Snapshot(true)
	if err != nil {
		t.Fatal("subsequent SAVE projection", err)
	}
	check(fresh, source, currentOrigins(continued.SavedDocument))
	t.Logf("4 raw Sacks and all descendants agree at title and mission-menu original LOAD; Item archive %d weight %d moved to unchanged Sack %#x, load %d -> %d; menu SAVE/fresh LOAD and next20 hashes agree", itemArchive, wantItem.Weight, sack.values["Identity"], oldLoad, int32(sack.values["Contents20"]))
}
