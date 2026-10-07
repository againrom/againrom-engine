package sim

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestCurrentActorIdentitiesKeepHoldingsRoutesAndScriptReferences(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 2)
	w.entities[0].Speed, w.entities[1].Speed = 256, 256
	bindOperationsPack(t, w, 0)
	w.script = mustScript(t, []ScriptCheck{{Op: ScriptCheckUnitDistance, Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true}},
		[]ScriptInstant{{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 19, HasUnit2: true}}, nil)
	w.ReserveEntityIDs([]EntityID{19, 29})
	Step(w, []Command{UseScroll(1, 0, 2), MoveTo(2, CellPoint{12, 7})})
	if len(w.scrollCasts) != 1 || len(w.routes[1]) == 0 {
		t.Fatalf("fixture needs a reserved scroll and movement route: casts=%d route=%d", len(w.scrollCasts), len(w.routes[1]))
	}
	before := mustMarshal(t, w)
	if err := w.UnmarshalBinary(before); err != nil {
		t.Fatal(err)
	}
	pack, equipment, route := w.carried[0], w.equipment[0], w.routes[0]
	targetRoute := w.routes[1]
	script := w.script
	objects := w.savedObjects
	wantActions := actionCopy(t, w.Actions())
	ids := map[EntityID]EntityID{1: 9, 2: 0, 19: 7}
	if err := wantActions.RemapActors(func(id EntityID, structure bool) (EntityID, error) {
		if structure {
			return id, nil
		}
		return ids[id], nil
	}); err != nil {
		t.Fatal(err)
	}
	wantActions.Actors[0], wantActions.Actors[1] = wantActions.Actors[1], wantActions.Actors[0]
	floor := uint64(30)
	if err := w.RestoreActorIdentities(ids, &floor); err != nil {
		t.Fatal(err)
	}
	i := indexOfEntity(w.entities, 9)
	if i != 1 || w.entities[0].ID != 0 || !reflect.DeepEqual(w.carried[i], pack) || !reflect.DeepEqual(w.equipment[i], equipment) || !reflect.DeepEqual(w.routes[i], route) || !reflect.DeepEqual(w.routes[0], targetRoute) {
		t.Fatalf("actor permutation: index=%d zero=%d pack=%t equipment=%t caster route=%t target route=%t", i, w.entities[0].ID,
			reflect.DeepEqual(w.carried[i], pack), reflect.DeepEqual(w.equipment[i], equipment), reflect.DeepEqual(w.routes[i], route), reflect.DeepEqual(w.routes[0], targetRoute))
	}
	if !reflect.DeepEqual(w.Actions(), wantActions) {
		t.Fatal("reserved scroll or actor action endpoints did not follow the permutation")
	}
	if w.script.checks[0].Unit != 9 || w.script.checks[0].Unit2 != 0 || w.script.instants[0].Unit2 != 7 || script.checks[0].Unit != 1 || script.instants[0].Unit2 != 19 {
		t.Fatal("script references or detached script ownership changed")
	}
	if w.savedObjects.Containers[0].Owner.Entity != 9 || objects.Containers[0].Owner.Entity != 1 {
		t.Fatal("object container owner was lost or shared storage was mutated")
	}
	if next, ok := w.NextEntityID(); !ok || next != 30 {
		t.Fatal("departed actor reservation was not retained", next, ok)
	}
	if err := w.RestoreActorIdentities(map[EntityID]EntityID{9: 1, 0: 2, 7: 19}, &floor); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("inverse identity permutation changed nonidentity state")
	}
}

func TestCurrentActorIdentitiesKeepOrdinaryGraphAndDeadProvenance(t *testing.T) {
	for _, name := range []string{"transport", "spatial", "dead"} {
		t.Run(name, func(t *testing.T) {
			var w *World
			switch name {
			case "transport":
				w = transportGraphWorld(t, 3)
			case "spatial":
				w = sourceCopyWorld(t)
				w.entities[0].HasAttackTarget, w.entities[0].AttackTargetKind, w.entities[0].AttackTarget = true, AttackTargetStructure, 9
			case "dead":
				w = deadWorld(t)
				if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40), deadInput(3, 5, -10017)}); err != nil {
					t.Fatal(err)
				}
			}
			before := mustMarshal(t, w)
			floor := w.entityIDFloor
			dead := w.OriginalDeadActors()
			ids, back := map[EntityID]EntityID{}, map[EntityID]EntityID{}
			for _, id := range w.ActorIdentityReferences() {
				ids[id], back[100-id] = 100-id, id
			}
			if err := w.RestoreActorIdentities(ids, nil); err != nil {
				t.Fatal(err)
			}
			if name == "transport" && (w.savedSpellGraph.Nodes[1].Target != 98 || w.savedSpellGraph.Nodes[1].Value.PE44 != 555) {
				t.Fatal("typed transport target or raw source key was remapped incorrectly")
			}
			if name == "spatial" && (w.entities[0].ID != 93 || w.entities[0].AttackTarget != 9 || w.structures[0].ID != 9) {
				t.Fatal("actor or independent structure namespace changed")
			}
			for i, d := range w.OriginalDeadActors() {
				if d.ID != ids[dead[i].ID] || d.Source != dead[i].Source || d.Current != dead[i].Current {
					t.Fatal("dead root identity or ordinary provenance changed")
				}
			}
			if err := w.RestoreActorIdentities(back, &floor); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, mustMarshal(t, w)) {
				t.Fatal("ordinary graph changed across inverse identity permutation")
			}
		})
	}
}

func TestCurrentActorIdentitiesKeepFutureAndPendingCastOrder(t *testing.T) {
	for _, pending := range []bool{false, true} {
		a, b := spMage(0, 1, 1, 60, 100, 100, 1<<1), spMage(4, 1, 2, 90, 100, 100, 1<<1)
		target := spEnt(9, 3, 1)
		target.HP, target.MaxHP = 1000, 1000
		rule := SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 5, DamageMin: 2, DamageMax: 23, Damaging: true, TargetsUnit: true}
		native := spWorld(t, 71, []SpellRule{rule}, a, b, target)
		native.ReserveEntityIDs([]EntityID{49})
		commands := []Command{Cast(4, 9, 1), Cast(0, 9, 1)}
		if pending {
			Step(native, commands)
			if len(native.bookCasts) != 2 {
				t.Fatal("fixture needs both pending casts")
			}
		}
		cold := *native
		if err := cold.UnmarshalBinary(mustMarshal(t, native)); err != nil {
			t.Fatal(err)
		}
		if err := cold.RestoreActorIdentities(map[EntityID]EntityID{0: 100, 4: 20, 9: 11}, nil); err != nil {
			t.Fatal(err)
		}
		floor := uint64(50)
		if err := cold.RestoreActorIdentities(map[EntityID]EntityID{100: 0, 20: 4, 11: 9}, &floor); err != nil {
			t.Fatal(err)
		}
		if !pending {
			Step(native, commands)
			Step(&cold, commands)
		}
		for tick := 0; tick < 64; tick++ {
			if native.Hash() != cold.Hash() {
				t.Fatalf("pending=%t diverged at tick%d", pending, tick)
			}
			Step(native, nil)
			Step(&cold, nil)
		}
		if spAt(t, &cold, 9).HP == 1000 {
			t.Fatal("casts never delivered damage")
		}
	}
}

func TestCurrentActorIdentitiesRejectInvalidMapsAtomically(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	bindOperationsPack(t, w, 0)
	before := mustMarshal(t, w)
	badFloor := entityIDLimit + 1
	for _, tc := range []struct {
		ids   map[EntityID]EntityID
		floor *uint64
	}{
		{map[EntityID]EntityID{1: 3}, nil},
		{map[EntityID]EntityID{1: 3, 2: 3}, nil},
		{map[EntityID]EntityID{1: 3, 2: 4}, &badFloor},
	} {
		if err := w.RestoreActorIdentities(tc.ids, tc.floor); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
			t.Fatal("invalid identity map was accepted or changed state", err)
		}
	}
	exhausted := entityIDLimit
	if err := w.RestoreActorIdentities(map[EntityID]EntityID{1: 0, 2: 1}, &exhausted); err != nil {
		t.Fatal(err)
	}
	if _, ok := w.NextEntityID(); ok {
		t.Fatal("exhausted actor namespace wrapped")
	}
}

func TestCurrentActorIdentitiesKeepNextSummonAfterPolicyDecode(t *testing.T) {
	native := hlGhostWorld(t, 51, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		hlGhostTemplate(), effectMage(1, 1, 1, 1<<25), spEnt(2, 2, 1))
	native.entities[1].HP, native.entities[1].Decay = -10, DecayBones
	native.ReserveEntityIDs([]EntityID{49})
	raw, err := json.Marshal(native.CurrentPolicy())
	if err != nil {
		t.Fatal(err)
	}
	var policy CurrentWorldPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	cold := *native
	if err := cold.UnmarshalBinary(mustMarshal(t, native)); err != nil {
		t.Fatal(err)
	}
	cold.entityIDFloor = 3
	if err := cold.RestoreActorIdentities(map[EntityID]EntityID{1: 11, 2: 10}, nil); err != nil {
		t.Fatal(err)
	}
	if err := cold.RestoreActorIdentities(map[EntityID]EntityID{11: 1, 10: 2}, policy.EntityIDFloor); err != nil {
		t.Fatal(err)
	}
	for _, w := range []*World{native, &cold} {
		spRunCast(w, Cast(1, 2, 25))
		if indexOfEntity(w.entities, 50) < 0 {
			t.Fatal("next summon reused a departed identity")
		}
	}
	if native.Hash() != cold.Hash() {
		t.Fatal("next summon or its action continued differently")
	}
}
