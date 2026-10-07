package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseResavedAttackBoundaryKeepsItsCurrentAction(t *testing.T) {
	for _, fixture := range []struct{ name, hash string }{
		{"game0002-bigsack.sav", "8842212e16cbb97ad05af059bdd637df2ab777b4c7f5bb5430353b68b5312c51"},
		{"game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			_, source := groundCorpusFile(t, "2026-09-24/"+fixture.name, fixture.hash)
			f := releaseFront(t)
			a := f.App("attack boundary source")
			a.Layout(1024, 768)
			open, _, err := f.RestoreOriginal(source)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			motions, _, _, present := f.live.world.SavedActorMotions()
			if !present {
				t.Fatal("restored World has no actor motions")
			}
			byID := make(map[sim.EntityID]sim.SavedActorMotion, len(motions))
			for _, motion := range motions {
				byID[motion.Entity] = motion
			}
			ids := []sim.EntityID{182, 183, 184, 185, 186, 225}
			keys := make(map[sim.EntityID]uint32, len(ids))
			for _, id := range ids {
				e, found := f.live.world.Entity(id)
				motion, bound := byID[id]
				if !found || !bound || !e.HasAttackTarget || e.AttackPhase != sim.AttackBoundaryOne ||
					!motion.Current || motion.Active || motion.Issue != "" || motion.ActorAction != 0 ||
					e.Transit != 0 || e.Turning() || !e.Alive() || e.OffMap {
					t.Fatalf("actor %d source boundary: entity=%+v motion=%+v", id, e, motion)
				}
				if e.SourceBinding.Identity == 0 {
					t.Fatalf("actor %d has no original object key", id)
				}
				keys[id] = e.SourceBinding.Identity
			}
			_, orders, _ := f.live.world.SavedGroups()
			currentOrders := make(map[sim.EntityID]byte, len(orders))
			for _, order := range orders {
				currentOrders[order.Entity] = order.Raw[9]
			}
			for _, id := range ids {
				if progress, found := currentOrders[id]; !found || progress != 0 {
					t.Fatalf("actor %d current order progress=%d found=%t", id, progress, found)
				}
			}
			sourceDoc, err := sav.DecodeDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			for id, key := range keys {
				found := false
				for i := range sourceDoc.Objects {
					record := &sourceDoc.Objects[i]
					identity, err := savedStructureValue(record, "Identity")
					if err != nil || identity != key {
						continue
					}
					action, err := savedMotionRaw(record, "U54", 4)
					if err != nil {
						t.Fatal(err)
					}
					complete, err := savedStructureValue(record, "U136")
					if err != nil {
						t.Fatal(err)
					}
					order, err := savedMotionRaw(record, "U158", 148)
					if err != nil {
						t.Fatal(err)
					}
					if got := binary.LittleEndian.Uint32(action); got != 0 || complete != 1 || order[9] != 0 {
						t.Fatalf("actor %d original boundary U54=%d U136=%d progress=%d", id, got, complete, order[9])
					}
					found = true
					break
				}
				if !found {
					t.Fatalf("actor %d original key %d is absent", id, key)
				}
			}
			before := f.live.world.Hash()
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			written, _, err := f.playerMissionSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			if f.live.world.Hash() != before {
				t.Fatal("SAVE changed the live World")
			}
			doc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			seen := make(map[sim.EntityID]bool, len(ids))
			for i := range doc.Objects {
				record := &doc.Objects[i]
				key, err := savedStructureValue(record, "Identity")
				if err != nil {
					continue
				}
				for id, want := range keys {
					if key != want {
						continue
					}
					raw, err := savedMotionRaw(record, "U54", 4)
					if err != nil {
						t.Fatal(err)
					}
					if got := binary.LittleEndian.Uint32(raw); got != 0 {
						t.Errorf("actor %d SAV U54=%d, current motion action=0 at the first attack boundary", id, got)
					}
					seen[id] = true
				}
			}
			if len(seen) != len(ids) {
				t.Fatalf("SAV bound %d of %d attack-boundary actors", len(seen), len(ids))
			}
			g := releaseFront(t)
			b := g.App("attack boundary cold load")
			b.Layout(1024, 768)
			reopen, _, err := g.RestoreOriginal(written)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.OpenMission(reopen); err != nil {
				t.Fatal(err)
			}
			for _, id := range ids {
				got, found := g.live.world.Entity(id)
				if !found || got.AttackPhase != sim.AttackBoundaryOne {
					t.Fatalf("actor %d cold LOAD boundary: found=%t phase=%d", id, found, got.AttackPhase)
				}
			}
			if got := g.live.world.Hash(); got != before {
				t.Errorf("cold LOAD World hash %016x, source %016x", got, before)
			}
			f.LiveAdvance(1)
			g.LiveAdvance(1)
			if got, want := g.live.world.Hash(), f.live.world.Hash(); got != want {
				t.Errorf("next tick World hash %016x, source continuation %016x", got, want)
			}
		})
	}
}

func boundaryActorWire(t *testing.T, doc sav.DocumentData, key uint32) (uint32, byte, uint32) {
	t.Helper()
	for i := range doc.Objects {
		record := &doc.Objects[i]
		identity, err := savedStructureValue(record, "Identity")
		if err != nil || identity != key {
			continue
		}
		action, err := savedMotionRaw(record, "U54", 4)
		if err != nil {
			t.Fatal(err)
		}
		order, err := savedMotionRaw(record, "U158", 148)
		if err != nil {
			t.Fatal(err)
		}
		complete, err := savedStructureValue(record, "U136")
		if err != nil {
			t.Fatal(err)
		}
		return binary.LittleEndian.Uint32(action), order[9], complete
	}
	t.Fatalf("actor key %d is absent from SAV", key)
	return 0, 0, 0
}

func TestReleaseResavedArmedAttackBoundaryKeepsItsCurrentAction(t *testing.T) {
	for _, fixture := range []struct {
		rel, hash string
		id        sim.EntityID
		current   bool
		exactHash bool
	}{
		{"2026-08-30/EXP-0278-human-runtime-en/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6", 5, false, false},
		{"2026-08-30/EXP-0278-human-runtime-en/game0013.sav", "b211b9ad621a2cec38ff5a1d1f3f632542a58e1e377c8f562b25ba48ca2aa5ea", 20, true, true},
	} {
		t.Run(fixture.rel, func(t *testing.T) {
			_, source := groundCorpusFile(t, fixture.rel, fixture.hash)
			f := releaseFront(t)
			a := f.App("armed attack boundary source")
			a.Layout(1024, 768)
			open, _, err := f.RestoreOriginal(source)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			e, found := f.live.world.Entity(fixture.id)
			if !found || !e.HasAttackTarget || e.AttackPhase != sim.AttackBoundaryOne || e.SourceBinding.Identity == 0 {
				t.Fatalf("actor %d has no armed boundary: found=%t entity=%+v", fixture.id, found, e)
			}
			motions, _, _, present := f.live.world.SavedActorMotions()
			motionFound := false
			for _, motion := range motions {
				if motion.Entity != fixture.id {
					continue
				}
				if motion.ActorAction != 3 || motion.Current != fixture.current || motion.Active {
					t.Fatalf("actor %d current motion=%+v", fixture.id, motion)
				}
				motionFound = true
			}
			if !present || !motionFound {
				t.Fatalf("actor %d has no saved motion", fixture.id)
			}
			_, orders, _ := f.live.world.SavedGroups()
			orderFound := false
			for _, order := range orders {
				if order.Entity == fixture.id {
					if order.Raw[9] != 1 {
						t.Fatalf("actor %d current order progress=%d", fixture.id, order.Raw[9])
					}
					orderFound = true
				}
			}
			if !orderFound {
				t.Fatalf("actor %d has no current Group order", fixture.id)
			}
			sourceDoc, err := sav.DecodeDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			if action, progress, complete := boundaryActorWire(t, sourceDoc, e.SourceBinding.Identity); action != 3 || progress != 1 || complete != 1 {
				t.Fatalf("actor %d source U54=%d progress=%d U136=%d", fixture.id, action, progress, complete)
			}
			before := f.live.world.Hash()
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			written, _, err := f.playerMissionSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			if f.live.world.Hash() != before {
				t.Fatal("SAVE changed the live World")
			}
			writtenDoc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			if action, progress, complete := boundaryActorWire(t, writtenDoc, e.SourceBinding.Identity); action != 3 || progress != 1 || complete != 1 {
				t.Errorf("actor %d saved U54=%d progress=%d U136=%d, want 3/1/1", fixture.id, action, progress, complete)
			}
			g := releaseFront(t)
			b := g.App("armed attack boundary cold load")
			b.Layout(1024, 768)
			reopen, _, err := g.RestoreOriginal(written)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.OpenMission(reopen); err != nil {
				t.Fatal(err)
			}
			loaded, found := g.live.world.Entity(fixture.id)
			if !found || loaded.AttackPhase != sim.AttackBoundaryOne {
				t.Fatalf("actor %d cold LOAD phase=%d found=%t", fixture.id, loaded.AttackPhase, found)
			}
			loadedMotions, _, _, _ := g.live.world.SavedActorMotions()
			loadedAction := uint32(0xffffffff)
			for _, motion := range loadedMotions {
				if motion.Entity == fixture.id {
					loadedAction = motion.ActorAction
				}
			}
			if loadedAction != 3 {
				t.Errorf("actor %d cold LOAD action=%d, want 3", fixture.id, loadedAction)
			}
			if fixture.exactHash && g.live.world.Hash() != before {
				t.Errorf("cold LOAD World hash %016x, source %016x", g.live.world.Hash(), before)
			}
			f.LiveAdvance(1)
			g.LiveAdvance(1)
			actionAfterTick := func(w *sim.World) uint32 {
				t.Helper()
				motions, _, _, _ := w.SavedActorMotions()
				for _, motion := range motions {
					if motion.Entity == fixture.id {
						return motion.ActorAction
					}
				}
				t.Fatalf("actor %d motion vanished on next tick", fixture.id)
				return 0
			}
			if got, want := actionAfterTick(g.live.world), actionAfterTick(f.live.world); got != want {
				t.Errorf("actor %d next tick action=%d, source=%d", fixture.id, got, want)
			}
			if fixture.exactHash && g.live.world.Hash() != f.live.world.Hash() {
				t.Errorf("next tick World hash %016x, source %016x", g.live.world.Hash(), f.live.world.Hash())
			}
		})
	}
}
