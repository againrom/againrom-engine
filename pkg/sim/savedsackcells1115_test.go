package sim

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"reflect"
	"strings"
	"testing"
)

const sackCell1115 = uint16(0x140a)
const sackKey1115 = uint32(0x78563412)

func sackPayload1115(key uint32) [52]byte {
	p := [52]byte{0: 11, 1: 2, 3: 0xa7, 45: 9, 46: 31, 47: 32, 48: 33, 49: 34, 50: 0xa6, 51: 0xb7}
	binary.LittleEndian.PutUint32(p[16:], key)
	return p
}

func sackCellWorld1115(t *testing.T, payload *[52]byte) *World {
	t.Helper()
	w := mustWorld(t, 1115, Bounds{32, 32}, nil)
	if err := w.ImportOriginalStructures(nil, nil, nil, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	cells := []SavedActorCell{}
	if payload != nil {
		cells = append(cells, SavedActorCell{Cell: sackCell1115, Payload: *payload,
			Ground: SavedActorSlot{Key: binary.LittleEndian.Uint32(payload[4:])},
			Air:    SavedActorSlot{Key: binary.LittleEndian.Uint32(payload[8:])}})
	}
	if err := w.ImportOriginalActorMotions(nil, cells, nil); err != nil {
		t.Fatal(err)
	}
	p := new(SavedCellPlanes)
	p.Costs[0], p.Costs[5] = 255, 6
	p.Cost[sackCell1115], p.CostKnown[sackCell1115] = 37, 1
	p.Static[sackCell1115], p.Dynamic[sackCell1115], p.Height[sackCell1115] = 0x12, 0x90, 17
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	return w
}

func copySackCellSavedWorld(t *testing.T, w *World) *World {
	t.Helper()
	var out World
	if err := out.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	return &out
}

// Expected state is edited from literal outcomes, never through the Sack,
// cell creation/deletion, recompute or projection functions under test.
func literalSackPlanes1115(w *World, cost, stat, dyn, grid byte) {
	w.savedCellPlanes.Cost[sackCell1115] = cost
	w.savedCellPlanes.CostKnown[sackCell1115] = 1
	w.savedCellPlanes.Static[sackCell1115] = stat
	w.savedCellPlanes.Dynamic[sackCell1115] = dyn
	w.grid[20*32+10] = grid
	w.savedMotion.Blocks = []SavedActorBlock{{Cell: 0x140a, Dyn: dyn, Static: stat}}
}

func assertSackCellWorld1115(t *testing.T, got, want *World, before []byte, changed bool) {
	t.Helper()
	actual, expected := mustMarshal(t, got), mustMarshal(t, want)
	expectedHash, oldHash := fnv.New64a(), fnv.New64a()
	_, _ = expectedHash.Write(expected)
	_, _ = oldHash.Write(before)
	if !bytes.Equal(actual, expected) || got.Hash() != expectedHash.Sum64() {
		t.Fatalf("complete state/hash differs: got %016x want %016x; cells=%+v blocks=%+v", got.Hash(), want.Hash(), got.savedMotion.Cells, got.savedMotion.Blocks)
	}
	if bytes.Equal(actual, before) == changed || (got.Hash() == oldHash.Sum64()) == changed {
		t.Fatalf("canonical state change=%v, want %v", !bytes.Equal(actual, before), changed)
	}
	if !reflect.DeepEqual(got.structureSlots, want.structureSlots) {
		t.Fatal("native structure slot cache differs")
	}
	cold := copySackCellSavedWorld(t, got)
	if !bytes.Equal(actual, mustMarshal(t, cold)) || got.Hash() != cold.Hash() {
		t.Fatal("current cell/plane/block state changed across native LOAD")
	}
}

func TestSavedSackCell1115RegistrationExistingIsOnlySlotWrite(t *testing.T) {
	for _, unresolvedBuilding := range []bool{false, true} {
		p := sackPayload1115(0)
		p[2], p[20], p[32] = 1, 0x7d, 0x63
		if unresolvedBuilding {
			binary.LittleEndian.PutUint32(p[12:], 0xdeadbeef)
		}
		w := sackCellWorld1115(t, &p)
		// Existing reuse needs neither a known current Cost nor a resolvable
		// Building because it performs no recomputation.
		w.savedCellPlanes.CostKnown[sackCell1115] = 0
		before := mustMarshal(t, w)
		want := copySackCellSavedWorld(t, w)
		copy(want.savedMotion.Cells[0].Payload[16:20], []byte{0x12, 0x34, 0x56, 0x78})
		if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); !ok || issue != "" {
			t.Fatalf("existing registration: %v %q", ok, issue)
		}
		assertSackCellWorld1115(t, w, want, before, true)
	}
}

func TestSavedSackCell1115RegistrationRefusalAndNewRecord(t *testing.T) {
	for _, tc := range []struct {
		name    string
		slot    uint32
		dynamic byte
		missing bool
	}{
		{"same-key", sackKey1115, 0x90, false},
		{"different-key", 0x11223344, 0x90, false},
		{"bit-zero-existing", 0, 1, false},
		{"bit-zero-missing", 0, 0xff, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := sackPayload1115(tc.slot)
			ptr := &p
			if tc.missing {
				ptr = nil
			}
			w := sackCellWorld1115(t, ptr)
			w.savedCellPlanes.Dynamic[sackCell1115] = tc.dynamic
			w.refreshSavedPlaneBlocks()
			before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
			if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); ok || issue != "" {
				t.Fatalf("local refusal: %v %q", ok, issue)
			}
			assertSackCellWorld1115(t, w, want, before, false)
		})
	}
	for _, absentMotion := range []bool{false, true} {
		w := sackCellWorld1115(t, nil)
		if absentMotion {
			w.savedMotion = nil
		}
		w.grid[20*32+10] |= blockMagicWall
		before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
		want.savedMotion = &savedActorMotionState{Cells: []SavedActorCell{{Cell: 0x140a,
			Payload: [52]byte{0: 37, 1: 0x12, 16: 0x12, 17: 0x34, 18: 0x56, 19: 0x78}}}}
		want.savedStructureCells = []SavedStructureCell{{Cell: 0x140a, BaselineCost: 37, BaselineStatic: 0x12}}
		literalSackPlanes1115(want, 37, 0x32, 0x32, 6)
		if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); !ok || issue != "" {
			t.Fatalf("new registration: %v %q", ok, issue)
		}
		assertSackCellWorld1115(t, w, want, before, true)
	}
}

func TestSavedSackCell1115RemovalIgnoresSlotIdentityAndResidue(t *testing.T) {
	for _, key := range []uint32{0, sackKey1115, 0xaabbccdd} {
		p := sackPayload1115(key)
		w := sackCellWorld1115(t, &p)
		w.writeCellTail(sackCell1115, [6]byte{0, 31, 7, 8, 6, 5})
		w.grid[20*32+10] |= blockMagicWall
		before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
		want.savedMotion.Cells, want.cellTails, want.savedStructureCells = nil, nil, nil
		literalSackPlanes1115(want, 11, 0x12, 0x32, 6)
		if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
			t.Fatalf("remove slot %08x: %v %q", key, ok, issue)
		}
		assertSackCellWorld1115(t, w, want, before, true)
		if w.savedSackAtCell(sackCell1115) != 0 || len(w.CellTails()) != 0 {
			t.Fatal("deleted node or tail remains visible")
		}
	}
	w := sackCellWorld1115(t, nil)
	before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
	if ok, issue := w.removeSavedSackCell(sackCell1115); ok || issue != "" {
		t.Fatalf("missing removal: %v %q", ok, issue)
	}
	assertSackCellWorld1115(t, w, want, before, false)
}

func TestSavedSackCell1115DeletionKeepsNeighborCellsTailsAndBlocks(t *testing.T) {
	p := sackPayload1115(sackKey1115)
	w := sackCellWorld1115(t, &p)
	left := SavedActorCell{Cell: 0x1409, Payload: [52]byte{0: 17, 1: 3, 16: 0x41}}
	right := SavedActorCell{Cell: 0x140b, Payload: [52]byte{0: 19, 1: 1, 16: 0x81}}
	w.savedMotion.Cells = []SavedActorCell{left, w.savedMotion.Cells[0], right}
	w.savedStructureCells = []SavedStructureCell{
		{Cell: 0x1409, BaselineCost: 17, BaselineStatic: 3},
		{Cell: 0x140a, BaselineCost: 11, BaselineStatic: 2},
		{Cell: 0x140b, BaselineCost: 19, BaselineStatic: 1},
	}
	w.writeCellTail(0x1409, [6]byte{0, 19, 5, 7, 9, 11})
	w.writeCellTail(0x140a, [6]byte{0, 31, 7, 8, 6, 5})
	w.writeCellTail(0x140b, [6]byte{0, 23, 15, 17, 19, 21})
	w.savedCellPlanes.Cost[0x1409], w.savedCellPlanes.CostKnown[0x1409] = 97, 1
	w.savedCellPlanes.Cost[0x140b], w.savedCellPlanes.CostKnown[0x140b] = 89, 1
	w.savedCellPlanes.Static[0x1409], w.savedCellPlanes.Dynamic[0x1409] = 3, 0x41
	w.savedCellPlanes.Static[0x140b], w.savedCellPlanes.Dynamic[0x140b] = 1, 0x42
	w.syncNativeSavedPlaneGrid()
	w.refreshSavedPlaneBlocks()
	before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
	want.savedMotion.Cells = []SavedActorCell{want.savedMotion.Cells[0], want.savedMotion.Cells[2]}
	want.cellTails = []cellTail{want.cellTails[0], want.cellTails[2]}
	want.savedStructureCells = []SavedStructureCell{want.savedStructureCells[0], want.savedStructureCells[2]}
	literalSackPlanes1115(want, 11, 0x12, 0x32, 2)
	want.savedMotion.Blocks = []SavedActorBlock{
		{Cell: 0x1409, Dyn: 0x41, Static: 3},
		{Cell: 0x140a, Dyn: 0x32, Static: 0x12},
		{Cell: 0x140b, Dyn: 0x42, Static: 1},
	}
	if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
		t.Fatalf("middle-node removal: %v %q", ok, issue)
	}
	assertSackCellWorld1115(t, w, want, before, true)
	if ok, issue := w.registerSavedSackCell(sackCell1115, sackKey1115); !ok || issue != "" {
		t.Fatalf("removed-cell reuse: %v %q", ok, issue)
	}
	wantNew := [52]byte{0: 11, 1: 0x12, 16: 0x12, 17: 0x34, 18: 0x56, 19: 0x78}
	if got := w.motionCell(sackCell1115); got == nil || got.Payload != wantNew || len(w.cellTails) != 2 {
		t.Fatal("new record resurrected the deleted tail or lost its restored baselines")
	}
}

func TestSavedSackCell1115RemovalRetainersAndLayers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		offset   int
		value    byte
		stat     byte
		dyn      byte
		grid     byte
		building bool
	}{
		{"ground", 4, 0x41, 0x32, 0x72, 2, false},
		{"air", 8, 0x81, 0x32, 0xb2, 2, false},
		{"building", 12, 0x91, 0x37, 0x37, 11, true},
		{"layer-count", 2, 1, 0x32, 0x32, 2, false},
		{"operation", 44, 26, 0x32, 0x32, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := sackPayload1115(sackKey1115)
			p[tc.offset] = tc.value
			w := sackCellWorld1115(t, &p)
			if tc.building {
				source := SavedStructure{ID: 9, Class: SavedBuilding, SourceKey: 0x91, ArchiveIndex: 9, Blocking: 1,
					Position: [12]byte{10, 20}, Base52: [22]byte{14: 1, 15: 1, 18: 1}}
				if err := w.ImportOriginalStructures([]Structure{{ID: 9, Col: 10, Row: 20, Width: 1, Height: 1, Blocking: 1}}, []SavedStructure{source},
					[]SavedStructureCell{{Cell: 0x140a, BaselineCost: 11, BaselineStatic: 2, ID: 9, HasStructure: true}}, bytes.Clone(w.grid)); err != nil {
					t.Fatal(err)
				}
			}
			before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
			clear(want.savedMotion.Cells[0].Payload[16:20])
			literalSackPlanes1115(want, 11, tc.stat, tc.dyn, tc.grid)
			if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
				t.Fatalf("retained removal: %v %q", ok, issue)
			}
			assertSackCellWorld1115(t, w, want, before, true)
		})
	}
	for _, offset := range []int{20, 24, 28, 32, 36, 40} {
		p := sackPayload1115(sackKey1115)
		p[offset] = 0x51 // Deliberately nonzero layer with count zero.
		w := sackCellWorld1115(t, &p)
		before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
		want.savedMotion.Cells, want.savedStructureCells = nil, nil
		dynamic := byte(0x32)
		if offset == 32 {
			dynamic = 0x37
		}
		literalSackPlanes1115(want, 11, 0x12, dynamic, 2)
		if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
			t.Fatalf("count-zero layer %02x: %v %q", offset, ok, issue)
		}
		assertSackCellWorld1115(t, w, want, before, true)
	}
}

func TestSavedSackCell1115RemovalCurrentBitFourAndSavedBaseline(t *testing.T) {
	for _, tc := range []struct{ baseline, current, stat, dyn byte }{
		{0, 2, 0, 0x20}, {0, 0x12, 0x10, 0x30},
		{0x10, 2, 0x10, 0x30}, {0x10, 0x12, 0x10, 0x30},
		{0x20, 2, 0x20, 0x20}, {0x20, 0x12, 0x30, 0x30},
		{0x92, 2, 0x92, 0xb2}, {0x92, 0x12, 0x92, 0xb2},
	} {
		p := sackPayload1115(sackKey1115)
		p[0], p[1] = 255, tc.baseline
		w := sackCellWorld1115(t, &p)
		w.savedCellPlanes.Static[sackCell1115] = tc.current
		w.savedMotion.Blocks[0].Static = tc.current
		before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
		want.savedMotion.Cells, want.savedStructureCells = nil, nil
		grid := byte(0)
		if tc.stat == 0x92 {
			grid = 2
		}
		literalSackPlanes1115(want, 255, tc.stat, tc.dyn, grid)
		if ok, issue := w.removeSavedSackCell(sackCell1115); !ok || issue != "" {
			t.Fatalf("baseline %02x/current %02x: %v %q", tc.baseline, tc.current, ok, issue)
		}
		assertSackCellWorld1115(t, w, want, before, true)
	}
}

func TestSavedSackCell1115UnsupportedPreflightIsAtomic(t *testing.T) {
	for _, tc := range []struct {
		name, issue string
		remove      bool
		payload     bool
		edit        func(*World)
		key         uint32
	}{
		{"register-no-planes", "plane authority", false, false, func(w *World) { w.savedCellPlanes = nil }, sackKey1115},
		{"remove-no-planes", "plane authority", true, true, func(w *World) { w.savedCellPlanes = nil }, sackKey1115},
		{"zero-opaque-key", "nonzero opaque Sack key", false, false, func(*World) {}, 0},
		{"unknown-new-cost", "known current cost", false, false, func(w *World) { w.savedCellPlanes.CostKnown[sackCell1115] = 0 }, sackKey1115},
		{"unresolved-building", "unresolved Building", true, true, func(w *World) { w.savedMotion.Cells[0].Payload[12] = 0x19 }, sackKey1115},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := sackPayload1115(sackKey1115)
			ptr := &p
			if !tc.payload {
				ptr = nil
			}
			w := sackCellWorld1115(t, ptr)
			tc.edit(w)
			before, want := mustMarshal(t, w), copySackCellSavedWorld(t, w)
			var ok bool
			var issue string
			if tc.remove {
				ok, issue = w.removeSavedSackCell(sackCell1115)
			} else {
				ok, issue = w.registerSavedSackCell(sackCell1115, tc.key)
			}
			if ok || !strings.Contains(issue, tc.issue) {
				t.Fatalf("unsupported result: %v %q", ok, issue)
			}
			assertSackCellWorld1115(t, w, want, before, false)
		})
	}
	var absent *World
	if ok, issue := absent.registerSavedSackCell(sackCell1115, sackKey1115); ok || issue == "" {
		t.Fatal("nil registration did not report missing authority")
	}
	if ok, issue := absent.removeSavedSackCell(sackCell1115); ok || issue == "" {
		t.Fatal("nil removal did not report missing authority")
	}
}

func TestSavedSackCell1115AccessorUsesStaticBitFive(t *testing.T) {
	for _, present := range []bool{false, true} {
		for _, key := range []uint32{0, sackKey1115} {
			for _, stat := range []byte{2, 0x22} {
				p := sackPayload1115(key)
				ptr := &p
				if !present {
					ptr = nil
				}
				w := sackCellWorld1115(t, ptr)
				w.savedCellPlanes.Static[sackCell1115] = stat
				w.savedMotion.Blocks[0].Static = stat
				before := mustMarshal(t, w)
				expected := uint32(0)
				if present && stat == 0x22 {
					expected = key
				}
				if got := w.savedSackAtCell(sackCell1115); got != expected || !bytes.Equal(before, mustMarshal(t, w)) {
					t.Fatalf("accessor present=%v key=%08x static=%02x: %08x", present, key, stat, got)
				}
			}
		}
	}
	var absent *World
	if absent.savedSackAtCell(sackCell1115) != 0 {
		t.Fatal("nil accessor invented a Sack")
	}
}
