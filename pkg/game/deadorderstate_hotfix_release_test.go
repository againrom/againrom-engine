package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// An actor loaded from an original SAV keeps its source order record. When
// it dies, the sim puts the entity back to guard, but the retained record
// still names the state it held alive. The SAV writer must write the dead
// actor's current state: before this fix it wrote the retained one, and LOAD
// refused its own file because defend, patrol, acquire and follow are not
// states a dead actor holds.
func TestReleaseDeadActorRetainedOrderStateSAVReloads(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	probe := releaseFront(t)
	open, _, err := probe.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := probe.App("dead order probe").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var victim, escort sim.Entity
	for _, e := range probe.live.world.Entities() {
		if e.MapUnitID != 0 || e.SourceBinding.Class == 0 || !e.Alive() || e.Domain != sim.DomainGround {
			continue
		}
		if victim.SourceBinding.Identity == 0 {
			victim = e
		} else if escort.SourceBinding.Identity == 0 && e.SourceBinding.Identity != victim.SourceBinding.Identity {
			escort = e
		}
	}
	if victim.SourceBinding.Identity == 0 || escort.SourceBinding.Identity == 0 {
		t.Fatal("fixture lacks two source-bound ground actors without a map unit")
	}
	var index uint16
	for _, a := range probe.live.mission.state.savedDocument.Actors {
		if a.EntityID == victim.ID && !a.Retired {
			index = a.ObjectIndex
		}
	}
	if index == 0 {
		t.Fatal("victim has no SAV actor record")
	}
	for name, state := range map[string]uint32{"defend": 8, "patrol": 0xa, "acquire": 0xc, "follow": 0x11} {
		t.Run(name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			record := &doc.Objects[index-1]
			u50, err := savedActorRaw(record, "U50", 4)
			if err != nil {
				t.Fatal(err)
			}
			binary.LittleEndian.PutUint32(u50, state)
			if state == 8 || state == 0x11 {
				order, err := savedActorRaw(record, "U158", 148)
				if err != nil {
					t.Fatal(err)
				}
				binary.LittleEndian.PutUint32(order[0x10:], escort.SourceBinding.Identity)
			}
			controlled, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			open, town, err := f.RestoreOriginal(controlled)
			if err != nil || town {
				t.Fatalf("controlled LOAD: town=%t %v", town, err)
			}
			if err := f.App("dead order state").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			alive := worldEntityByRuntimeID(t, f, victim.SourceBinding.RuntimeID)
			if uint32(alive.ActorState) != state {
				t.Fatalf("controlled LOAD installed state %#x, want %#x", alive.ActorState, state)
			}
			headlessDamage(t, f.live.world, alive.ID, alive.HP+16)
			dead := alive
			for i := 0; i < 300 && dead.Decay < sim.DecayBones; i++ {
				f.live.tick()
				dead = worldEntityByRuntimeID(t, f, victim.SourceBinding.RuntimeID)
			}
			if dead.Alive() || dead.Decay < sim.DecayBones {
				t.Fatalf("victim did not die: %+v", dead)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
			name, err := save(true)
			if err != nil || !IsOriginal(name) {
				t.Fatalf("ordinary mission SAVE = %q: %v", name, err)
			}
			cold := releaseFront(t)
			_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
			reopen, town, err := load(localOriginalSaveToken(name))
			if err != nil || town {
				t.Fatalf("LOAD of the engine's own SAV: town=%t %v", town, err)
			}
			if err := cold.App("dead order state reload").OpenMission(reopen); err != nil {
				t.Fatal(err)
			}
			back := worldEntityByRuntimeID(t, cold, victim.SourceBinding.RuntimeID)
			if back.Alive() || back.HP != dead.HP || back.Decay != dead.Decay || back.ActorState != dead.ActorState {
				t.Fatalf("reloaded corpse = HP %d stage %d state %#x, want HP %d stage %d state %#x",
					back.HP, back.Decay, back.ActorState, dead.HP, dead.Decay, dead.ActorState)
			}
		})
	}
}
