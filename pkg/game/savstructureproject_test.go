package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The archive is literal, not written with a Building/projector helper. The
// actor fixture contributes only its existing Player/Group/actor programme.
// Five distinct objects cover all subclasses and another exact Building;
// the final root references the FIRST object. Native/source/archive/local IDs
// deliberately differ, including the valid native handle zero.
func structureProjection1115Fixture(t *testing.T) (sav.DocumentData, *sim.World, []sim.SavedStructure) {
	t.Helper()
	a := &poolFixtureActor{cell: 0x0a09, hp: 100, maxHP: 100}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
	f, err := sav.Open(savedContainer(body))
	if err != nil {
		t.Fatal(err)
	}
	body = slices.Clone(body[:f.World.BlocksOff-8])
	body = binary.LittleEndian.AppendUint32(body, 6)
	classes := [...]string{"Building", "Outpost", "Tavern", "Shop", "Building"}
	native := [...]sim.StructureID{0, 17, 28, 91, 123}
	archives := [...]uint16{6, 8, 10, 12, 13}
	var sources []sim.SavedStructure
	var live []sim.Structure
	for i, class := range classes {
		if i == 4 {
			body = binary.LittleEndian.AppendUint16(body, 0x8005)
		} else {
			body = binary.LittleEndian.AppendUint16(body, 0xffff)
			body = binary.LittleEndian.AppendUint16(body, 1)
			body = binary.LittleEndian.AppendUint16(body, uint16(len(class)))
			body = append(body, class...)
		}
		record := [77]byte{}
		record[0], record[1], record[2], record[3] = byte(10+i), 10, byte(10+i), 10
		record[4], record[5], record[6], record[7] = 83, 127, 0xab, 0xcd
		binary.LittleEndian.PutUint32(record[8:], 0x778899aa)
		binary.LittleEndian.PutUint32(record[12:], uint32(900+i))
		record[16] = byte(20 + i)
		binary.LittleEndian.PutUint16(record[17:], uint16(300+i))
		binary.LittleEndian.PutUint16(record[23:], uint16(400+i))
		binary.LittleEndian.PutUint32(record[25:], uint32(500+i))
		binary.LittleEndian.PutUint32(record[29:], 0xabcde000+uint32(i))
		binary.LittleEndian.PutUint32(record[33:], 0xaabbccdd)
		for j := 37; j < 59; j++ {
			record[j] = byte(0x40 + i + j - 37)
		}
		record[59] = byte(i + 1)
		binary.LittleEndian.PutUint16(record[60:], 100)
		binary.LittleEndian.PutUint16(record[62:], uint16(100+i))
		binary.LittleEndian.PutUint16(record[64:], uint16(600+i))
		record[66], record[67], record[68] = 13, 1, 1
		binary.LittleEndian.PutUint32(record[69:], 5)
		binary.LittleEndian.PutUint32(record[73:], 1)
		body = append(body, record[:]...)
		source := sim.SavedStructure{ID: native[i], Class: []sim.SavedStructureClass{sim.SavedBuilding, sim.SavedOutpost, sim.SavedTavern, sim.SavedShop, sim.SavedBuilding}[i],
			SourceKey: 0xabcde000 + uint32(i), ArchiveIndex: archives[i], RuntimeID: uint32(900 + i),
			Token0C: byte(20 + i), Token0E: uint16(300 + i), Token18: uint16(400 + i), Token1C: uint32(500 + i),
			Reference: 0xaabbccdd, Kind: byte(i + 1), Field46: uint16(600 + i), Field48: 13, Blocking: 5}
		copy(source.Position[:], record[:12])
		copy(source.Base52[:], record[37:59])
		source.Base52[14], source.Base52[15] = 1, 1
		binary.LittleEndian.PutUint32(source.Base52[18:], 5)
		switch class {
		case "Outpost":
			source.OutpostWords = [4]uint32{0x12345678, 0x87654321, 0xabcdef01, 0xfedcba09}
			for _, word := range source.OutpostWords {
				body = binary.LittleEndian.AppendUint32(body, word)
			}
			source.OutpostRecords = [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}, {9, 10, 11, 12, 13, 14, 15, 16}}
			body = binary.LittleEndian.AppendUint16(body, 2)
			body = append(body, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16)
		case "Tavern":
			source.Tavern9C = 0xfedc7654
			body = binary.LittleEndian.AppendUint32(body, 0xfedc7654)
		case "Shop":
			source.Shop70 = 0x98763210
			body = binary.LittleEndian.AppendUint32(body, 0x98763210)
		}
		sources = append(sources, source)
		live = append(live, sim.Structure{ID: native[i], Col: int32(10 + i), Row: 10, Width: 1, Height: 1, Attach: 1, Blocking: 5, Field42: 100, MaxHealth: uint16(100 + i)})
	}
	body = binary.LittleEndian.AppendUint16(body, 6) // alias first object
	body = binary.LittleEndian.AppendUint32(body, 0) // SpellEffects
	body = binary.LittleEndian.AppendUint16(body, 0) // block overlay
	body = binary.LittleEndian.AppendUint16(body, 4) // cell overlay
	for i, cell := range []uint16{0x0a0a, 0x0a0b, 0x0a0a, 0xffff} {
		body = binary.LittleEndian.AppendUint16(body, cell)
		payload := [52]byte{byte(10 + i), byte(2 + i), 6, 0xa3}
		binary.LittleEndian.PutUint32(payload[4:], 0x11112222)
		binary.LittleEndian.PutUint32(payload[8:], 0x33334444)
		binary.LittleEndian.PutUint32(payload[12:], 0xabcde000)
		binary.LittleEndian.PutUint32(payload[16:], 0x55556666)
		for j := 0; j < 6; j++ {
			binary.LittleEndian.PutUint32(payload[20+4*j:], 0x77778800+uint32(j))
		}
		copy(payload[44:], []byte{26, 43, 51, 59, 67, 75, 0xb3, 0xc3})
		body = append(body, payload[:]...)
	}
	body = binary.LittleEndian.AppendUint32(body, 0x778899aa)
	body = append(body, make([]byte, 4374+4)...)
	raw := completeDocumentTail1115(t, &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}}, savedContainer(savedTrailer(body)))
	document, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	clear(raw)
	actor := sim.Entity{ID: 41, Owner: sim.SelfSlot, X: 9, Y: 10, HP: 100, MaxHP: 100, Reach: 1, Facing: 64,
		AttackCharge: 1, AttackRelax: 2, Speed: 50, DyingTime: 200, SecondaryDamage: sim.SecondaryDamage{Base: 20, Spread: 1}}
	world, err := sim.NewWorld(0, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	cells := []sim.SavedStructureCell{{Cell: 0x0a0a, ID: 17, HasStructure: true, BaselineCost: 71, BaselineStatic: 81},
		{Cell: 0x0a0b, BaselineCost: 72, BaselineStatic: 82}, {Cell: 0xffff, ID: 123, HasStructure: true, BaselineCost: 73, BaselineStatic: 83}}
	if err := world.ImportOriginalStructures(live, sources, cells, make([]byte, 32*32)); err != nil {
		t.Fatal(err)
	}
	return document, world, sources
}

func TestSavedStructuresProjection1115DamageGeometrySuffixesAndCells(t *testing.T) {
	document, world, source := structureProjection1115Fixture(t)
	before, err := sav.CloneDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	oldShared := document // Must remain owned by its original holder.
	worldBefore := world.Hash()
	sim.Step(world, []sim.Command{{Kind: sim.KindAttackStructure, Entity: 41, X: 0}})
	for i := 0; i < 128 && world.Structures()[0].Field42 == 100; i++ {
		sim.Step(world, nil)
	}
	if world.Structures()[0].Field42 != 84 || world.Hash() == worldBefore {
		t.Fatal("literal first physical blow did not subtract16", world.Structures()[0])
	}
	// Current shape has a separate live owner. Declaration is a public state
	// writer, not a claim that original buildings move during normal gameplay.
	live := world.Structures()
	live[4].Col, live[4].Row, live[4].Width, live[4].Height, live[4].Attach, live[4].MaxHealth = 21, 22, 3, 2, 0xaabbccdd, 54321
	world.DeclareStructures(live)
	if err := projectSavedStructures(&document, world); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, oldShared) {
		t.Fatal("projection mutated a shared source document")
	}
	if !reflect.DeepEqual(document.World.Buildings, before.World.Buildings) || document.World.Buildings[0] != document.World.Buildings[5] {
		t.Fatal("projection expanded or rebound duplicate roots")
	}
	for i, record := range document.Objects {
		if record.Class != "Building" && record.Class != "Outpost" && record.Class != "Tavern" && record.Class != "Shop" && !reflect.DeepEqual(record, before.Objects[i]) {
			t.Fatal("unowned nonstructure record changed", i)
		}
	}
	wantCells := slices.Clone(before.World.Cells)
	wantCells[2].Building, wantCells[2].Cost, wantCells[2].Static = 0xabcde001, 71, 81
	wantCells[1].Building, wantCells[1].Cost, wantCells[1].Static = 0, 72, 82
	wantCells[3].Building, wantCells[3].Cost, wantCells[3].Static = 0xabcde004, 73, 83
	if !reflect.DeepEqual(document.World.Cells, wantCells) {
		t.Fatal("projection lost cell order or unowned payload fields", document.World.Cells)
	}
	encoded, err := sav.EncodeDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	buildings, present, err := f.Buildings()
	if err != nil || !present || len(buildings) != 5 {
		t.Fatal("serialized roster/alias", len(buildings), present, err)
	}
	decoded, err := sav.DecodeDocumentData(encoded)
	if err != nil || !reflect.DeepEqual(decoded.World.Cells, wantCells) {
		t.Fatal("serialized complete cells differ", err)
	}
	for i, got := range buildings {
		wantSource, wantLive := source[i], live[i]
		if got.Identity != wantSource.SourceKey || uint32(wantLive.ID) == got.Identity || got.ArchiveIndex == document.World.Buildings[i] {
			t.Fatal("fixture did not separate identity namespaces", i, got, document.World.Buildings)
		}
		if got.Col != byte(wantLive.Col) || got.Row != byte(wantLive.Row) || got.Width != wantLive.Width || got.Height != wantLive.Height || got.Attach != wantLive.Attach || got.Health != wantLive.Field42 || got.MaxHealth != wantLive.MaxHealth {
			t.Fatal("serialized current health/geometry", i, got, wantLive)
		}
		wantPosition := wantSource.Position
		if i == 4 {
			wantPosition[0], wantPosition[1], wantPosition[2], wantPosition[3] = 21, 22, 21, 22
		}
		wantBase := wantSource.Base52
		wantBase[14], wantBase[15] = wantLive.Width, wantLive.Height
		if got.Position != wantPosition || got.Base52 != wantBase || got.RuntimeID != uint32(900+i) || got.AuthoredID != 0 || got.Token0C != byte(20+i) || got.Token0E != uint16(300+i) || got.Token18 != uint16(400+i) || got.Token1C != uint32(500+i) || got.Reference != 0xaabbccdd || got.Kind != byte(i+1) || got.Field46 != uint16(600+i) || got.Field48 != 13 || got.Blocking != 5 {
			t.Fatal("retained base/Token field changed", i, got)
		}
		if got.OutpostWords != wantSource.OutpostWords || !reflect.DeepEqual(got.OutpostRecords, wantSource.OutpostRecords) || got.Tavern9C != wantSource.Tavern9C || got.Shop70 != wantSource.Shop70 {
			t.Fatal("retained subclass suffix changed", i, got)
		}
	}
	// Mutable output blocks must not alias retained registry blocks, source
	// documents or independently decoded output.
	for i := range document.Objects {
		for j := range document.Objects[i].Raw {
			clear(document.Objects[i].Raw[j].Bytes)
		}
	}
	currentSource, _, _ := world.SavedStructures()
	if !reflect.DeepEqual(currentSource, source) || !reflect.DeepEqual(oldShared, before) || !bytes.Equal(buildings[1].OutpostRecords[0][:], []byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatal("projected output leaked mutation into an independent owner")
	}
}

func TestSavedStructuresProjection1115CurrentRetainedOwner(t *testing.T) {
	document, world, sources := structureProjection1115Fixture(t)
	_, cells, _ := world.SavedStructures()
	// The registry, not the original DTO, owns this retained state. This test
	// supplies different explicit values through its public importer; it does
	// not assert an original service or a generated-constructor default.
	sources[0].RuntimeID, sources[0].AuthoredID = 0x3210, 0x5432
	sources[0].Position[4], sources[0].Position[6] = 119, 237
	sources[0].Base52[5], sources[0].Token1C = 211, 0x10203040
	sources[1].OutpostWords = [4]uint32{9, 8, 7, 6}
	sources[1].OutpostRecords = append(sources[1].OutpostRecords, [8]byte{19, 20, 21, 22, 23, 24, 25, 26})
	sources[2].Tavern9C, sources[3].Shop70 = 23, 34
	if err := world.ImportOriginalStructures(world.Structures(), sources, cells, make([]byte, 32*32)); err != nil {
		t.Fatal(err)
	}
	if err := projectSavedStructures(&document, world); err != nil {
		t.Fatal(err)
	}
	encoded, err := sav.EncodeDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := f.Buildings()
	if err != nil || got[0].RuntimeID != 0x3210 || got[0].AuthoredID != 0x5432 || got[0].Position[4] != 119 || got[0].Position[6] != 237 || got[0].Base52[5] != 211 || got[0].Token1C != 0x10203040 || got[1].OutpostWords != [4]uint32{9, 8, 7, 6} || !reflect.DeepEqual(got[1].OutpostRecords, sources[1].OutpostRecords) || got[2].Tavern9C != 23 || got[3].Shop70 != 34 {
		t.Fatal("replayed stale source instead of current retained registry", got, err)
	}
}

func TestSavedStructuresProjection1115LateRefusalIsAtomic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*sav.DocumentData, *sim.World)
	}{
		{"last source key absent", func(d *sav.DocumentData, _ *sim.World) {
			savedStructureSetValue(&d.Objects[d.World.Buildings[4]-1], "Identity", 0x12345678)
		}},
		{"distinct objects share key", func(d *sav.DocumentData, _ *sim.World) {
			savedStructureSetValue(&d.Objects[d.World.Buildings[4]-1], "Identity", 0xabcde000)
		}},
		{"nonstructure collision", func(d *sav.DocumentData, _ *sim.World) {
			savedStructureSetValue(&d.Objects[d.Players[0]-1], "This", 0xabcde004)
		}},
		{"wrong class", func(d *sav.DocumentData, _ *sim.World) { d.Objects[d.World.Buildings[4]-1].Class = "Token" }},
		{"valid different class", func(d *sav.DocumentData, _ *sim.World) {
			record := d.Objects[d.World.Buildings[2]-1]
			record.Values = slices.Clone(record.Values)
			savedStructureSetValue(&record, "Identity", 0xabcde004)
			d.Objects[d.World.Buildings[4]-1] = record
		}},
		{"root roster missing", func(d *sav.DocumentData, _ *sim.World) { d.World.Buildings = d.World.Buildings[:4] }},
		{"live roster missing", func(_ *sav.DocumentData, w *sim.World) { w.DeclareStructures(w.Structures()[:4]) }},
		{"late native mismatch", func(_ *sav.DocumentData, w *sim.World) { x := w.Structures(); x[4].ID++; w.DeclareStructures(x) }},
		{"late geometry width", func(_ *sav.DocumentData, w *sim.World) { x := w.Structures(); x[4].Col = 256; w.DeclareStructures(x) }},
		{"late cell missing", func(d *sav.DocumentData, _ *sim.World) { d.World.Cells = d.World.Cells[:3] }},
		{"late cell changed", func(d *sav.DocumentData, _ *sim.World) { d.World.Cells[3].Cell = 0xfffe }},
		{"late malformed block", func(d *sav.DocumentData, _ *sim.World) { d.Objects[d.World.Buildings[4]-1].Raw[0].Bytes = []byte{1} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, world, _ := structureProjection1115Fixture(t)
			tc.mutate(&document, world)
			before := actorProjection1115Copy(t, document)
			worldBefore := world.Hash()
			if err := projectSavedStructures(&document, world); err == nil {
				t.Fatal("accepted invalid structure projection")
			}
			if !reflect.DeepEqual(document, before) || world.Hash() != worldBefore {
				t.Fatal("refusal changed document or World")
			}
		})
	}
}

func TestSavedStructuresProjection1115AbsentAndPresentEmptyDiffer(t *testing.T) {
	document, _, _ := structureProjection1115Fixture(t)
	before, err := sav.EncodeDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	world, err := sim.NewWorld(0, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := projectSavedStructures(&document, world); err != nil {
		t.Fatal("old native absent mode", err)
	}
	after, err := sav.EncodeDocumentData(document)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("old absent mode reconstructed or removed source state", err)
	}
	if err := world.ImportOriginalStructures(nil, nil, nil, make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	if err := projectSavedStructures(&document, world); err == nil {
		t.Fatal("present-empty was mistaken for absent legacy mode")
	}
	emptyRaw := completeDocumentFixture1115(t, &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}})
	empty, err := sav.DecodeDocumentData(emptyRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := projectSavedStructures(&empty, world); err != nil || len(empty.World.Buildings) != 0 || len(empty.World.Cells) != 0 {
		t.Fatal("present-empty matching document failed", err)
	}
	if err := projectSavedStructures(nil, world); err == nil {
		t.Fatal("accepted nil document")
	}
	if err := projectSavedStructures(&document, nil); err == nil {
		t.Fatal("accepted nil world")
	}
}
