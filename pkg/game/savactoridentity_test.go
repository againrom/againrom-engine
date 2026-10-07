package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func identityMission(t *testing.T) *Mission {
	t.Helper()
	w, err := sim.NewSpelledWorld(19, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 2, Y: 1, HP: 20, MaxHP: 20}, {ID: 9, X: 3, Y: 1, HP: 30, MaxHP: 30}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Mission{World: w,
		Start:         mapload.Start{IDs: []sim.EntityID{1, 2}, Roster: map[sim.EntityID]mapload.PartyMember{1: {ID: "hero"}, 2: {ID: "companion"}}},
		ActorManifest: &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: 1, Name: "hero"}, {ID: 2, Name: "companion"}, {ID: 9, Name: "unbound"}}},
		actorRegistry: &originalActorRegistry{actors: []originalActorBinding{{ID: 1}, {ID: 2}}, byOff: map[int]int{20: 0, 40: 1}},
		DeadArt:       map[sim.EntityID]uint16{9: 3},
		savedDocument: &SnapshotSAVDocument{Document: &sav.DocumentData{}, Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}},
			ActorEffects: &SnapshotSAVActorEffects{Version: 1, Rows: []SnapshotSAVActorEffect{{Entity: 2}}}},
	}
}

func TestCurrentActorIdentityBindingsPreserveZeroMissingAndMetadata(t *testing.T) {
	m := identityMission(t)
	start, manifest, roster, art, registry := m.Start.IDs, m.ActorManifest, m.Start.Roster, m.DeadArt, m.actorRegistry
	floor := uint64(30)
	a := &currentActionData{Bindings: []currentActionBinding{{ID: 9, Object: 1}, {ID: 0, Object: 2}, {ID: 2, Missing: true}},
		Policy: &sim.CurrentWorldPolicy{EntityIDFloor: &floor}}
	if err := restoreCurrentActorIdentities(m, a); err != nil {
		t.Fatal(err)
	}
	ents := m.World.Entities()
	if len(ents) != 3 || ents[0].ID != 0 || ents[0].HP != 20 || ents[1].ID != 9 || ents[1].HP != 10 || ents[2].ID != 10 || ents[2].HP != 30 {
		t.Fatal("ordinary object binding or collision-safe identity changed", ents)
	}
	if !reflect.DeepEqual(m.Start.IDs, []sim.EntityID{9, 0}) || m.Start.Roster[9].ID != "hero" || m.Start.Roster[0].ID != "companion" || m.DeadArt[10] != 3 ||
		m.ActorManifest.Actors[2].ID != 10 || m.savedDocument.ActorEffects.Rows[0].Entity != 0 || m.savedDocument.Actors[0].EntityID != 0 || m.savedDocument.Actors[0].ObjectIndex != 2 {
		t.Fatal("mission metadata no longer names its ordinary actors")
	}
	if !reflect.DeepEqual(start, []sim.EntityID{1, 2}) || manifest.Actors[0].ID != 1 || roster[1].ID != "hero" || art[9] != 3 {
		t.Fatal("detached restoration mutated shared metadata")
	}
	if bound, ok := m.actorRegistry.actor(40); !ok || bound.ID != 0 || registry.actors[1].ID != 2 {
		t.Fatal("ordinary archive lookup lost its identity or mutated the old registry")
	}
	if next, ok := m.World.NextEntityID(); !ok || next != 30 {
		t.Fatal("saved allocation floor changed", next, ok)
	}
	resolved, err := resolveCurrentActions(m, a)
	if err != nil || resolved[0] != 0 || resolved[9] != 9 || resolved[2] != 2 {
		t.Fatal("action resolution changed the restored identities", resolved, err)
	}
}

func TestCurrentActorIdentityBindingsRejectAmbiguityBeforeMutation(t *testing.T) {
	for _, bindings := range [][]currentActionBinding{
		{{ID: 7, Object: 1}, {ID: 7, Object: 2}},
		{{ID: 7, Object: 1}, {ID: 8, Object: 1}},
		{{ID: 7, Object: 1, Missing: true}},
		{{ID: 7, Object: 3}},
	} {
		m := identityMission(t)
		before := m.World.Hash()
		if err := restoreCurrentActorIdentities(m, &currentActionData{Bindings: bindings}); err == nil || m.World.Hash() != before || m.Start.IDs[0] != 1 {
			t.Fatal("invalid current binding was accepted or partially applied", bindings, err)
		}
	}
}
