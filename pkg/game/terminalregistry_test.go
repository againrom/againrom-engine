package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func requireTerminalWire(t *testing.T, doc sav.DocumentData) int {
	t.Helper()
	keys := map[uint32]bool{}
	for _, index := range doc.DeadActors {
		r := &doc.Objects[index-1]
		stage, _ := savedStructureValue(r, "Stage")
		health, _ := savedStructureValue(r, "Health")
		if stage < 2 || int16(health) > -10 {
			continue
		}
		key, _ := savedStructureValue(r, "Identity")
		keys[key] = true
		for _, field := range []string{"U50", "U54"} {
			value, err := savedMotionRaw(r, field, 4)
			if err != nil || binary.LittleEndian.Uint32(value) != 16 {
				t.Fatalf("terminal actor %#x %s=%x err=%v, want 10000000", key, field, value, err)
			}
		}
	}
	for _, cell := range doc.World.Cells {
		if keys[cell.GroundActor] || keys[cell.AirActor] {
			t.Fatalf("terminal actor retains ordinary cell %04x: ground=%x air=%x", cell.Cell, cell.GroundActor, cell.AirActor)
		}
	}
	return len(keys)
}

func TestTerminalRegistryOriginalBonesSaveColdLoadAndNextTick(t *testing.T) {
	f := terminalGhostSource(t)
	for _, c := range f.live.world.SavedCellRecords() {
		for _, dead := range f.live.world.OriginalDeadActors() {
			if c.Ground.Key == dead.Source.Identity || c.Air.Key == dead.Source.Identity {
				t.Fatal("original LOAD retained a terminal cell owner")
			}
		}
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentRuntimeSave(t, f)
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if requireTerminalWire(t, doc) != 1 {
			t.Fatal("ordinary document lost the retained corpse")
		}
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
		for tick := 0; tick < 3; tick++ {
			if f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatalf("terminal current state differs at cycle%d tick%d", cycle, tick)
			}
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
		}
		f = cold
	}
}

func terminalGhostSource(t *testing.T) *FrontEnd {
	t.Helper()
	runtime, hp := uint32(0xabcdef77), int16(-76)
	body := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(hp), maxHP: 30, stage: 4, human: true, runtime: &runtime}
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	f := currentRetainedRuntimeFront(t)
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{body}))
	raw = completeCurrentDeadDocument(t, f, raw)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := &doc.Objects[doc.DeadActors[0]-1]
	key, _ := savedStructureValue(r, "Identity")
	for _, field := range []string{"U50", "U54"} {
		value, err := savedMotionRaw(r, field, 4)
		if err != nil {
			t.Fatal(err)
		}
		binary.LittleEndian.PutUint32(value, 0)
	}
	for _, cell := range []uint16{0x0807, 0x0808, 0x180a} {
		doc.World.Cells = append(doc.World.Cells, sav.DocumentCellData{Cell: cell, Cost: 8, GroundActor: key})
		found := false
		for i := range doc.World.Blocks {
			if doc.World.Blocks[i].Cell == cell {
				doc.World.Blocks[i].Dyn, doc.World.Blocks[i].Static = 0x60, 0x20
				found = true
			}
		}
		if !found {
			doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: cell, Dyn: 0x60, Static: 0x20})
		}
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return openCurrentRetainedRuntime(t, f, raw)
}

func TestTerminalRegistryWirePreservesOtherOccupantsAndHealableActors(t *testing.T) {
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
	for i, tuple := range []struct {
		hp    int16
		stage uint32
	}{{-76, 4}, {0, 1}, {-1000, 1}, {5, 0}} {
		r := mustNewRecord("Unit")
		mustSetValue(&r, "Identity", uint32(i+1)*0x100)
		mustSetValue(&r, "Stage", tuple.stage)
		mustSetValue(&r, "Health", uint32(uint16(tuple.hp)))
		for _, field := range []string{"U50", "U54"} {
			value, _ := savedMotionRaw(&r, field, 4)
			binary.LittleEndian.PutUint32(value, 7)
		}
		doc.Objects = append(doc.Objects, r)
		doc.DeadActors = append(doc.DeadActors, uint16(i+1))
	}
	doc.World.Cells = []sav.DocumentCellData{
		{Cell: 0x0807, GroundActor: 0x100},
		{Cell: 0x0807, GroundActor: 0x400},
		{Cell: 0x0808, GroundActor: 0x100, AirActor: 0x200},
		{Cell: 0x0809, GroundActor: 0x300, AirActor: 0x100},
		{Cell: 0x180a, GroundActor: 0x100},
	}
	for _, cell := range []uint16{0x0807, 0x0808, 0x0809, 0x180a} {
		doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: cell, Dyn: 0xff, Static: 0x27})
	}
	if err := projectTerminalActorRegistry(&doc, nil); err != nil {
		t.Fatal(err)
	}
	if requireTerminalWire(t, doc) != 1 {
		t.Fatal("wire repair missed the terminal actor")
	}
	for i := 1; i < len(doc.Objects); i++ {
		for _, field := range []string{"U50", "U54"} {
			value, _ := savedMotionRaw(&doc.Objects[i], field, 4)
			if binary.LittleEndian.Uint32(value) != 7 {
				t.Fatal("wire repair altered a healable, dwelling or recovered action")
			}
		}
	}
	for _, b := range doc.World.Blocks {
		want := byte(0xbf)
		if b.Cell == 0x0807 {
			want = 0xff
		}
		if b.Cell == 0x0809 {
			want = 0x7f
		}
		if b.Dyn != want || b.Static != 0x27 {
			t.Fatalf("wire cell %04x flags %02x/%02x want %02x/27", b.Cell, b.Dyn, b.Static, want)
		}
	}
}
