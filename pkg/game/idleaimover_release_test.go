package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseIdleAIMoverCacheColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("idle AI mover").OpenMission(f.MissionOpener(131)); err != nil {
		t.Fatal(err)
	}
	const mapUnit = 88
	start := deadPatrolEntity(t, f, mapUnit)
	if !start.Alive() || start.Owner == sim.SelfSlot {
		t.Fatal("witness is not a living native AI", start)
	}
	f.live.pending = append(f.live.pending, sim.MoveTo(start.ID, sim.CellPoint{X: start.X - 1, Y: start.Y}))
	for n := 0; n < 256; n++ {
		f.live.tick()
		e := deadPatrolEntity(t, f, mapUnit)
		if !e.HasTarget && e.Transit == 0 && !e.Turning() && (e.X != start.X || e.Y != start.Y) {
			break
		}
	}
	idle := deadPatrolEntity(t, f, mapUnit)
	if !idle.Alive() || idle.HasTarget || idle.Transit != 0 || idle.Turning() || idle.X == idle.PostX && idle.Y == idle.PostY {
		t.Fatalf("AI did not become idle away from its post: current %d,%d post %d,%d transit %d target %t turning %t",
			idle.X, idle.Y, idle.PostX, idle.PostY, idle.Transit, idle.HasTarget, idle.Turning())
	}
	if idle.PostX != start.PostX || idle.PostY != start.PostY {
		t.Fatal("move changed the guard post", start, idle)
	}
	hash := f.live.world.Hash()
	store := SaveStore{Dir: t.TempDir()}
	name, doc := deadPatrolSave(t, f, store)
	mover, err := savedActorRaw(patrolRecordByMapUnit(t, doc, mapUnit), "U154", 180)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{byte(idle.X), byte(idle.Y), 128, 128}; !bytes.Equal(mover[0x82:0x86], want) {
		t.Fatalf("SAVE entry cache %x, want current occupied position %x; guard post %d,%d", mover[0x82:0x86], want, idle.PostX, idle.PostY)
	}
	if f.live.world.Hash() != hash {
		t.Fatal("SAVE changed the live World")
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("SAVE has no native continuation", err)
	}
	postPresent := false
	for _, a := range actions.Actions.Actors {
		if a.Entity == idle.ID {
			postPresent = a.PostX == idle.PostX && a.PostY == idle.PostY
		}
	}
	if !postPresent {
		t.Fatal("native continuation lost the independent guard post")
	}
	cold := deadPatrolLoad(t, store, name)
	restored := deadPatrolEntity(t, cold, mapUnit)
	if restored.X != idle.X || restored.Y != idle.Y || restored.PostX != idle.PostX || restored.PostY != idle.PostY || cold.live.world.Hash() != hash {
		t.Fatal("cold LOAD changed position, guard post or World hash", idle, restored)
	}
	f.live.tick()
	cold.live.tick()
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatalf("cold LOAD next tick hash %016x, live %016x", cold.live.world.Hash(), f.live.world.Hash())
	}
	t.Logf("map unit %d moved %d,%d -> %d,%d, post %d,%d; SAVE/LOAD hash %016x next tick %016x",
		mapUnit, start.X, start.Y, idle.X, idle.Y, idle.PostX, idle.PostY, hash, cold.live.world.Hash())

	t.Run("guard post loss control", func(t *testing.T) {
		for i := range actions.Actions.Actors {
			if actions.Actions.Actors[i].Entity == idle.ID {
				actions.Actions.Actors[i].PostX, actions.Actions.Actors[i].PostY = idle.X, idle.Y
			}
		}
		payload, err := json.Marshal(actions)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, payload); err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		loss := SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(loss.Dir, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		g := deadPatrolLoad(t, loss, name)
		changed := deadPatrolEntity(t, g, mapUnit)
		if changed.PostX != idle.X || changed.PostY != idle.Y || g.live.world.Hash() == hash {
			t.Fatal("guard post loss was invisible on cold LOAD", changed)
		}
		g.live.tick()
		if g.live.world.Hash() == cold.live.world.Hash() {
			t.Fatal("guard post loss was invisible after the next tick")
		}
	})
}
