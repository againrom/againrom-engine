package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentBuildingRootFixture(t *testing.T, completeCell bool) (*FrontEnd, Snapshot, *sim.World, []sim.Structure, []sim.SavedStructure) {
	t.Helper()
	f, snapshot, world := partialCurrentGraph(t)
	var err error
	snapshot.SavedDocument, err = f.materializeCurrentWorld(snapshot, world)
	if err != nil {
		t.Fatal(err)
	}
	live := []sim.Structure{
		{ID: 17, Kind: 1, Col: 18, Row: 19, Field42: 31, MaxHealth: 97, Width: 1, Height: 1, Blocking: 4, Attach: 1},
		{ID: 33, Kind: 1, Col: 18, Row: 19, Field42: 43, MaxHealth: 109, Width: 1, Height: 1, Blocking: 4},
	}
	source := []sim.SavedStructure{
		{ID: 17, Class: sim.GeneratedBuilding, SourceKey: 0x62001300, RuntimeID: 73, Kind: 1, Token0E: 1, Token18: 0x100, Blocking: 4},
		{ID: 33, Class: sim.GeneratedBuilding, SourceKey: 0x62001700, RuntimeID: 91, Kind: 1, Token0E: 1, Token18: 0, Blocking: 4},
	}
	for i := range source {
		source[i].Position = [12]byte{18, 19, 18, 19, 77, 99, 0xab, 0xcd}
		binary.LittleEndian.PutUint32(source[i].Position[8:], snapshot.SavedDocument.Document.World.TerrainIdentity)
		source[i].Base52[14], source[i].Base52[15] = 1, 1
		binary.LittleEndian.PutUint32(source[i].Base52[18:], 4)
	}
	cells := []sim.SavedStructureCell{{Cell: 0x1312, BaselineCost: 44, BaselineStatic: 0xa8, ID: 17, HasStructure: true}}
	if err := world.ImportOriginalStructures(live, source, cells, make([]byte, int(world.Bounds().Width*world.Bounds().Height))); err != nil {
		t.Fatal(err)
	}
	if completeCell {
		cell := sim.SavedActorCell{Cell: 0x1312, Payload: [52]byte{44, 0xa8, 0, 0x63}}
		cell.Payload[50], cell.Payload[51] = 0x91, 0xa3
		if err := world.ImportOriginalActorMotions(nil, []sim.SavedActorCell{cell}, nil); err != nil {
			t.Fatal(err)
		}
	}
	return f, snapshot, world, live, source
}

func TestCurrentBuildingRootsMaterializeWithoutRetainedDonors(t *testing.T) {
	f, snapshot, world, live, source := currentBuildingRootFixture(t, true)
	before, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := f.materializeCurrentWorld(snapshot, world)
	if err != nil {
		t.Fatal("current typed Building roots could not materialize", err)
	}
	if len(completed.Document.World.Buildings) != 2 {
		t.Fatal("retained early return lost current Building roots", completed.Document.World.Buildings)
	}
	unchanged, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil || !bytes.Equal(before, unchanged) {
		t.Fatal("completion changed its retained input", err)
	}
	snapshot.World, err = world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "current Buildings without retained donors")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range source {
		var found *sav.DocumentRecordData
		for _, object := range doc.World.Buildings {
			r := &doc.Objects[object-1]
			if actorProjectionValue(t, *r, "Identity") == want.SourceKey {
				found = r
			}
		}
		if found == nil || actorProjectionValue(t, *found, "T18") != uint32(want.Token18) || actorProjectionValue(t, *found, "B42") != uint32(live[i].Field42) {
			t.Fatal("current exact Building joined another same-kind/cell object", i)
		}
		position, err := savedMotionRaw(found, "Block12", 12)
		if err != nil || !bytes.Equal(position, want.Position[:]) {
			t.Fatal("current complete position lost", i, err)
		}
	}
	foundCell := false
	for _, c := range doc.World.Cells {
		if c.Cell == 0x1312 {
			foundCell = true
			if c.Building != source[0].SourceKey || c.Cost != 44 || c.Static != 0xa8 || c.Residue03 != 0x63 || c.Residue32 != 0xa391 {
				t.Fatal("current final Building/Cost/Static lost", c)
			}
		}
	}
	if !foundCell {
		t.Fatal("represented final Building cell omitted")
	}
	cold := coldCurrentScript(t, f, raw)
	checkCurrentBuildingSources(t, cold.live.world, live, source)
	cold.live.tick()
	next, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal("following tick cannot capture current Buildings", err)
	}
	nextRaw, err := cold.ExportCurrentSave(next, "next current Building save")
	if err != nil {
		t.Fatal(err)
	}
	checkCurrentBuildingSources(t, coldCurrentScript(t, cold, nextRaw).live.world, live, source)
}

func checkCurrentBuildingSources(t *testing.T, world *sim.World, live []sim.Structure, source []sim.SavedStructure) {
	t.Helper()
	got, cells, present := world.SavedStructures()
	if !present || len(got) != len(source) || len(world.Structures()) != len(live) {
		t.Fatal("cold LOAD lost current Building roster")
	}
	for i, want := range source {
		found := false
		for _, current := range got {
			if current.SourceKey != want.SourceKey {
				continue
			}
			found = true
			want.ID = current.ID
			if !reflect.DeepEqual(current, want) {
				t.Fatalf("current Building source changed: got %+v want %+v", current, want)
			}
			for _, st := range world.Structures() {
				if st.ID == current.ID {
					expected := live[i]
					expected.ID = current.ID
					if st != expected {
						t.Fatalf("current Building live fields changed: got %+v want %+v", st, expected)
					}
				}
			}
		}
		if !found {
			t.Fatal("current Building exact source key lost", want.SourceKey)
		}
	}
	if len(cells) == 0 {
		t.Fatal("current Building cells lost")
	}
}

func TestCurrentBuildingRootsInitializeUnrepresentedCellSuffix(t *testing.T) {
	f, snapshot, world, live, source := currentBuildingRootFixture(t, false)
	var err error
	snapshot.World, err = world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "current Building cell without prior payload")
	if err != nil {
		t.Fatal("absent unknown cell suffix refused current SAVE", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, cell := range doc.World.Cells {
		if cell.Cell == 0x1312 {
			found = true
			if cell.Building != source[0].SourceKey || cell.Cost != 44 || cell.Static != 0xa8 || cell.Residue03 != 0 || cell.Residue32 != 0 {
				t.Fatal("current Building fields or fresh cell initializer lost", cell)
			}
		}
	}
	if !found {
		t.Fatal("current Building cell omitted")
	}
	cold := coldCurrentScript(t, f, raw)
	checkCurrentBuildingSources(t, cold.live.world, live, source)
	cold.live.tick()
	next, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	nextRaw, err := cold.ExportCurrentSave(next, "next initialized Building cell save")
	if err != nil {
		t.Fatal(err)
	}
	checkCurrentBuildingSources(t, coldCurrentScript(t, cold, nextRaw).live.world, live, source)
}

func TestCurrentBuildingRootsRejectCrossObjectIdentityCollisionAtomically(t *testing.T) {
	f, snapshot, world, _, source := currentBuildingRootFixture(t, true)
	for i := range snapshot.SavedDocument.Document.Objects {
		record := &snapshot.SavedDocument.Document.Objects[i]
		if record.Class == "Unit" {
			mustSetValue(record, "Identity", source[0].SourceKey)
			break
		}
	}
	before, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.materializeCurrentWorld(snapshot, world)
	if err == nil || !strings.Contains(err.Error(), "cross-object Identity/This collision") {
		t.Fatal("Building key collision picked a donor or reminted the current key", err)
	}
	after, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed Building materialization mutated retained input", err)
	}
}

func TestCurrentBuildingRootsReplaceStalePositionAndCellOwnership(t *testing.T) {
	f, snapshot, world, live, source := currentBuildingRootFixture(t, false)
	var err error
	snapshot.SavedDocument, err = f.materializeCurrentWorld(snapshot, world)
	if err != nil {
		t.Fatal(err)
	}
	live[0].Col, live[0].Row, live[0].Field42, live[0].Blocking = 19, 20, 11, 0x100
	source[0].Position = [12]byte{19, 20, 19, 20, 25, 27, 0x83, 0x87}
	binary.LittleEndian.PutUint32(source[0].Position[8:], snapshot.SavedDocument.Document.World.TerrainIdentity)
	source[0].Token18, source[0].Blocking = 0, 0x100
	binary.LittleEndian.PutUint32(source[0].Base52[18:], 0x100)
	source[1].Token18 = 0xffff
	cells := []sim.SavedStructureCell{
		{Cell: 0x1312, BaselineCost: 13, BaselineStatic: 0x23},
		{Cell: 0x1413, BaselineCost: 51, BaselineStatic: 0x48, ID: 17, HasStructure: true},
	}
	if err := world.ImportOriginalStructures(live, source, cells, make([]byte, int(world.Bounds().Width*world.Bounds().Height))); err != nil {
		t.Fatal(err)
	}
	oldCell := sim.SavedActorCell{Cell: 0x1312, Payload: [52]byte{13, 0x23, 0, 0x61}}
	newCell := sim.SavedActorCell{Cell: 0x1413, Payload: [52]byte{51, 0x48, 0, 0x67}}
	if err := world.ImportOriginalActorMotions(nil, []sim.SavedActorCell{oldCell, newCell}, nil); err != nil {
		t.Fatal(err)
	}
	snapshot.World, err = world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "current Building position and cells replace stale document")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, cell := range doc.World.Cells {
		switch cell.Cell {
		case 0x1312:
			seen++
			if cell.Building != 0 || cell.Cost != 13 || cell.Static != 0x23 || cell.Residue03 != 0x61 {
				t.Fatal("explicit current cell clear lost", cell)
			}
		case 0x1413:
			seen++
			if cell.Building != source[0].SourceKey || cell.Cost != 51 || cell.Static != 0x48 || cell.Residue03 != 0x67 {
				t.Fatal("new current cell ownership lost", cell)
			}
		}
	}
	if seen != 2 {
		t.Fatal("current cleared/new cells missing", seen)
	}
	checkCurrentBuildingSources(t, coldCurrentScript(t, f, raw).live.world, live, source)
}
