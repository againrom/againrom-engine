package game

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentBuildingConstructorOwnsFreshFieldsBeforeFirstSave(t *testing.T) {
	f := structureFront(t, true, true)
	if err := f.App("current Building constructor").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	world := f.live.world
	sources, cells, present := world.SavedStructures()
	if !present || len(sources) != 3 || len(cells) != 3 {
		t.Fatal("fresh entry lacks complete current Building source/cell carriers", present, sources, cells)
	}
	keys, runtimes := map[uint32]bool{}, map[uint32]bool{}
	var terrain uint32
	for i, source := range sources {
		live := world.Structures()[i]
		key := binary.LittleEndian.Uint32(source.Position[8:])
		if key == 0 || terrain != 0 && terrain != key || source.ID != live.ID || source.SourceKey == 0 || keys[source.SourceKey] ||
			source.RuntimeID == 0 || runtimes[source.RuntimeID] || !source.Class.Generated() || source.Token18 != 2 ||
			source.Position[0] != byte(live.Col) || source.Position[1] != byte(live.Row) || !source.HasAuthored {
			t.Fatal("fresh Building lacks independent current identity/position/publication", source, live)
		}
		keys[source.SourceKey], runtimes[source.RuntimeID], terrain = true, true, key
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "fresh current Building carriers")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World.TerrainIdentity != terrain {
		t.Fatal("SAVE changed the current Terrain reference", err)
	}
	cold := coldCurrentBuildings(t, raw)
	checkCurrentBuildingSources(t, cold.live.world, world.Structures(), sources)
	if !reflect.DeepEqual(world.CurrentPolicy().Terrain, cold.live.world.CurrentPolicy().Terrain) ||
		!reflect.DeepEqual(world.StructureOccupancy(), cold.live.world.StructureOccupancy()) {
		t.Fatal("SAV LOAD changed fresh structure terrain or aliases")
	}
	ghost, ok := world.Entity(0)
	if !ok || ghost.Domain != sim.DomainGhost {
		t.Fatal("fixture lacks a resolved fresh Ghost", ghost)
	}
	command := sim.MoveTo(ghost.ID, sim.CellPoint{X: 1, Y: 8})
	sim.Step(world, []sim.Command{command})
	sim.Step(cold.live.world, []sim.Command{command})
	warmGhost, _ := world.Entity(ghost.ID)
	coldGhost, _ := cold.live.world.Entity(ghost.ID)
	if warmGhost.X != coldGhost.X || warmGhost.Y != coldGhost.Y ||
		!reflect.DeepEqual(world.Route(ghost.ID), cold.live.world.Route(ghost.ID)) {
		t.Fatal("SAV LOAD changed fresh Ghost routing", world.Route(ghost.ID), cold.live.world.Route(ghost.ID))
	}
	cold.live.tick()
	next, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	nextRaw, err := cold.ExportCurrentSave(next, "next fresh Building save")
	if err != nil {
		t.Fatal(err)
	}
	checkCurrentBuildingSources(t, coldCurrentBuildings(t, nextRaw).live.world, world.Structures(), sources)
	for i := range doc.World.Cells {
		cell := &doc.World.Cells[i]
		if cell.Cell != cells[0].Cell {
			continue
		}
		cell.Cost, cell.Static = 51, cell.Static|0x80
		edited, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		changed := coldCurrentBuildings(t, edited)
		checkCurrentBuildingSources(t, changed.live.world, world.Structures(), sources)
		_, currentCells, _ := changed.live.world.SavedStructures()
		if currentCells[0].BaselineCost != 51 || currentCells[0].BaselineStatic != cell.Static {
			t.Fatal("ordinary cell edit lost to fresh metadata", currentCells[0], cell)
		}
		return
	}
	t.Fatal("fresh Building Cell missing")
}

func coldCurrentBuildings(t *testing.T, raw []byte, duplicate ...bool) *FrontEnd {
	t.Helper()
	cold := structureFront(t, append([]bool{true, true}, duplicate...)...)
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("current Building LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("cold current Building LOAD", town, err)
	}
	return cold
}

func TestCurrentBuildingConstructorDuplicateAuthoredIDsKeepExactSubjects(t *testing.T) {
	f := structureFront(t, true, true, true)
	if err := f.App("duplicate current Building IDs").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal("known duplicate authored ID refused fresh entry", err)
	}
	sources, _, present := f.live.world.SavedStructures()
	live := f.live.world.Structures()
	if !present || len(sources) != 3 || sources[0].AuthoredID != 51 || sources[1].AuthoredID != 51 || sources[0].SourceKey == sources[1].SourceKey {
		t.Fatal("duplicate T08 replaced current exact subjects", sources)
	}
	for cycle := 0; cycle < 2; cycle++ {
		snapshot, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, "duplicate authored ID current save")
		if err != nil {
			t.Fatal(err)
		}
		if cycle == 0 {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class == "Building" && actorProjectionValue(t, *r, "Identity") == sources[1].SourceKey {
					mustSetValue(r, "T18", 0x100)
					sources[1].Token18 = 0x100
				}
			}
			doc, _, err = sav.ReindexDocumentData(doc)
			if err == nil {
				raw, err = sav.EncodeDocumentData(doc)
			}
			if err != nil {
				t.Fatal("current structure receipt cannot reindex", err)
			}
		}
		f = coldCurrentBuildings(t, raw, true)
		checkCurrentBuildingSources(t, f.live.world, live, sources)
		for _, source := range f.live.world.StructureOccupancy() {
			if source.ID > 2 {
				t.Fatal("current structural ID reminted", source)
			}
		}
		f.live.tick()
	}
}

func TestCurrentBuildingBindingsRejectLostSubjectsAtomically(t *testing.T) {
	f := structureFront(t, true, true, true)
	if err := f.App("current Building binding controls").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "current Building exact binding controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"missing", "present-empty", "absent", "duplicate-id", "same-key", "same-object", "object-swap", "wrong-class", "wrong-index", "no-generated-mode", "anchor-mismatch"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || a.StructureBindings == nil {
				t.Fatal("missing control receipt", err)
			}
			rows := *a.StructureBindings
			switch change {
			case "missing":
				rows = rows[:2]
			case "present-empty":
				rows = []currentStructureBinding{}
			case "duplicate-id":
				rows[1].ID = rows[0].ID
			case "same-key":
				rows[1].SourceKey = rows[0].SourceKey
			case "same-object":
				rows[1].Object = rows[0].Object
			case "object-swap":
				rows[0].Object, rows[1].Object = rows[1].Object, rows[0].Object
			case "wrong-class":
				rows[0].Class = sim.GeneratedOutpost
			case "wrong-index":
				rows[0].AuthoredIndex = 1
			case "no-generated-mode":
				for i := range a.ArchiveCoordinates {
					if a.ArchiveCoordinates[i].Object == rows[0].Object {
						a.ArchiveCoordinates[i].Generated, a.ArchiveCoordinates[i].ModeAnchor = false, nil
						a.ArchiveCoordinates[i].Native = 1
					}
				}
			case "anchor-mismatch":
				for i := range a.ArchiveCoordinates {
					if a.ArchiveCoordinates[i].Object == rows[0].Object {
						a.ArchiveCoordinates[i].ModeAnchor[0] ^= 1
					}
				}
			}
			a.StructureBindings = &rows
			if change == "absent" {
				a.StructureBindings = nil
			}
			payload, err := json.Marshal(a)
			if err == nil {
				err = sav.SetNativeActions(&doc.State, payload)
			}
			if err != nil {
				t.Fatal(err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			old, before := f.live, f.live.world.Hash()
			open, town, err := f.RestoreOriginal(bad)
			if err == nil && !town {
				err = f.App("corrupt current Building binding").OpenMission(open)
			}
			if err == nil || f.live != old || f.live.world.Hash() != before {
				t.Fatal("broken exact current receipt changed the live session", err)
			}
		})
	}
}

func TestCurrentBuildingBindingsKeepPresentEmptyRoster(t *testing.T) {
	f := structureFront(t, true, true)
	if err := f.App("current empty Building roster").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	if err := w.ImportOriginalStructures(nil, nil, nil, w.CurrentPolicy().Terrain.Block); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "present empty current Building roster")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a.StructureBindings == nil || len(*a.StructureBindings) != 0 || len(doc.World.Buildings) != 0 {
		t.Fatal("present empty current roster became absent", err)
	}
	cold := coldCurrentBuildings(t, raw)
	if source, _, present := cold.live.world.SavedStructures(); !present || len(source) != 0 || len(cold.live.world.Structures()) != 0 {
		t.Fatal("cold empty current roster resurrected ALM Buildings")
	}
}
