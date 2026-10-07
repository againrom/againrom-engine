package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type deadContainerState struct {
	present bool
	tail    [2]uint32
}

func currentDeadContainers(t *testing.T, world *sim.World) map[uint32]deadContainerState {
	t.Helper()
	out := make(map[uint32]deadContainerState)
	for _, body := range world.OriginalDeadActors() {
		if body.Source.Identity == 0 {
			t.Fatal("dead actor has no source identity")
		}
		if _, repeated := out[body.Source.Identity]; repeated {
			t.Fatalf("repeated dead identity %#x", body.Source.Identity)
		}
		out[body.Source.Identity] = deadContainerState{body.Source.ContainerPresent, body.Source.ContainerTail}
	}
	return out
}

func requireDeadContainerWire(t *testing.T, raw []byte, want map[uint32]deadContainerState) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) < len(want) {
		t.Fatalf("dead wire population %d, expected at least %d: %v", len(dead), len(want), err)
	}
	seen := make(map[uint32]bool)
	for _, body := range dead {
		state, exists := want[body.Identity]
		if !exists {
			continue
		}
		if seen[body.Identity] || body.ContainerPresent != state.present || body.ContainerTail != state.tail {
			t.Fatalf("dead wire identity %#x container %t/%v, want %+v", body.Identity, body.ContainerPresent, body.ContainerTail, state)
		}
		seen[body.Identity] = true
	}
	if len(seen) != len(want) {
		t.Fatalf("dead wire contains %d of %d source identities", len(seen), len(want))
	}
}

func TestReleaseCurrentDeadContainerUsesCurrentWorld(t *testing.T) {
	_, source := groundCorpusFile(t, "2026-09-15/mission111-portrait-input.ags", "522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35")
	snapshot, _, err := DecodeSave(source)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SavedDocument == nil || snapshot.SavedDocument.Document == nil {
		t.Fatal("source AGS has no retained SAV document")
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, town, err := f.Restore(snapshot)
	if err != nil || town {
		t.Fatal("source AGS restore", town, err)
	}
	if err := f.App("dead container current state").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	want := currentDeadContainers(t, f.live.world)
	if len(want) != 43 {
		t.Fatalf("source dead population = %d, want 43", len(want))
	}
	absent, present, stale := 0, 0, 0
	for identity, state := range want {
		if state.present {
			present++
			if state.tail != [2]uint32{10000, 0} {
				t.Fatalf("present dead actor %#x has tail %v", identity, state.tail)
			}
		} else {
			absent++
			if state.tail != [2]uint32{} {
				t.Fatalf("absent dead actor %#x has tail %v", identity, state.tail)
			}
		}
		for _, object := range snapshot.SavedDocument.Document.Objects {
			key, err := savedStructureValue(&object, "Identity")
			if err != nil || key != identity {
				continue
			}
			flag, err := savedStructureValue(&object, "HasInventory")
			if err == nil && flag == 1 && !state.present {
				stale++
			}
		}
	}
	if absent != 42 || present != 1 || stale != 42 {
		t.Fatalf("source absent/present/stale = %d/%d/%d, want 42/1/42", absent, present, stale)
	}
	first := currentRuntimeSave(t, f)
	requireDeadContainerWire(t, first, want)
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	open, town, err = cold.RestoreOriginal(first)
	if err != nil || town {
		t.Fatal("first SAV cold restore", town, err)
	}
	if err := cold.App("dead container cold state").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	got := currentDeadContainers(t, cold.live.world)
	for identity, state := range want {
		if current, exists := got[identity]; !exists || current != state {
			t.Fatalf("cold dead actor %#x container %+v, want %+v", identity, got[identity], state)
		}
	}
	for range 8 {
		cold.live.tick()
	}
	second := currentRuntimeSave(t, cold)
	requireDeadContainerWire(t, second, want)
}
