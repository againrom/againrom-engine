package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentCarrierFixture(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("native carrier absence").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.live.world.CellTails()) != 0 || len(f.live.world.SavedCellRecords()) != 0 || len(f.live.world.SavedDiaries()) != 0 {
		t.Fatal("fixture no longer represents absent native carriers")
	}
	return f, s
}

func openCurrentCarrierSave(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	opener, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("cold carrier state").OpenMission(opener)
	}
	if err != nil || town {
		t.Fatal("current carrier LOAD", err)
	}
	return f
}

func TestCurrentCarrierAbsenceKeepsStrictNativeWorldAcrossTwoCycles(t *testing.T) {
	f, s := currentCarrierFixture(t)
	var source sim.World
	if err := source.UnmarshalBinary(s.World); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw, err := f.ExportCurrentSave(s, "current carrier cycle")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || len(a.AbsentCellTails) == 0 || len(a.AbsentCellRecords) == 0 || len(a.AbsentDiaries) == 0 {
			t.Fatal("absent carriers have no final ordinary anchors", err)
		}
		f = openCurrentCarrierSave(t, raw)
		if f.live.world.Hash() != source.Hash() {
			t.Fatalf("cycle %d changed exact native World.Hash: %x -> %x", cycle, source.Hash(), f.live.world.Hash())
		}
		for tick := 0; tick < 20; tick++ {
			sim.Step(&source, nil)
			sim.Step(f.live.world, nil)
			if f.live.world.Hash() != source.Hash() {
				t.Fatalf("cycle %d tick %d changed exact continuation: %x -> %x", cycle, tick, source.Hash(), f.live.world.Hash())
			}
		}
		s, _, err = f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func currentNativeDeathSack(t *testing.T) *FrontEnd {
	t.Helper()
	f, _ := itemMutationOpen1115(t, 1, false)
	f.live.tick()
	actor := newGroupActors1115(t, f.live.world)[newGroupA]
	f.live.pending = append(f.live.pending, sim.Damage(actor.ID, actor.HP+10))
	var sack sim.SavedObjectID
	for tick := 0; tick < 100 && sack == 0; tick++ {
		f.live.tick()
		for _, row := range f.live.world.SavedObjects().Sacks {
			if !row.Retired && row.Token.Identity == 0 {
				sack = row.ID
			}
		}
	}
	if sack == 0 {
		t.Fatal("actual death did not construct a Sack with an absent native key")
	}
	return f
}

func TestCurrentSackAbsenceKeepsStrictNativeWorldAcrossTwoCycles(t *testing.T) {
	f := currentNativeDeathSack(t)
	source := f.live.world
	for cycle := 0; cycle < 2; cycle++ {
		doc, a := currentRootSAVDocument(t, f)
		if len(a.AbsentCellSacks) == 0 {
			t.Fatal("native Sack cell key has no absence anchor")
		}
		f = loadCurrentRootSAV(t, doc, func(t *testing.T) *FrontEnd { return itemMutationFront1115(t, 1) })
		if f.live.world.Hash() != source.Hash() {
			before, _ := source.MarshalBinary()
			after, _ := f.live.world.MarshalBinary()
			currentItemWorldDiagnostics(t, before, after)
			t.Fatalf("cycle %d changed exact native Sack World.Hash: %x -> %x", cycle, source.Hash(), f.live.world.Hash())
		}
		for tick := 0; tick < 20; tick++ {
			sim.Step(source, nil)
			sim.Step(f.live.world, nil)
			if f.live.world.Hash() != source.Hash() {
				t.Fatalf("cycle %d tick %d changed exact Sack continuation: %x -> %x", cycle, tick, source.Hash(), f.live.world.Hash())
			}
		}
	}
}

func TestCurrentDeathKeepsChangedOrdinaryMotionOrderAndSackKey(t *testing.T) {
	for _, field := range []string{"position", "mover", "order", "sack"} {
		t.Run(field, func(t *testing.T) {
			f := currentNativeDeathSack(t)
			doc, a := currentRootSAVDocument(t, f)
			motions, _, _, _ := f.live.world.SavedActorMotions()
			var actor sim.EntityID
			var object uint16
			for _, m := range motions {
				if m.Current || m.Issue == "" {
					continue
				}
				for _, b := range a.Bindings {
					if b.ID == m.Entity && !b.Structure && !b.Missing {
						actor, object = m.Entity, b.Object
					}
				}
			}
			if object == 0 {
				t.Fatal("death fixture lacks an ordinary frozen mover")
			}
			var sackCell uint16
			var sackKey uint32
			r := &doc.Objects[object-1]
			switch field {
			case "position":
				raw, _ := savedObjectRaw(r, "Block12", 12)
				raw[4] = 73
			case "mover":
				raw, _ := savedObjectRaw(r, "U154", 180)
				raw[0x60] = 83
			case "order":
				raw, _ := savedObjectRaw(r, "U158", 148)
				raw[0x40] = 93
			case "sack":
				for _, row := range a.AbsentCellSacks {
					if row.Carriers == sim.CurrentSackKeyRecord|sim.CurrentSackKeyMotion {
						sackCell, sackKey = row.Cell, row.Wire+1000
						for i := range doc.Objects {
							record := &doc.Objects[i]
							identity, _ := savedStructureValue(record, "Identity")
							if record.Class == "Sack" && identity == row.Wire {
								savedObjectSetValue(record, "Identity", sackKey)
							}
						}
						for i := range doc.World.Cells {
							if doc.World.Cells[i].Cell == row.Cell {
								doc.World.Cells[i].Sack = sackKey
							}
						}
						break
					}
				}
				if sackKey == 0 {
					t.Fatal("death fixture lacks both absent cell Sack carriers")
				}
			}
			for cycle := 0; cycle < 2; cycle++ {
				f = loadCurrentRootSAV(t, doc, func(t *testing.T) *FrontEnd { return itemMutationFront1115(t, 1) })
				motions, cells, _, _ := f.live.world.SavedActorMotions()
				found := false
				for _, m := range motions {
					if m.Entity == actor && (field == "position" && m.Position.FineX == 73 || field == "mover" && m.Mover[0x60] == 83) {
						found = !m.Current && m.Issue != ""
					}
				}
				if field == "order" {
					_, orders, _ := f.live.world.SavedGroups()
					for _, order := range orders {
						if order.Entity == actor && order.Raw[0x40] == 93 {
							found = true
						}
					}
				}
				if field == "sack" {
					for _, cell := range cells {
						if cell.Cell == sackCell && binary.LittleEndian.Uint32(cell.Payload[16:]) == sackKey {
							for _, record := range f.live.world.SavedCellRecords() {
								found = found || record.Cell == sackCell && record.Sack == sackKey
							}
						}
					}
				}
				if !found {
					t.Fatal("native continuation swallowed an ordinary field edit", cycle)
				}
				if cycle == 0 {
					doc, _ = currentRootSAVDocument(t, f)
				}
			}
		})
	}
}

func TestCurrentCarrierAbsenceKeepsOrdinaryEditsAndPresentRows(t *testing.T) {
	f, s := currentCarrierFixture(t)
	raw, err := f.ExportCurrentSave(s, "ordinary carrier values")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"tail", "residue", "ground", "diary count", "diary remaining", "diary self"} {
		t.Run(field, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			cell := &doc.World.Cells[0]
			diaries, err := currentDiaryRecords(&doc, a, f.live.world)
			if err != nil {
				t.Fatal(err)
			}
			owner := a.AbsentDiaries[0].Owner
			diary := diaries[owner].Record
			switch field {
			case "tail":
				cell.Operation, cell.Power = 26, 17
			case "residue":
				cell.LayerCount, cell.Residue03, cell.Residue32 = 3, 19, 0x1234
			case "ground":
				cell.GroundActor = 0
			case "diary count", "diary remaining":
				for i := range diary.Raw {
					r := &diary.Raw[i]
					if field == "diary count" && r.Name == "Journal" {
						binary.LittleEndian.PutUint32(r.Bytes, 713)
					}
					if field == "diary remaining" && r.Name == "JournalWords" {
						binary.LittleEndian.PutUint16(r.Bytes, 907)
					}
				}
			case "diary self":
				mustSetValue(diary, "D2C", 0x12345678)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			back, _ := sav.DecodeDocumentData(candidate)
			unchanged, _, _ := sav.NativeActions(back.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary edit changed absence policy")
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentCarrierSave(t, candidate)
				w := cold.live.world
				switch field {
				case "tail":
					want := []sim.CellTail{{X: int32(cell.Cell & 255), Y: int32(cell.Cell >> 8), Bytes: [6]byte{26, 17}}}
					if !reflect.DeepEqual(w.CellTails(), want) || len(w.SavedCellRecords()) != 0 || len(w.SavedDiaries()) != 0 {
						t.Fatal("ordinary tail lost or adopted unrelated carriers", cycle, w.CellTails(), w.SavedCellRecords(), w.SavedDiaries())
					}
				case "residue", "ground":
					rows := w.SavedCellRecords()
					if len(rows) != 1 || rows[0].Cell != cell.Cell || rows[0].LayerCount != cell.LayerCount || rows[0].Residue0 != cell.Residue03 || rows[0].Residue1 != ([2]byte{byte(cell.Residue32), byte(cell.Residue32 >> 8)}) || field == "ground" && rows[0].Ground.Key != 0 || len(w.CellTails()) != 0 || len(w.SavedDiaries()) != 0 {
						t.Fatal("ordinary residue lost or adopted unrelated carriers", cycle, rows)
					}
				default:
					rows := w.SavedDiaries()
					want, err := sav.ReadDocumentDiary(*diary)
					if err != nil || len(rows) != 1 || rows[0].Owner != owner || rows[0].Length != want.Length || !reflect.DeepEqual(rows[0].Entries, savDiaryEntriesToSaved(want.Entries)) || len(w.CellTails()) != 0 || len(w.SavedCellRecords()) != 0 {
						t.Fatal("ordinary Diary lost or adopted unrelated carriers", cycle, rows, want, err)
					}
				}
				if cycle == 0 {
					next, label, err := cold.Snapshot(true)
					if err != nil {
						t.Fatal(err)
					}
					candidate, err = cold.ExportCurrentSave(next, label)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

func TestCurrentLegacyCellSackAbsenceKeepsMotionAndChangedKey(t *testing.T) {
	for _, wire := range []uint32{91, 99} {
		w, err := sim.NewWorld(1, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.SetSavedCellRecords([]sim.SavedCellRecord{{Cell: 0x0101, Sack: wire}})
		cell := sim.SavedActorCell{Cell: 0x0101}
		binary.LittleEndian.PutUint32(cell.Payload[16:], wire)
		if err := w.ImportOriginalActorMotions(nil, []sim.SavedActorCell{cell}, nil); err != nil {
			t.Fatal(err)
		}
		doc := sav.DocumentData{World: &sav.DocumentWorldData{Cells: []sav.DocumentCellData{{Cell: cell.Cell, Sack: wire}}}}
		a := currentActionData{AbsentCellSacks: []currentAbsentCellSack{{Cell: cell.Cell, Wire: 91}}}
		if err := restoreCurrentCarrierAbsence(w, &doc, &a); err != nil {
			t.Fatal(err)
		}
		want := wire
		if wire == 91 {
			want = 0
		}
		_, cells, _, _ := w.SavedActorMotions()
		if w.SavedCellRecords()[0].Sack != want || binary.LittleEndian.Uint32(cells[0].Payload[16:]) != wire {
			t.Fatal("legacy record policy changed a motion key or ignored an ordinary edit")
		}
	}
}

func TestCurrentCarrierAbsenceMalformedPolicyDoesNotChangeSession(t *testing.T) {
	f, s := currentCarrierFixture(t)
	raw, err := f.ExportCurrentSave(s, "absence validation")
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"cell duplicate", "cell empty anchor", "diary duplicate", "diary object", "diary owner"} {
		doc, _ := sav.DecodeDocumentData(raw)
		a, _ := readCurrentActions(&doc)
		switch fault {
		case "cell duplicate":
			a.AbsentCellTails = append(a.AbsentCellTails, a.AbsentCellTails[0])
		case "cell empty anchor":
			a.AbsentCellRecords[0].Anchor = [32]byte{}
		case "diary duplicate":
			a.AbsentDiaries = append(a.AbsentDiaries, a.AbsentDiaries[0])
		case "diary object":
			a.AbsentDiaries[0].Object = 0
		case "diary owner":
			a.AbsentDiaries[0].Owner.Actor = 17
		}
		leaf, _ := json.Marshal(a)
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		candidate, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		live, hash := f.live, f.live.world.Hash()
		if _, _, err := f.RestoreOriginal(candidate); err == nil || f.live != live || f.live.world.Hash() != hash {
			t.Fatal("malformed absence policy changed live session", fault, err)
		}
	}
}

func TestCurrentCarrierAbsenceFollowsChangedDiaryOwners(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	opener, town, err := f.RestoreOriginal(spellbookSave1096(true, 1))
	if err == nil && !town {
		err = f.App("ordinary Diary owners").OpenMission(opener)
	}
	if err != nil || town {
		t.Fatal(err)
	}
	f.live.world.SetSavedDiaries(nil)
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	var actors []currentActionBinding
	for _, b := range a.Bindings {
		if b.Missing || b.Structure {
			continue
		}
		r := &doc.Objects[b.Object-1]
		if r.Class == "Human" || r.Class == "Humanoid" {
			actors = append(actors, b)
		}
	}
	if len(actors) != 2 {
		t.Fatal("owner control requires two distinct ordinary Human Diaries", len(actors))
	}
	left, right := &doc.Objects[actors[0].Object-1], &doc.Objects[actors[1].Object-1]
	leftRefs, _ := savedObjectRefs(left, "Diary")
	rightRefs, _ := savedObjectRefs(right, "Diary")
	if len(leftRefs) != 1 || len(rightRefs) != 1 || leftRefs[0] == 0 || rightRefs[0] == 0 || leftRefs[0] == rightRefs[0] {
		t.Fatal("fixture Diary edges are absent or already aliased")
	}
	want := map[sim.SavedDiaryOwner]sav.Diary{}
	want[sim.SavedDiaryOwner{Actor: actors[0].ID}], err = sav.ReadDocumentDiary(doc.Objects[rightRefs[0]-1])
	if err != nil {
		t.Fatal(err)
	}
	want[sim.SavedDiaryOwner{Actor: actors[1].ID}], err = sav.ReadDocumentDiary(doc.Objects[leftRefs[0]-1])
	if err != nil {
		t.Fatal(err)
	}
	mustSetRefs(left, "Diary", rightRefs)
	mustSetRefs(right, "Diary", leftRefs)
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		f = openCurrentCarrierSave(t, raw)
		got := f.live.world.SavedDiaries()
		if len(got) != 2 {
			t.Fatal("ordinary Diary owner edit was swallowed by absence", cycle, got)
		}
		for _, row := range got {
			d, ok := want[row.Owner]
			if !ok || row.Length != d.Length || !reflect.DeepEqual(row.Entries, savDiaryEntriesToSaved(d.Entries)) {
				t.Fatal("current Diary belongs to a different owner", cycle, row)
			}
		}
		if cycle == 0 {
			s, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = f.ExportCurrentSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCurrentCellAbsenceFitsFullWireKeyspace(t *testing.T) {
	rows := make(currentCellAbsence, 65536)
	for i := range rows {
		rows[i].Cell = uint16(i)
		for j := range rows[i].Anchor {
			rows[i].Anchor[j] = 255
		}
	}
	a := currentActionData{Version: 1, AbsentCellTails: rows, AbsentCellRecords: rows}
	raw, err := json.Marshal(a)
	if err != nil || len(raw) > sav.MaxNativeActions {
		t.Fatal("full cell keyspace exceeds native absence framing", len(raw), err)
	}
	doc := sav.DocumentData{World: &sav.DocumentWorldData{}}
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		t.Fatal(err)
	}
	back, err := readCurrentActions(&doc)
	if err != nil || !reflect.DeepEqual(back.AbsentCellTails, rows) || !reflect.DeepEqual(back.AbsentCellRecords, rows) {
		t.Fatal("full cell keyspace lost exact absence anchors", err)
	}
	for _, malformed := range []string{`"AQ=="`, `"not base64"`, `[0,1]`} {
		var decoded currentCellAbsence
		if err := json.Unmarshal([]byte(malformed), &decoded); err == nil {
			t.Fatal("invalid cell absence framing accepted", malformed)
		}
	}
}
