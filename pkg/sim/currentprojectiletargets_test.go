package sim

import (
	"reflect"
	"testing"
)

func currentProjectileTargetWorld(t *testing.T, binding SourceBinding, extra ...Entity) *World {
	t.Helper()
	ents := shotPair(4)
	ents[1].SourceBinding = binding
	ents = append(ents, extra...)
	for i := range ents {
		if ents[i].SourceBinding.Class != 0 {
			ents[i].SourceBinding.ArchiveIndex = uint16(ents[i].ID)
			ents[i].SourceBinding.Identity = uint32(100 + ents[i].ID)
			ents[i].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: ents[i].SourceBinding.ActorClass()}}
		}
	}
	w := shotWorld(t, ents...)
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 3, Phases: 4}) {
		t.Fatal("fixture did not release a projectile")
	}
	return w
}

func currentProjectileTargetValues(w *World, wire uint32) map[EntityID]ActorValues {
	values := make(map[EntityID]ActorValues)
	for _, e := range w.Entities() {
		v := e.Values()
		if e.ID == 2 && v.SourceBound && e.SourceBinding.RuntimeID != wire {
			v.RuntimeID = &ActorRuntimeCoordinate{Wire: wire, Value: e.SourceBinding.RuntimeID}
		}
		values[e.ID] = v
	}
	return values
}

func TestCurrentProjectileTargetsFollowRestoredActorCoordinates(t *testing.T) {
	for _, mode := range []struct {
		name    string
		binding SourceBinding
	}{
		{"native", SourceBinding{}},
		{"source runtime", SourceBinding{Class: 1, RuntimeID: 13}},
		{"source runtime absent", SourceBinding{Class: 1}},
		{"full width source runtime", SourceBinding{Class: 1, RuntimeID: 0xfedcba98}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			const wire = uint32(138)
			want := currentProjectileTargetWorld(t, mode.binding)
			cold := currentProjectileTargetWorld(t, SourceBinding{Class: 1, RuntimeID: wire})
			if err := cold.RestoreCurrentContinuation(nil, currentProjectileTargetValues(want, wire), want.Actions(), nil); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cold.SavedProjectiles(), want.SavedProjectiles()) {
				t.Fatalf("LOAD kept wire projectile target: %+v, want %+v", cold.SavedProjectiles(), want.SavedProjectiles())
			}
			if cold.Hash() != want.Hash() {
				t.Fatalf("LOAD changed complete World: %x/%x", cold.Hash(), want.Hash())
			}
			for tick := 0; tick < 8; tick++ {
				Step(cold, nil)
				Step(want, nil)
				if cold.Hash() != want.Hash() {
					t.Fatal("projectile continuation changed", tick)
				}
			}
		})
	}
}

func TestCurrentProjectileTargetsPreserveOrdinaryEditsAndUnboundRecords(t *testing.T) {
	for _, mode := range []string{"edited actor runtime", "edited projectile target", "zero target", "detached target", "full width detached target", "unbound record"} {
		t.Run(mode, func(t *testing.T) {
			const wire = uint32(138)
			var incoming, current []Entity
			if mode == "edited projectile target" {
				third := Entity{ID: 3, X: 7, Y: 4, HP: 100, MaxHP: 100, Owner: 2,
					SourceBinding: SourceBinding{Class: 1, RuntimeID: 90}}
				incoming = []Entity{third}
				third.SourceBinding.RuntimeID = 17
				current = []Entity{third}
			}
			cold := currentProjectileTargetWorld(t, SourceBinding{Class: 1, RuntimeID: wire}, incoming...)
			want := currentProjectileTargetWorld(t, SourceBinding{Class: 1, RuntimeID: 13}, current...)
			values := currentProjectileTargetValues(want, wire)
			switch mode {
			case "edited actor runtime":
				cold.entities[1].SourceBinding.RuntimeID = 61
				cold.savedProjectiles.Items[0].ActionTarget = 61
				want.entities[1].SourceBinding.RuntimeID = 61
				want.savedProjectiles.Items[0].ActionTarget = 61
			case "edited projectile target":
				cold.savedProjectiles.Items[0].ActionTarget = 90
				want.savedProjectiles.Items[0].ActionTarget = 17
				cold.savedWorldEffects.Projectiles[0].Target = 3
				want.savedWorldEffects.Projectiles[0].Target = 3
				v := values[3]
				v.RuntimeID = &ActorRuntimeCoordinate{Wire: 90, Value: 17}
				values[3] = v
			case "zero target":
				for _, w := range []*World{cold, want} {
					w.savedProjectiles.Items[0].ActionTarget = 0
					w.savedWorldEffects.Projectiles[0].Target = 0
					w.savedWorldEffects.Projectiles[0].HasTarget = false
				}
			case "detached target", "full width detached target":
				target := int32(39)
				if mode == "full width detached target" {
					wide := uint32(0xfedcba98)
					target = int32(wide)
				}
				for _, w := range []*World{cold, want} {
					w.savedProjectiles.Items[0].ActionTarget = target
					w.savedWorldEffects.Projectiles[0].Target = 99
					w.savedWorldEffects.Projectiles[0].TargetDetached = true
				}
			case "unbound record":
				for _, w := range []*World{cold, want} {
					w.savedProjectiles.Items[0].ActionTarget = 39
					w.savedWorldEffects = nil
				}
			}
			if err := cold.RestoreCurrentContinuation(nil, values, want.Actions(), nil); err != nil {
				t.Fatal(err)
			}
			if cold.Hash() != want.Hash() || !reflect.DeepEqual(cold.SavedProjectiles(), want.SavedProjectiles()) {
				t.Fatalf("ordinary edit or unbound record changed: %+v, want %+v", cold.SavedProjectiles(), want.SavedProjectiles())
			}
		})
	}
}

func TestCurrentProjectileTargetRestoreRejectsAmbiguityAtomically(t *testing.T) {
	const wire = uint32(138)
	third := Entity{ID: 3, X: 7, Y: 4, HP: 100, MaxHP: 100, Owner: 2,
		SourceBinding: SourceBinding{Class: 1, RuntimeID: wire}}
	w := currentProjectileTargetWorld(t, SourceBinding{Class: 1, RuntimeID: wire}, third)
	values := currentProjectileTargetValues(w, wire)
	for id, native := range map[EntityID]uint32{2: 13, 3: 17} {
		v := values[id]
		v.RuntimeID = &ActorRuntimeCoordinate{Wire: wire, Value: native}
		values[id] = v
	}
	before := w.Hash()
	projectiles := w.SavedProjectiles()
	if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err == nil {
		t.Fatal("ambiguous projectile wire coordinate accepted")
	}
	if w.Hash() != before || !reflect.DeepEqual(w.SavedProjectiles(), projectiles) {
		t.Fatal("failed transaction changed the projectile or World")
	}
}

func TestCurrentProjectileTargetRestoreDoesNotLeakOnLaterFailure(t *testing.T) {
	const wire = uint32(138)
	cold := currentProjectileTargetWorld(t, SourceBinding{Class: 1, RuntimeID: wire})
	want := currentProjectileTargetWorld(t, SourceBinding{})
	policy := cold.CurrentPolicy()
	invalid := entityIDLimit + 1
	policy.EntityIDFloor = &invalid
	before := cold.Hash()
	projectiles := cold.SavedProjectiles()
	if err := cold.RestoreCurrentContinuation(&policy, currentProjectileTargetValues(want, wire), want.Actions(), nil); err == nil {
		t.Fatal("invalid later identity floor accepted")
	}
	if cold.Hash() != before || !reflect.DeepEqual(cold.SavedProjectiles(), projectiles) {
		t.Fatal("later failure leaked projectile target restoration")
	}
}
