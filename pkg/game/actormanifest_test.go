package game

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func actorManifestWorld1111(t *testing.T, entities []sim.Entity) *sim.World {
	t.Helper()
	for i := range entities {
		entities[i].SourceBinding = sim.SourceBinding{Class: 1, ArchiveIndex: uint16(entities[i].ID + 1), TypeID: 35, TokenRow: 4, Face: 3}
		entities[i].ActorLoad = sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1, TypeID: 35}}
	}
	w, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), entities)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func actorManifestDriver1111(t *testing.T, w *sim.World, manifest *SnapshotActorManifest) *mapWorld {
	t.Helper()
	m := &alm.Map{Width: 8, Height: 8, Tiles: make([]uint16, 64), Altitudes: make([]uint8, 64)}
	art := worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	art.Name = "definition name"
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{35: art}}, worldFixtureViewer(t, m))
	mw.installActorManifest(manifest)
	return mw
}

func TestActorManifestNativeSnapshotFreshProcessAndNextMove(t *testing.T) {
	if path := os.Getenv("AGAINROM_ACTOR_MANIFEST_1111"); path != "" {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		saved, _, err := DecodeSave(payload)
		if err != nil {
			t.Fatal(err)
		}
		ms := &Mission{World: actorManifestWorld1111(t, nil)}
		if err := resumeWorld(ms, &saved, nil); err != nil {
			t.Fatal(err)
		}
		mw := actorManifestDriver1111(t, ms.World, ms.ActorManifest)
		draws := mw.entityDraws()
		if len(draws) != 1 || draws[0].ID != 17 || draws[0].Name != "saved source-only Unit" || draws[0].Art == nil ||
			draws[0].Owner != sim.SelfSlot || mw.tiers[17] != 3 || mw.view.LocalOwner() != sim.SelfSlot {
			t.Fatalf("fresh presentation/control binding: %+v tier=%v", draws, mw.tiers)
		}
		mw.enqueue(17, 3, 1)
		for i := 0; i < 32; i++ {
			mw.tick()
		}
		entity := mw.world.Entities()[0]
		if entity.ID != 17 || entity.X != 3 || entity.Y != 1 || entity.Owner != sim.SelfSlot {
			t.Fatalf("fresh-process next MapOrder: %+v", entity)
		}
		t.Log("fresh-process source-only metadata, owner, rendered class, and next MapOrder PASS")
		return
	}
	w := actorManifestWorld1111(t, []sim.Entity{{ID: 17, Class: 35, TypeID: 35, X: 1, Y: 1,
		Owner: sim.SelfSlot, HP: 9, MaxHP: 31, Speed: 100, RotationSpeed: 255}})
	manifest := &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{
		{ID: 17, Name: "saved source-only Unit"},
	}}
	mw := actorManifestDriver1111(t, w, manifest)
	mw.enqueue(17, 2, 1)
	for i := 0; i < 32; i++ {
		mw.tick()
	}
	if e := w.Entities()[0]; e.X != 2 || e.Y != 1 {
		t.Fatalf("initial MapOrder: %+v", e)
	}
	mw.mission = &missionNotices{state: &Mission{World: w, ActorManifest: manifest}}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t)}}
	f.liveDriver(mw, 10, nil)
	saved, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.ActorManifest, manifest) || saved.ActorManifest == manifest || len(saved.Party) != 0 {
		t.Fatalf("snapshot aliases manifest or promoted Unit to party: %+v", saved)
	}
	payload, err := EncodeSave(saved, label)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "actor-manifest.ags")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestActorManifestNativeSnapshotFreshProcessAndNextMove$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_ACTOR_MANIFEST_1111="+path)
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("next MapOrder PASS")) {
		t.Fatalf("fresh process: %v\n%s", err, output)
	}
	t.Log(string(output))
}

func TestActorManifestLateRefusalDoesNotPublishMetadata(t *testing.T) {
	w := actorManifestWorld1111(t, []sim.Entity{{ID: 4, HP: 9, MaxHP: 31}, {ID: 17, HP: 9, MaxHP: 31}})
	prior := &SnapshotActorManifest{Version: 1, Actors: []SnapshotActor{{ID: 4, Name: "prior"}}}
	ms := &Mission{World: w, ActorManifest: prior}
	before := w.Hash()
	for _, late := range []SnapshotActor{
		{ID: 99}, {ID: 4}, {ID: 17, Name: "bad\x00name"},
	} {
		bad := &SnapshotActorManifest{Version: 1, Actors: []SnapshotActor{{ID: 4}, late}}
		if err := restoreActorManifest(ms, bad); err == nil || ms.ActorManifest != prior || w.Hash() != before {
			t.Fatalf("late invalid binding published: %+v %v", late, err)
		}
	}
	if err := restoreActorManifest(ms, &SnapshotActorManifest{Version: 2}); err == nil || ms.ActorManifest != prior {
		t.Fatalf("future manifest version: %v", err)
	}
	if err := restoreActorManifest(ms, nil); err == nil || ms.ActorManifest != prior || w.Hash() != before {
		t.Fatalf("source world lost required manifest: %v", err)
	}
	legacy, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{{ID: 4, HP: 9, MaxHP: 31}})
	if err != nil {
		t.Fatal(err)
	}
	ms.World = legacy
	if err := restoreActorManifest(ms, nil); err != nil || ms.ActorManifest != nil {
		t.Fatalf("old no-manifest snapshot: %v", err)
	}
}

func TestActorManifestOmitsRemovedActorsWithoutAliasing(t *testing.T) {
	w := actorManifestWorld1111(t, []sim.Entity{{ID: 4, HP: 9, MaxHP: 31}})
	in := &SnapshotActorManifest{Version: 1, Actors: []SnapshotActor{{ID: 4}, {ID: 17}}}
	got, err := snapshotActorManifest(in, w)
	if err != nil || len(got.Actors) != 1 || len(in.Actors) != 2 {
		t.Fatalf("snapshot: %+v %v", got, err)
	}
	got.Actors[0].Name = "changed"
	if in.Actors[0].Name != "" {
		t.Fatal("snapshot shares caller metadata")
	}
}
