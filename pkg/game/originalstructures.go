package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"fmt"
	"slices"
)

// applyOriginalStructures installs the exact source roster and saved cell
// overlay. ALM is only an authored-script binding and terrain baseline.
func applyOriginalStructures(ms *Mission, source []sav.Building, present bool,
	table *mapload.Table, report *OriginalSaveResume, files ...*sav.File) error {
	return applyOriginalStructuresCurrent(ms, source, present, table, nil, nil, report, nil, files...)
}

// applyOriginalStructuresCurrent is the LOAD of the saved roster. src is the
// install the map's cell planes are read from and doc the mission document
// the cell planes import reads; each is nil where there is none.
func applyOriginalStructuresCurrent(ms *Mission, source []sav.Building, present bool, table *mapload.Table,
	src entrySource, doc *SnapshotSAVDocument, report *OriginalSaveResume, current *currentActionData, files ...*sav.File) error {
	if !present {
		return nil
	}
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original structures: mission has no map/world")
	}
	subjects, err := currentStructureSubjects(current, ms, source)
	if err != nil {
		return err
	}
	byMapID := make(map[uint32][]int)
	for i, object := range ms.Map.Objects {
		if object.Field12 != 0 {
			byMapID[uint32(object.Field12)] = append(byMapID[uint32(object.Field12)], i)
		}
	}
	var structures []sim.Structure
	var sources []sim.SavedStructure
	var counts originalStructureCounts
	seen := make(map[uint32]bool)
	byKey := make(map[uint32]sim.StructureID)
	matched := make(map[int]bool)
	for i, s := range source {
		if subjects == nil && s.AuthoredID != 0 && seen[s.AuthoredID] {
			return fmt.Errorf("original structures: ambiguous authored ID %d", s.AuthoredID)
		}
		if s.AuthoredID != 0 {
			seen[s.AuthoredID] = true
		}
		targets := byMapID[s.AuthoredID]
		if subjects == nil && len(targets) > 1 {
			return fmt.Errorf("original structures: ambiguous map authored ID %d", s.AuthoredID)
		}
		id := sim.StructureID(len(ms.Map.Objects) + i)
		ss := sim.SavedStructure{SourceKey: s.Identity, ArchiveIndex: s.ArchiveIndex,
			AuthoredID: s.AuthoredID, Position: s.Position, RuntimeID: s.RuntimeID,
			Token0C: s.Token0C, Token0E: s.Token0E, Token18: s.Token18, Token1C: s.Token1C,
			Reference: s.Reference, Base52: s.Base52, Kind: s.Kind, Field46: s.Field46,
			Field48: s.Field48, Blocking: s.Blocking, Tavern9C: s.Tavern9C, Shop70: s.Shop70,
			OutpostWords: s.OutpostWords, OutpostRecords: s.OutpostRecords}
		if subjects != nil {
			row := subjects[s.Identity]
			id, ss.HasAuthored, ss.AuthoredIndex = row.ID, row.HasAuthored, row.AuthoredIndex
			if row.HasAuthored {
				matched[int(row.AuthoredIndex)] = true
			}
		} else if len(targets) == 1 {
			id = sim.StructureID(targets[0])
			ss.HasAuthored, ss.AuthoredIndex = true, uint32(targets[0])
			matched[targets[0]] = true
		} else if s.AuthoredID == 0 {
			counts.Unbound++
		} else {
			counts.Unmatched++
		}
		switch s.Class {
		case "Building":
			ss.Class = sim.SavedBuilding
		case "Pointer":
			// The second game's Pointer is a Building with one more word
			// (R2-ENGINE-603); it keeps the Building's policy and the word as
			// read. A SAVE writes the action slot instead (DIV-4498).
			ss.Class, ss.Pointer6C = sim.SavedPointer, s.Pointer6C
		case "Outpost":
			ss.Class = sim.SavedOutpost
			counts.Subclasses++
		case "Tavern":
			ss.Class = sim.SavedTavern
			counts.Subclasses++
		case "Shop":
			ss.Class = sim.SavedShop
			counts.Subclasses++
		default:
			return fmt.Errorf("original structures: unsupported archive class %q", s.Class)
		}
		if subjects != nil && subjects[s.Identity].Class.Generated() {
			ss.Class, ss.ArchiveIndex = subjects[s.Identity].Class, 0
		}
		// The engine's records hold a Pointer as a Building; its binding
		// row holds the class and the word.
		if row := subjects[s.Identity]; subjects != nil && row.Class == sim.SavedPointer && ss.Class == sim.SavedBuilding {
			ss.Class, ss.Pointer6C = sim.SavedPointer, row.Pointer6C
		}
		ss.ID = id
		if _, exists := byKey[s.Identity]; exists || s.Identity == 0 {
			return fmt.Errorf("original structures: ambiguous source identity %#x", s.Identity)
		}
		byKey[s.Identity] = id
		structures = append(structures, sim.Structure{ID: id, Field42: s.Health, MaxHealth: s.MaxHealth,
			Col: int32(s.Col), Row: int32(s.Row), Width: s.Width, Height: s.Height, Attach: s.Attach, Blocking: s.Blocking})
		mapload.StructureUseMetadata(&structures[len(structures)-1], uint16(s.Kind), table)
		sources = append(sources, ss)
	}
	// TERR-STRUCT-068/072: terrain ingest has no Building attachment arm,
	// and archive construction does not attach. Actor occupancy is not static.
	grid := mapload.Passability(ms.Map)
	for i := range grid {
		// Original domain 2 reads static bit2 (scenery/border), not bit1.
		if grid[i]&2 != 0 || (i < len(ms.Map.Overlay) && ms.Map.Overlay[i] != 0) {
			grid[i] |= 8
		}
	}
	var cells []sim.SavedStructureCell
	if len(files) != 0 && files[0] != nil {
		f := files[0]
		rawCells, hasCells, err := f.StructureCells()
		if err != nil {
			return err
		}
		if !hasCells || f.World == nil {
			return fmt.Errorf("original structures: world cells absent")
		}
		last := make(map[uint16]int, len(rawCells))
		for i, c := range rawCells {
			last[c.Cell] = i
		}
		for i, c := range rawCells {
			if last[c.Cell] != i {
				continue
			}
			x := sim.SavedStructureCell{Cell: c.Cell, BaselineCost: c.BaselineCost, BaselineStatic: c.BaselineStatic}
			if c.BuildingKey != 0 {
				var ok bool
				x.ID, ok = byKey[c.BuildingKey]
				if !ok {
					return fmt.Errorf("original structures: cell %04x has unresolved Building identity %#x", c.Cell, c.BuildingKey)
				}
				x.HasStructure = true
			}
			cells = append(cells, x)
		}
		for _, b := range f.World.Blocks {
			x, y := b.Col(), b.Row()
			if x >= ms.Map.Width || y >= ms.Map.Height {
				continue
			}
			var block byte
			if b.Static&1 != 0 {
				block |= 1
			}
			if b.Static&2 != 0 {
				block |= 2
			}
			if b.Static&4 != 0 {
				block |= 8
			}
			grid[y*ms.Map.Width+x] = block
		}
	}
	cells, counts.Repaired, err = restoreVariableExtents(ms, structures, sources, cells, grid, src, doc, table)
	if err != nil {
		return err
	}
	if err := ms.World.ImportOriginalStructures(structures, sources, cells, grid); err != nil {
		return err
	}
	// A Pointer takes its slot before a pending use that names it is restored.
	transformWorldCarriers(carrierLoad{w: ms.World, m: ms.Map, table: table})
	counts.Restored, counts.Absent = len(structures), len(ms.Map.Objects)-len(matched)
	if report != nil {
		report.Structures = counts
	}
	return nil
}

type originalStructureCounts struct {
	Restored, Subclasses, Unbound, Unmatched, Topology, Absent int
	// Repaired counts the Buildings restoreVariableExtents rebuilt.
	Repaired int
}

// restoreVariableExtents rebuilds a loaded Building from the map where the
// document holds it at 1x1 and its map record's VariableExtent is larger
// (DIV-4627): builds before the extent read the English second game's flagged
// bridge kind saved its bridges collapsed, the deck left water. The Building
// must be bound to that record and keep its kind and anchor cell. Its
// rectangle and masks, its cell rows and their block bytes are the map's
// fresh build, read from the candidate World before the import replaces its
// roster. In doc, the mission document later stages and a SAVE read, each
// rebuilt cell's Block rows take that byte as a SAVE writes it, and its last
// Cell record names the Building (a cell without one gets one with the
// baselines). Any other Building loads as saved.
func restoreVariableExtents(ms *Mission, structures []sim.Structure, sources []sim.SavedStructure, cells []sim.SavedStructureCell,
	grid []byte, src entrySource, doc *SnapshotSAVDocument, table *mapload.Table) ([]sim.SavedStructureCell, int, error) {
	if _, _, imported := ms.World.SavedStructures(); imported {
		return cells, 0, nil
	}
	fresh := ms.World.Structures()
	type rebuild struct {
		id  sim.StructureID
		key uint32
	}
	built := make(map[sim.StructureID]rebuild)
	for i, s := range structures {
		bound := sources[i]
		if s.Width != 1 || s.Height != 1 || !bound.HasAuthored || uint64(bound.AuthoredIndex) >= uint64(len(ms.Map.Objects)) {
			continue
		}
		if w, h, ok := ms.Map.Objects[bound.AuthoredIndex].VariableExtent(); !ok || w*h <= 1 {
			continue
		}
		at := slices.IndexFunc(fresh, func(f sim.Structure) bool { return uint64(f.ID) == uint64(bound.AuthoredIndex) })
		if at < 0 || fresh[at].Kind != s.Kind || fresh[at].Col != s.Col || fresh[at].Row != s.Row {
			continue
		}
		f := fresh[at]
		structures[i].Width, structures[i].Height, structures[i].Attach, structures[i].Blocking = f.Width, f.Height, f.Attach, f.Blocking
		sources[i].Blocking = f.Blocking
		built[f.ID] = rebuild{s.ID, bound.SourceKey}
	}
	if len(built) == 0 {
		return cells, 0, nil
	}
	planes, err := currentSpatialPlanes(ms.World, ms.Map, src, table.ModTerrain())
	if err != nil {
		return nil, 0, err
	}
	owner := make(map[uint16]sim.SavedStructureCell, len(cells))
	for _, c := range cells {
		owner[c.Cell] = c
	}
	block := ms.World.CurrentPolicy().Terrain.Block
	type rebuiltCell struct {
		row   sim.SavedStructureCell
		key   uint32
		block byte
	}
	var rebuilt []rebuiltCell
	for _, row := range currentBuildingCells(ms.World, planes) {
		b, ok := built[row.ID]
		x, y := int(row.Cell&255), int(row.Cell>>8)
		if c, taken := owner[row.Cell]; !ok || taken && c.HasStructure && c.ID != b.id || x >= ms.Map.Width || y >= ms.Map.Height {
			continue
		}
		row.ID = b.id
		cells = append(cells, row)
		grid[y*ms.Map.Width+x] = block[y*ms.Map.Width+x]
		rebuilt = append(rebuilt, rebuiltCell{row, b.key, block[y*ms.Map.Width+x]})
	}
	if doc != nil && doc.Document != nil && doc.Document.World != nil {
		world := doc.Document.World
		blocks := make(map[uint16][]int)
		for i, b := range world.Blocks {
			blocks[b.Cell] = append(blocks[b.Cell], i)
		}
		last := make(map[uint16]int)
		for i, c := range world.Cells {
			last[c.Cell] = i
		}
		for _, c := range rebuilt {
			for _, i := range blocks[c.row.Cell] {
				world.Blocks[i] = terrainBlockRecord(world.Blocks[i], c.block)
			}
			if len(blocks[c.row.Cell]) == 0 {
				world.Blocks = append(world.Blocks, terrainBlockRecord(sav.BlockRecord{Cell: c.row.Cell}, c.block))
			}
			if i, ok := last[c.row.Cell]; ok {
				world.Cells[i].Building = c.key
			} else {
				world.Cells = append(world.Cells, sav.DocumentCellData{Cell: c.row.Cell,
					Cost: c.row.BaselineCost, Static: c.row.BaselineStatic, Building: c.key})
			}
		}
		slices.SortStableFunc(world.Blocks, func(a, b sav.BlockRecord) int { return int(a.Cell) - int(b.Cell) })
	}
	return cells, len(built), nil
}
