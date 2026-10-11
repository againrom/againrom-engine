package sav

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"testing"
)

func documentDataFromDocumentCellsOracle(d *saveDocument, origins *[]DocumentObjectOrigin) (DocumentData, error) {
	if d == nil || d.archive == nil || d.state == nil || d.campaign == nil {
		return DocumentData{}, fmt.Errorf("sav: incomplete document")
	}
	if err := documentDataEnvelope(d.version, d.label, d.archive.head.MapName, d.archive.marker, d.archive.global); err != nil {
		return DocumentData{}, err
	}
	if _, err := worldStateSerializedSize(d.state); err != nil {
		return DocumentData{}, err
	}
	if d.archive.world == nil {
		if err := validateCityState(d.state); err != nil {
			return DocumentData{}, err
		}
	} else if err := validateWorldState(d.state); err != nil {
		return DocumentData{}, err
	}
	if _, err := archiveCampaignSize(d.campaign); err != nil {
		return DocumentData{}, err
	}
	a := d.archive
	h := a.head
	out := DocumentData{Version: DocumentDataVersion, FileVersion: d.version, Label: cityDataCopy(d.label), Head: DocumentHeadData{h.CounterA, h.CounterB, h.MapName, h.Reserved, h.Mission, h.Difficulty, h.PlayerListField}, Marker: a.marker, GlobalDWord: a.global, Trailer: trailerBodyToArray(a.trailer)}
	b := documentDataBuilder{ids: map[*Record]uint16{}, inline: map[*Record]bool{}}
	var err error
	if out.Players, err = b.refs(a.players, 0); err != nil {
		return DocumentData{}, err
	}
	if out.DeadActors, err = b.refs(a.dead, 0); err != nil {
		return DocumentData{}, err
	}
	if w := a.world; w != nil {
		if len(w.blocks) > maxListElements || len(w.cells) > maxListElements {
			return DocumentData{}, fmt.Errorf("sav: document terrain count exceeds bound")
		}
		out.World = &DocumentWorldData{Blocks: cityDataCopy(w.blocks), TerrainIdentity: w.terrainIdentity, Session: DocumentSessionData(w.session)}
		for _, c := range w.cells {
			out.World.Cells = append(out.World.Cells, DocumentCellData(c))
		}
		for _, list := range []struct {
			src []*Record
			dst *[]uint16
		}{{w.buildings, &out.World.Buildings}, {w.effects, &out.World.Effects}, {w.sacks, &out.World.Sacks}} {
			if *list.dst, err = b.refs(list.src, 0); err != nil {
				return DocumentData{}, err
			}
		}
	}
	out.Objects = b.objects
	out.State.RootKind = d.state.rootKind
	for _, k := range documentSortedKeys(d.state.directoryKinds) {
		out.State.DirectoryRecords = append(out.State.DirectoryRecords, CityStateDirectoryData{k, d.state.directoryKinds[k]})
	}
	for _, k := range documentSortedKeys(d.state.values) {
		v := d.state.values[k]
		if v.kind != 2 {
			v.int32 = 0
		}
		out.State.ValueRecords = append(out.State.ValueRecords, CityStateRecordData{k, CityStateValueData{v.kind, v.int32, cityDataCopy(v.bytes)}})
	}
	out.Campaign = cityToDataCampaign(*d.campaign, nil)
	if len(out.Campaign.Children) == 0 {
		out.Campaign.Children = nil
	}
	if len(out.Campaign.Markers) == 0 {
		out.Campaign.Markers = nil
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(out), 0); err != nil {
		return DocumentData{}, err
	}
	if origins != nil {
		rows := make([]DocumentObjectOrigin, len(b.ids))
		for record, localID := range b.ids {
			archiveID := a.origins[record]
			if archiveID == 0 {
				return DocumentData{}, fmt.Errorf("sav: missing exact import origin for document object %d", localID)
			}
			rows[localID-1] = DocumentObjectOrigin{archiveID, localID}
		}
		*origins = rows
	}
	return out, nil
}

func saveDocumentFromDataCellsOracle(data DocumentData, permutation []uint16, borrowRaw bool) (*saveDocument, error) {
	if data.Version != DocumentDataVersion {
		return nil, fmt.Errorf("sav: unsupported document data version %d", data.Version)
	}
	if err := documentDataEnvelope(data.FileVersion, data.Label, data.Head.MapName, data.Marker, data.GlobalDWord); err != nil {
		return nil, err
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return nil, err
	}
	if permutation != nil && len(permutation) != len(data.Objects)+1 {
		return nil, fmt.Errorf("sav: invalid document index permutation extent")
	}
	if err := documentDataGraph(data, permutation); err != nil {
		return nil, err
	}
	if err := validateDocumentRoots(data); err != nil {
		return nil, err
	}
	if err := validateDocumentCampaign(data.Campaign); err != nil {
		return nil, err
	}
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range data.Objects {
		if err := documentRecordFromDataMode(r, objects[i], objects, false, 0, borrowRaw); err != nil {
			return nil, fmt.Errorf("sav: document object %d: %w", i+1, err)
		}
	}
	refs := func(ids []uint16) []*Record {
		var out []*Record
		for _, id := range ids {
			if id == 0 {
				out = append(out, nil)
			} else {
				out = append(out, objects[id-1])
			}
		}
		return out
	}
	h := data.Head
	a := &archiveDocument{head: Head{CounterA: h.CounterA, CounterB: h.CounterB, MapName: h.MapName, Reserved: h.Reserved, Mission: h.Mission, Difficulty: h.Difficulty, PlayerListField: h.PlayerListField}, players: refs(data.Players), dead: refs(data.DeadActors), marker: data.Marker, global: data.GlobalDWord, trailer: trailerBodyFromArray(data.Trailer)}
	if w := data.World; w != nil {
		if len(w.Blocks) > maxListElements || len(w.Cells) > maxListElements {
			return nil, fmt.Errorf("sav: document terrain count exceeds bound")
		}
		a.world = &archiveWorld{buildings: refs(w.Buildings), effects: refs(w.Effects), sacks: refs(w.Sacks), blocks: cityDataCopy(w.Blocks), terrainIdentity: w.TerrainIdentity, session: archiveSession(w.Session)}
		for _, cell := range w.Cells {
			a.world.cells = append(a.world.cells, archiveCell(cell))
		}
	}
	s, err := cityStateFromData(CityStateData{RootKind: data.State.RootKind, DirectoryRecords: data.State.DirectoryRecords, ValueRecords: data.State.ValueRecords}, 2)
	if err != nil {
		return nil, err
	}
	if a.world == nil {
		err = validateCityState(s)
	} else {
		err = validateWorldState(s)
	}
	if err != nil {
		return nil, err
	}
	campaign := cityFromDataCampaign(data.Campaign, nil)
	if _, err := archiveCampaignSize(&campaign); err != nil {
		return nil, err
	}
	if err := validateDocumentArchive(a); err != nil {
		return nil, err
	}
	return &saveDocument{version: data.FileVersion, label: cityDataCopy(data.Label), archive: a, state: s, campaign: &campaign}, nil
}

type documentCellsFixture struct {
	name string
	raw  []byte
}

func documentCellsFixtures(t *testing.T) []documentCellsFixture {
	t.Helper()
	world, _, _, _ := worldDocumentLiteral(t)
	fixtures := []documentCellsFixture{
		{"city", cityTestSource(t)},
		{"world", world},
		{"self-cycle", documentCycleLiteral(t, true)},
		{"mutual-cycle", documentCycleLiteral(t, false)},
	}
	classes := []string{"Token", "Item", "Shield", "Armor", "Weapon", "Building", "Effect", "Sack", "VirtualCaster", "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport", "Effect_DirectDamage", "Outpost", "Tavern", "Shop", "Diary", "Spell", "Spellbook", "Player", "Unit", "Humanoid", "Human"}
	want := slices.Clone(classes)
	sort.Strings(want)
	if !reflect.DeepEqual(want, documentSortedKeys(programmes)) {
		t.Fatal("cell conversion fixtures do not cover the class programme census")
	}
	for _, class := range classes {
		raw, _ := rootSackDocumentLiteral(t, func(s *stream) { archiveClassLiteral(s, class) })
		fixtures = append(fixtures, documentCellsFixture{class, raw})
	}
	return fixtures
}

func documentCellsArchiveDifferential(t *testing.T, d *saveDocument) DocumentData {
	t.Helper()
	var inputCells []archiveCell
	if d != nil && d.archive != nil && d.archive.world != nil {
		inputCells = slices.Clone(d.archive.world.cells)
	}
	var plain DocumentData
	for _, withOrigins := range []bool{false, true} {
		var gotOrigins, wantOrigins []DocumentObjectOrigin
		var gotOriginsTarget, wantOriginsTarget *[]DocumentObjectOrigin
		if withOrigins {
			gotOriginsTarget, wantOriginsTarget = &gotOrigins, &wantOrigins
		}
		got, err := documentDataFromDocument(d, gotOriginsTarget)
		want, wantErr := documentDataFromDocumentCellsOracle(d, wantOriginsTarget)
		if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotOrigins, wantOrigins) {
			t.Fatal("archive conversion changed complete DTO, origins or exact error", err, wantErr)
		}
		if !withOrigins {
			plain = want
		}
		if err == nil && got.World != nil {
			if len(inputCells) == 0 && got.World.Cells != nil {
				t.Fatal("zero cells changed canonical nil output")
			}
			for i := range got.World.Cells {
				got.World.Cells[i].Cell ^= 7
				got.World.Cells[i].Layers[0] ^= 7
			}
			if !reflect.DeepEqual(d.archive.world.cells, inputCells) {
				t.Fatal("DTO cells alias or changed the archive")
			}
		}
	}
	return plain
}

func documentCellsDataDifferential(t *testing.T, data DocumentData) {
	t.Helper()
	before, _ := documentDataGobCopy(t, data)
	var inputCells []DocumentCellData
	if data.World != nil {
		inputCells = slices.Clone(data.World.Cells)
	}
	for _, borrowRaw := range []bool{false, true} {
		for _, reindex := range []bool{false, true} {
			var gotPermutation, wantPermutation []uint16
			if reindex {
				gotPermutation = make([]uint16, len(data.Objects)+1)
				wantPermutation = make([]uint16, len(data.Objects)+1)
			}
			got, err := saveDocumentFromDataIndexedMode(data, gotPermutation, borrowRaw)
			want, wantErr := saveDocumentFromDataCellsOracle(data, wantPermutation, borrowRaw)
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotPermutation, wantPermutation) {
				t.Fatal("data conversion changed complete records, permutation or exact error", err, wantErr)
			}
			if err == nil {
				gotBytes, gotErr := serializeSaveDocument(got)
				wantBytes, wantErr := serializeSaveDocument(want)
				if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || !bytes.Equal(gotBytes, wantBytes) {
					t.Fatal("converted complete SAV bytes or serialization error differ", gotErr, wantErr)
				}
				if got.archive.world != nil {
					if len(inputCells) == 0 && got.archive.world.cells != nil {
						t.Fatal("zero cells changed canonical nil archive output")
					}
					for i := range got.archive.world.cells {
						got.archive.world.cells[i].Cell ^= 7
						got.archive.world.cells[i].Layers[0] ^= 7
					}
					if !reflect.DeepEqual(data.World.Cells, inputCells) {
						t.Fatal("archive cells alias or changed the DTO")
					}
				}
			}
			after, _ := documentDataGobCopy(t, data)
			if !bytes.Equal(before, after) {
				t.Fatal("conversion or output mutation changed the whole input DTO")
			}
		}
	}
}

func TestDocumentCellsConversionsMatchWholeFunctions(t *testing.T) {
	for _, fixture := range documentCellsFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			d, err := parseSaveDocument(fixture.raw)
			if err != nil {
				t.Fatal(err)
			}
			data := documentCellsArchiveDifferential(t, d)
			documentCellsDataDifferential(t, data)
		})
	}
}

func documentCellsPayload(n int) []archiveCell {
	if n == 0 {
		return nil
	}
	rows := make([]archiveCell, n)
	for i := range rows {
		rows[i] = archiveCell{
			Cell: uint16(i % 7), Cost: uint8(i + 1), Static: uint8(i + 2), LayerCount: uint8(i % 7), Residue03: uint8(i + 3),
			GroundActor: uint32(i + 11), AirActor: uint32(i + 12), Building: uint32(i + 13), Sack: uint32(i + 14),
			Layers:    [6]uint32{uint32(i), 1, 0xffffffff, uint32(i + 2), uint32(i + 3), uint32(i + 4)},
			Operation: uint8(i + 5), Power: uint8(i + 6), SourceX: uint8(i + 7), SourceY: uint8(i + 8),
			TargetX: uint8(i + 9), TargetY: uint8(i + 10), Residue32: uint16(i + 15),
		}
	}
	return rows
}

func TestDocumentCellsNilEmptyRepeatedAndBounded(t *testing.T) {
	raw, _, _, _ := worldDocumentLiteral(t)
	for _, n := range []int{-1, 0, 1, 3, 257, maxListElements, maxListElements + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			d, err := parseSaveDocument(raw)
			if err != nil {
				t.Fatal(err)
			}
			if n < 0 {
				d.archive.world.cells = []archiveCell{}
			} else {
				d.archive.world.cells = documentCellsPayload(n)
			}
			documentCellsArchiveDifferential(t, d)
			data, err := documentDataFromDocumentCellsOracle(d, nil)
			if n > maxListElements {
				if err == nil {
					t.Fatal("over-bound archive cells were accepted")
				}
				data = documentDataFixture(t)
				data.World.Cells = make([]DocumentCellData, n)
			} else if err != nil {
				t.Fatal(err)
			} else if n < 0 {
				data.World.Cells = []DocumentCellData{}
			}
			documentCellsDataDifferential(t, data)
		})
	}
}

func TestDocumentCellsDataFailuresMatchWholeFunction(t *testing.T) {
	mutations := []struct {
		name string
		edit func(*DocumentData)
	}{
		{"version", func(d *DocumentData) { d.Version++ }},
		{"label", func(d *DocumentData) { d.Label = []byte{'x', 0} }},
		{"root-range", func(d *DocumentData) { d.Players[0] = uint16(len(d.Objects) + 1) }},
		{"unreachable", func(d *DocumentData) { d.Objects = append(d.Objects, d.Objects[len(d.Objects)-1]) }},
		{"class", func(d *DocumentData) { d.Objects[len(d.Objects)-1].Class = "Unsupported" }},
		{"raw-extent", func(d *DocumentData) { documentRecordByClass(t, d, "Outpost").Raw[0].Bytes = []byte{0} }},
		{"field-order", func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Values[0], r.Values[1] = r.Values[1], r.Values[0]
		}},
		{"block-order", func(d *DocumentData) { d.World.Blocks[0], d.World.Blocks[1] = d.World.Blocks[1], d.World.Blocks[0] }},
		{"state-duplicate", func(d *DocumentData) { d.State.ValueRecords = append(d.State.ValueRecords, d.State.ValueRecords[0]) }},
		{"campaign", func(d *DocumentData) { d.Campaign.Base.DWords[5] = 3 }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			data := documentDataFixture(t)
			mutation.edit(&data)
			documentCellsDataDifferential(t, data)
		})
	}
	data := reverseDocumentIndices(t, documentDataFixture(t))
	documentCellsDataDifferential(t, data)
	for _, borrowRaw := range []bool{false, true} {
		gotPermutation, wantPermutation := []uint16{0}, []uint16{0}
		got, err := saveDocumentFromDataIndexedMode(data, gotPermutation, borrowRaw)
		want, wantErr := saveDocumentFromDataCellsOracle(data, wantPermutation, borrowRaw)
		if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotPermutation, wantPermutation) {
			t.Fatal("invalid permutation extent changed output or exact error", err, wantErr)
		}
	}
}

func TestDocumentCellsArchiveFailuresMatchWholeFunction(t *testing.T) {
	raw, _, _, _ := worldDocumentLiteral(t)
	for _, name := range []string{"nil", "archive", "state", "campaign", "version", "label", "class", "raw", "origin"} {
		t.Run(name, func(t *testing.T) {
			input := raw
			if name == "origin" {
				input = documentCycleLiteral(t, true)
			}
			d, err := parseSaveDocument(input)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "nil":
				d = nil
			case "archive":
				d.archive = nil
			case "state":
				d.state = nil
			case "campaign":
				d.campaign = nil
			case "version":
				d.version = 0
			case "label":
				d.label = []byte{'x', 0}
			case "class":
				d.archive.world.buildings[0].Class = "Unsupported"
			case "raw":
				r := d.archive.world.buildings[0]
				keys := documentSortedKeys(r.Raw)
				if len(keys) == 0 {
					t.Fatal("archive failure fixture has no raw field")
				}
				r.Raw[keys[0]] = []byte{0}
			case "origin":
				if d.archive.world == nil || len(d.archive.world.effects) == 0 || d.archive.world.effects[0] == nil || d.archive.origins[d.archive.world.effects[0]] == 0 {
					t.Fatal("origin failure fixture has no registered Effect root")
				}
				delete(d.archive.origins, d.archive.world.effects[0])
				var origins []DocumentObjectOrigin
				if _, err := documentDataFromDocument(d, &origins); err == nil || origins != nil {
					t.Fatal("incomplete origins were not refused without partial rows", err)
				}
			}
			documentCellsArchiveDifferential(t, d)
		})
	}
}

func documentCellsDecode(raw []byte, withOrigins bool, oracle bool) (DocumentData, []DocumentObjectOrigin, error) {
	d, err := parseSaveDocument(raw)
	if err != nil {
		return DocumentData{}, nil, err
	}
	var origins []DocumentObjectOrigin
	var target *[]DocumentObjectOrigin
	if withOrigins {
		target = &origins
	}
	var data DocumentData
	if oracle {
		data, err = documentDataFromDocumentCellsOracle(d, target)
	} else {
		data, err = documentDataFromDocument(d, target)
	}
	return data, origins, err
}

func TestDocumentCellsCorruptInputsMatchWholeFunction(t *testing.T) {
	world, _, _, _ := worldDocumentLiteral(t)
	for _, source := range []documentCellsFixture{{"city", cityTestSource(t)}, {"world", world}} {
		var inputs []documentCellsFixture
		for _, n := range []int{0, 1, 15, 16, len(source.raw) / 2, len(source.raw) - 1} {
			inputs = append(inputs, documentCellsFixture{fmt.Sprintf("prefix-%d", n), source.raw[:n]})
		}
		for _, index := range []int{0, 4, 8, 12, 16, len(source.raw) / 2, len(source.raw) - 1} {
			bad := bytes.Clone(source.raw)
			bad[index] ^= 0xff
			inputs = append(inputs, documentCellsFixture{fmt.Sprintf("damage-%d", index), bad})
		}
		for _, input := range inputs {
			t.Run(source.name+"/"+input.name, func(t *testing.T) {
				before := bytes.Clone(input.raw)
				for _, withOrigins := range []bool{false, true} {
					got, origins, err := documentCellsDecode(input.raw, withOrigins, false)
					want, wantOrigins, wantErr := documentCellsDecode(input.raw, withOrigins, true)
					if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(origins, wantOrigins) || !bytes.Equal(before, input.raw) {
						t.Fatal("corrupt input changed whole DTO, origins, bytes or exact error", err, wantErr)
					}
				}
			})
		}
	}
}
