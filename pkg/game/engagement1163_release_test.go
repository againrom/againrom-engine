package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The pinned original supplies state, target key and archive identities to
// independent Body readers. No target is selected from the imported orders.
func TestReleaseEngagement1163OriginalAndNativeContinuation(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-14/game0013.sav", "b211b9ad621a2cec38ff5a1d1f3f632542a58e1e377c8f562b25ba48ca2aa5ea")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	scalars, err := unitScalarExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	movement, err := mover1160Expected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	var actorArchive, targetArchive uint16
	var targetKey uint32
	count := 0
	for _, r := range scalars.records {
		if binary.LittleEndian.Uint32(r.raw["U50"]) == 3 {
			count++
			actorArchive = r.archive
			for _, m := range movement.records {
				if m.archive == actorArchive {
					targetKey = binary.LittleEndian.Uint32(m.order[0x0c:])
				}
			}
		}
	}
	if count != 1 || targetKey == 0 {
		t.Fatal("pinned source must contain one named state3 order", count, targetKey)
	}
	for _, r := range scalars.records {
		if r.values["Identity"] == targetKey {
			if targetArchive != 0 {
				t.Fatal("source target key is ambiguous")
			}
			targetArchive = r.archive
		}
	}
	if targetArchive == 0 || targetArchive == actorArchive {
		t.Fatal("pinned target has no distinct actor archive")
	}
	f.SetDeterministicFrames(true)
	app := f.App("Saved engagement acceptance")
	app.Layout(1024, 768)
	path := filepath.Join(t.TempDir(), "engagement.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "engagement.sav")
	mover1160Initial(t, movement, f.live.mission.state)
	previous := f.live
	mover1160Menu(t, app)
	groundAppLoad(t, app, list, "engagement.sav")
	if previous == f.live {
		t.Fatal("map-menu LOAD did not replace the driver")
	}
	mover1160Initial(t, movement, f.live.mission.state)
	bind := func(archive uint16) sim.EntityID {
		var found sim.EntityID
		for _, e := range f.live.world.Entities() {
			if e.SourceBinding.ArchiveIndex == archive {
				if found != 0 {
					t.Fatal("archive collapsed into multiple native actors", archive)
				}
				found = e.ID
			}
		}
		if found == 0 {
			t.Fatal("expected original actor absent", archive)
		}
		return found
	}
	actor, target := bind(actorArchive), bind(targetArchive)
	t.Logf("raw state3 actor archive%d/native%d target key%#x archive%d/native%d", actorArchive, actor, targetKey, targetArchive, target)
	initialHash := f.live.world.Hash()
	engaged := false
	for i := 0; i < 32; i++ {
		f.live.tick()
		e := group1155Entity(t, f.live.world, actor)
		if e.HasAttackTarget {
			if e.AttackTarget != target {
				t.Fatal("wrong resumed target", e.AttackTarget, target)
			}
			engaged = true
			t.Logf("first engagement tick%d actor%d state%d cell%d,%d target%d phase%d count%d", f.live.world.Tick(), actor, e.ActorState, e.X, e.Y, e.AttackTarget, e.AttackPhase, e.AttackCountdown)
			break
		}
	}
	if !engaged {
		t.Fatal("saved engagement did not resume in32 subticks", f.live.world.SavedGroupIssues())
	}
	// tick2987, not the pre-decode 2999: the pinned actor's own raw order byte
	// (U158[9], AI-ORDER-039's ord+0x09) is already 1, and AI-ORDER-039's arm 2
	// is the only writer of that byte — it fires once, on the tick that first
	// hands an attack from the pending-order switch to the act-state machine,
	// and the same arm's dispatch is gated on ord+0x09==0, so a second firing
	// on resume is not what the order machine does with an order object that
	// already reads 1. What actually governs continuation once ord+0x09=1 is
	// actor+0x54's own state (engage, `AI-STATE-011`) and the phase/countdown
	// pair HERO-CADENCE-023 reads at actor+0x58/+0x6c — exactly the bytes
	// importSavedActorActions now restores. The pinned raw phase/countdown
	// (7/1, Relaxing with one tick owed) is one tick from HERO-CADENCE-023's
	// own "at recovery zero it sets phase 0 and completion byte" boundary, so
	// world.tick() reaching that boundary immediately on resume (2987, one
	// tick past the pinned save's own 2986) is the engagement this actor's
	// order object was already mid-way through, not a fresh one. tick2999 was
	// this suite's own value from before importSavedActorActions existed:
	// with no restored phase/countdown, the actor's engage state alone drove a
	// full fresh charge from Ready, twelve ticks later. That is the behaviour
	// of the missing decode, not a second authentic continuation of the same
	// pinned order object.
	if f.live.world.Tick() != 2987 || f.live.world.Hash() == initialHash {
		t.Fatal("authentic engagement did not begin on its next AI phase")
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := engagement1163Projection(before.SavedDocument, f.live.world, actor, target); err != nil {
		t.Fatal(err)
	}
	// Two equally corrupted Documents cannot pass this direct World check.
	bad := cloneSavedDocumentFixture(t, before.SavedDocument)
	for _, b := range bad.Actors {
		if b.EntityID == actor {
			for _, field := range bad.Document.Objects[b.ObjectIndex-1].Raw {
				if field.Name == "U158" {
					binary.LittleEndian.PutUint32(field.Bytes[0x0c:], uint32(target))
				}
			}
		}
	}
	if err := engagement1163Projection(bad, f.live.world, actor, target); err == nil {
		t.Fatal("native-ID/source-key corruption escaped")
	}
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatal("ordinary mission SAVE", entries, err)
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Source free engagement LOAD")
	app2.Layout(1024, 768)
	save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("cold LOAD lost the active attack cycle")
	}
	if err := engagement1163Projection(fresh.live.mission.state.savedDocument, fresh.live.world, actor, target); err != nil {
		t.Fatal(err)
	}
	changed := false
	cutActor := group1155Entity(t, f.live.world, actor)
	for step := 1; step <= 32; step++ {
		leftTick, rightTick := f.live.world.Tick(), fresh.live.world.Tick()
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Tick() != leftTick+1 || fresh.live.world.Tick() != rightTick+1 || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("cold continuation step%d", step)
		}
		left, right := group1155Entity(t, f.live.world, actor), group1155Entity(t, fresh.live.world, actor)
		if left != right {
			t.Fatal("paired actor differs", step)
		}
		changed = changed || left != cutActor
		a, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		b, _, err := fresh.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if left.HasAttackTarget {
			if err := engagement1163Projection(a.SavedDocument, f.live.world, actor, left.AttackTarget); err != nil {
				t.Fatal(err)
			}
			if err := engagement1163Projection(b.SavedDocument, fresh.live.world, actor, left.AttackTarget); err != nil {
				t.Fatal(err)
			}
		}
		_, lo, _ := f.live.world.SavedGroups()
		_, ro, _ := fresh.live.world.SavedGroups()
		if !reflect.DeepEqual(lo, ro) {
			t.Fatal("cold continuation order differs", step)
		}
	}
	if !changed {
		t.Fatal("continuation never advances attack state")
	}
}

func engagement1163Projection(state *SnapshotSAVDocument, w *sim.World, actor, target sim.EntityID) error {
	if state == nil || state.Document == nil {
		return fmt.Errorf("missing engagement Document")
	}
	var index uint16
	for _, b := range state.Actors {
		if b.EntityID == actor && !b.Retired {
			index = b.ObjectIndex
		}
	}
	if index == 0 || int(index) > len(state.Document.Objects) {
		return fmt.Errorf("missing actor binding")
	}
	var key uint32
	for _, e := range w.Entities() {
		if e.ID == target {
			key = e.SourceBinding.Identity
		}
	}
	if key == 0 {
		return fmt.Errorf("target has no retained key")
	}
	var order []byte
	var stateBytes []byte
	for _, field := range state.Document.Objects[index-1].Raw {
		if field.Name == "U158" {
			order = field.Bytes
		}
		if field.Name == "U50" {
			stateBytes = field.Bytes
		}
	}
	if len(order) != 148 || len(stateBytes) != 4 || binary.LittleEndian.Uint32(stateBytes) != 3 || binary.LittleEndian.Uint32(order[0x0c:]) != key {
		return fmt.Errorf("current engagement target/state differs from World")
	}
	return nil
}
