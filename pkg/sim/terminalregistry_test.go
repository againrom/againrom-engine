package sim

import (
	"encoding/binary"
	"testing"
)

func terminalRegistryWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 17, Bounds{32, 32}, []Entity{
		{ID: 1, X: 15, Y: 16, HP: 10, MaxHP: 10, TokenSize: 2},
		{ID: 2, X: 17, Y: 16, HP: 10, MaxHP: 10},
	})
	w.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x1000}
	w.entities[1].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 2, Identity: 0x2000}
	w.savedMotion = &savedActorMotionState{}
	w.savedCellPlanes = &SavedCellPlanes{}
	for i, key := range []uint16{0x100f, 0x1010, 0x1011, 0x110f, 0x1110, 0x180a} {
		identity := uint32(0x1000)
		if i == 2 {
			identity = 0x2000
		}
		c := SavedActorCell{Cell: key, Ground: SavedActorSlot{Key: identity}, Air: SavedActorSlot{Key: 0x3000}}
		binary.LittleEndian.PutUint32(c.Payload[4:], identity)
		binary.LittleEndian.PutUint32(c.Payload[8:], c.Air.Key)
		w.savedMotion.Cells = append(w.savedMotion.Cells, c)
		w.savedCellRecords = append(w.savedCellRecords, SavedCellRecord{Cell: key, Ground: SavedCellActorSlot{Key: identity}, Air: SavedCellActorSlot{Key: 0x3000}})
		w.savedCellPlanes.Dynamic[key], w.savedCellPlanes.Static[key] = 0xe5, 0x25
	}
	w.refreshSavedPlaneBlocks()
	return w
}

func requireTerminalRegistry(t *testing.T, w *World) {
	t.Helper()
	for _, c := range w.savedMotion.Cells {
		want := uint32(0)
		mask := byte(0xa5)
		if c.Cell == 0x1011 {
			want, mask = 0x2000, 0xe5
		}
		if c.Ground.Key != want || binary.LittleEndian.Uint32(c.Payload[4:]) != want || c.Air.Key != 0x3000 {
			t.Fatalf("terminal actor cell %04x retained an edge or lost another occupant: %+v", c.Cell, c)
		}
		if w.savedCellPlanes.Dynamic[c.Cell] != mask || w.savedCellPlanes.Static[c.Cell] != 0x25 {
			t.Fatalf("terminal cell %04x flags=%02x/%02x, want %02x/25", c.Cell, w.savedCellPlanes.Dynamic[c.Cell], w.savedCellPlanes.Static[c.Cell], mask)
		}
	}
	for _, c := range w.savedCellRecords {
		if c.Ground.Key == 0x1000 || c.Air.Key != 0x3000 {
			t.Fatal("carried cell record retained the terminal owner")
		}
	}
}

func TestTerminalRegistryNativeTeardownClearsExactStaleFootprint(t *testing.T) {
	w := terminalRegistryWorld(t)
	Step(w, []Command{TerminalKill(1)})
	for range 64 {
		Step(w, nil)
	}
	if w.entities[0].Decay < DecayBones {
		t.Fatal("terminal command did not reach teardown")
	}
	requireTerminalRegistry(t, w)
}

func TestTerminalRegistryRestorePreservesHealableAndRecoveredActors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hp    int32
		stage DecayStage
		dwell uint16
	}{
		{"zero", 0, DecayFallen, 0},
		{"negative healable", -9, DecayFallen, 0},
		{"terminal dwell", -10, DecayFallen, 3},
		{"low health dwell", -1000, DecayFallen, 3},
		{"pending teardown", -10, DecayFallen, 0},
		{"recovered", 5, DecayNone, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := terminalRegistryWorld(t)
			w.entities[0].HP, w.entities[0].Decay, w.entities[0].Dwell = tc.hp, tc.stage, tc.dwell
			before := w.Hash()
			w.ReconcileTerminalActorRegistry(map[EntityID]uint32{1: 0x1000, 2: 0x2000})
			if w.Hash() != before {
				t.Fatal("registry repair altered a healable, dwelling or recovered actor")
			}
		})
	}
	w := terminalRegistryWorld(t)
	w.entities[0].HP, w.entities[0].Decay = -76, DecayStage(4)
	w.ReconcileTerminalActorRegistry(map[EntityID]uint32{1: 0x1000, 2: 0x2000})
	requireTerminalRegistry(t, w)
	before := w.Hash()
	w.ReconcileTerminalActorRegistry(map[EntityID]uint32{1: 0x1000, 2: 0x2000})
	if w.Hash() != before {
		t.Fatal("terminal registry repair is not idempotent")
	}
}

func TestTerminalRegistryNativeBoundSlotWithoutSourceIdentity(t *testing.T) {
	w := terminalRegistryWorld(t)
	w.entities[0].SourceBinding = SourceBinding{}
	w.savedMotion.Motions = []SavedActorMotion{{Entity: 1, Issue: "native movement supersedes original crossing", Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 128, FineY: 128}}}
	for i := range w.savedMotion.Cells {
		if w.savedMotion.Cells[i].Ground.Key == 0x1000 {
			w.savedMotion.Cells[i].Ground.Entity, w.savedMotion.Cells[i].Ground.Bound = 1, true
		}
	}
	Step(w, []Command{TerminalKill(1)})
	for range 64 {
		Step(w, nil)
	}
	requireTerminalRegistry(t, w)
	if w.savedMotion.Motions[0].ActorAction != 16 {
		t.Fatal("terminal bound motion retained an ordinary actor action")
	}
}
