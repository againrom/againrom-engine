package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

func currentCursorActor(t *testing.T, f *FrontEnd, runtime uint32) sim.Entity {
	t.Helper()
	for _, actor := range f.live.world.Entities() {
		if actor.SourceBinding.RuntimeID == runtime {
			return actor
		}
	}
	t.Fatalf("current owner %d missing", runtime)
	return sim.Entity{}
}

func currentCursorAcquire(t *testing.T, f *FrontEnd, runtime uint32, code uint16) {
	t.Helper()
	actor := currentCursorActor(t, f, runtime)
	currentCursorRunProgram(t, f, actor, sim.ScriptInstantAddItem, code)
}

// The controlled program is part of this fixture's installed map too. A cold
// SAV load reconstructs that same program before applying its saved latches.
func currentCursorRunProgram(t *testing.T, f *FrontEnd, actor sim.Entity, operation int32, code uint16) {
	t.Helper()
	// The installed mission actor is source-backed and may have no authored
	// map-unit row. The synthetic program binds its declared unit 91 directly
	// to this actor below, so MapUnitID is not part of this continuation setup.
	payload := make([]byte, 12+2*796+184)
	put := func(at int, value uint32) { binary.LittleEndian.PutUint32(payload[at:], value) }
	put(0, 1)
	put(4+0x40, uint32(operation))
	put(4+0x44, 1)
	put(4+0x4c, 91)
	put(4+0x74, 4)
	if operation == sim.ScriptInstantAddItem {
		put(4+0x50, uint32(code))
		put(4+0x78, 8)
	}
	put(4+796, 1)
	put(8+796+0x40, 0x10002)
	put(8+796+0x44, 1)
	put(8+796+0x74, 7)
	put(8+2*796, 1)
	put(12+2*796+0x80, 1)
	put(12+2*796+0x84, 1)
	put(12+2*796+0x98, 1)
	put(12+2*796+0xb4, 1)
	mapBytes := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: []synth.ALMUnit{{X: 5<<8 | 128, Y: 6<<8 | 128}}, Type7Payload: payload})
	binary.LittleEndian.PutUint16(rawSections1092(t, mapBytes)[6][0x40:], 91)
	m, err := alm.Open(mapBytes)
	if err != nil {
		t.Fatal(err)
	}
	program, _, err := mapload.CompileScript(m, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{91: actor.ID}})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, ScenarioArchive)}
	parameters, err := f.Archives.Containers.ReadFile(worldPrefix + "data/map.reg")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	npc, err := f.Archives.Containers.ReadFile("scenario/npc.reg")
	if err != nil {
		t.Fatal(err)
	}
	archives := [][]byte{synth.Archive([]synth.File{{Path: "10.alm", Data: mapBytes}, {Path: "npc.reg", Data: npc}})}
	if parameters != nil {
		paths = append(paths, filepath.Join(dir, WorldArchive))
		archives = append(archives, synth.Archive([]synth.File{{Path: "data/map.reg", Data: parameters}}))
	}
	for i, raw := range archives {
		if err := os.WriteFile(paths[i], raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fsys, err := vfs.Open(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fsys
	f.live.mission.state.Map = m
	installTestScript(t, f, program)
	for tick := 0; tick < 256; tick++ {
		f.live.tick()
		if f.live.world.ScriptLatched(0) {
			return
		}
	}
	t.Fatal("declared cursor program did not execute")
}

func currentWideCursorSource(t *testing.T, cursor uint32) (*FrontEnd, uint32) {
	t.Helper()
	f := itemObjectsOpen(t, false, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class == "Unit" {
				if _, ok := savedObjectRefs(r, "Inventory"); ok {
					savedObjectSetValue(r, "Inventory1C", cursor)
				}
			}
			if !savedItemClass(r.Class) {
				continue
			}
			savedObjectSetValue(r, "F4A", 0)
			identity, _ := savedStructureValue(r, "Identity")
			switch identity {
			case 0x420001, 0x420002:
				savedObjectSetValue(r, "F42", 65535)
			case 0x410001, 0x410002:
				savedObjectSetValue(r, "F42", 1)
			}
			if identity == 0x420002 || identity == 0x410002 {
				savedObjectSetValue(r, "F40", 0x0e07)
			}
		}
	})
	owner := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001).Owner.Entity
	if err := f.live.world.TakeSack(owner, 15, 16); err != nil {
		t.Fatal(err)
	}
	pack, _ := f.live.world.CarriedStacks(owner)
	if len(pack) != 2 || pack[0].Count != 65536 || pack[1].Count != 65536 {
		t.Fatalf("two native wide stacks required: %+v", pack)
	}
	for _, actor := range f.live.world.Entities() {
		if actor.ID == owner {
			return f, actor.SourceBinding.RuntimeID
		}
	}
	t.Fatal("wide owner missing")
	return nil, 0
}

func currentCursorCold(t *testing.T, doc sav.DocumentData, source ...*FrontEnd) *FrontEnd {
	t.Helper()
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := cellStateFront(t)
	if len(source) != 0 {
		cold.Archives.Containers = source[0].Archives.Containers
	}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("current count LOAD", town, err)
	}
	if err := cold.App("current native count").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return cold
}

func currentCursorRoundTrip(t *testing.T, f *FrontEnd) *FrontEnd {
	t.Helper()
	doc, _ := currentRootSAVDocument(t, f)
	cold := currentCursorCold(t, doc, f)
	if f.live.world.Hash() != cold.live.world.Hash() {
		currentItemWorldDiagnostics(t, snapshotCurrentObjects(t, f).World, snapshotCurrentObjects(t, cold).World)
		t.Fatal("current count SAVE changed exact World state")
	}
	return cold
}

func TestCurrentItemCursorKeepsNativeTopologyAndContinuation(t *testing.T) {
	for _, cursor := range []uint32{0, 1, 2, 3, 9, ^uint32(0) - 1, ^uint32(0)} {
		t.Run(fmt.Sprint(cursor), func(t *testing.T) {
			f, runtime := currentWideCursorSource(t, cursor)
			before := currentObjectOwners(t, f)
			cold := currentCursorRoundTrip(t, f)
			if got := currentObjectOwners(t, cold); !reflect.DeepEqual(before, got) {
				t.Fatal("wide projection changed holdings", firstObjectDifference(before, got))
			}
			load := currentCursorActor(t, cold, runtime).CurrentActorLoad()
			if load.Inventory.InsertIndex != cursor {
				t.Fatalf("saved cursor %d, want %d", load.Inventory.InsertIndex, cursor)
			}
			for i, code := range []uint16{0x777, 0x778, 0x779} {
				currentCursorAcquire(t, f, runtime, code)
				currentCursorAcquire(t, cold, runtime, code)
				want, got := currentObjectOwners(t, f), currentObjectOwners(t, cold)
				if !reflect.DeepEqual(want, got) {
					t.Fatalf("acquisition %d changed order: %s", i+1, firstObjectDifference(want, got))
				}
				if i == 0 {
					cold = currentCursorRoundTrip(t, cold)
				}
				if f.live.world.Hash() != cold.live.world.Hash() {
					t.Fatal("next acquisition changed exact World state", i)
				}
			}
		})
	}
}

func TestCurrentSackCursorKeepsNativeTopology(t *testing.T) {
	for _, cursor := range []uint32{1, 3, ^uint32(0) - 1, ^uint32(0)} {
		t.Run(fmt.Sprint(cursor), func(t *testing.T) {
			f, runtime := currentWideCursorSource(t, cursor)
			actor := currentCursorActor(t, f, runtime)
			currentCursorRunProgram(t, f, actor, sim.ScriptInstantDropAll, 0)
			var before sim.SavedObjectContainer
			for _, c := range f.live.world.SavedObjects().Containers {
				if c.Owner.Kind == sim.SavedOwnerSack && len(c.Items) == 2 {
					before = c
				}
			}
			if before.InsertIndex != cursor || len(before.Items) != 2 {
				t.Fatal("drop-all did not adopt the current container", before)
			}
			cold := currentCursorRoundTrip(t, f)
			var after sim.SavedObjectContainer
			for _, c := range cold.live.world.SavedObjects().Containers {
				if c.Owner == before.Owner {
					after = c
				}
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("Sack cursor/accumulator differs", before, after)
			}
			for i, id := range after.Items {
				item, ok := cold.live.world.SavedObjects().Item(id)
				if !ok || !cold.live.world.SavedObjects().HasLocation(id, sim.SavedItemLocation{Owner: after.Owner, Index: uint32(i)}) || item.Value.Count != 65536 || item.Value.Code != uint16(0x0e06+i) || item.Value.Price != 31 || item.Value.Weight != 0 {
					t.Fatalf("Sack slot %d lost current state: %+v", i, item)
				}
			}
			currentCursorRoundTrip(t, cold)
		})
	}
}
