package game

import (
	"reflect"
	"strings"
	"testing"
)

func flipPurseMoney1115(t *testing.T, snapshot *Snapshot, mask uint32) {
	t.Helper()
	object := snapshot.SavedDocument.GroupBindings.Players[0].ObjectIndex
	for i := range snapshot.SavedDocument.Document.Objects[object-1].Values {
		v := &snapshot.SavedDocument.Document.Objects[object-1].Values[i]
		if v.Name == "Money" {
			v.Value ^= mask
			return
		}
	}
	t.Fatal("independent fixture has no Player Money")
}

// Failure after staged SetPurse must not leak a partially installed purse.
// The unrelated motion contradiction deliberately fails final validation.
func TestPlayerPurses1115LateImportFailureIsAtomic(t *testing.T) {
	front, before := documentSnapshotFixture(t)
	bad := before
	var err error
	bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	flipPurseMoney1115(t, &bad, 0x80000001)
	actor := &bad.SavedDocument.Document.Objects[bad.SavedDocument.Actors[0].ObjectIndex-1]
	found := false
	for i := range actor.Raw {
		if actor.Raw[i].Name == "Block12" {
			actor.Raw[i].Bytes[4] ^= 1
			found = true
		}
	}
	if !found {
		t.Fatal("fixture has no actor Position")
	}
	mission := Mission{World: front.live.world, savedDocument: before.SavedDocument}
	worldPointer, documentPointer := mission.World, mission.savedDocument
	oldHash := mission.World.Hash()
	if err := importSavedPlayerPurses(&mission, bad.SavedDocument); err == nil || !strings.Contains(err.Error(), "Position") {
		t.Fatal("fixture did not reach final motion validation after staging money", err)
	}
	if mission.World != worldPointer || mission.savedDocument != documentPointer || mission.World.Hash() != oldHash {
		t.Fatal("late failed import published a purse or replaced an owner")
	}
	after, _, err := front.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("late failed import changed the active full Snapshot", err)
	}
}

func TestPlayerPurses1115ExactOwnersSurviveGameplayGraphReindex(t *testing.T) {
	f := newGroupFront(t, 2)
	want := [2]uint32{19, 0xfffffffe}
	open, town, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 2}, want))
	if err != nil || town {
		t.Fatal("literal two-Player LOAD", town, err)
	}
	app := f.App("Player purses across GiveUnit")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := groupDocumentSnapshot(t, f)
	for range 20 {
		f.live.tick()
	}
	after := groupDocumentSnapshot(t, f)
	if !f.live.world.ScriptLatched(0) || newGroupActors1115(t, f.live.world)[newGroupA].Owner != 2 {
		t.Fatal("ordinary GiveUnit did not execute")
	}
	if reflect.DeepEqual(before.SavedDocument.GroupBindings.Players, after.SavedDocument.GroupBindings.Players) {
		t.Fatal("fixture did not move a Player document index")
	}
	check := func(f *FrontEnd, s Snapshot) {
		t.Helper()
		if s.SavedDocument.PlayerPurses == nil || !reflect.DeepEqual(before.SavedDocument.PlayerPurses, s.SavedDocument.PlayerPurses) ||
			purseDocumentMoney1115(t, s.SavedDocument) != want || [2]uint32{f.live.world.Purse(1), f.live.world.Purse(2)} != want {
			t.Fatal("graph reindex changed a Player's exact purse ownership")
		}
	}
	check(f, after)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	name := crossingMenuSave(t, app, store)
	fresh := newGroupFront(t, 2)
	freshApp := fresh.App("fresh purses after GiveUnit")
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, name)
	check(fresh, groupDocumentSnapshot(t, fresh))
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("next action differs after reindexed Player purse LOAD")
		}
	}
}
